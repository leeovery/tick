package task

import (
	"testing"
	"time"
)

// helper to create a task with a given status and optional closed timestamp.
func makeTask(status Status, closed *time.Time) *Task {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	return &Task{
		ID:       "tick-a1b2c3",
		Title:    "Test task",
		Status:   status,
		Priority: 2,
		Created:  now,
		Updated:  now,
		Closed:   closed,
	}
}

func closedTime() *time.Time {
	t := time.Date(2026, 1, 16, 12, 0, 0, 0, time.UTC)
	return &t
}

func TestTransition_ValidTransitions(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		fromStatus Status
		toStatus   Status
		closed     *time.Time
	}{
		{
			name:       "it transitions open to in_progress via start",
			command:    "start",
			fromStatus: StatusOpen,
			toStatus:   StatusInProgress,
			closed:     nil,
		},
		{
			name:       "it transitions open to done via done",
			command:    "done",
			fromStatus: StatusOpen,
			toStatus:   StatusDone,
			closed:     nil,
		},
		{
			name:       "it transitions in_progress to done via done",
			command:    "done",
			fromStatus: StatusInProgress,
			toStatus:   StatusDone,
			closed:     nil,
		},
		{
			name:       "it transitions open to cancelled via cancel",
			command:    "cancel",
			fromStatus: StatusOpen,
			toStatus:   StatusCancelled,
			closed:     nil,
		},
		{
			name:       "it transitions in_progress to cancelled via cancel",
			command:    "cancel",
			fromStatus: StatusInProgress,
			toStatus:   StatusCancelled,
			closed:     nil,
		},
		{
			name:       "it transitions done to open via reopen",
			command:    "reopen",
			fromStatus: StatusDone,
			toStatus:   StatusOpen,
			closed:     closedTime(),
		},
		{
			name:       "it transitions cancelled to open via reopen",
			command:    "reopen",
			fromStatus: StatusCancelled,
			toStatus:   StatusOpen,
			closed:     closedTime(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := makeTask(tt.fromStatus, tt.closed)
			before := time.Now().UTC().Truncate(time.Second)

			result, err := Transition(task, tt.command)
			if err != nil {
				t.Fatalf("Transition returned unexpected error: %v", err)
			}

			after := time.Now().UTC().Truncate(time.Second)

			if task.Status != tt.toStatus {
				t.Errorf("expected status %q, got %q", tt.toStatus, task.Status)
			}
			if result.OldStatus != tt.fromStatus {
				t.Errorf("expected OldStatus %q, got %q", tt.fromStatus, result.OldStatus)
			}
			if result.NewStatus != tt.toStatus {
				t.Errorf("expected NewStatus %q, got %q", tt.toStatus, result.NewStatus)
			}
			if task.Updated.Before(before) || task.Updated.After(after) {
				t.Errorf("updated timestamp %v not in expected range [%v, %v]", task.Updated, before, after)
			}
		})
	}
}

func TestTransition_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		fromStatus Status
		closed     *time.Time
	}{
		{
			name:       "it rejects start on in_progress task",
			command:    "start",
			fromStatus: StatusInProgress,
		},
		{
			name:       "it rejects start on done task",
			command:    "start",
			fromStatus: StatusDone,
			closed:     closedTime(),
		},
		{
			name:       "it rejects start on cancelled task",
			command:    "start",
			fromStatus: StatusCancelled,
			closed:     closedTime(),
		},
		{
			name:       "it rejects done on done task",
			command:    "done",
			fromStatus: StatusDone,
			closed:     closedTime(),
		},
		{
			name:       "it rejects done on cancelled task",
			command:    "done",
			fromStatus: StatusCancelled,
			closed:     closedTime(),
		},
		{
			name:       "it rejects cancel on done task",
			command:    "cancel",
			fromStatus: StatusDone,
			closed:     closedTime(),
		},
		{
			name:       "it rejects cancel on cancelled task",
			command:    "cancel",
			fromStatus: StatusCancelled,
			closed:     closedTime(),
		},
		{
			name:       "it rejects reopen on open task",
			command:    "reopen",
			fromStatus: StatusOpen,
		},
		{
			name:       "it rejects reopen on in_progress task",
			command:    "reopen",
			fromStatus: StatusInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := makeTask(tt.fromStatus, tt.closed)

			_, err := Transition(task, tt.command)
			if err == nil {
				t.Fatalf("expected error for %s on %s task, got nil", tt.command, tt.fromStatus)
			}

			expectedMsg := "cannot " + tt.command + " task " + task.ID + " \u2014 status is '" + string(tt.fromStatus) + "'"
			if err.Error() != expectedMsg {
				t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
			}
		})
	}
}

func TestTransition_ClosedTimestamp(t *testing.T) {
	t.Run("it sets closed timestamp when transitioning to done", func(t *testing.T) {
		task := makeTask(StatusOpen, nil)
		before := time.Now().UTC().Truncate(time.Second)

		_, err := Transition(task, "done")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		after := time.Now().UTC().Truncate(time.Second)

		if task.Closed == nil {
			t.Fatal("expected closed timestamp to be set, got nil")
		}
		if task.Closed.Before(before) || task.Closed.After(after) {
			t.Errorf("closed timestamp %v not in expected range [%v, %v]", *task.Closed, before, after)
		}
	})

	t.Run("it sets closed timestamp when transitioning to cancelled", func(t *testing.T) {
		task := makeTask(StatusOpen, nil)
		before := time.Now().UTC().Truncate(time.Second)

		_, err := Transition(task, "cancel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		after := time.Now().UTC().Truncate(time.Second)

		if task.Closed == nil {
			t.Fatal("expected closed timestamp to be set, got nil")
		}
		if task.Closed.Before(before) || task.Closed.After(after) {
			t.Errorf("closed timestamp %v not in expected range [%v, %v]", *task.Closed, before, after)
		}
	})

	t.Run("it clears closed timestamp when reopening", func(t *testing.T) {
		task := makeTask(StatusDone, closedTime())

		_, err := Transition(task, "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Closed != nil {
			t.Errorf("expected closed to be nil after reopen, got %v", *task.Closed)
		}
	})
}

func TestTransition_UpdatedTimestamp(t *testing.T) {
	t.Run("it updates the updated timestamp on every valid transition", func(t *testing.T) {
		commands := []struct {
			command string
			status  Status
			closed  *time.Time
		}{
			{"start", StatusOpen, nil},
			{"done", StatusOpen, nil},
			{"cancel", StatusInProgress, nil},
			{"reopen", StatusDone, closedTime()},
		}

		for _, cmd := range commands {
			t.Run(cmd.command, func(t *testing.T) {
				task := makeTask(cmd.status, cmd.closed)
				originalUpdated := task.Updated

				// Small sleep to ensure time difference is detectable
				before := time.Now().UTC().Truncate(time.Second)

				_, err := Transition(task, cmd.command)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				after := time.Now().UTC().Truncate(time.Second)

				if task.Updated.Before(before) || task.Updated.After(after) {
					t.Errorf("updated timestamp %v not in expected range [%v, %v]", task.Updated, before, after)
				}
				// The original updated was 2026-01-15, so it should have changed
				if task.Updated.Equal(originalUpdated) {
					t.Error("expected updated timestamp to change from original value")
				}
			})
		}
	})
}

func TestTransition_NoModificationOnInvalid(t *testing.T) {
	t.Run("it does not modify task on invalid transition", func(t *testing.T) {
		original := makeTask(StatusDone, closedTime())
		originalStatus := original.Status
		originalUpdated := original.Updated
		originalClosed := *original.Closed

		_, err := Transition(original, "start")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if original.Status != originalStatus {
			t.Errorf("status was modified: expected %q, got %q", originalStatus, original.Status)
		}
		if !original.Updated.Equal(originalUpdated) {
			t.Errorf("updated was modified: expected %v, got %v", originalUpdated, original.Updated)
		}
		if original.Closed == nil {
			t.Error("closed was cleared on invalid transition")
		} else if !original.Closed.Equal(originalClosed) {
			t.Errorf("closed was modified: expected %v, got %v", originalClosed, *original.Closed)
		}
	})
}
