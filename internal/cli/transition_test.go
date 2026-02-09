package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestTransition(t *testing.T) {
	t.Run("it transitions task to in_progress via tick start", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Start me")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}
	})

	t.Run("it transitions task to done via tick done from open", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Done from open")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "done", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Status != task.StatusDone {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusDone)
		}
	})

	t.Run("it transitions task to done via tick done from in_progress", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Done from IP")
		tk.Status = task.StatusInProgress
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "done", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Status != task.StatusDone {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusDone)
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from open", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Cancel from open")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "cancel", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Status != task.StatusCancelled {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusCancelled)
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from in_progress", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Cancel from IP")
		tk.Status = task.StatusInProgress
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "cancel", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Status != task.StatusCancelled {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusCancelled)
		}
	})

	t.Run("it transitions task to open via tick reopen from done", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tk := task.NewTask("tick-aaaaaa", "Reopen from done")
		tk.Status = task.StatusDone
		tk.Closed = &now
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "reopen", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
	})

	t.Run("it transitions task to open via tick reopen from cancelled", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tk := task.NewTask("tick-aaaaaa", "Reopen from cancelled")
		tk.Status = task.StatusCancelled
		tk.Closed = &now
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "reopen", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
	})

	t.Run("it outputs status transition line on success", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Transition output")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		expected := "tick-aaaaaa: open \u2192 in_progress\n"
		if stdout.String() != expected {
			t.Errorf("output = %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("it suppresses output with --quiet flag", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Quiet transition")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it errors when task ID argument is missing", func(t *testing.T) {
		dir := initTickProject(t)

		commands := []string{"start", "done", "cancel", "reopen"}
		for _, cmd := range commands {
			t.Run(cmd, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", cmd}, dir, &stdout, &stderr, false)

				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				expectedMsg := "Task ID is required. Usage: tick " + cmd + " <id>"
				if !strings.Contains(stderr.String(), expectedMsg) {
					t.Errorf("expected %q in stderr, got %q", expectedMsg, stderr.String())
				}
			})
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "start", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("expected 'not found' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors on invalid transition", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Done task")
		tk.Status = task.StatusDone
		now := time.Now().UTC().Truncate(time.Second)
		tk.Closed = &now
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Cannot") {
			t.Errorf("expected 'Cannot' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it writes errors to stderr", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "start"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if stderr.String() == "" {
			t.Error("expected error written to stderr")
		}
		if stdout.String() != "" {
			t.Errorf("expected no output on stdout for error, got %q", stdout.String())
		}
	})

	t.Run("it exits with code 1 on error", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "done", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	t.Run("it normalizes task ID to lowercase", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Normalize test")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "start", "TICK-AAAAAA"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}
	})

	t.Run("it persists status change via atomic write", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Persist test")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Read back from file and verify persisted
		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("persisted status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}
	})

	t.Run("it sets closed timestamp on done/cancel", func(t *testing.T) {
		tests := []struct {
			name    string
			command string
		}{
			{"done", "done"},
			{"cancel", "cancel"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tk := task.NewTask("tick-aaaaaa", "Close test")
				dir := initTickProjectWithTasks(t, []task.Task{tk})

				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", tt.command, "tick-aaaaaa"}, dir, &stdout, &stderr, false)

				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				tasks := readTasksFromFile(t, dir)
				if tasks[0].Closed == nil {
					t.Error("expected closed timestamp to be set")
				}
			})
		}
	})

	t.Run("it clears closed timestamp on reopen", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tk := task.NewTask("tick-aaaaaa", "Reopen clear closed")
		tk.Status = task.StatusDone
		tk.Closed = &now
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "reopen", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Closed != nil {
			t.Error("expected closed timestamp to be cleared on reopen")
		}
	})
}
