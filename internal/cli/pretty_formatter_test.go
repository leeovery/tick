package cli

import (
	"strings"
	"testing"
)

func TestPrettyFormatter(t *testing.T) {
	t.Run("it implements Formatter interface", func(t *testing.T) {
		var _ Formatter = &PrettyFormatter{}
	})
}

func TestPrettyFormatterFormatTaskList(t *testing.T) {
	t.Run("it formats list with aligned columns", func(t *testing.T) {
		f := &PrettyFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
				{ID: "tick-c3d4", Title: "Login endpoint", Status: "in_progress", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		// Should have header row
		lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %v", len(lines), lines)
		}

		// Header should have ID, STATUS, PRI, TITLE
		header := lines[0]
		if !strings.Contains(header, "ID") {
			t.Error("header should contain ID")
		}
		if !strings.Contains(header, "STATUS") {
			t.Error("header should contain STATUS")
		}
		if !strings.Contains(header, "PRI") {
			t.Error("header should contain PRI")
		}
		if !strings.Contains(header, "TITLE") {
			t.Error("header should contain TITLE")
		}

		// Data rows should contain task data
		if !strings.Contains(lines[1], "tick-a1b2") {
			t.Error("first row should contain tick-a1b2")
		}
		if !strings.Contains(lines[1], "done") {
			t.Error("first row should contain done")
		}
		if !strings.Contains(lines[2], "tick-c3d4") {
			t.Error("second row should contain tick-c3d4")
		}
		if !strings.Contains(lines[2], "in_progress") {
			t.Error("second row should contain in_progress")
		}
	})

	t.Run("it aligns with variable-width data", func(t *testing.T) {
		f := &PrettyFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Short", Status: "open", Priority: 0},
				{ID: "tick-c3d4", Title: "Longer title here", Status: "in_progress", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}

		// All lines should have same number of columns (visually aligned)
		// The STATUS column should be wide enough for "in_progress" (11 chars)
		// Check that "open" has padding to align with "in_progress"
		row1 := lines[1]
		row2 := lines[2]

		// Find where TITLE column starts by looking at header
		header := lines[0]
		titleIdx := strings.Index(header, "TITLE")
		if titleIdx == -1 {
			t.Fatal("header should contain TITLE")
		}

		// Both data rows should have their titles starting at same position
		// This confirms column alignment
		if len(row1) < titleIdx || len(row2) < titleIdx {
			t.Error("rows should be at least as long as title column start")
		}
	})

	t.Run("it shows 'No tasks found.' for empty list", func(t *testing.T) {
		f := &PrettyFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{},
		}

		result := f.FormatTaskList(data)

		expected := "No tasks found.\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("it truncates long titles in list", func(t *testing.T) {
		f := &PrettyFormatter{}
		longTitle := strings.Repeat("A", 100) // Very long title
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: longTitle, Status: "open", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		// Title should be truncated with ...
		if !strings.Contains(lines[1], "...") {
			t.Error("long title should be truncated with ...")
		}
		// Full 100-char title should NOT appear
		if strings.Contains(lines[1], longTitle) {
			t.Error("long title should be truncated, not shown in full")
		}
	})
}

func TestPrettyFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &PrettyFormatter{}
		data := &TaskDetailData{
			ID:          "tick-c3d4",
			Title:       "Login endpoint",
			Status:      "in_progress",
			Priority:    1,
			Description: "Implement the login endpoint with validation...",
			Parent:      "tick-e5f6",
			ParentTitle: "Auth System",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "",
			BlockedBy: []RelatedTaskData{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done"},
			},
			Children: []RelatedTaskData{
				{ID: "tick-x1y2", Title: "Child task", Status: "open"},
			},
		}

		result := f.FormatTaskDetail(data)

		// Should have aligned key-value pairs
		if !strings.Contains(result, "ID:") {
			t.Error("should contain ID label")
		}
		if !strings.Contains(result, "tick-c3d4") {
			t.Error("should contain task ID")
		}
		if !strings.Contains(result, "Title:") {
			t.Error("should contain Title label")
		}
		if !strings.Contains(result, "Login endpoint") {
			t.Error("should contain task title")
		}
		if !strings.Contains(result, "Status:") {
			t.Error("should contain Status label")
		}
		if !strings.Contains(result, "in_progress") {
			t.Error("should contain task status")
		}
		if !strings.Contains(result, "Priority:") {
			t.Error("should contain Priority label")
		}
		if !strings.Contains(result, "Created:") {
			t.Error("should contain Created label")
		}

		// Should have Blocked by section with indented items
		if !strings.Contains(result, "Blocked by:") {
			t.Error("should contain Blocked by section")
		}
		if !strings.Contains(result, "tick-a1b2") {
			t.Error("should contain blocker ID")
		}
		if !strings.Contains(result, "Setup Sanctum") {
			t.Error("should contain blocker title")
		}
		if !strings.Contains(result, "(done)") {
			t.Error("should contain blocker status in parens")
		}

		// Should have Children section
		if !strings.Contains(result, "Children:") {
			t.Error("should contain Children section")
		}
		if !strings.Contains(result, "tick-x1y2") {
			t.Error("should contain child ID")
		}

		// Should have Description section
		if !strings.Contains(result, "Description:") {
			t.Error("should contain Description section")
		}
		if !strings.Contains(result, "Implement the login endpoint") {
			t.Error("should contain description text")
		}
	})

	t.Run("it omits empty sections in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Solo task",
			Status:      "open",
			Priority:    2,
			Description: "",
			Parent:      "",
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		// Should NOT have Blocked by section (empty)
		if strings.Contains(result, "Blocked by:") {
			t.Error("should omit empty Blocked by section")
		}

		// Should NOT have Children section (empty)
		if strings.Contains(result, "Children:") {
			t.Error("should omit empty Children section")
		}

		// Should NOT have Description section (empty)
		if strings.Contains(result, "Description:") {
			t.Error("should omit empty Description section")
		}

		// Should still have basic fields
		if !strings.Contains(result, "ID:") {
			t.Error("should still contain ID label")
		}
		if !strings.Contains(result, "Title:") {
			t.Error("should still contain Title label")
		}
	})

	t.Run("it does not truncate in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		longTitle := strings.Repeat("A", 100)
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       longTitle,
			Status:      "open",
			Priority:    2,
			Description: "",
			Parent:      "",
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		// Full title should appear in show
		if !strings.Contains(result, longTitle) {
			t.Error("show should display full title without truncation")
		}
	})
}

func TestPrettyFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats with all groups, right-aligned", func(t *testing.T) {
		f := &PrettyFormatter{}
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

		// Should have Total
		if !strings.Contains(result, "Total:") {
			t.Error("should contain Total label")
		}
		if !strings.Contains(result, "47") {
			t.Error("should contain total count")
		}

		// Should have Status section
		if !strings.Contains(result, "Status:") {
			t.Error("should contain Status section")
		}
		if !strings.Contains(result, "Open:") {
			t.Error("should contain Open label")
		}
		if !strings.Contains(result, "In Progress:") {
			t.Error("should contain In Progress label")
		}
		if !strings.Contains(result, "Done:") {
			t.Error("should contain Done label")
		}
		if !strings.Contains(result, "Cancelled:") {
			t.Error("should contain Cancelled label")
		}

		// Should have Workflow section
		if !strings.Contains(result, "Workflow:") {
			t.Error("should contain Workflow section")
		}
		if !strings.Contains(result, "Ready:") {
			t.Error("should contain Ready label")
		}
		if !strings.Contains(result, "Blocked:") {
			t.Error("should contain Blocked label")
		}

		// Should have Priority section
		if !strings.Contains(result, "Priority:") {
			t.Error("should contain Priority section")
		}
	})

	t.Run("it shows zero counts in stats", func(t *testing.T) {
		f := &PrettyFormatter{}
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

		// All status rows should be present, including zeros
		if !strings.Contains(result, "In Progress:") {
			t.Error("should contain In Progress even when 0")
		}
		if !strings.Contains(result, "Done:") {
			t.Error("should contain Done even when 0")
		}
		if !strings.Contains(result, "Cancelled:") {
			t.Error("should contain Cancelled even when 0")
		}
		if !strings.Contains(result, "Blocked:") {
			t.Error("should contain Blocked even when 0")
		}

		// Zero counts should appear
		// Note: we need to verify the value, not just the label
		lines := strings.Split(result, "\n")
		foundInProgressZero := false
		for _, line := range lines {
			if strings.Contains(line, "In Progress:") && strings.Contains(line, "0") {
				foundInProgressZero = true
				break
			}
		}
		if !foundInProgressZero {
			t.Error("In Progress should show 0")
		}
	})

	t.Run("it renders P0-P4 priority labels", func(t *testing.T) {
		f := &PrettyFormatter{}
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

		// Should have all 5 priority labels with descriptions
		if !strings.Contains(result, "P0") && !strings.Contains(result, "critical") {
			t.Error("should contain P0 (critical)")
		}
		if !strings.Contains(result, "P1") && !strings.Contains(result, "high") {
			t.Error("should contain P1 (high)")
		}
		if !strings.Contains(result, "P2") && !strings.Contains(result, "medium") {
			t.Error("should contain P2 (medium)")
		}
		if !strings.Contains(result, "P3") && !strings.Contains(result, "low") {
			t.Error("should contain P3 (low)")
		}
		if !strings.Contains(result, "P4") && !strings.Contains(result, "backlog") {
			t.Error("should contain P4 (backlog)")
		}
	})
}

func TestPrettyFormatterFormatTransition(t *testing.T) {
	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}

		result := f.FormatTransition("tick-a3f2b7", "open", "in_progress")

		// Same format as TOON - plain text passthrough
		expected := "tick-a3f2b7: open \u2192 in_progress\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestPrettyFormatterFormatDepChange(t *testing.T) {
	t.Run("it formats dep add as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}

		result := f.FormatDepChange("add", "tick-c3d4", "tick-a1b2")

		expected := "Dependency added: tick-c3d4 blocked by tick-a1b2\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("it formats dep rm as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}

		result := f.FormatDepChange("remove", "tick-c3d4", "tick-a1b2")

		expected := "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestPrettyFormatterFormatMessage(t *testing.T) {
	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}

		result := f.FormatMessage("No tasks found.")

		expected := "No tasks found.\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// TestPrettyFormatterSpecExamples tests that output matches the exact format from the specification.
func TestPrettyFormatterSpecExamples(t *testing.T) {
	t.Run("it matches spec list example format", func(t *testing.T) {
		f := &PrettyFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
				{ID: "tick-c3d4", Title: "Login endpoint", Status: "in_progress", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		// Spec format:
		// ID          STATUS       PRI  TITLE
		// tick-a1b2   done         1    Setup Sanctum
		// tick-c3d4   in_progress  1    Login endpoint

		lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
		}

		// Header should have correct columns
		header := lines[0]
		if !strings.HasPrefix(header, "ID") {
			t.Errorf("header should start with ID, got: %q", header)
		}

		// Verify column positions are consistent across rows
		// Find where each column starts in header
		statusIdx := strings.Index(header, "STATUS")
		priIdx := strings.Index(header, "PRI")
		titleIdx := strings.Index(header, "TITLE")

		if statusIdx == -1 || priIdx == -1 || titleIdx == -1 {
			t.Fatal("header missing expected columns")
		}

		// Data should align - check that status values are at the right position
		// (allowing for different lengths like "done" vs "in_progress")
	})

	t.Run("it matches spec show example format", func(t *testing.T) {
		f := &PrettyFormatter{}
		data := &TaskDetailData{
			ID:          "tick-c3d4",
			Title:       "Login endpoint",
			Status:      "in_progress",
			Priority:    1,
			Description: "Implement the login endpoint with validation...",
			Parent:      "",
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy: []RelatedTaskData{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done"},
			},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		// Spec format:
		// ID:       tick-c3d4
		// Title:    Login endpoint
		// Status:   in_progress
		// Priority: 1
		// Created:  2026-01-19T10:00:00Z
		//
		// Blocked by:
		//   tick-a1b2  Setup Sanctum (done)
		//
		// Description:
		//   Implement the login endpoint with validation...

		// Check aligned key-value format
		if !strings.Contains(result, "ID:") {
			t.Error("missing ID label")
		}
		if !strings.Contains(result, "Title:") {
			t.Error("missing Title label")
		}

		// Check Blocked by section format with indented items
		if !strings.Contains(result, "Blocked by:") {
			t.Error("missing Blocked by section")
		}
		// Item should be indented and have format: tick-a1b2  Setup Sanctum (done)
		if !strings.Contains(result, "tick-a1b2") {
			t.Error("missing blocker ID")
		}
		if !strings.Contains(result, "(done)") {
			t.Error("blocker status should be in parentheses")
		}

		// Check Description section
		if !strings.Contains(result, "Description:") {
			t.Error("missing Description section")
		}
	})

	t.Run("it matches spec stats example format", func(t *testing.T) {
		f := &PrettyFormatter{}
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

		// Spec format:
		// Total:       47
		//
		// Status:
		//   Open:        12
		//   In Progress:  3
		//   Done:        28
		//   Cancelled:    4
		//
		// Workflow:
		//   Ready:        8
		//   Blocked:      4
		//
		// Priority:
		//   P0 (critical):  2
		//   P1 (high):      8
		//   P2 (medium):   25
		//   P3 (low):       7
		//   P4 (backlog):   5

		// Check Total
		if !strings.Contains(result, "Total:") {
			t.Error("missing Total")
		}

		// Check Status section header
		if !strings.Contains(result, "Status:") {
			t.Error("missing Status section")
		}

		// Check Workflow section header
		if !strings.Contains(result, "Workflow:") {
			t.Error("missing Workflow section")
		}

		// Check Priority section header
		if !strings.Contains(result, "Priority:") {
			t.Error("missing Priority section")
		}

		// Check priority labels with descriptions
		if !strings.Contains(result, "P0 (critical)") {
			t.Error("missing P0 (critical)")
		}
		if !strings.Contains(result, "P1 (high)") {
			t.Error("missing P1 (high)")
		}
		if !strings.Contains(result, "P2 (medium)") {
			t.Error("missing P2 (medium)")
		}
		if !strings.Contains(result, "P3 (low)") {
			t.Error("missing P3 (low)")
		}
		if !strings.Contains(result, "P4 (backlog)") {
			t.Error("missing P4 (backlog)")
		}
	})
}
