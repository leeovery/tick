package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runList runs the tick list command with the given args and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runList(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "list"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

// runShow runs the tick show command with the given args and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runShow(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "show"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestList(t *testing.T) {
	t.Run("it lists all tasks with aligned columns", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Login endpoint", Status: task.StatusInProgress, Priority: 1, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), stdout)
		}

		// Check header (dynamic column widths based on data: ID=14, STATUS=13, PRI=5)
		header := lines[0]
		if header != "ID            STATUS       PRI  TYPE  TITLE" {
			t.Errorf("header = %q, want %q", header, "ID            STATUS       PRI  TYPE  TITLE")
		}

		// Check first data row
		if lines[1] != "tick-aaa111   done         1    -     Setup Sanctum" {
			t.Errorf("row 1 = %q, want %q", lines[1], "tick-aaa111   done         1    -     Setup Sanctum")
		}

		// Check second data row
		if lines[2] != "tick-bbb222   in_progress  1    -     Login endpoint" {
			t.Errorf("row 2 = %q, want %q", lines[2], "tick-bbb222   in_progress  1    -     Login endpoint")
		}
	})

	t.Run("it lists tasks ordered by priority then created date", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-low111", Title: "Low priority old", Status: task.StatusOpen, Priority: 3, Created: now, Updated: now},
			{ID: "tick-hi2222", Title: "High priority", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-low222", Title: "Low priority new", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-hi1111", Title: "High priority first", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runList(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d: %q", len(lines), stdout)
		}

		// Priority 1 tasks first, ordered by created ASC
		if !strings.HasPrefix(lines[1], "tick-hi1111") {
			t.Errorf("row 1 should start with tick-hi1111, got %q", lines[1])
		}
		if !strings.HasPrefix(lines[2], "tick-hi2222") {
			t.Errorf("row 2 should start with tick-hi2222, got %q", lines[2])
		}
		// Priority 3 tasks next, ordered by created ASC
		if !strings.HasPrefix(lines[3], "tick-low111") {
			t.Errorf("row 3 should start with tick-low111, got %q", lines[3])
		}
		if !strings.HasPrefix(lines[4], "tick-low222") {
			t.Errorf("row 4 should start with tick-low222, got %q", lines[4])
		}
	})

	t.Run("it prints 'No tasks found.' when no tasks exist", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, _, exitCode := runList(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it prints only task IDs with --quiet flag on list", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "First", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runList(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// IDs should be ordered by priority ASC then created ASC
		expected := "tick-aaa111\ntick-bbb222\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Run list twice — second call should still work (cache built on first, fresh on second)
		stdout1, _, exitCode1 := runList(t, dir)
		if exitCode1 != 0 {
			t.Fatalf("first list: exit code = %d, want 0", exitCode1)
		}
		stdout2, _, exitCode2 := runList(t, dir)
		if exitCode2 != 0 {
			t.Fatalf("second list: exit code = %d, want 0", exitCode2)
		}
		if stdout1 != stdout2 {
			t.Errorf("outputs differ: first = %q, second = %q", stdout1, stdout2)
		}
	})
}

func TestShow(t *testing.T) {
	t.Run("it shows full task details by ID", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-c3d4c3", Title: "Login endpoint", Status: task.StatusInProgress, Priority: 1, Created: now, Updated: updated},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-c3d4c3")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "ID:       tick-c3d4c3\n" +
			"Title:    Login endpoint\n" +
			"Status:   in_progress\n" +
			"Priority: 1\n" +
			"Type:     -\n" +
			"Created:  2026-01-19T10:00:00Z\n" +
			"Updated:  2026-01-19T14:30:00Z\n"

		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it shows blocked_by section with ID, title, and status of each blocker", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-aaa111"}, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "\nBlocked by:\n") {
			t.Errorf("stdout should contain 'Blocked by:' section, got %q", stdout)
		}
		if !strings.Contains(stdout, "  tick-aaa111  Setup Sanctum (done)\n") {
			t.Errorf("stdout should contain blocker details, got %q", stdout)
		}
	})

	t.Run("it shows children section with ID, title, and status of each child", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Auth System", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-child1", Title: "Sub-task one", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-parent")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "\nChildren:\n") {
			t.Errorf("stdout should contain 'Children:' section, got %q", stdout)
		}
		if !strings.Contains(stdout, "  tick-child1  Sub-task one (open)\n") {
			t.Errorf("stdout should contain child details, got %q", stdout)
		}
	})

	t.Run("it shows description section when description is present", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Description: "Implement the login endpoint...", Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "\nDescription:\n") {
			t.Errorf("stdout should contain 'Description:' section, got %q", stdout)
		}
		if !strings.Contains(stdout, "  Implement the login endpoint...\n") {
			t.Errorf("stdout should contain description text, got %q", stdout)
		}
	})

	t.Run("it omits blocked_by section when task has no dependencies", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No deps", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Blocked by:") {
			t.Errorf("stdout should not contain 'Blocked by:' section, got %q", stdout)
		}
	})

	t.Run("it omits children section when task has no children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No children", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Children:") {
			t.Errorf("stdout should not contain 'Children:' section, got %q", stdout)
		}
	})

	t.Run("it omits description section when description is empty", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No desc", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Description:") {
			t.Errorf("stdout should not contain 'Description:' section, got %q", stdout)
		}
	})

	t.Run("it shows parent field with ID and title when parent is set", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Auth System", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-child1", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Parent: "tick-parent", Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-child1")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "Parent:   tick-parent (Auth System)\n") {
			t.Errorf("stdout should contain parent field with ID and title, got %q", stdout)
		}
	})

	t.Run("it omits parent field when parent is null", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No parent", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Parent:") {
			t.Errorf("stdout should not contain 'Parent:' field, got %q", stdout)
		}
	})

	t.Run("it shows closed timestamp when task is done or cancelled", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closedTime := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-done11", Title: "Done task", Status: task.StatusDone, Priority: 1, Created: now, Updated: closedTime, Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-done11")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "Closed:   2026-01-19T16:00:00Z\n") {
			t.Errorf("stdout should contain closed timestamp, got %q", stdout)
		}
	})

	t.Run("it omits closed field when task is open or in_progress", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-open11")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Closed:") {
			t.Errorf("stdout should not contain 'Closed:' field, got %q", stdout)
		}
	})

	t.Run("it errors when task ID not found", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runShow(t, dir, "tick-xyz123")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error: task 'tick-xyz123' not found") {
			t.Errorf("stderr = %q, want to contain task not found message", stderr)
		}
	})

	t.Run("it errors when no ID argument provided to show", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runShow(t, dir)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "task ID is required. Usage: tick show <id>") {
			t.Errorf("stderr = %q, want to contain usage hint", stderr)
		}
	})

	t.Run("it normalizes input ID to lowercase for show lookup", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Found it", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "TICK-AAA111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "ID:       tick-aaa111") {
			t.Errorf("stdout should contain the task ID, got %q", stdout)
		}
	})

	t.Run("it outputs only task ID with --quiet flag on show", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Quiet show", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "--quiet", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "tick-aaa111\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Storage flow", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Run show twice — second call should still work (cache built on first, fresh on second)
		stdout1, _, exitCode1 := runShow(t, dir, "tick-aaa111")
		if exitCode1 != 0 {
			t.Fatalf("first show: exit code = %d, want 0", exitCode1)
		}
		stdout2, _, exitCode2 := runShow(t, dir, "tick-aaa111")
		if exitCode2 != 0 {
			t.Fatalf("second show: exit code = %d, want 0", exitCode2)
		}
		if stdout1 != stdout2 {
			t.Errorf("outputs differ: first = %q, second = %q", stdout1, stdout2)
		}
	})

	t.Run("queryShowData populates RelatedTask fields for blockers and children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-child1", Title: "Child one", Status: task.StatusInProgress, Priority: 2, Parent: "tick-parent", Created: now, Updated: now},
			{ID: "tick-blocker", Title: "Blocker task", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-blocked", Title: "Blocked task", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-blocker"}, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		fc := FormatConfig{Format: FormatPretty}
		store, err := openStore(dir, fc)
		if err != nil {
			t.Fatalf("openStore failed: %v", err)
		}
		defer store.Close()

		// Verify children are populated as RelatedTask with exported fields.
		parentData, err := queryShowData(store, "tick-parent")
		if err != nil {
			t.Fatalf("queryShowData for parent failed: %v", err)
		}
		if len(parentData.children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(parentData.children))
		}
		child := parentData.children[0]
		if child.ID != "tick-child1" {
			t.Errorf("child.ID = %q, want %q", child.ID, "tick-child1")
		}
		if child.Title != "Child one" {
			t.Errorf("child.Title = %q, want %q", child.Title, "Child one")
		}
		if child.Status != "in_progress" {
			t.Errorf("child.Status = %q, want %q", child.Status, "in_progress")
		}

		// Verify blockedBy are populated as RelatedTask with exported fields.
		blockedData, err := queryShowData(store, "tick-blocked")
		if err != nil {
			t.Fatalf("queryShowData for blocked task failed: %v", err)
		}
		if len(blockedData.blockedBy) != 1 {
			t.Fatalf("expected 1 blocker, got %d", len(blockedData.blockedBy))
		}
		blocker := blockedData.blockedBy[0]
		if blocker.ID != "tick-blocker" {
			t.Errorf("blocker.ID = %q, want %q", blocker.ID, "tick-blocker")
		}
		if blocker.Title != "Blocker task" {
			t.Errorf("blocker.Title = %q, want %q", blocker.Title, "Blocker task")
		}
		if blocker.Status != "done" {
			t.Errorf("blocker.Status = %q, want %q", blocker.Status, "done")
		}
	})
}
