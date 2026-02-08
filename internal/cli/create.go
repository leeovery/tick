package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

	s, err := a.openStore(tickDir)
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

	// Output via formatter
	if a.Quiet {
		fmt.Fprintln(a.Stdout, createdTask.ID)
		return nil
	}
	detail := taskToDetail(createdTask)
	return a.Formatter.FormatTaskDetail(a.Stdout, detail)
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

// taskToDetail converts a task.Task to a TaskDetail for formatter output.
// It populates basic fields; blocked_by and children are empty since
// create/update don't have the full DB context for related task details.
func taskToDetail(t *task.Task) TaskDetail {
	detail := TaskDetail{
		ID:          t.ID,
		Title:       t.Title,
		Status:      string(t.Status),
		Priority:    t.Priority,
		Description: t.Description,
		Parent:      t.Parent,
		Created:     t.Created.Format("2006-01-02T15:04:05Z"),
		Updated:     t.Updated.Format("2006-01-02T15:04:05Z"),
	}
	if t.Closed != nil {
		detail.Closed = t.Closed.Format("2006-01-02T15:04:05Z")
	}
	return detail
}
