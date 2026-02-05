// Package storage provides JSONL file storage and SQLite cache for tasks.
package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/leeovery/tick/internal/task"
	_ "github.com/mattn/go-sqlite3"
)

// Cache provides SQLite-backed caching for task queries.
// The cache is always rebuildable from JSONL and self-heals on mismatch.
type Cache struct {
	db   *sql.DB
	path string
}

const createSchema = `
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

// NewCache opens or creates a SQLite cache at the given path.
// Creates the parent directory if it doesn't exist.
// Creates the schema (tables and indexes) if not present.
func NewCache(path string) (*Cache, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Create schema
	if _, err := db.Exec(createSchema); err != nil {
		db.Close()
		return nil, err
	}

	return &Cache{
		db:   db,
		path: path,
	}, nil
}

// Close closes the database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}

// Rebuild clears all existing data and rebuilds from the provided tasks.
// Computes SHA256 hash of jsonlContent and stores it in metadata.
// All operations happen within a single transaction.
func (c *Cache) Rebuild(tasks []task.Task, jsonlContent []byte) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing data
	if _, err := tx.Exec(`DELETE FROM tasks`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM dependencies`); err != nil {
		return err
	}

	// Insert tasks
	taskStmt, err := tx.Prepare(`INSERT INTO tasks (id, title, status, priority, description, parent, created, updated, closed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer taskStmt.Close()

	// Insert dependencies
	depStmt, err := tx.Prepare(`INSERT INTO dependencies (task_id, blocked_by) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer depStmt.Close()

	for _, t := range tasks {
		_, err := taskStmt.Exec(t.ID, t.Title, string(t.Status), t.Priority, t.Description, t.Parent, t.Created, t.Updated, t.Closed)
		if err != nil {
			return err
		}

		// Insert dependency rows
		for _, blockedBy := range t.BlockedBy {
			if _, err := depStmt.Exec(t.ID, blockedBy); err != nil {
				return err
			}
		}
	}

	// Compute and store hash
	hash := sha256.Sum256(jsonlContent)
	hashStr := hex.EncodeToString(hash[:])

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, "jsonl_hash", hashStr)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// IsFresh checks if the cache is up-to-date with the given JSONL content.
// Returns true if the stored hash matches the hash of jsonlContent.
// Returns false if hash is missing or doesn't match.
func (c *Cache) IsFresh(jsonlContent []byte) (bool, error) {
	var storedHash string
	err := c.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, "jsonl_hash").Scan(&storedHash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(jsonlContent)
	currentHash := hex.EncodeToString(hash[:])

	return storedHash == currentHash, nil
}

// EnsureFresh opens or creates a cache and ensures it's fresh.
// If cache is missing, corrupted, or stale, it rebuilds from the provided tasks.
// Returns the cache for subsequent queries.
func EnsureFresh(cachePath string, tasks []task.Task, jsonlContent []byte) (*Cache, error) {
	// Try to open existing cache
	cache, err := NewCache(cachePath)
	if err != nil {
		// Cache is corrupted, delete and recreate
		os.Remove(cachePath)
		cache, err = NewCache(cachePath)
		if err != nil {
			return nil, err
		}
	}

	// Check freshness
	fresh, err := cache.IsFresh(jsonlContent)
	if err != nil {
		// Query error indicates corruption, delete and recreate
		cache.Close()
		os.Remove(cachePath)
		cache, err = NewCache(cachePath)
		if err != nil {
			return nil, err
		}
		fresh = false
	}

	if !fresh {
		if err := cache.Rebuild(tasks, jsonlContent); err != nil {
			cache.Close()
			return nil, err
		}
	}

	return cache, nil
}
