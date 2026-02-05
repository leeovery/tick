package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONFormatter(t *testing.T) {
	t.Run("it implements Formatter interface", func(t *testing.T) {
		var _ Formatter = &JSONFormatter{}
	})
}

func TestJSONFormatterFormatTaskList(t *testing.T) {
	t.Run("it formats list as JSON array", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
				{ID: "tick-c3d4", Title: "Login endpoint", Status: "open", Priority: 2},
			},
		}

		result := f.FormatTaskList(data)

		// Should be valid JSON
		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Should have 2 items
		if len(parsed) != 2 {
			t.Errorf("expected 2 items, got %d", len(parsed))
		}

		// Check first item has snake_case keys
		first := parsed[0]
		if first["id"] != "tick-a1b2" {
			t.Errorf("expected id 'tick-a1b2', got %v", first["id"])
		}
		if first["title"] != "Setup Sanctum" {
			t.Errorf("expected title 'Setup Sanctum', got %v", first["title"])
		}
		if first["status"] != "done" {
			t.Errorf("expected status 'done', got %v", first["status"])
		}
		// JSON unmarshals numbers as float64
		if first["priority"] != float64(1) {
			t.Errorf("expected priority 1, got %v", first["priority"])
		}
	})

	t.Run("it formats empty list as [] not null", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{},
		}

		result := f.FormatTaskList(data)

		// Should be [] not null
		trimmed := strings.TrimSpace(result)
		if trimmed != "[]" {
			t.Errorf("expected '[]', got %q", trimmed)
		}

		// Should be valid JSON
		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Should be empty array, not nil
		if parsed == nil {
			t.Error("expected empty array, got nil")
		}
		if len(parsed) != 0 {
			t.Errorf("expected 0 items, got %d", len(parsed))
		}
	})

	t.Run("it formats nil TaskListData as []", func(t *testing.T) {
		f := &JSONFormatter{}

		result := f.FormatTaskList(nil)

		trimmed := strings.TrimSpace(result)
		if trimmed != "[]" {
			t.Errorf("expected '[]', got %q", trimmed)
		}
	})
}

func TestJSONFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all fields", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Setup Sanctum",
			Status:      "in_progress",
			Priority:    1,
			Description: "Full description",
			Parent:      "tick-e5f6",
			ParentTitle: "Auth System",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "2026-01-19T16:00:00Z",
			BlockedBy: []RelatedTaskData{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
			},
			Children: []RelatedTaskData{
				{ID: "tick-j1k2", Title: "Child task", Status: "open"},
			},
		}

		result := f.FormatTaskDetail(data)

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Check all fields present with snake_case keys
		if parsed["id"] != "tick-a1b2" {
			t.Errorf("expected id 'tick-a1b2', got %v", parsed["id"])
		}
		if parsed["title"] != "Setup Sanctum" {
			t.Errorf("expected title 'Setup Sanctum', got %v", parsed["title"])
		}
		if parsed["status"] != "in_progress" {
			t.Errorf("expected status 'in_progress', got %v", parsed["status"])
		}
		if parsed["priority"] != float64(1) {
			t.Errorf("expected priority 1, got %v", parsed["priority"])
		}
		if parsed["description"] != "Full description" {
			t.Errorf("expected description 'Full description', got %v", parsed["description"])
		}
		if parsed["parent"] != "tick-e5f6" {
			t.Errorf("expected parent 'tick-e5f6', got %v", parsed["parent"])
		}
		if parsed["created"] != "2026-01-19T10:00:00Z" {
			t.Errorf("expected created timestamp, got %v", parsed["created"])
		}
		if parsed["updated"] != "2026-01-19T14:30:00Z" {
			t.Errorf("expected updated timestamp, got %v", parsed["updated"])
		}
		if parsed["closed"] != "2026-01-19T16:00:00Z" {
			t.Errorf("expected closed timestamp, got %v", parsed["closed"])
		}

		// Check blocked_by is array with item
		blockedBy, ok := parsed["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by should be array, got %T", parsed["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Errorf("expected 1 blocker, got %d", len(blockedBy))
		}

		// Check children is array with item
		children, ok := parsed["children"].([]interface{})
		if !ok {
			t.Fatalf("children should be array, got %T", parsed["children"])
		}
		if len(children) != 1 {
			t.Errorf("expected 1 child, got %d", len(children))
		}
	})

	t.Run("it omits parent/closed when null", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Root task",
			Status:      "open",
			Priority:    2,
			Description: "",
			Parent:      "", // No parent
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "", // Not closed
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// parent should be OMITTED (not present), not null
		if _, exists := parsed["parent"]; exists {
			t.Error("parent should be omitted when empty, not present as null")
		}

		// closed should be OMITTED (not present), not null
		if _, exists := parsed["closed"]; exists {
			t.Error("closed should be omitted when empty, not present as null")
		}
	})

	t.Run("it includes blocked_by/children as empty arrays", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Solo task",
			Status:      "open",
			Priority:    2,
			Description: "",
			Parent:      "",
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// blocked_by should ALWAYS be present as [] when empty
		blockedBy, exists := parsed["blocked_by"]
		if !exists {
			t.Fatal("blocked_by should always be present")
		}
		blockedByArr, ok := blockedBy.([]interface{})
		if !ok {
			t.Fatalf("blocked_by should be array, got %T", blockedBy)
		}
		if len(blockedByArr) != 0 {
			t.Errorf("blocked_by should be empty array, got %d items", len(blockedByArr))
		}

		// children should ALWAYS be present as [] when empty
		children, exists := parsed["children"]
		if !exists {
			t.Fatal("children should always be present")
		}
		childrenArr, ok := children.([]interface{})
		if !ok {
			t.Fatalf("children should be array, got %T", children)
		}
		if len(childrenArr) != 0 {
			t.Errorf("children should be empty array, got %d items", len(childrenArr))
		}
	})

	t.Run("it formats description as empty string not null", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "No description task",
			Status:      "open",
			Priority:    2,
			Description: "", // Empty, not null
			Parent:      "",
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// description should ALWAYS be present (even when empty)
		desc, exists := parsed["description"]
		if !exists {
			t.Fatal("description should always be present")
		}
		if desc != "" {
			t.Errorf("description should be empty string, got %v", desc)
		}
	})

	t.Run("it uses snake_case for all keys", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Task with blockers",
			Status:      "open",
			Priority:    2,
			Description: "Desc",
			Parent:      "tick-e5f6",
			ParentTitle: "Parent",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "2026-01-19T16:00:00Z",
			BlockedBy: []RelatedTaskData{
				{ID: "tick-c3d4", Title: "Blocker", Status: "done"},
			},
			Children: []RelatedTaskData{
				{ID: "tick-j1k2", Title: "Child", Status: "open"},
			},
		}

		result := f.FormatTaskDetail(data)

		// Check raw JSON for snake_case keys (not camelCase)
		if strings.Contains(result, "blockedBy") {
			t.Error("should use snake_case 'blocked_by', not camelCase 'blockedBy'")
		}
		if strings.Contains(result, "parentTitle") {
			t.Error("should not include parentTitle in output")
		}
		if !strings.Contains(result, "blocked_by") {
			t.Error("should contain 'blocked_by' key")
		}
	})
}

func TestJSONFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats as structured nested object", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &StatsData{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: []PriorityCount{
				{Priority: 0, Count: 2},
				{Priority: 1, Count: 8},
				{Priority: 2, Count: 25},
				{Priority: 3, Count: 7},
				{Priority: 4, Count: 5},
			},
		}

		result := f.FormatStats(data)

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Check top-level fields
		if parsed["total"] != float64(47) {
			t.Errorf("expected total 47, got %v", parsed["total"])
		}

		// Check by_status nested object
		byStatus, ok := parsed["by_status"].(map[string]interface{})
		if !ok {
			t.Fatalf("by_status should be object, got %T", parsed["by_status"])
		}
		if byStatus["open"] != float64(12) {
			t.Errorf("expected open 12, got %v", byStatus["open"])
		}
		if byStatus["in_progress"] != float64(3) {
			t.Errorf("expected in_progress 3, got %v", byStatus["in_progress"])
		}
		if byStatus["done"] != float64(28) {
			t.Errorf("expected done 28, got %v", byStatus["done"])
		}
		if byStatus["cancelled"] != float64(4) {
			t.Errorf("expected cancelled 4, got %v", byStatus["cancelled"])
		}
		if byStatus["ready"] != float64(8) {
			t.Errorf("expected ready 8, got %v", byStatus["ready"])
		}
		if byStatus["blocked"] != float64(4) {
			t.Errorf("expected blocked 4, got %v", byStatus["blocked"])
		}

		// Check by_priority
		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority should be array, got %T", parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Errorf("by_priority should have 5 entries, got %d", len(byPriority))
		}
	})

	t.Run("it includes 5 priority rows even at zero", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &StatsData{
			Total:      10,
			Open:       10,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      10,
			Blocked:    0,
			ByPriority: []PriorityCount{
				{Priority: 0, Count: 0},
				{Priority: 1, Count: 0},
				{Priority: 2, Count: 10},
				{Priority: 3, Count: 0},
				{Priority: 4, Count: 0},
			},
		}

		result := f.FormatStats(data)

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Check by_priority has all 5 entries
		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority should be array, got %T", parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Errorf("by_priority should always have 5 entries, got %d", len(byPriority))
		}

		// Check each priority level is present
		for i, item := range byPriority {
			entry, ok := item.(map[string]interface{})
			if !ok {
				t.Fatalf("priority entry should be object, got %T", item)
			}
			if entry["priority"] != float64(i) {
				t.Errorf("expected priority %d, got %v", i, entry["priority"])
			}
		}
	})
}

func TestJSONFormatterFormatTransitionDepMessage(t *testing.T) {
	t.Run("it formats transition as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}

		result := f.FormatTransition("tick-a3f2b7", "open", "in_progress")

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Check fields
		if parsed["id"] != "tick-a3f2b7" {
			t.Errorf("expected id 'tick-a3f2b7', got %v", parsed["id"])
		}
		if parsed["from"] != "open" {
			t.Errorf("expected from 'open', got %v", parsed["from"])
		}
		if parsed["to"] != "in_progress" {
			t.Errorf("expected to 'in_progress', got %v", parsed["to"])
		}
	})

	t.Run("it formats dep add as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}

		result := f.FormatDepChange("add", "tick-c3d4", "tick-a1b2")

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Check fields
		if parsed["action"] != "add" {
			t.Errorf("expected action 'add', got %v", parsed["action"])
		}
		if parsed["task_id"] != "tick-c3d4" {
			t.Errorf("expected task_id 'tick-c3d4', got %v", parsed["task_id"])
		}
		if parsed["blocked_by"] != "tick-a1b2" {
			t.Errorf("expected blocked_by 'tick-a1b2', got %v", parsed["blocked_by"])
		}
	})

	t.Run("it formats dep rm as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}

		result := f.FormatDepChange("remove", "tick-c3d4", "tick-a1b2")

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Check fields
		if parsed["action"] != "remove" {
			t.Errorf("expected action 'remove', got %v", parsed["action"])
		}
		if parsed["task_id"] != "tick-c3d4" {
			t.Errorf("expected task_id 'tick-c3d4', got %v", parsed["task_id"])
		}
		if parsed["blocked_by"] != "tick-a1b2" {
			t.Errorf("expected blocked_by 'tick-a1b2', got %v", parsed["blocked_by"])
		}
	})

	t.Run("it formats message as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}

		result := f.FormatMessage("No tasks found.")

		// Should be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// Check field
		if parsed["message"] != "No tasks found." {
			t.Errorf("expected message 'No tasks found.', got %v", parsed["message"])
		}
	})
}

func TestJSONFormatterProducesValidJSON(t *testing.T) {
	t.Run("it produces valid parseable JSON for list", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Task with \"quotes\" and \\ backslash", Status: "open", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Errorf("output should be valid JSON, got error: %v", err)
		}

		// Verify special characters survived round-trip
		if len(parsed) != 1 {
			t.Fatalf("expected 1 task, got %d", len(parsed))
		}
		if parsed[0]["title"] != "Task with \"quotes\" and \\ backslash" {
			t.Errorf("special characters should survive JSON round-trip, got %v", parsed[0]["title"])
		}
	})

	t.Run("it produces valid parseable JSON for detail", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Task with\nnewline",
			Status:      "open",
			Priority:    2,
			Description: "Line 1\nLine 2\nLine 3",
			Parent:      "",
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Errorf("output should be valid JSON, got error: %v", err)
		}

		// Verify newlines survived round-trip
		if !strings.Contains(parsed["description"].(string), "\n") {
			t.Error("newlines should survive in description")
		}
	})

	t.Run("it produces valid parseable JSON for stats", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &StatsData{
			Total:      0,
			Open:       0,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      0,
			Blocked:    0,
			ByPriority: []PriorityCount{
				{Priority: 0, Count: 0},
				{Priority: 1, Count: 0},
				{Priority: 2, Count: 0},
				{Priority: 3, Count: 0},
				{Priority: 4, Count: 0},
			},
		}

		result := f.FormatStats(data)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Errorf("output should be valid JSON, got error: %v", err)
		}
	})
}

func TestJSONFormatterUses2SpaceIndent(t *testing.T) {
	t.Run("it uses 2-space indent for list", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskListData{
			Tasks: []TaskRowData{
				{ID: "tick-a1b2", Title: "Test", Status: "open", Priority: 1},
			},
		}

		result := f.FormatTaskList(data)

		// For array of objects, structure is:
		// [
		//   {           <- 2 spaces for array item
		//     "id": ... <- 4 spaces (2 for array + 2 for object property)
		//   }
		// ]
		// Check for 2-space indent on the opening brace (array item level)
		if !strings.Contains(result, "\n  {") {
			t.Error("expected 2-space indent for array items")
		}
		// Check for 4-space indent on object properties (2 levels deep)
		if !strings.Contains(result, "\n    \"id\"") {
			t.Error("expected 4-space indent for object properties (2-space per level)")
		}
		// Should not use tabs
		if strings.Contains(result, "\t") {
			t.Error("should not use tab indent")
		}
	})

	t.Run("it uses 2-space indent for detail", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &TaskDetailData{
			ID:          "tick-a1b2",
			Title:       "Test",
			Status:      "open",
			Priority:    2,
			Description: "",
			Parent:      "",
			ParentTitle: "",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "",
			BlockedBy:   []RelatedTaskData{},
			Children:    []RelatedTaskData{},
		}

		result := f.FormatTaskDetail(data)

		// Should contain 2-space indent
		if !strings.Contains(result, "  \"id\"") {
			t.Error("expected 2-space indent")
		}
	})

	t.Run("it uses 2-space indent for stats", func(t *testing.T) {
		f := &JSONFormatter{}
		data := &StatsData{
			Total:      10,
			Open:       10,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      10,
			Blocked:    0,
			ByPriority: []PriorityCount{
				{Priority: 0, Count: 0},
				{Priority: 1, Count: 0},
				{Priority: 2, Count: 10},
				{Priority: 3, Count: 0},
				{Priority: 4, Count: 0},
			},
		}

		result := f.FormatStats(data)

		// Should contain 2-space indent
		if !strings.Contains(result, "  \"total\"") {
			t.Error("expected 2-space indent")
		}
	})
}
