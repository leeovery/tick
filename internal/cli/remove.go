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

// blastRadius holds pre-computed cascade and dependency information for the confirmation prompt.
type blastRadius struct {
	targetTasks   []idTitle
	cascadedTasks []idTitle
	affectedDeps  []idTitle
}

// idTitle pairs a task ID with its title for display.
type idTitle struct {
	id    string
	title string
}

// computeBlastRadius queries the store to validate IDs, compute cascade descendants,
// and identify surviving tasks with dependency references to be cleaned.
func computeBlastRadius(store interface {
	Query(func(*sql.DB) error) error
}, ids []string) (blastRadius, error) {
	var br blastRadius

	err := store.Query(func(db *sql.DB) error {
		// Validate all IDs exist (all-or-nothing).
		for _, id := range ids {
			var title string
			if err := db.QueryRow("SELECT title FROM tasks WHERE id = ?", id).Scan(&title); err != nil {
				return fmt.Errorf("task '%s' not found", id)
			}
			br.targetTasks = append(br.targetTasks, idTitle{id: id, title: title})
		}

		// Build the initial remove set from explicit targets.
		removeSet := make(map[string]bool, len(ids))
		for _, id := range ids {
			removeSet[id] = true
		}

		// Expand with transitive descendants via parent column.
		// Read all tasks with parents once, then iterate until stable.
		type taskInfo struct {
			id, title, parent string
		}
		var allTasks []taskInfo
		rows, err := db.Query("SELECT id, title, COALESCE(parent, '') FROM tasks")
		if err != nil {
			return fmt.Errorf("failed to query tasks: %w", err)
		}
		for rows.Next() {
			var ti taskInfo
			if err := rows.Scan(&ti.id, &ti.title, &ti.parent); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan task: %w", err)
			}
			allTasks = append(allTasks, ti)
		}
		rows.Close()

		// Iteratively expand until no new descendants found.
		changed := true
		for changed {
			changed = false
			for _, ti := range allTasks {
				normID := task.NormalizeID(ti.id)
				normParent := task.NormalizeID(ti.parent)
				if ti.parent != "" && removeSet[normParent] && !removeSet[normID] {
					removeSet[normID] = true
					br.cascadedTasks = append(br.cascadedTasks, idTitle{id: ti.id, title: ti.title})
					changed = true
				}
			}
		}

		// Find surviving tasks whose BlockedBy references any removed ID.
		depRows, err := db.Query("SELECT task_id, blocked_by FROM dependencies")
		if err != nil {
			return fmt.Errorf("failed to query dependencies: %w", err)
		}
		affectedSet := make(map[string]bool)
		for depRows.Next() {
			var taskID, blockedBy string
			if err := depRows.Scan(&taskID, &blockedBy); err != nil {
				depRows.Close()
				return fmt.Errorf("failed to scan dependency: %w", err)
			}
			normTaskID := task.NormalizeID(taskID)
			normBlockedBy := task.NormalizeID(blockedBy)
			if removeSet[normBlockedBy] && !removeSet[normTaskID] && !affectedSet[normTaskID] {
				affectedSet[normTaskID] = true
				var title string
				_ = db.QueryRow("SELECT title FROM tasks WHERE id = ?", taskID).Scan(&title)
				br.affectedDeps = append(br.affectedDeps, idTitle{id: taskID, title: title})
			}
		}
		depRows.Close()

		return nil
	})

	return br, err
}

// RunRemove executes the remove command: parses args, validates all IDs exist (all-or-nothing),
// expands the remove set with cascade descendants, prompts for confirmation (unless --force),
// filters targets from the task slice, cleans up dependency references on surviving tasks,
// and outputs the result through the formatter.
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
		br, err := computeBlastRadius(store, ids)
		if err != nil {
			return err
		}
		if err := confirmRemovalWithCascade(br, stdin, stderr); err != nil {
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

		// Build initial remove set from explicit targets.
		removeSet := make(map[string]bool, len(ids))
		for _, id := range ids {
			removeSet[id] = true
		}

		// Expand remove set with all transitive descendants.
		removeSet = collectDescendants(removeSet, tasks)

		// Collect removal info from the expanded set.
		for id := range removeSet {
			if idx, ok := existingIDs[id]; ok {
				result.Removed = append(result.Removed, RemovedTask{
					ID:    tasks[idx].ID,
					Title: tasks[idx].Title,
				})
			}
		}

		// Filter out removed tasks.
		filtered := make([]task.Task, 0, len(tasks)-len(removeSet))
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

// confirmRemovalWithCascade shows the confirmation prompt with full blast radius information.
// Returns nil if the user confirms, or errAborted if they decline.
func confirmRemovalWithCascade(br blastRadius, stdin io.Reader, stderr io.Writer) error {
	hasCascade := len(br.cascadedTasks) > 0
	hasDeps := len(br.affectedDeps) > 0

	if !hasCascade && !hasDeps && len(br.targetTasks) == 1 {
		// Simple prompt: single target, no cascade, no dep impact.
		fmt.Fprintf(stderr, "Remove task %s %q? [y/N] ", br.targetTasks[0].id, br.targetTasks[0].title)
	} else if len(br.targetTasks) == 1 && !hasCascade && hasDeps {
		// Single target with dep impact only.
		fmt.Fprintf(stderr, "Remove task %s %q?\n", br.targetTasks[0].id, br.targetTasks[0].title)
		fmt.Fprintf(stderr, "Will update dependencies on:\n")
		for _, d := range br.affectedDeps {
			fmt.Fprintf(stderr, "  %s %q\n", d.id, d.title)
		}
		fmt.Fprint(stderr, "Proceed? [y/N] ")
	} else {
		// Multi-target or cascade scenario.
		if len(br.targetTasks) == 1 {
			fmt.Fprintf(stderr, "Remove task %s %q?\n", br.targetTasks[0].id, br.targetTasks[0].title)
		} else {
			fmt.Fprintf(stderr, "Remove %d tasks?\n", len(br.targetTasks))
			for _, t := range br.targetTasks {
				fmt.Fprintf(stderr, "  %s %q\n", t.id, t.title)
			}
		}
		if hasCascade {
			fmt.Fprintf(stderr, "Will also remove descendants:\n")
			for _, c := range br.cascadedTasks {
				fmt.Fprintf(stderr, "  %s %q\n", c.id, c.title)
			}
		}
		if hasDeps {
			fmt.Fprintf(stderr, "Will update dependencies on:\n")
			for _, d := range br.affectedDeps {
				fmt.Fprintf(stderr, "  %s %q\n", d.id, d.title)
			}
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

// collectDescendants returns the union of targetIDs and all their transitive descendants
// found in the given task slice. It builds a parent-to-children index and walks recursively.
// ID comparison uses task.NormalizeID for case-insensitive matching.
func collectDescendants(targetIDs map[string]bool, tasks []task.Task) map[string]bool {
	result := make(map[string]bool, len(targetIDs))
	for id := range targetIDs {
		result[id] = true
	}

	if len(targetIDs) == 0 {
		return result
	}

	// Build parent -> children index using normalized IDs.
	childrenOf := make(map[string][]string)
	for i := range tasks {
		if tasks[i].Parent == "" {
			continue
		}
		parentNorm := task.NormalizeID(tasks[i].Parent)
		childNorm := task.NormalizeID(tasks[i].ID)
		childrenOf[parentNorm] = append(childrenOf[parentNorm], childNorm)
	}

	// Recursively collect descendants for each target.
	var walk func(id string)
	walk = func(id string) {
		for _, child := range childrenOf[id] {
			if !result[child] {
				result[child] = true
				walk(child)
			}
		}
	}

	for id := range targetIDs {
		walk(id)
	}

	return result
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
