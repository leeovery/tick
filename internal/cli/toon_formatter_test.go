package cli

import (
	"errors"
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
	doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detailWithDescription(description))))
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

	t.Run("it formats list as a decodable table of the given tasks", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Type: "feature", Created: now, Updated: now},
		}

		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskList(tasks)))

		rows := toonRows(t, doc, "tasks")
		if len(rows) != len(tasks) {
			t.Fatalf("tasks has %d rows, want %d", len(rows), len(tasks))
		}
		assertToonFields(t, rows[0], map[string]any{
			"id": "tick-a1b2", "title": "Setup Sanctum", "status": "done", "priority": float64(1), "type": "",
		})
		assertToonFields(t, rows[1], map[string]any{
			"id": "tick-c3d4", "title": "Login endpoint", "status": "open", "priority": float64(1), "type": "feature",
		})
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		result := formatted(t).of(f.FormatTaskList([]task.Task{}))
		expected := "tasks[0]{id,title,status,priority,type}:"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
		if rows := toonRows(t, decodeToonDoc(t, result), "tasks"); len(rows) != 0 {
			t.Errorf("tasks has %d rows, want 0", len(rows))
		}
	})

	t.Run("it formats zero tasks from nil slice as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		result := formatted(t).of(f.FormatTaskList(nil))
		expected := "tasks[0]{id,title,status,priority,type}:"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
		if rows := toonRows(t, decodeToonDoc(t, result), "tasks"); len(rows) != 0 {
			t.Errorf("tasks has %d rows, want 0", len(rows))
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
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
		blockedRows := toonRows(t, decodeToonDoc(t, sections[1]), "blocked_by")
		if len(blockedRows) != 2 {
			t.Fatalf("blocked_by has %d rows, want 2", len(blockedRows))
		}
		assertToonFields(t, blockedRows[0], map[string]any{"id": "tick-c3d4", "title": "Database migrations", "status": "done"})
		assertToonFields(t, blockedRows[1], map[string]any{"id": "tick-g7h8", "title": "Config setup", "status": "in_progress"})
		// Section 3: children
		assertCountZeroSection(t, sections[2], "children[0]{id,title,status}:")
		// Section 4: notes
		assertCountZeroSection(t, sections[3], "notes[0]{index,text,created}:")
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
		assertCountZeroSection(t, result, "blocked_by[0]{id,title,status}:")
		assertCountZeroSection(t, result, "children[0]{id,title,status}:")
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
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
		result := formatted(t).of(f.FormatTaskDetail(detailWithDescription(description)))
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
		doc := decodeToonDoc(t, formatted(t).of((&ToonFormatter{}).FormatTaskDetail(detailWithDescription(description))))
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detail)))
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
		rows := toonRows(t, decodeToonDoc(t, formatted(t).of(f.FormatTaskList(tasks))), "tasks")

		if len(rows) != 1 {
			t.Fatalf("tasks has %d rows, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{
			"id": "tick-a1b2", "title": "Setup, Deploy", "status": "open", "priority": float64(1), "type": "",
		})
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatStats(stats)))
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
		result := formatted(t).of(f.FormatStats(Stats{Total: 47, Open: 12, Done: 28, Ready: 8, Blocked: 4}))
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatStats(Stats{Total: 1, Open: 1, InProgress: 0})))
		assertToonFields(t, doc, map[string]any{"in_progress": float64(0)})
	})

	t.Run("it decodes counts as numbers", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := decodeToonDoc(t, formatted(t).of(f.FormatStats(Stats{Total: 47})))
		if _, ok := doc["total"].(float64); !ok {
			t.Errorf("total = %#v, want a float64", doc["total"])
		}
	})

	t.Run("it keeps the by_priority table unchanged", func(t *testing.T) {
		f := &ToonFormatter{}
		stats := Stats{Total: 47, ByPriority: [5]int{2, 8, 25, 7, 5}}
		rows := toonRows(t, decodeToonDoc(t, formatted(t).of(f.FormatStats(stats))), "by_priority")
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
		rows := toonRows(t, decodeToonDoc(t, formatted(t).of(f.FormatStats(stats))), "by_priority")

		if len(rows) != len(stats.ByPriority) {
			t.Fatalf("by_priority has %d rows, want %d", len(rows), len(stats.ByPriority))
		}
		for i, row := range rows {
			assertToonFields(t, row, map[string]any{
				"priority": float64(i),
				"count":    float64(stats.ByPriority[i]),
			})
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
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
		rows := toonRows(t, decodeToonDoc(t, formatted(t).of(f.FormatTaskList(tasks))), "tasks")

		if len(rows) != len(tasks) {
			t.Fatalf("tasks has %d rows, want %d", len(rows), len(tasks))
		}
		assertToonFields(t, rows[0], map[string]any{"id": "tick-a1b2", "type": "bug"})
		assertToonFields(t, rows[1], map[string]any{"id": "tick-c3d4", "type": "feature"})
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
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
		result := formatted(t).of(f.FormatTaskDetail(detailWith(tags, nil)))
		assertToonStringList(t, decodeToonDoc(t, result), "tags", tags)
	})

	t.Run("it keeps a ref containing a comma as one element", func(t *testing.T) {
		f := &ToonFormatter{}
		refs := []string{"https://x.dev/a?b=1,2"}
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detailWith(nil, refs))))
		assertToonStringList(t, doc, "refs", refs)
	})

	t.Run("it keeps a tag containing a space as one element", func(t *testing.T) {
		f := &ToonFormatter{}
		tags := []string{"has space", "plain"}
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detailWith(tags, nil))))
		assertToonStringList(t, doc, "tags", tags)
	})

	t.Run("it quotes a ref containing a URL colon", func(t *testing.T) {
		f := &ToonFormatter{}
		refs := []string{"https://x.dev/issues/3", "JIRA-456"}
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detailWith(nil, refs))))
		assertToonStringList(t, doc, "refs", refs)
	})

	t.Run("it emits a single-item refs list", func(t *testing.T) {
		f := &ToonFormatter{}
		refs := []string{"gh-123"}
		result := formatted(t).of(f.FormatTaskDetail(detailWith(nil, refs)))
		if !strings.Contains(result, "refs[1]: gh-123") {
			t.Errorf("should contain single-item inline refs list, got:\n%s", result)
		}
		assertToonStringList(t, decodeToonDoc(t, result), "refs", refs)
	})

	t.Run("it omits the tags section when the task has no tags", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detailWith(nil, []string{"gh-123"}))))
		assertToonKeysAbsent(t, doc, "tags")
	})

	t.Run("it omits the refs section when the task has no refs", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detailWith([]string{"backend"}, nil))))
		assertToonKeysAbsent(t, doc, "refs")
	})

	t.Run("it decodes the whole document when both tags and refs are present", func(t *testing.T) {
		f := &ToonFormatter{}
		tags := []string{"has space", "backend"}
		refs := []string{"https://x.dev/a?b=1,2", "PR #3"}
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detailWith(tags, refs))))
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
		doc := decodeToonDoc(t, result)
		assertToonFields(t, doc, map[string]any{
			"type":   "feature",
			"parent": "tick-e5f6",
			"closed": "2026-01-19T16:00:00Z",
		})
	})

	t.Run("it numbers notes from 1 in the toon notes table", func(t *testing.T) {
		f := &ToonFormatter{}
		result := formatted(t).of(f.FormatTaskDetail(detailWithNotes([]task.Note{
			{Text: "Started investigating", Created: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)},
			{Text: "Root cause found", Created: time.Date(2026, 2, 27, 14, 30, 0, 0, time.UTC)},
		})))
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
		result := formatted(t).of(f.FormatTaskDetail(detailWithNotes(nil)))
		assertCountZeroSection(t, result, "notes[0]{index,text,created}:")
	})

	t.Run("it keeps multi-line note text quoted", func(t *testing.T) {
		f := &ToonFormatter{}
		text := "multi\nline\nnote"
		result := formatted(t).of(f.FormatTaskDetail(detailWithNotes([]task.Note{
			{Text: text, Created: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)},
		})))
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
		result := formatted(t).of(f.FormatTaskDetail(detailWithNotes([]task.Note{
			{Text: text, Created: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)},
		})))
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
		result := formatted(t).of(f.FormatTaskDetail(detail))
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detail)))
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detail)))
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detail)))
		for _, key := range []string{"id", "blocked_by", "children", "notes"} {
			if _, ok := doc[key]; !ok {
				t.Errorf("key %q missing from decoded document", key)
			}
		}
	})

	t.Run("it returns the encoder's error from encodeToonFields", func(t *testing.T) {
		got, err := encodeToonFields(toon.Field{Key: "description", Value: "bell " + refusedChar + " here"})
		if err == nil {
			t.Fatalf("encodeToonFields error = nil, want the encoder's refusal")
		}
		if got != "" {
			t.Errorf("encodeToonFields = %q, want empty string", got)
		}
		if !strings.Contains(err.Error(), "description") {
			t.Errorf("error = %q, want it to name the description field", err)
		}
	})

	t.Run("it returns the encoder's error from encodeToonSection", func(t *testing.T) {
		got, err := encodeToonSection("notes", []toonNoteRow{{Index: 1, Text: "bell " + refusedChar + " here"}})
		if err == nil {
			t.Fatalf("encodeToonSection error = nil, want the encoder's refusal")
		}
		if got != "" {
			t.Errorf("encodeToonSection = %q, want empty string", got)
		}
		if !strings.Contains(err.Error(), "notes") {
			t.Errorf("error = %q, want it to name the notes section", err)
		}
	})

	t.Run("it falls back to the section name when the section refuses but no single row does", func(t *testing.T) {
		rows := []toonTaskRow{{ID: "tick-aaa111", Title: "an ordinary title"}, {ID: "tick-bbb222", Title: "another ordinary title"}}
		_, want := encodeToonSection("tasks", []toonTaskRow{{ID: "tick-aaa111", Title: "bell " + refusedChar + " title"}})

		got := sectionRefusal("tasks", rows, func(r toonTaskRow) string { return r.ID }, errors.Unwrap(want))

		if got.Error() != want.Error() {
			t.Errorf("error = %q, want the section-only message %q", got, want)
		}
	})

	t.Run("it keeps the emptied section header when the section holds no rows", func(t *testing.T) {
		got, err := buildNotesSection(nil, nil)
		if err != nil {
			t.Fatalf("buildNotesSection error = %v, want nil", err)
		}
		if want := "notes[0]{index,text,created}:"; got != want {
			t.Errorf("buildNotesSection = %q, want %q", got, want)
		}
	})

	t.Run("it omits the head rather than emitting a blank line when no top-level field is selected", func(t *testing.T) {
		head, err := buildTaskSection(task.Task{ID: "tick-a1b2", Title: "Anything"}, fieldSelection(t, "notes"))
		if err != nil {
			t.Fatalf("buildTaskSection error = %v, want nil", err)
		}
		if head != "" {
			t.Fatalf("buildTaskSection = %q, want empty string", head)
		}

		notes := "notes[0]{index,text,created}:"
		if got := joinToonSections([]string{head, notes}); got != notes {
			t.Errorf("joinToonSections = %q, want %q", got, notes)
		}
	})
}

func TestToonFormatDepTree(t *testing.T) {
	f := &ToonFormatter{}

	t.Run("it renders single chain as edge list in full graph mode", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Edges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
			},
			ChainCount:   1,
			LongestChain: 1,
			BlockedCount: 1,
		}))
		assertToonEdgeRows(t, decodeToonDoc(t, result), "dep_tree", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
		})
	})

	t.Run("it renders multi-level chain as edge list", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Edges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
				{From: "tick-bbb222", To: "tick-ccc333"},
			},
			ChainCount:   1,
			LongestChain: 2,
			BlockedCount: 2,
		}))
		assertToonEdgeRows(t, decodeToonDoc(t, result), "dep_tree", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-bbb222", To: "tick-ccc333"},
		})
	})

	t.Run("it renders multiple independent chains", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Edges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
				{From: "tick-ccc333", To: "tick-ddd444"},
			},
			ChainCount:   2,
			LongestChain: 1,
			BlockedCount: 2,
		}))
		assertToonEdgeRows(t, decodeToonDoc(t, result), "dep_tree", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-ccc333", To: "tick-ddd444"},
		})
	})

	t.Run("it emits one edge per dependency of a diamond", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Edges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
				{From: "tick-aaa111", To: "tick-ccc333"},
				{From: "tick-bbb222", To: "tick-ddd444"},
				{From: "tick-ccc333", To: "tick-ddd444"},
			},
			ChainCount:   1,
			LongestChain: 2,
			BlockedCount: 3,
		}))
		assertToonEdgeRows(t, decodeToonDoc(t, result), "dep_tree", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-aaa111", To: "tick-ccc333"},
			{From: "tick-bbb222", To: "tick-ddd444"},
			{From: "tick-ccc333", To: "tick-ddd444"},
		})
	})

	t.Run("it emits the dep tree summary as top-level named fields", func(t *testing.T) {
		doc := decodeToonDoc(t, formatted(t).of(f.FormatDepTree(DepTreeResult{
			Edges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
			},
			ChainCount:   3,
			LongestChain: 5,
			BlockedCount: 7,
		})))
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(3),
			"longest": float64(5),
			"blocked": float64(7),
		})
		assertToonKeysAbsent(t, doc, "summary")
	})

	t.Run("it keeps the dep_tree edge section unchanged", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Edges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
				{From: "tick-bbb222", To: "tick-ccc333"},
			},
			ChainCount:   1,
			LongestChain: 2,
			BlockedCount: 2,
		}))
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatDepTree(DepTreeResult{
			Trees: []DepTreeNode{
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
		})))
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(0),
			"longest": float64(2),
			"blocked": float64(1),
		})
	})

	t.Run("it renders the emptied full document for a result with no trees", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{}))
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
		doc := decodeToonDoc(t, formatted(t).of(f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
			BlockedBy: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"}},
			},
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "open"}},
			},
			BlockedByEdges: []DepTreeEdge{{From: "tick-aaa111", To: "tick-bbb222"}},
			BlocksEdges:    []DepTreeEdge{{From: "tick-bbb222", To: "tick-ccc333"}},
		})))

		assertToonFields(t, doc, map[string]any{"id": "tick-bbb222", "title": "B", "status": "open"})
		assertToonEdgeRows(t, doc, "blocked_by", []toonEdgeRow{{From: "tick-aaa111", To: "tick-bbb222"}})
		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{{From: "tick-bbb222", To: "tick-ccc333"}})
	})

	t.Run("it names the target as top-level fields on the no-dependencies branch", func(t *testing.T) {
		doc := decodeToonDoc(t, formatted(t).of(f.FormatDepTree(DepTreeResult{
			Target:  &DepTreeTask{ID: "tick-aaa111", Title: "Task A", Status: "open"},
			Message: "No dependencies.",
		})))

		assertToonFields(t, doc, map[string]any{"id": "tick-aaa111", "title": "Task A", "status": "open"})
		assertToonRowsEmpty(t, doc, "blocked_by", "blocks")
	})

	t.Run("it carries a count-zero blocked_by when only downstream exists", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"}},
			},
			BlocksEdges: []DepTreeEdge{{From: "tick-aaa111", To: "tick-bbb222"}},
		}))

		assertCountZeroSection(t, result, "blocked_by[0]{from,to}:")
		assertToonEdgeRows(t, decodeToonDoc(t, result), "blocks", []toonEdgeRow{{From: "tick-aaa111", To: "tick-bbb222"}})
	})

	t.Run("it carries a count-zero blocks when only upstream exists", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "open"},
			BlockedBy: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"}},
			},
			BlockedByEdges: []DepTreeEdge{{From: "tick-aaa111", To: "tick-bbb222"}},
		}))

		assertCountZeroSection(t, result, "blocks[0]{from,to}:")
		assertToonEdgeRows(t, decodeToonDoc(t, result), "blocked_by", []toonEdgeRow{{From: "tick-aaa111", To: "tick-bbb222"}})
	})

	t.Run("it quotes a target title containing a comma", func(t *testing.T) {
		title := "Parse, the header"
		doc := decodeToonDoc(t, formatted(t).of(f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-aaa111", Title: title, Status: "open"},
		})))

		assertToonFields(t, doc, map[string]any{"title": title})
	})

	t.Run("it emits no prose on the no-dependencies branch", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Target:  &DepTreeTask{ID: "tick-aaa111", Title: "Task A", Status: "open"},
			Message: "No dependencies.",
		}))

		if strings.Contains(result, "No dependencies.") {
			t.Errorf("output should carry no prose, got:\n%s", result)
		}
		decodeToonDoc(t, result)
	})

	t.Run("it renders wide graph with many edges", func(t *testing.T) {
		result := formatted(t).of(f.FormatDepTree(DepTreeResult{
			Edges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
				{From: "tick-aaa111", To: "tick-ccc333"},
				{From: "tick-aaa111", To: "tick-ddd444"},
				{From: "tick-aaa111", To: "tick-eee555"},
			},
			ChainCount:   1,
			LongestChain: 1,
			BlockedCount: 4,
		}))
		assertToonEdgeRows(t, decodeToonDoc(t, result), "dep_tree", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-aaa111", To: "tick-ccc333"},
			{From: "tick-aaa111", To: "tick-ddd444"},
			{From: "tick-aaa111", To: "tick-eee555"},
		})
	})

	t.Run("it renders the focused sections from the result's edge fields", func(t *testing.T) {
		doc := decodeToonDoc(t, formatted(t).of(f.FormatDepTree(DepTreeResult{
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
			BlocksEdges: []DepTreeEdge{
				{From: "tick-aaa111", To: "tick-bbb222"},
				{From: "tick-aaa111", To: "tick-ccc333"},
				{From: "tick-bbb222", To: "tick-ddd444"},
				{From: "tick-ccc333", To: "tick-ddd444"},
			},
		})))

		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-aaa111", To: "tick-ccc333"},
			{From: "tick-bbb222", To: "tick-ddd444"},
			{From: "tick-ccc333", To: "tick-ddd444"},
		})
	})
}

func fieldSelection(t *testing.T, value string) *FieldSelection {
	t.Helper()
	sel := newFieldSelection()
	if err := sel.addValue(value); err != nil {
		t.Fatalf("addValue(%q) failed: %v", value, err)
	}
	return sel
}

// richDetail carries every section and every optional scalar.
func richDetail() TaskDetail {
	created := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	closed := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	return TaskDetail{
		Task: task.Task{
			ID:          "tick-a1b2",
			Title:       "Add retry to the sync worker",
			Status:      task.StatusInProgress,
			Priority:    1,
			Type:        "bug",
			Parent:      "tick-p4r3",
			Description: "Fix the parser.\n\nSteps:\n  - read the header",
			Created:     created,
			Updated:     created,
			Closed:      &closed,
		},
		BlockedBy: []RelatedTask{{ID: "tick-c3d4", Title: "Blocker", Status: "open"}},
		Children:  []RelatedTask{{ID: "tick-e5f6", Title: "Child", Status: "open"}},
		Tags:      []string{"api"},
		Refs:      []string{"https://example.com"},
		Notes:     []task.Note{{Text: "Retried twice before it stuck", Created: created}},
	}
}

func assertToonKeySet(t *testing.T, doc map[string]any, want ...string) {
	t.Helper()
	got := make([]string, 0, len(doc))
	for key := range doc {
		got = append(got, key)
	}
	slices.Sort(got)
	sorted := slices.Clone(want)
	slices.Sort(sorted)
	if !slices.Equal(got, sorted) {
		t.Errorf("document keys = %v, want %v", got, sorted)
	}
}

func assertToonKeyOrder(t *testing.T, document string, first, second string) {
	t.Helper()
	firstAt := strings.Index(document, first)
	secondAt := strings.Index(document, second)
	if firstAt < 0 || secondAt < 0 {
		t.Fatalf("document missing %q or %q:\n%s", first, second, document)
	}
	if firstAt > secondAt {
		t.Errorf("%q should precede %q in:\n%s", first, second, document)
	}
}

func TestToonFilteredTaskDetail(t *testing.T) {
	f := &ToonFormatter{}

	filtered := func(t *testing.T, detail TaskDetail, value string) string {
		t.Helper()
		detail.Fields = fieldSelection(t, value)
		return formatted(t).of(f.FormatTaskDetail(detail))
	}

	t.Run("it renders only the selected scalars", func(t *testing.T) {
		doc := decodeToonDoc(t, filtered(t, richDetail(), "title,status"))
		assertToonKeySet(t, doc, "title", "status")
		assertToonFields(t, doc, map[string]any{
			"title":  "Add retry to the sync worker",
			"status": "in_progress",
		})
	})

	t.Run("it renders selected scalars in output order", func(t *testing.T) {
		result := filtered(t, richDetail(), "status,title")

		assertToonKeyOrder(t, result, "title:", "status:")
		doc := decodeToonDoc(t, result)
		assertToonKeySet(t, doc, "title", "status")
		assertToonFields(t, doc, map[string]any{
			"title":  "Add retry to the sync worker",
			"status": "in_progress",
		})
	})

	t.Run("it renders a selected section alone", func(t *testing.T) {
		doc := decodeToonDoc(t, filtered(t, richDetail(), "notes"))
		assertToonKeySet(t, doc, "notes")
		if rows := toonRows(t, doc, "notes"); len(rows) != 1 {
			t.Errorf("notes rows = %d, want 1", len(rows))
		}
	})

	t.Run("it mixes scalars and sections in output order", func(t *testing.T) {
		result := filtered(t, richDetail(), "description,notes,title")
		doc := decodeToonDoc(t, result)
		assertToonKeySet(t, doc, "title", "notes", "description")
		assertToonKeyOrder(t, result, "title:", "notes[")
		assertToonKeyOrder(t, result, "notes[", "description:")
	})

	t.Run("it separates the head block from the first section by one blank line", func(t *testing.T) {
		result := filtered(t, richDetail(), "title,notes")

		head, rest, ok := strings.Cut(result, "\n\n")
		if !ok {
			t.Fatalf("result = %q, want a head block and a section", result)
		}
		if strings.HasPrefix(rest, "\n") {
			t.Errorf("result = %q, want exactly one blank line after the head block", result)
		}
		assertToonFields(t, decodeToonDoc(t, head), map[string]any{"title": "Add retry to the sync worker"})
		if rows := toonRows(t, decodeToonDoc(t, rest), "notes"); len(rows) != 1 {
			t.Errorf("notes rows = %d, want 1", len(rows))
		}
	})

	t.Run("it does not carry id unless asked", func(t *testing.T) {
		assertToonKeysAbsent(t, decodeToonDoc(t, filtered(t, richDetail(), "title")), "id")
	})

	t.Run("it renders a count-zero header for an always-present section", func(t *testing.T) {
		for _, name := range []string{"notes", "children", "blocked_by"} {
			result := filtered(t, detailWithDescription(""), name)
			assertToonRowsEmpty(t, decodeToonDoc(t, result), name)
		}
	})

	t.Run("it renders nothing for a carried-only field the task lacks", func(t *testing.T) {
		for _, name := range []string{"tags", "refs", "description", "type", "parent", "closed"} {
			if result := filtered(t, detailWithDescription(""), name); result != "" {
				t.Errorf("--field %s = %q, want empty", name, result)
			}
		}
	})

	t.Run("it renders nothing when every selected name is empty", func(t *testing.T) {
		if result := filtered(t, detailWithDescription(""), "tags,refs"); result != "" {
			t.Errorf("result = %q, want empty", result)
		}
	})

	t.Run("it has no leading blank line when the head block is empty", func(t *testing.T) {
		result := filtered(t, richDetail(), "notes,description")
		if !strings.HasPrefix(result, "notes[") {
			t.Errorf("result = %q, want it to start with %q", result, "notes[")
		}
	})

	t.Run("it has no trailing blank line when the last selected field is empty", func(t *testing.T) {
		result := filtered(t, detailWithNotes(nil), "notes,description")
		if strings.HasSuffix(result, "\n") {
			t.Errorf("result = %q, want no trailing blank line", result)
		}
		assertCountZeroSection(t, result, "notes[0]{index,text,created}:")
		assertToonKeySet(t, decodeToonDoc(t, result), "notes")
	})

	t.Run("it renders the changed section for a mutation document", func(t *testing.T) {
		detail := richDetail()
		detail.Changes = &StatusChanges{Blocks: []CascadeResult{{
			TaskID: "tick-a1b2", TaskTitle: "Add retry to the sync worker",
			OldStatus: "open", NewStatus: "in_progress",
		}}}

		rows := toonRows(t, decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detail))), "changed")

		if len(rows) != 1 {
			t.Fatalf("changed has %d rows, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{
			"id": "tick-a1b2", "title": "Add retry to the sync worker",
			"from": "open", "to": "in_progress", "auto": false,
		})
	})

	t.Run("it leaves unfiltered output unchanged", func(t *testing.T) {
		detail := richDetail()

		doc := decodeToonDoc(t, formatted(t).of(f.FormatTaskDetail(detail)))

		assertToonKeySet(t, doc, "id", "title", "status", "priority", "type", "parent",
			"created", "updated", "closed", "blocked_by", "children", "tags", "refs", "notes", "description")
		assertToonFields(t, doc, map[string]any{
			"id":          "tick-a1b2",
			"title":       "Add retry to the sync worker",
			"status":      "in_progress",
			"priority":    float64(1),
			"type":        "bug",
			"parent":      "tick-p4r3",
			"created":     "2026-03-01T09:00:00Z",
			"updated":     "2026-03-01T09:00:00Z",
			"closed":      "2026-03-02T09:00:00Z",
			"description": detail.Task.Description,
		})
		assertToonRelatedRow(t, doc, "blocked_by", detail.BlockedBy[0])
		assertToonRelatedRow(t, doc, "children", detail.Children[0])
		assertToonStringList(t, doc, "tags", detail.Tags)
		assertToonStringList(t, doc, "refs", detail.Refs)
		assertToonNoteRows(t, doc, detail.Notes)
	})
}
