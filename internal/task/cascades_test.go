package task

import (
	"testing"
	"time"
)

func TestStateMachine_Cascades(t *testing.T) {
	var sm StateMachine
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	makeTestTask := func(id string, status Status, parent string) Task {
		return Task{
			ID:       id,
			Title:    "Task " + id,
			Status:   status,
			Priority: 2,
			Parent:   parent,
			Created:  now,
			Updated:  now,
		}
	}

	t.Run("it cascades start to open parent", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusInProgress, "tick-parent1")
		parent := makeTestTask("tick-parent1", StatusOpen, "")
		tasks := []Task{child, parent}

		changes := sm.Cascades(tasks, &child, "start")

		if len(changes) != 1 {
			t.Fatalf("expected 1 cascade change, got %d", len(changes))
		}
		if changes[0].Task.ID != "tick-parent1" {
			t.Errorf("expected cascade task tick-parent1, got %s", changes[0].Task.ID)
		}
		if changes[0].Action != "start" {
			t.Errorf("expected action 'start', got %q", changes[0].Action)
		}
		if changes[0].OldStatus != StatusOpen {
			t.Errorf("expected old status open, got %q", changes[0].OldStatus)
		}
		if changes[0].NewStatus != StatusInProgress {
			t.Errorf("expected new status in_progress, got %q", changes[0].NewStatus)
		}
	})

	t.Run("it cascades start through multiple open ancestors", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusInProgress, "tick-parent1")
		parent := makeTestTask("tick-parent1", StatusOpen, "tick-grandp1")
		grandparent := makeTestTask("tick-grandp1", StatusOpen, "")
		tasks := []Task{child, parent, grandparent}

		changes := sm.Cascades(tasks, &child, "start")

		if len(changes) != 2 {
			t.Fatalf("expected 2 cascade changes, got %d", len(changes))
		}
		if changes[0].Task.ID != "tick-parent1" {
			t.Errorf("expected first cascade tick-parent1, got %s", changes[0].Task.ID)
		}
		if changes[1].Task.ID != "tick-grandp1" {
			t.Errorf("expected second cascade tick-grandp1, got %s", changes[1].Task.ID)
		}
		for i, c := range changes {
			if c.OldStatus != StatusOpen {
				t.Errorf("change[%d]: expected old status open, got %q", i, c.OldStatus)
			}
			if c.NewStatus != StatusInProgress {
				t.Errorf("change[%d]: expected new status in_progress, got %q", i, c.NewStatus)
			}
			if c.Action != "start" {
				t.Errorf("change[%d]: expected action 'start', got %q", i, c.Action)
			}
		}
	})

	t.Run("it skips ancestor already in_progress", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusInProgress, "tick-parent1")
		parent := makeTestTask("tick-parent1", StatusInProgress, "tick-grandp1")
		grandparent := makeTestTask("tick-grandp1", StatusOpen, "")
		tasks := []Task{child, parent, grandparent}

		changes := sm.Cascades(tasks, &child, "start")

		// Parent is already in_progress, skip it but continue to grandparent
		if len(changes) != 1 {
			t.Fatalf("expected 1 cascade change, got %d", len(changes))
		}
		if changes[0].Task.ID != "tick-grandp1" {
			t.Errorf("expected cascade tick-grandp1, got %s", changes[0].Task.ID)
		}
	})

	t.Run("it stops at done terminal ancestor", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusInProgress, "tick-parent1")
		parent := makeTestTask("tick-parent1", StatusDone, "tick-grandp1")
		parent.Closed = closedTime()
		grandparent := makeTestTask("tick-grandp1", StatusOpen, "")
		tasks := []Task{child, parent, grandparent}

		changes := sm.Cascades(tasks, &child, "start")

		// Parent is done (terminal), stop the chain — do not cascade to grandparent
		if len(changes) != 0 {
			t.Fatalf("expected 0 cascade changes, got %d", len(changes))
		}
	})

	t.Run("it stops at cancelled terminal ancestor", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusInProgress, "tick-parent1")
		parent := makeTestTask("tick-parent1", StatusCancelled, "tick-grandp1")
		parent.Closed = closedTime()
		grandparent := makeTestTask("tick-grandp1", StatusOpen, "")
		tasks := []Task{child, parent, grandparent}

		changes := sm.Cascades(tasks, &child, "start")

		// Parent is cancelled (terminal), stop the chain — do not cascade to grandparent
		if len(changes) != 0 {
			t.Fatalf("expected 0 cascade changes, got %d", len(changes))
		}
	})

	t.Run("it handles deeply nested chain of 5+ levels", func(t *testing.T) {
		level5 := makeTestTask("tick-lv5", StatusInProgress, "tick-lv4")
		level4 := makeTestTask("tick-lv4", StatusOpen, "tick-lv3")
		level3 := makeTestTask("tick-lv3", StatusOpen, "tick-lv2")
		level2 := makeTestTask("tick-lv2", StatusOpen, "tick-lv1")
		level1 := makeTestTask("tick-lv1", StatusOpen, "tick-lv0")
		level0 := makeTestTask("tick-lv0", StatusOpen, "")
		tasks := []Task{level5, level4, level3, level2, level1, level0}

		changes := sm.Cascades(tasks, &level5, "start")

		if len(changes) != 5 {
			t.Fatalf("expected 5 cascade changes, got %d", len(changes))
		}

		expectedIDs := []string{"tick-lv4", "tick-lv3", "tick-lv2", "tick-lv1", "tick-lv0"}
		for i, expected := range expectedIDs {
			if changes[i].Task.ID != expected {
				t.Errorf("change[%d]: expected %s, got %s", i, expected, changes[i].Task.ID)
			}
		}
	})

	t.Run("it returns empty for task with no parent", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusInProgress, "")
		tasks := []Task{child}

		changes := sm.Cascades(tasks, &child, "start")

		if len(changes) != 0 {
			t.Fatalf("expected 0 cascade changes, got %d", len(changes))
		}
	})

	t.Run("it returns empty for non-start actions", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusDone, "tick-parent1")
		parent := makeTestTask("tick-parent1", StatusOpen, "")
		tasks := []Task{child, parent}

		actions := []string{"done", "cancel", "reopen"}
		for _, action := range actions {
			t.Run(action, func(t *testing.T) {
				changes := sm.Cascades(tasks, &child, action)
				if len(changes) != 0 {
					t.Errorf("expected 0 cascade changes for action %q, got %d", action, len(changes))
				}
			})
		}
	})

	t.Run("it does not mutate any task", func(t *testing.T) {
		child := makeTestTask("tick-child1", StatusInProgress, "tick-parent1")
		parent := makeTestTask("tick-parent1", StatusOpen, "tick-grandp1")
		grandparent := makeTestTask("tick-grandp1", StatusOpen, "")
		tasks := []Task{child, parent, grandparent}

		// Capture original states
		childStatus := child.Status
		parentStatus := parent.Status
		grandparentStatus := grandparent.Status
		childUpdated := child.Updated
		parentUpdated := parent.Updated
		grandparentUpdated := grandparent.Updated

		_ = sm.Cascades(tasks, &child, "start")

		// Verify no task was mutated
		if child.Status != childStatus {
			t.Errorf("child status mutated: %q -> %q", childStatus, child.Status)
		}
		if parent.Status != parentStatus {
			t.Errorf("parent status mutated: %q -> %q", parentStatus, parent.Status)
		}
		if grandparent.Status != grandparentStatus {
			t.Errorf("grandparent status mutated: %q -> %q", grandparentStatus, grandparent.Status)
		}
		if tasks[0].Status != childStatus {
			t.Errorf("tasks[0] status mutated: %q -> %q", childStatus, tasks[0].Status)
		}
		if tasks[1].Status != parentStatus {
			t.Errorf("tasks[1] status mutated: %q -> %q", parentStatus, tasks[1].Status)
		}
		if tasks[2].Status != grandparentStatus {
			t.Errorf("tasks[2] status mutated: %q -> %q", grandparentStatus, tasks[2].Status)
		}
		if !child.Updated.Equal(childUpdated) {
			t.Error("child Updated timestamp mutated")
		}
		if !parent.Updated.Equal(parentUpdated) {
			t.Error("parent Updated timestamp mutated")
		}
		if !grandparent.Updated.Equal(grandparentUpdated) {
			t.Error("grandparent Updated timestamp mutated")
		}
	})
}
