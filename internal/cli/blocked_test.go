package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestBlockedQuery(t *testing.T) {
	t.Run("it returns open task blocked by open/in_progress dep", func(t *testing.T) {
		openBlocker := task.NewTask("tick-aaaaaa", "Open blocker")
		blockedByOpen := task.NewTask("tick-bbbbbb", "Blocked by open")
		blockedByOpen.BlockedBy = []string{"tick-aaaaaa"}

		ipBlocker := task.NewTask("tick-cccccc", "IP blocker")
		ipBlocker.Status = task.StatusInProgress
		blockedByIP := task.NewTask("tick-dddddd", "Blocked by IP")
		blockedByIP.BlockedBy = []string{"tick-cccccc"}

		dir := initTickProjectWithTasks(t, []task.Task{openBlocker, blockedByOpen, ipBlocker, blockedByIP})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected task blocked by open dep to appear, got %q", output)
		}
		if !strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected task blocked by in_progress dep to appear, got %q", output)
		}
		// The blockers themselves are not blocked (they are ready or in_progress)
		if strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected open unblocked task to NOT appear in blocked, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected in_progress task to NOT appear in blocked, got %q", output)
		}
	})

	t.Run("it returns parent with open/in_progress children", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent with open child")
		child := task.NewTask("tick-bbbbbb", "Open child")
		child.Parent = "tick-aaaaaa"

		parent2 := task.NewTask("tick-cccccc", "Parent with IP child")
		childIP := task.NewTask("tick-dddddd", "IP child")
		childIP.Parent = "tick-cccccc"
		childIP.Status = task.StatusInProgress

		dir := initTickProjectWithTasks(t, []task.Task{parent, child, parent2, childIP})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected parent with open child to be blocked, got %q", output)
		}
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected parent with in_progress child to be blocked, got %q", output)
		}
		// Children themselves are not blocked (they are ready or in_progress)
		if strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected open child (ready leaf) to NOT appear in blocked, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected in_progress child to NOT appear in blocked, got %q", output)
		}
	})

	t.Run("it excludes task when all blockers done/cancelled", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		doneBlocker := task.NewTask("tick-aaaaaa", "Done blocker")
		doneBlocker.Status = task.StatusDone
		doneBlocker.Closed = &now

		cancelledBlocker := task.NewTask("tick-bbbbbb", "Cancelled blocker")
		cancelledBlocker.Status = task.StatusCancelled
		cancelledBlocker.Closed = &now

		unblocked := task.NewTask("tick-cccccc", "Unblocked task")
		unblocked.BlockedBy = []string{"tick-aaaaaa", "tick-bbbbbb"}

		dir := initTickProjectWithTasks(t, []task.Task{doneBlocker, cancelledBlocker, unblocked})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected task with all done/cancelled blockers to NOT be blocked, got %q", output)
		}
	})

	t.Run("it excludes in_progress/done/cancelled from output", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		// in_progress task with an open blocker - but since it's in_progress, not open,
		// it should NOT appear in blocked output
		blocker := task.NewTask("tick-aaaaaa", "Open blocker")
		ipTask := task.NewTask("tick-bbbbbb", "In progress blocked")
		ipTask.Status = task.StatusInProgress
		ipTask.BlockedBy = []string{"tick-aaaaaa"}

		doneTask := task.NewTask("tick-cccccc", "Done")
		doneTask.Status = task.StatusDone
		doneTask.Closed = &now

		cancelledTask := task.NewTask("tick-dddddd", "Cancelled")
		cancelledTask.Status = task.StatusCancelled
		cancelledTask.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{blocker, ipTask, doneTask, cancelledTask})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected in_progress task to NOT appear in blocked output, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected done task to NOT appear in blocked output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected cancelled task to NOT appear in blocked output, got %q", output)
		}
	})

	t.Run("it returns empty when no blocked tasks", func(t *testing.T) {
		// Only ready tasks - nothing blocked
		t1 := task.NewTask("tick-aaaaaa", "Ready task")

		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})

	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		// All three tasks are open and blocked by the same open blocker
		blocker := task.NewTask("tick-zzzzzz", "Open blocker")

		t1 := task.NewTask("tick-aaaaaa", "Low priority early")
		t1.Priority = 3
		t1.Created = now.Add(-2 * time.Hour)
		t1.Updated = now.Add(-2 * time.Hour)
		t1.BlockedBy = []string{"tick-zzzzzz"}

		t2 := task.NewTask("tick-bbbbbb", "High priority")
		t2.Priority = 1
		t2.Created = now
		t2.Updated = now
		t2.BlockedBy = []string{"tick-zzzzzz"}

		t3 := task.NewTask("tick-cccccc", "Low priority late")
		t3.Priority = 3
		t3.Created = now.Add(-1 * time.Hour)
		t3.Updated = now.Add(-1 * time.Hour)
		t3.BlockedBy = []string{"tick-zzzzzz"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines (header + 3 tasks), got %d: %q", len(lines), stdout.String())
		}

		// Priority 1 first, then priority 3 ordered by created (earlier first)
		if !strings.Contains(lines[1], "tick-bbbbbb") {
			t.Errorf("expected first task to be tick-bbbbbb (priority 1), got %q", lines[1])
		}
		if !strings.Contains(lines[2], "tick-aaaaaa") {
			t.Errorf("expected second task to be tick-aaaaaa (priority 3, earlier), got %q", lines[2])
		}
		if !strings.Contains(lines[3], "tick-cccccc") {
			t.Errorf("expected third task to be tick-cccccc (priority 3, later), got %q", lines[3])
		}
	})
}

func TestBlockedCommand(t *testing.T) {
	t.Run("it outputs aligned columns via tick blocked", func(t *testing.T) {
		blocker := task.NewTask("tick-zzzzzz", "Open blocker")
		t1 := task.NewTask("tick-aaaaaa", "Blocked task one")
		t1.Priority = 1
		t1.BlockedBy = []string{"tick-zzzzzz"}
		t2 := task.NewTask("tick-bbbbbb", "Blocked task two")
		t2.Priority = 2
		t2.BlockedBy = []string{"tick-zzzzzz"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), output)
		}

		// Check header
		if !strings.HasPrefix(lines[0], "ID") {
			t.Errorf("expected header to start with 'ID', got %q", lines[0])
		}
		if !strings.Contains(lines[0], "STATUS") {
			t.Errorf("expected header to contain 'STATUS', got %q", lines[0])
		}
		if !strings.Contains(lines[0], "PRI") {
			t.Errorf("expected header to contain 'PRI', got %q", lines[0])
		}
		if !strings.Contains(lines[0], "TITLE") {
			t.Errorf("expected header to contain 'TITLE', got %q", lines[0])
		}

		// Check data rows contain expected IDs
		if !strings.Contains(lines[1], "tick-aaaaaa") {
			t.Errorf("expected first row to contain tick-aaaaaa, got %q", lines[1])
		}
		if !strings.Contains(lines[1], "open") {
			t.Errorf("expected first row to contain 'open', got %q", lines[1])
		}
	})

	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})

	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		blocker := task.NewTask("tick-zzzzzz", "Open blocker")
		t1 := task.NewTask("tick-aaaaaa", "Blocked one")
		t1.BlockedBy = []string{"tick-zzzzzz"}
		t2 := task.NewTask("tick-bbbbbb", "Blocked two")
		t2.BlockedBy = []string{"tick-zzzzzz"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per blocked task), got %d: %q", len(lines), output)
		}

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "tick-") {
				t.Errorf("expected line to be a task ID, got %q", line)
			}
			if strings.Contains(line, " ") {
				t.Errorf("expected only ID (no spaces), got %q", line)
			}
		}
	})
}

func TestCancelUnblocksDependents(t *testing.T) {
	t.Run("cancel unblocks single dependent - moves to ready", func(t *testing.T) {
		blocker := task.NewTask("tick-aaaaaa", "Blocker to cancel")
		dependent := task.NewTask("tick-bbbbbb", "Dependent task")
		dependent.BlockedBy = []string{"tick-aaaaaa"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, dependent})

		// Verify dependent is blocked before cancel
		var stdout1, stderr1 bytes.Buffer
		code := Run([]string{"tick", "blocked"}, dir, &stdout1, &stderr1, false)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr1.String())
		}
		if !strings.Contains(stdout1.String(), "tick-bbbbbb") {
			t.Fatalf("expected dependent to be blocked initially, got %q", stdout1.String())
		}

		// Cancel the blocker
		var stdout2, stderr2 bytes.Buffer
		code = Run([]string{"tick", "cancel", "tick-aaaaaa"}, dir, &stdout2, &stderr2, false)
		if code != 0 {
			t.Fatalf("cancel failed: exit code %d; stderr: %s", code, stderr2.String())
		}

		// Verify dependent is now ready
		var stdout3, stderr3 bytes.Buffer
		code = Run([]string{"tick", "ready"}, dir, &stdout3, &stderr3, false)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr3.String())
		}
		if !strings.Contains(stdout3.String(), "tick-bbbbbb") {
			t.Errorf("expected dependent to be ready after cancel, got %q", stdout3.String())
		}

		// Verify dependent is no longer blocked
		var stdout4, stderr4 bytes.Buffer
		code = Run([]string{"tick", "blocked"}, dir, &stdout4, &stderr4, false)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr4.String())
		}
		if strings.Contains(stdout4.String(), "tick-bbbbbb") {
			t.Errorf("expected dependent to NOT be blocked after cancel, got %q", stdout4.String())
		}
	})

	t.Run("cancel unblocks multiple dependents", func(t *testing.T) {
		blocker := task.NewTask("tick-aaaaaa", "Shared blocker")
		dep1 := task.NewTask("tick-bbbbbb", "Dependent one")
		dep1.BlockedBy = []string{"tick-aaaaaa"}
		dep2 := task.NewTask("tick-cccccc", "Dependent two")
		dep2.BlockedBy = []string{"tick-aaaaaa"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, dep1, dep2})

		// Cancel the shared blocker
		var stdout1, stderr1 bytes.Buffer
		code := Run([]string{"tick", "cancel", "tick-aaaaaa"}, dir, &stdout1, &stderr1, false)
		if code != 0 {
			t.Fatalf("cancel failed: exit code %d; stderr: %s", code, stderr1.String())
		}

		// Verify both dependents are now ready
		var stdout2, stderr2 bytes.Buffer
		code = Run([]string{"tick", "ready"}, dir, &stdout2, &stderr2, false)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}
		output := stdout2.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected dep1 to be ready after cancel, got %q", output)
		}
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected dep2 to be ready after cancel, got %q", output)
		}
	})

	t.Run("cancel does not unblock dependent still blocked by another", func(t *testing.T) {
		blocker1 := task.NewTask("tick-aaaaaa", "Blocker one")
		blocker2 := task.NewTask("tick-bbbbbb", "Blocker two")
		dependent := task.NewTask("tick-cccccc", "Dependent on both")
		dependent.BlockedBy = []string{"tick-aaaaaa", "tick-bbbbbb"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker1, blocker2, dependent})

		// Cancel only blocker1
		var stdout1, stderr1 bytes.Buffer
		code := Run([]string{"tick", "cancel", "tick-aaaaaa"}, dir, &stdout1, &stderr1, false)
		if code != 0 {
			t.Fatalf("cancel failed: exit code %d; stderr: %s", code, stderr1.String())
		}

		// Dependent should still be blocked (blocker2 is still open)
		var stdout2, stderr2 bytes.Buffer
		code = Run([]string{"tick", "blocked"}, dir, &stdout2, &stderr2, false)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}
		if !strings.Contains(stdout2.String(), "tick-cccccc") {
			t.Errorf("expected dependent still blocked by open blocker2, got %q", stdout2.String())
		}

		// Dependent should NOT be ready
		var stdout3, stderr3 bytes.Buffer
		code = Run([]string{"tick", "ready"}, dir, &stdout3, &stderr3, false)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr3.String())
		}
		if strings.Contains(stdout3.String(), "tick-cccccc") {
			t.Errorf("expected dependent to NOT be ready when still blocked, got %q", stdout3.String())
		}
	})
}
