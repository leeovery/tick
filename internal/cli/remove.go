package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

// parseRemoveArgs extracts the task ID and --force flag from remove command arguments.
// Returns the normalized task ID and whether --force was set.
func parseRemoveArgs(args []string) (string, bool) {
	var id string
	var force bool

	for _, arg := range args {
		switch {
		case arg == "--force" || arg == "-f":
			force = true
		case !strings.HasPrefix(arg, "-"):
			if id == "" {
				id = task.NormalizeID(arg)
			}
		}
	}

	return id, force
}

// RunRemove executes the remove command: parses args, locates the target task,
// filters it from the task slice, cleans up dependency references on surviving tasks,
// and outputs the result through the formatter.
func RunRemove(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	id, force := parseRemoveArgs(args)

	if id == "" {
		return fmt.Errorf("task ID is required. Usage: tick remove <id> [<id>...]")
	}

	if !force {
		return fmt.Errorf("remove requires --force flag (interactive confirmation not yet implemented)")
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	var result RemovalResult

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find the target task.
		targetIdx := -1
		for i := range tasks {
			if task.NormalizeID(tasks[i].ID) == id {
				targetIdx = i
				break
			}
		}
		if targetIdx == -1 {
			return nil, fmt.Errorf("task '%s' not found", id)
		}

		// Record removal info.
		result.Removed = []RemovedTask{
			{ID: tasks[targetIdx].ID, Title: tasks[targetIdx].Title},
		}

		// Filter out the target task.
		filtered := make([]task.Task, 0, len(tasks)-1)
		for i := range tasks {
			if i != targetIdx {
				filtered = append(filtered, tasks[i])
			}
		}

		// Strip removed ID from surviving tasks' BlockedBy arrays.
		for i := range filtered {
			cleaned := stripFromBlockedBy(filtered[i].BlockedBy, id)
			if len(cleaned) != len(filtered[i].BlockedBy) {
				result.DepsUpdated = append(result.DepsUpdated, filtered[i].ID)
				filtered[i].BlockedBy = cleaned
			}
		}

		return filtered, nil
	})
	if err != nil {
		return err
	}

	if !fc.Quiet {
		fmt.Fprintln(stdout, fmtr.FormatRemoval(result))
	}

	return nil
}

// stripFromBlockedBy returns a new slice with all occurrences of removedID removed.
func stripFromBlockedBy(blockedBy []string, removedID string) []string {
	var result []string
	for _, dep := range blockedBy {
		if task.NormalizeID(dep) != removedID {
			result = append(result, dep)
		}
	}
	return result
}
