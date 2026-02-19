package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runRemove runs a tick remove command and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runRemove(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "remove"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
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

	t.Run("it errors when --force is not provided", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "My task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemove(t, dir, "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "--force") {
			t.Errorf("stderr should mention --force, got %q", stderr)
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

	t.Run("it returns error when --force flag is missing", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-abc123", Title: "My task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runRemove(t, dir, "tick-abc123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		wantErr := "Error: remove requires --force flag (interactive confirmation not yet implemented)\n"
		if stderr != wantErr {
			t.Errorf("stderr = %q, want %q", stderr, wantErr)
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
}
