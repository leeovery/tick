package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestToonFormatterImplementsInterface(t *testing.T) {
	t.Run("it implements the full Formatter interface", func(t *testing.T) {
		var _ Formatter = &ToonFormatter{}
	})
}

func TestToonFormatterFormatTaskList(t *testing.T) {
	t.Run("it formats list with correct header count and schema", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := []TaskRow{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: "open", Priority: 1},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "tasks[2]{id,title,status,priority}:\n" +
			"  tick-a1b2,Setup Sanctum,done,1\n" +
			"  tick-c3d4,Login endpoint,open,1\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, []TaskRow{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "tasks[0]{id,title,status,priority}:\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})
}

func TestToonFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:          "tick-a1b2",
			title:       "Setup Sanctum",
			status:      "in_progress",
			priority:    1,
			parent:      "tick-e5f6",
			created:     "2026-01-19T10:00:00Z",
			updated:     "2026-01-19T14:30:00Z",
			description: "Full task description here.\nCan be multiple lines.",
			blockedBy: []relatedTask{
				{id: "tick-c3d4", title: "Database migrations", status: "done"},
				{id: "tick-g7h8", title: "Config setup", status: "in_progress"},
			},
			children: []relatedTask{},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "task{id,title,status,priority,parent,created,updated}:\n" +
			"  tick-a1b2,Setup Sanctum,in_progress,1,tick-e5f6,2026-01-19T10:00:00Z,2026-01-19T14:30:00Z\n" +
			"\n" +
			"blocked_by[2]{id,title,status}:\n" +
			"  tick-c3d4,Database migrations,done\n" +
			"  tick-g7h8,Config setup,in_progress\n" +
			"\n" +
			"children[0]{id,title,status}:\n" +
			"\n" +
			"description:\n" +
			"  Full task description here.\n" +
			"  Can be multiple lines.\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it omits parent/closed from schema when null", func(t *testing.T) {
		f := &ToonFormatter{}
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

		expected := "task{id,title,status,priority,created,updated}:\n" +
			"  tick-a1b2,Simple task,open,2,2026-01-19T10:00:00Z,2026-01-19T10:00:00Z\n" +
			"\n" +
			"blocked_by[0]{id,title,status}:\n" +
			"\n" +
			"children[0]{id,title,status}:\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it renders blocked_by/children with count 0 when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:       "tick-a1b2",
			title:    "Task",
			status:   "open",
			priority: 2,
			created:  "2026-01-19T10:00:00Z",
			updated:  "2026-01-19T10:00:00Z",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "blocked_by[0]{id,title,status}:") {
			t.Errorf("expected blocked_by[0] header, got %q", got)
		}
		if !strings.Contains(got, "children[0]{id,title,status}:") {
			t.Errorf("expected children[0] header, got %q", got)
		}
	})

	t.Run("it omits description section when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:       "tick-a1b2",
			title:    "Task",
			status:   "open",
			priority: 2,
			created:  "2026-01-19T10:00:00Z",
			updated:  "2026-01-19T10:00:00Z",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		if strings.Contains(got, "description:") {
			t.Errorf("expected no description section when empty, got %q", got)
		}
	})

	t.Run("it renders multiline description as indented lines", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:          "tick-a1b2",
			title:       "Task",
			status:      "open",
			priority:    2,
			created:     "2026-01-19T10:00:00Z",
			updated:     "2026-01-19T10:00:00Z",
			description: "Line one.\nLine two.\nLine three.",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "description:\n  Line one.\n  Line two.\n  Line three.\n") {
			t.Errorf("expected multiline description as indented lines, got %q", got)
		}
	})

	t.Run("it includes closed in schema when present", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:       "tick-a1b2",
			title:    "Done task",
			status:   "done",
			priority: 1,
			created:  "2026-01-19T10:00:00Z",
			updated:  "2026-01-19T14:30:00Z",
			closed:   "2026-01-19T14:30:00Z",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "task{id,title,status,priority,created,updated,closed}:") {
			t.Errorf("expected closed in schema, got %q", got)
		}
		if !strings.Contains(got, "2026-01-19T14:30:00Z") {
			t.Errorf("expected closed timestamp in values, got %q", got)
		}
	})
}

func TestToonFormatterEscaping(t *testing.T) {
	t.Run("it escapes commas in titles", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := []TaskRow{
			{ID: "tick-a1b2", Title: "Hello, World", Status: "open", Priority: 1},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "tasks[1]{id,title,status,priority}:\n" +
			"  tick-a1b2,\"Hello, World\",open,1\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})
}

func TestToonFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats with all counts", func(t *testing.T) {
		f := &ToonFormatter{}
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

		expected := "stats{total,open,in_progress,done,cancelled,ready,blocked}:\n" +
			"  47,12,3,28,4,8,4\n" +
			"\n" +
			"by_priority[5]{priority,count}:\n" +
			"  0,2\n" +
			"  1,8\n" +
			"  2,25\n" +
			"  3,7\n" +
			"  4,5\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it formats by_priority with 5 rows including zeros", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &StatsData{
			Total:      5,
			Open:       5,
			ByPriority: [5]int{0, 0, 5, 0, 0},
		}

		err := f.FormatStats(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "stats{total,open,in_progress,done,cancelled,ready,blocked}:\n" +
			"  5,5,0,0,0,0,0\n" +
			"\n" +
			"by_priority[5]{priority,count}:\n" +
			"  0,0\n" +
			"  1,0\n" +
			"  2,5\n" +
			"  3,0\n" +
			"  4,0\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})
}

func TestToonFormatterFormatTransitionAndDep(t *testing.T) {
	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
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
		f := &ToonFormatter{}
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
		f := &ToonFormatter{}
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
		f := &ToonFormatter{}
		var buf bytes.Buffer

		f.FormatMessage(&buf, "Initialized .tick directory")

		expected := "Initialized .tick directory\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})
}
