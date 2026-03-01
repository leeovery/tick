package cli

import (
	"fmt"
	"io"
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
