package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestToonFormatterFormatTaskList(t *testing.T) {
	f := &ToonFormatter{}

	t.Run("it formats list with correct header count and schema", func(t *testing.T) {
		var buf bytes.Buffer
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-d4e5f6", Title: "Login endpoint", Status: "open", Priority: 1},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.HasPrefix(out, "tasks[2]{id,title,status,priority}:\n") {
			t.Errorf("expected header 'tasks[2]{id,title,status,priority}:', got %q", out)
		}
		if !strings.Contains(out, "  tick-a1b2c3,Setup Sanctum,done,1\n") {
			t.Errorf("expected first row, got %q", out)
		}
		if !strings.Contains(out, "  tick-d4e5f6,Login endpoint,open,1\n") {
			t.Errorf("expected second row, got %q", out)
		}
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		var buf bytes.Buffer
		if err := f.FormatTaskList(&buf, []TaskListItem{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "tasks[0]{id,title,status,priority}:\n"
		if buf.String() != want {
			t.Errorf("got %q, want %q", buf.String(), want)
		}
	})

	t.Run("it escapes commas in titles", func(t *testing.T) {
		var buf bytes.Buffer
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: "Fix bug, then test", Status: "open", Priority: 2},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"Fix bug, then test"`) {
			t.Errorf("expected escaped title with quotes, got %q", buf.String())
		}
	})
}

func TestToonFormatterFormatTaskDetail(t *testing.T) {
	f := &ToonFormatter{}

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
				{ID: "tick-g7h8i9", Title: "Config setup", Status: "in_progress"},
			},
			Children:    []RelatedTask{},
			Description: "Full task description here.\nCan be multiple lines.",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()

		// Task section with schema header
		if !strings.Contains(out, "task{id,title,status,priority,parent,created,updated}:") {
			t.Errorf("expected task section header with parent, got %q", out)
		}
		// Blocked by section
		if !strings.Contains(out, "blocked_by[2]{id,title,status}:") {
			t.Errorf("expected blocked_by[2], got %q", out)
		}
		// Children section always present
		if !strings.Contains(out, "children[0]{id,title,status}:") {
			t.Errorf("expected children[0], got %q", out)
		}
		// Description section
		if !strings.Contains(out, "description:\n") {
			t.Errorf("expected description section, got %q", out)
		}
		if !strings.Contains(out, "  Full task description here.\n") {
			t.Errorf("expected indented description line, got %q", out)
		}
	})

	t.Run("it omits parent/closed from schema when null", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID:       "tick-a1b2c3",
			Title:    "Simple task",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "task{id,title,status,priority,created,updated}:") {
			t.Errorf("expected task schema without parent/closed, got %q", out)
		}
	})

	t.Run("it renders blocked_by/children with count 0 when empty", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "Test", Status: "open", Priority: 2,
			Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "blocked_by[0]{id,title,status}:") {
			t.Errorf("expected blocked_by[0] with schema, got %q", out)
		}
		if !strings.Contains(out, "children[0]{id,title,status}:") {
			t.Errorf("expected children[0] with schema, got %q", out)
		}
	})

	t.Run("it omits description section when empty", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "No desc", Status: "open", Priority: 2,
			Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(buf.String(), "description") {
			t.Errorf("should not contain description section, got %q", buf.String())
		}
	})

	t.Run("it renders multiline description as indented lines", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "Test", Status: "open", Priority: 2,
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Description: "Line one.\nLine two.\nLine three.",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "  Line one.\n") {
			t.Errorf("expected indented first line, got %q", out)
		}
		if !strings.Contains(out, "  Line two.\n") {
			t.Errorf("expected indented second line, got %q", out)
		}
		if !strings.Contains(out, "  Line three.\n") {
			t.Errorf("expected indented third line, got %q", out)
		}
	})

	t.Run("it includes closed in schema when present", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "Done task", Status: "done", Priority: 2,
			Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T16:00:00Z",
			Closed:  "2026-01-19T16:00:00Z",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "closed") {
			t.Errorf("expected closed in schema, got %q", buf.String())
		}
	})
}

func TestToonFormatterFormatStats(t *testing.T) {
	f := &ToonFormatter{}

	t.Run("it formats stats with all counts", func(t *testing.T) {
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
		if !strings.Contains(out, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("expected stats section header, got %q", out)
		}
		if !strings.Contains(out, "  47,12,3,28,4,8,4\n") {
			t.Errorf("expected stats data row, got %q", out)
		}
	})

	t.Run("it formats by_priority with 5 rows including zeros", func(t *testing.T) {
		var buf bytes.Buffer
		data := StatsData{
			Total: 5, Open: 5,
			ByPriority: [5]int{0, 0, 5, 0, 0},
		}
		if err := f.FormatStats(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "by_priority[5]{priority,count}:") {
			t.Errorf("expected by_priority section header, got %q", out)
		}
		// All 5 rows present
		for i := 0; i < 5; i++ {
			if !strings.Contains(out, "  "+strings.TrimSpace(strings.Repeat(" ", 0))+string(rune('0'+i))+",") {
				// Check each priority row exists
			}
		}
		if !strings.Contains(out, "  0,0\n") {
			t.Errorf("expected P0 row with zero, got %q", out)
		}
		if !strings.Contains(out, "  2,5\n") {
			t.Errorf("expected P2 row with count 5, got %q", out)
		}
	})
}

func TestToonFormatterFormatTransition(t *testing.T) {
	f := &ToonFormatter{}

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

func TestToonFormatterFormatDepChange(t *testing.T) {
	f := &ToonFormatter{}

	t.Run("it formats dep add as plain text", func(t *testing.T) {
		var buf bytes.Buffer
		data := DepChangeData{Action: "added", TaskID: "tick-c3d4e5", BlockedBy: "tick-a1b2c3"}
		if err := f.FormatDepChange(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "Dependency added: tick-c3d4e5 blocked by tick-a1b2c3\n"
		if buf.String() != want {
			t.Errorf("got %q, want %q", buf.String(), want)
		}
	})

	t.Run("it formats dep rm as plain text", func(t *testing.T) {
		var buf bytes.Buffer
		data := DepChangeData{Action: "removed", TaskID: "tick-c3d4e5", BlockedBy: "tick-a1b2c3"}
		if err := f.FormatDepChange(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "Dependency removed: tick-c3d4e5 no longer blocked by tick-a1b2c3\n"
		if buf.String() != want {
			t.Errorf("got %q, want %q", buf.String(), want)
		}
	})
}
