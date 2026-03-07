package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestToonFormatterCascadeTransition(t *testing.T) {
	t.Run("it renders downward cancel cascade flat with ParentID present", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-parent1",
			TaskTitle: "Parent",
			OldStatus: "in_progress",
			NewStatus: "cancelled",
			Cascaded: []CascadeEntry{
				{ID: "tick-child1", Title: "Login", ParentID: "tick-parent1", OldStatus: "in_progress", NewStatus: "cancelled"},
				{ID: "tick-child2", Title: "Signup", ParentID: "tick-parent1", OldStatus: "open", NewStatus: "cancelled"},
			},
			Unchanged: []UnchangedEntry{
				{ID: "tick-child3", Title: "Logout", ParentID: "tick-parent1", Status: "done"},
			},
		})
		expected := "tick-parent1: in_progress \u2192 cancelled\n" +
			"tick-child1: in_progress \u2192 cancelled (auto)\n" +
			"tick-child2: open \u2192 cancelled (auto)\n" +
			"tick-child3: done (unchanged)"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it renders upward start cascade", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-child1",
			TaskTitle: "Child",
			OldStatus: "open",
			NewStatus: "in_progress",
			Cascaded: []CascadeEntry{
				{ID: "tick-parent1", Title: "Auth phase", ParentID: "tick-child1", OldStatus: "open", NewStatus: "in_progress"},
				{ID: "tick-grand1", Title: "Sprint 3", ParentID: "tick-child1", OldStatus: "open", NewStatus: "in_progress"},
			},
			Unchanged: nil,
		})
		expected := "tick-child1: open \u2192 in_progress\n" +
			"tick-parent1: open \u2192 in_progress (auto)\n" +
			"tick-grand1: open \u2192 in_progress (auto)"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it renders single cascade entry", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-abc123",
			TaskTitle: "Task",
			OldStatus: "in_progress",
			NewStatus: "done",
			Cascaded: []CascadeEntry{
				{ID: "tick-def456", Title: "Child", ParentID: "tick-abc123", OldStatus: "open", NewStatus: "done"},
			},
			Unchanged: nil,
		})
		expected := "tick-abc123: in_progress \u2192 done\n" +
			"tick-def456: open \u2192 done (auto)"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
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
			Unchanged: []UnchangedEntry{
				{ID: "tick-child3", Title: "Logout", ParentID: "tick-parent1", Status: "done"},
			},
		})
		expected := "tick-parent1: in_progress \u2192 cancelled\n" +
			"\n" +
			"Cascaded:\n" +
			"\u251c\u2500 tick-child1 \"Login\": in_progress \u2192 cancelled\n" +
			"\u251c\u2500 tick-child2 \"Signup\": open \u2192 cancelled\n" +
			"\u2514\u2500 tick-child3 \"Logout\": done (unchanged)"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})

	t.Run("it renders mixed cascaded and unchanged children", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-abc123",
			TaskTitle: "Task",
			OldStatus: "in_progress",
			NewStatus: "done",
			Cascaded: []CascadeEntry{
				{ID: "tick-def456", Title: "Child A", ParentID: "tick-abc123", OldStatus: "open", NewStatus: "done"},
			},
			Unchanged: []UnchangedEntry{
				{ID: "tick-ghi789", Title: "Child B", ParentID: "tick-abc123", Status: "done"},
			},
		})
		// Primary transition line
		if !strings.HasPrefix(result, "tick-abc123: in_progress \u2192 done") {
			t.Errorf("should start with primary transition, got:\n%s", result)
		}
		// Cascaded header
		if !strings.Contains(result, "\nCascaded:\n") {
			t.Errorf("should contain Cascaded: header, got:\n%s", result)
		}
		// Cascaded entry with tree char
		if !strings.Contains(result, "\u251c\u2500 tick-def456 \"Child A\": open \u2192 done") {
			t.Errorf("should contain cascaded entry with tree char, got:\n%s", result)
		}
		// Unchanged entry with tree char
		if !strings.Contains(result, "\u2514\u2500 tick-ghi789 \"Child B\": done (unchanged)") {
			t.Errorf("should contain unchanged entry with tree char, got:\n%s", result)
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
			Unchanged: []UnchangedEntry{
				{ID: "tick-child3", Title: "Logout", ParentID: "tick-parent1", Status: "done"},
			},
		})
		expected := "tick-parent1: in_progress \u2192 cancelled\n" +
			"\n" +
			"Cascaded:\n" +
			"\u251c\u2500 tick-child1 \"Login\": in_progress \u2192 cancelled\n" +
			"\u251c\u2500 tick-child2 \"Signup\": open \u2192 cancelled\n" +
			"\u2502  \u251c\u2500 tick-grand1 \"Form\": open \u2192 cancelled\n" +
			"\u2502  \u2514\u2500 tick-grand2 \"Validation\": open \u2192 cancelled\n" +
			"\u2514\u2500 tick-child3 \"Logout\": done (unchanged)"
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
			Unchanged: nil,
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

	t.Run("it renders unchanged terminal grandchildren in tree", func(t *testing.T) {
		f := &PrettyFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-parent1",
			TaskTitle: "Parent",
			OldStatus: "in_progress",
			NewStatus: "done",
			Cascaded: []CascadeEntry{
				{ID: "tick-child1", Title: "Child A", ParentID: "tick-parent1", OldStatus: "open", NewStatus: "done"},
				{ID: "tick-grand1", Title: "Grand A", ParentID: "tick-child1", OldStatus: "open", NewStatus: "done"},
			},
			Unchanged: []UnchangedEntry{
				{ID: "tick-grand2", Title: "Grand B", ParentID: "tick-child1", Status: "cancelled"},
			},
		})
		expected := "tick-parent1: in_progress \u2192 done\n" +
			"\n" +
			"Cascaded:\n" +
			"\u2514\u2500 tick-child1 \"Child A\": open \u2192 done\n" +
			"   \u251c\u2500 tick-grand1 \"Grand A\": open \u2192 done\n" +
			"   \u2514\u2500 tick-grand2 \"Grand B\": cancelled (unchanged)"
		if result != expected {
			t.Errorf("result:\n%s\nwant:\n%s", result, expected)
		}
	})
}

func TestJSONFormatterCascadeTransition(t *testing.T) {
	t.Run("it renders cascade as structured object", func(t *testing.T) {
		f := &JSONFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-abc123",
			TaskTitle: "Parent",
			OldStatus: "in_progress",
			NewStatus: "done",
			Cascaded: []CascadeEntry{
				{ID: "tick-def456", Title: "Child", OldStatus: "open", NewStatus: "done"},
			},
			Unchanged: []UnchangedEntry{
				{ID: "tick-ghi789", Title: "Other", Status: "done"},
			},
		})

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// Verify transition object
		transition, ok := parsed["transition"].(map[string]interface{})
		if !ok {
			t.Fatalf("transition should be object, got %T: %v", parsed["transition"], parsed["transition"])
		}
		if transition["id"] != "tick-abc123" {
			t.Errorf("transition.id = %v, want %q", transition["id"], "tick-abc123")
		}
		if transition["from"] != "in_progress" {
			t.Errorf("transition.from = %v, want %q", transition["from"], "in_progress")
		}
		if transition["to"] != "done" {
			t.Errorf("transition.to = %v, want %q", transition["to"], "done")
		}

		// Verify cascaded array
		cascaded, ok := parsed["cascaded"].([]interface{})
		if !ok {
			t.Fatalf("cascaded should be array, got %T: %v", parsed["cascaded"], parsed["cascaded"])
		}
		if len(cascaded) != 1 {
			t.Fatalf("cascaded length = %d, want 1", len(cascaded))
		}
		entry := cascaded[0].(map[string]interface{})
		if entry["id"] != "tick-def456" {
			t.Errorf("cascaded[0].id = %v, want %q", entry["id"], "tick-def456")
		}
		if entry["title"] != "Child" {
			t.Errorf("cascaded[0].title = %v, want %q", entry["title"], "Child")
		}
		if entry["from"] != "open" {
			t.Errorf("cascaded[0].from = %v, want %q", entry["from"], "open")
		}
		if entry["to"] != "done" {
			t.Errorf("cascaded[0].to = %v, want %q", entry["to"], "done")
		}

		// Verify unchanged array
		unchanged, ok := parsed["unchanged"].([]interface{})
		if !ok {
			t.Fatalf("unchanged should be array, got %T: %v", parsed["unchanged"], parsed["unchanged"])
		}
		if len(unchanged) != 1 {
			t.Fatalf("unchanged length = %d, want 1", len(unchanged))
		}
		unch := unchanged[0].(map[string]interface{})
		if unch["id"] != "tick-ghi789" {
			t.Errorf("unchanged[0].id = %v, want %q", unch["id"], "tick-ghi789")
		}
		if unch["title"] != "Other" {
			t.Errorf("unchanged[0].title = %v, want %q", unch["title"], "Other")
		}
		if unch["status"] != "done" {
			t.Errorf("unchanged[0].status = %v, want %q", unch["status"], "done")
		}
	})

	t.Run("it renders empty cascaded array as []", func(t *testing.T) {
		f := &JSONFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-abc123",
			TaskTitle: "Task",
			OldStatus: "open",
			NewStatus: "done",
			Cascaded:  nil,
			Unchanged: []UnchangedEntry{
				{ID: "tick-def456", Title: "Child", Status: "done"},
			},
		})

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		cascaded, ok := parsed["cascaded"].([]interface{})
		if !ok {
			t.Fatalf("cascaded should be array (not null), got %T: %v", parsed["cascaded"], parsed["cascaded"])
		}
		if len(cascaded) != 0 {
			t.Errorf("cascaded should be empty, got %d items", len(cascaded))
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
		cr := buildCascadeResult("tick-parent1", "Parent", result, cascades, tasks)

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

	t.Run("it collects unchanged terminal descendants recursively", func(t *testing.T) {
		parent := task.Task{ID: "tick-parent1", Title: "Parent", Status: task.StatusDone, Created: now, Updated: now}
		child1 := task.Task{ID: "tick-child1", Title: "Child A", Status: task.StatusDone, Parent: "tick-parent1", Created: now, Updated: now}
		grand1 := task.Task{ID: "tick-grand1", Title: "Grand A", Status: task.StatusDone, Parent: "tick-child1", Created: now, Updated: now}
		grandUnchanged := task.Task{ID: "tick-grand2", Title: "Grand B", Status: task.StatusCancelled, Parent: "tick-child1", Created: now, Updated: now}
		tasks := []task.Task{parent, child1, grand1, grandUnchanged}

		cascades := []task.CascadeChange{
			{Task: &tasks[1], Action: "done", OldStatus: task.StatusOpen, NewStatus: task.StatusDone},
			{Task: &tasks[2], Action: "done", OldStatus: task.StatusOpen, NewStatus: task.StatusDone},
		}

		result := task.TransitionResult{OldStatus: task.StatusInProgress, NewStatus: task.StatusDone}
		cr := buildCascadeResult("tick-parent1", "Parent", result, cascades, tasks)

		// tick-grand2 is a terminal grandchild not in cascades, should be in Unchanged
		if len(cr.Unchanged) != 1 {
			t.Fatalf("unchanged length = %d, want 1", len(cr.Unchanged))
		}
		if cr.Unchanged[0].ID != "tick-grand2" {
			t.Errorf("unchanged[0].ID = %q, want %q", cr.Unchanged[0].ID, "tick-grand2")
		}
		if cr.Unchanged[0].ParentID != "tick-child1" {
			t.Errorf("unchanged[0].ParentID = %q, want %q", cr.Unchanged[0].ParentID, "tick-child1")
		}
	})

	t.Run("it populates ParentID on unchanged entries for direct children", func(t *testing.T) {
		parent := task.Task{ID: "tick-parent1", Title: "Parent", Status: task.StatusCancelled, Created: now, Updated: now}
		child1 := task.Task{ID: "tick-child1", Title: "Login", Status: task.StatusCancelled, Parent: "tick-parent1", Created: now, Updated: now}
		childDone := task.Task{ID: "tick-child2", Title: "Done child", Status: task.StatusDone, Parent: "tick-parent1", Created: now, Updated: now}
		tasks := []task.Task{parent, child1, childDone}

		cascades := []task.CascadeChange{
			{Task: &tasks[1], Action: "cancel", OldStatus: task.StatusOpen, NewStatus: task.StatusCancelled},
		}

		result := task.TransitionResult{OldStatus: task.StatusInProgress, NewStatus: task.StatusCancelled}
		cr := buildCascadeResult("tick-parent1", "Parent", result, cascades, tasks)

		if len(cr.Unchanged) != 1 {
			t.Fatalf("unchanged length = %d, want 1", len(cr.Unchanged))
		}
		if cr.Unchanged[0].ParentID != "tick-parent1" {
			t.Errorf("unchanged[0].ParentID = %q, want %q", cr.Unchanged[0].ParentID, "tick-parent1")
		}
	})
}

func TestAllFormattersCascadeEmptyArrays(t *testing.T) {
	t.Run("all formatters handle both empty cascaded and unchanged", func(t *testing.T) {
		result := CascadeResult{
			TaskID:    "tick-abc123",
			TaskTitle: "Task",
			OldStatus: "open",
			NewStatus: "done",
			Cascaded:  nil,
			Unchanged: nil,
		}

		// Toon: should just show primary transition
		toon := (&ToonFormatter{}).FormatCascadeTransition(result)
		expected := "tick-abc123: open \u2192 done"
		if toon != expected {
			t.Errorf("ToonFormatter result = %q, want %q", toon, expected)
		}

		// Pretty: should just show primary transition (no Cascaded: header)
		pretty := (&PrettyFormatter{}).FormatCascadeTransition(result)
		if pretty != expected {
			t.Errorf("PrettyFormatter result = %q, want %q", pretty, expected)
		}

		// JSON: should have empty arrays
		jsonResult := (&JSONFormatter{}).FormatCascadeTransition(result)
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonResult), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, jsonResult)
		}

		cascaded, ok := parsed["cascaded"].([]interface{})
		if !ok {
			t.Fatalf("cascaded should be array, got %T: %v", parsed["cascaded"], parsed["cascaded"])
		}
		if len(cascaded) != 0 {
			t.Errorf("cascaded should be empty, got %d", len(cascaded))
		}

		unchanged, ok := parsed["unchanged"].([]interface{})
		if !ok {
			t.Fatalf("unchanged should be array, got %T: %v", parsed["unchanged"], parsed["unchanged"])
		}
		if len(unchanged) != 0 {
			t.Errorf("unchanged should be empty, got %d", len(unchanged))
		}
	})
}
