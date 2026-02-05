package task

import (
	"strings"
	"testing"
)

func TestValidateDependency(t *testing.T) {
	// Helper to create tasks with blocked_by relationships
	makeTask := func(id string, blockedBy ...string) Task {
		return Task{
			ID:        id,
			Title:     "Task " + id,
			Status:    StatusOpen,
			Priority:  2,
			BlockedBy: blockedBy,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
		}
	}

	// Helper to create task with parent
	makeTaskWithParent := func(id, parent string, blockedBy ...string) Task {
		task := makeTask(id, blockedBy...)
		task.Parent = parent
		return task
	}

	t.Run("it allows valid dependency between unrelated tasks", func(t *testing.T) {
		tasks := []Task{
			makeTask("tick-aaaaaa"),
			makeTask("tick-bbbbbb"),
		}

		err := ValidateDependency(tasks, "tick-aaaaaa", "tick-bbbbbb")
		if err != nil {
			t.Errorf("expected no error for valid dependency, got: %v", err)
		}
	})

	t.Run("it rejects direct self-reference", func(t *testing.T) {
		tasks := []Task{
			makeTask("tick-aaaaaa"),
		}

		err := ValidateDependency(tasks, "tick-aaaaaa", "tick-aaaaaa")
		if err == nil {
			t.Fatal("expected error for self-reference")
		}
		// Self-reference creates a trivial cycle
		if !strings.Contains(err.Error(), "creates cycle") {
			t.Errorf("error should mention cycle, got: %v", err)
		}
	})

	t.Run("it rejects 2-node cycle with path", func(t *testing.T) {
		// tick-a is blocked by tick-b, now trying to add tick-b blocked by tick-a
		tasks := []Task{
			makeTask("tick-aaaaaa", "tick-bbbbbb"), // tick-a blocked by tick-b
			makeTask("tick-bbbbbb"),
		}

		err := ValidateDependency(tasks, "tick-bbbbbb", "tick-aaaaaa")
		if err == nil {
			t.Fatal("expected error for 2-node cycle")
		}
		expected := "cannot add dependency - creates cycle: tick-aaaaaa → tick-bbbbbb → tick-aaaaaa"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it rejects 3+ node cycle with full path", func(t *testing.T) {
		// tick-a -> tick-b -> tick-c, trying to add tick-c -> tick-a
		tasks := []Task{
			makeTask("tick-aaaaaa", "tick-bbbbbb"), // tick-a blocked by tick-b
			makeTask("tick-bbbbbb", "tick-cccccc"), // tick-b blocked by tick-c
			makeTask("tick-cccccc"),
		}

		err := ValidateDependency(tasks, "tick-cccccc", "tick-aaaaaa")
		if err == nil {
			t.Fatal("expected error for 3-node cycle")
		}
		expected := "cannot add dependency - creates cycle: tick-aaaaaa → tick-bbbbbb → tick-cccccc → tick-aaaaaa"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it rejects child blocked by own parent", func(t *testing.T) {
		// tick-parent is parent of tick-child
		tasks := []Task{
			makeTask("tick-parent"),
			makeTaskWithParent("tick-child1", "tick-parent"),
		}

		err := ValidateDependency(tasks, "tick-child1", "tick-parent")
		if err == nil {
			t.Fatal("expected error for child blocked by parent")
		}
		expected := "cannot add dependency - tick-child1 cannot be blocked by its parent tick-parent\n       (would create unworkable task due to leaf-only ready rule)"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it allows parent blocked by own child", func(t *testing.T) {
		tasks := []Task{
			makeTask("tick-parent"),
			makeTaskWithParent("tick-child1", "tick-parent"),
		}

		err := ValidateDependency(tasks, "tick-parent", "tick-child1")
		if err != nil {
			t.Errorf("parent blocked by child should be allowed, got: %v", err)
		}
	})

	t.Run("it allows sibling dependencies", func(t *testing.T) {
		tasks := []Task{
			makeTask("tick-parent"),
			makeTaskWithParent("tick-child1", "tick-parent"),
			makeTaskWithParent("tick-child2", "tick-parent"),
		}

		err := ValidateDependency(tasks, "tick-child1", "tick-child2")
		if err != nil {
			t.Errorf("sibling dependency should be allowed, got: %v", err)
		}
	})

	t.Run("it allows cross-hierarchy dependencies", func(t *testing.T) {
		tasks := []Task{
			makeTask("tick-parent1"),
			makeTask("tick-parent2"),
			makeTaskWithParent("tick-child1", "tick-parent1"),
			makeTaskWithParent("tick-child2", "tick-parent2"),
		}

		err := ValidateDependency(tasks, "tick-child1", "tick-child2")
		if err != nil {
			t.Errorf("cross-hierarchy dependency should be allowed, got: %v", err)
		}
	})

	t.Run("it returns cycle path format: tick-a → tick-b → tick-a", func(t *testing.T) {
		// 4-node cycle to verify full path reconstruction
		tasks := []Task{
			makeTask("tick-aaaaaa", "tick-bbbbbb"),
			makeTask("tick-bbbbbb", "tick-cccccc"),
			makeTask("tick-cccccc", "tick-dddddd"),
			makeTask("tick-dddddd"),
		}

		err := ValidateDependency(tasks, "tick-dddddd", "tick-aaaaaa")
		if err == nil {
			t.Fatal("expected error for 4-node cycle")
		}
		expected := "cannot add dependency - creates cycle: tick-aaaaaa → tick-bbbbbb → tick-cccccc → tick-dddddd → tick-aaaaaa"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it detects cycle through existing multi-hop chain", func(t *testing.T) {
		// Existing chain: tick-x -> tick-y -> tick-z
		// Adding tick-z -> tick-x creates cycle
		tasks := []Task{
			makeTask("tick-xxxxxx", "tick-yyyyyy"),
			makeTask("tick-yyyyyy", "tick-zzzzzz"),
			makeTask("tick-zzzzzz"),
		}

		err := ValidateDependency(tasks, "tick-zzzzzz", "tick-xxxxxx")
		if err == nil {
			t.Fatal("expected error for cycle through existing chain")
		}
		if !strings.Contains(err.Error(), "creates cycle") {
			t.Errorf("error should mention cycle, got: %v", err)
		}
	})
}

func TestValidateDependencies(t *testing.T) {
	makeTask := func(id string, blockedBy ...string) Task {
		return Task{
			ID:        id,
			Title:     "Task " + id,
			Status:    StatusOpen,
			Priority:  2,
			BlockedBy: blockedBy,
			Created:   "2026-01-19T10:00:00Z",
			Updated:   "2026-01-19T10:00:00Z",
		}
	}

	t.Run("it validates multiple blocked_by IDs, fails on first error", func(t *testing.T) {
		// tick-a -> tick-b, trying to add tick-b blocked by both tick-c and tick-a
		tasks := []Task{
			makeTask("tick-aaaaaa", "tick-bbbbbb"),
			makeTask("tick-bbbbbb"),
			makeTask("tick-cccccc"),
		}

		// tick-cccccc is valid, tick-aaaaaa would create cycle
		// If validation is sequential, it should succeed for tick-cccccc then fail for tick-aaaaaa
		blockedByIDs := []string{"tick-cccccc", "tick-aaaaaa"}

		err := ValidateDependencies(tasks, "tick-bbbbbb", blockedByIDs)
		if err == nil {
			t.Fatal("expected error for cycle in batch validation")
		}
		// Should fail on tick-aaaaaa (second ID)
		if !strings.Contains(err.Error(), "creates cycle") {
			t.Errorf("error should mention cycle, got: %v", err)
		}
	})

	t.Run("it succeeds when all dependencies are valid", func(t *testing.T) {
		tasks := []Task{
			makeTask("tick-aaaaaa"),
			makeTask("tick-bbbbbb"),
			makeTask("tick-cccccc"),
		}

		err := ValidateDependencies(tasks, "tick-aaaaaa", []string{"tick-bbbbbb", "tick-cccccc"})
		if err != nil {
			t.Errorf("expected no error for valid dependencies, got: %v", err)
		}
	})

	t.Run("it returns empty error for empty blocked_by list", func(t *testing.T) {
		tasks := []Task{
			makeTask("tick-aaaaaa"),
		}

		err := ValidateDependencies(tasks, "tick-aaaaaa", nil)
		if err != nil {
			t.Errorf("expected no error for empty blocked_by list, got: %v", err)
		}

		err = ValidateDependencies(tasks, "tick-aaaaaa", []string{})
		if err != nil {
			t.Errorf("expected no error for empty blocked_by slice, got: %v", err)
		}
	})
}
