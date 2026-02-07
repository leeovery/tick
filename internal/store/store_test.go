package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/task"
	_ "github.com/mattn/go-sqlite3"
)

// setupTickDir creates a temporary .tick/ directory with an empty tasks.jsonl file.
func setupTickDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick dir: %v", err)
	}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return tickDir
}

// setupTickDirWithTasks creates a .tick/ directory with pre-populated tasks.
func setupTickDirWithTasks(t *testing.T, tasks []task.Task) string {
	t.Helper()
	tickDir := setupTickDir(t)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := task.WriteJSONL(jsonlPath, tasks); err != nil {
		t.Fatalf("failed to write tasks: %v", err)
	}
	return tickDir
}

// sampleTasks returns a set of test tasks.
func sampleTasks() []task.Task {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	return []task.Task{
		{
			ID:       "tick-aaa111",
			Title:    "First task",
			Status:   task.StatusOpen,
			Priority: 1,
			Created:  now,
			Updated:  now,
		},
		{
			ID:       "tick-bbb222",
			Title:    "Second task",
			Status:   task.StatusOpen,
			Priority: 2,
			Created:  now,
			Updated:  now,
		},
	}
}

func TestStore_WriteFlow(t *testing.T) {
	t.Run("it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		// Mutate: add a new task
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			now := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
			newTask := task.Task{
				ID:       "tick-ccc333",
				Title:    "Third task",
				Status:   task.StatusOpen,
				Priority: 3,
				Created:  now,
				Updated:  now,
			}
			return append(tasks, newTask), nil
		})
		if err != nil {
			t.Fatalf("Mutate returned error: %v", err)
		}

		// Verify JSONL was updated
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		readTasks, err := task.ReadJSONL(jsonlPath)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}
		if len(readTasks) != 3 {
			t.Fatalf("expected 3 tasks in JSONL, got %d", len(readTasks))
		}
		if readTasks[2].ID != "tick-ccc333" {
			t.Errorf("expected third task ID tick-ccc333, got %s", readTasks[2].ID)
		}

		// Verify SQLite cache was updated
		dbPath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks in cache: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 tasks in cache, got %d", count)
		}

		// Verify hash was stored
		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to read hash from cache: %v", err)
		}
		if storedHash == "" {
			t.Error("expected non-empty hash in metadata")
		}
	})
}

func TestStore_LockTimeout(t *testing.T) {
	t.Run("it returns error after lock timeout", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		// Use a very short timeout for testing
		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()
		s.lockTimeout = 50 * time.Millisecond

		// Hold an exclusive lock externally
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer fl.Unlock()

		// Mutate should timeout
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err == nil {
			t.Fatal("expected error from Mutate when lock is held, got nil")
		}
		if !strings.Contains(err.Error(), "could not acquire lock") {
			t.Errorf("expected lock timeout error, got: %v", err)
		}
	})

	t.Run("it surfaces correct error message on lock timeout", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()
		s.lockTimeout = 50 * time.Millisecond

		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer fl.Unlock()

		// Test Query timeout
		err = s.Query(func(db *sql.DB) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error from Query when lock is held, got nil")
		}

		expectedMsg := fmt.Sprintf("could not acquire lock on %s - another process may be using tick", lockPath)
		if err.Error() != expectedMsg {
			t.Errorf("expected error message:\n  %s\ngot:\n  %s", expectedMsg, err.Error())
		}
	})
}

func TestStore_ReadFlow(t *testing.T) {
	t.Run("it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		var count int
		err = s.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 tasks from query, got %d", count)
		}
	})
}

func TestStore_ExclusiveLock(t *testing.T) {
	t.Run("it acquires exclusive lock for write operations", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()
		s.lockTimeout = 50 * time.Millisecond

		// Track whether the lock is held during mutation
		lockPath := filepath.Join(tickDir, "lock")
		lockHeld := false

		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			// While inside mutation, try to acquire the same lock — should fail
			fl := flock.New(lockPath)
			got, err := fl.TryLock()
			if err == nil && !got {
				lockHeld = true
			}
			if got {
				fl.Unlock()
			}
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate returned error: %v", err)
		}
		if !lockHeld {
			t.Error("expected exclusive lock to be held during mutation")
		}
	})
}

func TestStore_SharedLock(t *testing.T) {
	t.Run("it acquires shared lock for read operations", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		lockPath := filepath.Join(tickDir, "lock")
		sharedLockAcquired := false

		err = s.Query(func(db *sql.DB) error {
			// While inside query, try to acquire a shared lock — should succeed
			fl := flock.New(lockPath)
			got, err := fl.TryRLock()
			if err == nil && got {
				sharedLockAcquired = true
				fl.Unlock()
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if !sharedLockAcquired {
			t.Error("expected shared lock to be acquirable during read (concurrent readers)")
		}
	})
}

func TestStore_ConcurrentReaders(t *testing.T) {
	t.Run("it allows concurrent shared locks (multiple readers)", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		const numReaders = 5
		var wg sync.WaitGroup
		errors := make([]error, numReaders)
		counts := make([]int, numReaders)

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				s, err := NewStore(tickDir)
				if err != nil {
					errors[idx] = err
					return
				}
				defer s.Close()

				errors[idx] = s.Query(func(db *sql.DB) error {
					// Small delay to ensure overlap
					time.Sleep(20 * time.Millisecond)
					return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&counts[idx])
				})
			}(i)
		}
		wg.Wait()

		for i := 0; i < numReaders; i++ {
			if errors[i] != nil {
				t.Errorf("reader %d returned error: %v", i, errors[i])
			}
			if counts[i] != 2 {
				t.Errorf("reader %d got count %d, expected 2", i, counts[i])
			}
		}
	})
}

func TestStore_SharedBlockedByExclusive(t *testing.T) {
	t.Run("it blocks shared lock while exclusive lock is held", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()
		s.lockTimeout = 100 * time.Millisecond

		// Hold exclusive lock externally
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external exclusive lock: %v", err)
		}
		defer fl.Unlock()

		// Query (shared lock) should fail
		err = s.Query(func(db *sql.DB) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error from Query when exclusive lock is held, got nil")
		}
		if !strings.Contains(err.Error(), "could not acquire lock") {
			t.Errorf("expected lock error, got: %v", err)
		}
	})
}

func TestStore_ExclusiveBlockedByShared(t *testing.T) {
	t.Run("it blocks exclusive lock while shared lock is held", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()
		s.lockTimeout = 100 * time.Millisecond

		// Hold shared lock externally
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryRLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external shared lock: %v", err)
		}
		defer fl.Unlock()

		// Mutate (exclusive lock) should fail
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err == nil {
			t.Fatal("expected error from Mutate when shared lock is held, got nil")
		}
		if !strings.Contains(err.Error(), "could not acquire lock") {
			t.Errorf("expected lock error, got: %v", err)
		}
	})
}

func TestStore_LockReleaseOnMutationError(t *testing.T) {
	t.Run("it releases lock on mutation function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		// Mutation that returns an error
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return nil, fmt.Errorf("intentional mutation error")
		})
		if err == nil {
			t.Fatal("expected error from Mutate, got nil")
		}

		// Lock should be released — a new operation should succeed
		err = s.Query(func(db *sql.DB) error {
			var count int
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		})
		if err != nil {
			t.Errorf("Query after failed Mutate returned error (lock leaked?): %v", err)
		}
	})
}

func TestStore_LockReleaseOnQueryError(t *testing.T) {
	t.Run("it releases lock on query function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		// Query that returns an error
		err = s.Query(func(db *sql.DB) error {
			return fmt.Errorf("intentional query error")
		})
		if err == nil {
			t.Fatal("expected error from Query, got nil")
		}

		// Lock should be released — a new operation should succeed
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err != nil {
			t.Errorf("Mutate after failed Query returned error (lock leaked?): %v", err)
		}
	})
}

func TestStore_SQLiteFailureAfterJSONLWrite(t *testing.T) {
	t.Run("it continues when JSONL write succeeds but SQLite update fails", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		// Make cache.db path point to a directory (will cause Open to fail)
		dbDir := filepath.Join(tickDir, "cache.db")
		// First do a normal mutation to create cache.db
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("initial Mutate returned error: %v", err)
		}

		// Now corrupt the cache.db by replacing it with a directory
		os.Remove(dbDir)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			t.Fatalf("failed to create directory at cache.db path: %v", err)
		}

		// Mutation should still succeed (JSONL write succeeds, SQLite failure is logged)
		now := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return append(tasks, task.Task{
				ID:       "tick-ddd444",
				Title:    "New task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			}), nil
		})
		if err != nil {
			t.Fatalf("Mutate should succeed even when SQLite fails, got error: %v", err)
		}

		// Verify JSONL was still updated
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		readTasks, err := task.ReadJSONL(jsonlPath)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}
		if len(readTasks) != 3 {
			t.Errorf("expected 3 tasks in JSONL after mutation, got %d", len(readTasks))
		}
	})
}

func TestStore_StaleCacheRebuild(t *testing.T) {
	t.Run("it rebuilds stale cache during write before applying mutation", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		// Do initial read to create cache
		err = s.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}

		// Modify JSONL externally (simulate git pull), making cache stale
		now := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
		externalTasks := []task.Task{
			{
				ID:       "tick-ext111",
				Title:    "External task",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
		}
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := task.WriteJSONL(jsonlPath, externalTasks); err != nil {
			t.Fatalf("failed to write external tasks: %v", err)
		}

		// Mutate should rebuild cache from stale JSONL, then the mutation sees the external task
		var seenTasks []task.Task
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			seenTasks = tasks
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate returned error: %v", err)
		}

		// The mutation should have seen the externally-modified task list
		if len(seenTasks) != 1 {
			t.Fatalf("expected mutation to see 1 task (external), got %d", len(seenTasks))
		}
		if seenTasks[0].ID != "tick-ext111" {
			t.Errorf("expected mutation to see tick-ext111, got %s", seenTasks[0].ID)
		}
	})

	t.Run("it rebuilds stale cache during read before running query", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer s.Close()

		// Do initial read to create cache
		err = s.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}

		// Modify JSONL externally, making cache stale
		now := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
		externalTasks := []task.Task{
			{
				ID:       "tick-ext222",
				Title:    "External read task",
				Status:   task.StatusDone,
				Priority: 0,
				Created:  now,
				Updated:  now,
			},
		}
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := task.WriteJSONL(jsonlPath, externalTasks); err != nil {
			t.Fatalf("failed to write external tasks: %v", err)
		}

		// Query should rebuild cache and see the external data
		var count int
		var taskID string
		err = s.Query(func(db *sql.DB) error {
			if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
				return err
			}
			return db.QueryRow("SELECT id FROM tasks").Scan(&taskID)
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task after stale cache rebuild, got %d", count)
		}
		if taskID != "tick-ext222" {
			t.Errorf("expected task ID tick-ext222, got %s", taskID)
		}
	})
}

