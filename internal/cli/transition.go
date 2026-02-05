package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// runTransition executes a status transition command (start, done, cancel, reopen).
func (a *App) runTransition(command string, args []string) int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Parse task ID from args (args[0] = "tick", args[1] = command, args[2] = ID)
	if len(args) < 3 {
		fmt.Fprintf(a.Stderr, "Error: Task ID is required. Usage: tick %s <id>\n", command)
		return 1
	}

	// Normalize ID to lowercase
	taskID := task.NormalizeID(args[2])

	// Open store
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	var transitionResult task.TransitionResult

	// Execute mutation
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task by ID
		var targetTask *task.Task
		for i := range tasks {
			if tasks[i].ID == taskID {
				targetTask = &tasks[i]
				break
			}
		}

		if targetTask == nil {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		// Apply transition
		result, err := task.Transition(targetTask, command)
		if err != nil {
			return nil, err
		}

		transitionResult = result
		return tasks, nil
	})

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output (unless --quiet)
	if !a.formatConfig.Quiet {
		formatter := a.formatConfig.Formatter()
		fmt.Fprint(a.Stdout, formatter.FormatTransition(taskID, string(transitionResult.OldStatus), string(transitionResult.NewStatus)))
	}

	return 0
}
