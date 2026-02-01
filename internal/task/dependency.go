package task

import (
	"fmt"
	"strings"
)

// ValidateDependency checks that adding newBlockedByID as a blocker of taskID
// does not create a cycle or violate the child-blocked-by-parent rule.
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
	// Self-reference check.
	if taskID == newBlockedByID {
		return fmt.Errorf("task cannot be blocked by itself")
	}

	// Build lookup maps.
	taskMap := make(map[string]*Task, len(tasks))
	for i := range tasks {
		taskMap[tasks[i].ID] = &tasks[i]
	}

	// Child-blocked-by-parent check.
	t := taskMap[taskID]
	if t != nil && t.Parent == newBlockedByID {
		return fmt.Errorf("Cannot add dependency — %s cannot be blocked by its parent %s", taskID, newBlockedByID)
	}

	// Cycle detection: starting from newBlockedByID, follow blocked_by edges.
	// If we can reach taskID, adding this edge would create a cycle.
	// We track the path for error reporting.
	if path := detectCycle(taskMap, taskID, newBlockedByID); path != nil {
		return fmt.Errorf("Cannot add dependency — creates cycle: %s", formatPath(path))
	}

	return nil
}

// ValidateDependencies validates multiple blocked_by IDs sequentially,
// failing on the first error.
func ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error {
	for _, depID := range blockedByIDs {
		if err := ValidateDependency(tasks, taskID, depID); err != nil {
			return err
		}
	}
	return nil
}

// detectCycle performs BFS from startID following blocked_by edges.
// Returns the cycle path if targetID is reachable, nil otherwise.
// The path represents: targetID is blocked by ... is blocked by startID,
// and we're about to add startID blocked by targetID.
func detectCycle(taskMap map[string]*Task, targetID, startID string) []string {
	type entry struct {
		id   string
		path []string
	}

	queue := []entry{{id: startID, path: []string{startID}}}
	visited := map[string]bool{startID: true}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		t := taskMap[current.id]
		if t == nil {
			continue
		}

		for _, blockerID := range t.BlockedBy {
			if blockerID == targetID {
				// Found cycle: path + targetID
				return append(current.path, targetID)
			}
			if !visited[blockerID] {
				visited[blockerID] = true
				newPath := make([]string, len(current.path)+1)
				copy(newPath, current.path)
				newPath[len(current.path)] = blockerID
				queue = append(queue, entry{id: blockerID, path: newPath})
			}
		}
	}

	return nil
}

// formatPath formats a cycle path as "a → b → c → a".
func formatPath(path []string) string {
	return strings.Join(path, " → ")
}
