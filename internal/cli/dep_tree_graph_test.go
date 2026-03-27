package cli

import (
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func makeTask(id, title string, status task.Status, blockedBy ...string) task.Task {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	return task.Task{
		ID:        id,
		Title:     title,
		Status:    status,
		Priority:  2,
		BlockedBy: blockedBy,
		Created:   now,
		Updated:   now,
	}
}

func TestBuildFullDepTree(t *testing.T) {
	t.Run("it returns empty result for project with no dependencies", func(t *testing.T) {
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 0 {
			t.Errorf("Roots = %d, want 0", len(result.Roots))
		}
		if result.Summary != "0 chains, longest: 0, 0 blocked" {
			t.Errorf("Summary = %q, want %q", result.Summary, "0 chains, longest: 0, 0 blocked")
		}
		if result.Message != "No dependencies found." {
			t.Errorf("Message = %q, want %q", result.Message, "No dependencies found.")
		}
		if result.ChainCount != 0 {
			t.Errorf("ChainCount = %d, want 0", result.ChainCount)
		}
		if result.LongestChain != 0 {
			t.Errorf("LongestChain = %d, want 0", result.LongestChain)
		}
		if result.BlockedCount != 0 {
			t.Errorf("BlockedCount = %d, want 0", result.BlockedCount)
		}
	})

	t.Run("it returns empty result when task list is empty", func(t *testing.T) {
		result := BuildFullDepTree(nil)

		if len(result.Roots) != 0 {
			t.Errorf("Roots = %d, want 0", len(result.Roots))
		}
		if result.Summary != "0 chains, longest: 0, 0 blocked" {
			t.Errorf("Summary = %q, want %q", result.Summary, "0 chains, longest: 0, 0 blocked")
		}
		if result.Message != "No dependencies found." {
			t.Errorf("Message = %q, want %q", result.Message, "No dependencies found.")
		}
		if result.ChainCount != 0 {
			t.Errorf("ChainCount = %d, want 0", result.ChainCount)
		}
		if result.LongestChain != 0 {
			t.Errorf("LongestChain = %d, want 0", result.LongestChain)
		}
		if result.BlockedCount != 0 {
			t.Errorf("BlockedCount = %d, want 0", result.BlockedCount)
		}
	})

	t.Run("it builds a single linear chain", func(t *testing.T) {
		// A blocks B blocks C: A -> B -> C
		// BlockedBy is the reverse: B blocked by A, C blocked by B
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusInProgress, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-bbb222"),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 1 {
			t.Fatalf("Roots = %d, want 1", len(result.Roots))
		}
		root := result.Roots[0]
		if root.Task.ID != "tick-aaa111" {
			t.Errorf("root ID = %q, want %q", root.Task.ID, "tick-aaa111")
		}
		if len(root.Children) != 1 {
			t.Fatalf("root children = %d, want 1", len(root.Children))
		}
		if root.Children[0].Task.ID != "tick-bbb222" {
			t.Errorf("child ID = %q, want %q", root.Children[0].Task.ID, "tick-bbb222")
		}
		if len(root.Children[0].Children) != 1 {
			t.Fatalf("grandchild count = %d, want 1", len(root.Children[0].Children))
		}
		if root.Children[0].Children[0].Task.ID != "tick-ccc333" {
			t.Errorf("grandchild ID = %q, want %q", root.Children[0].Children[0].Task.ID, "tick-ccc333")
		}
		if result.Summary != "1 chain, longest: 2, 2 blocked" {
			t.Errorf("Summary = %q, want %q", result.Summary, "1 chain, longest: 2, 2 blocked")
		}
		if result.ChainCount != 1 {
			t.Errorf("ChainCount = %d, want 1", result.ChainCount)
		}
		if result.LongestChain != 2 {
			t.Errorf("LongestChain = %d, want 2", result.LongestChain)
		}
		if result.BlockedCount != 2 {
			t.Errorf("BlockedCount = %d, want 2", result.BlockedCount)
		}
		if result.Message != "" {
			t.Errorf("Message = %q, want empty", result.Message)
		}
	})

	t.Run("it builds multiple independent chains", func(t *testing.T) {
		// Chain 1: A -> B
		// Chain 2: C -> D
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen),
			makeTask("tick-ddd444", "Task D", task.StatusOpen, "tick-ccc333"),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 2 {
			t.Fatalf("Roots = %d, want 2", len(result.Roots))
		}
		// Roots should be A and C (both block others, neither is blocked)
		rootIDs := map[string]bool{}
		for _, r := range result.Roots {
			rootIDs[r.Task.ID] = true
		}
		if !rootIDs["tick-aaa111"] {
			t.Error("expected tick-aaa111 as root")
		}
		if !rootIDs["tick-ccc333"] {
			t.Error("expected tick-ccc333 as root")
		}
		if result.Summary != "2 chains, longest: 1, 2 blocked" {
			t.Errorf("Summary = %q, want %q", result.Summary, "2 chains, longest: 1, 2 blocked")
		}
		if result.ChainCount != 2 {
			t.Errorf("ChainCount = %d, want 2", result.ChainCount)
		}
		if result.LongestChain != 1 {
			t.Errorf("LongestChain = %d, want 1", result.LongestChain)
		}
		if result.BlockedCount != 2 {
			t.Errorf("BlockedCount = %d, want 2", result.BlockedCount)
		}
	})

	t.Run("it duplicates diamond dependency without deduplication", func(t *testing.T) {
		// A blocks B and C, both B and C block D
		// Diamond: A -> B -> D, A -> C -> D
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ddd444", "Task D", task.StatusOpen, "tick-bbb222", "tick-ccc333"),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 1 {
			t.Fatalf("Roots = %d, want 1", len(result.Roots))
		}
		root := result.Roots[0]
		if root.Task.ID != "tick-aaa111" {
			t.Errorf("root ID = %q, want %q", root.Task.ID, "tick-aaa111")
		}
		if len(root.Children) != 2 {
			t.Fatalf("root children = %d, want 2", len(root.Children))
		}
		// Both B and C should have D as a child (duplicated)
		for _, child := range root.Children {
			if len(child.Children) != 1 {
				t.Errorf("child %s has %d children, want 1", child.Task.ID, len(child.Children))
				continue
			}
			if child.Children[0].Task.ID != "tick-ddd444" {
				t.Errorf("grandchild of %s = %q, want %q", child.Task.ID, child.Children[0].Task.ID, "tick-ddd444")
			}
		}
	})

	t.Run("it omits tasks with no dependency relationships", func(t *testing.T) {
		// A -> B, C has no deps
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C (no deps)", task.StatusOpen),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 1 {
			t.Fatalf("Roots = %d, want 1", len(result.Roots))
		}
		if result.Roots[0].Task.ID != "tick-aaa111" {
			t.Errorf("root ID = %q, want %q", result.Roots[0].Task.ID, "tick-aaa111")
		}
		// C should not appear anywhere in the tree
		if containsID(result.Roots, "tick-ccc333") {
			t.Error("task with no dependencies should be omitted from tree")
		}
	})

	t.Run("it handles task blocked by multiple roots", func(t *testing.T) {
		// A blocks C, B blocks C (A and B are both roots)
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-aaa111", "tick-bbb222"),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 2 {
			t.Fatalf("Roots = %d, want 2", len(result.Roots))
		}
		// C should appear as child of both A and B (duplicated)
		for _, root := range result.Roots {
			if len(root.Children) != 1 {
				t.Errorf("root %s has %d children, want 1", root.Task.ID, len(root.Children))
				continue
			}
			if root.Children[0].Task.ID != "tick-ccc333" {
				t.Errorf("child of %s = %q, want %q", root.Task.ID, root.Children[0].Task.ID, "tick-ccc333")
			}
		}
	})

	t.Run("it computes longest chain across multiple chains", func(t *testing.T) {
		// Chain 1: A -> B (length 1)
		// Chain 2: C -> D -> E (length 2)
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen),
			makeTask("tick-ddd444", "Task D", task.StatusOpen, "tick-ccc333"),
			makeTask("tick-eee555", "Task E", task.StatusOpen, "tick-ddd444"),
		}

		result := BuildFullDepTree(tasks)

		if result.Summary != "2 chains, longest: 2, 3 blocked" {
			t.Errorf("Summary = %q, want %q", result.Summary, "2 chains, longest: 2, 3 blocked")
		}
		if result.ChainCount != 2 {
			t.Errorf("ChainCount = %d, want 2", result.ChainCount)
		}
		if result.LongestChain != 2 {
			t.Errorf("LongestChain = %d, want 2", result.LongestChain)
		}
		if result.BlockedCount != 3 {
			t.Errorf("BlockedCount = %d, want 3", result.BlockedCount)
		}
	})

	t.Run("it counts blocked tasks correctly", func(t *testing.T) {
		// A -> B, A -> C: 2 blocked tasks
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-aaa111"),
		}

		result := BuildFullDepTree(tasks)

		if result.Summary != "1 chain, longest: 1, 2 blocked" {
			t.Errorf("Summary = %q, want %q", result.Summary, "1 chain, longest: 1, 2 blocked")
		}
		if result.ChainCount != 1 {
			t.Errorf("ChainCount = %d, want 1", result.ChainCount)
		}
		if result.LongestChain != 1 {
			t.Errorf("LongestChain = %d, want 1", result.LongestChain)
		}
		if result.BlockedCount != 2 {
			t.Errorf("BlockedCount = %d, want 2", result.BlockedCount)
		}
	})
}

func TestBuildFocusedDepTree(t *testing.T) {
	t.Run("it builds focused upstream tree", func(t *testing.T) {
		// A -> B -> C, focused on C
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-bbb222"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-ccc333")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Target == nil {
			t.Fatal("Target is nil")
		}
		if result.Target.ID != "tick-ccc333" {
			t.Errorf("Target.ID = %q, want %q", result.Target.ID, "tick-ccc333")
		}
		// BlockedBy: C is blocked by B, B is blocked by A
		if len(result.BlockedBy) != 1 {
			t.Fatalf("BlockedBy = %d, want 1", len(result.BlockedBy))
		}
		if result.BlockedBy[0].Task.ID != "tick-bbb222" {
			t.Errorf("BlockedBy[0].ID = %q, want %q", result.BlockedBy[0].Task.ID, "tick-bbb222")
		}
		if len(result.BlockedBy[0].Children) != 1 {
			t.Fatalf("BlockedBy[0].Children = %d, want 1", len(result.BlockedBy[0].Children))
		}
		if result.BlockedBy[0].Children[0].Task.ID != "tick-aaa111" {
			t.Errorf("upstream grandchild ID = %q, want %q", result.BlockedBy[0].Children[0].Task.ID, "tick-aaa111")
		}
		// Blocks: C blocks nothing
		if len(result.Blocks) != 0 {
			t.Errorf("Blocks = %d, want 0", len(result.Blocks))
		}
	})

	t.Run("it builds focused downstream tree", func(t *testing.T) {
		// A -> B -> C, focused on A
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-bbb222"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-aaa111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// BlockedBy: A is not blocked by anything
		if len(result.BlockedBy) != 0 {
			t.Errorf("BlockedBy = %d, want 0", len(result.BlockedBy))
		}
		// Blocks: A blocks B, B blocks C
		if len(result.Blocks) != 1 {
			t.Fatalf("Blocks = %d, want 1", len(result.Blocks))
		}
		if result.Blocks[0].Task.ID != "tick-bbb222" {
			t.Errorf("Blocks[0].ID = %q, want %q", result.Blocks[0].Task.ID, "tick-bbb222")
		}
		if len(result.Blocks[0].Children) != 1 {
			t.Fatalf("Blocks[0].Children = %d, want 1", len(result.Blocks[0].Children))
		}
		if result.Blocks[0].Children[0].Task.ID != "tick-ccc333" {
			t.Errorf("downstream grandchild ID = %q, want %q", result.Blocks[0].Children[0].Task.ID, "tick-ccc333")
		}
	})

	t.Run("it builds focused view for mid-chain task", func(t *testing.T) {
		// A -> B -> C, focused on B
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusInProgress, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-bbb222"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-bbb222")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Target.ID != "tick-bbb222" {
			t.Errorf("Target.ID = %q, want %q", result.Target.ID, "tick-bbb222")
		}
		// BlockedBy: B is blocked by A
		if len(result.BlockedBy) != 1 {
			t.Fatalf("BlockedBy = %d, want 1", len(result.BlockedBy))
		}
		if result.BlockedBy[0].Task.ID != "tick-aaa111" {
			t.Errorf("BlockedBy[0].ID = %q, want %q", result.BlockedBy[0].Task.ID, "tick-aaa111")
		}
		// Blocks: B blocks C
		if len(result.Blocks) != 1 {
			t.Fatalf("Blocks = %d, want 1", len(result.Blocks))
		}
		if result.Blocks[0].Task.ID != "tick-ccc333" {
			t.Errorf("Blocks[0].ID = %q, want %q", result.Blocks[0].Task.ID, "tick-ccc333")
		}
	})

	t.Run("it returns no dependencies for isolated task in focused mode", func(t *testing.T) {
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-aaa111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Target == nil {
			t.Fatal("Target is nil")
		}
		if result.Target.ID != "tick-aaa111" {
			t.Errorf("Target.ID = %q, want %q", result.Target.ID, "tick-aaa111")
		}
		if len(result.BlockedBy) != 0 {
			t.Errorf("BlockedBy = %d, want 0", len(result.BlockedBy))
		}
		if len(result.Blocks) != 0 {
			t.Errorf("Blocks = %d, want 0", len(result.Blocks))
		}
		if result.Message != "No dependencies." {
			t.Errorf("Message = %q, want %q", result.Message, "No dependencies.")
		}
	})

	t.Run("it returns error for nonexistent task ID in focused mode", func(t *testing.T) {
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
		}

		_, err := BuildFocusedDepTree(tasks, "tick-nonexist")
		if err == nil {
			t.Fatal("expected error for nonexistent ID")
		}
		want := `task "tick-nonexist" not found`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it duplicates diamond in focused downstream", func(t *testing.T) {
		// A blocks B and C, B and C both block D
		// Focused on A: downstream shows B->D and C->D (D duplicated)
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ddd444", "Task D", task.StatusOpen, "tick-bbb222", "tick-ccc333"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-aaa111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Blocks) != 2 {
			t.Fatalf("Blocks = %d, want 2", len(result.Blocks))
		}
		// Both B and C should have D as child
		for _, block := range result.Blocks {
			if len(block.Children) != 1 {
				t.Errorf("block %s has %d children, want 1", block.Task.ID, len(block.Children))
				continue
			}
			if block.Children[0].Task.ID != "tick-ddd444" {
				t.Errorf("child of %s = %q, want %q", block.Task.ID, block.Children[0].Task.ID, "tick-ddd444")
			}
		}
	})

	t.Run("it handles focused view with only upstream dependencies", func(t *testing.T) {
		// A -> B, focused on B (only upstream, nothing downstream)
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-bbb222")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.BlockedBy) != 1 {
			t.Fatalf("BlockedBy = %d, want 1", len(result.BlockedBy))
		}
		if result.BlockedBy[0].Task.ID != "tick-aaa111" {
			t.Errorf("BlockedBy[0].ID = %q, want %q", result.BlockedBy[0].Task.ID, "tick-aaa111")
		}
		if len(result.Blocks) != 0 {
			t.Errorf("Blocks = %d, want 0", len(result.Blocks))
		}
		if result.Message != "" {
			t.Errorf("Message = %q, want empty (has deps)", result.Message)
		}
	})

	t.Run("it handles focused view with only downstream dependencies", func(t *testing.T) {
		// A -> B, focused on A (only downstream, nothing upstream)
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-aaa111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.BlockedBy) != 0 {
			t.Errorf("BlockedBy = %d, want 0", len(result.BlockedBy))
		}
		if len(result.Blocks) != 1 {
			t.Fatalf("Blocks = %d, want 1", len(result.Blocks))
		}
		if result.Blocks[0].Task.ID != "tick-bbb222" {
			t.Errorf("Blocks[0].ID = %q, want %q", result.Blocks[0].Task.ID, "tick-bbb222")
		}
		if result.Message != "" {
			t.Errorf("Message = %q, want empty (has deps)", result.Message)
		}
	})
}

func TestCycleGuard(t *testing.T) {
	t.Run("it terminates walkDownstream with circular dependency A blocks B blocks A", func(t *testing.T) {
		// Corrupted data: A blocked by B, B blocked by A — a cycle
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen, "tick-bbb222"),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
		}

		// Full graph: no task has empty BlockedBy, so no roots.
		// But focused view exercises walkDownstream directly.
		result, err := BuildFocusedDepTree(tasks, "tick-aaa111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Blocks: A blocks B (walkDownstream). B blocks A but cycle guard stops.
		// The tree should be finite — B appears as child, but A should NOT recurse infinitely.
		if len(result.Blocks) != 1 {
			t.Fatalf("Blocks = %d, want 1", len(result.Blocks))
		}
		if result.Blocks[0].Task.ID != "tick-bbb222" {
			t.Errorf("Blocks[0].ID = %q, want %q", result.Blocks[0].Task.ID, "tick-bbb222")
		}
		// B's children should include A (since B blocks A), but A's children must stop (cycle guard)
		totalNodes := countNodes(result.Blocks)
		if totalNodes > 10 {
			t.Errorf("tree has %d nodes, expected finite tree (cycle guard should prevent unbounded growth)", totalNodes)
		}
	})

	t.Run("it terminates walkUpstream with circular dependency A blocks B blocks A", func(t *testing.T) {
		// Corrupted data: A blocked by B, B blocked by A — a cycle
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen, "tick-bbb222"),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-aaa111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// BlockedBy: A is blocked by B (walkUpstream). B is blocked by A but cycle guard stops.
		if len(result.BlockedBy) != 1 {
			t.Fatalf("BlockedBy = %d, want 1", len(result.BlockedBy))
		}
		if result.BlockedBy[0].Task.ID != "tick-bbb222" {
			t.Errorf("BlockedBy[0].ID = %q, want %q", result.BlockedBy[0].Task.ID, "tick-bbb222")
		}
		totalNodes := countNodes(result.BlockedBy)
		if totalNodes > 10 {
			t.Errorf("tree has %d nodes, expected finite tree (cycle guard should prevent unbounded growth)", totalNodes)
		}
	})

	t.Run("it terminates walkDownstream with three-node cycle", func(t *testing.T) {
		// Corrupted data: A->B->C->A cycle
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen, "tick-ccc333"),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-bbb222"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-aaa111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Downstream from A: A blocks B, B blocks C, C blocks A (cycle stops)
		totalNodes := countNodes(result.Blocks)
		if totalNodes > 10 {
			t.Errorf("downstream tree has %d nodes, expected finite tree", totalNodes)
		}
	})

	t.Run("it terminates full graph with circular dependency", func(t *testing.T) {
		// All tasks in a cycle — every task is blocked, so no root in full mode
		// But BuildFullDepTree should not hang
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen, "tick-bbb222"),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
		}

		result := BuildFullDepTree(tasks)

		// No roots (all tasks are blocked), so should get "No dependencies found."
		if len(result.Roots) != 0 {
			t.Errorf("Roots = %d, want 0 (all tasks in cycle have BlockedBy)", len(result.Roots))
		}
	})

	t.Run("it preserves acyclic diamond duplication after cycle guard addition", func(t *testing.T) {
		// Diamond: A -> B -> D, A -> C -> D (D should still be duplicated)
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ddd444", "Task D", task.StatusOpen, "tick-bbb222", "tick-ccc333"),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 1 {
			t.Fatalf("Roots = %d, want 1", len(result.Roots))
		}
		root := result.Roots[0]
		if len(root.Children) != 2 {
			t.Fatalf("root children = %d, want 2", len(root.Children))
		}
		// Both B and C should have D as child (diamond duplication preserved)
		for _, child := range root.Children {
			if len(child.Children) != 1 {
				t.Errorf("child %s has %d children, want 1 (diamond duplication)", child.Task.ID, len(child.Children))
				continue
			}
			if child.Children[0].Task.ID != "tick-ddd444" {
				t.Errorf("grandchild of %s = %q, want %q", child.Task.ID, child.Children[0].Task.ID, "tick-ddd444")
			}
		}
	})

	t.Run("it duplicates diamond node with children under both paths", func(t *testing.T) {
		// Deep diamond: A -> B -> D -> E, A -> C -> D -> E
		// D has child E. With permanent-visited, D visited via B would suppress
		// D's children when reached via C. Ancestor tracking fixes this.
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-aaa111"),
			makeTask("tick-ddd444", "Task D", task.StatusOpen, "tick-bbb222", "tick-ccc333"),
			makeTask("tick-eee555", "Task E", task.StatusOpen, "tick-ddd444"),
		}

		result := BuildFullDepTree(tasks)

		if len(result.Roots) != 1 {
			t.Fatalf("Roots = %d, want 1", len(result.Roots))
		}
		root := result.Roots[0]
		if len(root.Children) != 2 {
			t.Fatalf("root children = %d, want 2", len(root.Children))
		}
		// Both B and C should have D as child, and D should have E as child in both branches
		for _, child := range root.Children {
			if len(child.Children) != 1 {
				t.Errorf("child %s has %d children, want 1", child.Task.ID, len(child.Children))
				continue
			}
			d := child.Children[0]
			if d.Task.ID != "tick-ddd444" {
				t.Errorf("grandchild of %s = %q, want %q", child.Task.ID, d.Task.ID, "tick-ddd444")
				continue
			}
			if len(d.Children) != 1 {
				t.Errorf("D under %s has %d children, want 1 (E should appear)", child.Task.ID, len(d.Children))
				continue
			}
			if d.Children[0].Task.ID != "tick-eee555" {
				t.Errorf("great-grandchild of %s = %q, want %q", child.Task.ID, d.Children[0].Task.ID, "tick-eee555")
			}
		}
	})

	t.Run("it preserves acyclic focused view after cycle guard addition", func(t *testing.T) {
		// A -> B -> C, focused on B
		tasks := []task.Task{
			makeTask("tick-aaa111", "Task A", task.StatusOpen),
			makeTask("tick-bbb222", "Task B", task.StatusInProgress, "tick-aaa111"),
			makeTask("tick-ccc333", "Task C", task.StatusOpen, "tick-bbb222"),
		}

		result, err := BuildFocusedDepTree(tasks, "tick-bbb222")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Target.ID != "tick-bbb222" {
			t.Errorf("Target.ID = %q, want %q", result.Target.ID, "tick-bbb222")
		}
		if len(result.BlockedBy) != 1 {
			t.Fatalf("BlockedBy = %d, want 1", len(result.BlockedBy))
		}
		if result.BlockedBy[0].Task.ID != "tick-aaa111" {
			t.Errorf("BlockedBy[0].ID = %q, want %q", result.BlockedBy[0].Task.ID, "tick-aaa111")
		}
		if len(result.Blocks) != 1 {
			t.Fatalf("Blocks = %d, want 1", len(result.Blocks))
		}
		if result.Blocks[0].Task.ID != "tick-ccc333" {
			t.Errorf("Blocks[0].ID = %q, want %q", result.Blocks[0].Task.ID, "tick-ccc333")
		}
	})
}

// countNodes counts the total number of nodes in a tree (recursive).
func countNodes(nodes []DepTreeNode) int {
	count := len(nodes)
	for _, n := range nodes {
		count += countNodes(n.Children)
	}
	return count
}

// containsID checks if any node in the tree has the given ID (recursive).
func containsID(nodes []DepTreeNode, id string) bool {
	for _, n := range nodes {
		if n.Task.ID == id {
			return true
		}
		if containsID(n.Children, id) {
			return true
		}
	}
	return false
}
