package cli

import (
	"strings"
	"testing"
)

func TestBlockedCommand(t *testing.T) {
	t.Run("returns open task blocked by open dep", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocked", "--blocked-by", id1)
		id2 := extractID(t, out2)

		stdout, _, code := runCmd(t, dir, "tick", "blocked")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, id2) {
			t.Errorf("blocked task %s should appear, got %q", id2, stdout)
		}
	})

	t.Run("returns open task blocked by in_progress dep", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocked", "--blocked-by", id1)
		id2 := extractID(t, out2)
		runCmd(t, dir, "tick", "start", id1)

		stdout, _, _ := runCmd(t, dir, "tick", "blocked")
		if !strings.Contains(stdout, id2) {
			t.Errorf("blocked task should appear")
		}
	})

	t.Run("returns parent with open children", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent")
		idP := extractID(t, out1)
		createTask(t, dir, "Child", "--parent", idP)

		stdout, _, _ := runCmd(t, dir, "tick", "blocked")
		if !strings.Contains(stdout, idP) {
			t.Errorf("parent with open children should be blocked, got %q", stdout)
		}
	})

	t.Run("excludes task when all blockers done/cancelled", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent", "--blocked-by", id1)
		id2 := extractID(t, out2)
		runCmd(t, dir, "tick", "done", id1)

		stdout, _, _ := runCmd(t, dir, "tick", "blocked")
		if strings.Contains(stdout, id2) {
			t.Errorf("unblocked task should not appear in blocked output")
		}
	})

	t.Run("excludes in_progress/done/cancelled from output", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "IP task")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Done task")
		id2 := extractID(t, out2)

		runCmd(t, dir, "tick", "start", id1)
		runCmd(t, dir, "tick", "done", id2)

		stdout, _, _ := runCmd(t, dir, "tick", "blocked")
		if strings.Contains(stdout, id1) {
			t.Errorf("in_progress task should not appear in blocked")
		}
		if strings.Contains(stdout, id2) {
			t.Errorf("done task should not appear in blocked")
		}
	})

	t.Run("returns empty when no blocked tasks", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Free task")

		stdout, _, code := runCmd(t, dir, "tick", "blocked")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "No tasks found.") {
			t.Errorf("expected 'No tasks found.', got %q", stdout)
		}
	})

	t.Run("outputs IDs only with --quiet", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		createTask(t, dir, "Blocked", "--blocked-by", id1)

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "blocked")
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

	t.Run("cancel unblocks single dependent", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent", "--blocked-by", id1)
		id2 := extractID(t, out2)

		// Before cancel: dependent is blocked
		stdout1, _, _ := runCmd(t, dir, "tick", "blocked")
		if !strings.Contains(stdout1, id2) {
			t.Fatalf("dependent should be blocked initially")
		}

		// Cancel blocker
		runCmd(t, dir, "tick", "cancel", id1)

		// After cancel: dependent should be ready
		stdout2, _, _ := runCmd(t, dir, "tick", "ready")
		if !strings.Contains(stdout2, id2) {
			t.Errorf("dependent should be ready after blocker cancelled, got %q", stdout2)
		}
	})

	t.Run("cancel unblocks multiple dependents", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dep A", "--blocked-by", id1)
		idA := extractID(t, out2)
		out3, _, _ := createTask(t, dir, "Dep B", "--blocked-by", id1)
		idB := extractID(t, out3)

		runCmd(t, dir, "tick", "cancel", id1)

		stdout, _, _ := runCmd(t, dir, "tick", "ready")
		if !strings.Contains(stdout, idA) {
			t.Errorf("dep A should be ready")
		}
		if !strings.Contains(stdout, idB) {
			t.Errorf("dep B should be ready")
		}
	})

	t.Run("cancel does not unblock dependent still blocked by another", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker1")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocker2")
		id2 := extractID(t, out2)
		out3, _, _ := createTask(t, dir, "Dependent", "--blocked-by", id1+","+id2)
		id3 := extractID(t, out3)

		runCmd(t, dir, "tick", "cancel", id1)

		// Still blocked by blocker2
		stdout, _, _ := runCmd(t, dir, "tick", "blocked")
		if !strings.Contains(stdout, id3) {
			t.Errorf("dependent should still be blocked, got %q", stdout)
		}
	})
}
