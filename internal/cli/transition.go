package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/engine"
	"github.com/leeovery/tick/internal/task"
)

// runTransition returns a handler for a transition command (start, done, cancel,
// reopen). Each handler parses the positional task ID, looks up the task via the
// storage engine's Mutate flow, applies the transition, persists, and outputs
// the transition result.
func runTransition(command string) func(*Context) error {
	return func(ctx *Context) error {
		if len(ctx.Args) == 0 {
			return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
		}

		id := task.NormalizeID(ctx.Args[0])

		tickDir, err := DiscoverTickDir(ctx.WorkDir)
		if err != nil {
			return err
		}

		store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
		if err != nil {
			return err
		}
		defer store.Close()

		var result task.TransitionResult

		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			for i := range tasks {
				if tasks[i].ID == id {
					r, err := task.Transition(&tasks[i], command)
					if err != nil {
						return nil, err
					}
					result = r
					return tasks, nil
				}
			}
			return nil, fmt.Errorf("Task '%s' not found", id)
		})
		if err != nil {
			return err
		}

		if !ctx.Quiet {
			return ctx.Fmt.FormatTransition(ctx.Stdout, &TransitionData{
				ID:        id,
				OldStatus: string(result.OldStatus),
				NewStatus: string(result.NewStatus),
			})
		}

		return nil
	}
}
