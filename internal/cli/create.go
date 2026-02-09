package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/engine"
	"github.com/leeovery/tick/internal/task"
)

// runCreate implements the "tick create" command. It creates a new task with the
// given title and optional flags, validates inputs, persists via the storage
// engine's Mutate flow, and outputs the created task details.
func runCreate(ctx *Context) error {
	title, opts, err := parseCreateArgs(ctx.Args)
	if err != nil {
		return err
	}

	// Validate title.
	title, err = task.ValidateTitle(title)
	if err != nil {
		return err
	}

	// Validate priority.
	if err := task.ValidatePriority(opts.priority); err != nil {
		return err
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

	// Track the created task for output.
	var createdTask task.Task

	// Execute mutation.
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID existence lookup.
		existing := make(map[string]int, len(tasks))
		for i, t := range tasks {
			existing[t.ID] = i
		}

		// Generate unique ID.
		id, err := task.GenerateID(func(candidate string) bool {
			_, found := existing[candidate]
			return found
		})
		if err != nil {
			return nil, err
		}

		// Normalize reference IDs.
		blockedBy := normalizeIDs(opts.blockedBy)
		blocks := normalizeIDs(opts.blocks)
		parent := task.NormalizeID(opts.parent)

		// Validate self-references.
		if err := task.ValidateBlockedBy(id, blockedBy); err != nil {
			return nil, err
		}
		if err := task.ValidateParent(id, parent); err != nil {
			return nil, err
		}

		// Validate all referenced IDs exist.
		if err := validateIDsExist(existing, blockedBy, "--blocked-by"); err != nil {
			return nil, err
		}
		if err := validateIDsExist(existing, blocks, "--blocks"); err != nil {
			return nil, err
		}
		if parent != "" {
			if err := validateIDsExist(existing, []string{parent}, "--parent"); err != nil {
				return nil, err
			}
		}

		// Build the new task.
		newTask := task.NewTask(id, title)
		newTask.Priority = opts.priority
		newTask.Description = opts.description
		if len(blockedBy) > 0 {
			newTask.BlockedBy = blockedBy
		}
		newTask.Parent = parent

		// For --blocks: add new task's ID to target tasks' blocked_by.
		now := time.Now().UTC().Truncate(time.Second)
		for _, blockID := range blocks {
			idx := existing[blockID]
			tasks[idx].BlockedBy = append(tasks[idx].BlockedBy, id)
			tasks[idx].Updated = now
		}

		createdTask = newTask
		return append(tasks, newTask), nil
	})
	if err != nil {
		return err
	}

	// Output.
	if ctx.Quiet {
		fmt.Fprintln(ctx.Stdout, createdTask.ID)
		return nil
	}

	return ctx.Fmt.FormatTaskDetail(ctx.Stdout, taskToShowData(createdTask))
}

// createOpts holds parsed optional flags for the create command.
type createOpts struct {
	priority    int
	description string
	blockedBy   []string
	blocks      []string
	parent      string
}

// parseCreateArgs parses the positional title and flags from create command args.
// Returns the title, parsed options, and any error.
func parseCreateArgs(args []string) (string, createOpts, error) {
	opts := createOpts{
		priority: task.DefaultPriority,
	}

	var title string
	titleFound := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--priority":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("--priority requires a value")
			}
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return "", opts, fmt.Errorf("--priority must be an integer, got %q", args[i])
			}
			opts.priority = p

		case arg == "--description":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("--description requires a value")
			}
			opts.description = args[i]

		case arg == "--blocked-by":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("--blocked-by requires a value")
			}
			opts.blockedBy = splitCSV(args[i])

		case arg == "--blocks":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("--blocks requires a value")
			}
			opts.blocks = splitCSV(args[i])

		case arg == "--parent":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("--parent requires a value")
			}
			opts.parent = args[i]

		case strings.HasPrefix(arg, "-"):
			return "", opts, fmt.Errorf("unknown flag '%s'", arg)

		default:
			if !titleFound {
				title = arg
				titleFound = true
			} else {
				return "", opts, fmt.Errorf("unexpected argument '%s'", arg)
			}
		}
	}

	if !titleFound {
		return "", opts, fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
	}

	return title, opts, nil
}

// validateIDsExist checks that all IDs in the slice exist in the existing map.
// Returns an error referencing the flag name if any ID is not found.
func validateIDsExist(existing map[string]int, ids []string, flagName string) error {
	for _, id := range ids {
		if _, found := existing[id]; !found {
			return fmt.Errorf("task '%s' not found (referenced in %s)", id, flagName)
		}
	}
	return nil
}

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// normalizeIDs normalizes a slice of IDs to lowercase.
func normalizeIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	normalized := make([]string, len(ids))
	for i, id := range ids {
		normalized[i] = task.NormalizeID(id)
	}
	return normalized
}
