package cli

import (
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// RunTransition executes a status transition command (start, done, cancel, reopen).
// It normalizes the task ID, looks up the task, applies the transition, persists the change,
// and outputs the transition line to stdout.
func RunTransition(dir string, command string, quiet bool, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick %s <id>", command)
	}

	id := task.NormalizeID(args[0])

	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var result task.TransitionResult

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		for i := range tasks {
			if tasks[i].ID == id {
				r, err := task.Transition(&tasks[i], command)
				if err != nil {
					return nil, err
				}
				result = r
				return tasks, nil
			}
		}
		return nil, fmt.Errorf("task '%s' not found", id)
	})
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Fprintf(stdout, "%s: %s \u2192 %s\n", id, result.OldStatus, result.NewStatus)
	}

	return nil
}
