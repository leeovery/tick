// Package storage provides a unified Store that composes the JSONL reader/writer
// and SQLite cache with file locking for safe concurrent access.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"

	"github.com/leeovery/tick/internal/storage/jsonl"
	"github.com/leeovery/tick/internal/storage/sqlite"
	"github.com/leeovery/tick/internal/task"
)

const defaultLockTimeout = 5 * time.Second

// Logger is an optional interface for verbose/debug logging.
// When set on a Store, key operations will log through it.
type Logger interface {
	Log(msg string)
}

// Store orchestrates JSONL reads/writes and SQLite cache operations
// with file locking for concurrent access safety.
type Store struct {
	tickDir     string
	jsonlPath   string
	cachePath   string
	lockPath    string
	lockTimeout time.Duration
	logger      Logger
}

// SetLogger sets an optional logger for verbose output of internal operations.
func (s *Store) SetLogger(l Logger) {
	s.logger = l
}

// logVerbose writes a message through the logger if one is set.
func (s *Store) logVerbose(msg string) {
	if s.logger != nil {
		s.logger.Log(msg)
	}
}

// NewStore creates a Store for the given .tick/ directory.
// It validates the directory exists and contains tasks.jsonl.
func NewStore(tickDir string) (*Store, error) {
	return NewStoreWithTimeout(tickDir, defaultLockTimeout)
}

// NewStoreWithTimeout creates a Store with a custom lock timeout.
func NewStoreWithTimeout(tickDir string, lockTimeout time.Duration) (*Store, error) {
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
		cachePath:   filepath.Join(tickDir, "cache.db"),
		lockPath:    filepath.Join(tickDir, "lock"),
		lockTimeout: lockTimeout,
	}, nil
}

// Close is a no-op for now; the Store opens/closes the cache per operation.
func (s *Store) Close() error {
	return nil
}

// Mutate executes a write operation with the full mutation flow:
// acquire exclusive lock -> read JSONL -> check freshness -> mutate -> atomic write -> update cache -> release lock.
func (s *Store) Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error {
	fl := flock.New(s.lockPath)

	s.logVerbose("lock: acquiring exclusive lock")
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
	if err != nil || !locked {
		return fmt.Errorf("Could not acquire lock on .tick/lock - another process may be using tick")
	}
	s.logVerbose("lock: exclusive lock acquired")
	defer func() {
		fl.Unlock()
		s.logVerbose("lock: exclusive lock released")
	}()

	// Read raw JSONL content
	rawContent, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	// Parse tasks from raw content
	tasks, err := jsonl.ParseTasks(rawContent)
	if err != nil {
		return fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}
	s.logVerbose(fmt.Sprintf("freshness: read %d tasks from JSONL", len(tasks)))

	// Check freshness and rebuild cache if stale
	s.logVerbose("freshness: checking cache hash")
	cache, err := sqlite.EnsureFresh(s.cachePath, tasks, rawContent)
	if err != nil {
		return fmt.Errorf("failed to ensure cache freshness: %w", err)
	}
	defer cache.Close()
	s.logVerbose("cache: freshness check complete")

	// Apply mutation
	modified, err := fn(tasks)
	if err != nil {
		return fmt.Errorf("mutation failed: %w", err)
	}

	// Write modified tasks atomically
	s.logVerbose("write: atomic write to tasks.jsonl")
	if err := jsonl.WriteTasks(s.jsonlPath, modified); err != nil {
		return fmt.Errorf("failed to write tasks.jsonl: %w", err)
	}
	s.logVerbose(fmt.Sprintf("write: wrote %d tasks", len(modified)))

	// Read back the written content for hash computation
	newRawContent, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		log.Printf("warning: failed to read tasks.jsonl for cache update: %v", err)
		return nil
	}

	// Update SQLite cache — if this fails, log and continue
	s.logVerbose("cache: rebuilding SQLite cache with new hash")
	if err := cache.Rebuild(modified, newRawContent); err != nil {
		log.Printf("warning: failed to update SQLite cache: %v", err)
		return nil
	}
	s.logVerbose("cache: rebuild complete")

	return nil
}

// Query executes a read operation with the full query flow:
// acquire shared lock -> read JSONL -> check freshness -> query SQLite -> release lock.
func (s *Store) Query(fn func(db *sql.DB) error) error {
	fl := flock.New(s.lockPath)

	s.logVerbose("lock: acquiring shared lock")
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryRLockContext(ctx, 10*time.Millisecond)
	if err != nil || !locked {
		return fmt.Errorf("Could not acquire lock on .tick/lock - another process may be using tick")
	}
	s.logVerbose("lock: shared lock acquired")
	defer func() {
		fl.Unlock()
		s.logVerbose("lock: shared lock released")
	}()

	// Read raw JSONL content
	rawContent, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	// Parse tasks from raw content
	tasks, err := jsonl.ParseTasks(rawContent)
	if err != nil {
		return fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}
	s.logVerbose(fmt.Sprintf("freshness: read %d tasks from JSONL", len(tasks)))

	// Check freshness and rebuild cache if stale
	s.logVerbose("freshness: checking cache hash")
	cache, err := sqlite.EnsureFresh(s.cachePath, tasks, rawContent)
	if err != nil {
		return fmt.Errorf("failed to ensure cache freshness: %w", err)
	}
	defer cache.Close()
	s.logVerbose("cache: freshness check complete")

	// Execute query
	if err := fn(cache.DB()); err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	return nil
}
