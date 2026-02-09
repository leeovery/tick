package cli

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestRebuild(t *testing.T) {
	t.Run("it rebuilds cache from JSONL", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache.db was created and has the tasks.
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cachePath); err != nil {
			t.Fatalf("expected cache.db to exist: %v", err)
		}

		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("opening cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("querying task count: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 tasks in cache, got %d", count)
		}
	})

	t.Run("it handles missing cache.db (fresh build)", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		// Ensure no cache.db exists.
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		os.Remove(cachePath)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache.db was created.
		if _, err := os.Stat(cachePath); err != nil {
			t.Fatalf("expected cache.db to exist after rebuild: %v", err)
		}
	})

	t.Run("it overwrites valid existing cache", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		// Run list to populate cache.
		var out1, err1 bytes.Buffer
		Run([]string{"tick", "list"}, dir, &out1, &err1, false)

		// Add a second task directly to JSONL (bypassing cache).
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		data, err := t2.MarshalJSON()
		if err != nil {
			t.Fatalf("marshaling task: %v", err)
		}
		f, err := os.OpenFile(jsonlPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatalf("opening tasks.jsonl: %v", err)
		}
		if _, err := f.Write(data); err != nil {
			t.Fatalf("writing task data: %v", err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			t.Fatalf("writing newline: %v", err)
		}
		f.Close()

		// Rebuild should pick up both tasks.
		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify cache has both tasks.
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("opening cache.db: %v", err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err != nil {
			t.Fatalf("querying task count: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 tasks in cache after rebuild, got %d", count)
		}
	})

	t.Run("it updates hash in metadata table after rebuild", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("opening cache.db: %v", err)
		}
		defer db.Close()

		var hash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key='jsonl_hash'").Scan(&hash)
		if err != nil {
			t.Fatalf("expected hash in metadata: %v", err)
		}
		if hash == "" {
			t.Error("expected non-empty hash in metadata")
		}
	})

	t.Run("it acquires exclusive lock during rebuild", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		// Run with verbose to verify lock acquisition.
		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--verbose", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stderrStr := stderr.String()
		if !strings.Contains(stderrStr, "verbose: lock acquired (exclusive)") {
			t.Errorf("expected exclusive lock log, got:\n%s", stderrStr)
		}
		if !strings.Contains(stderrStr, "verbose: lock released") {
			t.Errorf("expected lock released log, got:\n%s", stderrStr)
		}
	})

	t.Run("it outputs confirmation message with task count", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		t3 := task.NewTask("tick-cccccc", "Task C")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if !strings.Contains(output, "3") {
			t.Errorf("expected output to contain task count '3', got %q", output)
		}
		if !strings.Contains(strings.ToLower(output), "rebuilt") {
			t.Errorf("expected output to contain 'rebuilt', got %q", output)
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it logs rebuild steps with --verbose", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--verbose", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stderrStr := stderr.String()
		expectedPhrases := []string{
			"verbose: deleting existing cache",
			"verbose: reading tasks.jsonl",
			"verbose: rebuilding cache",
			"verbose: hash updated",
		}
		for _, phrase := range expectedPhrases {
			if !strings.Contains(stderrStr, phrase) {
				t.Errorf("expected stderr to contain %q, got:\n%s", phrase, stderrStr)
			}
		}
	})

	t.Run("it handles empty JSONL (0 tasks rebuilt)", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "rebuild"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if !strings.Contains(output, "0") {
			t.Errorf("expected output to contain '0' tasks, got %q", output)
		}
	})
}
