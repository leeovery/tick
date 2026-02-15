package migrate

import (
	"errors"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// mockStore is a test double for Store.Mutate. It captures the mutation function
// and optionally returns an error.
type mockStore struct {
	mutateErr   error
	mutateCalls int
	// tasks holds the current task list passed to the mutation function.
	tasks []task.Task
	// mutated holds the result of the mutation function after Mutate is called.
	mutated []task.Task
}

func (m *mockStore) Mutate(fn func(tasks []task.Task) ([]task.Task, error)) error {
	m.mutateCalls++
	if m.mutateErr != nil {
		return m.mutateErr
	}
	result, err := fn(m.tasks)
	if err != nil {
		return err
	}
	m.mutated = result
	return nil
}

func TestStoreTaskCreator(t *testing.T) {
	t.Run("StoreTaskCreator creates a tick task from MigratedTask with all fields", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		p := 3
		created := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 12, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

		mt := MigratedTask{
			Title:       "Implement login flow",
			Status:      task.StatusDone,
			Priority:    &p,
			Description: "Full description here",
			Created:     created,
			Updated:     updated,
			Closed:      closed,
		}

		id, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}
		if id == "" {
			t.Fatal("expected non-empty ID")
		}

		// Verify the task was persisted via the store
		if len(store.mutated) != 1 {
			t.Fatalf("expected 1 task in store, got %d", len(store.mutated))
		}
		tk := store.mutated[0]
		if tk.ID != id {
			t.Errorf("task ID = %q, want %q", tk.ID, id)
		}
		if tk.Title != "Implement login flow" {
			t.Errorf("title = %q, want %q", tk.Title, "Implement login flow")
		}
		if tk.Status != task.StatusDone {
			t.Errorf("status = %q, want %q", tk.Status, task.StatusDone)
		}
		if tk.Priority != 3 {
			t.Errorf("priority = %d, want 3", tk.Priority)
		}
		if tk.Description != "Full description here" {
			t.Errorf("description = %q, want %q", tk.Description, "Full description here")
		}
		if !tk.Created.Equal(created) {
			t.Errorf("created = %v, want %v", tk.Created, created)
		}
		if !tk.Updated.Equal(updated) {
			t.Errorf("updated = %v, want %v", tk.Updated, updated)
		}
		if tk.Closed == nil {
			t.Fatal("expected non-nil closed time")
		}
		if !tk.Closed.Equal(closed) {
			t.Errorf("closed = %v, want %v", *tk.Closed, closed)
		}
	})

	t.Run("StoreTaskCreator applies default status open when MigratedTask status is empty", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		mt := MigratedTask{
			Title: "Task with empty status",
		}

		_, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}

		tk := store.mutated[0]
		if tk.Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tk.Status, task.StatusOpen)
		}
	})

	t.Run("StoreTaskCreator applies default priority 2 when MigratedTask priority is nil", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		mt := MigratedTask{
			Title: "Task with nil priority",
		}

		_, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}

		tk := store.mutated[0]
		if tk.Priority != 2 {
			t.Errorf("priority = %d, want 2", tk.Priority)
		}
	})

	t.Run("StoreTaskCreator applies default Created as time.Now when zero", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		mt := MigratedTask{
			Title: "Task with zero created",
		}

		before := time.Now().UTC().Truncate(time.Second)
		_, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		tk := store.mutated[0]
		if tk.Created.Before(before) || tk.Created.After(after) {
			t.Errorf("created = %v, want between %v and %v", tk.Created, before, after)
		}
	})

	t.Run("StoreTaskCreator applies default Updated as Created when zero", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		created := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
		mt := MigratedTask{
			Title:   "Task with zero updated",
			Created: created,
		}

		_, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}

		tk := store.mutated[0]
		if !tk.Updated.Equal(created) {
			t.Errorf("updated = %v, want %v (same as created)", tk.Updated, created)
		}
	})

	t.Run("StoreTaskCreator generates a tick ID for each created task", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		mt := MigratedTask{Title: "Task one"}
		id1, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}

		store2 := &mockStore{}
		creator2 := NewStoreTaskCreator(store2)
		id2, err := creator2.CreateTask(MigratedTask{Title: "Task two"})
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}

		// Both should have tick- prefix
		if len(id1) < 5 || id1[:5] != "tick-" {
			t.Errorf("id1 = %q, want tick- prefix", id1)
		}
		if len(id2) < 5 || id2[:5] != "tick-" {
			t.Errorf("id2 = %q, want tick- prefix", id2)
		}
		// IDs should be different (probabilistically - 6 hex chars = 16M possibilities)
		if id1 == id2 {
			t.Errorf("expected different IDs, both = %q", id1)
		}
	})

	t.Run("StoreTaskCreator returns error when store write fails", func(t *testing.T) {
		storeErr := errors.New("disk full")
		store := &mockStore{mutateErr: storeErr}
		creator := NewStoreTaskCreator(store)

		mt := MigratedTask{Title: "Some task"}
		_, err := creator.CreateTask(mt)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, storeErr) {
			t.Errorf("err = %v, want %v", err, storeErr)
		}
	})

	t.Run("StoreTaskCreator does not set Closed when MigratedTask Closed is zero", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		mt := MigratedTask{
			Title: "Open task",
		}

		_, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}

		tk := store.mutated[0]
		if tk.Closed != nil {
			t.Errorf("expected nil Closed, got %v", *tk.Closed)
		}
	})

	t.Run("StoreTaskCreator preserves provided priority value", func(t *testing.T) {
		store := &mockStore{}
		creator := NewStoreTaskCreator(store)

		p := 0
		mt := MigratedTask{
			Title:    "Zero priority task",
			Priority: &p,
		}

		_, err := creator.CreateTask(mt)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}

		tk := store.mutated[0]
		if tk.Priority != 0 {
			t.Errorf("priority = %d, want 0", tk.Priority)
		}
	})
}
