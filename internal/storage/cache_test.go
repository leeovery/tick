package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
	_ "github.com/mattn/go-sqlite3"
)

func TestCacheSchema(t *testing.T) {
	t.Run("it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		db := cache.DB()

		// Verify tasks table exists with correct columns
		taskCols := queryColumns(t, db, "tasks")
		expectedTaskCols := map[string]bool{
			"id": true, "title": true, "status": true, "priority": true,
			"description": true, "parent": true, "created": true,
			"updated": true, "closed": true,
		}
		if len(taskCols) != len(expectedTaskCols) {
			t.Errorf("tasks table: expected %d columns, got %d: %v", len(expectedTaskCols), len(taskCols), taskCols)
		}
		for col := range expectedTaskCols {
			if !taskCols[col] {
				t.Errorf("tasks table missing column %q", col)
			}
		}

		// Verify dependencies table exists with correct columns
		depCols := queryColumns(t, db, "dependencies")
		expectedDepCols := map[string]bool{"task_id": true, "blocked_by": true}
		if len(depCols) != len(expectedDepCols) {
			t.Errorf("dependencies table: expected %d columns, got %d: %v", len(expectedDepCols), len(depCols), depCols)
		}
		for col := range expectedDepCols {
			if !depCols[col] {
				t.Errorf("dependencies table missing column %q", col)
			}
		}

		// Verify metadata table exists with correct columns
		metaCols := queryColumns(t, db, "metadata")
		expectedMetaCols := map[string]bool{"key": true, "value": true}
		if len(metaCols) != len(expectedMetaCols) {
			t.Errorf("metadata table: expected %d columns, got %d: %v", len(expectedMetaCols), len(metaCols), metaCols)
		}
		for col := range expectedMetaCols {
			if !metaCols[col] {
				t.Errorf("metadata table missing column %q", col)
			}
		}

		// Verify indexes exist
		indexes := queryIndexes(t, db)
		expectedIndexes := map[string]bool{
			"idx_tasks_status":   true,
			"idx_tasks_priority": true,
			"idx_tasks_parent":   true,
		}
		for idx := range expectedIndexes {
			if !indexes[idx] {
				t.Errorf("missing index %q", idx)
			}
		}
	})
}

// queryColumns returns a set of column names for the given table.
func queryColumns(t *testing.T, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) error: %v", table, err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scanning table_info row: %v", err)
		}
		cols[name] = true
	}
	return cols
}

// queryIndexes returns a set of index names for the database.
func queryIndexes(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		t.Fatalf("querying indexes: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scanning index row: %v", err)
		}
		indexes[name] = true
	}
	return indexes
}

func TestCacheRebuild(t *testing.T) {
	t.Run("it rebuilds cache from parsed tasks — all fields round-trip correctly", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

		tasks := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: "Detailed description",
				BlockedBy:   []string{"tick-x1y2z3"},
				Parent:      "tick-p1a2b3",
				Created:     created,
				Updated:     updated,
				Closed:      &closed,
			},
		}

		rawJSONL := []byte(`{"id":"tick-a1b2c3","title":"Full task"}`)

		if err := cache.Rebuild(tasks, rawJSONL); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()

		// Query the task back and verify all fields.
		var id, title, status, createdStr, updatedStr string
		var priority int
		var description, parent, closedStr sql.NullString
		err = db.QueryRow("SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id = ?", "tick-a1b2c3").
			Scan(&id, &title, &status, &priority, &description, &parent, &createdStr, &updatedStr, &closedStr)
		if err != nil {
			t.Fatalf("querying task: %v", err)
		}

		if id != "tick-a1b2c3" {
			t.Errorf("id = %q, want %q", id, "tick-a1b2c3")
		}
		if title != "Full task" {
			t.Errorf("title = %q, want %q", title, "Full task")
		}
		if status != "in_progress" {
			t.Errorf("status = %q, want %q", status, "in_progress")
		}
		if priority != 1 {
			t.Errorf("priority = %d, want %d", priority, 1)
		}
		if !description.Valid || description.String != "Detailed description" {
			t.Errorf("description = %v, want %q", description, "Detailed description")
		}
		if !parent.Valid || parent.String != "tick-p1a2b3" {
			t.Errorf("parent = %v, want %q", parent, "tick-p1a2b3")
		}
		if createdStr != "2026-01-19T10:00:00Z" {
			t.Errorf("created = %q, want %q", createdStr, "2026-01-19T10:00:00Z")
		}
		if updatedStr != "2026-01-19T14:00:00Z" {
			t.Errorf("updated = %q, want %q", updatedStr, "2026-01-19T14:00:00Z")
		}
		if !closedStr.Valid || closedStr.String != "2026-01-19T16:00:00Z" {
			t.Errorf("closed = %v, want %q", closedStr, "2026-01-19T16:00:00Z")
		}
	})

	t.Run("it normalizes blocked_by array into dependencies table rows", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:        "tick-a1b2c3",
				Title:     "Task with deps",
				Status:    task.StatusOpen,
				Priority:  2,
				BlockedBy: []string{"tick-d4e5f6", "tick-g7h8i9"},
				Created:   created,
				Updated:   created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()
		rows, err := db.Query("SELECT task_id, blocked_by FROM dependencies WHERE task_id = ? ORDER BY blocked_by", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("querying dependencies: %v", err)
		}
		defer rows.Close()

		type dep struct {
			taskID    string
			blockedBy string
		}
		var deps []dep
		for rows.Next() {
			var d dep
			if err := rows.Scan(&d.taskID, &d.blockedBy); err != nil {
				t.Fatalf("scanning dependency row: %v", err)
			}
			deps = append(deps, d)
		}

		if len(deps) != 2 {
			t.Fatalf("expected 2 dependency rows, got %d", len(deps))
		}
		if deps[0].blockedBy != "tick-d4e5f6" {
			t.Errorf("deps[0].blocked_by = %q, want %q", deps[0].blockedBy, "tick-d4e5f6")
		}
		if deps[1].blockedBy != "tick-g7h8i9" {
			t.Errorf("deps[1].blocked_by = %q, want %q", deps[1].blockedBy, "tick-g7h8i9")
		}
	})

	t.Run("it stores JSONL content hash in metadata table after rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		rawJSONL := []byte(`{"id":"tick-a1b2c3","title":"Test task"}`)
		expectedHash := computeTestHash(rawJSONL)

		if err := cache.Rebuild(nil, rawJSONL); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		var storedHash string
		err = cache.DB().QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("querying metadata: %v", err)
		}

		if storedHash != expectedHash {
			t.Errorf("stored hash = %q, want %q", storedHash, expectedHash)
		}
	})
}

func TestCacheFreshness(t *testing.T) {
	t.Run("it detects fresh cache (hash matches) and skips rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		rawJSONL := []byte(`{"id":"tick-a1b2c3","title":"Test task"}`)

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		// Rebuild to set the hash.
		if err := cache.Rebuild(nil, rawJSONL); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		// Check freshness with same content.
		fresh, err := cache.IsFresh(rawJSONL)
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

		originalJSONL := []byte(`{"id":"tick-a1b2c3","title":"Original"}`)
		modifiedJSONL := []byte(`{"id":"tick-a1b2c3","title":"Modified"}`)

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		// Rebuild with original content.
		if err := cache.Rebuild(nil, originalJSONL); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		// Check freshness with modified content.
		fresh, err := cache.IsFresh(modifiedJSONL)
		if err != nil {
			t.Fatalf("IsFresh returned error: %v", err)
		}
		if fresh {
			t.Error("expected cache to be stale, got fresh")
		}
	})
}

func TestCacheEdgeCases(t *testing.T) {
	t.Run("it handles empty task list (zero rows, hash still stored)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		rawJSONL := []byte("")
		expectedHash := computeTestHash(rawJSONL)

		if err := cache.Rebuild(nil, rawJSONL); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()

		// Zero rows in tasks.
		var taskCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount); err != nil {
			t.Fatalf("querying tasks count: %v", err)
		}
		if taskCount != 0 {
			t.Errorf("expected 0 tasks, got %d", taskCount)
		}

		// Zero rows in dependencies.
		var depCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&depCount); err != nil {
			t.Fatalf("querying dependencies count: %v", err)
		}
		if depCount != 0 {
			t.Errorf("expected 0 dependencies, got %d", depCount)
		}

		// Hash still stored.
		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("querying metadata: %v", err)
		}
		if storedHash != expectedHash {
			t.Errorf("stored hash = %q, want %q", storedHash, expectedHash)
		}
	})

	t.Run("it replaces all existing data on rebuild (no stale rows)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// First rebuild with task A.
		tasksV1 := []task.Task{
			{
				ID:        "tick-aaaaaa",
				Title:     "Task A",
				Status:    task.StatusOpen,
				Priority:  2,
				BlockedBy: []string{"tick-bbbbbb"},
				Created:   created,
				Updated:   created,
			},
		}
		if err := cache.Rebuild(tasksV1, []byte("v1")); err != nil {
			t.Fatalf("first Rebuild returned error: %v", err)
		}

		// Second rebuild with task B (task A should be gone).
		tasksV2 := []task.Task{
			{
				ID:       "tick-cccccc",
				Title:    "Task C",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(tasksV2, []byte("v2")); err != nil {
			t.Fatalf("second Rebuild returned error: %v", err)
		}

		db := cache.DB()

		// Only task C should exist.
		var taskCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount); err != nil {
			t.Fatalf("querying tasks count: %v", err)
		}
		if taskCount != 1 {
			t.Errorf("expected 1 task after rebuild, got %d", taskCount)
		}

		var id string
		if err := db.QueryRow("SELECT id FROM tasks").Scan(&id); err != nil {
			t.Fatalf("querying task id: %v", err)
		}
		if id != "tick-cccccc" {
			t.Errorf("task id = %q, want %q", id, "tick-cccccc")
		}

		// No dependency rows should remain.
		var depCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&depCount); err != nil {
			t.Fatalf("querying dependencies count: %v", err)
		}
		if depCount != 0 {
			t.Errorf("expected 0 dependencies after rebuild, got %d", depCount)
		}
	})

	t.Run("it rebuilds within a single transaction (all-or-nothing)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// First, insert valid data.
		validTasks := []task.Task{
			{
				ID:       "tick-valid1",
				Title:    "Valid task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(validTasks, []byte("valid")); err != nil {
			t.Fatalf("valid Rebuild returned error: %v", err)
		}

		// Try to rebuild with duplicate task IDs — should fail due to PRIMARY KEY constraint.
		duplicateTasks := []task.Task{
			{
				ID:       "tick-dup111",
				Title:    "Dup 1",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-dup111",
				Title:    "Dup 2 same ID",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		err = cache.Rebuild(duplicateTasks, []byte("dup"))
		if err == nil {
			t.Fatal("expected error for duplicate task IDs, got nil")
		}

		// The original valid data should still be intact because the failed rebuild was rolled back.
		var count int
		if err := cache.DB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("querying tasks count: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task (original) after failed rebuild, got %d", count)
		}

		var id string
		if err := cache.DB().QueryRow("SELECT id FROM tasks").Scan(&id); err != nil {
			t.Fatalf("querying task id: %v", err)
		}
		if id != "tick-valid1" {
			t.Errorf("task id = %q, want %q (original should be preserved)", id, "tick-valid1")
		}
	})
}

// computeTestHash is a test helper that computes SHA256 hash of data.
func computeTestHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
