package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// handleDep implements the dep subcommand, routing to add/rm sub-subcommands.
func (a *App) handleDep(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}

	if len(subArgs) == 0 {
		return fmt.Errorf("sub-command required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>")
	}

	subCmd := subArgs[0]
	rest := subArgs[1:]

	switch subCmd {
	case "add":
		return RunDepAdd(dir, fc, fmtr, rest, a.Stdout)
	case "rm":
		return RunDepRm(dir, fc, fmtr, rest, a.Stdout)
	default:
		return fmt.Errorf("unknown dep sub-command '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>", subCmd)
	}
}

// parseDepArgs extracts two positional IDs from the args, normalizing to lowercase.
// Returns taskID, blockedByID, and error.
func parseDepArgs(args []string, subCmd string) (string, string, error) {
	var positional []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		positional = append(positional, arg)
	}

	if len(positional) < 2 {
		return "", "", fmt.Errorf("two IDs required. Usage: tick dep %s <task_id> <blocked_by_id>", subCmd)
	}

	taskID := task.NormalizeID(positional[0])
	blockedByID := task.NormalizeID(positional[1])

	return taskID, blockedByID, nil
}

// RunDepAdd executes the dep add command: validates inputs, adds the dependency,
// persists via the storage engine, and outputs confirmation via the Formatter.
func RunDepAdd(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	taskID, blockedByID, err := parseDepArgs(args, "add")
	if err != nil {
		return err
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task_id.
		taskIdx := -1
		for i := range tasks {
			if tasks[i].ID == taskID {
				taskIdx = i
				break
			}
		}
		if taskIdx == -1 {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		// Find blocked_by_id.
		blockedByFound := false
		for i := range tasks {
			if tasks[i].ID == blockedByID {
				blockedByFound = true
				break
			}
		}
		if !blockedByFound {
			return nil, fmt.Errorf("task '%s' not found", blockedByID)
		}

		// Check duplicate.
		for _, dep := range tasks[taskIdx].BlockedBy {
			if dep == blockedByID {
				return nil, fmt.Errorf("dependency already exists: %s is already blocked by %s", taskID, blockedByID)
			}
		}

		// Validate dependency (self-ref, cycle, child-blocked-by-parent).
		if err := task.ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return nil, err
		}

		// Add dependency and update timestamp.
		tasks[taskIdx].BlockedBy = append(tasks[taskIdx].BlockedBy, blockedByID)
		tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return err
	}

	if !fc.Quiet {
		fmt.Fprintln(stdout, fmtr.FormatDepChange("added", taskID, blockedByID))
	}

	return nil
}

// RunDepRm executes the dep rm command: finds the task, removes the dependency
// from blocked_by, persists via the storage engine, and outputs confirmation via the Formatter.
func RunDepRm(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	taskID, blockedByID, err := parseDepArgs(args, "rm")
	if err != nil {
		return err
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task_id.
		taskIdx := -1
		for i := range tasks {
			if tasks[i].ID == taskID {
				taskIdx = i
				break
			}
		}
		if taskIdx == -1 {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		// Check blocked_by_id exists in the blocked_by array.
		depIdx := -1
		for i, dep := range tasks[taskIdx].BlockedBy {
			if dep == blockedByID {
				depIdx = i
				break
			}
		}
		if depIdx == -1 {
			return nil, fmt.Errorf("%s is not a dependency of %s", blockedByID, taskID)
		}

		// Remove from blocked_by (preserve order).
		tasks[taskIdx].BlockedBy = append(
			tasks[taskIdx].BlockedBy[:depIdx],
			tasks[taskIdx].BlockedBy[depIdx+1:]...,
		)
		tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return err
	}

	if !fc.Quiet {
		fmt.Fprintln(stdout, fmtr.FormatDepChange("removed", taskID, blockedByID))
	}

	return nil
}
