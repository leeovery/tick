package task

import "slices"

// TransitionResult holds the old and new status after a successful transition.
// Auto carries the same flag as the transition's TransitionRecord: false when the
// caller asked for the change, true when the system produced it.
type TransitionResult struct {
	OldStatus Status
	NewStatus Status
	Auto      bool
}

// Transition applies a status transition to the given task by command name.
// Valid commands: "start", "done", "cancel", "reopen".
//
// On success, the task's Status, Updated, and Closed fields are mutated in place,
// and a TransitionResult is returned with the old and new status.
//
// On failure (invalid command or invalid transition), the task is not modified
// and an error is returned.
//
// This is a convenience wrapper that delegates to StateMachine.Transition.
func Transition(t *Task, command string) (TransitionResult, error) {
	var sm StateMachine
	return sm.Transition(t, command)
}

// statusIn checks whether s is contained in the given slice of statuses.
func statusIn(s Status, statuses []Status) bool {
	return slices.Contains(statuses, s)
}
