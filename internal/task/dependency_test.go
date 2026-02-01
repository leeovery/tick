package task

import (
	"strings"
	"testing"
)

func TestValidateDependency(t *testing.T) {
	t.Run("allows valid dependency between unrelated tasks", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa001"},
			{ID: "tick-bbb002"},
		}
		err := ValidateDependency(tasks, "tick-aaa001", "tick-bbb002")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("rejects direct self-reference", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa001"},
		}
		err := ValidateDependency(tasks, "tick-aaa001", "tick-aaa001")
		if err == nil {
			t.Fatal("expected error for self-reference")
		}
	})

	t.Run("rejects 2-node cycle with path", func(t *testing.T) {
		// A blocked by B, now try to add B blocked by A → cycle
		tasks := []Task{
			{ID: "tick-aaa001", BlockedBy: []string{"tick-bbb002"}},
			{ID: "tick-bbb002"},
		}
		err := ValidateDependency(tasks, "tick-bbb002", "tick-aaa001")
		if err == nil {
			t.Fatal("expected cycle error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "cycle") {
			t.Errorf("error should mention cycle, got %q", msg)
		}
		if !strings.Contains(msg, "→") {
			t.Errorf("error should contain path arrows, got %q", msg)
		}
	})

	t.Run("rejects 3+ node cycle with full path", func(t *testing.T) {
		// A blocked by B, B blocked by C, now try to add C blocked by A → cycle A→B→C→A
		tasks := []Task{
			{ID: "tick-aaa001", BlockedBy: []string{"tick-bbb002"}},
			{ID: "tick-bbb002", BlockedBy: []string{"tick-ccc003"}},
			{ID: "tick-ccc003"},
		}
		err := ValidateDependency(tasks, "tick-ccc003", "tick-aaa001")
		if err == nil {
			t.Fatal("expected cycle error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "cycle") {
			t.Errorf("error should mention cycle, got %q", msg)
		}
		// Path should include all three nodes
		if !strings.Contains(msg, "tick-aaa001") || !strings.Contains(msg, "tick-bbb002") || !strings.Contains(msg, "tick-ccc003") {
			t.Errorf("error should contain full path, got %q", msg)
		}
	})

	t.Run("rejects child blocked by own parent", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child0", Parent: "tick-parent"},
		}
		err := ValidateDependency(tasks, "tick-child0", "tick-parent")
		if err == nil {
			t.Fatal("expected error for child blocked by parent")
		}
		msg := err.Error()
		if !strings.Contains(msg, "parent") {
			t.Errorf("error should mention parent, got %q", msg)
		}
	})

	t.Run("allows parent blocked by own child", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child0", Parent: "tick-parent"},
		}
		err := ValidateDependency(tasks, "tick-parent", "tick-child0")
		if err != nil {
			t.Errorf("parent blocked by child should be allowed, got: %v", err)
		}
	})

	t.Run("allows sibling dependencies", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-parent"},
			{ID: "tick-child1", Parent: "tick-parent"},
			{ID: "tick-child2", Parent: "tick-parent"},
		}
		err := ValidateDependency(tasks, "tick-child2", "tick-child1")
		if err != nil {
			t.Errorf("sibling deps should be allowed, got: %v", err)
		}
	})

	t.Run("allows cross-hierarchy dependencies", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-par001"},
			{ID: "tick-chi001", Parent: "tick-par001"},
			{ID: "tick-par002"},
			{ID: "tick-chi002", Parent: "tick-par002"},
		}
		err := ValidateDependency(tasks, "tick-chi002", "tick-chi001")
		if err != nil {
			t.Errorf("cross-hierarchy deps should be allowed, got: %v", err)
		}
	})

	t.Run("detects cycle through existing multi-hop chain", func(t *testing.T) {
		// A→B→C→D, try to add D→A
		tasks := []Task{
			{ID: "tick-aaa001", BlockedBy: []string{"tick-bbb002"}},
			{ID: "tick-bbb002", BlockedBy: []string{"tick-ccc003"}},
			{ID: "tick-ccc003", BlockedBy: []string{"tick-ddd004"}},
			{ID: "tick-ddd004"},
		}
		err := ValidateDependency(tasks, "tick-ddd004", "tick-aaa001")
		if err == nil {
			t.Fatal("expected cycle error")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("error should mention cycle, got %q", err.Error())
		}
	})
}

func TestValidateDependencies(t *testing.T) {
	t.Run("validates multiple blocked_by IDs, fails on first error", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa001"},
			{ID: "tick-bbb002"},
		}
		// First is valid, second is self-ref
		err := ValidateDependencies(tasks, "tick-aaa001", []string{"tick-bbb002", "tick-aaa001"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("passes when all dependencies valid", func(t *testing.T) {
		tasks := []Task{
			{ID: "tick-aaa001"},
			{ID: "tick-bbb002"},
			{ID: "tick-ccc003"},
		}
		err := ValidateDependencies(tasks, "tick-aaa001", []string{"tick-bbb002", "tick-ccc003"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
