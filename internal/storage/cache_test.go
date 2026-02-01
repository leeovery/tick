package storage

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func testCachePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "cache.db")
}

func fullTask() task.Task {
	closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
	return task.Task{
		ID:          "tick-a1b2c3",
		Title:       "Full task",
		Status:      task.StatusDone,
		Priority:    1,
		Description: "Detailed description",
		BlockedBy:   []string{"tick-d4e5f6", "tick-g7h8i9"},
		Parent:      "tick-j0k1l2",
		Created:     time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC),
		Updated:     time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC),
		Closed:      &closed,
	}
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func TestNewCache(t *testing.T) {
	t.Run("creates cache.db with correct schema", func(t *testing.T) {
		dbPath := testCachePath(t)

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		// Verify tables exist
		tables := []string{"tasks", "dependencies", "metadata"}
		for _, table := range tables {
			var name string
			err := cache.db.QueryRow(
				"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
			).Scan(&name)
			if err != nil {
				t.Errorf("table %q not found: %v", table, err)
			}
		}

		// Verify indexes exist
		indexes := []string{"idx_tasks_status", "idx_tasks_priority", "idx_tasks_parent"}
		for _, idx := range indexes {
			var name string
			err := cache.db.QueryRow(
				"SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx,
			).Scan(&name)
			if err != nil {
				t.Errorf("index %q not found: %v", idx, err)
			}
		}
	})
}

func TestCacheRebuild(t *testing.T) {
	t.Run("rebuilds cache from parsed tasks with all fields", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		ft := fullTask()
		jsonlContent := []byte(`{"id":"tick-a1b2c3"}`) // simplified for hash
		if err := cache.Rebuild([]task.Task{ft}, jsonlContent); err != nil {
			t.Fatalf("Rebuild() error: %v", err)
		}

		// Verify task fields
		var id, title, status, created, updated string
		var priority int
		var description, parent, closed sql.NullString
		err = cache.db.QueryRow("SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id=?", "tick-a1b2c3").
			Scan(&id, &title, &status, &priority, &description, &parent, &created, &updated, &closed)
		if err != nil {
			t.Fatalf("query task: %v", err)
		}
		if id != "tick-a1b2c3" {
			t.Errorf("id = %q, want %q", id, "tick-a1b2c3")
		}
		if title != "Full task" {
			t.Errorf("title = %q, want %q", title, "Full task")
		}
		if status != "done" {
			t.Errorf("status = %q, want %q", status, "done")
		}
		if priority != 1 {
			t.Errorf("priority = %d, want 1", priority)
		}
		if !description.Valid || description.String != "Detailed description" {
			t.Errorf("description = %v, want %q", description, "Detailed description")
		}
		if !parent.Valid || parent.String != "tick-j0k1l2" {
			t.Errorf("parent = %v, want %q", parent, "tick-j0k1l2")
		}
		if !closed.Valid {
			t.Error("closed should be set")
		}
	})

	t.Run("normalizes blocked_by into dependencies table", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		ft := fullTask()
		if err := cache.Rebuild([]task.Task{ft}, []byte("content")); err != nil {
			t.Fatalf("Rebuild() error: %v", err)
		}

		rows, err := cache.db.Query("SELECT blocked_by FROM dependencies WHERE task_id=? ORDER BY blocked_by", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("query dependencies: %v", err)
		}
		defer rows.Close()

		var deps []string
		for rows.Next() {
			var dep string
			if err := rows.Scan(&dep); err != nil {
				t.Fatalf("scan dependency: %v", err)
			}
			deps = append(deps, dep)
		}
		if len(deps) != 2 || deps[0] != "tick-d4e5f6" || deps[1] != "tick-g7h8i9" {
			t.Errorf("dependencies = %v, want [tick-d4e5f6 tick-g7h8i9]", deps)
		}
	})

	t.Run("stores JSONL content hash in metadata table", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		content := []byte("test content for hashing")
		if err := cache.Rebuild([]task.Task{}, content); err != nil {
			t.Fatalf("Rebuild() error: %v", err)
		}

		var storedHash string
		err = cache.db.QueryRow("SELECT value FROM metadata WHERE key='jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("query hash: %v", err)
		}

		expectedHash := hashBytes(content)
		if storedHash != expectedHash {
			t.Errorf("stored hash = %q, want %q", storedHash, expectedHash)
		}
	})

	t.Run("handles empty task list", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		content := []byte("")
		if err := cache.Rebuild([]task.Task{}, content); err != nil {
			t.Fatalf("Rebuild() error: %v", err)
		}

		var count int
		cache.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if count != 0 {
			t.Errorf("expected 0 tasks, got %d", count)
		}

		// Hash should still be stored
		var storedHash string
		err = cache.db.QueryRow("SELECT value FROM metadata WHERE key='jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("hash should be stored even for empty content: %v", err)
		}
	})

	t.Run("replaces all existing data on rebuild", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		// First rebuild with one task
		t1 := sampleTask("tick-old111", "Old task")
		if err := cache.Rebuild([]task.Task{t1}, []byte("v1")); err != nil {
			t.Fatalf("first Rebuild() error: %v", err)
		}

		// Second rebuild with different task
		t2 := sampleTask("tick-new222", "New task")
		if err := cache.Rebuild([]task.Task{t2}, []byte("v2")); err != nil {
			t.Fatalf("second Rebuild() error: %v", err)
		}

		var count int
		cache.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if count != 1 {
			t.Errorf("expected 1 task after rebuild, got %d", count)
		}

		var id string
		cache.db.QueryRow("SELECT id FROM tasks").Scan(&id)
		if id != "tick-new222" {
			t.Errorf("expected tick-new222, got %q", id)
		}
	})
}

func TestCacheFreshness(t *testing.T) {
	t.Run("detects fresh cache and skips rebuild", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		content := []byte("some content")
		if err := cache.Rebuild([]task.Task{}, content); err != nil {
			t.Fatalf("Rebuild() error: %v", err)
		}

		fresh, err := cache.IsFresh(content)
		if err != nil {
			t.Fatalf("IsFresh() error: %v", err)
		}
		if !fresh {
			t.Error("cache should be fresh after rebuild with same content")
		}
	})

	t.Run("detects stale cache on hash mismatch", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		if err := cache.Rebuild([]task.Task{}, []byte("original")); err != nil {
			t.Fatalf("Rebuild() error: %v", err)
		}

		fresh, err := cache.IsFresh([]byte("modified"))
		if err != nil {
			t.Fatalf("IsFresh() error: %v", err)
		}
		if fresh {
			t.Error("cache should be stale after content change")
		}
	})

	t.Run("treats missing hash as stale", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		// No rebuild yet, so no hash stored
		fresh, err := cache.IsFresh([]byte("anything"))
		if err != nil {
			t.Fatalf("IsFresh() error: %v", err)
		}
		if fresh {
			t.Error("cache with no hash should be stale")
		}
	})
}

func TestEnsureFresh(t *testing.T) {
	t.Run("rebuilds when stale", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		tasks := []task.Task{sampleTask("tick-abc123", "Test task")}
		content := []byte(`{"id":"tick-abc123","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`)

		if err := cache.EnsureFresh(content, tasks); err != nil {
			t.Fatalf("EnsureFresh() error: %v", err)
		}

		// Should now have the task
		var count int
		cache.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if count != 1 {
			t.Errorf("expected 1 task, got %d", count)
		}
	})

	t.Run("skips rebuild when fresh", func(t *testing.T) {
		dbPath := testCachePath(t)
		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		content := []byte("some content")
		tasks := []task.Task{}
		if err := cache.Rebuild(tasks, content); err != nil {
			t.Fatalf("Rebuild() error: %v", err)
		}

		// EnsureFresh with same content should be a no-op
		if err := cache.EnsureFresh(content, tasks); err != nil {
			t.Fatalf("EnsureFresh() error: %v", err)
		}
	})
}

func TestCacheMissingDB(t *testing.T) {
	t.Run("rebuilds from scratch when cache.db is missing", func(t *testing.T) {
		dbPath := testCachePath(t)
		// Don't create the file — NewCache should create it

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() error: %v", err)
		}
		defer cache.Close()

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("cache.db should be created by NewCache")
		}
	})
}

func TestCacheCorrupted(t *testing.T) {
	t.Run("deletes and recreates cache.db when corrupted", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		// Write garbage to simulate corruption
		if err := os.WriteFile(dbPath, []byte("not a sqlite database"), 0644); err != nil {
			t.Fatalf("failed to write corrupted file: %v", err)
		}

		cache, err := NewCacheWithRecovery(dbPath)
		if err != nil {
			t.Fatalf("NewCacheWithRecovery() error: %v", err)
		}
		defer cache.Close()

		// Should be able to rebuild successfully
		tasks := []task.Task{sampleTask("tick-abc123", "Test")}
		if err := cache.Rebuild(tasks, []byte("content")); err != nil {
			t.Fatalf("Rebuild after recovery error: %v", err)
		}

		var count int
		cache.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if count != 1 {
			t.Errorf("expected 1 task, got %d", count)
		}
	})
}
