package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// updateOpts holds parsed options for the update command.
// Pointer fields distinguish "not provided" from "provided with empty value".
type updateOpts struct {
	id               string
	title            *string
	description      *string
	priority         *int
	parent           *string
	blocks           []string
	clearDescription bool
	taskType         *string
	clearType        bool
	tags             *[]string
	clearTags        bool
	refs             *[]string
	clearRefs        bool
}

// hasChanges reports whether at least one update flag was provided.
func (o updateOpts) hasChanges() bool {
	return o.title != nil || o.description != nil || o.priority != nil || o.parent != nil || len(o.blocks) > 0 || o.clearDescription || o.taskType != nil || o.clearType || o.tags != nil || o.clearTags || o.refs != nil || o.clearRefs
}

// parseUpdateArgs parses the subcommand arguments for `tick update`.
// It extracts the task ID (first positional arg) and command-specific flags.
func parseUpdateArgs(args []string) (updateOpts, error) {
	var opts updateOpts

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--title":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--title requires a value")
			}
			v := args[i]
			opts.title = &v
		case arg == "--description":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--description requires a value")
			}
			v := args[i]
			opts.description = &v
		case arg == "--priority":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--priority requires a value")
			}
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("--priority must be an integer, got %q", args[i])
			}
			opts.priority = &p
		case arg == "--parent":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--parent requires a value")
			}
			v := strings.ToLower(strings.TrimSpace(args[i]))
			opts.parent = &v
		case arg == "--clear-description":
			opts.clearDescription = true
		case arg == "--type":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--type requires a value")
			}
			v := args[i]
			opts.taskType = &v
		case arg == "--clear-type":
			opts.clearType = true
		case arg == "--tags":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--tags requires a value")
			}
			v := strings.Split(args[i], ",")
			opts.tags = &v
		case arg == "--clear-tags":
			opts.clearTags = true
		case arg == "--refs":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--refs requires a value")
			}
			v := strings.Split(args[i], ",")
			opts.refs = &v
		case arg == "--clear-refs":
			opts.clearRefs = true
		case arg == "--blocks":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--blocks requires a value")
			}
			opts.blocks = parseCommaSeparatedIDs(args[i])
		default:
			// Positional argument: task ID (first one wins)
			if opts.id == "" {
				opts.id = strings.ToLower(strings.TrimSpace(arg))
			}
		}
		i++
	}
	return opts, nil
}

// rule3Result holds the output of a Rule 3 evaluation for cascade display.
type rule3Result struct {
	parentID    string
	parentTitle string
	result      task.TransitionResult
	cascades    []task.CascadeChange
}

// autoCompleteParentIfTerminal checks if the original parent's remaining children are all
// terminal after a child was reparented away. If so, it triggers auto-completion via
// ApplySystemTransition: done if at least one child is done, cancelled if all children
// are cancelled. Returns nil if auto-completion does not apply.
func autoCompleteParentIfTerminal(tasks []task.Task, origParentID string, sm *task.StateMachine) *rule3Result {
	action, shouldComplete := task.EvaluateParentCompletion(tasks, origParentID)
	if !shouldComplete {
		return nil
	}

	// Find the parent to apply the transition.
	normalizedParentID := task.NormalizeID(origParentID)
	var parentIdx int
	for i := range tasks {
		if task.NormalizeID(tasks[i].ID) == normalizedParentID {
			parentIdx = i
			break
		}
	}

	oldStatus := tasks[parentIdx].Status
	_, cascades, err := sm.ApplySystemTransition(tasks, &tasks[parentIdx], action)
	if err != nil {
		return nil
	}

	return &rule3Result{
		parentID:    tasks[parentIdx].ID,
		parentTitle: tasks[parentIdx].Title,
		result: task.TransitionResult{
			OldStatus: oldStatus,
			NewStatus: tasks[parentIdx].Status,
		},
		cascades: cascades,
	}
}

// RunUpdate executes the update command: validates inputs, applies changes via the storage engine,
// and outputs the updated task details via the Formatter.
func RunUpdate(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	opts, err := parseUpdateArgs(args)
	if err != nil {
		return err
	}

	if opts.id == "" {
		return fmt.Errorf("task ID is required. Usage: tick update <id> [options]")
	}

	if !opts.hasChanges() {
		return fmt.Errorf("at least one flag is required: --title, --description, --clear-description, --priority, --type, --clear-type, --tags, --clear-tags, --refs, --clear-refs, --parent, --blocks")
	}

	// Validate title if provided.
	if opts.title != nil {
		trimmed := task.TrimTitle(*opts.title)
		if err := task.ValidateTitle(trimmed); err != nil {
			return err
		}
	}

	// Validate description flags.
	if opts.description != nil && opts.clearDescription {
		return fmt.Errorf("--description and --clear-description are mutually exclusive")
	}
	if opts.description != nil {
		trimmed := task.TrimDescription(*opts.description)
		if err := task.ValidateDescriptionUpdate(trimmed); err != nil {
			return err
		}
	}

	// Validate type flags.
	if opts.taskType != nil && opts.clearType {
		return fmt.Errorf("--type and --clear-type are mutually exclusive")
	}
	if opts.taskType != nil {
		normalized, typeErr := validateTypeFlag(*opts.taskType)
		if typeErr != nil {
			return typeErr
		}
		opts.taskType = &normalized
	}

	// Validate tags flags.
	if opts.tags != nil && opts.clearTags {
		return fmt.Errorf("--tags and --clear-tags are mutually exclusive")
	}
	if opts.tags != nil {
		deduped, tagsErr := validateTagsFlag(*opts.tags, "--tags cannot be empty; use --clear-tags to remove all tags")
		if tagsErr != nil {
			return tagsErr
		}
		opts.tags = &deduped
	}

	// Validate refs flags.
	if opts.refs != nil && opts.clearRefs {
		return fmt.Errorf("--refs and --clear-refs are mutually exclusive")
	}
	if opts.refs != nil {
		deduped, refsErr := validateRefsFlag(*opts.refs, "--refs cannot be empty; use --clear-refs to remove all refs")
		if refsErr != nil {
			return refsErr
		}
		opts.refs = &deduped
	}

	// Validate priority if provided.
	if opts.priority != nil {
		if err := task.ValidatePriority(*opts.priority); err != nil {
			return err
		}
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	// Resolve partial IDs via store.ResolveID.
	opts.id, err = store.ResolveID(opts.id)
	if err != nil {
		return err
	}
	if opts.parent != nil && *opts.parent != "" {
		resolved, resolveErr := store.ResolveID(*opts.parent)
		if resolveErr != nil {
			return resolveErr
		}
		opts.parent = &resolved
	}
	for i, blockID := range opts.blocks {
		opts.blocks[i], err = store.ResolveID(blockID)
		if err != nil {
			return err
		}
	}

	var updatedID string

	// Rule 6: reopen of done parent when reparenting to it.
	var r6Triggered bool
	var r6ParentID string
	var r6Result task.TransitionResult
	var r6CascadeResult *CascadeResult

	// Rule 3: auto-completion of original parent when reparenting away.
	var r3Result task.TransitionResult
	var r3ParentID string
	var r3CascadeResult *CascadeResult

	err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID set for reference validation with normalized keys.
		idSet := make(map[string]bool, len(tasks))
		for _, t := range tasks {
			idSet[task.NormalizeID(t.ID)] = true
		}

		var sm task.StateMachine

		// Validate referenced IDs exist and handle Rule 6 (reopen done parent).
		if opts.parent != nil && *opts.parent != "" {
			if *opts.parent == opts.id {
				return nil, fmt.Errorf("task %s cannot be its own parent", opts.id)
			}
			if !idSet[*opts.parent] {
				return nil, fmt.Errorf("task %q not found (referenced in --parent)", *opts.parent)
			}
			// Validate parent allows adding children (Rule 7: blocks cancelled parent).
			// If parent is done, trigger reopen cascade (Rule 6).
			r, c, reopened, valErr := validateAndReopenParent(tasks, *opts.parent, &sm)
			if valErr != nil {
				return nil, valErr
			}
			if reopened {
				r6Triggered = true
				r6Result = r
				// Find parent title and build cascade result while tasks slice is valid.
				normalizedParent := task.NormalizeID(*opts.parent)
				var parentTitle string
				for _, tk := range tasks {
					if task.NormalizeID(tk.ID) == normalizedParent {
						r6ParentID = tk.ID
						parentTitle = tk.Title
						break
					}
				}
				cr := buildCascadeResult(r6ParentID, parentTitle, r, c, tasks)
				r6CascadeResult = &cr
			}
		}
		for _, blockID := range opts.blocks {
			if !idSet[blockID] {
				return nil, fmt.Errorf("task %q not found (referenced in --blocks)", blockID)
			}
		}

		now := time.Now().UTC().Truncate(time.Second)

		// Find and update the target task.
		found := false
		for i := range tasks {
			if task.NormalizeID(tasks[i].ID) != opts.id {
				continue
			}
			found = true

			if opts.title != nil {
				tasks[i].Title = task.TrimTitle(*opts.title)
			}
			if opts.clearDescription {
				tasks[i].Description = ""
			} else if opts.description != nil {
				tasks[i].Description = task.TrimDescription(*opts.description)
			}
			if opts.priority != nil {
				tasks[i].Priority = *opts.priority
			}
			if opts.clearType {
				tasks[i].Type = ""
			} else if opts.taskType != nil {
				tasks[i].Type = *opts.taskType
			}
			if opts.clearTags {
				tasks[i].Tags = nil
			} else if opts.tags != nil {
				tasks[i].Tags = *opts.tags
			}
			if opts.clearRefs {
				tasks[i].Refs = nil
			} else if opts.refs != nil {
				tasks[i].Refs = *opts.refs
			}

			// Capture original parent before updating.
			originalParent := tasks[i].Parent

			if opts.parent != nil {
				tasks[i].Parent = *opts.parent
			}
			tasks[i].Updated = now
			updatedID = tasks[i].ID

			// Evaluate Rule 3 on original parent if parent changed and original was non-empty.
			if opts.parent != nil && originalParent != *opts.parent && originalParent != "" {
				r3 := autoCompleteParentIfTerminal(tasks, originalParent, &sm)
				if r3 != nil {
					r3ParentID = r3.parentID
					r3Result = r3.result
					cr := buildCascadeResult(r3.parentID, r3.parentTitle, r3.result, r3.cascades, tasks)
					r3CascadeResult = &cr
				}
			}

			break
		}

		if !found {
			return nil, fmt.Errorf("task '%s' not found", opts.id)
		}

		// For --blocks: add this task's ID to target tasks' blocked_by and refresh updated.
		if len(opts.blocks) > 0 {
			applyBlocks(tasks, opts.id, opts.blocks, now)

			// Validate dependencies (cycle detection + child-blocked-by-parent + cancelled blocker) against full task list.
			for _, blockID := range opts.blocks {
				if err := sm.ValidateAddDep(tasks, blockID, opts.id); err != nil {
					return nil, err
				}
			}
		}

		return tasks, nil
	})
	if err != nil {
		return err
	}

	// Output updated task detail.
	if err := outputMutationResult(store, updatedID, fc, fmtr, stdout); err != nil {
		return err
	}

	// Output Rule 6 cascade info (reopen of done parent).
	if r6Triggered && !fc.Quiet {
		outputTransitionOrCascade(stdout, fmtr, r6ParentID, string(r6Result.OldStatus), string(r6Result.NewStatus), r6CascadeResult)
	}

	// Output Rule 3 cascade info (auto-completion of original parent).
	if r3CascadeResult != nil && !fc.Quiet {
		outputTransitionOrCascade(stdout, fmtr, r3ParentID, string(r3Result.OldStatus), string(r3Result.NewStatus), r3CascadeResult)
	}

	return nil
}
