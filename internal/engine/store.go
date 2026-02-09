// Package engine provides the Store type that composes JSONL storage and SQLite
// cache into a unified storage engine with file-locking for concurrent access safety.
package engine

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
	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

const defaultLockTimeout = 5 * time.Second

// lockTimeoutMsg is the error message returned when lock acquisition times out.
const lockTimeoutMsg = "Could not acquire lock on .tick/lock - another process may be using tick"

// Store composes the JSONL reader/writer and SQLite cache into a unified storage
// engine with file locking for concurrent access safety.
type Store struct {
	jsonlPath   string
	lockPath    string
	cache       *cache.Cache
	lockTimeout time.Duration
}

// Option configures a Store.
type Option func(*Store)

// WithLockTimeout sets a custom lock timeout duration. The default is 5 seconds.
func WithLockTimeout(d time.Duration) Option {
	return func(s *Store) {
		s.lockTimeout = d
	}
}

// NewStore creates a new Store that manages the .tick/ directory at tickDir.
// It validates that the directory exists and contains tasks.jsonl.
func NewStore(tickDir string, opts ...Option) (*Store, error) {
	info, err := os.Stat(tickDir)
	if err != nil {
		return nil, fmt.Errorf("tick directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("tick path is not a directory: %s", tickDir)
	}

	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		return nil, fmt.Errorf("tasks.jsonl not found in %s: %w", tickDir, err)
	}

	c, err := cache.New(filepath.Join(tickDir, "cache.db"))
	if err != nil {
		return nil, fmt.Errorf("opening cache: %w", err)
	}

	s := &Store{
		jsonlPath:   jsonlPath,
		lockPath:    filepath.Join(tickDir, "lock"),
		cache:       c,
		lockTimeout: defaultLockTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Close releases resources held by the Store.
func (s *Store) Close() error {
	if s.cache != nil {
		return s.cache.Close()
	}
	return nil
}

// Mutate executes a write mutation flow with exclusive file locking:
//  1. Acquire exclusive lock
//  2. Read tasks.jsonl into memory
//  3. Check cache freshness, rebuild if stale
//  4. Pass tasks to mutation function
//  5. Write modified tasks via atomic rewrite
//  6. Update SQLite cache
//  7. Release lock (via defer)
//
// If JSONL write succeeds but SQLite update fails, a warning is logged and
// the operation returns success.
func (s *Store) Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error {
	unlock, err := s.acquireExclusive()
	if err != nil {
		return err
	}
	defer unlock()

	tasks, jsonlData, err := s.readAndEnsureFresh()
	if err != nil {
		return err
	}

	// Apply mutation.
	modified, err := fn(tasks)
	if err != nil {
		return err
	}

	// Atomic write to JSONL.
	if err := storage.WriteTasks(s.jsonlPath, modified); err != nil {
		return fmt.Errorf("writing tasks.jsonl: %w", err)
	}

	// Update SQLite cache. If this fails, log warning and continue.
	_ = jsonlData // original data used for freshness; re-read for new hash
	s.updateCache(modified)

	return nil
}

// Query executes a read query flow with shared file locking:
//  1. Acquire shared lock
//  2. Read tasks.jsonl into memory
//  3. Check cache freshness, rebuild if stale
//  4. Execute query function against SQLite
//  5. Release lock (via defer)
func (s *Store) Query(fn func(db *sql.DB) error) error {
	unlock, err := s.acquireShared()
	if err != nil {
		return err
	}
	defer unlock()

	if _, _, err := s.readAndEnsureFresh(); err != nil {
		return err
	}

	return fn(s.cache.DB())
}

// acquireExclusive acquires an exclusive file lock with the configured timeout.
// It returns an unlock function that must be deferred by the caller.
func (s *Store) acquireExclusive() (unlock func(), err error) {
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)

	locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
	if !locked || err != nil {
		cancel()
		return nil, fmt.Errorf("%s", lockTimeoutMsg)
	}

	return func() {
		_ = fl.Unlock()
		cancel()
	}, nil
}

// acquireShared acquires a shared file lock with the configured timeout.
// It returns an unlock function that must be deferred by the caller.
func (s *Store) acquireShared() (unlock func(), err error) {
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)

	locked, err := fl.TryRLockContext(ctx, 50*time.Millisecond)
	if !locked || err != nil {
		cancel()
		return nil, fmt.Errorf("%s", lockTimeoutMsg)
	}

	return func() {
		_ = fl.Unlock()
		cancel()
	}, nil
}

// readAndEnsureFresh reads tasks.jsonl once, parses it, and ensures the SQLite
// cache is up-to-date. Returns the parsed tasks and raw JSONL bytes.
func (s *Store) readAndEnsureFresh() ([]task.Task, []byte, error) {
	jsonlData, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading tasks.jsonl: %w", err)
	}

	tasks, err := storage.ParseTasks(jsonlData)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing tasks.jsonl: %w", err)
	}

	if err := s.ensureFresh(tasks, jsonlData); err != nil {
		return nil, nil, err
	}

	return tasks, jsonlData, nil
}

// ensureFresh checks if the cache is up-to-date with the given JSONL data and
// rebuilds it if stale.
func (s *Store) ensureFresh(tasks []task.Task, jsonlData []byte) error {
	fresh, err := s.cache.IsFresh(jsonlData)
	if err != nil {
		log.Printf("warning: cache freshness check failed, rebuilding: %v", err)
		fresh = false
	}

	if !fresh {
		if err := s.cache.Rebuild(tasks, jsonlData); err != nil {
			return fmt.Errorf("rebuilding cache: %w", err)
		}
	}

	return nil
}

// updateCache re-reads the JSONL file and rebuilds the SQLite cache. If any step
// fails, a warning is logged to stderr and the error is swallowed (JSONL is the
// source of truth; the cache self-heals on next read).
func (s *Store) updateCache(tasks []task.Task) {
	newJSONLData, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		log.Printf("warning: could not read tasks.jsonl after write for cache update: %v", err)
		return
	}

	if err := s.cache.Rebuild(tasks, newJSONLData); err != nil {
		log.Printf("warning: SQLite cache update failed after successful JSONL write: %v", err)
	}
}
