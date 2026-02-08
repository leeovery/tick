package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runDep implements the `tick dep` command with sub-subcommands `add` and `rm`.
func (a *App) runDep(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Subcommand required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>")
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "add":
		return a.runDepAdd(subArgs)
	case "rm":
		return a.runDepRm(subArgs)
	default:
		return fmt.Errorf("Unknown dep subcommand '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>", subcommand)
	}
}

// runDepAdd implements `tick dep add <task_id> <blocked_by_id>`.
// It validates both IDs exist, checks for self-ref, duplicate, cycle, and
// child-blocked-by-parent, then adds the dependency and persists.
func (a *App) runDepAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Two task IDs required. Usage: tick dep add <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(strings.TrimSpace(args[0]))
	blockedByID := task.NormalizeID(strings.TrimSpace(args[1]))

	// Self-reference check (before store access)
	if taskID == blockedByID {
		return fmt.Errorf("cannot add dependency - creates cycle: %s \u2192 %s", taskID, taskID)
	}

	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task_id
		taskIdx := -1
		for i := range tasks {
			if tasks[i].ID == taskID {
				taskIdx = i
				break
			}
		}
		if taskIdx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", taskID)
		}

		// Find blocked_by_id
		blockedByExists := false
		for _, t := range tasks {
			if t.ID == blockedByID {
				blockedByExists = true
				break
			}
		}
		if !blockedByExists {
			return nil, fmt.Errorf("Task '%s' not found", blockedByID)
		}

		// Check duplicate
		for _, dep := range tasks[taskIdx].BlockedBy {
			if dep == blockedByID {
				return nil, fmt.Errorf("Task '%s' is already blocked by '%s'", taskID, blockedByID)
			}
		}

		// Validate dependency (cycle + child-blocked-by-parent)
		if err := task.ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return nil, err
		}

		// Add dependency
		tasks[taskIdx].BlockedBy = append(tasks[taskIdx].BlockedBy, blockedByID)
		tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return err
	}

	// Output via formatter
	return a.Formatter.FormatDepChange(a.Stdout, taskID, blockedByID, "added", a.Quiet)
}

// runDepRm implements `tick dep rm <task_id> <blocked_by_id>`.
// It looks up task_id, checks blocked_by_id in array, removes it, and persists.
// Note: rm does NOT validate that blocked_by_id exists as a task -- only checks
// array membership (supports removing stale refs).
func (a *App) runDepRm(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Two task IDs required. Usage: tick dep rm <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(strings.TrimSpace(args[0]))
	blockedByID := task.NormalizeID(strings.TrimSpace(args[1]))

	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task_id
		taskIdx := -1
		for i := range tasks {
			if tasks[i].ID == taskID {
				taskIdx = i
				break
			}
		}
		if taskIdx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", taskID)
		}

		// Check blocked_by_id in array
		depIdx := -1
		for i, dep := range tasks[taskIdx].BlockedBy {
			if dep == blockedByID {
				depIdx = i
				break
			}
		}
		if depIdx == -1 {
			return nil, fmt.Errorf("Task '%s' is not blocked by '%s'", taskID, blockedByID)
		}

		// Remove dependency
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

	// Output via formatter
	return a.Formatter.FormatDepChange(a.Stdout, taskID, blockedByID, "removed", a.Quiet)
}
