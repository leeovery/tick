package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestJSONFormatter(t *testing.T) {
	// Compile-time interface verification.
	var _ Formatter = (*JSONFormatter)(nil)

	t.Run("it formats list as JSON array", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)

		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}
		if len(parsed) != 2 {
			t.Fatalf("expected 2 items, got %d", len(parsed))
		}

		// Verify first task fields
		if parsed[0]["id"] != "tick-a1b2" {
			t.Errorf("first task id = %v, want %q", parsed[0]["id"], "tick-a1b2")
		}
		if parsed[0]["title"] != "Setup Sanctum" {
			t.Errorf("first task title = %v, want %q", parsed[0]["title"], "Setup Sanctum")
		}
		if parsed[0]["status"] != "done" {
			t.Errorf("first task status = %v, want %q", parsed[0]["status"], "done")
		}
		if parsed[0]["priority"] != float64(1) {
			t.Errorf("first task priority = %v, want %v", parsed[0]["priority"], 1)
		}

		// Verify second task
		if parsed[1]["id"] != "tick-c3d4" {
			t.Errorf("second task id = %v, want %q", parsed[1]["id"], "tick-c3d4")
		}
	})

	t.Run("it formats empty list as [] not null", func(t *testing.T) {
		f := &JSONFormatter{}

		// Empty slice
		result := f.FormatTaskList([]task.Task{})
		if result != "[]" {
			t.Errorf("empty slice result = %q, want %q", result, "[]")
		}

		// Nil slice
		resultNil := f.FormatTaskList(nil)
		if resultNil != "[]" {
			t.Errorf("nil slice result = %q, want %q", resultNil, "[]")
		}
	})

	t.Run("it formats show with all fields", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:          "tick-a1b2",
				Title:       "Setup Sanctum",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: "Full description here.",
				Parent:      "tick-e5f6",
				Created:     now,
				Updated:     updated,
				Closed:      &closed,
			},
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
			},
			Children: []RelatedTask{
				{ID: "tick-g7h8", Title: "Sub task", Status: "open"},
			},
			ParentTitle: "Auth System",
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// Verify all expected fields
		if parsed["id"] != "tick-a1b2" {
			t.Errorf("id = %v, want %q", parsed["id"], "tick-a1b2")
		}
		if parsed["title"] != "Setup Sanctum" {
			t.Errorf("title = %v, want %q", parsed["title"], "Setup Sanctum")
		}
		if parsed["status"] != "in_progress" {
			t.Errorf("status = %v, want %q", parsed["status"], "in_progress")
		}
		if parsed["priority"] != float64(1) {
			t.Errorf("priority = %v, want %v", parsed["priority"], 1)
		}
		if parsed["description"] != "Full description here." {
			t.Errorf("description = %v, want %q", parsed["description"], "Full description here.")
		}
		if parsed["parent"] != "tick-e5f6" {
			t.Errorf("parent = %v, want %q", parsed["parent"], "tick-e5f6")
		}
		if parsed["created"] != "2026-01-19T10:00:00Z" {
			t.Errorf("created = %v, want %q", parsed["created"], "2026-01-19T10:00:00Z")
		}
		if parsed["updated"] != "2026-01-19T14:30:00Z" {
			t.Errorf("updated = %v, want %q", parsed["updated"], "2026-01-19T14:30:00Z")
		}
		if parsed["closed"] != "2026-01-19T16:00:00Z" {
			t.Errorf("closed = %v, want %q", parsed["closed"], "2026-01-19T16:00:00Z")
		}

		// Verify blocked_by array
		blockedBy, ok := parsed["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %v", parsed["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Fatalf("blocked_by length = %d, want 1", len(blockedBy))
		}
		blocker := blockedBy[0].(map[string]interface{})
		if blocker["id"] != "tick-c3d4" {
			t.Errorf("blocker id = %v, want %q", blocker["id"], "tick-c3d4")
		}

		// Verify children array
		children, ok := parsed["children"].([]interface{})
		if !ok {
			t.Fatalf("children is not an array: %v", parsed["children"])
		}
		if len(children) != 1 {
			t.Fatalf("children length = %d, want 1", len(children))
		}
		child := children[0].(map[string]interface{})
		if child["id"] != "tick-g7h8" {
			t.Errorf("child id = %v, want %q", child["id"], "tick-g7h8")
		}
	})

	t.Run("it omits parent/closed when null", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Simple task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
				// Parent empty, Closed nil
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// parent should not be present
		if _, exists := parsed["parent"]; exists {
			t.Errorf("parent should be omitted when empty, got %v", parsed["parent"])
		}
		// closed should not be present
		if _, exists := parsed["closed"]; exists {
			t.Errorf("closed should be omitted when nil, got %v", parsed["closed"])
		}
	})

	t.Run("it includes blocked_by/children as empty arrays", func(t *testing.T) {
		f := &JSONFormatter{}
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

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// blocked_by must be present as empty array, not null or missing
		blockedBy, ok := parsed["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by should be array, got %T: %v", parsed["blocked_by"], parsed["blocked_by"])
		}
		if len(blockedBy) != 0 {
			t.Errorf("blocked_by should be empty, got %d items", len(blockedBy))
		}

		// children must be present as empty array, not null or missing
		children, ok := parsed["children"].([]interface{})
		if !ok {
			t.Fatalf("children should be array, got %T: %v", parsed["children"], parsed["children"])
		}
		if len(children) != 0 {
			t.Errorf("children should be empty, got %d items", len(children))
		}
	})

	t.Run("it includes blocked_by/children as empty arrays even with nil input", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Nil deps",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			// BlockedBy and Children are nil (not initialized)
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// Must still be [], not null
		blockedBy, ok := parsed["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by should be array even with nil input, got %T: %v", parsed["blocked_by"], parsed["blocked_by"])
		}
		if len(blockedBy) != 0 {
			t.Errorf("blocked_by should be empty, got %d items", len(blockedBy))
		}

		children, ok := parsed["children"].([]interface{})
		if !ok {
			t.Fatalf("children should be array even with nil input, got %T: %v", parsed["children"], parsed["children"])
		}
		if len(children) != 0 {
			t.Errorf("children should be empty, got %d items", len(children))
		}
	})

	t.Run("it formats description as empty string not null", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No description",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
				// Description is empty string (zero value)
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// description must be present as empty string, not null or missing
		desc, exists := parsed["description"]
		if !exists {
			t.Fatal("description should be present even when empty")
		}
		descStr, ok := desc.(string)
		if !ok {
			t.Fatalf("description should be string, got %T: %v", desc, desc)
		}
		if descStr != "" {
			t.Errorf("description = %q, want empty string", descStr)
		}
	})

	t.Run("it uses snake_case for all keys", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:          "tick-a1b2",
				Title:       "Test task",
				Status:      task.StatusDone,
				Priority:    1,
				Description: "desc",
				Parent:      "tick-e5f6",
				Created:     now,
				Updated:     now,
				Closed:      &closed,
			},
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4", Title: "Blocker", Status: "open"},
			},
			Children: []RelatedTask{
				{ID: "tick-g7h8", Title: "Child", Status: "open"},
			},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		// All top-level keys must be snake_case
		expectedKeys := []string{"id", "title", "status", "priority", "description", "parent", "created", "updated", "closed", "blocked_by", "children"}
		for _, key := range expectedKeys {
			if _, exists := parsed[key]; !exists {
				t.Errorf("expected snake_case key %q not found", key)
			}
		}

		// No camelCase keys should exist
		camelKeys := []string{"blockedBy", "BlockedBy", "parentTitle", "ParentTitle", "ID", "Title", "Status", "Priority"}
		for _, key := range camelKeys {
			if _, exists := parsed[key]; exists {
				t.Errorf("found non-snake_case key %q", key)
			}
		}
	})

	t.Run("it formats stats as structured nested object", func(t *testing.T) {
		f := &JSONFormatter{}
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
		result := f.FormatStats(stats)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// Verify top-level total
		if parsed["total"] != float64(47) {
			t.Errorf("total = %v, want 47", parsed["total"])
		}

		// Verify by_status nested object
		byStatus, ok := parsed["by_status"].(map[string]interface{})
		if !ok {
			t.Fatalf("by_status should be object, got %T: %v", parsed["by_status"], parsed["by_status"])
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

		// Verify workflow nested object
		workflow, ok := parsed["workflow"].(map[string]interface{})
		if !ok {
			t.Fatalf("workflow should be object, got %T: %v", parsed["workflow"], parsed["workflow"])
		}
		if workflow["ready"] != float64(8) {
			t.Errorf("workflow.ready = %v, want 8", workflow["ready"])
		}
		if workflow["blocked"] != float64(4) {
			t.Errorf("workflow.blocked = %v, want 4", workflow["blocked"])
		}

		// Verify by_priority array
		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority should be array, got %T: %v", parsed["by_priority"], parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}
		expectedCounts := []float64{2, 8, 25, 7, 5}
		for i, expected := range expectedCounts {
			entry := byPriority[i].(map[string]interface{})
			if entry["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %v", i, entry["priority"], i)
			}
			if entry["count"] != expected {
				t.Errorf("by_priority[%d].count = %v, want %v", i, entry["count"], expected)
			}
		}
	})

	t.Run("it includes 5 priority rows even at zero", func(t *testing.T) {
		f := &JSONFormatter{}
		stats := Stats{
			Total:      0,
			ByPriority: [5]int{0, 0, 0, 0, 0},
		}
		result := f.FormatStats(stats)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority should be array, got %T: %v", parsed["by_priority"], parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}
		for i := 0; i < 5; i++ {
			entry := byPriority[i].(map[string]interface{})
			if entry["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %v", i, entry["priority"], i)
			}
			if entry["count"] != float64(0) {
				t.Errorf("by_priority[%d].count = %v, want 0", i, entry["count"])
			}
		}
	})

	t.Run("it formats transition/dep/message as JSON objects", func(t *testing.T) {
		f := &JSONFormatter{}

		// Transition
		transResult := f.FormatTransition("tick-a1b2", "open", "in_progress")
		var transObj map[string]interface{}
		if err := json.Unmarshal([]byte(transResult), &transObj); err != nil {
			t.Fatalf("transition invalid JSON: %v\nresult: %s", err, transResult)
		}
		if transObj["id"] != "tick-a1b2" {
			t.Errorf("transition id = %v, want %q", transObj["id"], "tick-a1b2")
		}
		if transObj["from"] != "open" {
			t.Errorf("transition from = %v, want %q", transObj["from"], "open")
		}
		if transObj["to"] != "in_progress" {
			t.Errorf("transition to = %v, want %q", transObj["to"], "in_progress")
		}

		// Dep change - added
		depAddResult := f.FormatDepChange("added", "tick-c3d4", "tick-a1b2")
		var depAddObj map[string]interface{}
		if err := json.Unmarshal([]byte(depAddResult), &depAddObj); err != nil {
			t.Fatalf("dep add invalid JSON: %v\nresult: %s", err, depAddResult)
		}
		if depAddObj["action"] != "added" {
			t.Errorf("dep action = %v, want %q", depAddObj["action"], "added")
		}
		if depAddObj["task_id"] != "tick-c3d4" {
			t.Errorf("dep task_id = %v, want %q", depAddObj["task_id"], "tick-c3d4")
		}
		if depAddObj["blocked_by"] != "tick-a1b2" {
			t.Errorf("dep blocked_by = %v, want %q", depAddObj["blocked_by"], "tick-a1b2")
		}

		// Dep change - removed
		depRmResult := f.FormatDepChange("removed", "tick-c3d4", "tick-a1b2")
		var depRmObj map[string]interface{}
		if err := json.Unmarshal([]byte(depRmResult), &depRmObj); err != nil {
			t.Fatalf("dep remove invalid JSON: %v\nresult: %s", err, depRmResult)
		}
		if depRmObj["action"] != "removed" {
			t.Errorf("dep remove action = %v, want %q", depRmObj["action"], "removed")
		}

		// Message
		msgResult := f.FormatMessage("Tick initialized in /path/to/project")
		var msgObj map[string]interface{}
		if err := json.Unmarshal([]byte(msgResult), &msgObj); err != nil {
			t.Fatalf("message invalid JSON: %v\nresult: %s", err, msgResult)
		}
		if msgObj["message"] != "Tick initialized in /path/to/project" {
			t.Errorf("message = %v, want %q", msgObj["message"], "Tick initialized in /path/to/project")
		}
	})

	t.Run("it produces valid parseable JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

		outputs := []struct {
			name   string
			output string
		}{
			{"list", f.FormatTaskList([]task.Task{
				{ID: "tick-a1b2", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			})},
			{"empty list", f.FormatTaskList(nil)},
			{"detail", f.FormatTaskDetail(TaskDetail{
				Task: task.Task{
					ID: "tick-a1b2", Title: "Full task", Status: task.StatusDone,
					Priority: 1, Description: "desc", Parent: "tick-e5f6",
					Created: now, Updated: now, Closed: &closed,
				},
				BlockedBy: []RelatedTask{{ID: "tick-c3d4", Title: "B", Status: "open"}},
				Children:  []RelatedTask{{ID: "tick-g7h8", Title: "C", Status: "done"}},
			})},
			{"transition", f.FormatTransition("tick-a1b2", "open", "done")},
			{"dep add", f.FormatDepChange("added", "tick-c3d4", "tick-a1b2")},
			{"dep remove", f.FormatDepChange("removed", "tick-c3d4", "tick-a1b2")},
			{"message", f.FormatMessage("Hello world")},
			{"stats", f.FormatStats(Stats{
				Total: 10, Open: 5, InProgress: 2, Done: 2, Cancelled: 1,
				Ready: 3, Blocked: 2, ByPriority: [5]int{1, 2, 3, 2, 2},
			})},
		}

		for _, tc := range outputs {
			t.Run(tc.name, func(t *testing.T) {
				if !json.Valid([]byte(tc.output)) {
					t.Errorf("output is not valid JSON:\n%s", tc.output)
				}
			})
		}
	})

	t.Run("it formats single task removal as JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "My task"},
			},
			DepsUpdated: []string{},
		})

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		removed, ok := parsed["removed"].([]interface{})
		if !ok {
			t.Fatalf("removed should be array, got %T: %v", parsed["removed"], parsed["removed"])
		}
		if len(removed) != 1 {
			t.Fatalf("removed length = %d, want 1", len(removed))
		}
		item := removed[0].(map[string]interface{})
		if item["id"] != "tick-a1b2" {
			t.Errorf("removed[0].id = %v, want %q", item["id"], "tick-a1b2")
		}
		if item["title"] != "My task" {
			t.Errorf("removed[0].title = %v, want %q", item["title"], "My task")
		}

		depsUpdated, ok := parsed["deps_updated"].([]interface{})
		if !ok {
			t.Fatalf("deps_updated should be array, got %T: %v", parsed["deps_updated"], parsed["deps_updated"])
		}
		if len(depsUpdated) != 0 {
			t.Errorf("deps_updated length = %d, want 0", len(depsUpdated))
		}
	})

	t.Run("it formats multiple task removal as JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "First task"},
				{ID: "tick-c3d4", Title: "Second task"},
			},
			DepsUpdated: []string{},
		})

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		removed, ok := parsed["removed"].([]interface{})
		if !ok {
			t.Fatalf("removed should be array, got %T: %v", parsed["removed"], parsed["removed"])
		}
		if len(removed) != 2 {
			t.Fatalf("removed length = %d, want 2", len(removed))
		}
		first := removed[0].(map[string]interface{})
		if first["id"] != "tick-a1b2" {
			t.Errorf("removed[0].id = %v, want %q", first["id"], "tick-a1b2")
		}
		second := removed[1].(map[string]interface{})
		if second["id"] != "tick-c3d4" {
			t.Errorf("removed[1].id = %v, want %q", second["id"], "tick-c3d4")
		}
	})

	t.Run("it formats removal with dependency updates as JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "My task"},
			},
			DepsUpdated: []string{"tick-e5f6", "tick-g7h8"},
		})

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		depsUpdated, ok := parsed["deps_updated"].([]interface{})
		if !ok {
			t.Fatalf("deps_updated should be array, got %T: %v", parsed["deps_updated"], parsed["deps_updated"])
		}
		if len(depsUpdated) != 2 {
			t.Fatalf("deps_updated length = %d, want 2", len(depsUpdated))
		}
		if depsUpdated[0] != "tick-e5f6" {
			t.Errorf("deps_updated[0] = %v, want %q", depsUpdated[0], "tick-e5f6")
		}
		if depsUpdated[1] != "tick-g7h8" {
			t.Errorf("deps_updated[1] = %v, want %q", depsUpdated[1], "tick-g7h8")
		}
	})

	t.Run("it formats removal with empty deps_updated as empty array in JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "My task"},
			},
			DepsUpdated: nil,
		})

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		depsUpdated, ok := parsed["deps_updated"].([]interface{})
		if !ok {
			t.Fatalf("deps_updated should be array (not null), got %T: %v", parsed["deps_updated"], parsed["deps_updated"])
		}
		if len(depsUpdated) != 0 {
			t.Errorf("deps_updated should be empty, got %d items", len(depsUpdated))
		}
	})

	t.Run("it formats removal with empty removed as empty array in JSON", func(t *testing.T) {
		f := &JSONFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed:     nil,
			DepsUpdated: nil,
		})

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		removed, ok := parsed["removed"].([]interface{})
		if !ok {
			t.Fatalf("removed should be array (not null), got %T: %v", parsed["removed"], parsed["removed"])
		}
		if len(removed) != 0 {
			t.Errorf("removed should be empty, got %d items", len(removed))
		}

		depsUpdated, ok := parsed["deps_updated"].([]interface{})
		if !ok {
			t.Fatalf("deps_updated should be array (not null), got %T: %v", parsed["deps_updated"], parsed["deps_updated"])
		}
		if len(depsUpdated) != 0 {
			t.Errorf("deps_updated should be empty, got %d items", len(depsUpdated))
		}
	})

	t.Run("it includes type in json list items", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Fix login bug", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Add search", Status: task.StatusDone, Priority: 2, Type: "feature", Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)

		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}
		if len(parsed) != 2 {
			t.Fatalf("expected 2 items, got %d", len(parsed))
		}
		if parsed[0]["type"] != "bug" {
			t.Errorf("first task type = %v, want %q", parsed[0]["type"], "bug")
		}
		if parsed[1]["type"] != "feature" {
			t.Errorf("second task type = %v, want %q", parsed[1]["type"], "feature")
		}
	})

	t.Run("it displays tags in json format show output", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Tagged task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			Tags:      []string{"backend", "ui"},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		tags, ok := parsed["tags"].([]interface{})
		if !ok {
			t.Fatalf("tags should be array, got %T: %v", parsed["tags"], parsed["tags"])
		}
		if len(tags) != 2 {
			t.Fatalf("tags length = %d, want 2", len(tags))
		}
		if tags[0] != "backend" {
			t.Errorf("tags[0] = %v, want %q", tags[0], "backend")
		}
		if tags[1] != "ui" {
			t.Errorf("tags[1] = %v, want %q", tags[1], "ui")
		}
	})

	t.Run("it shows empty tags array in json format when task has no tags", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No tags",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		tags, ok := parsed["tags"].([]interface{})
		if !ok {
			t.Fatalf("tags should be array (not null), got %T: %v", parsed["tags"], parsed["tags"])
		}
		if len(tags) != 0 {
			t.Errorf("tags should be empty, got %d items", len(tags))
		}
	})

	t.Run("it includes type in json show output", func(t *testing.T) {
		f := &JSONFormatter{}
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

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}
		if parsed["type"] != "bug" {
			t.Errorf("type = %v, want %q", parsed["type"], "bug")
		}
	})

	t.Run("it includes empty type string in json list when unset", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "No type task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)

		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}
		if len(parsed) != 1 {
			t.Fatalf("expected 1 item, got %d", len(parsed))
		}
		typeVal, exists := parsed[0]["type"]
		if !exists {
			t.Fatal("type key should be present even when unset")
		}
		typeStr, ok := typeVal.(string)
		if !ok {
			t.Fatalf("type should be string, got %T: %v", typeVal, typeVal)
		}
		if typeStr != "" {
			t.Errorf("type = %q, want empty string", typeStr)
		}
	})

	t.Run("it uses 2-space indentation", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Test", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)

		// Should contain 2-space indented lines (not tabs, not 4 spaces)
		expected := "  \"id\": \"tick-a1b2\""
		if !strings.Contains(result, expected) {
			t.Errorf("expected 2-space indentation with %q, got:\n%s", expected, result)
		}
	})

	t.Run("it displays refs in json show output", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Task with refs",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			Refs:      []string{"gh-123", "JIRA-456"},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		refs, ok := parsed["refs"].([]interface{})
		if !ok {
			t.Fatalf("refs should be array, got %T: %v", parsed["refs"], parsed["refs"])
		}
		if len(refs) != 2 {
			t.Fatalf("refs length = %d, want 2", len(refs))
		}
		if refs[0] != "gh-123" {
			t.Errorf("refs[0] = %v, want %q", refs[0], "gh-123")
		}
		if refs[1] != "JIRA-456" {
			t.Errorf("refs[1] = %v, want %q", refs[1], "JIRA-456")
		}
	})

	t.Run("it shows empty refs array in json when no refs", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No refs",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		refs, ok := parsed["refs"].([]interface{})
		if !ok {
			t.Fatalf("refs should be array (not null), got %T: %v", parsed["refs"], parsed["refs"])
		}
		if len(refs) != 0 {
			t.Errorf("refs should be empty, got %d items", len(refs))
		}
	})

	t.Run("it displays notes in json show output with text and created", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
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
			Notes: []task.Note{
				{Text: "Started investigating the auth flow", Created: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)},
				{Text: "Root cause found", Created: time.Date(2026, 2, 27, 14, 30, 0, 0, time.UTC)},
			},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		notes, ok := parsed["notes"].([]interface{})
		if !ok {
			t.Fatalf("notes should be array, got %T: %v", parsed["notes"], parsed["notes"])
		}
		if len(notes) != 2 {
			t.Fatalf("notes length = %d, want 2", len(notes))
		}

		note0 := notes[0].(map[string]interface{})
		if note0["text"] != "Started investigating the auth flow" {
			t.Errorf("notes[0].text = %v, want %q", note0["text"], "Started investigating the auth flow")
		}
		if note0["created"] != "2026-02-27T10:00:00Z" {
			t.Errorf("notes[0].created = %v, want %q", note0["created"], "2026-02-27T10:00:00Z")
		}

		note1 := notes[1].(map[string]interface{})
		if note1["text"] != "Root cause found" {
			t.Errorf("notes[1].text = %v, want %q", note1["text"], "Root cause found")
		}
		if note1["created"] != "2026-02-27T14:30:00Z" {
			t.Errorf("notes[1].created = %v, want %q", note1["created"], "2026-02-27T14:30:00Z")
		}
	})

	t.Run("it shows empty notes array in json when no notes", func(t *testing.T) {
		f := &JSONFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No notes",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		notes, ok := parsed["notes"].([]interface{})
		if !ok {
			t.Fatalf("notes should be array (not null), got %T: %v", parsed["notes"], parsed["notes"])
		}
		if len(notes) != 0 {
			t.Errorf("notes should be empty, got %d items", len(notes))
		}
	})
}
