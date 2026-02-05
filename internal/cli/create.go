package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// CreateFlags holds flags specific to the create command.
type CreateFlags struct {
	Priority    int
	Description string
	BlockedBy   []string
	Blocks      []string
	Parent      string
}

// runCreate executes the create subcommand.
func (a *App) runCreate(args []string) int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Parse create-specific flags and get title
	flags, title, err := parseCreateArgs(args)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Validate title is provided
	if title == "" {
		fmt.Fprintf(a.Stderr, "Error: Title is required. Usage: tick create \"<title>\" [options]\n")
		return 1
	}

	// Trim and validate title
	title = task.TrimTitle(title)
	if title == "" {
		fmt.Fprintf(a.Stderr, "Error: Title cannot be empty\n")
		return 1
	}
	if err := task.ValidateTitle(title); err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Validate priority
	if err := task.ValidatePriority(flags.Priority); err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Normalize IDs to lowercase
	flags.BlockedBy = normalizeIDs(flags.BlockedBy)
	flags.Blocks = normalizeIDs(flags.Blocks)
	flags.Parent = task.NormalizeID(flags.Parent)

	// Open store
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	var createdTask task.Task

	// Execute mutation
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID lookup
		idSet := make(map[string]bool)
		for _, t := range tasks {
			idSet[t.ID] = true
		}

		// Validate --blocked-by references exist
		for _, id := range flags.BlockedBy {
			if !idSet[id] {
				return nil, fmt.Errorf("task '%s' not found", id)
			}
		}

		// Validate --blocks references exist
		for _, id := range flags.Blocks {
			if !idSet[id] {
				return nil, fmt.Errorf("task '%s' not found", id)
			}
		}

		// Validate --parent reference exists
		if flags.Parent != "" && !idSet[flags.Parent] {
			return nil, fmt.Errorf("task '%s' not found", flags.Parent)
		}

		// Generate unique ID
		newID, err := task.GenerateID(func(id string) (bool, error) {
			return idSet[id], nil
		})
		if err != nil {
			return nil, err
		}

		// Create timestamp
		created, updated := task.DefaultTimestamps()

		// Build new task
		newTask := task.Task{
			ID:          newID,
			Title:       title,
			Status:      task.StatusOpen,
			Priority:    flags.Priority,
			Description: flags.Description,
			BlockedBy:   flags.BlockedBy,
			Parent:      flags.Parent,
			Created:     created,
			Updated:     updated,
		}

		// Validate self-reference
		if err := task.ValidateBlockedBy(newTask.ID, newTask.BlockedBy); err != nil {
			return nil, err
		}
		if err := task.ValidateParent(newTask.ID, newTask.Parent); err != nil {
			return nil, err
		}

		// Update --blocks targets: add new task's ID to their blocked_by
		for i := range tasks {
			for _, blocksID := range flags.Blocks {
				if tasks[i].ID == blocksID {
					tasks[i].BlockedBy = append(tasks[i].BlockedBy, newTask.ID)
					tasks[i].Updated = updated
				}
			}
		}

		createdTask = newTask

		// Append new task
		return append(tasks, newTask), nil
	})

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output
	if a.flags.Quiet {
		fmt.Fprintln(a.Stdout, createdTask.ID)
	} else {
		a.printTaskDetails(createdTask)
	}

	return 0
}

// parseCreateArgs parses create command arguments and returns flags and title.
func parseCreateArgs(args []string) (CreateFlags, string, error) {
	flags := CreateFlags{
		Priority: task.DefaultPriority(),
	}
	var title string

	// args[0] is "tick", args[1] is "create", rest are flags and title
	i := 2
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "--priority" && i+1 < len(args):
			p, err := strconv.Atoi(args[i+1])
			if err != nil {
				return flags, "", fmt.Errorf("invalid priority: %s", args[i+1])
			}
			flags.Priority = p
			i += 2

		case arg == "--description" && i+1 < len(args):
			flags.Description = args[i+1]
			i += 2

		case arg == "--blocked-by" && i+1 < len(args):
			ids := strings.Split(args[i+1], ",")
			for _, id := range ids {
				id = strings.TrimSpace(id)
				if id != "" {
					flags.BlockedBy = append(flags.BlockedBy, id)
				}
			}
			i += 2

		case arg == "--blocks" && i+1 < len(args):
			ids := strings.Split(args[i+1], ",")
			for _, id := range ids {
				id = strings.TrimSpace(id)
				if id != "" {
					flags.Blocks = append(flags.Blocks, id)
				}
			}
			i += 2

		case arg == "--parent" && i+1 < len(args):
			flags.Parent = args[i+1]
			i += 2

		case !strings.HasPrefix(arg, "-"):
			// Positional argument - treat as title
			if title == "" {
				title = arg
			}
			i++

		default:
			// Unknown flag or missing value
			i++
		}
	}

	return flags, title, nil
}

// normalizeIDs normalizes a slice of IDs to lowercase.
func normalizeIDs(ids []string) []string {
	if ids == nil {
		return nil
	}
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = task.NormalizeID(id)
	}
	return result
}

// printTaskDetails outputs task details in basic format.
func (a *App) printTaskDetails(t task.Task) {
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
	fmt.Fprintf(a.Stdout, "Created:     %s\n", t.Created)
	fmt.Fprintf(a.Stdout, "Updated:     %s\n", t.Updated)
}
