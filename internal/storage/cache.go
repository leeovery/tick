package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/leeovery/tick/internal/task"
	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  priority INTEGER NOT NULL DEFAULT 2,
  description TEXT,
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

CREATE TABLE IF NOT EXISTS metadata (
  key TEXT PRIMARY KEY,
  value TEXT
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent);
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
	if _, err := tx.Exec("DELETE FROM dependencies"); err != nil {
		return fmt.Errorf("failed to clear dependencies: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tasks"); err != nil {
		return fmt.Errorf("failed to clear tasks: %w", err)
	}

	// Insert all tasks.
	insertTask, err := tx.Prepare(`INSERT INTO tasks (id, title, status, priority, description, parent, created, updated, closed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare task insert: %w", err)
	}
	defer insertTask.Close()

	insertDep, err := tx.Prepare(`INSERT INTO dependencies (task_id, blocked_by) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare dependency insert: %w", err)
	}
	defer insertDep.Close()

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
	}

	// Store the JSONL content hash.
	hash := computeHash(rawJSONL)
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO metadata (key, value) VALUES ('jsonl_hash', ?)`,
		hash,
	); err != nil {
		return fmt.Errorf("failed to store JSONL hash: %w", err)
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

// computeHash returns the hex-encoded SHA256 hash of the given data.
func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
