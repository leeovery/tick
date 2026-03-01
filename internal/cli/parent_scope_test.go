package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestParentScope(t *testing.T) {
	now := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)

	t.Run("it returns all descendants of parent (direct children)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Child one", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch2222", Title: "Child two", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-other1", Title: "Unrelated", Status: task.StatusOpen, Priority: 2, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("child one should appear in parent-scoped list")
		}
		if !strings.Contains(stdout, "tick-ch2222") {
			t.Error("child two should appear in parent-scoped list")
		}
		if strings.Contains(stdout, "tick-other1") {
			t.Error("unrelated task should not appear in parent-scoped list")
		}
	})

	t.Run("it returns all descendants recursively (3+ levels deep)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-root01", Title: "Root", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-lvl1a1", Title: "Level 1a", Status: task.StatusOpen, Priority: 2, Parent: "tick-root01", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-lvl2a1", Title: "Level 2a", Status: task.StatusOpen, Priority: 2, Parent: "tick-lvl1a1", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-lvl3a1", Title: "Level 3a", Status: task.StatusOpen, Priority: 2, Parent: "tick-lvl2a1", Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-root01")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-lvl1a1") {
			t.Error("level 1 descendant should appear")
		}
		if !strings.Contains(stdout, "tick-lvl2a1") {
			t.Error("level 2 descendant should appear")
		}
		if !strings.Contains(stdout, "tick-lvl3a1") {
			t.Error("level 3 descendant should appear")
		}
	})

	t.Run("it excludes parent task itself from results", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Child one", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-parent") {
				t.Error("parent task itself should not appear in parent-scoped results")
			}
		}
	})

	t.Run("it returns empty result when parent has no descendants", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-lonely", Title: "Lonely parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-lonely")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it errors with task not found for non-existent parent ID", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--parent", "tick-nope01")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "task 'tick-nope01' not found") {
			t.Errorf("stderr = %q, want to contain task not found error", stderr)
		}
	})

	t.Run("it returns only ready tasks within parent scope with tick ready --parent", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			// ch1 is open, no blockers, no children => ready
			{ID: "tick-ch1111", Title: "Ready child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			// ch2 is open, blocked by blk => NOT ready
			{ID: "tick-blk001", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-ch2222", Title: "Blocked child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", BlockedBy: []string{"tick-blk001"}, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
			// An outside ready task that should not appear
			{ID: "tick-out001", Title: "Outside ready", Status: task.StatusOpen, Priority: 2, Created: now.Add(4 * time.Second), Updated: now.Add(4 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runReady(t, dir, "--parent", "tick-parent")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("ready child within parent scope should appear")
		}
		if strings.Contains(stdout, "tick-ch2222") {
			t.Error("blocked child should not appear in ready")
		}
		if strings.Contains(stdout, "tick-out001") {
			t.Error("outside task should not appear in parent-scoped ready")
		}
	})

	t.Run("it returns only blocked tasks within parent scope with tick blocked --parent", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			// ch1 is open, blocked => blocked
			{ID: "tick-blk001", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch1111", Title: "Blocked child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", BlockedBy: []string{"tick-blk001"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			// ch2 is open, ready
			{ID: "tick-ch2222", Title: "Ready child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
			// An outside blocked task that should not appear
			{ID: "tick-out001", Title: "Outside blocked", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk001"}, Created: now.Add(4 * time.Second), Updated: now.Add(4 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runBlocked(t, dir, "--parent", "tick-parent")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("blocked child within parent scope should appear")
		}
		if strings.Contains(stdout, "tick-ch2222") {
			t.Error("ready child should not appear in blocked")
		}
		if strings.Contains(stdout, "tick-out001") {
			t.Error("outside task should not appear in parent-scoped blocked")
		}
	})

	t.Run("it combines --parent with --status filter", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Open child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch2222", Title: "Done child", Status: task.StatusDone, Priority: 2, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent", "--status", "done")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch2222") {
			t.Error("done child should appear with --parent + --status done")
		}
		if strings.Contains(stdout, "tick-ch1111") {
			t.Error("open child should not appear with --status done")
		}
	})

	t.Run("it combines --parent with --priority filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "P1 child", Status: task.StatusOpen, Priority: 1, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch2222", Title: "P3 child", Status: task.StatusOpen, Priority: 3, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent", "--priority", "1")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("p1 child should appear with --parent + --priority 1")
		}
		if strings.Contains(stdout, "tick-ch2222") {
			t.Error("p3 child should not appear with --priority 1")
		}
	})

	t.Run("it combines --parent with --ready and --priority", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Ready p1 child", Status: task.StatusOpen, Priority: 1, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch2222", Title: "Ready p3 child", Status: task.StatusOpen, Priority: 3, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-blk001", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
			{ID: "tick-ch3333", Title: "Blocked p1 child", Status: task.StatusOpen, Priority: 1, Parent: "tick-parent", BlockedBy: []string{"tick-blk001"}, Created: now.Add(4 * time.Second), Updated: now.Add(4 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent", "--ready", "--priority", "1")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("ready p1 child should appear")
		}
		if strings.Contains(stdout, "tick-ch2222") {
			t.Error("ready p3 child should not appear (wrong priority)")
		}
		if strings.Contains(stdout, "tick-ch3333") {
			t.Error("blocked p1 child should not appear (not ready)")
		}
	})

	t.Run("it combines --parent with --blocked and --status", func(t *testing.T) {
		// This is a bit unusual (blocked implies open status), so --blocked + --status open
		// should work. --blocked + --status done is contradictory => empty.
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-blk001", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch1111", Title: "Blocked child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", BlockedBy: []string{"tick-blk001"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// --parent + --blocked + --status open should find blocked descendants
		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent", "--blocked", "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("blocked open child in parent scope should appear")
		}
	})

	t.Run("it handles case-insensitive parent ID", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Child one", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "TICK-PARENT")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("child should appear with case-insensitive parent ID")
		}
	})

	t.Run("it excludes tasks outside the parent subtree", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-par001", Title: "Parent A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-par002", Title: "Parent B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch1111", Title: "Child of A", Status: task.StatusOpen, Priority: 2, Parent: "tick-par001", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-ch2222", Title: "Child of B", Status: task.StatusOpen, Priority: 2, Parent: "tick-par002", Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
			{ID: "tick-orphan", Title: "No parent", Status: task.StatusOpen, Priority: 2, Created: now.Add(4 * time.Second), Updated: now.Add(4 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-par001")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("child of A should appear")
		}
		if strings.Contains(stdout, "tick-ch2222") {
			t.Error("child of B should not appear in A's scope")
		}
		if strings.Contains(stdout, "tick-orphan") {
			t.Error("orphan task should not appear in A's scope")
		}
	})

	t.Run("it outputs IDs only with --quiet within scoped set", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Child one", Status: task.StatusOpen, Priority: 1, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch2222", Title: "Child two", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-other1", Title: "Outside", Status: task.StatusOpen, Priority: 2, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--quiet", "--parent", "tick-parent")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "tick-ch1111\ntick-ch2222\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it resolves partial parent ID and returns children", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Child one", Status: task.StatusOpen, Priority: 2, Parent: "tick-a3f1b2", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch2222", Title: "Child two", Status: task.StatusOpen, Priority: 2, Parent: "tick-a3f1b2", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-other1", Title: "Unrelated", Status: task.StatusOpen, Priority: 2, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--parent", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ch1111") {
			t.Error("child one should appear in parent-scoped list with partial ID")
		}
		if !strings.Contains(stdout, "tick-ch2222") {
			t.Error("child two should appear in parent-scoped list with partial ID")
		}
		if strings.Contains(stdout, "tick-other1") {
			t.Error("unrelated task should not appear in parent-scoped list")
		}
	})

	t.Run("it errors with ambiguous partial parent ID", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Parent A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-a3f1b3", Title: "Parent B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch1111", Title: "Child of A", Status: task.StatusOpen, Priority: 2, Parent: "tick-a3f1b2", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runList(t, dir, "--parent", "a3f")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "ambiguous") {
			t.Errorf("stderr should contain 'ambiguous', got %q", stderr)
		}
		if !strings.Contains(stderr, "tick-a3f1b2") {
			t.Errorf("stderr should list tick-a3f1b2, got %q", stderr)
		}
		if !strings.Contains(stderr, "tick-a3f1b3") {
			t.Errorf("stderr should list tick-a3f1b3, got %q", stderr)
		}
	})

	t.Run("it errors with not found for non-matching partial parent ID", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Some task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runList(t, dir, "--parent", "zzz")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it returns No tasks found when descendants exist but none match filters", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ch1111", Title: "Open child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ch2222", Title: "Done child", Status: task.StatusDone, Priority: 2, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// All descendants are open or done, none are cancelled
		stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent", "--status", "cancelled")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})
}
