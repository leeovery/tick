package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommand(t *testing.T) {
	t.Run("it lists all tasks with aligned columns", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Setup Sanctum")
		setupTask(t, dir, "tick-c3d4", "Login endpoint")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Check header
		if !strings.Contains(output, "ID") {
			t.Errorf("output should contain ID header, got %q", output)
		}
		if !strings.Contains(output, "STATUS") {
			t.Errorf("output should contain STATUS header, got %q", output)
		}
		if !strings.Contains(output, "PRI") {
			t.Errorf("output should contain PRI header, got %q", output)
		}
		if !strings.Contains(output, "TITLE") {
			t.Errorf("output should contain TITLE header, got %q", output)
		}

		// Check task content
		if !strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should contain tick-a1b2, got %q", output)
		}
		if !strings.Contains(output, "Setup Sanctum") {
			t.Errorf("output should contain 'Setup Sanctum', got %q", output)
		}
	})

	t.Run("it lists tasks ordered by priority then created date", func(t *testing.T) {
		dir := setupTickDir(t)
		// Create tasks with different priorities
		setupTaskWithPriority(t, dir, "tick-low", "Low priority task", 4, "2026-01-19T12:00:00Z")
		setupTaskWithPriority(t, dir, "tick-high", "High priority task", 0, "2026-01-19T11:00:00Z")
		setupTaskWithPriority(t, dir, "tick-med1", "Medium priority first", 2, "2026-01-19T09:00:00Z")
		setupTaskWithPriority(t, dir, "tick-med2", "Medium priority second", 2, "2026-01-19T10:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Skip header, check order: priority 0 first, then 2 (by created), then 4
		// First data line should be priority 0
		if len(lines) < 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d", len(lines))
		}

		// Priority 0 should be first (tick-high)
		if !strings.Contains(lines[1], "tick-high") {
			t.Errorf("first task should be tick-high (priority 0), got %q", lines[1])
		}
		// Priority 2 with earlier created date should be next (tick-med1)
		if !strings.Contains(lines[2], "tick-med1") {
			t.Errorf("second task should be tick-med1 (priority 2, earlier), got %q", lines[2])
		}
		// Priority 2 with later created date (tick-med2)
		if !strings.Contains(lines[3], "tick-med2") {
			t.Errorf("third task should be tick-med2 (priority 2, later), got %q", lines[3])
		}
		// Priority 4 should be last (tick-low)
		if !strings.Contains(lines[4], "tick-low") {
			t.Errorf("last task should be tick-low (priority 4), got %q", lines[4])
		}
	})

	t.Run("it prints 'No tasks found.' when no tasks exist", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}

		output := stdout.String()
		if !strings.Contains(output, "No tasks found.") {
			t.Errorf("output should contain 'No tasks found.', got %q", output)
		}
	})

	t.Run("it prints only task IDs with --quiet flag on list", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "First task")
		setupTask(t, dir, "tick-c3d4", "Second task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have exactly 2 lines with IDs only
		if len(lines) != 2 {
			t.Errorf("expected 2 lines with IDs, got %d: %q", len(lines), output)
		}

		// Should not contain headers or titles
		if strings.Contains(output, "STATUS") {
			t.Errorf("quiet output should not contain STATUS header, got %q", output)
		}
		if strings.Contains(output, "First task") {
			t.Errorf("quiet output should not contain task titles, got %q", output)
		}
	})

	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// After list runs, cache.db should exist (created during query)
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Error("cache.db should be created during list command")
		}
	})
}

func TestShowCommand(t *testing.T) {
	t.Run("it shows full task details by ID", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Setup Sanctum", "open", 1, "Full description here", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T14:30:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID:") || !strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should contain ID, got %q", output)
		}
		if !strings.Contains(output, "Title:") || !strings.Contains(output, "Setup Sanctum") {
			t.Errorf("output should contain Title, got %q", output)
		}
		if !strings.Contains(output, "Status:") || !strings.Contains(output, "open") {
			t.Errorf("output should contain Status, got %q", output)
		}
		if !strings.Contains(output, "Priority:") || !strings.Contains(output, "1") {
			t.Errorf("output should contain Priority, got %q", output)
		}
		if !strings.Contains(output, "Created:") {
			t.Errorf("output should contain Created, got %q", output)
		}
		if !strings.Contains(output, "Updated:") {
			t.Errorf("output should contain Updated, got %q", output)
		}
	})

	t.Run("it shows blocked_by section with ID, title, and status of each blocker", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Blocker task", "done", 1, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Blocked by:") {
			t.Errorf("output should contain 'Blocked by:' section, got %q", output)
		}
		if !strings.Contains(output, "tick-blocker") {
			t.Errorf("blocked_by section should contain blocker ID, got %q", output)
		}
		if !strings.Contains(output, "Blocker task") {
			t.Errorf("blocked_by section should contain blocker title, got %q", output)
		}
		if !strings.Contains(output, "(done)") {
			t.Errorf("blocked_by section should contain blocker status, got %q", output)
		}
	})

	t.Run("it shows children section with ID, title, and status of each child", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-child1", "Child task one", "open", 2, "", "tick-parent", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")
		setupTaskFull(t, dir, "tick-child2", "Child task two", "done", 2, "", "tick-parent", nil, "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-parent"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Children:") {
			t.Errorf("output should contain 'Children:' section, got %q", output)
		}
		if !strings.Contains(output, "tick-child1") {
			t.Errorf("children section should contain child1 ID, got %q", output)
		}
		if !strings.Contains(output, "Child task one") {
			t.Errorf("children section should contain child1 title, got %q", output)
		}
		if !strings.Contains(output, "tick-child2") {
			t.Errorf("children section should contain child2 ID, got %q", output)
		}
	})

	t.Run("it shows description section when description is present", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Task with description", "open", 2, "This is a detailed description.\nWith multiple lines.", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Description:") {
			t.Errorf("output should contain 'Description:' section, got %q", output)
		}
		if !strings.Contains(output, "This is a detailed description.") {
			t.Errorf("output should contain description text, got %q", output)
		}
	})

	t.Run("it omits blocked_by section when task has no dependencies", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "No deps task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Blocked by:") {
			t.Errorf("output should not contain 'Blocked by:' section when empty, got %q", output)
		}
	})

	t.Run("it omits children section when task has no children", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "No children task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Children:") {
			t.Errorf("output should not contain 'Children:' section when empty, got %q", output)
		}
	})

	t.Run("it omits description section when description is empty", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "No description task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Description:") {
			t.Errorf("output should not contain 'Description:' section when empty, got %q", output)
		}
	})

	t.Run("it shows parent field with ID and title when parent is set", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-child", "Child task", "open", 2, "", "tick-parent", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-child"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Parent:") {
			t.Errorf("output should contain 'Parent:' field, got %q", output)
		}
		if !strings.Contains(output, "tick-parent") {
			t.Errorf("Parent field should contain parent ID, got %q", output)
		}
		if !strings.Contains(output, "Parent task") {
			t.Errorf("Parent field should contain parent title, got %q", output)
		}
	})

	t.Run("it omits parent field when parent is null", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "No parent task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Parent:") {
			t.Errorf("output should not contain 'Parent:' field when null, got %q", output)
		}
	})

	t.Run("it shows closed timestamp when task is done or cancelled", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-done", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T14:00:00Z", "2026-01-19T14:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-done"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Closed:") {
			t.Errorf("output should contain 'Closed:' field for done task, got %q", output)
		}
		if !strings.Contains(output, "2026-01-19T14:00:00Z") {
			t.Errorf("output should contain closed timestamp, got %q", output)
		}
	})

	t.Run("it omits closed field when task is open or in_progress", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-open"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Closed:") {
			t.Errorf("output should not contain 'Closed:' field for open task, got %q", output)
		}
	})

	t.Run("it errors when task ID not found", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.HasPrefix(errOutput, "Error: ") {
			t.Errorf("error should start with 'Error: ', got %q", errOutput)
		}
		if !strings.Contains(errOutput, "tick-nonexistent") {
			t.Errorf("error should contain task ID, got %q", errOutput)
		}
		if !strings.Contains(errOutput, "not found") {
			t.Errorf("error should contain 'not found', got %q", errOutput)
		}
	})

	t.Run("it errors when no ID argument provided to show", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "show"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.HasPrefix(errOutput, "Error: ") {
			t.Errorf("error should start with 'Error: ', got %q", errOutput)
		}
		if !strings.Contains(errOutput, "Task ID is required") {
			t.Errorf("error should contain 'Task ID is required', got %q", errOutput)
		}
		if !strings.Contains(errOutput, "Usage:") {
			t.Errorf("error should contain 'Usage:', got %q", errOutput)
		}
	})

	t.Run("it normalizes input ID to lowercase for show lookup", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Test task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Use uppercase ID
		code := app.Run([]string{"tick", "--pretty", "show", "TICK-A1B2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-a1b2") {
			t.Errorf("should find task with normalized ID, got %q", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag on show", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Test task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-a1b2" {
			t.Errorf("quiet output should be task ID only, got %q", output)
		}
	})

	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Test task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "show", "tick-a1b2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// After show runs, cache.db should exist (created during query)
		cachePath := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Error("cache.db should be created during show command")
		}
	})
}

