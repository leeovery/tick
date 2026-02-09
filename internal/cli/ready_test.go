package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestReadyQuery(t *testing.T) {
	t.Run("it returns open task with no blockers and no children", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Simple open task")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected output to contain tick-aaaaaa, got %q", output)
		}
		if !strings.Contains(output, "Simple open task") {
			t.Errorf("expected output to contain task title, got %q", output)
		}
	})

	t.Run("it excludes task with open/in_progress blocker", func(t *testing.T) {
		blocker := task.NewTask("tick-aaaaaa", "Open blocker")
		blocked := task.NewTask("tick-bbbbbb", "Blocked task")
		blocked.BlockedBy = []string{"tick-aaaaaa"}

		blockerIP := task.NewTask("tick-cccccc", "IP blocker")
		blockerIP.Status = task.StatusInProgress
		blockedByIP := task.NewTask("tick-dddddd", "Blocked by IP")
		blockedByIP.BlockedBy = []string{"tick-cccccc"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, blocked, blockerIP, blockedByIP})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// blocker (tick-aaaaaa) is open with no blockers/children, so it IS ready
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected open unblocked task tick-aaaaaa to be ready, got %q", output)
		}
		// blocked (tick-bbbbbb) has open blocker, so NOT ready
		if strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected blocked task tick-bbbbbb to NOT be ready, got %q", output)
		}
		// blockerIP is in_progress, not open, so NOT ready
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected in_progress task tick-cccccc to NOT be ready, got %q", output)
		}
		// blockedByIP has in_progress blocker, so NOT ready
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected task blocked by IP to NOT be ready, got %q", output)
		}
	})

	t.Run("it includes task when all blockers done/cancelled", func(t *testing.T) {
		doneBlocker := task.NewTask("tick-aaaaaa", "Done blocker")
		doneBlocker.Status = task.StatusDone
		now := time.Now().UTC().Truncate(time.Second)
		doneBlocker.Closed = &now

		cancelledBlocker := task.NewTask("tick-bbbbbb", "Cancelled blocker")
		cancelledBlocker.Status = task.StatusCancelled
		cancelledBlocker.Closed = &now

		unblocked := task.NewTask("tick-cccccc", "Unblocked task")
		unblocked.BlockedBy = []string{"tick-aaaaaa", "tick-bbbbbb"}

		dir := initTickProjectWithTasks(t, []task.Task{doneBlocker, cancelledBlocker, unblocked})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected task with all done/cancelled blockers to be ready, got %q", output)
		}
	})

	t.Run("it excludes parent with open/in_progress children", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")
		child := task.NewTask("tick-bbbbbb", "Open child")
		child.Parent = "tick-aaaaaa"

		parent2 := task.NewTask("tick-cccccc", "Parent 2")
		childIP := task.NewTask("tick-dddddd", "IP child")
		childIP.Parent = "tick-cccccc"
		childIP.Status = task.StatusInProgress

		dir := initTickProjectWithTasks(t, []task.Task{parent, child, parent2, childIP})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Parent with open child: NOT ready
		if strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected parent with open child to NOT be ready, got %q", output)
		}
		// Open child is ready (leaf, unblocked)
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected open child tick-bbbbbb to be ready, got %q", output)
		}
		// Parent with in_progress child: NOT ready
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected parent with in_progress child to NOT be ready, got %q", output)
		}
	})

	t.Run("it includes parent when all children closed", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")

		now := time.Now().UTC().Truncate(time.Second)
		child1 := task.NewTask("tick-bbbbbb", "Done child")
		child1.Parent = "tick-aaaaaa"
		child1.Status = task.StatusDone
		child1.Closed = &now

		child2 := task.NewTask("tick-cccccc", "Cancelled child")
		child2.Parent = "tick-aaaaaa"
		child2.Status = task.StatusCancelled
		child2.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{parent, child1, child2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected parent with all closed children to be ready, got %q", output)
		}
	})

	t.Run("it excludes in_progress/done/cancelled tasks", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		ipTask := task.NewTask("tick-aaaaaa", "In progress")
		ipTask.Status = task.StatusInProgress

		doneTask := task.NewTask("tick-bbbbbb", "Done")
		doneTask.Status = task.StatusDone
		doneTask.Closed = &now

		cancelledTask := task.NewTask("tick-cccccc", "Cancelled")
		cancelledTask.Status = task.StatusCancelled
		cancelledTask.Closed = &now

		openTask := task.NewTask("tick-dddddd", "Open and ready")

		dir := initTickProjectWithTasks(t, []task.Task{ipTask, doneTask, cancelledTask, openTask})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected in_progress task to NOT be ready, got %q", output)
		}
		if strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected done task to NOT be ready, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected cancelled task to NOT be ready, got %q", output)
		}
		if !strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected open task to be ready, got %q", output)
		}
	})

	t.Run("it handles deep nesting - only deepest incomplete ready", func(t *testing.T) {
		// grandparent -> parent -> child (leaf)
		// Only the leaf should be ready
		grandparent := task.NewTask("tick-aaaaaa", "Grandparent")
		parent := task.NewTask("tick-bbbbbb", "Parent")
		parent.Parent = "tick-aaaaaa"
		child := task.NewTask("tick-cccccc", "Leaf child")
		child.Parent = "tick-bbbbbb"

		dir := initTickProjectWithTasks(t, []task.Task{grandparent, parent, child})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Grandparent has open child (parent), so NOT ready
		if strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected grandparent to NOT be ready, got %q", output)
		}
		// Parent has open child (child), so NOT ready
		if strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected parent to NOT be ready, got %q", output)
		}
		// Child is a leaf, open, no blockers: READY
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected leaf child to be ready, got %q", output)
		}
	})

	t.Run("it returns empty list when no tasks ready", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		doneTask := task.NewTask("tick-aaaaaa", "Done task")
		doneTask.Status = task.StatusDone
		doneTask.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{doneTask})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

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

		t1 := task.NewTask("tick-aaaaaa", "Low priority early")
		t1.Priority = 3
		t1.Created = now.Add(-2 * time.Hour)
		t1.Updated = now.Add(-2 * time.Hour)

		t2 := task.NewTask("tick-bbbbbb", "High priority")
		t2.Priority = 1
		t2.Created = now
		t2.Updated = now

		t3 := task.NewTask("tick-cccccc", "Low priority late")
		t3.Priority = 3
		t3.Created = now.Add(-1 * time.Hour)
		t3.Updated = now.Add(-1 * time.Hour)

		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

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

func TestReadyCommand(t *testing.T) {
	t.Run("it outputs aligned columns via tick ready", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Ready task one")
		t1.Priority = 1
		t2 := task.NewTask("tick-bbbbbb", "Ready task two")
		t2.Priority = 2
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

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

		// Check data rows
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
		code := Run([]string{"tick", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})

	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task one")
		t2 := task.NewTask("tick-bbbbbb", "Task two")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per task), got %d: %q", len(lines), output)
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
