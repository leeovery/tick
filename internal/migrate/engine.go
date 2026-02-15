package migrate

import "strings"

// completedStatuses defines statuses that indicate a task is finished.
var completedStatuses = map[string]bool{
	"done":      true,
	"cancelled": true,
}

// Options configures Engine behavior.
type Options struct {
	PendingOnly bool
}

// filterPending returns a new slice containing only tasks whose status
// is not "done" or "cancelled". Tasks with status "open", "in_progress",
// or "" (empty) are retained.
func filterPending(tasks []MigratedTask) []MigratedTask {
	result := make([]MigratedTask, 0, len(tasks))
	for _, t := range tasks {
		if !completedStatuses[t.Status] {
			result = append(result, t)
		}
	}
	return result
}

// TaskCreator abstracts tick-core task creation so the migrate package
// remains decoupled from tick-core internals.
type TaskCreator interface {
	// CreateTask persists a MigratedTask into tick's data store.
	// It returns the generated tick ID, or an error if insertion fails.
	CreateTask(t MigratedTask) (string, error)
}

// Engine orchestrates migration from a Provider to tick's data store
// via a TaskCreator.
type Engine struct {
	creator TaskCreator
	opts    Options
}

// NewEngine creates an Engine that uses the given TaskCreator for persistence.
// Options controls filtering and other engine behaviors.
func NewEngine(creator TaskCreator, opts Options) *Engine {
	return &Engine{creator: creator, opts: opts}
}

// Run fetches tasks from the provider, validates each one, inserts valid tasks
// via the TaskCreator, and returns a Result per task. Both validation and
// insertion failures are recorded as failed Results; processing continues for
// all remaining tasks. The returned error is non-nil only when provider.Tasks()
// fails — individual task failures are captured in the Results slice.
func (e *Engine) Run(provider Provider) ([]Result, error) {
	tasks, err := provider.Tasks()
	if err != nil {
		return nil, err
	}

	if e.opts.PendingOnly {
		tasks = filterPending(tasks)
	}

	results := make([]Result, 0, len(tasks))
	for _, task := range tasks {
		if err := task.Validate(); err != nil {
			title := task.Title
			if strings.TrimSpace(title) == "" {
				title = FallbackTitle
			}
			results = append(results, Result{Title: title, Success: false, Err: err})
			continue
		}

		if _, err := e.creator.CreateTask(task); err != nil {
			results = append(results, Result{Title: task.Title, Success: false, Err: err})
			continue
		}

		results = append(results, Result{Title: task.Title, Success: true})
	}

	return results, nil
}
