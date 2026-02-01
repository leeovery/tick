package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

func (a *App) cmdUpdate(workDir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick update <id> [flags]")
	}

	taskID := task.NormalizeID(args[0])
	remaining := args[1:]

	// Parse update flags.
	var (
		titleFlag       *string
		descriptionFlag *string
		priorityFlag    *int
		parentFlag      *string
		blocksFlag      []string
	)

	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--title":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--title requires a value")
			}
			i++
			v := remaining[i]
			titleFlag = &v
		case "--description":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--description requires a value")
			}
			i++
			v := remaining[i]
			descriptionFlag = &v
		case "--priority":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--priority requires a value")
			}
			i++
			p, err := strconv.Atoi(remaining[i])
			if err != nil {
				return fmt.Errorf("invalid priority value: %s", remaining[i])
			}
			if err := task.ValidatePriority(p); err != nil {
				return err
			}
			priorityFlag = &p
		case "--parent":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--parent requires a value")
			}
			i++
			v := remaining[i]
			parentFlag = &v
		case "--blocks":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--blocks requires a value")
			}
			i++
			for _, id := range strings.Split(remaining[i], ",") {
				id = strings.TrimSpace(id)
				if id != "" {
					blocksFlag = append(blocksFlag, task.NormalizeID(id))
				}
			}
		}
	}

	// Validate at least one flag provided.
	if titleFlag == nil && descriptionFlag == nil && priorityFlag == nil && parentFlag == nil && len(blocksFlag) == 0 {
		return fmt.Errorf("At least one flag is required. Available: --title, --description, --priority, --parent, --blocks")
	}

	// Validate title if provided.
	if titleFlag != nil {
		trimmed := task.TrimTitle(*titleFlag)
		if err := task.ValidateTitle(trimmed); err != nil {
			return err
		}
		titleFlag = &trimmed
	}

	// Validate parent if non-empty.
	if parentFlag != nil && *parentFlag != "" {
		normalized := task.NormalizeID(*parentFlag)
		parentFlag = &normalized
		if err := task.ValidateParent(taskID, normalized); err != nil {
			return err
		}
	}

	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	var updatedTask task.Task

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		taskMap := make(map[string]*task.Task, len(tasks))
		for i := range tasks {
			taskMap[tasks[i].ID] = &tasks[i]
		}

		t, ok := taskMap[taskID]
		if !ok {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		now := time.Now().UTC().Truncate(time.Second)

		if titleFlag != nil {
			t.Title = *titleFlag
		}
		if descriptionFlag != nil {
			t.Description = *descriptionFlag
		}
		if priorityFlag != nil {
			t.Priority = *priorityFlag
		}
		if parentFlag != nil {
			if *parentFlag == "" {
				t.Parent = ""
			} else {
				if _, exists := taskMap[*parentFlag]; !exists {
					return nil, fmt.Errorf("task '%s' not found", *parentFlag)
				}
				t.Parent = *parentFlag
			}
		}

		// --blocks: add this task to targets' blocked_by
		for _, blockID := range blocksFlag {
			target, exists := taskMap[blockID]
			if !exists {
				return nil, fmt.Errorf("task '%s' not found", blockID)
			}
			target.BlockedBy = append(target.BlockedBy, taskID)
			target.Updated = now
		}

		t.Updated = now
		updatedTask = *t

		return tasks, nil
	})
	if err != nil {
		return err
	}

	if a.opts.Quiet {
		fmt.Fprintln(a.stdout, updatedTask.ID)
		return nil
	}

	return a.queryAndFormatTaskDetail(store, updatedTask.ID)
}
