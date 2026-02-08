package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runUpdate implements the `tick update <id>` command.
// It updates task fields (title, description, priority, parent, blocks) with at least one flag required.
func (a *App) runUpdate(args []string) error {
	id, flags, err := a.parseUpdateArgs(args)
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

	var updatedTask *task.Task

	err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find the task by ID
		idx := -1
		for i := range tasks {
			if tasks[i].ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", id)
		}

		// Build existence lookup
		existingIDs := make(map[string]bool, len(tasks))
		for _, t := range tasks {
			existingIDs[t.ID] = true
		}

		// Apply --title
		if flags.title != nil {
			validTitle, err := task.ValidateTitle(*flags.title)
			if err != nil {
				return nil, fmt.Errorf("invalid title: %w", err)
			}
			tasks[idx].Title = validTitle
		}

		// Apply --description
		if flags.description != nil {
			tasks[idx].Description = *flags.description
		}

		// Apply --priority
		if flags.priority != nil {
			if err := task.ValidatePriority(*flags.priority); err != nil {
				return nil, err
			}
			tasks[idx].Priority = *flags.priority
		}

		// Apply --parent
		if flags.parent != nil {
			parentVal := *flags.parent
			if parentVal == "" {
				// Clear parent
				tasks[idx].Parent = ""
			} else {
				if parentVal == id {
					return nil, fmt.Errorf("task %s cannot be its own parent", id)
				}
				if !existingIDs[parentVal] {
					return nil, fmt.Errorf("task '%s' not found (referenced in --parent)", parentVal)
				}
				tasks[idx].Parent = parentVal
			}
		}

		// Apply --blocks
		if flags.blocks != nil {
			for _, blockID := range flags.blocks {
				if !existingIDs[blockID] {
					return nil, fmt.Errorf("task '%s' not found (referenced in --blocks)", blockID)
				}
			}
			now := time.Now().UTC().Truncate(time.Second)
			for i := range tasks {
				for _, blockTarget := range flags.blocks {
					if tasks[i].ID == blockTarget {
						// Add this task's ID to target's blocked_by if not already present
						found := false
						for _, dep := range tasks[i].BlockedBy {
							if dep == id {
								found = true
								break
							}
						}
						if !found {
							tasks[i].BlockedBy = append(tasks[i].BlockedBy, id)
							tasks[i].Updated = now
						}
					}
				}
			}
		}

		// Set updated timestamp
		tasks[idx].Updated = time.Now().UTC().Truncate(time.Second)

		updatedTask = &tasks[idx]
		return tasks, nil
	})
	if err != nil {
		return err
	}

	// Output via formatter
	if a.Quiet {
		fmt.Fprintln(a.Stdout, updatedTask.ID)
		return nil
	}
	detail := taskToDetail(updatedTask)
	return a.Formatter.FormatTaskDetail(a.Stdout, detail)
}

// updateFlags holds parsed flags for the update command.
type updateFlags struct {
	title       *string
	description *string
	priority    *int
	parent      *string
	blocks      []string
}

// hasAnyFlag returns true if at least one flag has been set.
func (f *updateFlags) hasAnyFlag() bool {
	return f.title != nil || f.description != nil || f.priority != nil || f.parent != nil || f.blocks != nil
}

// parseUpdateArgs parses the positional ID and command-specific flags from args.
func (a *App) parseUpdateArgs(args []string) (string, *updateFlags, error) {
	flags := &updateFlags{}
	var id string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--title":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--title requires a value")
			}
			i++
			val := args[i]
			flags.title = &val

		case arg == "--description":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--description requires a value")
			}
			i++
			val := args[i]
			flags.description = &val

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

		case arg == "--parent":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--parent requires a value")
			}
			i++
			val := strings.TrimSpace(args[i])
			if val != "" {
				val = task.NormalizeID(val)
			}
			flags.parent = &val

		case arg == "--blocks":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--blocks requires a value")
			}
			i++
			ids := parseCommaSeparatedIDs(args[i])
			flags.blocks = ids

		default:
			// First non-flag argument is the ID
			if id == "" {
				id = task.NormalizeID(strings.TrimSpace(arg))
			} else {
				return "", nil, fmt.Errorf("unexpected argument '%s'", arg)
			}
		}
	}

	if id == "" {
		return "", nil, fmt.Errorf("Task ID is required. Usage: tick update <id> [options]")
	}

	if !flags.hasAnyFlag() {
		return "", nil, fmt.Errorf("No flags provided. At least one flag is required.\n\nAvailable options:\n  --title \"<text>\"           New title\n  --description \"<text>\"     New description (use \"\" to clear)\n  --priority <0-4>           New priority level\n  --parent <id>              New parent task (use \"\" to clear)\n  --blocks <id,...>          Tasks this task blocks")
	}

	return id, flags, nil
}
