package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/task"
)

// buildBlocksIndex inverts the BlockedBy relationships into a "blocks" direction map.
// For each task T with BlockedBy = [X, Y], the result maps X -> [T.ID] and Y -> [T.ID].
func buildBlocksIndex(tasks []task.Task) map[string][]string {
	blocks := make(map[string][]string)
	for _, t := range tasks {
		for _, dep := range t.BlockedBy {
			blocks[dep] = append(blocks[dep], t.ID)
		}
	}
	return blocks
}

// buildTaskIndex creates a lookup map from task ID to task.
func buildTaskIndex(tasks []task.Task) map[string]task.Task {
	idx := make(map[string]task.Task, len(tasks))
	for _, t := range tasks {
		idx[t.ID] = t
	}
	return idx
}

// toDepTreeTask converts a task.Task to the minimal DepTreeTask for rendering.
func toDepTreeTask(t task.Task) DepTreeTask {
	return DepTreeTask{
		ID:     t.ID,
		Title:  t.Title,
		Status: string(t.Status),
	}
}

// walkDownstream recursively walks the "blocks" direction from a given task ID,
// producing a tree of DepTreeNode. No deduplication — diamond dependencies are duplicated.
func walkDownstream(id string, blocks map[string][]string, taskIdx map[string]task.Task) []DepTreeNode {
	children, ok := blocks[id]
	if !ok {
		return nil
	}
	var nodes []DepTreeNode
	for _, childID := range children {
		t, exists := taskIdx[childID]
		if !exists {
			continue
		}
		node := DepTreeNode{
			Task:     toDepTreeTask(t),
			Children: walkDownstream(childID, blocks, taskIdx),
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// walkUpstream recursively walks the "blocked by" direction from a given task ID,
// producing a tree of DepTreeNode. No deduplication — diamond dependencies are duplicated.
func walkUpstream(id string, taskIdx map[string]task.Task) []DepTreeNode {
	t, exists := taskIdx[id]
	if !exists {
		return nil
	}
	if len(t.BlockedBy) == 0 {
		return nil
	}
	var nodes []DepTreeNode
	for _, depID := range t.BlockedBy {
		dep, exists := taskIdx[depID]
		if !exists {
			continue
		}
		node := DepTreeNode{
			Task:     toDepTreeTask(dep),
			Children: walkUpstream(depID, taskIdx),
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// longestPath computes the longest path (in edges) from a root node downward.
func longestPath(node DepTreeNode) int {
	if len(node.Children) == 0 {
		return 0
	}
	max := 0
	for _, child := range node.Children {
		depth := 1 + longestPath(child)
		if depth > max {
			max = depth
		}
	}
	return max
}

// BuildFullDepTree builds a dependency tree for all tasks that participate in dependencies.
// Tasks with no dependency relationships (neither blocking nor blocked) are omitted.
// Returns a DepTreeResult with Roots (top-level trees) and a Summary line.
func BuildFullDepTree(tasks []task.Task) DepTreeResult {
	blocks := buildBlocksIndex(tasks)
	taskIdx := buildTaskIndex(tasks)

	// Identify tasks that participate in dependency relationships
	participants := make(map[string]bool)
	for _, t := range tasks {
		if len(t.BlockedBy) > 0 {
			participants[t.ID] = true
			for _, dep := range t.BlockedBy {
				participants[dep] = true
			}
		}
	}

	// Roots are tasks that block others but are not themselves blocked
	var roots []DepTreeNode
	for _, t := range tasks {
		if !participants[t.ID] {
			continue
		}
		if len(t.BlockedBy) > 0 {
			continue
		}
		// This task participates and is not blocked — it's a root
		if _, blocksOthers := blocks[t.ID]; blocksOthers {
			node := DepTreeNode{
				Task:     toDepTreeTask(t),
				Children: walkDownstream(t.ID, blocks, taskIdx),
			}
			roots = append(roots, node)
		}
	}

	// Count blocked tasks (tasks with at least one BlockedBy entry)
	blocked := 0
	for _, t := range tasks {
		if len(t.BlockedBy) > 0 {
			blocked++
		}
	}

	// Count connected components (chains) using union-find over participants
	chains := countChains(tasks, participants)

	// Compute longest path across all roots
	longest := 0
	for _, root := range roots {
		depth := longestPath(root)
		if depth > longest {
			longest = depth
		}
	}

	// Build summary
	chainWord := "chains"
	if chains == 1 {
		chainWord = "chain"
	}
	summary := fmt.Sprintf("%d %s, longest: %d, %d blocked", chains, chainWord, longest, blocked)

	var message string
	if len(roots) == 0 {
		message = "No dependencies found."
	}

	return DepTreeResult{
		Roots:   roots,
		Summary: summary,
		Message: message,
	}
}

// countChains counts connected components among tasks that participate in dependencies.
func countChains(tasks []task.Task, participants map[string]bool) int {
	if len(participants) == 0 {
		return 0
	}

	// Build adjacency list (undirected) for participants only
	adj := make(map[string][]string)
	for _, t := range tasks {
		if !participants[t.ID] {
			continue
		}
		for _, dep := range t.BlockedBy {
			if participants[dep] {
				adj[t.ID] = append(adj[t.ID], dep)
				adj[dep] = append(adj[dep], t.ID)
			}
		}
	}

	visited := make(map[string]bool)
	components := 0

	for id := range participants {
		if visited[id] {
			continue
		}
		components++
		// BFS to mark all reachable nodes
		queue := []string{id}
		visited[id] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, neighbor := range adj[cur] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return components
}

// BuildFocusedDepTree builds a dependency tree focused on a specific task,
// showing both upstream (blocked by) and downstream (blocks) transitive dependencies.
// Returns an error if the target task ID is not found.
func BuildFocusedDepTree(tasks []task.Task, targetID string) (DepTreeResult, error) {
	taskIdx := buildTaskIndex(tasks)

	target, exists := taskIdx[targetID]
	if !exists {
		return DepTreeResult{}, fmt.Errorf("task %q not found", targetID)
	}

	blocks := buildBlocksIndex(tasks)

	targetDTT := toDepTreeTask(target)
	blockedBy := walkUpstream(targetID, taskIdx)
	downstream := walkDownstream(targetID, blocks, taskIdx)

	var message string
	if len(blockedBy) == 0 && len(downstream) == 0 {
		message = "No dependencies."
	}

	return DepTreeResult{
		Target:    &targetDTT,
		BlockedBy: blockedBy,
		Blocks:    downstream,
		Message:   message,
	}, nil
}
