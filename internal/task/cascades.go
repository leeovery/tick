package task

// CascadeChange describes a single cascaded status change computed by Cascades.
// It is pure data — Cascades does not mutate any tasks.
type CascadeChange struct {
	Task      *Task
	Action    string
	OldStatus Status
	NewStatus Status
}

// Cascades computes the cascade changes triggered by a status transition on the changed task.
// It is pure — it does not mutate any tasks. The caller is responsible for applying the changes.
//
// For action "start" (Rule 2): walks up the parent chain and emits a CascadeChange for each
// open ancestor, setting it to in_progress. Ancestors already in_progress are skipped but the
// chain continues. Terminal ancestors (done/cancelled) stop the chain.
//
// For other actions, returns nil (not yet implemented).
func (sm StateMachine) Cascades(tasks []Task, changed *Task, action string) []CascadeChange {
	if action != "start" {
		return nil
	}

	// Build map of normalized task ID to *Task (pointer into the slice).
	taskMap := make(map[string]*Task, len(tasks))
	for i := range tasks {
		taskMap[NormalizeID(tasks[i].ID)] = &tasks[i]
	}

	var changes []CascadeChange
	current := changed

	for current.Parent != "" {
		parent, ok := taskMap[NormalizeID(current.Parent)]
		if !ok {
			break
		}

		switch parent.Status {
		case StatusOpen:
			changes = append(changes, CascadeChange{
				Task:      parent,
				Action:    "start",
				OldStatus: StatusOpen,
				NewStatus: StatusInProgress,
			})
		case StatusInProgress:
			// Already in_progress — skip but continue walking up
		default:
			// Terminal state (done/cancelled) — stop the chain
			return changes
		}

		current = parent
	}

	return changes
}
