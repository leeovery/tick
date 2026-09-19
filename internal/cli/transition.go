package cli

import (
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/task"
)

// RunTransition executes a status transition command (start, done, cancel, reopen).
// It resolves the task ID (supporting partial prefixes), looks up the task, applies the
// transition and any cascading status changes, persists all changes atomically, and
// outputs the result via the Formatter.
func RunTransition(dir string, command string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick %s <id>", command)
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	id, err := store.ResolveID(args[0])
	if err != nil {
		return err
	}

	var cascadeResult *CascadeResult
	var sm task.StateMachine

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		for i := range tasks {
			if tasks[i].ID == id {
				r, c, mutErr := sm.ApplyUserTransition(tasks, &tasks[i], command)
				if mutErr != nil {
					return nil, mutErr
				}
				cr := buildCascadeResult(id, tasks[i].Title, r, c, tasks, false)
				cascadeResult = &cr
				return tasks, nil
			}
		}
		return nil, fmt.Errorf("task '%s' not found", id)
	})
	if err != nil {
		return err
	}

	if !fc.Quiet {
		outputStatusChanges(stdout, fmtr, *cascadeResult)
	}

	return nil
}

// buildCascadeResult constructs a CascadeResult from the primary transition, cascade
// changes, and the full task list. It populates ParentID on each cascade entry from the
// task's Parent field. primaryAuto is false only when the caller asked for the primary
// transition; cascades are always auto.
func buildCascadeResult(id, title string, result task.TransitionResult, cascades []task.CascadeChange, tasks []task.Task, primaryAuto bool) CascadeResult {
	cr := CascadeResult{
		TaskID:      id,
		TaskTitle:   title,
		OldStatus:   string(result.OldStatus),
		NewStatus:   string(result.NewStatus),
		PrimaryAuto: primaryAuto,
	}

	// Detect upward cascade: if any cascaded task is the primary task's parent,
	// the cascade went upward (child start triggers parent/grandparent start).
	// Find the primary task's parent.
	var primaryParent string
	for i := range tasks {
		if task.NormalizeID(tasks[i].ID) == task.NormalizeID(id) {
			primaryParent = tasks[i].Parent
			break
		}
	}
	isUpward := false
	for _, c := range cascades {
		if task.NormalizeID(c.Task.ID) == task.NormalizeID(primaryParent) {
			isUpward = true
			break
		}
	}

	for _, c := range cascades {
		parentID := c.Task.Parent
		if isUpward {
			// Upward cascades render flat: all entries are roots relative to the primary task.
			parentID = id
		}
		cr.Cascaded = append(cr.Cascaded, CascadeEntry{
			ID:        c.Task.ID,
			Title:     c.Task.Title,
			ParentID:  parentID,
			OldStatus: string(c.OldStatus),
			NewStatus: string(c.NewStatus),
		})
	}

	return cr
}

// statusChangeSet accumulates status changes keyed by task ID, collapsing repeat
// movements of one task into a single row reading from its first From to its last To.
type statusChangeSet struct {
	order []string
	byID  map[string]StatusChange
}

// add merges a change into any change already held for the same task, so a task moved
// twice reads from the status it held first to the status it holds last.
func (s *statusChangeSet) add(c StatusChange) {
	key := task.NormalizeID(c.ID)
	stored, seen := s.byID[key]
	if !seen {
		if s.byID == nil {
			s.byID = make(map[string]StatusChange)
		}
		s.order = append(s.order, key)
		s.byID[key] = c
		return
	}
	stored.To = c.To
	if stored.Title == "" {
		stored.Title = c.Title
	}
	stored.Auto = stored.Auto && c.Auto
	s.byID[key] = stored
}

// rows returns the accumulated changes in first-seen order, dropping any task that
// ends where it started. Never nil, so JSON renders [] rather than null.
func (s *statusChangeSet) rows() []StatusChange {
	rows := make([]StatusChange, 0, len(s.order))
	for _, key := range s.order {
		if c := s.byID[key]; c.From != c.To {
			rows = append(rows, c)
		}
	}
	return rows
}

// mergeStatusChanges collapses the rows of several blocks into one set in which each
// task appears at most once. Rows hold only copied values, so blocks built inside a
// Mutate closure can be merged after it returns.
func mergeStatusChanges(blocks ...CascadeResult) []StatusChange {
	var set statusChangeSet
	for _, b := range blocks {
		for _, c := range b.Changed() {
			set.add(c)
		}
	}
	return set.rows()
}
