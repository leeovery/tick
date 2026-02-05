package task

import (
	"fmt"
	"strings"
)

// ValidateDependency checks if adding a blocked_by relationship would create
// an invalid state: either a cycle or a child blocked by its own parent.
// Returns nil if the dependency is valid.
// Pure function - no I/O.
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
	normalizedTaskID := NormalizeID(taskID)
	normalizedBlockedByID := NormalizeID(newBlockedByID)

	// Build lookup maps for efficient access
	taskByID := make(map[string]Task)
	for _, t := range tasks {
		taskByID[NormalizeID(t.ID)] = t
	}

	// Check 1: Child blocked by parent
	// If the task's parent equals the newBlockedByID, reject
	if task, exists := taskByID[normalizedTaskID]; exists {
		if task.Parent != "" && NormalizeID(task.Parent) == normalizedBlockedByID {
			return fmt.Errorf("cannot add dependency - %s cannot be blocked by its parent %s\n       (would create unworkable task due to leaf-only ready rule)",
				normalizedTaskID, normalizedBlockedByID)
		}
	}

	// Check 2: Cycle detection via DFS
	// We need to detect if taskID is reachable from newBlockedByID by following blocked_by edges
	// If we're adding newBlockedByID to taskID's blocked_by, then:
	// - taskID will depend on newBlockedByID
	// - If there's a path from newBlockedByID back to taskID via blocked_by, we have a cycle
	//
	// The cycle exists if: newBlockedByID -> ... -> taskID (following blocked_by edges)
	// In other words: is taskID reachable from newBlockedByID?
	// Actually, we need to think about this more carefully:
	//
	// blocked_by means "X is blocked by Y" -> X depends on Y -> Y must be done first
	// So blocked_by edges point from dependent to dependency.
	//
	// If taskID's blocked_by includes newBlockedByID, that means:
	// taskID depends on newBlockedByID
	//
	// A cycle would be: A depends on B depends on ... depends on A
	// So we need to check: does newBlockedByID (or any task reachable from it via blocked_by)
	// eventually reach taskID?
	//
	// Starting from newBlockedByID, follow its blocked_by edges (what it depends on).
	// If we reach taskID, adding this dependency would create:
	// taskID -> newBlockedByID -> ... -> taskID (a cycle)

	cyclePath := detectCycle(taskByID, normalizedTaskID, normalizedBlockedByID)
	if len(cyclePath) > 0 {
		return fmt.Errorf("cannot add dependency - creates cycle: %s", strings.Join(cyclePath, " → "))
	}

	return nil
}

// detectCycle performs BFS to find if adding taskID blocked_by newBlockedByID creates a cycle.
// Returns the cycle path if found, or nil if no cycle.
//
// The cycle path format shows the dependency chain: A → B → C → A
// meaning "A is blocked by B, B is blocked by C, C is blocked by A"
func detectCycle(taskByID map[string]Task, taskID, newBlockedByID string) []string {
	// Self-reference is the simplest cycle
	if taskID == newBlockedByID {
		return []string{taskID, taskID}
	}

	// We're adding: taskID is blocked by newBlockedByID
	// A cycle exists if there's already a path from newBlockedByID back to taskID
	// via existing blocked_by relationships.
	//
	// Example: tick-a.blocked_by = [tick-b] (existing)
	// Adding: tick-b.blocked_by = [tick-a]
	// Cycle: tick-a -> tick-b -> tick-a
	//
	// We need to find who depends on taskID (i.e., who has taskID in their blocked_by)
	// and trace back. But actually, the simpler approach is to trace forward from
	// newBlockedByID following blocked_by edges until we reach taskID.
	//
	// If newBlockedByID -> ... -> taskID exists, then adding taskID -> newBlockedByID
	// creates: taskID -> newBlockedByID -> ... -> taskID

	// First check if newBlockedByID exists
	if _, exists := taskByID[newBlockedByID]; !exists {
		return nil
	}

	// BFS from newBlockedByID following blocked_by edges to find taskID
	// But we need the path to show who depends on whom, starting from
	// the task that will loop back.
	//
	// After finding newBlockedByID -> ... -> taskID, the cycle is:
	// newBlockedByID -> ... -> taskID -> newBlockedByID
	// But we want it in the format starting from a task and ending at itself.
	//
	// Let's think about it differently:
	// - Currently: newBlockedByID has blocked_by chain leading to taskID
	// - We're adding: taskID blocked by newBlockedByID
	// - The cycle means: starting from newBlockedByID, following blocked_by,
	//   we reach taskID, which (after adding) will be blocked by newBlockedByID
	//
	// The path should be: newBlockedByID -> ... -> taskID -> newBlockedByID
	// But looking at the expected output "tick-a -> tick-b -> tick-a", let's trace:
	// - tick-a.blocked_by = [tick-b] (tick-a depends on tick-b)
	// - Adding tick-b.blocked_by = [tick-a] (tick-b will depend on tick-a)
	// - taskID = tick-b, newBlockedByID = tick-a
	// - Expected: tick-a -> tick-b -> tick-a
	//
	// So the path starts from newBlockedByID (tick-a) and the chain is:
	// tick-a is blocked by tick-b (from tick-a.blocked_by = [tick-b])
	// tick-b will be blocked by tick-a (the new edge)
	// So: tick-a -> tick-b -> tick-a
	//
	// We need to find path: newBlockedByID's blocked_by chain leading to taskID
	// Path: [newBlockedByID, ..., taskID, newBlockedByID]

	type node struct {
		id   string
		path []string
	}

	visited := make(map[string]bool)
	queue := []node{{id: newBlockedByID, path: []string{newBlockedByID}}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.id] {
			continue
		}
		visited[current.id] = true

		task, exists := taskByID[current.id]
		if !exists {
			continue
		}

		for _, blockedBy := range task.BlockedBy {
			normalizedBlockedBy := NormalizeID(blockedBy)

			newPath := make([]string, len(current.path)+1)
			copy(newPath, current.path)
			newPath[len(current.path)] = normalizedBlockedBy

			if normalizedBlockedBy == taskID {
				// Found path from newBlockedByID to taskID!
				// The cycle is: newBlockedByID -> ... -> taskID -> newBlockedByID
				// Which equals: newPath (starts with newBlockedByID, ends with taskID) + newBlockedByID
				cyclePath := append(newPath, newBlockedByID)
				return cyclePath
			}

			if !visited[normalizedBlockedBy] {
				queue = append(queue, node{id: normalizedBlockedBy, path: newPath})
			}
		}
	}

	return nil
}

// ValidateDependencies validates multiple blocked_by IDs sequentially.
// Fails on first error encountered.
// Pure function - no I/O.
func ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error {
	for _, blockedByID := range blockedByIDs {
		if err := ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return err
		}
	}
	return nil
}
