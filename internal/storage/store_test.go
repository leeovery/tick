package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/task"
)

// setupTickDir creates a .tick/ directory with an empty tasks.jsonl file for testing.
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

// setupTickDirWithTasks creates a .tick/ directory with tasks pre-written to tasks.jsonl.
func setupTickDirWithTasks(t *testing.T, tasks []task.Task) string {
	t.Helper()
	tickDir := setupTickDir(t)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := WriteJSONL(jsonlPath, tasks); err != nil {
		t.Fatalf("failed to write tasks: %v", err)
	}
	return tickDir
}

func TestStoreMutate(t *testing.T) {
	t.Run("it acquires exclusive lock for write operations", func(t *testing.T) {
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		lockPath := filepath.Join(tickDir, "lock")

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			// While inside the mutation, the lock file should exist
			// (created by flock).
			if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
				t.Error("lock file does not exist during mutation")
			}
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate returned error: %v", err)
		}
	})
}

func TestStoreQuery(t *testing.T) {
	t.Run("it acquires shared lock for read operations", func(t *testing.T) {
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		lockPath := filepath.Join(tickDir, "lock")

		err = store.Query(func(db *sql.DB) error {
			// While inside the query, the lock file should exist
			// (created by flock).
			if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
				t.Error("lock file does not exist during query")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})
}

func TestStoreLockTimeout(t *testing.T) {
	t.Run("it returns error after lock timeout", func(t *testing.T) {
		tickDir := setupTickDir(t)
		lockPath := filepath.Join(tickDir, "lock")

		// Hold an exclusive lock externally using flock directly.
		externalLock := flock.New(lockPath)
		if err := externalLock.Lock(); err != nil {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer func() { _ = externalLock.Unlock() }()

		store, err := NewStore(tickDir, WithLockTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			t.Error("mutation function should not have been called")
			return tasks, nil
		})
		if err == nil {
			t.Fatal("expected lock timeout error, got nil")
		}

		expected := "could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it surfaces correct error message on lock timeout", func(t *testing.T) {
		tickDir := setupTickDir(t)
		lockPath := filepath.Join(tickDir, "lock")

		// Hold an exclusive lock externally.
		externalLock := flock.New(lockPath)
		if err := externalLock.Lock(); err != nil {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer func() { _ = externalLock.Unlock() }()

		store, err := NewStore(tickDir, WithLockTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Test timeout on Query too.
		err = store.Query(func(db *sql.DB) error {
			t.Error("query function should not have been called")
			return nil
		})
		if err == nil {
			t.Fatal("expected lock timeout error, got nil")
		}

		expected := "could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})
}

func TestStoreConcurrentLocks(t *testing.T) {
	t.Run("it allows concurrent shared locks (multiple readers)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		lockPath := filepath.Join(tickDir, "lock")

		// Acquire two shared locks simultaneously using flock directly.
		lock1 := flock.New(lockPath)
		lock2 := flock.New(lockPath)

		if err := lock1.RLock(); err != nil {
			t.Fatalf("failed to acquire first shared lock: %v", err)
		}
		defer func() { _ = lock1.Unlock() }()

		// Second shared lock should succeed while first is held.
		locked, err := lock2.TryRLock()
		if err != nil {
			t.Fatalf("second shared lock returned error: %v", err)
		}
		if !locked {
			t.Error("expected second shared lock to succeed, but it was blocked")
		}
		if locked {
			_ = lock2.Unlock()
		}
	})

	t.Run("it blocks shared lock while exclusive lock is held", func(t *testing.T) {
		tickDir := setupTickDir(t)
		lockPath := filepath.Join(tickDir, "lock")

		// Hold exclusive lock.
		exclusiveLock := flock.New(lockPath)
		if err := exclusiveLock.Lock(); err != nil {
			t.Fatalf("failed to acquire exclusive lock: %v", err)
		}
		defer func() { _ = exclusiveLock.Unlock() }()

		// Shared lock attempt should fail (non-blocking TryRLock).
		sharedLock := flock.New(lockPath)
		locked, err := sharedLock.TryRLock()
		if err != nil {
			t.Fatalf("shared lock returned error: %v", err)
		}
		if locked {
			_ = sharedLock.Unlock()
			t.Error("expected shared lock to be blocked by exclusive lock, but it succeeded")
		}
	})

	t.Run("it blocks exclusive lock while shared lock is held", func(t *testing.T) {
		tickDir := setupTickDir(t)
		lockPath := filepath.Join(tickDir, "lock")

		// Hold shared lock.
		sharedLock := flock.New(lockPath)
		if err := sharedLock.RLock(); err != nil {
			t.Fatalf("failed to acquire shared lock: %v", err)
		}
		defer func() { _ = sharedLock.Unlock() }()

		// Exclusive lock attempt should fail (non-blocking TryLock).
		exclusiveLock := flock.New(lockPath)
		locked, err := exclusiveLock.TryLock()
		if err != nil {
			t.Fatalf("exclusive lock returned error: %v", err)
		}
		if locked {
			_ = exclusiveLock.Unlock()
			t.Error("expected exclusive lock to be blocked by shared lock, but it succeeded")
		}
	})
}

func TestStoreWriteFlow(t *testing.T) {
	t.Run("it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		initialTasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Existing task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, initialTasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Mutate: add a new task.
		newTask := task.Task{
			ID:       "tick-bbbbbb",
			Title:    "New task",
			Status:   task.StatusOpen,
			Priority: 1,
			Created:  created,
			Updated:  created,
		}

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			if len(tasks) != 1 {
				t.Errorf("expected 1 task passed to mutation, got %d", len(tasks))
			}
			if tasks[0].ID != "tick-aaaaaa" {
				t.Errorf("expected task ID tick-aaaaaa, got %q", tasks[0].ID)
			}
			return append(tasks, newTask), nil
		})
		if err != nil {
			t.Fatalf("Mutate returned error: %v", err)
		}

		// Verify JSONL was updated.
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		readBack, err := ReadJSONL(jsonlPath)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}
		if len(readBack) != 2 {
			t.Fatalf("expected 2 tasks in JSONL, got %d", len(readBack))
		}
		if readBack[1].ID != "tick-bbbbbb" {
			t.Errorf("second task ID = %q, want %q", readBack[1].ID, "tick-bbbbbb")
		}

		// Verify SQLite cache was updated.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
				return err
			}
			if count != 2 {
				t.Errorf("expected 2 tasks in SQLite cache, got %d", count)
			}

			var title string
			if err := db.QueryRow("SELECT title FROM tasks WHERE id = ?", "tick-bbbbbb").Scan(&title); err != nil {
				return err
			}
			if title != "New task" {
				t.Errorf("SQLite task title = %q, want %q", title, "New task")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})
}

func TestStoreReadFlow(t *testing.T) {
	t.Run("it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Task A",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-bbbbbb",
				Title:    "Task B",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		err = store.Query(func(db *sql.DB) error {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
				return err
			}
			if count != 2 {
				t.Errorf("expected 2 tasks in SQLite, got %d", count)
			}

			var title string
			if err := db.QueryRow("SELECT title FROM tasks WHERE id = ?", "tick-bbbbbb").Scan(&title); err != nil {
				return err
			}
			if title != "Task B" {
				t.Errorf("SQLite task title = %q, want %q", title, "Task B")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})
}

func TestStoreLockRelease(t *testing.T) {
	t.Run("it releases lock on mutation function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Mutate with an error.
		mutErr := store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return nil, fmt.Errorf("intentional mutation error")
		})
		if mutErr == nil {
			t.Fatal("expected error from mutation, got nil")
		}

		// Lock should be released — subsequent mutation should succeed.
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("subsequent Mutate after error returned error (lock leaked?): %v", err)
		}
	})

	t.Run("it releases lock on query function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Query with an error.
		queryErr := store.Query(func(db *sql.DB) error {
			return fmt.Errorf("intentional query error")
		})
		if queryErr == nil {
			t.Fatal("expected error from query, got nil")
		}

		// Lock should be released — subsequent query should succeed.
		err = store.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("subsequent Query after error returned error (lock leaked?): %v", err)
		}
	})
}

func TestStoreSQLiteFailure(t *testing.T) {
	t.Run("it continues when JSONL write succeeds but SQLite update fails", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		initialTasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Existing task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, initialTasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		newTask := task.Task{
			ID:       "tick-bbbbbb",
			Title:    "New task",
			Status:   task.StatusOpen,
			Priority: 1,
			Created:  created,
			Updated:  created,
		}

		// Make cache.db path a directory to force SQLite rebuild failure.
		cachePath := filepath.Join(tickDir, "cache.db")

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			// After JSONL is read and cache is initially built, corrupt the cache
			// so the post-write rebuild fails. Remove the cache file and replace
			// with a directory.
			os.Remove(cachePath)
			if mkErr := os.MkdirAll(cachePath, 0755); mkErr != nil {
				t.Fatalf("failed to create directory at cache path: %v", mkErr)
			}
			return append(tasks, newTask), nil
		})

		// Should succeed despite SQLite failure (JSONL-first principle).
		if err != nil {
			t.Fatalf("Mutate returned error when SQLite fails: %v", err)
		}

		// JSONL should have the new task.
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		readBack, err := ReadJSONL(jsonlPath)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}
		if len(readBack) != 2 {
			t.Fatalf("expected 2 tasks in JSONL, got %d", len(readBack))
		}

		// Clean up directory so next query can recreate cache.
		os.RemoveAll(cachePath)

		// Next query should self-heal by rebuilding cache from JSONL.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
				return err
			}
			if count != 2 {
				t.Errorf("expected 2 tasks after self-heal, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query after self-heal returned error: %v", err)
		}
	})
}

func TestStoreRebuild(t *testing.T) {
	t.Run("it rebuilds cache from JSONL and returns task count", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
			{ID: "tick-bbbbbb", Title: "Task B", Status: task.StatusDone, Priority: 1, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		count, err := store.Rebuild()
		if err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}
		if count != 2 {
			t.Errorf("Rebuild returned count %d, want 2", count)
		}

		// Verify cache has the tasks by querying through the store.
		err = store.Query(func(db *sql.DB) error {
			var dbCount int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&dbCount); qErr != nil {
				return qErr
			}
			if dbCount != 2 {
				t.Errorf("cache task count = %d, want 2", dbCount)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query after Rebuild returned error: %v", err)
		}
	})

	t.Run("it works when cache.db does not exist", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		// Ensure no cache.db exists.
		cachePath := filepath.Join(tickDir, "cache.db")
		os.Remove(cachePath)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		count, err := store.Rebuild()
		if err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}
		if count != 1 {
			t.Errorf("Rebuild returned count %d, want 1", count)
		}

		// Verify cache was created.
		if _, statErr := os.Stat(cachePath); os.IsNotExist(statErr) {
			t.Error("cache.db should exist after Rebuild")
		}
	})

	t.Run("it works when cache.db is corrupted", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		// Write garbage to cache.db to simulate corruption.
		cachePath := filepath.Join(tickDir, "cache.db")
		if wErr := os.WriteFile(cachePath, []byte("not a sqlite database"), 0644); wErr != nil {
			t.Fatalf("failed to write corrupted cache: %v", wErr)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		count, err := store.Rebuild()
		if err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}
		if count != 1 {
			t.Errorf("Rebuild returned count %d, want 1", count)
		}

		// Verify cache is now valid by querying.
		err = store.Query(func(db *sql.DB) error {
			var dbCount int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&dbCount); qErr != nil {
				return qErr
			}
			if dbCount != 1 {
				t.Errorf("cache task count = %d, want 1", dbCount)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query after Rebuild with corrupted cache returned error: %v", err)
		}
	})

	t.Run("it updates hash in metadata after rebuild", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.Rebuild()
		if err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		// Verify hash is stored by querying via store.
		err = store.Query(func(db *sql.DB) error {
			var hash string
			if qErr := db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&hash); qErr != nil {
				return qErr
			}
			if hash == "" {
				t.Error("hash should not be empty after Rebuild")
			}
			if len(hash) != 64 {
				t.Errorf("hash length = %d, want 64 (SHA256 hex)", len(hash))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query after Rebuild returned error: %v", err)
		}
	})

	t.Run("it acquires exclusive lock during rebuild", func(t *testing.T) {
		tickDir := setupTickDir(t)
		lockPath := filepath.Join(tickDir, "lock")

		// Hold an exclusive lock externally.
		externalLock := flock.New(lockPath)
		if lErr := externalLock.Lock(); lErr != nil {
			t.Fatalf("failed to acquire external lock: %v", lErr)
		}
		defer func() { _ = externalLock.Unlock() }()

		store, err := NewStore(tickDir, WithLockTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.Rebuild()
		if err == nil {
			t.Fatal("expected lock timeout error, got nil")
		}

		expected := "could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it logs verbose messages during rebuild", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		var logged []string
		store, err := NewStore(tickDir, WithVerbose(func(msg string) {
			logged = append(logged, msg)
		}))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.Rebuild()
		if err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		// Check for expected verbose messages.
		expectedMessages := []string{
			"acquiring exclusive lock",
			"lock acquired",
			"deleting cache.db",
			"reading JSONL",
			"rebuilding cache with 1 tasks",
			"hash updated",
			"lock released",
		}
		if len(logged) != len(expectedMessages) {
			t.Errorf("logged %d messages, want %d: %v", len(logged), len(expectedMessages), logged)
		}
		for i, want := range expectedMessages {
			if i >= len(logged) {
				break
			}
			if logged[i] != want {
				t.Errorf("log[%d] = %q, want %q", i, logged[i], want)
			}
		}
	})

	t.Run("it handles empty JSONL returning 0 tasks", func(t *testing.T) {
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		count, err := store.Rebuild()
		if err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}
		if count != 0 {
			t.Errorf("Rebuild returned count %d, want 0", count)
		}
	})
}

func TestStoreCacheFreshnessRecovery(t *testing.T) {
	t.Run("it rebuilds automatically when cache.db is missing", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		// Ensure no cache.db exists before Store is used.
		cachePath := filepath.Join(tickDir, "cache.db")
		os.Remove(cachePath)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Query should trigger ensureFresh which creates the cache from scratch.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("expected 1 task in cache, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})

	t.Run("it deletes and rebuilds when cache.db is corrupted", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		// Write garbage data to cache.db to simulate corruption.
		cachePath := filepath.Join(tickDir, "cache.db")
		if wErr := os.WriteFile(cachePath, []byte("this is not a sqlite database"), 0644); wErr != nil {
			t.Fatalf("failed to write corrupted file: %v", wErr)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Query should recover from corruption by deleting and recreating cache.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("expected 1 task in cache after corruption recovery, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error after corrupted cache: %v", err)
		}
	})

	t.Run("it detects stale cache via hash mismatch and rebuilds", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		initialTasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Original task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, initialTasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Prime the cache with initial data.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}

		// Externally modify JSONL to make the cache stale.
		updatedTasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Updated task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-bbbbbb",
				Title:    "New external task",
				Status:   task.StatusOpen,
				Priority: 3,
				Created:  created,
				Updated:  created,
			},
		}
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if wErr := WriteJSONL(jsonlPath, updatedTasks); wErr != nil {
			t.Fatalf("failed to write updated JSONL: %v", wErr)
		}

		// Query should detect hash mismatch and rebuild.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 2 {
				t.Errorf("expected 2 tasks after stale rebuild, got %d", count)
			}

			var title string
			if qErr := db.QueryRow("SELECT title FROM tasks WHERE id = ?", "tick-aaaaaa").Scan(&title); qErr != nil {
				return qErr
			}
			if title != "Updated task" {
				t.Errorf("title = %q, want %q", title, "Updated task")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})

	t.Run("it handles freshness check errors from corrupted metadata", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Prime the cache so the store has an open cache connection.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}

		// Corrupt the metadata table by dropping and replacing it with an
		// incompatible schema. This makes IsFresh fail with a scan error
		// because the column types no longer match the expected query.
		// We use the store's own cache DB connection so the corruption
		// is visible on the next ensureFresh call.
		cachePath := filepath.Join(tickDir, "cache.db")
		corruptDB, cErr := sql.Open("sqlite", cachePath)
		if cErr != nil {
			t.Fatalf("failed to open cache for corruption: %v", cErr)
		}
		_, cErr = corruptDB.Exec("DROP TABLE metadata; CREATE TABLE metadata (broken INTEGER)")
		if cErr != nil {
			t.Fatalf("failed to corrupt metadata table: %v", cErr)
		}
		corruptDB.Close()

		// Close the store so it drops the cached connection and will reopen
		// the corrupted file on next use.
		store.Close()

		store, err = NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Query should recover: IsFresh fails on corrupted metadata,
		// store deletes cache and recreates it, then rebuilds.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("expected 1 task after metadata corruption recovery, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error after metadata corruption: %v", err)
		}
	})
}

func TestReadTasks(t *testing.T) {
	t.Run("it returns parsed tasks from JSONL under shared lock", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
			{ID: "tick-bbbbbb", Title: "Task B", Status: task.StatusDone, Priority: 1, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ReadTasks()
		if err != nil {
			t.Fatalf("ReadTasks returned error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("ReadTasks returned %d tasks, want 2", len(got))
		}
		if got[0].ID != "tick-aaaaaa" {
			t.Errorf("got[0].ID = %q, want %q", got[0].ID, "tick-aaaaaa")
		}
		if got[1].ID != "tick-bbbbbb" {
			t.Errorf("got[1].ID = %q, want %q", got[1].ID, "tick-bbbbbb")
		}
	})

	t.Run("it returns empty slice for empty JSONL", func(t *testing.T) {
		tickDir := setupTickDir(t)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ReadTasks()
		if err != nil {
			t.Fatalf("ReadTasks returned error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("ReadTasks returned %d tasks, want 0", len(got))
		}
	})

	t.Run("it uses shared lock (not exclusive)", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)
		lockPath := filepath.Join(tickDir, "lock")

		// Hold a shared lock externally — ReadTasks should still work
		// because it also uses a shared lock.
		externalLock := flock.New(lockPath)
		if err := externalLock.RLock(); err != nil {
			t.Fatalf("failed to acquire external shared lock: %v", err)
		}
		defer func() { _ = externalLock.Unlock() }()

		store, err := NewStore(tickDir, WithLockTimeout(500*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ReadTasks()
		if err != nil {
			t.Fatalf("ReadTasks should succeed with concurrent shared lock, got error: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("ReadTasks returned %d tasks, want 1", len(got))
		}
	})

	t.Run("it blocks when exclusive lock is held", func(t *testing.T) {
		tickDir := setupTickDir(t)
		lockPath := filepath.Join(tickDir, "lock")

		// Hold an exclusive lock externally.
		externalLock := flock.New(lockPath)
		if err := externalLock.Lock(); err != nil {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer func() { _ = externalLock.Unlock() }()

		store, err := NewStore(tickDir, WithLockTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ReadTasks()
		if err == nil {
			t.Fatal("expected lock timeout error, got nil")
		}

		expected := "could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it does not write to JSONL", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		beforeStat, err := os.Stat(jsonlPath)
		if err != nil {
			t.Fatalf("failed to stat JSONL before: %v", err)
		}
		beforeContent, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read JSONL before: %v", err)
		}

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ReadTasks()
		if err != nil {
			t.Fatalf("ReadTasks returned error: %v", err)
		}

		afterStat, err := os.Stat(jsonlPath)
		if err != nil {
			t.Fatalf("failed to stat JSONL after: %v", err)
		}
		afterContent, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read JSONL after: %v", err)
		}

		if !beforeStat.ModTime().Equal(afterStat.ModTime()) {
			t.Errorf("JSONL mod time changed: before=%v, after=%v", beforeStat.ModTime(), afterStat.ModTime())
		}
		if string(beforeContent) != string(afterContent) {
			t.Errorf("JSONL content changed after ReadTasks")
		}
	})
}

func TestStoreStaleCacheRebuild(t *testing.T) {
	t.Run("it rebuilds stale cache during write before applying mutation", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Task A",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// First, build the cache.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}

		// Externally modify JSONL (simulating git pull).
		modifiedTasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Task A Modified",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-cccccc",
				Title:    "Task C External",
				Status:   task.StatusOpen,
				Priority: 3,
				Created:  created,
				Updated:  created,
			},
		}
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := WriteJSONL(jsonlPath, modifiedTasks); err != nil {
			t.Fatalf("failed to write modified JSONL: %v", err)
		}

		// Mutate should see the externally modified tasks (cache rebuilt).
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			if len(tasks) != 2 {
				t.Errorf("expected 2 tasks (after external modification), got %d", len(tasks))
			}
			// Verify it sees the modified title.
			if tasks[0].Title != "Task A Modified" {
				t.Errorf("task[0].Title = %q, want %q", tasks[0].Title, "Task A Modified")
			}
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate returned error: %v", err)
		}
	})

	t.Run("it rebuilds stale cache during read before running query", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Task A",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// First, build the cache.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}

		// Externally modify JSONL.
		modifiedTasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Task A Updated",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-dddddd",
				Title:    "Task D External",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := WriteJSONL(jsonlPath, modifiedTasks); err != nil {
			t.Fatalf("failed to write modified JSONL: %v", err)
		}

		// Query should see updated data.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
				return err
			}
			if count != 2 {
				t.Errorf("expected 2 tasks after stale rebuild, got %d", count)
			}

			var title string
			if err := db.QueryRow("SELECT title FROM tasks WHERE id = ?", "tick-aaaaaa").Scan(&title); err != nil {
				return err
			}
			if title != "Task A Updated" {
				t.Errorf("title = %q, want %q", title, "Task A Updated")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})
}

func TestStoreSchemaVersionCheck(t *testing.T) {
	t.Run("it deletes and rebuilds cache when schema_version is wrong", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}

		// Prime the cache.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}
		store.Close()

		// Directly update schema_version to a wrong value using raw SQL.
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache for tampering: %v", err)
		}
		_, err = db.Exec("UPDATE metadata SET value = '999' WHERE key = 'schema_version'")
		if err != nil {
			t.Fatalf("failed to update schema_version: %v", err)
		}
		db.Close()

		// Open a new store and trigger Query. The version mismatch should
		// cause delete+rebuild.
		store2, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store2.Close()

		err = store2.Query(func(db *sql.DB) error {
			// Task data should be correct after rebuild.
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("expected 1 task after version-triggered rebuild, got %d", count)
			}

			var title string
			if qErr := db.QueryRow("SELECT title FROM tasks WHERE id = ?", "tick-aaaaaa").Scan(&title); qErr != nil {
				return qErr
			}
			if title != "Task A" {
				t.Errorf("title = %q, want %q", title, "Task A")
			}

			// schema_version should now equal CurrentSchemaVersion.
			var verStr string
			if qErr := db.QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&verStr); qErr != nil {
				return fmt.Errorf("schema_version not found after rebuild: %w", qErr)
			}
			if verStr != fmt.Sprintf("%d", CurrentSchemaVersion()) {
				t.Errorf("schema_version = %q, want %q", verStr, fmt.Sprintf("%d", CurrentSchemaVersion()))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})

	t.Run("it deletes and rebuilds cache when schema_version row is missing", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}

		// Prime the cache.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}
		store.Close()

		// Directly delete the schema_version row (simulates pre-versioning cache).
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache for tampering: %v", err)
		}
		_, err = db.Exec("DELETE FROM metadata WHERE key = 'schema_version'")
		if err != nil {
			t.Fatalf("failed to delete schema_version: %v", err)
		}
		db.Close()

		// Open a new store and trigger Query. The missing version should
		// cause delete+rebuild.
		store2, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store2.Close()

		err = store2.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("expected 1 task after missing-version rebuild, got %d", count)
			}

			// schema_version should now exist and equal CurrentSchemaVersion.
			var verStr string
			if qErr := db.QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&verStr); qErr != nil {
				return fmt.Errorf("schema_version not found after rebuild: %w", qErr)
			}
			if verStr != fmt.Sprintf("%d", CurrentSchemaVersion()) {
				t.Errorf("schema_version = %q, want %q", verStr, fmt.Sprintf("%d", CurrentSchemaVersion()))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})

	t.Run("it preserves cache when schema_version matches", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		var logged []string
		store, err := NewStore(tickDir, WithVerbose(func(msg string) {
			logged = append(logged, msg)
		}))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// Prime the cache (first query triggers build).
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}

		// Clear logs from the priming query.
		logged = nil

		// Second query on the same store — cache is fresh, version matches.
		err = store.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("expected 1 task, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("second Query returned error: %v", err)
		}

		// Verify no "schema version mismatch" message was logged.
		for _, msg := range logged {
			if msg == "schema version mismatch" || containsSubstring(msg, "schema version mismatch") {
				t.Errorf("unexpected schema version mismatch log: %q", msg)
			}
		}
	})

	t.Run("it handles SchemaVersion query error gracefully", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}

		// Prime the cache.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}
		store.Close()

		// Corrupt the metadata table so SchemaVersion() returns an error.
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache for corruption: %v", err)
		}
		_, err = db.Exec("DROP TABLE metadata; CREATE TABLE metadata (broken INTEGER)")
		if err != nil {
			t.Fatalf("failed to corrupt metadata: %v", err)
		}
		db.Close()

		// Open a new store and trigger Query. The SchemaVersion error should
		// cause delete+rebuild recovery.
		store2, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store2.Close()

		err = store2.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("expected 1 task after schema error recovery, got %d", count)
			}

			// schema_version should be correct after recovery.
			var verStr string
			if qErr := db.QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&verStr); qErr != nil {
				return fmt.Errorf("schema_version not found after recovery: %w", qErr)
			}
			if verStr != fmt.Sprintf("%d", CurrentSchemaVersion()) {
				t.Errorf("schema_version = %q, want %q", verStr, fmt.Sprintf("%d", CurrentSchemaVersion()))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})
}

func TestStoreSchemaVersionUpgrade(t *testing.T) {
	t.Run("it recovers from pre-versioning cache with missing column and serves correct query results", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Task A",
				Status:   task.StatusOpen,
				Priority: 2,
				Type:     "feature",
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}

		// Prime the cache via Query so cache.db is populated.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}
		store.Close()

		// Tamper with cache.db to simulate a pre-versioning, old-schema cache:
		// 1. Remove the schema_version row from metadata
		// 2. Drop the tasks table and recreate it WITHOUT the type column
		// 3. Re-insert task data without the type field
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache for tampering: %v", err)
		}

		// Remove schema_version.
		if _, err := db.Exec("DELETE FROM metadata WHERE key = 'schema_version'"); err != nil {
			t.Fatalf("failed to delete schema_version: %v", err)
		}

		// Recreate tasks table without the type column.
		if _, err := db.Exec("DROP TABLE tasks"); err != nil {
			t.Fatalf("failed to drop tasks table: %v", err)
		}
		_, err = db.Exec(`CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'open',
			priority INTEGER NOT NULL DEFAULT 2,
			description TEXT,
			parent TEXT,
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			closed TEXT
		)`)
		if err != nil {
			t.Fatalf("failed to recreate tasks table without type column: %v", err)
		}
		// Re-insert task data without type.
		_, err = db.Exec(
			"INSERT INTO tasks (id, title, status, priority, created, updated) VALUES (?, ?, ?, ?, ?, ?)",
			"tick-aaaaaa", "Task A", "open", 2,
			"2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z",
		)
		if err != nil {
			t.Fatalf("failed to insert task without type: %v", err)
		}
		db.Close()

		// Reopen the store. Query with a SELECT that references t.type --
		// this is the exact pattern that previously failed with "no such column: t.type".
		store2, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store2.Close()

		err = store2.Query(func(db *sql.DB) error {
			var id, status, title string
			var typeVal sql.NullString
			var priority int
			qErr := db.QueryRow(
				"SELECT t.id, t.status, t.priority, t.title, t.type FROM tasks t WHERE t.id = ?",
				"tick-aaaaaa",
			).Scan(&id, &status, &priority, &title, &typeVal)
			if qErr != nil {
				return fmt.Errorf("query referencing t.type failed: %w", qErr)
			}
			if id != "tick-aaaaaa" {
				t.Errorf("id = %q, want %q", id, "tick-aaaaaa")
			}
			if title != "Task A" {
				t.Errorf("title = %q, want %q", title, "Task A")
			}
			if status != "open" {
				t.Errorf("status = %q, want %q", status, "open")
			}
			if priority != 2 {
				t.Errorf("priority = %d, want %d", priority, 2)
			}
			if !typeVal.Valid || typeVal.String != "feature" {
				t.Errorf("type = %v, want %q", typeVal, "feature")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
	})

	t.Run("it handles sequential queries after version-triggered rebuild without re-triggering rebuild", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-bbbbbb",
				Title:    "Task B",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}
		tickDir := setupTickDirWithTasks(t, tasks)

		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}

		// Prime the cache.
		err = store.Query(func(db *sql.DB) error { return nil })
		if err != nil {
			t.Fatalf("initial Query returned error: %v", err)
		}
		store.Close()

		// Tamper: set schema_version to 0 (simulating missing/pre-versioning).
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache for tampering: %v", err)
		}
		_, err = db.Exec("UPDATE metadata SET value = '0' WHERE key = 'schema_version'")
		if err != nil {
			t.Fatalf("failed to update schema_version: %v", err)
		}
		db.Close()

		// Reopen with verbose logging to capture messages.
		var logged []string
		store2, err := NewStore(tickDir, WithVerbose(func(msg string) {
			logged = append(logged, msg)
		}))
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store2.Close()

		// First query: should trigger version-mismatch rebuild.
		err = store2.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("first query: expected 1 task, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("first Query returned error: %v", err)
		}

		// Verify first query logged version mismatch and rebuild.
		foundMismatch := false
		foundRebuild := false
		for _, msg := range logged {
			if containsSubstring(msg, "schema version mismatch") {
				foundMismatch = true
			}
			if containsSubstring(msg, "rebuilding cache") {
				foundRebuild = true
			}
		}
		if !foundMismatch {
			t.Errorf("first query did not log schema version mismatch; logs: %v", logged)
		}
		if !foundRebuild {
			t.Errorf("first query did not log rebuilding cache; logs: %v", logged)
		}

		// Clear logs before second query.
		logged = nil

		// Second query: should NOT trigger rebuild — cache is now fresh.
		err = store2.Query(func(db *sql.DB) error {
			var count int
			if qErr := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); qErr != nil {
				return qErr
			}
			if count != 1 {
				t.Errorf("second query: expected 1 task, got %d", count)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("second Query returned error: %v", err)
		}

		// Verify second query logged "cache is fresh" and no mismatch/rebuild.
		foundFresh := false
		for _, msg := range logged {
			if containsSubstring(msg, "cache is fresh") {
				foundFresh = true
			}
			if containsSubstring(msg, "schema version mismatch") {
				t.Errorf("second query unexpectedly logged schema version mismatch: %q", msg)
			}
			if containsSubstring(msg, "rebuilding cache") {
				t.Errorf("second query unexpectedly logged rebuilding cache: %q", msg)
			}
		}
		if !foundFresh {
			t.Errorf("second query did not log cache is fresh; logs: %v", logged)
		}
	})
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && strings.Contains(s, substr))
}
