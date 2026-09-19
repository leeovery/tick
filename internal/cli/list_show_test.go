package cli

import (
	"bytes"
	"encoding/json"
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

	t.Run("it displays tags in show output", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Tagged task", Status: task.StatusOpen, Priority: 2, Tags: []string{"backend", "ui"}, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "Tags:     backend, ui") {
			t.Errorf("should contain tags line, got:\n%s", stdout)
		}
	})

	t.Run("it omits tags section in show output when task has no tags", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No tags task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Tags:") {
			t.Errorf("should not contain Tags section, got:\n%s", stdout)
		}
	})

	t.Run("it displays tags in alphabetical order", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Sorted tags", Status: task.StatusOpen, Priority: 2, Tags: []string{"zebra", "alpha", "middle"}, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "Tags:     alpha, middle, zebra") {
			t.Errorf("tags should be alphabetically sorted, got:\n%s", stdout)
		}
	})

	t.Run("it displays all 10 tags when task has maximum tags", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tags := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet"}
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Max tags", Status: task.StatusOpen, Priority: 2, Tags: tags, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		for _, tag := range tags {
			if !strings.Contains(stdout, tag) {
				t.Errorf("should contain tag %q, got:\n%s", tag, stdout)
			}
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

	t.Run("it displays refs in show output when task has refs", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task with refs", Status: task.StatusOpen, Priority: 2,
				Refs: []string{"gh-123", "JIRA-456"}, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "\nRefs:\n") {
			t.Errorf("stdout should contain 'Refs:' section, got %q", stdout)
		}
		if !strings.Contains(stdout, "  gh-123\n") {
			t.Errorf("stdout should contain ref 'gh-123', got %q", stdout)
		}
		if !strings.Contains(stdout, "  JIRA-456\n") {
			t.Errorf("stdout should contain ref 'JIRA-456', got %q", stdout)
		}
	})

	t.Run("it omits refs section in show when task has no refs", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No refs task", Status: task.StatusOpen, Priority: 2,
				Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Refs:") {
			t.Errorf("stdout should not contain 'Refs:' section, got %q", stdout)
		}
	})

	t.Run("it displays notes in show output when task has notes", func(t *testing.T) {
		now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task with notes", Status: task.StatusInProgress, Priority: 1,
				Notes: []task.Note{
					{Text: "Started investigating", Created: now},
					{Text: "Root cause found", Created: time.Date(2026, 2, 27, 14, 30, 0, 0, time.UTC)},
				},
				Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "\nNotes:\n") {
			t.Errorf("stdout should contain 'Notes:' section, got %q", stdout)
		}
		if !strings.Contains(stdout, "  2026-02-27 10:00  Started investigating\n") {
			t.Errorf("stdout should contain first note, got %q", stdout)
		}
		if !strings.Contains(stdout, "  2026-02-27 14:30  Root cause found\n") {
			t.Errorf("stdout should contain second note, got %q", stdout)
		}
	})

	t.Run("it omits notes section in show when task has no notes", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No notes task", Status: task.StatusOpen, Priority: 2,
				Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "Notes:") {
			t.Errorf("stdout should not contain 'Notes:' section, got %q", stdout)
		}
	})

	t.Run("it displays type in show output when task has a type", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Bug task", Status: task.StatusOpen, Priority: 2,
				Type: "bug", Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "Type:     bug\n") {
			t.Errorf("stdout should contain 'Type:     bug', got %q", stdout)
		}
	})

	t.Run("it displays dash for type in show output when task has no type", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No type task", Status: task.StatusOpen, Priority: 2,
				Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "Type:     -\n") {
			t.Errorf("stdout should contain 'Type:     -', got %q", stdout)
		}
	})

	t.Run("it displays type in post-mutation output after create with --type", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, stderr, exitCode := runCreate(t, dir, "Feature task", "--type", "feature")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "Type:     feature\n") {
			t.Errorf("stdout should contain 'Type:     feature', got %q", stdout)
		}
	})

	t.Run("it displays type in post-mutation output after update", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Feature task", Status: task.StatusOpen, Priority: 2,
				Type: "feature", Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--title", "Updated title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "Type:     feature\n") {
			t.Errorf("stdout should contain 'Type:     feature', got %q", stdout)
		}
	})
}

func TestShowBareField(t *testing.T) {
	created := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	description := "Fix the parser.\n\nSteps:\n  - read the header\n  - validate"

	newProject := func(t *testing.T) string {
		t.Helper()
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Add login", Status: task.StatusOpen, Priority: 2,
				Description: description, Created: created, Updated: created,
				Tags: []string{"api"}, Refs: []string{"https://example.com"},
				Notes: []task.Note{{Text: "looked at it", Created: created}}},
		})
		return dir
	}

	bare := func(t *testing.T, dir string, args ...string) string {
		t.Helper()
		stdout, stderr, code := runShow(t, dir, args...)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		return stdout
	}

	t.Run("it prints a single field's value bare", func(t *testing.T) {
		got := bare(t, newProject(t), "tick-a1b2c3", "--field", "title")
		if got != "Add login\n" {
			t.Errorf("stdout = %q, want %q", got, "Add login\n")
		}
	})

	t.Run("it prints a multi-line description raw", func(t *testing.T) {
		got := bare(t, newProject(t), "tick-a1b2c3", "--field", "description")
		if got != description+"\n" {
			t.Errorf("stdout = %q, want %q", got, description+"\n")
		}
	})

	t.Run("it prints a dash-leading value unescaped", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "- read the header", Status: task.StatusOpen, Priority: 2,
				Created: created, Updated: created},
		})

		got := bare(t, dir, "tick-a1b2c3", "--field", "title")
		if got != "- read the header\n" {
			t.Errorf("stdout = %q, want %q", got, "- read the header\n")
		}
	})

	t.Run("it prints a header-shaped value unescaped", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Add login", Status: task.StatusOpen, Priority: 2,
				Description: "Steps:\nDescription: nested", Created: created, Updated: created},
		})

		got := bare(t, dir, "tick-a1b2c3", "--field", "description")
		if got != "Steps:\nDescription: nested\n" {
			t.Errorf("stdout = %q, want %q", got, "Steps:\nDescription: nested\n")
		}
	})

	for _, format := range []string{"--json", "--pretty", "--toon"} {
		t.Run("it ignores "+format+" for a bare value", func(t *testing.T) {
			dir := newProject(t)

			plain := bare(t, dir, "tick-a1b2c3", "--field", "description")
			formatted := bare(t, dir, "tick-a1b2c3", format, "--field", "description")
			if formatted != plain {
				t.Errorf("stdout with %s = %q, want %q", format, formatted, plain)
			}
		})
	}

	t.Run("it prints nothing for an absent value", func(t *testing.T) {
		got := bare(t, newProject(t), "tick-a1b2c3", "--field", "closed")
		if got != "" {
			t.Errorf("stdout = %q, want empty", got)
		}
	})

	t.Run("it prints nothing for an absent type", func(t *testing.T) {
		got := bare(t, newProject(t), "tick-a1b2c3", "--field", "type")
		if got != "" {
			t.Errorf("stdout = %q, want empty", got)
		}
	})

	t.Run("it prints nothing for an empty description", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Add login", Status: task.StatusOpen, Priority: 2,
				Created: created, Updated: created},
		})

		got := bare(t, dir, "tick-a1b2c3", "--field", "description")
		if got != "" {
			t.Errorf("stdout = %q, want empty", got)
		}
	})

	t.Run("it prints the resolved id for a partial id request", func(t *testing.T) {
		got := bare(t, newProject(t), "a1b2", "--field", "id")
		if got != "tick-a1b2c3\n" {
			t.Errorf("stdout = %q, want %q", got, "tick-a1b2c3\n")
		}
	})

	t.Run("it prints the priority as a number", func(t *testing.T) {
		got := bare(t, newProject(t), "tick-a1b2c3", "--field", "priority")
		if got != "2\n" {
			t.Errorf("stdout = %q, want %q", got, "2\n")
		}
	})

	t.Run("it treats a repeated name as one field", func(t *testing.T) {
		got := bare(t, newProject(t), "tick-a1b2c3", "--field", "title,title")
		if got != "Add login\n" {
			t.Errorf("stdout = %q, want %q", got, "Add login\n")
		}
	})

	for _, tc := range []struct{ field, want string }{
		{"notes", "Notes:\n  2026-02-10 12:00  looked at it\n"},
		{"tags", "Tags:     api\n"},
		{"refs", "Refs:\n  https://example.com\n"},
		{"title,status", "Title:    Add login\nStatus:   open\n"},
	} {
		t.Run("it renders "+tc.field+" as a document not a bare value", func(t *testing.T) {
			got := bare(t, newProject(t), "tick-a1b2c3", "--field", tc.field)
			if got != tc.want {
				t.Errorf("stdout with --field %s = %q, want %q", tc.field, got, tc.want)
			}
		})
	}
}

func TestShowFilteredDocument(t *testing.T) {
	created := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	description := "Fix the parser.\n\nSteps:\n  - read the header"

	richProject := func(t *testing.T) string {
		t.Helper()
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Add login", Status: task.StatusOpen, Priority: 2,
				Description: description, Created: created, Updated: created,
				Tags: []string{"api"}, Refs: []string{"https://example.com"},
				Notes: []task.Note{{Text: "looked at it", Created: created}}},
		})
		return dir
	}

	bareProject := func(t *testing.T) string {
		t.Helper()
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Add login", Status: task.StatusOpen, Priority: 2,
				Created: created, Updated: created},
		})
		return dir
	}

	show := func(t *testing.T, dir string, args ...string) string {
		t.Helper()
		stdout, stderr, code := runShow(t, dir, args...)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		return stdout
	}

	t.Run("it decodes a filtered document", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--toon", "--field", "description,notes")

		doc := decodeToonDoc(t, stdout)
		assertToonKeySet(t, doc, "description", "notes")
		assertToonFields(t, doc, map[string]any{"description": description})
		if rows := toonRows(t, doc, "notes"); len(rows) != 1 {
			t.Fatalf("notes rows = %d, want 1", len(rows))
		}
	})

	t.Run("it renders a list section as a document not a bare value", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--toon", "--field", "tags")

		assertToonStringList(t, decodeToonDoc(t, stdout), "tags", []string{"api"})
	})

	t.Run("it renders a count-zero header for an always-present section", func(t *testing.T) {
		stdout := show(t, bareProject(t), "tick-a1b2c3", "--toon", "--field", "notes")

		if stdout != "notes[0]{index,text,created}:\n" {
			t.Errorf("stdout = %q, want %q", stdout, "notes[0]{index,text,created}:\n")
		}
	})

	t.Run("it prints nothing when every selected name is empty", func(t *testing.T) {
		stdout := show(t, bareProject(t), "tick-a1b2c3", "--toon", "--field", "tags,refs")

		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("it does not carry unselected keys", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--toon", "--field", "title,status")

		assertToonKeySet(t, decodeToonDoc(t, stdout), "title", "status")
	})

	t.Run("it renders only the selected header lines in pretty", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--field", "title,status")

		want := "Title:    Add login\nStatus:   open\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it renders a pretty list section as a labelled document not a bare value", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--field", "tags")

		want := "Tags:     api\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it mixes a pretty header line and a block", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--field", "title,notes")

		want := "Title:    Add login\n\nNotes:\n  2026-02-10 12:00  looked at it\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it renders a dash for a selected empty type in pretty", func(t *testing.T) {
		stdout := show(t, bareProject(t), "tick-a1b2c3", "--field", "type,tags")

		if stdout != "Type:     -\n" {
			t.Errorf("stdout = %q, want %q", stdout, "Type:     -\n")
		}
	})

	t.Run("it prints nothing in pretty for a field the task does not carry", func(t *testing.T) {
		for _, field := range []string{"tags", "refs", "parent", "closed", "blocked_by", "children", "notes", "description"} {
			if stdout := show(t, bareProject(t), "tick-a1b2c3", "--field", field); stdout != "" {
				t.Errorf("stdout for --field %s = %q, want empty", field, stdout)
			}
		}
	})

	t.Run("it has no leading blank line in pretty when no header line survives", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--field", "notes,description")

		want := "Notes:\n  2026-02-10 12:00  looked at it\n" +
			"\n" +
			"Description:\n" +
			"  Fix the parser.\n" +
			"  \n" +
			"  Steps:\n" +
			"    - read the header\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	parseJSON := func(t *testing.T, stdout string) map[string]any {
		t.Helper()
		var parsed map[string]any
		if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout)
		}
		return parsed
	}

	t.Run("it renders the selected keys as one json object", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--json", "--field", "title,status")

		doc := parseJSON(t, stdout)
		assertJSONKeySet(t, doc, "title", "status")
		if doc["title"] != "Add login" {
			t.Errorf("title = %v, want %q", doc["title"], "Add login")
		}
		if doc["status"] != "open" {
			t.Errorf("status = %v, want %q", doc["status"], "open")
		}
	})

	t.Run("it keeps the index on selected json notes", func(t *testing.T) {
		stdout := show(t, richProject(t), "tick-a1b2c3", "--json", "--field", "notes")

		doc := parseJSON(t, stdout)
		assertJSONKeySet(t, doc, "notes")
		notes, ok := doc["notes"].([]any)
		if !ok || len(notes) != 1 {
			t.Fatalf("notes = %v, want 1 entry", doc["notes"])
		}
		note, ok := notes[0].(map[string]any)
		if !ok {
			t.Fatalf("note = %v, want an object", notes[0])
		}
		if note["index"] != float64(1) {
			t.Errorf("note index = %v, want 1", note["index"])
		}
		if note["text"] != "looked at it" {
			t.Errorf("note text = %v, want %q", note["text"], "looked at it")
		}
	})

	t.Run("it renders selected empty json lists as empty arrays", func(t *testing.T) {
		stdout := show(t, bareProject(t), "tick-a1b2c3", "--json", "--field", "tags,refs")

		doc := parseJSON(t, stdout)
		assertJSONKeySet(t, doc, "tags", "refs")
		assertJSONEmptyArray(t, doc, "tags")
		assertJSONEmptyArray(t, doc, "refs")
	})

	t.Run("it prints nothing in json when no selected key survives", func(t *testing.T) {
		stdout := show(t, bareProject(t), "tick-a1b2c3", "--json", "--field", "parent,closed")

		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("it keeps json key order stable across runs", func(t *testing.T) {
		dir := richProject(t)

		first := show(t, dir, "tick-a1b2c3", "--json", "--field", "status,title,notes")
		second := show(t, dir, "tick-a1b2c3", "--json", "--field", "status,title,notes")
		if first != second {
			t.Errorf("repeated runs differ:\n%s\n%s", first, second)
		}
	})
}

func TestShowFieldPositions(t *testing.T) {
	created := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	notes := []task.Note{
		{Text: "first note", Created: created},
		{Text: "second note", Created: created.Add(time.Hour)},
		{Text: "third note", Created: created.Add(2 * time.Hour)},
	}
	tags := []string{"api", "ui"}
	refs := []string{"https://example.com", "https://example.org"}
	firstChild := RelatedTask{ID: "tick-c1c1c1", Title: "Child one", Status: "open"}

	newProject := func(t *testing.T) string {
		t.Helper()
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Add login", Status: task.StatusOpen, Priority: 2,
				Created: created, Updated: created, Tags: tags, Refs: refs, Notes: notes},
			{ID: "tick-c1c1c1", Title: "Child one", Status: task.StatusOpen, Priority: 2,
				Parent: "tick-a1b2c3", Created: created, Updated: created},
			{ID: "tick-c2c2c2", Title: "Child two", Status: task.StatusOpen, Priority: 2,
				Parent: "tick-a1b2c3", Created: created, Updated: created},
		})
		return dir
	}

	show := func(t *testing.T, dir string, args ...string) string {
		t.Helper()
		stdout, stderr, code := runShow(t, dir, args...)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		return stdout
	}

	showToon := func(t *testing.T, field string) map[string]any {
		t.Helper()
		return decodeToonDoc(t, show(t, newProject(t), "tick-a1b2c3", "--toon", "--field", field))
	}

	assertNoteRows := func(t *testing.T, doc map[string]any, want []task.Note, wantPositions []int) {
		t.Helper()
		rows := toonRows(t, doc, "notes")
		if len(rows) != len(want) {
			t.Fatalf("notes rows = %d, want %d", len(rows), len(want))
		}
		for i, row := range rows {
			assertToonFields(t, row, map[string]any{
				"index":   float64(wantPositions[i]),
				"text":    want[i].Text,
				"created": task.FormatTimestamp(want[i].Created),
			})
		}
	}

	t.Run("it narrows the notes section to one position", func(t *testing.T) {
		doc := showToon(t, "title,notes.2")

		assertToonKeySet(t, doc, "title", "notes")
		if rows := toonRows(t, doc, "notes"); len(rows) != 1 {
			t.Fatalf("notes rows = %d, want 1", len(rows))
		}
	})

	t.Run("it keeps the real index on a narrowed note", func(t *testing.T) {
		assertNoteRows(t, showToon(t, "title,notes.2"), notes[1:2], []int{2})
	})

	t.Run("it narrows an inline list to one item", func(t *testing.T) {
		assertToonStringList(t, showToon(t, "title,tags.2"), "tags", []string{"ui"})
	})

	t.Run("it narrows a refs list to one item", func(t *testing.T) {
		assertToonStringList(t, showToon(t, "title,refs.2"), "refs", []string{"https://example.org"})
	})

	t.Run("it narrows a table to one row", func(t *testing.T) {
		assertToonRelatedRow(t, showToon(t, "children.1"), "children", firstChild)
	})

	t.Run("it narrows to several positions in output order", func(t *testing.T) {
		for _, field := range []string{"notes.1,notes.3", "notes.3,notes.1"} {
			assertNoteRows(t, showToon(t, field), []task.Note{notes[0], notes[2]}, []int{1, 3})
		}
	})

	t.Run("it collapses a repeated position", func(t *testing.T) {
		assertNoteRows(t, showToon(t, "title,notes.2,notes.2"), notes[1:2], []int{2})

		stdout := show(t, newProject(t), "tick-a1b2c3", "--field", "notes.2,notes.2")
		if stdout != "second note\n" {
			t.Errorf("stdout = %q, want %q", stdout, "second note\n")
		}
	})

	t.Run("it returns the whole section when named both whole and by position", func(t *testing.T) {
		assertNoteRows(t, showToon(t, "notes,notes.2"), notes, []int{1, 2, 3})
	})

	t.Run("it leaves other selected fields whole", func(t *testing.T) {
		doc := showToon(t, "notes.2,tags")

		assertNoteRows(t, doc, notes[1:2], []int{2})
		assertToonStringList(t, doc, "tags", tags)
	})

	t.Run("it prints a note's text bare for a lone position", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--field", "notes.2")

		if stdout != "second note\n" {
			t.Errorf("stdout = %q, want %q", stdout, "second note\n")
		}
	})

	t.Run("it prints a tag bare for a lone position", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--field", "tags.1")

		if stdout != "api\n" {
			t.Errorf("stdout = %q, want %q", stdout, "api\n")
		}
	})

	t.Run("it prints a ref bare for a lone position", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--field", "refs.2")

		if stdout != "https://example.org\n" {
			t.Errorf("stdout = %q, want %q", stdout, "https://example.org\n")
		}
	})

	t.Run("it returns a one-row section for a lone children position", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--field", "children.1")

		want := "Children:\n  tick-c1c1c1  Child one (open)\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it narrows notes in json with the real index", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--json", "--field", "notes.2,title")

		var doc map[string]any
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout)
		}
		assertJSONKeySet(t, doc, "notes", "title")
		entries, ok := doc["notes"].([]any)
		if !ok || len(entries) != 1 {
			t.Fatalf("notes = %v, want 1 entry", doc["notes"])
		}
		note, ok := entries[0].(map[string]any)
		if !ok {
			t.Fatalf("note = %v, want an object", entries[0])
		}
		if note["index"] != float64(2) {
			t.Errorf("note index = %v, want 2", note["index"])
		}
		if note["text"] != "second note" {
			t.Errorf("note text = %v, want %q", note["text"], "second note")
		}
	})

	t.Run("it narrows notes in pretty", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--pretty", "--field", "notes.2,title")

		want := "Title:    Add login\n\nNotes:\n  2026-02-10 13:00  second note\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it narrows a pretty tags line", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--pretty", "--field", "title,tags.2")

		want := "Title:    Add login\nTags:     ui\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it narrows a pretty children block", func(t *testing.T) {
		stdout := show(t, newProject(t), "tick-a1b2c3", "--pretty", "--field", "title,children.2")

		want := "Title:    Add login\n\nChildren:\n  tick-c2c2c2  Child two (open)\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it renders every item without positions", func(t *testing.T) {
		doc := decodeToonDoc(t, show(t, newProject(t), "tick-a1b2c3", "--toon"))

		assertNoteRows(t, doc, notes, []int{1, 2, 3})
		assertToonStringList(t, doc, "tags", tags)
		if rows := toonRows(t, doc, "children"); len(rows) != 2 {
			t.Fatalf("children rows = %d, want 2", len(rows))
		}
	})
}
