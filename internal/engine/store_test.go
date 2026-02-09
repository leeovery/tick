package engine

import (
	"context"
	"database/sql"
	"fmt"
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

// setupTickDir creates a .tick/ directory with an empty tasks.jsonl for testing.
func setupTickDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		t.Fatalf("creating .tick dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tickDir, "tasks.jsonl"), []byte{}, 0644); err != nil {
		t.Fatalf("creating tasks.jsonl: %v", err)
	}
	return tickDir
}

// setupTickDirWithTasks creates a .tick/ directory with pre-populated tasks.jsonl.
func setupTickDirWithTasks(t *testing.T, tasks []task.Task) string {
	t.Helper()
	tickDir := setupTickDir(t)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

	var content []byte
	for _, tk := range tasks {
		data, err := tk.MarshalJSON()
		if err != nil {
			t.Fatalf("marshaling task: %v", err)
		}
		content = append(content, data...)
		content = append(content, '\n')
	}
	if err := os.WriteFile(jsonlPath, content, 0644); err != nil {
		t.Fatalf("writing tasks.jsonl: %v", err)
	}
	return tickDir
}

// sampleTasks returns a set of tasks for testing.
func sampleTasks() []task.Task {
	created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
	return []task.Task{
		{
			ID:       "tick-a1b2c3",
			Title:    "First task",
			Status:   task.StatusOpen,
			Priority: 1,
			Created:  created,
			Updated:  updated,
		},
		{
			ID:       "tick-d4e5f6",
			Title:    "Second task",
			Status:   task.StatusDone,
			Priority: 0,
			Created:  created,
			Updated:  updated,
		},
	}
}

func TestStore(t *testing.T) {
	t.Run("it acquires exclusive lock for write operations", func(t *testing.T) {
		tickDir := setupTickDir(t)
		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)

		// Use a channel to verify we're inside the mutation while checking the lock
		inMutation := make(chan struct{})
		done := make(chan struct{})

		go func() {
			defer close(done)
			err := s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
				close(inMutation)
				// Hold the lock for a bit so we can test it's held
				time.Sleep(100 * time.Millisecond)
				return tasks, nil
			})
			if err != nil {
				t.Errorf("Mutate: %v", err)
			}
		}()

		<-inMutation

		// Try to acquire exclusive lock from outside — should fail since Store holds it
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		locked, _ := fl.TryLockContext(ctx, 10*time.Millisecond)
		if locked {
			_ = fl.Unlock()
			t.Error("expected exclusive lock to be held by Store during Mutate, but was able to acquire it")
		}

		<-done
	})

	t.Run("it acquires shared lock for read operations", func(t *testing.T) {
		tickDir := setupTickDir(t)
		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)

		inQuery := make(chan struct{})
		done := make(chan struct{})

		go func() {
			defer close(done)
			err := s.Query(func(db *sql.DB) error {
				close(inQuery)
				time.Sleep(100 * time.Millisecond)
				return nil
			})
			if err != nil {
				t.Errorf("Query: %v", err)
			}
		}()

		<-inQuery

		// Try to acquire shared lock from outside — should succeed (shared locks are compatible)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		locked, lockErr := fl.TryRLockContext(ctx, 10*time.Millisecond)
		if !locked {
			t.Errorf("expected shared lock to be acquirable during Query, but failed: %v", lockErr)
		} else {
			_ = fl.Unlock()
		}

		// Try to acquire exclusive lock from outside — should fail (shared lock blocks exclusive)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel2()
		locked, _ = fl.TryLockContext(ctx2, 10*time.Millisecond)
		if locked {
			_ = fl.Unlock()
			t.Error("expected exclusive lock to be blocked during Query with shared lock, but acquired it")
		}

		<-done
	})

	t.Run("it returns error after lock timeout", func(t *testing.T) {
		tickDir := setupTickDir(t)

		// Hold an exclusive lock externally
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		if err := fl.Lock(); err != nil {
			t.Fatalf("acquiring external lock: %v", err)
		}
		defer func() { _ = fl.Unlock() }()

		s, err := NewStore(tickDir, WithLockTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
		expected := "Could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it allows concurrent shared locks (multiple readers)", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, sampleTasks())
		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		const numReaders = 5
		var wg sync.WaitGroup
		var activeReaders atomic.Int32
		var maxConcurrent atomic.Int32
		errors := make(chan error, numReaders)

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := s.Query(func(db *sql.DB) error {
					cur := activeReaders.Add(1)
					// Track max concurrent readers
					for {
						old := maxConcurrent.Load()
						if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
							break
						}
					}
					time.Sleep(50 * time.Millisecond)
					activeReaders.Add(-1)
					return nil
				})
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Query error: %v", err)
		}

		if maxConcurrent.Load() < 2 {
			t.Errorf("expected at least 2 concurrent readers, got max %d", maxConcurrent.Load())
		}
	})

	t.Run("it blocks shared lock while exclusive lock is held", func(t *testing.T) {
		tickDir := setupTickDir(t)
		s, err := NewStore(tickDir, WithLockTimeout(200*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		// Hold exclusive lock externally
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		if err := fl.Lock(); err != nil {
			t.Fatalf("acquiring external lock: %v", err)
		}
		defer func() { _ = fl.Unlock() }()

		err = s.Query(func(db *sql.DB) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected timeout error for Query while exclusive lock held, got nil")
		}
		expected := "Could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it blocks exclusive lock while shared lock is held", func(t *testing.T) {
		tickDir := setupTickDir(t)
		s, err := NewStore(tickDir, WithLockTimeout(200*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		// Hold shared lock externally
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		if err := fl.RLock(); err != nil {
			t.Fatalf("acquiring external shared lock: %v", err)
		}
		defer func() { _ = fl.Unlock() }()

		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		if err == nil {
			t.Fatal("expected timeout error for Mutate while shared lock held, got nil")
		}
		expected := "Could not acquire lock on .tick/lock - another process may be using tick"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		newTask := task.Task{
			ID:       "tick-new123",
			Title:    "New task",
			Status:   task.StatusOpen,
			Priority: 2,
			Created:  time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Updated:  time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		}

		err = s.Mutate(func(existing []task.Task) ([]task.Task, error) {
			if len(existing) != 2 {
				return nil, fmt.Errorf("expected 2 tasks, got %d", len(existing))
			}
			return append(existing, newTask), nil
		})
		if err != nil {
			t.Fatalf("Mutate: %v", err)
		}

		// Verify JSONL was written with 3 tasks
		err = s.Query(func(db *sql.DB) error {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
				return fmt.Errorf("counting tasks: %w", err)
			}
			if count != 3 {
				return fmt.Errorf("expected 3 tasks in cache, got %d", count)
			}

			// Verify the new task is in the cache
			var title string
			if err := db.QueryRow("SELECT title FROM tasks WHERE id=?", "tick-new123").Scan(&title); err != nil {
				return fmt.Errorf("querying new task: %w", err)
			}
			if title != "New task" {
				return fmt.Errorf("title = %q, want %q", title, "New task")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Query verification: %v", err)
		}
	})

	t.Run("it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		var taskCount int
		err = s.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
		})
		if err != nil {
			t.Fatalf("Query: %v", err)
		}
		if taskCount != 2 {
			t.Errorf("expected 2 tasks, got %d", taskCount)
		}
	})

	t.Run("it releases lock on mutation function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		mutErr := fmt.Errorf("mutation failed")
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return nil, mutErr
		})
		if err == nil {
			t.Fatal("expected error from Mutate")
		}

		// Lock should be released — try to acquire it
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
		if !locked {
			t.Errorf("lock should be released after mutation error, but could not acquire: %v", err)
		} else {
			_ = fl.Unlock()
		}
	})

	t.Run("it releases lock on query function error (no leak)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		queryErr := fmt.Errorf("query failed")
		err = s.Query(func(db *sql.DB) error {
			return queryErr
		})
		if err == nil {
			t.Fatal("expected error from Query")
		}

		// Lock should be released
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
		if !locked {
			t.Errorf("lock should be released after query error, but could not acquire: %v", err)
		} else {
			_ = fl.Unlock()
		}
	})

	t.Run("it continues when JSONL write succeeds but SQLite update fails", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		newTask := task.Task{
			ID:       "tick-new123",
			Title:    "New task",
			Status:   task.StatusOpen,
			Priority: 2,
			Created:  time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Updated:  time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		}

		// Mutate should succeed even though SQLite update fails.
		// Close the cache DB inside the mutation callback — after the freshness
		// check has passed but before the post-write cache rebuild runs.
		err = s.Mutate(func(existing []task.Task) ([]task.Task, error) {
			// Close the cache DB to force the post-write SQLite update to fail.
			s.cache.Close()
			return append(existing, newTask), nil
		})
		if err != nil {
			t.Fatalf("Mutate should succeed even when SQLite fails, got: %v", err)
		}

		// Verify JSONL was written (re-read directly from file)
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		data, readErr := os.ReadFile(jsonlPath)
		if readErr != nil {
			t.Fatalf("reading tasks.jsonl: %v", readErr)
		}
		if !strings.Contains(string(data), "tick-new123") {
			t.Error("JSONL should contain new task after Mutate, even when SQLite failed")
		}
	})

	t.Run("it rebuilds stale cache during write before applying mutation", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		// Mutate — the cache hasn't been built yet, so this forces a build
		err = s.Mutate(func(existing []task.Task) ([]task.Task, error) {
			// Verify the tasks were read from JSONL and passed to us
			if len(existing) != 2 {
				return nil, fmt.Errorf("expected 2 tasks from stale cache rebuild, got %d", len(existing))
			}
			return existing, nil
		})
		if err != nil {
			t.Fatalf("Mutate: %v", err)
		}
	})

	t.Run("it rebuilds stale cache during read before running query", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		s, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		// Query on fresh store — cache needs rebuild
		var count int
		err = s.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		})
		if err != nil {
			t.Fatalf("Query: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 tasks after cache rebuild, got %d", count)
		}
	})

	t.Run("it surfaces correct error message on lock timeout", func(t *testing.T) {
		tickDir := setupTickDir(t)

		// Hold exclusive lock
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		if err := fl.Lock(); err != nil {
			t.Fatalf("acquiring external lock: %v", err)
		}
		defer func() { _ = fl.Unlock() }()

		s, err := NewStore(tickDir, WithLockTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		// Test Mutate timeout message
		err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			return tasks, nil
		})
		expected := "Could not acquire lock on .tick/lock - another process may be using tick"
		if err == nil || err.Error() != expected {
			t.Errorf("Mutate error = %v, want %q", err, expected)
		}

		// Test Query timeout message
		err = s.Query(func(db *sql.DB) error { return nil })
		if err == nil || err.Error() != expected {
			t.Errorf("Query error = %v, want %q", err, expected)
		}
	})
}
