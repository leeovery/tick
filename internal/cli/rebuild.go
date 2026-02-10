package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/storage"
)

// RunRebuild executes the rebuild command: forces a complete SQLite cache rebuild
// from JSONL, bypassing the freshness check. Acquires an exclusive lock during
// the entire operation to prevent concurrent access.
func RunRebuild(dir string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	cachePath := filepath.Join(tickDir, "cache.db")
	lockPath := filepath.Join(tickDir, "lock")

	// Acquire exclusive lock.
	fileLock := flock.New(lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fc.Logger.Log("acquiring exclusive lock")
	locked, err := fileLock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil || !locked {
		return fmt.Errorf("could not acquire lock on .tick/lock - another process may be using tick")
	}
	defer func() {
		_ = fileLock.Unlock()
		fc.Logger.Log("lock released")
	}()
	fc.Logger.Log("lock acquired")

	// Delete existing cache.db if present.
	fc.Logger.Log("deleting cache.db")
	os.Remove(cachePath)

	// Read JSONL.
	fc.Logger.Log("reading JSONL")
	rawJSONL, err := os.ReadFile(jsonlPath)
	if err != nil {
		return fmt.Errorf("failed to read tasks.jsonl: %w", err)
	}

	tasks, err := storage.ParseJSONL(rawJSONL)
	if err != nil {
		return fmt.Errorf("failed to parse tasks.jsonl: %w", err)
	}

	// Open fresh cache (creates schema).
	cache, err := storage.OpenCache(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}
	defer cache.Close()

	// Rebuild cache from tasks.
	fc.Logger.Log(fmt.Sprintf("rebuilding cache with %d tasks", len(tasks)))
	if err := cache.Rebuild(tasks, rawJSONL); err != nil {
		return fmt.Errorf("failed to rebuild cache: %w", err)
	}
	fc.Logger.Log("hash updated")

	// Output confirmation.
	if !fc.Quiet {
		msg := fmt.Sprintf("Cache rebuilt: %d tasks", len(tasks))
		fmt.Fprintln(stdout, fmtr.FormatMessage(msg))
	}

	return nil
}
