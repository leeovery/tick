package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestTransitionStart_TransitionsToInProgress(t *testing.T) {
	t.Run("it transitions task to in_progress via tick start", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("expected status %q, got %q", task.StatusInProgress, tasks[0].Status)
		}
	})
}

func TestTransitionDone_FromOpen(t *testing.T) {
	t.Run("it transitions task to done via tick done from open", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "done", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusDone {
			t.Errorf("expected status %q, got %q", task.StatusDone, tasks[0].Status)
		}
	})
}

func TestTransitionDone_FromInProgress(t *testing.T) {
	t.Run("it transitions task to done via tick done from in_progress", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "done", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusDone {
			t.Errorf("expected status %q, got %q", task.StatusDone, tasks[0].Status)
		}
	})
}

func TestTransitionCancel_FromOpen(t *testing.T) {
	t.Run("it transitions task to cancelled via tick cancel from open", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "cancel", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusCancelled {
			t.Errorf("expected status %q, got %q", task.StatusCancelled, tasks[0].Status)
		}
	})
}

func TestTransitionCancel_FromInProgress(t *testing.T) {
	t.Run("it transitions task to cancelled via tick cancel from in_progress", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "cancel", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusCancelled {
			t.Errorf("expected status %q, got %q", task.StatusCancelled, tasks[0].Status)
		}
	})
}

func TestTransitionReopen_FromDone(t *testing.T) {
	t.Run("it transitions task to open via tick reopen from done", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "reopen", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("expected status %q, got %q", task.StatusOpen, tasks[0].Status)
		}
	})
}

func TestTransitionReopen_FromCancelled(t *testing.T) {
	t.Run("it transitions task to open via tick reopen from cancelled", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "reopen", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("expected status %q, got %q", task.StatusOpen, tasks[0].Status)
		}
	})
}

func TestTransition_OutputsStatusTransitionLine(t *testing.T) {
	t.Run("it outputs status transition line on success", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		expected := "tick-aaa111: open \u2192 in_progress"
		if output != expected {
			t.Errorf("expected output %q, got %q", expected, output)
		}
	})
}

func TestTransition_QuietSuppressesOutput(t *testing.T) {
	t.Run("it suppresses output with --quiet flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
	})
}

func TestTransition_ErrorMissingID(t *testing.T) {
	t.Run("it errors when task ID argument is missing", func(t *testing.T) {
		commands := []string{"start", "done", "cancel", "reopen"}

		for _, cmd := range commands {
			t.Run(cmd, func(t *testing.T) {
				dir := setupInitializedDir(t)
				var stdout, stderr bytes.Buffer

				app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
				code := app.Run([]string{"tick", cmd})
				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}

				errMsg := stderr.String()
				if !strings.Contains(errMsg, "Error:") {
					t.Errorf("expected error on stderr, got %q", errMsg)
				}
				if !strings.Contains(errMsg, "Task ID is required") {
					t.Errorf("expected 'Task ID is required' in error, got %q", errMsg)
				}
				expectedUsage := "Usage: tick " + cmd + " <id>"
				if !strings.Contains(errMsg, expectedUsage) {
					t.Errorf("expected usage hint %q in error, got %q", expectedUsage, errMsg)
				}
			})
		}
	})
}

func TestTransition_ErrorTaskNotFound(t *testing.T) {
	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "start", "tick-nonexist"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "tick-nonexist") {
			t.Errorf("expected error to contain task ID, got %q", errMsg)
		}
	})
}

func TestTransition_ErrorInvalidTransition(t *testing.T) {
	t.Run("it errors on invalid transition", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "start", "tick-aaa111"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
	})
}

func TestTransition_ErrorsToStderr(t *testing.T) {
	t.Run("it writes errors to stderr", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "start", "tick-nonexist"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		if stderr.String() == "" {
			t.Error("expected error output on stderr, got nothing")
		}
		if stdout.String() != "" {
			t.Errorf("expected no stdout on error, got %q", stdout.String())
		}
	})
}

func TestTransition_ExitCode1OnError(t *testing.T) {
	t.Run("it exits with code 1 on error", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}

		// Missing ID
		code := app.Run([]string{"tick", "start"})
		if code != 1 {
			t.Errorf("expected exit code 1 for missing ID, got %d", code)
		}

		// Not found
		stderr.Reset()
		code = app.Run([]string{"tick", "start", "tick-nonexist"})
		if code != 1 {
			t.Errorf("expected exit code 1 for not found, got %d", code)
		}
	})
}

func TestTransition_NormalizesIDToLowercase(t *testing.T) {
	t.Run("it normalizes task ID to lowercase", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		// Use uppercase ID
		code := app.Run([]string{"tick", "start", "TICK-AAA111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("expected status %q, got %q", task.StatusInProgress, tasks[0].Status)
		}

		// Output should use lowercase ID
		output := strings.TrimSpace(stdout.String())
		if !strings.HasPrefix(output, "tick-aaa111:") {
			t.Errorf("expected output to use lowercase ID, got %q", output)
		}
	})
}

func TestTransition_PersistsViaAtomicWrite(t *testing.T) {
	t.Run("it persists status change via atomic write", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Re-read from file to verify persistence
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("expected persisted status %q, got %q", task.StatusInProgress, tasks[0].Status)
		}
		// Updated timestamp should be refreshed
		if !tasks[0].Updated.After(now) {
			t.Errorf("expected updated timestamp to be refreshed, got %v (original: %v)", tasks[0].Updated, now)
		}
	})
}

func TestTransition_SetsClosedTimestamp(t *testing.T) {
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
				now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
				existing := []task.Task{
					{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
				}
				dir := setupInitializedDirWithTasks(t, existing)
				var stdout, stderr bytes.Buffer

				app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
				code := app.Run([]string{"tick", tt.command, "tick-aaa111"})
				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				tasks := readTasksFromDir(t, dir)
				if tasks[0].Closed == nil {
					t.Fatal("expected closed timestamp to be set")
				}
			})
		}
	})
}

func TestTransition_ClearsClosedTimestampOnReopen(t *testing.T) {
	t.Run("it clears closed timestamp on reopen", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "reopen", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Closed != nil {
			t.Errorf("expected closed timestamp to be cleared after reopen, got %v", tasks[0].Closed)
		}
	})
}
