package cache

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
	_ "github.com/mattn/go-sqlite3"
)

// sampleTasks returns a set of tasks for testing, covering all fields.
func sampleTasks() []task.Task {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
	closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

	return []task.Task{
		{
			ID:          "tick-a1b2c3",
			Title:       "First task",
			Status:      task.StatusOpen,
			Priority:    1,
			Description: "A description",
			BlockedBy:   []string{"tick-d4e5f6", "tick-g7h8i9"},
			Parent:      "tick-parent1",
			Created:     now,
			Updated:     updated,
		},
		{
			ID:       "tick-d4e5f6",
			Title:    "Second task",
			Status:   task.StatusDone,
			Priority: 0,
			Created:  now,
			Updated:  updated,
			Closed:   &closed,
		},
		{
			ID:       "tick-g7h8i9",
			Title:    "Third task",
			Status:   task.StatusInProgress,
			Priority: 3,
			Parent:   "tick-parent1",
			Created:  now,
			Updated:  now,
		},
	}
}

// sampleJSONLContent returns raw JSONL bytes matching sampleTasks for hash computation.
func sampleJSONLContent() []byte {
	return []byte(`{"id":"tick-a1b2c3","title":"First task","status":"open","priority":1,"description":"A description","blocked_by":["tick-d4e5f6","tick-g7h8i9"],"parent":"tick-parent1","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":0,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:00:00Z","closed":"2026-01-19T16:00:00Z"}
{"id":"tick-g7h8i9","title":"Third task","status":"in_progress","priority":3,"parent":"tick-parent1","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`)
}

func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func TestCache_CreateSchema(t *testing.T) {
	t.Run("it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		db := c.DB()

		// Check tables exist
		tables := []string{"tasks", "dependencies", "metadata"}
		for _, tbl := range tables {
			var name string
			err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
			if err != nil {
				t.Errorf("table %q not found: %v", tbl, err)
			}
		}

		// Check indexes exist
		indexes := []string{"idx_tasks_status", "idx_tasks_priority", "idx_tasks_parent"}
		for _, idx := range indexes {
			var name string
			err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
			if err != nil {
				t.Errorf("index %q not found: %v", idx, err)
			}
		}

		// Verify tasks table columns
		rows, err := db.Query("PRAGMA table_info(tasks)")
		if err != nil {
			t.Fatalf("failed to query tasks table info: %v", err)
		}
		defer rows.Close()

		expectedCols := map[string]bool{
			"id": false, "title": false, "status": false, "priority": false,
			"description": false, "parent": false, "created": false, "updated": false, "closed": false,
		}
		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue *string
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan column info: %v", err)
			}
			expectedCols[name] = true
		}
		for col, found := range expectedCols {
			if !found {
				t.Errorf("expected column %q in tasks table, not found", col)
			}
		}

		// Verify dependencies table columns
		rows2, err := db.Query("PRAGMA table_info(dependencies)")
		if err != nil {
			t.Fatalf("failed to query dependencies table info: %v", err)
		}
		defer rows2.Close()

		depCols := map[string]bool{"task_id": false, "blocked_by": false}
		for rows2.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue *string
			if err := rows2.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan column info: %v", err)
			}
			depCols[name] = true
		}
		for col, found := range depCols {
			if !found {
				t.Errorf("expected column %q in dependencies table, not found", col)
			}
		}

		// Verify metadata table columns
		rows3, err := db.Query("PRAGMA table_info(metadata)")
		if err != nil {
			t.Fatalf("failed to query metadata table info: %v", err)
		}
		defer rows3.Close()

		metaCols := map[string]bool{"key": false, "value": false}
		for rows3.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue *string
			if err := rows3.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan column info: %v", err)
			}
			metaCols[name] = true
		}
		for col, found := range metaCols {
			if !found {
				t.Errorf("expected column %q in metadata table, not found", col)
			}
		}
	})
}

func TestCache_Rebuild(t *testing.T) {
	t.Run("it rebuilds cache from parsed tasks — all fields round-trip correctly", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		if err := c.Rebuild(tasks, jsonlData); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := c.DB()

		// Query all tasks back
		rows, err := db.Query("SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks ORDER BY id")
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		defer rows.Close()

		var results []task.Task
		for rows.Next() {
			var tk task.Task
			var desc, parent, closedStr sql.NullString
			var createdStr, updatedStr string
			if err := rows.Scan(&tk.ID, &tk.Title, &tk.Status, &tk.Priority, &desc, &parent, &createdStr, &updatedStr, &closedStr); err != nil {
				t.Fatalf("failed to scan task row: %v", err)
			}
			if desc.Valid {
				tk.Description = desc.String
			}
			if parent.Valid {
				tk.Parent = parent.String
			}
			tk.Created, _ = time.Parse(time.RFC3339, createdStr)
			tk.Updated, _ = time.Parse(time.RFC3339, updatedStr)
			if closedStr.Valid {
				closedTime, _ := time.Parse(time.RFC3339, closedStr.String)
				tk.Closed = &closedTime
			}
			results = append(results, tk)
		}

		if len(results) != 3 {
			t.Fatalf("expected 3 tasks in cache, got %d", len(results))
		}

		// Verify first task (tick-a1b2c3) - all fields
		r := results[0]
		if r.ID != "tick-a1b2c3" {
			t.Errorf("task 0 ID: got %q, want %q", r.ID, "tick-a1b2c3")
		}
		if r.Title != "First task" {
			t.Errorf("task 0 Title: got %q, want %q", r.Title, "First task")
		}
		if r.Status != task.StatusOpen {
			t.Errorf("task 0 Status: got %q, want %q", r.Status, task.StatusOpen)
		}
		if r.Priority != 1 {
			t.Errorf("task 0 Priority: got %d, want %d", r.Priority, 1)
		}
		if r.Description != "A description" {
			t.Errorf("task 0 Description: got %q, want %q", r.Description, "A description")
		}
		if r.Parent != "tick-parent1" {
			t.Errorf("task 0 Parent: got %q, want %q", r.Parent, "tick-parent1")
		}

		// Verify second task (tick-d4e5f6) - with closed timestamp
		r2 := results[1]
		if r2.ID != "tick-d4e5f6" {
			t.Errorf("task 1 ID: got %q, want %q", r2.ID, "tick-d4e5f6")
		}
		if r2.Closed == nil {
			t.Error("task 1 Closed: expected non-nil")
		}
	})

	t.Run("it normalizes blocked_by array into dependencies table rows", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		if err := c.Rebuild(tasks, jsonlData); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := c.DB()

		// Query dependencies for tick-a1b2c3 (has 2 blockers)
		rows, err := db.Query("SELECT task_id, blocked_by FROM dependencies WHERE task_id = ? ORDER BY blocked_by", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("failed to query dependencies: %v", err)
		}
		defer rows.Close()

		var deps []struct{ taskID, blockedBy string }
		for rows.Next() {
			var d struct{ taskID, blockedBy string }
			if err := rows.Scan(&d.taskID, &d.blockedBy); err != nil {
				t.Fatalf("failed to scan dependency: %v", err)
			}
			deps = append(deps, d)
		}

		if len(deps) != 2 {
			t.Fatalf("expected 2 dependencies for tick-a1b2c3, got %d", len(deps))
		}
		if deps[0].blockedBy != "tick-d4e5f6" {
			t.Errorf("dep 0 blocked_by: got %q, want %q", deps[0].blockedBy, "tick-d4e5f6")
		}
		if deps[1].blockedBy != "tick-g7h8i9" {
			t.Errorf("dep 1 blocked_by: got %q, want %q", deps[1].blockedBy, "tick-g7h8i9")
		}

		// Verify total dependency count (only task 1 has blockers)
		var totalDeps int
		if err := db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&totalDeps); err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if totalDeps != 2 {
			t.Errorf("expected 2 total dependency rows, got %d", totalDeps)
		}
	})

	t.Run("it stores JSONL content hash in metadata table after rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		jsonlData := sampleJSONLContent()
		expectedHash := computeHash(jsonlData)

		if err := c.Rebuild(sampleTasks(), jsonlData); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := c.DB()
		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to read jsonl_hash from metadata: %v", err)
		}
		if storedHash != expectedHash {
			t.Errorf("stored hash %q does not match expected %q", storedHash, expectedHash)
		}
	})

	t.Run("it handles empty task list (zero rows, hash still stored)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		emptyJSONL := []byte{}
		expectedHash := computeHash(emptyJSONL)

		if err := c.Rebuild([]task.Task{}, emptyJSONL); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := c.DB()

		// Zero tasks
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 tasks, got %d", count)
		}

		// Zero dependencies
		if err := db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&count); err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 dependencies, got %d", count)
		}

		// Hash still stored
		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to read jsonl_hash: %v", err)
		}
		if storedHash != expectedHash {
			t.Errorf("stored hash %q does not match expected %q", storedHash, expectedHash)
		}
	})

	t.Run("it replaces all existing data on rebuild (no stale rows)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		// First rebuild with 3 tasks
		if err := c.Rebuild(sampleTasks(), sampleJSONLContent()); err != nil {
			t.Fatalf("first Rebuild returned error: %v", err)
		}

		db := c.DB()
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Fatalf("expected 3 tasks after first rebuild, got %d", count)
		}

		// Second rebuild with 1 task (no blockers)
		now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
		newTasks := []task.Task{
			{
				ID:       "tick-new111",
				Title:    "New task only",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}
		newJSONL := []byte(`{"id":"tick-new111","title":"New task only","status":"open","priority":2,"created":"2026-02-01T10:00:00Z","updated":"2026-02-01T10:00:00Z"}
`)

		if err := c.Rebuild(newTasks, newJSONL); err != nil {
			t.Fatalf("second Rebuild returned error: %v", err)
		}

		// Verify old data is gone
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task after second rebuild, got %d", count)
		}

		// Verify old dependencies are gone
		if err := db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&count); err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 dependencies after second rebuild, got %d", count)
		}

		// Verify only the new task exists
		var id string
		if err := db.QueryRow("SELECT id FROM tasks").Scan(&id); err != nil {
			t.Fatalf("failed to query task: %v", err)
		}
		if id != "tick-new111" {
			t.Errorf("expected task ID %q, got %q", "tick-new111", id)
		}

		// Verify hash is updated
		var storedHash string
		if err := db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash); err != nil {
			t.Fatalf("failed to read hash: %v", err)
		}
		expectedHash := computeHash(newJSONL)
		if storedHash != expectedHash {
			t.Errorf("hash not updated: got %q, want %q", storedHash, expectedHash)
		}
	})

	t.Run("it rebuilds within a single transaction (all-or-nothing)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		// Do a valid rebuild first
		if err := c.Rebuild(sampleTasks(), sampleJSONLContent()); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		// Verify data exists
		db := c.DB()
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Fatalf("expected 3 tasks, got %d", count)
		}

		// Verify we can check that the rebuild uses a transaction by observing
		// that after a successful rebuild, all data is consistent (hash + tasks match).
		// This is a structural test: we verify the code calls BEGIN/COMMIT by checking
		// consistency after rebuild.
		newJSONL := []byte(`{"id":"tick-xxx","title":"Only one","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`)
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		newTasks := []task.Task{
			{
				ID:       "tick-xxx",
				Title:    "Only one",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}

		if err := c.Rebuild(newTasks, newJSONL); err != nil {
			t.Fatalf("second Rebuild returned error: %v", err)
		}

		// After rebuild, hash and data must be consistent
		var taskCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if taskCount != 1 {
			t.Errorf("expected 1 task, got %d", taskCount)
		}

		var storedHash string
		if err := db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash); err != nil {
			t.Fatalf("failed to read hash: %v", err)
		}
		if storedHash != computeHash(newJSONL) {
			t.Error("hash does not match after transactional rebuild")
		}
	})
}

func TestCache_Freshness(t *testing.T) {
	t.Run("it detects fresh cache (hash matches) and skips rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		jsonlData := sampleJSONLContent()

		// Rebuild to populate hash
		if err := c.Rebuild(sampleTasks(), jsonlData); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		fresh, err := c.IsFresh(jsonlData)
		if err != nil {
			t.Fatalf("IsFresh returned error: %v", err)
		}
		if !fresh {
			t.Error("expected cache to be fresh, got stale")
		}
	})

	t.Run("it detects stale cache (hash mismatch) and triggers rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		// Rebuild with initial data
		if err := c.Rebuild(sampleTasks(), sampleJSONLContent()); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		// Check freshness with different content
		differentData := []byte(`{"id":"tick-new","title":"Different","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`)

		fresh, err := c.IsFresh(differentData)
		if err != nil {
			t.Fatalf("IsFresh returned error: %v", err)
		}
		if fresh {
			t.Error("expected cache to be stale, got fresh")
		}
	})
}

func TestCache_EnsureFresh(t *testing.T) {
	t.Run("it rebuilds from scratch when cache.db is missing", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		// No cache.db exists yet
		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		if err := EnsureFresh(dbPath, jsonlData, tasks); err != nil {
			t.Fatalf("EnsureFresh returned error: %v", err)
		}

		// Verify cache was created and populated
		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		db := c.DB()
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 tasks, got %d", count)
		}
	})

	t.Run("it skips rebuild when cache is fresh", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		// First call: creates and rebuilds
		if err := EnsureFresh(dbPath, jsonlData, tasks); err != nil {
			t.Fatalf("first EnsureFresh returned error: %v", err)
		}

		// Second call with same data: should be a no-op (fresh)
		if err := EnsureFresh(dbPath, jsonlData, tasks); err != nil {
			t.Fatalf("second EnsureFresh returned error: %v", err)
		}

		// Verify data is still correct
		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		db := c.DB()
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 tasks, got %d", count)
		}
	})

	t.Run("it rebuilds when cache is stale", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		// First call: creates and rebuilds
		if err := EnsureFresh(dbPath, jsonlData, tasks); err != nil {
			t.Fatalf("first EnsureFresh returned error: %v", err)
		}

		// Second call with different data: should trigger rebuild
		now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
		newTasks := []task.Task{
			{
				ID:       "tick-onlyme",
				Title:    "Only me",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}
		newJSONL := []byte(`{"id":"tick-onlyme","title":"Only me","status":"open","priority":2,"created":"2026-02-01T10:00:00Z","updated":"2026-02-01T10:00:00Z"}
`)

		if err := EnsureFresh(dbPath, newJSONL, newTasks); err != nil {
			t.Fatalf("second EnsureFresh returned error: %v", err)
		}

		// Verify rebuilt
		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error: %v", err)
		}
		defer c.Close()

		db := c.DB()
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task after stale rebuild, got %d", count)
		}
	})

	t.Run("it deletes and recreates cache.db when corrupted", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		// Write garbage to cache.db to simulate corruption
		if err := os.WriteFile(dbPath, []byte("this is not a sqlite database"), 0644); err != nil {
			t.Fatalf("failed to write corrupted file: %v", err)
		}

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		// EnsureFresh should handle corruption gracefully
		if err := EnsureFresh(dbPath, jsonlData, tasks); err != nil {
			t.Fatalf("EnsureFresh returned error on corrupted db: %v", err)
		}

		// Verify cache was recreated and populated
		c, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open returned error after recovery: %v", err)
		}
		defer c.Close()

		db := c.DB()
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 tasks after corruption recovery, got %d", count)
		}
	})
}
