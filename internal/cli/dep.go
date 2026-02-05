package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// runDep executes the dep subcommand with its sub-subcommands (add, rm).
func (a *App) runDep(args []string) int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// args[0] = "tick", args[1] = "dep", args[2] = subcommand
	if len(args) < 3 {
		fmt.Fprintf(a.Stderr, "Error: Missing dep subcommand. Usage: tick dep <add|rm> <task_id> <blocked_by_id>\n")
		return 1
	}

	subcommand := args[2]

	switch subcommand {
	case "add":
		return a.runDepAdd(tickDir, args)
	case "rm":
		return a.runDepRm(tickDir, args)
	default:
		fmt.Fprintf(a.Stderr, "Error: Unknown dep subcommand '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>\n", subcommand)
		return 1
	}
}

// runDepAdd executes the dep add subcommand.
func (a *App) runDepAdd(tickDir string, args []string) int {
	// args[0] = "tick", args[1] = "dep", args[2] = "add", args[3] = task_id, args[4] = blocked_by_id
	if len(args) < 5 {
		fmt.Fprintf(a.Stderr, "Error: Two IDs required. Usage: tick dep add <task_id> <blocked_by_id>\n")
		return 1
	}

	taskID := task.NormalizeID(args[3])
	blockedByID := task.NormalizeID(args[4])

	// Open store
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	// Execute mutation
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID lookup
		idSet := make(map[string]bool)
		for _, t := range tasks {
			idSet[t.ID] = true
		}

		// Validate task_id exists
		if !idSet[taskID] {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		// Validate blocked_by_id exists
		if !idSet[blockedByID] {
			return nil, fmt.Errorf("task '%s' not found", blockedByID)
		}

		// Find the task
		var targetTask *task.Task
		for i := range tasks {
			if tasks[i].ID == taskID {
				targetTask = &tasks[i]
				break
			}
		}

		// Check for duplicate dependency
		for _, existingBlocker := range targetTask.BlockedBy {
			if task.NormalizeID(existingBlocker) == blockedByID {
				return nil, fmt.Errorf("task '%s' is already blocked by '%s'", taskID, blockedByID)
			}
		}

		// Validate dependency (cycle detection, child-blocked-by-parent)
		if err := task.ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return nil, err
		}

		// Add dependency and update timestamp
		targetTask.BlockedBy = append(targetTask.BlockedBy, blockedByID)
		_, updated := task.DefaultTimestamps()
		targetTask.Updated = updated

		return tasks, nil
	})

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output (unless --quiet)
	if !a.flags.Quiet {
		fmt.Fprintf(a.Stdout, "Dependency added: %s blocked by %s\n", taskID, blockedByID)
	}

	return 0
}

// runDepRm executes the dep rm subcommand.
func (a *App) runDepRm(tickDir string, args []string) int {
	// args[0] = "tick", args[1] = "dep", args[2] = "rm", args[3] = task_id, args[4] = blocked_by_id
	if len(args) < 5 {
		fmt.Fprintf(a.Stderr, "Error: Two IDs required. Usage: tick dep rm <task_id> <blocked_by_id>\n")
		return 1
	}

	taskID := task.NormalizeID(args[3])
	blockedByID := task.NormalizeID(args[4])

	// Open store
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	// Execute mutation
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find task_id (note: we do NOT validate blocked_by_id exists as a task - supports removing stale refs)
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

		// Check if blocked_by_id is in the array
		found := false
		newBlockedBy := make([]string, 0, len(targetTask.BlockedBy))
		for _, existingBlocker := range targetTask.BlockedBy {
			if task.NormalizeID(existingBlocker) == blockedByID {
				found = true
				continue // Skip this one (remove it)
			}
			newBlockedBy = append(newBlockedBy, existingBlocker)
		}

		if !found {
			return nil, fmt.Errorf("task '%s' is not blocked by '%s'", taskID, blockedByID)
		}

		// Update blocked_by and timestamp
		targetTask.BlockedBy = newBlockedBy
		_, updated := task.DefaultTimestamps()
		targetTask.Updated = updated

		return tasks, nil
	})

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output (unless --quiet)
	if !a.flags.Quiet {
		fmt.Fprintf(a.Stdout, "Dependency removed: %s no longer blocked by %s\n", taskID, blockedByID)
	}

	return 0
}
