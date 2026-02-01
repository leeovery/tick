package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"

	"github.com/leeovery/tick/internal/task"
)

const defaultLockTimeout = 5 * time.Second

// Store orchestrates JSONL storage, SQLite cache, and file locking.
type Store struct {
	tickDir     string
	jsonlPath   string
	cachePath   string
	lockPath    string
	cache       *Cache
	lockTimeout time.Duration
}

// NewStore creates a new Store for the given .tick directory.
// The directory must exist and contain tasks.jsonl.
func NewStore(tickDir string) (*Store, error) {
	info, err := os.Stat(tickDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("tick directory does not exist: %s", tickDir)
	}

	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		return nil, fmt.Errorf("tasks.jsonl not found in %s", tickDir)
	}

	cachePath := filepath.Join(tickDir, "cache.db")
	cache, err := NewCacheWithRecovery(cachePath)
	if err != nil {
		return nil, fmt.Errorf("opening cache: %w", err)
	}

	return &Store{
		tickDir:     tickDir,
		jsonlPath:   jsonlPath,
		cachePath:   cachePath,
		lockPath:    filepath.Join(tickDir, "lock"),
		cache:       cache,
		lockTimeout: defaultLockTimeout,
	}, nil
}

// Close releases the cache database connection.
func (s *Store) Close() error {
	if s.cache != nil {
		return s.cache.Close()
	}
	return nil
}

func (s *Store) acquireExclusiveLock() (*flock.Flock, error) {
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		return nil, fmt.Errorf("could not acquire lock on .tick/lock - another process may be using tick")
	}
	return fl, nil
}

func (s *Store) acquireSharedLock() (*flock.Flock, error) {
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryRLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		return nil, fmt.Errorf("could not acquire lock on .tick/lock - another process may be using tick")
	}
	return fl, nil
}

func (s *Store) readAndEnsureFresh() ([]byte, []task.Task, error) {
	content, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading tasks.jsonl: %w", err)
	}

	tasks, err := ReadJSONLBytes(content)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing tasks.jsonl: %w", err)
	}

	if err := s.cache.EnsureFresh(content, tasks); err != nil {
		return nil, nil, fmt.Errorf("ensuring cache freshness: %w", err)
	}

	return content, tasks, nil
}

// Mutate executes a write operation with exclusive locking.
// The mutation function receives the current tasks and returns the modified list.
func (s *Store) Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error {
	fl, err := s.acquireExclusiveLock()
	if err != nil {
		return err
	}
	defer fl.Unlock()

	_, tasks, err := s.readAndEnsureFresh()
	if err != nil {
		return err
	}

	modified, err := fn(tasks)
	if err != nil {
		return fmt.Errorf("mutation failed: %w", err)
	}

	// Atomic JSONL write
	if err := WriteJSONL(s.jsonlPath, modified); err != nil {
		return fmt.Errorf("writing tasks.jsonl: %w", err)
	}

	// Update SQLite cache — if this fails, log and continue
	newContent, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to read tasks.jsonl for cache update: %v\n", err)
		return nil
	}
	if err := s.cache.Rebuild(modified, newContent); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cache update failed (will self-heal on next read): %v\n", err)
	}

	return nil
}

// Query executes a read operation with shared locking.
// The query function receives the SQLite database connection.
func (s *Store) Query(fn func(db *sql.DB) error) error {
	fl, err := s.acquireSharedLock()
	if err != nil {
		return err
	}
	defer fl.Unlock()

	if _, _, err := s.readAndEnsureFresh(); err != nil {
		return err
	}

	return fn(s.cache.DB())
}
