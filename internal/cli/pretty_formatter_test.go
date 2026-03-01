package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestPrettyFormatter(t *testing.T) {
	// Compile-time interface verification.
	var _ Formatter = (*PrettyFormatter)(nil)

	t.Run("it formats list with aligned columns", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: task.StatusInProgress, Priority: 1, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		expected := "" +
			"ID          STATUS       PRI  TYPE  TITLE\n" +
			"tick-a1b2   done         1    -     Setup Sanctum\n" +
			"tick-c3d4   in_progress  1    -     Login endpoint"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it aligns with variable-width data", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2c3", Title: "A short task", Status: task.StatusOpen, Priority: 0, Created: now, Updated: now},
			{ID: "tick-d4", Title: "Another task here", Status: task.StatusInProgress, Priority: 3, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), result)
		}
		// Dynamic column widths: ID col should fit longest ID + padding,
		// STATUS col should fit "in_progress" (11 chars) + padding
		// All columns should be aligned
		// Check header and rows have consistent alignment
		headerIDEnd := strings.Index(lines[0], "STATUS")
		row1IDEnd := strings.Index(lines[1], "open")
		row2IDEnd := strings.Index(lines[2], "in_progress")
		if headerIDEnd != row1IDEnd || headerIDEnd != row2IDEnd {
			t.Errorf("STATUS column not aligned:\nheader STATUS at %d\nrow1 status at %d\nrow2 status at %d\n%s",
				headerIDEnd, row1IDEnd, row2IDEnd, result)
		}

		headerPRIStart := strings.Index(lines[0], "PRI")
		row1PRIStart := strings.Index(lines[1], "0")
		row2PRIStart := strings.Index(lines[2], "3")
		if headerPRIStart != row1PRIStart || headerPRIStart != row2PRIStart {
			t.Errorf("PRI column not aligned:\nheader PRI at %d\nrow1 pri at %d\nrow2 pri at %d\n%s",
				headerPRIStart, row1PRIStart, row2PRIStart, result)
		}
	})

	t.Run("it shows No tasks found for empty list", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatTaskList([]task.Task{})
		expected := "No tasks found."
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it shows No tasks found for nil list", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatTaskList(nil)
		expected := "No tasks found."
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:          "tick-c3d4",
				Title:       "Login endpoint",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: "Implement the login endpoint with validation...",
				Parent:      "tick-e5f6",
				Created:     now,
				Updated:     updated,
			},
			BlockedBy: []RelatedTask{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done"},
			},
			Children: []RelatedTask{
				{ID: "tick-g7h8", Title: "Validation logic", Status: "open"},
			},
			ParentTitle: "Auth System",
		}
		result := f.FormatTaskDetail(detail)
		expected := "" +
			"ID:       tick-c3d4\n" +
			"Title:    Login endpoint\n" +
			"Status:   in_progress\n" +
			"Priority: 1\n" +
			"Type:     -\n" +
			"Parent:   tick-e5f6 (Auth System)\n" +
			"Created:  2026-01-19T10:00:00Z\n" +
			"Updated:  2026-01-19T14:30:00Z\n" +
			"\n" +
			"Blocked by:\n" +
			"  tick-a1b2  Setup Sanctum (done)\n" +
			"\n" +
			"Children:\n" +
			"  tick-g7h8  Validation logic (open)\n" +
			"\n" +
			"Description:\n" +
			"  Implement the login endpoint with validation..."
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it omits empty sections in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Simple task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		expected := "" +
			"ID:       tick-a1b2\n" +
			"Title:    Simple task\n" +
			"Status:   open\n" +
			"Priority: 2\n" +
			"Type:     -\n" +
			"Created:  2026-01-19T10:00:00Z\n" +
			"Updated:  2026-01-19T10:00:00Z"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
		// Must not contain section headers for empty sections
		if strings.Contains(result, "Blocked by:") {
			t.Error("should not contain Blocked by section when empty")
		}
		if strings.Contains(result, "Children:") {
			t.Error("should not contain Children section when empty")
		}
		if strings.Contains(result, "Description:") {
			t.Error("should not contain Description section when empty")
		}
	})

	t.Run("it includes closed timestamp when present in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Done task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  now,
				Updated:  now,
				Closed:   &closed,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if !strings.Contains(result, "Closed:   2026-01-19T16:00:00Z") {
			t.Errorf("should contain Closed timestamp, got:\n%s", result)
		}
	})

	t.Run("it formats stats with all groups right-aligned", func(t *testing.T) {
		f := &PrettyFormatter{}
		stats := Stats{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}
		result := f.FormatStats(stats)
		expected := "" +
			"Total:       47\n" +
			"\n" +
			"Status:\n" +
			"  Open:        12\n" +
			"  In Progress:  3\n" +
			"  Done:        28\n" +
			"  Cancelled:    4\n" +
			"\n" +
			"Workflow:\n" +
			"  Ready:        8\n" +
			"  Blocked:      4\n" +
			"\n" +
			"Priority:\n" +
			"  P0 (critical):  2\n" +
			"  P1 (high):      8\n" +
			"  P2 (medium):   25\n" +
			"  P3 (low):       7\n" +
			"  P4 (backlog):   5"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it shows zero counts in stats", func(t *testing.T) {
		f := &PrettyFormatter{}
		stats := Stats{
			Total:      0,
			Open:       0,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      0,
			Blocked:    0,
			ByPriority: [5]int{0, 0, 0, 0, 0},
		}
		result := f.FormatStats(stats)
		expected := "" +
			"Total:        0\n" +
			"\n" +
			"Status:\n" +
			"  Open:         0\n" +
			"  In Progress:  0\n" +
			"  Done:         0\n" +
			"  Cancelled:    0\n" +
			"\n" +
			"Workflow:\n" +
			"  Ready:        0\n" +
			"  Blocked:      0\n" +
			"\n" +
			"Priority:\n" +
			"  P0 (critical):  0\n" +
			"  P1 (high):      0\n" +
			"  P2 (medium):    0\n" +
			"  P3 (low):       0\n" +
			"  P4 (backlog):   0"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it renders P0-P4 priority labels", func(t *testing.T) {
		f := &PrettyFormatter{}
		stats := Stats{
			Total:      15,
			Open:       15,
			ByPriority: [5]int{1, 2, 3, 4, 5},
		}
		result := f.FormatStats(stats)
		requiredLabels := []string{
			"P0 (critical):",
			"P1 (high):",
			"P2 (medium):",
			"P3 (low):",
			"P4 (backlog):",
		}
		for _, label := range requiredLabels {
			if !strings.Contains(result, label) {
				t.Errorf("missing priority label %q in:\n%s", label, result)
			}
		}
	})

	t.Run("it truncates long titles in list", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		longTitle := "This is a very long title that exceeds the maximum display width for list output"
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: longTitle, Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), result)
		}
		// Title in list should be truncated with "..."
		if !strings.HasSuffix(lines[1], "...") {
			t.Errorf("long title should be truncated with ..., got: %q", lines[1])
		}
		// Title in list should not exceed 50 chars
		// Find where title starts (after PRI column)
		headerTitleStart := strings.Index(lines[0], "TITLE")
		titleContent := lines[1][headerTitleStart:]
		if len(titleContent) > 50 {
			t.Errorf("truncated title should not exceed 50 chars, got %d: %q", len(titleContent), titleContent)
		}
	})

	t.Run("it does not truncate in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		longTitle := "This is a very long title that exceeds the maximum display width for list output"
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    longTitle,
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if !strings.Contains(result, longTitle) {
			t.Errorf("show should contain full title %q, got:\n%s", longTitle, result)
		}
	})

	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatTransition("tick-a3f2b7", "open", "in_progress")
		expected := "tick-a3f2b7: open \u2192 in_progress"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats dep change as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		resultAdd := f.FormatDepChange("added", "tick-c3d4", "tick-a1b2")
		expectedAdd := "Dependency added: tick-c3d4 blocked by tick-a1b2"
		if resultAdd != expectedAdd {
			t.Errorf("add result = %q, want %q", resultAdd, expectedAdd)
		}
		resultRm := f.FormatDepChange("removed", "tick-c3d4", "tick-a1b2")
		expectedRm := "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2"
		if resultRm != expectedRm {
			t.Errorf("rm result = %q, want %q", resultRm, expectedRm)
		}
	})

	t.Run("it formats single task removal via baseFormatter", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "My task"},
			},
		})
		expected := `Removed tick-a1b2 "My task"`
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it shows type column in pretty list output", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Fix login bug", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Add search", Status: task.StatusInProgress, Priority: 2, Type: "feature", Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), result)
		}
		// Header must have TYPE column between PRI and TITLE
		header := lines[0]
		priIdx := strings.Index(header, "PRI")
		typeIdx := strings.Index(header, "TYPE")
		titleIdx := strings.Index(header, "TITLE")
		if typeIdx < 0 {
			t.Fatalf("TYPE column not found in header: %q", header)
		}
		if typeIdx <= priIdx {
			t.Errorf("TYPE should come after PRI: PRI at %d, TYPE at %d", priIdx, typeIdx)
		}
		if typeIdx >= titleIdx {
			t.Errorf("TYPE should come before TITLE: TYPE at %d, TITLE at %d", typeIdx, titleIdx)
		}
		// Rows must contain type values
		if !strings.Contains(lines[1], "bug") {
			t.Errorf("row 1 should contain type 'bug': %q", lines[1])
		}
		if !strings.Contains(lines[2], "feature") {
			t.Errorf("row 2 should contain type 'feature': %q", lines[2])
		}
	})

	t.Run("it shows dash for unset type in pretty list", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "No type task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), result)
		}
		// The TYPE column should show "-" for unset type
		header := lines[0]
		typeIdx := strings.Index(header, "TYPE")
		if typeIdx < 0 {
			t.Fatalf("TYPE column not found in header: %q", header)
		}
		// Check that the row contains a dash in the type column area
		row := lines[1]
		// Extract the type value from the row at the same position as TYPE in header
		// The type value starts at typeIdx and ends before TITLE starts
		titleIdx := strings.Index(header, "TITLE")
		typeVal := strings.TrimSpace(row[typeIdx:titleIdx])
		if typeVal != "-" {
			t.Errorf("type column for unset type = %q, want %q", typeVal, "-")
		}
	})

	t.Run("it shows type in pretty show output", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Fix login bug",
				Status:   task.StatusOpen,
				Priority: 1,
				Type:     "bug",
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if !strings.Contains(result, "Type:     bug") {
			t.Errorf("show output should contain 'Type:     bug', got:\n%s", result)
		}
		// Type should appear after Priority
		priIdx := strings.Index(result, "Priority:")
		typeIdx := strings.Index(result, "Type:")
		if typeIdx < priIdx {
			t.Errorf("Type should appear after Priority: Priority at %d, Type at %d", priIdx, typeIdx)
		}
	})

	t.Run("it shows dash for unset type in pretty show output", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No type task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if !strings.Contains(result, "Type:     -") {
			t.Errorf("show output should contain 'Type:     -' for unset type, got:\n%s", result)
		}
	})

	t.Run("it displays tags in pretty format show output", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Tagged task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			Tags:      []string{"backend", "ui", "urgent"},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if !strings.Contains(result, "Tags:     backend, ui, urgent") {
			t.Errorf("should contain 'Tags:     backend, ui, urgent', got:\n%s", result)
		}
		// Tags should appear after Type
		typeIdx := strings.Index(result, "Type:")
		tagsIdx := strings.Index(result, "Tags:")
		if tagsIdx < typeIdx {
			t.Errorf("Tags should appear after Type: Type at %d, Tags at %d", typeIdx, tagsIdx)
		}
	})

	t.Run("it omits tags section in pretty format when task has no tags", func(t *testing.T) {
		f := &PrettyFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No tags task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if strings.Contains(result, "Tags:") {
			t.Errorf("should not contain Tags section when empty, got:\n%s", result)
		}
	})

	t.Run("it formats message as plain text passthrough", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatMessage("Tick initialized in /path/to/project")
		expected := "Tick initialized in /path/to/project"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})
}
