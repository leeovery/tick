package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestBlockedQuery(t *testing.T) {
	t.Run("it returns open task blocked by open dep", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Blocker task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-blocked should be blocked (has open blocker)
		if !strings.Contains(output, "tick-blocked") {
			t.Errorf("output should contain tick-blocked (blocked by open task), got %q", output)
		}
		// tick-blocker should NOT be blocked (is ready)
		if strings.Contains(output, "tick-blocker") {
			t.Errorf("output should NOT contain tick-blocker (not blocked), got %q", output)
		}
	})

	t.Run("it returns open task blocked by in_progress dep", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "In progress blocker", "in_progress", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-blocked should be blocked (has in_progress blocker)
		if !strings.Contains(output, "tick-blocked") {
			t.Errorf("output should contain tick-blocked (blocked by in_progress task), got %q", output)
		}
	})

	t.Run("it returns parent with open children", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-child", "Open child", "open", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-parent should be blocked (has open child)
		if !strings.Contains(output, "tick-parent") {
			t.Errorf("output should contain tick-parent (has open child), got %q", output)
		}
		// tick-child should NOT be blocked (is ready)
		if strings.Contains(output, "tick-child") {
			t.Errorf("output should NOT contain tick-child (not blocked), got %q", output)
		}
	})

	t.Run("it returns parent with in_progress children", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-child", "In progress child", "in_progress", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-parent should be blocked (has in_progress child)
		if !strings.Contains(output, "tick-parent") {
			t.Errorf("output should contain tick-parent (has in_progress child), got %q", output)
		}
	})

	t.Run("it excludes task when all blockers done", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Done blocker", "done", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "2026-01-19T09:30:00Z")
		setupTaskFull(t, dir, "tick-task", "Unblocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-task should NOT be blocked (blocker is done)
		if strings.Contains(output, "tick-task") {
			t.Errorf("output should NOT contain tick-task (blocker done), got %q", output)
		}
	})

	t.Run("it excludes task when all blockers cancelled", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Cancelled blocker", "cancelled", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "2026-01-19T09:30:00Z")
		setupTaskFull(t, dir, "tick-task", "Unblocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-task should NOT be blocked (blocker is cancelled, which counts as closed)
		if strings.Contains(output, "tick-task") {
			t.Errorf("output should NOT contain tick-task (blocker cancelled), got %q", output)
		}
	})

	t.Run("it excludes in_progress from output", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Blocker task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "In progress blocked", "in_progress", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// in_progress task should NOT be in blocked output (only open tasks)
		if strings.Contains(output, "tick-blocked") {
			t.Errorf("output should NOT contain tick-blocked (in_progress), got %q", output)
		}
	})

	t.Run("it excludes done from output", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should NOT contain tick-a1b2 (done), got %q", output)
		}
	})

	t.Run("it excludes cancelled from output", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Cancelled task", "cancelled", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-a1b2") {
			t.Errorf("output should NOT contain tick-a1b2 (cancelled), got %q", output)
		}
	})

	t.Run("it returns empty when no blocked tasks", func(t *testing.T) {
		dir := setupTickDir(t)
		// All tasks are ready (no blockers, no children)
		setupTaskFull(t, dir, "tick-ready", "Ready task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "blocked"})
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
		// Create a common blocker
		setupTaskFull(t, dir, "tick-blocker", "Blocker", "open", 0, "", "", nil, "2026-01-19T08:00:00Z", "2026-01-19T08:00:00Z", "")
		// Create blocked tasks in non-sorted order
		setupTaskFull(t, dir, "tick-low", "Low priority blocked", "open", 4, "", "", []string{"tick-blocker"}, "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z", "")
		setupTaskFull(t, dir, "tick-high", "High priority blocked", "open", 0, "", "", []string{"tick-blocker"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")
		setupTaskFull(t, dir, "tick-med1", "Medium first blocked", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-med2", "Medium second blocked", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Skip header, check order: priority 0, then 2 (by created), then 4
		if len(lines) < 5 {
			t.Fatalf("expected 5 lines (header + 4 blocked tasks), got %d: %q", len(lines), output)
		}

		// Priority 0 should be first
		if !strings.Contains(lines[1], "tick-high") {
			t.Errorf("first blocked task should be tick-high (priority 0), got %q", lines[1])
		}
		// Priority 2 with earlier created
		if !strings.Contains(lines[2], "tick-med1") {
			t.Errorf("second blocked task should be tick-med1 (priority 2, earlier), got %q", lines[2])
		}
		// Priority 2 with later created
		if !strings.Contains(lines[3], "tick-med2") {
			t.Errorf("third blocked task should be tick-med2 (priority 2, later), got %q", lines[3])
		}
		// Priority 4 should be last
		if !strings.Contains(lines[4], "tick-low") {
			t.Errorf("last blocked task should be tick-low (priority 4), got %q", lines[4])
		}
	})
}

func TestBlockedCommand(t *testing.T) {
	t.Run("it outputs aligned columns via tick blocked", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Blocker", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "blocked"})
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

		code := app.Run([]string{"tick", "--pretty", "blocked"})
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
		setupTaskFull(t, dir, "tick-blocker", "Blocker", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-a1b2", "Blocked task one", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-c3d4", "Blocked task two", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have exactly 2 lines with IDs only (not the blocker which is ready)
		if len(lines) != 2 {
			t.Errorf("expected 2 lines with IDs, got %d: %q", len(lines), output)
		}

		// Should not contain headers or titles
		if strings.Contains(output, "STATUS") {
			t.Errorf("quiet output should not contain STATUS header, got %q", output)
		}
		if strings.Contains(output, "Blocked task") {
			t.Errorf("quiet output should not contain task titles, got %q", output)
		}

		// Should contain blocked task IDs
		if !strings.Contains(output, "tick-a1b2") {
			t.Errorf("quiet output should contain tick-a1b2, got %q", output)
		}
		if !strings.Contains(output, "tick-c3d4") {
			t.Errorf("quiet output should contain tick-c3d4, got %q", output)
		}
	})
}

func TestCancelUnblocksDependents(t *testing.T) {
	t.Run("cancel unblocks single dependent - moves to ready", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Blocker task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// First verify tick-blocked is blocked
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "tick-blocked") {
			t.Fatalf("tick-blocked should initially be blocked, got %q", stdout.String())
		}

		// Cancel the blocker
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "cancel", "tick-blocker"})
		if code != 0 {
			t.Fatalf("expected exit code 0 for cancel, got %d; stderr: %s", code, stderr.String())
		}

		// Verify tick-blocked is now ready (not blocked)
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0 for ready, got %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "tick-blocked") {
			t.Errorf("tick-blocked should now be ready, got %q", stdout.String())
		}

		// Verify tick-blocked is no longer in blocked list
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
		if strings.Contains(stdout.String(), "tick-blocked") {
			t.Errorf("tick-blocked should no longer be blocked, got %q", stdout.String())
		}
	})

	t.Run("cancel unblocks multiple dependents", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker", "Blocker task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-dep1", "First dependent", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-dep2", "Second dependent", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Cancel the blocker
		code := app.Run([]string{"tick", "cancel", "tick-blocker"})
		if code != 0 {
			t.Fatalf("expected exit code 0 for cancel, got %d; stderr: %s", code, stderr.String())
		}

		// Verify both dependents are now ready
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0 for ready, got %d; stderr: %s", code, stderr.String())
		}
		output := stdout.String()
		if !strings.Contains(output, "tick-dep1") {
			t.Errorf("tick-dep1 should now be ready, got %q", output)
		}
		if !strings.Contains(output, "tick-dep2") {
			t.Errorf("tick-dep2 should now be ready, got %q", output)
		}
	})

	t.Run("cancel does not unblock dependent still blocked by another", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-blocker1", "First blocker", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocker2", "Second blocker", "open", 1, "", "", nil, "2026-01-19T09:30:00Z", "2026-01-19T09:30:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked by two", "open", 2, "", "", []string{"tick-blocker1", "tick-blocker2"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Cancel only the first blocker
		code := app.Run([]string{"tick", "cancel", "tick-blocker1"})
		if code != 0 {
			t.Fatalf("expected exit code 0 for cancel, got %d; stderr: %s", code, stderr.String())
		}

		// Verify tick-blocked is still blocked (by blocker2)
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0 for blocked, got %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "tick-blocked") {
			t.Errorf("tick-blocked should still be blocked by tick-blocker2, got %q", stdout.String())
		}

		// Verify tick-blocked is NOT in ready
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0 for ready, got %d; stderr: %s", code, stderr.String())
		}
		if strings.Contains(stdout.String(), "tick-blocked") {
			t.Errorf("tick-blocked should NOT be ready (still blocked), got %q", stdout.String())
		}
	})
}
