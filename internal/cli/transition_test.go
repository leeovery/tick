package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runTransition runs a tick transition command (start/done/cancel/reopen) and returns
// stdout, stderr, and exit code.
func runTransition(t *testing.T, dir string, command string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	fullArgs := append([]string{"tick", command}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestTransitionCommands(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it transitions task to in_progress via tick start", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{openTask})

		_, _, exitCode := runTransition(t, dir, "start", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}
	})

	t.Run("it transitions task to done via tick done from open", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{openTask})

		_, _, exitCode := runTransition(t, dir, "done", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Status != task.StatusDone {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusDone)
		}
	})

	t.Run("it transitions task to done via tick done from in_progress", func(t *testing.T) {
		ipTask := task.Task{
			ID: "tick-aaa111", Title: "IP task", Status: task.StatusInProgress,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{ipTask})

		_, _, exitCode := runTransition(t, dir, "done", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Status != task.StatusDone {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusDone)
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from open", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{openTask})

		_, _, exitCode := runTransition(t, dir, "cancel", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Status != task.StatusCancelled {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusCancelled)
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from in_progress", func(t *testing.T) {
		ipTask := task.Task{
			ID: "tick-aaa111", Title: "IP task", Status: task.StatusInProgress,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{ipTask})

		_, _, exitCode := runTransition(t, dir, "cancel", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Status != task.StatusCancelled {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusCancelled)
		}
	})

	t.Run("it transitions task to open via tick reopen from done", func(t *testing.T) {
		closedAt := now
		doneTask := task.Task{
			ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone,
			Priority: 2, Created: now, Updated: now, Closed: &closedAt,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{doneTask})

		_, _, exitCode := runTransition(t, dir, "reopen", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
	})

	t.Run("it transitions task to open via tick reopen from cancelled", func(t *testing.T) {
		closedAt := now
		cancelledTask := task.Task{
			ID: "tick-aaa111", Title: "Cancelled task", Status: task.StatusCancelled,
			Priority: 2, Created: now, Updated: now, Closed: &closedAt,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{cancelledTask})

		_, _, exitCode := runTransition(t, dir, "reopen", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
	})

	t.Run("it outputs status transition line on success", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{openTask})

		stdout, _, exitCode := runTransition(t, dir, "start", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "tick-aaa111: open \u2192 in_progress\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it suppresses output with --quiet flag", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{openTask})

		stdout, stderr, exitCode := runTransition(t, dir, "start", "--quiet", "tick-aaa111")
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

	t.Run("it errors when task ID argument is missing", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runTransition(t, dir, "start")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should contain Error:, got %q", stderr)
		}
		if !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, "start") {
			t.Errorf("stderr should contain usage hint with command name, got %q", stderr)
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runTransition(t, dir, "start", "tick-nonexist")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it errors on invalid transition", func(t *testing.T) {
		closedAt := now
		doneTask := task.Task{
			ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone,
			Priority: 2, Created: now, Updated: now, Closed: &closedAt,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{doneTask})

		_, stderr, exitCode := runTransition(t, dir, "start", "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "cannot start") {
			t.Errorf("stderr should contain 'cannot start', got %q", stderr)
		}
	})

	t.Run("it writes errors to stderr", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, stderr, _ := runTransition(t, dir, "start")
		if stdout != "" {
			t.Errorf("stdout should be empty on error, got %q", stdout)
		}
		if stderr == "" {
			t.Error("stderr should contain error message")
		}
	})

	t.Run("it exits with code 1 on error", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, _, exitCode := runTransition(t, dir, "start")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it normalizes task ID to lowercase", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{openTask})

		stdout, _, exitCode := runTransition(t, dir, "start", "TICK-AAA111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Verify the task was found and transitioned despite uppercase input.
		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}

		// Output should use lowercase ID.
		if !strings.Contains(stdout, "tick-aaa111") {
			t.Errorf("stdout should contain lowercase ID, got %q", stdout)
		}
	})

	t.Run("it persists status change via atomic write", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{openTask})

		_, _, exitCode := runTransition(t, dir, "start", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Read back from disk to confirm persistence.
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("persisted status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}
		// Updated timestamp should be refreshed.
		if !tasks[0].Updated.After(now.Add(-time.Second)) {
			t.Error("updated timestamp should be refreshed")
		}
	})

	t.Run("it sets closed timestamp on done/cancel", func(t *testing.T) {
		tests := []struct {
			name    string
			command string
		}{
			{"done sets closed", "done"},
			{"cancel sets closed", "cancel"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				openTask := task.Task{
					ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
					Priority: 2, Created: now, Updated: now,
				}
				dir, tickDir := setupTickProjectWithTasks(t, []task.Task{openTask})

				before := time.Now().UTC().Truncate(time.Second)
				_, _, exitCode := runTransition(t, dir, tt.command, "tick-aaa111")
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0", exitCode)
				}
				after := time.Now().UTC().Truncate(time.Second)

				tasks := readPersistedTasks(t, tickDir)
				if tasks[0].Closed == nil {
					t.Fatal("expected closed timestamp to be set, got nil")
				}
				if tasks[0].Closed.Before(before) || tasks[0].Closed.After(after) {
					t.Errorf("closed timestamp %v not in expected range [%v, %v]", *tasks[0].Closed, before, after)
				}
			})
		}
	})

	t.Run("it clears closed timestamp on reopen", func(t *testing.T) {
		closedAt := now
		doneTask := task.Task{
			ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone,
			Priority: 2, Created: now, Updated: now, Closed: &closedAt,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{doneTask})

		_, _, exitCode := runTransition(t, dir, "reopen", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		if tasks[0].Closed != nil {
			t.Errorf("expected closed to be nil after reopen, got %v", *tasks[0].Closed)
		}
	})
}
