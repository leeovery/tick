package task

import "fmt"

// ApplyWithCascades applies a primary transition to target, then processes all resulting
// cascades via a queue until no further cascades are produced. Each cascaded task is
// mutated in-place and receives a TransitionRecord with Auto: true. The primary task
// receives a TransitionRecord with Auto: false.
//
// target must be a pointer to an element within the tasks slice (not a separate copy),
// so that mutations are visible both through the pointer and through the slice.
//
// A seen-map keyed by normalized task ID prevents duplicate processing. The target task
// is pre-seeded in the seen-map so it cannot be re-processed by cascades.
//
// Returns the primary TransitionResult, the full list of applied CascadeChanges, and any error.
// On error (invalid primary transition), no tasks are mutated.
func (sm StateMachine) ApplyWithCascades(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error) {
	// Rule 9: block reopen if direct parent is cancelled.
	if action == "reopen" && target.Parent != "" {
		parentNorm := NormalizeID(target.Parent)
		for i := range tasks {
			if NormalizeID(tasks[i].ID) == parentNorm {
				if tasks[i].Status == StatusCancelled {
					return TransitionResult{}, nil, fmt.Errorf("cannot reopen task under cancelled parent, reopen parent first")
				}
				break
			}
		}
	}

	// Step 1: Apply the primary transition.
	result, err := sm.Transition(target, action)
	if err != nil {
		return TransitionResult{}, nil, err
	}

	// Step 2: Record transition history on the primary task (Auto: false).
	target.Transitions = append(target.Transitions, TransitionRecord{
		From: result.OldStatus,
		To:   result.NewStatus,
		At:   target.Updated,
		Auto: false,
	})

	// Step 3: Compute initial cascade list.
	// Build a taskMap so we can look up the slice-backed pointer for Cascades calls.
	taskMap := buildTaskMap(tasks)
	targetInSlice := taskMap[NormalizeID(target.ID)]

	initialCascades := sm.Cascades(tasks, targetInSlice, action)

	// Step 4: Initialize queue, seen-map, and results.
	queue := make([]CascadeChange, len(initialCascades))
	copy(queue, initialCascades)

	seen := make(map[string]bool)
	seen[NormalizeID(target.ID)] = true

	var results []CascadeChange

	// Step 5: Process queue.
	for len(queue) > 0 {
		change := queue[0]
		queue = queue[1:]

		nid := NormalizeID(change.Task.ID)
		if seen[nid] {
			continue
		}
		seen[nid] = true

		// Apply the cascaded transition.
		cResult, cErr := sm.Transition(change.Task, change.Action)
		if cErr != nil {
			// Cascade transitions should always be valid given the rules, but skip if not.
			continue
		}

		// Record transition history with Auto: true.
		change.Task.Transitions = append(change.Task.Transitions, TransitionRecord{
			From: cResult.OldStatus,
			To:   cResult.NewStatus,
			At:   change.Task.Updated,
			Auto: true,
		})

		results = append(results, change)

		// Compute further cascades from this change.
		// Use the pointer from the tasks slice (via taskMap) for correct ancestry walking.
		changedInSlice := taskMap[nid]
		further := sm.Cascades(tasks, changedInSlice, change.Action)
		queue = append(queue, further...)
	}

	// Step 6: Return.
	return result, results, nil
}
