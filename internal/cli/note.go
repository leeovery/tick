package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// handleNote implements the note subcommand, routing to add and remove sub-subcommands.
func (a *App) handleNote(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}

	if len(subArgs) == 0 {
		return fmt.Errorf("sub-command required. Usage: tick note <add|remove> <task_id> <text>")
	}

	subCmd := subArgs[0]
	rest := subArgs[1:]

	switch subCmd {
	case "add":
		return RunNoteAdd(dir, fc, fmtr, rest, a.Stdout)
	case "remove":
		return RunNoteRemove(dir, fc, fmtr, rest, a.Stdout)
	default:
		return fmt.Errorf("unknown note sub-command '%s'. Usage: tick note <add|remove> <task_id> <text>", subCmd)
	}
}

// RunNoteAdd executes the note add command: parses args, validates text,
// resolves partial ID, appends a Note to the task, and outputs the result.
func RunNoteAdd(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	// Parse positional args: first non-flag is ID, remaining non-flags are joined as text.
	var positional []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		positional = append(positional, arg)
	}

	if len(positional) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick note add <task_id> <text>")
	}

	rawID := task.NormalizeID(positional[0])
	textParts := positional[1:]

	text := task.TrimNoteText(strings.Join(textParts, " "))

	if err := task.ValidateNoteText(text); err != nil {
		return err
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	id, err := store.ResolveID(rawID)
	if err != nil {
		return err
	}

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		for i := range tasks {
			if tasks[i].ID == id {
				now := time.Now().UTC().Truncate(time.Second)
				note := task.Note{
					Text:    text,
					Created: now,
				}
				tasks[i].Notes = append(tasks[i].Notes, note)
				tasks[i].Updated = now
				return tasks, nil
			}
		}
		return nil, fmt.Errorf("task '%s' not found", id)
	})
	if err != nil {
		return err
	}

	return outputMutationResult(store, id, fc, fmtr, stdout)
}

// RunNoteRemove executes the note remove command: parses args (task ID and 1-based index),
// validates the index, resolves partial ID, removes the note at the given position,
// and outputs the result.
func RunNoteRemove(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("task ID is required. Usage: tick note remove <task_id> <index>")
	}
	if len(args) < 2 {
		return fmt.Errorf("index is required. Usage: tick note remove <task_id> <index>")
	}

	rawID := task.NormalizeID(args[0])

	index, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid index %q: must be an integer", args[1])
	}
	if index < 1 {
		return fmt.Errorf("index must be >= 1, got %d", index)
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	id, err := store.ResolveID(rawID)
	if err != nil {
		return err
	}

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		for i := range tasks {
			if tasks[i].ID == id {
				if len(tasks[i].Notes) == 0 {
					return nil, fmt.Errorf("task has no notes to remove")
				}
				if index > len(tasks[i].Notes) {
					return nil, fmt.Errorf("index %d out of range: task has %d note(s)", index, len(tasks[i].Notes))
				}
				idx := index - 1
				tasks[i].Notes = append(tasks[i].Notes[:idx], tasks[i].Notes[idx+1:]...)
				tasks[i].Updated = time.Now().UTC().Truncate(time.Second)
				return tasks, nil
			}
		}
		return nil, fmt.Errorf("task '%s' not found", id)
	})
	if err != nil {
		return err
	}

	return outputMutationResult(store, id, fc, fmtr, stdout)
}
