package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrettyFormatter_ImplementsFormatter(t *testing.T) {
	t.Run("it implements the full Formatter interface", func(t *testing.T) {
		var _ Formatter = &PrettyFormatter{}
	})
}

func TestPrettyFormatter_FormatTaskList(t *testing.T) {
	t.Run("it formats list with aligned columns", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		rows := []listRow{
			{ID: "tick-a1b2", Status: "done", Priority: 1, Title: "Setup Sanctum"},
			{ID: "tick-c3d4", Status: "in_progress", Priority: 1, Title: "Login endpoint"},
		}

		err := f.FormatTaskList(&buf, rows, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

		// Should have header + 2 data rows
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), got)
		}

		// Header must contain all column names
		header := lines[0]
		for _, col := range []string{"ID", "STATUS", "PRI", "TITLE"} {
			if !strings.Contains(header, col) {
				t.Errorf("header missing %q, got %q", col, header)
			}
		}

		// Data rows must contain task data
		if !strings.Contains(lines[1], "tick-a1b2") || !strings.Contains(lines[1], "done") || !strings.Contains(lines[1], "Setup Sanctum") {
			t.Errorf("unexpected first row: %q", lines[1])
		}
		if !strings.Contains(lines[2], "tick-c3d4") || !strings.Contains(lines[2], "in_progress") || !strings.Contains(lines[2], "Login endpoint") {
			t.Errorf("unexpected second row: %q", lines[2])
		}
	})

	t.Run("it aligns with variable-width data", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		rows := []listRow{
			{ID: "tick-a1b2c3", Status: "in_progress", Priority: 1, Title: "A long task title here"},
			{ID: "tick-d4", Status: "open", Priority: 0, Title: "Short"},
		}

		err := f.FormatTaskList(&buf, rows, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), got)
		}

		// All TITLE values should start at the same column position
		headerTitleIdx := strings.Index(lines[0], "TITLE")
		row1TitleIdx := strings.Index(lines[1], "A long task title here")
		row2TitleIdx := strings.Index(lines[2], "Short")

		if headerTitleIdx < 0 || row1TitleIdx < 0 || row2TitleIdx < 0 {
			t.Fatalf("could not find TITLE column values in output:\n%s", got)
		}

		if headerTitleIdx != row1TitleIdx || headerTitleIdx != row2TitleIdx {
			t.Errorf("TITLE columns not aligned: header=%d, row1=%d, row2=%d\n%s",
				headerTitleIdx, row1TitleIdx, row2TitleIdx, got)
		}

		// STATUS column should be dynamically sized to fit "in_progress" (11 chars)
		// "open" should be padded to match
		headerStatusIdx := strings.Index(lines[0], "STATUS")
		row1StatusIdx := strings.Index(lines[1], "in_progress")
		row2StatusIdx := strings.Index(lines[2], "open")

		if headerStatusIdx != row1StatusIdx || headerStatusIdx != row2StatusIdx {
			t.Errorf("STATUS columns not aligned: header=%d, row1=%d, row2=%d\n%s",
				headerStatusIdx, row1StatusIdx, row2StatusIdx, got)
		}
	})

	t.Run("it shows 'No tasks found.' for empty list", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := strings.TrimSpace(buf.String())
		if got != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", got)
		}

		// Should NOT contain header columns
		if strings.Contains(buf.String(), "ID") || strings.Contains(buf.String(), "STATUS") {
			t.Errorf("empty list should have no headers, got:\n%s", buf.String())
		}
	})

	t.Run("it truncates long titles in list", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		longTitle := strings.Repeat("A", 100)
		rows := []listRow{
			{ID: "tick-a1b2", Status: "open", Priority: 2, Title: longTitle},
		}

		err := f.FormatTaskList(&buf, rows, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Title should be truncated with ...
		if !strings.Contains(got, "...") {
			t.Errorf("expected long title to be truncated with '...', got:\n%s", got)
		}

		// Full title should NOT appear
		if strings.Contains(got, longTitle) {
			t.Errorf("expected long title to be truncated, but full title found:\n%s", got)
		}
	})

	t.Run("it outputs only IDs in quiet mode", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		rows := []listRow{
			{ID: "tick-a1b2", Status: "open", Priority: 1, Title: "Task one"},
			{ID: "tick-c3d4", Status: "done", Priority: 2, Title: "Task two"},
		}

		err := f.FormatTaskList(&buf, rows, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := strings.TrimSpace(buf.String())
		lines := strings.Split(got, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), got)
		}
		if lines[0] != "tick-a1b2" || lines[1] != "tick-c3d4" {
			t.Errorf("expected IDs only, got:\n%s", got)
		}
	})
}

func TestPrettyFormatter_FormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:          "tick-c3d4",
			Title:       "Login endpoint",
			Status:      "in_progress",
			Priority:    1,
			Parent:      "tick-e5f6",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "2026-01-19T16:00:00Z",
			Description: "Implement the login endpoint with validation...",
			BlockedBy: []RelatedTask{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done"},
			},
			Children: []RelatedTask{
				{ID: "tick-x1y2", Title: "Subtask one", Status: "open"},
			},
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Check key-value pairs with aligned labels
		expectedFields := []string{
			"ID:",
			"Title:",
			"Status:",
			"Priority:",
			"Parent:",
			"Created:",
			"Updated:",
			"Closed:",
		}
		for _, field := range expectedFields {
			if !strings.Contains(got, field) {
				t.Errorf("missing field %q in output:\n%s", field, got)
			}
		}

		// Check values are present
		for _, val := range []string{"tick-c3d4", "Login endpoint", "in_progress", "tick-e5f6"} {
			if !strings.Contains(got, val) {
				t.Errorf("missing value %q in output:\n%s", val, got)
			}
		}

		// Check blocked by section
		if !strings.Contains(got, "Blocked by:") {
			t.Errorf("missing 'Blocked by:' section:\n%s", got)
		}
		if !strings.Contains(got, "tick-a1b2") || !strings.Contains(got, "Setup Sanctum") {
			t.Errorf("missing blocked by details:\n%s", got)
		}

		// Check children section
		if !strings.Contains(got, "Children:") {
			t.Errorf("missing 'Children:' section:\n%s", got)
		}
		if !strings.Contains(got, "tick-x1y2") || !strings.Contains(got, "Subtask one") {
			t.Errorf("missing children details:\n%s", got)
		}

		// Check description section
		if !strings.Contains(got, "Description:") {
			t.Errorf("missing 'Description:' section:\n%s", got)
		}
		if !strings.Contains(got, "Implement the login endpoint") {
			t.Errorf("missing description content:\n%s", got)
		}
	})

	t.Run("it omits empty sections in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Simple task",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
			// No Parent, no Closed, no BlockedBy, no Children, no Description
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Parent, Closed should be omitted
		if strings.Contains(got, "Parent:") {
			t.Errorf("should omit Parent when empty:\n%s", got)
		}
		if strings.Contains(got, "Closed:") {
			t.Errorf("should omit Closed when empty:\n%s", got)
		}

		// Sections should be omitted
		if strings.Contains(got, "Blocked by:") {
			t.Errorf("should omit Blocked by section when empty:\n%s", got)
		}
		if strings.Contains(got, "Children:") {
			t.Errorf("should omit Children section when empty:\n%s", got)
		}
		if strings.Contains(got, "Description:") {
			t.Errorf("should omit Description section when empty:\n%s", got)
		}
	})

	t.Run("it does not truncate in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		longTitle := strings.Repeat("A", 100)
		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    longTitle,
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Full title should be present, not truncated
		if !strings.Contains(got, longTitle) {
			t.Errorf("title should NOT be truncated in show, got:\n%s", got)
		}
	})

	t.Run("it aligns labels in show output", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Test task",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

		// All values should start at the same column (after the aligned label)
		// Find value start positions
		var valueStarts []int
		for _, line := range lines {
			colonIdx := strings.Index(line, ":")
			if colonIdx >= 0 && colonIdx < len(line)-1 {
				// Find the first non-space character after colon+spaces
				rest := line[colonIdx+1:]
				trimmed := strings.TrimLeft(rest, " ")
				valueStart := colonIdx + 1 + (len(rest) - len(trimmed))
				valueStarts = append(valueStarts, valueStart)
			}
		}

		// All value start positions should be the same
		if len(valueStarts) > 1 {
			for i := 1; i < len(valueStarts); i++ {
				if valueStarts[i] != valueStarts[0] {
					t.Errorf("values not aligned: position %d=%d vs position 0=%d\n%s",
						i, valueStarts[i], valueStarts[0], got)
					break
				}
			}
		}
	})
}

func TestPrettyFormatter_FormatStats(t *testing.T) {
	t.Run("it formats stats with all groups, right-aligned", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		stats := StatsData{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Total group
		if !strings.Contains(got, "Total:") {
			t.Errorf("missing Total, got:\n%s", got)
		}

		// Status group
		for _, label := range []string{"Open:", "In Progress:", "Done:", "Cancelled:"} {
			if !strings.Contains(got, label) {
				t.Errorf("missing %q in status group, got:\n%s", label, got)
			}
		}

		// Workflow group
		for _, label := range []string{"Ready:", "Blocked:"} {
			if !strings.Contains(got, label) {
				t.Errorf("missing %q in workflow group, got:\n%s", label, got)
			}
		}

		// Priority group header
		if !strings.Contains(got, "Priority:") {
			t.Errorf("missing Priority group header, got:\n%s", got)
		}

		// Check groups are separated (Section headers: Status:, Workflow:, Priority:)
		if !strings.Contains(got, "Status:") {
			t.Errorf("missing Status: group header, got:\n%s", got)
		}
		if !strings.Contains(got, "Workflow:") {
			t.Errorf("missing Workflow: group header, got:\n%s", got)
		}
	})

	t.Run("it shows zero counts in stats", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		stats := StatsData{
			Total:      0,
			Open:       0,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      0,
			Blocked:    0,
			ByPriority: [5]int{0, 0, 0, 0, 0},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// All groups should still be present even with zeros
		for _, label := range []string{"Total:", "Open:", "In Progress:", "Done:", "Cancelled:", "Ready:", "Blocked:"} {
			if !strings.Contains(got, label) {
				t.Errorf("missing %q with zero counts, got:\n%s", label, got)
			}
		}

		// Priority labels should all be present
		for _, label := range []string{"P0", "P1", "P2", "P3", "P4"} {
			if !strings.Contains(got, label) {
				t.Errorf("missing priority label %q with zero counts, got:\n%s", label, got)
			}
		}
	})

	t.Run("it renders P0-P4 priority labels", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		stats := StatsData{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Check P0-P4 labels with descriptors
		expectedLabels := []string{
			"P0 (critical):",
			"P1 (high):",
			"P2 (medium):",
			"P3 (low):",
			"P4 (backlog):",
		}
		for _, label := range expectedLabels {
			if !strings.Contains(got, label) {
				t.Errorf("missing priority label %q, got:\n%s", label, got)
			}
		}

		// Check values are present
		if !strings.Contains(got, "2") || !strings.Contains(got, "25") {
			t.Errorf("missing priority values, got:\n%s", got)
		}
	})

	t.Run("it right-aligns numbers in stats", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		stats := StatsData{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Numbers should be right-aligned - check that single digit numbers
		// are preceded by spaces compared to double-digit numbers.
		// In the spec example:
		//   Open:        12
		//   In Progress:  3
		// The "3" should be right-aligned with "12"
		lines := strings.Split(got, "\n")

		// Find the "Open:" and "In Progress:" lines within the Status section
		var openLine, progressLine string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Open:") {
				openLine = line
			}
			if strings.HasPrefix(trimmed, "In Progress:") {
				progressLine = line
			}
		}

		if openLine == "" || progressLine == "" {
			t.Fatalf("could not find Open/In Progress lines in:\n%s", got)
		}

		// The digit "2" in "12" and "3" should end at the same column
		openEnd := len(strings.TrimRight(openLine, " \t"))
		progressEnd := len(strings.TrimRight(progressLine, " \t"))

		if openEnd != progressEnd {
			t.Errorf("numbers not right-aligned: Open ends at %d, In Progress ends at %d\n%s",
				openEnd, progressEnd, got)
		}
	})
}

func TestPrettyFormatter_FormatTransition(t *testing.T) {
	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatTransition(&buf, "tick-a1b2", "open", "in_progress")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		want := "tick-a1b2: open \u2192 in_progress\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestPrettyFormatter_FormatDepChange(t *testing.T) {
	t.Run("it formats dep added as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatDepChange(&buf, "tick-c3d4", "tick-a1b2", "added", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		want := "Dependency added: tick-c3d4 blocked by tick-a1b2\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it formats dep removed as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatDepChange(&buf, "tick-c3d4", "tick-a1b2", "removed", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		want := "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it suppresses dep output in quiet mode", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatDepChange(&buf, "tick-c3d4", "tick-a1b2", "added", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if buf.String() != "" {
			t.Errorf("expected empty output in quiet mode, got %q", buf.String())
		}
	})
}

func TestPrettyFormatter_FormatMessage(t *testing.T) {
	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatMessage(&buf, "Tick initialized in /path/to/.tick")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		want := "Tick initialized in /path/to/.tick\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
