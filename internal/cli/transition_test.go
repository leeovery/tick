package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestTransitionCommands(t *testing.T) {
	t.Run("it transitions task to in_progress via tick start", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "start", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != "in_progress" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "in_progress")
		}
	})

	t.Run("it transitions task to done via tick done from open", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "done", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "done" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "done")
		}
	})

	t.Run("it transitions task to done via tick done from in_progress", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "in_progress", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "done", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "done" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "done")
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from open", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "cancel", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "cancelled" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "cancelled")
		}
	})

	t.Run("it transitions task to cancelled via tick cancel from in_progress", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "in_progress", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "cancel", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "cancelled" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "cancelled")
		}
	})

	t.Run("it transitions task to open via tick reopen from done", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "reopen", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "open" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "open")
		}
	})

	t.Run("it transitions task to open via tick reopen from cancelled", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "cancelled", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "reopen", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "open" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "open")
		}
	})

	t.Run("it outputs status transition line on success", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "start", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		expected := "tick-a1b2c3: open → in_progress\n"
		if output != expected {
			t.Errorf("output = %q, want %q", output, expected)
		}
	})

	t.Run("it suppresses output with --quiet flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "start", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it errors when task ID argument is missing", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "start"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Task ID is required") {
			t.Errorf("error should mention 'Task ID is required', got %q", errOutput)
		}
		if !strings.Contains(errOutput, "Usage: tick start <id>") {
			t.Errorf("error should include usage hint, got %q", errOutput)
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "start", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "tick-nonexistent") {
			t.Errorf("error should include task ID, got %q", errOutput)
		}
		if !strings.Contains(errOutput, "not found") {
			t.Errorf("error should mention 'not found', got %q", errOutput)
		}
	})

	t.Run("it errors on invalid transition", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "start", "tick-a1b2c3"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "cannot start") {
			t.Errorf("error should mention invalid transition, got %q", errOutput)
		}
	})

	t.Run("it writes errors to stderr", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "start", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		// Error should be in stderr, not stdout
		if stderr.String() == "" {
			t.Error("error should be written to stderr")
		}
		if stdout.String() != "" {
			t.Errorf("stdout should be empty on error, got %q", stdout.String())
		}
	})

	t.Run("it exits with code 1 on error", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Missing ID
		code := app.Run([]string{"tick", "start"})
		if code != 1 {
			t.Errorf("expected exit code 1 for missing ID, got %d", code)
		}

		// Non-existent task
		stderr.Reset()
		code = app.Run([]string{"tick", "start", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1 for non-existent task, got %d", code)
		}

		// Invalid transition
		setupTask(t, dir, "tick-done", "Done task")
		// Change its status to done
		setupTaskFull(t, dir, "tick-done2", "Done task 2", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z")
		stderr.Reset()
		code = app.Run([]string{"tick", "start", "tick-done2"})
		if code != 1 {
			t.Errorf("expected exit code 1 for invalid transition, got %d", code)
		}
	})

	t.Run("it normalizes task ID to lowercase", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Use uppercase ID
		code := app.Run([]string{"tick", "start", "TICK-A1B2C3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "in_progress" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "in_progress")
		}

		// Output should use normalized lowercase ID
		output := stdout.String()
		if !strings.Contains(output, "tick-a1b2c3") {
			t.Errorf("output should use normalized lowercase ID, got %q", output)
		}
	})

	t.Run("it persists status change via atomic write", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "start", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Create new app to read fresh from disk
		var stdout2, stderr2 bytes.Buffer
		app2 := &App{
			Stdout: &stdout2,
			Stderr: &stderr2,
			Cwd:    dir,
		}

		// Verify persisted by reading again
		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "in_progress" {
			t.Errorf("persisted status = %q, want %q", tasks[0].Status, "in_progress")
		}

		// Verify via another operation
		code = app2.Run([]string{"tick", "done", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}

		tasks = readTasksFromDir(t, dir)
		if tasks[0].Status != "done" {
			t.Errorf("second transition status = %q, want %q", tasks[0].Status, "done")
		}
	})

	t.Run("it sets closed timestamp on done/cancel", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-done", "Task for done")
		setupTask(t, dir, "tick-cancel", "Task for cancel")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Test done
		code := app.Run([]string{"tick", "done", "tick-done"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var doneTask, cancelTask *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			Description string   `json:"description,omitempty"`
			BlockedBy   []string `json:"blocked_by,omitempty"`
			Parent      string   `json:"parent,omitempty"`
			Created     string   `json:"created"`
			Updated     string   `json:"updated"`
			Closed      string   `json:"closed,omitempty"`
		}
		for i := range tasks {
			if tasks[i].ID == "tick-done" {
				doneTask = &tasks[i]
			}
			if tasks[i].ID == "tick-cancel" {
				cancelTask = &tasks[i]
			}
		}

		if doneTask.Closed == "" {
			t.Error("closed timestamp should be set after done")
		}

		// Test cancel
		code = app.Run([]string{"tick", "cancel", "tick-cancel"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks = readTasksFromDir(t, dir)
		for i := range tasks {
			if tasks[i].ID == "tick-cancel" {
				cancelTask = &tasks[i]
			}
		}

		if cancelTask.Closed == "" {
			t.Error("closed timestamp should be set after cancel")
		}
	})

	t.Run("it clears closed timestamp on reopen", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z")

		// Verify closed is set initially
		tasks := readTasksFromDir(t, dir)
		if tasks[0].Closed == "" {
			t.Fatal("closed timestamp should be set initially")
		}

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "reopen", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks = readTasksFromDir(t, dir)
		if tasks[0].Closed != "" {
			t.Errorf("closed timestamp should be cleared after reopen, got %q", tasks[0].Closed)
		}
	})
}
