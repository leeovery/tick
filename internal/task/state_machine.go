package task

import (
	"fmt"
	"strings"
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

// ValidateAddDep checks that adding blockerID as a blocker of taskID
// does not create a circular dependency or a child-blocked-by-parent relationship.
// It takes the full task list to build a dependency graph for cycle detection.
// All ID comparisons are case-insensitive via NormalizeID.
func (sm StateMachine) ValidateAddDep(tasks []Task, taskID, blockerID string) error {
	taskID = NormalizeID(taskID)
	blockerID = NormalizeID(blockerID)

	if err := smValidateChildBlockedByParent(tasks, taskID, blockerID); err != nil {
		return err
	}

	return smDetectCycle(tasks, taskID, blockerID)
}

// smValidateChildBlockedByParent rejects a dependency where a child task would be
// blocked by its own parent, which creates a deadlock with the leaf-only ready rule.
func smValidateChildBlockedByParent(tasks []Task, taskID, blockerID string) error {
	for i := range tasks {
		if NormalizeID(tasks[i].ID) == taskID && NormalizeID(tasks[i].Parent) == blockerID && blockerID != "" {
			return fmt.Errorf(
				"cannot add dependency - %s cannot be blocked by its parent %s\n(would create unworkable task due to leaf-only ready rule)",
				taskID, blockerID,
			)
		}
	}
	return nil
}

// smDetectCycle performs DFS from blockerID following blocked_by edges to
// determine if taskID is reachable. If so, a cycle would be created.
// Returns an error with the full cycle path using original (non-normalized) IDs.
func smDetectCycle(tasks []Task, taskID, blockerID string) error {
	// Self-reference: taskID blocked by itself
	if taskID == blockerID {
		return fmt.Errorf(
			"cannot add dependency - creates cycle: %s → %s",
			taskID, taskID,
		)
	}

	// Build adjacency map with normalized keys and normalized dep values.
	// Also build origID map: normalized → first original ID seen (for error messages).
	blockedByMap := make(map[string][]string, len(tasks))
	origID := make(map[string]string, len(tasks))
	for i := range tasks {
		nid := NormalizeID(tasks[i].ID)
		if _, ok := origID[nid]; !ok {
			origID[nid] = tasks[i].ID
		}
		if len(tasks[i].BlockedBy) > 0 {
			deps := make([]string, len(tasks[i].BlockedBy))
			for j, dep := range tasks[i].BlockedBy {
				nd := NormalizeID(dep)
				deps[j] = nd
				if _, ok := origID[nd]; !ok {
					origID[nd] = dep
				}
			}
			blockedByMap[nid] = deps
		}
	}

	// DFS from blockerID following blocked_by edges looking for taskID
	visited := make(map[string]bool)
	path := []string{taskID, blockerID}

	if found := smDFS(blockerID, taskID, blockedByMap, visited, &path); found {
		// Convert normalized path to original IDs for error messages.
		display := make([]string, len(path))
		for i, p := range path {
			if orig, ok := origID[p]; ok {
				display[i] = orig
			} else {
				display[i] = p
			}
		}
		return fmt.Errorf(
			"cannot add dependency - creates cycle: %s",
			strings.Join(display, " → "),
		)
	}

	return nil
}

// smDFS performs depth-first search from current following blocked_by edges,
// looking for target. It tracks the path for error reporting.
func smDFS(current, target string, blockedByMap map[string][]string, visited map[string]bool, path *[]string) bool {
	if current == target {
		return true
	}

	if visited[current] {
		return false
	}
	visited[current] = true

	for _, dep := range blockedByMap[current] {
		*path = append(*path, dep)
		if smDFS(dep, target, blockedByMap, visited, path) {
			return true
		}
		*path = (*path)[:len(*path)-1]
	}

	return false
}
