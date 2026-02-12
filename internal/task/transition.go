package task

import (
	"fmt"
	"time"
)

// TransitionResult holds the old and new status after a successful transition,
// enabling the caller to format output like "tick-a3f2b7: open -> in_progress".
type TransitionResult struct {
	OldStatus Status
	NewStatus Status
}

// transitionTable maps each command to the set of valid source statuses and the target status.
var transitionTable = map[string]struct {
	from []Status
	to   Status
}{
	"start":  {from: []Status{StatusOpen}, to: StatusInProgress},
	"done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
	"cancel": {from: []Status{StatusOpen, StatusInProgress}, to: StatusCancelled},
	"reopen": {from: []Status{StatusDone, StatusCancelled}, to: StatusOpen},
}

// Transition applies a status transition to the given task by command name.
// Valid commands: "start", "done", "cancel", "reopen".
//
// On success, the task's Status, Updated, and Closed fields are mutated in place,
// and a TransitionResult is returned with the old and new status.
//
// On failure (invalid command or invalid transition), the task is not modified
// and an error is returned.
func Transition(t *Task, command string) (TransitionResult, error) {
	rule, ok := transitionTable[command]
	if !ok {
		return TransitionResult{}, fmt.Errorf("unknown command %q", command)
	}

	if !statusIn(t.Status, rule.from) {
		return TransitionResult{}, fmt.Errorf(
			"cannot %s task %s \u2014 status is '%s'",
			command, t.ID, t.Status,
		)
	}

	oldStatus := t.Status
	now := time.Now().UTC().Truncate(time.Second)

	t.Status = rule.to
	t.Updated = now

	switch rule.to {
	case StatusDone, StatusCancelled:
		t.Closed = &now
	case StatusOpen:
		// reopen clears closed
		t.Closed = nil
	}

	return TransitionResult{
		OldStatus: oldStatus,
		NewStatus: rule.to,
	}, nil
}

// statusIn checks whether s is contained in the given slice of statuses.
func statusIn(s Status, statuses []Status) bool {
	for _, v := range statuses {
		if s == v {
			return true
		}
	}
	return false
}
