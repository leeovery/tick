package task

// TransitionResult holds the old and new status after a successful transition,
// enabling the caller to format output like "tick-a3f2b7: open -> in_progress".
type TransitionResult struct {
	OldStatus Status
	NewStatus Status
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
	for _, v := range statuses {
		if s == v {
			return true
		}
	}
	return false
}
