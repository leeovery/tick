package cli

import (
	"strings"
	"testing"
)

// taskJSONL returns a JSONL line for a task with the given fields.
// Helper for building complex ready-query test scenarios.
func taskJSONL(id, title, status string, priority int, blockedBy []string, parent string, created string) string {
	line := `{"id":"` + id + `","title":"` + title + `","status":"` + status + `","priority":` + itoa(priority)
	if len(blockedBy) > 0 {
		line += `,"blocked_by":[`
		for i, b := range blockedBy {
			if i > 0 {
				line += ","
			}
			line += `"` + b + `"`
		}
		line += `]`
	}
	if parent != "" {
		line += `,"parent":"` + parent + `"`
	}
	line += `,"created":"` + created + `","updated":"` + created + `"`
	if status == "done" || status == "cancelled" {
		line += `,"closed":"` + created + `"`
	}
	line += `}`
	return line
}

// itoa converts an int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func TestReadyQuery(t *testing.T) {
	t.Run("it returns open task with no blockers and no children", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "Simple task", "open", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("output = %q, want it to contain 'tick-aaa111'", output)
		}
		if !strings.Contains(output, "Simple task") {
			t.Errorf("output = %q, want it to contain 'Simple task'", output)
		}
	})

	t.Run("it excludes task with open blocker", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-blocked", "Blocked task", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-blocker") {
			t.Errorf("output should contain tick-blocker (unblocked open task), got: %q", output)
		}
		if strings.Contains(output, "tick-blocked") {
			t.Errorf("output should NOT contain tick-blocked (has open blocker), got: %q", output)
		}
	})

	t.Run("it excludes task with in_progress blocker", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "in_progress", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-blocked", "Blocked task", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "tick-blocked") {
			t.Errorf("output should NOT contain tick-blocked (has in_progress blocker), got: %q", output)
		}
	})

	t.Run("it includes task when all blockers done or cancelled", func(t *testing.T) {
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

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-task01") {
			t.Errorf("output should contain tick-task01 (all blockers done/cancelled), got: %q", output)
		}
	})

	t.Run("it excludes parent with open children", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-parent", "Parent task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-child1", "Child task", "open", 2, nil, "tick-parent", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "tick-parent") {
			t.Errorf("output should NOT contain tick-parent (has open children), got: %q", output)
		}
		if !strings.Contains(output, "tick-child1") {
			t.Errorf("output should contain tick-child1 (leaf open task), got: %q", output)
		}
	})

	t.Run("it excludes parent with in_progress children", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-parent", "Parent task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-child1", "Child task", "in_progress", 2, nil, "tick-parent", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "tick-parent") {
			t.Errorf("output should NOT contain tick-parent (has in_progress children), got: %q", output)
		}
	})

	t.Run("it includes parent when all children closed", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-parent", "Parent task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-child1", "Done child", "done", 2, nil, "tick-parent", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-child2", "Cancelled child", "cancelled", 2, nil, "tick-parent", "2026-01-19T12:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-parent") {
			t.Errorf("output should contain tick-parent (all children closed), got: %q", output)
		}
	})

	t.Run("it excludes in_progress tasks", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "In progress task", "in_progress", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want 'No tasks found.' (in_progress excluded from ready)", output)
		}
	})

	t.Run("it excludes done tasks", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "Done task", "done", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want 'No tasks found.' (done excluded from ready)", output)
		}
	})

	t.Run("it excludes cancelled tasks", func(t *testing.T) {
		content := taskJSONL("tick-aaa111", "Cancelled task", "cancelled", 2, nil, "", "2026-01-19T10:00:00Z") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want 'No tasks found.' (cancelled excluded from ready)", output)
		}
	})

	t.Run("it handles deep nesting -- only deepest incomplete ready", func(t *testing.T) {
		// Grandparent -> Parent -> Child (open leaf)
		// Only the child should appear in ready
		content := strings.Join([]string{
			taskJSONL("tick-grand1", "Grandparent", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-par001", "Parent", "open", 2, nil, "tick-grand1", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-child1", "Leaf child", "open", 2, nil, "tick-par001", "2026-01-19T12:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "tick-grand1") {
			t.Errorf("output should NOT contain tick-grand1 (has open child), got: %q", output)
		}
		if strings.Contains(output, "tick-par001") {
			t.Errorf("output should NOT contain tick-par001 (has open child), got: %q", output)
		}
		if !strings.Contains(output, "tick-child1") {
			t.Errorf("output should contain tick-child1 (deepest leaf), got: %q", output)
		}
	})

	t.Run("it returns empty list when no tasks ready", func(t *testing.T) {
		// All tasks are blocked
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Blocker", "in_progress", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Blocked", "open", 2, []string{"tick-aaa111"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want 'No tasks found.'", output)
		}
	})

	t.Run("it orders by priority ASC then created ASC", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-low001", "Low priority", "open", 3, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-hi0001", "High priority early", "open", 1, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-hi0002", "High priority late", "open", 1, nil, "", "2026-01-19T12:00:00Z"),
			taskJSONL("tick-med001", "Medium priority", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

		// Should have header + 4 data rows
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d:\n%s", len(lines), output)
		}

		// Priority 1 first (two tasks), then priority 2, then priority 3
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

	t.Run("it outputs aligned columns via tick ready", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Setup Sanctum", "open", 1, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Login endpoint", "open", 2, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
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

		// Verify column alignment: STATUS column starts at same position
		headerStatusPos := strings.Index(header, "STATUS")
		row1StatusPos := strings.Index(lines[1], "open")
		row2StatusPos := strings.Index(lines[2], "open")
		if headerStatusPos != row1StatusPos || headerStatusPos != row2StatusPos {
			t.Errorf("columns not aligned: header STATUS at %d, row1 status at %d, row2 status at %d",
				headerStatusPos, row1StatusPos, row2StatusPos)
		}

		// Verify data content
		if !strings.Contains(lines[1], "tick-aaa111") {
			t.Errorf("row 1 = %q, want it to contain 'tick-aaa111'", lines[1])
		}
		if !strings.Contains(lines[2], "tick-bbb222") {
			t.Errorf("row 2 = %q, want it to contain 'tick-bbb222'", lines[2])
		}
	})

	t.Run("it prints 'No tasks found.' when empty", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("output = %q, want %q", output, "No tasks found.")
		}
	})

	t.Run("it outputs IDs only with --quiet", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Task one", "open", 1, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Task two", "open", 2, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "ready"})
		if err != nil {
			t.Fatalf("ready returned error: %v", err)
		}

		output := strings.TrimRight(stdout.String(), "\n")
		lines := strings.Split(output, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per task), got %d:\n%q", len(lines), output)
		}

		if lines[0] != "tick-aaa111" {
			t.Errorf("line 0 = %q, want %q", lines[0], "tick-aaa111")
		}
		if lines[1] != "tick-bbb222" {
			t.Errorf("line 1 = %q, want %q", lines[1], "tick-bbb222")
		}
	})
}

func TestReadyViaListFlag(t *testing.T) {
	t.Run("it works via list --ready flag as well", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-blocker", "Blocker", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-blocked", "Blocked task", "open", 2, []string{"tick-blocker"}, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list", "--ready"})
		if err != nil {
			t.Fatalf("list --ready returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-blocker") {
			t.Errorf("output should contain tick-blocker (unblocked), got: %q", output)
		}
		if strings.Contains(output, "tick-blocked") {
			t.Errorf("output should NOT contain tick-blocked (blocked), got: %q", output)
		}
	})
}
