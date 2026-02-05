package task

import (
	"testing"
	"time"
)

func TestTransition(t *testing.T) {
	// Helper to create a task with given status
	makeTask := func(id string, status Status) *Task {
		return &Task{
			ID:      id,
			Title:   "Test task",
			Status:  status,
			Created: "2026-01-19T10:00:00Z",
			Updated: "2026-01-19T10:00:00Z",
		}
	}

	// Helper to create a closed task (done or cancelled)
	makeClosedTask := func(id string, status Status) *Task {
		task := makeTask(id, status)
		task.Closed = "2026-01-19T12:00:00Z"
		return task
	}

	// Valid transitions
	t.Run("it transitions open to in_progress via start", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)
		result, err := Transition(task, "start")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != StatusInProgress {
			t.Errorf("status = %q, want %q", task.Status, StatusInProgress)
		}
		if result.OldStatus != StatusOpen {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusOpen)
		}
		if result.NewStatus != StatusInProgress {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusInProgress)
		}
	})

	t.Run("it transitions open to done via done", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)
		result, err := Transition(task, "done")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != StatusDone {
			t.Errorf("status = %q, want %q", task.Status, StatusDone)
		}
		if result.OldStatus != StatusOpen {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusOpen)
		}
		if result.NewStatus != StatusDone {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusDone)
		}
	})

	t.Run("it transitions in_progress to done via done", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusInProgress)
		result, err := Transition(task, "done")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != StatusDone {
			t.Errorf("status = %q, want %q", task.Status, StatusDone)
		}
		if result.OldStatus != StatusInProgress {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusInProgress)
		}
		if result.NewStatus != StatusDone {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusDone)
		}
	})

	t.Run("it transitions open to cancelled via cancel", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)
		result, err := Transition(task, "cancel")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != StatusCancelled {
			t.Errorf("status = %q, want %q", task.Status, StatusCancelled)
		}
		if result.OldStatus != StatusOpen {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusOpen)
		}
		if result.NewStatus != StatusCancelled {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusCancelled)
		}
	})

	t.Run("it transitions in_progress to cancelled via cancel", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusInProgress)
		result, err := Transition(task, "cancel")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != StatusCancelled {
			t.Errorf("status = %q, want %q", task.Status, StatusCancelled)
		}
		if result.OldStatus != StatusInProgress {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusInProgress)
		}
		if result.NewStatus != StatusCancelled {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusCancelled)
		}
	})

	t.Run("it transitions done to open via reopen", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusDone)
		result, err := Transition(task, "reopen")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != StatusOpen {
			t.Errorf("status = %q, want %q", task.Status, StatusOpen)
		}
		if result.OldStatus != StatusDone {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusDone)
		}
		if result.NewStatus != StatusOpen {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusOpen)
		}
	})

	t.Run("it transitions cancelled to open via reopen", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusCancelled)
		result, err := Transition(task, "reopen")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != StatusOpen {
			t.Errorf("status = %q, want %q", task.Status, StatusOpen)
		}
		if result.OldStatus != StatusCancelled {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusCancelled)
		}
		if result.NewStatus != StatusOpen {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusOpen)
		}
	})

	// Invalid transitions - start
	t.Run("it rejects start on in_progress task", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusInProgress)
		_, err := Transition(task, "start")

		if err == nil {
			t.Fatal("expected error for start on in_progress task")
		}
		expected := "cannot start task tick-a3f2b7 - status is 'in_progress'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it rejects start on done task", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusDone)
		_, err := Transition(task, "start")

		if err == nil {
			t.Fatal("expected error for start on done task")
		}
		expected := "cannot start task tick-a3f2b7 - status is 'done'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it rejects start on cancelled task", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusCancelled)
		_, err := Transition(task, "start")

		if err == nil {
			t.Fatal("expected error for start on cancelled task")
		}
		expected := "cannot start task tick-a3f2b7 - status is 'cancelled'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	// Invalid transitions - done
	t.Run("it rejects done on done task", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusDone)
		_, err := Transition(task, "done")

		if err == nil {
			t.Fatal("expected error for done on done task")
		}
		expected := "cannot done task tick-a3f2b7 - status is 'done'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it rejects done on cancelled task", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusCancelled)
		_, err := Transition(task, "done")

		if err == nil {
			t.Fatal("expected error for done on cancelled task")
		}
		expected := "cannot done task tick-a3f2b7 - status is 'cancelled'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	// Invalid transitions - cancel
	t.Run("it rejects cancel on done task", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusDone)
		_, err := Transition(task, "cancel")

		if err == nil {
			t.Fatal("expected error for cancel on done task")
		}
		expected := "cannot cancel task tick-a3f2b7 - status is 'done'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it rejects cancel on cancelled task", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusCancelled)
		_, err := Transition(task, "cancel")

		if err == nil {
			t.Fatal("expected error for cancel on cancelled task")
		}
		expected := "cannot cancel task tick-a3f2b7 - status is 'cancelled'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	// Invalid transitions - reopen
	t.Run("it rejects reopen on open task", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)
		_, err := Transition(task, "reopen")

		if err == nil {
			t.Fatal("expected error for reopen on open task")
		}
		expected := "cannot reopen task tick-a3f2b7 - status is 'open'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it rejects reopen on in_progress task", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusInProgress)
		_, err := Transition(task, "reopen")

		if err == nil {
			t.Fatal("expected error for reopen on in_progress task")
		}
		expected := "cannot reopen task tick-a3f2b7 - status is 'in_progress'"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	// Timestamp tests
	t.Run("it sets closed timestamp when transitioning to done", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)

		before := time.Now().UTC()
		_, err := Transition(task, "done")
		after := time.Now().UTC()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Closed == "" {
			t.Fatal("closed timestamp should be set")
		}

		closedTime, err := time.Parse(time.RFC3339, task.Closed)
		if err != nil {
			t.Fatalf("failed to parse closed timestamp: %v", err)
		}
		if closedTime.Before(before.Truncate(time.Second)) || closedTime.After(after.Add(time.Second)) {
			t.Errorf("closed timestamp %v not in expected range", closedTime)
		}
	})

	t.Run("it sets closed timestamp when transitioning to cancelled", func(t *testing.T) {
		task := makeTask("tick-a3f2b7", StatusOpen)

		before := time.Now().UTC()
		_, err := Transition(task, "cancel")
		after := time.Now().UTC()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Closed == "" {
			t.Fatal("closed timestamp should be set")
		}

		closedTime, err := time.Parse(time.RFC3339, task.Closed)
		if err != nil {
			t.Fatalf("failed to parse closed timestamp: %v", err)
		}
		if closedTime.Before(before.Truncate(time.Second)) || closedTime.After(after.Add(time.Second)) {
			t.Errorf("closed timestamp %v not in expected range", closedTime)
		}
	})

	t.Run("it clears closed timestamp when reopening", func(t *testing.T) {
		task := makeClosedTask("tick-a3f2b7", StatusDone)

		// Verify closed is set before reopen
		if task.Closed == "" {
			t.Fatal("closed timestamp should be set before reopen")
		}

		_, err := Transition(task, "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Closed != "" {
			t.Errorf("closed timestamp should be cleared, got %q", task.Closed)
		}
	})

	t.Run("it updates the updated timestamp on every valid transition", func(t *testing.T) {
		tests := []struct {
			name    string
			status  Status
			command string
			closed  bool
		}{
			{"start", StatusOpen, "start", false},
			{"done from open", StatusOpen, "done", false},
			{"done from in_progress", StatusInProgress, "done", false},
			{"cancel from open", StatusOpen, "cancel", false},
			{"cancel from in_progress", StatusInProgress, "cancel", false},
			{"reopen from done", StatusDone, "reopen", true},
			{"reopen from cancelled", StatusCancelled, "reopen", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var task *Task
				if tt.closed {
					task = makeClosedTask("tick-a3f2b7", tt.status)
				} else {
					task = makeTask("tick-a3f2b7", tt.status)
				}

				originalUpdated := task.Updated
				before := time.Now().UTC()
				_, err := Transition(task, tt.command)
				after := time.Now().UTC()

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if task.Updated == originalUpdated {
					t.Error("updated timestamp should have changed")
				}

				updatedTime, err := time.Parse(time.RFC3339, task.Updated)
				if err != nil {
					t.Fatalf("failed to parse updated timestamp: %v", err)
				}
				if updatedTime.Before(before.Truncate(time.Second)) || updatedTime.After(after.Add(time.Second)) {
					t.Errorf("updated timestamp %v not in expected range", updatedTime)
				}
			})
		}
	})

	t.Run("it does not modify task on invalid transition", func(t *testing.T) {
		// Create a task and capture its state
		task := makeClosedTask("tick-a3f2b7", StatusDone)
		originalStatus := task.Status
		originalUpdated := task.Updated
		originalClosed := task.Closed

		// Attempt invalid transition (start on done task)
		_, err := Transition(task, "start")

		if err == nil {
			t.Fatal("expected error for invalid transition")
		}

		// Verify task was not modified
		if task.Status != originalStatus {
			t.Errorf("status was modified: got %q, want %q", task.Status, originalStatus)
		}
		if task.Updated != originalUpdated {
			t.Errorf("updated was modified: got %q, want %q", task.Updated, originalUpdated)
		}
		if task.Closed != originalClosed {
			t.Errorf("closed was modified: got %q, want %q", task.Closed, originalClosed)
		}
	})
}
