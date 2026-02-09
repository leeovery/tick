package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
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

		data := []TaskRow{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: "open", Priority: 2},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Must be valid JSON.
		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		if len(parsed) != 2 {
			t.Fatalf("expected 2 items, got %d", len(parsed))
		}

		// Check first item fields.
		if parsed[0]["id"] != "tick-a1b2" {
			t.Errorf("expected id tick-a1b2, got %v", parsed[0]["id"])
		}
		if parsed[0]["title"] != "Setup Sanctum" {
			t.Errorf("expected title Setup Sanctum, got %v", parsed[0]["title"])
		}
		if parsed[0]["status"] != "done" {
			t.Errorf("expected status done, got %v", parsed[0]["status"])
		}
		// JSON numbers unmarshal as float64.
		if parsed[0]["priority"] != float64(1) {
			t.Errorf("expected priority 1, got %v", parsed[0]["priority"])
		}
	})

	t.Run("it formats empty list as [] not null", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, []TaskRow{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := strings.TrimSpace(buf.String())
		if got != "[]" {
			t.Errorf("expected [], got %q", got)
		}
	})

	t.Run("it produces valid parseable JSON for list", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := []TaskRow{
			{ID: "tick-a1b2", Title: "Task one", Status: "open", Priority: 0},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !json.Valid(buf.Bytes()) {
			t.Errorf("output is not valid JSON: %s", buf.String())
		}
	})

	t.Run("it uses snake_case keys for list", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := []TaskRow{
			{ID: "tick-a1b2", Title: "Task", Status: "open", Priority: 1},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		for _, key := range []string{`"id"`, `"title"`, `"status"`, `"priority"`} {
			if !strings.Contains(got, key) {
				t.Errorf("expected key %s in output, got %s", key, got)
			}
		}
	})

	t.Run("it uses 2-space indentation for list", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := []TaskRow{
			{ID: "tick-a1b2", Title: "Task", Status: "open", Priority: 1},
		}

		err := f.FormatTaskList(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		// 2-space indented JSON: array elements indented with 2 spaces, object fields with 4.
		if !strings.Contains(got, "\n  {") {
			t.Errorf("expected 2-space indentation for array elements: %s", got)
		}
		if !strings.Contains(got, "\n    \"id\"") {
			t.Errorf("expected 4-space indentation for object fields (2 array + 2 object): %s", got)
		}
		// Must not contain tab indentation.
		if strings.Contains(got, "\t") {
			t.Errorf("expected no tab indentation: %s", got)
		}
	})
}

func TestJSONFormatterFormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all fields", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:          "tick-a1b2",
			title:       "Setup Sanctum",
			status:      "in_progress",
			priority:    1,
			parent:      "tick-e5f6",
			created:     "2026-01-19T10:00:00Z",
			updated:     "2026-01-19T14:30:00Z",
			closed:      "2026-01-19T14:30:00Z",
			description: "Full task description.",
			blockedBy: []relatedTask{
				{id: "tick-c3d4", title: "Database migrations", status: "done"},
			},
			children: []relatedTask{
				{id: "tick-g7h8", title: "Sub task", status: "open"},
			},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		// Verify all expected keys present.
		expectedKeys := []string{"id", "title", "status", "priority", "parent", "created", "updated", "closed", "description", "blocked_by", "children"}
		for _, key := range expectedKeys {
			if _, ok := parsed[key]; !ok {
				t.Errorf("missing key %q in output", key)
			}
		}

		if parsed["id"] != "tick-a1b2" {
			t.Errorf("expected id tick-a1b2, got %v", parsed["id"])
		}
		if parsed["priority"] != float64(1) {
			t.Errorf("expected priority 1, got %v", parsed["priority"])
		}
		if parsed["description"] != "Full task description." {
			t.Errorf("expected description, got %v", parsed["description"])
		}

		// Check blocked_by is an array with 1 element.
		blockedBy, ok := parsed["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", parsed["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Errorf("expected 1 blocked_by item, got %d", len(blockedBy))
		}

		// Check children is an array with 1 element.
		children, ok := parsed["children"].([]interface{})
		if !ok {
			t.Fatalf("children is not an array: %T", parsed["children"])
		}
		if len(children) != 1 {
			t.Errorf("expected 1 children item, got %d", len(children))
		}
	})

	t.Run("it omits parent/closed when null", func(t *testing.T) {
		f := &JSONFormatter{}
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

		got := buf.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		if _, ok := parsed["parent"]; ok {
			t.Errorf("expected parent to be omitted when empty, but it was present: %v", parsed["parent"])
		}
		if _, ok := parsed["closed"]; ok {
			t.Errorf("expected closed to be omitted when empty, but it was present: %v", parsed["closed"])
		}
	})

	t.Run("it includes blocked_by/children as empty arrays", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:       "tick-a1b2",
			title:    "No deps task",
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

		// Must contain empty arrays, not null.
		if !strings.Contains(got, `"blocked_by": []`) {
			t.Errorf("expected blocked_by as empty array [], got: %s", got)
		}
		if !strings.Contains(got, `"children": []`) {
			t.Errorf("expected children as empty array [], got: %s", got)
		}

		// Also verify via parsing.
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		blockedBy, ok := parsed["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by should be array, got %T", parsed["blocked_by"])
		}
		if len(blockedBy) != 0 {
			t.Errorf("expected empty blocked_by, got %d items", len(blockedBy))
		}

		children, ok := parsed["children"].([]interface{})
		if !ok {
			t.Fatalf("children should be array, got %T", parsed["children"])
		}
		if len(children) != 0 {
			t.Errorf("expected empty children, got %d items", len(children))
		}
	})

	t.Run("it formats description as empty string not null", func(t *testing.T) {
		f := &JSONFormatter{}
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

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		desc, ok := parsed["description"]
		if !ok {
			t.Fatal("expected description key to be present")
		}
		descStr, ok := desc.(string)
		if !ok {
			t.Fatalf("expected description to be string, got %T", desc)
		}
		if descStr != "" {
			t.Errorf("expected empty string description, got %q", descStr)
		}
	})

	t.Run("it uses snake_case for all keys in show", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &showData{
			id:       "tick-a1b2",
			title:    "Task",
			status:   "open",
			priority: 2,
			parent:   "tick-p1",
			created:  "2026-01-19T10:00:00Z",
			updated:  "2026-01-19T10:00:00Z",
			closed:   "2026-01-19T14:00:00Z",
			blockedBy: []relatedTask{
				{id: "tick-b1", title: "Blocker", status: "open"},
			},
			children: []relatedTask{},
		}

		err := f.FormatTaskDetail(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Verify snake_case keys are used.
		snakeKeys := []string{`"id"`, `"title"`, `"status"`, `"priority"`, `"parent"`, `"created"`, `"updated"`, `"closed"`, `"description"`, `"blocked_by"`, `"children"`}
		for _, key := range snakeKeys {
			if !strings.Contains(got, key) {
				t.Errorf("expected snake_case key %s in output", key)
			}
		}

		// Ensure no camelCase variants.
		camelKeys := []string{`"blockedBy"`, `"BlockedBy"`, `"oldStatus"`, `"newStatus"`, `"parentTitle"`}
		for _, key := range camelKeys {
			if strings.Contains(got, key) {
				t.Errorf("unexpected camelCase key %s in output", key)
			}
		}
	})
}

func TestJSONFormatterFormatStats(t *testing.T) {
	t.Run("it formats stats as structured nested object", func(t *testing.T) {
		f := &JSONFormatter{}
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

		got := buf.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		// Check top-level total.
		if parsed["total"] != float64(47) {
			t.Errorf("expected total 47, got %v", parsed["total"])
		}

		// Check by_status nested object.
		byStatus, ok := parsed["by_status"].(map[string]interface{})
		if !ok {
			t.Fatalf("by_status is not an object: %T", parsed["by_status"])
		}
		if byStatus["open"] != float64(12) {
			t.Errorf("expected by_status.open 12, got %v", byStatus["open"])
		}
		if byStatus["in_progress"] != float64(3) {
			t.Errorf("expected by_status.in_progress 3, got %v", byStatus["in_progress"])
		}
		if byStatus["done"] != float64(28) {
			t.Errorf("expected by_status.done 28, got %v", byStatus["done"])
		}
		if byStatus["cancelled"] != float64(4) {
			t.Errorf("expected by_status.cancelled 4, got %v", byStatus["cancelled"])
		}

		// Check workflow nested object.
		workflow, ok := parsed["workflow"].(map[string]interface{})
		if !ok {
			t.Fatalf("workflow is not an object: %T", parsed["workflow"])
		}
		if workflow["ready"] != float64(8) {
			t.Errorf("expected workflow.ready 8, got %v", workflow["ready"])
		}
		if workflow["blocked"] != float64(4) {
			t.Errorf("expected workflow.blocked 4, got %v", workflow["blocked"])
		}

		// Check by_priority array.
		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority is not an array: %T", parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(byPriority))
		}

		// Verify each entry has priority and count.
		expectedCounts := []float64{2, 8, 25, 7, 5}
		for i, entry := range byPriority {
			obj, ok := entry.(map[string]interface{})
			if !ok {
				t.Fatalf("by_priority[%d] is not an object: %T", i, entry)
			}
			if obj["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %d", i, obj["priority"], i)
			}
			if obj["count"] != expectedCounts[i] {
				t.Errorf("by_priority[%d].count = %v, want %v", i, obj["count"], expectedCounts[i])
			}
		}
	})

	t.Run("it includes 5 priority rows even at zero", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &StatsData{
			Total:      0,
			ByPriority: [5]int{0, 0, 0, 0, 0},
		}

		err := f.FormatStats(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority is not an array: %T", parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(byPriority))
		}

		for i, entry := range byPriority {
			obj, ok := entry.(map[string]interface{})
			if !ok {
				t.Fatalf("by_priority[%d] is not an object: %T", i, entry)
			}
			if obj["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %d", i, obj["priority"], i)
			}
			if obj["count"] != float64(0) {
				t.Errorf("by_priority[%d].count = %v, want 0", i, obj["count"])
			}
		}
	})

	t.Run("it uses snake_case keys for stats", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		data := &StatsData{
			Total:      10,
			Open:       5,
			InProgress: 2,
			Done:       2,
			Cancelled:  1,
			Ready:      3,
			Blocked:    2,
			ByPriority: [5]int{1, 2, 3, 2, 2},
		}

		err := f.FormatStats(&buf, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		snakeKeys := []string{`"total"`, `"by_status"`, `"in_progress"`, `"by_priority"`}
		for _, key := range snakeKeys {
			if !strings.Contains(got, key) {
				t.Errorf("expected snake_case key %s in output", key)
			}
		}

		// Ensure no camelCase.
		camelKeys := []string{`"inProgress"`, `"InProgress"`, `"byStatus"`, `"ByStatus"`, `"byPriority"`, `"ByPriority"`}
		for _, key := range camelKeys {
			if strings.Contains(got, key) {
				t.Errorf("unexpected camelCase key %s in output", key)
			}
		}
	})
}

func TestJSONFormatterFormatTransitionDepMessage(t *testing.T) {
	t.Run("it formats transition as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}
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

		got := buf.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		if parsed["id"] != "tick-a1b2" {
			t.Errorf("expected id tick-a1b2, got %v", parsed["id"])
		}
		if parsed["from"] != "open" {
			t.Errorf("expected from open, got %v", parsed["from"])
		}
		if parsed["to"] != "in_progress" {
			t.Errorf("expected to in_progress, got %v", parsed["to"])
		}
	})

	t.Run("it formats dep change as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}
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

		got := buf.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		if parsed["action"] != "added" {
			t.Errorf("expected action added, got %v", parsed["action"])
		}
		if parsed["task_id"] != "tick-a1b2" {
			t.Errorf("expected task_id tick-a1b2, got %v", parsed["task_id"])
		}
		if parsed["blocked_by"] != "tick-c3d4" {
			t.Errorf("expected blocked_by tick-c3d4, got %v", parsed["blocked_by"])
		}
	})

	t.Run("it formats message as JSON object", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		f.FormatMessage(&buf, "Initialized .tick directory")

		got := buf.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, got)
		}

		if parsed["message"] != "Initialized .tick directory" {
			t.Errorf("expected message, got %v", parsed["message"])
		}
	})

	t.Run("it produces valid parseable JSON for all formats", func(t *testing.T) {
		f := &JSONFormatter{}

		// List.
		var buf1 bytes.Buffer
		_ = f.FormatTaskList(&buf1, []TaskRow{{ID: "t1", Title: "T", Status: "open", Priority: 1}})
		if !json.Valid(buf1.Bytes()) {
			t.Errorf("list output is not valid JSON: %s", buf1.String())
		}

		// Detail.
		var buf2 bytes.Buffer
		_ = f.FormatTaskDetail(&buf2, &showData{
			id: "t1", title: "T", status: "open", priority: 1,
			created: "2026-01-19T10:00:00Z", updated: "2026-01-19T10:00:00Z",
		})
		if !json.Valid(buf2.Bytes()) {
			t.Errorf("detail output is not valid JSON: %s", buf2.String())
		}

		// Stats.
		var buf3 bytes.Buffer
		_ = f.FormatStats(&buf3, &StatsData{Total: 1, ByPriority: [5]int{0, 1, 0, 0, 0}})
		if !json.Valid(buf3.Bytes()) {
			t.Errorf("stats output is not valid JSON: %s", buf3.String())
		}

		// Transition.
		var buf4 bytes.Buffer
		_ = f.FormatTransition(&buf4, &TransitionData{ID: "t1", OldStatus: "open", NewStatus: "done"})
		if !json.Valid(buf4.Bytes()) {
			t.Errorf("transition output is not valid JSON: %s", buf4.String())
		}

		// DepChange.
		var buf5 bytes.Buffer
		_ = f.FormatDepChange(&buf5, &DepChangeData{Action: "added", TaskID: "t1", BlockedByID: "t2"})
		if !json.Valid(buf5.Bytes()) {
			t.Errorf("dep change output is not valid JSON: %s", buf5.String())
		}

		// Message.
		var buf6 bytes.Buffer
		f.FormatMessage(&buf6, "Hello")
		if !json.Valid(buf6.Bytes()) {
			t.Errorf("message output is not valid JSON: %s", buf6.String())
		}
	})
}
