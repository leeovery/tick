package cli

import (
	"fmt"
	"strings"

	"github.com/leeovery/tick/internal/store"
	"github.com/leeovery/tick/internal/task"
)

// runTransition implements the shared handler for tick start, done, cancel, and reopen commands.
// It parses the ID, loads the task, applies the transition, persists via storage, and outputs the result.
func (a *App) runTransition(command string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
	}

	id := task.NormalizeID(strings.TrimSpace(args[0]))

	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := store.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	var result *task.TransitionResult

	err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find the task by ID
		idx := -1
		for i := range tasks {
			if tasks[i].ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", id)
		}

		// Apply the transition
		r, err := task.Transition(&tasks[idx], command)
		if err != nil {
			return nil, err
		}
		result = r

		return tasks, nil
	})
	if err != nil {
		return err
	}

	// Output
	if !a.Quiet {
		fmt.Fprintf(a.Stdout, "%s: %s \u2192 %s\n", id, result.OldStatus, result.NewStatus)
	}

	return nil
}
