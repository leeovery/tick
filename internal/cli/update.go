package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// updateFlags holds the parsed flags for the update command.
type updateFlags struct {
	id                  string
	title               string
	titleProvided       bool
	description         string
	descriptionProvided bool
	priority            *int
	parent              string
	parentProvided      bool
	blocks              []string
}

// parseUpdateArgs parses the positional ID and flag arguments for the update command.
func parseUpdateArgs(args []string) (*updateFlags, error) {
	flags := &updateFlags{}

	if len(args) == 0 {
		return nil, fmt.Errorf("Task ID is required. Usage: tick update <id>")
	}

	// First arg is the task ID (unless it's a flag, which means ID is missing)
	if strings.HasPrefix(args[0], "--") {
		return nil, fmt.Errorf("Task ID is required. Usage: tick update <id>")
	}

	flags.id = task.NormalizeID(args[0])

	// Parse flags
	i := 1
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "--title":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--title requires a value")
			}
			flags.title = args[i]
			flags.titleProvided = true
		case "--description":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--description requires a value")
			}
			flags.description = args[i]
			flags.descriptionProvided = true
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
		case "--parent":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--parent requires a value")
			}
			flags.parent = args[i]
			flags.parentProvided = true
			if flags.parent != "" {
				flags.parent = task.NormalizeID(flags.parent)
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
		default:
			return nil, fmt.Errorf("unknown flag %q for update command", arg)
		}
		i++
	}

	return flags, nil
}

// hasAnyFlag returns true if at least one update flag was provided.
func (f *updateFlags) hasAnyFlag() bool {
	return f.titleProvided || f.descriptionProvided || f.priority != nil || f.parentProvided || len(f.blocks) > 0
}

// runUpdate implements the `tick update <id>` command.
func (a *App) runUpdate(args []string) error {
	flags, err := parseUpdateArgs(args)
	if err != nil {
		return err
	}

	if !flags.hasAnyFlag() {
		return fmt.Errorf("No update flags provided. Use at least one of: --title, --description, --priority, --parent, --blocks")
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

	var updatedTask *task.Task

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build lookup map
		existingIDs := make(map[string]bool)
		for _, t := range tasks {
			existingIDs[task.NormalizeID(t.ID)] = true
		}

		// Find the task by normalized ID
		idx := -1
		for i := range tasks {
			if task.NormalizeID(tasks[i].ID) == flags.id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", flags.id)
		}

		// Validate flags before applying any mutations
		if flags.titleProvided {
			if _, err := task.ValidateTitle(flags.title); err != nil {
				return nil, fmt.Errorf("invalid title: %w", err)
			}
		}

		if flags.priority != nil {
			if err := task.ValidatePriority(*flags.priority); err != nil {
				return nil, err
			}
		}

		if flags.parentProvided && flags.parent != "" {
			if err := task.ValidateParent(tasks[idx].ID, flags.parent); err != nil {
				return nil, err
			}
			if !existingIDs[flags.parent] {
				return nil, fmt.Errorf("task %q not found (referenced in --parent)", flags.parent)
			}
		}

		for _, id := range flags.blocks {
			if !existingIDs[id] {
				return nil, fmt.Errorf("task %q not found (referenced in --blocks)", id)
			}
		}

		// Apply mutations
		now := time.Now().UTC().Truncate(time.Second)

		if flags.titleProvided {
			cleanTitle, _ := task.ValidateTitle(flags.title)
			tasks[idx].Title = cleanTitle
		}

		if flags.descriptionProvided {
			tasks[idx].Description = flags.description
		}

		if flags.priority != nil {
			tasks[idx].Priority = *flags.priority
		}

		if flags.parentProvided {
			tasks[idx].Parent = flags.parent
		}

		// Handle --blocks: add this task's ID to target tasks' blocked_by
		if len(flags.blocks) > 0 {
			sourceID := tasks[idx].ID
			for i := range tasks {
				normalizedID := task.NormalizeID(tasks[i].ID)
				for _, blocksID := range flags.blocks {
					if normalizedID == blocksID {
						// Skip if source ID already present in target's blocked_by
						alreadyPresent := false
						for _, existingID := range tasks[i].BlockedBy {
							if task.NormalizeID(existingID) == task.NormalizeID(sourceID) {
								alreadyPresent = true
								break
							}
						}
						if !alreadyPresent {
							tasks[i].BlockedBy = append(tasks[i].BlockedBy, sourceID)
							tasks[i].Updated = now
						}
						break
					}
				}
			}
		}

		tasks[idx].Updated = now

		// Capture the updated task for output
		updatedCopy := tasks[idx]
		updatedTask = &updatedCopy

		return tasks, nil
	})
	if err != nil {
		return unwrapMutationError(err)
	}

	// Output
	if a.config.Quiet {
		fmt.Fprintln(a.stdout, updatedTask.ID)
		return nil
	}

	// Query full show data for formatted output
	data, err := queryShowData(store, updatedTask.ID)
	if err != nil {
		// Fallback: if query fails, just print the ID
		fmt.Fprintln(a.stdout, updatedTask.ID)
		return nil
	}

	return a.formatter.FormatTaskDetail(a.stdout, data)
}
