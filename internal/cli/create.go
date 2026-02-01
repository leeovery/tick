package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

func (a *App) cmdCreate(workDir string, args []string) error {
	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	// Parse create-specific flags and title
	var title, description, parent string
	var blockedBy, blocks []string
	priority := -1 // sentinel: use default

	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--priority":
			if i+1 >= len(args) {
				return fmt.Errorf("--priority requires a value")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("invalid priority value: %s", args[i])
			}
			if err := task.ValidatePriority(p); err != nil {
				return err
			}
			priority = p
		case "--description":
			if i+1 >= len(args) {
				return fmt.Errorf("--description requires a value")
			}
			i++
			description = args[i]
		case "--blocked-by":
			if i+1 >= len(args) {
				return fmt.Errorf("--blocked-by requires a value")
			}
			i++
			for _, id := range strings.Split(args[i], ",") {
				id = strings.TrimSpace(id)
				if id != "" {
					blockedBy = append(blockedBy, task.NormalizeID(id))
				}
			}
		case "--blocks":
			if i+1 >= len(args) {
				return fmt.Errorf("--blocks requires a value")
			}
			i++
			for _, id := range strings.Split(args[i], ",") {
				id = strings.TrimSpace(id)
				if id != "" {
					blocks = append(blocks, task.NormalizeID(id))
				}
			}
		case "--parent":
			if i+1 >= len(args) {
				return fmt.Errorf("--parent requires a value")
			}
			i++
			parent = task.NormalizeID(args[i])
		default:
			positional = append(positional, args[i])
		}
	}

	// Title is first positional arg
	if len(positional) == 0 {
		return fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
	}
	title = task.TrimTitle(positional[0])
	if err := task.ValidateTitle(title); err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	var createdTask task.Task

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build existence lookup
		taskMap := make(map[string]*task.Task, len(tasks))
		for i := range tasks {
			taskMap[tasks[i].ID] = &tasks[i]
		}

		existsFn := func(id string) bool {
			_, ok := taskMap[id]
			return ok
		}

		// Generate ID
		id, err := task.GenerateID(existsFn)
		if err != nil {
			return nil, err
		}

		// Validate blocked_by references exist
		for _, depID := range blockedBy {
			if _, ok := taskMap[depID]; !ok {
				return nil, fmt.Errorf("task '%s' not found", depID)
			}
		}

		// Validate blocks references exist
		for _, blockID := range blocks {
			if _, ok := taskMap[blockID]; !ok {
				return nil, fmt.Errorf("task '%s' not found", blockID)
			}
		}

		// Validate parent exists
		if parent != "" {
			if _, ok := taskMap[parent]; !ok {
				return nil, fmt.Errorf("task '%s' not found", parent)
			}
		}

		// Validate no self-references
		if err := task.ValidateBlockedBy(id, blockedBy); err != nil {
			return nil, err
		}
		if err := task.ValidateParent(id, parent); err != nil {
			return nil, err
		}

		// Build the task
		newTask := task.NewTask(id, title, priority)
		newTask.Description = description
		newTask.Parent = parent
		if len(blockedBy) > 0 {
			newTask.BlockedBy = blockedBy
		}

		// Apply --blocks: add this task's ID to target tasks' blocked_by
		now := time.Now().UTC().Truncate(time.Second)
		for _, blockID := range blocks {
			t := taskMap[blockID]
			t.BlockedBy = append(t.BlockedBy, id)
			t.Updated = now
		}

		createdTask = newTask
		return append(tasks, newTask), nil
	})

	if err != nil {
		return err
	}

	// Output
	if a.opts.Quiet {
		fmt.Fprintln(a.stdout, createdTask.ID)
	} else {
		a.printTaskBasic(createdTask)
	}

	return nil
}

// printTaskBasic outputs basic task details (Phase 1 format — simple key-value).
func (a *App) printTaskBasic(t task.Task) {
	fmt.Fprintf(a.stdout, "ID:       %s\n", t.ID)
	fmt.Fprintf(a.stdout, "Title:    %s\n", t.Title)
	fmt.Fprintf(a.stdout, "Status:   %s\n", t.Status)
	fmt.Fprintf(a.stdout, "Priority: %d\n", t.Priority)
	if t.Description != "" {
		fmt.Fprintf(a.stdout, "Description: %s\n", t.Description)
	}
	if t.Parent != "" {
		fmt.Fprintf(a.stdout, "Parent:   %s\n", t.Parent)
	}
	if len(t.BlockedBy) > 0 {
		fmt.Fprintf(a.stdout, "Blocked by: %s\n", strings.Join(t.BlockedBy, ", "))
	}
	fmt.Fprintf(a.stdout, "Created:  %s\n", task.FormatTimestamp(t.Created))
}
