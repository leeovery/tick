package migrate

import (
	"time"

	"github.com/leeovery/tick/internal/task"
)

// Mutator abstracts tick-core's store mutation interface.
// This decouples the migration package from the concrete storage implementation.
type Mutator interface {
	Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error
}

// StoreTaskCreator implements TaskCreator by persisting MigratedTask values
// into tick's data store via the Mutator interface.
type StoreTaskCreator struct {
	store Mutator
}

// Compile-time check that StoreTaskCreator satisfies TaskCreator.
var _ TaskCreator = (*StoreTaskCreator)(nil)

// NewStoreTaskCreator creates a StoreTaskCreator that writes to the given store.
func NewStoreTaskCreator(store Mutator) *StoreTaskCreator {
	return &StoreTaskCreator{store: store}
}

// CreateTask generates a tick ID, applies defaults to the MigratedTask fields,
// builds a tick-core Task, and persists it via the store. Returns the generated
// ID or an error if persistence fails.
func (c *StoreTaskCreator) CreateTask(mt MigratedTask) (string, error) {
	var generatedID string

	err := c.store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
		// Build ID existence checker from current tasks.
		idSet := make(map[string]bool, len(tasks))
		for _, t := range tasks {
			idSet[task.NormalizeID(t.ID)] = true
		}
		exists := func(id string) bool {
			return idSet[id]
		}

		id, err := task.GenerateID(exists)
		if err != nil {
			return nil, err
		}
		generatedID = id

		// Apply defaults.
		status := task.Status(mt.Status)
		if mt.Status == "" {
			status = task.StatusOpen
		}

		priority := 2
		if mt.Priority != nil {
			priority = *mt.Priority
		}

		created := mt.Created
		if created.IsZero() {
			created = time.Now().UTC().Truncate(time.Second)
		}

		updated := mt.Updated
		if updated.IsZero() {
			updated = created
		}

		var closed *time.Time
		if !mt.Closed.IsZero() {
			c := mt.Closed
			closed = &c
		}

		newTask := task.Task{
			ID:          id,
			Title:       mt.Title,
			Status:      status,
			Priority:    priority,
			Description: mt.Description,
			Created:     created,
			Updated:     updated,
			Closed:      closed,
		}

		return append(tasks, newTask), nil
	})

	if err != nil {
		return "", err
	}

	return generatedID, nil
}
