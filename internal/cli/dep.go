package cli

import (
	"fmt"
	"time"

	"github.com/leeovery/tick/internal/engine"
	"github.com/leeovery/tick/internal/task"
)

// runDep implements the "tick dep" command, which dispatches to sub-subcommands
// "add" and "rm" for managing task dependencies post-creation.
func runDep(ctx *Context) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("dep requires a subcommand: add, rm")
	}

	subcmd := ctx.Args[0]
	args := ctx.Args[1:]

	switch subcmd {
	case "add":
		return runDepAdd(ctx, args)
	case "rm":
		return runDepRm(ctx, args)
	default:
		return fmt.Errorf("unknown dep subcommand '%s'. Available: add, rm", subcmd)
	}
}

// runDepAdd implements "tick dep add <task_id> <blocked_by_id>". It looks up
// both tasks, validates the dependency (self-ref, duplicate, cycle,
// child-blocked-by-parent), adds the dependency, and persists via atomic write.
func runDepAdd(ctx *Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("dep add requires two IDs: tick dep add <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(args[0])
	blockedByID := task.NormalizeID(args[1])

	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID lookup.
		existing := make(map[string]int, len(tasks))
		for i, t := range tasks {
			existing[t.ID] = i
		}

		// Verify task_id exists.
		taskIdx, found := existing[taskID]
		if !found {
			return nil, fmt.Errorf("Task '%s' not found", taskID)
		}

		// Verify blocked_by_id exists.
		if _, found := existing[blockedByID]; !found {
			return nil, fmt.Errorf("Task '%s' not found", blockedByID)
		}

		// Check duplicate.
		for _, dep := range tasks[taskIdx].BlockedBy {
			if task.NormalizeID(dep) == blockedByID {
				return nil, fmt.Errorf("Task '%s' is already blocked by '%s'", taskID, blockedByID)
			}
		}

		// Validate dependency (self-ref, cycle, child-blocked-by-parent).
		if err := task.ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return nil, err
		}

		// Add the dependency.
		tasks[taskIdx].BlockedBy = append(tasks[taskIdx].BlockedBy, blockedByID)
		tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return err
	}

	if !ctx.Quiet {
		return ctx.Fmt.FormatDepChange(ctx.Stdout, &DepChangeData{
			Action:      "added",
			TaskID:      taskID,
			BlockedByID: blockedByID,
		})
	}

	return nil
}

// runDepRm implements "tick dep rm <task_id> <blocked_by_id>". It looks up the
// task, verifies the dependency exists in blocked_by, removes it, and persists
// via atomic write. It does NOT validate that blocked_by_id exists as a task,
// supporting removal of stale references.
func runDepRm(ctx *Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("dep rm requires two IDs: tick dep rm <task_id> <blocked_by_id>")
	}

	taskID := task.NormalizeID(args[0])
	blockedByID := task.NormalizeID(args[1])

	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Find the task.
		taskIdx := -1
		for i, t := range tasks {
			if t.ID == taskID {
				taskIdx = i
				break
			}
		}
		if taskIdx == -1 {
			return nil, fmt.Errorf("Task '%s' not found", taskID)
		}

		// Find and remove the dependency from blocked_by.
		depIdx := -1
		for i, dep := range tasks[taskIdx].BlockedBy {
			if task.NormalizeID(dep) == blockedByID {
				depIdx = i
				break
			}
		}
		if depIdx == -1 {
			return nil, fmt.Errorf("Task '%s' is not blocked by '%s'", taskID, blockedByID)
		}

		// Remove the dependency (order-preserving).
		tasks[taskIdx].BlockedBy = append(
			tasks[taskIdx].BlockedBy[:depIdx],
			tasks[taskIdx].BlockedBy[depIdx+1:]...,
		)
		tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)

		return tasks, nil
	})
	if err != nil {
		return err
	}

	if !ctx.Quiet {
		return ctx.Fmt.FormatDepChange(ctx.Stdout, &DepChangeData{
			Action:      "removed",
			TaskID:      taskID,
			BlockedByID: blockedByID,
		})
	}

	return nil
}
