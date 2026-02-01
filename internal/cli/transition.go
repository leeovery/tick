package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// cmdTransition implements start, done, cancel, and reopen subcommands.
func (a *App) cmdTransition(workDir string, args []string, command string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
	}

	taskID := task.NormalizeID(args[0])

	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	var result task.TransitionResult

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		for i := range tasks {
			if tasks[i].ID == taskID {
				r, err := task.Transition(&tasks[i], command)
				if err != nil {
					return nil, err
				}
				result = r
				return tasks, nil
			}
		}
		return nil, fmt.Errorf("task '%s' not found", taskID)
	})
	if err != nil {
		return err
	}

	if !a.opts.Quiet {
		return a.fmtr.FormatTransition(a.stdout, TransitionData{
			ID:        taskID,
			OldStatus: string(result.OldStatus),
			NewStatus: string(result.NewStatus),
		})
	}

	return nil
}
