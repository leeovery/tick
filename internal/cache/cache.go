// Package cache provides SQLite-based caching for task data.
// The cache is expendable and always rebuildable from the JSONL source of truth.
// It uses SHA256 hash-based freshness detection to self-heal on every operation.
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

// Cache wraps a SQLite database used as a query cache for task data.
type Cache struct {
	db *sql.DB
}

// Open opens or creates the SQLite cache database at the given path
// and ensures the schema is initialized.
func Open(path string) (*Cache, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize cache schema: %w", err)
	}

	return &Cache{db: db}, nil
}

// DB returns the underlying *sql.DB for direct queries.
func (c *Cache) DB() *sql.DB {
	return c.db
}

// Close closes the database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}

// Rebuild clears all existing data and repopulates the cache from the given
// tasks and raw JSONL content. The entire operation runs in a single transaction.
// The SHA256 hash of jsonlData is stored in the metadata table for freshness detection.
func (c *Cache) Rebuild(tasks []task.Task, jsonlData []byte) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin rebuild transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing data
	if _, err := tx.Exec("DELETE FROM dependencies"); err != nil {
		return fmt.Errorf("failed to clear dependencies: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tasks"); err != nil {
		return fmt.Errorf("failed to clear tasks: %w", err)
	}

	// Insert all tasks
	taskStmt, err := tx.Prepare("INSERT INTO tasks (id, title, status, priority, description, parent, created, updated, closed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare task insert: %w", err)
	}
	defer taskStmt.Close()

	depStmt, err := tx.Prepare("INSERT INTO dependencies (task_id, blocked_by) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare dependency insert: %w", err)
	}
	defer depStmt.Close()

	for _, t := range tasks {
		var desc, parent, closed *string
		if t.Description != "" {
			desc = &t.Description
		}
		if t.Parent != "" {
			parent = &t.Parent
		}
		if t.Closed != nil {
			s := t.Closed.UTC().Format("2006-01-02T15:04:05Z")
			closed = &s
		}

		created := t.Created.UTC().Format("2006-01-02T15:04:05Z")
		updated := t.Updated.UTC().Format("2006-01-02T15:04:05Z")

		if _, err := taskStmt.Exec(t.ID, t.Title, string(t.Status), t.Priority, desc, parent, created, updated, closed); err != nil {
			return fmt.Errorf("failed to insert task %s: %w", t.ID, err)
		}

		// Insert dependencies (normalize blocked_by array)
		for _, blockedBy := range t.BlockedBy {
			if _, err := depStmt.Exec(t.ID, blockedBy); err != nil {
				return fmt.Errorf("failed to insert dependency %s -> %s: %w", t.ID, blockedBy, err)
			}
		}
	}

	// Store hash
	hash := computeSHA256(jsonlData)
	if _, err := tx.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('jsonl_hash', ?)", hash); err != nil {
		return fmt.Errorf("failed to store jsonl_hash: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rebuild transaction: %w", err)
	}

	return nil
}

// IsFresh checks whether the cache is up-to-date with the given JSONL content.
// It compares the SHA256 hash of jsonlData with the hash stored in the metadata table.
// Returns true if hashes match (cache is fresh), false if they differ (stale).
func (c *Cache) IsFresh(jsonlData []byte) (bool, error) {
	currentHash := computeSHA256(jsonlData)

	var storedHash string
	err := c.db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read jsonl_hash from metadata: %w", err)
	}

	return storedHash == currentHash, nil
}

// EnsureFresh is the gatekeeper called on every operation. It opens (or creates)
// the cache, checks freshness, and triggers a rebuild if stale.
// If the cache file is missing, it creates it from scratch.
// If the cache file is corrupted, it deletes it, recreates, and rebuilds.
func EnsureFresh(dbPath string, jsonlData []byte, tasks []task.Task) error {
	c, err := Open(dbPath)
	if err != nil {
		// Could be corrupted — try to recover
		return recoverAndRebuild(dbPath, jsonlData, tasks)
	}
	defer c.Close()

	fresh, err := c.IsFresh(jsonlData)
	if err != nil {
		// Query error — possibly corrupted schema
		c.Close()
		return recoverAndRebuild(dbPath, jsonlData, tasks)
	}

	if fresh {
		return nil
	}

	return c.Rebuild(tasks, jsonlData)
}

// recoverAndRebuild handles corrupted cache files by deleting and recreating them.
func recoverAndRebuild(dbPath string, jsonlData []byte, tasks []task.Task) error {
	log.Printf("warning: cache at %s appears corrupted, rebuilding from scratch", dbPath)

	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove corrupted cache: %w", err)
	}

	c, err := Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create new cache after recovery: %w", err)
	}
	defer c.Close()

	return c.Rebuild(tasks, jsonlData)
}

// computeSHA256 returns the hex-encoded SHA256 hash of the given data.
func computeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
