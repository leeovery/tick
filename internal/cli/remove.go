package cli

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

// errAborted is a sentinel error returned when the user declines a confirmation prompt.
// App.Run detects this to return exit code 1 without the "Error: " prefix.
var errAborted = errors.New("aborted")

// parseRemoveArgs extracts task IDs and --force flag from remove command arguments.
// Returns deduplicated, normalized task IDs (preserving first-occurrence order) and whether --force was set.
func parseRemoveArgs(args []string) ([]string, bool) {
	var ids []string
	var force bool
	seen := map[string]bool{}

	for _, arg := range args {
		switch {
		case arg == "--force" || arg == "-f":
			force = true
		case strings.HasPrefix(arg, "-"):
			// Skip unknown flags.
		default:
			normalized := task.NormalizeID(arg)
			if !seen[normalized] {
				seen[normalized] = true
				ids = append(ids, normalized)
			}
		}
	}

	return ids, force
}

// RunRemove executes the remove command: parses args, locates the target task,
// prompts for confirmation (unless --force), filters it from the task slice,
// cleans up dependency references on surviving tasks, and outputs the result through the formatter.
func RunRemove(dir string, fc FormatConfig, fmtr Formatter, args []string, stdin io.Reader, stderr io.Writer, stdout io.Writer) error {
	ids, force := parseRemoveArgs(args)

	if len(ids) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick remove <id> [<id>...]")
	}

	// Tasks 3-2 through 3-5 extend to full slice; for now use first ID only.
	id := ids[0]

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	if !force {
		if err := confirmRemoval(store, id, stdin, stderr); err != nil {
			return err
		}
	}

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

// confirmRemoval looks up the task title and prompts the user for confirmation.
// Returns nil if the user confirms, or errAborted if they decline.
func confirmRemoval(store interface {
	Query(func(*sql.DB) error) error
}, id string, stdin io.Reader, stderr io.Writer) error {
	var title string
	err := store.Query(func(db *sql.DB) error {
		return db.QueryRow("SELECT title FROM tasks WHERE id = ?", id).Scan(&title)
	})
	if err != nil {
		return fmt.Errorf("task '%s' not found", id)
	}

	fmt.Fprintf(stderr, "Remove task %s %q? [y/N] ", id, title)

	line, _ := bufio.NewReader(stdin).ReadString('\n')
	response := strings.ToLower(strings.TrimSpace(line))

	if response == "y" || response == "yes" {
		return nil
	}

	fmt.Fprintln(stderr, "Aborted.")
	return errAborted
}
