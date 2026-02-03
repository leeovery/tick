package cli

import (
	"fmt"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// runDep dispatches the dep subcommands: add and rm.
func (a *App) runDep(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Usage: tick dep <add|rm> <task_id> <blocked_by_id>")
	}

	subcmd := args[0]
	cmdArgs := args[1:]

	switch subcmd {
	case "add":
		return a.runDepAdd(cmdArgs)
	case "rm":
		return a.runDepRm(cmdArgs)
	default:
		return fmt.Errorf("Unknown dep subcommand '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>", subcmd)
	}
}

// runDepAdd implements `tick dep add <task_id> <blocked_by_id>`.
func (a *App) runDepAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Two task IDs required. Usage: tick dep add <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(args[0])
	blockedByID := task.NormalizeID(args[1])

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

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task by ID
		taskIdx := -1
		blockedByExists := false
		for i := range tasks {
			normalizedID := task.NormalizeID(tasks[i].ID)
			if normalizedID == taskID {
				taskIdx = i
			}
			if normalizedID == blockedByID {
				blockedByExists = true
			}
		}

		if taskIdx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", taskID)
		}
		if !blockedByExists {
			return nil, fmt.Errorf("Task '%s' not found", blockedByID)
		}

		// Check for duplicate dependency
		for _, existing := range tasks[taskIdx].BlockedBy {
			if task.NormalizeID(existing) == blockedByID {
				return nil, fmt.Errorf("Task '%s' is already blocked by '%s'", taskID, blockedByID)
			}
		}

		// Validate dependency (self-ref, cycle, child-blocked-by-parent)
		if err := task.ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return nil, err
		}

		// Add dependency and update timestamp
		tasks[taskIdx].BlockedBy = append(tasks[taskIdx].BlockedBy, blockedByID)
		tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return unwrapMutationError(err)
	}

	// Output
	if !a.config.Quiet {
		return a.formatter.FormatDepChange(a.stdout, "added", taskID, blockedByID)
	}

	return nil
}

// runDepRm implements `tick dep rm <task_id> <blocked_by_id>`.
func (a *App) runDepRm(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Two task IDs required. Usage: tick dep rm <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(args[0])
	blockedByID := task.NormalizeID(args[1])

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

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task by ID
		taskIdx := -1
		for i := range tasks {
			if task.NormalizeID(tasks[i].ID) == taskID {
				taskIdx = i
				break
			}
		}

		if taskIdx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", taskID)
		}

		// Find and remove the dependency from blocked_by (by array membership, not task existence)
		found := false
		newBlockedBy := make([]string, 0, len(tasks[taskIdx].BlockedBy))
		for _, dep := range tasks[taskIdx].BlockedBy {
			if task.NormalizeID(dep) == blockedByID {
				found = true
				continue
			}
			newBlockedBy = append(newBlockedBy, dep)
		}

		if !found {
			return nil, fmt.Errorf("Task '%s' is not blocked by '%s'", taskID, blockedByID)
		}

		// Update blocked_by and timestamp
		tasks[taskIdx].BlockedBy = newBlockedBy
		tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return unwrapMutationError(err)
	}

	// Output
	if !a.config.Quiet {
		return a.formatter.FormatDepChange(a.stdout, "removed", taskID, blockedByID)
	}

	return nil
}
