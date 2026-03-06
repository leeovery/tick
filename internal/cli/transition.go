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
		if len(cascades) == 0 {
			fmt.Fprintln(stdout, fmtr.FormatTransition(id, string(result.OldStatus), string(result.NewStatus)))
		} else {
			cr := buildCascadeResult(id, targetTitle, result, cascades, allTasks)
			fmt.Fprintln(stdout, fmtr.FormatCascadeTransition(cr))
		}
	}

	return nil
}

// buildCascadeResult constructs a CascadeResult from the primary transition, cascade
// changes, and the full task list. It populates Unchanged with children of the primary
// task that are in terminal states and were not part of the cascade changes.
func buildCascadeResult(id, title string, result task.TransitionResult, cascades []task.CascadeChange, tasks []task.Task) CascadeResult {
	cr := CascadeResult{
		TaskID:    id,
		TaskTitle: title,
		OldStatus: string(result.OldStatus),
		NewStatus: string(result.NewStatus),
	}

	// Build set of cascaded task IDs for quick lookup.
	cascadedIDs := make(map[string]bool, len(cascades))
	for _, c := range cascades {
		cascadedIDs[task.NormalizeID(c.Task.ID)] = true
		cr.Cascaded = append(cr.Cascaded, CascadeEntry{
			ID:        c.Task.ID,
			Title:     c.Task.Title,
			OldStatus: string(c.OldStatus),
			NewStatus: string(c.NewStatus),
		})
	}

	// Populate Unchanged: children of the primary task that are terminal and not cascaded.
	normalizedID := task.NormalizeID(id)
	for i := range tasks {
		if task.NormalizeID(tasks[i].Parent) != normalizedID {
			continue
		}
		childNID := task.NormalizeID(tasks[i].ID)
		if cascadedIDs[childNID] {
			continue
		}
		if tasks[i].Status == task.StatusDone || tasks[i].Status == task.StatusCancelled {
			cr.Unchanged = append(cr.Unchanged, UnchangedEntry{
				ID:     tasks[i].ID,
				Title:  tasks[i].Title,
				Status: string(tasks[i].Status),
			})
		}
	}

	return cr
}
