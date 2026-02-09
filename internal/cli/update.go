package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/engine"
	"github.com/leeovery/tick/internal/task"
)

// updateOpts holds parsed optional flags for the update command. Pointer fields
// distinguish "flag provided" from "flag not provided" (nil = not provided).
type updateOpts struct {
	title       *string
	description *string
	priority    *int
	parent      *string
	blocks      []string
}

// hasAnyFlag returns true if at least one flag was provided.
func (o updateOpts) hasAnyFlag() bool {
	return o.title != nil || o.description != nil || o.priority != nil || o.parent != nil || o.blocks != nil
}

// runUpdate implements the "tick update" command. It modifies task fields after
// creation. At least one flag is required. On success it outputs the full task
// details (or just the ID with --quiet).
func runUpdate(ctx *Context) error {
	id, opts, err := parseUpdateArgs(ctx.Args)
	if err != nil {
		return err
	}

	// Validate title if provided.
	if opts.title != nil {
		validated, err := task.ValidateTitle(*opts.title)
		if err != nil {
			return err
		}
		opts.title = &validated
	}

	// Validate priority if provided.
	if opts.priority != nil {
		if err := task.ValidatePriority(*opts.priority); err != nil {
			return err
		}
	}

	// Discover .tick/ directory.
	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	// Open storage engine.
	store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
	if err != nil {
		return err
	}
	defer store.Close()

	var updatedTask task.Task

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID existence lookup.
		existing := make(map[string]int, len(tasks))
		for i, t := range tasks {
			existing[t.ID] = i
		}

		// Find the target task.
		idx, found := existing[id]
		if !found {
			return nil, fmt.Errorf("Task '%s' not found", id)
		}

		// Validate parent if provided.
		if opts.parent != nil {
			parent := *opts.parent
			if parent != "" {
				if err := task.ValidateParent(id, parent); err != nil {
					return nil, err
				}
				if err := validateIDsExist(existing, []string{parent}, "--parent"); err != nil {
					return nil, err
				}
			}
		}

		// Validate blocks if provided.
		if opts.blocks != nil {
			blocks := normalizeIDs(opts.blocks)
			opts.blocks = blocks
			if err := validateIDsExist(existing, blocks, "--blocks"); err != nil {
				return nil, err
			}
		}

		// Apply changes.
		if opts.title != nil {
			tasks[idx].Title = *opts.title
		}
		if opts.description != nil {
			tasks[idx].Description = *opts.description
		}
		if opts.priority != nil {
			tasks[idx].Priority = *opts.priority
		}
		if opts.parent != nil {
			tasks[idx].Parent = *opts.parent
		}

		now := time.Now().UTC().Truncate(time.Second)

		// Handle --blocks: add this task's ID to target tasks' blocked_by.
		if opts.blocks != nil {
			for _, blockID := range opts.blocks {
				bIdx := existing[blockID]
				tasks[bIdx].BlockedBy = append(tasks[bIdx].BlockedBy, id)
				tasks[bIdx].Updated = now
			}
		}

		tasks[idx].Updated = now
		updatedTask = tasks[idx]
		return tasks, nil
	})
	if err != nil {
		return err
	}

	// Output.
	if ctx.Quiet {
		fmt.Fprintln(ctx.Stdout, updatedTask.ID)
		return nil
	}

	return ctx.Fmt.FormatTaskDetail(ctx.Stdout, taskToShowData(updatedTask))
}

// parseUpdateArgs parses the positional ID and flags from update command args.
// Returns the normalized ID, parsed options, and any error.
func parseUpdateArgs(args []string) (string, updateOpts, error) {
	var opts updateOpts

	if len(args) == 0 {
		return "", opts, fmt.Errorf("Task ID is required. Usage: tick update <id> [options]")
	}

	id := task.NormalizeID(args[0])
	remaining := args[1:]

	for i := 0; i < len(remaining); i++ {
		arg := remaining[i]
		switch {
		case arg == "--title":
			i++
			if i >= len(remaining) {
				return "", opts, fmt.Errorf("--title requires a value")
			}
			v := remaining[i]
			opts.title = &v

		case arg == "--description":
			i++
			if i >= len(remaining) {
				return "", opts, fmt.Errorf("--description requires a value")
			}
			v := remaining[i]
			opts.description = &v

		case arg == "--priority":
			i++
			if i >= len(remaining) {
				return "", opts, fmt.Errorf("--priority requires a value")
			}
			p, err := strconv.Atoi(remaining[i])
			if err != nil {
				return "", opts, fmt.Errorf("--priority must be an integer, got %q", remaining[i])
			}
			opts.priority = &p

		case arg == "--parent":
			i++
			if i >= len(remaining) {
				return "", opts, fmt.Errorf("--parent requires a value")
			}
			v := task.NormalizeID(remaining[i])
			opts.parent = &v

		case arg == "--blocks":
			i++
			if i >= len(remaining) {
				return "", opts, fmt.Errorf("--blocks requires a value")
			}
			opts.blocks = splitCSV(remaining[i])

		case strings.HasPrefix(arg, "-"):
			return "", opts, fmt.Errorf("unknown flag '%s'", arg)

		default:
			return "", opts, fmt.Errorf("unexpected argument '%s'", arg)
		}
	}

	if !opts.hasAnyFlag() {
		return "", opts, fmt.Errorf("at least one flag is required. Available: --title, --description, --priority, --parent, --blocks")
	}

	return id, opts, nil
}
