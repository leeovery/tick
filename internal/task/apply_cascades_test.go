package task

import (
	"testing"
	"time"
)

func assertTransition(t *testing.T, task Task, index int, from, to Status, auto bool) {
	t.Helper()
	if len(task.Transitions) <= index {
		t.Fatalf("expected at least %d transition(s), got %d", index+1, len(task.Transitions))
	}
	tr := task.Transitions[index]
	if tr.From != from {
		t.Errorf("Transitions[%d].From = %q, want %q", index, tr.From, from)
	}
	if tr.To != to {
		t.Errorf("Transitions[%d].To = %q, want %q", index, tr.To, to)
	}
	if tr.Auto != auto {
		t.Errorf("Transitions[%d].Auto = %v, want %v", index, tr.Auto, auto)
	}
}

func TestStateMachine_ApplyUserTransition(t *testing.T) {
	var sm StateMachine
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	makeTask := func(id string, status Status, parent string) Task {
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

	t.Run("it applies primary transition and returns result", func(t *testing.T) {
		child := makeTask("tick-child1", StatusOpen, "")
		tasks := []Task{child}

		result, cascades, err := sm.ApplyUserTransition(tasks, &tasks[0], "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.OldStatus != StatusOpen {
			t.Errorf("OldStatus = %q, want %q", result.OldStatus, StatusOpen)
		}
		if result.NewStatus != StatusInProgress {
			t.Errorf("NewStatus = %q, want %q", result.NewStatus, StatusInProgress)
		}
		if tasks[0].Status != StatusInProgress {
			t.Errorf("task status = %q, want %q", tasks[0].Status, StatusInProgress)
		}
		if len(cascades) != 0 {
			t.Errorf("expected 0 cascades, got %d", len(cascades))
		}
	})

	t.Run("it records transition history on primary task", func(t *testing.T) {
		child := makeTask("tick-child1", StatusOpen, "")
		tasks := []Task{child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[0], "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertTransition(t, tasks[0], 0, StatusOpen, StatusInProgress, false)
		if tasks[0].Transitions[0].At.IsZero() {
			t.Error("transition At should not be zero")
		}
	})

	t.Run("it applies single-level upward start cascade", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusOpen, "")
		child := makeTask("tick-child1", StatusOpen, "tick-parent1")
		tasks := []Task{parent, child}

		result, cascades, err := sm.ApplyUserTransition(tasks, &tasks[1], "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.OldStatus != StatusOpen || result.NewStatus != StatusInProgress {
			t.Errorf("primary result = %q->%q, want open->in_progress", result.OldStatus, result.NewStatus)
		}

		if len(cascades) != 1 {
			t.Fatalf("expected 1 cascade, got %d", len(cascades))
		}
		if cascades[0].Task.ID != "tick-parent1" {
			t.Errorf("cascade task = %q, want tick-parent1", cascades[0].Task.ID)
		}
		if cascades[0].OldStatus != StatusOpen || cascades[0].NewStatus != StatusInProgress {
			t.Errorf("cascade = %q->%q, want open->in_progress", cascades[0].OldStatus, cascades[0].NewStatus)
		}

		// Parent should actually be mutated
		if tasks[0].Status != StatusInProgress {
			t.Errorf("parent status = %q, want in_progress", tasks[0].Status)
		}
	})

	t.Run("it applies multi-level downward cancel cascade", func(t *testing.T) {
		grandparent := makeTask("tick-gp1", StatusInProgress, "")
		parent := makeTask("tick-parent1", StatusInProgress, "tick-gp1")
		child := makeTask("tick-child1", StatusOpen, "tick-parent1")
		tasks := []Task{grandparent, parent, child}

		_, cascades, err := sm.ApplyUserTransition(tasks, &tasks[0], "cancel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should cascade cancel down to parent and child
		if len(cascades) != 2 {
			t.Fatalf("expected 2 cascades, got %d", len(cascades))
		}

		// All non-terminal descendants should be cancelled
		if tasks[1].Status != StatusCancelled {
			t.Errorf("parent status = %q, want cancelled", tasks[1].Status)
		}
		if tasks[2].Status != StatusCancelled {
			t.Errorf("child status = %q, want cancelled", tasks[2].Status)
		}

		// Verify transition records on cascaded tasks
		assertTransition(t, tasks[1], 0, StatusInProgress, StatusCancelled, true)
		assertTransition(t, tasks[2], 0, StatusOpen, StatusCancelled, true)
	})

	t.Run("it chains upward completion after downward cascade", func(t *testing.T) {
		// grandparent has parent as only child, parent has child1(done) and child2(open)
		// When we done child2, all parent's children become terminal -> parent auto-done
		// Then grandparent's only child (parent) is terminal -> grandparent auto-done
		grandparent := makeTask("tick-gp1", StatusInProgress, "")
		parent := makeTask("tick-parent1", StatusInProgress, "tick-gp1")
		child1 := makeTask("tick-child1", StatusDone, "tick-parent1")
		child1.Closed = closedTime()
		child2 := makeTask("tick-child2", StatusOpen, "tick-parent1")
		tasks := []Task{grandparent, parent, child1, child2}

		_, cascades, err := sm.ApplyUserTransition(tasks, &tasks[3], "done")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// parent should auto-done (Rule 3), then grandparent should auto-done (Rule 3 chained)
		parentCascaded := false
		gpCascaded := false
		for _, c := range cascades {
			if c.Task.ID == "tick-parent1" && c.NewStatus == StatusDone {
				parentCascaded = true
			}
			if c.Task.ID == "tick-gp1" && c.NewStatus == StatusDone {
				gpCascaded = true
			}
		}
		if !parentCascaded {
			t.Error("expected parent to be cascaded to done")
		}
		if !gpCascaded {
			t.Error("expected grandparent to be cascaded to done")
		}

		if tasks[0].Status != StatusDone {
			t.Errorf("grandparent status = %q, want done", tasks[0].Status)
		}
		if tasks[1].Status != StatusDone {
			t.Errorf("parent status = %q, want done", tasks[1].Status)
		}

		// Verify transition records on cascaded tasks
		assertTransition(t, tasks[0], 0, StatusInProgress, StatusDone, true)
		assertTransition(t, tasks[1], 0, StatusInProgress, StatusDone, true)
	})

	t.Run("it deduplicates via seen-map", func(t *testing.T) {
		// Parent with two children. When parent is cancelled, both children get cancel cascade.
		// The seen-map should prevent processing either child twice even if somehow enqueued again.
		parent := makeTask("tick-parent1", StatusInProgress, "")
		child1 := makeTask("tick-child1", StatusOpen, "tick-parent1")
		child2 := makeTask("tick-child2", StatusOpen, "tick-parent1")
		tasks := []Task{parent, child1, child2}

		_, cascades, err := sm.ApplyUserTransition(tasks, &tasks[0], "cancel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Count how many times each child appears
		counts := make(map[string]int)
		for _, c := range cascades {
			counts[c.Task.ID]++
		}
		for id, count := range counts {
			if count > 1 {
				t.Errorf("task %s appeared %d times in cascades, expected at most 1", id, count)
			}
		}
	})

	t.Run("it returns empty cascades for leaf task", func(t *testing.T) {
		leaf := makeTask("tick-leaf1", StatusOpen, "")
		tasks := []Task{leaf}

		_, cascades, err := sm.ApplyUserTransition(tasks, &tasks[0], "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(cascades) != 0 {
			t.Errorf("expected 0 cascades for leaf task, got %d", len(cascades))
		}
	})

	t.Run("it returns error for invalid primary transition", func(t *testing.T) {
		task := makeTask("tick-task1", StatusDone, "")
		task.Closed = closedTime()
		originalStatus := task.Status
		tasks := []Task{task}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[0], "start")
		if err == nil {
			t.Fatal("expected error for invalid transition, got nil")
		}

		// Task should not be mutated
		if tasks[0].Status != originalStatus {
			t.Errorf("task status mutated to %q on error, expected %q", tasks[0].Status, originalStatus)
		}
		if len(tasks[0].Transitions) != 0 {
			t.Error("no transition history should be recorded on error")
		}
	})

	t.Run("it handles reopen cascade chain", func(t *testing.T) {
		// done grandparent -> done parent -> done child
		// Reopen child -> parent should reopen (Rule 5) -> grandparent should reopen (Rule 5)
		grandparent := makeTask("tick-gp1", StatusDone, "")
		grandparent.Closed = closedTime()
		parent := makeTask("tick-parent1", StatusDone, "tick-gp1")
		parent.Closed = closedTime()
		child := makeTask("tick-child1", StatusDone, "tick-parent1")
		child.Closed = closedTime()
		tasks := []Task{grandparent, parent, child}

		_, cascades, err := sm.ApplyUserTransition(tasks, &tasks[2], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parentReopened := false
		gpReopened := false
		for _, c := range cascades {
			if c.Task.ID == "tick-parent1" && c.NewStatus == StatusOpen {
				parentReopened = true
			}
			if c.Task.ID == "tick-gp1" && c.NewStatus == StatusOpen {
				gpReopened = true
			}
		}
		if !parentReopened {
			t.Error("expected parent to be reopened")
		}
		if !gpReopened {
			t.Error("expected grandparent to be reopened")
		}

		if tasks[0].Status != StatusOpen {
			t.Errorf("grandparent status = %q, want open", tasks[0].Status)
		}
		if tasks[1].Status != StatusOpen {
			t.Errorf("parent status = %q, want open", tasks[1].Status)
		}
		if tasks[2].Status != StatusOpen {
			t.Errorf("child status = %q, want open", tasks[2].Status)
		}

		// Verify transition records on cascaded tasks
		assertTransition(t, tasks[0], 0, StatusDone, StatusOpen, true)
		assertTransition(t, tasks[1], 0, StatusDone, StatusOpen, true)
	})

	t.Run("it records auto transitions on all cascaded tasks", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusOpen, "")
		child := makeTask("tick-child1", StatusOpen, "tick-parent1")
		tasks := []Task{parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[1], "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Primary task: Auto = false
		assertTransition(t, tasks[1], 0, StatusOpen, StatusInProgress, false)

		// Cascaded parent: Auto = true
		assertTransition(t, tasks[0], 0, StatusOpen, StatusInProgress, true)
	})

	t.Run("it blocks reopen under cancelled direct parent", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusCancelled, "")
		parent.Closed = closedTime()
		child := makeTask("tick-child1", StatusCancelled, "tick-parent1")
		child.Closed = closedTime()
		tasks := []Task{parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[1], "reopen")
		if err == nil {
			t.Fatal("expected error for reopen under cancelled parent, got nil")
		}

		expected := "cannot reopen task under cancelled parent, reopen parent first"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}

		// Task should not be mutated
		if tasks[1].Status != StatusCancelled {
			t.Errorf("task status mutated to %q on error, expected cancelled", tasks[1].Status)
		}
		if len(tasks[1].Transitions) != 0 {
			t.Error("no transition history should be recorded on error")
		}
	})

	t.Run("it allows reopen with no parent", func(t *testing.T) {
		child := makeTask("tick-child1", StatusCancelled, "")
		child.Closed = closedTime()
		tasks := []Task{child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[0], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tasks[0].Status != StatusOpen {
			t.Errorf("expected status open, got %q", tasks[0].Status)
		}
	})

	t.Run("it allows reopen under open parent", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusOpen, "")
		child := makeTask("tick-child1", StatusDone, "tick-parent1")
		child.Closed = closedTime()
		tasks := []Task{parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[1], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tasks[1].Status != StatusOpen {
			t.Errorf("expected status open, got %q", tasks[1].Status)
		}
	})

	t.Run("it allows reopen under done parent", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusDone, "")
		parent.Closed = closedTime()
		child := makeTask("tick-child1", StatusDone, "tick-parent1")
		child.Closed = closedTime()
		tasks := []Task{parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[1], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tasks[1].Status != StatusOpen {
			t.Errorf("expected status open, got %q", tasks[1].Status)
		}
	})

	t.Run("it allows reopen under in_progress parent", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusInProgress, "")
		child := makeTask("tick-child1", StatusDone, "tick-parent1")
		child.Closed = closedTime()
		tasks := []Task{parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[1], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tasks[1].Status != StatusOpen {
			t.Errorf("expected status open, got %q", tasks[1].Status)
		}
	})

	t.Run("it allows reopen when grandparent is cancelled but direct parent is not", func(t *testing.T) {
		grandparent := makeTask("tick-gp1", StatusCancelled, "")
		grandparent.Closed = closedTime()
		parent := makeTask("tick-parent1", StatusDone, "tick-gp1")
		parent.Closed = closedTime()
		child := makeTask("tick-child1", StatusDone, "tick-parent1")
		child.Closed = closedTime()
		tasks := []Task{grandparent, parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[2], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tasks[2].Status != StatusOpen {
			t.Errorf("expected status open, got %q", tasks[2].Status)
		}
	})

	t.Run("it proceeds with reopen when parent ID references non-existent task", func(t *testing.T) {
		child := makeTask("tick-child1", StatusCancelled, "tick-missing")
		child.Closed = closedTime()
		tasks := []Task{child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[0], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tasks[0].Status != StatusOpen {
			t.Errorf("expected status open, got %q", tasks[0].Status)
		}
	})

	t.Run("it skips Rule 9 check for non-reopen actions", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusCancelled, "")
		parent.Closed = closedTime()
		child := makeTask("tick-child1", StatusOpen, "tick-parent1")
		tasks := []Task{parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[1], "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tasks[1].Status != StatusInProgress {
			t.Errorf("expected status in_progress, got %q", tasks[1].Status)
		}
	})

	t.Run("it records auto=false on primary target for user transition", func(t *testing.T) {
		parent := makeTask("tick-parent1", StatusOpen, "")
		child := makeTask("tick-child1", StatusOpen, "tick-parent1")
		tasks := []Task{parent, child}

		_, _, err := sm.ApplyUserTransition(tasks, &tasks[1], "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertTransition(t, tasks[1], 0, StatusOpen, StatusInProgress, false)

		// Cascade transitions should still be Auto = true
		assertTransition(t, tasks[0], 0, StatusOpen, StatusInProgress, true)
	})

	t.Run("it records auto=true on primary target for system transition", func(t *testing.T) {
		// Simulate Rule 6: done parent being reopened by system when child is added.
		parent := makeTask("tick-parent1", StatusDone, "")
		parent.Closed = closedTime()
		child := makeTask("tick-child1", StatusOpen, "tick-parent1")
		tasks := []Task{parent, child}

		_, _, err := sm.ApplySystemTransition(tasks, &tasks[0], "reopen")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertTransition(t, tasks[0], 0, StatusDone, StatusOpen, true)
	})
}
