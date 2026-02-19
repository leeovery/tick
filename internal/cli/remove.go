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

// RunRemove executes the remove command: parses args, validates all IDs exist (all-or-nothing),
// prompts for confirmation (unless --force), filters targets from the task slice,
// cleans up dependency references on surviving tasks, and outputs the result through the formatter.
func RunRemove(dir string, fc FormatConfig, fmtr Formatter, args []string, stdin io.Reader, stderr io.Writer, stdout io.Writer) error {
	ids, force := parseRemoveArgs(args)

	if len(ids) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick remove <id> [<id>...]")
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	if !force {
		if err := confirmRemoval(store, ids, stdin, stderr); err != nil {
			return err
		}
	}

	var result RemovalResult

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build lookup of existing task IDs for O(1) validation.
		existingIDs := make(map[string]int, len(tasks))
		for i := range tasks {
			existingIDs[task.NormalizeID(tasks[i].ID)] = i
		}

		// Validate all IDs exist before any mutation (all-or-nothing).
		for _, id := range ids {
			if _, ok := existingIDs[id]; !ok {
				return nil, fmt.Errorf("task '%s' not found", id)
			}
		}

		// Build remove set for O(1) filtering and collect removal info.
		removeSet := make(map[string]bool, len(ids))
		for _, id := range ids {
			idx := existingIDs[id]
			result.Removed = append(result.Removed, RemovedTask{
				ID:    tasks[idx].ID,
				Title: tasks[idx].Title,
			})
			removeSet[id] = true
		}

		// Filter out removed tasks.
		filtered := make([]task.Task, 0, len(tasks)-len(ids))
		for i := range tasks {
			if !removeSet[task.NormalizeID(tasks[i].ID)] {
				filtered = append(filtered, tasks[i])
			}
		}

		// Strip all removed IDs from surviving tasks' BlockedBy arrays.
		for i := range filtered {
			cleaned := stripIDsFromBlockedBy(filtered[i].BlockedBy, removeSet)
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

// stripIDsFromBlockedBy returns a new slice with all IDs in removeSet removed.
func stripIDsFromBlockedBy(blockedBy []string, removeSet map[string]bool) []string {
	var result []string
	for _, dep := range blockedBy {
		if !removeSet[task.NormalizeID(dep)] {
			result = append(result, dep)
		}
	}
	return result
}

// confirmRemoval looks up task titles for all IDs and prompts the user for confirmation.
// Returns nil if the user confirms, or errAborted if they decline.
func confirmRemoval(store interface {
	Query(func(*sql.DB) error) error
}, ids []string, stdin io.Reader, stderr io.Writer) error {
	// Look up each task's title for the prompt.
	type idTitle struct {
		id    string
		title string
	}
	targets := make([]idTitle, 0, len(ids))
	for _, id := range ids {
		var title string
		err := store.Query(func(db *sql.DB) error {
			return db.QueryRow("SELECT title FROM tasks WHERE id = ?", id).Scan(&title)
		})
		if err != nil {
			return fmt.Errorf("task '%s' not found", id)
		}
		targets = append(targets, idTitle{id: id, title: title})
	}

	if len(targets) == 1 {
		fmt.Fprintf(stderr, "Remove task %s %q? [y/N] ", targets[0].id, targets[0].title)
	} else {
		fmt.Fprintf(stderr, "Remove %d tasks?\n", len(targets))
		for _, t := range targets {
			fmt.Fprintf(stderr, "  %s %q\n", t.id, t.title)
		}
		fmt.Fprint(stderr, "Proceed? [y/N] ")
	}

	line, _ := bufio.NewReader(stdin).ReadString('\n')
	response := strings.ToLower(strings.TrimSpace(line))

	if response == "y" || response == "yes" {
		return nil
	}

	fmt.Fprintln(stderr, "Aborted.")
	return errAborted
}
