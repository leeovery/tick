package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestReady_OpenTaskNoBlockersNoChildren(t *testing.T) {
	t.Run("it returns open task with no blockers and no children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Simple task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected output to contain tick-aaa111, got %q", output)
		}
		if !strings.Contains(output, "Simple task") {
			t.Errorf("expected output to contain 'Simple task', got %q", output)
		}
	})
}

func TestReady_ExcludesTaskWithOpenBlocker(t *testing.T) {
	t.Run("it excludes task with open blocker", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-aaa111 should be ready (open, no blockers, no children)
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 in ready output, got %q", output)
		}
		// tick-bbb222 should NOT be ready (blocked by open task)
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 to be excluded (blocked by open task), got %q", output)
		}
	})
}

func TestReady_ExcludesTaskWithInProgressBlocker(t *testing.T) {
	t.Run("it excludes task with in_progress blocker", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker in progress", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 to be excluded (blocked by in_progress task), got %q", output)
		}
	})
}

func TestReady_IncludesTaskWhenAllBlockersDoneOrCancelled(t *testing.T) {
	t.Run("it includes task when all blockers done or cancelled", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker done", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-bbb222", Title: "Blocker cancelled", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-ccc333", Title: "Unblocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111", "tick-bbb222"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 in ready output (all blockers closed), got %q", output)
		}
	})
}

func TestReady_ExcludesParentWithOpenChildren(t *testing.T) {
	t.Run("it excludes parent with open children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Open child", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), Parent: "tick-aaa111"},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Parent should NOT be ready (has open child)
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 to be excluded (has open child), got %q", output)
		}
		// Child should be ready (open, no blockers, no children)
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in ready output, got %q", output)
		}
	})
}

func TestReady_ExcludesParentWithInProgressChildren(t *testing.T) {
	t.Run("it excludes parent with in_progress children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "In-progress child", Status: task.StatusInProgress, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), Parent: "tick-aaa111"},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 to be excluded (has in_progress child), got %q", output)
		}
	})
}

func TestReady_IncludesParentWhenAllChildrenClosed(t *testing.T) {
	t.Run("it includes parent when all children closed", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Done child", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), Parent: "tick-aaa111", Closed: &closed},
			{ID: "tick-ccc333", Title: "Cancelled child", Status: task.StatusCancelled, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), Parent: "tick-aaa111", Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 in ready output (all children closed), got %q", output)
		}
	})
}

func TestReady_ExcludesNonOpenStatuses(t *testing.T) {
	t.Run("it excludes in_progress, done, and cancelled tasks", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "In progress task", Status: task.StatusInProgress, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-ccc333", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), Closed: &closed},
			{ID: "tick-ddd444", Title: "Cancelled task", Status: task.StatusCancelled, Priority: 2, Created: now.Add(3 * time.Minute), Updated: now.Add(3 * time.Minute), Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 in ready output, got %q", output)
		}
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 to be excluded (in_progress), got %q", output)
		}
		if strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 to be excluded (done), got %q", output)
		}
		if strings.Contains(output, "tick-ddd444") {
			t.Errorf("expected tick-ddd444 to be excluded (cancelled), got %q", output)
		}
	})
}

func TestReady_DeepNesting(t *testing.T) {
	t.Run("it handles deep nesting - only deepest incomplete ready", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Grandparent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), Parent: "tick-aaa111"},
			{ID: "tick-ccc333", Title: "Leaf child", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), Parent: "tick-bbb222"},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Only the deepest leaf should be ready
		if !strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 (leaf) in ready output, got %q", output)
		}
		// Parent should NOT be ready (has open child)
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 to be excluded (has open child), got %q", output)
		}
		// Grandparent should NOT be ready (has open child)
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 to be excluded (has open child), got %q", output)
		}
	})
}

func TestReady_EmptyList(t *testing.T) {
	t.Run("it returns empty list when no tasks ready", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			// All tasks are done, none are ready
			{ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if !strings.Contains(output, "tasks[0]") {
			t.Errorf("expected empty list indicator 'tasks[0]', got %q", output)
		}
	})
}

func TestReady_OrderByPriorityThenCreated(t *testing.T) {
	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-low111", Title: "Low priority", Status: task.StatusOpen, Priority: 3, Created: now, Updated: now},
			{ID: "tick-hi2222", Title: "High priority newer", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-hi1111", Title: "High priority older", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-med111", Title: "Med priority", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 4 task rows
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d:\n%s", len(lines), output)
		}

		// Priority 1 first (older then newer), then 2, then 3
		if !strings.Contains(lines[1], "tick-hi1111") {
			t.Errorf("expected first task to be high priority older, got %q", lines[1])
		}
		if !strings.Contains(lines[2], "tick-hi2222") {
			t.Errorf("expected second task to be high priority newer, got %q", lines[2])
		}
		if !strings.Contains(lines[3], "tick-med111") {
			t.Errorf("expected third task to be med priority, got %q", lines[3])
		}
		if !strings.Contains(lines[4], "tick-low111") {
			t.Errorf("expected fourth task to be low priority, got %q", lines[4])
		}
	})
}

func TestReady_AlignedColumnsOutput(t *testing.T) {
	t.Run("it outputs aligned columns via tick ready", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Login endpoint", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d:\n%s", len(lines), output)
		}

		header := lines[0]
		if !strings.Contains(header, "tasks[") {
			t.Errorf("expected header to contain 'tasks[', got %q", header)
		}
		if !strings.Contains(header, "id") {
			t.Errorf("expected header to contain 'id', got %q", header)
		}
		if !strings.Contains(header, "status") {
			t.Errorf("expected header to contain 'status', got %q", header)
		}
		if !strings.Contains(header, "priority") {
			t.Errorf("expected header to contain 'priority', got %q", header)
		}
		if !strings.Contains(header, "title") {
			t.Errorf("expected header to contain 'title', got %q", header)
		}

		// Data rows contain task values
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected output to contain tick-aaa111, got %q", output)
		}
		if !strings.Contains(output, "open") {
			t.Errorf("expected output to contain 'open', got %q", output)
		}
		if !strings.Contains(output, "Setup Sanctum") {
			t.Errorf("expected output to contain 'Setup Sanctum', got %q", output)
		}
	})
}

func TestReady_NoTasksFoundMessage(t *testing.T) {
	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if !strings.Contains(output, "tasks[0]") {
			t.Errorf("expected empty list indicator 'tasks[0]', got %q", output)
		}
	})
}

func TestReady_QuietFlag(t *testing.T) {
	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "First ready", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second ready", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per task), got %d:\n%s", len(lines), output)
		}

		if strings.TrimSpace(lines[0]) != "tick-aaa111" {
			t.Errorf("expected first line to be 'tick-aaa111', got %q", lines[0])
		}
		if strings.TrimSpace(lines[1]) != "tick-bbb222" {
			t.Errorf("expected second line to be 'tick-bbb222', got %q", lines[1])
		}
	})
}
