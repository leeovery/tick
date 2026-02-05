package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestListFilters(t *testing.T) {
	t.Run("it filters to ready tasks with --ready", func(t *testing.T) {
		dir := setupTickDir(t)
		// Ready: open, no blockers, no open children
		setupTaskFull(t, dir, "tick-ready", "Ready task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		// Blocked: open, has open blocker
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-ready"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-ready should be in output (ready)
		if !strings.Contains(output, "tick-ready") {
			t.Errorf("output should contain tick-ready, got %q", output)
		}
		// tick-blocked should NOT be in output (blocked)
		if strings.Contains(output, "tick-blocked") {
			t.Errorf("output should NOT contain tick-blocked, got %q", output)
		}
	})

	t.Run("it filters to blocked tasks with --blocked", func(t *testing.T) {
		dir := setupTickDir(t)
		// Ready: open, no blockers, no open children
		setupTaskFull(t, dir, "tick-ready", "Ready task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		// Blocked: open, has open blocker
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-ready"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-blocked should be in output (blocked)
		if !strings.Contains(output, "tick-blocked") {
			t.Errorf("output should contain tick-blocked, got %q", output)
		}
		// tick-ready should NOT be in output (ready, not blocked)
		if strings.Contains(output, "tick-ready") {
			t.Errorf("output should NOT contain tick-ready, got %q", output)
		}
	})

	t.Run("it filters by --status open", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-done", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--status", "open"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-open") {
			t.Errorf("output should contain tick-open, got %q", output)
		}
		if strings.Contains(output, "tick-done") {
			t.Errorf("output should NOT contain tick-done, got %q", output)
		}
	})

	t.Run("it filters by --status in_progress", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-progress", "In progress task", "in_progress", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--status", "in_progress"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-progress") {
			t.Errorf("output should contain tick-progress, got %q", output)
		}
		if strings.Contains(output, "tick-open") {
			t.Errorf("output should NOT contain tick-open, got %q", output)
		}
	})

	t.Run("it filters by --status done", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-done", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--status", "done"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-done") {
			t.Errorf("output should contain tick-done, got %q", output)
		}
		if strings.Contains(output, "tick-open") {
			t.Errorf("output should NOT contain tick-open, got %q", output)
		}
	})

	t.Run("it filters by --status cancelled", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-cancelled", "Cancelled task", "cancelled", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--status", "cancelled"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-cancelled") {
			t.Errorf("output should contain tick-cancelled, got %q", output)
		}
		if strings.Contains(output, "tick-open") {
			t.Errorf("output should NOT contain tick-open, got %q", output)
		}
	})

	t.Run("it filters by --priority", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-high", "High priority", "open", 0, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-med", "Medium priority", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-low", "Low priority", "open", 4, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--priority", "2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-med") {
			t.Errorf("output should contain tick-med, got %q", output)
		}
		if strings.Contains(output, "tick-high") {
			t.Errorf("output should NOT contain tick-high, got %q", output)
		}
		if strings.Contains(output, "tick-low") {
			t.Errorf("output should NOT contain tick-low, got %q", output)
		}
	})

	t.Run("it combines --ready with --priority", func(t *testing.T) {
		dir := setupTickDir(t)
		// Ready P1
		setupTaskFull(t, dir, "tick-ready-p1", "Ready P1", "open", 1, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		// Ready P2
		setupTaskFull(t, dir, "tick-ready-p2", "Ready P2", "open", 2, "", "", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")
		// Blocked P1
		setupTaskFull(t, dir, "tick-blocked-p1", "Blocked P1", "open", 1, "", "", []string{"tick-ready-p1"}, "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--ready", "--priority", "1"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Only tick-ready-p1 should match (ready AND priority 1)
		if !strings.Contains(output, "tick-ready-p1") {
			t.Errorf("output should contain tick-ready-p1, got %q", output)
		}
		if strings.Contains(output, "tick-ready-p2") {
			t.Errorf("output should NOT contain tick-ready-p2 (wrong priority), got %q", output)
		}
		if strings.Contains(output, "tick-blocked-p1") {
			t.Errorf("output should NOT contain tick-blocked-p1 (blocked), got %q", output)
		}
	})

	t.Run("it combines --status with --priority", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-done-p1", "Done P1", "done", 1, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")
		setupTaskFull(t, dir, "tick-done-p2", "Done P2", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")
		setupTaskFull(t, dir, "tick-open-p1", "Open P1", "open", 1, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--status", "done", "--priority", "1"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Only tick-done-p1 should match (done AND priority 1)
		if !strings.Contains(output, "tick-done-p1") {
			t.Errorf("output should contain tick-done-p1, got %q", output)
		}
		if strings.Contains(output, "tick-done-p2") {
			t.Errorf("output should NOT contain tick-done-p2 (wrong priority), got %q", output)
		}
		if strings.Contains(output, "tick-open-p1") {
			t.Errorf("output should NOT contain tick-open-p1 (wrong status), got %q", output)
		}
	})

	t.Run("it errors when --ready and --blocked both set", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Test task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--ready", "--blocked"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "--ready") || !strings.Contains(errOutput, "--blocked") {
			t.Errorf("error should mention both --ready and --blocked, got %q", errOutput)
		}
		if !strings.Contains(errOutput, "mutually exclusive") {
			t.Errorf("error should mention 'mutually exclusive', got %q", errOutput)
		}
	})

	t.Run("it errors for invalid status value", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--status", "invalid"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "invalid") {
			t.Errorf("error should mention invalid value, got %q", errOutput)
		}
		// Should list valid options
		if !strings.Contains(errOutput, "open") || !strings.Contains(errOutput, "in_progress") ||
			!strings.Contains(errOutput, "done") || !strings.Contains(errOutput, "cancelled") {
			t.Errorf("error should list valid options, got %q", errOutput)
		}
	})

	t.Run("it errors for invalid priority value", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--priority", "5"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "priority") {
			t.Errorf("error should mention priority, got %q", errOutput)
		}
		if !strings.Contains(errOutput, "0") || !strings.Contains(errOutput, "4") {
			t.Errorf("error should mention valid range 0-4, got %q", errOutput)
		}
	})

	t.Run("it errors for non-numeric priority value", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list", "--priority", "high"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "priority") {
			t.Errorf("error should mention priority, got %q", errOutput)
		}
	})

	t.Run("it returns 'No tasks found.' when no matches", func(t *testing.T) {
		dir := setupTickDir(t)
		// Only open tasks exist
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Search for done tasks (none exist)
		code := app.Run([]string{"tick", "list", "--status", "done"})
		if code != 0 {
			t.Fatalf("expected exit code 0 (not error), got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "No tasks found.") {
			t.Errorf("output should contain 'No tasks found.', got %q", output)
		}
	})

	t.Run("it returns empty result for contradictory filters (--status done --ready)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-open", "Open ready", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-done", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// --status done --ready is contradictory (ready requires status=open)
		code := app.Run([]string{"tick", "list", "--status", "done", "--ready"})
		// Should not error, just return empty
		if code != 0 {
			t.Fatalf("expected exit code 0 (not error), got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "No tasks found.") {
			t.Errorf("contradictory filters should return 'No tasks found.', got %q", output)
		}
	})

	t.Run("it outputs IDs only with --quiet after filtering", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-done1", "Done 1", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")
		setupTaskFull(t, dir, "tick-done2", "Done 2", "done", 2, "", "", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "2026-01-19T12:00:00Z")
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "list", "--status", "done"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have exactly 2 IDs (done tasks only)
		if len(lines) != 2 {
			t.Errorf("expected 2 lines with IDs, got %d: %q", len(lines), output)
		}

		// Should not contain headers or titles
		if strings.Contains(output, "STATUS") {
			t.Errorf("quiet output should not contain STATUS header, got %q", output)
		}
		if strings.Contains(output, "Done 1") || strings.Contains(output, "Done 2") {
			t.Errorf("quiet output should not contain task titles, got %q", output)
		}

		// Should contain IDs
		if !strings.Contains(output, "tick-done1") {
			t.Errorf("quiet output should contain tick-done1, got %q", output)
		}
		if !strings.Contains(output, "tick-done2") {
			t.Errorf("quiet output should contain tick-done2, got %q", output)
		}
		// Should NOT contain open task
		if strings.Contains(output, "tick-open") {
			t.Errorf("quiet output should NOT contain tick-open, got %q", output)
		}
	})

	t.Run("it returns all tasks with no filters", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-open", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-progress", "In progress", "in_progress", 2, "", "", nil, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")
		setupTaskFull(t, dir, "tick-done", "Done task", "done", 2, "", "", nil, "2026-01-19T12:00:00Z", "2026-01-19T12:00:00Z", "2026-01-19T13:00:00Z")
		setupTaskFull(t, dir, "tick-cancelled", "Cancelled task", "cancelled", 2, "", "", nil, "2026-01-19T14:00:00Z", "2026-01-19T14:00:00Z", "2026-01-19T15:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// All tasks should be in output
		if !strings.Contains(output, "tick-open") {
			t.Errorf("output should contain tick-open, got %q", output)
		}
		if !strings.Contains(output, "tick-progress") {
			t.Errorf("output should contain tick-progress, got %q", output)
		}
		if !strings.Contains(output, "tick-done") {
			t.Errorf("output should contain tick-done, got %q", output)
		}
		if !strings.Contains(output, "tick-cancelled") {
			t.Errorf("output should contain tick-cancelled, got %q", output)
		}
	})

	t.Run("it maintains deterministic ordering", func(t *testing.T) {
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

		code := app.Run([]string{"tick", "list", "--status", "open"})
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

	t.Run("it matches tick ready output", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-ready", "Ready task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-ready"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdoutList, stderrList bytes.Buffer
		appList := &App{
			Stdout: &stdoutList,
			Stderr: &stderrList,
			Cwd:    dir,
		}

		codeList := appList.Run([]string{"tick", "list", "--ready"})
		if codeList != 0 {
			t.Fatalf("list --ready expected exit code 0, got %d; stderr: %s", codeList, stderrList.String())
		}

		var stdoutReady, stderrReady bytes.Buffer
		appReady := &App{
			Stdout: &stdoutReady,
			Stderr: &stderrReady,
			Cwd:    dir,
		}

		codeReady := appReady.Run([]string{"tick", "ready"})
		if codeReady != 0 {
			t.Fatalf("ready expected exit code 0, got %d; stderr: %s", codeReady, stderrReady.String())
		}

		// Output should match
		if stdoutList.String() != stdoutReady.String() {
			t.Errorf("list --ready should match tick ready\nlist --ready: %q\ntick ready: %q", stdoutList.String(), stdoutReady.String())
		}
	})

	t.Run("it matches tick blocked output", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-ready", "Ready task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-ready"}, "2026-01-19T11:00:00Z", "2026-01-19T11:00:00Z", "")

		var stdoutList, stderrList bytes.Buffer
		appList := &App{
			Stdout: &stdoutList,
			Stderr: &stderrList,
			Cwd:    dir,
		}

		codeList := appList.Run([]string{"tick", "list", "--blocked"})
		if codeList != 0 {
			t.Fatalf("list --blocked expected exit code 0, got %d; stderr: %s", codeList, stderrList.String())
		}

		var stdoutBlocked, stderrBlocked bytes.Buffer
		appBlocked := &App{
			Stdout: &stdoutBlocked,
			Stderr: &stderrBlocked,
			Cwd:    dir,
		}

		codeBlocked := appBlocked.Run([]string{"tick", "blocked"})
		if codeBlocked != 0 {
			t.Fatalf("blocked expected exit code 0, got %d; stderr: %s", codeBlocked, stderrBlocked.String())
		}

		// Output should match
		if stdoutList.String() != stdoutBlocked.String() {
			t.Errorf("list --blocked should match tick blocked\nlist --blocked: %q\ntick blocked: %q", stdoutList.String(), stdoutBlocked.String())
		}
	})
}
