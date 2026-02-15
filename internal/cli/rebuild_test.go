package cli

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// runRebuild runs the tick rebuild command with the given args and returns stdout, stderr, and exit code.
func runRebuild(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  false, // default to TOON format
	}
	fullArgs := append([]string{"tick", "rebuild"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestRebuild(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

	t.Run("it rebuilds cache from JSONL", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task two", Status: task.StatusDone, Priority: 1, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runRebuild(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		// Verify cache.db was created and contains the tasks.
		cachePath := filepath.Join(tickDir, "cache.db")
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatal("cache.db should exist after rebuild")
		}

		// Open cache and verify task count.
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to query task count: %v", err)
		}
		if count != 2 {
			t.Errorf("task count = %d, want 2", count)
		}

		// Verify output mentions the count.
		if !strings.Contains(stdout, "2") {
			t.Errorf("stdout should contain task count '2', got %q", stdout)
		}
		_ = stderr
	})

	t.Run("it handles missing cache.db (fresh build)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		// Ensure no cache.db exists.
		cachePath := filepath.Join(tickDir, "cache.db")
		os.Remove(cachePath)
		if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
			t.Fatal("cache.db should not exist before rebuild")
		}

		stdout, stderr, exitCode := runRebuild(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		// Cache should now exist.
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatal("cache.db should exist after rebuild")
		}

		if !strings.Contains(stdout, "1") {
			t.Errorf("stdout should contain task count '1', got %q", stdout)
		}
	})

	t.Run("it overwrites valid existing cache", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		// Run rebuild once to create cache.
		_, _, exitCode := runRebuild(t, dir)
		if exitCode != 0 {
			t.Fatal("first rebuild failed")
		}

		// Add another task to JSONL.
		tasks = append(tasks, task.Task{
			ID: "tick-bbb222", Title: "Task two", Status: task.StatusOpen, Priority: 1,
			Created: now.Add(time.Second), Updated: now.Add(time.Second),
		})
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		data, err := storage.MarshalJSONL(tasks)
		if err != nil {
			t.Fatalf("failed to marshal tasks: %v", err)
		}
		if err := os.WriteFile(jsonlPath, data, 0644); err != nil {
			t.Fatalf("failed to write tasks.jsonl: %v", err)
		}

		// Rebuild again — should overwrite the old cache.
		_, stderr, exitCode := runRebuild(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		// Verify new cache has 2 tasks.
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to query task count: %v", err)
		}
		if count != 2 {
			t.Errorf("task count = %d, want 2", count)
		}
	})

	t.Run("it updates hash in metadata table after rebuild", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runRebuild(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		// Verify metadata table has the hash.
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var hash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&hash)
		if err != nil {
			t.Fatalf("failed to query hash: %v", err)
		}
		if hash == "" {
			t.Error("hash should not be empty after rebuild")
		}
		// Hash should be a 64-char hex string (SHA256).
		if len(hash) != 64 {
			t.Errorf("hash length = %d, want 64", len(hash))
		}
	})

	t.Run("it acquires exclusive lock during rebuild", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		// Hold the lock to simulate concurrent access.
		lockPath := filepath.Join(tickDir, "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire test lock: %v", err)
		}
		defer func() { _ = fl.Unlock() }()

		// Rebuild should fail because the lock is held.
		_, stderr, exitCode := runRebuild(t, dir)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1 (lock held)", exitCode)
		}
		if !strings.Contains(stderr, "lock") {
			t.Errorf("stderr should mention lock, got %q", stderr)
		}
	})

	t.Run("it outputs confirmation message with task count", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task two", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ccc333", Title: "Task three", Status: task.StatusDone, Priority: 2, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second), Closed: &now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runRebuild(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "Cache rebuilt: 3 tasks\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runRebuild(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if stdout != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout)
		}

		// But the cache should still be rebuilt.
		cachePath := filepath.Join(tickDir, "cache.db")
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatal("cache.db should exist after quiet rebuild")
		}
	})

	t.Run("it logs rebuild steps with --verbose", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runRebuild(t, dir, "--verbose")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		// Verbose output should go to stderr.
		if stderr == "" {
			t.Fatal("stderr should contain verbose output")
		}

		// Check for expected verbose messages.
		expectedMessages := []string{
			"deleting cache.db",
			"reading JSONL",
			"rebuilding cache with",
			"hash updated",
		}
		for _, msg := range expectedMessages {
			if !strings.Contains(stderr, msg) {
				t.Errorf("stderr should contain %q, got %q", msg, stderr)
			}
		}

		// All verbose lines should be prefixed.
		for _, line := range strings.Split(strings.TrimSpace(stderr), "\n") {
			if !strings.HasPrefix(line, "verbose: ") {
				t.Errorf("verbose line %q does not have 'verbose: ' prefix", line)
			}
		}

		// Stdout should still have the confirmation message.
		if !strings.Contains(stdout, "Cache rebuilt:") {
			t.Errorf("stdout should contain confirmation, got %q", stdout)
		}
	})

	t.Run("it handles empty JSONL with 0 tasks rebuilt", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		stdout, stderr, exitCode := runRebuild(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "Cache rebuilt: 0 tasks\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}

		// Cache should still exist with correct schema.
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to query task count: %v", err)
		}
		if count != 0 {
			t.Errorf("task count = %d, want 0", count)
		}
	})
}
