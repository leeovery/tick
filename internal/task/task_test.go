package task

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var idPattern = regexp.MustCompile(`^tick-[0-9a-f]{6}$`)

func TestGenerateID(t *testing.T) {
	t.Run("generates IDs matching tick-{6 hex} pattern", func(t *testing.T) {
		noCollisions := func(string) bool { return false }

		id, err := GenerateID(noCollisions)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !idPattern.MatchString(id) {
			t.Errorf("ID %q does not match pattern tick-{6 hex}", id)
		}
	})

	t.Run("retries on collision up to 5 times", func(t *testing.T) {
		attempts := 0
		collidesUntilFifth := func(string) bool {
			attempts++
			return attempts < 5
		}

		id, err := GenerateID(collidesUntilFifth)
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
		alwaysCollides := func(string) bool { return true }

		_, err := GenerateID(alwaysCollides)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		want := "failed to generate unique ID after 5 attempts - task list may be too large"
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("generates unique IDs across 100 calls", func(t *testing.T) {
		noCollisions := func(string) bool { return false }
		seen := make(map[string]bool)

		for i := 0; i < 100; i++ {
			id, err := GenerateID(noCollisions)
			if err != nil {
				t.Fatalf("iteration %d: %v", i, err)
			}
			if seen[id] {
				t.Fatalf("duplicate ID %q at iteration %d", id, i)
			}
			seen[id] = true
		}
	})
}

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"all uppercase", "TICK-A3F2B7", "tick-a3f2b7"},
		{"mixed case", "Tick-A3F2B7", "tick-a3f2b7"},
		{"already lowercase", "tick-a3f2b7", "tick-a3f2b7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeID(tt.input); got != tt.want {
				t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateTitle(t *testing.T) {
	t.Run("valid titles", func(t *testing.T) {
		tests := []struct {
			name  string
			title string
		}{
			{"simple title", "Setup authentication"},
			{"exactly 500 chars", strings.Repeat("a", 500)},
			{"with internal spaces", "hello world"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := ValidateTitle(tt.title); err != nil {
					t.Errorf("expected valid, got error: %v", err)
				}
			})
		}
	})

	t.Run("invalid titles", func(t *testing.T) {
		tests := []struct {
			name  string
			title string
		}{
			{"empty string", ""},
			{"whitespace only", "   "},
			{"exceeds 500 chars", strings.Repeat("a", 501)},
			{"contains newline", "line1\nline2"},
			{"contains carriage return", "line1\rline2"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := ValidateTitle(tt.title); err == nil {
					t.Error("expected error, got nil")
				}
			})
		}
	})
}

func TestTrimTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"leading/trailing spaces", "  hello  ", "hello"},
		{"tabs", "\thello\t", "hello"},
		{"no whitespace", "hello", "hello"},
		{"internal spaces preserved", "  hello world  ", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimTitle(tt.input); got != tt.want {
				t.Errorf("TrimTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePriority(t *testing.T) {
	t.Run("valid priorities", func(t *testing.T) {
		for p := 0; p <= 4; p++ {
			if err := ValidatePriority(p); err != nil {
				t.Errorf("priority %d: unexpected error: %v", p, err)
			}
		}
	})

	t.Run("invalid priorities", func(t *testing.T) {
		tests := []struct {
			name     string
			priority int
		}{
			{"negative", -1},
			{"too high", 5},
			{"way too high", 100},
			{"way too low", -100},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := ValidatePriority(tt.priority); err == nil {
					t.Errorf("priority %d: expected error, got nil", tt.priority)
				}
			})
		}
	})
}

func TestValidateBlockedBy(t *testing.T) {
	tests := []struct {
		name      string
		taskID    string
		blockedBy []string
		wantErr   bool
	}{
		{"valid reference", "tick-a1b2c3", []string{"tick-d4e5f6"}, false},
		{"self-reference", "tick-a1b2c3", []string{"tick-a1b2c3"}, true},
		{"case-insensitive self-reference", "tick-a1b2c3", []string{"TICK-A1B2C3"}, true},
		{"empty list", "tick-a1b2c3", []string{}, false},
		{"self among others", "tick-a1b2c3", []string{"tick-d4e5f6", "tick-a1b2c3"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBlockedBy(tt.taskID, tt.blockedBy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBlockedBy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateParent(t *testing.T) {
	tests := []struct {
		name     string
		taskID   string
		parentID string
		wantErr  bool
	}{
		{"valid parent", "tick-a1b2c3", "tick-d4e5f6", false},
		{"self-reference", "tick-a1b2c3", "tick-a1b2c3", true},
		{"case-insensitive self-reference", "tick-a1b2c3", "TICK-A1B2C3", true},
		{"empty parent", "tick-a1b2c3", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParent(tt.taskID, tt.parentID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		task := NewTask("tick-abc123", "Test task", -1)

		if task.ID != "tick-abc123" {
			t.Errorf("ID = %q, want %q", task.ID, "tick-abc123")
		}
		if task.Title != "Test task" {
			t.Errorf("Title = %q, want %q", task.Title, "Test task")
		}
		if task.Status != StatusOpen {
			t.Errorf("Status = %q, want %q", task.Status, StatusOpen)
		}
		if task.Priority != 2 {
			t.Errorf("Priority = %d, want 2 (default)", task.Priority)
		}
		if task.Description != "" {
			t.Errorf("Description = %q, want empty", task.Description)
		}
		if task.BlockedBy != nil {
			t.Errorf("BlockedBy = %v, want nil", task.BlockedBy)
		}
		if task.Parent != "" {
			t.Errorf("Parent = %q, want empty", task.Parent)
		}
		if task.Closed != nil {
			t.Errorf("Closed = %v, want nil", task.Closed)
		}
	})

	t.Run("explicit priority", func(t *testing.T) {
		tests := []struct {
			name     string
			priority int
			want     int
		}{
			{"priority 0", 0, 0},
			{"priority 1", 1, 1},
			{"priority 4", 4, 4},
			{"negative uses default", -1, 2},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				task := NewTask("tick-abc123", "Test", tt.priority)
				if task.Priority != tt.want {
					t.Errorf("Priority = %d, want %d", task.Priority, tt.want)
				}
			})
		}
	})

	t.Run("timestamps are current UTC", func(t *testing.T) {
		before := time.Now().UTC().Truncate(time.Second)
		task := NewTask("tick-abc123", "Test task", 2)
		after := time.Now().UTC().Add(time.Second).Truncate(time.Second)

		if task.Created.Before(before) || task.Created.After(after) {
			t.Errorf("Created = %v, not within [%v, %v]", task.Created, before, after)
		}
		if !task.Created.Equal(task.Updated) {
			t.Error("Created and Updated should be equal on new task")
		}
	})
}

func TestStatusConstants(t *testing.T) {
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
			if got := string(tt.status); got != tt.want {
				t.Errorf("status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"specific time", time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC), "2026-01-19T10:00:00Z"},
		{"midnight", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "2026-01-01T00:00:00Z"},
		{"end of day", time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC), "2026-12-31T23:59:59Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatTimestamp(tt.time); got != tt.want {
				t.Errorf("FormatTimestamp() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("matches ISO 8601 pattern", func(t *testing.T) {
		pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
		task := NewTask("tick-abc123", "Test", 2)
		formatted := FormatTimestamp(task.Created)
		if !pattern.MatchString(formatted) {
			t.Errorf("timestamp %q does not match ISO 8601 UTC format", formatted)
		}
	})
}
