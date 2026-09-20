package cli

import (
	"encoding/json"
	"slices"
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

		var parsed []map[string]any
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

		assertRenderedJSONIsEmptyArray(t, f.FormatTaskList([]task.Task{}))
		assertRenderedJSONIsEmptyArray(t, f.FormatTaskList(nil))
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

		var parsed map[string]any
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
		blockedBy, ok := parsed["blocked_by"].([]any)
		if !ok {
			t.Fatalf("blocked_by is not an array: %v", parsed["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Fatalf("blocked_by length = %d, want 1", len(blockedBy))
		}
		blocker := blockedBy[0].(map[string]any)
		if blocker["id"] != "tick-c3d4" {
			t.Errorf("blocker id = %v, want %q", blocker["id"], "tick-c3d4")
		}

		// Verify children array
		children, ok := parsed["children"].([]any)
		if !ok {
			t.Fatalf("children is not an array: %v", parsed["children"])
		}
		if len(children) != 1 {
			t.Fatalf("children length = %d, want 1", len(children))
		}
		child := children[0].(map[string]any)
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

		var parsed map[string]any
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// blocked_by must be present as empty array, not null or missing
		blockedBy, ok := parsed["blocked_by"].([]any)
		if !ok {
			t.Fatalf("blocked_by should be array, got %T: %v", parsed["blocked_by"], parsed["blocked_by"])
		}
		if len(blockedBy) != 0 {
			t.Errorf("blocked_by should be empty, got %d items", len(blockedBy))
		}

		// children must be present as empty array, not null or missing
		children, ok := parsed["children"].([]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// Must still be [], not null
		blockedBy, ok := parsed["blocked_by"].([]any)
		if !ok {
			t.Fatalf("blocked_by should be array even with nil input, got %T: %v", parsed["blocked_by"], parsed["blocked_by"])
		}
		if len(blockedBy) != 0 {
			t.Errorf("blocked_by should be empty, got %d items", len(blockedBy))
		}

		children, ok := parsed["children"].([]any)
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

		var parsed map[string]any
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

		var parsed map[string]any
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// Verify top-level total
		if parsed["total"] != float64(47) {
			t.Errorf("total = %v, want 47", parsed["total"])
		}

		// Verify by_status nested object
		byStatus, ok := parsed["by_status"].(map[string]any)
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
		workflow, ok := parsed["workflow"].(map[string]any)
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
		byPriority, ok := parsed["by_priority"].([]any)
		if !ok {
			t.Fatalf("by_priority should be array, got %T: %v", parsed["by_priority"], parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}
		expectedCounts := []float64{2, 8, 25, 7, 5}
		for i, expected := range expectedCounts {
			entry := byPriority[i].(map[string]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		byPriority, ok := parsed["by_priority"].([]any)
		if !ok {
			t.Fatalf("by_priority should be array, got %T: %v", parsed["by_priority"], parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}
		for i := range 5 {
			entry := byPriority[i].(map[string]any)
			if entry["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %v", i, entry["priority"], i)
			}
			if entry["count"] != float64(0) {
				t.Errorf("by_priority[%d].count = %v, want 0", i, entry["count"])
			}
		}
	})

	t.Run("it formats dep/message as JSON objects", func(t *testing.T) {
		f := &JSONFormatter{}

		// Dep change - added
		depAddResult := f.FormatDepChange("added", "tick-c3d4", "tick-a1b2")
		var depAddObj map[string]any
		if err := json.Unmarshal([]byte(depAddResult), &depAddObj); err != nil {
			t.Fatalf("dep add invalid JSON: %v\nresult: %s", err, depAddResult)
		}
		if depAddObj["action"] != "added" {
			t.Errorf("dep action = %v, want %q", depAddObj["action"], "added")
		}
		if depAddObj["task_id"] != "tick-c3d4" {
			t.Errorf("dep task_id = %v, want %q", depAddObj["task_id"], "tick-c3d4")
		}
		if depAddObj["blocker"] != "tick-a1b2" {
			t.Errorf("dep blocker = %v, want %q", depAddObj["blocker"], "tick-a1b2")
		}

		// Dep change - removed
		depRmResult := f.FormatDepChange("removed", "tick-c3d4", "tick-a1b2")
		var depRmObj map[string]any
		if err := json.Unmarshal([]byte(depRmResult), &depRmObj); err != nil {
			t.Fatalf("dep remove invalid JSON: %v\nresult: %s", err, depRmResult)
		}
		if depRmObj["action"] != "removed" {
			t.Errorf("dep remove action = %v, want %q", depRmObj["action"], "removed")
		}

		// Message
		msgResult := f.FormatMessage("Tick initialized in /path/to/project")
		var msgObj map[string]any
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		removed, ok := parsed["removed"].([]any)
		if !ok {
			t.Fatalf("removed should be array, got %T: %v", parsed["removed"], parsed["removed"])
		}
		if len(removed) != 1 {
			t.Fatalf("removed length = %d, want 1", len(removed))
		}
		item := removed[0].(map[string]any)
		if item["id"] != "tick-a1b2" {
			t.Errorf("removed[0].id = %v, want %q", item["id"], "tick-a1b2")
		}
		if item["title"] != "My task" {
			t.Errorf("removed[0].title = %v, want %q", item["title"], "My task")
		}

		depsUpdated, ok := parsed["deps_updated"].([]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		removed, ok := parsed["removed"].([]any)
		if !ok {
			t.Fatalf("removed should be array, got %T: %v", parsed["removed"], parsed["removed"])
		}
		if len(removed) != 2 {
			t.Fatalf("removed length = %d, want 2", len(removed))
		}
		first := removed[0].(map[string]any)
		if first["id"] != "tick-a1b2" {
			t.Errorf("removed[0].id = %v, want %q", first["id"], "tick-a1b2")
		}
		second := removed[1].(map[string]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		depsUpdated, ok := parsed["deps_updated"].([]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		depsUpdated, ok := parsed["deps_updated"].([]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		removed, ok := parsed["removed"].([]any)
		if !ok {
			t.Fatalf("removed should be array (not null), got %T: %v", parsed["removed"], parsed["removed"])
		}
		if len(removed) != 0 {
			t.Errorf("removed should be empty, got %d items", len(removed))
		}

		depsUpdated, ok := parsed["deps_updated"].([]any)
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

		var parsed []map[string]any
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		tags, ok := parsed["tags"].([]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		tags, ok := parsed["tags"].([]any)
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

		var parsed map[string]any
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

		var parsed []map[string]any
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		refs, ok := parsed["refs"].([]any)
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		refs, ok := parsed["refs"].([]any)
		if !ok {
			t.Fatalf("refs should be array (not null), got %T: %v", parsed["refs"], parsed["refs"])
		}
		if len(refs) != 0 {
			t.Errorf("refs should be empty, got %d items", len(refs))
		}
	})

	t.Run("it numbers notes from 1 in json output", func(t *testing.T) {
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		notes, ok := parsed["notes"].([]any)
		if !ok {
			t.Fatalf("notes should be array, got %T: %v", parsed["notes"], parsed["notes"])
		}
		if len(notes) != 2 {
			t.Fatalf("notes length = %d, want 2", len(notes))
		}

		note0 := notes[0].(map[string]any)
		if note0["index"] != float64(1) {
			t.Errorf("notes[0].index = %v, want 1", note0["index"])
		}
		if note0["text"] != "Started investigating the auth flow" {
			t.Errorf("notes[0].text = %v, want %q", note0["text"], "Started investigating the auth flow")
		}
		if note0["created"] != "2026-02-27T10:00:00Z" {
			t.Errorf("notes[0].created = %v, want %q", note0["created"], "2026-02-27T10:00:00Z")
		}

		note1 := notes[1].(map[string]any)
		if note1["index"] != float64(2) {
			t.Errorf("notes[1].index = %v, want 2", note1["index"])
		}
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		notes, ok := parsed["notes"].([]any)
		if !ok {
			t.Fatalf("notes should be array (not null), got %T: %v", parsed["notes"], parsed["notes"])
		}
		if len(notes) != 0 {
			t.Errorf("notes should be empty, got %d items", len(notes))
		}
	})
}

func TestJSONFormatDepTree(t *testing.T) {
	f := &JSONFormatter{}

	t.Run("it renders full graph mode as structured JSON", func(t *testing.T) {
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		if parsed["mode"] != "full" {
			t.Errorf("mode = %v, want %q", parsed["mode"], "full")
		}
		if parsed["chains"] != float64(1) {
			t.Errorf("chains = %v, want 1", parsed["chains"])
		}
		if parsed["longest"] != float64(1) {
			t.Errorf("longest = %v, want 1", parsed["longest"])
		}
		if parsed["blocked"] != float64(1) {
			t.Errorf("blocked = %v, want 1", parsed["blocked"])
		}

		roots, ok := parsed["roots"].([]any)
		if !ok {
			t.Fatalf("roots should be array, got %T: %v", parsed["roots"], parsed["roots"])
		}
		if len(roots) != 1 {
			t.Fatalf("roots length = %d, want 1", len(roots))
		}

		root := roots[0].(map[string]any)
		rootTask, ok := root["task"].(map[string]any)
		if !ok {
			t.Fatalf("root.task should be object, got %T", root["task"])
		}
		if rootTask["id"] != "tick-aaa111" {
			t.Errorf("root task id = %v, want %q", rootTask["id"], "tick-aaa111")
		}

		children, ok := root["children"].([]any)
		if !ok {
			t.Fatalf("root.children should be array, got %T", root["children"])
		}
		if len(children) != 1 {
			t.Fatalf("root.children length = %d, want 1", len(children))
		}

		child := children[0].(map[string]any)
		childTask := child["task"].(map[string]any)
		if childTask["id"] != "tick-bbb222" {
			t.Errorf("child task id = %v, want %q", childTask["id"], "tick-bbb222")
		}
	})

	t.Run("it renders multi-level chain in full graph", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "A", Status: "open"},
					Children: []DepTreeNode{
						{
							Task: DepTreeTask{ID: "tick-bbb222", Title: "B", Status: "in_progress"},
							Children: []DepTreeNode{
								{Task: DepTreeTask{ID: "tick-ccc333", Title: "C", Status: "done"}},
							},
						},
					},
				},
			},
			ChainCount:   1,
			LongestChain: 2,
			BlockedCount: 2,
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		roots := parsed["roots"].([]any)
		root := roots[0].(map[string]any)
		level1 := root["children"].([]any)
		if len(level1) != 1 {
			t.Fatalf("level 1 children = %d, want 1", len(level1))
		}
		child1 := level1[0].(map[string]any)
		level2 := child1["children"].([]any)
		if len(level2) != 1 {
			t.Fatalf("level 2 children = %d, want 1", len(level2))
		}
		grandchild := level2[0].(map[string]any)
		gcTask := grandchild["task"].(map[string]any)
		if gcTask["id"] != "tick-ccc333" {
			t.Errorf("grandchild id = %v, want %q", gcTask["id"], "tick-ccc333")
		}
		if gcTask["status"] != "done" {
			t.Errorf("grandchild status = %v, want %q", gcTask["status"], "done")
		}
	})

	t.Run("it renders roots as [] not null when empty", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Roots:        nil,
			ChainCount:   0,
			LongestChain: 0,
			BlockedCount: 0,
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		roots, ok := parsed["roots"].([]any)
		if !ok {
			t.Fatalf("roots should be array (not null), got %T: %v", parsed["roots"], parsed["roots"])
		}
		if len(roots) != 0 {
			t.Errorf("roots should be empty, got %d items", len(roots))
		}
	})

	t.Run("it renders leaf children as [] not null", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Roots: []DepTreeNode{
				{
					Task: DepTreeTask{ID: "tick-aaa111", Title: "Root", Status: "open"},
					Children: []DepTreeNode{
						{Task: DepTreeTask{ID: "tick-bbb222", Title: "Leaf", Status: "open"}},
					},
				},
			},
			ChainCount:   1,
			LongestChain: 1,
			BlockedCount: 1,
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		roots := parsed["roots"].([]any)
		root := roots[0].(map[string]any)
		children := root["children"].([]any)
		leaf := children[0].(map[string]any)

		leafChildren, ok := leaf["children"].([]any)
		if !ok {
			t.Fatalf("leaf children should be array (not null), got %T: %v", leaf["children"], leaf["children"])
		}
		if len(leafChildren) != 0 {
			t.Errorf("leaf children should be empty, got %d items", len(leafChildren))
		}
	})

	t.Run("it duplicates diamond dependency nodes in tree", func(t *testing.T) {
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

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		roots := parsed["roots"].([]any)
		root := roots[0].(map[string]any)
		children := root["children"].([]any)
		if len(children) != 2 {
			t.Fatalf("root children = %d, want 2", len(children))
		}

		// B's children should contain D
		b := children[0].(map[string]any)
		bChildren := b["children"].([]any)
		if len(bChildren) != 1 {
			t.Fatalf("B children = %d, want 1", len(bChildren))
		}
		bChild := bChildren[0].(map[string]any)
		bChildTask := bChild["task"].(map[string]any)
		if bChildTask["id"] != "tick-ddd444" {
			t.Errorf("B child id = %v, want %q", bChildTask["id"], "tick-ddd444")
		}

		// C's children should also contain D (duplicate)
		c := children[1].(map[string]any)
		cChildren := c["children"].([]any)
		if len(cChildren) != 1 {
			t.Fatalf("C children = %d, want 1", len(cChildren))
		}
		cChild := cChildren[0].(map[string]any)
		cChildTask := cChild["task"].(map[string]any)
		if cChildTask["id"] != "tick-ddd444" {
			t.Errorf("C child id = %v, want %q", cChildTask["id"], "tick-ddd444")
		}
	})

	t.Run("it renders focused mode with both directions", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-bbb222", Title: "Target", Status: "in_progress"},
			BlockedBy: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-aaa111", Title: "Blocker", Status: "open"}},
			},
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-ccc333", Title: "Blocked", Status: "open"}},
			},
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		if parsed["mode"] != "focused" {
			t.Errorf("mode = %v, want %q", parsed["mode"], "focused")
		}

		target, ok := parsed["target"].(map[string]any)
		if !ok {
			t.Fatalf("target should be object, got %T", parsed["target"])
		}
		if target["id"] != "tick-bbb222" {
			t.Errorf("target id = %v, want %q", target["id"], "tick-bbb222")
		}

		blockedBy, ok := parsed["blocked_by"].([]any)
		if !ok {
			t.Fatalf("blocked_by should be array, got %T: %v", parsed["blocked_by"], parsed["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Fatalf("blocked_by length = %d, want 1", len(blockedBy))
		}

		blocks, ok := parsed["blocks"].([]any)
		if !ok {
			t.Fatalf("blocks should be array, got %T: %v", parsed["blocks"], parsed["blocks"])
		}
		if len(blocks) != 1 {
			t.Fatalf("blocks length = %d, want 1", len(blocks))
		}
	})

	t.Run("it emits an empty blocked_by when only downstream exists", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target:    &DepTreeTask{ID: "tick-aaa111", Title: "Root blocker", Status: "open"},
			BlockedBy: nil,
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-bbb222", Title: "Blocked", Status: "open"}},
			},
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		blockedBy, ok := parsed["blocked_by"].([]any)
		if !ok {
			t.Fatalf("blocked_by should be array (not null), got %T: %v", parsed["blocked_by"], parsed["blocked_by"])
		}
		if len(blockedBy) != 0 {
			t.Errorf("blocked_by should be empty, got %d items", len(blockedBy))
		}

		blocks, ok := parsed["blocks"].([]any)
		if !ok {
			t.Fatalf("blocks should be array, got %T: %v", parsed["blocks"], parsed["blocks"])
		}
		if len(blocks) != 1 {
			t.Fatalf("blocks length = %d, want 1", len(blocks))
		}
	})

	t.Run("it emits an empty blocks when only upstream exists", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-bbb222", Title: "Leaf", Status: "open"},
			BlockedBy: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-aaa111", Title: "Blocker", Status: "done"}},
			},
			Blocks: nil,
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		blocks, ok := parsed["blocks"].([]any)
		if !ok {
			t.Fatalf("blocks should be array (not null), got %T: %v", parsed["blocks"], parsed["blocks"])
		}
		if len(blocks) != 0 {
			t.Errorf("blocks should be empty, got %d items", len(blocks))
		}

		blockedBy, ok := parsed["blocked_by"].([]any)
		if !ok {
			t.Fatalf("blocked_by should be array, got %T: %v", parsed["blocked_by"], parsed["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Fatalf("blocked_by length = %d, want 1", len(blockedBy))
		}
	})

	t.Run("it uses snake_case for all keys", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-bbb222", Title: "Target", Status: "open"},
			BlockedBy: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-aaa111", Title: "Blocker", Status: "open"}},
			},
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-ccc333", Title: "Blocked", Status: "open"}},
			},
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		// Expected keys
		expectedKeys := []string{"mode", "target", "blocked_by", "blocks"}
		for _, key := range expectedKeys {
			if _, exists := parsed[key]; !exists {
				t.Errorf("expected snake_case key %q not found", key)
			}
		}

		// No camelCase keys
		camelKeys := []string{"blockedBy", "BlockedBy", "Blocks", "Target", "Mode"}
		for _, key := range camelKeys {
			if _, exists := parsed[key]; exists {
				t.Errorf("found non-snake_case key %q", key)
			}
		}

		// Check full graph keys too
		fullResult := f.FormatDepTree(DepTreeResult{
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

		var fullParsed map[string]any
		if err := json.Unmarshal([]byte(fullResult), &fullParsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, fullResult)
		}

		fullExpected := []string{"mode", "roots", "chains", "longest", "blocked"}
		for _, key := range fullExpected {
			if _, exists := fullParsed[key]; !exists {
				t.Errorf("expected snake_case key %q not found in full graph", key)
			}
		}
	})

	t.Run("it renders target task with id, title, status", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{
			Target: &DepTreeTask{ID: "tick-abc123", Title: "My target task", Status: "in_progress"},
			Blocks: []DepTreeNode{
				{Task: DepTreeTask{ID: "tick-def456", Title: "Downstream", Status: "open"}},
			},
		})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		target, ok := parsed["target"].(map[string]any)
		if !ok {
			t.Fatalf("target should be object, got %T", parsed["target"])
		}
		if target["id"] != "tick-abc123" {
			t.Errorf("target id = %v, want %q", target["id"], "tick-abc123")
		}
		if target["title"] != "My target task" {
			t.Errorf("target title = %v, want %q", target["title"], "My target task")
		}
		if target["status"] != "in_progress" {
			t.Errorf("target status = %v, want %q", target["status"], "in_progress")
		}
	})

	t.Run("it produces valid 2-space indented JSON", func(t *testing.T) {
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

		// Must be valid JSON
		if !json.Valid([]byte(result)) {
			t.Fatalf("output is not valid JSON:\n%s", result)
		}

		// Must contain 2-space indentation
		if !strings.Contains(result, "  \"mode\"") {
			t.Errorf("expected 2-space indentation, got:\n%s", result)
		}

		// Must not contain tab indentation
		if strings.Contains(result, "\t") {
			t.Errorf("should not contain tab indentation, got:\n%s", result)
		}
	})

	t.Run("it emits both directions on a focused document with no dependencies", func(t *testing.T) {
		parsed := parseFocusedNoDepsJSON(t, f)

		if _, exists := parsed["message"]; exists {
			t.Errorf("message key should be absent, got %v", parsed["message"])
		}
		if _, exists := parsed["blocked_by"]; !exists {
			t.Error("blocked_by key should be present")
		}
		if _, exists := parsed["blocks"]; !exists {
			t.Error("blocks key should be present")
		}
	})

	t.Run("it emits blocked_by and blocks as empty arrays never null", func(t *testing.T) {
		parsed := parseFocusedNoDepsJSON(t, f)

		for _, key := range []string{"blocked_by", "blocks"} {
			nodes, ok := parsed[key].([]any)
			if !ok {
				t.Fatalf("%s should be array (not null), got %T: %v", key, parsed[key], parsed[key])
			}
			if len(nodes) != 0 {
				t.Errorf("%s should be empty, got %d items", key, len(nodes))
			}
		}
	})

	t.Run("it keeps target as a nested object", func(t *testing.T) {
		parsed := parseFocusedNoDepsJSON(t, f)

		target, ok := parsed["target"].(map[string]any)
		if !ok {
			t.Fatalf("target should be object, got %T: %v", parsed["target"], parsed["target"])
		}
		if target["id"] != "tick-aaa111" {
			t.Errorf("target.id = %v, want %q", target["id"], "tick-aaa111")
		}
		if target["title"] != "Task A" {
			t.Errorf("target.title = %v, want %q", target["title"], "Task A")
		}
		if target["status"] != "open" {
			t.Errorf("target.status = %v, want %q", target["status"], "open")
		}
	})

	t.Run("it emits the emptied full document instead of a message", func(t *testing.T) {
		result := f.FormatDepTree(DepTreeResult{Message: "No dependencies found."})

		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
		}

		if _, exists := parsed["message"]; exists {
			t.Errorf("message key should be absent, got %v", parsed["message"])
		}

		roots, ok := parsed["roots"].([]any)
		if !ok {
			t.Fatalf("roots should be array (not null), got %T: %v", parsed["roots"], parsed["roots"])
		}
		if len(roots) != 0 {
			t.Errorf("roots should be empty, got %d items", len(roots))
		}

		for _, key := range []string{"chains", "longest", "blocked"} {
			if parsed[key] != float64(0) {
				t.Errorf("%s = %v, want 0", key, parsed[key])
			}
		}
	})

	t.Run("it keeps mode on both documents", func(t *testing.T) {
		full := f.FormatDepTree(DepTreeResult{Message: "No dependencies found."})
		focused := f.FormatDepTree(DepTreeResult{
			Target:  &DepTreeTask{ID: "tick-aaa111", Title: "Task A", Status: "open"},
			Message: "No dependencies.",
		})

		for _, tc := range []struct{ output, want string }{
			{full, "full"},
			{focused, "focused"},
		} {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(tc.output), &parsed); err != nil {
				t.Fatalf("invalid JSON: %v\nresult: %s", err, tc.output)
			}
			if parsed["mode"] != tc.want {
				t.Errorf("mode = %v, want %q", parsed["mode"], tc.want)
			}
		}
	})

	t.Run("it still renders a message for init", func(t *testing.T) {
		var parsed map[string]any
		output := f.FormatMessage("Initialized tick project")
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nresult: %s", err, output)
		}
		if parsed["message"] != "Initialized tick project" {
			t.Errorf("message = %v, want %q", parsed["message"], "Initialized tick project")
		}
	})
}

// parseFocusedNoDepsJSON formats a focused dep-tree result with no dependencies
// either way and returns the unmarshalled document.
func parseFocusedNoDepsJSON(t *testing.T, f *JSONFormatter) map[string]any {
	t.Helper()

	result := f.FormatDepTree(DepTreeResult{
		Target:  &DepTreeTask{ID: "tick-aaa111", Title: "Task A", Status: "open"},
		Message: "No dependencies.",
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nresult: %s", err, result)
	}
	return parsed
}

func TestJSONFilteredTaskDetail(t *testing.T) {
	f := &JSONFormatter{}

	filtered := func(t *testing.T, detail TaskDetail, value string) string {
		t.Helper()
		detail.Fields = fieldSelection(t, value)
		return f.FormatTaskDetail(detail)
	}

	parse := func(t *testing.T, document string) map[string]any {
		t.Helper()
		var parsed map[string]any
		if err := json.Unmarshal([]byte(document), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\ndocument: %s", err, document)
		}
		return parsed
	}

	t.Run("it renders only the selected keys", func(t *testing.T) {
		doc := parse(t, filtered(t, richDetail(), "title,status"))

		assertJSONKeySet(t, doc, "title", "status")
		if doc["title"] != "Add retry to the sync worker" {
			t.Errorf("title = %v, want %q", doc["title"], "Add retry to the sync worker")
		}
		if doc["status"] != "in_progress" {
			t.Errorf("status = %v, want %q", doc["status"], "in_progress")
		}
	})

	t.Run("it does not carry id unless asked", func(t *testing.T) {
		assertJSONKeySet(t, parse(t, filtered(t, richDetail(), "title")), "title")
	})

	t.Run("it renders the selected scalars with their own types", func(t *testing.T) {
		doc := parse(t, filtered(t, richDetail(), "id,priority,type,parent,created,updated,closed"))

		assertJSONKeySet(t, doc, "id", "priority", "type", "parent", "created", "updated", "closed")
		for key, want := range map[string]any{
			"id":       "tick-a1b2",
			"priority": float64(1),
			"type":     "bug",
			"parent":   "tick-p4r3",
			"created":  "2026-03-01T09:00:00Z",
			"updated":  "2026-03-01T09:00:00Z",
			"closed":   "2026-03-02T09:00:00Z",
		} {
			if doc[key] != want {
				t.Errorf("%s = %v, want %v", key, doc[key], want)
			}
		}
	})

	t.Run("it keeps the index on selected notes", func(t *testing.T) {
		created := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
		detail := detailWithNotes([]task.Note{
			{Text: "first", Created: created},
			{Text: "second", Created: created},
		})
		doc := parse(t, filtered(t, detail, "notes"))

		assertJSONKeySet(t, doc, "notes")
		notes, ok := doc["notes"].([]any)
		if !ok || len(notes) != 2 {
			t.Fatalf("notes = %v, want 2 entries", doc["notes"])
		}
		for i, want := range []string{"first", "second"} {
			note, ok := notes[i].(map[string]any)
			if !ok {
				t.Fatalf("note %d = %v, want an object", i, notes[i])
			}
			if note["index"] != float64(i+1) {
				t.Errorf("note %d index = %v, want %d", i, note["index"], i+1)
			}
			if note["text"] != want {
				t.Errorf("note %d text = %v, want %q", i, note["text"], want)
			}
			if note["created"] != "2026-03-01T09:00:00Z" {
				t.Errorf("note %d created = %v, want %q", i, note["created"], "2026-03-01T09:00:00Z")
			}
		}
	})

	t.Run("it renders the selected related sections as arrays", func(t *testing.T) {
		doc := parse(t, filtered(t, richDetail(), "blocked_by,children"))

		assertJSONKeySet(t, doc, "blocked_by", "children")
		for _, key := range []string{"blocked_by", "children"} {
			rows, ok := doc[key].([]any)
			if !ok || len(rows) != 1 {
				t.Fatalf("%s = %v, want 1 entry", key, doc[key])
			}
		}
	})

	t.Run("it renders empty related sections as empty arrays", func(t *testing.T) {
		doc := parse(t, filtered(t, detailWithDescription(""), "blocked_by,children"))

		assertJSONKeySet(t, doc, "blocked_by", "children")
		for _, key := range []string{"blocked_by", "children"} {
			assertJSONEmptyArray(t, doc, key)
		}
	})

	t.Run("it renders selected empty tags as an empty array", func(t *testing.T) {
		doc := parse(t, filtered(t, detailWithDescription(""), "tags,title"))

		assertJSONKeySet(t, doc, "tags", "title")
		assertJSONEmptyArray(t, doc, "tags")
	})

	t.Run("it renders selected empty refs as an empty array", func(t *testing.T) {
		doc := parse(t, filtered(t, detailWithDescription(""), "refs,title"))

		assertJSONKeySet(t, doc, "refs", "title")
		assertJSONEmptyArray(t, doc, "refs")
	})

	t.Run("it renders selected empty notes as an empty array", func(t *testing.T) {
		doc := parse(t, filtered(t, detailWithNotes(nil), "notes"))

		assertJSONKeySet(t, doc, "notes")
		assertJSONEmptyArray(t, doc, "notes")
	})

	t.Run("it renders a selected empty description as an empty string", func(t *testing.T) {
		doc := parse(t, filtered(t, detailWithDescription(""), "description"))

		assertJSONKeySet(t, doc, "description")
		if doc["description"] != "" {
			t.Errorf("description = %v, want %q", doc["description"], "")
		}
	})

	t.Run("it renders a selected empty type as an empty string", func(t *testing.T) {
		doc := parse(t, filtered(t, detailWithDescription(""), "type"))

		assertJSONKeySet(t, doc, "type")
		if doc["type"] != "" {
			t.Errorf("type = %v, want %q", doc["type"], "")
		}
	})

	t.Run("it omits a selected absent parent", func(t *testing.T) {
		assertJSONKeySet(t, parse(t, filtered(t, detailWithDescription(""), "parent,title")), "title")
	})

	t.Run("it omits a selected absent closed", func(t *testing.T) {
		assertJSONKeySet(t, parse(t, filtered(t, detailWithDescription(""), "closed,title")), "title")
	})

	t.Run("it renders nothing when no selected key survives", func(t *testing.T) {
		if result := filtered(t, detailWithDescription(""), "parent,closed"); result != "" {
			t.Errorf("result = %q, want empty", result)
		}
	})

	t.Run("it never carries the changed key", func(t *testing.T) {
		detail := richDetail()
		detail.Changes = &StatusChanges{}
		detail.Fields = fieldSelection(t, "title")

		assertJSONKeySet(t, parse(t, f.FormatTaskDetail(detail)), "title")
	})

	t.Run("it leaves unfiltered output unchanged", func(t *testing.T) {
		doc := parse(t, f.FormatTaskDetail(richDetail()))

		assertJSONKeySet(t, doc, "id", "title", "status", "priority", "type", "tags", "refs",
			"notes", "description", "parent", "created", "updated", "closed", "blocked_by", "children")
	})

	t.Run("it returns a filtered document's keys in document order", func(t *testing.T) {
		for _, tc := range []struct {
			selection string
			want      []string
		}{
			{"type,tags", []string{"type", "tags"}},
			{"title,notes.2", []string{"title", "notes"}},
			{"status,title,id", []string{"id", "title", "status"}},
		} {
			got := jsonKeyOrder(t, filtered(t, richDetail(), tc.selection))
			if !slices.Equal(got, tc.want) {
				t.Errorf("%s: key order = %v, want %v", tc.selection, got, tc.want)
			}
		}
	})

	t.Run("it orders a filtered document as the full document restricted to the selection", func(t *testing.T) {
		full := jsonKeyOrder(t, f.FormatTaskDetail(richDetail()))

		for _, selection := range []string{
			"closed,created,id",
			"children,blocked_by,description",
			"refs,notes,tags,title",
			"updated,priority,parent,status,type",
		} {
			got := jsonKeyOrder(t, filtered(t, richDetail(), selection))
			want := slices.DeleteFunc(slices.Clone(full), func(key string) bool {
				return !slices.Contains(got, key)
			})
			if !slices.Equal(got, want) {
				t.Errorf("%s: key order = %v, want %v", selection, got, want)
			}
		}
	})

	t.Run("it renders a full document's keys in the declared order", func(t *testing.T) {
		want := []string{"id", "title", "status", "priority", "type", "tags", "refs", "notes",
			"description", "parent", "created", "updated", "closed", "blocked_by", "children"}
		if got := jsonKeyOrder(t, f.FormatTaskDetail(richDetail())); !slices.Equal(got, want) {
			t.Errorf("key order = %v, want %v", got, want)
		}

		detail := richDetail()
		detail.Changes = &StatusChanges{}
		withChanged := append(slices.Clone(want), "changed")
		if got := jsonKeyOrder(t, f.FormatTaskDetail(detail)); !slices.Equal(got, withChanged) {
			t.Errorf("key order with changes = %v, want %v", got, withChanged)
		}
	})

	t.Run("it omits parent and closed from a filtered document when the task does not carry them", func(t *testing.T) {
		got := jsonKeyOrder(t, filtered(t, detailWithDescription(""), "title,parent,closed,status"))
		if want := []string{"title", "status"}; !slices.Equal(got, want) {
			t.Errorf("key order = %v, want %v", got, want)
		}
	})
}

// jsonKeyOrder returns the document's top-level keys in the order they were
// emitted, which unmarshalling into a map loses.
func jsonKeyOrder(t *testing.T, document string) []string {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(document))
	opening, err := decoder.Token()
	if err != nil {
		t.Fatalf("reading document: %v\ndocument: %s", err, document)
	}
	if delim, ok := opening.(json.Delim); !ok || delim != '{' {
		t.Fatalf("document opens with %v, want an object", opening)
	}

	keys := make([]string, 0, 16)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			t.Fatalf("reading key: %v\ndocument: %s", err, document)
		}
		key, ok := token.(string)
		if !ok {
			t.Fatalf("key token = %v, want a string", token)
		}
		keys = append(keys, key)

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("reading value for %q: %v\ndocument: %s", key, err, document)
		}
	}
	return keys
}

func assertJSONKeySet(t *testing.T, doc map[string]any, want ...string) {
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

func assertJSONEmptyArray(t *testing.T, doc map[string]any, key string) {
	t.Helper()
	value, ok := doc[key].([]any)
	if !ok {
		t.Fatalf("%s = %v, want an array", key, doc[key])
	}
	if value == nil || len(value) != 0 {
		t.Errorf("%s = %v, want an empty array", key, value)
	}
}

func assertRenderedJSONIsEmptyArray(t *testing.T, rendered string) {
	t.Helper()
	var parsed []any
	if err := json.Unmarshal([]byte(rendered), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nrendered: %s", err, rendered)
	}
	if parsed == nil {
		t.Errorf("rendered JSON decoded to null, want an empty array: %s", rendered)
		return
	}
	if len(parsed) != 0 {
		t.Errorf("rendered JSON decoded to %d items, want 0", len(parsed))
	}
}
