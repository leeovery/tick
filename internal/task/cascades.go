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
// For action "done" or "cancel" (Rule 4): walks downward through all non-terminal descendants
// using BFS, emitting a CascadeChange for each. Children already done or cancelled are skipped.
// Dependency state does not gate cascade (advisory dependency principle).
//
// For other actions, returns nil.
func (sm StateMachine) Cascades(tasks []Task, changed *Task, action string) []CascadeChange {
	switch action {
	case "start":
		return sm.cascadeUpwardStart(tasks, changed)
	case "done", "cancel":
		return sm.cascadeDownwardTerminal(tasks, changed, action)
	default:
		return nil
	}
}

// cascadeUpwardStart walks up the parent chain from changed, emitting CascadeChange entries
// for open ancestors (Rule 2). Terminal ancestors stop the walk.
func (sm StateMachine) cascadeUpwardStart(tasks []Task, changed *Task) []CascadeChange {
	taskMap := buildTaskMap(tasks)

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

// cascadeDownwardTerminal walks downward from changed via BFS, emitting CascadeChange entries
// for all non-terminal descendants (Rule 4). Terminal children are skipped.
func (sm StateMachine) cascadeDownwardTerminal(tasks []Task, changed *Task, action string) []CascadeChange {
	childrenMap := buildChildrenMap(tasks)
	targetStatus := transitions[action].to

	var changes []CascadeChange
	queue := []string{NormalizeID(changed.ID)}

	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]

		for _, child := range childrenMap[parentID] {
			if child.Status == StatusDone || child.Status == StatusCancelled {
				continue
			}
			changes = append(changes, CascadeChange{
				Task:      child,
				Action:    action,
				OldStatus: child.Status,
				NewStatus: targetStatus,
			})
			queue = append(queue, NormalizeID(child.ID))
		}
	}

	return changes
}

// buildTaskMap creates a map from normalized task ID to pointer into the slice.
func buildTaskMap(tasks []Task) map[string]*Task {
	m := make(map[string]*Task, len(tasks))
	for i := range tasks {
		m[NormalizeID(tasks[i].ID)] = &tasks[i]
	}
	return m
}

// buildChildrenMap creates a map from normalized parent ID to child task pointers.
func buildChildrenMap(tasks []Task) map[string][]*Task {
	m := make(map[string][]*Task)
	for i := range tasks {
		if tasks[i].Parent != "" {
			parentID := NormalizeID(tasks[i].Parent)
			m[parentID] = append(m[parentID], &tasks[i])
		}
	}
	return m
}
