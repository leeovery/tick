package task

import (
	"fmt"
	"time"
)

// TransitionResult holds the outcome of a successful status transition.
type TransitionResult struct {
	OldStatus Status
	NewStatus Status
}

// Transition applies a status transition to a task by command name.
// Valid commands: "start", "done", "cancel", "reopen".
// On success, modifies the task in place and returns the transition result.
// On failure, returns an error without modifying the task.
func Transition(t *Task, command string) (TransitionResult, error) {
	newStatus, ok := resolveTransition(t.Status, command)
	if !ok {
		return TransitionResult{}, fmt.Errorf(
			"Cannot %s task %s — status is '%s'", command, t.ID, t.Status,
		)
	}

	oldStatus := t.Status
	now := time.Now().UTC().Truncate(time.Second)

	t.Status = newStatus
	t.Updated = now

	switch newStatus {
	case StatusDone, StatusCancelled:
		t.Closed = &now
	case StatusOpen:
		// reopen clears closed
		if command == "reopen" {
			t.Closed = nil
		}
	}

	return TransitionResult{OldStatus: oldStatus, NewStatus: newStatus}, nil
}

// resolveTransition returns the target status for a given current status and
// command, or false if the transition is invalid.
func resolveTransition(current Status, command string) (Status, bool) {
	type key struct {
		status  Status
		command string
	}

	transitions := map[key]Status{
		{StatusOpen, "start"}:        StatusInProgress,
		{StatusOpen, "done"}:         StatusDone,
		{StatusInProgress, "done"}:   StatusDone,
		{StatusOpen, "cancel"}:       StatusCancelled,
		{StatusInProgress, "cancel"}: StatusCancelled,
		{StatusDone, "reopen"}:       StatusOpen,
		{StatusCancelled, "reopen"}:  StatusOpen,
	}

	target, ok := transitions[key{current, command}]
	return target, ok
}
