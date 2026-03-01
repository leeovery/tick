package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/leeovery/tick/internal/task"
	_ "modernc.org/sqlite"
)

const schemaVersion = 1

const schemaSQL = `
CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  priority INTEGER NOT NULL DEFAULT 2,
  description TEXT,
  type TEXT,
  parent TEXT,
  created TEXT NOT NULL,
  updated TEXT NOT NULL,
  closed TEXT
);

CREATE TABLE IF NOT EXISTS dependencies (
  task_id TEXT NOT NULL,
  blocked_by TEXT NOT NULL,
  PRIMARY KEY (task_id, blocked_by)
);

CREATE TABLE IF NOT EXISTS task_tags (
  task_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (task_id, tag)
);

CREATE TABLE IF NOT EXISTS task_refs (
  task_id TEXT NOT NULL,
  ref TEXT NOT NULL,
  PRIMARY KEY (task_id, ref)
);

CREATE TABLE IF NOT EXISTS task_notes (
  task_id TEXT NOT NULL,
  text TEXT NOT NULL,
  created TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS metadata (
  key TEXT PRIMARY KEY,
  value TEXT
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent);
CREATE INDEX IF NOT EXISTS idx_task_tags_tag ON task_tags(tag);
CREATE INDEX IF NOT EXISTS idx_task_notes_task_id ON task_notes(task_id);
`

// Cache wraps a SQLite database used as a query cache for Tick tasks.
type Cache struct {
	db   *sql.DB
	path string
}

// OpenCache opens or creates a SQLite cache at the given path and ensures the schema exists.
func OpenCache(path string) (*Cache, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create cache schema: %w", err)
	}

	return &Cache{db: db, path: path}, nil
}

// Close closes the underlying database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}

// DB returns the underlying *sql.DB for direct queries (used in tests).
func (c *Cache) DB() *sql.DB {
	return c.db
}

// Rebuild clears all existing data and repopulates the cache from the given tasks and raw JSONL content.
// The entire operation runs in a single transaction for atomicity.
func (c *Cache) Rebuild(tasks []task.Task, rawJSONL []byte) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin rebuild transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Clear existing data.
	if _, err := tx.Exec("DELETE FROM task_notes"); err != nil {
		return fmt.Errorf("failed to clear task_notes: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM task_refs"); err != nil {
		return fmt.Errorf("failed to clear task_refs: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM task_tags"); err != nil {
		return fmt.Errorf("failed to clear task_tags: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM dependencies"); err != nil {
		return fmt.Errorf("failed to clear dependencies: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tasks"); err != nil {
		return fmt.Errorf("failed to clear tasks: %w", err)
	}

	// Insert all tasks.
	insertTask, err := tx.Prepare(`INSERT INTO tasks (id, title, status, priority, description, type, parent, created, updated, closed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare task insert: %w", err)
	}
	defer insertTask.Close()

	insertDep, err := tx.Prepare(`INSERT INTO dependencies (task_id, blocked_by) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare dependency insert: %w", err)
	}
	defer insertDep.Close()

	insertTag, err := tx.Prepare(`INSERT INTO task_tags (task_id, tag) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare tag insert: %w", err)
	}
	defer insertTag.Close()

	insertRef, err := tx.Prepare(`INSERT INTO task_refs (task_id, ref) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare ref insert: %w", err)
	}
	defer insertRef.Close()

	insertNote, err := tx.Prepare(`INSERT INTO task_notes (task_id, text, created) VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare note insert: %w", err)
	}
	defer insertNote.Close()

	for _, t := range tasks {
		var closedStr *string
		if t.Closed != nil {
			s := task.FormatTimestamp(*t.Closed)
			closedStr = &s
		}

		var parentStr *string
		if t.Parent != "" {
			parentStr = &t.Parent
		}

		var typeStr *string
		if t.Type != "" {
			typeStr = &t.Type
		}

		var descStr *string
		if t.Description != "" {
			descStr = &t.Description
		}

		if _, err := insertTask.Exec(
			t.ID,
			t.Title,
			string(t.Status),
			t.Priority,
			descStr,
			typeStr,
			parentStr,
			task.FormatTimestamp(t.Created),
			task.FormatTimestamp(t.Updated),
			closedStr,
		); err != nil {
			return fmt.Errorf("failed to insert task %s: %w", t.ID, err)
		}

		for _, dep := range t.BlockedBy {
			if _, err := insertDep.Exec(t.ID, dep); err != nil {
				return fmt.Errorf("failed to insert dependency %s -> %s: %w", t.ID, dep, err)
			}
		}

		for _, tag := range t.Tags {
			if _, err := insertTag.Exec(t.ID, tag); err != nil {
				return fmt.Errorf("failed to insert tag %s -> %s: %w", t.ID, tag, err)
			}
		}

		for _, ref := range t.Refs {
			if _, err := insertRef.Exec(t.ID, ref); err != nil {
				return fmt.Errorf("failed to insert ref %s -> %s: %w", t.ID, ref, err)
			}
		}

		for _, note := range t.Notes {
			if _, err := insertNote.Exec(t.ID, note.Text, task.FormatTimestamp(note.Created)); err != nil {
				return fmt.Errorf("failed to insert note for %s: %w", t.ID, err)
			}
		}
	}

	// Store the JSONL content hash.
	hash := computeHash(rawJSONL)
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO metadata (key, value) VALUES ('jsonl_hash', ?)`,
		hash,
	); err != nil {
		return fmt.Errorf("failed to store JSONL hash: %w", err)
	}

	// Store the schema version.
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO metadata (key, value) VALUES ('schema_version', ?)`,
		fmt.Sprintf("%d", schemaVersion),
	); err != nil {
		return fmt.Errorf("failed to store schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rebuild transaction: %w", err)
	}

	return nil
}

// IsFresh checks whether the cache is up-to-date with the given raw JSONL content.
// Returns true if the stored hash matches the computed hash of rawJSONL.
func (c *Cache) IsFresh(rawJSONL []byte) (bool, error) {
	var storedHash string
	err := c.db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read JSONL hash from metadata: %w", err)
	}

	currentHash := computeHash(rawJSONL)
	return storedHash == currentHash, nil
}

// SchemaVersion reads the schema version from the metadata table.
// Returns 0, nil when the row is missing (pre-versioning cache).
func (c *Cache) SchemaVersion() (int, error) {
	var value string
	err := c.db.QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read schema version from metadata: %w", err)
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse schema version %q: %w", value, err)
	}
	return v, nil
}

// CurrentSchemaVersion returns the compiled-in schema version constant.
func CurrentSchemaVersion() int {
	return schemaVersion
}

// computeHash returns the hex-encoded SHA256 hash of the given data.
func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
