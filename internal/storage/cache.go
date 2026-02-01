package storage

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/leeovery/tick/internal/task"
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

// Cache is a SQLite-backed cache for task data.
type Cache struct {
	db     *sql.DB
	dbPath string
}

// NewCache opens or creates a SQLite cache at the given path and ensures the schema exists.
func NewCache(dbPath string) (*Cache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening cache database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating cache schema: %w", err)
	}

	return &Cache{db: db, dbPath: dbPath}, nil
}

// NewCacheWithRecovery opens a cache, deleting and recreating it if corrupted.
func NewCacheWithRecovery(dbPath string) (*Cache, error) {
	cache, err := NewCache(dbPath)
	if err != nil {
		// Corrupted — delete and retry
		os.Remove(dbPath)
		cache, err = NewCache(dbPath)
		if err != nil {
			return nil, fmt.Errorf("recovery failed: %w", err)
		}
	}
	return cache, nil
}

// Close closes the underlying database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}

// DB returns the underlying database connection for queries.
func (c *Cache) DB() *sql.DB {
	return c.db
}

func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// IsFresh checks if the cache matches the given JSONL content.
// Returns true if the stored hash matches the content hash.
func (c *Cache) IsFresh(jsonlContent []byte) (bool, error) {
	var storedHash string
	err := c.db.QueryRow("SELECT value FROM metadata WHERE key='jsonl_hash'").Scan(&storedHash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("querying hash: %w", err)
	}

	currentHash := computeHash(jsonlContent)
	return storedHash == currentHash, nil
}

// Rebuild replaces all cache data with the given tasks and stores the content hash.
// The entire operation runs in a single transaction.
func (c *Cache) Rebuild(tasks []task.Task, jsonlContent []byte) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing data
	if _, err := tx.Exec("DELETE FROM dependencies"); err != nil {
		return fmt.Errorf("clearing dependencies: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tasks"); err != nil {
		return fmt.Errorf("clearing tasks: %w", err)
	}

	// Insert all tasks
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
		var description, parent, closed sql.NullString

		if t.Description != "" {
			description = sql.NullString{String: t.Description, Valid: true}
		}
		if t.Parent != "" {
			parent = sql.NullString{String: t.Parent, Valid: true}
		}
		if t.Closed != nil {
			closed = sql.NullString{String: task.FormatTimestamp(*t.Closed), Valid: true}
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

		// Insert dependencies
		for _, dep := range t.BlockedBy {
			if _, err := depStmt.Exec(t.ID, dep); err != nil {
				return fmt.Errorf("inserting dependency %s -> %s: %w", t.ID, dep, err)
			}
		}
	}

	// Store content hash
	contentHash := computeHash(jsonlContent)
	if _, err := tx.Exec(
		"INSERT OR REPLACE INTO metadata (key, value) VALUES ('jsonl_hash', ?)",
		contentHash,
	); err != nil {
		return fmt.Errorf("storing content hash: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing rebuild: %w", err)
	}

	return nil
}

// EnsureFresh checks if the cache is fresh and rebuilds if stale.
// This is the gatekeeper called on every operation.
func (c *Cache) EnsureFresh(jsonlContent []byte, tasks []task.Task) error {
	fresh, err := c.IsFresh(jsonlContent)
	if err != nil {
		// Query error — might be corrupted, rebuild
		return c.Rebuild(tasks, jsonlContent)
	}
	if fresh {
		return nil
	}
	return c.Rebuild(tasks, jsonlContent)
}
