package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
	_ "modernc.org/sqlite"
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
			"type": true, "description": true, "parent": true, "created": true,
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

	t.Run("it creates task_tags table in schema", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		db := cache.DB()

		// Verify task_tags table exists with correct columns
		tagCols := queryColumns(t, db, "task_tags")
		expectedTagCols := map[string]bool{"task_id": true, "tag": true}
		if len(tagCols) != len(expectedTagCols) {
			t.Errorf("task_tags table: expected %d columns, got %d: %v", len(expectedTagCols), len(tagCols), tagCols)
		}
		for col := range expectedTagCols {
			if !tagCols[col] {
				t.Errorf("task_tags table missing column %q", col)
			}
		}

		// Verify index on tag column
		indexes := queryIndexes(t, db)
		if !indexes["idx_task_tags_tag"] {
			t.Errorf("missing index %q", "idx_task_tags_tag")
		}
	})

	t.Run("it creates task_notes table in schema", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		db := cache.DB()

		// Verify task_notes table exists with correct columns
		noteCols := queryColumns(t, db, "task_notes")
		expectedNoteCols := map[string]bool{"task_id": true, "text": true, "created": true}
		if len(noteCols) != len(expectedNoteCols) {
			t.Errorf("task_notes table: expected %d columns, got %d: %v", len(expectedNoteCols), len(noteCols), noteCols)
		}
		for col := range expectedNoteCols {
			if !noteCols[col] {
				t.Errorf("task_notes table missing column %q", col)
			}
		}

		// Verify index on task_id column
		indexes := queryIndexes(t, db)
		if !indexes["idx_task_notes_task_id"] {
			t.Errorf("missing index %q", "idx_task_notes_task_id")
		}
	})

	t.Run("it creates task_refs table in schema", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		db := cache.DB()

		// Verify task_refs table exists with correct columns
		refCols := queryColumns(t, db, "task_refs")
		expectedRefCols := map[string]bool{"task_id": true, "ref": true}
		if len(refCols) != len(expectedRefCols) {
			t.Errorf("task_refs table: expected %d columns, got %d: %v", len(expectedRefCols), len(refCols), refCols)
		}
		for col := range expectedRefCols {
			if !refCols[col] {
				t.Errorf("task_refs table missing column %q", col)
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
				Type:        "feature",
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
		var typeStr sql.NullString
		var description, parent, closedStr sql.NullString
		err = db.QueryRow("SELECT id, title, status, priority, type, description, parent, created, updated, closed FROM tasks WHERE id = ?", "tick-a1b2c3").
			Scan(&id, &title, &status, &priority, &typeStr, &description, &parent, &createdStr, &updatedStr, &closedStr)
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
		if !typeStr.Valid || typeStr.String != "feature" {
			t.Errorf("type = %v, want %q", typeStr, "feature")
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

	t.Run("it stores type value in SQLite after rebuild", func(t *testing.T) {
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
				ID:       "tick-a1b2c3",
				Title:    "Typed task",
				Status:   task.StatusOpen,
				Priority: 2,
				Type:     "bug",
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		var typeVal sql.NullString
		err = cache.DB().QueryRow("SELECT type FROM tasks WHERE id = ?", "tick-a1b2c3").Scan(&typeVal)
		if err != nil {
			t.Fatalf("querying type: %v", err)
		}
		if !typeVal.Valid || typeVal.String != "bug" {
			t.Errorf("type = %v, want %q", typeVal, "bug")
		}
	})

	t.Run("it stores NULL for empty type after rebuild", func(t *testing.T) {
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
				ID:       "tick-a1b2c3",
				Title:    "No type task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		var typeVal sql.NullString
		err = cache.DB().QueryRow("SELECT type FROM tasks WHERE id = ?", "tick-a1b2c3").Scan(&typeVal)
		if err != nil {
			t.Fatalf("querying type: %v", err)
		}
		if typeVal.Valid {
			t.Errorf("type = %q, want NULL", typeVal.String)
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

	t.Run("it populates task_tags during rebuild for task with tags", func(t *testing.T) {
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
				ID:       "tick-a1b2c3",
				Title:    "Tagged task",
				Status:   task.StatusOpen,
				Priority: 2,
				Tags:     []string{"frontend", "urgent", "v2"},
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()
		rows, err := db.Query("SELECT task_id, tag FROM task_tags WHERE task_id = ? ORDER BY tag", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("querying task_tags: %v", err)
		}
		defer rows.Close()

		type tagRow struct {
			taskID string
			tag    string
		}
		var tags []tagRow
		for rows.Next() {
			var r tagRow
			if err := rows.Scan(&r.taskID, &r.tag); err != nil {
				t.Fatalf("scanning tag row: %v", err)
			}
			tags = append(tags, r)
		}

		if len(tags) != 3 {
			t.Fatalf("expected 3 tag rows, got %d", len(tags))
		}
		expectedTags := []string{"frontend", "urgent", "v2"}
		for i, want := range expectedTags {
			if tags[i].tag != want {
				t.Errorf("tags[%d].tag = %q, want %q", i, tags[i].tag, want)
			}
			if tags[i].taskID != "tick-a1b2c3" {
				t.Errorf("tags[%d].task_id = %q, want %q", i, tags[i].taskID, "tick-a1b2c3")
			}
		}
	})

	t.Run("it inserts no rows for task with empty tags slice", func(t *testing.T) {
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
				ID:       "tick-a1b2c3",
				Title:    "Untagged task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM task_tags WHERE task_id = ?", "tick-a1b2c3").Scan(&count)
		if err != nil {
			t.Fatalf("querying task_tags count: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 tag rows, got %d", count)
		}
	})

	t.Run("it clears stale tags on rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// First rebuild with tags
		tasksV1 := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Tagged task",
				Status:   task.StatusOpen,
				Priority: 2,
				Tags:     []string{"frontend", "urgent"},
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(tasksV1, []byte("v1")); err != nil {
			t.Fatalf("first Rebuild returned error: %v", err)
		}

		// Second rebuild with different tags (old ones should be gone)
		tasksV2 := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Tagged task",
				Status:   task.StatusOpen,
				Priority: 2,
				Tags:     []string{"backend"},
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(tasksV2, []byte("v2")); err != nil {
			t.Fatalf("second Rebuild returned error: %v", err)
		}

		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM task_tags").Scan(&count)
		if err != nil {
			t.Fatalf("querying task_tags count: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 tag row after rebuild, got %d", count)
		}

		var tag string
		err = cache.DB().QueryRow("SELECT tag FROM task_tags WHERE task_id = ?", "tick-a1b2c3").Scan(&tag)
		if err != nil {
			t.Fatalf("querying tag: %v", err)
		}
		if tag != "backend" {
			t.Errorf("tag = %q, want %q", tag, "backend")
		}
	})

	t.Run("it populates refs in task_refs during rebuild", func(t *testing.T) {
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
				ID:       "tick-a1b2c3",
				Title:    "Task with refs",
				Status:   task.StatusOpen,
				Priority: 2,
				Refs:     []string{"https://github.com/org/repo/issues/1", "docs/spec.md"},
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()
		rows, err := db.Query("SELECT task_id, ref FROM task_refs WHERE task_id = ? ORDER BY ref", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("querying task_refs: %v", err)
		}
		defer rows.Close()

		type refRow struct {
			taskID string
			ref    string
		}
		var refs []refRow
		for rows.Next() {
			var r refRow
			if err := rows.Scan(&r.taskID, &r.ref); err != nil {
				t.Fatalf("scanning ref row: %v", err)
			}
			refs = append(refs, r)
		}

		if len(refs) != 2 {
			t.Fatalf("expected 2 ref rows, got %d", len(refs))
		}
		expectedRefs := []string{"docs/spec.md", "https://github.com/org/repo/issues/1"}
		for i, want := range expectedRefs {
			if refs[i].ref != want {
				t.Errorf("refs[%d].ref = %q, want %q", i, refs[i].ref, want)
			}
			if refs[i].taskID != "tick-a1b2c3" {
				t.Errorf("refs[%d].task_id = %q, want %q", i, refs[i].taskID, "tick-a1b2c3")
			}
		}
	})

	t.Run("it inserts no rows for task with empty refs slice", func(t *testing.T) {
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
				ID:       "tick-a1b2c3",
				Title:    "No refs task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM task_refs WHERE task_id = ?", "tick-a1b2c3").Scan(&count)
		if err != nil {
			t.Fatalf("querying task_refs count: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 ref rows, got %d", count)
		}
	})

	t.Run("it clears stale refs on rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// First rebuild with refs
		tasksV1 := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Ref task",
				Status:   task.StatusOpen,
				Priority: 2,
				Refs:     []string{"https://old-url.com", "old-doc.md"},
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(tasksV1, []byte("v1")); err != nil {
			t.Fatalf("first Rebuild returned error: %v", err)
		}

		// Second rebuild with different refs (old ones should be gone)
		tasksV2 := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Ref task",
				Status:   task.StatusOpen,
				Priority: 2,
				Refs:     []string{"https://new-url.com"},
				Created:  created,
				Updated:  created,
			},
		}
		if err := cache.Rebuild(tasksV2, []byte("v2")); err != nil {
			t.Fatalf("second Rebuild returned error: %v", err)
		}

		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM task_refs").Scan(&count)
		if err != nil {
			t.Fatalf("querying task_refs count: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 ref row after rebuild, got %d", count)
		}

		var ref string
		err = cache.DB().QueryRow("SELECT ref FROM task_refs WHERE task_id = ?", "tick-a1b2c3").Scan(&ref)
		if err != nil {
			t.Fatalf("querying ref: %v", err)
		}
		if ref != "https://new-url.com" {
			t.Errorf("ref = %q, want %q", ref, "https://new-url.com")
		}
	})

	t.Run("it populates notes in task_notes during rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		note1Time := time.Date(2026, 1, 20, 9, 0, 0, 0, time.UTC)
		note2Time := time.Date(2026, 1, 21, 14, 30, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Task with notes",
				Status:   task.StatusOpen,
				Priority: 2,
				Notes: []task.Note{
					{Text: "First note", Created: note1Time},
					{Text: "Second note", Created: note2Time},
				},
				Created: created,
				Updated: created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()
		rows, err := db.Query("SELECT task_id, text, created FROM task_notes WHERE task_id = ? ORDER BY rowid", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("querying task_notes: %v", err)
		}
		defer rows.Close()

		type noteRow struct {
			taskID  string
			text    string
			created string
		}
		var notes []noteRow
		for rows.Next() {
			var r noteRow
			if err := rows.Scan(&r.taskID, &r.text, &r.created); err != nil {
				t.Fatalf("scanning note row: %v", err)
			}
			notes = append(notes, r)
		}

		if len(notes) != 2 {
			t.Fatalf("expected 2 note rows, got %d", len(notes))
		}
		if notes[0].taskID != "tick-a1b2c3" {
			t.Errorf("notes[0].task_id = %q, want %q", notes[0].taskID, "tick-a1b2c3")
		}
		if notes[0].text != "First note" {
			t.Errorf("notes[0].text = %q, want %q", notes[0].text, "First note")
		}
		if notes[0].created != "2026-01-20T09:00:00Z" {
			t.Errorf("notes[0].created = %q, want %q", notes[0].created, "2026-01-20T09:00:00Z")
		}
		if notes[1].taskID != "tick-a1b2c3" {
			t.Errorf("notes[1].task_id = %q, want %q", notes[1].taskID, "tick-a1b2c3")
		}
		if notes[1].text != "Second note" {
			t.Errorf("notes[1].text = %q, want %q", notes[1].text, "Second note")
		}
		if notes[1].created != "2026-01-21T14:30:00Z" {
			t.Errorf("notes[1].created = %q, want %q", notes[1].created, "2026-01-21T14:30:00Z")
		}
	})

	t.Run("it inserts no rows for task with empty notes slice", func(t *testing.T) {
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
				ID:       "tick-a1b2c3",
				Title:    "No notes task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM task_notes WHERE task_id = ?", "tick-a1b2c3").Scan(&count)
		if err != nil {
			t.Fatalf("querying task_notes count: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 note rows, got %d", count)
		}
	})

	t.Run("it clears stale notes on rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		noteTime := time.Date(2026, 1, 20, 9, 0, 0, 0, time.UTC)

		// First rebuild with notes
		tasksV1 := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Noted task",
				Status:   task.StatusOpen,
				Priority: 2,
				Notes: []task.Note{
					{Text: "Old note 1", Created: noteTime},
					{Text: "Old note 2", Created: noteTime},
				},
				Created: created,
				Updated: created,
			},
		}
		if err := cache.Rebuild(tasksV1, []byte("v1")); err != nil {
			t.Fatalf("first Rebuild returned error: %v", err)
		}

		// Second rebuild with different note (old ones should be gone)
		tasksV2 := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Noted task",
				Status:   task.StatusOpen,
				Priority: 2,
				Notes: []task.Note{
					{Text: "New note", Created: noteTime},
				},
				Created: created,
				Updated: created,
			},
		}
		if err := cache.Rebuild(tasksV2, []byte("v2")); err != nil {
			t.Fatalf("second Rebuild returned error: %v", err)
		}

		var count int
		err = cache.DB().QueryRow("SELECT COUNT(*) FROM task_notes").Scan(&count)
		if err != nil {
			t.Fatalf("querying task_notes count: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 note row after rebuild, got %d", count)
		}

		var text string
		err = cache.DB().QueryRow("SELECT text FROM task_notes WHERE task_id = ?", "tick-a1b2c3").Scan(&text)
		if err != nil {
			t.Fatalf("querying note text: %v", err)
		}
		if text != "New note" {
			t.Errorf("text = %q, want %q", text, "New note")
		}
	})

	t.Run("it preserves note ordering", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		// Notes with timestamps out of insertion order to verify slice order is preserved
		note1Time := time.Date(2026, 1, 22, 12, 0, 0, 0, time.UTC)
		note2Time := time.Date(2026, 1, 20, 8, 0, 0, 0, time.UTC)
		note3Time := time.Date(2026, 1, 21, 16, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Ordered notes task",
				Status:   task.StatusOpen,
				Priority: 2,
				Notes: []task.Note{
					{Text: "Note A", Created: note1Time},
					{Text: "Note B", Created: note2Time},
					{Text: "Note C", Created: note3Time},
				},
				Created: created,
				Updated: created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()
		rows, err := db.Query("SELECT text FROM task_notes WHERE task_id = ? ORDER BY rowid", "tick-a1b2c3")
		if err != nil {
			t.Fatalf("querying task_notes: %v", err)
		}
		defer rows.Close()

		var texts []string
		for rows.Next() {
			var text string
			if err := rows.Scan(&text); err != nil {
				t.Fatalf("scanning note row: %v", err)
			}
			texts = append(texts, text)
		}

		if len(texts) != 3 {
			t.Fatalf("expected 3 note rows, got %d", len(texts))
		}
		expectedTexts := []string{"Note A", "Note B", "Note C"}
		for i, want := range expectedTexts {
			if texts[i] != want {
				t.Errorf("texts[%d] = %q, want %q", i, texts[i], want)
			}
		}
	})

	t.Run("it handles rebuild with multiple tasks having different tag sets", func(t *testing.T) {
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
				ID:       "tick-aaaaaa",
				Title:    "Task A",
				Status:   task.StatusOpen,
				Priority: 2,
				Tags:     []string{"frontend", "urgent"},
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-bbbbbb",
				Title:    "Task B",
				Status:   task.StatusOpen,
				Priority: 2,
				Tags:     []string{"backend"},
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-cccccc",
				Title:    "Task C (no tags)",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}

		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		db := cache.DB()

		// Total tag rows: 2 (A) + 1 (B) + 0 (C) = 3
		var totalCount int
		err = db.QueryRow("SELECT COUNT(*) FROM task_tags").Scan(&totalCount)
		if err != nil {
			t.Fatalf("querying total task_tags count: %v", err)
		}
		if totalCount != 3 {
			t.Errorf("expected 3 total tag rows, got %d", totalCount)
		}

		// Task A has 2 tags
		var countA int
		err = db.QueryRow("SELECT COUNT(*) FROM task_tags WHERE task_id = ?", "tick-aaaaaa").Scan(&countA)
		if err != nil {
			t.Fatalf("querying task A tags count: %v", err)
		}
		if countA != 2 {
			t.Errorf("expected 2 tags for task A, got %d", countA)
		}

		// Task B has 1 tag
		var countB int
		err = db.QueryRow("SELECT COUNT(*) FROM task_tags WHERE task_id = ?", "tick-bbbbbb").Scan(&countB)
		if err != nil {
			t.Fatalf("querying task B tags count: %v", err)
		}
		if countB != 1 {
			t.Errorf("expected 1 tag for task B, got %d", countB)
		}

		// Task C has 0 tags
		var countC int
		err = db.QueryRow("SELECT COUNT(*) FROM task_tags WHERE task_id = ?", "tick-cccccc").Scan(&countC)
		if err != nil {
			t.Fatalf("querying task C tags count: %v", err)
		}
		if countC != 0 {
			t.Errorf("expected 0 tags for task C, got %d", countC)
		}

		// Rebuild again with same data — idempotent
		if err := cache.Rebuild(tasks, []byte("raw")); err != nil {
			t.Fatalf("second Rebuild returned error: %v", err)
		}

		var totalCountAfter int
		err = db.QueryRow("SELECT COUNT(*) FROM task_tags").Scan(&totalCountAfter)
		if err != nil {
			t.Fatalf("querying total task_tags count after second rebuild: %v", err)
		}
		if totalCountAfter != 3 {
			t.Errorf("expected 3 total tag rows after idempotent rebuild, got %d", totalCountAfter)
		}
	})
}

func TestSchemaVersion(t *testing.T) {
	t.Run("it stores schema_version in metadata after rebuild", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		if err := cache.Rebuild(nil, []byte("")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		var value string
		err = cache.DB().QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&value)
		if err != nil {
			t.Fatalf("querying schema_version: %v", err)
		}
		if value != "1" {
			t.Errorf("schema_version = %q, want %q", value, "1")
		}
	})

	t.Run("it returns current schema version via SchemaVersion()", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		if err := cache.Rebuild(nil, []byte("")); err != nil {
			t.Fatalf("Rebuild returned error: %v", err)
		}

		version, err := cache.SchemaVersion()
		if err != nil {
			t.Fatalf("SchemaVersion returned error: %v", err)
		}
		if version != 1 {
			t.Errorf("SchemaVersion() = %d, want %d", version, 1)
		}
	})

	t.Run("it returns 0 when schema_version row is missing", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		// No Rebuild() called — metadata table exists but has no schema_version row.
		version, err := cache.SchemaVersion()
		if err != nil {
			t.Fatalf("SchemaVersion returned error: %v", err)
		}
		if version != 0 {
			t.Errorf("SchemaVersion() = %d, want %d", version, 0)
		}
	})

	t.Run("it stores schema_version in the same transaction as jsonl_hash", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "cache.db")

		cache, err := OpenCache(dbPath)
		if err != nil {
			t.Fatalf("OpenCache returned error: %v", err)
		}
		defer cache.Close()

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// First, do a valid rebuild so metadata has known values.
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

		// Attempt rebuild with duplicate task IDs to force transaction rollback.
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

		// Verify schema_version was NOT updated (still from valid rebuild).
		version, err := cache.SchemaVersion()
		if err != nil {
			t.Fatalf("SchemaVersion returned error: %v", err)
		}
		if version != 1 {
			t.Errorf("SchemaVersion() = %d, want %d (original should be preserved)", version, 1)
		}

		// Verify jsonl_hash was also NOT updated (still from valid rebuild).
		var storedHash string
		err = cache.DB().QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("querying jsonl_hash: %v", err)
		}
		expectedHash := computeTestHash([]byte("valid"))
		if storedHash != expectedHash {
			t.Errorf("jsonl_hash = %q, want %q (original should be preserved)", storedHash, expectedHash)
		}
	})

	t.Run("it returns compiled-in version via CurrentSchemaVersion()", func(t *testing.T) {
		version := CurrentSchemaVersion()
		if version != 1 {
			t.Errorf("CurrentSchemaVersion() = %d, want %d", version, 1)
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
