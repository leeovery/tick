package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestJSONFormatterFormatTaskList(t *testing.T) {
	f := &JSONFormatter{}

	t.Run("it formats list as JSON array", func(t *testing.T) {
		var buf bytes.Buffer
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: "Setup Sanctum", Status: "done", Priority: 1},
			{ID: "tick-d4e5f6", Title: "Login endpoint", Status: "open", Priority: 1},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result []map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
		}
		if len(result) != 2 {
			t.Errorf("expected 2 items, got %d", len(result))
		}
	})

	t.Run("it formats empty list as [] not null", func(t *testing.T) {
		var buf bytes.Buffer
		if err := f.FormatTaskList(&buf, []TaskListItem{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		trimmed := bytes.TrimSpace(buf.Bytes())
		if string(trimmed) != "[]" {
			t.Errorf("expected [], got %q", string(trimmed))
		}
	})

	t.Run("it uses snake_case for all keys", func(t *testing.T) {
		var buf bytes.Buffer
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: "Test", Status: "open", Priority: 2},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result []map[string]any
		json.Unmarshal(buf.Bytes(), &result)
		for _, key := range []string{"id", "title", "status", "priority"} {
			if _, ok := result[0][key]; !ok {
				t.Errorf("missing snake_case key %q", key)
			}
		}
	})

	t.Run("it produces valid parseable JSON", func(t *testing.T) {
		var buf bytes.Buffer
		tasks := []TaskListItem{
			{ID: "tick-a1b2c3", Title: `Title with "quotes"`, Status: "open", Priority: 2},
		}
		if err := f.FormatTaskList(&buf, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !json.Valid(buf.Bytes()) {
			t.Errorf("output is not valid JSON: %s", buf.String())
		}
	})
}

func TestJSONFormatterFormatTaskDetail(t *testing.T) {
	f := &JSONFormatter{}

	t.Run("it formats show with all fields", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID:       "tick-a1b2c3",
			Title:    "Setup Sanctum",
			Status:   "in_progress",
			Priority: 1,
			Parent:   &RelatedTask{ID: "tick-e5f6a7", Title: "Auth", Status: "open"},
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T14:30:00Z",
			Closed:   "2026-01-19T16:00:00Z",
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4e5", Title: "DB migrations", Status: "done"},
			},
			Children:    []RelatedTask{{ID: "tick-x1y2z3", Title: "Child", Status: "open"}},
			Description: "Some description",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result["id"] != "tick-a1b2c3" {
			t.Error("expected id field")
		}
		if result["closed"] != "2026-01-19T16:00:00Z" {
			t.Error("expected closed field")
		}
	})

	t.Run("it omits parent/closed when null", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "Test", Status: "open", Priority: 2,
			Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]any
		json.Unmarshal(buf.Bytes(), &result)
		if _, ok := result["parent"]; ok {
			t.Error("parent should be omitted when nil")
		}
		if _, ok := result["closed"]; ok {
			t.Error("closed should be omitted when empty")
		}
	})

	t.Run("it includes blocked_by/children as empty arrays", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "Test", Status: "open", Priority: 2,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]any
		json.Unmarshal(buf.Bytes(), &result)

		bb, ok := result["blocked_by"]
		if !ok {
			t.Fatal("blocked_by should be present")
		}
		if arr, ok := bb.([]any); !ok || len(arr) != 0 {
			t.Errorf("blocked_by should be empty array, got %v", bb)
		}

		ch, ok := result["children"]
		if !ok {
			t.Fatal("children should be present")
		}
		if arr, ok := ch.([]any); !ok || len(arr) != 0 {
			t.Errorf("children should be empty array, got %v", ch)
		}
	})

	t.Run("it formats description as empty string not null", func(t *testing.T) {
		var buf bytes.Buffer
		detail := TaskDetail{
			ID: "tick-a1b2c3", Title: "Test", Status: "open", Priority: 2,
			Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
		}
		if err := f.FormatTaskDetail(&buf, detail); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]any
		json.Unmarshal(buf.Bytes(), &result)
		desc, ok := result["description"]
		if !ok {
			t.Fatal("description should be present")
		}
		if desc != "" {
			t.Errorf("description should be empty string, got %v", desc)
		}
	})
}

func TestJSONFormatterFormatStats(t *testing.T) {
	f := &JSONFormatter{}

	t.Run("it formats stats as structured nested object", func(t *testing.T) {
		var buf bytes.Buffer
		data := StatsData{
			Total: 47, Open: 12, InProgress: 3, Done: 28, Cancelled: 4,
			Ready: 8, Blocked: 4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}
		if err := f.FormatStats(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result["total"] != float64(47) {
			t.Errorf("expected total 47, got %v", result["total"])
		}
	})

	t.Run("it includes 5 priority rows even at zero", func(t *testing.T) {
		var buf bytes.Buffer
		data := StatsData{}
		if err := f.FormatStats(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]any
		json.Unmarshal(buf.Bytes(), &result)
		bp, ok := result["by_priority"]
		if !ok {
			t.Fatal("expected by_priority")
		}
		arr, ok := bp.([]any)
		if !ok {
			t.Fatal("by_priority should be array")
		}
		if len(arr) != 5 {
			t.Errorf("expected 5 priority entries, got %d", len(arr))
		}
	})
}

func TestJSONFormatterFormatTransition(t *testing.T) {
	f := &JSONFormatter{}

	t.Run("it formats transition as JSON object", func(t *testing.T) {
		var buf bytes.Buffer
		data := TransitionData{ID: "tick-a1b2c3", OldStatus: "open", NewStatus: "in_progress"}
		if err := f.FormatTransition(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]string
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result["id"] != "tick-a1b2c3" {
			t.Error("expected id")
		}
		if result["from"] != "open" {
			t.Error("expected from")
		}
		if result["to"] != "in_progress" {
			t.Error("expected to")
		}
	})
}

func TestJSONFormatterFormatDepChange(t *testing.T) {
	f := &JSONFormatter{}

	t.Run("it formats dep change as JSON object", func(t *testing.T) {
		var buf bytes.Buffer
		data := DepChangeData{Action: "added", TaskID: "tick-c3d4e5", BlockedBy: "tick-a1b2c3"}
		if err := f.FormatDepChange(&buf, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]string
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result["action"] != "added" {
			t.Error("expected action")
		}
		if result["task_id"] != "tick-c3d4e5" {
			t.Error("expected task_id")
		}
		if result["blocked_by"] != "tick-a1b2c3" {
			t.Error("expected blocked_by")
		}
	})
}

func TestJSONFormatterFormatMessage(t *testing.T) {
	f := &JSONFormatter{}

	t.Run("it formats message as JSON object", func(t *testing.T) {
		var buf bytes.Buffer
		if err := f.FormatMessage(&buf, "Cache rebuilt."); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var result map[string]string
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result["message"] != "Cache rebuilt." {
			t.Errorf("expected message, got %v", result["message"])
		}
	})
}
