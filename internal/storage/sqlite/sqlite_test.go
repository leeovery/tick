package sqlite

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

// mustParseTime parses an RFC3339 time string or fails the test.
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", s, err)
	}
	return ts
}

// timePtr returns a pointer to the given time.
func timePtr(t time.Time) *time.Time {
	return &t
}

// hashBytes computes the SHA256 hex string for the given bytes.
func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// sampleTasks returns a set of tasks covering all fields for testing.
func sampleTasks(t *testing.T) []task.Task {
	t.Helper()
	created := mustParseTime(t, "2026-01-19T10:00:00Z")
	updated := mustParseTime(t, "2026-01-19T14:00:00Z")
	closed := mustParseTime(t, "2026-01-19T16:00:00Z")

	return []task.Task{
		{
			ID:          "tick-a1b2c3",
			Title:       "First task",
			Status:      task.StatusOpen,
			Priority:    2,
			Description: "A description",
			BlockedBy:   []string{"tick-x1y2z3", "tick-m4n5o6"},
			Parent:      "tick-p7q8r9",
			Created:     created,
			Updated:     updated,
		},
		{
			ID:       "tick-d4e5f6",
			Title:    "Second task",
			Status:   task.StatusDone,
			Priority: 1,
			Created:  created,
			Updated:  updated,
			Closed:   timePtr(closed),
		},
		{
			ID:          "tick-g7h8i9",
			Title:       "Third task",
			Status:      task.StatusInProgress,
			Priority:    0,
			Description: "Details\nwith newlines",
			BlockedBy:   []string{"tick-d4e5f6"},
			Created:     created,
			Updated:     updated,
		},
	}
}

func TestCreatesCacheDBWithCorrectSchema(t *testing.T) {
	t.Run("it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		db := cache.DB()

		// Verify tasks table exists with correct columns
		rows, err := db.Query("PRAGMA table_info(tasks)")
		if err != nil {
			t.Fatalf("failed to query tasks table info: %v", err)
		}
		defer rows.Close()

		taskCols := make(map[string]string)
		for rows.Next() {
			var cid int
			var name, colType string
			var notnull int
			var dfltValue sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan column info: %v", err)
			}
			taskCols[name] = colType
		}

		expectedTaskCols := map[string]string{
			"id": "TEXT", "title": "TEXT", "status": "TEXT",
			"priority": "INTEGER", "description": "TEXT", "parent": "TEXT",
			"created": "TEXT", "updated": "TEXT", "closed": "TEXT",
		}
		for col, wantType := range expectedTaskCols {
			gotType, ok := taskCols[col]
			if !ok {
				t.Errorf("tasks table missing column %q", col)
				continue
			}
			if gotType != wantType {
				t.Errorf("tasks.%s type = %q, want %q", col, gotType, wantType)
			}
		}

		// Verify dependencies table exists with correct columns
		rows2, err := db.Query("PRAGMA table_info(dependencies)")
		if err != nil {
			t.Fatalf("failed to query dependencies table info: %v", err)
		}
		defer rows2.Close()

		depCols := make(map[string]string)
		for rows2.Next() {
			var cid int
			var name, colType string
			var notnull int
			var dfltValue sql.NullString
			var pk int
			if err := rows2.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan column info: %v", err)
			}
			depCols[name] = colType
		}

		expectedDepCols := map[string]string{"task_id": "TEXT", "blocked_by": "TEXT"}
		for col, wantType := range expectedDepCols {
			gotType, ok := depCols[col]
			if !ok {
				t.Errorf("dependencies table missing column %q", col)
				continue
			}
			if gotType != wantType {
				t.Errorf("dependencies.%s type = %q, want %q", col, gotType, wantType)
			}
		}

		// Verify metadata table exists with correct columns
		rows3, err := db.Query("PRAGMA table_info(metadata)")
		if err != nil {
			t.Fatalf("failed to query metadata table info: %v", err)
		}
		defer rows3.Close()

		metaCols := make(map[string]string)
		for rows3.Next() {
			var cid int
			var name, colType string
			var notnull int
			var dfltValue sql.NullString
			var pk int
			if err := rows3.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err != nil {
				t.Fatalf("failed to scan column info: %v", err)
			}
			metaCols[name] = colType
		}

		expectedMetaCols := map[string]string{"key": "TEXT", "value": "TEXT"}
		for col, wantType := range expectedMetaCols {
			gotType, ok := metaCols[col]
			if !ok {
				t.Errorf("metadata table missing column %q", col)
				continue
			}
			if gotType != wantType {
				t.Errorf("metadata.%s type = %q, want %q", col, gotType, wantType)
			}
		}

		// Verify indexes exist
		indexRows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_%'")
		if err != nil {
			t.Fatalf("failed to query indexes: %v", err)
		}
		defer indexRows.Close()

		indexes := make(map[string]bool)
		for indexRows.Next() {
			var name string
			if err := indexRows.Scan(&name); err != nil {
				t.Fatalf("failed to scan index name: %v", err)
			}
			indexes[name] = true
		}

		expectedIndexes := []string{"idx_tasks_status", "idx_tasks_priority", "idx_tasks_parent"}
		for _, idx := range expectedIndexes {
			if !indexes[idx] {
				t.Errorf("missing index %q", idx)
			}
		}
	})
}

func TestRebuildFromParsedTasks(t *testing.T) {
	t.Run("it rebuilds cache from parsed tasks — all fields round-trip correctly", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		tasks := sampleTasks(t)
		rawContent := []byte("fake jsonl content")

		if err := cache.Rebuild(tasks, rawContent); err != nil {
			t.Fatalf("Rebuild() returned error: %v", err)
		}

		db := cache.DB()

		// Verify first task (with all optional fields)
		var id, title, status, created, updated string
		var priority int
		var description, parent, closed sql.NullString
		err = db.QueryRow("SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id = ?", "tick-a1b2c3").
			Scan(&id, &title, &status, &priority, &description, &parent, &created, &updated, &closed)
		if err != nil {
			t.Fatalf("failed to query task tick-a1b2c3: %v", err)
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
		if priority != 2 {
			t.Errorf("priority = %d, want %d", priority, 2)
		}
		if !description.Valid || description.String != "A description" {
			t.Errorf("description = %v, want %q", description, "A description")
		}
		if !parent.Valid || parent.String != "tick-p7q8r9" {
			t.Errorf("parent = %v, want %q", parent, "tick-p7q8r9")
		}
		if created != "2026-01-19T10:00:00Z" {
			t.Errorf("created = %q, want %q", created, "2026-01-19T10:00:00Z")
		}
		if updated != "2026-01-19T14:00:00Z" {
			t.Errorf("updated = %q, want %q", updated, "2026-01-19T14:00:00Z")
		}
		if closed.Valid {
			t.Errorf("closed = %q, want NULL", closed.String)
		}

		// Verify second task (with closed timestamp)
		var closed2 sql.NullString
		err = db.QueryRow("SELECT closed FROM tasks WHERE id = ?", "tick-d4e5f6").Scan(&closed2)
		if err != nil {
			t.Fatalf("failed to query task tick-d4e5f6: %v", err)
		}
		if !closed2.Valid || closed2.String != "2026-01-19T16:00:00Z" {
			t.Errorf("closed = %v, want %q", closed2, "2026-01-19T16:00:00Z")
		}

		// Verify total task count
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}
	})
}

func TestNormalizesBlockedByIntoDependencies(t *testing.T) {
	t.Run("it normalizes blocked_by array into dependencies table rows", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		tasks := sampleTasks(t)
		rawContent := []byte("fake jsonl content")

		if err := cache.Rebuild(tasks, rawContent); err != nil {
			t.Fatalf("Rebuild() returned error: %v", err)
		}

		db := cache.DB()

		// tick-a1b2c3 has blocked_by: [tick-x1y2z3, tick-m4n5o6]
		rows, err := db.Query("SELECT blocked_by FROM dependencies WHERE task_id = ? ORDER BY blocked_by", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("failed to query dependencies: %v", err)
		}
		defer rows.Close()

		var deps []string
		for rows.Next() {
			var blockedBy string
			if err := rows.Scan(&blockedBy); err != nil {
				t.Fatalf("failed to scan blocked_by: %v", err)
			}
			deps = append(deps, blockedBy)
		}

		if len(deps) != 2 {
			t.Fatalf("tick-a1b2c3 dependency count = %d, want 2", len(deps))
		}
		if deps[0] != "tick-m4n5o6" {
			t.Errorf("deps[0] = %q, want %q", deps[0], "tick-m4n5o6")
		}
		if deps[1] != "tick-x1y2z3" {
			t.Errorf("deps[1] = %q, want %q", deps[1], "tick-x1y2z3")
		}

		// tick-g7h8i9 has blocked_by: [tick-d4e5f6]
		var depCount int
		err = db.QueryRow("SELECT COUNT(*) FROM dependencies WHERE task_id = ?", "tick-g7h8i9").Scan(&depCount)
		if err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if depCount != 1 {
			t.Errorf("tick-g7h8i9 dependency count = %d, want 1", depCount)
		}

		// tick-d4e5f6 has no blocked_by
		err = db.QueryRow("SELECT COUNT(*) FROM dependencies WHERE task_id = ?", "tick-d4e5f6").Scan(&depCount)
		if err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if depCount != 0 {
			t.Errorf("tick-d4e5f6 dependency count = %d, want 0", depCount)
		}
	})
}

func TestStoresHashInMetadata(t *testing.T) {
	t.Run("it stores JSONL content hash in metadata table after rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		rawContent := []byte("some jsonl content here")
		expectedHash := hashBytes(rawContent)

		if err := cache.Rebuild(nil, rawContent); err != nil {
			t.Fatalf("Rebuild() returned error: %v", err)
		}

		db := cache.DB()
		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = ?", "jsonl_hash").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to query jsonl_hash: %v", err)
		}

		if storedHash != expectedHash {
			t.Errorf("stored hash = %q, want %q", storedHash, expectedHash)
		}
	})
}

func TestDetectsFreshCache(t *testing.T) {
	t.Run("it detects fresh cache (hash matches) and skips rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		rawContent := []byte("jsonl content")

		// Initial rebuild
		if err := cache.Rebuild(nil, rawContent); err != nil {
			t.Fatalf("Rebuild() returned error: %v", err)
		}

		// Check freshness - should be fresh
		fresh, err := cache.IsFresh(rawContent)
		if err != nil {
			t.Fatalf("IsFresh() returned error: %v", err)
		}
		if !fresh {
			t.Error("IsFresh() = false, want true for matching hash")
		}
	})
}

func TestDetectsStaleCache(t *testing.T) {
	t.Run("it detects stale cache (hash mismatch) and triggers rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		rawContent := []byte("original content")
		if err := cache.Rebuild(nil, rawContent); err != nil {
			t.Fatalf("Rebuild() returned error: %v", err)
		}

		// Check freshness with different content - should be stale
		newContent := []byte("modified content")
		fresh, err := cache.IsFresh(newContent)
		if err != nil {
			t.Fatalf("IsFresh() returned error: %v", err)
		}
		if fresh {
			t.Error("IsFresh() = true, want false for mismatched hash")
		}
	})
}

func TestRebuildsWhenCacheDBMissing(t *testing.T) {
	t.Run("it rebuilds from scratch when cache.db is missing", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		tasks := sampleTasks(t)
		rawContent := []byte("jsonl file bytes")

		// EnsureFresh should create the DB and rebuild
		cache, err := EnsureFresh(dbPath, tasks, rawContent)
		if err != nil {
			t.Fatalf("EnsureFresh() returned error: %v", err)
		}
		defer cache.Close()

		// Verify DB was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Fatal("cache.db was not created")
		}

		// Verify tasks were inserted
		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}

		// Verify hash was stored
		var storedHash string
		err = cache.DB().QueryRow("SELECT value FROM metadata WHERE key = ?", "jsonl_hash").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to query hash: %v", err)
		}
		if storedHash != hashBytes(rawContent) {
			t.Errorf("stored hash = %q, want %q", storedHash, hashBytes(rawContent))
		}
	})
}

func TestDeletesAndRecreatesCorruptedDB(t *testing.T) {
	t.Run("it deletes and recreates cache.db when corrupted", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		// Write garbage to create a corrupted DB file
		if err := os.WriteFile(dbPath, []byte("this is not a sqlite database"), 0644); err != nil {
			t.Fatalf("failed to write corrupted file: %v", err)
		}

		tasks := sampleTasks(t)
		rawContent := []byte("jsonl bytes")

		// EnsureFresh should detect corruption, delete, recreate, and rebuild
		cache, err := EnsureFresh(dbPath, tasks, rawContent)
		if err != nil {
			t.Fatalf("EnsureFresh() returned error: %v", err)
		}
		defer cache.Close()

		// Verify tasks were inserted correctly
		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}
	})
}

func TestHandlesEmptyTaskList(t *testing.T) {
	t.Run("it handles empty task list (zero rows, hash still stored)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		emptyContent := []byte("")

		if err := cache.Rebuild(nil, emptyContent); err != nil {
			t.Fatalf("Rebuild() returned error: %v", err)
		}

		db := cache.DB()

		// Zero rows in tasks
		var taskCount int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if taskCount != 0 {
			t.Errorf("task count = %d, want 0", taskCount)
		}

		// Zero rows in dependencies
		var depCount int
		err = db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&depCount)
		if err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if depCount != 0 {
			t.Errorf("dependency count = %d, want 0", depCount)
		}

		// Hash is still stored
		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = ?", "jsonl_hash").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to query hash: %v", err)
		}
		expectedHash := hashBytes(emptyContent)
		if storedHash != expectedHash {
			t.Errorf("stored hash = %q, want %q", storedHash, expectedHash)
		}
	})
}

func TestReplacesAllExistingDataOnRebuild(t *testing.T) {
	t.Run("it replaces all existing data on rebuild (no stale rows)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		// First rebuild with sample tasks
		tasks := sampleTasks(t)
		if err := cache.Rebuild(tasks, []byte("v1")); err != nil {
			t.Fatalf("first Rebuild() returned error: %v", err)
		}

		// Second rebuild with a single different task
		created := mustParseTime(t, "2026-02-01T10:00:00Z")
		newTasks := []task.Task{
			{
				ID:       "tick-new111",
				Title:    "Brand new task",
				Status:   task.StatusOpen,
				Priority: 3,
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(newTasks, []byte("v2")); err != nil {
			t.Fatalf("second Rebuild() returned error: %v", err)
		}

		db := cache.DB()

		// Only the new task should exist
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("task count = %d, want 1", count)
		}

		var id string
		err = db.QueryRow("SELECT id FROM tasks").Scan(&id)
		if err != nil {
			t.Fatalf("failed to query task: %v", err)
		}
		if id != "tick-new111" {
			t.Errorf("task id = %q, want %q", id, "tick-new111")
		}

		// Old dependencies should be gone
		var depCount int
		err = db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&depCount)
		if err != nil {
			t.Fatalf("failed to count dependencies: %v", err)
		}
		if depCount != 0 {
			t.Errorf("dependency count = %d, want 0 (stale rows remain)", depCount)
		}

		// Hash should be updated
		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = ?", "jsonl_hash").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to query hash: %v", err)
		}
		if storedHash != hashBytes([]byte("v2")) {
			t.Errorf("stored hash = %q, want %q", storedHash, hashBytes([]byte("v2")))
		}
	})
}

func TestRebuildIsTransactional(t *testing.T) {
	t.Run("it rebuilds within a single transaction (all-or-nothing)", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		// First, do a successful rebuild
		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		goodTasks := []task.Task{
			{
				ID:       "tick-good11",
				Title:    "Good task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(goodTasks, []byte("good")); err != nil {
			t.Fatalf("good Rebuild() returned error: %v", err)
		}

		// Now attempt a rebuild with a task that violates the schema
		// (duplicate primary key within same rebuild = should fail insert)
		badTasks := []task.Task{
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
				Title:    "Dup 2",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		err = cache.Rebuild(badTasks, []byte("bad"))
		if err == nil {
			t.Fatal("Rebuild() with duplicate IDs expected error, got nil")
		}

		// Original data should still be intact (transaction rolled back)
		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("task count = %d, want 1 (original data should be intact)", count)
		}

		var id string
		err = cache.DB().QueryRow("SELECT id FROM tasks").Scan(&id)
		if err != nil {
			t.Fatalf("failed to query task: %v", err)
		}
		if id != "tick-good11" {
			t.Errorf("task id = %q, want %q (original data should be intact)", id, "tick-good11")
		}

		// Hash should still be the old one
		var storedHash string
		err = cache.DB().QueryRow("SELECT value FROM metadata WHERE key = ?", "jsonl_hash").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to query hash: %v", err)
		}
		if storedHash != hashBytes([]byte("good")) {
			t.Errorf("stored hash = %q, want %q (should be unchanged after failed rebuild)", storedHash, hashBytes([]byte("good")))
		}
	})
}

func TestEnsureFreshSkipsRebuildWhenFresh(t *testing.T) {
	t.Run("EnsureFresh skips rebuild when cache is fresh", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		tasks := sampleTasks(t)
		rawContent := []byte("jsonl data")

		// First call — creates and rebuilds
		cache, err := EnsureFresh(dbPath, tasks, rawContent)
		if err != nil {
			t.Fatalf("first EnsureFresh() returned error: %v", err)
		}
		cache.Close()

		// Second call with same content — should skip rebuild
		// We verify by checking that tasks remain the same
		cache2, err := EnsureFresh(dbPath, nil, rawContent)
		if err != nil {
			t.Fatalf("second EnsureFresh() returned error: %v", err)
		}
		defer cache2.Close()

		// Tasks from the first rebuild should still be present (not overwritten with nil)
		var count int
		err = cache2.DB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3 (data should be preserved from first rebuild)", count)
		}
	})
}

func TestEnsureFreshRebuildsWhenStale(t *testing.T) {
	t.Run("EnsureFresh triggers rebuild when cache is stale", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		tasks := sampleTasks(t)
		rawContent := []byte("v1 content")

		// First call — creates and rebuilds
		cache, err := EnsureFresh(dbPath, tasks, rawContent)
		if err != nil {
			t.Fatalf("first EnsureFresh() returned error: %v", err)
		}
		cache.Close()

		// Second call with different content — should trigger rebuild
		created := mustParseTime(t, "2026-02-01T10:00:00Z")
		newTasks := []task.Task{
			{
				ID:       "tick-new222",
				Title:    "New only task",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}
		newContent := []byte("v2 content")

		cache2, err := EnsureFresh(dbPath, newTasks, newContent)
		if err != nil {
			t.Fatalf("second EnsureFresh() returned error: %v", err)
		}
		defer cache2.Close()

		// Should have the new tasks, not the old ones
		var count int
		err = cache2.DB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("task count = %d, want 1", count)
		}
	})
}

func TestIsFreshMissingHashTreatedAsStale(t *testing.T) {
	t.Run("it treats missing jsonl_hash in metadata as stale", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := NewCache(dbPath)
		if err != nil {
			t.Fatalf("NewCache() returned error: %v", err)
		}
		defer cache.Close()

		// Don't call Rebuild — metadata table is empty
		fresh, err := cache.IsFresh([]byte("some content"))
		if err != nil {
			t.Fatalf("IsFresh() returned error: %v", err)
		}
		if fresh {
			t.Error("IsFresh() = true, want false when no hash in metadata")
		}
	})
}

func TestEnsureFreshWithCorruptedSchemaDB(t *testing.T) {
	t.Run("it handles corrupted schema by deleting and rebuilding", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		// Create a valid SQLite DB but with wrong schema
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open db: %v", err)
		}
		_, err = db.Exec("CREATE TABLE wrong_table (id INTEGER)")
		if err != nil {
			t.Fatalf("failed to create wrong table: %v", err)
		}
		db.Close()

		tasks := sampleTasks(t)
		rawContent := []byte("jsonl data")

		// EnsureFresh should detect the schema mismatch (query errors) and rebuild
		cache, err := EnsureFresh(dbPath, tasks, rawContent)
		if err != nil {
			t.Fatalf("EnsureFresh() returned error: %v", err)
		}
		defer cache.Close()

		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}
	})
}
