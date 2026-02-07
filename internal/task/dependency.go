package task

import (
	"fmt"
	"strings"
)

// ValidateDependency checks whether adding newBlockedByID as a dependency of
// taskID would create an invalid state. It rejects two cases:
//   - Circular dependencies (including self-reference)
//   - Child blocked by its own direct parent
//
// The tasks slice provides the full task list for graph traversal.
// This is a pure function with no I/O.
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
	if err := checkChildBlockedByParent(tasks, taskID, newBlockedByID); err != nil {
		return err
	}

	if err := checkCycle(tasks, taskID, newBlockedByID); err != nil {
		return err
	}

	return nil
}

// ValidateDependencies validates multiple blocked_by IDs sequentially,
// returning an error on the first invalid dependency found.
func ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error {
	for _, blockedByID := range blockedByIDs {
		if err := ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return err
		}
	}
	return nil
}

// checkChildBlockedByParent rejects a dependency where a task would be
// blocked by its own direct parent. This creates an unworkable state
// due to the leaf-only ready rule.
func checkChildBlockedByParent(tasks []Task, taskID, newBlockedByID string) error {
	for _, t := range tasks {
		if t.ID == taskID && t.Parent == newBlockedByID {
			return fmt.Errorf(
				"Error: Cannot add dependency - %s cannot be blocked by its parent %s",
				taskID, newBlockedByID,
			)
		}
	}
	return nil
}

// checkCycle detects whether adding taskID -> newBlockedByID creates a cycle
// in the dependency graph. It follows blocked_by edges from newBlockedByID
// using BFS and checks if taskID is reachable. If a cycle is found, the full
// path is reconstructed and returned in the error message.
func checkCycle(tasks []Task, taskID, newBlockedByID string) error {
	// Self-reference is the simplest cycle
	if taskID == newBlockedByID {
		return fmt.Errorf(
			"Error: Cannot add dependency - creates cycle: %s \u2192 %s",
			taskID, taskID,
		)
	}

	// Build a map of taskID -> list of tasks that block it (blocked_by edges)
	blockedByMap := buildBlockedByMap(tasks)

	// BFS from newBlockedByID following blocked_by edges.
	// If we reach taskID, adding taskID -> newBlockedByID creates a cycle.
	// Track parent for path reconstruction.
	parent := make(map[string]string)
	visited := make(map[string]bool)
	queue := []string{newBlockedByID}
	visited[newBlockedByID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, blocker := range blockedByMap[current] {
			if blocker == taskID {
				// Cycle found. Reconstruct path: taskID -> newBlockedByID -> ... -> current -> taskID
				path := reconstructPath(parent, newBlockedByID, current)
				return fmt.Errorf(
					"Error: Cannot add dependency - creates cycle: %s",
					formatCyclePath(taskID, path),
				)
			}
			if !visited[blocker] {
				visited[blocker] = true
				parent[blocker] = current
				queue = append(queue, blocker)
			}
		}
	}

	return nil
}

// buildBlockedByMap creates a lookup from task ID to its blocked_by list.
func buildBlockedByMap(tasks []Task) map[string][]string {
	m := make(map[string][]string, len(tasks))
	for _, t := range tasks {
		if len(t.BlockedBy) > 0 {
			m[t.ID] = t.BlockedBy
		}
	}
	return m
}

// reconstructPath traces back from end to start using the parent map,
// returning the path as a slice from start to end (inclusive).
func reconstructPath(parent map[string]string, start, end string) []string {
	path := []string{end}
	current := end
	for current != start {
		current = parent[current]
		path = append(path, current)
	}
	// Reverse the path so it goes start -> ... -> end
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// formatCyclePath formats the cycle as "taskID -> path[0] -> path[1] -> ... -> taskID".
func formatCyclePath(taskID string, path []string) string {
	parts := make([]string, 0, len(path)+2)
	parts = append(parts, taskID)
	parts = append(parts, path...)
	parts = append(parts, taskID)
	return strings.Join(parts, " \u2192 ")
}
