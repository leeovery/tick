package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/store"
	"github.com/leeovery/tick/internal/task"
)

// runCreate implements the `tick create` command.
// It creates a new task with the given title and optional flags.
func (a *App) runCreate(args []string) error {
	title, flags, err := a.parseCreateArgs(args)
	if err != nil {
		return err
	}

	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := store.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	var createdTask *task.Task

	err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build existence lookup
		existingIDs := make(map[string]bool, len(tasks))
		for _, t := range tasks {
			existingIDs[t.ID] = true
		}

		// Validate blocked-by IDs exist
		for _, id := range flags.blockedBy {
			if !existingIDs[id] {
				return nil, fmt.Errorf("task '%s' not found (referenced in --blocked-by)", id)
			}
		}

		// Validate blocks IDs exist
		for _, id := range flags.blocks {
			if !existingIDs[id] {
				return nil, fmt.Errorf("task '%s' not found (referenced in --blocks)", id)
			}
		}

		// Validate parent ID exists
		if flags.parent != "" {
			if !existingIDs[flags.parent] {
				return nil, fmt.Errorf("task '%s' not found (referenced in --parent)", flags.parent)
			}
		}

		// Build task options
		opts := &task.TaskOptions{
			Description: flags.description,
			BlockedBy:   flags.blockedBy,
			Parent:      flags.parent,
		}
		if flags.priority != nil {
			opts.Priority = flags.priority
		}

		// Create the new task
		exists := func(id string) bool { return existingIDs[id] }
		newTask, err := task.NewTask(title, opts, exists)
		if err != nil {
			return nil, err
		}
		createdTask = newTask

		// Append the new task
		tasks = append(tasks, *newTask)

		// Handle --blocks: add new task's ID to target tasks' blocked_by
		if len(flags.blocks) > 0 {
			now := time.Now().UTC().Truncate(time.Second)
			for i := range tasks {
				for _, blockTarget := range flags.blocks {
					if tasks[i].ID == blockTarget {
						tasks[i].BlockedBy = append(tasks[i].BlockedBy, newTask.ID)
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

	// Output
	if a.Quiet {
		fmt.Fprintln(a.Stdout, createdTask.ID)
	} else {
		a.printTaskDetails(createdTask)
	}

	return nil
}

// createFlags holds parsed flags for the create command.
type createFlags struct {
	priority    *int
	description string
	blockedBy   []string
	blocks      []string
	parent      string
}

// parseCreateArgs parses the positional title and command-specific flags from args.
func (a *App) parseCreateArgs(args []string) (string, *createFlags, error) {
	flags := &createFlags{}
	var title string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--priority":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--priority requires a value")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return "", nil, fmt.Errorf("invalid priority value '%s': must be an integer 0-4", args[i])
			}
			flags.priority = &p

		case arg == "--description":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--description requires a value")
			}
			i++
			flags.description = args[i]

		case arg == "--blocked-by":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--blocked-by requires a value")
			}
			i++
			ids := parseCommaSeparatedIDs(args[i])
			flags.blockedBy = ids

		case arg == "--blocks":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--blocks requires a value")
			}
			i++
			ids := parseCommaSeparatedIDs(args[i])
			flags.blocks = ids

		case arg == "--parent":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--parent requires a value")
			}
			i++
			flags.parent = task.NormalizeID(strings.TrimSpace(args[i]))

		default:
			// First non-flag argument is the title
			if title == "" {
				title = arg
			} else {
				return "", nil, fmt.Errorf("unexpected argument '%s'", arg)
			}
		}
	}

	if title == "" {
		return "", nil, fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
	}

	return title, flags, nil
}

// parseCommaSeparatedIDs splits a comma-separated string of IDs and normalizes each to lowercase.
func parseCommaSeparatedIDs(s string) []string {
	parts := strings.Split(s, ",")
	var ids []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			ids = append(ids, task.NormalizeID(trimmed))
		}
	}
	return ids
}

// printTaskDetails outputs task details to stdout.
// This is a basic Phase 1 format; full formatting comes in Phase 4.
func (a *App) printTaskDetails(t *task.Task) {
	fmt.Fprintf(a.Stdout, "ID:          %s\n", t.ID)
	fmt.Fprintf(a.Stdout, "Title:       %s\n", t.Title)
	fmt.Fprintf(a.Stdout, "Status:      %s\n", t.Status)
	fmt.Fprintf(a.Stdout, "Priority:    %d\n", t.Priority)
	if t.Description != "" {
		fmt.Fprintf(a.Stdout, "Description: %s\n", t.Description)
	}
	if len(t.BlockedBy) > 0 {
		fmt.Fprintf(a.Stdout, "Blocked by:  %s\n", strings.Join(t.BlockedBy, ", "))
	}
	if t.Parent != "" {
		fmt.Fprintf(a.Stdout, "Parent:      %s\n", t.Parent)
	}
	fmt.Fprintf(a.Stdout, "Created:     %s\n", t.Created.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(a.Stdout, "Updated:     %s\n", t.Updated.Format("2006-01-02T15:04:05Z"))
}
