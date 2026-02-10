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

// createOpts holds parsed options for the create command.
type createOpts struct {
	title       string
	priority    int
	description string
	blockedBy   []string
	blocks      []string
	parent      string
}

// parseCreateArgs parses the subcommand arguments for `tick create`.
// It extracts the title (first positional arg) and command-specific flags.
func parseCreateArgs(args []string) (createOpts, error) {
	opts := createOpts{priority: 2}

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--priority":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--priority requires a value")
			}
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("--priority must be an integer, got %q", args[i])
			}
			opts.priority = p
		case arg == "--description":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--description requires a value")
			}
			opts.description = args[i]
		case arg == "--blocked-by":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--blocked-by requires a value")
			}
			opts.blockedBy = parseCommaSeparatedIDs(args[i])
		case arg == "--blocks":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--blocks requires a value")
			}
			opts.blocks = parseCommaSeparatedIDs(args[i])
		case arg == "--parent":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--parent requires a value")
			}
			opts.parent = task.NormalizeID(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "-"):
			// Unknown flag — skip (global flags already extracted)
		default:
			// Positional argument: title (first one wins)
			if opts.title == "" {
				opts.title = arg
			}
		}
		i++
	}
	return opts, nil
}

// RunCreate executes the create command: validates inputs, generates an ID,
// persists via the storage engine, and outputs the created task via the Formatter.
func RunCreate(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	opts, err := parseCreateArgs(args)
	if err != nil {
		return err
	}

	// Validate title presence.
	if opts.title == "" {
		return fmt.Errorf("title is required. Usage: tick create \"<title>\" [options]")
	}

	trimmedTitle := task.TrimTitle(opts.title)
	if err := task.ValidateTitle(trimmedTitle); err != nil {
		return err
	}

	// Validate priority.
	if err := task.ValidatePriority(opts.priority); err != nil {
		return err
	}

	// Discover .tick/ directory and open store.
	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir, storeOpts(fc)...)
	if err != nil {
		return err
	}
	defer store.Close()

	var createdTask task.Task

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build an ID existence checker.
		idSet := make(map[string]bool, len(tasks))
		for _, t := range tasks {
			idSet[t.ID] = true
		}
		exists := func(id string) bool {
			return idSet[id]
		}

		// Generate unique ID.
		id, err := task.GenerateID(exists)
		if err != nil {
			return nil, err
		}

		// Validate all referenced IDs exist and no self-references.
		if err := validateRefs(id, opts, idSet); err != nil {
			return nil, err
		}

		now := time.Now().UTC().Truncate(time.Second)

		newTask := task.Task{
			ID:          id,
			Title:       trimmedTitle,
			Status:      task.StatusOpen,
			Priority:    opts.priority,
			Description: opts.description,
			BlockedBy:   opts.blockedBy,
			Parent:      opts.parent,
			Created:     now,
			Updated:     now,
		}

		// For --blocks: add new task's ID to target tasks' blocked_by and refresh updated.
		if len(opts.blocks) > 0 {
			applyBlocks(tasks, id, opts.blocks, now)
		}

		tasks = append(tasks, newTask)

		// Validate dependencies (cycle detection + child-blocked-by-parent) against full task list.
		if len(opts.blockedBy) > 0 {
			if err := task.ValidateDependencies(tasks, id, opts.blockedBy); err != nil {
				return nil, err
			}
		}
		for _, blockID := range opts.blocks {
			if err := task.ValidateDependency(tasks, blockID, id); err != nil {
				return nil, err
			}
		}

		createdTask = newTask
		return tasks, nil
	})

	if err != nil {
		return err
	}

	// Output.
	if fc.Quiet {
		fmt.Fprintln(stdout, createdTask.ID)
	} else {
		detail := TaskDetail{Task: createdTask}
		fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
	}

	return nil
}

// validateRefs checks that all referenced IDs (blocked-by, blocks, parent) exist
// in the task set and that none reference the new task itself.
func validateRefs(newID string, opts createOpts, idSet map[string]bool) error {
	for _, depID := range opts.blockedBy {
		if depID == newID {
			return fmt.Errorf("task %s cannot be blocked by itself", newID)
		}
		if !idSet[depID] {
			return fmt.Errorf("task %q not found (referenced in --blocked-by)", depID)
		}
	}
	for _, blockID := range opts.blocks {
		if blockID == newID {
			return fmt.Errorf("task %s cannot block itself", newID)
		}
		if !idSet[blockID] {
			return fmt.Errorf("task %q not found (referenced in --blocks)", blockID)
		}
	}
	if opts.parent != "" {
		if opts.parent == newID {
			return fmt.Errorf("task %s cannot be its own parent", newID)
		}
		if !idSet[opts.parent] {
			return fmt.Errorf("task %q not found (referenced in --parent)", opts.parent)
		}
	}
	return nil
}
