package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestBlocked_ReturnsOpenTaskBlockedByOpenDep(t *testing.T) {
	t.Run("it returns open task blocked by open dep", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-bbb222 should be blocked (has open blocker)
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in blocked output, got %q", output)
		}
		// tick-aaa111 should NOT be blocked (it's ready, not blocked)
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 to be excluded (it is ready, not blocked), got %q", output)
		}
	})
}

func TestBlocked_ReturnsOpenTaskBlockedByInProgressDep(t *testing.T) {
	t.Run("it returns open task blocked by in_progress dep", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker in progress", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in blocked output (blocked by in_progress dep), got %q", output)
		}
	})
}

func TestBlocked_ReturnsParentWithOpenChildren(t *testing.T) {
	t.Run("it returns parent with open children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Open child", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), Parent: "tick-aaa111"},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Parent should be blocked (has open child)
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 in blocked output (has open child), got %q", output)
		}
		// Child should NOT be blocked (it is ready - open, no blockers, no children)
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 to be excluded (it is ready), got %q", output)
		}
	})
}

func TestBlocked_ReturnsParentWithInProgressChildren(t *testing.T) {
	t.Run("it returns parent with in_progress children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "In-progress child", Status: task.StatusInProgress, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), Parent: "tick-aaa111"},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 in blocked output (has in_progress child), got %q", output)
		}
	})
}

func TestBlocked_ExcludesTaskWhenAllBlockersDoneOrCancelled(t *testing.T) {
	t.Run("it excludes task when all blockers done or cancelled", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker done", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-bbb222", Title: "Blocker cancelled", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-ccc333", Title: "Unblocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111", "tick-bbb222"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-ccc333 should NOT be blocked (all blockers are closed)
		if strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 to be excluded (all blockers closed), got %q", output)
		}
	})
}

func TestBlocked_ExcludesNonOpenStatuses(t *testing.T) {
	t.Run("it excludes in_progress, done, and cancelled from output", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "In-progress task with blocker", Status: task.StatusInProgress, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Done task with blocker", Status: task.StatusDone, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), BlockedBy: []string{"tick-aaa111"}, Closed: &closed},
			{ID: "tick-ddd444", Title: "Cancelled task with blocker", Status: task.StatusCancelled, Priority: 2, Created: now.Add(3 * time.Minute), Updated: now.Add(3 * time.Minute), BlockedBy: []string{"tick-aaa111"}, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// None of the non-open statuses should appear
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 (in_progress) to be excluded, got %q", output)
		}
		if strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 (done) to be excluded, got %q", output)
		}
		if strings.Contains(output, "tick-ddd444") {
			t.Errorf("expected tick-ddd444 (cancelled) to be excluded, got %q", output)
		}
	})
}

func TestBlocked_EmptyWhenNoBlockedTasks(t *testing.T) {
	t.Run("it returns empty when no blocked tasks", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			// All tasks are ready (open, no blockers, no children)
			{ID: "tick-aaa111", Title: "Ready task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if !strings.Contains(output, "tasks[0]") {
			t.Errorf("expected empty list indicator 'tasks[0]', got %q", output)
		}
	})
}

func TestBlocked_OrderByPriorityThenCreated(t *testing.T) {
	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			// Shared blocker
			{ID: "tick-blocker", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			// Blocked tasks with different priorities and creation times
			{ID: "tick-low111", Title: "Low priority blocked", Status: task.StatusOpen, Priority: 3, Created: now, Updated: now, BlockedBy: []string{"tick-blocker"}},
			{ID: "tick-hi2222", Title: "High priority newer blocked", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-blocker"}},
			{ID: "tick-hi1111", Title: "High priority older blocked", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now, BlockedBy: []string{"tick-blocker"}},
			{ID: "tick-med111", Title: "Med priority blocked", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now, BlockedBy: []string{"tick-blocker"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 4 blocked task rows (tick-blocker is not blocked)
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

func TestBlocked_AlignedColumnsOutput(t *testing.T) {
	t.Run("it outputs aligned columns via tick blocked", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked by aaa", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Also blocked", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 blocked tasks), got %d:\n%s", len(lines), output)
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
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected output to contain tick-bbb222, got %q", output)
		}
		if !strings.Contains(output, "open") {
			t.Errorf("expected output to contain 'open', got %q", output)
		}
		if !strings.Contains(output, "Blocked by aaa") {
			t.Errorf("expected output to contain 'Blocked by aaa', got %q", output)
		}
	})
}

func TestBlocked_NoTasksFoundMessage(t *testing.T) {
	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if !strings.Contains(output, "tasks[0]") {
			t.Errorf("expected empty list indicator 'tasks[0]', got %q", output)
		}
	})
}

func TestBlocked_QuietOutputsIDsOnly(t *testing.T) {
	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked first", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Blocked second", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per blocked task), got %d:\n%s", len(lines), output)
		}

		if strings.TrimSpace(lines[0]) != "tick-bbb222" {
			t.Errorf("expected first line to be 'tick-bbb222', got %q", lines[0])
		}
		if strings.TrimSpace(lines[1]) != "tick-ccc333" {
			t.Errorf("expected second line to be 'tick-ccc333', got %q", lines[1])
		}
	})
}

func TestBlocked_CancelUnblocksSingleDependent(t *testing.T) {
	t.Run("cancel unblocks single dependent - moves to ready", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Dependent task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)

		// Step 1: Verify task is blocked
		var stdout1, stderr1 bytes.Buffer
		app1 := &App{Stdout: &stdout1, Stderr: &stderr1, Dir: dir}
		code := app1.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("blocked check: expected exit code 0, got %d; stderr: %s", code, stderr1.String())
		}
		if !strings.Contains(stdout1.String(), "tick-bbb222") {
			t.Fatalf("expected tick-bbb222 in blocked output before cancel, got %q", stdout1.String())
		}

		// Step 2: Cancel the blocker
		var stdout2, stderr2 bytes.Buffer
		app2 := &App{Stdout: &stdout2, Stderr: &stderr2, Dir: dir}
		code = app2.Run([]string{"tick", "cancel", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("cancel: expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}

		// Step 3: Verify dependent is now ready (not blocked)
		var stdout3, stderr3 bytes.Buffer
		app3 := &App{Stdout: &stdout3, Stderr: &stderr3, Dir: dir}
		code = app3.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("ready check: expected exit code 0, got %d; stderr: %s", code, stderr3.String())
		}
		if !strings.Contains(stdout3.String(), "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in ready output after cancel, got %q", stdout3.String())
		}

		// Step 4: Verify dependent is no longer blocked
		var stdout4, stderr4 bytes.Buffer
		app4 := &App{Stdout: &stdout4, Stderr: &stderr4, Dir: dir}
		code = app4.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("blocked recheck: expected exit code 0, got %d; stderr: %s", code, stderr4.String())
		}
		if strings.Contains(stdout4.String(), "tick-bbb222") {
			t.Errorf("expected tick-bbb222 to NOT be in blocked output after cancel, got %q", stdout4.String())
		}
	})
}

func TestBlocked_CancelUnblocksMultipleDependents(t *testing.T) {
	t.Run("cancel unblocks multiple dependents", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Dependent 1", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Dependent 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), BlockedBy: []string{"tick-aaa111"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)

		// Step 1: Verify both are blocked
		var stdout1, stderr1 bytes.Buffer
		app1 := &App{Stdout: &stdout1, Stderr: &stderr1, Dir: dir}
		code := app1.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("blocked check: expected exit code 0, got %d; stderr: %s", code, stderr1.String())
		}
		if !strings.Contains(stdout1.String(), "tick-bbb222") {
			t.Fatalf("expected tick-bbb222 in blocked output before cancel")
		}
		if !strings.Contains(stdout1.String(), "tick-ccc333") {
			t.Fatalf("expected tick-ccc333 in blocked output before cancel")
		}

		// Step 2: Cancel the blocker
		var stdout2, stderr2 bytes.Buffer
		app2 := &App{Stdout: &stdout2, Stderr: &stderr2, Dir: dir}
		code = app2.Run([]string{"tick", "cancel", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("cancel: expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}

		// Step 3: Verify both are now ready
		var stdout3, stderr3 bytes.Buffer
		app3 := &App{Stdout: &stdout3, Stderr: &stderr3, Dir: dir}
		code = app3.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("ready check: expected exit code 0, got %d; stderr: %s", code, stderr3.String())
		}
		if !strings.Contains(stdout3.String(), "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in ready output after cancel, got %q", stdout3.String())
		}
		if !strings.Contains(stdout3.String(), "tick-ccc333") {
			t.Errorf("expected tick-ccc333 in ready output after cancel, got %q", stdout3.String())
		}
	})
}

func TestBlocked_CancelDoesNotUnblockDependentStillBlockedByAnother(t *testing.T) {
	t.Run("cancel does not unblock dependent still blocked by another", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "First blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second blocker", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-ccc333", Title: "Double blocked", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), BlockedBy: []string{"tick-aaa111", "tick-bbb222"}},
		}
		dir := setupInitializedDirWithTasks(t, tasks)

		// Step 1: Verify task is blocked
		var stdout1, stderr1 bytes.Buffer
		app1 := &App{Stdout: &stdout1, Stderr: &stderr1, Dir: dir}
		code := app1.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("blocked check: expected exit code 0, got %d; stderr: %s", code, stderr1.String())
		}
		if !strings.Contains(stdout1.String(), "tick-ccc333") {
			t.Fatalf("expected tick-ccc333 in blocked output before cancel")
		}

		// Step 2: Cancel only the first blocker
		var stdout2, stderr2 bytes.Buffer
		app2 := &App{Stdout: &stdout2, Stderr: &stderr2, Dir: dir}
		code = app2.Run([]string{"tick", "cancel", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("cancel: expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}

		// Step 3: Verify task is still blocked (second blocker still open)
		var stdout3, stderr3 bytes.Buffer
		app3 := &App{Stdout: &stdout3, Stderr: &stderr3, Dir: dir}
		code = app3.Run([]string{"tick", "blocked"})
		if code != 0 {
			t.Fatalf("blocked recheck: expected exit code 0, got %d; stderr: %s", code, stderr3.String())
		}
		if !strings.Contains(stdout3.String(), "tick-ccc333") {
			t.Errorf("expected tick-ccc333 to still be blocked (second blocker open), got %q", stdout3.String())
		}

		// Step 4: Verify task is NOT in ready
		var stdout4, stderr4 bytes.Buffer
		app4 := &App{Stdout: &stdout4, Stderr: &stderr4, Dir: dir}
		code = app4.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("ready check: expected exit code 0, got %d; stderr: %s", code, stderr4.String())
		}
		if strings.Contains(stdout4.String(), "tick-ccc333") {
			t.Errorf("expected tick-ccc333 to NOT be in ready output (still has open blocker), got %q", stdout4.String())
		}
	})
}
