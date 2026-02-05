package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDepCommands(t *testing.T) {
	t.Run("it adds a dependency between two existing tasks", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *struct {
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
			if tasks[i].ID == "tick-aaaaaa" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("task tick-aaaaaa not found")
		}
		if len(taskA.BlockedBy) != 1 || taskA.BlockedBy[0] != "tick-bbbbbb" {
			t.Errorf("blocked_by = %v, want [tick-bbbbbb]", taskA.BlockedBy)
		}
	})

	t.Run("it removes an existing dependency", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-aaaaaa", "Task A", "open", 2, "", "", []string{"tick-bbbbbb"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *struct {
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
			if tasks[i].ID == "tick-aaaaaa" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("task tick-aaaaaa not found")
		}
		if len(taskA.BlockedBy) != 0 {
			t.Errorf("blocked_by = %v, want []", taskA.BlockedBy)
		}
	})

	t.Run("it outputs confirmation on success for add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		expected := "Dependency added: tick-aaaaaa blocked by tick-bbbbbb\n"
		if stdout.String() != expected {
			t.Errorf("output = %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("it outputs confirmation on success for rm", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-aaaaaa", "Task A", "open", 2, "", "", []string{"tick-bbbbbb"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		expected := "Dependency removed: tick-aaaaaa no longer blocked by tick-bbbbbb\n"
		if stdout.String() != expected {
			t.Errorf("output = %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("it updates task updated timestamp on add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				// Updated timestamp should be different from created timestamp
				if task.Updated == "2026-01-19T10:00:00Z" {
					t.Error("updated timestamp should be refreshed")
				}
				break
			}
		}
	})

	t.Run("it updates task updated timestamp on rm", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-aaaaaa", "Task A", "open", 2, "", "", []string{"tick-bbbbbb"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				// Updated timestamp should be different from created timestamp
				if task.Updated == "2026-01-19T10:00:00Z" {
					t.Error("updated timestamp should be refreshed")
				}
				break
			}
		}
	})

	t.Run("it errors when task_id not found on add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-nonexistent", "tick-bbbbbb"})
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

	t.Run("it errors when task_id not found on rm", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "rm", "tick-nonexistent", "tick-bbbbbb"})
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

	t.Run("it errors when blocked_by_id not found on add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "tick-nonexistent") {
			t.Errorf("error should include blocked_by ID, got %q", errOutput)
		}
		if !strings.Contains(errOutput, "not found") {
			t.Errorf("error should mention 'not found', got %q", errOutput)
		}
	})

	t.Run("it errors on duplicate dependency on add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-aaaaaa", "Task A", "open", 2, "", "", []string{"tick-bbbbbb"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "already") || !strings.Contains(errOutput, "blocked by") {
			t.Errorf("error should mention duplicate dependency, got %q", errOutput)
		}
	})

	t.Run("it errors when dependency not found on rm", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "not blocked by") {
			t.Errorf("error should mention dependency not found, got %q", errOutput)
		}
	})

	t.Run("it errors on self-reference on add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-aaaaaa"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "cycle") {
			t.Errorf("error should mention cycle (self-reference creates cycle), got %q", errOutput)
		}
	})

	t.Run("it errors when add creates cycle", func(t *testing.T) {
		dir := setupTickDir(t)
		// tick-aaaaaa is blocked by tick-bbbbbb (existing)
		setupTaskFull(t, dir, "tick-aaaaaa", "Task A", "open", 2, "", "", []string{"tick-bbbbbb"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Trying to add tick-bbbbbb blocked by tick-aaaaaa creates cycle
		code := app.Run([]string{"tick", "dep", "add", "tick-bbbbbb", "tick-aaaaaa"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "cycle") {
			t.Errorf("error should mention cycle, got %q", errOutput)
		}
	})

	t.Run("it errors when add creates child-blocked-by-parent", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-parent", "Parent task")
		setupTaskFull(t, dir, "tick-child1", "Child task", "open", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Trying to add tick-child1 blocked by tick-parent (its parent)
		code := app.Run([]string{"tick", "dep", "add", "tick-child1", "tick-parent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "cannot be blocked by its parent") {
			t.Errorf("error should mention child-blocked-by-parent, got %q", errOutput)
		}
	})

	t.Run("it normalizes IDs to lowercase", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Use uppercase IDs
		code := app.Run([]string{"tick", "dep", "add", "TICK-AAAAAA", "TICK-BBBBBB"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "tick-bbbbbb" {
					t.Errorf("blocked_by = %v, want [tick-bbbbbb]", task.BlockedBy)
				}
				break
			}
		}

		// Output should use normalized lowercase IDs
		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("output should use lowercase task ID, got %q", output)
		}
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("output should use lowercase blocked_by ID, got %q", output)
		}
	})

	t.Run("it suppresses output with --quiet on add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}

		// Verify dependency was still added
		tasks := readTasksFromDir(t, dir)
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "tick-bbbbbb" {
					t.Errorf("dependency should be added even with --quiet")
				}
				break
			}
		}
	})

	t.Run("it suppresses output with --quiet on rm", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-aaaaaa", "Task A", "open", 2, "", "", []string{"tick-bbbbbb"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}

		// Verify dependency was still removed
		tasks := readTasksFromDir(t, dir)
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				if len(task.BlockedBy) != 0 {
					t.Errorf("dependency should be removed even with --quiet")
				}
				break
			}
		}
	})

	t.Run("it errors when fewer than two IDs provided for add", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// No IDs
		code := app.Run([]string{"tick", "dep", "add"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Usage:") {
			t.Errorf("error should include usage hint, got %q", errOutput)
		}

		// One ID only
		stderr.Reset()
		code = app.Run([]string{"tick", "dep", "add", "tick-aaaaaa"})
		if code != 1 {
			t.Errorf("expected exit code 1 for single ID, got %d", code)
		}
		errOutput = stderr.String()
		if !strings.Contains(errOutput, "Usage:") {
			t.Errorf("error should include usage hint, got %q", errOutput)
		}
	})

	t.Run("it errors when fewer than two IDs provided for rm", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// No IDs
		code := app.Run([]string{"tick", "dep", "rm"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Usage:") {
			t.Errorf("error should include usage hint, got %q", errOutput)
		}

		// One ID only
		stderr.Reset()
		code = app.Run([]string{"tick", "dep", "rm", "tick-aaaaaa"})
		if code != 1 {
			t.Errorf("expected exit code 1 for single ID, got %d", code)
		}
		errOutput = stderr.String()
		if !strings.Contains(errOutput, "Usage:") {
			t.Errorf("error should include usage hint, got %q", errOutput)
		}
	})

	t.Run("it persists via atomic write", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-aaaaaa", "Task A")
		setupTask(t, dir, "tick-bbbbbb", "Task B")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"})
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

		// Verify persisted by reading again with different app instance
		tasks := readTasksFromDir(t, dir)
		var found bool
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "tick-bbbbbb" {
					t.Errorf("dependency not persisted correctly")
				}
				found = true
				break
			}
		}
		if !found {
			t.Error("task tick-aaaaaa not found in persisted data")
		}

		// Verify via another operation (rm)
		code = app2.Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}

		tasks = readTasksFromDir(t, dir)
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				if len(task.BlockedBy) != 0 {
					t.Errorf("dependency removal not persisted correctly")
				}
				break
			}
		}
	})

	t.Run("it allows rm to remove stale refs without validating blocked_by_id exists", func(t *testing.T) {
		dir := setupTickDir(t)
		// Task A has a blocked_by reference to a non-existent task (stale ref)
		setupTaskFull(t, dir, "tick-aaaaaa", "Task A", "open", 2, "", "", []string{"tick-deleted"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// rm should work even though tick-deleted doesn't exist as a task
		code := app.Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-deleted"})
		if code != 0 {
			t.Fatalf("expected exit code 0 (rm allows removing stale refs), got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		for _, task := range tasks {
			if task.ID == "tick-aaaaaa" {
				if len(task.BlockedBy) != 0 {
					t.Errorf("stale ref should be removed, got blocked_by = %v", task.BlockedBy)
				}
				break
			}
		}
	})

	t.Run("it errors with unknown dep subcommand", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep", "unknown"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Unknown") || !strings.Contains(errOutput, "dep") {
			t.Errorf("error should mention unknown dep subcommand, got %q", errOutput)
		}
	})

	t.Run("it errors with missing dep subcommand", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "dep"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Usage:") {
			t.Errorf("error should include usage hint, got %q", errOutput)
		}
	})
}
