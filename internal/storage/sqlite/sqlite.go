// Package sqlite provides an auto-rebuilding SQLite cache for task queries.
// The cache is expendable and always rebuildable from the JSONL source of truth.
package sqlite

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/leeovery/tick/internal/task"
)

const timeFormat = "2006-01-02T15:04:05Z"

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
	db     *sql.DB
	dbPath string
}

// NewCache opens or creates a SQLite cache at the given path and initializes the schema.
func NewCache(dbPath string) (*Cache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache db: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize cache schema: %w", err)
	}

	return &Cache{db: db, dbPath: dbPath}, nil
}

// DB returns the underlying *sql.DB for direct queries.
func (c *Cache) DB() *sql.DB {
	return c.db
}

// Close closes the underlying database connection.
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Rebuild clears all cache data and repopulates from the given tasks and raw JSONL content.
// The entire operation runs in a single transaction for atomicity.
func (c *Cache) Rebuild(tasks []task.Task, rawContent []byte) error {
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
		var description, parent, closed *string

		if t.Description != "" {
			description = &t.Description
		}
		if t.Parent != "" {
			parent = &t.Parent
		}
		if t.Closed != nil {
			s := t.Closed.UTC().Format(timeFormat)
			closed = &s
		}

		_, err := taskStmt.Exec(
			t.ID,
			t.Title,
			string(t.Status),
			t.Priority,
			description,
			parent,
			t.Created.UTC().Format(timeFormat),
			t.Updated.UTC().Format(timeFormat),
			closed,
		)
		if err != nil {
			return fmt.Errorf("failed to insert task %s: %w", t.ID, err)
		}

		for _, dep := range t.BlockedBy {
			if _, err := depStmt.Exec(t.ID, dep); err != nil {
				return fmt.Errorf("failed to insert dependency %s -> %s: %w", t.ID, dep, err)
			}
		}
	}

	// Store hash
	hash := computeHash(rawContent)
	_, err = tx.Exec(
		"INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)",
		"jsonl_hash", hash,
	)
	if err != nil {
		return fmt.Errorf("failed to store jsonl_hash: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rebuild transaction: %w", err)
	}

	return nil
}

// IsFresh checks whether the cache is up-to-date with the given raw JSONL content.
// Returns true if the stored hash matches the hash of rawContent.
// Returns false if there is no stored hash (empty metadata) or hashes differ.
func (c *Cache) IsFresh(rawContent []byte) (bool, error) {
	var storedHash string
	err := c.db.QueryRow("SELECT value FROM metadata WHERE key = ?", "jsonl_hash").Scan(&storedHash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to query jsonl_hash: %w", err)
	}

	return storedHash == computeHash(rawContent), nil
}

// EnsureFresh is the gatekeeper called on every operation. It opens or creates the cache,
// checks freshness, and triggers a rebuild if stale. If the cache file is missing or corrupted,
// it is recreated from scratch.
func EnsureFresh(dbPath string, tasks []task.Task, rawContent []byte) (*Cache, error) {
	cache, err := tryOpen(dbPath)
	if err != nil {
		// Missing or corrupted — delete and recreate
		log.Printf("warning: cache db unusable, recreating: %v", err)
		return recreateAndRebuild(dbPath, tasks, rawContent)
	}

	fresh, err := cache.IsFresh(rawContent)
	if err != nil {
		// Query error — corrupted schema or similar
		cache.Close()
		log.Printf("warning: cache freshness check failed, recreating: %v", err)
		os.Remove(dbPath)
		return recreateAndRebuild(dbPath, tasks, rawContent)
	}

	if fresh {
		return cache, nil
	}

	if err := cache.Rebuild(tasks, rawContent); err != nil {
		return nil, fmt.Errorf("failed to rebuild cache: %w", err)
	}

	return cache, nil
}

// tryOpen attempts to open the cache and verify it has the expected schema.
func tryOpen(dbPath string) (*Cache, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cache db does not exist: %w", err)
	}

	cache, err := NewCache(dbPath)
	if err != nil {
		return nil, err
	}

	// Verify schema by attempting a simple query on each table
	if _, err := cache.db.Exec("SELECT 1 FROM tasks LIMIT 0"); err != nil {
		cache.Close()
		return nil, fmt.Errorf("tasks table unusable: %w", err)
	}
	if _, err := cache.db.Exec("SELECT 1 FROM dependencies LIMIT 0"); err != nil {
		cache.Close()
		return nil, fmt.Errorf("dependencies table unusable: %w", err)
	}
	if _, err := cache.db.Exec("SELECT 1 FROM metadata LIMIT 0"); err != nil {
		cache.Close()
		return nil, fmt.Errorf("metadata table unusable: %w", err)
	}

	return cache, nil
}

// recreateAndRebuild deletes any existing cache file, creates a fresh one, and runs a full rebuild.
func recreateAndRebuild(dbPath string, tasks []task.Task, rawContent []byte) (*Cache, error) {
	os.Remove(dbPath)

	cache, err := NewCache(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create new cache: %w", err)
	}

	if err := cache.Rebuild(tasks, rawContent); err != nil {
		cache.Close()
		return nil, fmt.Errorf("failed to rebuild new cache: %w", err)
	}

	return cache, nil
}

// computeHash returns the SHA256 hex digest of the given data.
func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
