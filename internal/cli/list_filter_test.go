package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestListFilter(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

	t.Run("it filters to ready tasks with --ready", func(t *testing.T) {
		// tick-blk111 is open, no deps => ready
		// tick-dep111 is open, blocked by open tick-blk111 => NOT ready
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Depends on open", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--ready")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-blk111") {
			t.Error("ready task should appear with --ready flag")
		}
		if strings.Contains(stdout, "tick-dep111") {
			t.Error("blocked task should not appear with --ready flag")
		}
	})

	t.Run("it filters to blocked tasks with --blocked", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Depends on open", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--blocked")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("blocked task should appear with --blocked flag")
		}
		// The blocker itself is NOT blocked
		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-blk111") {
				t.Error("unblocked task should not appear with --blocked flag")
			}
		}
	})

	t.Run("it filters by --status open", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-done11", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
			{ID: "tick-ip1111", Title: "In progress", Status: task.StatusInProgress, Priority: 2, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-open11") {
			t.Error("open task should appear with --status open")
		}
		if strings.Contains(stdout, "tick-done11") {
			t.Error("done task should not appear with --status open")
		}
		if strings.Contains(stdout, "tick-ip1111") {
			t.Error("in_progress task should not appear with --status open")
		}
	})

	t.Run("it filters by --status in_progress", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ip1111", Title: "In progress", Status: task.StatusInProgress, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "in_progress")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ip1111") {
			t.Error("in_progress task should appear with --status in_progress")
		}
		if strings.Contains(stdout, "tick-open11") {
			t.Error("open task should not appear with --status in_progress")
		}
	})

	t.Run("it filters by --status done", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-done11", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "done")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-done11") {
			t.Error("done task should appear with --status done")
		}
		if strings.Contains(stdout, "tick-open11") {
			t.Error("open task should not appear with --status done")
		}
	})

	t.Run("it filters by --status cancelled", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-canc11", Title: "Cancelled task", Status: task.StatusCancelled, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "cancelled")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-canc11") {
			t.Error("cancelled task should appear with --status cancelled")
		}
		if strings.Contains(stdout, "tick-open11") {
			t.Error("open task should not appear with --status cancelled")
		}
	})

	t.Run("it filters by --priority", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-pri111", Title: "Priority 1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-pri222", Title: "Priority 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-pri333", Title: "Priority 3", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--priority", "2")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-pri222") {
			t.Error("priority 2 task should appear with --priority 2")
		}
		if strings.Contains(stdout, "tick-pri111") {
			t.Error("priority 1 task should not appear with --priority 2")
		}
		if strings.Contains(stdout, "tick-pri333") {
			t.Error("priority 3 task should not appear with --priority 2")
		}
	})

	t.Run("it combines --ready with --priority", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-rp1111", Title: "Ready p1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-rp2222", Title: "Ready p2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-blk111", Title: "Blocker", Status: task.StatusOpen, Priority: 1, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-dep111", Title: "Blocked p1", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-blk111"}, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--ready", "--priority", "1")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		// tick-rp1111 is ready, priority 1 => shown
		if !strings.Contains(stdout, "tick-rp1111") {
			t.Error("ready priority 1 task should appear")
		}
		// tick-rp2222 is ready, but priority 2 => excluded
		if strings.Contains(stdout, "tick-rp2222") {
			t.Error("ready priority 2 task should not appear with --priority 1")
		}
		// tick-blk111 is ready (no deps, open), priority 1 => shown
		if !strings.Contains(stdout, "tick-blk111") {
			t.Error("blocker (ready, priority 1) should appear")
		}
		// tick-dep111 is blocked, priority 1 => excluded by --ready
		if strings.Contains(stdout, "tick-dep111") {
			t.Error("blocked task should not appear with --ready")
		}
	})

	t.Run("it combines --status with --priority", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-op1111", Title: "Open p1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-op2222", Title: "Open p2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-dn1111", Title: "Done p1", Status: task.StatusDone, Priority: 1, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "open", "--priority", "1")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-op1111") {
			t.Error("open priority 1 task should appear")
		}
		if strings.Contains(stdout, "tick-op2222") {
			t.Error("open priority 2 task should not appear with --priority 1")
		}
		if strings.Contains(stdout, "tick-dn1111") {
			t.Error("done task should not appear with --status open")
		}
	})

	t.Run("it errors when --ready and --blocked both set", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--ready", "--blocked")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "--ready and --blocked are mutually exclusive") {
			t.Errorf("stderr = %q, want to contain mutual exclusion error", stderr)
		}
	})

	t.Run("it errors for invalid status value", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--status", "invalid")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "invalid status") {
			t.Errorf("stderr = %q, want to contain 'invalid status'", stderr)
		}
		// Should list valid options
		if !strings.Contains(stderr, "open") || !strings.Contains(stderr, "in_progress") || !strings.Contains(stderr, "done") || !strings.Contains(stderr, "cancelled") {
			t.Errorf("stderr = %q, want to contain valid status options", stderr)
		}
	})

	t.Run("it errors for invalid priority value", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--priority", "5")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "invalid priority") {
			t.Errorf("stderr = %q, want to contain 'invalid priority'", stderr)
		}
		if !strings.Contains(stderr, "0-4") {
			t.Errorf("stderr = %q, want to contain valid range '0-4'", stderr)
		}
	})

	t.Run("it errors for non-numeric priority value", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--priority", "abc")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "invalid priority") {
			t.Errorf("stderr = %q, want to contain 'invalid priority'", stderr)
		}
	})

	t.Run("it returns 'No tasks found.' when no matches", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-done11", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it outputs IDs only with --quiet after filtering", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open p1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-open22", Title: "Open p2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ip1111", Title: "In progress", Status: task.StatusInProgress, Priority: 1, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--quiet", "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "tick-open11\ntick-open22\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it returns all tasks with no filters", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-done11", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
			{ID: "tick-ip1111", Title: "In progress", Status: task.StatusInProgress, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-open11") {
			t.Error("open task should appear with no filters")
		}
		if !strings.Contains(stdout, "tick-done11") {
			t.Error("done task should appear with no filters")
		}
		if !strings.Contains(stdout, "tick-ip1111") {
			t.Error("in_progress task should appear with no filters")
		}
	})

	t.Run("it maintains deterministic ordering", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-low111", Title: "Low priority old", Status: task.StatusOpen, Priority: 3, Created: now, Updated: now},
			{ID: "tick-hi2222", Title: "High priority new", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-low222", Title: "Low priority new", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-hi1111", Title: "High priority old", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
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

	t.Run("contradictory filters return empty result no error", func(t *testing.T) {
		// --status done + --ready is contradictory (ready only applies to open tasks)
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-open11", Title: "Open ready", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-done11", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--status", "done", "--ready")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it filters list by --type bug", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-bug111", Title: "Fix crash", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-feat11", Title: "Add feature", Status: task.StatusOpen, Priority: 2, Type: "feature", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-notype", Title: "No type", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--type", "bug")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-bug111") {
			t.Error("bug task should appear with --type bug")
		}
		if strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should not appear with --type bug")
		}
		if strings.Contains(stdout, "tick-notype") {
			t.Error("untyped task should not appear with --type bug")
		}
	})

	t.Run("it normalizes --type filter input to lowercase", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-bug111", Title: "Fix crash", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-feat11", Title: "Add feature", Status: task.StatusOpen, Priority: 2, Type: "feature", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--type", "BUG")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-bug111") {
			t.Error("bug task should appear with --type BUG (normalized)")
		}
		if strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should not appear with --type BUG")
		}
	})

	t.Run("it errors on invalid --type filter value", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--type", "epic")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "invalid type") {
			t.Errorf("stderr = %q, want to contain 'invalid type'", stderr)
		}
		if !strings.Contains(stderr, "bug") || !strings.Contains(stderr, "feature") || !strings.Contains(stderr, "task") || !strings.Contains(stderr, "chore") {
			t.Errorf("stderr = %q, want to contain allowed type values", stderr)
		}
	})

	t.Run("it returns empty list when no tasks match --type filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-bug111", Title: "Fix crash", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--type", "chore")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it filters ready tasks by --type", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-feat11", Title: "Feature ready", Status: task.StatusOpen, Priority: 2, Type: "feature", Created: now, Updated: now},
			{ID: "tick-bug111", Title: "Bug ready", Status: task.StatusOpen, Priority: 2, Type: "bug", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runReady(t, dir, "--type", "feature")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should appear in ready with --type feature")
		}
		if strings.Contains(stdout, "tick-bug111") {
			t.Error("bug task should not appear in ready with --type feature")
		}
	})

	t.Run("it filters blocked tasks by --type", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk000", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-chore1", Title: "Chore blocked", Status: task.StatusOpen, Priority: 2, Type: "chore", BlockedBy: []string{"tick-blk000"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-feat11", Title: "Feature blocked", Status: task.StatusOpen, Priority: 2, Type: "feature", BlockedBy: []string{"tick-blk000"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runBlocked(t, dir, "--type", "chore")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-chore1") {
			t.Error("chore task should appear in blocked with --type chore")
		}
		if strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should not appear in blocked with --type chore")
		}
	})

	t.Run("it combines --type with --status filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-bug111", Title: "Bug open", Status: task.StatusOpen, Priority: 2, Type: "bug", Created: now, Updated: now},
			{ID: "tick-bug222", Title: "Bug in progress", Status: task.StatusInProgress, Priority: 2, Type: "bug", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-feat11", Title: "Feature open", Status: task.StatusOpen, Priority: 2, Type: "feature", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--type", "bug", "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-bug111") {
			t.Error("open bug task should appear with --type bug --status open")
		}
		if strings.Contains(stdout, "tick-bug222") {
			t.Error("in_progress bug should not appear with --status open")
		}
		if strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should not appear with --type bug")
		}
	})

	t.Run("it combines --type with --priority filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-bug111", Title: "Bug p1", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-bug222", Title: "Bug p2", Status: task.StatusOpen, Priority: 2, Type: "bug", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-feat11", Title: "Feature p1", Status: task.StatusOpen, Priority: 1, Type: "feature", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--type", "bug", "--priority", "1")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-bug111") {
			t.Error("bug priority 1 task should appear with --type bug --priority 1")
		}
		if strings.Contains(stdout, "tick-bug222") {
			t.Error("bug priority 2 task should not appear with --priority 1")
		}
		if strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should not appear with --type bug")
		}
	})

	t.Run("it returns all tasks when --type not specified", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-bug111", Title: "Bug task", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-feat11", Title: "Feature task", Status: task.StatusOpen, Priority: 2, Type: "feature", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-notype", Title: "No type task", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-bug111") {
			t.Error("bug task should appear with no --type filter")
		}
		if !strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should appear with no --type filter")
		}
		if !strings.Contains(stdout, "tick-notype") {
			t.Error("untyped task should appear with no --type filter")
		}
	})

	t.Run("it errors when --type flag has no value", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--type")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "--type requires a value") {
			t.Errorf("stderr = %q, want to contain '--type requires a value'", stderr)
		}
	})

	t.Run("it limits list results with --count", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task 1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ccc333", Title: "Task 3", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--count", "2")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		// header + 2 data rows
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), stdout)
		}
		if !strings.HasPrefix(lines[1], "tick-aaa111") {
			t.Errorf("row 1 should start with tick-aaa111, got %q", lines[1])
		}
		if !strings.HasPrefix(lines[2], "tick-bbb222") {
			t.Errorf("row 2 should start with tick-bbb222, got %q", lines[2])
		}
	})

	t.Run("it returns all tasks when --count exceeds result set size", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task 1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ccc333", Title: "Task 3", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--count", "100")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		// header + 3 data rows
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines (header + 3 tasks), got %d: %q", len(lines), stdout)
		}
	})

	t.Run("it errors on --count 0", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--count", "0")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "must be >= 1") {
			t.Errorf("stderr = %q, want to contain 'must be >= 1'", stderr)
		}
	})

	t.Run("it errors on --count negative", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--count", "-1")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "must be >= 1") {
			t.Errorf("stderr = %q, want to contain 'must be >= 1'", stderr)
		}
	})

	t.Run("it errors on --count non-integer", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--count", "abc")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "invalid count") {
			t.Errorf("stderr = %q, want to contain 'invalid count'", stderr)
		}
	})

	t.Run("it errors on --count without value", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--count")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "--count requires a value") {
			t.Errorf("stderr = %q, want to contain '--count requires a value'", stderr)
		}
	})

	t.Run("it limits ready results with --count", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Ready 1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Ready 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ccc333", Title: "Ready 3", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runReady(t, dir, "--count", "2")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), stdout)
		}
	})

	t.Run("it limits blocked results with --count", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk000", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa111", Title: "Blocked 1", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-blk000"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-bbb222", Title: "Blocked 2", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk000"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-ccc333", Title: "Blocked 3", Status: task.StatusOpen, Priority: 3, BlockedBy: []string{"tick-blk000"}, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runBlocked(t, dir, "--count", "2")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), stdout)
		}
	})

	t.Run("it combines --count with --type filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-bug111", Title: "Bug 1", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-bug222", Title: "Bug 2", Status: task.StatusOpen, Priority: 2, Type: "bug", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-bug333", Title: "Bug 3", Status: task.StatusOpen, Priority: 3, Type: "bug", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-feat11", Title: "Feature 1", Status: task.StatusOpen, Priority: 1, Type: "feature", Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--type", "bug", "--count", "2")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		// header + 2 bug tasks
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 bug tasks), got %d: %q", len(lines), stdout)
		}
		if !strings.HasPrefix(lines[1], "tick-bug111") {
			t.Errorf("row 1 should start with tick-bug111, got %q", lines[1])
		}
		if !strings.HasPrefix(lines[2], "tick-bug222") {
			t.Errorf("row 2 should start with tick-bug222, got %q", lines[2])
		}
		if strings.Contains(stdout, "tick-feat11") {
			t.Error("feature task should not appear with --type bug")
		}
	})

	t.Run("it returns all results when --count not specified", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task 1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ccc333", Title: "Task 3", Status: task.StatusOpen, Priority: 3, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		// header + 3 data rows
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines (header + 3 tasks), got %d: %q", len(lines), stdout)
		}
	})
}
