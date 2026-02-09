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
			t.Fatalf("unexpected error: %v", err)
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
	})

	t.Run("it rejects 2-node cycle with path", func(t *testing.T) {
		// tick-bbb is already blocked by tick-aaa.
		// Adding tick-aaa blocked by tick-bbb would create: tick-aaa -> tick-bbb -> tick-aaa
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: nil},
			{ID: "tick-bbb", BlockedBy: []string{"tick-aaa"}},
		}
		err := ValidateDependency(tasks, "tick-aaa", "tick-bbb")
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "creates cycle") {
			t.Errorf("error should mention cycle: %q", errMsg)
		}
		// Path format: tick-aaa -> tick-bbb -> tick-aaa
		wantPath := "tick-aaa \u2192 tick-bbb \u2192 tick-aaa"
		if !strings.Contains(errMsg, wantPath) {
			t.Errorf("error should contain path %q, got: %q", wantPath, errMsg)
		}
	})

	t.Run("it rejects 3+ node cycle with full path", func(t *testing.T) {
		// tick-bbb blocked by tick-aaa, tick-ccc blocked by tick-bbb.
		// Adding tick-aaa blocked by tick-ccc would create: tick-aaa -> tick-ccc -> tick-bbb -> tick-aaa
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: nil},
			{ID: "tick-bbb", BlockedBy: []string{"tick-aaa"}},
			{ID: "tick-ccc", BlockedBy: []string{"tick-bbb"}},
		}
		err := ValidateDependency(tasks, "tick-aaa", "tick-ccc")
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		errMsg := err.Error()
		wantPath := "tick-aaa \u2192 tick-ccc \u2192 tick-bbb \u2192 tick-aaa"
		if !strings.Contains(errMsg, wantPath) {
			t.Errorf("error should contain path %q, got: %q", wantPath, errMsg)
		}
	})

	t.Run("it rejects child blocked by own parent", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent", BlockedBy: nil},
			{ID: "tick-child", Parent: "tick-parent", BlockedBy: nil},
		}
		err := ValidateDependency(tasks, "tick-child", "tick-parent")
		if err == nil {
			t.Fatal("expected error for child blocked by parent, got nil")
		}
		errMsg := err.Error()
		want := "Cannot add dependency - tick-child cannot be blocked by its parent tick-parent (would create unworkable task due to leaf-only ready rule)"
		if errMsg != want {
			t.Errorf("error message:\ngot:  %q\nwant: %q", errMsg, want)
		}
	})

	t.Run("it allows parent blocked by own child", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent", BlockedBy: nil},
			{ID: "tick-child", Parent: "tick-parent", BlockedBy: nil},
		}
		err := ValidateDependency(tasks, "tick-parent", "tick-child")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("it allows sibling dependencies", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent", BlockedBy: nil},
			{ID: "tick-sib1", Parent: "tick-parent", BlockedBy: nil},
			{ID: "tick-sib2", Parent: "tick-parent", BlockedBy: nil},
		}
		err := ValidateDependency(tasks, "tick-sib1", "tick-sib2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("it allows cross-hierarchy dependencies", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-par1", BlockedBy: nil},
			{ID: "tick-par2", BlockedBy: nil},
			{ID: "tick-ch1", Parent: "tick-par1", BlockedBy: nil},
			{ID: "tick-ch2", Parent: "tick-par2", BlockedBy: nil},
		}
		err := ValidateDependency(tasks, "tick-ch1", "tick-ch2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("it returns cycle path format: tick-a -> tick-b -> tick-a", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-a", BlockedBy: nil},
			{ID: "tick-b", BlockedBy: []string{"tick-a"}},
		}
		err := ValidateDependency(tasks, "tick-a", "tick-b")
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		want := "Cannot add dependency - creates cycle: tick-a \u2192 tick-b \u2192 tick-a"
		if err.Error() != want {
			t.Errorf("error message:\ngot:  %q\nwant: %q", err.Error(), want)
		}
	})

	t.Run("it detects cycle through existing multi-hop chain", func(t *testing.T) {
		// Chain: tick-d blocked by tick-c, tick-c blocked by tick-b, tick-b blocked by tick-a
		// Adding tick-a blocked by tick-d creates: tick-a -> tick-d -> tick-c -> tick-b -> tick-a
		tasks := []Task{
			{ID: "tick-a", BlockedBy: nil},
			{ID: "tick-b", BlockedBy: []string{"tick-a"}},
			{ID: "tick-c", BlockedBy: []string{"tick-b"}},
			{ID: "tick-d", BlockedBy: []string{"tick-c"}},
		}
		err := ValidateDependency(tasks, "tick-a", "tick-d")
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		errMsg := err.Error()
		wantPath := "tick-a \u2192 tick-d \u2192 tick-c \u2192 tick-b \u2192 tick-a"
		if !strings.Contains(errMsg, wantPath) {
			t.Errorf("error should contain path %q, got: %q", wantPath, errMsg)
		}
	})
}

func TestValidateDependencies(t *testing.T) {
	t.Run("it validates multiple blocked_by IDs, fails on first error", func(t *testing.T) {
		// tick-bbb already blocked by tick-aaa
		// Batch: add tick-aaa blocked by [tick-ccc, tick-bbb]
		// tick-ccc is fine, tick-bbb creates a cycle
		// But sequential order means tick-ccc is checked first (passes), then tick-bbb (fails)
		tasks := []Task{
			{ID: "tick-aaa", BlockedBy: nil},
			{ID: "tick-bbb", BlockedBy: []string{"tick-aaa"}},
			{ID: "tick-ccc", BlockedBy: nil},
		}
		err := ValidateDependencies(tasks, "tick-aaa", []string{"tick-ccc", "tick-bbb"})
		if err == nil {
			t.Fatal("expected error for cycle in batch, got nil")
		}
		errMsg := err.Error()
		// Should fail on tick-bbb (the second one), not tick-ccc
		if !strings.Contains(errMsg, "tick-bbb") {
			t.Errorf("error should mention tick-bbb: %q", errMsg)
		}
	})
}
