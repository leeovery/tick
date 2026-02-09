package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestShow(t *testing.T) {
	t.Run("it shows full task details by ID", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Login endpoint")
		tk.Status = task.StatusInProgress
		tk.Priority = 1
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID:       tick-aaaaaa") {
			t.Errorf("expected ID line, got %q", output)
		}
		if !strings.Contains(output, "Title:    Login endpoint") {
			t.Errorf("expected Title line, got %q", output)
		}
		if !strings.Contains(output, "Status:   in_progress") {
			t.Errorf("expected Status line, got %q", output)
		}
		if !strings.Contains(output, "Priority: 1") {
			t.Errorf("expected Priority line, got %q", output)
		}
		if !strings.Contains(output, "Created:") {
			t.Errorf("expected Created line, got %q", output)
		}
		if !strings.Contains(output, "Updated:") {
			t.Errorf("expected Updated line, got %q", output)
		}
	})

	t.Run("it shows blocked_by section with ID, title, and status of each blocker", func(t *testing.T) {
		blocker := task.NewTask("tick-aaaaaa", "Setup Sanctum")
		blocker.Status = task.StatusDone

		blocked := task.NewTask("tick-bbbbbb", "Login endpoint")
		blocked.BlockedBy = []string{"tick-aaaaaa"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, blocked})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Blocked by:") {
			t.Errorf("expected 'Blocked by:' section, got %q", output)
		}
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected blocker ID in output, got %q", output)
		}
		if !strings.Contains(output, "Setup Sanctum") {
			t.Errorf("expected blocker title in output, got %q", output)
		}
		if !strings.Contains(output, "(done)") {
			t.Errorf("expected blocker status in output, got %q", output)
		}
	})

	t.Run("it shows children section with ID, title, and status of each child", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")
		child := task.NewTask("tick-bbbbbb", "Sub-task one")
		child.Parent = "tick-aaaaaa"

		dir := initTickProjectWithTasks(t, []task.Task{parent, child})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Children:") {
			t.Errorf("expected 'Children:' section, got %q", output)
		}
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected child ID in output, got %q", output)
		}
		if !strings.Contains(output, "Sub-task one") {
			t.Errorf("expected child title in output, got %q", output)
		}
		if !strings.Contains(output, "(open)") {
			t.Errorf("expected child status in output, got %q", output)
		}
	})

	t.Run("it shows description section when description is present", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Described task")
		tk.Description = "Implement the login endpoint..."
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Description:") {
			t.Errorf("expected 'Description:' section, got %q", output)
		}
		if !strings.Contains(output, "Implement the login endpoint...") {
			t.Errorf("expected description text in output, got %q", output)
		}
	})

	t.Run("it omits blocked_by section when task has no dependencies", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Independent task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Blocked by:") {
			t.Errorf("expected no 'Blocked by:' section for task without deps, got %q", output)
		}
	})

	t.Run("it omits children section when task has no children", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Childless task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Children:") {
			t.Errorf("expected no 'Children:' section for task without children, got %q", output)
		}
	})

	t.Run("it omits description section when description is empty", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "No description task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Description:") {
			t.Errorf("expected no 'Description:' section for task without description, got %q", output)
		}
	})

	t.Run("it shows parent field with ID and title when parent is set", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")
		child := task.NewTask("tick-bbbbbb", "Child task")
		child.Parent = "tick-aaaaaa"
		dir := initTickProjectWithTasks(t, []task.Task{parent, child})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Parent:") {
			t.Errorf("expected 'Parent:' field, got %q", output)
		}
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected parent ID in output, got %q", output)
		}
		if !strings.Contains(output, "Parent task") {
			t.Errorf("expected parent title in output, got %q", output)
		}
	})

	t.Run("it omits parent field when parent is null", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Orphan task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Parent:") {
			t.Errorf("expected no 'Parent:' field for task without parent, got %q", output)
		}
	})

	t.Run("it shows closed timestamp when task is done or cancelled", func(t *testing.T) {
		closedTime := time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC)
		tk := task.NewTask("tick-aaaaaa", "Done task")
		tk.Status = task.StatusDone
		tk.Closed = &closedTime
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Closed:") {
			t.Errorf("expected 'Closed:' field for done task, got %q", output)
		}
		if !strings.Contains(output, "2026-01-19T14:30:00Z") {
			t.Errorf("expected closed timestamp in output, got %q", output)
		}
	})

	t.Run("it omits closed field when task is open or in_progress", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Open task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Closed:") {
			t.Errorf("expected no 'Closed:' field for open task, got %q", output)
		}
	})

	t.Run("it errors when task ID not found", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "show", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Task 'tick-nonexist' not found") {
			t.Errorf("expected 'not found' error, got %q", stderr.String())
		}
	})

	t.Run("it errors when no ID argument provided to show", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "show"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Task ID is required") {
			t.Errorf("expected 'Task ID is required' error, got %q", stderr.String())
		}
	})

	t.Run("it normalizes input ID to lowercase for show lookup", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Lowercase task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "show", "TICK-AAAAAA"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected task to be found with uppercase ID, got %q", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag on show", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Quiet show task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaaaaa" {
			t.Errorf("expected only task ID, got %q", output)
		}
	})

	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		// This test verifies show uses store.Query (shared lock + freshness).
		// initTickProjectWithTasks writes JSONL but no cache, so Query must rebuild.
		tk := task.NewTask("tick-aaaaaa", "Cached show task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if !strings.Contains(stdout.String(), "tick-aaaaaa") {
			t.Errorf("expected output to contain tick-aaaaaa, got %q", stdout.String())
		}
	})
}
