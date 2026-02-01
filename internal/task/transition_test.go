package task

import (
	"testing"
	"time"
)

func TestTransition(t *testing.T) {
	// Helper to create a task with a given status.
	makeTask := func(status Status) Task {
		now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		return Task{
			ID:      "tick-abc123",
			Title:   "Test task",
			Status:  status,
			Created: now,
			Updated: now,
		}
	}

	t.Run("valid transitions", func(t *testing.T) {
		tests := []struct {
			name       string
			from       Status
			command    string
			wantStatus Status
		}{
			{"start: open → in_progress", StatusOpen, "start", StatusInProgress},
			{"done: open → done", StatusOpen, "done", StatusDone},
			{"done: in_progress → done", StatusInProgress, "done", StatusDone},
			{"cancel: open → cancelled", StatusOpen, "cancel", StatusCancelled},
			{"cancel: in_progress → cancelled", StatusInProgress, "cancel", StatusCancelled},
			{"reopen: done → open", StatusDone, "reopen", StatusOpen},
			{"reopen: cancelled → open", StatusCancelled, "reopen", StatusOpen},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				task := makeTask(tt.from)
				result, err := Transition(&task, tt.command)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if task.Status != tt.wantStatus {
					t.Errorf("status = %q, want %q", task.Status, tt.wantStatus)
				}
				if result.OldStatus != tt.from {
					t.Errorf("OldStatus = %q, want %q", result.OldStatus, tt.from)
				}
				if result.NewStatus != tt.wantStatus {
					t.Errorf("NewStatus = %q, want %q", result.NewStatus, tt.wantStatus)
				}
			})
		}
	})

	t.Run("invalid transitions", func(t *testing.T) {
		tests := []struct {
			name    string
			from    Status
			command string
		}{
			{"start on in_progress", StatusInProgress, "start"},
			{"start on done", StatusDone, "start"},
			{"start on cancelled", StatusCancelled, "start"},
			{"done on done", StatusDone, "done"},
			{"done on cancelled", StatusCancelled, "done"},
			{"cancel on done", StatusDone, "cancel"},
			{"cancel on cancelled", StatusCancelled, "cancel"},
			{"reopen on open", StatusOpen, "reopen"},
			{"reopen on in_progress", StatusInProgress, "reopen"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				task := makeTask(tt.from)
				_, err := Transition(&task, tt.command)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			})
		}
	})

	t.Run("sets closed timestamp on done", func(t *testing.T) {
		task := makeTask(StatusOpen)
		before := time.Now().UTC().Truncate(time.Second)
		Transition(&task, "done")
		after := time.Now().UTC().Add(time.Second).Truncate(time.Second)

		if task.Closed == nil {
			t.Fatal("Closed should be set after done")
		}
		if task.Closed.Before(before) || task.Closed.After(after) {
			t.Errorf("Closed = %v, not within [%v, %v]", *task.Closed, before, after)
		}
	})

	t.Run("sets closed timestamp on cancel", func(t *testing.T) {
		task := makeTask(StatusOpen)
		before := time.Now().UTC().Truncate(time.Second)
		Transition(&task, "cancel")
		after := time.Now().UTC().Add(time.Second).Truncate(time.Second)

		if task.Closed == nil {
			t.Fatal("Closed should be set after cancel")
		}
		if task.Closed.Before(before) || task.Closed.After(after) {
			t.Errorf("Closed = %v, not within [%v, %v]", *task.Closed, before, after)
		}
	})

	t.Run("clears closed timestamp on reopen", func(t *testing.T) {
		task := makeTask(StatusDone)
		closedTime := time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC)
		task.Closed = &closedTime

		Transition(&task, "reopen")

		if task.Closed != nil {
			t.Errorf("Closed should be nil after reopen, got %v", *task.Closed)
		}
	})

	t.Run("updates timestamp on every valid transition", func(t *testing.T) {
		task := makeTask(StatusOpen)
		originalUpdated := task.Updated

		time.Sleep(time.Millisecond) // ensure time advances
		Transition(&task, "start")

		if !task.Updated.After(originalUpdated) {
			t.Errorf("Updated should advance, got %v (was %v)", task.Updated, originalUpdated)
		}
	})

	t.Run("does not modify task on invalid transition", func(t *testing.T) {
		task := makeTask(StatusDone)
		originalStatus := task.Status
		originalUpdated := task.Updated
		closedTime := time.Date(2026, 1, 14, 10, 0, 0, 0, time.UTC)
		task.Closed = &closedTime
		originalClosed := *task.Closed

		Transition(&task, "start")

		if task.Status != originalStatus {
			t.Errorf("Status changed on invalid transition: %q → %q", originalStatus, task.Status)
		}
		if task.Updated != originalUpdated {
			t.Errorf("Updated changed on invalid transition")
		}
		if task.Closed == nil || *task.Closed != originalClosed {
			t.Errorf("Closed changed on invalid transition")
		}
	})

	t.Run("error message includes command and status", func(t *testing.T) {
		task := makeTask(StatusDone)
		_, err := Transition(&task, "start")
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !containsAll(msg, "start", "tick-abc123", "done") {
			t.Errorf("error message should include command, ID, and status, got: %q", msg)
		}
	})
}

// containsAll checks that s contains all of the given substrings.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
