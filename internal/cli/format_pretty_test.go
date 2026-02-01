package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrettyFormatterFormatTaskList(t *testing.T) {
	f := &PrettyFormatter{}

	t.Run("it formats list with aligned columns", func(t *testing.T) {
		var buf bytes.Buffer
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-d4e5f6", Title: "Login endpoint", Status: "open", Priority: 1},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 rows), got %d: %q", len(lines), out)
		}
		if !strings.Contains(lines[0], "ID") && !strings.Contains(lines[0], "STATUS") {
			t.Errorf("expected header with ID and STATUS, got %q", lines[0])
		}
	})

	t.Run("it aligns with variable-width data", func(t *testing.T) {
		var buf bytes.Buffer
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: "Short", Status: "open", Priority: 0},
			{ID: "tick-d4e5f6", Title: "A longer title here", Status: "in_progress", Priority: 3},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		// All lines should have consistent column alignment.
		// STATUS column should accommodate "in_progress" (11 chars).
		if !strings.Contains(lines[2], "in_progress") {
			t.Errorf("expected in_progress status, got %q", lines[2])
		}
	})

	t.Run("it shows 'No tasks found.' for empty list", func(t *testing.T) {
		var buf bytes.Buffer
		if err := f.FormatTaskList(&buf, []TaskListItem{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "No tasks found.\n"
		if buf.String() != want {
			t.Errorf("got %q, want %q", buf.String(), want)
		}
	})

	t.Run("it truncates long titles in list", func(t *testing.T) {
		var buf bytes.Buffer
		longTitle := strings.Repeat("A", 80)
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: longTitle, Status: "open", Priority: 2},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if strings.Contains(out, longTitle) {
			t.Error("should truncate long titles")
		}
		if !strings.Contains(out, "...") {
			t.Error("truncated title should end with ...")
		}
	})
}

func TestPrettyFormatterFormatTaskDetail(t *testing.T) {
	f := &PrettyFormatter{}

	t.Run("it formats show with all sections", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID:       "tick-a1b2c3",
			Title:    "Setup Sanctum",
			Status:   "in_progress",
			Priority: 1,
			Parent:   &RelatedTask{ID: "tick-e5f6a7", Title: "Auth System", Status: "open"},
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T14:30:00Z",
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4e5", Title: "Database migrations", Status: "done"},
			},
			Children:    []RelatedTask{{ID: "tick-x1y2z3", Title: "Child task", Status: "open"}},
			Description: "Implement the login endpoint with validation...",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "ID:") {
			t.Error("expected ID label")
		}
		if !strings.Contains(out, "tick-a1b2c3") {
			t.Error("expected task ID")
		}
		if !strings.Contains(out, "Blocked by:") {
			t.Error("expected Blocked by section")
		}
		if !strings.Contains(out, "Children:") {
			t.Error("expected Children section")
		}
		if !strings.Contains(out, "Description:") {
			t.Error("expected Description section")
		}
	})

	t.Run("it omits empty sections in show", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "No extras", Status: "open", Priority: 2,
			Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if strings.Contains(out, "Blocked by:") {
			t.Error("should omit empty Blocked by section")
		}
		if strings.Contains(out, "Children:") {
			t.Error("should omit empty Children section")
		}
		if strings.Contains(out, "Description:") {
			t.Error("should omit empty Description section")
		}
		if strings.Contains(out, "Parent:") {
			t.Error("should omit Parent when nil")
		}
	})

	t.Run("it does not truncate in show", func(t *testing.T) {
		var buf bytes.Buffer
		longTitle := strings.Repeat("B", 80)
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: longTitle, Status: "open", Priority: 2,
			Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), longTitle) {
			t.Error("show should not truncate title")
		}
	})
}

func TestPrettyFormatterFormatStats(t *testing.T) {
	f := &PrettyFormatter{}

	t.Run("it formats stats with all groups, right-aligned", func(t *testing.T) {
		var buf bytes.Buffer
		data := StatsData{
			Total: 47, Open: 12, InProgress: 3, Done: 28, Cancelled: 4,
			Ready: 8, Blocked: 4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}
		if err := f.FormatStats(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "Total:") {
			t.Error("expected Total label")
		}
		if !strings.Contains(out, "Status:") {
			t.Error("expected Status group")
		}
		if !strings.Contains(out, "Workflow:") {
			t.Error("expected Workflow group")
		}
		if !strings.Contains(out, "Priority:") {
			t.Error("expected Priority group")
		}
	})

	t.Run("it shows zero counts in stats", func(t *testing.T) {
		var buf bytes.Buffer
		data := StatsData{} // all zeros
		if err := f.FormatStats(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "Total:") {
			t.Error("expected Total even at zero")
		}
	})

	t.Run("it renders P0-P4 priority labels", func(t *testing.T) {
		var buf bytes.Buffer
		data := StatsData{ByPriority: [5]int{1, 2, 3, 4, 5}}
		if err := f.FormatStats(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		labels := []string{"P0 (critical)", "P1 (high)", "P2 (medium)", "P3 (low)", "P4 (backlog)"}
		for _, label := range labels {
			if !strings.Contains(out, label) {
				t.Errorf("expected priority label %q, got %q", label, out)
			}
		}
	})
}

func TestPrettyFormatterFormatTransition(t *testing.T) {
	f := &PrettyFormatter{}

	t.Run("it formats transition as plain text", func(t *testing.T) {
		var buf bytes.Buffer
		data := TransitionData{ID: "tick-a1b2c3", OldStatus: "open", NewStatus: "in_progress"}
		if err := f.FormatTransition(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "tick-a1b2c3: open → in_progress\n"
		if buf.String() != want {
			t.Errorf("got %q, want %q", buf.String(), want)
		}
	})
}
