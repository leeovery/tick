package task

import (
	"strings"
	"testing"
)

func TestValidateDependency(t *testing.T) {
	t.Run("it allows valid dependency between unrelated tasks", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: nil},
			{ID: "tick-bbb", BlockedBy: nil},
		}

		err := ValidateDependency(tasks, "tick-aaa", "tick-bbb")
		if err != nil {
			t.Errorf("expected no error for valid dependency, got: %v", err)
		}
	})

	t.Run("it rejects direct self-reference", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: nil},
		}

		err := ValidateDependency(tasks, "tick-aaa", "tick-aaa")
		if err == nil {
			t.Fatal("expected error for self-reference, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-aaa \u2192 tick-aaa"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it rejects 2-node cycle with path", func(t *testing.T) {
		// tick-aaa is already blocked by tick-bbb
		// Adding tick-bbb blocked by tick-aaa would create: tick-bbb -> tick-aaa -> tick-bbb
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb", BlockedBy: nil},
		}

		err := ValidateDependency(tasks, "tick-bbb", "tick-aaa")
		if err == nil {
			t.Fatal("expected error for 2-node cycle, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-bbb \u2192 tick-aaa \u2192 tick-bbb"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it rejects 3+ node cycle with full path", func(t *testing.T) {
		// tick-aaa blocked by tick-bbb, tick-bbb blocked by tick-ccc
		// Adding tick-ccc blocked by tick-aaa would create: tick-ccc -> tick-aaa -> tick-bbb -> tick-ccc
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb", BlockedBy: []string{"tick-ccc"}},
			{ID: "tick-ccc", BlockedBy: nil},
		}

		err := ValidateDependency(tasks, "tick-ccc", "tick-aaa")
		if err == nil {
			t.Fatal("expected error for 3+ node cycle, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-ccc \u2192 tick-aaa \u2192 tick-bbb \u2192 tick-ccc"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it rejects child blocked by own parent", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child", Parent: "tick-parent"},
		}

		err := ValidateDependency(tasks, "tick-child", "tick-parent")
		if err == nil {
			t.Fatal("expected error for child blocked by parent, got nil")
		}

		expected := "cannot add dependency - tick-child cannot be blocked by its parent tick-parent\n(would create unworkable task due to leaf-only ready rule)"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it allows parent blocked by own child", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child", Parent: "tick-parent"},
		}

		err := ValidateDependency(tasks, "tick-parent", "tick-child")
		if err != nil {
			t.Errorf("expected no error for parent blocked by child, got: %v", err)
		}
	})

	t.Run("it allows sibling dependencies", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-sib1", Parent: "tick-parent"},
			{ID: "tick-sib2", Parent: "tick-parent"},
		}

		err := ValidateDependency(tasks, "tick-sib1", "tick-sib2")
		if err != nil {
			t.Errorf("expected no error for sibling dependency, got: %v", err)
		}
	})

	t.Run("it allows cross-hierarchy dependencies", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-p1"},
			{ID: "tick-p2"},
			{ID: "tick-c1", Parent: "tick-p1"},
			{ID: "tick-c2", Parent: "tick-p2"},
		}

		err := ValidateDependency(tasks, "tick-c1", "tick-c2")
		if err != nil {
			t.Errorf("expected no error for cross-hierarchy dependency, got: %v", err)
		}
	})

	t.Run("it returns cycle path format: tick-a -> tick-b -> tick-a", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-a", BlockedBy: []string{"tick-b"}},
			{ID: "tick-b", BlockedBy: nil},
		}

		err := ValidateDependency(tasks, "tick-b", "tick-a")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-b \u2192 tick-a \u2192 tick-b"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("it detects cycle through existing multi-hop chain", func(t *testing.T) {
		// Chain: tick-a -> tick-b -> tick-c -> tick-d (existing)
		// Adding tick-d blocked by tick-a would create cycle
		tasks := []Task{
			{ID: "tick-a", BlockedBy: []string{"tick-b"}},
			{ID: "tick-b", BlockedBy: []string{"tick-c"}},
			{ID: "tick-c", BlockedBy: []string{"tick-d"}},
			{ID: "tick-d", BlockedBy: nil},
		}

		err := ValidateDependency(tasks, "tick-d", "tick-a")
		if err == nil {
			t.Fatal("expected error for multi-hop cycle, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-d \u2192 tick-a \u2192 tick-b \u2192 tick-c \u2192 tick-d"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})
}

func TestValidateDependencyMixedCase(t *testing.T) {
	t.Run("it detects cycle with mixed-case IDs", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb", BlockedBy: nil},
		}

		// Pass mixed-case IDs — should still detect the cycle.
		err := ValidateDependency(tasks, "TICK-BBB", "TICK-AAA")
		if err == nil {
			t.Fatal("expected error for mixed-case cycle, got nil")
		}
		if !strings.Contains(err.Error(), "creates cycle") {
			t.Errorf("expected cycle error, got: %v", err)
		}
	})

	t.Run("it detects child-blocked-by-parent with mixed-case IDs", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child", Parent: "tick-parent"},
		}

		err := ValidateDependency(tasks, "TICK-CHILD", "TICK-PARENT")
		if err == nil {
			t.Fatal("expected error for mixed-case child-blocked-by-parent, got nil")
		}
		if !strings.Contains(err.Error(), "cannot be blocked by its parent") {
			t.Errorf("expected parent error, got: %v", err)
		}
	})

	t.Run("it allows valid dependency with mixed-case IDs", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: nil},
			{ID: "tick-bbb", BlockedBy: nil},
		}

		err := ValidateDependency(tasks, "TICK-AAA", "TICK-BBB")
		if err != nil {
			t.Errorf("expected no error for valid mixed-case dependency, got: %v", err)
		}
	})
}

func TestValidateDependencies(t *testing.T) {
	t.Run("it validates multiple blocked_by IDs, fails on first error", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: []string{"tick-bbb"}},
			{ID: "tick-bbb", BlockedBy: nil},
			{ID: "tick-ccc", BlockedBy: nil},
		}

		// tick-bbb -> tick-aaa creates cycle; tick-ccc is valid
		// Should fail on tick-aaa (the cycle) before checking tick-ccc
		err := ValidateDependencies(tasks, "tick-bbb", []string{"tick-aaa", "tick-ccc"})
		if err == nil {
			t.Fatal("expected error for batch validation, got nil")
		}

		expected := "cannot add dependency - creates cycle: tick-bbb \u2192 tick-aaa \u2192 tick-bbb"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})
}
