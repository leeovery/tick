package task

import (
	"strings"
	"testing"
	"time"
)

func TestStateMachine_Transition_ValidTransitions(t *testing.T) {
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

	var sm StateMachine
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := makeTask(tt.fromStatus, tt.closed)
			before := time.Now().UTC().Truncate(time.Second)

			result, err := sm.Transition(task, tt.command)
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

func TestStateMachine_Transition_InvalidTransitions(t *testing.T) {
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

	var sm StateMachine
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := makeTask(tt.fromStatus, tt.closed)

			_, err := sm.Transition(task, tt.command)
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

func TestStateMachine_Transition_UnknownCommand(t *testing.T) {
	t.Run("it returns error for unknown command", func(t *testing.T) {
		var sm StateMachine
		task := makeTask(StatusOpen, nil)

		_, err := sm.Transition(task, "foo")
		if err == nil {
			t.Fatal("expected error for unknown command, got nil")
		}

		expected := `unknown command "foo"`
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})
}

func TestStateMachine_Transition_NoModificationOnError(t *testing.T) {
	t.Run("it does not modify task on invalid transition", func(t *testing.T) {
		var sm StateMachine
		original := makeTask(StatusDone, closedTime())
		originalStatus := original.Status
		originalUpdated := original.Updated
		originalClosed := *original.Closed

		_, err := sm.Transition(original, "start")
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

	t.Run("it does not modify task on unknown command", func(t *testing.T) {
		var sm StateMachine
		original := makeTask(StatusOpen, nil)
		originalStatus := original.Status
		originalUpdated := original.Updated

		_, err := sm.Transition(original, "foo")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if original.Status != originalStatus {
			t.Errorf("status was modified: expected %q, got %q", originalStatus, original.Status)
		}
		if !original.Updated.Equal(originalUpdated) {
			t.Errorf("updated was modified: expected %v, got %v", originalUpdated, original.Updated)
		}
		if original.Closed != nil {
			t.Error("closed was set on unknown command")
		}
	})
}

func TestStateMachine_Transition_ClosedTimestamp(t *testing.T) {
	t.Run("it sets closed timestamp when transitioning to done", func(t *testing.T) {
		var sm StateMachine
		task := makeTask(StatusOpen, nil)
		before := time.Now().UTC().Truncate(time.Second)

		_, err := sm.Transition(task, "done")
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
		var sm StateMachine
		task := makeTask(StatusOpen, nil)
		before := time.Now().UTC().Truncate(time.Second)

		_, err := sm.Transition(task, "cancel")
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
		var sm StateMachine
		task := makeTask(StatusDone, closedTime())

		_, err := sm.Transition(task, "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Closed != nil {
			t.Errorf("expected closed to be nil after reopen, got %v", *task.Closed)
		}
	})
}

func TestStateMachine_Transition_UpdatedTimestamp(t *testing.T) {
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

		var sm StateMachine
		for _, cmd := range commands {
			t.Run(cmd.command, func(t *testing.T) {
				task := makeTask(cmd.status, cmd.closed)
				originalUpdated := task.Updated

				before := time.Now().UTC().Truncate(time.Second)

				_, err := sm.Transition(task, cmd.command)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				after := time.Now().UTC().Truncate(time.Second)

				if task.Updated.Before(before) || task.Updated.After(after) {
					t.Errorf("updated timestamp %v not in expected range [%v, %v]", task.Updated, before, after)
				}
				if task.Updated.Equal(originalUpdated) {
					t.Error("expected updated timestamp to change from original value")
				}
			})
		}
	})
}

func TestStateMachine_ValidateAddDep(t *testing.T) {
	var sm StateMachine

	t.Run("it allows valid dependency between unrelated tasks", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa"},
			{ID: "tick-bbb"},
		}

		err := sm.ValidateAddDep(tasks, "tick-aaa", "tick-bbb")
		if err != nil {
			t.Errorf("expected no error for valid dependency, got: %v", err)
		}
	})

	t.Run("it rejects direct self-reference", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa"},
		}

		err := sm.ValidateAddDep(tasks, "tick-aaa", "tick-aaa")
		if err == nil {
			t.Fatal("expected error for self-reference, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-aaa → tick-aaa"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it rejects 2-node cycle with path", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb"},
		}

		err := sm.ValidateAddDep(tasks, "tick-bbb", "tick-aaa")
		if err == nil {
			t.Fatal("expected error for 2-node cycle, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-bbb → tick-aaa → tick-bbb"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it rejects 3+ node cycle with full path", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb", BlockedBy: []string{"tick-ccc"}},
			{ID: "tick-ccc"},
		}

		err := sm.ValidateAddDep(tasks, "tick-ccc", "tick-aaa")
		if err == nil {
			t.Fatal("expected error for 3+ node cycle, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-ccc → tick-aaa → tick-bbb → tick-ccc"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it rejects child blocked by own parent", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child", Parent: "tick-parent"},
		}

		err := sm.ValidateAddDep(tasks, "tick-child", "tick-parent")
		if err == nil {
			t.Fatal("expected error for child blocked by parent, got nil")
		}

		expected := "cannot add dependency - tick-child cannot be blocked by its parent tick-parent\n(would create unworkable task due to leaf-only ready rule)"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it allows parent blocked by own child", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child", Parent: "tick-parent"},
		}

		err := sm.ValidateAddDep(tasks, "tick-parent", "tick-child")
		if err != nil {
			t.Errorf("expected no error for parent blocked by child, got: %v", err)
		}
	})

	t.Run("it detects cycle with mixed-case IDs", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb"},
		}

		err := sm.ValidateAddDep(tasks, "TICK-BBB", "TICK-AAA")
		if err == nil {
			t.Fatal("expected error for mixed-case cycle, got nil")
		}
		if !strings.Contains(err.Error(), "creates cycle") {
			t.Errorf("expected cycle error, got: %v", err)
		}
	})

	t.Run("it blocks adding dependency on cancelled task", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", Status: StatusOpen},
			{ID: "tick-bbb", Status: StatusCancelled},
		}

		err := sm.ValidateAddDep(tasks, "tick-aaa", "tick-bbb")
		if err == nil {
			t.Fatal("expected error for cancelled blocker, got nil")
		}

		expected := "cannot add dependency on cancelled task, reopen it first"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it allows adding dependency on open task", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", Status: StatusOpen},
			{ID: "tick-bbb", Status: StatusOpen},
		}

		err := sm.ValidateAddDep(tasks, "tick-aaa", "tick-bbb")
		if err != nil {
			t.Errorf("expected no error for open blocker, got: %v", err)
		}
	})

	t.Run("it allows adding dependency on in_progress task", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", Status: StatusOpen},
			{ID: "tick-bbb", Status: StatusInProgress},
		}

		err := sm.ValidateAddDep(tasks, "tick-aaa", "tick-bbb")
		if err != nil {
			t.Errorf("expected no error for in_progress blocker, got: %v", err)
		}
	})

	t.Run("it allows adding dependency on done task", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", Status: StatusOpen},
			{ID: "tick-bbb", Status: StatusDone},
		}

		err := sm.ValidateAddDep(tasks, "tick-aaa", "tick-bbb")
		if err != nil {
			t.Errorf("expected no error for done blocker, got: %v", err)
		}
	})

	t.Run("it still detects cycles after cancelled check passes", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", Status: StatusOpen, BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb", Status: StatusOpen},
		}

		err := sm.ValidateAddDep(tasks, "tick-bbb", "tick-aaa")
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		if !strings.Contains(err.Error(), "creates cycle") {
			t.Errorf("expected cycle error, got: %v", err)
		}
	})

	t.Run("it detects child-blocked-by-parent with mixed-case IDs", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child", Parent: "tick-parent"},
		}

		err := sm.ValidateAddDep(tasks, "TICK-CHILD", "TICK-PARENT")
		if err == nil {
			t.Fatal("expected error for mixed-case child-blocked-by-parent, got nil")
		}
		if !strings.Contains(err.Error(), "cannot be blocked by its parent") {
			t.Errorf("expected parent error, got: %v", err)
		}
	})
}

func TestStateMachine_ValidateAddChild(t *testing.T) {
	var sm StateMachine

	t.Run("it blocks adding child to cancelled parent", func(t *testing.T) {
		parent := makeTask(StatusCancelled, closedTime())

		err := sm.ValidateAddChild(parent)
		if err == nil {
			t.Fatal("expected error for cancelled parent, got nil")
		}

		expected := "cannot add child to cancelled task, reopen it first"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it allows adding child to open parent", func(t *testing.T) {
		parent := makeTask(StatusOpen, nil)

		err := sm.ValidateAddChild(parent)
		if err != nil {
			t.Errorf("expected no error for open parent, got: %v", err)
		}
	})

	t.Run("it allows adding child to in_progress parent", func(t *testing.T) {
		parent := makeTask(StatusInProgress, nil)

		err := sm.ValidateAddChild(parent)
		if err != nil {
			t.Errorf("expected no error for in_progress parent, got: %v", err)
		}
	})

	t.Run("it allows adding child to done parent", func(t *testing.T) {
		parent := makeTask(StatusDone, closedTime())

		err := sm.ValidateAddChild(parent)
		if err != nil {
			t.Errorf("expected no error for done parent, got: %v", err)
		}
	})
}
