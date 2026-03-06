package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToonFormatterCascadeTransition(t *testing.T) {
	t.Run("it renders downward cancel cascade", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatCascadeTransition(CascadeResult{
			TaskID:    "tick-parent1",
			TaskTitle: "Parent",
			OldStatus: "in_progress",
			NewStatus: "cancelled",
			Cascaded: []CascadeEntry{
				{ID: "tick-child1", Title: "Login", OldStatus: "in_progress", NewStatus: "cancelled"},
				{ID: "tick-child2", Title: "Signup", OldStatus: "open", NewStatus: "cancelled"},
			},
			Unchanged: []UnchangedEntry{
				{ID: "tick-child3", Title: "Logout", Status: "done"},
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
				{ID: "tick-parent1", Title: "Auth phase", OldStatus: "open", NewStatus: "in_progress"},
				{ID: "tick-grand1", Title: "Sprint 3", OldStatus: "open", NewStatus: "in_progress"},
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
				{ID: "tick-def456", Title: "Child", OldStatus: "open", NewStatus: "done"},
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
				{ID: "tick-child1", Title: "Login", OldStatus: "in_progress", NewStatus: "cancelled"},
				{ID: "tick-child2", Title: "Signup", OldStatus: "open", NewStatus: "cancelled"},
			},
			Unchanged: []UnchangedEntry{
				{ID: "tick-child3", Title: "Logout", Status: "done"},
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
				{ID: "tick-def456", Title: "Child A", OldStatus: "open", NewStatus: "done"},
			},
			Unchanged: []UnchangedEntry{
				{ID: "tick-ghi789", Title: "Child B", Status: "done"},
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
