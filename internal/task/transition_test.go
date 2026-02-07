package task

import (
	"testing"
	"time"
)

// makeTask creates a test task with the given ID and status.
// Closed is set for done/cancelled statuses.
func makeTask(id string, status Status) Task {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	t := Task{
		ID:      id,
		Title:   "Test task",
		Status:  status,
		Created: now,
		Updated: now,
	}
	if status == StatusDone || status == StatusCancelled {
		closed := now
		t.Closed = &closed
	}
	return t
}

func TestTransition_ValidTransitions(t *testing.T) {
	tests := []struct {
		name       string
		from       Status
		command    string
		wantStatus Status
	}{
		{"it transitions open to in_progress via start", StatusOpen, "start", StatusInProgress},
		{"it transitions open to done via done", StatusOpen, "done", StatusDone},
		{"it transitions in_progress to done via done", StatusInProgress, "done", StatusDone},
		{"it transitions open to cancelled via cancel", StatusOpen, "cancel", StatusCancelled},
		{"it transitions in_progress to cancelled via cancel", StatusInProgress, "cancel", StatusCancelled},
		{"it transitions done to open via reopen", StatusDone, "reopen", StatusOpen},
		{"it transitions cancelled to open via reopen", StatusCancelled, "reopen", StatusOpen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := makeTask("tick-a3f2b7", tt.from)

			result, err := Transition(&task, tt.command)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if task.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, task.Status)
			}
			if result.OldStatus != tt.from {
				t.Errorf("expected old status %q, got %q", tt.from, result.OldStatus)
			}
			if result.NewStatus != tt.wantStatus {
				t.Errorf("expected new status %q, got %q", tt.wantStatus, result.NewStatus)
			}
		})
	}
}

func TestTransition_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    Status
		command string
	}{
		{"it rejects start on in_progress task", StatusInProgress, "start"},
		{"it rejects start on done task", StatusDone, "start"},
		{"it rejects start on cancelled task", StatusCancelled, "start"},
		{"it rejects done on done task", StatusDone, "done"},
		{"it rejects done on cancelled task", StatusCancelled, "done"},
		{"it rejects cancel on done task", StatusDone, "cancel"},
		{"it rejects cancel on cancelled task", StatusCancelled, "cancel"},
		{"it rejects reopen on open task", StatusOpen, "reopen"},
		{"it rejects reopen on in_progress task", StatusInProgress, "reopen"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := makeTask("tick-a3f2b7", tt.from)

			_, err := Transition(&task, tt.command)
			if err == nil {
				t.Fatalf("expected error for %s on %s task, got nil", tt.command, tt.from)
			}

			wantMsg := "Error: Cannot " + tt.command + " task tick-a3f2b7 — status is '" + string(tt.from) + "'"
			if err.Error() != wantMsg {
				t.Errorf("expected error %q, got %q", wantMsg, err.Error())
			}
		})
	}
}

func TestTransition_ClosedTimestamp(t *testing.T) {
	t.Run("it sets closed timestamp when transitioning to done", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)
		if task.Closed != nil {
			t.Fatal("precondition: closed should be nil for open task")
		}

		before := time.Now().UTC().Truncate(time.Second)
		_, err := Transition(&task, "done")
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Closed == nil {
			t.Fatal("expected closed timestamp to be set")
		}
		if task.Closed.Before(before) || task.Closed.After(after) {
			t.Errorf("closed timestamp %v not within expected range [%v, %v]", task.Closed, before, after)
		}
	})

	t.Run("it sets closed timestamp when transitioning to cancelled", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)
		if task.Closed != nil {
			t.Fatal("precondition: closed should be nil for open task")
		}

		before := time.Now().UTC().Truncate(time.Second)
		_, err := Transition(&task, "cancel")
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Closed == nil {
			t.Fatal("expected closed timestamp to be set")
		}
		if task.Closed.Before(before) || task.Closed.After(after) {
			t.Errorf("closed timestamp %v not within expected range [%v, %v]", task.Closed, before, after)
		}
	})

	t.Run("it clears closed timestamp when reopening", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusDone)
		if task.Closed == nil {
			t.Fatal("precondition: closed should be set for done task")
		}

		_, err := Transition(&task, "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Closed != nil {
			t.Errorf("expected closed to be nil after reopen, got %v", task.Closed)
		}
	})
}

func TestTransition_UpdatedTimestamp(t *testing.T) {
	t.Run("it updates the updated timestamp on every valid transition", func(t *testing.T) {
		commands := []struct {
			from    Status
			command string
		}{
			{StatusOpen, "start"},
			{StatusOpen, "done"},
			{StatusInProgress, "done"},
			{StatusOpen, "cancel"},
			{StatusInProgress, "cancel"},
			{StatusDone, "reopen"},
			{StatusCancelled, "reopen"},
		}

		for _, c := range commands {
			t.Run(c.command+"_from_"+string(c.from), func(t *testing.T) {
				task := makeTask("tick-a3f2b7", c.from)
				originalUpdated := task.Updated

				// Small sleep to ensure time advances
				time.Sleep(time.Millisecond)

				before := time.Now().UTC().Truncate(time.Second)
				_, err := Transition(&task, c.command)
				after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !task.Updated.After(originalUpdated) && !task.Updated.Equal(before) {
					t.Errorf("expected updated timestamp to be refreshed, got %v (original was %v)", task.Updated, originalUpdated)
				}
				if task.Updated.Before(before) || task.Updated.After(after) {
					t.Errorf("updated timestamp %v not within expected range [%v, %v]", task.Updated, before, after)
				}
			})
		}
	})
}

func TestTransition_NoModificationOnError(t *testing.T) {
	t.Run("it does not modify task on invalid transition", func(t *testing.T) {
		original := makeTask("tick-a3f2b7", StatusDone)
		task := makeTask("tick-a3f2b7", StatusDone)

		_, err := Transition(&task, "start")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify no fields were modified
		if task.Status != original.Status {
			t.Errorf("status was modified: got %q, want %q", task.Status, original.Status)
		}
		if !task.Updated.Equal(original.Updated) {
			t.Errorf("updated was modified: got %v, want %v", task.Updated, original.Updated)
		}
		if task.Closed == nil && original.Closed != nil {
			t.Error("closed was cleared on error")
		}
		if task.Closed != nil && original.Closed == nil {
			t.Error("closed was set on error")
		}
		if task.Closed != nil && original.Closed != nil && !task.Closed.Equal(*original.Closed) {
			t.Errorf("closed was modified: got %v, want %v", task.Closed, original.Closed)
		}
	})
}
