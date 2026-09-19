package cli

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestToonFormatterCascadeTransition(t *testing.T) {
	t.Run("it renders a downward cancel cascade as one changed table", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := f.FormatCascadeTransition(CascadeResult{
			Changed: []StatusChange{
				{ID: "tick-parent1", Title: "Parent", From: "in_progress", To: "cancelled"},
				{ID: "tick-child1", Title: "Login", From: "in_progress", To: "cancelled", Auto: true},
				{ID: "tick-child2", Title: "Signup", From: "open", To: "cancelled", Auto: true},
			},
		})

		rows := toonRows(t, decodeToonDoc(t, doc), "changed")
		if len(rows) != 3 {
			t.Fatalf("changed has %d rows, want 3", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{"id": "tick-parent1", "title": "Parent", "from": "in_progress", "to": "cancelled", "auto": false})
		assertToonFields(t, rows[1], map[string]any{"id": "tick-child1", "title": "Login", "from": "in_progress", "to": "cancelled", "auto": true})
		assertToonFields(t, rows[2], map[string]any{"id": "tick-child2", "title": "Signup", "from": "open", "to": "cancelled", "auto": true})
	})

	t.Run("it renders an upward start cascade as one changed table", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := f.FormatCascadeTransition(CascadeResult{
			Changed: []StatusChange{
				{ID: "tick-child1", Title: "Child", From: "open", To: "in_progress"},
				{ID: "tick-parent1", Title: "Auth phase", From: "open", To: "in_progress", Auto: true},
				{ID: "tick-grand1", Title: "Sprint 3", From: "open", To: "in_progress", Auto: true},
			},
		})

		rows := toonRows(t, decodeToonDoc(t, doc), "changed")
		if len(rows) != 3 {
			t.Fatalf("changed has %d rows, want 3", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{"id": "tick-child1", "auto": false})
		assertToonFields(t, rows[1], map[string]any{"id": "tick-parent1", "auto": true})
		assertToonFields(t, rows[2], map[string]any{"id": "tick-grand1", "auto": true})
	})

	t.Run("it renders a single change as a one-row changed table", func(t *testing.T) {
		f := &ToonFormatter{}
		doc := f.FormatCascadeTransition(CascadeResult{
			Changed: []StatusChange{
				{ID: "tick-abc123", Title: "Task", From: "in_progress", To: "done"},
			},
		})

		rows := toonRows(t, decodeToonDoc(t, doc), "changed")
		if len(rows) != 1 {
			t.Fatalf("changed has %d rows, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{"id": "tick-abc123", "title": "Task", "from": "in_progress", "to": "done", "auto": false})
	})
}

func TestPrettyFormatterCascadeTransition(t *testing.T) {
	t.Run("it renders downward cancel cascade with tree", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-parent1",
			TaskTitle: "Parent",
			OldStatus: "in_progress",
			NewStatus: "cancelled",
			Cascaded: []CascadeEntry{
				{ID: "tick-child1", Title: "Login", ParentID: "tick-parent1", OldStatus: "in_progress", NewStatus: "cancelled"},
				{ID: "tick-child2", Title: "Signup", ParentID: "tick-parent1", OldStatus: "open", NewStatus: "cancelled"},
			},
		})
		expected := "tick-parent1: in_progress \u2192 cancelled\n" +
			"\n" +
			"Cascaded:\n" +
			"\u251c\u2500 tick-child1 \"Login\": in_progress \u2192 cancelled\n" +
			"\u2514\u2500 tick-child2 \"Signup\": open \u2192 cancelled"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it renders downward cascade with 3-level hierarchy", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-parent1",
			TaskTitle: "Parent",
			OldStatus: "in_progress",
			NewStatus: "cancelled",
			Cascaded: []CascadeEntry{
				{ID: "tick-child1", Title: "Login", ParentID: "tick-parent1", OldStatus: "in_progress", NewStatus: "cancelled"},
				{ID: "tick-child2", Title: "Signup", ParentID: "tick-parent1", OldStatus: "open", NewStatus: "cancelled"},
				{ID: "tick-grand1", Title: "Form", ParentID: "tick-child2", OldStatus: "open", NewStatus: "cancelled"},
				{ID: "tick-grand2", Title: "Validation", ParentID: "tick-child2", OldStatus: "open", NewStatus: "cancelled"},
			},
		})
		expected := "tick-parent1: in_progress \u2192 cancelled\n" +
			"\n" +
			"Cascaded:\n" +
			"\u251c\u2500 tick-child1 \"Login\": in_progress \u2192 cancelled\n" +
			"\u2514\u2500 tick-child2 \"Signup\": open \u2192 cancelled\n" +
			"   \u251c\u2500 tick-grand1 \"Form\": open \u2192 cancelled\n" +
			"   \u2514\u2500 tick-grand2 \"Validation\": open \u2192 cancelled"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it renders upward cascade with grandparent chain", func(t *testing.T) {
		f := &PrettyFormatter{}
		// Upward cascades: child started -> parent started -> grandparent started
		// In upward cascades, each entry's ParentID is the primary task's ID since
		// they form a chain (each is the ancestor of the previous).
		// The chain is: child -> parent -> grandparent, rendered flat since each
		// cascaded task is at a different level of the ancestor chain.
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-child1",
			TaskTitle: "Child",
			OldStatus: "open",
			NewStatus: "in_progress",
			Cascaded: []CascadeEntry{
				{ID: "tick-parent1", Title: "Auth phase", ParentID: "tick-child1", OldStatus: "open", NewStatus: "in_progress"},
				{ID: "tick-grand1", Title: "Sprint 3", ParentID: "tick-child1", OldStatus: "open", NewStatus: "in_progress"},
			},
		})
		expected := "tick-child1: open \u2192 in_progress\n" +
			"\n" +
			"Cascaded:\n" +
			"\u251c\u2500 tick-parent1 \"Auth phase\": open \u2192 in_progress\n" +
			"\u2514\u2500 tick-grand1 \"Sprint 3\": open \u2192 in_progress"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

}

func decodeChangedDoc(t *testing.T, doc string) map[string]any {
	t.Helper()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(doc), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\ndocument: %s", err, doc)
	}
	if len(parsed) != 1 {
		t.Errorf("document has keys %v, want changed only", slices.Sorted(maps.Keys(parsed)))
	}
	return parsed
}

func changedRows(t *testing.T, doc string) []any {
	t.Helper()
	parsed := decodeChangedDoc(t, doc)
	rows, ok := parsed["changed"].([]any)
	if !ok {
		t.Fatalf("changed = %#v, want an array", parsed["changed"])
	}
	return rows
}

func assertChangedRow(t *testing.T, row any, want map[string]any) {
	t.Helper()
	got, ok := row.(map[string]any)
	if !ok {
		t.Fatalf("row = %#v, want an object", row)
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Errorf("row[%q] = %#v, want %#v", key, got[key], wantValue)
		}
	}
	if _, isBool := got["auto"].(bool); !isBool {
		t.Errorf("row auto = %#v (%T), want a JSON boolean", got["auto"], got["auto"])
	}
}

func TestJSONFormatterCascadeTransition(t *testing.T) {
	t.Run("it renders a single transition as a one-element changed list", func(t *testing.T) {
		f := &JSONFormatter{}
		rows := changedRows(t, f.FormatCascadeTransition(CascadeResult{
			Changed: []StatusChange{
				{ID: "tick-abc123", Title: "Solo", From: "open", To: "in_progress"},
			},
		}))

		if len(rows) != 1 {
			t.Fatalf("changed has %d rows, want 1", len(rows))
		}
		assertChangedRow(t, rows[0], map[string]any{
			"id": "tick-abc123", "title": "Solo", "from": "open", "to": "in_progress", "auto": false,
		})
	})

	t.Run("it renders a cascade as one changed list", func(t *testing.T) {
		f := &JSONFormatter{}
		rows := changedRows(t, f.FormatCascadeTransition(CascadeResult{
			Changed: []StatusChange{
				{ID: "tick-abc123", Title: "Parent", From: "in_progress", To: "done"},
				{ID: "tick-def456", Title: "Child", From: "open", To: "done", Auto: true},
			},
		}))

		if len(rows) != 2 {
			t.Fatalf("changed has %d rows, want 2", len(rows))
		}
		assertChangedRow(t, rows[0], map[string]any{
			"id": "tick-abc123", "title": "Parent", "from": "in_progress", "to": "done", "auto": false,
		})
		assertChangedRow(t, rows[1], map[string]any{
			"id": "tick-def456", "title": "Child", "from": "open", "to": "done", "auto": true,
		})
	})

	t.Run("it emits changed as an empty array never null", func(t *testing.T) {
		f := &JSONFormatter{}
		parsed := decodeChangedDoc(t, f.FormatCascadeTransition(CascadeResult{}))

		rows, ok := parsed["changed"].([]any)
		if !ok {
			t.Fatalf("changed = %#v, want an array", parsed["changed"])
		}
		if rows == nil {
			t.Error("changed decoded to a nil slice, want a non-nil empty slice")
		}
		if len(rows) != 0 {
			t.Errorf("changed has %d rows, want 0", len(rows))
		}
	})
}

func TestBuildCascadeResult(t *testing.T) {
	now := time.Now()

	t.Run("it populates ParentID on cascade entries", func(t *testing.T) {
		parent := task.Task{ID: "tick-parent1", Title: "Parent", Status: task.StatusCancelled, Created: now, Updated: now}
		child1 := task.Task{ID: "tick-child1", Title: "Login", Status: task.StatusCancelled, Parent: "tick-parent1", Created: now, Updated: now}
		child2 := task.Task{ID: "tick-child2", Title: "Signup", Status: task.StatusCancelled, Parent: "tick-parent1", Created: now, Updated: now}
		tasks := []task.Task{parent, child1, child2}

		cascades := []task.CascadeChange{
			{Task: &tasks[1], Action: "cancel", OldStatus: task.StatusInProgress, NewStatus: task.StatusCancelled},
			{Task: &tasks[2], Action: "cancel", OldStatus: task.StatusOpen, NewStatus: task.StatusCancelled},
		}

		result := task.TransitionResult{OldStatus: task.StatusInProgress, NewStatus: task.StatusCancelled}
		cr := buildCascadeResult("tick-parent1", "Parent", result, cascades, tasks, false)

		if len(cr.Cascaded) != 2 {
			t.Fatalf("cascaded length = %d, want 2", len(cr.Cascaded))
		}
		if cr.Cascaded[0].ParentID != "tick-parent1" {
			t.Errorf("cascaded[0].ParentID = %q, want %q", cr.Cascaded[0].ParentID, "tick-parent1")
		}
		if cr.Cascaded[1].ParentID != "tick-parent1" {
			t.Errorf("cascaded[1].ParentID = %q, want %q", cr.Cascaded[1].ParentID, "tick-parent1")
		}
	})

}

func TestAllFormattersCascadeEmptyArrays(t *testing.T) {
	t.Run("all formatters handle empty cascaded", func(t *testing.T) {
		result := CascadeResult{
			TaskID:    "tick-abc123",
			TaskTitle: "Task",
			OldStatus: "open",
			NewStatus: "done",
			Cascaded:  nil,
			Changed:   []StatusChange{{ID: "tick-abc123", Title: "Task", From: "open", To: "done"}},
		}

		// Toon: the changed table carries only the requested row.
		rows := toonRows(t, decodeToonDoc(t, (&ToonFormatter{}).FormatCascadeTransition(result)), "changed")
		if len(rows) != 1 {
			t.Fatalf("changed has %d rows, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{"id": "tick-abc123", "title": "Task", "from": "open", "to": "done", "auto": false})

		// Pretty: should just show primary transition (no Cascaded: header)
		pretty := (&PrettyFormatter{}).FormatCascadeTransition(result)
		expected := "tick-abc123: open \u2192 done"
		if pretty != expected {
			t.Errorf("PrettyFormatter result = %q, want %q", pretty, expected)
		}

		jsonRows := changedRows(t, (&JSONFormatter{}).FormatCascadeTransition(result))
		if len(jsonRows) != 1 {
			t.Fatalf("changed has %d rows, want 1", len(jsonRows))
		}
		assertChangedRow(t, jsonRows[0], map[string]any{
			"id": "tick-abc123", "title": "Task", "from": "open", "to": "done", "auto": false,
		})
	})
}
