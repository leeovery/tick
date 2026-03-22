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

	var result task.TransitionResult
	var cascadeResult *CascadeResult
	var sm task.StateMachine

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		for i := range tasks {
			if tasks[i].ID == id {
				r, c, mutErr := sm.ApplyUserTransition(tasks, &tasks[i], command)
				if mutErr != nil {
					return nil, mutErr
				}
				result = r
				if len(c) > 0 {
					cr := buildCascadeResult(id, tasks[i].Title, r, c, tasks)
					cascadeResult = &cr
				}
				return tasks, nil
			}
		}
		return nil, fmt.Errorf("task '%s' not found", id)
	})
	if err != nil {
		return err
	}

	if !fc.Quiet {
		outputTransitionOrCascade(stdout, fmtr, id, string(result.OldStatus), string(result.NewStatus), cascadeResult)
	}

	return nil
}

// buildCascadeResult constructs a CascadeResult from the primary transition, cascade
// changes, and the full task list. It populates ParentID on each cascade entry from the
// task's Parent field.
func buildCascadeResult(id, title string, result task.TransitionResult, cascades []task.CascadeChange, tasks []task.Task) CascadeResult {
	cr := CascadeResult{
		TaskID:    id,
		TaskTitle: title,
		OldStatus: string(result.OldStatus),
		NewStatus: string(result.NewStatus),
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
