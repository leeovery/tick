package task

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	t.Run("it generates IDs matching tick-{6 hex} pattern", func(t *testing.T) {
		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)

		// Generate several IDs to confirm pattern consistency
		for i := 0; i < 20; i++ {
			id, err := GenerateID(func(id string) bool { return false })
			if err != nil {
				t.Fatalf("GenerateID returned error: %v", err)
			}
			if !pattern.MatchString(id) {
				t.Errorf("ID %q does not match pattern tick-{6 hex}", id)
			}
		}
	})

	t.Run("it retries on collision up to 5 times", func(t *testing.T) {
		attempts := 0
		// Collide on first 4 attempts, succeed on 5th
		exists := func(id string) bool {
			attempts++
			return attempts < 5
		}

		id, err := GenerateID(exists)
		if err != nil {
			t.Fatalf("expected success after retries, got error: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty ID")
		}
		if attempts != 5 {
			t.Errorf("expected 5 attempts, got %d", attempts)
		}
	})

	t.Run("it errors after 5 collision retries", func(t *testing.T) {
		// Always collide
		exists := func(id string) bool { return true }

		_, err := GenerateID(exists)
		if err == nil {
			t.Fatal("expected error after 5 retries, got nil")
		}

		expectedMsg := "Failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestNormalizeID(t *testing.T) {
	t.Run("it normalizes IDs to lowercase", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{"uppercase", "TICK-A3F2B7", "tick-a3f2b7"},
			{"mixed case", "Tick-A3f2B7", "tick-a3f2b7"},
			{"already lowercase", "tick-a3f2b7", "tick-a3f2b7"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := NormalizeID(tt.input)
				if got != tt.want {
					t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})
}

func TestValidateTitle(t *testing.T) {
	t.Run("it rejects empty title", func(t *testing.T) {
		tests := []struct {
			name  string
			title string
		}{
			{"empty string", ""},
			{"only spaces", "   "},
			{"only tabs", "\t\t"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ValidateTitle(tt.title)
				if err == nil {
					t.Error("expected error for empty title, got nil")
				}
			})
		}
	})

	t.Run("it rejects title exceeding 500 characters", func(t *testing.T) {
		longTitle := strings.Repeat("a", 501)
		_, err := ValidateTitle(longTitle)
		if err == nil {
			t.Error("expected error for title exceeding 500 chars, got nil")
		}
	})

	t.Run("it counts characters not bytes for title length", func(t *testing.T) {
		// 500 CJK characters = 1500 bytes in UTF-8, but only 500 characters
		cjkTitle := strings.Repeat("\u4e16", 500) // 500 x "世"
		got, err := ValidateTitle(cjkTitle)
		if err != nil {
			t.Fatalf("expected no error for 500 multi-byte characters, got %v", err)
		}
		if got != cjkTitle {
			t.Errorf("expected title to be preserved, got different value")
		}

		// 501 CJK characters should be rejected
		cjkTooLong := strings.Repeat("\u4e16", 501)
		_, err = ValidateTitle(cjkTooLong)
		if err == nil {
			t.Error("expected error for 501 multi-byte characters, got nil")
		}
	})

	t.Run("it rejects title with newlines", func(t *testing.T) {
		tests := []struct {
			name  string
			title string
		}{
			{"line feed", "hello\nworld"},
			{"carriage return", "hello\rworld"},
			{"crlf", "hello\r\nworld"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ValidateTitle(tt.title)
				if err == nil {
					t.Errorf("expected error for title with newlines %q, got nil", tt.title)
				}
			})
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{"leading spaces", "  hello", "hello"},
			{"trailing spaces", "hello  ", "hello"},
			{"both sides", "  hello  ", "hello"},
			{"tabs", "\thello\t", "hello"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := ValidateTitle(tt.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("ValidateTitle(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})
}

func TestValidatePriority(t *testing.T) {
	t.Run("it rejects priority outside 0-4", func(t *testing.T) {
		tests := []struct {
			name     string
			priority int
		}{
			{"negative", -1},
			{"too high", 5},
			{"way too high", 100},
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
			err := ValidatePriority(p)
			if err != nil {
				t.Errorf("expected no error for priority %d, got %v", p, err)
			}
		}
	})
}

func TestValidateBlockedBy(t *testing.T) {
	t.Run("it rejects self-reference in blocked_by", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-x9y8z7", "tick-a1b2c3"})
		if err == nil {
			t.Error("expected error for self-reference in blocked_by, got nil")
		}
	})

	t.Run("it accepts valid blocked_by references", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-x9y8z7", "tick-d4e5f6"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestValidateParent(t *testing.T) {
	t.Run("it rejects self-reference in parent", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-a1b2c3")
		if err == nil {
			t.Error("expected error for self-reference in parent, got nil")
		}
	})

	t.Run("it accepts valid parent reference", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-x9y8z7")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestNewTask(t *testing.T) {
	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		noCollision := func(id string) bool { return false }

		task, err := NewTask("My test task", nil, noCollision)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Priority != 2 {
			t.Errorf("expected default priority 2, got %d", task.Priority)
		}
	})

	t.Run("it sets created and updated timestamps to current UTC time", func(t *testing.T) {
		noCollision := func(id string) bool { return false }

		before := time.Now().UTC().Truncate(time.Second)
		task, err := NewTask("My test task", nil, noCollision)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		if task.Created.Before(before) || task.Created.After(after) {
			t.Errorf("created timestamp %v not within expected range [%v, %v]", task.Created, before, after)
		}
		if task.Updated.Before(before) || task.Updated.After(after) {
			t.Errorf("updated timestamp %v not within expected range [%v, %v]", task.Updated, before, after)
		}
		if !task.Created.Equal(task.Updated) {
			t.Errorf("expected created and updated to be equal, got created=%v, updated=%v", task.Created, task.Updated)
		}
		if task.Created.Location() != time.UTC {
			t.Errorf("expected UTC timezone, got %v", task.Created.Location())
		}
	})

	t.Run("it has all 10 fields with correct types", func(t *testing.T) {
		noCollision := func(id string) bool { return false }
		opts := &TaskOptions{
			Priority:    intPtr(1),
			Description: "A description",
			BlockedBy:   []string{"tick-aaaaaa"},
			Parent:      "tick-bbbbbb",
		}

		task, err := NewTask("Full task", opts, noCollision)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify all fields are accessible and have correct values
		if task.ID == "" {
			t.Error("expected non-empty ID")
		}
		if task.Title != "Full task" {
			t.Errorf("expected title %q, got %q", "Full task", task.Title)
		}
		if task.Status != StatusOpen {
			t.Errorf("expected status %q, got %q", StatusOpen, task.Status)
		}
		if task.Priority != 1 {
			t.Errorf("expected priority 1, got %d", task.Priority)
		}
		if task.Description != "A description" {
			t.Errorf("expected description %q, got %q", "A description", task.Description)
		}
		if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "tick-aaaaaa" {
			t.Errorf("expected blocked_by [tick-aaaaaa], got %v", task.BlockedBy)
		}
		if task.Parent != "tick-bbbbbb" {
			t.Errorf("expected parent %q, got %q", "tick-bbbbbb", task.Parent)
		}
		if task.Created.IsZero() {
			t.Error("expected non-zero created timestamp")
		}
		if task.Updated.IsZero() {
			t.Error("expected non-zero updated timestamp")
		}
		if task.Closed != nil {
			t.Error("expected nil closed timestamp for new task")
		}
	})
}

func intPtr(i int) *int {
	return &i
}
