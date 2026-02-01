package cli

import (
	"fmt"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

func (a *App) cmdDep(workDir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Usage: tick dep <add|rm> <task_id> <blocked_by_id>")
	}

	subcmd := args[0]
	switch subcmd {
	case "add":
		return a.cmdDepAdd(workDir, args[1:])
	case "rm":
		return a.cmdDepRm(workDir, args[1:])
	default:
		return fmt.Errorf("Unknown dep subcommand '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>", subcmd)
	}
}

func (a *App) cmdDepAdd(workDir string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("dep add requires two IDs. Usage: tick dep add <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(args[0])
	blockedByID := task.NormalizeID(args[1])

	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		taskMap := make(map[string]*task.Task, len(tasks))
		for i := range tasks {
			taskMap[tasks[i].ID] = &tasks[i]
		}

		t, ok := taskMap[taskID]
		if !ok {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		if _, ok := taskMap[blockedByID]; !ok {
			return nil, fmt.Errorf("task '%s' not found", blockedByID)
		}

		// Check duplicate.
		for _, existing := range t.BlockedBy {
			if existing == blockedByID {
				return nil, fmt.Errorf("dependency already exists: %s is already blocked by %s", taskID, blockedByID)
			}
		}

		// Validate (cycle, child-blocked-by-parent).
		if err := task.ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return nil, err
		}

		t.BlockedBy = append(t.BlockedBy, blockedByID)
		t.Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return err
	}

	if !a.opts.Quiet {
		return a.fmtr.FormatDepChange(a.stdout, DepChangeData{
			Action: "added", TaskID: taskID, BlockedBy: blockedByID,
		})
	}

	return nil
}

func (a *App) cmdDepRm(workDir string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("dep rm requires two IDs. Usage: tick dep rm <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(args[0])
	blockedByID := task.NormalizeID(args[1])

	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		taskMap := make(map[string]*task.Task, len(tasks))
		for i := range tasks {
			taskMap[tasks[i].ID] = &tasks[i]
		}

		t, ok := taskMap[taskID]
		if !ok {
			return nil, fmt.Errorf("task '%s' not found", taskID)
		}

		// Find and remove.
		found := false
		newBlockedBy := make([]string, 0, len(t.BlockedBy))
		for _, dep := range t.BlockedBy {
			if dep == blockedByID {
				found = true
				continue
			}
			newBlockedBy = append(newBlockedBy, dep)
		}

		if !found {
			return nil, fmt.Errorf("%s is not blocked by %s", taskID, blockedByID)
		}

		if len(newBlockedBy) == 0 {
			t.BlockedBy = nil
		} else {
			t.BlockedBy = newBlockedBy
		}
		t.Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return err
	}

	if !a.opts.Quiet {
		return a.fmtr.FormatDepChange(a.stdout, DepChangeData{
			Action: "removed", TaskID: taskID, BlockedBy: blockedByID,
		})
	}

	return nil
}
