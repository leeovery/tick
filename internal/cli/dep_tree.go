package cli

import (
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/task"
)

// RunDepTree executes the dep tree command: displays the dependency tree
// for all tasks (full graph mode) or a specific task (focused mode).
func RunDepTree(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	if fc.Quiet {
		return nil
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	tasks, err := store.ReadTasks()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return runFullDepTree(tasks, fmtr, stdout)
	}
	return runFocusedDepTree(store, tasks, args[0], fmtr, stdout)
}

// runFullDepTree builds and outputs the full dependency graph.
func runFullDepTree(tasks []task.Task, fmtr Formatter, stdout io.Writer) error {
	result := BuildFullDepTree(tasks)

	if len(result.Roots) == 0 {
		fmt.Fprintln(stdout, fmtr.FormatMessage(result.Message))
		return nil
	}

	fmt.Fprintln(stdout, fmtr.FormatDepTree(result))
	return nil
}

// runFocusedDepTree builds and outputs the focused dependency view for a single task.
func runFocusedDepTree(store interface{ ResolveID(string) (string, error) }, tasks []task.Task, rawID string, fmtr Formatter, stdout io.Writer) error {
	id := task.NormalizeID(rawID)

	resolvedID, err := store.ResolveID(id)
	if err != nil {
		return err
	}

	result, err := BuildFocusedDepTree(tasks, resolvedID)
	if err != nil {
		return err
	}

	if len(result.BlockedBy) == 0 && len(result.Blocks) == 0 {
		fmt.Fprintf(stdout, "%s %s [%s]\n", result.Target.ID, result.Target.Title, result.Target.Status)
		fmt.Fprintln(stdout, fmtr.FormatMessage(result.Message))
		return nil
	}

	fmt.Fprintln(stdout, fmtr.FormatDepTree(result))
	return nil
}
