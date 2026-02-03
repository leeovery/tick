package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/flock"

	"github.com/leeovery/tick/internal/task"
)

// Helper to create a .tick/ directory with an empty tasks.jsonl.
func setupTickDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick dir: %v", err)
	}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return tickDir
}

// Helper to create a .tick/ directory with tasks already in tasks.jsonl.
func setupTickDirWithTasks(t *testing.T, tasks []task.Task) string {
	t.Helper()
	tickDir := setupTickDir(t)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

	var lines []string
	for _, tk := range tasks {
		line := fmt.Sprintf(
			`{"id":"%s","title":"%s","status":"%s","priority":%d,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`,
			tk.ID, tk.Title, string(tk.Status), tk.Priority,
		)
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
	return tickDir
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", s, err)
	}
	return ts
}

func TestNewStore(t *testing.T) {
	t.Run("it creates store when .tick/ directory exists with tasks.jsonl", func(t *testing.T) {
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		if store == nil {
			t.Fatal("NewStore() returned nil store")
		}
	})

	t.Run("it returns error when .tick/ directory does not exist", func(t *testing.T) {
		_, err := NewStore("/nonexistent/path/.tick")
		if err == nil {
			t.Fatal("NewStore() expected error for nonexistent directory, got nil")
		}
	})

	t.Run("it returns error when tasks.jsonl does not exist in .tick/", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}
		// No tasks.jsonl created

		_, err := NewStore(tickDir)
		if err == nil {
			t.Fatal("NewStore() expected error for missing tasks.jsonl, got nil")
		}
	})
}

func TestMutateExclusiveLock(t *testing.T) {
	t.Run("it acquires exclusive lock for write operations", func(t *testing.T) {
		tickDir := setupTickDir(t)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		lockAcquired := false
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			// If we get here, the exclusive lock was acquired
			lockAcquired = true

			// Verify the lock file exists and is locked
			lockPath := filepath.Join(tickDir, "lock")
			fl := flock.New(lockPath)
			// Try to acquire another exclusive lock — should fail immediately
			locked, err := fl.TryLock()
			if err != nil {
				t.Errorf("TryLock() returned error: %v", err)
			}
			if locked {
				fl.Unlock()
				t.Error("was able to acquire exclusive lock while Mutate holds it")
			}

			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate() returned error: %v", err)
		}
		if !lockAcquired {
			t.Error("mutation function was not called")
		}
	})
}

func TestQuerySharedLock(t *testing.T) {
	t.Run("it acquires shared lock for read operations", func(t *testing.T) {
		tickDir := setupTickDir(t)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		lockAcquired := false
		err = store.Query(func(db *sql.DB) error {
			lockAcquired = true

			// Verify the lock file exists and has a shared lock
			lockPath := filepath.Join(tickDir, "lock")
			fl := flock.New(lockPath)
			// Try to acquire exclusive lock — should fail (shared lock held)
			locked, err := fl.TryLock()
			if err != nil {
				t.Errorf("TryLock() returned error: %v", err)
			}
			if locked {
				fl.Unlock()
				t.Error("was able to acquire exclusive lock while Query holds shared lock")
			}

			return nil
		})
		if err != nil {
			t.Fatalf("Query() returned error: %v", err)
		}
		if !lockAcquired {
			t.Error("query function was not called")
		}
	})
}

func TestLockTimeout(t *testing.T) {
	t.Run("it returns error after lock timeout", func(t *testing.T) {
		tickDir := setupTickDir(t)

		// Acquire an exclusive lock externally
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil {
			t.Fatalf("TryLock() returned error: %v", err)
		}
		if !locked {
			t.Fatal("failed to acquire external lock")
		}
		defer fl.Unlock()

		store, err := NewStoreWithTimeout(tickDir, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("NewStoreWithTimeout() returned error: %v", err)
		}
		defer store.Close()

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			t.Error("mutation function should not have been called")
			return tasks, nil
		})
		if err == nil {
			t.Fatal("Mutate() expected timeout error, got nil")
		}
	})

	t.Run("it surfaces correct error message on lock timeout", func(t *testing.T) {
		tickDir := setupTickDir(t)

		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil {
			t.Fatalf("TryLock() returned error: %v", err)
		}
		if !locked {
			t.Fatal("failed to acquire external lock")
		}
		defer fl.Unlock()

		store, err := NewStoreWithTimeout(tickDir, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("NewStoreWithTimeout() returned error: %v", err)
		}
		defer store.Close()

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedMsg := "Could not acquire lock on .tick/lock - another process may be using tick"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), expectedMsg)
		}
	})
}

func TestConcurrentSharedLocks(t *testing.T) {
	t.Run("it allows concurrent shared locks (multiple readers)", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Test task", Status: task.StatusOpen, Priority: 2},
		})

		// Establish the cache first with a single query so concurrent readers don't race on creation
		initStore, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		err = initStore.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query() returned error: %v", err)
		}
		initStore.Close()

		const numReaders = 5
		var wg sync.WaitGroup
		errs := make(chan error, numReaders)
		started := make(chan struct{}, numReaders)

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				store, err := NewStore(tickDir)
				if err != nil {
					errs <- fmt.Errorf("NewStore() returned error: %w", err)
					return
				}
				defer store.Close()

				err = store.Query(func(db *sql.DB) error {
					started <- struct{}{}
					// Hold the lock briefly to ensure overlap
					time.Sleep(50 * time.Millisecond)
					return nil
				})
				if err != nil {
					errs <- fmt.Errorf("Query() returned error: %w", err)
				}
			}()
		}

		wg.Wait()
		close(errs)
		close(started)

		for err := range errs {
			t.Errorf("concurrent reader error: %v", err)
		}

		// All readers should have started
		count := 0
		for range started {
			count++
		}
		if count != numReaders {
			t.Errorf("started %d readers, want %d", count, numReaders)
		}
	})
}

func TestSharedBlockedByExclusive(t *testing.T) {
	t.Run("it blocks shared lock while exclusive lock is held", func(t *testing.T) {
		tickDir := setupTickDir(t)

		// Hold an exclusive lock
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil {
			t.Fatalf("TryLock() returned error: %v", err)
		}
		if !locked {
			t.Fatal("failed to acquire external exclusive lock")
		}
		defer fl.Unlock()

		store, err := NewStoreWithTimeout(tickDir, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("NewStoreWithTimeout() returned error: %v", err)
		}
		defer store.Close()

		err = store.Query(func(db *sql.DB) error {
			t.Error("query function should not have been called while exclusive lock held")
			return nil
		})
		if err == nil {
			t.Fatal("Query() expected timeout error while exclusive lock held, got nil")
		}
	})
}

func TestExclusiveBlockedByShared(t *testing.T) {
	t.Run("it blocks exclusive lock while shared lock is held", func(t *testing.T) {
		tickDir := setupTickDir(t)

		// Hold a shared lock
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryRLock()
		if err != nil {
			t.Fatalf("TryRLock() returned error: %v", err)
		}
		if !locked {
			t.Fatal("failed to acquire external shared lock")
		}
		defer fl.Unlock()

		store, err := NewStoreWithTimeout(tickDir, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("NewStoreWithTimeout() returned error: %v", err)
		}
		defer store.Close()

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			t.Error("mutation function should not have been called while shared lock held")
			return tasks, nil
		})
		if err == nil {
			t.Fatal("Mutate() expected timeout error while shared lock held, got nil")
		}
	})
}

func TestFullWriteFlow(t *testing.T) {
	t.Run("it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock", func(t *testing.T) {
		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		tickDir := setupTickDirWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Existing task", Status: task.StatusOpen, Priority: 2},
		})

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		// Mutate: add a new task
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			if len(tasks) != 1 {
				t.Errorf("mutation received %d tasks, want 1", len(tasks))
			}
			if tasks[0].ID != "tick-a1b2c3" {
				t.Errorf("tasks[0].ID = %q, want %q", tasks[0].ID, "tick-a1b2c3")
			}
			newTask := task.Task{
				ID:       "tick-new111",
				Title:    "New task",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  created,
				Updated:  created,
			}
			return append(tasks, newTask), nil
		})
		if err != nil {
			t.Fatalf("Mutate() returned error: %v", err)
		}

		// Verify JSONL was updated
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "tick-a1b2c3") {
			t.Error("tasks.jsonl missing original task tick-a1b2c3")
		}
		if !strings.Contains(content, "tick-new111") {
			t.Error("tasks.jsonl missing new task tick-new111")
		}

		// Verify SQLite cache was updated
		err = store.Query(func(db *sql.DB) error {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
				return fmt.Errorf("failed to count tasks: %w", err)
			}
			if count != 2 {
				t.Errorf("cache task count = %d, want 2", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query() returned error: %v", err)
		}
	})
}

func TestFullReadFlow(t *testing.T) {
	t.Run("it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Task one", Status: task.StatusOpen, Priority: 2},
			{ID: "tick-d4e5f6", Title: "Task two", Status: task.StatusDone, Priority: 1},
		})

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		var queriedCount int
		err = store.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&queriedCount)
		})
		if err != nil {
			t.Fatalf("Query() returned error: %v", err)
		}
		if queriedCount != 2 {
			t.Errorf("queried task count = %d, want 2", queriedCount)
		}
	})
}

func TestReleasesLockOnMutationError(t *testing.T) {
	t.Run("it releases lock on mutation function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		// Mutation that returns an error
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return nil, errors.New("mutation failed")
		})
		if err == nil {
			t.Fatal("Mutate() expected error, got nil")
		}

		// Lock should be released — verify by acquiring it
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil {
			t.Fatalf("TryLock() returned error: %v", err)
		}
		if !locked {
			t.Error("could not acquire lock after Mutate error — lock was not released")
		} else {
			fl.Unlock()
		}
	})
}

func TestReleasesLockOnQueryError(t *testing.T) {
	t.Run("it releases lock on query function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		err = store.Query(func(db *sql.DB) error {
			return errors.New("query failed")
		})
		if err == nil {
			t.Fatal("Query() expected error, got nil")
		}

		// Lock should be released
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil {
			t.Fatalf("TryLock() returned error: %v", err)
		}
		if !locked {
			t.Error("could not acquire lock after Query error — lock was not released")
		} else {
			fl.Unlock()
		}
	})
}

func TestContinuesWhenSQLiteFails(t *testing.T) {
	t.Run("it continues when JSONL write succeeds but SQLite update fails", func(t *testing.T) {
		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		// Mutate and return tasks with duplicate IDs — JSONL write succeeds (no uniqueness check),
		// but SQLite Rebuild fails (PRIMARY KEY constraint violation).
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return []task.Task{
				{ID: "tick-dup111", Title: "Dup 1", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
				{ID: "tick-dup111", Title: "Dup 2", Status: task.StatusOpen, Priority: 1, Created: created, Updated: created},
			}, nil
		})
		// The operation should succeed — JSONL is source of truth
		if err != nil {
			t.Fatalf("Mutate() returned error: %v (should succeed even with SQLite failure)", err)
		}

		// Verify JSONL was written with both entries (even with duplicate IDs)
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Errorf("tasks.jsonl has %d lines, want 2", len(lines))
		}
	})
}

func TestRebuildsStaleOnWrite(t *testing.T) {
	t.Run("it rebuilds stale cache during write before applying mutation", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Task one", Status: task.StatusOpen, Priority: 2},
		})

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		// First, do a query to establish the cache
		err = store.Query(func(db *sql.DB) error {
			var count int
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		})
		if err != nil {
			t.Fatalf("initial Query() returned error: %v", err)
		}

		// Externally modify the JSONL (simulating another process)
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		newContent := `{"id":"tick-a1b2c3","title":"Task one","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-extern1","title":"External task","status":"open","priority":3,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to write modified tasks.jsonl: %v", err)
		}

		// Mutate should detect stale cache, rebuild, and present 2 tasks to the mutation fn
		var tasksReceived int
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			tasksReceived = len(tasks)
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate() returned error: %v", err)
		}
		if tasksReceived != 2 {
			t.Errorf("mutation received %d tasks, want 2 (stale cache should have been rebuilt)", tasksReceived)
		}
	})
}

func TestRebuildsStaleOnRead(t *testing.T) {
	t.Run("it rebuilds stale cache during read before running query", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Task one", Status: task.StatusOpen, Priority: 2},
		})

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore() returned error: %v", err)
		}
		defer store.Close()

		// First query to establish cache
		err = store.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("initial Query() returned error: %v", err)
		}

		// Externally modify the JSONL
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		newContent := `{"id":"tick-a1b2c3","title":"Task one","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-extern1","title":"External task","status":"open","priority":3,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(jsonlPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to write modified tasks.jsonl: %v", err)
		}

		// Query should detect stale cache and rebuild before querying
		var taskCount int
		err = store.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
		})
		if err != nil {
			t.Fatalf("Query() returned error: %v", err)
		}
		if taskCount != 2 {
			t.Errorf("task count from SQLite = %d, want 2 (stale cache should have been rebuilt)", taskCount)
		}
	})
}
