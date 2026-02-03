package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// createFlags holds the parsed flags for the create command.
type createFlags struct {
	title         string
	titleProvided bool
	priority      *int
	description   string
	blockedBy     []string
	blocks        []string
	parent        string
}

// parseCreateArgs parses the positional and flag arguments for the create command.
func parseCreateArgs(args []string) (*createFlags, error) {
	flags := &createFlags{}

	i := 0
	// First non-flag arg is the title
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			break
		}
		if !flags.titleProvided {
			flags.title = arg
			flags.titleProvided = true
		}
		i++
	}

	// Parse command-specific flags
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "--priority":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--priority requires a value")
			}
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("--priority must be an integer, got %q", args[i])
			}
			flags.priority = &p
		case "--description":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--description requires a value")
			}
			flags.description = args[i]
		case "--blocked-by":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--blocked-by requires a value")
			}
			ids := strings.Split(args[i], ",")
			for _, id := range ids {
				trimmed := strings.TrimSpace(id)
				if trimmed != "" {
					flags.blockedBy = append(flags.blockedBy, task.NormalizeID(trimmed))
				}
			}
		case "--blocks":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--blocks requires a value")
			}
			ids := strings.Split(args[i], ",")
			for _, id := range ids {
				trimmed := strings.TrimSpace(id)
				if trimmed != "" {
					flags.blocks = append(flags.blocks, task.NormalizeID(trimmed))
				}
			}
		case "--parent":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--parent requires a value")
			}
			flags.parent = task.NormalizeID(args[i])
		default:
			return nil, fmt.Errorf("unknown flag %q for create command", arg)
		}
		i++
	}

	return flags, nil
}

// runCreate implements the `tick create` command.
func (a *App) runCreate(args []string) error {
	flags, err := parseCreateArgs(args)
	if err != nil {
		return err
	}

	// Validate title presence
	if !flags.titleProvided {
		return fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
	}

	// Validate title content (trim and check empty/whitespace)
	trimmedTitle := strings.TrimSpace(flags.title)
	if trimmedTitle == "" {
		return fmt.Errorf("Title cannot be empty")
	}

	// Discover .tick/ directory
	tickDir, err := DiscoverTickDir(a.workDir)
	if err != nil {
		return err
	}

	// Open storage
	store, err := storage.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var createdTask *task.Task

	// Execute mutation
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build exists lookup
		existingIDs := make(map[string]bool)
		for _, t := range tasks {
			existingIDs[task.NormalizeID(t.ID)] = true
		}

		// Validate --blocked-by IDs exist
		for _, id := range flags.blockedBy {
			if !existingIDs[id] {
				return nil, fmt.Errorf("task %q not found (referenced in --blocked-by)", id)
			}
		}

		// Validate --blocks IDs exist
		for _, id := range flags.blocks {
			if !existingIDs[id] {
				return nil, fmt.Errorf("task %q not found (referenced in --blocks)", id)
			}
		}

		// Validate --parent ID exists
		if flags.parent != "" {
			if !existingIDs[flags.parent] {
				return nil, fmt.Errorf("task %q not found (referenced in --parent)", flags.parent)
			}
		}

		// Validate priority if provided
		if flags.priority != nil {
			if err := task.ValidatePriority(*flags.priority); err != nil {
				return nil, err
			}
		}

		// Build TaskOptions
		opts := &task.TaskOptions{
			Priority:    flags.priority,
			Description: flags.description,
			BlockedBy:   flags.blockedBy,
			Parent:      flags.parent,
		}

		// Create the task with collision check against existing IDs
		existsFn := func(id string) bool {
			return existingIDs[task.NormalizeID(id)]
		}

		newTask, err := task.NewTask(trimmedTitle, opts, existsFn)
		if err != nil {
			return nil, err
		}

		createdTask = newTask

		// Append the new task
		modified := append(tasks, *newTask)

		// Handle --blocks: add new task's ID to target tasks' blocked_by
		if len(flags.blocks) > 0 {
			now := time.Now().UTC().Truncate(time.Second)
			for i := range modified {
				normalizedID := task.NormalizeID(modified[i].ID)
				for _, blocksID := range flags.blocks {
					if normalizedID == blocksID {
						modified[i].BlockedBy = append(modified[i].BlockedBy, newTask.ID)
						modified[i].Updated = now
						break
					}
				}
			}
		}

		return modified, nil
	})
	if err != nil {
		return err
	}

	// Output
	if a.config.Quiet {
		fmt.Fprintln(a.stdout, createdTask.ID)
	} else {
		a.printTaskDetails(createdTask)
	}

	return nil
}

// printTaskDetails outputs task details in a basic format (Phase 1).
func (a *App) printTaskDetails(t *task.Task) {
	fmt.Fprintf(a.stdout, "ID:       %s\n", t.ID)
	fmt.Fprintf(a.stdout, "Title:    %s\n", t.Title)
	fmt.Fprintf(a.stdout, "Status:   %s\n", string(t.Status))
	fmt.Fprintf(a.stdout, "Priority: %d\n", t.Priority)

	if t.Description != "" {
		fmt.Fprintf(a.stdout, "Description: %s\n", t.Description)
	}

	if len(t.BlockedBy) > 0 {
		fmt.Fprintf(a.stdout, "Blocked by: %s\n", strings.Join(t.BlockedBy, ", "))
	}

	if t.Parent != "" {
		fmt.Fprintf(a.stdout, "Parent:   %s\n", t.Parent)
	}

	fmt.Fprintf(a.stdout, "Created:  %s\n", formatTime(t.Created))
	fmt.Fprintf(a.stdout, "Updated:  %s\n", formatTime(t.Updated))
}

// formatTime formats a time.Time as ISO 8601 UTC.
func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
