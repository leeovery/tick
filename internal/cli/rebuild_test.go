package cli

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestRebuild(t *testing.T) {
	t.Run("it rebuilds cache from JSONL", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Task two", "done", 1, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-ccc333", "Task three", "open", 0, []string{"tick-aaa111"}, "", "2026-01-19T12:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		// Verify cache.db exists and has the tasks
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		if count != 3 {
			t.Errorf("task count = %d, want 3", count)
		}

		// Verify dependencies
		var depCount int
		err = db.QueryRow("SELECT COUNT(*) FROM dependencies").Scan(&depCount)
		if err != nil {
			t.Fatalf("failed to query dependencies: %v", err)
		}
		if depCount != 1 {
			t.Errorf("dependency count = %d, want 1", depCount)
		}
	})

	t.Run("it handles missing cache.db (fresh build)", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		// Ensure no cache.db exists
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
			t.Fatalf("cache.db should not exist before rebuild, stat error: %v", err)
		}

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		// Verify cache.db was created
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatal("cache.db should exist after rebuild")
		}

		// Verify task count in new cache
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		if count != 1 {
			t.Errorf("task count = %d, want 1", count)
		}
	})

	t.Run("it overwrites valid existing cache", func(t *testing.T) {
		// Start with 2 tasks and build cache via a list command
		initialContent := strings.Join([]string{
			taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Task two", "open", 1, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, initialContent)

		// Trigger cache creation by running list
		app1 := NewApp()
		app1.workDir = dir
		var discard strings.Builder
		app1.stdout = &discard
		if err := app1.Run([]string{"tick", "list"}); err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		// Verify cache has 2 tasks
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		var beforeCount int
		db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&beforeCount)
		db.Close()
		if beforeCount != 2 {
			t.Fatalf("before rebuild: task count = %d, want 2", beforeCount)
		}

		// Now update JSONL to have 3 tasks (simulating manual edit)
		newContent := strings.Join([]string{
			taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Task two", "open", 1, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-ccc333", "Task three", "open", 0, nil, "", "2026-01-19T12:00:00Z"),
		}, "\n") + "\n"
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to write updated tasks.jsonl: %v", err)
		}

		// Run rebuild
		app2 := NewApp()
		app2.workDir = dir
		var stdout strings.Builder
		app2.stdout = &stdout

		err = app2.Run([]string{"tick", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		// Verify cache now has 3 tasks
		db2, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db after rebuild: %v", err)
		}
		defer db2.Close()

		var afterCount int
		db2.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&afterCount)
		if afterCount != 3 {
			t.Errorf("after rebuild: task count = %d, want 3", afterCount)
		}
	})

	t.Run("it updates hash in metadata table after rebuild", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		// Check that metadata table has a jsonl_hash entry
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var hash string
		err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&hash)
		if err != nil {
			t.Fatalf("failed to query jsonl_hash: %v", err)
		}
		if hash == "" {
			t.Error("jsonl_hash is empty, want non-empty hash")
		}
	})

	t.Run("it acquires exclusive lock during rebuild", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		// Run with verbose to see lock messages
		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "--verbose", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		output := stderr.String()
		if !strings.Contains(output, "exclusive lock") {
			t.Errorf("verbose output should mention exclusive lock, got:\n%s", output)
		}
	})

	t.Run("it outputs confirmation message with task count", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Task two", "done", 1, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-ccc333", "Task three", "open", 0, nil, "", "2026-01-19T12:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "3") {
			t.Errorf("output should contain task count '3', got: %q", output)
		}
		lowered := strings.ToLower(output)
		if !strings.Contains(lowered, "rebuilt") && !strings.Contains(lowered, "rebuild") {
			t.Errorf("output should mention rebuild, got: %q", output)
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		output := stdout.String()
		if output != "" {
			t.Errorf("output = %q, want empty string with --quiet", output)
		}
	})

	t.Run("it logs rebuild steps with --verbose", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Task two", "done", 1, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "--verbose", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		output := stderr.String()

		// Should log key rebuild steps
		if !strings.Contains(output, "delete") || !strings.Contains(output, "Delete") {
			if !strings.Contains(strings.ToLower(output), "delet") {
				t.Errorf("verbose output should log delete step, got:\n%s", output)
			}
		}
		if !strings.Contains(output, "read") || !strings.Contains(output, "Read") {
			if !strings.Contains(strings.ToLower(output), "read") {
				t.Errorf("verbose output should log read step, got:\n%s", output)
			}
		}
		if !strings.Contains(output, "hash") {
			t.Errorf("verbose output should log hash update step, got:\n%s", output)
		}
	})

	t.Run("it handles empty JSONL (0 tasks rebuilt)", func(t *testing.T) {
		dir := setupInitializedTickDir(t) // empty tasks.jsonl

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "rebuild"})
		if err != nil {
			t.Fatalf("rebuild returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "0") {
			t.Errorf("output should contain '0' for empty rebuild, got: %q", output)
		}

		// Verify cache.db exists with correct schema
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query tasks: %v", err)
		}
		if count != 0 {
			t.Errorf("task count = %d, want 0", count)
		}
	})
}
