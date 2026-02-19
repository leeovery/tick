package cli

import (
	"bufio"
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
	targetTasks   []RemovedTask
	cascadedTasks []RemovedTask
	affectedDeps  []RemovedTask
}

// executeRemoval validates IDs, expands descendants, computes blast radius, and optionally
// performs the removal. When computeOnly is true, it returns the original task slice unmodified
// (for confirmation prompts). When false, it filters removed tasks and cleans up dependencies.
func executeRemoval(tasks []task.Task, ids []string, computeOnly bool) ([]task.Task, blastRadius, RemovalResult, error) {
	var br blastRadius
	var result RemovalResult

	// Build lookup of existing task IDs for O(1) validation.
	existingIDs := make(map[string]int, len(tasks))
	for i := range tasks {
		existingIDs[task.NormalizeID(tasks[i].ID)] = i
	}

	// Validate all IDs exist before any mutation (all-or-nothing).
	for _, id := range ids {
		idx, ok := existingIDs[id]
		if !ok {
			return nil, br, result, fmt.Errorf("task '%s' not found", id)
		}
		br.targetTasks = append(br.targetTasks, RemovedTask{
			ID:    tasks[idx].ID,
			Title: tasks[idx].Title,
		})
	}

	// Build initial remove set from explicit targets.
	targetSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		targetSet[id] = true
	}

	// Expand remove set with all transitive descendants.
	removeSet := collectDescendants(targetSet, tasks)

	// Identify cascaded descendants (in removeSet but not in explicit targets).
	for id := range removeSet {
		if !targetSet[id] {
			if idx, ok := existingIDs[id]; ok {
				br.cascadedTasks = append(br.cascadedTasks, RemovedTask{
					ID:    tasks[idx].ID,
					Title: tasks[idx].Title,
				})
			}
		}
	}

	// Identify surviving tasks with dependency references to removed IDs.
	for i := range tasks {
		normID := task.NormalizeID(tasks[i].ID)
		if removeSet[normID] {
			continue
		}
		for _, dep := range tasks[i].BlockedBy {
			if removeSet[task.NormalizeID(dep)] {
				br.affectedDeps = append(br.affectedDeps, RemovedTask{
					ID:    tasks[i].ID,
					Title: tasks[i].Title,
				})
				break
			}
		}
	}

	if computeOnly {
		return tasks, br, result, nil
	}

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

	return filtered, br, result, nil
}

// RunRemove executes the remove command: parses args, validates all IDs exist (all-or-nothing),
// expands the remove set with cascade descendants, filters targets from the task slice,
// cleans up dependency references on surviving tasks, and outputs the result through the formatter.
// Confirmation prompts are handled by handleRemove before calling this function.
func RunRemove(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	ids, _ := parseRemoveArgs(args)

	if len(ids) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick remove <id> [<id>...]")
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	// Real removal: compute and persist.
	var result RemovalResult
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		var filtered []task.Task
		var execErr error
		filtered, _, result, execErr = executeRemoval(tasks, ids, false)
		return filtered, execErr
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
		fmt.Fprintf(stderr, "Remove task %s %q? [y/N] ", br.targetTasks[0].ID, br.targetTasks[0].Title)
	} else if len(br.targetTasks) == 1 && !hasCascade && hasDeps {
		// Single target with dep impact only.
		fmt.Fprintf(stderr, "Remove task %s %q?\n", br.targetTasks[0].ID, br.targetTasks[0].Title)
		fmt.Fprintf(stderr, "Will update dependencies on:\n")
		for _, d := range br.affectedDeps {
			fmt.Fprintf(stderr, "  %s %q\n", d.ID, d.Title)
		}
		fmt.Fprint(stderr, "Proceed? [y/N] ")
	} else {
		// Multi-target or cascade scenario.
		if len(br.targetTasks) == 1 {
			fmt.Fprintf(stderr, "Remove task %s %q?\n", br.targetTasks[0].ID, br.targetTasks[0].Title)
		} else {
			fmt.Fprintf(stderr, "Remove %d tasks?\n", len(br.targetTasks))
			for _, t := range br.targetTasks {
				fmt.Fprintf(stderr, "  %s %q\n", t.ID, t.Title)
			}
		}
		if hasCascade {
			fmt.Fprintf(stderr, "Will also remove descendants:\n")
			for _, c := range br.cascadedTasks {
				fmt.Fprintf(stderr, "  %s %q\n", c.ID, c.Title)
			}
		}
		if hasDeps {
			fmt.Fprintf(stderr, "Will update dependencies on:\n")
			for _, d := range br.affectedDeps {
				fmt.Fprintf(stderr, "  %s %q\n", d.ID, d.Title)
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
