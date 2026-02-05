// Package storage provides JSONL file storage and SQLite cache for tasks.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/task"
)

// Store orchestrates JSONL storage and SQLite cache with file locking.
// It provides atomic read and write operations that ensure data consistency
// across concurrent access.
type Store struct {
	tickDir   string
	jsonlPath string
	cachePath string
	lockPath  string
	flock     *flock.Flock
}

// lockTimeout is the maximum time to wait for acquiring a lock.
const lockTimeout = 5 * time.Second

// NewStore creates a new Store for the given .tick directory.
// The directory must exist and contain tasks.jsonl.
func NewStore(tickDir string) (*Store, error) {
	// Validate directory exists
	info, err := os.Stat(tickDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", tickDir)
	}

	// Validate tasks.jsonl exists
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		return nil, err
	}

	return &Store{
		tickDir:   tickDir,
		jsonlPath: jsonlPath,
		cachePath: filepath.Join(tickDir, "cache.db"),
		lockPath:  filepath.Join(tickDir, "lock"),
		flock:     flock.New(filepath.Join(tickDir, "lock")),
	}, nil
}

// Close releases any resources held by the Store.
func (s *Store) Close() error {
	// Nothing to close currently - cache is opened/closed per operation
	return nil
}

// Mutate executes a write mutation flow:
// 1. Acquire exclusive lock
// 2. Read JSONL + check freshness
// 3. Pass tasks to mutation function
// 4. Write modified tasks via atomic rewrite
// 5. Update SQLite cache
// 6. Release lock
//
// If JSONL write succeeds but SQLite update fails, a warning is logged
// but success is returned (next read will self-heal).
func (s *Store) Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error {
	// Acquire exclusive lock with timeout
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	locked, err := s.flock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		if errors.Is(err, context.DeadlineExceeded) || !locked {
			return errors.New("could not acquire lock on .tick/lock - another process may be using tick")
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer s.flock.Unlock()

	// Read JSONL and compute hash
	tasks, jsonlContent, err := s.readJSONLWithContent()
	if err != nil {
		return err
	}

	// Ensure cache is fresh
	cache, err := EnsureFresh(s.cachePath, tasks, jsonlContent)
	if err != nil {
		return err
	}
	defer cache.Close()

	// Apply mutation
	modified, err := fn(tasks)
	if err != nil {
		return err
	}

	// Write modified tasks atomically
	if err := WriteJSONL(s.jsonlPath, modified); err != nil {
		return err
	}

	// Read new content for hash
	newContent, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		// JSONL write succeeded, this is unexpected - log warning and continue
		fmt.Fprintf(os.Stderr, "warning: failed to read JSONL after write: %v\n", err)
		return nil
	}

	// Update SQLite cache
	if err := cache.Rebuild(modified, newContent); err != nil {
		// JSONL write succeeded, SQLite failed - log warning and continue
		fmt.Fprintf(os.Stderr, "warning: failed to update SQLite cache: %v (will self-heal on next read)\n", err)
		return nil
	}

	return nil
}

// Query executes a read query flow:
// 1. Acquire shared lock
// 2. Read JSONL + check freshness
// 3. Execute query against SQLite
// 4. Release lock
func (s *Store) Query(fn func(db *sql.DB) error) error {
	// Acquire shared lock with timeout
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	locked, err := s.flock.TryRLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		if errors.Is(err, context.DeadlineExceeded) || !locked {
			return errors.New("could not acquire lock on .tick/lock - another process may be using tick")
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer s.flock.Unlock()

	// Read JSONL and compute hash
	tasks, jsonlContent, err := s.readJSONLWithContent()
	if err != nil {
		return err
	}

	// Ensure cache is fresh
	cache, err := EnsureFresh(s.cachePath, tasks, jsonlContent)
	if err != nil {
		return err
	}
	defer cache.Close()

	// Execute query
	return fn(cache.db)
}

// Rebuild forces a complete rebuild of the SQLite cache from JSONL.
// Acquires exclusive lock, deletes existing cache.db, reads JSONL,
// creates fresh cache with schema, inserts all tasks, and updates hash.
// Returns the number of tasks rebuilt.
func (s *Store) Rebuild() (int, error) {
	// Acquire exclusive lock with timeout
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	locked, err := s.flock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		if errors.Is(err, context.DeadlineExceeded) || !locked {
			return 0, errors.New("could not acquire lock on .tick/lock - another process may be using tick")
		}
		return 0, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer s.flock.Unlock()

	// Delete existing cache.db if present
	os.Remove(s.cachePath)

	// Read JSONL and compute hash
	tasks, jsonlContent, err := s.readJSONLWithContent()
	if err != nil {
		return 0, err
	}

	// Create fresh cache with schema
	cache, err := NewCache(s.cachePath)
	if err != nil {
		return 0, err
	}
	defer cache.Close()

	// Rebuild: insert all tasks and update hash
	if err := cache.Rebuild(tasks, jsonlContent); err != nil {
		return 0, err
	}

	return len(tasks), nil
}

// readJSONLWithContent reads JSONL file and returns both parsed tasks and raw content.
func (s *Store) readJSONLWithContent() ([]task.Task, []byte, error) {
	content, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return nil, nil, err
	}

	tasks, err := ReadJSONL(s.jsonlPath)
	if err != nil {
		return nil, nil, err
	}

	return tasks, content, nil
}
