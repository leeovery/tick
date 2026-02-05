package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leeovery/tick/internal/task"
	_ "github.com/mattn/go-sqlite3"
)

func TestCacheSchema(t *testing.T) {
	t.Run("it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		// Verify tasks table exists with correct columns
		rows, err := cache.db.Query(`PRAGMA table_info(tasks)`)
		if err != nil {
			t.Fatalf("failed to query tasks table info: %v", err)
		}
		defer rows.Close()

		expectedTaskColumns := map[string]string{
			"id":          "TEXT",
			"title":       "TEXT",
			"status":      "TEXT",
			"priority":    "INTEGER",
			"description": "TEXT",
			"parent":      "TEXT",
			"created":     "TEXT",
			"updated":     "TEXT",
			"closed":      "TEXT",
		}
		taskColumns := make(map[string]string)
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan row: %v", err)
			}
			taskColumns[name] = ctype
		}

		for col, typ := range expectedTaskColumns {
			if taskColumns[col] != typ {
				t.Errorf("tasks.%s type = %q, want %q", col, taskColumns[col], typ)
			}
		}

		// Verify dependencies table exists with correct columns
		rows, err = cache.db.Query(`PRAGMA table_info(dependencies)`)
		if err != nil {
			t.Fatalf("failed to query dependencies table info: %v", err)
		}
		defer rows.Close()

		expectedDepColumns := map[string]string{
			"task_id":    "TEXT",
			"blocked_by": "TEXT",
		}
		depColumns := make(map[string]string)
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan row: %v", err)
			}
			depColumns[name] = ctype
		}

		for col, typ := range expectedDepColumns {
			if depColumns[col] != typ {
				t.Errorf("dependencies.%s type = %q, want %q", col, depColumns[col], typ)
			}
		}

		// Verify metadata table exists with correct columns
		rows, err = cache.db.Query(`PRAGMA table_info(metadata)`)
		if err != nil {
			t.Fatalf("failed to query metadata table info: %v", err)
		}
		defer rows.Close()

		expectedMetaColumns := map[string]string{
			"key":   "TEXT",
			"value": "TEXT",
		}
		metaColumns := make(map[string]string)
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan row: %v", err)
			}
			metaColumns[name] = ctype
		}

		for col, typ := range expectedMetaColumns {
			if metaColumns[col] != typ {
				t.Errorf("metadata.%s type = %q, want %q", col, metaColumns[col], typ)
			}
		}

		// Verify indexes exist
		rows, err = cache.db.Query(`SELECT name FROM sqlite_master WHERE type='index' AND sql IS NOT NULL`)
		if err != nil {
			t.Fatalf("failed to query indexes: %v", err)
		}
		defer rows.Close()

		indexes := make(map[string]bool)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatalf("failed to scan index name: %v", err)
			}
			indexes[name] = true
		}

		expectedIndexes := []string{"idx_tasks_status", "idx_tasks_priority", "idx_tasks_parent"}
		for _, idx := range expectedIndexes {
			if !indexes[idx] {
				t.Errorf("missing index: %s", idx)
			}
		}
	})
}

func TestCacheRebuild(t *testing.T) {
	t.Run("it rebuilds cache from parsed tasks — all fields round-trip correctly", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		tasks := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Complete task",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: "Full description with\nmultiple lines",
				BlockedBy:   []string{"tick-x1y2z3", "tick-d4e5f6"},
				Parent:      "tick-parent1",
				Created:     "2026-01-19T10:00:00Z",
				Updated:     "2026-01-19T14:00:00Z",
				Closed:      "2026-01-19T16:00:00Z",
			},
		}

		jsonlContent := []byte(`{"id":"tick-a1b2c3","title":"Complete task"}`)

		err = cache.Rebuild(tasks, jsonlContent)
		if err != nil {
			t.Fatalf("Rebuild failed: %v", err)
		}

		// Query task back
		var id, title, status, description, parent, created, updated, closed string
		var priority int
		err = cache.db.QueryRow(`SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id = ?`, "tick-a1b2c3").Scan(
			&id, &title, &status, &priority, &description, &parent, &created, &updated, &closed,
		)
		if err != nil {
			t.Fatalf("failed to query task: %v", err)
		}

		if id != "tick-a1b2c3" {
			t.Errorf("id = %q, want %q", id, "tick-a1b2c3")
		}
		if title != "Complete task" {
			t.Errorf("title = %q, want %q", title, "Complete task")
		}
		if status != "in_progress" {
			t.Errorf("status = %q, want %q", status, "in_progress")
		}
		if priority != 1 {
			t.Errorf("priority = %d, want %d", priority, 1)
		}
		if description != "Full description with\nmultiple lines" {
			t.Errorf("description = %q, want %q", description, "Full description with\nmultiple lines")
		}
		if parent != "tick-parent1" {
			t.Errorf("parent = %q, want %q", parent, "tick-parent1")
		}
		if created != "2026-01-19T10:00:00Z" {
			t.Errorf("created = %q, want %q", created, "2026-01-19T10:00:00Z")
		}
		if updated != "2026-01-19T14:00:00Z" {
			t.Errorf("updated = %q, want %q", updated, "2026-01-19T14:00:00Z")
		}
		if closed != "2026-01-19T16:00:00Z" {
			t.Errorf("closed = %q, want %q", closed, "2026-01-19T16:00:00Z")
		}
	})

	t.Run("it normalizes blocked_by array into dependencies table rows", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		tasks := []task.Task{
			{
				ID:        "tick-a1b2c3",
				Title:     "Task with dependencies",
				Status:    task.StatusOpen,
				Priority:  2,
				BlockedBy: []string{"tick-x1y2z3", "tick-d4e5f6", "tick-g7h8i9"},
				Created:   "2026-01-19T10:00:00Z",
				Updated:   "2026-01-19T10:00:00Z",
			},
		}

		err = cache.Rebuild(tasks, []byte(`{}`))
		if err != nil {
			t.Fatalf("Rebuild failed: %v", err)
		}

		// Query dependencies
		rows, err := cache.db.Query(`SELECT task_id, blocked_by FROM dependencies WHERE task_id = ? ORDER BY blocked_by`, "tick-a1b2c3")
		if err != nil {
			t.Fatalf("failed to query dependencies: %v", err)
		}
		defer rows.Close()

		var deps []string
		for rows.Next() {
			var taskID, blockedBy string
			if err := rows.Scan(&taskID, &blockedBy); err != nil {
				t.Fatalf("failed to scan row: %v", err)
			}
			if taskID != "tick-a1b2c3" {
				t.Errorf("task_id = %q, want %q", taskID, "tick-a1b2c3")
			}
			deps = append(deps, blockedBy)
		}

		if len(deps) != 3 {
			t.Fatalf("expected 3 dependency rows, got %d", len(deps))
		}

		expected := []string{"tick-d4e5f6", "tick-g7h8i9", "tick-x1y2z3"}
		for i, dep := range deps {
			if dep != expected[i] {
				t.Errorf("deps[%d] = %q, want %q", i, dep, expected[i])
			}
		}
	})

	t.Run("it stores JSONL content hash in metadata table after rebuild", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		jsonlContent := []byte(`{"id":"tick-a1b2c3","title":"Test task"}`)

		err = cache.Rebuild([]task.Task{}, jsonlContent)
		if err != nil {
			t.Fatalf("Rebuild failed: %v", err)
		}

		// Query hash from metadata
		var storedHash string
		err = cache.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, "jsonl_hash").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to query hash: %v", err)
		}

		// Compute expected hash (SHA256 of the jsonlContent bytes)
		expectedHash := "6a0a4619eb96b36b6b54a959f2aa7e3c470b77ca6702e66e1d6fd5c25bea4658"
		if storedHash != expectedHash {
			t.Errorf("hash = %q, want %q", storedHash, expectedHash)
		}
	})
}

func TestCacheFreshness(t *testing.T) {
	t.Run("it detects fresh cache (hash matches) and skips rebuild", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		jsonlContent := []byte(`{"id":"tick-a1b2c3","title":"Test task"}`)

		// Initial rebuild
		err = cache.Rebuild([]task.Task{}, jsonlContent)
		if err != nil {
			t.Fatalf("Rebuild failed: %v", err)
		}

		// Check freshness with same content
		fresh, err := cache.IsFresh(jsonlContent)
		if err != nil {
			t.Fatalf("IsFresh failed: %v", err)
		}

		if !fresh {
			t.Error("expected cache to be fresh")
		}
	})

	t.Run("it detects stale cache (hash mismatch) and triggers rebuild", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		jsonlContent := []byte(`{"id":"tick-a1b2c3","title":"Test task"}`)

		// Initial rebuild
		err = cache.Rebuild([]task.Task{}, jsonlContent)
		if err != nil {
			t.Fatalf("Rebuild failed: %v", err)
		}

		// Check freshness with different content
		newContent := []byte(`{"id":"tick-a1b2c3","title":"Modified task"}`)
		fresh, err := cache.IsFresh(newContent)
		if err != nil {
			t.Fatalf("IsFresh failed: %v", err)
		}

		if fresh {
			t.Error("expected cache to be stale")
		}
	})

	t.Run("it treats missing hash as stale", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		// No rebuild done, so no hash in metadata
		fresh, err := cache.IsFresh([]byte(`{}`))
		if err != nil {
			t.Fatalf("IsFresh failed: %v", err)
		}

		if fresh {
			t.Error("expected cache to be stale when hash is missing")
		}
	})
}

func TestEnsureFresh(t *testing.T) {
	t.Run("it rebuilds from scratch when cache.db is missing", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
			},
		}
		jsonlContent := []byte(`{"id":"tick-a1b2c3"}`)

		// Cache doesn't exist yet
		cache, err := EnsureFresh(cachePath, tasks, jsonlContent)
		if err != nil {
			t.Fatalf("EnsureFresh failed: %v", err)
		}
		defer cache.Close()

		// Verify task was inserted
		var count int
		err = cache.db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task, got %d", count)
		}
	})

	t.Run("it skips rebuild when cache is fresh", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
			},
		}
		jsonlContent := []byte(`{"id":"tick-a1b2c3"}`)

		// First call creates cache
		cache1, err := EnsureFresh(cachePath, tasks, jsonlContent)
		if err != nil {
			t.Fatalf("EnsureFresh failed: %v", err)
		}
		cache1.Close()

		// Insert a marker to detect if rebuild happens
		cache2, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		_, err = cache2.db.Exec(`INSERT INTO metadata (key, value) VALUES ('marker', 'test')`)
		if err != nil {
			t.Fatalf("failed to insert marker: %v", err)
		}
		cache2.Close()

		// Second call with same content should not rebuild
		cache3, err := EnsureFresh(cachePath, tasks, jsonlContent)
		if err != nil {
			t.Fatalf("EnsureFresh failed: %v", err)
		}
		defer cache3.Close()

		// Marker should still exist (not cleared by rebuild)
		var marker string
		err = cache3.db.QueryRow(`SELECT value FROM metadata WHERE key = 'marker'`).Scan(&marker)
		if err != nil {
			t.Fatalf("marker was deleted, rebuild happened when it shouldn't have: %v", err)
		}
		if marker != "test" {
			t.Errorf("marker = %q, want %q", marker, "test")
		}
	})

	t.Run("it triggers rebuild when cache is stale", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
			},
		}
		jsonlContent := []byte(`{"id":"tick-a1b2c3"}`)

		// First call creates cache
		cache1, err := EnsureFresh(cachePath, tasks, jsonlContent)
		if err != nil {
			t.Fatalf("EnsureFresh failed: %v", err)
		}
		cache1.Close()

		// Second call with different content should trigger rebuild
		newTasks := []task.Task{
			{
				ID:       "tick-d4e5f6",
				Title:    "New task",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  "2026-01-19T11:00:00Z",
				Updated:  "2026-01-19T11:00:00Z",
			},
		}
		newContent := []byte(`{"id":"tick-d4e5f6"}`)

		cache2, err := EnsureFresh(cachePath, newTasks, newContent)
		if err != nil {
			t.Fatalf("EnsureFresh failed: %v", err)
		}
		defer cache2.Close()

		// Old task should be gone, new task should exist
		var count int
		err = cache2.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id = 'tick-a1b2c3'`).Scan(&count)
		if err != nil {
			t.Fatalf("failed to count old tasks: %v", err)
		}
		if count != 0 {
			t.Error("old task should have been removed")
		}

		err = cache2.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id = 'tick-d4e5f6'`).Scan(&count)
		if err != nil {
			t.Fatalf("failed to count new tasks: %v", err)
		}
		if count != 1 {
			t.Error("new task should exist")
		}
	})

	t.Run("it deletes and recreates cache.db when corrupted", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick dir: %v", err)
		}
		cachePath := filepath.Join(tickDir, "cache.db")

		// Write garbage to cache.db to corrupt it
		if err := os.WriteFile(cachePath, []byte("not a valid sqlite database"), 0644); err != nil {
			t.Fatalf("failed to write corrupt cache: %v", err)
		}

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
			},
		}
		jsonlContent := []byte(`{"id":"tick-a1b2c3"}`)

		// EnsureFresh should handle corruption and rebuild
		cache, err := EnsureFresh(cachePath, tasks, jsonlContent)
		if err != nil {
			t.Fatalf("EnsureFresh failed on corrupted cache: %v", err)
		}
		defer cache.Close()

		// Verify task was inserted
		var count int
		err = cache.db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task, got %d", count)
		}
	})
}

func TestCacheEmptyTasks(t *testing.T) {
	t.Run("it handles empty task list (zero rows, hash still stored)", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		// Empty content represents empty JSONL file
		jsonlContent := []byte("")

		err = cache.Rebuild([]task.Task{}, jsonlContent)
		if err != nil {
			t.Fatalf("Rebuild failed: %v", err)
		}

		// Verify zero tasks
		var count int
		err = cache.db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 tasks, got %d", count)
		}

		// Verify zero dependencies
		err = cache.db.QueryRow(`SELECT COUNT(*) FROM dependencies`).Scan(&count)
		if err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 dependencies, got %d", count)
		}

		// Verify hash is still stored
		var hash string
		err = cache.db.QueryRow(`SELECT value FROM metadata WHERE key = 'jsonl_hash'`).Scan(&hash)
		if err != nil {
			t.Fatalf("hash should be stored even for empty content: %v", err)
		}
		if hash == "" {
			t.Error("hash should not be empty")
		}
	})
}

func TestCacheReplaceAllData(t *testing.T) {
	t.Run("it replaces all existing data on rebuild (no stale rows)", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		// Initial rebuild with 3 tasks
		initialTasks := []task.Task{
			{ID: "tick-a1b2c3", Title: "Task 1", Status: task.StatusOpen, Priority: 2, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z"},
			{ID: "tick-d4e5f6", Title: "Task 2", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-a1b2c3"}, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z"},
			{ID: "tick-g7h8i9", Title: "Task 3", Status: task.StatusOpen, Priority: 0, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z"},
		}
		err = cache.Rebuild(initialTasks, []byte("initial"))
		if err != nil {
			t.Fatalf("initial Rebuild failed: %v", err)
		}

		// Verify initial state
		var taskCount, depCount int
		cache.db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&taskCount)
		cache.db.QueryRow(`SELECT COUNT(*) FROM dependencies`).Scan(&depCount)
		if taskCount != 3 {
			t.Fatalf("expected 3 initial tasks, got %d", taskCount)
		}
		if depCount != 1 {
			t.Fatalf("expected 1 initial dependency, got %d", depCount)
		}

		// Rebuild with only 1 task (no dependencies)
		newTasks := []task.Task{
			{ID: "tick-newone", Title: "New task", Status: task.StatusDone, Priority: 4, Created: "2026-01-19T11:00:00Z", Updated: "2026-01-19T11:00:00Z"},
		}
		err = cache.Rebuild(newTasks, []byte("new"))
		if err != nil {
			t.Fatalf("second Rebuild failed: %v", err)
		}

		// Verify all old data is gone
		cache.db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&taskCount)
		cache.db.QueryRow(`SELECT COUNT(*) FROM dependencies`).Scan(&depCount)
		if taskCount != 1 {
			t.Errorf("expected 1 task after rebuild, got %d", taskCount)
		}
		if depCount != 0 {
			t.Errorf("expected 0 dependencies after rebuild, got %d", depCount)
		}

		// Verify old task IDs don't exist
		var oldCount int
		cache.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id IN ('tick-a1b2c3', 'tick-d4e5f6', 'tick-g7h8i9')`).Scan(&oldCount)
		if oldCount != 0 {
			t.Error("old task IDs should not exist")
		}

		// Verify new task exists
		var newTitle string
		err = cache.db.QueryRow(`SELECT title FROM tasks WHERE id = 'tick-newone'`).Scan(&newTitle)
		if err != nil {
			t.Fatalf("failed to query new task: %v", err)
		}
		if newTitle != "New task" {
			t.Errorf("title = %q, want %q", newTitle, "New task")
		}
	})
}

func TestCacheTransaction(t *testing.T) {
	t.Run("it rebuilds within a single transaction (all-or-nothing)", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, ".tick", "cache.db")

		cache, err := NewCache(cachePath)
		if err != nil {
			t.Fatalf("NewCache failed: %v", err)
		}
		defer cache.Close()

		// Insert initial data
		initialTasks := []task.Task{
			{ID: "tick-a1b2c3", Title: "Original", Status: task.StatusOpen, Priority: 2, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z"},
		}
		err = cache.Rebuild(initialTasks, []byte("original"))
		if err != nil {
			t.Fatalf("initial Rebuild failed: %v", err)
		}

		// Try to rebuild with a task that will cause a constraint violation
		// (duplicate ID in the same batch would fail)
		badTasks := []task.Task{
			{ID: "tick-newone", Title: "Task 1", Status: task.StatusOpen, Priority: 2, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z"},
			{ID: "tick-newone", Title: "Task 2 (duplicate ID)", Status: task.StatusOpen, Priority: 2, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z"},
		}

		err = cache.Rebuild(badTasks, []byte("bad"))
		if err == nil {
			t.Fatal("expected error for duplicate ID")
		}

		// Original data should still be intact due to transaction rollback
		var title string
		err = cache.db.QueryRow(`SELECT title FROM tasks WHERE id = 'tick-a1b2c3'`).Scan(&title)
		if err != nil {
			t.Fatalf("original task should still exist: %v", err)
		}
		if title != "Original" {
			t.Errorf("title = %q, want %q", title, "Original")
		}

		// New task should not exist
		var count int
		cache.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id = 'tick-newone'`).Scan(&count)
		if count != 0 {
			t.Error("new task should not exist after failed transaction")
		}
	})
}
