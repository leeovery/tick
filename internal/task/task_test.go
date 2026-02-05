package task

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	t.Run("it generates IDs matching tick-{6 hex} pattern", func(t *testing.T) {
		pattern := regexp.MustCompile(`^tick-[a-f0-9]{6}$`)

		// Generate several IDs to ensure pattern consistency
		for i := 0; i < 10; i++ {
			id, err := GenerateID(func(id string) (bool, error) {
				return false, nil // No collisions
			})
			if err != nil {
				t.Fatalf("GenerateID failed: %v", err)
			}
			if !pattern.MatchString(id) {
				t.Errorf("ID %q does not match pattern tick-{6 hex}", id)
			}
		}
	})

	t.Run("it retries on collision up to 5 times", func(t *testing.T) {
		attempts := 0
		id, err := GenerateID(func(id string) (bool, error) {
			attempts++
			// Return collision for first 4 attempts, then success
			return attempts < 5, nil
		})
		if err != nil {
			t.Fatalf("GenerateID should succeed after retries: %v", err)
		}
		if attempts != 5 {
			t.Errorf("expected 5 attempts, got %d", attempts)
		}
		if id == "" {
			t.Error("expected valid ID")
		}
	})

	t.Run("it errors after 5 collision retries", func(t *testing.T) {
		attempts := 0
		_, err := GenerateID(func(id string) (bool, error) {
			attempts++
			return true, nil // Always collide
		})
		if err == nil {
			t.Fatal("expected error after 5 retries")
		}
		if attempts != 5 {
			t.Errorf("expected exactly 5 attempts, got %d", attempts)
		}
		expected := "failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != expected {
			t.Errorf("error message = %q, want %q", err.Error(), expected)
		}
	})
}

func TestNormalizeID(t *testing.T) {
	t.Run("it normalizes IDs to lowercase", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"tick-A1B2C3", "tick-a1b2c3"},
			{"TICK-A1B2C3", "tick-a1b2c3"},
			{"Tick-AbCdEf", "tick-abcdef"},
			{"tick-a1b2c3", "tick-a1b2c3"},
		}

		for _, tt := range tests {
			result := NormalizeID(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})
}

func TestTitleValidation(t *testing.T) {
	t.Run("it rejects empty title", func(t *testing.T) {
		err := ValidateTitle("")
		if err == nil {
			t.Error("expected error for empty title")
		}
		if err != nil && err.Error() != "title is required" {
			t.Errorf("error message = %q, want %q", err.Error(), "title is required")
		}
	})

	t.Run("it rejects title exceeding 500 characters", func(t *testing.T) {
		longTitle := strings.Repeat("a", 501)
		err := ValidateTitle(longTitle)
		if err == nil {
			t.Error("expected error for title exceeding 500 characters")
		}
		if err != nil && err.Error() != "title exceeds 500 characters" {
			t.Errorf("error message = %q, want %q", err.Error(), "title exceeds 500 characters")
		}
	})

	t.Run("it rejects title with newlines", func(t *testing.T) {
		err := ValidateTitle("line1\nline2")
		if err == nil {
			t.Error("expected error for title with newline")
		}
		if err != nil && err.Error() != "title cannot contain newlines" {
			t.Errorf("error message = %q, want %q", err.Error(), "title cannot contain newlines")
		}

		// Also test carriage return
		err = ValidateTitle("line1\rline2")
		if err == nil {
			t.Error("expected error for title with carriage return")
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		result := TrimTitle("  hello world  ")
		if result != "hello world" {
			t.Errorf("TrimTitle = %q, want %q", result, "hello world")
		}

		result = TrimTitle("\thello\t")
		if result != "hello" {
			t.Errorf("TrimTitle = %q, want %q", result, "hello")
		}
	})

	t.Run("it accepts valid title at boundary", func(t *testing.T) {
		// 500 characters should be accepted
		title500 := strings.Repeat("a", 500)
		err := ValidateTitle(title500)
		if err != nil {
			t.Errorf("title of 500 chars should be valid: %v", err)
		}
	})
}

func TestPriorityValidation(t *testing.T) {
	t.Run("it rejects priority outside 0-4", func(t *testing.T) {
		tests := []struct {
			priority int
			valid    bool
		}{
			{-1, false},
			{0, true},
			{1, true},
			{2, true},
			{3, true},
			{4, true},
			{5, false},
			{100, false},
		}

		for _, tt := range tests {
			err := ValidatePriority(tt.priority)
			if tt.valid && err != nil {
				t.Errorf("priority %d should be valid: %v", tt.priority, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("priority %d should be invalid", tt.priority)
			}
			if !tt.valid && err != nil && err.Error() != "priority must be between 0 and 4" {
				t.Errorf("error message = %q, want %q", err.Error(), "priority must be between 0 and 4")
			}
		}
	})
}

func TestSelfReferenceValidation(t *testing.T) {
	t.Run("it rejects self-reference in blocked_by", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		blockedBy := []string{"tick-x1y2z3", "tick-a1b2c3", "tick-d4e5f6"}

		err := ValidateBlockedBy(taskID, blockedBy)
		if err == nil {
			t.Error("expected error for self-reference in blocked_by")
		}
		if err != nil && err.Error() != "task cannot block itself" {
			t.Errorf("error message = %q, want %q", err.Error(), "task cannot block itself")
		}
	})

	t.Run("it accepts blocked_by without self-reference", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		blockedBy := []string{"tick-x1y2z3", "tick-d4e5f6"}

		err := ValidateBlockedBy(taskID, blockedBy)
		if err != nil {
			t.Errorf("blocked_by without self-reference should be valid: %v", err)
		}
	})

	t.Run("it accepts empty blocked_by", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		err := ValidateBlockedBy(taskID, nil)
		if err != nil {
			t.Errorf("nil blocked_by should be valid: %v", err)
		}

		err = ValidateBlockedBy(taskID, []string{})
		if err != nil {
			t.Errorf("empty blocked_by should be valid: %v", err)
		}
	})

	t.Run("it rejects self-reference in parent", func(t *testing.T) {
		taskID := "tick-a1b2c3"

		err := ValidateParent(taskID, "tick-a1b2c3")
		if err == nil {
			t.Error("expected error for self-reference in parent")
		}
		if err != nil && err.Error() != "task cannot be its own parent" {
			t.Errorf("error message = %q, want %q", err.Error(), "task cannot be its own parent")
		}
	})

	t.Run("it accepts parent without self-reference", func(t *testing.T) {
		taskID := "tick-a1b2c3"

		err := ValidateParent(taskID, "tick-x1y2z3")
		if err != nil {
			t.Errorf("parent without self-reference should be valid: %v", err)
		}
	})

	t.Run("it accepts empty parent", func(t *testing.T) {
		taskID := "tick-a1b2c3"

		err := ValidateParent(taskID, "")
		if err != nil {
			t.Errorf("empty parent should be valid: %v", err)
		}
	})

	t.Run("blocked_by self-reference check is case-insensitive", func(t *testing.T) {
		taskID := "tick-a1b2c3"
		blockedBy := []string{"TICK-A1B2C3"} // Uppercase self-reference

		err := ValidateBlockedBy(taskID, blockedBy)
		if err == nil {
			t.Error("expected error for case-insensitive self-reference in blocked_by")
		}
	})

	t.Run("parent self-reference check is case-insensitive", func(t *testing.T) {
		taskID := "tick-a1b2c3"

		err := ValidateParent(taskID, "TICK-A1B2C3") // Uppercase self-reference
		if err == nil {
			t.Error("expected error for case-insensitive self-reference in parent")
		}
	})
}

func TestTaskDefaults(t *testing.T) {
	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		priority := DefaultPriority()
		if priority != 2 {
			t.Errorf("DefaultPriority() = %d, want 2", priority)
		}
	})

	t.Run("it sets created and updated timestamps to current UTC time", func(t *testing.T) {
		before := time.Now().UTC()
		created, updated := DefaultTimestamps()
		after := time.Now().UTC()

		// Parse timestamps
		createdTime, err := time.Parse(time.RFC3339, created)
		if err != nil {
			t.Fatalf("failed to parse created timestamp: %v", err)
		}
		updatedTime, err := time.Parse(time.RFC3339, updated)
		if err != nil {
			t.Fatalf("failed to parse updated timestamp: %v", err)
		}

		// Should be in UTC
		if createdTime.Location() != time.UTC {
			t.Error("created timestamp should be in UTC")
		}
		if updatedTime.Location() != time.UTC {
			t.Error("updated timestamp should be in UTC")
		}

		// Should be between before and after
		if createdTime.Before(before.Truncate(time.Second)) || createdTime.After(after.Add(time.Second)) {
			t.Errorf("created timestamp %v not in expected range", createdTime)
		}
		if updatedTime.Before(before.Truncate(time.Second)) || updatedTime.After(after.Add(time.Second)) {
			t.Errorf("updated timestamp %v not in expected range", updatedTime)
		}

		// Created and updated should be equal initially
		if created != updated {
			t.Errorf("created (%s) and updated (%s) should be equal initially", created, updated)
		}
	})
}

func TestStatusEnum(t *testing.T) {
	t.Run("status constants are defined", func(t *testing.T) {
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

func TestTaskStruct(t *testing.T) {
	t.Run("Task struct has all 10 fields with correct types", func(t *testing.T) {
		task := Task{
			ID:          "tick-a1b2c3",
			Title:       "Test task",
			Status:      StatusOpen,
			Priority:    2,
			Description: "Description here",
			BlockedBy:   []string{"tick-x1y2z3"},
			Parent:      "tick-parent1",
			Created:     "2026-01-19T10:00:00Z",
			Updated:     "2026-01-19T10:00:00Z",
			Closed:      "2026-01-19T16:00:00Z",
		}

		// Verify all fields can be set and retrieved
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
		if task.Description != "Description here" {
			t.Errorf("Description = %q, want %q", task.Description, "Description here")
		}
		if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "tick-x1y2z3" {
			t.Errorf("BlockedBy = %v, want [tick-x1y2z3]", task.BlockedBy)
		}
		if task.Parent != "tick-parent1" {
			t.Errorf("Parent = %q, want %q", task.Parent, "tick-parent1")
		}
		if task.Created != "2026-01-19T10:00:00Z" {
			t.Errorf("Created = %q, want %q", task.Created, "2026-01-19T10:00:00Z")
		}
		if task.Updated != "2026-01-19T10:00:00Z" {
			t.Errorf("Updated = %q, want %q", task.Updated, "2026-01-19T10:00:00Z")
		}
		if task.Closed != "2026-01-19T16:00:00Z" {
			t.Errorf("Closed = %q, want %q", task.Closed, "2026-01-19T16:00:00Z")
		}
	})

	t.Run("optional fields can be zero values", func(t *testing.T) {
		task := Task{
			ID:       "tick-a1b2c3",
			Title:    "Required fields only",
			Status:   StatusOpen,
			Priority: 2,
			Created:  "2026-01-19T10:00:00Z",
			Updated:  "2026-01-19T10:00:00Z",
			// Description, BlockedBy, Parent, Closed are optional
		}

		if task.Description != "" {
			t.Errorf("Description should be empty string, got %q", task.Description)
		}
		if task.BlockedBy != nil {
			t.Errorf("BlockedBy should be nil, got %v", task.BlockedBy)
		}
		if task.Parent != "" {
			t.Errorf("Parent should be empty string, got %q", task.Parent)
		}
		if task.Closed != "" {
			t.Errorf("Closed should be empty string, got %q", task.Closed)
		}
	})
}
