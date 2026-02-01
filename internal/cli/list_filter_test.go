package cli

import (
	"strings"
	"testing"
)

func TestListFilters(t *testing.T) {
	t.Run("filters to ready tasks with --ready", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocked", "--blocked-by", id1)
		id2 := extractID(t, out2)

		stdout, _, code := runCmd(t, dir, "tick", "list", "--ready")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, id1) {
			t.Errorf("ready task should appear")
		}
		if strings.Contains(stdout, id2) {
			t.Errorf("blocked task should not appear with --ready")
		}
	})

	t.Run("filters to blocked tasks with --blocked", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocked", "--blocked-by", id1)
		id2 := extractID(t, out2)

		stdout, _, code := runCmd(t, dir, "tick", "list", "--blocked")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, id2) {
			t.Errorf("blocked task should appear")
		}
		if strings.Contains(stdout, id1) {
			t.Errorf("ready task should not appear with --blocked")
		}
	})

	t.Run("filters by --status for all 4 values", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Open task")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "IP task")
		id2 := extractID(t, out2)
		out3, _, _ := createTask(t, dir, "Done task")
		id3 := extractID(t, out3)
		out4, _, _ := createTask(t, dir, "Cancelled task")
		id4 := extractID(t, out4)

		runCmd(t, dir, "tick", "start", id2)
		runCmd(t, dir, "tick", "done", id3)
		runCmd(t, dir, "tick", "cancel", id4)

		// Filter open
		stdout, _, _ := runCmd(t, dir, "tick", "list", "--status", "open")
		if !strings.Contains(stdout, id1) {
			t.Errorf("open task should appear with --status open")
		}
		if strings.Contains(stdout, id2) || strings.Contains(stdout, id3) || strings.Contains(stdout, id4) {
			t.Errorf("non-open tasks should not appear")
		}

		// Filter in_progress
		stdout2, _, _ := runCmd(t, dir, "tick", "list", "--status", "in_progress")
		if !strings.Contains(stdout2, id2) {
			t.Errorf("in_progress task should appear")
		}

		// Filter done
		stdout3, _, _ := runCmd(t, dir, "tick", "list", "--status", "done")
		if !strings.Contains(stdout3, id3) {
			t.Errorf("done task should appear")
		}

		// Filter cancelled
		stdout4, _, _ := runCmd(t, dir, "tick", "list", "--status", "cancelled")
		if !strings.Contains(stdout4, id4) {
			t.Errorf("cancelled task should appear")
		}
	})

	t.Run("filters by --priority", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "High prio", "--priority", "0")
		createTask(t, dir, "Default prio")

		stdout, _, code := runCmd(t, dir, "tick", "list", "--priority", "0")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "High prio") {
			t.Errorf("priority 0 task should appear")
		}
		if strings.Contains(stdout, "Default prio") {
			t.Errorf("priority 2 task should not appear with --priority 0")
		}
	})

	t.Run("combines --ready with --priority", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Ready high", "--priority", "0")
		createTask(t, dir, "Ready low", "--priority", "3")

		stdout, _, _ := runCmd(t, dir, "tick", "list", "--ready", "--priority", "0")
		if !strings.Contains(stdout, "Ready high") {
			t.Errorf("ready + priority 0 should appear")
		}
		if strings.Contains(stdout, "Ready low") {
			t.Errorf("ready + priority 3 should not appear with --priority 0")
		}
	})

	t.Run("combines --status with --priority", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Open high", "--priority", "0")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Open low", "--priority", "3")
		id2 := extractID(t, out2)
		runCmd(t, dir, "tick", "start", id1)
		_ = id2

		stdout, _, _ := runCmd(t, dir, "tick", "list", "--status", "open", "--priority", "3")
		if !strings.Contains(stdout, "Open low") {
			t.Errorf("open + priority 3 should appear")
		}
		if strings.Contains(stdout, "Open high") {
			t.Errorf("in_progress task should not appear with --status open")
		}
	})

	t.Run("errors when --ready and --blocked both set", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "list", "--ready", "--blocked")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "mutually exclusive") {
			t.Errorf("expected mutual exclusion error, got %q", stderr)
		}
	})

	t.Run("errors for invalid status value", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "list", "--status", "invalid")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "invalid status") {
			t.Errorf("expected invalid status error, got %q", stderr)
		}
	})

	t.Run("errors for invalid priority value", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "list", "--priority", "9")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "priority") {
			t.Errorf("expected priority error, got %q", stderr)
		}
	})

	t.Run("returns 'No tasks found.' when no matches", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		stdout, _, code := runCmd(t, dir, "tick", "list", "--status", "done")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "No tasks found.") {
			t.Errorf("expected 'No tasks found.', got %q", stdout)
		}
	})

	t.Run("outputs IDs only with --quiet after filtering", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "list", "--ready")
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

	t.Run("returns all tasks with no filters", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Task A")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Task B")
		id2 := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "list")
		if !strings.Contains(stdout, id1) || !strings.Contains(stdout, id2) {
			t.Errorf("no filters should return all tasks, got %q", stdout)
		}
	})
}
