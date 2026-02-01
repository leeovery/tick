package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/task"
)

func setupTickDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".tick")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create .tick dir: %v", err)
	}
	// Create empty tasks.jsonl
	if err := os.WriteFile(filepath.Join(dir, "tasks.jsonl"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return dir
}

func TestNewStore(t *testing.T) {
	t.Run("opens store with valid tick directory", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()
	})

	t.Run("errors if tick directory does not exist", func(t *testing.T) {
		_, err := NewStore("/nonexistent/.tick")
		if err == nil {
			t.Fatal("expected error for nonexistent directory")
		}
	})

	t.Run("errors if tasks.jsonl does not exist", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		os.MkdirAll(tickDir, 0755)
		// Don't create tasks.jsonl
		_, err := NewStore(tickDir)
		if err == nil {
			t.Fatal("expected error for missing tasks.jsonl")
		}
	})
}

func TestStoreMutate(t *testing.T) {
	t.Run("executes full write flow", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		// Create a task via mutation
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			newTask := task.NewTask("tick-abc123", "Test task", 2)
			return append(tasks, newTask), nil
		})
		if err != nil {
			t.Fatalf("Mutate() error: %v", err)
		}

		// Verify via query
		var count int
		err = store.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		})
		if err != nil {
			t.Fatalf("Query() error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task, got %d", count)
		}

		// Verify JSONL was written
		content, err := os.ReadFile(filepath.Join(dir, "tasks.jsonl"))
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}
		if len(content) == 0 {
			t.Error("tasks.jsonl should not be empty after mutation")
		}
	})

	t.Run("releases lock on mutation function error", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		// Mutation that errors
		mutErr := store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return nil, os.ErrNotExist
		})
		if mutErr == nil {
			t.Fatal("expected error from mutation")
		}

		// Lock should be released — another operation should succeed
		err = store.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("Query after failed Mutate should succeed: %v", err)
		}
	})

	t.Run("rebuilds stale cache before applying mutation", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		// Write JSONL directly (bypassing store) to make cache stale
		jsonlPath := filepath.Join(dir, "tasks.jsonl")
		content := `{"id":"tick-ext111","title":"External","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		os.WriteFile(jsonlPath, []byte(content+"\n"), 0644)

		// Mutate should see the external task
		var sawExternal bool
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			for _, t := range tasks {
				if t.ID == "tick-ext111" {
					sawExternal = true
				}
			}
			return tasks, nil
		})
		if err != nil {
			t.Fatalf("Mutate() error: %v", err)
		}
		if !sawExternal {
			t.Error("mutation should see externally-added task after cache rebuild")
		}
	})
}

func TestStoreQuery(t *testing.T) {
	t.Run("executes full read flow", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		// Add a task first
		store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return append(tasks, task.NewTask("tick-abc123", "Test", 2)), nil
		})

		// Query for it
		var title string
		err = store.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT title FROM tasks WHERE id=?", "tick-abc123").Scan(&title)
		})
		if err != nil {
			t.Fatalf("Query() error: %v", err)
		}
		if title != "Test" {
			t.Errorf("title = %q, want %q", title, "Test")
		}
	})

	t.Run("releases lock on query function error", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		// Query that errors
		store.Query(func(db *sql.DB) error {
			return os.ErrNotExist
		})

		// Lock should be released
		err = store.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("Query after failed Query should succeed: %v", err)
		}
	})

	t.Run("rebuilds stale cache before running query", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		// Write JSONL directly
		jsonlPath := filepath.Join(dir, "tasks.jsonl")
		content := `{"id":"tick-ext111","title":"External","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		os.WriteFile(jsonlPath, []byte(content+"\n"), 0644)

		var title string
		err = store.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT title FROM tasks WHERE id=?", "tick-ext111").Scan(&title)
		})
		if err != nil {
			t.Fatalf("Query() error: %v", err)
		}
		if title != "External" {
			t.Errorf("title = %q, want %q", title, "External")
		}
	})
}

func TestStoreLocking(t *testing.T) {
	t.Run("allows concurrent shared locks", func(t *testing.T) {
		dir := setupTickDir(t)
		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		var wg sync.WaitGroup
		var concurrentReads int64
		const readers = 5

		for i := 0; i < readers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				store.Query(func(db *sql.DB) error {
					atomic.AddInt64(&concurrentReads, 1)
					time.Sleep(50 * time.Millisecond) // Hold lock briefly
					return nil
				})
			}()
		}

		wg.Wait()
		if concurrentReads != readers {
			t.Errorf("expected %d concurrent reads, got %d", readers, concurrentReads)
		}
	})

	t.Run("surfaces correct error message on lock timeout", func(t *testing.T) {
		dir := setupTickDir(t)
		lockPath := filepath.Join(dir, "lock")

		// Hold exclusive lock externally
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer fl.Unlock()

		store, err := NewStore(dir)
		if err != nil {
			t.Fatalf("NewStore() error: %v", err)
		}
		defer store.Close()

		// Override timeout for test
		store.lockTimeout = 100 * time.Millisecond

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err == nil {
			t.Fatal("expected lock timeout error")
		}
		want := "could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})
}
