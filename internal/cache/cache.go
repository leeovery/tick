// Package cache provides a SQLite-based query cache for tasks that
// auto-rebuilds from the JSONL source of truth using SHA256 freshness detection.
package cache

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/leeovery/tick/internal/task"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
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

// Cache wraps a SQLite database used as a query cache for tasks.
type Cache struct {
	db   *sql.DB
	path string
}

// New opens or creates a SQLite cache database at the given path and
// initializes the schema (tables and indexes) if not present.
func New(dbPath string) (*Cache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening cache database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing cache schema: %w", err)
	}

	return &Cache{db: db, path: dbPath}, nil
}

// Close closes the underlying database connection.
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// DB returns the underlying database connection for queries.
func (c *Cache) DB() *sql.DB {
	return c.db
}

// Rebuild clears all existing data and repopulates the cache from the given
// tasks within a single transaction. It also computes and stores the SHA256
// hash of the raw JSONL content in the metadata table.
func (c *Cache) Rebuild(tasks []task.Task, jsonlData []byte) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning rebuild transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Clear existing data.
	if _, err := tx.Exec("DELETE FROM dependencies"); err != nil {
		return fmt.Errorf("clearing dependencies: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tasks"); err != nil {
		return fmt.Errorf("clearing tasks: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM metadata"); err != nil {
		return fmt.Errorf("clearing metadata: %w", err)
	}

	// Insert tasks.
	taskStmt, err := tx.Prepare(`INSERT INTO tasks (id, title, status, priority, description, parent, created, updated, closed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing task insert: %w", err)
	}
	defer taskStmt.Close()

	depStmt, err := tx.Prepare(`INSERT INTO dependencies (task_id, blocked_by) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing dependency insert: %w", err)
	}
	defer depStmt.Close()

	for _, t := range tasks {
		var description, parent, closed *string
		if t.Description != "" {
			description = &t.Description
		}
		if t.Parent != "" {
			parent = &t.Parent
		}
		if t.Closed != nil {
			s := task.FormatTimestamp(*t.Closed)
			closed = &s
		}

		_, err := taskStmt.Exec(
			t.ID,
			t.Title,
			string(t.Status),
			t.Priority,
			description,
			parent,
			task.FormatTimestamp(t.Created),
			task.FormatTimestamp(t.Updated),
			closed,
		)
		if err != nil {
			return fmt.Errorf("inserting task %s: %w", t.ID, err)
		}

		// Insert dependencies.
		for _, dep := range t.BlockedBy {
			if _, err := depStmt.Exec(t.ID, dep); err != nil {
				return fmt.Errorf("inserting dependency %s -> %s: %w", t.ID, dep, err)
			}
		}
	}

	// Store hash.
	hash := computeHash(jsonlData)
	if _, err := tx.Exec(`INSERT INTO metadata (key, value) VALUES ('jsonl_hash', ?)`, hash); err != nil {
		return fmt.Errorf("storing jsonl hash: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing rebuild transaction: %w", err)
	}
	return nil
}

// IsFresh computes the SHA256 hash of the given JSONL content and compares it
// with the hash stored in the metadata table. Returns true if they match.
func (c *Cache) IsFresh(jsonlData []byte) (bool, error) {
	var storedHash string
	err := c.db.QueryRow("SELECT value FROM metadata WHERE key='jsonl_hash'").Scan(&storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("querying jsonl hash: %w", err)
	}
	return storedHash == computeHash(jsonlData), nil
}

// EnsureFresh opens the cache at dbPath, checks freshness against the given
// JSONL content, and triggers a full rebuild if stale or missing. If the cache
// file is corrupted, it is deleted, recreated, and rebuilt.
func EnsureFresh(dbPath string, tasks []task.Task, jsonlData []byte) (*Cache, error) {
	c, err := New(dbPath)
	if err != nil {
		log.Printf("warning: cache corrupt or unreadable, recreating: %v", err)
		c, err = recreate(dbPath)
		if err != nil {
			return nil, err
		}
	}

	fresh, err := c.IsFresh(jsonlData)
	if err != nil {
		log.Printf("warning: cache query failed, recreating: %v", err)
		c.Close()
		c, err = recreate(dbPath)
		if err != nil {
			return nil, err
		}
		fresh = false
	}

	if !fresh {
		if err := c.Rebuild(tasks, jsonlData); err != nil {
			return nil, fmt.Errorf("rebuilding cache: %w", err)
		}
	}

	return c, nil
}

// recreate removes the cache file at dbPath and creates a fresh database.
func recreate(dbPath string) (*Cache, error) {
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("removing corrupt cache: %w", err)
	}
	c, err := New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("recreating cache: %w", err)
	}
	return c, nil
}

// computeHash returns the hex-encoded SHA256 hash of the given data.
func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
