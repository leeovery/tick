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
	var cascades []task.CascadeChange
	var targetTitle string
	var allTasks []task.Task
	var sm task.StateMachine

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		for i := range tasks {
			if tasks[i].ID == id {
				r, c, err := sm.ApplyWithCascades(tasks, &tasks[i], command)
				if err != nil {
					return nil, err
				}
				result = r
				cascades = c
				targetTitle = tasks[i].Title
				allTasks = tasks
				return tasks, nil
			}
		}
		return nil, fmt.Errorf("task '%s' not found", id)
	})
	if err != nil {
		return err
	}

	if !fc.Quiet {
		outputTransitionOrCascade(stdout, fmtr, id, targetTitle, result, cascades, allTasks)
	}

	return nil
}

// buildCascadeResult constructs a CascadeResult from the primary transition, cascade
// changes, and the full task list. It populates ParentID on each cascade entry from the
// task's Parent field, and collects unchanged terminal descendants recursively (all levels,
// not just direct children).
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

	// Build set of cascaded task IDs for quick lookup.
	cascadedIDs := make(map[string]bool, len(cascades))
	for _, c := range cascades {
		cascadedIDs[task.NormalizeID(c.Task.ID)] = true
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

	// Build set of all involved task IDs (primary + cascaded) for descendant walking.
	involvedIDs := make(map[string]bool, len(cascades)+1)
	involvedIDs[task.NormalizeID(id)] = true
	for nid := range cascadedIDs {
		involvedIDs[nid] = true
	}

	// Populate Unchanged: descendants of involved tasks that are terminal and not cascaded.
	for i := range tasks {
		parentNID := task.NormalizeID(tasks[i].Parent)
		if !involvedIDs[parentNID] {
			continue
		}
		childNID := task.NormalizeID(tasks[i].ID)
		if cascadedIDs[childNID] {
			continue
		}
		if tasks[i].Status == task.StatusDone || tasks[i].Status == task.StatusCancelled {
			cr.Unchanged = append(cr.Unchanged, UnchangedEntry{
				ID:       tasks[i].ID,
				Title:    tasks[i].Title,
				ParentID: tasks[i].Parent,
				Status:   string(tasks[i].Status),
			})
		}
	}

	return cr
}
