package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestPrettyFormatterImplementsInterface(t *testing.T) {
	t.Run("it implements the full Formatter interface", func(t *testing.T) {
		var _ Formatter = &PrettyFormatter{}
	})
}

func TestPrettyFormatterFormatTaskList(t *testing.T) {
	t.Run("it formats list with aligned columns", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		tasks := []TaskRow{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: "in_progress", Priority: 1},
		}

		err := f.FormatTaskList(&buf, tasks)
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		want := "ID          STATUS       PRI  TITLE\n" +
			"tick-a1b2   done         1    Setup Sanctum\n" +
			"tick-c3d4   in_progress  1    Login endpoint\n"

		if buf.String() != want {
			t.Errorf("FormatTaskList() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it aligns with variable-width data", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		tasks := []TaskRow{
			{ID: "tick-aaaaaa", Title: "Short", Status: "open", Priority: 0},
			{ID: "tick-bb", Title: "A longer title here", Status: "in_progress", Priority: 4},
		}

		err := f.FormatTaskList(&buf, tasks)
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		got := buf.String()
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 rows), got %d:\n%s", len(lines), got)
		}

		// All lines should have consistent column alignment.
		// The STATUS column should start at the same position for all lines.
		// Find position of STATUS in header
		headerStatusIdx := strings.Index(lines[0], "STATUS")
		if headerStatusIdx < 0 {
			t.Fatal("header missing STATUS column")
		}

		// The status values in data rows should start at the same index
		for i := 1; i < len(lines); i++ {
			line := lines[i]
			// Find where the status value starts by checking the status field
			parts := strings.Fields(line)
			if len(parts) < 4 {
				t.Fatalf("row %d has fewer than 4 fields: %q", i, line)
			}
			statusIdx := strings.Index(line, parts[1])
			if statusIdx != headerStatusIdx {
				t.Errorf("row %d STATUS starts at %d, header STATUS at %d\nline: %q\nheader: %q",
					i, statusIdx, headerStatusIdx, line, lines[0])
			}
		}
	})

	t.Run("it shows 'No tasks found.' for empty list", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, []TaskRow{})
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		want := "No tasks found.\n"
		if buf.String() != want {
			t.Errorf("FormatTaskList() = %q, want %q", buf.String(), want)
		}
	})

	t.Run("it truncates long titles in list", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		longTitle := strings.Repeat("A", 100)
		tasks := []TaskRow{
			{ID: "tick-a1b2", Title: longTitle, Status: "open", Priority: 2},
		}

		err := f.FormatTaskList(&buf, tasks)
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		got := buf.String()
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		dataLine := lines[1]
		// Title should be truncated and end with "..."
		if !strings.HasSuffix(strings.TrimRight(dataLine, " \n"), "...") {
			t.Errorf("expected truncated title ending with '...', got: %q", dataLine)
		}
		// Full 100-char title should NOT appear
		if strings.Contains(got, longTitle) {
			t.Error("full long title should not appear in list output")
		}
	})
}

func TestPrettyFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:       "tick-c3d4",
			Title:    "Login endpoint",
			Status:   "in_progress",
			Priority: 1,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T14:30:00Z",
			BlockedBy: []relatedTask{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done"},
			},
			Children: []relatedTask{
				{ID: "tick-e5f6", Title: "Validate input", Status: "open"},
			},
			Description: "Implement the login endpoint with validation...",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		want := "ID:       tick-c3d4\n" +
			"Title:    Login endpoint\n" +
			"Status:   in_progress\n" +
			"Priority: 1\n" +
			"Created:  2026-01-19T10:00:00Z\n" +
			"Updated:  2026-01-19T14:30:00Z\n" +
			"\n" +
			"Blocked by:\n" +
			"  tick-a1b2  Setup Sanctum (done)\n" +
			"\n" +
			"Children:\n" +
			"  tick-e5f6  Validate input (open)\n" +
			"\n" +
			"Description:\n" +
			"  Implement the login endpoint with validation...\n"

		if buf.String() != want {
			t.Errorf("FormatTaskDetail() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it omits empty sections in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:        "tick-a1b2",
			Title:     "Simple task",
			Status:    "open",
			Priority:  2,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
			BlockedBy: nil,
			Children:  nil,
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		got := buf.String()

		// Should NOT contain empty section headers
		if strings.Contains(got, "Blocked by:") {
			t.Errorf("output should not contain 'Blocked by:' when empty, got:\n%s", got)
		}
		if strings.Contains(got, "Children:") {
			t.Errorf("output should not contain 'Children:' when empty, got:\n%s", got)
		}
		if strings.Contains(got, "Description:") {
			t.Errorf("output should not contain 'Description:' when empty, got:\n%s", got)
		}

		want := "ID:       tick-a1b2\n" +
			"Title:    Simple task\n" +
			"Status:   open\n" +
			"Priority: 2\n" +
			"Created:  2026-01-19T10:00:00Z\n" +
			"Updated:  2026-01-19T10:00:00Z\n"

		if got != want {
			t.Errorf("FormatTaskDetail() =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("it does not truncate in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		longTitle := strings.Repeat("A", 100)
		data := &showData{
			ID:       "tick-a1b2",
			Title:    longTitle,
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, longTitle) {
			t.Errorf("show output should contain full long title, got:\n%s", got)
		}
	})
}

func TestPrettyFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats with all groups, right-aligned", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		stats := &StatsData{
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
			t.Fatalf("FormatStats() error: %v", err)
		}

		want := "Total:       47\n" +
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
			"  P4 (backlog):   5\n"

		if buf.String() != want {
			t.Errorf("FormatStats() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it shows zero counts in stats", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		stats := &StatsData{
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
			t.Fatalf("FormatStats() error: %v", err)
		}

		got := buf.String()

		// All sections and rows should still be present even with zeros
		if !strings.Contains(got, "Total:") {
			t.Error("missing Total: line")
		}
		if !strings.Contains(got, "Status:") {
			t.Error("missing Status: section")
		}
		if !strings.Contains(got, "Workflow:") {
			t.Error("missing Workflow: section")
		}
		if !strings.Contains(got, "Priority:") {
			t.Error("missing Priority: section")
		}
		if !strings.Contains(got, "P0 (critical):") {
			t.Error("missing P0 (critical): row")
		}
		if !strings.Contains(got, "P4 (backlog):") {
			t.Error("missing P4 (backlog): row")
		}
	})

	t.Run("it renders P0-P4 priority labels", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		stats := &StatsData{
			Total:      10,
			Open:       10,
			ByPriority: [5]int{1, 2, 3, 2, 2},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("FormatStats() error: %v", err)
		}

		got := buf.String()
		expectedLabels := []string{
			"P0 (critical):",
			"P1 (high):",
			"P2 (medium):",
			"P3 (low):",
			"P4 (backlog):",
		}
		for _, label := range expectedLabels {
			if !strings.Contains(got, label) {
				t.Errorf("missing priority label %q in output:\n%s", label, got)
			}
		}
	})

	t.Run("it returns error for non-StatsData input", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatStats(&buf, "not stats data")
		if err == nil {
			t.Fatal("FormatStats() expected error for non-StatsData input, got nil")
		}
	})
}

func TestPrettyFormatterFormatTransitionAndDep(t *testing.T) {
	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatTransition(&buf, "tick-a3f2b7", task.StatusOpen, task.StatusInProgress)
		if err != nil {
			t.Fatalf("FormatTransition() error: %v", err)
		}

		want := "tick-a3f2b7: open \u2192 in_progress\n"
		if buf.String() != want {
			t.Errorf("FormatTransition() = %q, want %q", buf.String(), want)
		}
	})

	t.Run("it formats dep add as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatDepChange(&buf, "added", "tick-c3d4", "tick-a1b2")
		if err != nil {
			t.Fatalf("FormatDepChange() error: %v", err)
		}

		want := "Dependency added: tick-c3d4 blocked by tick-a1b2\n"
		if buf.String() != want {
			t.Errorf("FormatDepChange() = %q, want %q", buf.String(), want)
		}
	})

	t.Run("it formats dep removed as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatDepChange(&buf, "removed", "tick-c3d4", "tick-a1b2")
		if err != nil {
			t.Fatalf("FormatDepChange() error: %v", err)
		}

		want := "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2\n"
		if buf.String() != want {
			t.Errorf("FormatDepChange() = %q, want %q", buf.String(), want)
		}
	})

	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatMessage(&buf, "Tick initialized in /path/to/project")
		if err != nil {
			t.Fatalf("FormatMessage() error: %v", err)
		}

		want := "Tick initialized in /path/to/project\n"
		if buf.String() != want {
			t.Errorf("FormatMessage() = %q, want %q", buf.String(), want)
		}
	})
}
