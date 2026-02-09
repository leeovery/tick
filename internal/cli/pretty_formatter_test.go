package cli

import (
	"bytes"
	"strings"
	"testing"
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

		data := []TaskRow{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: "in_progress", Priority: 1},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "ID         STATUS       PRI  TITLE\n" +
			"tick-a1b2  done           1  Setup Sanctum\n" +
			"tick-c3d4  in_progress    1  Login endpoint\n"
		if buf.String() != expected {
			t.Errorf("output =\n%s\nwant =\n%s", buf.String(), expected)
		}
	})

	t.Run("it aligns with variable-width data", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := []TaskRow{
			{ID: "tick-ab", Title: "Short task", Status: "open", Priority: 2},
			{ID: "tick-cdef12", Title: "Longer task name", Status: "in_progress", Priority: 1},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "ID           STATUS       PRI  TITLE\n" +
			"tick-ab      open           2  Short task\n" +
			"tick-cdef12  in_progress    1  Longer task name\n"
		if buf.String() != expected {
			t.Errorf("output =\n%s\nwant =\n%s", buf.String(), expected)
		}
	})

	t.Run("it shows 'No tasks found.' for empty list", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, []TaskRow{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "No tasks found.\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it truncates long titles in list", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		longTitle := "This is a very long title that should be truncated because it exceeds the maximum width"
		data := []TaskRow{
			{ID: "tick-a1b2", Title: longTitle, Status: "open", Priority: 1},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		// Title should be truncated to 50 chars with "..."
		truncated := longTitle[:47] + "..."
		if !strings.Contains(got, truncated) {
			t.Errorf("expected truncated title %q in output:\n%s", truncated, got)
		}
		if strings.Contains(got, longTitle) {
			t.Errorf("expected title to be truncated, but found full title in output:\n%s", got)
		}
	})
}

func TestPrettyFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:          "tick-c3d4",
			title:       "Login endpoint",
			status:      "in_progress",
			priority:    1,
			parent:      "tick-e5f6",
			parentTitle: "Auth epic",
			created:     "2026-01-19T10:00:00Z",
			updated:     "2026-01-19T14:30:00Z",
			description: "Implement the login endpoint with validation.",
			blockedBy: []relatedTask{
				{id: "tick-a1b2", title: "Setup Sanctum", status: "done"},
			},
			children: []relatedTask{
				{id: "tick-g7h8", title: "Token refresh", status: "open"},
			},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "ID:       tick-c3d4\n" +
			"Title:    Login endpoint\n" +
			"Status:   in_progress\n" +
			"Priority: 1\n" +
			"Parent:   tick-e5f6  Auth epic\n" +
			"Created:  2026-01-19T10:00:00Z\n" +
			"Updated:  2026-01-19T14:30:00Z\n" +
			"\n" +
			"Blocked by:\n" +
			"  tick-a1b2  Setup Sanctum (done)\n" +
			"\n" +
			"Children:\n" +
			"  tick-g7h8  Token refresh (open)\n" +
			"\n" +
			"Description:\n" +
			"  Implement the login endpoint with validation.\n"
		if buf.String() != expected {
			t.Errorf("output =\n%s\nwant =\n%s", buf.String(), expected)
		}
	})

	t.Run("it omits empty sections in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:       "tick-a1b2",
			title:    "Simple task",
			status:   "open",
			priority: 2,
			created:  "2026-01-19T10:00:00Z",
			updated:  "2026-01-19T10:00:00Z",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "ID:       tick-a1b2\n" +
			"Title:    Simple task\n" +
			"Status:   open\n" +
			"Priority: 2\n" +
			"Created:  2026-01-19T10:00:00Z\n" +
			"Updated:  2026-01-19T10:00:00Z\n"
		if buf.String() != expected {
			t.Errorf("output =\n%s\nwant =\n%s", buf.String(), expected)
		}

		got := buf.String()
		if strings.Contains(got, "Blocked by:") {
			t.Error("expected no Blocked by section when empty")
		}
		if strings.Contains(got, "Children:") {
			t.Error("expected no Children section when empty")
		}
		if strings.Contains(got, "Description:") {
			t.Error("expected no Description section when empty")
		}
	})

	t.Run("it does not truncate in show", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		longTitle := "This is a very long title that should NOT be truncated in show because show displays full details"
		data := &showData{
			id:       "tick-a1b2",
			title:    longTitle,
			status:   "open",
			priority: 1,
			created:  "2026-01-19T10:00:00Z",
			updated:  "2026-01-19T10:00:00Z",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, longTitle) {
			t.Errorf("expected full title in show output, but it was truncated:\n%s", got)
		}
	})
}

func TestPrettyFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats with all groups, right-aligned", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &StatsData{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}

		err := f.FormatStats(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "Total:       47\n" +
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
		if buf.String() != expected {
			t.Errorf("output =\n%s\nwant =\n%s", buf.String(), expected)
		}
	})

	t.Run("it shows zero counts in stats", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &StatsData{
			Total:      0,
			Open:       0,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      0,
			Blocked:    0,
			ByPriority: [5]int{0, 0, 0, 0, 0},
		}

		err := f.FormatStats(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		// All groups must be present even with zeros.
		if !strings.Contains(got, "Total:") {
			t.Error("expected Total row")
		}
		if !strings.Contains(got, "Status:") {
			t.Error("expected Status group")
		}
		if !strings.Contains(got, "Workflow:") {
			t.Error("expected Workflow group")
		}
		if !strings.Contains(got, "Priority:") {
			t.Error("expected Priority group")
		}
		if !strings.Contains(got, "Open:") {
			t.Error("expected Open row")
		}
		if !strings.Contains(got, "Cancelled:") {
			t.Error("expected Cancelled row")
		}
	})

	t.Run("it renders P0-P4 priority labels", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &StatsData{
			ByPriority: [5]int{1, 2, 3, 4, 5},
		}

		err := f.FormatStats(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		labels := []string{
			"P0 (critical):",
			"P1 (high):",
			"P2 (medium):",
			"P3 (low):",
			"P4 (backlog):",
		}
		for _, label := range labels {
			if !strings.Contains(got, label) {
				t.Errorf("expected label %q in output:\n%s", label, got)
			}
		}
	})
}

func TestPrettyFormatterTransitionAndDep(t *testing.T) {
	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &TransitionData{
			ID:        "tick-a1b2",
			OldStatus: "open",
			NewStatus: "in_progress",
		}

		err := f.FormatTransition(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "tick-a1b2: open \u2192 in_progress\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it formats dep add as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &DepChangeData{
			Action:      "added",
			TaskID:      "tick-a1b2",
			BlockedByID: "tick-c3d4",
		}

		err := f.FormatDepChange(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "Dependency added: tick-a1b2 blocked by tick-c3d4\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it formats dep removed as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		data := &DepChangeData{
			Action:      "removed",
			TaskID:      "tick-a1b2",
			BlockedByID: "tick-c3d4",
		}

		err := f.FormatDepChange(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "Dependency removed: tick-a1b2 no longer blocked by tick-c3d4\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &PrettyFormatter{}
		var buf bytes.Buffer

		f.FormatMessage(&buf, "Initialized .tick directory")

		expected := "Initialized .tick directory\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})
}
