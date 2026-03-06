package task

import (
	"fmt"
	"time"
)

// StateMachine consolidates task status transition logic.
// It is stateless — no fields, no constructor needed — serving as a method grouping.
type StateMachine struct{}

// transitionRule defines a valid status transition: the set of valid source statuses and the target.
type transitionRule struct {
	from []Status
	to   Status
}

// transitions maps each command to its transition rule.
var transitions = map[string]transitionRule{
	"start":  {from: []Status{StatusOpen}, to: StatusInProgress},
	"done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
	"cancel": {from: []Status{StatusOpen, StatusInProgress}, to: StatusCancelled},
	"reopen": {from: []Status{StatusDone, StatusCancelled}, to: StatusOpen},
}

// Transition applies a status transition to the given task by action name.
// Valid actions: "start", "done", "cancel", "reopen".
//
// On success, the task's Status, Updated, and Closed fields are mutated in place,
// and a TransitionResult is returned with the old and new status.
//
// On failure (unknown action or invalid transition), the task is not modified
// and an error is returned.
func (sm StateMachine) Transition(t *Task, action string) (TransitionResult, error) {
	rule, ok := transitions[action]
	if !ok {
		return TransitionResult{}, fmt.Errorf("unknown command %q", action)
	}

	if !statusIn(t.Status, rule.from) {
		return TransitionResult{}, fmt.Errorf(
			"cannot %s task %s — status is '%s'",
			action, t.ID, t.Status,
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
