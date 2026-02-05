package task

import (
	"fmt"
	"time"
)

// TransitionResult contains the old and new status after a successful transition.
type TransitionResult struct {
	OldStatus Status
	NewStatus Status
}

// Transition applies a status transition command to a task.
// Valid commands: start, done, cancel, reopen
// On success, modifies the task's status, updated timestamp, and closed timestamp.
// On failure, returns an error and leaves the task unmodified.
func Transition(task *Task, command string) (TransitionResult, error) {
	oldStatus := task.Status

	// Determine target status and validate transition
	var newStatus Status
	var valid bool

	switch command {
	case "start":
		// open -> in_progress
		valid = oldStatus == StatusOpen
		newStatus = StatusInProgress

	case "done":
		// open or in_progress -> done
		valid = oldStatus == StatusOpen || oldStatus == StatusInProgress
		newStatus = StatusDone

	case "cancel":
		// open or in_progress -> cancelled
		valid = oldStatus == StatusOpen || oldStatus == StatusInProgress
		newStatus = StatusCancelled

	case "reopen":
		// done or cancelled -> open
		valid = oldStatus == StatusDone || oldStatus == StatusCancelled
		newStatus = StatusOpen

	default:
		return TransitionResult{}, fmt.Errorf("unknown command: %s", command)
	}

	if !valid {
		return TransitionResult{}, fmt.Errorf("cannot %s task %s - status is '%s'", command, task.ID, oldStatus)
	}

	// Apply transition
	now := time.Now().UTC().Format(time.RFC3339)

	task.Status = newStatus
	task.Updated = now

	// Handle closed timestamp
	if newStatus == StatusDone || newStatus == StatusCancelled {
		task.Closed = now
	} else if command == "reopen" {
		task.Closed = ""
	}

	return TransitionResult{
		OldStatus: oldStatus,
		NewStatus: newStatus,
	}, nil
}
