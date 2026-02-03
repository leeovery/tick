package task

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestGenerateID(t *testing.T) {
	t.Run("it generates IDs matching tick-{6 hex} pattern", func(t *testing.T) {
		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
		existsFn := func(id string) bool { return false }

		id, err := GenerateID(existsFn)
		if err != nil {
			t.Fatalf("GenerateID() returned error: %v", err)
		}
		if !pattern.MatchString(id) {
			t.Errorf("GenerateID() = %q, want match for pattern %q", id, pattern.String())
		}
	})

	t.Run("it retries on collision up to 5 times", func(t *testing.T) {
		attempts := 0
		existsFn := func(id string) bool {
			attempts++
			// Collide on first 4 attempts, succeed on 5th
			return attempts < 5
		}

		id, err := GenerateID(existsFn)
		if err != nil {
			t.Fatalf("GenerateID() returned error: %v", err)
		}
		if id == "" {
			t.Error("GenerateID() returned empty ID")
		}
		if attempts != 5 {
			t.Errorf("expected 5 attempts, got %d", attempts)
		}
	})

	t.Run("it errors after 5 collision retries", func(t *testing.T) {
		existsFn := func(id string) bool { return true }

		_, err := GenerateID(existsFn)
		if err == nil {
			t.Fatal("GenerateID() expected error, got nil")
		}

		want := "Failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != want {
			t.Errorf("GenerateID() error = %q, want %q", err.Error(), want)
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

func TestValidateTitle(t *testing.T) {
	t.Run("it rejects empty title", func(t *testing.T) {
		tests := []string{"", "   ", "\t", "  \t  "}
		for _, title := range tests {
			t.Run(fmt.Sprintf("%q", title), func(t *testing.T) {
				_, err := ValidateTitle(title)
				if err == nil {
					t.Errorf("ValidateTitle(%q) expected error, got nil", title)
				}
			})
		}
	})

	t.Run("it rejects title exceeding 500 characters", func(t *testing.T) {
		longTitle := strings.Repeat("a", 501)
		_, err := ValidateTitle(longTitle)
		if err == nil {
			t.Fatal("ValidateTitle() expected error for title > 500 chars, got nil")
		}
	})

	t.Run("it accepts title at exactly 500 characters", func(t *testing.T) {
		title := strings.Repeat("a", 500)
		got, err := ValidateTitle(title)
		if err != nil {
			t.Fatalf("ValidateTitle() returned error: %v", err)
		}
		if len(got) != 500 {
			t.Errorf("ValidateTitle() returned title length %d, want 500", len(got))
		}
	})

	t.Run("it accepts multi-byte Unicode title at exactly 500 characters", func(t *testing.T) {
		// Each '漢' is 3 bytes in UTF-8 but 1 character (rune).
		// 500 runes * 3 bytes = 1500 bytes, but should be accepted as 500 characters.
		title := strings.Repeat("漢", 500)
		got, err := ValidateTitle(title)
		if err != nil {
			t.Fatalf("ValidateTitle() returned error for 500-rune multi-byte title: %v", err)
		}
		if utf8.RuneCountInString(got) != 500 {
			t.Errorf("ValidateTitle() returned title rune count %d, want 500", utf8.RuneCountInString(got))
		}
	})

	t.Run("it rejects multi-byte Unicode title exceeding 500 characters", func(t *testing.T) {
		// 501 runes of multi-byte characters should be rejected
		title := strings.Repeat("漢", 501)
		_, err := ValidateTitle(title)
		if err == nil {
			t.Fatal("ValidateTitle() expected error for 501-rune multi-byte title, got nil")
		}
	})

	t.Run("it rejects title with newlines", func(t *testing.T) {
		tests := []string{"line1\nline2", "line1\rline2", "line1\r\nline2"}
		for _, title := range tests {
			t.Run(fmt.Sprintf("%q", title), func(t *testing.T) {
				_, err := ValidateTitle(title)
				if err == nil {
					t.Errorf("ValidateTitle(%q) expected error, got nil", title)
				}
			})
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"  hello  ", "hello"},
			{"\thello\t", "hello"},
			{"  hello world  ", "hello world"},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%q", tt.input), func(t *testing.T) {
				got, err := ValidateTitle(tt.input)
				if err != nil {
					t.Fatalf("ValidateTitle(%q) returned error: %v", tt.input, err)
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
		invalidPriorities := []int{-1, 5, 10, -100, 100}
		for _, p := range invalidPriorities {
			t.Run(fmt.Sprintf("priority_%d", p), func(t *testing.T) {
				err := ValidatePriority(p)
				if err == nil {
					t.Errorf("ValidatePriority(%d) expected error, got nil", p)
				}
			})
		}
	})

	t.Run("it accepts valid priorities 0-4", func(t *testing.T) {
		for p := 0; p <= 4; p++ {
			t.Run(fmt.Sprintf("priority_%d", p), func(t *testing.T) {
				err := ValidatePriority(p)
				if err != nil {
					t.Errorf("ValidatePriority(%d) returned error: %v", p, err)
				}
			})
		}
	})
}

func TestValidateBlockedBy(t *testing.T) {
	t.Run("it rejects self-reference in blocked_by", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		blockedBy := []string{"tick-d4e5f6", "tick-a1b2c3"}

		err := ValidateBlockedBy(taskID, blockedBy)
		if err == nil {
			t.Error("ValidateBlockedBy() expected error for self-reference, got nil")
		}
	})

	t.Run("it accepts valid blocked_by without self-reference", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		blockedBy := []string{"tick-d4e5f6", "tick-g7h8i9"}

		err := ValidateBlockedBy(taskID, blockedBy)
		if err != nil {
			t.Errorf("ValidateBlockedBy() returned error: %v", err)
		}
	})
}

func TestValidateParent(t *testing.T) {
	t.Run("it rejects self-reference in parent", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		parentID := "tick-a1b2c3"

		err := ValidateParent(taskID, parentID)
		if err == nil {
			t.Error("ValidateParent() expected error for self-reference, got nil")
		}
	})

	t.Run("it accepts valid parent without self-reference", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		parentID := "tick-d4e5f6"

		err := ValidateParent(taskID, parentID)
		if err != nil {
			t.Errorf("ValidateParent() returned error: %v", err)
		}
	})

	t.Run("it accepts empty parent", func(t *testing.T) {
		taskID := "tick-a1b2c3"

		err := ValidateParent(taskID, "")
		if err != nil {
			t.Errorf("ValidateParent() returned error: %v", err)
		}
	})
}

func TestNewTask(t *testing.T) {
	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		existsFn := func(id string) bool { return false }

		task, err := NewTask("My task title", nil, existsFn)
		if err != nil {
			t.Fatalf("NewTask() returned error: %v", err)
		}
		if task.Priority != 2 {
			t.Errorf("NewTask() priority = %d, want 2", task.Priority)
		}
	})

	t.Run("it sets created and updated timestamps to current UTC time", func(t *testing.T) {
		existsFn := func(id string) bool { return false }
		before := time.Now().UTC().Truncate(time.Second)

		task, err := NewTask("My task title", nil, existsFn)
		if err != nil {
			t.Fatalf("NewTask() returned error: %v", err)
		}

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		if task.Created.Before(before) || task.Created.After(after) {
			t.Errorf("NewTask() created = %v, want between %v and %v", task.Created, before, after)
		}
		if task.Updated.Before(before) || task.Updated.After(after) {
			t.Errorf("NewTask() updated = %v, want between %v and %v", task.Updated, before, after)
		}
		if !task.Created.Equal(task.Updated) {
			t.Errorf("NewTask() created (%v) != updated (%v), want equal", task.Created, task.Updated)
		}
		if task.Created.Location() != time.UTC {
			t.Errorf("NewTask() created timezone = %v, want UTC", task.Created.Location())
		}
	})

	t.Run("it creates task with all fields properly initialized", func(t *testing.T) {
		existsFn := func(id string) bool { return false }
		priority := 1
		opts := &TaskOptions{
			Priority:    &priority,
			Description: "A description",
			BlockedBy:   []string{"tick-aaaaaa"},
			Parent:      "tick-bbbbbb",
		}

		task, err := NewTask("My task", opts, existsFn)
		if err != nil {
			t.Fatalf("NewTask() returned error: %v", err)
		}

		if task.ID == "" {
			t.Error("NewTask() ID is empty")
		}
		if task.Title != "My task" {
			t.Errorf("NewTask() title = %q, want %q", task.Title, "My task")
		}
		if task.Status != StatusOpen {
			t.Errorf("NewTask() status = %q, want %q", task.Status, StatusOpen)
		}
		if task.Priority != 1 {
			t.Errorf("NewTask() priority = %d, want 1", task.Priority)
		}
		if task.Description != "A description" {
			t.Errorf("NewTask() description = %q, want %q", task.Description, "A description")
		}
		if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "tick-aaaaaa" {
			t.Errorf("NewTask() blocked_by = %v, want [tick-aaaaaa]", task.BlockedBy)
		}
		if task.Parent != "tick-bbbbbb" {
			t.Errorf("NewTask() parent = %q, want %q", task.Parent, "tick-bbbbbb")
		}
		if task.Closed != nil {
			t.Errorf("NewTask() closed = %v, want nil", task.Closed)
		}
	})
}

func TestTaskTimestampFormat(t *testing.T) {
	t.Run("timestamps use ISO 8601 UTC format", func(t *testing.T) {
		existsFn := func(id string) bool { return false }

		task, err := NewTask("Test task", nil, existsFn)
		if err != nil {
			t.Fatalf("NewTask() returned error: %v", err)
		}

		formatted := task.Created.Format(time.RFC3339)
		if !strings.HasSuffix(formatted, "Z") {
			t.Errorf("Created timestamp format = %q, want UTC (Z suffix)", formatted)
		}
	})
}

func TestStatusConstants(t *testing.T) {
	t.Run("status enum has correct values", func(t *testing.T) {
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
