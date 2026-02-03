package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// runTransition implements the shared flow for start, done, cancel, and reopen commands.
func (a *App) runTransition(command string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
	}

	id := task.NormalizeID(args[0])

	// Discover .tick/ directory
	tickDir, err := DiscoverTickDir(a.workDir)
	if err != nil {
		return err
	}

	// Open storage
	store, err := storage.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var oldStatus, newStatus task.Status

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find the task by normalized ID
		idx := -1
		for i := range tasks {
			if task.NormalizeID(tasks[i].ID) == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", id)
		}

		// Apply transition
		old, new, err := task.Transition(&tasks[idx], command)
		if err != nil {
			return nil, err
		}

		oldStatus = old
		newStatus = new

		return tasks, nil
	})
	if err != nil {
		return unwrapMutationError(err)
	}

	// Output
	if !a.config.Quiet {
		fmt.Fprintf(a.stdout, "%s: %s \u2192 %s\n", id, oldStatus, newStatus)
	}

	return nil
}
