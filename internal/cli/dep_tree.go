package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/leeovery/tick/internal/task"
)

// RunDepTree executes the dep tree command: displays the dependency tree
// for all tasks (full graph mode) or a specific task (focused mode). Every
// argument is positional, so the post-marker literals follow the flag half in order.
func RunDepTree(dir string, fc FormatConfig, fmtr Formatter, flagArgs, literals []string, stdout io.Writer) error {
	args := slices.Concat(flagArgs, literals)
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
	fmt.Fprintln(stdout, fmtr.FormatDepTree(BuildFullDepTree(tasks)))
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

	fmt.Fprintln(stdout, fmtr.FormatDepTree(result))
	return nil
}
