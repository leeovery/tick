package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestParentScope(t *testing.T) {
	t.Run("it returns all descendants of parent (direct children)", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")
		child1 := task.NewTask("tick-bbbbbb", "Child one")
		child1.Parent = "tick-aaaaaa"
		child2 := task.NewTask("tick-cccccc", "Child two")
		child2.Parent = "tick-aaaaaa"
		unrelated := task.NewTask("tick-dddddd", "Unrelated task")

		dir := initTickProjectWithTasks(t, []task.Task{parent, child1, child2, unrelated})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected child tick-bbbbbb in output, got %q", output)
		}
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected child tick-cccccc in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected unrelated tick-dddddd NOT in output, got %q", output)
		}
	})

	t.Run("it returns all descendants recursively (3+ levels deep)", func(t *testing.T) {
		grandparent := task.NewTask("tick-aaaaaa", "Grandparent")
		parent := task.NewTask("tick-bbbbbb", "Parent")
		parent.Parent = "tick-aaaaaa"
		child := task.NewTask("tick-cccccc", "Child")
		child.Parent = "tick-bbbbbb"
		grandchild := task.NewTask("tick-dddddd", "Grandchild")
		grandchild.Parent = "tick-cccccc"

		dir := initTickProjectWithTasks(t, []task.Task{grandparent, parent, child, grandchild})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected parent tick-bbbbbb in output, got %q", output)
		}
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected child tick-cccccc in output, got %q", output)
		}
		if !strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected grandchild tick-dddddd in output, got %q", output)
		}
	})

	t.Run("it excludes parent task itself from results", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")
		child := task.NewTask("tick-bbbbbb", "Child task")
		child.Parent = "tick-aaaaaa"

		dir := initTickProjectWithTasks(t, []task.Task{parent, child})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// The parent itself should not appear in results
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines[1:] { // skip header
			if strings.Contains(line, "tick-aaaaaa") {
				t.Errorf("expected parent tick-aaaaaa NOT in results, got %q", line)
			}
		}
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected child tick-bbbbbb in output, got %q", output)
		}
	})

	t.Run("it returns empty result when parent has no descendants", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Lonely parent")

		dir := initTickProjectWithTasks(t, []task.Task{parent})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})

	t.Run("it errors with Task not found for non-existent parent ID", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "tick-nonexist") {
			t.Errorf("expected error to mention task ID, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "not found") {
			t.Errorf("expected error to contain 'not found', got %q", errMsg)
		}
	})

	t.Run("it returns only ready tasks within parent scope with tick ready --parent", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent")
		readyChild := task.NewTask("tick-bbbbbb", "Ready child")
		readyChild.Parent = "tick-aaaaaa"
		blocker := task.NewTask("tick-cccccc", "Blocker")
		blockedChild := task.NewTask("tick-dddddd", "Blocked child")
		blockedChild.Parent = "tick-aaaaaa"
		blockedChild.BlockedBy = []string{"tick-cccccc"}
		outsideReady := task.NewTask("tick-eeeeee", "Outside ready")

		dir := initTickProjectWithTasks(t, []task.Task{parent, readyChild, blocker, blockedChild, outsideReady})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "ready", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected ready child tick-bbbbbb in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected blocked child tick-dddddd NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-eeeeee") {
			t.Errorf("expected outside task tick-eeeeee NOT in output, got %q", output)
		}
	})

	t.Run("it returns only blocked tasks within parent scope with tick blocked --parent", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent")
		blocker := task.NewTask("tick-bbbbbb", "Blocker")
		blockedChild := task.NewTask("tick-cccccc", "Blocked child")
		blockedChild.Parent = "tick-aaaaaa"
		blockedChild.BlockedBy = []string{"tick-bbbbbb"}
		readyChild := task.NewTask("tick-dddddd", "Ready child")
		readyChild.Parent = "tick-aaaaaa"
		outsideBlocked := task.NewTask("tick-eeeeee", "Outside blocked")
		outsideBlocked.BlockedBy = []string{"tick-bbbbbb"}

		dir := initTickProjectWithTasks(t, []task.Task{parent, blocker, blockedChild, readyChild, outsideBlocked})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "blocked", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected blocked child tick-cccccc in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected ready child tick-dddddd NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-eeeeee") {
			t.Errorf("expected outside blocked tick-eeeeee NOT in output, got %q", output)
		}
	})

	t.Run("it combines --parent with --status filter", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		parent := task.NewTask("tick-aaaaaa", "Parent")
		openChild := task.NewTask("tick-bbbbbb", "Open child")
		openChild.Parent = "tick-aaaaaa"
		doneChild := task.NewTask("tick-cccccc", "Done child")
		doneChild.Parent = "tick-aaaaaa"
		doneChild.Status = task.StatusDone
		doneChild.Closed = &now
		outsideDone := task.NewTask("tick-dddddd", "Outside done")
		outsideDone.Status = task.StatusDone
		outsideDone.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{parent, openChild, doneChild, outsideDone})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa", "--status", "done"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected done child tick-cccccc in output, got %q", output)
		}
		if strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected open child tick-bbbbbb NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected outside done tick-dddddd NOT in output, got %q", output)
		}
	})

	t.Run("it combines --parent with --priority filter", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent")
		p1Child := task.NewTask("tick-bbbbbb", "P1 child")
		p1Child.Parent = "tick-aaaaaa"
		p1Child.Priority = 1
		p3Child := task.NewTask("tick-cccccc", "P3 child")
		p3Child.Parent = "tick-aaaaaa"
		p3Child.Priority = 3
		outsideP1 := task.NewTask("tick-dddddd", "Outside P1")
		outsideP1.Priority = 1

		dir := initTickProjectWithTasks(t, []task.Task{parent, p1Child, p3Child, outsideP1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa", "--priority", "1"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected P1 child tick-bbbbbb in output, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected P3 child tick-cccccc NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected outside P1 tick-dddddd NOT in output, got %q", output)
		}
	})

	t.Run("it combines --parent with --ready and --priority", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent")
		readyP1 := task.NewTask("tick-bbbbbb", "Ready P1 child")
		readyP1.Parent = "tick-aaaaaa"
		readyP1.Priority = 1
		readyP3 := task.NewTask("tick-cccccc", "Ready P3 child")
		readyP3.Parent = "tick-aaaaaa"
		readyP3.Priority = 3
		blocker := task.NewTask("tick-dddddd", "Blocker")
		blockedP1 := task.NewTask("tick-eeeeee", "Blocked P1 child")
		blockedP1.Parent = "tick-aaaaaa"
		blockedP1.Priority = 1
		blockedP1.BlockedBy = []string{"tick-dddddd"}

		dir := initTickProjectWithTasks(t, []task.Task{parent, readyP1, readyP3, blocker, blockedP1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa", "--ready", "--priority", "1"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected ready P1 child in output, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected ready P3 child NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-eeeeee") {
			t.Errorf("expected blocked P1 child NOT in output, got %q", output)
		}
	})

	t.Run("it combines --parent with --blocked and --status", func(t *testing.T) {
		// Note: --blocked already implies status=open, so --status open is redundant
		// but should still compose correctly
		parent := task.NewTask("tick-aaaaaa", "Parent")
		blocker := task.NewTask("tick-bbbbbb", "Blocker")
		blockedChild := task.NewTask("tick-cccccc", "Blocked child")
		blockedChild.Parent = "tick-aaaaaa"
		blockedChild.BlockedBy = []string{"tick-bbbbbb"}
		readyChild := task.NewTask("tick-dddddd", "Ready child")
		readyChild.Parent = "tick-aaaaaa"

		dir := initTickProjectWithTasks(t, []task.Task{parent, blocker, blockedChild, readyChild})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa", "--blocked", "--status", "open"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected blocked child tick-cccccc in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected ready child tick-dddddd NOT in output, got %q", output)
		}
	})

	t.Run("it handles case-insensitive parent ID", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")
		child := task.NewTask("tick-bbbbbb", "Child task")
		child.Parent = "tick-aaaaaa"

		dir := initTickProjectWithTasks(t, []task.Task{parent, child})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "TICK-AAAAAA"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected child tick-bbbbbb in output, got %q", output)
		}
	})

	t.Run("it excludes tasks outside the parent subtree", func(t *testing.T) {
		parent1 := task.NewTask("tick-aaaaaa", "Parent 1")
		child1 := task.NewTask("tick-bbbbbb", "Child of parent 1")
		child1.Parent = "tick-aaaaaa"
		parent2 := task.NewTask("tick-cccccc", "Parent 2")
		child2 := task.NewTask("tick-dddddd", "Child of parent 2")
		child2.Parent = "tick-cccccc"
		rootTask := task.NewTask("tick-eeeeee", "Root task")

		dir := initTickProjectWithTasks(t, []task.Task{parent1, child1, parent2, child2, rootTask})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected child of parent 1 in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected child of parent 2 NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-eeeeee") {
			t.Errorf("expected root task NOT in output, got %q", output)
		}
	})

	t.Run("it outputs IDs only with --quiet within scoped set", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent")
		child1 := task.NewTask("tick-bbbbbb", "Child one")
		child1.Parent = "tick-aaaaaa"
		child2 := task.NewTask("tick-cccccc", "Child two")
		child2.Parent = "tick-aaaaaa"
		outside := task.NewTask("tick-dddddd", "Outside")

		dir := initTickProjectWithTasks(t, []task.Task{parent, child1, child2, outside})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "list", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per child), got %d: %q", len(lines), output)
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

		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected outside task tick-dddddd NOT in quiet output, got %q", output)
		}
	})

	t.Run("it returns No tasks found when descendants exist but none match filters", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent")
		child := task.NewTask("tick-bbbbbb", "Open child")
		child.Parent = "tick-aaaaaa"
		child.Priority = 2

		dir := initTickProjectWithTasks(t, []task.Task{parent, child})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa", "--status", "done"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})
}
