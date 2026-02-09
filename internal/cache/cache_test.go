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
)

// sampleTasks returns a set of tasks covering all fields for testing.
func sampleTasks() []task.Task {
	created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
	closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

	return []task.Task{
		{
			ID:          "tick-a1b2c3",
			Title:       "First task",
			Status:      task.StatusOpen,
			Priority:    1,
			Description: "A description\nwith newlines",
			BlockedBy:   []string{"tick-d4e5f6", "tick-g7h8i9"},
			Parent:      "tick-parent",
			Created:     created,
			Updated:     updated,
		},
		{
			ID:       "tick-d4e5f6",
			Title:    "Second task",
			Status:   task.StatusDone,
			Priority: 0,
			Created:  created,
			Updated:  updated,
			Closed:   &closed,
		},
		{
			ID:       "tick-g7h8i9",
			Title:    "Third task",
			Status:   task.StatusInProgress,
			Priority: 4,
			Created:  created,
			Updated:  created,
		},
	}
}

// sampleJSONLContent returns raw JSONL bytes corresponding to sampleTasks.
func sampleJSONLContent() []byte {
	return []byte(`{"id":"tick-a1b2c3","title":"First task","status":"open","priority":1,"description":"A description\nwith newlines","blocked_by":["tick-d4e5f6","tick-g7h8i9"],"parent":"tick-parent","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":0,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:00:00Z","closed":"2026-01-19T16:00:00Z"}
{"id":"tick-g7h8i9","title":"Third task","status":"in_progress","priority":4,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`)
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func TestCache(t *testing.T) {
	t.Run("it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		// Verify tables exist
		tables := []string{"tasks", "dependencies", "metadata"}
		for _, tbl := range tables {
			var name string
			err := c.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
			if err != nil {
				t.Errorf("table %q not found: %v", tbl, err)
			}
		}

		// Verify indexes exist
		indexes := []string{"idx_tasks_status", "idx_tasks_priority", "idx_tasks_parent"}
		for _, idx := range indexes {
			var name string
			err := c.db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
			if err != nil {
				t.Errorf("index %q not found: %v", idx, err)
			}
		}

		// Verify tasks table columns
		rows, err := c.db.Query("PRAGMA table_info(tasks)")
		if err != nil {
			t.Fatalf("PRAGMA table_info(tasks): %v", err)
		}
		defer rows.Close()

		expectedCols := map[string]bool{
			"id": false, "title": false, "status": false, "priority": false,
			"description": false, "parent": false, "created": false,
			"updated": false, "closed": false,
		}
		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
				t.Fatalf("scanning column info: %v", err)
			}
			if _, ok := expectedCols[name]; ok {
				expectedCols[name] = true
			}
		}
		for col, found := range expectedCols {
			if !found {
				t.Errorf("expected column %q in tasks table", col)
			}
		}

		// Verify dependencies table has task_id and blocked_by columns
		depRows, err := c.db.Query("PRAGMA table_info(dependencies)")
		if err != nil {
			t.Fatalf("PRAGMA table_info(dependencies): %v", err)
		}
		defer depRows.Close()

		depCols := map[string]bool{"task_id": false, "blocked_by": false}
		for depRows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dflt sql.NullString
			if err := depRows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
				t.Fatalf("scanning column info: %v", err)
			}
			if _, ok := depCols[name]; ok {
				depCols[name] = true
			}
		}
		for col, found := range depCols {
			if !found {
				t.Errorf("expected column %q in dependencies table", col)
			}
		}
	})

	t.Run("it rebuilds cache from parsed tasks - all fields round-trip correctly", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		if err := c.Rebuild(tasks, jsonlData); err != nil {
			t.Fatalf("Rebuild: %v", err)
		}

		// Verify task fields round-trip
		var id, title, status, created, updated string
		var priority int
		var description, parent, closed sql.NullString
		err = c.db.QueryRow("SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id=?", "tick-a1b2c3").
			Scan(&id, &title, &status, &priority, &description, &parent, &created, &updated, &closed)
		if err != nil {
			t.Fatalf("querying task: %v", err)
		}
		if id != "tick-a1b2c3" {
			t.Errorf("id = %q, want %q", id, "tick-a1b2c3")
		}
		if title != "First task" {
			t.Errorf("title = %q, want %q", title, "First task")
		}
		if status != "open" {
			t.Errorf("status = %q, want %q", status, "open")
		}
		if priority != 1 {
			t.Errorf("priority = %d, want %d", priority, 1)
		}
		if !description.Valid || description.String != "A description\nwith newlines" {
			t.Errorf("description = %v, want %q", description, "A description\nwith newlines")
		}
		if !parent.Valid || parent.String != "tick-parent" {
			t.Errorf("parent = %v, want %q", parent, "tick-parent")
		}
		if created != "2026-01-19T10:00:00Z" {
			t.Errorf("created = %q, want %q", created, "2026-01-19T10:00:00Z")
		}
		if updated != "2026-01-19T14:00:00Z" {
			t.Errorf("updated = %q, want %q", updated, "2026-01-19T14:00:00Z")
		}
		if closed.Valid {
			t.Errorf("closed should be null for open task, got %q", closed.String)
		}

		// Check the done task has closed timestamp
		var closedVal sql.NullString
		err = c.db.QueryRow("SELECT closed FROM tasks WHERE id=?", "tick-d4e5f6").Scan(&closedVal)
		if err != nil {
			t.Fatalf("querying closed: %v", err)
		}
		if !closedVal.Valid || closedVal.String != "2026-01-19T16:00:00Z" {
			t.Errorf("closed = %v, want %q", closedVal, "2026-01-19T16:00:00Z")
		}

		// Check total row count
		var count int
		if err := c.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("counting tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}
	})

	t.Run("it normalizes blocked_by array into dependencies table rows", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		if err := c.Rebuild(tasks, jsonlData); err != nil {
			t.Fatalf("Rebuild: %v", err)
		}

		// tick-a1b2c3 is blocked by tick-d4e5f6 and tick-g7h8i9
		rows, err := c.db.Query("SELECT blocked_by FROM dependencies WHERE task_id=? ORDER BY blocked_by", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("querying dependencies: %v", err)
		}
		defer rows.Close()

		var deps []string
		for rows.Next() {
			var dep string
			if err := rows.Scan(&dep); err != nil {
				t.Fatalf("scanning dep: %v", err)
			}
			deps = append(deps, dep)
		}
		if len(deps) != 2 {
			t.Fatalf("expected 2 dependencies, got %d: %v", len(deps), deps)
		}
		if deps[0] != "tick-d4e5f6" || deps[1] != "tick-g7h8i9" {
			t.Errorf("dependencies = %v, want [tick-d4e5f6, tick-g7h8i9]", deps)
		}

		// Tasks without blocked_by should have no dependency rows
		var count int
		err = c.db.QueryRow("SELECT COUNT(*) FROM dependencies WHERE task_id=?", "tick-d4e5f6").Scan(&count)
		if err != nil {
			t.Fatalf("counting deps: %v", err)
		}
		if count != 0 {
			t.Errorf("task with no blocked_by should have 0 dep rows, got %d", count)
		}
	})

	t.Run("it stores JSONL content hash in metadata table after rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		jsonlData := sampleJSONLContent()
		expectedHash := hashBytes(jsonlData)

		if err := c.Rebuild(sampleTasks(), jsonlData); err != nil {
			t.Fatalf("Rebuild: %v", err)
		}

		var storedHash string
		err = c.db.QueryRow("SELECT value FROM metadata WHERE key='jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("querying hash: %v", err)
		}
		if storedHash != expectedHash {
			t.Errorf("stored hash = %q, want %q", storedHash, expectedHash)
		}
	})

	t.Run("it detects fresh cache (hash matches) and skips rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		// Initial rebuild
		if err := c.Rebuild(tasks, jsonlData); err != nil {
			t.Fatalf("Rebuild: %v", err)
		}

		// Check freshness with same data
		fresh, err := c.IsFresh(jsonlData)
		if err != nil {
			t.Fatalf("IsFresh: %v", err)
		}
		if !fresh {
			t.Error("cache should be fresh when hash matches")
		}
	})

	t.Run("it detects stale cache (hash mismatch) and triggers rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		// Initial rebuild
		if err := c.Rebuild(tasks, jsonlData); err != nil {
			t.Fatalf("Rebuild: %v", err)
		}

		// Check freshness with different data
		differentData := []byte(`{"id":"tick-new123","title":"New","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}` + "\n")
		fresh, err := c.IsFresh(differentData)
		if err != nil {
			t.Fatalf("IsFresh: %v", err)
		}
		if fresh {
			t.Error("cache should be stale when hash mismatches")
		}
	})

	t.Run("it rebuilds from scratch when cache.db is missing", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		// EnsureFresh creates the cache from scratch if missing
		c, err := EnsureFresh(dbPath, tasks, jsonlData)
		if err != nil {
			t.Fatalf("EnsureFresh: %v", err)
		}
		defer c.Close()

		// Verify data was inserted
		var count int
		if err := c.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("counting tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}
	})

	t.Run("it deletes and recreates cache.db when corrupted", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		// Write garbage to cache.db to simulate corruption
		if err := os.WriteFile(dbPath, []byte("not a sqlite database"), 0644); err != nil {
			t.Fatalf("writing corrupt file: %v", err)
		}

		tasks := sampleTasks()
		jsonlData := sampleJSONLContent()

		// EnsureFresh should recover from corruption
		c, err := EnsureFresh(dbPath, tasks, jsonlData)
		if err != nil {
			t.Fatalf("EnsureFresh with corrupt db: %v", err)
		}
		defer c.Close()

		// Verify data was inserted correctly after recovery
		var count int
		if err := c.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("counting tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}
	})

	t.Run("it handles empty task list (zero rows, hash still stored)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		emptyJSONL := []byte{}
		expectedHash := hashBytes(emptyJSONL)

		if err := c.Rebuild([]task.Task{}, emptyJSONL); err != nil {
			t.Fatalf("Rebuild: %v", err)
		}

		// Zero task rows
		var count int
		if err := c.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("counting tasks: %v", err)
		}
		if count != 0 {
			t.Errorf("task count = %d, want 0", count)
		}

		// Hash still stored
		var storedHash string
		err = c.db.QueryRow("SELECT value FROM metadata WHERE key='jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("querying hash: %v", err)
		}
		if storedHash != expectedHash {
			t.Errorf("stored hash = %q, want %q", storedHash, expectedHash)
		}
	})

	t.Run("it replaces all existing data on rebuild (no stale rows)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		// First rebuild with sample tasks
		if err := c.Rebuild(sampleTasks(), sampleJSONLContent()); err != nil {
			t.Fatalf("first Rebuild: %v", err)
		}

		var count int
		if err := c.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("counting: %v", err)
		}
		if count != 3 {
			t.Fatalf("expected 3 tasks after first rebuild, got %d", count)
		}

		// Second rebuild with only one task
		now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
		oneTask := []task.Task{
			{
				ID:       "tick-newone",
				Title:    "Only task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}
		newJSONL := []byte(`{"id":"tick-newone","title":"Only task","status":"open","priority":2,"created":"2026-02-01T10:00:00Z","updated":"2026-02-01T10:00:00Z"}` + "\n")

		if err := c.Rebuild(oneTask, newJSONL); err != nil {
			t.Fatalf("second Rebuild: %v", err)
		}

		// Should have only 1 task now, not 4
		if err := c.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("counting: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task after second rebuild, got %d", count)
		}

		// Old dependencies should be gone
		var depCount int
		if err := c.db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&depCount); err != nil {
			t.Fatalf("counting deps: %v", err)
		}
		if depCount != 0 {
			t.Errorf("expected 0 dependencies after second rebuild, got %d", depCount)
		}
	})

	t.Run("it rebuilds within a single transaction (all-or-nothing)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		c, err := New(dbPath)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer c.Close()

		// First rebuild with valid data
		if err := c.Rebuild(sampleTasks(), sampleJSONLContent()); err != nil {
			t.Fatalf("first Rebuild: %v", err)
		}

		// Attempt rebuild with a task that will cause an error (duplicate ID within same batch)
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		badTasks := []task.Task{
			{ID: "tick-aaaaaa", Title: "Good task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaaaaa", Title: "Duplicate ID", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		badJSONL := []byte(`{"id":"tick-aaaaaa","title":"Good task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-aaaaaa","title":"Duplicate ID","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`)

		err = c.Rebuild(badTasks, badJSONL)
		if err == nil {
			t.Fatal("expected error for duplicate task IDs, got nil")
		}

		// Original data should still be intact (transaction rolled back)
		var count int
		if err := c.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("counting tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("original data should be intact after failed rebuild, got %d tasks (want 3)", count)
		}
	})
}
