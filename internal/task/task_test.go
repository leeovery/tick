package task

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	t.Run("it generates IDs matching tick-{6 hex} pattern", func(t *testing.T) {
		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)

		id, err := GenerateID(func(id string) bool { return false })
		if err != nil {
			t.Fatalf("GenerateID returned error: %v", err)
		}
		if !pattern.MatchString(id) {
			t.Errorf("ID %q does not match pattern tick-{6 hex}", id)
		}
	})

	t.Run("it retries on collision up to 5 times", func(t *testing.T) {
		attempts := 0
		exists := func(id string) bool {
			attempts++
			// Collide on first 4 attempts, succeed on 5th
			return attempts < 5
		}

		id, err := GenerateID(exists)
		if err != nil {
			t.Fatalf("GenerateID returned error: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty ID")
		}
		if attempts != 5 {
			t.Errorf("expected 5 attempts, got %d", attempts)
		}
	})

	t.Run("it errors after 5 collision retries", func(t *testing.T) {
		exists := func(id string) bool { return true }

		_, err := GenerateID(exists)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedMsg := "failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != expectedMsg {
			t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("it normalizes IDs to lowercase", func(t *testing.T) {
		result := NormalizeID("TICK-A3F2B7")
		expected := "tick-a3f2b7"
		if result != expected {
			t.Errorf("NormalizeID(%q) = %q, want %q", "TICK-A3F2B7", result, expected)
		}
	})
}

func TestValidateTitle(t *testing.T) {
	t.Run("it rejects empty title", func(t *testing.T) {
		err := ValidateTitle("")
		if err == nil {
			t.Fatal("expected error for empty title, got nil")
		}
	})

	t.Run("it rejects title exceeding 500 characters", func(t *testing.T) {
		longTitle := strings.Repeat("a", 501)
		err := ValidateTitle(longTitle)
		if err == nil {
			t.Fatal("expected error for title exceeding 500 chars, got nil")
		}
	})

	t.Run("it rejects title with newlines", func(t *testing.T) {
		err := ValidateTitle("line one\nline two")
		if err == nil {
			t.Fatal("expected error for title with newlines, got nil")
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		result := TrimTitle("  hello world  ")
		expected := "hello world"
		if result != expected {
			t.Errorf("TrimTitle(%q) = %q, want %q", "  hello world  ", result, expected)
		}
	})

	t.Run("it rejects whitespace-only title", func(t *testing.T) {
		err := ValidateTitle("   ")
		if err == nil {
			t.Fatal("expected error for whitespace-only title, got nil")
		}
	})

	t.Run("it accepts valid title at 500 characters", func(t *testing.T) {
		title := strings.Repeat("a", 500)
		err := ValidateTitle(title)
		if err != nil {
			t.Errorf("expected no error for 500-char title, got: %v", err)
		}
	})

	t.Run("it counts multi-byte Unicode characters as single characters", func(t *testing.T) {
		// 500 Chinese characters (3 bytes each = 1500 bytes, but 500 runes)
		title500 := strings.Repeat("\u4e16", 500)
		err := ValidateTitle(title500)
		if err != nil {
			t.Errorf("expected no error for 500 multi-byte chars (1500 bytes), got: %v", err)
		}

		// 501 Chinese characters should be rejected
		title501 := strings.Repeat("\u4e16", 501)
		err = ValidateTitle(title501)
		if err == nil {
			t.Fatal("expected error for 501 multi-byte chars, got nil")
		}
	})
}

func TestTrimDescription(t *testing.T) {
	t.Run("it trims whitespace from description", func(t *testing.T) {
		result := TrimDescription("  hello world  ")
		expected := "hello world"
		if result != expected {
			t.Errorf("TrimDescription(%q) = %q, want %q", "  hello world  ", result, expected)
		}
	})

	t.Run("it returns empty string for whitespace-only input", func(t *testing.T) {
		result := TrimDescription("   ")
		if result != "" {
			t.Errorf("TrimDescription(%q) = %q, want empty", "   ", result)
		}
	})

	t.Run("it returns empty string for empty input", func(t *testing.T) {
		result := TrimDescription("")
		if result != "" {
			t.Errorf("TrimDescription(%q) = %q, want empty", "", result)
		}
	})
}

func TestValidateDescriptionUpdate(t *testing.T) {
	t.Run("it rejects empty description", func(t *testing.T) {
		err := ValidateDescriptionUpdate("")
		if err == nil {
			t.Fatal("expected error for empty description, got nil")
		}
		if !strings.Contains(err.Error(), "--clear-description") {
			t.Errorf("error should mention --clear-description, got %q", err.Error())
		}
	})

	t.Run("it rejects whitespace-only description", func(t *testing.T) {
		err := ValidateDescriptionUpdate("   ")
		if err == nil {
			t.Fatal("expected error for whitespace-only description, got nil")
		}
		if !strings.Contains(err.Error(), "--clear-description") {
			t.Errorf("error should mention --clear-description, got %q", err.Error())
		}
	})

	t.Run("it accepts valid description", func(t *testing.T) {
		err := ValidateDescriptionUpdate("A valid description")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestValidatePriority(t *testing.T) {
	t.Run("it rejects priority outside 0-4", func(t *testing.T) {
		tests := []struct {
			name     string
			priority int
		}{
			{"negative priority", -1},
			{"priority too high", 5},
			{"large priority", 100},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidatePriority(tt.priority)
				if err == nil {
					t.Errorf("expected error for priority %d, got nil", tt.priority)
				}
			})
		}
	})

	t.Run("it accepts valid priorities", func(t *testing.T) {
		for p := 0; p <= 4; p++ {
			t.Run(fmt.Sprintf("priority %d", p), func(t *testing.T) {
				err := ValidatePriority(p)
				if err != nil {
					t.Errorf("expected no error for priority %d, got: %v", p, err)
				}
			})
		}
	})
}

func TestValidateBlockedBy(t *testing.T) {
	t.Run("it rejects self-reference in blocked_by", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-a1b2c3"})
		if err == nil {
			t.Fatal("expected error for self-reference in blocked_by, got nil")
		}
	})

	t.Run("it accepts valid blocked_by references", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-d4e5f6"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestValidateParent(t *testing.T) {
	t.Run("it rejects self-reference in parent", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-a1b2c3")
		if err == nil {
			t.Fatal("expected error for self-reference in parent, got nil")
		}
	})

	t.Run("it accepts valid parent reference", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-d4e5f6")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("it accepts empty parent", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "")
		if err != nil {
			t.Errorf("expected no error for empty parent, got: %v", err)
		}
	})
}

func TestNewTask(t *testing.T) {
	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		before := time.Now().UTC().Truncate(time.Second)
		task, err := NewTask("Test task", nil)
		after := time.Now().UTC().Truncate(time.Second)
		if err != nil {
			t.Fatalf("NewTask returned error: %v", err)
		}
		if task.Priority != 2 {
			t.Errorf("expected default priority 2, got %d", task.Priority)
		}
		// Also verify timestamps are set (truncated to second precision per ISO 8601)
		if task.Created.Before(before) || task.Created.After(after) {
			t.Errorf("created timestamp %v not between %v and %v", task.Created, before, after)
		}
	})

	t.Run("it sets created and updated timestamps to current UTC time", func(t *testing.T) {
		before := time.Now().UTC().Truncate(time.Second)
		task, err := NewTask("Test task", nil)
		after := time.Now().UTC().Truncate(time.Second)
		if err != nil {
			t.Fatalf("NewTask returned error: %v", err)
		}

		// Timestamps are stored at second precision per ISO 8601 format
		if task.Created.Before(before) || task.Created.After(after) {
			t.Errorf("created timestamp %v not in expected range [%v, %v]", task.Created, before, after)
		}
		if task.Updated.Before(before) || task.Updated.After(after) {
			t.Errorf("updated timestamp %v not in expected range [%v, %v]", task.Updated, before, after)
		}
		if !task.Created.Equal(task.Updated) {
			t.Errorf("created %v and updated %v should be equal on new task", task.Created, task.Updated)
		}
		if task.Created.Location() != time.UTC {
			t.Error("created timestamp should be UTC")
		}
	})

	t.Run("it has all 10 fields with correct Go types", func(t *testing.T) {
		task, err := NewTask("Test task", nil)
		if err != nil {
			t.Fatalf("NewTask returned error: %v", err)
		}

		// Verify the struct has all required fields by accessing them
		_ = task.ID          // string
		_ = task.Title       // string
		_ = task.Status      // Status
		_ = task.Priority    // int
		_ = task.Description // string
		_ = task.BlockedBy   // []string
		_ = task.Parent      // string
		_ = task.Created     // time.Time
		_ = task.Updated     // time.Time
		_ = task.Closed      // *time.Time

		// Verify defaults
		if task.Status != StatusOpen {
			t.Errorf("expected status %q, got %q", StatusOpen, task.Status)
		}
		if task.Description != "" {
			t.Errorf("expected empty description, got %q", task.Description)
		}
		if task.BlockedBy != nil {
			t.Errorf("expected nil blocked_by, got %v", task.BlockedBy)
		}
		if task.Parent != "" {
			t.Errorf("expected empty parent, got %q", task.Parent)
		}
		if task.Closed != nil {
			t.Errorf("expected nil closed, got %v", task.Closed)
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("it defines all four status constants", func(t *testing.T) {
		if StatusOpen != "open" {
			t.Errorf("StatusOpen = %q, want %q", StatusOpen, "open")
		}
		if StatusInProgress != "in_progress" {
			t.Errorf("StatusInProgress = %q, want %q", StatusInProgress, "in_progress")
		}
		if StatusDone != "done" {
			t.Errorf("StatusDone = %q, want %q", StatusDone, "done")
		}
		if StatusCancelled != "cancelled" {
			t.Errorf("StatusCancelled = %q, want %q", StatusCancelled, "cancelled")
		}
	})
}

func TestTimestampFormat(t *testing.T) {
	t.Run("it formats timestamps as ISO 8601 UTC", func(t *testing.T) {
		ts := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		formatted := FormatTimestamp(ts)
		expected := "2026-01-19T10:00:00Z"
		if formatted != expected {
			t.Errorf("FormatTimestamp = %q, want %q", formatted, expected)
		}
	})
}

func TestTaskMarshalJSON(t *testing.T) {
	t.Run("it round-trips minimal task through JSON", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		original := Task{
			ID:       "tick-a1b2c3",
			Title:    "Test task",
			Status:   StatusOpen,
			Priority: 2,
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got Task
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if got.ID != original.ID {
			t.Errorf("ID = %q, want %q", got.ID, original.ID)
		}
		if got.Status != original.Status {
			t.Errorf("Status = %q, want %q", got.Status, original.Status)
		}
		if !got.Created.Equal(original.Created) {
			t.Errorf("Created = %v, want %v", got.Created, original.Created)
		}
		if got.Closed != nil {
			t.Errorf("Closed = %v, want nil", got.Closed)
		}
	})

	t.Run("it round-trips full task with closed timestamp", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		original := Task{
			ID:          "tick-c3d4e5",
			Title:       "Full task",
			Status:      StatusDone,
			Priority:    1,
			Description: "Details here",
			BlockedBy:   []string{"tick-a1b2c3"},
			Parent:      "tick-e5f6a7",
			Created:     created,
			Updated:     updated,
			Closed:      &closed,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got Task
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if got.Status != StatusDone {
			t.Errorf("Status = %q, want %q", got.Status, StatusDone)
		}
		if !got.Updated.Equal(updated) {
			t.Errorf("Updated = %v, want %v", got.Updated, updated)
		}
		if got.Closed == nil {
			t.Fatal("Closed is nil, want non-nil")
		}
		if !got.Closed.Equal(closed) {
			t.Errorf("Closed = %v, want %v", got.Closed, closed)
		}
	})

	t.Run("it produces correct timestamp format in JSON output", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		t1 := Task{
			ID:       "tick-a1b2c3",
			Title:    "Test",
			Status:   StatusOpen,
			Priority: 2,
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(t1)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		s := string(data)
		if !strings.Contains(s, `"created":"2026-01-19T10:00:00Z"`) {
			t.Errorf("JSON should contain ISO 8601 created timestamp, got: %s", s)
		}
		if !strings.Contains(s, `"updated":"2026-01-19T10:00:00Z"`) {
			t.Errorf("JSON should contain ISO 8601 updated timestamp, got: %s", s)
		}
	})

	t.Run("it omits optional fields when empty", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		t1 := Task{
			ID:       "tick-a1b2c3",
			Title:    "Minimal",
			Status:   StatusOpen,
			Priority: 2,
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(t1)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		s := string(data)
		for _, field := range []string{"description", "blocked_by", "parent", "closed", "type"} {
			if strings.Contains(s, `"`+field+`"`) {
				t.Errorf("optional field %q should be omitted, got: %s", field, s)
			}
		}
	})

	t.Run("it marshals type field when set", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "Typed task",
			Status:   StatusOpen,
			Priority: 2,
			Type:     "bug",
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal raw error: %v", err)
		}
		typ, ok := raw["type"]
		if !ok {
			t.Fatal("expected 'type' key in JSON output, not found")
		}
		if typ != "bug" {
			t.Errorf("type = %q, want %q", typ, "bug")
		}
	})

	t.Run("it omits type field when empty", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "No type",
			Status:   StatusOpen,
			Priority: 2,
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		s := string(data)
		if strings.Contains(s, `"type"`) {
			t.Errorf("type field should be omitted when empty, got: %s", s)
		}
	})

	t.Run("it unmarshals type field from JSON", func(t *testing.T) {
		jsonStr := `{"id":"tick-a1b2c3","title":"Typed","status":"open","priority":2,"type":"feature","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		var tk Task
		if err := json.Unmarshal([]byte(jsonStr), &tk); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if tk.Type != "feature" {
			t.Errorf("Type = %q, want %q", tk.Type, "feature")
		}
	})

	t.Run("it unmarshals task without type field (backward compat)", func(t *testing.T) {
		jsonStr := `{"id":"tick-a1b2c3","title":"Legacy","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		var tk Task
		if err := json.Unmarshal([]byte(jsonStr), &tk); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if tk.Type != "" {
			t.Errorf("Type = %q, want empty string for backward compat", tk.Type)
		}
	})
}

func TestValidateType(t *testing.T) {
	t.Run("it validates allowed type values", func(t *testing.T) {
		for _, typ := range []string{"bug", "feature", "task", "chore"} {
			t.Run(typ, func(t *testing.T) {
				err := ValidateType(typ)
				if err != nil {
					t.Errorf("ValidateType(%q) returned error: %v", typ, err)
				}
			})
		}
	})

	t.Run("it rejects invalid type value", func(t *testing.T) {
		err := ValidateType("enhancement")
		if err == nil {
			t.Fatal("expected error for invalid type, got nil")
		}
		// Error should list allowed values
		for _, allowed := range []string{"bug", "feature", "task", "chore"} {
			if !strings.Contains(err.Error(), allowed) {
				t.Errorf("error should list allowed value %q, got: %q", allowed, err.Error())
			}
		}
	})

	t.Run("it allows empty type (optional field)", func(t *testing.T) {
		err := ValidateType("")
		if err != nil {
			t.Errorf("ValidateType(%q) should allow empty, got: %v", "", err)
		}
	})

	t.Run("it rejects mixed-case invalid type after normalization", func(t *testing.T) {
		// Even after normalization, "Enhancement" -> "enhancement" is still invalid
		normalized := NormalizeType("Enhancement")
		err := ValidateType(normalized)
		if err == nil {
			t.Fatal("expected error for invalid type 'enhancement' (normalized from 'Enhancement'), got nil")
		}
	})
}

func TestNormalizeType(t *testing.T) {
	t.Run("it normalizes type input to trimmed lowercase", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"  BUG  ", "bug"},
			{"Feature", "feature"},
			{" TASK ", "task"},
			{"  chore  ", "chore"},
			{"", ""},
			{"  ", ""},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
				got := NormalizeType(tt.input)
				if got != tt.want {
					t.Errorf("NormalizeType(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})
}

func TestValidateTypeNotEmpty(t *testing.T) {
	t.Run("it rejects empty string on --type flag context", func(t *testing.T) {
		err := ValidateTypeNotEmpty("")
		if err == nil {
			t.Fatal("expected error for empty type on --type flag, got nil")
		}
		if !strings.Contains(err.Error(), "--clear-type") {
			t.Errorf("error should mention --clear-type, got: %q", err.Error())
		}
	})

	t.Run("it rejects whitespace-only input on --type flag context", func(t *testing.T) {
		normalized := NormalizeType("   ")
		err := ValidateTypeNotEmpty(normalized)
		if err == nil {
			t.Fatal("expected error for whitespace-only type on --type flag, got nil")
		}
		if !strings.Contains(err.Error(), "--clear-type") {
			t.Errorf("error should mention --clear-type, got: %q", err.Error())
		}
	})
}
