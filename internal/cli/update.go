package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// updateOpts holds parsed options for the update command.
// Pointer fields distinguish "not provided" from "provided with empty value".
type updateOpts struct {
	id          string
	title       *string
	description *string
	priority    *int
	parent      *string
	blocks      []string
}

// hasChanges reports whether at least one update flag was provided.
func (o updateOpts) hasChanges() bool {
	return o.title != nil || o.description != nil || o.priority != nil || o.parent != nil || len(o.blocks) > 0
}

// parseUpdateArgs parses the subcommand arguments for `tick update`.
// It extracts the task ID (first positional arg) and command-specific flags.
func parseUpdateArgs(args []string) (updateOpts, error) {
	var opts updateOpts

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--title":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--title requires a value")
			}
			v := args[i]
			opts.title = &v
		case arg == "--description":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--description requires a value")
			}
			v := args[i]
			opts.description = &v
		case arg == "--priority":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--priority requires a value")
			}
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("--priority must be an integer, got %q", args[i])
			}
			opts.priority = &p
		case arg == "--parent":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--parent requires a value")
			}
			v := args[i]
			if v != "" {
				v = task.NormalizeID(strings.TrimSpace(v))
			}
			opts.parent = &v
		case arg == "--blocks":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--blocks requires a value")
			}
			ids := strings.Split(args[i], ",")
			for _, id := range ids {
				normalized := task.NormalizeID(strings.TrimSpace(id))
				if normalized != "" {
					opts.blocks = append(opts.blocks, normalized)
				}
			}
		case strings.HasPrefix(arg, "-"):
			// Unknown flag — skip (global flags already extracted)
		default:
			// Positional argument: task ID (first one wins)
			if opts.id == "" {
				opts.id = task.NormalizeID(arg)
			}
		}
		i++
	}
	return opts, nil
}

// RunUpdate executes the update command: validates inputs, applies changes via the storage engine,
// and outputs the updated task details via the Formatter.
func RunUpdate(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	opts, err := parseUpdateArgs(args)
	if err != nil {
		return err
	}

	if opts.id == "" {
		return fmt.Errorf("task ID is required. Usage: tick update <id> [options]")
	}

	if !opts.hasChanges() {
		return fmt.Errorf("at least one flag is required: --title, --description, --priority, --parent, --blocks")
	}

	// Validate title if provided.
	if opts.title != nil {
		trimmed := task.TrimTitle(*opts.title)
		if err := task.ValidateTitle(trimmed); err != nil {
			return err
		}
	}

	// Validate priority if provided.
	if opts.priority != nil {
		if err := task.ValidatePriority(*opts.priority); err != nil {
			return err
		}
	}

	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir, storeOpts(fc)...)
	if err != nil {
		return err
	}
	defer store.Close()

	var updatedID string

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID set for reference validation.
		idSet := make(map[string]bool, len(tasks))
		for _, t := range tasks {
			idSet[t.ID] = true
		}

		// Validate referenced IDs exist.
		if opts.parent != nil && *opts.parent != "" {
			if *opts.parent == opts.id {
				return nil, fmt.Errorf("task %s cannot be its own parent", opts.id)
			}
			if !idSet[*opts.parent] {
				return nil, fmt.Errorf("task %q not found (referenced in --parent)", *opts.parent)
			}
		}
		for _, blockID := range opts.blocks {
			if !idSet[blockID] {
				return nil, fmt.Errorf("task %q not found (referenced in --blocks)", blockID)
			}
		}

		now := time.Now().UTC().Truncate(time.Second)

		// Find and update the target task.
		found := false
		for i := range tasks {
			if tasks[i].ID != opts.id {
				continue
			}
			found = true

			if opts.title != nil {
				tasks[i].Title = task.TrimTitle(*opts.title)
			}
			if opts.description != nil {
				tasks[i].Description = *opts.description
			}
			if opts.priority != nil {
				tasks[i].Priority = *opts.priority
			}
			if opts.parent != nil {
				tasks[i].Parent = *opts.parent
			}
			tasks[i].Updated = now
			updatedID = tasks[i].ID
			break
		}

		if !found {
			return nil, fmt.Errorf("task '%s' not found", opts.id)
		}

		// For --blocks: add this task's ID to target tasks' blocked_by and refresh updated.
		if len(opts.blocks) > 0 {
			for i := range tasks {
				for _, blockID := range opts.blocks {
					if tasks[i].ID == blockID {
						tasks[i].BlockedBy = append(tasks[i].BlockedBy, opts.id)
						tasks[i].Updated = now
					}
				}
			}
		}

		return tasks, nil
	})
	if err != nil {
		return err
	}

	// Output: quiet mode outputs only the ID.
	if fc.Quiet {
		fmt.Fprintln(stdout, updatedID)
		return nil
	}

	// Full output: query the task with relationships like `tick show`.
	data, err := queryShowData(store, updatedID)
	if err != nil {
		return err
	}

	detail := showDataToTaskDetail(data)
	fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
	return nil
}
