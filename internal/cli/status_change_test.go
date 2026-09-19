package cli

import (
	"reflect"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func statusChangeBlock(primaryAuto bool, primary StatusChange, cascades ...StatusChange) CascadeResult {
	set := statusChangeSet{}
	primary.Auto = primaryAuto
	set.add(primary)
	for _, c := range cascades {
		c.Auto = true
		set.add(c)
	}
	return CascadeResult{Changed: set.rows()}
}

func TestStatusChanges(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	parent := task.Task{
		ID: "tick-ppp111", Title: "Parent", Status: task.StatusOpen,
		Priority: 2, Created: now, Updated: now,
	}
	child := task.Task{
		ID: "tick-ccc111", Title: "Child", Status: task.StatusOpen,
		Priority: 2, Parent: "tick-ppp111", Created: now, Updated: now,
	}
	allTasks := []task.Task{parent, child}
	primaryResult := task.TransitionResult{OldStatus: task.StatusOpen, NewStatus: task.StatusInProgress}
	cascades := []task.CascadeChange{
		{Task: &parent, OldStatus: task.StatusOpen, NewStatus: task.StatusInProgress},
	}

	t.Run("it records the requested change with auto false", func(t *testing.T) {
		cr := buildCascadeResult("tick-ccc111", "Child", primaryResult, cascades, allTasks, false)

		want := StatusChange{ID: "tick-ccc111", Title: "Child", From: "open", To: "in_progress", Auto: false}
		if len(cr.Changed) != 2 {
			t.Fatalf("Changed rows = %d, want 2", len(cr.Changed))
		}
		if cr.Changed[0] != want {
			t.Errorf("Changed[0] = %+v, want %+v", cr.Changed[0], want)
		}
	})

	t.Run("it marks cascaded changes auto true", func(t *testing.T) {
		cr := buildCascadeResult("tick-ccc111", "Child", primaryResult, cascades, allTasks, false)

		want := StatusChange{ID: "tick-ppp111", Title: "Parent", From: "open", To: "in_progress", Auto: true}
		if len(cr.Changed) != 2 {
			t.Fatalf("Changed rows = %d, want 2", len(cr.Changed))
		}
		if cr.Changed[1] != want {
			t.Errorf("Changed[1] = %+v, want %+v", cr.Changed[1], want)
		}
	})

	t.Run("it marks every row auto true when the primary was system-initiated", func(t *testing.T) {
		cr := buildCascadeResult("tick-ccc111", "Child", primaryResult, cascades, allTasks, true)

		for _, row := range cr.Changed {
			if !row.Auto {
				t.Errorf("row %s has Auto=false, want every row auto", row.ID)
			}
		}
	})

	t.Run("it builds a cascade result for a transition with no cascades", func(t *testing.T) {
		cr := buildCascadeResult("tick-ccc111", "Child", primaryResult, nil, allTasks, false)

		if len(cr.Cascaded) != 0 {
			t.Fatalf("Cascaded = %d entries, want 0", len(cr.Cascaded))
		}
		if len(cr.Changed) != 1 {
			t.Fatalf("Changed rows = %d, want 1", len(cr.Changed))
		}
	})

	t.Run("it collapses a task moved twice into one row", func(t *testing.T) {
		b1 := statusChangeBlock(true, StatusChange{ID: "tick-ppp111", Title: "Parent", From: "done", To: "open"})
		b2 := statusChangeBlock(true, StatusChange{ID: "tick-ppp111", Title: "Parent", From: "open", To: "cancelled"})

		got := mergeStatusChanges(b1, b2)

		want := []StatusChange{{ID: "tick-ppp111", Title: "Parent", From: "done", To: "cancelled", Auto: true}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("merged = %+v, want %+v", got, want)
		}
	})

	t.Run("it drops a task that ends where it started", func(t *testing.T) {
		b1 := statusChangeBlock(true, StatusChange{ID: "tick-ppp111", Title: "Parent", From: "done", To: "open"})
		b2 := statusChangeBlock(true, StatusChange{ID: "tick-ppp111", Title: "Parent", From: "open", To: "done"})

		got := mergeStatusChanges(b1, b2)

		if len(got) != 0 {
			t.Errorf("merged = %+v, want no rows", got)
		}
	})

	t.Run("it keeps first-seen order across blocks", func(t *testing.T) {
		b1 := statusChangeBlock(false,
			StatusChange{ID: "tick-ccc111", Title: "Child", From: "open", To: "done"},
			StatusChange{ID: "tick-ppp111", Title: "Parent", From: "open", To: "done"},
		)
		b2 := statusChangeBlock(true,
			StatusChange{ID: "tick-ggg111", Title: "Grandparent", From: "open", To: "done"},
			StatusChange{ID: "tick-ppp111", Title: "Parent", From: "done", To: "cancelled"},
		)

		got := mergeStatusChanges(b1, b2)

		wantIDs := []string{"tick-ccc111", "tick-ppp111", "tick-ggg111"}
		if len(got) != len(wantIDs) {
			t.Fatalf("merged = %d rows, want %d: %+v", len(got), len(wantIDs), got)
		}
		for i, id := range wantIDs {
			if got[i].ID != id {
				t.Errorf("row %d = %s, want %s", i, got[i].ID, id)
			}
		}
		if got[0].Auto {
			t.Errorf("requested change should carry Auto=false, got %+v", got[0])
		}
	})

	t.Run("it keeps the first non-empty title when a task is seen twice", func(t *testing.T) {
		b1 := statusChangeBlock(true, StatusChange{ID: "tick-ppp111", Title: "Parent", From: "done", To: "open"})
		b2 := statusChangeBlock(true, StatusChange{ID: "tick-ppp111", From: "open", To: "cancelled"})

		got := mergeStatusChanges(b1, b2)

		if len(got) != 1 {
			t.Fatalf("merged = %d rows, want 1", len(got))
		}
		if got[0].Title != "Parent" {
			t.Errorf("Title = %q, want %q", got[0].Title, "Parent")
		}
	})

	t.Run("it returns an empty non-nil slice when nothing changed", func(t *testing.T) {
		got := mergeStatusChanges()
		if got == nil {
			t.Fatal("mergeStatusChanges() = nil, want empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("mergeStatusChanges() = %+v, want no rows", got)
		}

		noop := statusChangeBlock(true, StatusChange{ID: "tick-ppp111", Title: "Parent", From: "done", To: "done"})
		if noop.Changed == nil {
			t.Fatal("rows() = nil, want empty non-nil slice")
		}
		if len(noop.Changed) != 0 {
			t.Errorf("rows() = %+v, want no rows", noop.Changed)
		}
	})

	t.Run("it leaves a single block's rows unchanged", func(t *testing.T) {
		b := buildCascadeResult("tick-ccc111", "Child", primaryResult, cascades, allTasks, false)

		got := mergeStatusChanges(b)

		if !reflect.DeepEqual(got, b.Changed) {
			t.Errorf("merged = %+v, want %+v", got, b.Changed)
		}
	})
}
