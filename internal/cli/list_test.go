package cli

import (
	"os"
	"strings"
	"testing"
)

func TestListCommand(t *testing.T) {
	t.Run("it lists all tasks with aligned columns", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Setup Sanctum","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Login endpoint","status":"in_progress","priority":1,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--pretty", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

		// Should have header + 2 data rows
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

		// Verify data rows contain expected values
		if !strings.Contains(lines[1], "tick-aaa111") {
			t.Errorf("row 1 = %q, want it to contain 'tick-aaa111'", lines[1])
		}
		if !strings.Contains(lines[1], "done") {
			t.Errorf("row 1 = %q, want it to contain 'done'", lines[1])
		}
		if !strings.Contains(lines[2], "tick-bbb222") {
			t.Errorf("row 2 = %q, want it to contain 'tick-bbb222'", lines[2])
		}

		// Verify column alignment: all rows should have consistent column positions
		// ID column is 12 chars wide, STATUS is 12, PRI is 4
		// Check that STATUS starts at approximately the same position in each line
		headerStatusPos := strings.Index(header, "STATUS")
		row1StatusPos := strings.Index(lines[1], "done")
		row2StatusPos := strings.Index(lines[2], "in_progress")
		if headerStatusPos != row1StatusPos || headerStatusPos != row2StatusPos {
			t.Errorf("columns not aligned: header STATUS at %d, row1 status at %d, row2 status at %d",
				headerStatusPos, row1StatusPos, row2StatusPos)
		}
	})

	t.Run("it lists tasks ordered by priority then created date", func(t *testing.T) {
		// Priority 2, created first
		// Priority 1, created second
		// Priority 1, created third
		// Expected order: tick-bbb222 (pri 1, earlier), tick-ccc333 (pri 1, later), tick-aaa111 (pri 2)
		content := `{"id":"tick-aaa111","title":"Low priority","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"High priority early","status":"open","priority":1,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
{"id":"tick-ccc333","title":"High priority late","status":"open","priority":1,"created":"2026-01-19T12:00:00Z","updated":"2026-01-19T12:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

		if len(lines) != 4 {
			t.Fatalf("expected 4 lines (header + 3 tasks), got %d:\n%s", len(lines), output)
		}

		// First data row should be tick-bbb222 (priority 1, earlier created)
		if !strings.Contains(lines[1], "tick-bbb222") {
			t.Errorf("first task = %q, want tick-bbb222 (highest priority, earliest created)", lines[1])
		}
		// Second data row should be tick-ccc333 (priority 1, later created)
		if !strings.Contains(lines[2], "tick-ccc333") {
			t.Errorf("second task = %q, want tick-ccc333 (highest priority, later created)", lines[2])
		}
		// Third data row should be tick-aaa111 (priority 2)
		if !strings.Contains(lines[3], "tick-aaa111") {
			t.Errorf("third task = %q, want tick-aaa111 (lower priority)", lines[3])
		}
	})

	t.Run("it prints 'No tasks found.' when no tasks exist", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--pretty", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want %q", output, "No tasks found.")
		}
	})

	t.Run("it prints only task IDs with --quiet flag on list", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task one","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Task two","status":"open","priority":2,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := strings.TrimRight(stdout.String(), "\n")
		lines := strings.Split(output, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per task), got %d:\n%q", len(lines), output)
		}

		// Should be just IDs, ordered by priority then created
		if lines[0] != "tick-aaa111" {
			t.Errorf("line 0 = %q, want %q", lines[0], "tick-aaa111")
		}
		if lines[1] != "tick-bbb222" {
			t.Errorf("line 1 = %q, want %q", lines[1], "tick-bbb222")
		}
	})

	t.Run("it returns all tasks with no filters", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Open task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "In progress", "in_progress", 1, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-ccc333", "Done task", "done", 3, nil, "", "2026-01-19T12:00:00Z"),
			taskJSONL("tick-ddd444", "Cancelled", "cancelled", 0, nil, "", "2026-01-19T13:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

		// header + 4 data rows
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d:\n%s", len(lines), output)
		}

		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("output should contain tick-aaa111")
		}
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("output should contain tick-bbb222")
		}
		if !strings.Contains(output, "tick-ccc333") {
			t.Errorf("output should contain tick-ccc333")
		}
		if !strings.Contains(output, "tick-ddd444") {
			t.Errorf("output should contain tick-ddd444")
		}
	})

	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		// This test verifies list goes through the Query path (which does shared lock + freshness)
		// We verify by confirming SQLite cache is created after a list operation
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		// The list command should have triggered a Query which creates cache.db
		// (EnsureFresh creates it if missing)
		cachePath := dir + "/.tick/cache.db"
		if _, err := os.Stat(cachePath); err != nil {
			t.Errorf("cache.db should exist after list (created by read flow): %v", err)
		}
	})
}

func TestListFilterFlags(t *testing.T) {
	// Shared test data: a mix of statuses, priorities, and dependencies
	mixedContent := func() string {
		return strings.Join([]string{
			// Ready tasks (open, no blockers, no open children)
			taskJSONL("tick-ready1", "Ready P1", "open", 1, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-ready2", "Ready P2", "open", 2, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-ready3", "Ready P3", "open", 3, nil, "", "2026-01-19T12:00:00Z"),
			// Blocked task (open, blocked by tick-ready1)
			taskJSONL("tick-blk01", "Blocked P2", "open", 2, []string{"tick-ready1"}, "", "2026-01-19T13:00:00Z"),
			// In-progress task
			taskJSONL("tick-inpr1", "In Progress", "in_progress", 1, nil, "", "2026-01-19T14:00:00Z"),
			// Done task
			taskJSONL("tick-done1", "Done task", "done", 2, nil, "", "2026-01-19T15:00:00Z"),
			// Cancelled task
			taskJSONL("tick-canc1", "Cancelled task", "cancelled", 4, nil, "", "2026-01-19T16:00:00Z"),
		}, "\n") + "\n"
	}

	t.Run("it filters to ready tasks with --ready", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--ready"})
		if err != nil {
			t.Fatalf("list --ready returned error: %v", err)
		}

		output := stdout.String()
		// Ready tasks: tick-ready1, tick-ready2, tick-ready3
		if !strings.Contains(output, "tick-ready1") {
			t.Errorf("output should contain tick-ready1 (ready task)")
		}
		if !strings.Contains(output, "tick-ready2") {
			t.Errorf("output should contain tick-ready2 (ready task)")
		}
		if !strings.Contains(output, "tick-ready3") {
			t.Errorf("output should contain tick-ready3 (ready task)")
		}
		// NOT blocked, in_progress, done, or cancelled
		if strings.Contains(output, "tick-blk01") {
			t.Errorf("output should NOT contain tick-blk01 (blocked task)")
		}
		if strings.Contains(output, "tick-inpr1") {
			t.Errorf("output should NOT contain tick-inpr1 (in_progress)")
		}
		if strings.Contains(output, "tick-done1") {
			t.Errorf("output should NOT contain tick-done1 (done)")
		}
		if strings.Contains(output, "tick-canc1") {
			t.Errorf("output should NOT contain tick-canc1 (cancelled)")
		}
	})

	t.Run("it filters to blocked tasks with --blocked", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--blocked"})
		if err != nil {
			t.Fatalf("list --blocked returned error: %v", err)
		}

		output := stdout.String()
		// Only tick-blk01 is blocked
		if !strings.Contains(output, "tick-blk01") {
			t.Errorf("output should contain tick-blk01 (blocked task)")
		}
		// Ready tasks are not blocked
		if strings.Contains(output, "tick-ready1") {
			t.Errorf("output should NOT contain tick-ready1 (ready, not blocked)")
		}
		if strings.Contains(output, "tick-inpr1") {
			t.Errorf("output should NOT contain tick-inpr1 (in_progress, not open)")
		}
	})

	t.Run("it filters by --status open", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--status", "open"})
		if err != nil {
			t.Fatalf("list --status open returned error: %v", err)
		}

		output := stdout.String()
		// open tasks: tick-ready1, tick-ready2, tick-ready3, tick-blk01
		if !strings.Contains(output, "tick-ready1") {
			t.Errorf("output should contain tick-ready1 (open)")
		}
		if !strings.Contains(output, "tick-blk01") {
			t.Errorf("output should contain tick-blk01 (open)")
		}
		if strings.Contains(output, "tick-inpr1") {
			t.Errorf("output should NOT contain tick-inpr1 (in_progress)")
		}
		if strings.Contains(output, "tick-done1") {
			t.Errorf("output should NOT contain tick-done1 (done)")
		}
		if strings.Contains(output, "tick-canc1") {
			t.Errorf("output should NOT contain tick-canc1 (cancelled)")
		}
	})

	t.Run("it filters by --status in_progress", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--status", "in_progress"})
		if err != nil {
			t.Fatalf("list --status in_progress returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-inpr1") {
			t.Errorf("output should contain tick-inpr1 (in_progress)")
		}
		if strings.Contains(output, "tick-ready1") {
			t.Errorf("output should NOT contain tick-ready1 (open)")
		}
	})

	t.Run("it filters by --status done", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--status", "done"})
		if err != nil {
			t.Fatalf("list --status done returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-done1") {
			t.Errorf("output should contain tick-done1 (done)")
		}
		if strings.Contains(output, "tick-ready1") {
			t.Errorf("output should NOT contain tick-ready1 (open)")
		}
	})

	t.Run("it filters by --status cancelled", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--status", "cancelled"})
		if err != nil {
			t.Fatalf("list --status cancelled returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-canc1") {
			t.Errorf("output should contain tick-canc1 (cancelled)")
		}
		if strings.Contains(output, "tick-ready1") {
			t.Errorf("output should NOT contain tick-ready1 (open)")
		}
	})

	t.Run("it filters by --priority", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--priority", "1"})
		if err != nil {
			t.Fatalf("list --priority 1 returned error: %v", err)
		}

		output := stdout.String()
		// Priority 1 tasks: tick-ready1 (open, pri 1), tick-inpr1 (in_progress, pri 1)
		if !strings.Contains(output, "tick-ready1") {
			t.Errorf("output should contain tick-ready1 (priority 1)")
		}
		if !strings.Contains(output, "tick-inpr1") {
			t.Errorf("output should contain tick-inpr1 (priority 1)")
		}
		// Priority != 1 tasks should not appear
		if strings.Contains(output, "tick-ready2") {
			t.Errorf("output should NOT contain tick-ready2 (priority 2)")
		}
		if strings.Contains(output, "tick-canc1") {
			t.Errorf("output should NOT contain tick-canc1 (priority 4)")
		}
	})

	t.Run("it combines --ready with --priority", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--ready", "--priority", "1"})
		if err != nil {
			t.Fatalf("list --ready --priority 1 returned error: %v", err)
		}

		output := stdout.String()
		// Only ready AND priority 1: tick-ready1
		if !strings.Contains(output, "tick-ready1") {
			t.Errorf("output should contain tick-ready1 (ready, priority 1)")
		}
		// tick-ready2 is ready but priority 2
		if strings.Contains(output, "tick-ready2") {
			t.Errorf("output should NOT contain tick-ready2 (priority 2)")
		}
		// tick-inpr1 is priority 1 but not ready
		if strings.Contains(output, "tick-inpr1") {
			t.Errorf("output should NOT contain tick-inpr1 (in_progress, not ready)")
		}
	})

	t.Run("it combines --status with --priority", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--status", "open", "--priority", "2"})
		if err != nil {
			t.Fatalf("list --status open --priority 2 returned error: %v", err)
		}

		output := stdout.String()
		// open AND priority 2: tick-ready2, tick-blk01
		if !strings.Contains(output, "tick-ready2") {
			t.Errorf("output should contain tick-ready2 (open, priority 2)")
		}
		if !strings.Contains(output, "tick-blk01") {
			t.Errorf("output should contain tick-blk01 (open, priority 2)")
		}
		// tick-ready1 is open but priority 1
		if strings.Contains(output, "tick-ready1") {
			t.Errorf("output should NOT contain tick-ready1 (priority 1)")
		}
		// tick-done1 is priority 2 but done
		if strings.Contains(output, "tick-done1") {
			t.Errorf("output should NOT contain tick-done1 (done)")
		}
	})

	t.Run("it errors when --ready and --blocked both set", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--ready", "--blocked"})
		if err == nil {
			t.Fatal("expected error for --ready + --blocked, got nil")
		}

		if !strings.Contains(err.Error(), "--ready") || !strings.Contains(err.Error(), "--blocked") {
			t.Errorf("error = %q, want it to mention --ready and --blocked", err.Error())
		}
	})

	t.Run("it errors for invalid status value", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--status", "invalid"})
		if err == nil {
			t.Fatal("expected error for invalid status, got nil")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "invalid") {
			t.Errorf("error = %q, want it to mention 'invalid'", errMsg)
		}
		// Should mention valid values
		if !strings.Contains(errMsg, "open") || !strings.Contains(errMsg, "done") {
			t.Errorf("error = %q, want it to list valid status options", errMsg)
		}
	})

	t.Run("it errors for invalid priority value", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--priority", "9"})
		if err == nil {
			t.Fatal("expected error for invalid priority, got nil")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "0") || !strings.Contains(errMsg, "4") {
			t.Errorf("error = %q, want it to mention valid priority range 0-4", errMsg)
		}
	})

	t.Run("it errors for non-numeric priority value", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--priority", "high"})
		if err == nil {
			t.Fatal("expected error for non-numeric priority, got nil")
		}
	})

	t.Run("it returns 'No tasks found.' when no matches", func(t *testing.T) {
		// Only have open tasks with priority 2, filter for priority 0
		content := taskJSONL("tick-aaa111", "Open task", "open", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--pretty", "list", "--priority", "0"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want %q", output, "No tasks found.")
		}
	})

	t.Run("it returns empty for contradictory filters without error", func(t *testing.T) {
		// --status done --ready is contradictory (ready only shows open tasks)
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Open task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Done task", "done", 2, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--pretty", "list", "--status", "done", "--ready"})
		if err != nil {
			t.Fatalf("expected no error for contradictory filters, got: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want %q (contradictory filters should return empty)", output, "No tasks found.")
		}
	})

	t.Run("it outputs IDs only with --quiet after filtering", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "list", "--status", "open", "--priority", "2"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := strings.TrimRight(stdout.String(), "\n")
		lines := strings.Split(output, "\n")

		// open AND priority 2: tick-ready2, tick-blk01 (ordered by priority ASC, created ASC)
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d:\n%q", len(lines), output)
		}

		if lines[0] != "tick-ready2" {
			t.Errorf("line 0 = %q, want %q", lines[0], "tick-ready2")
		}
		if lines[1] != "tick-blk01" {
			t.Errorf("line 1 = %q, want %q", lines[1], "tick-blk01")
		}
	})

	t.Run("it maintains deterministic ordering", func(t *testing.T) {
		dir := setupTickDirWithContent(t, mixedContent())

		// Run twice and compare output
		var output1, output2 string

		for i, outputPtr := range []*string{&output1, &output2} {
			_ = i
			app := NewApp()
			app.workDir = dir
			var stdout strings.Builder
			app.stdout = &stdout

			err := app.Run([]string{"tick", "list", "--status", "open"})
			if err != nil {
				t.Fatalf("list run %d returned error: %v", i+1, err)
			}
			*outputPtr = stdout.String()
		}

		if output1 != output2 {
			t.Errorf("non-deterministic output:\nrun 1:\n%s\nrun 2:\n%s", output1, output2)
		}
	})
}
