package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadyQuery(t *testing.T) {
	t.Run("it returns open task with no blockers and no children", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Simple open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should contain tick-a1b2, got %q", output)
		}
		if !strings.Contains(output, "Simple open task") {
			t.Errorf("output should contain task title, got %q", output)
		}
	})

	t.Run("it excludes task with open blocker", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Blocker task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-blocker should be ready (no blockers, no children)
		if !strings.Contains(output, "tick-blocker") {
			t.Errorf("output should contain tick-blocker (ready), got %q", output)
		}
		// tick-blocked should NOT be ready (has open blocker)
		if strings.Contains(output, "tick-blocked") {
			t.Errorf("output should NOT contain tick-blocked (blocked by open task), got %q", output)
		}
	})

	t.Run("it excludes task with in_progress blocker", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "In progress blocker", "in_progress", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-blocked should NOT be ready (has in_progress blocker)
		if strings.Contains(output, "tick-blocked") {
			t.Errorf("output should NOT contain tick-blocked (blocked by in_progress task), got %q", output)
		}
	})

	t.Run("it includes task when all blockers done", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Done blocker", "done", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "2026-01-19T09:30:00Z")
		setupTaskFull(t, dir, "tick-ready", "Ready task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-ready should be ready (blocker is done)
		if !strings.Contains(output, "tick-ready") {
			t.Errorf("output should contain tick-ready (blocker done), got %q", output)
		}
	})

	t.Run("it includes task when all blockers cancelled", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Cancelled blocker", "cancelled", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "2026-01-19T09:30:00Z")
		setupTaskFull(t, dir, "tick-ready", "Ready task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-ready should be ready (blocker is cancelled, which counts as closed)
		if !strings.Contains(output, "tick-ready") {
			t.Errorf("output should contain tick-ready (blocker cancelled), got %q", output)
		}
	})

	t.Run("it excludes parent with open children", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-child", "Open child", "open", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-child should be ready (leaf task)
		if !strings.Contains(output, "tick-child") {
			t.Errorf("output should contain tick-child (leaf task), got %q", output)
		}
		// tick-parent should NOT be ready (has open child)
		if strings.Contains(output, "tick-parent") {
			t.Errorf("output should NOT contain tick-parent (has open child), got %q", output)
		}
	})

	t.Run("it excludes parent with in_progress children", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-child", "In progress child", "in_progress", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-parent should NOT be ready (has in_progress child)
		if strings.Contains(output, "tick-parent") {
			t.Errorf("output should NOT contain tick-parent (has in_progress child), got %q", output)
		}
	})

	t.Run("it includes parent when all children closed", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-child1", "Done child", "done", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")
		setupTaskFull(t, dir, "tick-child2", "Cancelled child", "cancelled", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-parent should be ready (all children closed)
		if !strings.Contains(output, "tick-parent") {
			t.Errorf("output should contain tick-parent (all children closed), got %q", output)
		}
	})

	t.Run("it excludes in_progress tasks", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "In progress task", "in_progress", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should NOT contain tick-a1b2 (in_progress), got %q", output)
		}
	})

	t.Run("it excludes done tasks", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should NOT contain tick-a1b2 (done), got %q", output)
		}
	})

	t.Run("it excludes cancelled tasks", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Cancelled task", "cancelled", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should NOT contain tick-a1b2 (cancelled), got %q", output)
		}
	})

	t.Run("it handles deep nesting - only deepest incomplete ready", func(t *testing.T) {
		dir := setupTickDir(t)
		// Create a 3-level hierarchy
		setupTaskFull(t, dir, "tick-root", "Root", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-mid", "Middle", "open", 1, "", "tick-root", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-leaf", "Leaf", "open", 1, "", "tick-mid", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Only tick-leaf should be ready (deepest level)
		if !strings.Contains(output, "tick-leaf") {
			t.Errorf("output should contain tick-leaf (deepest incomplete), got %q", output)
		}
		// tick-mid should NOT be ready (has open child)
		if strings.Contains(output, "tick-mid") {
			t.Errorf("output should NOT contain tick-mid (has open child), got %q", output)
		}
		// tick-root should NOT be ready (has open child)
		if strings.Contains(output, "tick-root") {
			t.Errorf("output should NOT contain tick-root (has open child), got %q", output)
		}
	})

	t.Run("it returns empty list when no tasks ready", func(t *testing.T) {
		dir := setupTickDir(t)
		// All tasks are done or blocked
		setupTaskFull(t, dir, "tick-done", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "ready"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}

		output := stdout.String()
		if !strings.Contains(output, "No tasks found.") {
			t.Errorf("output should contain 'No tasks found.', got %q", output)
		}
	})

	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		dir := setupTickDir(t)
		// Create tasks in non-sorted order
		setupTaskFull(t, dir, "tick-low", "Low priority", "open", 4, "", "", nil, "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z", "")
		setupTaskFull(t, dir, "tick-high", "High priority", "open", 0, "", "", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")
		setupTaskFull(t, dir, "tick-med1", "Medium first", "open", 2, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-med2", "Medium second", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Skip header, check order: priority 0, then 2 (by created), then 4
		if len(lines) < 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d", len(lines))
		}

		// Priority 0 should be first
		if !strings.Contains(lines[1], "tick-high") {
			t.Errorf("first task should be tick-high (priority 0), got %q", lines[1])
		}
		// Priority 2 with earlier created
		if !strings.Contains(lines[2], "tick-med1") {
			t.Errorf("second task should be tick-med1 (priority 2, earlier), got %q", lines[2])
		}
		// Priority 2 with later created
		if !strings.Contains(lines[3], "tick-med2") {
			t.Errorf("third task should be tick-med2 (priority 2, later), got %q", lines[3])
		}
		// Priority 4 should be last
		if !strings.Contains(lines[4], "tick-low") {
			t.Errorf("last task should be tick-low (priority 4), got %q", lines[4])
		}
	})
}

func TestReadyCommand(t *testing.T) {
	t.Run("it outputs aligned columns via tick ready", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Ready task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "ready"})
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
	})

	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "ready"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}

		output := stdout.String()
		if !strings.Contains(output, "No tasks found.") {
			t.Errorf("output should contain 'No tasks found.', got %q", output)
		}
	})

	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Ready task one", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-c3d4", "Ready task two", "open", 2, "", "", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "ready"})
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
		if strings.Contains(output, "Ready task") {
			t.Errorf("quiet output should not contain task titles, got %q", output)
		}

		// Should contain IDs
		if !strings.Contains(output, "tick-a1b2") {
			t.Errorf("quiet output should contain tick-a1b2, got %q", output)
		}
		if !strings.Contains(output, "tick-c3d4") {
			t.Errorf("quiet output should contain tick-c3d4, got %q", output)
		}
	})
}
