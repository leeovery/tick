package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runBlocked runs the tick blocked command with the given args and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runBlocked(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "blocked"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestBlocked(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

	t.Run("it returns open task blocked by open dep", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Depends on open", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("task blocked by open dep should appear in blocked")
		}
		// The blocker itself is NOT blocked (it's ready), so it should not appear
		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "tick-blk111") {
				t.Error("unblocked task should not appear in blocked")
			}
		}
	})

	t.Run("it returns open task blocked by in_progress dep", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker in progress", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Depends on in_progress", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("task blocked by in_progress dep should appear in blocked")
		}
	})

	t.Run("it returns parent with open children", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-child1", Title: "Open child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-parent") {
			t.Error("parent with open children should appear in blocked")
		}
	})

	t.Run("it returns parent with in_progress children", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-child1", Title: "In progress child", Status: task.StatusInProgress, Priority: 2, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-parent") {
			t.Error("parent with in_progress children should appear in blocked")
		}
	})

	t.Run("it excludes task when all blockers done or cancelled", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker done", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
			{ID: "tick-blk222", Title: "Blocker cancelled", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
			{ID: "tick-dep111", Title: "Unblocked", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111", "tick-blk222"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-dep111") {
			t.Error("task with all blockers done/cancelled should not appear in blocked")
		}
	})

	t.Run("it excludes in_progress tasks from output", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "In progress", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-aaa111") {
			t.Error("in_progress task should not appear in blocked (only open)")
		}
	})

	t.Run("it excludes done tasks from output", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-aaa111") {
			t.Error("done task should not appear in blocked")
		}
	})

	t.Run("it excludes cancelled tasks from output", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Cancelled task", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-aaa111") {
			t.Error("cancelled task should not appear in blocked")
		}
	})

	t.Run("it returns empty when no blocked tasks", func(t *testing.T) {
		// All tasks are ready (open, no blockers, no children)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Ready task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it returns empty when no tasks exist", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		// All tasks are blocked by a common open blocker
		tasks := []task.Task{
			{ID: "tick-blk000", Title: "Common blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-low111", Title: "Low priority old", Status: task.StatusOpen, Priority: 3, BlockedBy: []string{"tick-blk000"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-hi2222", Title: "High priority new", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-blk000"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-low222", Title: "Low priority new", Status: task.StatusOpen, Priority: 3, BlockedBy: []string{"tick-blk000"}, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
			{ID: "tick-hi1111", Title: "High priority old", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-blk000"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines (header + 4 blocked tasks), got %d: %q", len(lines), stdout)
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

	t.Run("it outputs aligned columns via tick blocked", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk000", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-blk000"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-bbb222", Title: "Login endpoint", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk000"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 blocked tasks), got %d: %q", len(lines), stdout)
		}

		// Check header (dynamic column widths: ID=14, STATUS=8, PRI=5)
		header := lines[0]
		if header != "ID            STATUS  PRI  TYPE  TITLE" {
			t.Errorf("header = %q, want %q", header, "ID            STATUS  PRI  TYPE  TITLE")
		}

		// Check aligned rows
		if lines[1] != "tick-aaa111   open    1    -     Setup Sanctum" {
			t.Errorf("row 1 = %q, want %q", lines[1], "tick-aaa111   open    1    -     Setup Sanctum")
		}
		if lines[2] != "tick-bbb222   open    2    -     Login endpoint" {
			t.Errorf("row 2 = %q, want %q", lines[2], "tick-bbb222   open    2    -     Login endpoint")
		}
	})

	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk000", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa111", Title: "First", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-blk000"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-bbb222", Title: "Second", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk000"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "tick-aaa111\ntick-bbb222\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it returns child of dependency-blocked parent in blocked", func(t *testing.T) {
		// Phase 1 (open) -- blocker
		// Phase 2 (open, blocked_by: Phase 1)
		//   └─ subtask-A (open) -- should be blocked due to ancestor
		tasks := []task.Task{
			{ID: "tick-phase1", Title: "Phase 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-phase2", Title: "Phase 2", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-phase1"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-sub00a", Title: "Subtask A", Status: task.StatusOpen, Priority: 2, Parent: "tick-phase2", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-sub00a") {
			t.Error("child of dependency-blocked parent should appear in blocked")
		}
	})

	t.Run("it returns grandchild of dependency-blocked grandparent in blocked", func(t *testing.T) {
		// Blocker (open)
		// Grandparent (open, blocked_by: Blocker)
		//   └─ Parent (open, no own blockers)
		//       └─ Grandchild (open) -- should be blocked
		tasks := []task.Task{
			{ID: "tick-blkr01", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-gp0001", Title: "Grandparent", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blkr01"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-par001", Title: "Parent", Status: task.StatusOpen, Priority: 2, Parent: "tick-gp0001", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-gc0001", Title: "Grandchild", Status: task.StatusOpen, Priority: 2, Parent: "tick-par001", Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-gc0001") {
			t.Error("grandchild of dependency-blocked grandparent should appear in blocked")
		}
	})

	t.Run("it returns descendant behind intermediate grouping task under blocked ancestor in blocked", func(t *testing.T) {
		// Phase 1 (open)
		// Phase 2 (open, blocked_by: Phase 1)
		//   └─ Group A (open, no own blockers)
		//       └─ subtask-X (open) -- should be blocked
		tasks := []task.Task{
			{ID: "tick-phase1", Title: "Phase 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-phase2", Title: "Phase 2", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-phase1"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-grp00a", Title: "Group A", Status: task.StatusOpen, Priority: 2, Parent: "tick-phase2", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-sub00x", Title: "Subtask X", Status: task.StatusOpen, Priority: 2, Parent: "tick-grp00a", Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-sub00x") {
			t.Error("subtask behind intermediate grouping task under blocked ancestor should appear in blocked")
		}
	})

	t.Run("it excludes descendant from blocked when ancestor blocker resolved", func(t *testing.T) {
		// Phase 1 (done) -- blocker is resolved
		// Phase 2 (open, blocked_by: Phase 1) -- blocker is done, so Phase 2 is unblocked
		//   └─ subtask-A (open) -- should NOT be blocked (ancestor unblocked)
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-phase1", Title: "Phase 1", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
			{ID: "tick-phase2", Title: "Phase 2", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-phase1"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-sub00a", Title: "Subtask A", Status: task.StatusOpen, Priority: 2, Parent: "tick-phase2", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if strings.Contains(stdout, "tick-sub00a") {
			t.Error("descendant should not appear in blocked when ancestor blocker is resolved")
		}
	})

	t.Run("it maintains stats count consistency with blocked ancestors", func(t *testing.T) {
		// Mixed scenario with ancestor-blocked tasks:
		// Phase 1 (open) -- blocker, should be ready
		// Phase 2 (open, blocked_by: Phase 1) -- blocked (own dep)
		//   └─ subtask-A (open) -- blocked (ancestor)
		// Independent (open) -- ready
		tasks := []task.Task{
			{ID: "tick-phase1", Title: "Phase 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-phase2", Title: "Phase 2", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-phase1"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-sub00a", Title: "Subtask A", Status: task.StatusOpen, Priority: 2, Parent: "tick-phase2", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-indep1", Title: "Independent", Status: task.StatusOpen, Priority: 2, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Count ready tasks from list --ready
		readyStdout, _, exitCode := runReady(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("ready exit code = %d, want 0", exitCode)
		}
		readyIDs := strings.Split(strings.TrimRight(readyStdout, "\n"), "\n")
		readyCount := len(readyIDs)
		if readyIDs[0] == "" {
			readyCount = 0
		}

		// Get stats
		statsStdout, _, exitCode := runStats(t, dir, "--json")
		if exitCode != 0 {
			t.Fatalf("stats exit code = %d, want 0", exitCode)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(statsStdout)), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, statsStdout)
		}

		workflow := parsed["workflow"].(map[string]interface{})
		statsReady := int(workflow["ready"].(float64))
		statsBlocked := int(workflow["blocked"].(float64))

		if statsReady != readyCount {
			t.Errorf("stats ready = %d, list --ready count = %d; should match", statsReady, readyCount)
		}

		// Verify ready + blocked = open count
		byStatus := parsed["by_status"].(map[string]interface{})
		openCount := int(byStatus["open"].(float64))
		if statsReady+statsBlocked != openCount {
			t.Errorf("ready(%d) + blocked(%d) = %d, but open = %d; should match",
				statsReady, statsBlocked, statsReady+statsBlocked, openCount)
		}
	})

	t.Run("partial unblock: two blockers one cancelled still blocked", func(t *testing.T) {
		closedTime := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker cancelled", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now, Closed: &closedTime},
			{ID: "tick-blk222", Title: "Blocker still open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Partially unblocked", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111", "tick-blk222"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("task with one open blocker remaining should appear in blocked")
		}
	})
}

func TestCancelUnblocksDependents(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

	t.Run("cancel unblocks single dependent moves to ready", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Dependent", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Before cancel: dependent should be in blocked
		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "tick-dep111") {
			t.Fatal("dependent should be blocked before cancel")
		}

		// Cancel the blocker
		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
			IsTTY:  true,
		}
		exitCode = app.Run([]string{"tick", "cancel", "tick-blk111"})
		if exitCode != 0 {
			t.Fatalf("cancel exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}

		// After cancel: dependent should NOT be in blocked
		stdout, _, exitCode = runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if strings.Contains(stdout, "tick-dep111") {
			t.Error("dependent should not appear in blocked after blocker cancelled")
		}

		// After cancel: dependent should be in ready
		stdout, _, exitCode = runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("dependent should appear in ready after blocker cancelled")
		}
	})

	t.Run("cancel unblocks multiple dependents", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Dependent 1", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-dep222", Title: "Dependent 2", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Cancel the blocker
		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "cancel", "tick-blk111"})
		if exitCode != 0 {
			t.Fatalf("cancel exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}

		// Both dependents should appear in ready
		stdout, _, exitCode := runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("dependent 1 should appear in ready after blocker cancelled")
		}
		if !strings.Contains(stdout, "tick-dep222") {
			t.Error("dependent 2 should appear in ready after blocker cancelled")
		}

		// Neither should be in blocked
		stdout, _, exitCode = runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if strings.Contains(stdout, "tick-dep111") {
			t.Error("dependent 1 should not appear in blocked after blocker cancelled")
		}
		if strings.Contains(stdout, "tick-dep222") {
			t.Error("dependent 2 should not appear in blocked after blocker cancelled")
		}
	})

	t.Run("cancel does not unblock dependent still blocked by another", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk111", Title: "Blocker 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-blk222", Title: "Blocker 2", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-dep111", Title: "Dependent", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111", "tick-blk222"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Cancel only blocker 1
		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "cancel", "tick-blk111"})
		if exitCode != 0 {
			t.Fatalf("cancel exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}

		// Dependent should still be in blocked (blocker 2 still open)
		stdout, _, exitCode := runBlocked(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "tick-dep111") {
			t.Error("dependent should still appear in blocked (blocker 2 still open)")
		}

		// Dependent should NOT be in ready
		stdout, _, exitCode = runReady(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if strings.Contains(stdout, "tick-dep111") {
			t.Error("dependent should not appear in ready (blocker 2 still open)")
		}
	})
}
