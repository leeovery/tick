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

		err := app.Run([]string{"tick", "list"})
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

		err := app.Run([]string{"tick", "list"})
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
