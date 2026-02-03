package cli

import (
	"strings"
	"testing"
)

func TestBlockedQuery(t *testing.T) {
	t.Run("it returns open task blocked by open dep", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-blocked", "Blocked task", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-blocked") {
			t.Errorf("output should contain tick-blocked (has open blocker), got: %q", output)
		}
		// The blocker itself is not blocked (it's ready), so it should not appear
		if strings.Contains(output, "tick-blocker") {
			t.Errorf("output should NOT contain tick-blocker (it is ready, not blocked), got: %q", output)
		}
	})

	t.Run("it returns open task blocked by in_progress dep", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "in_progress", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-blocked", "Blocked task", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-blocked") {
			t.Errorf("output should contain tick-blocked (has in_progress blocker), got: %q", output)
		}
	})

	t.Run("it returns parent with open children", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-parent", "Parent task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-child1", "Child task", "open", 2, nil, "tick-parent", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-parent") {
			t.Errorf("output should contain tick-parent (has open children), got: %q", output)
		}
		// tick-child1 is a ready leaf, not blocked
		if strings.Contains(output, "tick-child1") {
			t.Errorf("output should NOT contain tick-child1 (it is ready, not blocked), got: %q", output)
		}
	})

	t.Run("it returns parent with in_progress children", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-parent", "Parent task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-child1", "Child task", "in_progress", 2, nil, "tick-parent", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-parent") {
			t.Errorf("output should contain tick-parent (has in_progress children), got: %q", output)
		}
	})

	t.Run("it excludes task when all blockers done or cancelled", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-done01", "Done blocker", "done", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-cancel", "Cancelled blocker", "cancelled", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-task01", "Unblocked task", "open", 2, []string{"tick-done01", "tick-cancel"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		// tick-task01 has all blockers closed, so it's ready, not blocked
		if strings.Contains(output, "tick-task01") {
			t.Errorf("output should NOT contain tick-task01 (all blockers done/cancelled), got: %q", output)
		}
	})

	t.Run("it excludes in_progress tasks from output", func(t *testing.T) {
		// An in_progress task is not "open", so it should not appear in blocked output
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-inprog", "In progress", "in_progress", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "tick-inprog") {
			t.Errorf("output should NOT contain tick-inprog (in_progress, not open), got: %q", output)
		}
	})

	t.Run("it excludes done and cancelled from output", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-done01", "Done task", "done", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-cancel", "Cancelled task", "cancelled", 2, nil, "", "2026-01-19T10:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want 'No tasks found.' (done/cancelled excluded)", output)
		}
	})

	t.Run("it returns empty when no blocked tasks", func(t *testing.T) {
		// All tasks are ready (open, no blockers, no children)
		content := taskJSONL("tick-aaa111", "Simple task", "open", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want 'No tasks found.'", output)
		}
	})

	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		// All blocked by a common open blocker
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T09:00:00Z"),
			taskJSONL("tick-low001", "Low pri blocked", "open", 3, []string{"tick-blocker"}, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-hi0001", "High pri early", "open", 1, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-hi0002", "High pri late", "open", 1, []string{"tick-blocker"}, "", "2026-01-19T12:00:00Z"),
			taskJSONL("tick-med001", "Med pri blocked", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T10:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

		// Should have header + 4 blocked tasks (tick-blocker is ready, not blocked)
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d:\n%s", len(lines), output)
		}

		if !strings.Contains(lines[1], "tick-hi0001") {
			t.Errorf("first task = %q, want tick-hi0001 (pri 1, earliest)", lines[1])
		}
		if !strings.Contains(lines[2], "tick-hi0002") {
			t.Errorf("second task = %q, want tick-hi0002 (pri 1, later)", lines[2])
		}
		if !strings.Contains(lines[3], "tick-med001") {
			t.Errorf("third task = %q, want tick-med001 (pri 2)", lines[3])
		}
		if !strings.Contains(lines[4], "tick-low001") {
			t.Errorf("fourth task = %q, want tick-low001 (pri 3)", lines[4])
		}
	})

	t.Run("it outputs aligned columns via tick blocked", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T09:00:00Z"),
			taskJSONL("tick-aaa111", "Blocked one", "open", 1, []string{"tick-blocker"}, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Blocked two", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d:\n%s", len(lines), output)
		}

		// Verify header
		header := lines[0]
		if !strings.HasPrefix(header, "ID") {
			t.Errorf("header = %q, want it to start with 'ID'", header)
		}
		if !strings.Contains(header, "STATUS") {
			t.Errorf("header = %q, want it to contain 'STATUS'", header)
		}
		if !strings.Contains(header, "PRI") {
			t.Errorf("header = %q, want it to contain 'PRI'", header)
		}
		if !strings.Contains(header, "TITLE") {
			t.Errorf("header = %q, want it to contain 'TITLE'", header)
		}

		// Verify column alignment
		headerStatusPos := strings.Index(header, "STATUS")
		row1StatusPos := strings.Index(lines[1], "open")
		row2StatusPos := strings.Index(lines[2], "open")
		if headerStatusPos != row1StatusPos || headerStatusPos != row2StatusPos {
			t.Errorf("columns not aligned: header STATUS at %d, row1 at %d, row2 at %d",
				headerStatusPos, row1StatusPos, row2StatusPos)
		}
	})

	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want %q", output, "No tasks found.")
		}
	})

	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T09:00:00Z"),
			taskJSONL("tick-aaa111", "Blocked one", "open", 1, []string{"tick-blocker"}, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Blocked two", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}

		output := strings.TrimRight(stdout.String(), "\n")
		lines := strings.Split(output, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per blocked task), got %d:\n%q", len(lines), output)
		}

		if lines[0] != "tick-aaa111" {
			t.Errorf("line 0 = %q, want %q", lines[0], "tick-aaa111")
		}
		if lines[1] != "tick-bbb222" {
			t.Errorf("line 1 = %q, want %q", lines[1], "tick-bbb222")
		}
	})
}

func TestBlockedViaListFlag(t *testing.T) {
	t.Run("it works via list --blocked flag", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-blocked", "Blocked task", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--blocked"})
		if err != nil {
			t.Fatalf("list --blocked returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-blocked") {
			t.Errorf("output should contain tick-blocked (blocked), got: %q", output)
		}
		if strings.Contains(output, "tick-blocker") {
			t.Errorf("output should NOT contain tick-blocker (ready, not blocked), got: %q", output)
		}
	})
}

func TestCancelUnblocksDependents(t *testing.T) {
	t.Run("cancel unblocks single dependent -- moves to ready", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-depend1", "Dependent", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		// First verify tick-depend1 is blocked
		app1 := NewApp()
		app1.workDir = dir
		var stdout1 strings.Builder
		app1.stdout = &stdout1

		err := app1.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}
		if !strings.Contains(stdout1.String(), "tick-depend1") {
			t.Fatalf("tick-depend1 should be blocked initially, got: %q", stdout1.String())
		}

		// Cancel the blocker
		app2 := NewApp()
		app2.workDir = dir
		var stdout2 strings.Builder
		app2.stdout = &stdout2

		err = app2.Run([]string{"tick", "cancel", "tick-blocker"})
		if err != nil {
			t.Fatalf("cancel returned error: %v", err)
		}

		// Now verify tick-depend1 is ready (no longer blocked)
		app3 := NewApp()
		app3.workDir = dir
		var stdout3 strings.Builder
		app3.stdout = &stdout3

		err = app3.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}
		if !strings.Contains(stdout3.String(), "tick-depend1") {
			t.Errorf("tick-depend1 should be ready after blocker cancelled, got: %q", stdout3.String())
		}

		// And not in blocked anymore
		app4 := NewApp()
		app4.workDir = dir
		var stdout4 strings.Builder
		app4.stdout = &stdout4

		err = app4.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}
		if strings.Contains(stdout4.String(), "tick-depend1") {
			t.Errorf("tick-depend1 should NOT be blocked after blocker cancelled, got: %q", stdout4.String())
		}
	})

	t.Run("cancel unblocks multiple dependents", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-depend1", "Dependent 1", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-depend2", "Dependent 2", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T12:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		// Cancel the blocker
		app1 := NewApp()
		app1.workDir = dir
		var stdout1 strings.Builder
		app1.stdout = &stdout1

		err := app1.Run([]string{"tick", "cancel", "tick-blocker"})
		if err != nil {
			t.Fatalf("cancel returned error: %v", err)
		}

		// Both dependents should now be ready
		app2 := NewApp()
		app2.workDir = dir
		var stdout2 strings.Builder
		app2.stdout = &stdout2

		err = app2.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout2.String()
		if !strings.Contains(output, "tick-depend1") {
			t.Errorf("tick-depend1 should be ready after blocker cancelled, got: %q", output)
		}
		if !strings.Contains(output, "tick-depend2") {
			t.Errorf("tick-depend2 should be ready after blocker cancelled, got: %q", output)
		}
	})

	t.Run("cancel does not unblock dependent still blocked by another", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-block1", "Blocker 1", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-block2", "Blocker 2", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-depend1", "Dependent", "open", 2, []string{"tick-block1", "tick-block2"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		// Cancel only one blocker
		app1 := NewApp()
		app1.workDir = dir
		var stdout1 strings.Builder
		app1.stdout = &stdout1

		err := app1.Run([]string{"tick", "cancel", "tick-block1"})
		if err != nil {
			t.Fatalf("cancel returned error: %v", err)
		}

		// Dependent should still be blocked (tick-block2 is still open)
		app2 := NewApp()
		app2.workDir = dir
		var stdout2 strings.Builder
		app2.stdout = &stdout2

		err = app2.Run([]string{"tick", "blocked"})
		if err != nil {
			t.Fatalf("blocked returned error: %v", err)
		}
		if !strings.Contains(stdout2.String(), "tick-depend1") {
			t.Errorf("tick-depend1 should still be blocked (tick-block2 still open), got: %q", stdout2.String())
		}

		// Not in ready
		app3 := NewApp()
		app3.workDir = dir
		var stdout3 strings.Builder
		app3.stdout = &stdout3

		err = app3.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}
		if strings.Contains(stdout3.String(), "tick-depend1") {
			t.Errorf("tick-depend1 should NOT be ready (still blocked by tick-block2), got: %q", stdout3.String())
		}
	})
}
