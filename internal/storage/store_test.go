package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/task"
)

func TestStoreExclusiveLock(t *testing.T) {
	t.Run("it acquires exclusive lock for write operations", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Track if we got the lock
		lockAcquired := false

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			// Check that the lock file exists and is locked
			lockPath := filepath.Join(tickDir, "lock")
			fl := flock.New(lockPath)

			// Try to acquire exclusive lock - should fail if Store holds it
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
			if err == nil && locked {
				fl.Unlock()
				t.Error("expected to not acquire lock, but did")
			} else {
				lockAcquired = true
			}

			return tasks, nil
		})

		if err != nil {
			t.Fatalf("Mutate failed: %v", err)
		}

		if !lockAcquired {
			t.Error("lock was not acquired during mutation")
		}
	})
}

func TestStoreSharedLock(t *testing.T) {
	t.Run("it acquires shared lock for read operations", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Track if we got the shared lock
		sharedLockHeld := false

		err = store.Query(func(db *sql.DB) error {
			// Check that we can acquire another shared lock (readers don't block readers)
			lockPath := filepath.Join(tickDir, "lock")
			fl := flock.New(lockPath)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			locked, err := fl.TryRLockContext(ctx, 10*time.Millisecond)
			if err == nil && locked {
				fl.Unlock()
				sharedLockHeld = true
			}

			return nil
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if !sharedLockHeld {
			t.Error("expected to acquire shared lock while query holds shared lock")
		}
	})
}

func TestStoreLockTimeout(t *testing.T) {
	t.Run("it returns error after 5-second lock timeout", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping long test in short mode")
		}

		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		// Hold exclusive lock from outside
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer fl.Unlock()

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		start := time.Now()
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		elapsed := time.Since(start)

		if err == nil {
			t.Fatal("expected error for lock timeout")
		}

		// Should timeout after ~5 seconds
		if elapsed < 4*time.Second || elapsed > 7*time.Second {
			t.Errorf("expected timeout around 5 seconds, got %v", elapsed)
		}
	})
}

func TestStoreConcurrentSharedLocks(t *testing.T) {
	t.Run("it allows concurrent shared locks (multiple readers)", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		var wg sync.WaitGroup
		var successCount int32
		numReaders := 5

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := store.Query(func(db *sql.DB) error {
					// Hold the lock for a bit
					time.Sleep(100 * time.Millisecond)
					return nil
				})
				if err == nil {
					atomic.AddInt32(&successCount, 1)
				}
			}()
		}

		wg.Wait()

		if int(successCount) != numReaders {
			t.Errorf("expected %d successful readers, got %d", numReaders, successCount)
		}
	})
}

func TestStoreSharedBlocksExclusive(t *testing.T) {
	t.Run("it blocks exclusive lock while shared lock is held", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		// Hold shared lock from outside
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryRLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external shared lock: %v", err)
		}
		defer fl.Unlock()

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Try to mutate - should block
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
				return tasks, nil
			})
		}()

		select {
		case err := <-done:
			// If we got result before context timeout, the lock wasn't blocking
			if err == nil {
				t.Error("mutation succeeded when shared lock was held")
			}
		case <-ctx.Done():
			// Timeout - exclusive lock was properly blocked
		}
	})
}

func TestStoreExclusiveBlocksShared(t *testing.T) {
	t.Run("it blocks shared lock while exclusive lock is held", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		// Hold exclusive lock from outside
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external exclusive lock: %v", err)
		}
		defer fl.Unlock()

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Try to query - should block
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- store.Query(func(db *sql.DB) error {
				return nil
			})
		}()

		select {
		case err := <-done:
			// If we got result before context timeout, the lock wasn't blocking
			if err == nil {
				t.Error("query succeeded when exclusive lock was held")
			}
		case <-ctx.Done():
			// Timeout - shared lock was properly blocked
		}
	})
}

func TestStoreWriteFlow(t *testing.T) {
	t.Run("it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl with one task
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		initialContent := `{"id":"tick-a1b2c3","title":"Original task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(initialContent), 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Mutate to add a new task
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			if len(tasks) != 1 {
				t.Errorf("expected 1 task, got %d", len(tasks))
			}

			newTask := task.Task{
				ID:       "tick-d4e5f6",
				Title:    "New task",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  "2026-01-19T11:00:00Z",
				Updated:  "2026-01-19T11:00:00Z",
			}
			return append(tasks, newTask), nil
		})
		if err != nil {
			t.Fatalf("Mutate failed: %v", err)
		}

		// Verify JSONL was updated
		content, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 lines in JSONL, got %d", len(lines))
		}

		// Verify SQLite cache was updated by querying it
		err = store.Query(func(db *sql.DB) error {
			var count int
			err := db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count)
			if err != nil {
				return err
			}
			if count != 2 {
				t.Errorf("expected 2 tasks in cache, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
	})
}

func TestStoreReadFlow(t *testing.T) {
	t.Run("it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl with tasks
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		content := `{"id":"tick-a1b2c3","title":"Task 1","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Task 2","status":"done","priority":1,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T12:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		var queryResult []string
		err = store.Query(func(db *sql.DB) error {
			rows, err := db.Query(`SELECT id FROM tasks ORDER BY id`)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					return err
				}
				queryResult = append(queryResult, id)
			}
			return rows.Err()
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(queryResult) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(queryResult))
		}
		if queryResult[0] != "tick-a1b2c3" || queryResult[1] != "tick-d4e5f6" {
			t.Errorf("unexpected query result: %v", queryResult)
		}
	})
}

func TestStoreMutationErrorReleasesLock(t *testing.T) {
	t.Run("it releases lock on mutation function error (no leak)", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Mutate with error
		mutateErr := store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return nil, os.ErrInvalid // Return an error
		})

		if mutateErr == nil {
			t.Fatal("expected error from mutation")
		}

		// Verify lock was released by acquiring it again
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
		if err != nil || !locked {
			t.Error("lock was not released after mutation error")
		}
		if locked {
			fl.Unlock()
		}
	})
}

func TestStoreQueryErrorReleasesLock(t *testing.T) {
	t.Run("it releases lock on query function error (no leak)", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Query with error
		queryErr := store.Query(func(db *sql.DB) error {
			return os.ErrInvalid // Return an error
		})

		if queryErr == nil {
			t.Fatal("expected error from query")
		}

		// Verify lock was released by acquiring it again
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
		if err != nil || !locked {
			t.Error("lock was not released after query error")
		}
		if locked {
			fl.Unlock()
		}
	})
}

func TestStoreJSONLSuccessSQLiteFailure(t *testing.T) {
	t.Run("it continues when JSONL write succeeds but SQLite update fails", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}

		// First, do a successful mutation to create the cache
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return []task.Task{
				{
					ID:       "tick-a1b2c3",
					Title:    "First task",
					Status:   task.StatusOpen,
					Priority: 2,
					Created:  "2026-01-19T10:00:00Z",
					Updated:  "2026-01-19T10:00:00Z",
				},
			}, nil
		})
		if err != nil {
			t.Fatalf("Initial Mutate failed: %v", err)
		}

		// Close the cache to simulate SQLite unavailability
		store.Close()

		// Make cache.db read-only to cause SQLite update to fail
		cachePath := filepath.Join(tickDir, "cache.db")
		os.Chmod(cachePath, 0444)
		defer os.Chmod(cachePath, 0644)

		// Create new store
		store2, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store2.Close()

		// Capture stderr to verify warning is logged
		// (We can't easily test stderr in this setup, but we verify the operation succeeds)

		// Mutate should succeed even if SQLite fails
		err = store2.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return append(tasks, task.Task{
				ID:       "tick-d4e5f6",
				Title:    "Second task",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  "2026-01-19T11:00:00Z",
				Updated:  "2026-01-19T11:00:00Z",
			}), nil
		})

		// The mutation should succeed (JSONL is source of truth)
		if err != nil {
			t.Fatalf("Mutate failed when it should have succeeded: %v", err)
		}

		// Verify JSONL was updated
		content, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 lines in JSONL, got %d", len(lines))
		}
	})
}

func TestStoreStaleCache_Write(t *testing.T) {
	t.Run("it rebuilds stale cache during write before applying mutation", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl with one task
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		content := `{"id":"tick-a1b2c3","title":"Task 1","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Build initial cache
		err = store.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("initial Query failed: %v", err)
		}

		// Modify JSONL externally (simulating git pull or external edit)
		newContent := `{"id":"tick-a1b2c3","title":"Task 1","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-external","title":"External task","status":"open","priority":1,"created":"2026-01-19T12:00:00Z","updated":"2026-01-19T12:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to modify tasks.jsonl: %v", err)
		}

		// Mutate should see the external task (cache was rebuilt from modified JSONL)
		var taskCount int
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			taskCount = len(tasks)
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate failed: %v", err)
		}

		if taskCount != 2 {
			t.Errorf("expected 2 tasks after stale cache rebuild, got %d", taskCount)
		}
	})
}

func TestStoreStaleCache_Read(t *testing.T) {
	t.Run("it rebuilds stale cache during read before running query", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl with one task
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		content := `{"id":"tick-a1b2c3","title":"Task 1","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		// Build initial cache
		err = store.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("initial Query failed: %v", err)
		}

		// Modify JSONL externally
		newContent := `{"id":"tick-a1b2c3","title":"Task 1","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-external","title":"External task","status":"open","priority":1,"created":"2026-01-19T12:00:00Z","updated":"2026-01-19T12:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to modify tasks.jsonl: %v", err)
		}

		// Query should see the external task (cache was rebuilt)
		var count int
		err = store.Query(func(db *sql.DB) error {
			return db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count)
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if count != 2 {
			t.Errorf("expected 2 tasks after stale cache rebuild, got %d", count)
		}
	})
}

func TestStoreLockTimeoutMessage(t *testing.T) {
	t.Run("it surfaces correct error message on lock timeout", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping long test in short mode")
		}

		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		// Create tasks.jsonl
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		// Hold exclusive lock from outside
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer fl.Unlock()

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore failed: %v", err)
		}
		defer store.Close()

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})

		if err == nil {
			t.Fatal("expected error for lock timeout")
		}

		expectedMsg := "could not acquire lock on .tick/lock - another process may be using tick"
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(expectedMsg)) {
			t.Errorf("error message = %q, want to contain %q", err.Error(), expectedMsg)
		}
	})
}

func TestNewStoreValidation(t *testing.T) {
	t.Run("it validates .tick directory exists and contains tasks.jsonl", func(t *testing.T) {
		dir := t.TempDir()

		// Try with non-existent directory
		_, err := NewStore(filepath.Join(dir, "nonexistent"))
		if err == nil {
			t.Error("expected error for non-existent directory")
		}

		// Create .tick but not tasks.jsonl
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}

		_, err = NewStore(tickDir)
		if err == nil {
			t.Error("expected error for missing tasks.jsonl")
		}

		// Create tasks.jsonl - should succeed
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Errorf("expected success for valid directory, got: %v", err)
		}
		if store != nil {
			store.Close()
		}
	})
}
