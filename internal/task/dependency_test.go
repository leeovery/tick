package task

import (
	"strings"
	"testing"
)

// depTask creates a test task for dependency validation with the given ID,
// parent, and blocked_by list.
func depTask(id string, parent string, blockedBy ...string) Task {
	return Task{
		ID:        id,
		Title:     "Test task " + id,
		Status:    StatusOpen,
		Priority:  2,
		BlockedBy: blockedBy,
		Parent:    parent,
	}
}

func TestValidateDependency(t *testing.T) {
	t.Run("it allows valid dependency between unrelated tasks", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-aaa", ""),
			depTask("tick-bbb", ""),
		}

		err := ValidateDependency(tasks, "tick-aaa", "tick-bbb")
		if err != nil {
			t.Errorf("expected no error for valid dependency, got %v", err)
		}
	})

	t.Run("it rejects direct self-reference", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-aaa", ""),
		}

		err := ValidateDependency(tasks, "tick-aaa", "tick-aaa")
		if err == nil {
			t.Fatal("expected error for self-reference, got nil")
		}
		if !strings.Contains(err.Error(), "creates cycle") {
			t.Errorf("expected cycle error, got %q", err.Error())
		}
		if !strings.Contains(err.Error(), "tick-aaa") {
			t.Errorf("expected error to contain task ID, got %q", err.Error())
		}
	})

	t.Run("it rejects 2-node cycle with path", func(t *testing.T) {
		// tick-bbb is already blocked by tick-aaa
		// Now trying to add tick-aaa blocked by tick-bbb => cycle
		tasks := []Task{
			depTask("tick-aaa", ""),
			depTask("tick-bbb", "", "tick-aaa"),
		}

		err := ValidateDependency(tasks, "tick-aaa", "tick-bbb")
		if err == nil {
			t.Fatal("expected error for 2-node cycle, got nil")
		}
		want := "Error: Cannot add dependency - creates cycle: tick-aaa \u2192 tick-bbb \u2192 tick-aaa"
		if err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err.Error())
		}
	})

	t.Run("it rejects 3+ node cycle with full path", func(t *testing.T) {
		// tick-bbb blocked by tick-aaa
		// tick-ccc blocked by tick-bbb
		// Now trying to add tick-aaa blocked by tick-ccc => cycle: aaa -> ccc -> bbb -> aaa
		tasks := []Task{
			depTask("tick-aaa", ""),
			depTask("tick-bbb", "", "tick-aaa"),
			depTask("tick-ccc", "", "tick-bbb"),
		}

		err := ValidateDependency(tasks, "tick-aaa", "tick-ccc")
		if err == nil {
			t.Fatal("expected error for 3+ node cycle, got nil")
		}
		want := "Error: Cannot add dependency - creates cycle: tick-aaa \u2192 tick-ccc \u2192 tick-bbb \u2192 tick-aaa"
		if err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err.Error())
		}
	})

	t.Run("it rejects child blocked by own parent", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-parent", ""),
			depTask("tick-child", "tick-parent"),
		}

		err := ValidateDependency(tasks, "tick-child", "tick-parent")
		if err == nil {
			t.Fatal("expected error for child blocked by parent, got nil")
		}
		want := "Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"
		if err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err.Error())
		}
	})

	t.Run("it allows parent blocked by own child", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-parent", ""),
			depTask("tick-child", "tick-parent"),
		}

		err := ValidateDependency(tasks, "tick-parent", "tick-child")
		if err != nil {
			t.Errorf("expected no error for parent blocked by child, got %v", err)
		}
	})

	t.Run("it allows sibling dependencies", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-parent", ""),
			depTask("tick-sib1", "tick-parent"),
			depTask("tick-sib2", "tick-parent"),
		}

		err := ValidateDependency(tasks, "tick-sib2", "tick-sib1")
		if err != nil {
			t.Errorf("expected no error for sibling dependency, got %v", err)
		}
	})

	t.Run("it allows cross-hierarchy dependencies", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-p1", ""),
			depTask("tick-c1", "tick-p1"),
			depTask("tick-p2", ""),
			depTask("tick-c2", "tick-p2"),
		}

		err := ValidateDependency(tasks, "tick-c1", "tick-c2")
		if err != nil {
			t.Errorf("expected no error for cross-hierarchy dependency, got %v", err)
		}
	})

	t.Run("it returns cycle path format: tick-a -> tick-b -> tick-a", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-a", ""),
			depTask("tick-b", "", "tick-a"),
		}

		err := ValidateDependency(tasks, "tick-a", "tick-b")
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		want := "Error: Cannot add dependency - creates cycle: tick-a \u2192 tick-b \u2192 tick-a"
		if err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err.Error())
		}
	})

	t.Run("it detects cycle through existing multi-hop chain", func(t *testing.T) {
		// Chain: tick-d blocked by tick-c, tick-c blocked by tick-b, tick-b blocked by tick-a
		// Adding tick-a blocked by tick-d creates: a -> d -> c -> b -> a
		tasks := []Task{
			depTask("tick-a", ""),
			depTask("tick-b", "", "tick-a"),
			depTask("tick-c", "", "tick-b"),
			depTask("tick-d", "", "tick-c"),
		}

		err := ValidateDependency(tasks, "tick-a", "tick-d")
		if err == nil {
			t.Fatal("expected error for multi-hop cycle, got nil")
		}
		want := "Error: Cannot add dependency - creates cycle: tick-a \u2192 tick-d \u2192 tick-c \u2192 tick-b \u2192 tick-a"
		if err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err.Error())
		}
	})
}

func TestValidateDependencies(t *testing.T) {
	t.Run("it validates multiple blocked_by IDs, fails on first error", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-parent", ""),
			depTask("tick-child", "tick-parent"),
			depTask("tick-other", ""),
		}

		// tick-other is valid, tick-parent is invalid (child blocked by parent)
		err := ValidateDependencies(tasks, "tick-child", []string{"tick-other", "tick-parent"})
		if err == nil {
			t.Fatal("expected error for batch with invalid dependency, got nil")
		}
		// Should fail on tick-parent (child blocked by parent)
		want := "Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"
		if err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err.Error())
		}
	})

	t.Run("it succeeds when all dependencies are valid", func(t *testing.T) {
		tasks := []Task{
			depTask("tick-aaa", ""),
			depTask("tick-bbb", ""),
			depTask("tick-ccc", ""),
		}

		err := ValidateDependencies(tasks, "tick-aaa", []string{"tick-bbb", "tick-ccc"})
		if err != nil {
			t.Errorf("expected no error for valid batch, got %v", err)
		}
	})
}
