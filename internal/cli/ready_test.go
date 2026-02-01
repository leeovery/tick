package cli

import (
	"strings"
	"testing"
)

func TestReadyCommand(t *testing.T) {
	t.Run("returns open task with no blockers and no children", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Simple task")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "ready")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, id) {
			t.Errorf("expected task %s in ready output, got %q", id, stdout)
		}
	})

	t.Run("excludes task with open blocker", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocked", "--blocked-by", id1)
		id2 := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		if strings.Contains(stdout, id2) {
			t.Errorf("blocked task %s should not be in ready output", id2)
		}
		// Blocker should be ready
		if !strings.Contains(stdout, id1) {
			t.Errorf("blocker %s should be in ready output", id1)
		}
	})

	t.Run("excludes task with in_progress blocker", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocked", "--blocked-by", id1)
		id2 := extractID(t, out2)
		runCmd(t, dir, "tick", "start", id1)

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		if strings.Contains(stdout, id2) {
			t.Errorf("blocked task should not be ready")
		}
	})

	t.Run("includes task when all blockers done/cancelled", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker1")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocker2")
		id2 := extractID(t, out2)
		out3, _, _ := createTask(t, dir, "Dependent", "--blocked-by", id1+","+id2)
		id3 := extractID(t, out3)

		runCmd(t, dir, "tick", "done", id1)
		runCmd(t, dir, "tick", "cancel", id2)

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		if !strings.Contains(stdout, id3) {
			t.Errorf("task with all blockers closed should be ready, got %q", stdout)
		}
	})

	t.Run("excludes parent with open children", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent")
		idP := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Child", "--parent", idP)
		idC := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		if strings.Contains(stdout, idP) {
			t.Errorf("parent with open child should not be ready")
		}
		// Child should be ready
		if !strings.Contains(stdout, idC) {
			t.Errorf("child should be ready")
		}
	})

	t.Run("includes parent when all children closed", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent")
		idP := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Child", "--parent", idP)
		idC := extractID(t, out2)
		runCmd(t, dir, "tick", "done", idC)

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		if !strings.Contains(stdout, idP) {
			t.Errorf("parent with all children closed should be ready, got %q", stdout)
		}
	})

	t.Run("excludes in_progress/done/cancelled tasks", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "IP task")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Done task")
		id2 := extractID(t, out2)
		out3, _, _ := createTask(t, dir, "Cancelled task")
		id3 := extractID(t, out3)

		runCmd(t, dir, "tick", "start", id1)
		runCmd(t, dir, "tick", "done", id2)
		runCmd(t, dir, "tick", "cancel", id3)

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		if strings.Contains(stdout, id1) {
			t.Errorf("in_progress task should not be ready")
		}
		if strings.Contains(stdout, id2) {
			t.Errorf("done task should not be ready")
		}
		if strings.Contains(stdout, id3) {
			t.Errorf("cancelled task should not be ready")
		}
	})

	t.Run("returns empty list when no tasks ready", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		createTask(t, dir, "Blocked", "--blocked-by", id1)
		runCmd(t, dir, "tick", "start", id1)

		stdout, _, code := runCmd(t, dir, "tick", "ready")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "No tasks found.") {
			t.Errorf("expected 'No tasks found.', got %q", stdout)
		}
	})

	t.Run("orders by priority ASC then created ASC", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Low prio", "--priority", "3")
		createTask(t, dir, "High prio", "--priority", "0")
		createTask(t, dir, "Med prio", "--priority", "2")

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		// Skip header
		if len(lines) < 4 {
			t.Fatalf("expected header + 3 tasks, got %d lines", len(lines))
		}
		if !strings.Contains(lines[1], "High prio") {
			t.Errorf("first should be High prio, got %q", lines[1])
		}
		if !strings.Contains(lines[3], "Low prio") {
			t.Errorf("last should be Low prio, got %q", lines[3])
		}
	})

	t.Run("outputs IDs only with --quiet", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task one")

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "ready")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "tick-") {
				t.Errorf("--quiet should output only IDs, got %q", line)
			}
		}
	})
}
