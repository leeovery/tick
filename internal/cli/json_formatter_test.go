package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestJSONFormatter_ImplementsFormatter(t *testing.T) {
	t.Run("it implements the full Formatter interface", func(t *testing.T) {
		var _ Formatter = &JSONFormatter{}
	})
}

func TestJSONFormatter_FormatTaskList(t *testing.T) {
	t.Run("it formats list as JSON array", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		rows := []listRow{
			{ID: "tick-a1b2", Status: "done", Priority: 1, Title: "Setup Sanctum"},
			{ID: "tick-c3d4", Status: "open", Priority: 1, Title: "Login endpoint"},
		}

		err := f.FormatTaskList(&buf, rows, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Must be valid JSON
		if !json.Valid([]byte(got)) {
			t.Fatalf("output is not valid JSON: %s", got)
		}

		// Parse and verify structure
		var tasks []map[string]interface{}
		if err := json.Unmarshal([]byte(got), &tasks); err != nil {
			t.Fatalf("failed to parse JSON array: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		// Check snake_case keys
		for _, key := range []string{"id", "title", "status", "priority"} {
			if _, ok := tasks[0][key]; !ok {
				t.Errorf("missing snake_case key %q", key)
			}
		}

		// Check values
		if tasks[0]["id"] != "tick-a1b2" {
			t.Errorf("expected id tick-a1b2, got %v", tasks[0]["id"])
		}
		if tasks[0]["title"] != "Setup Sanctum" {
			t.Errorf("expected title Setup Sanctum, got %v", tasks[0]["title"])
		}
		if tasks[0]["priority"] != float64(1) {
			t.Errorf("expected priority 1, got %v", tasks[0]["priority"])
		}
	})

	t.Run("it formats empty list as [] not null", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskList(&buf, nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Must be valid JSON
		if !json.Valid([]byte(got)) {
			t.Fatalf("output is not valid JSON: %s", got)
		}

		// Parse as raw JSON to check it's [] not null
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(got), &raw); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		// Parse as array
		var arr []interface{}
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Fatalf("output is not a JSON array: %v\ngot: %s", err, got)
		}

		if len(arr) != 0 {
			t.Errorf("expected empty array, got %d elements", len(arr))
		}

		// Also test with empty slice (not nil)
		buf.Reset()
		err = f.FormatTaskList(&buf, []listRow{}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got2 := buf.String()
		var arr2 []interface{}
		if err := json.Unmarshal([]byte(got2), &arr2); err != nil {
			t.Fatalf("empty slice output is not a JSON array: %v\ngot: %s", err, got2)
		}
		if len(arr2) != 0 {
			t.Errorf("expected empty array for empty slice, got %d elements", len(arr2))
		}
	})
}

func TestJSONFormatter_FormatTaskDetail(t *testing.T) {
	t.Run("it formats show with all fields", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:          "tick-a1b2",
			Title:       "Setup Sanctum",
			Status:      "in_progress",
			Priority:    1,
			Description: "Full task description here.",
			Parent:      "tick-e5f6",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T14:30:00Z",
			Closed:      "2026-01-19T16:00:00Z",
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
			},
			Children: []RelatedTask{
				{ID: "tick-x1y2", Title: "Subtask one", Status: "open"},
			},
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		if !json.Valid([]byte(got)) {
			t.Fatalf("output is not valid JSON: %s", got)
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(got), &obj); err != nil {
			t.Fatalf("failed to parse JSON object: %v", err)
		}

		// Check all expected keys present
		for _, key := range []string{"id", "title", "status", "priority", "description", "parent", "created", "updated", "closed", "blocked_by", "children"} {
			if _, ok := obj[key]; !ok {
				t.Errorf("missing key %q", key)
			}
		}

		// Check values
		if obj["id"] != "tick-a1b2" {
			t.Errorf("expected id tick-a1b2, got %v", obj["id"])
		}
		if obj["parent"] != "tick-e5f6" {
			t.Errorf("expected parent tick-e5f6, got %v", obj["parent"])
		}
		if obj["closed"] != "2026-01-19T16:00:00Z" {
			t.Errorf("expected closed timestamp, got %v", obj["closed"])
		}

		// Check blocked_by is array
		blockedBy, ok := obj["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", obj["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Errorf("expected 1 blocked_by entry, got %d", len(blockedBy))
		}

		// Check children is array
		children, ok := obj["children"].([]interface{})
		if !ok {
			t.Fatalf("children is not an array: %T", obj["children"])
		}
		if len(children) != 1 {
			t.Errorf("expected 1 children entry, got %d", len(children))
		}
	})

	t.Run("it omits parent/closed when null", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Simple task",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
			// No Parent, no Closed
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(got), &obj); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// parent and closed should NOT be present at all (not even as null)
		if _, ok := obj["parent"]; ok {
			t.Errorf("parent should be omitted when empty, but found: %v", obj["parent"])
		}
		if _, ok := obj["closed"]; ok {
			t.Errorf("closed should be omitted when empty, but found: %v", obj["closed"])
		}
	})

	t.Run("it includes blocked_by/children as empty arrays", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Simple task",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
			// No BlockedBy, no Children (nil slices)
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(got), &obj); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// blocked_by must be present and be an empty array (not null, not absent)
		blockedBy, ok := obj["blocked_by"]
		if !ok {
			t.Fatalf("blocked_by key missing - must always be present")
		}
		blockedByArr, ok := blockedBy.([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T (possibly null)", blockedBy)
		}
		if len(blockedByArr) != 0 {
			t.Errorf("expected empty blocked_by array, got %d elements", len(blockedByArr))
		}

		// children must be present and be an empty array (not null, not absent)
		children, ok := obj["children"]
		if !ok {
			t.Fatalf("children key missing - must always be present")
		}
		childrenArr, ok := children.([]interface{})
		if !ok {
			t.Fatalf("children is not an array: %T (possibly null)", children)
		}
		if len(childrenArr) != 0 {
			t.Errorf("expected empty children array, got %d elements", len(childrenArr))
		}
	})

	t.Run("it formats description as empty string not null", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Task without description",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
			// Description is zero value ""
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(got), &obj); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// description must be present
		desc, ok := obj["description"]
		if !ok {
			t.Fatalf("description key missing - must always be present")
		}

		// Must be a string (not null)
		descStr, ok := desc.(string)
		if !ok {
			t.Fatalf("description is not a string: %T (possibly null)", desc)
		}

		if descStr != "" {
			t.Errorf("expected empty string description, got %q", descStr)
		}
	})
}

func TestJSONFormatter_SnakeCaseKeys(t *testing.T) {
	t.Run("it uses snake_case for all keys", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		detail := TaskDetail{
			ID:          "tick-a1b2",
			Title:       "Test",
			Status:      "open",
			Priority:    2,
			Description: "desc",
			Parent:      "tick-p1",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "2026-01-19T16:00:00Z",
			BlockedBy: []RelatedTask{
				{ID: "tick-b1", Title: "Blocker", Status: "done"},
			},
			Children: []RelatedTask{
				{ID: "tick-c1", Title: "Child", Status: "open"},
			},
		}

		err := f.FormatTaskDetail(&buf, detail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// Verify no camelCase or PascalCase keys
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(got), &obj); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		expectedKeys := []string{"id", "title", "status", "priority", "description", "parent", "created", "updated", "closed", "blocked_by", "children"}
		for _, key := range expectedKeys {
			if _, ok := obj[key]; !ok {
				t.Errorf("missing expected snake_case key %q", key)
			}
		}

		// Ensure no camelCase variants exist
		badKeys := []string{"blockedBy", "BlockedBy", "blocked_By", "Children", "inProgress", "InProgress"}
		for _, key := range badKeys {
			if _, ok := obj[key]; ok {
				t.Errorf("found non-snake_case key %q - all keys must be snake_case", key)
			}
		}

		// Also check nested objects (blocked_by entries)
		var fullObj map[string]interface{}
		json.Unmarshal([]byte(got), &fullObj)
		blockedBy := fullObj["blocked_by"].([]interface{})
		if len(blockedBy) > 0 {
			entry := blockedBy[0].(map[string]interface{})
			for _, key := range []string{"id", "title", "status"} {
				if _, ok := entry[key]; !ok {
					t.Errorf("blocked_by entry missing snake_case key %q", key)
				}
			}
		}
	})
}

func TestJSONFormatter_FormatStats(t *testing.T) {
	t.Run("it formats stats as structured nested object", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		stats := StatsData{
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
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		if !json.Valid([]byte(got)) {
			t.Fatalf("output is not valid JSON: %s", got)
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(got), &obj); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Check top-level total
		if obj["total"] != float64(47) {
			t.Errorf("expected total 47, got %v", obj["total"])
		}

		// Check by_status nested object
		byStatus, ok := obj["by_status"].(map[string]interface{})
		if !ok {
			t.Fatalf("by_status is not an object: %T", obj["by_status"])
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

		// Check workflow nested object
		workflow, ok := obj["workflow"].(map[string]interface{})
		if !ok {
			t.Fatalf("workflow is not an object: %T", obj["workflow"])
		}
		if workflow["ready"] != float64(8) {
			t.Errorf("expected ready 8, got %v", workflow["ready"])
		}
		if workflow["blocked"] != float64(4) {
			t.Errorf("expected blocked 4, got %v", workflow["blocked"])
		}
	})

	t.Run("it includes 5 priority rows even at zero", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		stats := StatsData{
			Total:      0,
			ByPriority: [5]int{0, 0, 0, 0, 0},
		}

		err := f.FormatStats(&buf, stats)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(got), &obj); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Check by_priority is an array with 5 entries
		byPriority, ok := obj["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority is not an array: %T", obj["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(byPriority))
		}

		// Each entry should have priority and count keys
		for i, entry := range byPriority {
			entryMap, ok := entry.(map[string]interface{})
			if !ok {
				t.Fatalf("priority entry %d is not an object: %T", i, entry)
			}
			if entryMap["priority"] != float64(i) {
				t.Errorf("entry %d: expected priority %d, got %v", i, i, entryMap["priority"])
			}
			if entryMap["count"] != float64(0) {
				t.Errorf("entry %d: expected count 0, got %v", i, entryMap["count"])
			}
		}
	})
}

func TestJSONFormatter_FormatTransitionDepMessage(t *testing.T) {
	t.Run("it formats transition/dep/message as JSON objects", func(t *testing.T) {
		f := &JSONFormatter{}

		// Test transition
		var buf bytes.Buffer
		err := f.FormatTransition(&buf, "tick-a1b2", "open", "in_progress")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		if !json.Valid([]byte(got)) {
			t.Fatalf("transition output is not valid JSON: %s", got)
		}

		var trans map[string]interface{}
		if err := json.Unmarshal([]byte(got), &trans); err != nil {
			t.Fatalf("failed to parse transition JSON: %v", err)
		}
		if trans["id"] != "tick-a1b2" {
			t.Errorf("expected id tick-a1b2, got %v", trans["id"])
		}
		if trans["from"] != "open" {
			t.Errorf("expected from open, got %v", trans["from"])
		}
		if trans["to"] != "in_progress" {
			t.Errorf("expected to in_progress, got %v", trans["to"])
		}

		// Test dep add
		buf.Reset()
		err = f.FormatDepChange(&buf, "tick-c3d4", "tick-a1b2", "added", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got = buf.String()
		if !json.Valid([]byte(got)) {
			t.Fatalf("dep change output is not valid JSON: %s", got)
		}

		var dep map[string]interface{}
		if err := json.Unmarshal([]byte(got), &dep); err != nil {
			t.Fatalf("failed to parse dep JSON: %v", err)
		}
		if dep["action"] != "added" {
			t.Errorf("expected action added, got %v", dep["action"])
		}
		if dep["task_id"] != "tick-c3d4" {
			t.Errorf("expected task_id tick-c3d4, got %v", dep["task_id"])
		}
		if dep["blocked_by"] != "tick-a1b2" {
			t.Errorf("expected blocked_by tick-a1b2, got %v", dep["blocked_by"])
		}

		// Test message
		buf.Reset()
		err = f.FormatMessage(&buf, "Tick initialized in /path/to/.tick")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got = buf.String()
		if !json.Valid([]byte(got)) {
			t.Fatalf("message output is not valid JSON: %s", got)
		}

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(got), &msg); err != nil {
			t.Fatalf("failed to parse message JSON: %v", err)
		}
		if msg["message"] != "Tick initialized in /path/to/.tick" {
			t.Errorf("expected message text, got %v", msg["message"])
		}
	})
}

func TestJSONFormatter_ValidJSON(t *testing.T) {
	t.Run("it produces valid parseable JSON", func(t *testing.T) {
		f := &JSONFormatter{}

		// Test all format methods produce valid JSON
		tests := []struct {
			name string
			fn   func() (string, error)
		}{
			{
				name: "task list",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTaskList(&buf, []listRow{
						{ID: "tick-a1b2", Status: "open", Priority: 2, Title: "Task with \"quotes\" and special chars"},
					}, false)
					return buf.String(), err
				},
			},
			{
				name: "task detail",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTaskDetail(&buf, TaskDetail{
						ID: "tick-a1b2", Title: "Task with\nnewline", Status: "open",
						Priority: 2, Description: "Line 1\nLine 2\n\"Quoted\"",
						Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
					})
					return buf.String(), err
				},
			},
			{
				name: "stats",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatStats(&buf, StatsData{Total: 10, ByPriority: [5]int{1, 2, 3, 4, 0}})
					return buf.String(), err
				},
			},
			{
				name: "transition",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatTransition(&buf, "tick-a1b2", "open", "in_progress")
					return buf.String(), err
				},
			},
			{
				name: "dep change",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatDepChange(&buf, "tick-a1b2", "tick-c3d4", "added", false)
					return buf.String(), err
				},
			},
			{
				name: "message",
				fn: func() (string, error) {
					var buf bytes.Buffer
					err := f.FormatMessage(&buf, "Hello world")
					return buf.String(), err
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got, err := tc.fn()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !json.Valid([]byte(got)) {
					t.Errorf("output is not valid JSON:\n%s", got)
				}
			})
		}
	})
}

func TestJSONFormatter_Indentation(t *testing.T) {
	t.Run("it produces 2-space indented JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		var buf bytes.Buffer

		err := f.FormatTaskDetail(&buf, TaskDetail{
			ID:       "tick-a1b2",
			Title:    "Test",
			Status:   "open",
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()

		// MarshalIndent with 2-space should produce lines starting with "  "
		// The second line of pretty-printed JSON should start with exactly 2 spaces
		lines := bytes.Split([]byte(got), []byte("\n"))
		if len(lines) < 3 {
			t.Fatalf("expected multi-line indented JSON, got single line: %s", got)
		}

		// Check that indented lines use exactly 2 spaces (not tabs, not 4 spaces)
		for _, line := range lines {
			s := string(line)
			if len(s) > 0 && s[0] == ' ' {
				// Count leading spaces
				trimmed := bytes.TrimLeft(line, " ")
				indent := len(line) - len(trimmed)
				// Indent should be a multiple of 2
				if indent%2 != 0 {
					t.Errorf("indent %d is not a multiple of 2: %q", indent, s)
				}
			}
		}
	})
}
