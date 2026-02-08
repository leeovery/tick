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
	"github.com/leeovery/tick/internal/task"
	_ "github.com/mattn/go-sqlite3"
)

func TestRebuild_RebuildsCacheFromJSONL(t *testing.T) {
	t.Run("it rebuilds cache from JSONL", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ccc333", Title: "Task C", Status: task.StatusDone, Priority: 3, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache.db exists and has the correct tasks
		dbPath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 tasks in cache, got %d", count)
		}
	})
}

func TestRebuild_HandlesMissingCacheDB(t *testing.T) {
	t.Run("it handles missing cache.db (fresh build)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		// Ensure no cache.db exists
		dbPath := filepath.Join(dir, ".tick", "cache.db")
		os.Remove(dbPath)

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache.db was created
		if _, err := os.Stat(dbPath); err != nil {
			t.Fatalf("expected cache.db to exist after rebuild: %v", err)
		}
	})
}

func TestRebuild_OverwritesValidExistingCache(t *testing.T) {
	t.Run("it overwrites valid existing cache", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		// First, build cache via a list command (triggers EnsureFresh)
		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("list failed: exit code %d; stderr: %s", code, stderr.String())
		}

		// Modify JSONL to add a task (cache is now stale but rebuild doesn't care)
		tasks = append(tasks, task.Task{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now})
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		if err := task.WriteJSONL(jsonlPath, tasks); err != nil {
			t.Fatalf("failed to write tasks: %v", err)
		}

		// Rebuild should replace cache regardless of freshness
		stdout.Reset()
		stderr.Reset()
		app2 := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app2.Run([]string{"tick", "rebuild"})
		if code != 0 {
			t.Fatalf("rebuild failed: exit code %d; stderr: %s", code, stderr.String())
		}

		// Verify cache has 2 tasks
		dbPath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 tasks in cache after rebuild, got %d", count)
		}
	})
}

func TestRebuild_UpdatesHashInMetadataTable(t *testing.T) {
	t.Run("it updates hash in metadata table after rebuild", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Read hash from metadata table
		dbPath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var storedHash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
		if err != nil {
			t.Fatalf("failed to read hash from metadata: %v", err)
		}
		if storedHash == "" {
			t.Error("expected non-empty hash in metadata after rebuild")
		}
	})
}

func TestRebuild_AcquiresExclusiveLock(t *testing.T) {
	t.Run("it acquires exclusive lock during rebuild", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		// Hold an exclusive lock externally
		lockPath := filepath.Join(dir, ".tick", "lock")
		fl := flock.New(lockPath)
		locked, err := fl.TryLock()
		if err != nil || !locked {
			t.Fatalf("failed to acquire external lock: %v", err)
		}
		defer fl.Unlock()

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "rebuild"})
		// Should fail because lock is held
		if code != 1 {
			t.Fatalf("expected exit code 1 when lock is held, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "lock") {
			t.Errorf("expected lock error message, got %q", errMsg)
		}
	})
}

func TestRebuild_OutputsConfirmationWithTaskCount(t *testing.T) {
	t.Run("it outputs confirmation message with task count", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ccc333", Title: "Task C", Status: task.StatusDone, Priority: 3, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "3") {
			t.Errorf("expected output to contain task count '3', got %q", output)
		}
		if !strings.Contains(output, "rebuilt") || !strings.Contains(output, "Rebuilt") {
			// Accept either case
			if !strings.Contains(strings.ToLower(output), "rebuilt") {
				t.Errorf("expected output to contain 'rebuilt', got %q", output)
			}
		}
	})
}

func TestRebuild_SuppressesOutputWithQuiet(t *testing.T) {
	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
	})
}

func TestRebuild_LogsRebuildStepsWithVerbose(t *testing.T) {
	t.Run("it logs rebuild steps with --verbose", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--verbose", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stderrStr := stderr.String()

		// Should log key rebuild steps
		if !strings.Contains(stderrStr, "verbose:") {
			t.Errorf("expected verbose output on stderr, got %q", stderrStr)
		}

		// Check for delete/read/insert/hash steps
		hasDelete := strings.Contains(strings.ToLower(stderrStr), "delet")
		hasRead := strings.Contains(strings.ToLower(stderrStr), "read") || strings.Contains(strings.ToLower(stderrStr), "jsonl")
		hasInsert := strings.Contains(strings.ToLower(stderrStr), "insert") || strings.Contains(strings.ToLower(stderrStr), "rebuild")
		hasHash := strings.Contains(strings.ToLower(stderrStr), "hash")

		if !hasDelete {
			t.Errorf("expected verbose to log delete step, got %q", stderrStr)
		}
		if !hasRead {
			t.Errorf("expected verbose to log read step, got %q", stderrStr)
		}
		if !hasInsert {
			t.Errorf("expected verbose to log insert/rebuild step, got %q", stderrStr)
		}
		if !hasHash {
			t.Errorf("expected verbose to log hash update step, got %q", stderrStr)
		}
	})
}

func TestRebuild_HandlesEmptyJSONL(t *testing.T) {
	t.Run("it handles empty JSONL (0 tasks rebuilt)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "0") {
			t.Errorf("expected output to contain '0' tasks, got %q", output)
		}

		// Verify cache.db was created with correct schema
		dbPath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("failed to count tasks: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 tasks in cache, got %d", count)
		}
	})
}
