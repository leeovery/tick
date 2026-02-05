package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// UpdateFlags holds flags specific to the update command.
type UpdateFlags struct {
	Title       *string  // nil means not provided
	Description *string  // nil means not provided, empty string means clear
	Priority    *int     // nil means not provided
	Parent      *string  // nil means not provided, empty string means clear
	Blocks      []string // IDs this task blocks
}

// runUpdate executes the update subcommand.
func (a *App) runUpdate(args []string) int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Parse update-specific flags and get task ID
	flags, taskID, err := parseUpdateArgs(args)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Validate task ID is provided
	if taskID == "" {
		fmt.Fprintf(a.Stderr, "Error: Task ID is required. Usage: tick update <id> [options]\n")
		return 1
	}

	// Normalize task ID to lowercase
	taskID = task.NormalizeID(taskID)

	// Validate at least one flag is provided
	if flags.Title == nil && flags.Description == nil && flags.Priority == nil && flags.Parent == nil && len(flags.Blocks) == 0 {
		fmt.Fprintf(a.Stderr, "Error: At least one flag is required. Available flags:\n")
		fmt.Fprintf(a.Stderr, "  --title \"<text>\"        New title\n")
		fmt.Fprintf(a.Stderr, "  --description \"<text>\"  New description (use \"\" to clear)\n")
		fmt.Fprintf(a.Stderr, "  --priority <0-4>        New priority level\n")
		fmt.Fprintf(a.Stderr, "  --parent <id>           New parent task (use \"\" to clear)\n")
		fmt.Fprintf(a.Stderr, "  --blocks <id,...>       Tasks this task blocks\n")
		return 1
	}

	// Validate title if provided
	if flags.Title != nil {
		trimmed := task.TrimTitle(*flags.Title)
		if trimmed == "" {
			fmt.Fprintf(a.Stderr, "Error: Title cannot be empty\n")
			return 1
		}
		if err := task.ValidateTitle(trimmed); err != nil {
			fmt.Fprintf(a.Stderr, "Error: %s\n", err)
			return 1
		}
		flags.Title = &trimmed
	}

	// Validate priority if provided
	if flags.Priority != nil {
		if err := task.ValidatePriority(*flags.Priority); err != nil {
			fmt.Fprintf(a.Stderr, "Error: %s\n", err)
			return 1
		}
	}

	// Normalize IDs
	if flags.Parent != nil && *flags.Parent != "" {
		normalized := task.NormalizeID(*flags.Parent)
		flags.Parent = &normalized
	}
	flags.Blocks = normalizeIDs(flags.Blocks)

	// Validate self-referencing parent
	if flags.Parent != nil && *flags.Parent != "" && *flags.Parent == taskID {
		fmt.Fprintf(a.Stderr, "Error: task cannot be its own parent\n")
		return 1
	}

	// Open store
	a.WriteVerbose("store open %s", tickDir)
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	var updatedTask task.Task

	// Execute mutation
	a.WriteVerbose("lock acquire exclusive")
	a.WriteVerbose("cache freshness check")
	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID lookup
		idSet := make(map[string]bool)
		for _, t := range tasks {
			idSet[t.ID] = true
		}

		// Find the task to update
		var targetIdx int = -1
		for i := range tasks {
			if tasks[i].ID == taskID {
				targetIdx = i
				break
			}
		}

		if targetIdx == -1 {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		// Validate --parent reference exists (if non-empty)
		if flags.Parent != nil && *flags.Parent != "" && !idSet[*flags.Parent] {
			return nil, fmt.Errorf("task '%s' not found", *flags.Parent)
		}

		// Validate --blocks references exist
		for _, id := range flags.Blocks {
			if !idSet[id] {
				return nil, fmt.Errorf("task '%s' not found", id)
			}
		}

		// Validate --blocks dependencies (cycle detection, child-blocked-by-parent)
		for _, blocksID := range flags.Blocks {
			if err := task.ValidateDependency(tasks, blocksID, taskID); err != nil {
				return nil, err
			}
		}

		// Get current timestamp for updates
		now := time.Now().UTC().Format(time.RFC3339)

		// Apply updates to target task
		if flags.Title != nil {
			tasks[targetIdx].Title = *flags.Title
		}
		if flags.Description != nil {
			tasks[targetIdx].Description = *flags.Description
		}
		if flags.Priority != nil {
			tasks[targetIdx].Priority = *flags.Priority
		}
		if flags.Parent != nil {
			tasks[targetIdx].Parent = *flags.Parent
		}

		// Update --blocks targets: add this task's ID to their blocked_by
		for i := range tasks {
			for _, blocksID := range flags.Blocks {
				if tasks[i].ID == blocksID {
					// Check if already in blocked_by
					found := false
					for _, existing := range tasks[i].BlockedBy {
						if existing == taskID {
							found = true
							break
						}
					}
					if !found {
						tasks[i].BlockedBy = append(tasks[i].BlockedBy, taskID)
						tasks[i].Updated = now
					}
				}
			}
		}

		// Refresh updated timestamp
		tasks[targetIdx].Updated = now

		updatedTask = tasks[targetIdx]
		return tasks, nil
	})
	a.WriteVerbose("atomic write complete")
	a.WriteVerbose("lock release")

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output
	if a.formatConfig.Quiet {
		fmt.Fprintln(a.Stdout, updatedTask.ID)
	} else {
		// Build task detail data for formatter
		data := &TaskDetailData{
			ID:          updatedTask.ID,
			Title:       updatedTask.Title,
			Status:      string(updatedTask.Status),
			Priority:    updatedTask.Priority,
			Description: updatedTask.Description,
			Parent:      updatedTask.Parent,
			Created:     updatedTask.Created,
			Updated:     updatedTask.Updated,
			Closed:      updatedTask.Closed,
			BlockedBy:   make([]RelatedTaskData, 0),
			Children:    make([]RelatedTaskData, 0),
		}
		formatter := a.formatConfig.Formatter()
		fmt.Fprint(a.Stdout, formatter.FormatTaskDetail(data))
	}

	return 0
}

// parseUpdateArgs parses update command arguments and returns flags and task ID.
func parseUpdateArgs(args []string) (UpdateFlags, string, error) {
	var flags UpdateFlags
	var taskID string

	// args[0] is "tick", args[1] is "update", rest are ID, flags, and values
	i := 2
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "--title" && i+1 < len(args):
			title := args[i+1]
			flags.Title = &title
			i += 2

		case arg == "--description" && i+1 < len(args):
			desc := args[i+1]
			flags.Description = &desc
			i += 2

		case arg == "--priority" && i+1 < len(args):
			p, err := strconv.Atoi(args[i+1])
			if err != nil {
				return flags, "", fmt.Errorf("invalid priority: %s", args[i+1])
			}
			flags.Priority = &p
			i += 2

		case arg == "--parent" && i+1 < len(args):
			parent := args[i+1]
			flags.Parent = &parent
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

		case !strings.HasPrefix(arg, "-"):
			// Positional argument - treat as task ID
			if taskID == "" {
				taskID = arg
			}
			i++

		default:
			// Unknown flag or missing value
			i++
		}
	}

	return flags, taskID, nil
}
