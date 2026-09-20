package cli

import (
	"fmt"
	"slices"

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
// The ancestors set tracks the current path to prevent infinite recursion on cycles.
// Nodes are removed from ancestors after processing so the same node can appear via different paths.
func walkDownstream(id string, blocks map[string][]string, taskIdx map[string]task.Task, ancestors map[string]bool) []DepTreeNode {
	if ancestors[id] {
		return nil
	}
	ancestors[id] = true

	children, ok := blocks[id]
	if !ok {
		delete(ancestors, id)
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
			Children: walkDownstream(childID, blocks, taskIdx, ancestors),
		}
		nodes = append(nodes, node)
	}
	delete(ancestors, id)
	return nodes
}

// walkUpstream recursively walks the "blocked by" direction from a given task ID,
// producing a tree of DepTreeNode. No deduplication — diamond dependencies are duplicated.
// The ancestors set tracks the current path to prevent infinite recursion on cycles.
// Nodes are removed from ancestors after processing so the same node can appear via different paths.
func walkUpstream(id string, taskIdx map[string]task.Task, ancestors map[string]bool) []DepTreeNode {
	if ancestors[id] {
		return nil
	}
	ancestors[id] = true

	t, exists := taskIdx[id]
	if !exists {
		delete(ancestors, id)
		return nil
	}
	if len(t.BlockedBy) == 0 {
		delete(ancestors, id)
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
			Children: walkUpstream(depID, taskIdx, ancestors),
		}
		nodes = append(nodes, node)
	}
	delete(ancestors, id)
	return nodes
}

// longestPath computes the longest path (in edges) from a root node downward.
func longestPath(node DepTreeNode) int {
	deepest := 0
	for _, child := range node.Children {
		deepest = max(deepest, 1+longestPath(child))
	}
	return deepest
}

// BuildFullDepTree builds a dependency tree for all tasks that participate in dependencies.
// Tasks with no dependency relationships (neither blocking nor blocked) are omitted.
func BuildFullDepTree(tasks []task.Task) DepTreeResult {
	blocks := buildBlocksIndex(tasks)
	taskIdx := buildTaskIndex(tasks)

	orderedParticipants, participants := collectParticipants(tasks)

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
				Children: walkDownstream(t.ID, blocks, taskIdx, make(map[string]bool)),
			}
			roots = append(roots, node)
		}
	}

	emitted := make(map[string]bool)
	collectTreeIDs(roots, emitted)
	unrooted := buildSeededTrees(orderedParticipants, emitted, blocks, taskIdx)

	// Count blocked tasks (tasks with at least one BlockedBy entry)
	blocked := 0
	for _, t := range tasks {
		if len(t.BlockedBy) > 0 {
			blocked++
		}
	}

	chains := countChains(tasks, participants)

	trees := slices.Concat(roots, unrooted)
	longest := 0
	for _, tree := range trees {
		longest = max(longest, longestPath(tree))
	}

	// Build summary
	chainWord := "chains"
	if chains == 1 {
		chainWord = "chain"
	}
	summary := fmt.Sprintf("%d %s, longest: %d, %d blocked", chains, chainWord, longest, blocked)

	var message string
	if len(trees) == 0 {
		message = "No dependencies found."
	}

	return DepTreeResult{
		Trees:        trees,
		Edges:        collectStoredEdges(tasks),
		Summary:      summary,
		ChainCount:   chains,
		LongestChain: longest,
		BlockedCount: blocked,
		Message:      message,
	}
}

// collectStoredEdges returns one edge per BlockedBy entry, tasks in slice order and each
// task's blockers in stored order.
func collectStoredEdges(tasks []task.Task) []DepTreeEdge {
	var edges []DepTreeEdge
	for _, t := range tasks {
		for _, dep := range t.BlockedBy {
			edges = append(edges, DepTreeEdge{From: dep, To: t.ID})
		}
	}
	return edges
}

// collectParticipants returns the IDs of every task that participates in a dependency
// relationship, in first-seen order, alongside the same IDs as a set. A blocker ID that
// no task record matches participates like any other.
func collectParticipants(tasks []task.Task) ([]string, map[string]bool) {
	var ordered []string
	seen := make(map[string]bool)
	add := func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		ordered = append(ordered, id)
	}
	for _, t := range tasks {
		if len(t.BlockedBy) == 0 {
			continue
		}
		add(t.ID)
		for _, dep := range t.BlockedBy {
			add(dep)
		}
	}
	return ordered, seen
}

// collectTreeIDs adds the ID of every node in the given trees to seen.
func collectTreeIDs(nodes []DepTreeNode, seen map[string]bool) {
	for _, n := range nodes {
		seen[n.Task.ID] = true
		collectTreeIDs(n.Children, seen)
	}
}

// depTreeMissingStatus is the status carried by a dependency participant no task record matches.
const depTreeMissingStatus = "missing"

// buildSeededTrees seeds a downstream walk from each participant the walk from the roots
// left unemitted, so the edges of a cycle or of a dangling blocker still reach the output.
// Participants that block nothing need no seed: each is reached as the target of a blocker's edge.
func buildSeededTrees(participants []string, emitted map[string]bool, blocks map[string][]string, taskIdx map[string]task.Task) []DepTreeNode {
	var unrooted []DepTreeNode
	for _, id := range participants {
		if emitted[id] || len(blocks[id]) == 0 {
			continue
		}
		node := DepTreeNode{
			Task:     DepTreeTask{ID: id, Status: depTreeMissingStatus},
			Children: walkDownstream(id, blocks, taskIdx, make(map[string]bool)),
		}
		if t, exists := taskIdx[id]; exists {
			node.Task = toDepTreeTask(t)
		}
		unrooted = append(unrooted, node)
		emitted[id] = true
		collectTreeIDs(node.Children, emitted)
	}
	return unrooted
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
	blockedBy := walkUpstream(targetID, taskIdx, make(map[string]bool))
	downstream := walkDownstream(targetID, blocks, taskIdx, make(map[string]bool))

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
