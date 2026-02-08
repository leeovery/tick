package task

import (
	"fmt"
	"time"
)

// TransitionResult holds the old and new status after a successful transition.
type TransitionResult struct {
	OldStatus Status
	NewStatus Status
}

// validTransitions maps each command to the set of statuses it can transition from,
// and the resulting target status.
var validTransitions = map[string]struct {
	from []Status
	to   Status
}{
	"start":  {from: []Status{StatusOpen}, to: StatusInProgress},
	"done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
	"cancel": {from: []Status{StatusOpen, StatusInProgress}, to: StatusCancelled},
	"reopen": {from: []Status{StatusDone, StatusCancelled}, to: StatusOpen},
}

// Transition applies a status transition to the given task by command name.
// Valid commands: start, done, cancel, reopen.
// On success, it updates the task's status, updated timestamp, and closed timestamp
// (set on done/cancelled, cleared on reopen). Returns old and new status.
// On invalid transition, the task is not modified and an error is returned.
func Transition(t *Task, command string) (*TransitionResult, error) {
	rule, ok := validTransitions[command]
	if !ok {
		return nil, fmt.Errorf("cannot %s task %s \u2014 unknown command", command, t.ID)
	}

	if !statusIn(t.Status, rule.from) {
		return nil, fmt.Errorf("cannot %s task %s \u2014 status is '%s'", command, t.ID, t.Status)
	}

	oldStatus := t.Status
	now := time.Now().UTC().Truncate(time.Second)

	t.Status = rule.to
	t.Updated = now

	switch rule.to {
	case StatusDone, StatusCancelled:
		t.Closed = &now
	case StatusOpen:
		t.Closed = nil
	}

	return &TransitionResult{
		OldStatus: oldStatus,
		NewStatus: rule.to,
	}, nil
}

// statusIn checks whether s is in the given list of statuses.
func statusIn(s Status, list []Status) bool {
	for _, item := range list {
		if s == item {
			return true
		}
	}
	return false
}
