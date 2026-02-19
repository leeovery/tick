package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runRemoveWithStdin runs a tick remove command with custom stdin and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runRemoveWithStdin(t *testing.T, dir string, stdin string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Stdin:  strings.NewReader(stdin),
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "remove"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

// runRemove runs a tick remove command and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
// Provides empty stdin — for tests that need custom stdin, use runRemoveWithStdin.
func runRemove(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	return runRemoveWithStdin(t, dir, "", args...)
}

// readJSONLBytes reads the raw bytes of the tasks.jsonl file.
func readJSONLBytes(t *testing.T, tickDir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(tickDir, "tasks.jsonl"))
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}
	return data
}

func TestRunRemove(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it removes a single task from JSONL when --force is provided", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Task to remove", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-def456", Title: "Task to keep", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, _, exitCode := runRemove(t, dir, "tick-abc123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-def456" {
			t.Errorf("remaining task ID = %q, want %q", tasks[0].ID, "tick-def456")
		}
	})

	t.Run("it matches task ID case-insensitively", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Task to remove", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runRemove(t, dir, "TICK-ABC123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Fatalf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("it cleans up BlockedBy references on surviving tasks", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Blocker task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-def456", Title: "Blocked task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			BlockedBy: []string{"tick-abc123"},
		}
		taskC := task.Task{
			ID: "tick-ghi789", Title: "Also blocked", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			BlockedBy: []string{"tick-abc123", "tick-def456"},
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB, taskC})

		_, _, exitCode := runRemove(t, dir, "tick-abc123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		for _, tk := range tasks {
			for _, dep := range tk.BlockedBy {
				if dep == "tick-abc123" {
					t.Errorf("task %s still has tick-abc123 in BlockedBy", tk.ID)
				}
			}
		}

		// taskC should still have tick-def456 in BlockedBy
		for _, tk := range tasks {
			if tk.ID == "tick-ghi789" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-def456" {
					t.Errorf("task tick-ghi789 BlockedBy = %v, want [tick-def456]", tk.BlockedBy)
				}
			}
		}
	})

	t.Run("it reports dependency-updated tasks in RemovalResult", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Blocker", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-def456", Title: "Blocked one", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			BlockedBy: []string{"tick-abc123"},
		}
		taskC := task.Task{
			ID: "tick-ghi789", Title: "Blocked two", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			BlockedBy: []string{"tick-abc123"},
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB, taskC})

		stdout, _, exitCode := runRemove(t, dir, "tick-abc123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Output should mention dependency updates for both tasks
		if !strings.Contains(stdout, "tick-def456") {
			t.Errorf("stdout should mention tick-def456 in dep update, got %q", stdout)
		}
		if !strings.Contains(stdout, "tick-ghi789") {
			t.Errorf("stdout should mention tick-ghi789 in dep update, got %q", stdout)
		}
		if !strings.Contains(stdout, "Updated dependencies") {
			t.Errorf("stdout should contain 'Updated dependencies', got %q", stdout)
		}
	})

	t.Run("it outputs removal through formatter when not quiet", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "My task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		stdout, _, exitCode := runRemove(t, dir, "tick-abc123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "Removed tick-abc123") {
			t.Errorf("stdout should contain 'Removed tick-abc123', got %q", stdout)
		}
		if !strings.Contains(stdout, "My task") {
			t.Errorf("stdout should contain task title, got %q", stdout)
		}
	})

	t.Run("it suppresses output with --quiet flag", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "My task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		stdout, stderr, exitCode := runRemove(t, dir, "--quiet", "tick-abc123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if stdout != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("stderr should be empty on success, got %q", stderr)
		}
	})

	t.Run("it aborts when --force is omitted and stdin is empty (EOF)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "My task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemove(t, dir, "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Aborted.") {
			t.Errorf("stderr should contain 'Aborted.', got %q", stderr)
		}
	})

	t.Run("it errors when no task ID is provided", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runRemove(t, dir, "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "task ID is required") {
			t.Errorf("stderr should contain 'task ID is required', got %q", stderr)
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runRemove(t, dir, "tick-nonexist", "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "task 'tick-nonexist' not found") {
			t.Errorf("stderr should contain \"task 'tick-nonexist' not found\", got %q", stderr)
		}
	})

	t.Run("it dispatches remove command through App.Run", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Dispatch test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Stdin:  strings.NewReader(""),
			Getwd:  func() (string, error) { return dir, nil },
			IsTTY:  true,
		}
		code := app.Run([]string{"tick", "remove", "tick-abc123", "--force"})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderrBuf.String())
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after remove, got %d", len(tasks))
		}
	})

	t.Run("it returns exact spec-mandated message when no arguments provided", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runRemove(t, dir)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		wantErr := "Error: task ID is required. Usage: tick remove <id> [<id>...]\n"
		if stderr != wantErr {
			t.Errorf("stderr = %q, want %q", stderr, wantErr)
		}
	})

	t.Run("it returns not-found error for nonexistent task ID", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runRemove(t, dir, "tick-nonexist", "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		wantErr := "Error: task 'tick-nonexist' not found\n"
		if stderr != wantErr {
			t.Errorf("stderr = %q, want %q", stderr, wantErr)
		}
	})

	t.Run("it includes Stdin field on App struct and threads to RunRemove", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Stdin test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		var stdoutBuf, stderrBuf bytes.Buffer
		stdinReader := strings.NewReader("")
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Stdin:  stdinReader,
			Getwd:  func() (string, error) { return dir, nil },
			IsTTY:  true,
		}
		code := app.Run([]string{"tick", "remove", "tick-abc123", "--force"})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderrBuf.String())
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after remove, got %d", len(tasks))
		}
	})

	t.Run("it passes nil stdin safely for non-remove commands", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
			IsTTY:  true,
		}
		// Run a non-remove command (list) with nil Stdin — should not panic or error.
		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", code, stderrBuf.String())
		}
	})

	t.Run("it returns exact spec-mandated message when only --force provided without ID", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runRemove(t, dir, "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		wantErr := "Error: task ID is required. Usage: tick remove <id> [<id>...]\n"
		if stderr != wantErr {
			t.Errorf("stderr = %q, want %q", stderr, wantErr)
		}
	})

	t.Run("it does not modify other tasks when removing one", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Task to remove", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-def456", Title: "Untouched task", Status: task.StatusInProgress,
			Priority: 3, Created: now, Updated: now,
			Description: "Keep me intact",
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, _, exitCode := runRemove(t, dir, "tick-abc123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-def456" {
			t.Errorf("remaining task ID = %q, want %q", tasks[0].ID, "tick-def456")
		}
		if tasks[0].Title != "Untouched task" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "Untouched task")
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}
		if tasks[0].Priority != 3 {
			t.Errorf("priority = %d, want 3", tasks[0].Priority)
		}
		if tasks[0].Description != "Keep me intact" {
			t.Errorf("description = %q, want %q", tasks[0].Description, "Keep me intact")
		}
	})

	t.Run("it prompts for confirmation on stderr when --force is not provided", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "My task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, _ := runRemoveWithStdin(t, dir, "n\n", "tick-abc123")
		wantPrompt := `Remove task tick-abc123 "My task"? [y/N] `
		if !strings.Contains(stderr, wantPrompt) {
			t.Errorf("stderr should contain prompt %q, got %q", wantPrompt, stderr)
		}
	})

	t.Run("it proceeds with removal when user enters y", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Confirm me", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		stdout, _, exitCode := runRemoveWithStdin(t, dir, "y\n", "tick-abc123")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after confirmed remove, got %d", len(tasks))
		}
		if !strings.Contains(stdout, "Removed tick-abc123") {
			t.Errorf("stdout should contain 'Removed tick-abc123', got %q", stdout)
		}
	})

	t.Run("it proceeds with removal when user enters Y (uppercase)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Upper Y", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runRemoveWithStdin(t, dir, "Y\n", "tick-abc123")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after confirmed remove, got %d", len(tasks))
		}
	})

	t.Run("it proceeds with removal when user enters yes", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Yes test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runRemoveWithStdin(t, dir, "yes\n", "tick-abc123")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after confirmed remove, got %d", len(tasks))
		}
	})

	t.Run("it proceeds with removal when user enters YES (case-insensitive)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "YES test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runRemoveWithStdin(t, dir, "YES\n", "tick-abc123")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after confirmed remove, got %d", len(tasks))
		}
	})

	t.Run("it aborts when user presses Enter (empty input)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Empty enter", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemoveWithStdin(t, dir, "\n", "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Aborted.") {
			t.Errorf("stderr should contain 'Aborted.', got %q", stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Errorf("expected 1 task (no removal), got %d", len(tasks))
		}
	})

	t.Run("it aborts when user enters n", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "No test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemoveWithStdin(t, dir, "n\n", "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Aborted.") {
			t.Errorf("stderr should contain 'Aborted.', got %q", stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Errorf("expected 1 task (no removal), got %d", len(tasks))
		}
	})

	t.Run("it aborts when user enters arbitrary text", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Arbitrary test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemoveWithStdin(t, dir, "maybe\n", "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Aborted.") {
			t.Errorf("stderr should contain 'Aborted.', got %q", stderr)
		}
	})

	t.Run("it trims whitespace from user input before comparing", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Trim test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runRemoveWithStdin(t, dir, "  y  \n", "tick-abc123")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after confirmed remove, got %d", len(tasks))
		}
	})

	t.Run("it writes Aborted message to stderr on decline", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Abort stderr test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemoveWithStdin(t, dir, "n\n", "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Aborted.") {
			t.Errorf("stderr should contain 'Aborted.', got %q", stderr)
		}
	})

	t.Run("it does not write to stdout on abort", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "No stdout test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		stdout, _, exitCode := runRemoveWithStdin(t, dir, "n\n", "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if stdout != "" {
			t.Errorf("stdout should be empty on abort, got %q", stdout)
		}
	})

	t.Run("it skips prompt entirely when --force is provided", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Force test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemove(t, dir, "tick-abc123", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// stderr should not contain the confirmation prompt.
		if strings.Contains(stderr, "[y/N]") {
			t.Errorf("stderr should not contain prompt with --force, got %q", stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after forced remove, got %d", len(tasks))
		}
	})

	t.Run("it returns exit code 1 on abort without Error prefix via App.Run", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "Exit code test", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemoveWithStdin(t, dir, "n\n", "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should not contain 'Error:' on abort, got %q", stderr)
		}
	})

	t.Run("it removes multiple tasks when all IDs are valid", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "First task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Second task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskC := task.Task{
			ID: "tick-ccc333", Title: "Survivor", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB, taskC})

		stdout, _, exitCode := runRemove(t, dir, "tick-aaa111", "tick-bbb222", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-ccc333" {
			t.Errorf("remaining task ID = %q, want %q", tasks[0].ID, "tick-ccc333")
		}

		// Output should mention both removed tasks.
		if !strings.Contains(stdout, "tick-aaa111") {
			t.Errorf("stdout should mention tick-aaa111, got %q", stdout)
		}
		if !strings.Contains(stdout, "tick-bbb222") {
			t.Errorf("stdout should mention tick-bbb222, got %q", stdout)
		}
	})

	t.Run("it fails with not-found error when first ID is valid but second is invalid", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Valid task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemove(t, dir, "tick-aaa111", "tick-nonexist", "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "task 'tick-nonexist' not found") {
			t.Errorf("stderr should contain \"task 'tick-nonexist' not found\", got %q", stderr)
		}

		// All-or-nothing: valid task must still exist.
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task (no removal), got %d", len(tasks))
		}
		if tasks[0].ID != "tick-aaa111" {
			t.Errorf("task ID = %q, want %q", tasks[0].ID, "tick-aaa111")
		}
	})

	t.Run("it fails with not-found error when all IDs are invalid", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Existing task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemove(t, dir, "tick-nonexist", "tick-alsofake", "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "task 'tick-nonexist' not found") {
			t.Errorf("stderr should contain \"task 'tick-nonexist' not found\", got %q", stderr)
		}

		// No tasks should be removed.
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task (no removal), got %d", len(tasks))
		}
	})

	t.Run("it removes zero tasks when any ID is invalid (all-or-nothing)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Keep me A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Keep me B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, _, exitCode := runRemove(t, dir, "tick-aaa111", "tick-bbb222", "tick-fake99", "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks (no removal), got %d", len(tasks))
		}
	})

	t.Run("it reports the first invalid ID in the error message", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Valid", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemove(t, dir, "tick-aaa111", "tick-bad111", "tick-bad222", "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		wantErr := "Error: task 'tick-bad111' not found\n"
		if stderr != wantErr {
			t.Errorf("stderr = %q, want %q", stderr, wantErr)
		}
	})

	t.Run("it cleans up BlockedBy references for all removed task IDs", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Blocker A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Blocker B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskC := task.Task{
			ID: "tick-ccc333", Title: "Survivor blocked by both", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			BlockedBy: []string{"tick-aaa111", "tick-bbb222", "tick-ddd444"},
		}
		taskD := task.Task{
			ID: "tick-ddd444", Title: "Another survivor", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			BlockedBy: []string{"tick-aaa111"},
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB, taskC, taskD})

		_, _, exitCode := runRemove(t, dir, "tick-aaa111", "tick-bbb222", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		for _, tk := range tasks {
			for _, dep := range tk.BlockedBy {
				if dep == "tick-aaa111" || dep == "tick-bbb222" {
					t.Errorf("task %s still has removed ID %s in BlockedBy", tk.ID, dep)
				}
			}
		}

		// taskC should retain only tick-ddd444.
		for _, tk := range tasks {
			if tk.ID == "tick-ccc333" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-ddd444" {
					t.Errorf("task tick-ccc333 BlockedBy = %v, want [tick-ddd444]", tk.BlockedBy)
				}
			}
		}

		// taskD should have empty BlockedBy.
		for _, tk := range tasks {
			if tk.ID == "tick-ddd444" {
				if len(tk.BlockedBy) != 0 {
					t.Errorf("task tick-ddd444 BlockedBy = %v, want []", tk.BlockedBy)
				}
			}
		}
	})

	t.Run("it preserves single-ID removal behavior", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Single remove", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Unrelated", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		stdout, _, exitCode := runRemove(t, dir, "tick-aaa111", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-bbb222" {
			t.Errorf("remaining task ID = %q, want %q", tasks[0].ID, "tick-bbb222")
		}
		if !strings.Contains(stdout, "Removed tick-aaa111") {
			t.Errorf("stdout should contain 'Removed tick-aaa111', got %q", stdout)
		}
	})

	t.Run("it does not modify JSONL when validation fails", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Protected", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		// Read JSONL content before the failed remove attempt.
		jsonlBefore := readJSONLBytes(t, tickDir)

		_, _, exitCode := runRemove(t, dir, "tick-aaa111", "tick-nonexist", "--force")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}

		// JSONL content should be byte-identical.
		jsonlAfter := readJSONLBytes(t, tickDir)
		if !bytes.Equal(jsonlBefore, jsonlAfter) {
			t.Errorf("JSONL was modified when validation failed\nbefore: %q\nafter:  %q", jsonlBefore, jsonlAfter)
		}
	})
}

func TestParseRemoveArgs(t *testing.T) {
	t.Run("single ID returns slice of length 1", func(t *testing.T) {
		ids, force := parseRemoveArgs([]string{"tick-abc123"})
		if len(ids) != 1 {
			t.Fatalf("len(ids) = %d, want 1", len(ids))
		}
		if ids[0] != "tick-abc123" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "tick-abc123")
		}
		if force {
			t.Errorf("force = true, want false")
		}
	})

	t.Run("multiple IDs returned in order", func(t *testing.T) {
		ids, _ := parseRemoveArgs([]string{"tick-aaa111", "tick-bbb222", "tick-ccc333"})
		if len(ids) != 3 {
			t.Fatalf("len(ids) = %d, want 3", len(ids))
		}
		want := []string{"tick-aaa111", "tick-bbb222", "tick-ccc333"}
		for i, w := range want {
			if ids[i] != w {
				t.Errorf("ids[%d] = %q, want %q", i, ids[i], w)
			}
		}
	})

	t.Run("IDs normalized to lowercase", func(t *testing.T) {
		ids, _ := parseRemoveArgs([]string{"TICK-AAA111", "Tick-Bbb222"})
		if len(ids) != 2 {
			t.Fatalf("len(ids) = %d, want 2", len(ids))
		}
		if ids[0] != "tick-aaa111" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "tick-aaa111")
		}
		if ids[1] != "tick-bbb222" {
			t.Errorf("ids[1] = %q, want %q", ids[1], "tick-bbb222")
		}
	})

	t.Run("deduplicates identical IDs", func(t *testing.T) {
		ids, _ := parseRemoveArgs([]string{"tick-aaa111", "tick-aaa111"})
		if len(ids) != 1 {
			t.Fatalf("len(ids) = %d, want 1", len(ids))
		}
		if ids[0] != "tick-aaa111" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "tick-aaa111")
		}
	})

	t.Run("deduplicates case-variant IDs", func(t *testing.T) {
		ids, _ := parseRemoveArgs([]string{"TICK-AAA111", "tick-aaa111"})
		if len(ids) != 1 {
			t.Fatalf("len(ids) = %d, want 1", len(ids))
		}
		if ids[0] != "tick-aaa111" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "tick-aaa111")
		}
	})

	t.Run("preserves first-occurrence order after dedup", func(t *testing.T) {
		ids, _ := parseRemoveArgs([]string{"tick-bbb222", "tick-aaa111", "tick-bbb222", "tick-ccc333"})
		if len(ids) != 3 {
			t.Fatalf("len(ids) = %d, want 3", len(ids))
		}
		want := []string{"tick-bbb222", "tick-aaa111", "tick-ccc333"}
		for i, w := range want {
			if ids[i] != w {
				t.Errorf("ids[%d] = %q, want %q", i, ids[i], w)
			}
		}
	})

	t.Run("extracts --force from between IDs", func(t *testing.T) {
		ids, force := parseRemoveArgs([]string{"tick-aaa111", "--force", "tick-bbb222"})
		if !force {
			t.Errorf("force = false, want true")
		}
		if len(ids) != 2 {
			t.Fatalf("len(ids) = %d, want 2", len(ids))
		}
		if ids[0] != "tick-aaa111" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "tick-aaa111")
		}
		if ids[1] != "tick-bbb222" {
			t.Errorf("ids[1] = %q, want %q", ids[1], "tick-bbb222")
		}
	})

	t.Run("extracts -f shorthand flag", func(t *testing.T) {
		ids, force := parseRemoveArgs([]string{"tick-aaa111", "-f"})
		if !force {
			t.Errorf("force = false, want true")
		}
		if len(ids) != 1 {
			t.Fatalf("len(ids) = %d, want 1", len(ids))
		}
		if ids[0] != "tick-aaa111" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "tick-aaa111")
		}
	})

	t.Run("handles --force before and after all IDs", func(t *testing.T) {
		ids, force := parseRemoveArgs([]string{"--force", "tick-aaa111", "tick-bbb222"})
		if !force {
			t.Errorf("force = false, want true (before)")
		}
		if len(ids) != 2 {
			t.Fatalf("len(ids) = %d, want 2", len(ids))
		}

		ids2, force2 := parseRemoveArgs([]string{"tick-aaa111", "tick-bbb222", "--force"})
		if !force2 {
			t.Errorf("force = false, want true (after)")
		}
		if len(ids2) != 2 {
			t.Fatalf("len(ids2) = %d, want 2", len(ids2))
		}
	})

	t.Run("skips unknown flags mixed with IDs", func(t *testing.T) {
		ids, force := parseRemoveArgs([]string{"--verbose", "tick-aaa111", "-x", "tick-bbb222"})
		if force {
			t.Errorf("force = true, want false")
		}
		if len(ids) != 2 {
			t.Fatalf("len(ids) = %d, want 2", len(ids))
		}
		if ids[0] != "tick-aaa111" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "tick-aaa111")
		}
		if ids[1] != "tick-bbb222" {
			t.Errorf("ids[1] = %q, want %q", ids[1], "tick-bbb222")
		}
	})

	t.Run("returns empty slice when only flags or no args provided", func(t *testing.T) {
		ids, _ := parseRemoveArgs([]string{"--force", "-x"})
		if len(ids) != 0 {
			t.Errorf("len(ids) = %d, want 0", len(ids))
		}

		ids2, _ := parseRemoveArgs([]string{})
		if len(ids2) != 0 {
			t.Errorf("len(ids2) = %d, want 0", len(ids2))
		}

		ids3, _ := parseRemoveArgs(nil)
		if len(ids3) != 0 {
			t.Errorf("len(ids3) = %d, want 0", len(ids3))
		}
	})
}
