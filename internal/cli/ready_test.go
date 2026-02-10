package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runReady runs the tick ready command with the given args and returns stdout, stderr, and exit code.
func runReady(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	fullArgs := append([]string{"tick", "ready"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestReady(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

	t.Run("it returns open task with no blockers and no children", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Simple task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (header + 1 task), got %d: %q", len(lines), stdout)
		}
		if !strings.HasPrefix(lines[1], "tick-aaa111") {
			t.Errorf("row 1 should start with tick-aaa111, got %q", lines[1])
		}
	})

	t.Run("it excludes task with open blocker", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Depends on open", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		// Only the blocker task should be ready, not the dependent
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-dep111") {
				t.Error("dependent task with open blocker should not appear in ready")
			}
		}
		// The blocker itself should be ready (open, no blockers, no children)
		found := false
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-blk111") {
				found = true
			}
		}
		if !found {
			t.Error("blocker task (open, no deps) should appear in ready")
		}
	})

	t.Run("it excludes task with in_progress blocker", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker in progress", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Depends on in_progress", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// The dependent should not be ready (blocker is in_progress)
		if strings.Contains(stdout, "tick-dep111") {
			t.Error("dependent task with in_progress blocker should not appear in ready")
		}
	})

	t.Run("it includes task when all blockers done", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker done", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
			{ID: "tick-dep111", Title: "Unblocked", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("task with all blockers done should appear in ready")
		}
	})

	t.Run("it includes task when all blockers cancelled", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker cancelled", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
			{ID: "tick-dep111", Title: "Unblocked by cancel", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("task with all blockers cancelled should appear in ready")
		}
	})

	t.Run("it excludes parent with open children", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-child1", Title: "Open child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Parent should not be ready (has open children)
		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-parent") {
				t.Error("parent with open children should not appear in ready")
			}
		}
		// The child should be ready
		found := false
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-child1") {
				found = true
			}
		}
		if !found {
			t.Error("open child (leaf) should appear in ready")
		}
	})

	t.Run("it excludes parent with in_progress children", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-child1", Title: "In progress child", Status: task.StatusInProgress, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Parent should not be ready (has in_progress children)
		if strings.Contains(stdout, "tick-parent") {
			t.Error("parent with in_progress children should not appear in ready")
		}
	})

	t.Run("it includes parent when all children closed", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-child1", Title: "Done child", Status: task.StatusDone, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
			{ID: "tick-child2", Title: "Cancelled child", Status: task.StatusCancelled, Priority: 2, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-parent") {
			t.Error("parent with all children closed should appear in ready")
		}
	})

	t.Run("it excludes in_progress tasks", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "In progress", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-aaa111") {
			t.Error("in_progress task should not appear in ready")
		}
	})

	t.Run("it excludes done tasks", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-aaa111") {
			t.Error("done task should not appear in ready")
		}
	})

	t.Run("it excludes cancelled tasks", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Cancelled task", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-aaa111") {
			t.Error("cancelled task should not appear in ready")
		}
	})

	t.Run("it handles deep nesting - only deepest incomplete ready", func(t *testing.T) {
		// Grandparent -> Parent -> Child (leaf)
		// Only the deepest leaf should be ready
		tasks := []task.Task{
			{ID: "tick-gp0001", Title: "Grandparent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-par001", Title: "Parent", Status: task.StatusOpen, Priority: 2, Parent: "tick-gp0001", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-child1", Title: "Leaf child", Status: task.StatusOpen, Priority: 2, Parent: "tick-par001", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		// Only the leaf should appear
		if !strings.Contains(stdout, "tick-child1") {
			t.Error("deepest leaf child should appear in ready")
		}
		// Grandparent and parent should NOT appear (they have open children)
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-gp0001") {
				t.Error("grandparent with open children should not appear in ready")
			}
			if strings.HasPrefix(line, "tick-par001") {
				t.Error("parent with open children should not appear in ready")
			}
		}
	})

	t.Run("it returns empty list when no tasks ready", func(t *testing.T) {
		// All tasks are done, none are open
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-low111", Title: "Low priority old", Status: task.StatusOpen, Priority: 3, Created: now, Updated: now},
			{ID: "tick-hi2222", Title: "High priority new", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-low222", Title: "Low priority new", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-hi1111", Title: "High priority old", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
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

	t.Run("it outputs aligned columns via tick ready", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Login endpoint", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), stdout)
		}

		// Check header matches list format
		header := lines[0]
		if header != "ID          STATUS       PRI  TITLE" {
			t.Errorf("header = %q, want %q", header, "ID          STATUS       PRI  TITLE")
		}

		// Check aligned rows
		if lines[1] != "tick-aaa111 open         1    Setup Sanctum" {
			t.Errorf("row 1 = %q, want %q", lines[1], "tick-aaa111 open         1    Setup Sanctum")
		}
		if lines[2] != "tick-bbb222 open         2    Login endpoint" {
			t.Errorf("row 2 = %q, want %q", lines[2], "tick-bbb222 open         2    Login endpoint")
		}
	})

	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "First", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runReady(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "tick-aaa111\ntick-bbb222\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})
}
