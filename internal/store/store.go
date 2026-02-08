// Package store provides a unified storage engine that orchestrates JSONL
// read/write and SQLite cache operations with file locking for concurrent access safety.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/cache"
	"github.com/leeovery/tick/internal/task"
)

const lockTimeout = 5 * time.Second

// Store orchestrates JSONL read/write and SQLite cache operations
// with file locking for concurrent access safety.
type Store struct {
	tickDir     string
	jsonlPath   string
	dbPath      string
	lockPath    string
	lockTimeout time.Duration

	// LogFunc is an optional verbose logging function injected by the CLI.
	// When set, the store logs key operations (lock, hash, cache, write).
	// When nil, no logging occurs.
	LogFunc func(format string, args ...interface{})
}

// NewStore creates a new Store that validates the .tick/ directory exists
// and contains tasks.jsonl.
func NewStore(tickDir string) (*Store, error) {
	info, err := os.Stat(tickDir)
	if err != nil {
		return nil, fmt.Errorf("tick directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("tick path is not a directory: %s", tickDir)
	}

	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		return nil, fmt.Errorf("tasks.jsonl not found in tick directory: %w", err)
	}

	return &Store{
		tickDir:     tickDir,
		jsonlPath:   jsonlPath,
		dbPath:      filepath.Join(tickDir, "cache.db"),
		lockPath:    filepath.Join(tickDir, "lock"),
		lockTimeout: lockTimeout,
	}, nil
}

// Close is a no-op for now; the store does not hold persistent resources.
func (s *Store) Close() error {
	return nil
}

// vlog writes a verbose log message if LogFunc is set.
func (s *Store) vlog(format string, args ...interface{}) {
	if s.LogFunc != nil {
		s.LogFunc(format, args...)
	}
}

// Mutate executes the write mutation flow:
// 1. Acquire exclusive lock
// 2. Read tasks.jsonl into memory, compute SHA256 hash
// 3. Check SQLite freshness, rebuild if stale
// 4. Pass tasks to mutation function
// 5. Write modified tasks via atomic rewrite
// 6. Update SQLite cache
// 7. Release lock (via defer)
func (s *Store) Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error {
	// Acquire exclusive lock
	s.vlog("acquiring exclusive lock on %s", s.lockPath)
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
	}
	if !locked {
		return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
	}
	s.vlog("exclusive lock acquired")
	defer func() {
		fl.Unlock()
		s.vlog("exclusive lock released")
	}()

	// Read JSONL once and parse from memory
	jsonlData, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	tasks, err := task.ReadJSONLFromBytes(jsonlData)
	if err != nil {
		return fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}

	// Ensure cache freshness
	s.vlog("checking cache freshness via hash comparison")
	if err := cache.EnsureFresh(s.dbPath, jsonlData, tasks); err != nil {
		return fmt.Errorf("failed to ensure cache freshness: %w", err)
	}
	s.vlog("cache is fresh")

	// Apply mutation
	modified, err := fn(tasks)
	if err != nil {
		return err
	}

	// Serialize tasks to bytes for both writing and hash computation
	newJSONLData, err := task.SerializeJSONL(modified)
	if err != nil {
		return fmt.Errorf("failed to serialize tasks: %w", err)
	}

	// Atomic write to JSONL
	s.vlog("atomic write to %s", s.jsonlPath)
	if err := task.WriteJSONL(s.jsonlPath, modified); err != nil {
		return fmt.Errorf("failed to write tasks.jsonl: %w", err)
	}
	s.vlog("atomic write complete")

	// Update SQLite cache with new data
	s.vlog("rebuilding cache with new hash")
	c, err := cache.Open(s.dbPath)
	if err != nil {
		log.Printf("warning: failed to open cache for update: %v", err)
		return nil
	}
	defer c.Close()

	if err := c.Rebuild(modified, newJSONLData); err != nil {
		log.Printf("warning: failed to update cache after JSONL write: %v", err)
		return nil
	}
	s.vlog("cache rebuild complete")

	return nil
}

// Rebuild forces a complete cache rebuild from JSONL, bypassing freshness checks.
// It acquires an exclusive lock, deletes existing cache.db, reads JSONL,
// creates a new cache, inserts all tasks, and updates the hash.
// Returns the number of tasks rebuilt.
func (s *Store) Rebuild() (int, error) {
	// Acquire exclusive lock
	s.vlog("acquiring exclusive lock on %s", s.lockPath)
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return 0, fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
	}
	if !locked {
		return 0, fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
	}
	s.vlog("exclusive lock acquired")
	defer func() {
		fl.Unlock()
		s.vlog("exclusive lock released")
	}()

	// Delete existing cache.db if present
	s.vlog("deleting existing cache.db at %s", s.dbPath)
	if err := os.Remove(s.dbPath); err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to delete cache.db: %w", err)
	}

	// Read JSONL
	s.vlog("reading JSONL from %s", s.jsonlPath)
	jsonlData, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	tasks, err := task.ReadJSONLFromBytes(jsonlData)
	if err != nil {
		return 0, fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}
	s.vlog("parsed %d tasks from JSONL", len(tasks))

	// Create new cache, insert all tasks, update hash
	s.vlog("rebuilding cache with %d tasks", len(tasks))
	c, err := cache.Open(s.dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create cache: %w", err)
	}
	defer c.Close()

	if err := c.Rebuild(tasks, jsonlData); err != nil {
		return 0, fmt.Errorf("failed to rebuild cache: %w", err)
	}
	s.vlog("cache rebuild complete, hash updated")

	return len(tasks), nil
}

// Query executes the read query flow:
// 1. Acquire shared lock
// 2. Read tasks.jsonl into memory, compute SHA256 hash
// 3. Check SQLite freshness, rebuild if stale
// 4. Execute query function against SQLite
// 5. Release lock (via defer)
func (s *Store) Query(fn func(db *sql.DB) error) error {
	// Acquire shared lock
	s.vlog("acquiring shared lock on %s", s.lockPath)
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryRLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
	}
	if !locked {
		return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
	}
	s.vlog("shared lock acquired")
	defer func() {
		fl.Unlock()
		s.vlog("shared lock released")
	}()

	// Read JSONL once and parse from memory
	jsonlData, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	tasks, err := task.ReadJSONLFromBytes(jsonlData)
	if err != nil {
		return fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}

	// Ensure cache freshness
	s.vlog("checking cache freshness via hash comparison")
	if err := cache.EnsureFresh(s.dbPath, jsonlData, tasks); err != nil {
		return fmt.Errorf("failed to ensure cache freshness: %w", err)
	}
	s.vlog("cache is fresh")

	// Open cache for querying
	c, err := cache.Open(s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open cache for query: %w", err)
	}
	defer c.Close()

	// Execute query
	return fn(c.DB())
}
