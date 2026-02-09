package task

import (
	"fmt"
	"strings"
)

// ValidateDependency checks whether adding a blocked_by edge from taskID to
// newBlockedByID would create a circular dependency or violate the
// child-blocked-by-parent rule. It operates on an in-memory slice of tasks
// and performs no I/O.
//
// It returns an error if:
//   - taskID equals newBlockedByID (self-reference)
//   - the task's parent equals newBlockedByID (child blocked by parent)
//   - following blocked_by edges from newBlockedByID reaches taskID (cycle)
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
	normTaskID := NormalizeID(taskID)
	normNewDep := NormalizeID(newBlockedByID)

	// Self-reference check.
	if normTaskID == normNewDep {
		return fmt.Errorf("Cannot add dependency - creates cycle: %s \u2192 %s", taskID, taskID)
	}

	// Build lookup maps in a single pass over tasks:
	//   origID:      normalized ID -> original ID (for error messages)
	//   blockedByMap: normalized ID -> normalized blocked_by IDs (for BFS)
	origID := make(map[string]string, len(tasks))
	blockedByMap := make(map[string][]string, len(tasks))
	var taskParent string
	for _, t := range tasks {
		nid := NormalizeID(t.ID)
		origID[nid] = t.ID
		if nid == normTaskID && t.Parent != "" {
			taskParent = NormalizeID(t.Parent)
		}
		for _, dep := range t.BlockedBy {
			blockedByMap[nid] = append(blockedByMap[nid], NormalizeID(dep))
		}
	}

	// Child-blocked-by-parent check.
	if taskParent == normNewDep {
		return fmt.Errorf(
			"Cannot add dependency - %s cannot be blocked by its parent %s (would create unworkable task due to leaf-only ready rule)",
			taskID, newBlockedByID,
		)
	}

	// Cycle detection via BFS from newBlockedByID following blocked_by edges.
	// If we can reach taskID, adding this edge would create a cycle.
	type node struct {
		id   string
		path []string // path of original IDs from newBlockedByID to this node
	}

	queue := []node{{id: normNewDep, path: []string{getOrigID(origID, newBlockedByID, normNewDep)}}}
	visited := map[string]bool{normNewDep: true}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range blockedByMap[current.id] {
			if dep == normTaskID {
				// Cycle found. Build the full path: taskID -> ... -> taskID
				cyclePath := append([]string{taskID}, current.path...)
				cyclePath = append(cyclePath, dep)
				// Use original IDs from the task list for the final node.
				cyclePath[len(cyclePath)-1] = getOrigID(origID, taskID, normTaskID)
				return fmt.Errorf(
					"Cannot add dependency - creates cycle: %s",
					strings.Join(cyclePath, " \u2192 "),
				)
			}
			if !visited[dep] {
				visited[dep] = true
				newPath := make([]string, len(current.path))
				copy(newPath, current.path)
				newPath = append(newPath, getOrigID(origID, dep, dep))
				queue = append(queue, node{id: dep, path: newPath})
			}
		}
	}

	return nil
}

// ValidateDependencies validates a batch of blocked_by IDs for a task,
// checking each sequentially and failing on the first error. It performs
// no I/O.
func ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error {
	for _, depID := range blockedByIDs {
		if err := ValidateDependency(tasks, taskID, depID); err != nil {
			return err
		}
	}
	return nil
}

// getOrigID returns the original (non-normalized) ID for display purposes.
// It prefers the ID stored in the origID map; if not found, falls back to
// the provided fallback.
func getOrigID(origID map[string]string, fallback, normKey string) string {
	if orig, ok := origID[normKey]; ok {
		return orig
	}
	return fallback
}
