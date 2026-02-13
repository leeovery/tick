package doctor

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// DependencyCycleCheck validates that no circular dependency chains exist in the
// task graph. It uses DFS with three-color marking to detect all cycles.
// Self-references are excluded (handled by SelfReferentialDepCheck).
// It is read-only and never modifies the file.
type DependencyCycleCheck struct{}

// Run executes the dependency cycle check. It reads the tick directory path from
// the context (via TickDirKey), parses task relationships, builds a dependency
// graph, and runs DFS with three-color marking to detect cycles. Returns a single
// passing result if no cycles are found, or one failing result per unique cycle.
func (c *DependencyCycleCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)

	tasks, err := getTaskRelationships(ctx, tickDir)
	if err != nil {
		return []CheckResult{{
			Name:       "Dependency cycles",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}

	// Build set of known task IDs.
	knownIDs := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		knownIDs[task.ID] = struct{}{}
	}

	// Build adjacency list, excluding self-references and orphaned targets.
	adj := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		for _, depID := range task.BlockedBy {
			if depID == task.ID {
				continue
			}
			if _, exists := knownIDs[depID]; !exists {
				continue
			}
			adj[task.ID] = append(adj[task.ID], depID)
		}
	}

	// DFS with three-color marking.
	const (
		white = 0 // unvisited
		gray  = 1 // in progress
		black = 2 // done
	)

	color := make(map[string]int, len(tasks))
	for _, task := range tasks {
		color[task.ID] = white
	}

	seen := make(map[string]struct{})
	var cycles [][]string
	var path []string

	var dfs func(node string)
	dfs = func(node string) {
		color[node] = gray
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if color[neighbor] == gray {
				// Found a cycle. Extract it from the path.
				cycle := extractCycle(path, neighbor)
				normalized := normalizeCycle(cycle)
				key := strings.Join(normalized, ",")
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					cycles = append(cycles, normalized)
				}
			} else if color[neighbor] == white {
				dfs(neighbor)
			}
		}

		path = path[:len(path)-1]
		color[node] = black
	}

	// Iterate in sorted order for determinism.
	sortedIDs := make([]string, 0, len(knownIDs))
	for id := range knownIDs {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)

	for _, id := range sortedIDs {
		if color[id] == white {
			dfs(id)
		}
	}

	if len(cycles) == 0 {
		return []CheckResult{{
			Name:   "Dependency cycles",
			Passed: true,
		}}
	}

	// Sort cycles for deterministic output.
	sort.Slice(cycles, func(i, j int) bool {
		return strings.Join(cycles[i], ",") < strings.Join(cycles[j], ",")
	})

	var results []CheckResult
	for _, cycle := range cycles {
		// Format: first node repeated at end to show cycle closing.
		parts := make([]string, len(cycle)+1)
		copy(parts, cycle)
		parts[len(cycle)] = cycle[0]

		results = append(results, CheckResult{
			Name:       "Dependency cycles",
			Passed:     false,
			Severity:   SeverityError,
			Details:    fmt.Sprintf("Dependency cycle: %s", strings.Join(parts, " \u2192 ")),
			Suggestion: "Manual fix required",
		})
	}

	return results
}

// extractCycle extracts the cycle portion from the DFS path when a back-edge
// to a gray node is found. The cycle starts at the gray node's position in the
// path through the current node.
func extractCycle(path []string, target string) []string {
	for i, node := range path {
		if node == target {
			cycle := make([]string, len(path)-i)
			copy(cycle, path[i:])
			return cycle
		}
	}
	return nil
}

// normalizeCycle rotates the cycle so the lexicographically smallest ID is first.
func normalizeCycle(cycle []string) []string {
	if len(cycle) == 0 {
		return cycle
	}

	minIdx := 0
	for i, id := range cycle {
		if id < cycle[minIdx] {
			minIdx = i
		}
	}

	normalized := make([]string, len(cycle))
	for i := range cycle {
		normalized[i] = cycle[(minIdx+i)%len(cycle)]
	}
	return normalized
}
