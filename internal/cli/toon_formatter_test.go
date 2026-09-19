package cli

import (
	"slices"
	"strings"
	"testing"
	"time"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/task"
)

// awkwardDescription carries a blank line, a two-space indent, a header-shaped
// line, a bullet, a double quote, a tab and a carriage return.
const awkwardDescription = "Fix it.\n\n  indented by two\nSteps:\n- read the header\nHe said \"go\" \there\r\nlast line."

func detailWithDescription(description string) TaskDetail {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	return TaskDetail{
		Task: task.Task{
			ID:          "tick-a1b2",
			Title:       "With description",
			Status:      task.StatusOpen,
			Priority:    2,
			Description: description,
			Created:     now,
			Updated:     now,
		},
		BlockedBy: []RelatedTask{},
		Children:  []RelatedTask{},
	}
}

func detailWithNotes(notes []task.Note) TaskDetail {
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	return TaskDetail{
		Task: task.Task{
			ID:       "tick-a1b2",
			Title:    "Task with notes",
			Status:   task.StatusInProgress,
			Priority: 1,
			Created:  now,
			Updated:  now,
		},
		BlockedBy: []RelatedTask{},
		Children:  []RelatedTask{},
		Notes:     notes,
	}
}

func assertDescriptionRoundTrip(t *testing.T, description string) {
	t.Helper()
	f := &ToonFormatter{}
	doc := decodeToonDoc(t, f.FormatTaskDetail(detailWithDescription(description)))
	got, ok := doc["description"].(string)
	if !ok {
		t.Fatalf("description = %#v, want a string", doc["description"])
	}
	if got != description {
		t.Errorf("description = %q, want %q", got, description)
	}
}

func TestToonFormatter(t *testing.T) {
	// Compile-time interface verification.
	var _ Formatter = (*ToonFormatter)(nil)

	t.Run("it formats list with correct header count and schema", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %q", len(lines), result)
		}
		expectedHeader := "tasks[2]{id,title,status,priority,type}:"
		if lines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", lines[0], expectedHeader)
		}
		expectedRow1 := `  tick-a1b2,Setup Sanctum,done,1,""`
		if lines[1] != expectedRow1 {
			t.Errorf("row 1 = %q, want %q", lines[1], expectedRow1)
		}
		expectedRow2 := `  tick-c3d4,Login endpoint,open,1,""`
		if lines[2] != expectedRow2 {
			t.Errorf("row 2 = %q, want %q", lines[2], expectedRow2)
		}
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatTaskList([]task.Task{})
		expected := "tasks[0]{id,title,status,priority,type}:"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats zero tasks from nil slice as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatTaskList(nil)
		expected := "tasks[0]{id,title,status,priority,type}:"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:          "tick-a1b2",
				Title:       "Setup Sanctum",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: "Full task description here.\nCan be multiple lines.",
				Parent:      "tick-e5f6",
				Created:     now,
				Updated:     updated,
			},
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
				{ID: "tick-g7h8", Title: "Config setup", Status: "in_progress"},
			},
			Children:    []RelatedTask{},
			ParentTitle: "Auth System",
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		if len(sections) != 5 {
			t.Fatalf("expected 5 sections, got %d: %q", len(sections), result)
		}
		// Section 1: the task's own fields
		doc := decodeToonDoc(t, sections[0])
		assertToonFields(t, doc, map[string]any{
			"id":       "tick-a1b2",
			"title":    "Setup Sanctum",
			"status":   "in_progress",
			"priority": float64(1),
			"parent":   "tick-e5f6",
			"created":  "2026-01-19T10:00:00Z",
			"updated":  "2026-01-19T14:30:00Z",
		})
		assertToonKeysAbsent(t, doc, "task")
		// Section 2: blocked_by
		blockedLines := strings.Split(sections[1], "\n")
		expectedBlockedHeader := "blocked_by[2]{id,title,status}:"
		if blockedLines[0] != expectedBlockedHeader {
			t.Errorf("blocked_by header = %q, want %q", blockedLines[0], expectedBlockedHeader)
		}
		// Section 3: children
		expectedChildren := "children[0]{id,title,status}:"
		if sections[2] != expectedChildren {
			t.Errorf("children section = %q, want %q", sections[2], expectedChildren)
		}
		// Section 4: notes
		expectedNotes := "notes[0]{index,text,created}:"
		if sections[3] != expectedNotes {
			t.Errorf("notes section = %q, want %q", sections[3], expectedNotes)
		}
		// Section 5: description
		assertToonFields(t, decodeToonDoc(t, sections[4]), map[string]any{
			"description": detail.Task.Description,
		})
	})

	t.Run("it omits type, parent and closed when the task does not carry them", func(t *testing.T) {
		f := &ToonFormatter{}
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
		doc := decodeToonDoc(t, result)
		assertToonFields(t, doc, map[string]any{
			"id":       "tick-a1b2",
			"title":    "Simple task",
			"status":   "open",
			"priority": float64(2),
			"created":  "2026-01-19T10:00:00Z",
			"updated":  "2026-01-19T10:00:00Z",
		})
		assertToonKeysAbsent(t, doc, "task", "type", "parent", "closed")
	})

	t.Run("it renders blocked_by and children with count 0 when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No deps",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if !strings.Contains(result, "blocked_by[0]{id,title,status}:") {
			t.Errorf("missing blocked_by[0] section in: %q", result)
		}
		if !strings.Contains(result, "children[0]{id,title,status}:") {
			t.Errorf("missing children[0] section in: %q", result)
		}
		assertToonRowsEmpty(t, decodeToonDoc(t, result), "blocked_by", "children")
	})

	t.Run("it omits the description section when the description is empty", func(t *testing.T) {
		f := &ToonFormatter{}
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
		assertToonKeysAbsent(t, decodeToonDoc(t, result), "description")
		// Should have exactly 4 sections (task, blocked_by, children, notes)
		sections := strings.Split(result, "\n\n")
		if len(sections) != 4 {
			t.Errorf("expected 4 sections (no description), got %d: %q", len(sections), result)
		}
	})

	t.Run("it emits the description as one quoted value", func(t *testing.T) {
		f := &ToonFormatter{}
		description := "Line one.\nLine two.\nLine three."
		result := f.FormatTaskDetail(detailWithDescription(description))
		sections := strings.Split(result, "\n\n")
		descSection := sections[len(sections)-1]
		if strings.Contains(descSection, "\n") {
			t.Errorf("description section spans multiple lines: %q", descSection)
		}
		want := `description: "Line one.\nLine two.\nLine three."`
		if descSection != want {
			t.Errorf("description section = %q, want %q", descSection, want)
		}
		assertToonFields(t, decodeToonDoc(t, result), map[string]any{"description": description})
	})

	t.Run("it round-trips blank lines inside the description", func(t *testing.T) {
		assertDescriptionRoundTrip(t, "Fix it.\n\nSteps.")
	})

	t.Run("it round-trips interior lines with leading spaces", func(t *testing.T) {
		assertDescriptionRoundTrip(t, "Fix it.\n  indented by two\nback to the margin.")
	})

	t.Run("it round-trips a header-shaped line", func(t *testing.T) {
		description := "Fix it.\nSteps:\nrun the thing."
		assertDescriptionRoundTrip(t, description)
		doc := decodeToonDoc(t, (&ToonFormatter{}).FormatTaskDetail(detailWithDescription(description)))
		assertToonKeysAbsent(t, doc, "Steps")
	})

	t.Run("it round-trips a line beginning with a dash", func(t *testing.T) {
		assertDescriptionRoundTrip(t, "Fix it.\n- read the header\n- write it back")
	})

	t.Run("it round-trips a description beginning with a dash", func(t *testing.T) {
		assertDescriptionRoundTrip(t, "- read the header\n- write it back")
	})

	t.Run("it round-trips embedded quotes, tabs and carriage returns", func(t *testing.T) {
		assertDescriptionRoundTrip(t, "He said \"go\".\ncol\tcol\r\nafter the CR")
	})

	t.Run("it round-trips an awkward description byte-identically", func(t *testing.T) {
		assertDescriptionRoundTrip(t, awkwardDescription)
	})

	t.Run("it decodes the whole document with a description present", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:          "tick-a1b2",
				Title:       "Everything",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: awkwardDescription,
				Created:     now,
				Updated:     now,
			},
			BlockedBy: []RelatedTask{{ID: "tick-c3d4", Title: "Migrations", Status: "done"}},
			Children:  []RelatedTask{{ID: "tick-g7h8", Title: "Config setup", Status: "open"}},
			Tags:      []string{"backend"},
			Refs:      []string{"gh-123"},
			Notes:     []task.Note{{Text: "Started investigating", Created: now}},
		}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detail))
		for _, key := range []string{"id", "title", "status", "priority", "created", "updated", "blocked_by", "children", "tags", "refs", "notes"} {
			if _, ok := doc[key]; !ok {
				t.Errorf("key %q missing from decoded document", key)
			}
		}
		assertToonFields(t, doc, map[string]any{"description": awkwardDescription})
	})

	t.Run("it escapes commas in titles", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Setup, Deploy", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		expectedRow := `  tick-a1b2,"Setup, Deploy",open,1,""`
		if lines[1] != expectedRow {
			t.Errorf("row = %q, want %q", lines[1], expectedRow)
		}
	})

	t.Run("it emits stats counts as top-level named fields", func(t *testing.T) {
		f := &ToonFormatter{}
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
		doc := decodeToonDoc(t, f.FormatStats(stats))
		assertToonFields(t, doc, map[string]any{
			"total":       float64(47),
			"open":        float64(12),
			"in_progress": float64(3),
			"done":        float64(28),
			"cancelled":   float64(4),
			"ready":       float64(8),
			"blocked":     float64(4),
		})
		assertToonKeysAbsent(t, doc, "stats")
	})

	t.Run("it emits the counts in their established order", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatStats(Stats{Total: 47, Open: 12, Done: 28, Ready: 8, Blocked: 4})
		want := []string{"total", "open", "in_progress", "done", "cancelled", "ready", "blocked"}
		lines := strings.Split(strings.Split(result, "\n\n")[0], "\n")
		if len(lines) != len(want) {
			t.Fatalf("counts section = %q, want %d lines", lines, len(want))
		}
		for i, key := range want {
			if !strings.HasPrefix(lines[i], key+": ") {
				t.Errorf("line %d = %q, want field %q", i, lines[i], key)
			}
		}
	})

	t.Run("it emits a zero count rather than omitting it", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := decodeToonDoc(t, f.FormatStats(Stats{Total: 1, Open: 1, InProgress: 0}))
		assertToonFields(t, doc, map[string]any{"in_progress": float64(0)})
	})

	t.Run("it decodes counts as numbers", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := decodeToonDoc(t, f.FormatStats(Stats{Total: 47}))
		if _, ok := doc["total"].(float64); !ok {
			t.Errorf("total = %#v, want a float64", doc["total"])
		}
	})

	t.Run("it keeps the by_priority table unchanged", func(t *testing.T) {
		f := &ToonFormatter{}
		stats := Stats{Total: 47, ByPriority: [5]int{2, 8, 25, 7, 5}}
		rows := toonRows(t, decodeToonDoc(t, f.FormatStats(stats)), "by_priority")
		if len(rows) != 5 {
			t.Fatalf("by_priority has %d rows, want 5", len(rows))
		}
		for i, row := range rows {
			if row["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %#v, want %d", i, row["priority"], i)
			}
			if row["count"] != float64(stats.ByPriority[i]) {
				t.Errorf("by_priority[%d].count = %#v, want %d", i, row["count"], stats.ByPriority[i])
			}
		}
	})

	t.Run("it formats by_priority with 5 rows including zeros", func(t *testing.T) {
		f := &ToonFormatter{}
		stats := Stats{
			Total:      10,
			Open:       10,
			ByPriority: [5]int{0, 5, 3, 0, 2},
		}
		result := f.FormatStats(stats)
		sections := strings.Split(result, "\n\n")
		if len(sections) != 2 {
			t.Fatalf("expected 2 sections, got %d: %q", len(sections), result)
		}
		priorityLines := strings.Split(sections[1], "\n")
		expectedPriorityHeader := "by_priority[5]{priority,count}:"
		if priorityLines[0] != expectedPriorityHeader {
			t.Errorf("by_priority header = %q, want %q", priorityLines[0], expectedPriorityHeader)
		}
		if len(priorityLines) != 6 {
			t.Fatalf("expected 6 lines (header + 5 rows), got %d: %q", len(priorityLines), sections[1])
		}
		expectedRows := []string{
			"  0,0",
			"  1,5",
			"  2,3",
			"  3,0",
			"  4,2",
		}
		for i, expected := range expectedRows {
			if priorityLines[i+1] != expected {
				t.Errorf("priority row %d = %q, want %q", i, priorityLines[i+1], expected)
			}
		}
	})

	t.Run("it formats dep change as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
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

	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatMessage("Tick initialized in /path/to/project")
		expected := "Tick initialized in /path/to/project"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it includes closed in show schema when present", func(t *testing.T) {
		f := &ToonFormatter{}
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
		doc := decodeToonDoc(t, result)
		assertToonFields(t, doc, map[string]any{
			"id":       "tick-a1b2",
			"title":    "Done task",
			"status":   "done",
			"priority": float64(1),
			"created":  "2026-01-19T10:00:00Z",
			"updated":  "2026-01-19T10:00:00Z",
			"closed":   "2026-01-19T16:00:00Z",
		})
	})

	t.Run("it formats single task removal via baseFormatter", func(t *testing.T) {
		f := &ToonFormatter{}
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

	t.Run("it includes type in toon list rows", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Fix login bug", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Add search", Status: task.StatusDone, Priority: 2, Type: "feature", Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %q", len(lines), result)
		}
		// Header should include type in schema
		expectedHeader := "tasks[2]{id,title,status,priority,type}:"
		if lines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", lines[0], expectedHeader)
		}
		// Rows should include type value
		expectedRow1 := "  tick-a1b2,Fix login bug,open,1,bug"
		if lines[1] != expectedRow1 {
			t.Errorf("row 1 = %q, want %q", lines[1], expectedRow1)
		}
		expectedRow2 := "  tick-c3d4,Add search,done,2,feature"
		if lines[2] != expectedRow2 {
			t.Errorf("row 2 = %q, want %q", lines[2], expectedRow2)
		}
	})

	t.Run("it includes type in toon show when set", func(t *testing.T) {
		f := &ToonFormatter{}
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
		doc := decodeToonDoc(t, result)
		assertToonFields(t, doc, map[string]any{
			"id":       "tick-a1b2",
			"title":    "Fix login bug",
			"status":   "open",
			"priority": float64(1),
			"type":     "bug",
			"created":  "2026-01-19T10:00:00Z",
			"updated":  "2026-01-19T10:00:00Z",
		})
	})

	t.Run("it omits type from toon show when empty", func(t *testing.T) {
		f := &ToonFormatter{}
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
		doc := decodeToonDoc(t, result)
		assertToonKeysAbsent(t, doc, "type")
	})

	detailWith := func(tags, refs []string) TaskDetail {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		return TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Tagged task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			Tags:      tags,
			Refs:      refs,
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
	}

	t.Run("it emits tags as an inline list that decodes to the stored tags", func(t *testing.T) {
		f := &ToonFormatter{}
		tags := []string{"backend", "ui"}
		result := f.FormatTaskDetail(detailWith(tags, nil))
		assertToonStringList(t, decodeToonDoc(t, result), "tags", tags)
	})

	t.Run("it keeps a ref containing a comma as one element", func(t *testing.T) {
		f := &ToonFormatter{}
		refs := []string{"https://x.dev/a?b=1,2"}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWith(nil, refs)))
		assertToonStringList(t, doc, "refs", refs)
	})

	t.Run("it keeps a tag containing a space as one element", func(t *testing.T) {
		f := &ToonFormatter{}
		tags := []string{"has space", "plain"}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWith(tags, nil)))
		assertToonStringList(t, doc, "tags", tags)
	})

	t.Run("it quotes a ref containing a URL colon", func(t *testing.T) {
		f := &ToonFormatter{}
		refs := []string{"https://x.dev/issues/3", "JIRA-456"}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWith(nil, refs)))
		assertToonStringList(t, doc, "refs", refs)
	})

	t.Run("it emits a single-item refs list", func(t *testing.T) {
		f := &ToonFormatter{}
		refs := []string{"gh-123"}
		result := f.FormatTaskDetail(detailWith(nil, refs))
		if !strings.Contains(result, "refs[1]: gh-123") {
			t.Errorf("should contain single-item inline refs list, got:\n%s", result)
		}
		assertToonStringList(t, decodeToonDoc(t, result), "refs", refs)
	})

	t.Run("it omits the tags section when the task has no tags", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWith(nil, []string{"gh-123"})))
		assertToonKeysAbsent(t, doc, "tags")
	})

	t.Run("it omits the refs section when the task has no refs", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWith([]string{"backend"}, nil)))
		assertToonKeysAbsent(t, doc, "refs")
	})

	t.Run("it decodes the whole document when both tags and refs are present", func(t *testing.T) {
		f := &ToonFormatter{}
		tags := []string{"has space", "backend"}
		refs := []string{"https://x.dev/a?b=1,2", "PR #3"}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWith(tags, refs)))
		assertToonStringList(t, doc, "tags", tags)
		assertToonStringList(t, doc, "refs", refs)
	})

	t.Run("it includes type, parent and closed when the task carries them", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Done child",
				Status:   task.StatusDone,
				Priority: 1,
				Type:     "feature",
				Parent:   "tick-e5f6",
				Created:  now,
				Updated:  now,
				Closed:   &closed,
			},
			BlockedBy:   []RelatedTask{},
			Children:    []RelatedTask{},
			ParentTitle: "Parent task",
		}
		result := f.FormatTaskDetail(detail)
		doc := decodeToonDoc(t, result)
		assertToonFields(t, doc, map[string]any{
			"type":   "feature",
			"parent": "tick-e5f6",
			"closed": "2026-01-19T16:00:00Z",
		})
	})

	t.Run("it numbers notes from 1 in the toon notes table", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatTaskDetail(detailWithNotes([]task.Note{
			{Text: "Started investigating", Created: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)},
			{Text: "Root cause found", Created: time.Date(2026, 2, 27, 14, 30, 0, 0, time.UTC)},
		}))
		if !strings.Contains(result, "notes[2]{index,text,created}:") {
			t.Errorf("should contain notes section header with the index column, got:\n%s", result)
		}
		notes := decodeToonNotes(t, result)
		if len(notes) != 2 {
			t.Fatalf("notes length = %d, want 2", len(notes))
		}
		wantNotes := []map[string]any{
			{"index": float64(1), "text": "Started investigating", "created": "2026-02-27T10:00:00Z"},
			{"index": float64(2), "text": "Root cause found", "created": "2026-02-27T14:30:00Z"},
		}
		for i, want := range wantNotes {
			for key, wantValue := range want {
				if notes[i][key] != wantValue {
					t.Errorf("notes[%d].%s = %#v, want %#v", i, key, notes[i][key], wantValue)
				}
			}
		}
	})

	t.Run("it carries the index column on the empty notes section", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatTaskDetail(detailWithNotes(nil))
		if !strings.Contains(result, "notes[0]{index,text,created}:") {
			t.Errorf("should contain empty notes section 'notes[0]{index,text,created}:', got:\n%s", result)
		}
		if notes := decodeToonNotes(t, result); len(notes) != 0 {
			t.Errorf("notes = %#v, want empty", notes)
		}
	})

	t.Run("it keeps multi-line note text quoted", func(t *testing.T) {
		f := &ToonFormatter{}
		text := "multi\nline\nnote"
		result := f.FormatTaskDetail(detailWithNotes([]task.Note{
			{Text: text, Created: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)},
		}))
		notes := decodeToonNotes(t, result)
		if len(notes) != 1 {
			t.Fatalf("notes length = %d, want 1", len(notes))
		}
		if notes[0]["text"] != text {
			t.Errorf("notes[0].text = %#v, want %#v", notes[0]["text"], text)
		}
		if notes[0]["index"] != float64(1) {
			t.Errorf("notes[0].index = %#v, want %#v", notes[0]["index"], float64(1))
		}
	})

	t.Run("it keeps a note beginning with a dash intact", func(t *testing.T) {
		f := &ToonFormatter{}
		text := "- read the header, then retry"
		result := f.FormatTaskDetail(detailWithNotes([]task.Note{
			{Text: text, Created: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)},
		}))
		notes := decodeToonNotes(t, result)
		if len(notes) != 1 {
			t.Fatalf("notes length = %d, want 1", len(notes))
		}
		if notes[0]["text"] != text {
			t.Errorf("notes[0].text = %#v, want %#v", notes[0]["text"], text)
		}
	})
	t.Run("it emits the task's own fields as top-level named fields", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Add retry to the sync worker",
				Status:   task.StatusDone,
				Priority: 0,
				Type:     "feature",
				Parent:   "tick-e5f6",
				Created:  now,
				Updated:  now,
				Closed:   &closed,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		doc := decodeToonDoc(t, result)
		assertToonFields(t, doc, map[string]any{
			"id":       "tick-a1b2",
			"title":    "Add retry to the sync worker",
			"status":   "done",
			"priority": float64(0),
			"created":  "2026-01-19T10:00:00Z",
			"updated":  "2026-01-19T10:00:00Z",
		})
		assertToonKeysAbsent(t, doc, "task")

		head, _, _ := strings.Cut(result, "\n\n")
		var keys []string
		for line := range strings.SplitSeq(head, "\n") {
			key, _, _ := strings.Cut(line, ":")
			keys = append(keys, key)
		}
		wantKeys := []string{"id", "title", "status", "priority", "type", "parent", "created", "updated", "closed"}
		if !slices.Equal(keys, wantKeys) {
			t.Errorf("field order = %v, want %v", keys, wantKeys)
		}
	})

	t.Run("it round-trips a title containing a comma, a colon and a leading dash", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		title := "- fix: retries, backoff and jitter"
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    title,
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detail))
		if doc["title"] != title {
			t.Errorf("title = %#v, want %#v", doc["title"], title)
		}
	})

	t.Run("it emits created and updated as quoted timestamps", func(t *testing.T) {
		f := &ToonFormatter{}
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Timestamps",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  created,
				Updated:  updated,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detail))
		for key, want := range map[string]string{
			"created": task.FormatTimestamp(created),
			"updated": task.FormatTimestamp(updated),
		} {
			got, ok := doc[key].(string)
			if !ok {
				t.Errorf("%s = %#v, want a string", key, doc[key])
				continue
			}
			if got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
	})

	t.Run("it decodes the whole document when sections are joined by blank lines", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Everything",
				Status:   task.StatusInProgress,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{{ID: "tick-c3d4", Title: "Migrations", Status: "done"}},
			Children:  []RelatedTask{{ID: "tick-g7h8", Title: "Config setup", Status: "open"}},
			Notes:     []task.Note{{Text: "Started investigating", Created: now}},
		}
		doc := decodeToonDoc(t, f.FormatTaskDetail(detail))
		for _, key := range []string{"id", "blocked_by", "children", "notes"} {
			if _, ok := doc[key]; !ok {
				t.Errorf("key %q missing from decoded document", key)
			}
		}
	})

	t.Run("it returns an empty string when the encoder rejects a field value", func(t *testing.T) {
		if got := encodeToonFields(toon.Field{Key: "x", Value: make(chan int)}); got != "" {
			t.Errorf("encodeToonFields = %q, want empty string", got)
		}
	})

	t.Run("it omits the head rather than emitting a blank line when the head cannot be encoded", func(t *testing.T) {
		firstSection := "blocked_by[0]{id,title,status}:"
		got := joinToonSections([]string{"", firstSection, "notes[0]{index,text,created}:"})
		want := firstSection + "\n\nnotes[0]{index,text,created}:"
		if got != want {
			t.Errorf("joinToonSections = %q, want %q", got, want)
		}
	})
}

func TestToonFormatDepTree(t *testing.T) {
	f := &ToonFormatter{}

	t.Run("it renders single chain as edge list in full graph mode", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "Root", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-bbb222", Title: "Child", Status: "open"}},
					},
				},
			},
			ChainCount:   1,
			LongestChain: 1,
			BlockedCount: 1,
		})
		lines := strings.Split(result, "\n")
		if lines[0] != "dep_tree[1]{from,to}:" {
			t.Errorf("header = %q, want %q", lines[0], "dep_tree[1]{from,to}:")
		}
		if lines[1] != "  tick-aaa111,tick-bbb222" {
			t.Errorf("edge = %q, want %q", lines[1], "  tick-aaa111,tick-bbb222")
		}
	})

	t.Run("it renders multi-level chain as edge list", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
					Children: []DepTreeNode{
						{
							Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
							Children: []DepTreeNode{
								{Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"}},
							},
						},
					},
				},
			},
			ChainCount:   1,
			LongestChain: 2,
			BlockedCount: 2,
		})
		lines := strings.Split(result, "\n")
		if lines[0] != "dep_tree[2]{from,to}:" {
			t.Errorf("header = %q, want %q", lines[0], "dep_tree[2]{from,to}:")
		}
		if lines[1] != "  tick-aaa111,tick-bbb222" {
			t.Errorf("edge 1 = %q, want %q", lines[1], "  tick-aaa111,tick-bbb222")
		}
		if lines[2] != "  tick-bbb222,tick-ccc333" {
			t.Errorf("edge 2 = %q, want %q", lines[2], "  tick-bbb222,tick-ccc333")
		}
	})

	t.Run("it renders multiple independent chains", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"}},
					},
				},
				{
					Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-ddd444", Title: "D", Status: "open"}},
					},
				},
			},
			ChainCount:   2,
			LongestChain: 1,
			BlockedCount: 2,
		})
		lines := strings.Split(result, "\n")
		if lines[0] != "dep_tree[2]{from,to}:" {
			t.Errorf("header = %q, want %q", lines[0], "dep_tree[2]{from,to}:")
		}
		if lines[1] != "  tick-aaa111,tick-bbb222" {
			t.Errorf("edge 1 = %q, want %q", lines[1], "  tick-aaa111,tick-bbb222")
		}
		if lines[2] != "  tick-ccc333,tick-ddd444" {
			t.Errorf("edge 2 = %q, want %q", lines[2], "  tick-ccc333,tick-ddd444")
		}
	})

	t.Run("it duplicates edges for diamond dependencies", func(t *testing.T) {
		// A -> B, A -> C, B -> D, C -> D (D appears twice)
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
					Children: []DepTreeNode{
						{
							Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
							Children: []DepTreeNode{
								{Task: DepTreeTask{ID: "tick-ddd444", Title: "D", Status: "open"}},
							},
						},
						{
							Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"},
							Children: []DepTreeNode{
								{Task: DepTreeTask{ID: "tick-ddd444", Title: "D", Status: "open"}},
							},
						},
					},
				},
			},
			ChainCount:   1,
			LongestChain: 2,
			BlockedCount: 3,
		})
		lines := strings.Split(result, "\n")
		if lines[0] != "dep_tree[4]{from,to}:" {
			t.Errorf("header = %q, want %q", lines[0], "dep_tree[4]{from,to}:")
		}
		expectedEdges := []string{
			"  tick-aaa111,tick-bbb222",
			"  tick-bbb222,tick-ddd444",
			"  tick-aaa111,tick-ccc333",
			"  tick-ccc333,tick-ddd444",
		}
		for i, want := range expectedEdges {
			if lines[i+1] != want {
				t.Errorf("edge %d = %q, want %q", i, lines[i+1], want)
			}
		}
	})

	t.Run("it emits the dep tree summary as top-level named fields", func(t *testing.T) {
		doc := decodeToonDoc(t, f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"}},
					},
				},
			},
			ChainCount:   3,
			LongestChain: 5,
			BlockedCount: 7,
		}))
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(3),
			"longest": float64(5),
			"blocked": float64(7),
		})
		assertToonKeysAbsent(t, doc, "summary")
	})

	t.Run("it keeps the dep_tree edge section unchanged", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
					Children: []DepTreeNode{
						{
							Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
							Children: []DepTreeNode{
								{Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"}},
							},
						},
					},
				},
			},
			ChainCount:   1,
			LongestChain: 2,
			BlockedCount: 2,
		})
		lines := strings.Split(result, "\n")
		wantLines := []string{
			"dep_tree[2]{from,to}:",
			"  tick-aaa111,tick-bbb222",
			"  tick-bbb222,tick-ccc333",
		}
		if !slices.Equal(lines[:len(wantLines)], wantLines) {
			t.Errorf("edge section = %v, want %v", lines[:len(wantLines)], wantLines)
		}
		rows := toonRows(t, decodeToonDoc(t, result), "dep_tree")
		wantRows := []map[string]any{
			{"from": "tick-aaa111", "to": "tick-bbb222"},
			{"from": "tick-bbb222", "to": "tick-ccc333"},
		}
		if len(rows) != len(wantRows) {
			t.Fatalf("dep_tree length = %d, want %d", len(rows), len(wantRows))
		}
		for i, want := range wantRows {
			if rows[i]["from"] != want["from"] || rows[i]["to"] != want["to"] {
				t.Errorf("dep_tree[%d] = %#v, want %#v", i, rows[i], want)
			}
		}
	})

	t.Run("it emits zero-valued summary fields", func(t *testing.T) {
		doc := decodeToonDoc(t, f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"}},
					},
				},
			},
			ChainCount:   0,
			LongestChain: 2,
			BlockedCount: 1,
		}))
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(0),
			"longest": float64(2),
			"blocked": float64(1),
		})
	})

	t.Run("it renders the emptied full document for a result with no roots", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{})
		if first, _, _ := strings.Cut(result, "\n"); first != "dep_tree[0]{from,to}:" {
			t.Errorf("header = %q, want %q", first, "dep_tree[0]{from,to}:")
		}
		doc := decodeToonDoc(t, result)
		if rows := toonRows(t, doc, "dep_tree"); len(rows) != 0 {
			t.Errorf("dep_tree length = %d, want 0", len(rows))
		}
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(0),
			"longest": float64(0),
			"blocked": float64(0),
		})
	})

	t.Run("it names the target as top-level fields on the populated branch", func(t *testing.T) {
		doc := decodeToonDoc(t, f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
			BlockedBy: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"}},
			},
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"}},
			},
		}))

		assertToonFields(t, doc, map[string]any{"id": "tick-bbb222", "title": "B", "status": "open"})
		assertToonEdgeRows(t, doc, "blocked_by", []toonEdgeRow{{From: "tick-aaa111", To: "tick-bbb222"}})
		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{{From: "tick-bbb222", To: "tick-ccc333"}})
	})

	t.Run("it names the target as top-level fields on the no-dependencies branch", func(t *testing.T) {
		doc := decodeToonDoc(t, f.FormatDepTree(DepTreeResult{
			Target:  &DepTreeTask{ID: "tick-aaa111", Title: "Task A", Status: "open"},
			Message: "No dependencies.",
		}))

		assertToonFields(t, doc, map[string]any{"id": "tick-aaa111", "title": "Task A", "status": "open"})
		assertToonRowsEmpty(t, doc, "blocked_by", "blocks")
	})

	t.Run("it carries a count-zero blocked_by when only downstream exists", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"}},
			},
		})

		if !strings.Contains(result, "blocked_by[0]{from,to}:") {
			t.Errorf("output should carry a count-zero blocked_by header, got:\n%s", result)
		}
		doc := decodeToonDoc(t, result)
		assertToonRowsEmpty(t, doc, "blocked_by")
		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{{From: "tick-aaa111", To: "tick-bbb222"}})
	})

	t.Run("it carries a count-zero blocks when only upstream exists", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
			BlockedBy: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"}},
			},
		})

		if !strings.Contains(result, "blocks[0]{from,to}:") {
			t.Errorf("output should carry a count-zero blocks header, got:\n%s", result)
		}
		doc := decodeToonDoc(t, result)
		assertToonRowsEmpty(t, doc, "blocks")
		assertToonEdgeRows(t, doc, "blocked_by", []toonEdgeRow{{From: "tick-aaa111", To: "tick-bbb222"}})
	})

	t.Run("it quotes a target title containing a comma", func(t *testing.T) {
		title := "Parse, the header"
		doc := decodeToonDoc(t, f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-aaa111", Title: title, Status: "open"},
		}))

		assertToonFields(t, doc, map[string]any{"title": title})
	})

	t.Run("it emits no prose on the no-dependencies branch", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target:  &DepTreeTask{ID: "tick-aaa111", Title: "Task A", Status: "open"},
			Message: "No dependencies.",
		})

		if strings.Contains(result, "No dependencies.") {
			t.Errorf("output should carry no prose, got:\n%s", result)
		}
		decodeToonDoc(t, result)
	})

	t.Run("it renders wide graph with many edges", func(t *testing.T) {
		// One root blocking 4 tasks
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "Root", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"}},
						{Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"}},
						{Task: DepTreeTask{ID: "tick-ddd444", Title: "D", Status: "open"}},
						{Task: DepTreeTask{ID: "tick-eee555", Title: "E", Status: "open"}},
					},
				},
			},
			ChainCount:   1,
			LongestChain: 1,
			BlockedCount: 4,
		})
		lines := strings.Split(result, "\n")
		if lines[0] != "dep_tree[4]{from,to}:" {
			t.Errorf("header = %q, want %q", lines[0], "dep_tree[4]{from,to}:")
		}
		expectedEdges := []string{
			"  tick-aaa111,tick-bbb222",
			"  tick-aaa111,tick-ccc333",
			"  tick-aaa111,tick-ddd444",
			"  tick-aaa111,tick-eee555",
		}
		for i, want := range expectedEdges {
			if lines[i+1] != want {
				t.Errorf("edge %d = %q, want %q", i, lines[i+1], want)
			}
		}
	})

	t.Run("it keeps diamond duplication in the blocks direction", func(t *testing.T) {
		doc := decodeToonDoc(t, f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
			Blocks: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-ddd444", Title: "D", Status: "open"}},
					},
				},
				{
					Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-ddd444", Title: "D", Status: "open"}},
					},
				},
			},
		}))

		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-bbb222", To: "tick-ddd444"},
			{From: "tick-aaa111", To: "tick-ccc333"},
			{From: "tick-ccc333", To: "tick-ddd444"},
		})
	})
}
