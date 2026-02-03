package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestJSONFormatterImplementsInterface(t *testing.T) {
	t.Run("it implements the full Formatter interface", func(t *testing.T) {
		var _ Formatter = &JSONFormatter{}
	})
}

func TestJSONFormatterFormatTaskList(t *testing.T) {
	t.Run("it formats list as JSON array", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		tasks := []TaskRow{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: "open", Priority: 1},
		}

		err := f.FormatTaskList(&buf, tasks)
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		// Must be valid JSON
		var result []map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if len(result) != 2 {
			t.Fatalf("expected 2 items, got %d", len(result))
		}

		// Check first item fields
		if result[0]["id"] != "tick-a1b2" {
			t.Errorf("result[0][id] = %v, want tick-a1b2", result[0]["id"])
		}
		if result[0]["title"] != "Setup Sanctum" {
			t.Errorf("result[0][title] = %v, want Setup Sanctum", result[0]["title"])
		}
		if result[0]["status"] != "done" {
			t.Errorf("result[0][status] = %v, want done", result[0]["status"])
		}
		// JSON numbers are float64
		if result[0]["priority"] != float64(1) {
			t.Errorf("result[0][priority] = %v, want 1", result[0]["priority"])
		}
	})

	t.Run("it formats empty list as [] not null", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, []TaskRow{})
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		got := strings.TrimSpace(buf.String())
		if got != "[]" {
			t.Errorf("FormatTaskList([]) = %q, want %q", got, "[]")
		}
	})

	t.Run("it formats nil list as [] not null", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, nil)
		if err != nil {
			t.Fatalf("FormatTaskList() error: %v", err)
		}

		got := strings.TrimSpace(buf.String())
		if got != "[]" {
			t.Errorf("FormatTaskList(nil) = %q, want %q", got, "[]")
		}
	})
}

func TestJSONFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all fields", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:       "tick-a1b2",
			Title:    "Setup Sanctum",
			Status:   "in_progress",
			Priority: 1,
			Parent:   "tick-e5f6",
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T14:30:00Z",
			Closed:   "2026-01-19T16:00:00Z",
			BlockedBy: []relatedTask{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
			},
			Children: []relatedTask{
				{ID: "tick-g7h8", Title: "Config setup", Status: "in_progress"},
			},
			Description: "Full task description here.",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}

		// Check all fields present
		if result["id"] != "tick-a1b2" {
			t.Errorf("id = %v, want tick-a1b2", result["id"])
		}
		if result["title"] != "Setup Sanctum" {
			t.Errorf("title = %v, want Setup Sanctum", result["title"])
		}
		if result["status"] != "in_progress" {
			t.Errorf("status = %v, want in_progress", result["status"])
		}
		if result["priority"] != float64(1) {
			t.Errorf("priority = %v, want 1", result["priority"])
		}
		if result["parent"] != "tick-e5f6" {
			t.Errorf("parent = %v, want tick-e5f6", result["parent"])
		}
		if result["created"] != "2026-01-19T10:00:00Z" {
			t.Errorf("created = %v, want 2026-01-19T10:00:00Z", result["created"])
		}
		if result["updated"] != "2026-01-19T14:30:00Z" {
			t.Errorf("updated = %v, want 2026-01-19T14:30:00Z", result["updated"])
		}
		if result["closed"] != "2026-01-19T16:00:00Z" {
			t.Errorf("closed = %v, want 2026-01-19T16:00:00Z", result["closed"])
		}
		if result["description"] != "Full task description here." {
			t.Errorf("description = %v, want 'Full task description here.'", result["description"])
		}

		// Check blocked_by is array with 1 item
		blockedBy, ok := result["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", result["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Fatalf("blocked_by length = %d, want 1", len(blockedBy))
		}
		dep := blockedBy[0].(map[string]interface{})
		if dep["id"] != "tick-c3d4" {
			t.Errorf("blocked_by[0].id = %v, want tick-c3d4", dep["id"])
		}

		// Check children is array with 1 item
		children, ok := result["children"].([]interface{})
		if !ok {
			t.Fatalf("children is not an array: %T", result["children"])
		}
		if len(children) != 1 {
			t.Fatalf("children length = %d, want 1", len(children))
		}
	})

	t.Run("it omits parent/closed when null", func(t *testing.T) {
		f := &JSONFormatter{}
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

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// parent and closed should not be present
		if _, exists := result["parent"]; exists {
			t.Errorf("parent should be omitted when empty, got: %v", result["parent"])
		}
		if _, exists := result["closed"]; exists {
			t.Errorf("closed should be omitted when empty, got: %v", result["closed"])
		}
	})

	t.Run("it includes blocked_by/children as empty arrays", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:        "tick-a1b2",
			Title:     "No deps",
			Status:    "open",
			Priority:  2,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
			BlockedBy: nil,
			Children:  nil,
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// blocked_by must be present as []
		blockedBy, ok := result["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T (value: %v)", result["blocked_by"], result["blocked_by"])
		}
		if len(blockedBy) != 0 {
			t.Errorf("blocked_by should be empty, got %d items", len(blockedBy))
		}

		// children must be present as []
		children, ok := result["children"].([]interface{})
		if !ok {
			t.Fatalf("children is not an array: %T (value: %v)", result["children"], result["children"])
		}
		if len(children) != 0 {
			t.Errorf("children should be empty, got %d items", len(children))
		}
	})

	t.Run("it formats description as empty string not null", func(t *testing.T) {
		f := &JSONFormatter{}
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

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		// description must be present as empty string, not null or absent
		desc, exists := result["description"]
		if !exists {
			t.Fatal("description field should be present even when empty")
		}
		descStr, ok := desc.(string)
		if !ok {
			t.Fatalf("description is not a string: %T", desc)
		}
		if descStr != "" {
			t.Errorf("description = %q, want empty string", descStr)
		}
	})
}

func TestJSONFormatterSnakeCase(t *testing.T) {
	t.Run("it uses snake_case for all keys", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &showData{
			ID:       "tick-a1b2",
			Title:    "Test",
			Status:   "open",
			Priority: 2,
			Parent:   "tick-parent",
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
			Closed:   "2026-01-19T16:00:00Z",
			BlockedBy: []relatedTask{
				{ID: "tick-c3d4", Title: "Blocker", Status: "done"},
			},
			Children: []relatedTask{
				{ID: "tick-e5f6", Title: "Child", Status: "open"},
			},
			Description: "desc",
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("FormatTaskDetail() error: %v", err)
		}

		got := buf.String()

		// All keys must be snake_case
		expectedKeys := []string{
			`"id"`, `"title"`, `"status"`, `"priority"`,
			`"parent"`, `"created"`, `"updated"`, `"closed"`,
			`"description"`, `"blocked_by"`, `"children"`,
		}
		for _, key := range expectedKeys {
			if !strings.Contains(got, key) {
				t.Errorf("output missing snake_case key %s\noutput: %s", key, got)
			}
		}

		// Should NOT contain camelCase variants
		if strings.Contains(got, `"blockedBy"`) {
			t.Errorf("output contains camelCase key 'blockedBy' instead of snake_case")
		}
		if strings.Contains(got, `"parentTitle"`) {
			t.Errorf("output contains camelCase key 'parentTitle'")
		}
	})
}

func TestJSONFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats as structured nested object", func(t *testing.T) {
		f := &JSONFormatter{}
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

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}

		// Check top-level total
		if result["total"] != float64(47) {
			t.Errorf("total = %v, want 47", result["total"])
		}

		// Check by_status nested object
		byStatus, ok := result["by_status"].(map[string]interface{})
		if !ok {
			t.Fatalf("by_status is not an object: %T", result["by_status"])
		}
		if byStatus["open"] != float64(12) {
			t.Errorf("by_status.open = %v, want 12", byStatus["open"])
		}
		if byStatus["in_progress"] != float64(3) {
			t.Errorf("by_status.in_progress = %v, want 3", byStatus["in_progress"])
		}
		if byStatus["done"] != float64(28) {
			t.Errorf("by_status.done = %v, want 28", byStatus["done"])
		}
		if byStatus["cancelled"] != float64(4) {
			t.Errorf("by_status.cancelled = %v, want 4", byStatus["cancelled"])
		}

		// Check workflow nested object
		workflow, ok := result["workflow"].(map[string]interface{})
		if !ok {
			t.Fatalf("workflow is not an object: %T", result["workflow"])
		}
		if workflow["ready"] != float64(8) {
			t.Errorf("workflow.ready = %v, want 8", workflow["ready"])
		}
		if workflow["blocked"] != float64(4) {
			t.Errorf("workflow.blocked = %v, want 4", workflow["blocked"])
		}

		// Check by_priority array
		byPriority, ok := result["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority is not an array: %T", result["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}

		// Check each priority entry
		expected := []struct {
			priority float64
			count    float64
		}{
			{0, 2}, {1, 8}, {2, 25}, {3, 7}, {4, 5},
		}
		for i, exp := range expected {
			entry, ok := byPriority[i].(map[string]interface{})
			if !ok {
				t.Fatalf("by_priority[%d] is not an object: %T", i, byPriority[i])
			}
			if entry["priority"] != exp.priority {
				t.Errorf("by_priority[%d].priority = %v, want %v", i, entry["priority"], exp.priority)
			}
			if entry["count"] != exp.count {
				t.Errorf("by_priority[%d].count = %v, want %v", i, entry["count"], exp.count)
			}
		}
	})

	t.Run("it includes 5 priority rows even at zero", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		stats := &StatsData{
			Total:      0,
			Open:       0,
			InProgress: 0,
			Done:       0,
			Cancelled:  0,
			Ready:      0,
			Blocked:    0,
			ByPriority: [5]int{0, 0, 0, 0, 0},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("FormatStats() error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		byPriority, ok := result["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority is not an array: %T", result["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}

		for i := 0; i < 5; i++ {
			entry, ok := byPriority[i].(map[string]interface{})
			if !ok {
				t.Fatalf("by_priority[%d] is not an object", i)
			}
			if entry["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %v", i, entry["priority"], float64(i))
			}
			if entry["count"] != float64(0) {
				t.Errorf("by_priority[%d].count = %v, want 0", i, entry["count"])
			}
		}
	})

	t.Run("it returns error for non-StatsData input", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatStats(&buf, "not stats data")
		if err == nil {
			t.Fatal("FormatStats() expected error for non-StatsData input, got nil")
		}
	})
}

func TestJSONFormatterFormatTransitionDepMessage(t *testing.T) {
	t.Run("it formats transition as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatTransition(&buf, "tick-a3f2b7", task.StatusOpen, task.StatusInProgress)
		if err != nil {
			t.Fatalf("FormatTransition() error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if result["id"] != "tick-a3f2b7" {
			t.Errorf("id = %v, want tick-a3f2b7", result["id"])
		}
		if result["from"] != "open" {
			t.Errorf("from = %v, want open", result["from"])
		}
		if result["to"] != "in_progress" {
			t.Errorf("to = %v, want in_progress", result["to"])
		}
	})

	t.Run("it formats dep change as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatDepChange(&buf, "added", "tick-c3d4", "tick-a1b2")
		if err != nil {
			t.Fatalf("FormatDepChange() error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if result["action"] != "added" {
			t.Errorf("action = %v, want added", result["action"])
		}
		if result["task_id"] != "tick-c3d4" {
			t.Errorf("task_id = %v, want tick-c3d4", result["task_id"])
		}
		if result["blocked_by"] != "tick-a1b2" {
			t.Errorf("blocked_by = %v, want tick-a1b2", result["blocked_by"])
		}
	})

	t.Run("it formats message as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatMessage(&buf, "Tick initialized in /path/to/project")
		if err != nil {
			t.Fatalf("FormatMessage() error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if result["message"] != "Tick initialized in /path/to/project" {
			t.Errorf("message = %v, want 'Tick initialized in /path/to/project'", result["message"])
		}
	})
}

func TestJSONFormatterProducesValidJSON(t *testing.T) {
	t.Run("it produces valid parseable JSON", func(t *testing.T) {
		f := &JSONFormatter{}

		// Test all format methods produce valid JSON
		tests := []struct {
			name string
			fn   func() (string, error)
		}{
			{
				"list",
				func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTaskList(&buf, []TaskRow{
						{ID: "tick-a1b2", Title: "Test", Status: "open", Priority: 2},
					})
					return buf.String(), err
				},
			},
			{
				"detail",
				func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTaskDetail(&buf, &showData{
						ID: "tick-a1b2", Title: "Test", Status: "open",
						Priority: 2, Created: "2026-01-19T10:00:00Z",
						Updated: "2026-01-19T10:00:00Z",
					})
					return buf.String(), err
				},
			},
			{
				"stats",
				func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatStats(&buf, &StatsData{Total: 1, ByPriority: [5]int{0, 0, 1, 0, 0}})
					return buf.String(), err
				},
			},
			{
				"transition",
				func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTransition(&buf, "tick-a1b2", task.StatusOpen, task.StatusDone)
					return buf.String(), err
				},
			},
			{
				"dep",
				func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatDepChange(&buf, "added", "tick-a1b2", "tick-c3d4")
					return buf.String(), err
				},
			},
			{
				"message",
				func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatMessage(&buf, "hello")
					return buf.String(), err
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				output, err := tt.fn()
				if err != nil {
					t.Fatalf("error: %v", err)
				}
				if !json.Valid([]byte(output)) {
					t.Errorf("output is not valid JSON:\n%s", output)
				}
			})
		}
	})
}

func TestJSONFormatterIndentation(t *testing.T) {
	t.Run("it produces 2-space indented JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatMessage(&buf, "test")
		if err != nil {
			t.Fatalf("FormatMessage() error: %v", err)
		}

		got := buf.String()
		// 2-space indented JSON for {"message": "test"} should contain "  " prefix
		if !strings.Contains(got, "  \"message\"") {
			t.Errorf("output should be 2-space indented, got:\n%s", got)
		}

		// Should NOT contain 4-space or tab indentation
		if strings.Contains(got, "    \"message\"") {
			t.Errorf("output should use 2-space indent, not 4-space:\n%s", got)
		}
		if strings.Contains(got, "\t\"message\"") {
			t.Errorf("output should use 2-space indent, not tabs:\n%s", got)
		}
	})
}
