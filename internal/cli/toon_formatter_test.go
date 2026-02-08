package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestToonFormatter_ImplementsFormatter(t *testing.T) {
	t.Run("it implements the full Formatter interface", func(t *testing.T) {
		var _ Formatter = &ToonFormatter{}
	})
}

func TestToonFormatter_FormatTaskList(t *testing.T) {
	t.Run("it formats list with correct header count and schema", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		rows := []listRow{
			{ID: "tick-a1b2", Status: "done", Priority: 1, Title: "Setup Sanctum"},
			{ID: "tick-c3d4", Status: "open", Priority: 1, Title: "Login endpoint"},
		}

		err := f.FormatTaskList(&buf, rows, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		want := "tasks[2]{id,title,status,priority}:\n  tick-a1b2,Setup Sanctum,done,1\n  tick-c3d4,Login endpoint,open,1\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		want := "tasks[0]{id,title,status,priority}:\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("it escapes commas in titles", func(t *testing.T) {
		f := &ToonFormatter{}

		tests := []struct {
			name string
			fn   func() (string, error)
		}{
			{
				name: "in list",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTaskList(&buf, []listRow{
						{ID: "tick-a1b2", Status: "open", Priority: 2, Title: "Setup, configure auth"},
					}, false)
					return buf.String(), err
				},
			},
			{
				name: "in task detail title",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTaskDetail(&buf, TaskDetail{
						ID: "tick-a1b2", Title: "Setup, configure auth", Status: "open",
						Priority: 2, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
					})
					return buf.String(), err
				},
			},
			{
				name: "in related task title",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTaskDetail(&buf, TaskDetail{
						ID: "tick-a1b2", Title: "Task", Status: "open",
						Priority: 2, Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
						BlockedBy: []RelatedTask{
							{ID: "tick-c3d4", Title: "Fix, deploy infra", Status: "done"},
						},
					})
					return buf.String(), err
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got, err := tc.fn()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// toon-go quotes strings containing commas
				if !strings.Contains(got, `"`) {
					t.Errorf("expected comma-containing value to be quoted, got:\n%s", got)
				}
			})
		}
	})
}

func TestToonFormatter_FormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:          "tick-a1b2",
			Title:       "Setup Sanctum",
			Status:      "in_progress",
			Priority:    1,
			Parent:      "tick-e5f6",
			ParentTitle: "Epic task",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "2026-01-19T16:00:00Z",
			Description: "Full task description here.",
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
				{ID: "tick-g7h8", Title: "Config setup", Status: "in_progress"},
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

		// Check task section with all fields including parent and closed
		if !strings.Contains(got, "task{id,title,status,priority,parent,parent_title,created,updated,closed}:") {
			t.Errorf("missing task header with parent, parent_title, and closed, got:\n%s", got)
		}

		// Check blocked_by section
		if !strings.Contains(got, "blocked_by[2]{id,title,status}:") {
			t.Errorf("missing blocked_by header, got:\n%s", got)
		}

		// Check children section
		if !strings.Contains(got, "children[1]{id,title,status}:") {
			t.Errorf("missing children header, got:\n%s", got)
		}

		// Check description section
		if !strings.Contains(got, "description:") {
			t.Errorf("missing description section, got:\n%s", got)
		}
		if !strings.Contains(got, "  Full task description here.") {
			t.Errorf("missing description content, got:\n%s", got)
		}
	})

	t.Run("it omits parent/closed from schema when null", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Simple task",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
			// No Parent, no Closed
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Schema should NOT include parent or closed
		if strings.Contains(got, "parent") {
			t.Errorf("schema should omit parent when empty, got:\n%s", got)
		}
		if strings.Contains(got, "closed") {
			t.Errorf("schema should omit closed when empty, got:\n%s", got)
		}
		// Should have the basic fields
		if !strings.Contains(got, "task{id,title,status,priority,created,updated}:") {
			t.Errorf("missing expected task header, got:\n%s", got)
		}
	})

	t.Run("it renders blocked_by/children with count 0 when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Simple task",
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

		if !strings.Contains(got, "blocked_by[0]{id,title,status}:") {
			t.Errorf("missing blocked_by[0] section, got:\n%s", got)
		}
		if !strings.Contains(got, "children[0]{id,title,status}:") {
			t.Errorf("missing children[0] section, got:\n%s", got)
		}
	})

	t.Run("it omits description section when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Simple task",
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

		if strings.Contains(got, "description:") {
			t.Errorf("description section should be omitted when empty, got:\n%s", got)
		}
	})

	t.Run("it renders multiline description as indented lines", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:          "tick-a1b2",
			Title:       "Task with desc",
			Status:      "open",
			Priority:    2,
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Description: "Full task description here.\nCan be multiple lines.",
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		if !strings.Contains(got, "description:\n  Full task description here.\n  Can be multiple lines.\n") {
			t.Errorf("multiline description not properly indented, got:\n%s", got)
		}
	})
}

func TestToonFormatter_FormatStats(t *testing.T) {
	t.Run("it formats stats with all counts", func(t *testing.T) {
		f := &ToonFormatter{}
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

		if !strings.Contains(got, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("missing stats header, got:\n%s", got)
		}
		if !strings.Contains(got, "  47,12,3,28,4,8,4") {
			t.Errorf("missing stats data row, got:\n%s", got)
		}
	})

	t.Run("it formats by_priority with 5 rows including zeros", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		stats := StatsData{
			Total:      10,
			Open:       10,
			ByPriority: [5]int{0, 5, 3, 2, 0},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		if !strings.Contains(got, "by_priority[5]{priority,count}:") {
			t.Errorf("missing by_priority header, got:\n%s", got)
		}

		// Check all 5 rows including zeros
		lines := strings.Split(got, "\n")
		foundPriRows := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "0,") || strings.HasPrefix(trimmed, "1,") ||
				strings.HasPrefix(trimmed, "2,") || strings.HasPrefix(trimmed, "3,") ||
				strings.HasPrefix(trimmed, "4,") {
				foundPriRows++
			}
		}
		if foundPriRows != 5 {
			t.Errorf("expected 5 priority rows, found %d, got:\n%s", foundPriRows, got)
		}

		// Specifically check zero counts are present
		if !strings.Contains(got, "  0,0\n") {
			t.Errorf("expected rows with zero count, got:\n%s", got)
		}
	})
}

func TestToonFormatter_FormatTransitionAndDep(t *testing.T) {
	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
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

	t.Run("it formats dep change as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
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

	t.Run("it formats dep removal as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
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

	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
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
