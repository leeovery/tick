package task

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	t.Run("it generates IDs matching tick-{6 hex} pattern", func(t *testing.T) {
		id, err := GenerateID(func(id string) bool { return false })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
		if !pattern.MatchString(id) {
			t.Errorf("ID %q does not match pattern tick-{6 hex}", id)
		}
	})

	t.Run("it retries on collision up to 5 times", func(t *testing.T) {
		var calls atomic.Int32
		// Collide on first 4 calls, succeed on 5th
		id, err := GenerateID(func(id string) bool {
			n := calls.Add(1)
			return n < 5
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty ID")
		}
		if calls.Load() != 5 {
			t.Errorf("expected 5 calls to exists, got %d", calls.Load())
		}
	})

	t.Run("it errors after 5 collision retries", func(t *testing.T) {
		_, err := GenerateID(func(id string) bool { return true })
		if err == nil {
			t.Fatal("expected error after 5 retries, got nil")
		}
		expected := "Failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != expected {
			t.Errorf("unexpected error message:\ngot:  %q\nwant: %q", err.Error(), expected)
		}
	})
}

func TestValidateTitle(t *testing.T) {
	t.Run("it rejects empty title", func(t *testing.T) {
		tests := []string{"", "   ", "\t"}
		for _, title := range tests {
			_, err := ValidateTitle(title)
			if err == nil {
				t.Errorf("expected error for title %q, got nil", title)
			}
		}
	})

	t.Run("it rejects title exceeding 500 characters", func(t *testing.T) {
		longTitle := strings.Repeat("a", 501)
		_, err := ValidateTitle(longTitle)
		if err == nil {
			t.Fatal("expected error for title >500 chars, got nil")
		}
	})

	t.Run("it rejects title with newlines", func(t *testing.T) {
		tests := []string{"line1\nline2", "line1\rline2", "line1\r\nline2"}
		for _, title := range tests {
			_, err := ValidateTitle(title)
			if err == nil {
				t.Errorf("expected error for title with newline %q, got nil", title)
			}
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		got, err := ValidateTitle("  hello world  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", got)
		}
	})

	t.Run("it accepts valid title at 500 chars", func(t *testing.T) {
		title := strings.Repeat("a", 500)
		got, err := ValidateTitle(title)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != title {
			t.Error("expected title to be returned unchanged")
		}
	})

	t.Run("it counts characters not bytes for max length", func(t *testing.T) {
		// 200 CJK characters = 200 runes but 600 bytes (3 bytes each in UTF-8).
		// This should be accepted (200 < 500) but would be rejected if using byte count.
		title := strings.Repeat("\u4e16", 200) // U+4E16 = "世" (3 bytes in UTF-8)
		got, err := ValidateTitle(title)
		if err != nil {
			t.Fatalf("expected multi-byte title of 200 chars (600 bytes) to be accepted, got error: %v", err)
		}
		if got != title {
			t.Error("expected title to be returned unchanged")
		}
	})
}

func TestValidatePriority(t *testing.T) {
	t.Run("it rejects priority outside 0-4", func(t *testing.T) {
		invalid := []int{-1, 5, 10, -100, 999}
		for _, p := range invalid {
			if err := ValidatePriority(p); err == nil {
				t.Errorf("expected error for priority %d, got nil", p)
			}
		}
	})

	t.Run("it accepts valid priorities 0 through 4", func(t *testing.T) {
		for p := 0; p <= 4; p++ {
			if err := ValidatePriority(p); err != nil {
				t.Errorf("unexpected error for priority %d: %v", p, err)
			}
		}
	})
}

func TestValidateBlockedBy(t *testing.T) {
	t.Run("it rejects self-reference in blocked_by", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-d4e5f6", "tick-a1b2c3"})
		if err == nil {
			t.Fatal("expected error for self-reference in blocked_by")
		}
	})

	t.Run("it accepts blocked_by without self-reference", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-d4e5f6", "tick-g7h8i9"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("it accepts empty blocked_by", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("it detects self-reference case-insensitively", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"TICK-A1B2C3"})
		if err == nil {
			t.Fatal("expected error for case-insensitive self-reference in blocked_by")
		}
	})
}

func TestValidateParent(t *testing.T) {
	t.Run("it rejects self-reference in parent", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-a1b2c3")
		if err == nil {
			t.Fatal("expected error for self-reference in parent")
		}
	})

	t.Run("it accepts different parent ID", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-d4e5f6")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("it accepts empty parent", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("it detects self-reference case-insensitively", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "TICK-A1B2C3")
		if err == nil {
			t.Fatal("expected error for case-insensitive self-reference in parent")
		}
	})
}

func TestNormalizeID(t *testing.T) {
	t.Run("it normalizes IDs to lowercase", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"TICK-A3F2B7", "tick-a3f2b7"},
			{"Tick-A3f2B7", "tick-a3f2b7"},
			{"tick-a3f2b7", "tick-a3f2b7"},
			{"TICK-ABCDEF", "tick-abcdef"},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				got := NormalizeID(tt.input)
				if got != tt.want {
					t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})
}

func TestTaskStruct(t *testing.T) {
	t.Run("it has all 10 fields with correct types", func(t *testing.T) {
		now := time.Now().UTC()
		task := Task{
			ID:          "tick-a1b2c3",
			Title:       "Test task",
			Status:      StatusOpen,
			Priority:    2,
			Description: "A description",
			BlockedBy:   []string{"tick-d4e5f6"},
			Parent:      "tick-g7h8i9",
			Created:     now,
			Updated:     now,
			Closed:      &now,
		}

		if task.ID != "tick-a1b2c3" {
			t.Errorf("ID = %q, want %q", task.ID, "tick-a1b2c3")
		}
		if task.Title != "Test task" {
			t.Errorf("Title = %q, want %q", task.Title, "Test task")
		}
		if task.Status != StatusOpen {
			t.Errorf("Status = %q, want %q", task.Status, StatusOpen)
		}
		if task.Priority != 2 {
			t.Errorf("Priority = %d, want %d", task.Priority, 2)
		}
		if task.Description != "A description" {
			t.Errorf("Description = %q, want %q", task.Description, "A description")
		}
		if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "tick-d4e5f6" {
			t.Errorf("BlockedBy = %v, want [tick-d4e5f6]", task.BlockedBy)
		}
		if task.Parent != "tick-g7h8i9" {
			t.Errorf("Parent = %q, want %q", task.Parent, "tick-g7h8i9")
		}
		if !task.Created.Equal(now) {
			t.Errorf("Created = %v, want %v", task.Created, now)
		}
		if !task.Updated.Equal(now) {
			t.Errorf("Updated = %v, want %v", task.Updated, now)
		}
		if task.Closed == nil || !task.Closed.Equal(now) {
			t.Errorf("Closed = %v, want %v", task.Closed, now)
		}
	})

	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		task := NewTask("tick-aabbcc", "Test task")
		if task.Priority != DefaultPriority {
			t.Errorf("default priority = %d, want %d", task.Priority, DefaultPriority)
		}
	})

	t.Run("it sets created and updated timestamps to current UTC time", func(t *testing.T) {
		before := time.Now().UTC().Truncate(time.Second)
		task := NewTask("tick-aabbcc", "Test task")
		after := time.Now().UTC().Add(time.Second).Truncate(time.Second)

		if task.Created.Before(before) || task.Created.After(after) {
			t.Errorf("Created %v not between %v and %v", task.Created, before, after)
		}
		if !task.Created.Equal(task.Updated) {
			t.Errorf("Updated %v should equal Created %v", task.Updated, task.Created)
		}
		if task.Created.Location() != time.UTC {
			t.Errorf("Created not in UTC: %v", task.Created.Location())
		}
	})
}

func TestStatusConstants(t *testing.T) {
	t.Run("it defines correct status values", func(t *testing.T) {
		tests := []struct {
			status Status
			want   string
		}{
			{StatusOpen, "open"},
			{StatusInProgress, "in_progress"},
			{StatusDone, "done"},
			{StatusCancelled, "cancelled"},
		}
		for _, tt := range tests {
			t.Run(tt.want, func(t *testing.T) {
				if string(tt.status) != tt.want {
					t.Errorf("status = %q, want %q", tt.status, tt.want)
				}
			})
		}
	})
}

func TestTaskTimestampFormat(t *testing.T) {
	t.Run("it formats timestamps as ISO 8601 UTC", func(t *testing.T) {
		ts := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		formatted := FormatTimestamp(ts)
		expected := "2026-01-19T10:00:00Z"
		if formatted != expected {
			t.Errorf("FormatTimestamp = %q, want %q", formatted, expected)
		}
	})
}

func TestTaskJSONSerialization(t *testing.T) {
	t.Run("it omits optional fields when empty", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		task := Task{
			ID:       "tick-a1b2c3",
			Title:    "Test task",
			Status:   StatusOpen,
			Priority: 2,
			Created:  now,
			Updated:  now,
		}

		data, err := json.Marshal(task)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		jsonStr := string(data)
		// Optional fields should not appear
		if strings.Contains(jsonStr, "description") {
			t.Error("description should be omitted when empty")
		}
		if strings.Contains(jsonStr, "blocked_by") {
			t.Error("blocked_by should be omitted when empty")
		}
		if strings.Contains(jsonStr, "parent") {
			t.Error("parent should be omitted when empty")
		}
		if strings.Contains(jsonStr, "closed") {
			t.Error("closed should be omitted when nil")
		}
		// Required fields must be present
		if !strings.Contains(jsonStr, "id") {
			t.Error("id should be present")
		}
		if !strings.Contains(jsonStr, "updated") {
			t.Error("updated should always be present")
		}
	})

	t.Run("it includes optional fields when set", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		task := Task{
			ID:          "tick-a1b2c3",
			Title:       "Test task",
			Status:      StatusDone,
			Priority:    1,
			Description: "Details here",
			BlockedBy:   []string{"tick-d4e5f6"},
			Parent:      "tick-g7h8i9",
			Created:     now,
			Updated:     now,
			Closed:      &closed,
		}

		data, err := json.Marshal(task)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		jsonStr := string(data)
		if !strings.Contains(jsonStr, `"description":"Details here"`) {
			t.Errorf("expected description in JSON, got: %s", jsonStr)
		}
		if !strings.Contains(jsonStr, `"blocked_by":["tick-d4e5f6"]`) {
			t.Errorf("expected blocked_by in JSON, got: %s", jsonStr)
		}
		if !strings.Contains(jsonStr, `"parent":"tick-g7h8i9"`) {
			t.Errorf("expected parent in JSON, got: %s", jsonStr)
		}
		if !strings.Contains(jsonStr, `"closed":"2026-01-19T16:00:00Z"`) {
			t.Errorf("expected closed in JSON, got: %s", jsonStr)
		}
	})

	t.Run("it formats timestamps as ISO 8601 UTC in JSON", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		task := Task{
			ID:       "tick-a1b2c3",
			Title:    "Test task",
			Status:   StatusOpen,
			Priority: 2,
			Created:  now,
			Updated:  now,
		}

		data, err := json.Marshal(task)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		jsonStr := string(data)
		if !strings.Contains(jsonStr, `"created":"2026-01-19T10:00:00Z"`) {
			t.Errorf("expected ISO 8601 UTC timestamp in JSON, got: %s", jsonStr)
		}
	})
}
