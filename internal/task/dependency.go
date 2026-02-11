package task

import (
	"fmt"
	"strings"
)

// ValidateDependency checks that adding newBlockedByID as a blocker of taskID
// does not create a circular dependency or a child-blocked-by-parent relationship.
// It takes the full task list to build a dependency graph for cycle detection.
// All ID comparisons are case-insensitive via NormalizeID.
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
	taskID = NormalizeID(taskID)
	newBlockedByID = NormalizeID(newBlockedByID)

	if err := validateChildBlockedByParent(tasks, taskID, newBlockedByID); err != nil {
		return err
	}

	return detectCycle(tasks, taskID, newBlockedByID)
}

// ValidateDependencies validates multiple blocked_by IDs for a single task,
// checking each sequentially and failing on the first error.
func ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error {
	for _, blockedByID := range blockedByIDs {
		if err := ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return err
		}
	}
	return nil
}

// validateChildBlockedByParent rejects a dependency where a child task would be
// blocked by its own parent, which creates a deadlock with the leaf-only ready rule.
func validateChildBlockedByParent(tasks []Task, taskID, newBlockedByID string) error {
	for i := range tasks {
		if NormalizeID(tasks[i].ID) == taskID && NormalizeID(tasks[i].Parent) == newBlockedByID && newBlockedByID != "" {
			return fmt.Errorf(
				"cannot add dependency - %s cannot be blocked by its parent %s\n(would create unworkable task due to leaf-only ready rule)",
				taskID, newBlockedByID,
			)
		}
	}
	return nil
}

// detectCycle performs DFS from newBlockedByID following blocked_by edges to
// determine if taskID is reachable. If so, a cycle would be created.
// Returns an error with the full cycle path using original (non-normalized) IDs.
func detectCycle(tasks []Task, taskID, newBlockedByID string) error {
	// Self-reference: taskID blocked by itself
	if taskID == newBlockedByID {
		return fmt.Errorf(
			"cannot add dependency - creates cycle: %s \u2192 %s",
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

	// DFS from newBlockedByID following blocked_by edges looking for taskID
	visited := make(map[string]bool)
	path := []string{taskID, newBlockedByID}

	if found := dfs(newBlockedByID, taskID, blockedByMap, visited, &path); found {
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
			strings.Join(display, " \u2192 "),
		)
	}

	return nil
}

// dfs performs depth-first search from current following blocked_by edges,
// looking for target. It tracks the path for error reporting.
func dfs(current, target string, blockedByMap map[string][]string, visited map[string]bool, path *[]string) bool {
	if current == target {
		return true
	}

	if visited[current] {
		return false
	}
	visited[current] = true

	for _, dep := range blockedByMap[current] {
		*path = append(*path, dep)
		if dfs(dep, target, blockedByMap, visited, path) {
			return true
		}
		*path = (*path)[:len(*path)-1]
	}

	return false
}
