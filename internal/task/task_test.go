package task

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	t.Run("generates IDs matching tick-{6 hex} pattern", func(t *testing.T) {
		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
		exists := func(id string) bool { return false }

		id, err := GenerateID(exists)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !pattern.MatchString(id) {
			t.Errorf("ID %q does not match pattern tick-{6 hex}", id)
		}
	})

	t.Run("retries on collision up to 5 times", func(t *testing.T) {
		attempts := 0
		exists := func(id string) bool {
			attempts++
			return attempts < 5 // first 4 collide, 5th succeeds
		}

		id, err := GenerateID(exists)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty ID")
		}
		if attempts != 5 {
			t.Errorf("expected 5 attempts, got %d", attempts)
		}
	})

	t.Run("errors after 5 collision retries", func(t *testing.T) {
		exists := func(id string) bool { return true } // always collide

		_, err := GenerateID(exists)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		expected := "failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})
}

func TestNormalizeID(t *testing.T) {
	t.Run("normalizes IDs to lowercase", func(t *testing.T) {
		cases := []struct {
			input, want string
		}{
			{"TICK-A3F2B7", "tick-a3f2b7"},
			{"Tick-A3F2B7", "tick-a3f2b7"},
			{"tick-a3f2b7", "tick-a3f2b7"},
		}
		for _, tc := range cases {
			got := NormalizeID(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})
}

func TestValidateTitle(t *testing.T) {
	t.Run("rejects empty title", func(t *testing.T) {
		err := ValidateTitle("")
		if err == nil {
			t.Fatal("expected error for empty title")
		}
	})

	t.Run("rejects whitespace-only title", func(t *testing.T) {
		err := ValidateTitle("   ")
		if err == nil {
			t.Fatal("expected error for whitespace-only title")
		}
	})

	t.Run("rejects title exceeding 500 characters", func(t *testing.T) {
		long := strings.Repeat("a", 501)
		err := ValidateTitle(long)
		if err == nil {
			t.Fatal("expected error for title exceeding 500 chars")
		}
	})

	t.Run("accepts title at exactly 500 characters", func(t *testing.T) {
		exact := strings.Repeat("a", 500)
		err := ValidateTitle(exact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects title with newlines", func(t *testing.T) {
		err := ValidateTitle("line1\nline2")
		if err == nil {
			t.Fatal("expected error for title with newline")
		}
	})

	t.Run("trims whitespace from title", func(t *testing.T) {
		// ValidateTitle should accept trimmed titles;
		// TrimTitle provides the trimmed value
		trimmed := TrimTitle("  hello world  ")
		if trimmed != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", trimmed)
		}
	})
}

func TestValidatePriority(t *testing.T) {
	t.Run("accepts valid priorities 0-4", func(t *testing.T) {
		for p := 0; p <= 4; p++ {
			if err := ValidatePriority(p); err != nil {
				t.Errorf("priority %d should be valid, got error: %v", p, err)
			}
		}
	})

	t.Run("rejects priority outside 0-4", func(t *testing.T) {
		for _, p := range []int{-1, 5, 100, -100} {
			if err := ValidatePriority(p); err == nil {
				t.Errorf("priority %d should be rejected", p)
			}
		}
	})
}

func TestValidateBlockedBy(t *testing.T) {
	t.Run("rejects self-reference in blocked_by", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-a1b2c3"})
		if err == nil {
			t.Fatal("expected error for self-reference")
		}
	})

	t.Run("accepts valid blocked_by references", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"tick-d4e5f6"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateParent(t *testing.T) {
	t.Run("rejects self-reference in parent", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-a1b2c3")
		if err == nil {
			t.Fatal("expected error for self-reference")
		}
	})

	t.Run("accepts valid parent reference", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "tick-d4e5f6")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts empty parent", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNewTask(t *testing.T) {
	t.Run("sets default priority to 2 when not specified", func(t *testing.T) {
		task := NewTask("tick-abc123", "Test task", -1) // -1 signals "use default"
		if task.Priority != 2 {
			t.Errorf("expected default priority 2, got %d", task.Priority)
		}
	})

	t.Run("sets created and updated timestamps to current UTC time", func(t *testing.T) {
		before := time.Now().UTC().Truncate(time.Second)
		task := NewTask("tick-abc123", "Test task", 2)
		after := time.Now().UTC().Add(time.Second).Truncate(time.Second)

		if task.Created.Before(before) || task.Created.After(after) {
			t.Errorf("created timestamp %v not within expected range", task.Created)
		}
		if task.Updated.Before(before) || task.Updated.After(after) {
			t.Errorf("updated timestamp %v not within expected range", task.Updated)
		}
		if !task.Created.Equal(task.Updated) {
			t.Error("created and updated should be equal on new task")
		}
	})

	t.Run("has all 10 fields with correct types", func(t *testing.T) {
		task := NewTask("tick-abc123", "Test task", 1)
		// Verify fields exist and have correct zero/default values
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

		if task.ID != "tick-abc123" {
			t.Errorf("expected ID tick-abc123, got %s", task.ID)
		}
		if task.Title != "Test task" {
			t.Errorf("expected title 'Test task', got %s", task.Title)
		}
		if task.Status != StatusOpen {
			t.Errorf("expected status open, got %s", task.Status)
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

func TestStatusConstants(t *testing.T) {
	t.Run("defines all four status values", func(t *testing.T) {
		statuses := []Status{StatusOpen, StatusInProgress, StatusDone, StatusCancelled}
		expected := []string{"open", "in_progress", "done", "cancelled"}
		for i, s := range statuses {
			if string(s) != expected[i] {
				t.Errorf("expected %q, got %q", expected[i], s)
			}
		}
	})
}

func TestTimestampFormat(t *testing.T) {
	t.Run("timestamps use ISO 8601 UTC format", func(t *testing.T) {
		task := NewTask("tick-abc123", "Test task", 2)
		formatted := FormatTimestamp(task.Created)
		// Should match YYYY-MM-DDTHH:MM:SSZ
		pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
		if !pattern.MatchString(formatted) {
			t.Errorf("timestamp %q does not match ISO 8601 UTC format", formatted)
		}
	})
}

func TestGenerateIDUniqueness(t *testing.T) {
	t.Run("generates different IDs on successive calls", func(t *testing.T) {
		exists := func(id string) bool { return false }
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id, err := GenerateID(exists)
			if err != nil {
				t.Fatalf("unexpected error on iteration %d: %v", i, err)
			}
			if ids[id] {
				t.Fatalf("duplicate ID %q on iteration %d", id, i)
			}
			ids[id] = true
		}
	})
}

// Ensure the error message matches spec exactly
func TestCollisionErrorMessage(t *testing.T) {
	t.Run("collision error message matches spec", func(t *testing.T) {
		exists := func(id string) bool { return true }
		_, err := GenerateID(exists)
		if err == nil {
			t.Fatal("expected error")
		}
		want := "failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})
}

func TestTrimTitle(t *testing.T) {
	t.Run("trims leading and trailing whitespace", func(t *testing.T) {
		cases := []struct {
			input, want string
		}{
			{"  hello  ", "hello"},
			{"\thello\t", "hello"},
			{"hello", "hello"},
			{"  hello world  ", "hello world"},
		}
		for _, tc := range cases {
			got := TrimTitle(tc.input)
			if got != tc.want {
				t.Errorf("TrimTitle(%q) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})
}

func TestValidateTitleRejectsCarriageReturn(t *testing.T) {
	t.Run("rejects title with carriage return", func(t *testing.T) {
		err := ValidateTitle("line1\rline2")
		if err == nil {
			t.Fatal("expected error for title with carriage return")
		}
	})
}

// Verify default priority sentinel works
func TestNewTaskDefaultPriority(t *testing.T) {
	t.Run("uses explicit priority when provided", func(t *testing.T) {
		task := NewTask("tick-abc123", "Test", 0)
		if task.Priority != 0 {
			t.Errorf("expected priority 0, got %d", task.Priority)
		}

		task = NewTask("tick-abc123", "Test", 4)
		if task.Priority != 4 {
			t.Errorf("expected priority 4, got %d", task.Priority)
		}
	})
}

func TestValidateBlockedByCaseInsensitive(t *testing.T) {
	t.Run("self-reference detection is case-insensitive", func(t *testing.T) {
		err := ValidateBlockedBy("tick-a1b2c3", []string{"TICK-A1B2C3"})
		if err == nil {
			t.Fatal("expected error for case-insensitive self-reference")
		}
	})
}

func TestValidateParentCaseInsensitive(t *testing.T) {
	t.Run("self-reference detection is case-insensitive", func(t *testing.T) {
		err := ValidateParent("tick-a1b2c3", "TICK-A1B2C3")
		if err == nil {
			t.Fatal("expected error for case-insensitive self-reference")
		}
	})
}

// Quick sanity: FormatTimestamp produces the exact expected string
func TestFormatTimestampExact(t *testing.T) {
	ts := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	got := FormatTimestamp(ts)
	want := "2026-01-19T10:00:00Z"
	if got != want {
		t.Errorf("FormatTimestamp = %q, want %q", got, want)
	}
}

func init() {
	// Suppress unused import warning — fmt is used in tests
	_ = fmt.Sprintf
}
