package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestToonFormatterFormatTaskList(t *testing.T) {
	t.Run("it formats list with correct header count and schema", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		tasks := []TaskRow{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: "open", Priority: 1},
		}

		err := f.FormatTaskList(&buf, tasks)
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		want := "tasks[2]{id,title,status,priority}:\n  tick-a1b2,Setup Sanctum,done,1\n  tick-c3d4,Login endpoint,open,1\n"
		if buf.String() != want {
			t.Errorf("FormatTaskList() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, []TaskRow{})
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		want := "tasks[0]{id,title,status,priority}:\n"
		if buf.String() != want {
			t.Errorf("FormatTaskList() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it escapes commas in titles", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		tasks := []TaskRow{
			{ID: "tick-a1b2", Title: "Setup, with commas", Status: "open", Priority: 2},
		}

		err := f.FormatTaskList(&buf, tasks)
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		want := "tasks[1]{id,title,status,priority}:\n  tick-a1b2,\"Setup, with commas\",open,2\n"
		if buf.String() != want {
			t.Errorf("FormatTaskList() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})
}

func TestToonFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:       "tick-a1b2",
			Title:    "Setup Sanctum",
			Status:   "in_progress",
			Priority: 1,
			Parent:   "tick-e5f6",
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T14:30:00Z",
			BlockedBy: []relatedTask{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
				{ID: "tick-g7h8", Title: "Config setup", Status: "in_progress"},
			},
			Children:    []relatedTask{},
			Description: "Full task description here.\nCan be multiple lines.",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		want := "task{id,title,status,priority,parent,created,updated}:\n" +
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

		if buf.String() != want {
			t.Errorf("FormatTaskDetail() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it omits parent/closed from schema when null", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:        "tick-a1b2",
			Title:     "Simple task",
			Status:    "open",
			Priority:  2,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
			BlockedBy: []relatedTask{},
			Children:  []relatedTask{},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		want := "task{id,title,status,priority,created,updated}:\n" +
			"  tick-a1b2,Simple task,open,2,2026-01-19T10:00:00Z,2026-01-19T10:00:00Z\n" +
			"\n" +
			"blocked_by[0]{id,title,status}:\n" +
			"\n" +
			"children[0]{id,title,status}:\n"

		if buf.String() != want {
			t.Errorf("FormatTaskDetail() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it renders blocked_by/children with count 0 when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:        "tick-a1b2",
			Title:     "No deps",
			Status:    "open",
			Priority:  2,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
			BlockedBy: []relatedTask{},
			Children:  []relatedTask{},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		got := buf.String()
		// Verify blocked_by[0] and children[0] with schema are present
		if !strings.Contains(got, "blocked_by[0]{id,title,status}:") {
			t.Errorf("output missing 'blocked_by[0]{id,title,status}:', got:\n%s", got)
		}
		if !strings.Contains(got, "children[0]{id,title,status}:") {
			t.Errorf("output missing 'children[0]{id,title,status}:', got:\n%s", got)
		}
	})

	t.Run("it omits description section when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:        "tick-a1b2",
			Title:     "No description",
			Status:    "open",
			Priority:  2,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
			BlockedBy: []relatedTask{},
			Children:  []relatedTask{},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		got := buf.String()
		if strings.Contains(got, "description:") {
			t.Errorf("output should not contain 'description:' when empty, got:\n%s", got)
		}
	})

	t.Run("it renders multiline description as indented lines", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:          "tick-a1b2",
			Title:       "With description",
			Status:      "open",
			Priority:    2,
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Description: "Line one.\nLine two.\nLine three.",
			BlockedBy:   []relatedTask{},
			Children:    []relatedTask{},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		got := buf.String()
		wantDesc := "description:\n  Line one.\n  Line two.\n  Line three.\n"
		if !strings.Contains(got, wantDesc) {
			t.Errorf("output missing expected description section.\ngot:\n%q\nwant to contain:\n%q", got, wantDesc)
		}
	})

	t.Run("it includes closed in schema when present", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:        "tick-a1b2",
			Title:     "Done task",
			Status:    "done",
			Priority:  1,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T16:00:00Z",
			Closed:    "2026-01-19T16:00:00Z",
			BlockedBy: []relatedTask{},
			Children:  []relatedTask{},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "task{id,title,status,priority,created,updated,closed}:") {
			t.Errorf("output missing closed in schema, got:\n%s", got)
		}
		if !strings.Contains(got, "2026-01-19T16:00:00Z\n") {
			t.Errorf("output missing closed value, got:\n%s", got)
		}
	})
}

func TestToonFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats with all counts", func(t *testing.T) {
		f := &ToonFormatter{}
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

		want := "stats{total,open,in_progress,done,cancelled,ready,blocked}:\n" +
			"  47,12,3,28,4,8,4\n" +
			"\n" +
			"by_priority[5]{priority,count}:\n" +
			"  0,2\n" +
			"  1,8\n" +
			"  2,25\n" +
			"  3,7\n" +
			"  4,5\n"

		if buf.String() != want {
			t.Errorf("FormatStats() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})

	t.Run("it formats by_priority with 5 rows including zeros", func(t *testing.T) {
		f := &ToonFormatter{}
		var buf bytes.Buffer

		stats := &StatsData{
			Total:      5,
			Open:       5,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      5,
			Blocked:    0,
			ByPriority: [5]int{0, 0, 5, 0, 0},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("FormatStats() error: %v", err)
		}

		want := "stats{total,open,in_progress,done,cancelled,ready,blocked}:\n" +
			"  5,5,0,0,0,5,0\n" +
			"\n" +
			"by_priority[5]{priority,count}:\n" +
			"  0,0\n" +
			"  1,0\n" +
			"  2,5\n" +
			"  3,0\n" +
			"  4,0\n"

		if buf.String() != want {
			t.Errorf("FormatStats() =\n%q\nwant:\n%q", buf.String(), want)
		}
	})
}

func TestToonFormatterFormatTransitionAndDep(t *testing.T) {
	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
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
		f := &ToonFormatter{}
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
		f := &ToonFormatter{}
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
		f := &ToonFormatter{}
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

func TestToonFormatterImplementsInterface(t *testing.T) {
	t.Run("it implements the full Formatter interface", func(t *testing.T) {
		var _ Formatter = &ToonFormatter{}
	})
}
