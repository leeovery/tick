package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// TestWorkflow_CreateStartDoneLifecycle exercises the full task lifecycle:
// create -> start -> done, verifying task state at each step and that
// ready/blocked/stats commands return consistent results.
func TestWorkflow_CreateStartDoneLifecycle(t *testing.T) {
	t.Run("it tracks a task through its full lifecycle", func(t *testing.T) {
		dir := setupInitializedDir(t)

		// Step 1: Create a task with --quiet to capture the ID
		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "create", "Lifecycle task", "--priority", "1"})
		if code != 0 {
			t.Fatalf("create failed: exit %d; stderr: %s", code, stderr.String())
		}
		taskID := strings.TrimSpace(stdout.String())

		// Step 2: Verify it appears in ready list
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), taskID) {
			t.Errorf("expected task %s in ready list, got %q", taskID, stdout.String())
		}

		// Step 3: Start the task
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "start", taskID})
		if code != 0 {
			t.Fatalf("start failed: exit %d; stderr: %s", code, stderr.String())
		}
		output := strings.TrimSpace(stdout.String())
		if !strings.Contains(output, "open") || !strings.Contains(output, "in_progress") {
			t.Errorf("expected transition output, got %q", output)
		}

		// Step 4: Verify it no longer appears in ready list (in_progress is not ready)
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		if strings.Contains(stdout.String(), taskID) {
			t.Errorf("expected task %s NOT in ready list after start, got %q", taskID, stdout.String())
		}

		// Step 5: Complete the task
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "done", taskID})
		if code != 0 {
			t.Fatalf("done failed: exit %d; stderr: %s", code, stderr.String())
		}

		// Step 6: Verify via show that status is done and closed timestamp is set
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "show", taskID})
		if code != 0 {
			t.Fatalf("show failed: exit %d; stderr: %s", code, stderr.String())
		}
		showOutput := stdout.String()
		if !strings.Contains(showOutput, "done") {
			t.Errorf("expected status 'done' in show output, got %q", showOutput)
		}
		if !strings.Contains(showOutput, "closed") {
			t.Errorf("expected 'closed' timestamp in show output, got %q", showOutput)
		}

		// Step 7: Verify stats reflect the completed task
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "stats"})
		if code != 0 {
			t.Fatalf("stats failed: exit %d; stderr: %s", code, stderr.String())
		}
		statsOutput := stdout.String()
		if !strings.Contains(statsOutput, "1") {
			t.Errorf("expected total count in stats, got %q", statsOutput)
		}
	})
}

// TestWorkflow_CancelUnblocksDependents exercises the cancel-unblocks-dependents flow:
// create A, create B blocked-by A, verify B is blocked, cancel A, verify B is now ready.
func TestWorkflow_CancelUnblocksDependents(t *testing.T) {
	t.Run("it unblocks dependents when a blocker is cancelled", func(t *testing.T) {
		dir := setupInitializedDir(t)

		// Create task A
		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "create", "Blocker task"})
		if code != 0 {
			t.Fatalf("create A failed: exit %d; stderr: %s", code, stderr.String())
		}
		taskA := strings.TrimSpace(stdout.String())

		// Create task B blocked by A
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "create", "Dependent task", "--blocked-by", taskA})
		if code != 0 {
			t.Fatalf("create B failed: exit %d; stderr: %s", code, stderr.String())
		}
		taskB := strings.TrimSpace(stdout.String())

		// Verify B is blocked
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "blocked"})
		if code != 0 {
			t.Fatalf("blocked failed: exit %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), taskB) {
			t.Errorf("expected task %s in blocked list, got %q", taskB, stdout.String())
		}

		// Verify B is NOT in ready
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		if strings.Contains(stdout.String(), taskB) {
			t.Errorf("expected task %s NOT in ready list, got %q", taskB, stdout.String())
		}

		// Cancel task A
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "cancel", taskA})
		if code != 0 {
			t.Fatalf("cancel A failed: exit %d; stderr: %s", code, stderr.String())
		}

		// Verify B is now ready (unblocked because A is cancelled)
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), taskB) {
			t.Errorf("expected task %s in ready list after cancel, got %q", taskB, stdout.String())
		}

		// Verify B is no longer blocked
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "blocked"})
		if code != 0 {
			t.Fatalf("blocked failed: exit %d; stderr: %s", code, stderr.String())
		}
		if strings.Contains(stdout.String(), taskB) {
			t.Errorf("expected task %s NOT in blocked list after cancel, got %q", taskB, stdout.String())
		}
	})
}

// TestWorkflow_HierarchyAndReadyRule exercises parent/child hierarchy with the leaf-only ready rule:
// create parent, create child with parent, verify parent is NOT ready (has open child),
// complete child, verify parent becomes ready.
func TestWorkflow_HierarchyAndReadyRule(t *testing.T) {
	t.Run("it applies leaf-only ready rule to parent/child hierarchy", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-parent1", Title: "Parent task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-child01", Title: "Child task", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent1", Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, existing)

		// Parent should NOT be in ready (has open child)
		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		readyIDs := stdout.String()
		if strings.Contains(readyIDs, "tick-parent1") {
			t.Errorf("expected parent NOT in ready (has open child), got %q", readyIDs)
		}
		// Child should be in ready
		if !strings.Contains(readyIDs, "tick-child01") {
			t.Errorf("expected child in ready, got %q", readyIDs)
		}

		// Complete the child
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "done", "tick-child01"})
		if code != 0 {
			t.Fatalf("done child failed: exit %d; stderr: %s", code, stderr.String())
		}

		// Parent should now be in ready (no more open children)
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		readyIDs = stdout.String()
		if !strings.Contains(readyIDs, "tick-parent1") {
			t.Errorf("expected parent in ready after child done, got %q", readyIDs)
		}
	})
}

// TestWorkflow_DepAddRemoveWithReadyCheck exercises dep add/rm and verifies ready query consistency:
// create A and B, verify both ready, dep add B blocked-by A, verify B blocked,
// dep rm, verify B ready again.
func TestWorkflow_DepAddRemoveWithReadyCheck(t *testing.T) {
	t.Run("it updates ready/blocked state when deps are added and removed", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, existing)

		// Both should be ready initially
		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "tick-aaa111") || !strings.Contains(stdout.String(), "tick-bbb222") {
			t.Errorf("expected both tasks in ready, got %q", stdout.String())
		}

		// Add dependency: B blocked by A
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "dep", "add", "tick-bbb222", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("dep add failed: exit %d; stderr: %s", code, stderr.String())
		}

		// B should now be blocked
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "blocked"})
		if code != 0 {
			t.Fatalf("blocked failed: exit %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "tick-bbb222") {
			t.Errorf("expected B in blocked after dep add, got %q", stdout.String())
		}

		// Remove dependency
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "dep", "rm", "tick-bbb222", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("dep rm failed: exit %d; stderr: %s", code, stderr.String())
		}

		// B should be ready again
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "--quiet", "ready"})
		if code != 0 {
			t.Fatalf("ready failed: exit %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "tick-bbb222") {
			t.Errorf("expected B in ready after dep rm, got %q", stdout.String())
		}
	})
}

// TestWorkflow_RebuildPreservesData exercises cache rebuild and data integrity:
// create tasks, rebuild cache, verify list still works correctly.
func TestWorkflow_RebuildPreservesData(t *testing.T) {
	t.Run("it preserves data after cache rebuild", func(t *testing.T) {
		dir := setupInitializedDir(t)

		// Create a task
		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "create", "Rebuild test"})
		if code != 0 {
			t.Fatalf("create failed: exit %d; stderr: %s", code, stderr.String())
		}
		taskID := strings.TrimSpace(stdout.String())

		// Rebuild cache
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "rebuild"})
		if code != 0 {
			t.Fatalf("rebuild failed: exit %d; stderr: %s", code, stderr.String())
		}

		// Verify task still queryable
		stdout.Reset()
		stderr.Reset()
		app = &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code = app.Run([]string{"tick", "show", taskID})
		if code != 0 {
			t.Fatalf("show after rebuild failed: exit %d; stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Rebuild test") {
			t.Errorf("expected task title in show after rebuild, got %q", stdout.String())
		}
	})
}
