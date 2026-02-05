package cli

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestRebuildCommand(t *testing.T) {
	t.Run("it rebuilds cache from JSONL", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")
		setupTask(t, dir, "tick-c3d4", "Task two")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache exists and has tasks
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 tasks in cache, got %d", count)
		}
	})

	t.Run("it handles missing cache.db (fresh build)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")

		// Ensure no cache.db exists
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		os.Remove(cachePath)

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache was created
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Error("expected cache.db to be created")
		}

		// Verify task is in cache
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 task in cache, got %d", count)
		}
	})

	t.Run("it overwrites valid existing cache", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")

		// Run a list to build cache first
		var stdout1, stderr1 bytes.Buffer
		app1 := &App{Stdout: &stdout1, Stderr: &stderr1, Cwd: dir}
		app1.Run([]string{"tick", "--pretty", "list"})

		// Add another task directly to JSONL (bypassing cache update)
		setupTask(t, dir, "tick-c3d4", "Task two")

		// Now rebuild
		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache has both tasks
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 tasks in cache after rebuild, got %d", count)
		}
	})

	t.Run("it updates hash in metadata table after rebuild", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Check metadata table has hash
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache: %v", err)
		}
		defer db.Close()

		var hash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&hash)
		if err != nil {
			t.Fatalf("failed to query hash: %v", err)
		}
		if hash == "" {
			t.Error("expected hash to be set in metadata table")
		}
		// SHA256 produces 64 hex chars
		if len(hash) != 64 {
			t.Errorf("expected 64-char SHA256 hash, got %d chars: %q", len(hash), hash)
		}
	})

	t.Run("it acquires exclusive lock during rebuild", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		// Simply verify rebuild succeeds (lock is implicit in the operation)
		code := app.Run([]string{"tick", "--pretty", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify verbose output mentions lock when enabled
		var stdoutV, stderrV bytes.Buffer
		appV := &App{Stdout: &stdoutV, Stderr: &stderrV, Cwd: dir}

		code = appV.Run([]string{"tick", "--pretty", "--verbose", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderrV.String())
		}

		verboseOutput := stderrV.String()
		if !strings.Contains(verboseOutput, "lock acquire exclusive") {
			t.Errorf("verbose output should mention lock acquire, got %q", verboseOutput)
		}
	})

	t.Run("it outputs confirmation message with task count", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")
		setupTask(t, dir, "tick-c3d4", "Task two")
		setupTask(t, dir, "tick-e5f6", "Task three")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "3") {
			t.Errorf("output should contain task count 3, got %q", output)
		}
		if !strings.Contains(output, "Rebuilt") {
			t.Errorf("output should contain 'Rebuilt', got %q", output)
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--quiet", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected empty output with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it logs rebuild steps with --verbose", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Task one")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "--verbose", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		verboseOutput := stderr.String()

		// Check for key verbose steps
		if !strings.Contains(verboseOutput, "lock acquire exclusive") {
			t.Errorf("verbose should log lock acquire, got %q", verboseOutput)
		}
		if !strings.Contains(verboseOutput, "delete") {
			t.Errorf("verbose should log cache delete step, got %q", verboseOutput)
		}
		if !strings.Contains(verboseOutput, "read") || !strings.Contains(verboseOutput, "JSONL") {
			t.Errorf("verbose should log JSONL read step, got %q", verboseOutput)
		}
		if !strings.Contains(verboseOutput, "insert") || !strings.Contains(verboseOutput, "1") {
			t.Errorf("verbose should log insert count, got %q", verboseOutput)
		}
		if !strings.Contains(verboseOutput, "hash") {
			t.Errorf("verbose should log hash update, got %q", verboseOutput)
		}
		if !strings.Contains(verboseOutput, "lock release") {
			t.Errorf("verbose should log lock release, got %q", verboseOutput)
		}
	})

	t.Run("it handles empty JSONL with 0 tasks rebuilt", func(t *testing.T) {
		dir := setupTickDir(t)
		// No tasks added - empty JSONL

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "rebuild"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "0") {
			t.Errorf("output should show 0 tasks rebuilt, got %q", output)
		}

		// Verify cache exists with correct schema
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 tasks in cache, got %d", count)
		}
	})
}
