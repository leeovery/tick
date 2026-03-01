package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/task"
)

const defaultLockTimeout = 5 * time.Second

const lockErrMsg = "could not acquire lock on .tick/lock - another process may be using tick"

// Store orchestrates JSONL persistence and SQLite cache with file locking.
type Store struct {
	tickDir     string
	jsonlPath   string
	cachePath   string
	lockTimeout time.Duration
	fileLock    *flock.Flock
	cache       *Cache
	// verboseLog is an optional logging function for verbose debug output.
	// When non-nil, key operations log debug messages through it.
	verboseLog func(msg string)
}

// StoreOption configures a Store.
type StoreOption func(*Store)

// WithLockTimeout sets the lock acquisition timeout.
func WithLockTimeout(d time.Duration) StoreOption {
	return func(s *Store) {
		s.lockTimeout = d
	}
}

// WithVerbose sets a logging function for verbose debug output.
// Key operations (lock, cache, hash, write) will call this function.
func WithVerbose(fn func(msg string)) StoreOption {
	return func(s *Store) {
		s.verboseLog = fn
	}
}

// NewStore creates a Store that orchestrates JSONL and SQLite cache operations.
// The tickDir must be an existing .tick/ directory containing a tasks.jsonl file.
func NewStore(tickDir string, opts ...StoreOption) (*Store, error) {
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tasks.jsonl not found in %s", tickDir)
	}

	s := &Store{
		tickDir:     tickDir,
		jsonlPath:   jsonlPath,
		cachePath:   filepath.Join(tickDir, "cache.db"),
		lockTimeout: defaultLockTimeout,
		fileLock:    flock.New(filepath.Join(tickDir, "lock")),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// verbose logs a message if verbose logging is enabled.
func (s *Store) verbose(msg string) {
	if s.verboseLog != nil {
		s.verboseLog(msg)
	}
}

// Close releases any resources held by the Store, including the SQLite cache connection.
func (s *Store) Close() error {
	if s.cache != nil {
		return s.cache.Close()
	}
	return nil
}

// acquireExclusive acquires an exclusive file lock and returns an unlock function.
func (s *Store) acquireExclusive() (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	s.verbose("acquiring exclusive lock")
	locked, err := s.fileLock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		return nil, errors.New(lockErrMsg)
	}
	s.verbose("lock acquired")
	return func() {
		_ = s.fileLock.Unlock()
		s.verbose("lock released")
	}, nil
}

// acquireShared acquires a shared file lock and returns an unlock function.
func (s *Store) acquireShared() (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()

	s.verbose("acquiring shared lock")
	locked, err := s.fileLock.TryRLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		return nil, errors.New(lockErrMsg)
	}
	s.verbose("lock acquired")
	return func() {
		_ = s.fileLock.Unlock()
		s.verbose("lock released")
	}, nil
}

// removeCache removes the cache file, ignoring not-exist errors.
func (s *Store) removeCache() error {
	if err := os.Remove(s.cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing cache: %w", err)
	}
	return nil
}

// ReadTasks reads and parses the JSONL file under a shared lock without writing.
// This is a lightweight read-only operation that does not touch the cache or JSONL file.
func (s *Store) ReadTasks() ([]task.Task, error) {
	unlock, err := s.acquireShared()
	if err != nil {
		return nil, err
	}
	defer unlock()

	s.verbose("reading JSONL (read-only)")
	rawJSONL, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	tasks, err := ParseJSONL(rawJSONL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}

	return tasks, nil
}

// Mutate executes a write mutation with exclusive file locking.
// The full flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock.
func (s *Store) Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error {
	unlock, err := s.acquireExclusive()
	if err != nil {
		return err
	}
	defer unlock()

	_, tasks, err := s.readAndEnsureFresh()
	if err != nil {
		return err
	}

	// Apply mutation.
	mutated, err := fn(tasks)
	if err != nil {
		return err
	}

	// Marshal to bytes once — used for both atomic write and cache rebuild (no re-read).
	newRawJSONL, err := MarshalJSONL(mutated)
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Atomic write to JSONL.
	s.verbose("writing JSONL atomically")
	if err := WriteJSONLRaw(s.jsonlPath, newRawJSONL); err != nil {
		return fmt.Errorf("failed to write tasks.jsonl: %w", err)
	}

	// Rebuild cache from the same bytes that were written — no re-read needed.
	s.verbose("rebuilding cache from JSONL")
	if err := s.cache.Rebuild(mutated, newRawJSONL); err != nil {
		log.Printf("warning: failed to update cache after write: %v", err)
		// Close the corrupted cache so it will be recreated on next use.
		s.cache.Close()
		s.cache = nil
		return nil
	}

	return nil
}

// Rebuild forces a complete cache rebuild from JSONL. It acquires an exclusive lock,
// deletes the existing cache.db, reads tasks.jsonl, creates a fresh cache, and populates it.
// Returns the number of tasks rebuilt.
func (s *Store) Rebuild() (int, error) {
	unlock, err := s.acquireExclusive()
	if err != nil {
		return 0, err
	}
	defer unlock()

	// Close existing cache if open.
	if s.cache != nil {
		s.cache.Close()
		s.cache = nil
	}

	// Delete existing cache.db.
	s.verbose("deleting cache.db")
	if err := s.removeCache(); err != nil {
		return 0, err
	}

	// Read JSONL.
	s.verbose("reading JSONL")
	rawJSONL, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	tasks, err := ParseJSONL(rawJSONL)
	if err != nil {
		return 0, fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}

	// Open fresh cache (creates schema).
	cache, err := OpenCache(s.cachePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create cache: %w", err)
	}
	s.cache = cache

	// Rebuild cache from tasks.
	s.verbose(fmt.Sprintf("rebuilding cache with %d tasks", len(tasks)))
	if err := s.cache.Rebuild(tasks, rawJSONL); err != nil {
		return 0, fmt.Errorf("failed to rebuild cache: %w", err)
	}
	s.verbose("hash updated")

	return len(tasks), nil
}

// Query executes a read query with shared file locking.
// The full flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock.
func (s *Store) Query(fn func(db *sql.DB) error) error {
	unlock, err := s.acquireShared()
	if err != nil {
		return err
	}
	defer unlock()

	_, _, err = s.readAndEnsureFresh()
	if err != nil {
		return err
	}

	return fn(s.cache.DB())
}

// ResolveID resolves a user-supplied ID input (with or without tick- prefix, any case)
// to a canonical full task ID. Exact full-ID match bypasses prefix search. Minimum 3 hex
// chars required for prefix matching. Returns an error for ambiguous or not-found inputs.
func (s *Store) ResolveID(input string) (string, error) {
	originalInput := input

	// Strip tick- prefix (case-insensitive).
	lower := strings.ToLower(input)
	hex := lower
	if strings.HasPrefix(lower, "tick-") {
		hex = lower[5:]
	}

	// Minimum length check.
	if len(hex) < 3 {
		return "", errors.New("partial ID must be at least 3 hex characters")
	}

	// Single query call: exact match first (6 hex chars), then prefix search fallback.
	var resolved string
	err := s.Query(func(db *sql.DB) error {
		fullID := "tick-" + hex

		// Exact full-ID match: 6 hex chars -> try exact match first.
		if len(hex) == 6 {
			var found string
			scanErr := db.QueryRow("SELECT id FROM tasks WHERE id = ?", fullID).Scan(&found)
			if scanErr == nil {
				resolved = found
				return nil
			}
			// If not found, fall through to prefix search.
		}

		// Prefix search.
		rows, err := db.Query("SELECT id FROM tasks WHERE id LIKE ? ORDER BY id", fullID+"%")
		if err != nil {
			return err
		}
		defer rows.Close()

		var matches []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			matches = append(matches, id)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		switch len(matches) {
		case 0:
			return fmt.Errorf("task '%s' not found", originalInput)
		case 1:
			resolved = matches[0]
			return nil
		default:
			return fmt.Errorf("ambiguous ID '%s' matches: %s", originalInput, strings.Join(matches, ", "))
		}
	})
	if err != nil {
		return "", err
	}

	return resolved, nil
}

// readAndEnsureFresh reads JSONL once, parses tasks, and ensures the SQLite cache is up-to-date.
// The file is read exactly once — the same bytes are used for both parsing and hash computation.
func (s *Store) readAndEnsureFresh() ([]byte, []task.Task, error) {
	rawJSONL, err := os.ReadFile(s.jsonlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	tasks, err := ParseJSONL(rawJSONL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}

	if err := s.ensureFresh(rawJSONL, tasks); err != nil {
		return nil, nil, fmt.Errorf("failed to ensure cache freshness: %w", err)
	}

	return rawJSONL, tasks, nil
}

// ensureFresh checks if the persistent cache is up-to-date with the given JSONL content.
// It opens the cache on first use (lazy init) and handles corruption by closing, deleting, and reopening.
func (s *Store) ensureFresh(rawJSONL []byte, tasks []task.Task) error {
	// Lazy init: open cache on first use.
	if s.cache == nil {
		cache, err := OpenCache(s.cachePath)
		if err != nil {
			// Cache file might be corrupted — delete and recreate.
			log.Printf("warning: cache open failed, recreating: %v", err)
			if rmErr := s.removeCache(); rmErr != nil {
				return rmErr
			}
			cache, err = OpenCache(s.cachePath)
			if err != nil {
				return fmt.Errorf("failed to recreate cache: %w", err)
			}
		}
		s.cache = cache
	}

	fresh, err := s.cache.IsFresh(rawJSONL)
	if err != nil {
		// Query error — corrupted schema. Close, delete, and recreate.
		log.Printf("warning: cache freshness check failed, recreating: %v", err)
		s.cache.Close()
		s.cache = nil
		if rmErr := s.removeCache(); rmErr != nil {
			return rmErr
		}
		cache, err := OpenCache(s.cachePath)
		if err != nil {
			return fmt.Errorf("failed to recreate cache after corruption: %w", err)
		}
		s.cache = cache
		fresh = false
	}

	if fresh {
		s.verbose("hash match: yes")
		s.verbose("cache is fresh")
	} else {
		s.verbose("hash match: no")
		s.verbose("rebuilding cache from JSONL")
		if err := s.cache.Rebuild(tasks, rawJSONL); err != nil {
			return fmt.Errorf("failed to rebuild cache: %w", err)
		}
	}

	return nil
}
