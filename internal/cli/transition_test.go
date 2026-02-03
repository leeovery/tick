package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openTaskJSONL returns a JSONL line for an open task with the given ID.
func openTaskJSONL(id string) string {
	return `{"id":"` + id + `","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
}

// inProgressTaskJSONL returns a JSONL line for an in_progress task with the given ID.
func inProgressTaskJSONL(id string) string {
	return `{"id":"` + id + `","title":"Test task","status":"in_progress","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
}

// doneTaskJSONL returns a JSONL line for a done task with the given ID.
func doneTaskJSONL(id string) string {
	return `{"id":"` + id + `","title":"Test task","status":"done","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z","closed":"2026-01-19T12:00:00Z"}`
}

// cancelledTaskJSONL returns a JSONL line for a cancelled task with the given ID.
func cancelledTaskJSONL(id string) string {
	return `{"id":"` + id + `","title":"Test task","status":"cancelled","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z","closed":"2026-01-19T12:00:00Z"}`
}

// readTaskByID reads tasks.jsonl and returns the task map matching the given ID.
func readTaskByID(t *testing.T, dir, id string) map[string]interface{} {
	t.Helper()
	tasks := readTasksJSONL(t, dir)
	for _, tk := range tasks {
		if tk["id"] == id {
			return tk
		}
	}
	t.Fatalf("task %q not found in tasks.jsonl", id)
	return nil
}

func TestTransitionCommands(t *testing.T) {
	t.Run("it transitions task to in_progress via tick start", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "start", "tick-aaa111"})
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "in_progress" {
			t.Errorf("status = %q, want %q", tk["status"], "in_progress")
		}
	})

	t.Run("it transitions task to done via tick done from open", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "done", "tick-aaa111"})
		if err != nil {
			t.Fatalf("done returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "done" {
			t.Errorf("status = %q, want %q", tk["status"], "done")
		}
	})

	t.Run("it transitions task to done via tick done from in_progress", func(t *testing.T) {
		dir := setupTickDirWithContent(t, inProgressTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "done", "tick-aaa111"})
		if err != nil {
			t.Fatalf("done returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "done" {
			t.Errorf("status = %q, want %q", tk["status"], "done")
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from open", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "cancel", "tick-aaa111"})
		if err != nil {
			t.Fatalf("cancel returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "cancelled" {
			t.Errorf("status = %q, want %q", tk["status"], "cancelled")
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from in_progress", func(t *testing.T) {
		dir := setupTickDirWithContent(t, inProgressTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "cancel", "tick-aaa111"})
		if err != nil {
			t.Fatalf("cancel returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "cancelled" {
			t.Errorf("status = %q, want %q", tk["status"], "cancelled")
		}
	})

	t.Run("it transitions task to open via tick reopen from done", func(t *testing.T) {
		dir := setupTickDirWithContent(t, doneTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "reopen", "tick-aaa111"})
		if err != nil {
			t.Fatalf("reopen returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "open" {
			t.Errorf("status = %q, want %q", tk["status"], "open")
		}
	})

	t.Run("it transitions task to open via tick reopen from cancelled", func(t *testing.T) {
		dir := setupTickDirWithContent(t, cancelledTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "reopen", "tick-aaa111"})
		if err != nil {
			t.Fatalf("reopen returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "open" {
			t.Errorf("status = %q, want %q", tk["status"], "open")
		}
	})

	t.Run("it outputs status transition line on success", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "start", "tick-aaa111"})
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		want := "tick-aaa111: open \u2192 in_progress"
		if output != want {
			t.Errorf("output = %q, want %q", output, want)
		}
	})

	t.Run("it suppresses output with --quiet flag", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "start", "tick-aaa111"})
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}

		if stdout.String() != "" {
			t.Errorf("stdout = %q, want empty with --quiet", stdout.String())
		}
	})

	t.Run("it errors when task ID argument is missing", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		commands := []string{"start", "done", "cancel", "reopen"}
		for _, cmd := range commands {
			t.Run(cmd, func(t *testing.T) {
				app := NewApp()
				app.workDir = dir

				err := app.Run([]string{"tick", cmd})
				if err == nil {
					t.Fatalf("%s expected error for missing ID, got nil", cmd)
				}
				wantContains := "Task ID is required"
				if !strings.Contains(err.Error(), wantContains) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), wantContains)
				}
				wantUsage := "tick " + cmd + " <id>"
				if !strings.Contains(err.Error(), wantUsage) {
					t.Errorf("error = %q, want it to contain usage %q", err.Error(), wantUsage)
				}
			})
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "start", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error for non-existent task, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}
	})

	t.Run("it errors on invalid transition", func(t *testing.T) {
		dir := setupTickDirWithContent(t, doneTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "start", "tick-aaa111"})
		if err == nil {
			t.Fatal("expected error for invalid transition, got nil")
		}
		if !strings.Contains(err.Error(), "Cannot start") {
			t.Errorf("error = %q, want it to contain 'Cannot start'", err.Error())
		}
	})

	t.Run("it writes errors to stderr", func(t *testing.T) {
		// This test verifies via integration binary test that errors go to stderr.
		// At the unit level, errors are returned from Run() and main.go writes to stderr.
		// We verify the error is returned (not written to stdout).
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "start", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// stdout should be empty -- errors come back as return values
		if stdout.String() != "" {
			t.Errorf("stdout = %q, want empty (errors should not go to stdout)", stdout.String())
		}
	})

	t.Run("it exits with code 1 on error", func(t *testing.T) {
		// At the unit level, returning non-nil error from Run() causes exit code 1 in main.go.
		// We verify the error is non-nil for various error conditions.
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "start"})
		if err == nil {
			t.Fatal("expected error (would cause exit 1), got nil")
		}
	})

	t.Run("it normalizes task ID to lowercase", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		// Pass uppercase ID
		err := app.Run([]string{"tick", "start", "TICK-AAA111"})
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["status"] != "in_progress" {
			t.Errorf("status = %q, want %q", tk["status"], "in_progress")
		}

		// Output should use normalized (lowercase) ID
		output := strings.TrimSpace(stdout.String())
		if !strings.HasPrefix(output, "tick-aaa111:") {
			t.Errorf("output = %q, want it to start with 'tick-aaa111:'", output)
		}
	})

	t.Run("it persists status change via atomic write", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "start", "tick-aaa111"})
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}

		// Read back the file directly and verify the task was persisted
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		var taskMap map[string]interface{}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if err := json.Unmarshal([]byte(lines[0]), &taskMap); err != nil {
			t.Fatalf("failed to parse JSONL: %v", err)
		}
		if taskMap["status"] != "in_progress" {
			t.Errorf("persisted status = %q, want %q", taskMap["status"], "in_progress")
		}
	})

	t.Run("it sets closed timestamp on done", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "done", "tick-aaa111"})
		if err != nil {
			t.Fatalf("done returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		closed, ok := tk["closed"].(string)
		if !ok || closed == "" {
			t.Error("closed timestamp should be set on done")
		}
	})

	t.Run("it sets closed timestamp on cancel", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "cancel", "tick-aaa111"})
		if err != nil {
			t.Fatalf("cancel returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		closed, ok := tk["closed"].(string)
		if !ok || closed == "" {
			t.Error("closed timestamp should be set on cancel")
		}
	})

	t.Run("it clears closed timestamp on reopen", func(t *testing.T) {
		dir := setupTickDirWithContent(t, doneTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "reopen", "tick-aaa111"})
		if err != nil {
			t.Fatalf("reopen returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if _, hasClosed := tk["closed"]; hasClosed {
			t.Error("closed timestamp should be cleared (omitted) on reopen")
		}
	})
}
