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

// LogFunc is a function that receives verbose log messages.
type LogFunc func(format string, args ...any)

// Store orchestrates JSONL storage, SQLite cache, and file locking.
type Store struct {
	tickDir     string
	jsonlPath   string
	cachePath   string
	lockPath    string
	cache       *Cache
	lockTimeout time.Duration
	log         LogFunc
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

// SetLogger sets a function to receive verbose log messages.
func (s *Store) SetLogger(fn LogFunc) {
	s.log = fn
}

func (s *Store) logf(format string, args ...any) {
	if s.log != nil {
		s.log(format, args...)
	}
}

// Close releases the cache database connection.
func (s *Store) Close() error {
	if s.cache != nil {
		return s.cache.Close()
	}
	return nil
}

func (s *Store) acquireExclusiveLock() (*flock.Flock, error) {
	s.logf("acquiring exclusive lock")
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		return nil, fmt.Errorf("could not acquire lock on .tick/lock - another process may be using tick")
	}
	s.logf("exclusive lock acquired")
	return fl, nil
}

func (s *Store) acquireSharedLock() (*flock.Flock, error) {
	s.logf("acquiring shared lock")
	fl := flock.New(s.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	locked, err := fl.TryRLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		return nil, fmt.Errorf("could not acquire lock on .tick/lock - another process may be using tick")
	}
	s.logf("shared lock acquired")
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

	fresh, freshErr := s.cache.IsFresh(content)
	if freshErr == nil && fresh {
		s.logf("cache fresh (hash match)")
	} else {
		s.logf("cache stale, rebuilding (%d tasks)", len(tasks))
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
	s.logf("atomic write (%d tasks)", len(modified))
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

// ForceRebuild deletes the existing cache and rebuilds from JSONL,
// bypassing the freshness check. Acquires exclusive lock.
// Returns the number of tasks rebuilt.
func (s *Store) ForceRebuild() (int, error) {
	fl, err := s.acquireExclusiveLock()
	if err != nil {
		return 0, err
	}
	defer fl.Unlock()

	// Delete existing cache.
	s.logf("deleting existing cache")
	if err := s.cache.Close(); err != nil {
		return 0, fmt.Errorf("closing cache: %w", err)
	}
	os.Remove(s.cachePath)

	// Reopen cache (creates fresh).
	cache, err := NewCacheWithRecovery(s.cachePath)
	if err != nil {
		return 0, fmt.Errorf("reopening cache: %w", err)
	}
	s.cache = cache

	// Read JSONL.
	content, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading tasks.jsonl: %w", err)
	}

	tasks, err := ReadJSONLBytes(content)
	if err != nil {
		return 0, fmt.Errorf("parsing tasks.jsonl: %w", err)
	}

	s.logf("rebuilding cache (%d tasks)", len(tasks))
	if err := s.cache.Rebuild(tasks, content); err != nil {
		return 0, fmt.Errorf("rebuilding cache: %w", err)
	}

	return len(tasks), nil
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
