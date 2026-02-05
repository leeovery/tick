package cli

import (
	"strings"
	"testing"
)

func TestToonFormatter(t *testing.T) {
	t.Run("it implements Formatter interface", func(t *testing.T) {
		var _ Formatter = &ToonFormatter{}
	})
}

func TestToonFormatterFormatTaskList(t *testing.T) {
	t.Run("it formats list with correct header count and schema", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
				{ID: "tick-c3d4", Title: "Login endpoint", Status: "open", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		// Should have header with count and schema
		if !strings.HasPrefix(result, "tasks[2]{id,title,status,priority}:") {
			t.Errorf("expected header 'tasks[2]{id,title,status,priority}:', got %q", strings.Split(result, "\n")[0])
		}

		// Should have indented data rows
		lines := strings.Split(result, "\n")
		if len(lines) < 3 {
			t.Fatalf("expected at least 3 lines, got %d", len(lines))
		}

		// Check first data row
		if !strings.HasPrefix(lines[1], "  tick-a1b2,") {
			t.Errorf("expected first data row to start with '  tick-a1b2,', got %q", lines[1])
		}
		if !strings.Contains(lines[1], "Setup Sanctum") {
			t.Errorf("expected first data row to contain 'Setup Sanctum', got %q", lines[1])
		}
		if !strings.Contains(lines[1], "done") {
			t.Errorf("expected first data row to contain 'done', got %q", lines[1])
		}
		if !strings.Contains(lines[1], ",1") {
			t.Errorf("expected first data row to contain priority ',1', got %q", lines[1])
		}

		// Check second data row
		if !strings.HasPrefix(lines[2], "  tick-c3d4,") {
			t.Errorf("expected second data row to start with '  tick-c3d4,', got %q", lines[2])
		}
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{},
		}

		result := f.FormatTaskList(data)

		// Should have header with zero count, schema present, no data rows
		expected := "tasks[0]{id,title,status,priority}:\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("it escapes commas in titles", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Fix bug, urgent", Status: "open", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		// Title with comma should be quoted by toon-go
		lines := strings.Split(result, "\n")
		if len(lines) < 2 {
			t.Fatalf("expected at least 2 lines, got %d", len(lines))
		}

		// The title should be escaped (quoted) by toon-go
		// Format: tick-a1b2,"Fix bug, urgent",open,1
		if !strings.Contains(lines[1], `"Fix bug, urgent"`) {
			t.Errorf("expected comma in title to be quoted, got %q", lines[1])
		}
	})
}

func TestToonFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Setup Sanctum",
			Status:      "in_progress",
			Priority:    1,
			Parent:      "tick-e5f6",
			ParentTitle: "Auth System",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "",
			BlockedBy: []RelatedTaskData{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
				{ID: "tick-g7h8", Title: "Config setup", Status: "in_progress"},
			},
			Children: []RelatedTaskData{
				{ID: "tick-j1k2", Title: "Child task", Status: "open"},
			},
			Description: "Full task description here.\nCan be multiple lines.",
		}

		result := f.FormatTaskDetail(data)

		// Should have task section with parent (since parent is set)
		if !strings.Contains(result, "task{") {
			t.Error("expected task section header")
		}
		if !strings.Contains(result, ",parent,") {
			t.Error("expected parent in schema when parent is set")
		}

		// Should have blocked_by section with count
		if !strings.Contains(result, "blocked_by[2]{id,title,status}:") {
			t.Error("expected blocked_by section with count 2")
		}

		// Should have children section with count
		if !strings.Contains(result, "children[1]{id,title,status}:") {
			t.Error("expected children section with count 1")
		}

		// Should have description section
		if !strings.Contains(result, "description:") {
			t.Error("expected description section")
		}
	})

	t.Run("it omits parent/closed from schema when null", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Root task",
			Status:      "open",
			Priority:    2,
			Parent:      "", // No parent
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "", // Not closed
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
			Description: "",
		}

		result := f.FormatTaskDetail(data)

		// Schema should NOT include parent or closed
		lines := strings.Split(result, "\n")
		taskLine := lines[0]
		if strings.Contains(taskLine, ",parent") {
			t.Errorf("schema should not include parent when null, got %q", taskLine)
		}
		if strings.Contains(taskLine, ",closed") {
			t.Errorf("schema should not include closed when null, got %q", taskLine)
		}

		// Should still have required fields
		if !strings.Contains(taskLine, "id") {
			t.Error("schema should include id")
		}
		if !strings.Contains(taskLine, "title") {
			t.Error("schema should include title")
		}
	})

	t.Run("it renders blocked_by/children with count 0 when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Solo task",
			Status:      "open",
			Priority:    2,
			Parent:      "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
			Description: "",
		}

		result := f.FormatTaskDetail(data)

		// blocked_by and children should ALWAYS be present, even with 0 count
		if !strings.Contains(result, "blocked_by[0]{id,title,status}:") {
			t.Errorf("expected blocked_by[0]{id,title,status}: section, result: %s", result)
		}
		if !strings.Contains(result, "children[0]{id,title,status}:") {
			t.Errorf("expected children[0]{id,title,status}: section, result: %s", result)
		}
	})

	t.Run("it omits description section when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "No description task",
			Status:      "open",
			Priority:    2,
			Parent:      "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
			Description: "",
		}

		result := f.FormatTaskDetail(data)

		// description section should be OMITTED when empty
		if strings.Contains(result, "description:") {
			t.Errorf("expected no description section when empty, result: %s", result)
		}
	})

	t.Run("it renders multiline description as indented lines", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Task with description",
			Status:      "open",
			Priority:    2,
			Parent:      "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
			Description: "First line.\nSecond line.",
		}

		result := f.FormatTaskDetail(data)

		// description section should have indented lines
		if !strings.Contains(result, "description:") {
			t.Error("expected description section")
		}
		// Each line should be indented with 2 spaces
		if !strings.Contains(result, "\n  First line.\n") {
			t.Errorf("expected indented 'First line.', result: %s", result)
		}
		if !strings.Contains(result, "\n  Second line.") {
			t.Errorf("expected indented 'Second line.', result: %s", result)
		}
	})

	t.Run("it includes parent in schema when set", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Child task",
			Status:      "open",
			Priority:    2,
			Parent:      "tick-e5f6",
			ParentTitle: "Parent task",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
			Description: "",
		}

		result := f.FormatTaskDetail(data)

		// Schema should include parent
		lines := strings.Split(result, "\n")
		taskLine := lines[0]
		if !strings.Contains(taskLine, ",parent,") && !strings.HasSuffix(strings.TrimSuffix(taskLine, ":"), ",parent}") {
			t.Errorf("schema should include parent when set, got %q", taskLine)
		}
	})

	t.Run("it includes closed in schema when set", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Done task",
			Status:      "done",
			Priority:    2,
			Parent:      "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T16:00:00Z",
			Closed:      "2026-01-19T16:00:00Z",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
			Description: "",
		}

		result := f.FormatTaskDetail(data)

		// Schema should include closed
		lines := strings.Split(result, "\n")
		taskLine := lines[0]
		if !strings.Contains(taskLine, "closed") {
			t.Errorf("schema should include closed when set, got %q", taskLine)
		}
	})
}

func TestToonFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats with all counts", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &StatsData{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: []PriorityCount{
				{Priority: 0, Count: 2},
				{Priority: 1, Count: 8},
				{Priority: 2, Count: 25},
				{Priority: 3, Count: 7},
				{Priority: 4, Count: 5},
			},
		}

		result := f.FormatStats(data)

		// Should have stats section with schema
		if !strings.Contains(result, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("expected stats header, got %s", result)
		}

		// Should have stats data row
		if !strings.Contains(result, "47,12,3,28,4,8,4") {
			t.Errorf("expected stats data '47,12,3,28,4,8,4', got %s", result)
		}

		// Should have by_priority section
		if !strings.Contains(result, "by_priority[5]{priority,count}:") {
			t.Errorf("expected by_priority header, got %s", result)
		}
	})

	t.Run("it formats by_priority with 5 rows including zeros", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &StatsData{
			Total:      10,
			Open:       10,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      10,
			Blocked:    0,
			ByPriority: []PriorityCount{
				{Priority: 0, Count: 0},
				{Priority: 1, Count: 0},
				{Priority: 2, Count: 10},
				{Priority: 3, Count: 0},
				{Priority: 4, Count: 0},
			},
		}

		result := f.FormatStats(data)

		// Should have exactly 5 priority rows
		if !strings.Contains(result, "by_priority[5]{priority,count}:") {
			t.Errorf("expected by_priority[5] header, got %s", result)
		}

		// Should have all 5 rows, including zeros
		lines := strings.Split(result, "\n")
		var priorityLines []string
		inPriority := false
		for _, line := range lines {
			if strings.Contains(line, "by_priority") {
				inPriority = true
				continue
			}
			if inPriority && strings.HasPrefix(line, "  ") {
				priorityLines = append(priorityLines, line)
			}
		}

		if len(priorityLines) != 5 {
			t.Errorf("expected 5 priority rows, got %d: %v", len(priorityLines), priorityLines)
		}

		// Check that zeros are included
		if !strings.Contains(result, "0,0") {
			t.Errorf("expected priority rows with zero counts, got %s", result)
		}
	})
}

func TestToonFormatterFormatTransition(t *testing.T) {
	t.Run("it formats transition/dep as plain text", func(t *testing.T) {
		f := &ToonFormatter{}

		result := f.FormatTransition("tick-a3f2b7", "open", "in_progress")

		// Should be plain text, same as human-readable
		expected := "tick-a3f2b7: open \u2192 in_progress\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestToonFormatterFormatDepChange(t *testing.T) {
	t.Run("it formats dep add as plain text", func(t *testing.T) {
		f := &ToonFormatter{}

		result := f.FormatDepChange("add", "tick-c3d4", "tick-a1b2")

		expected := "Dependency added: tick-c3d4 blocked by tick-a1b2\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("it formats dep rm as plain text", func(t *testing.T) {
		f := &ToonFormatter{}

		result := f.FormatDepChange("remove", "tick-c3d4", "tick-a1b2")

		expected := "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestToonFormatterFormatMessage(t *testing.T) {
	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &ToonFormatter{}

		result := f.FormatMessage("No tasks found.")

		expected := "No tasks found.\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// TestToonFormatterSpecExamples tests that output matches the exact format from the specification.
func TestToonFormatterSpecExamples(t *testing.T) {
	t.Run("it matches spec list example format", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
				{ID: "tick-c3d4", Title: "Login endpoint", Status: "open", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		// Verify structure matches spec:
		// tasks[2]{id,title,status,priority}:
		//   tick-a1b2,Setup Sanctum,done,1
		//   tick-c3d4,Login endpoint,open,1
		lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
		}
		if lines[0] != "tasks[2]{id,title,status,priority}:" {
			t.Errorf("header mismatch: %q", lines[0])
		}
		if lines[1] != "  tick-a1b2,Setup Sanctum,done,1" {
			t.Errorf("row 1 mismatch: %q", lines[1])
		}
		if lines[2] != "  tick-c3d4,Login endpoint,open,1" {
			t.Errorf("row 2 mismatch: %q", lines[2])
		}
	})

	t.Run("it matches spec show example format", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Setup Sanctum",
			Status:      "in_progress",
			Priority:    1,
			Parent:      "tick-e5f6",
			ParentTitle: "Auth System",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "",
			BlockedBy: []RelatedTaskData{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
				{ID: "tick-g7h8", Title: "Config setup", Status: "in_progress"},
			},
			Children:    []RelatedTaskData{},
			Description: "Full task description here.\nCan be multiple lines.",
		}

		result := f.FormatTaskDetail(data)

		// Verify key parts of spec format:
		// task{id,title,status,priority,parent,created,updated}:
		//   tick-a1b2,Setup Sanctum,in_progress,1,tick-e5f6,2026-01-19T10:00:00Z,2026-01-19T14:30:00Z
		//
		// blocked_by[2]{id,title,status}:
		//   tick-c3d4,Database migrations,done
		//   tick-g7h8,Config setup,in_progress
		//
		// children[0]{id,title,status}:
		//
		// description:
		//   Full task description here.
		//   Can be multiple lines.
		if !strings.Contains(result, "task{id,title,status,priority,parent,created,updated}:") {
			t.Error("missing task schema with parent")
		}
		if !strings.Contains(result, "  tick-a1b2,Setup Sanctum,in_progress,1,tick-e5f6,") {
			t.Error("missing task data row with parent")
		}
		if !strings.Contains(result, "blocked_by[2]{id,title,status}:") {
			t.Error("missing blocked_by section")
		}
		if !strings.Contains(result, "  tick-c3d4,Database migrations,done") {
			t.Error("missing blocker row")
		}
		if !strings.Contains(result, "children[0]{id,title,status}:") {
			t.Error("missing children section")
		}
		if !strings.Contains(result, "description:") {
			t.Error("missing description section")
		}
		if !strings.Contains(result, "  Full task description here.") {
			t.Error("missing description first line")
		}
		if !strings.Contains(result, "  Can be multiple lines.") {
			t.Error("missing description second line")
		}
	})

	t.Run("it matches spec stats example format", func(t *testing.T) {
		f := &ToonFormatter{}
		data := &StatsData{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: []PriorityCount{
				{Priority: 0, Count: 2},
				{Priority: 1, Count: 8},
				{Priority: 2, Count: 25},
				{Priority: 3, Count: 7},
				{Priority: 4, Count: 5},
			},
		}

		result := f.FormatStats(data)

		// Verify structure matches spec:
		// stats{total,open,in_progress,done,cancelled,ready,blocked}:
		//   47,12,3,28,4,8,4
		//
		// by_priority[5]{priority,count}:
		//   0,2
		//   1,8
		//   2,25
		//   3,7
		//   4,5
		if !strings.Contains(result, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Error("missing stats schema")
		}
		if !strings.Contains(result, "  47,12,3,28,4,8,4") {
			t.Error("missing stats data")
		}
		if !strings.Contains(result, "by_priority[5]{priority,count}:") {
			t.Error("missing by_priority header")
		}
		// Check all 5 priority rows
		if !strings.Contains(result, "  0,2") {
			t.Error("missing priority 0 row")
		}
		if !strings.Contains(result, "  1,8") {
			t.Error("missing priority 1 row")
		}
		if !strings.Contains(result, "  2,25") {
			t.Error("missing priority 2 row")
		}
		if !strings.Contains(result, "  3,7") {
			t.Error("missing priority 3 row")
		}
		if !strings.Contains(result, "  4,5") {
			t.Error("missing priority 4 row")
		}
	})
}
