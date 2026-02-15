package migrate

import (
	"errors"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestMigratedTaskValidation(t *testing.T) {
	t.Run("MigratedTask with only title is valid", func(t *testing.T) {
		mt := MigratedTask{Title: "Buy groceries"}
		err := mt.Validate()
		if err != nil {
			t.Errorf("expected valid, got error: %v", err)
		}
	})

	t.Run("MigratedTask with all fields populated is valid", func(t *testing.T) {
		p := 3
		created := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 12, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		mt := MigratedTask{
			Title:       "Implement login flow",
			Status:      task.StatusDone,
			Priority:    &p,
			Description: "Full markdown description here",
			Created:     created,
			Updated:     updated,
			Closed:      closed,
		}
		err := mt.Validate()
		if err != nil {
			t.Errorf("expected valid, got error: %v", err)
		}
	})

	t.Run("MigratedTask with empty title is invalid", func(t *testing.T) {
		mt := MigratedTask{Title: ""}
		err := mt.Validate()
		if err == nil {
			t.Fatal("expected error for empty title, got nil")
		}
	})

	t.Run("MigratedTask with whitespace-only title is invalid", func(t *testing.T) {
		mt := MigratedTask{Title: "   \t  "}
		err := mt.Validate()
		if err == nil {
			t.Fatal("expected error for whitespace-only title, got nil")
		}
	})

	t.Run("MigratedTask with invalid status is rejected", func(t *testing.T) {
		mt := MigratedTask{Title: "Test", Status: task.Status("completed")}
		err := mt.Validate()
		if err == nil {
			t.Fatal("expected error for invalid status, got nil")
		}
	})

	t.Run("MigratedTask with valid status values are accepted", func(t *testing.T) {
		statuses := []task.Status{task.StatusOpen, task.StatusInProgress, task.StatusDone, task.StatusCancelled}
		for _, status := range statuses {
			t.Run(string(status), func(t *testing.T) {
				mt := MigratedTask{Title: "Test", Status: status}
				err := mt.Validate()
				if err != nil {
					t.Errorf("expected valid for status %q, got error: %v", status, err)
				}
			})
		}
	})

	t.Run("MigratedTask with empty status is valid (defaults applied later)", func(t *testing.T) {
		mt := MigratedTask{Title: "Test"}
		err := mt.Validate()
		if err != nil {
			t.Errorf("expected valid for empty status, got error: %v", err)
		}
	})

	t.Run("MigratedTask with priority out of range is rejected", func(t *testing.T) {
		tests := []struct {
			name     string
			priority int
		}{
			{"negative", -1},
			{"too high", 5},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := tt.priority
				mt := MigratedTask{Title: "Test", Priority: &p}
				err := mt.Validate()
				if err == nil {
					t.Errorf("expected error for priority %d, got nil", tt.priority)
				}
			})
		}
	})

	t.Run("MigratedTask with priority in range is accepted", func(t *testing.T) {
		tests := []struct {
			name     string
			priority int
		}{
			{"lower bound", 0},
			{"upper bound", 4},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := tt.priority
				mt := MigratedTask{Title: "Test", Priority: &p}
				err := mt.Validate()
				if err != nil {
					t.Errorf("expected valid for priority %d, got error: %v", tt.priority, err)
				}
			})
		}
	})
}

// TestProviderInterface verifies at compile time that a mock can satisfy the Provider interface.
func TestProviderInterface(t *testing.T) {
	t.Run("Provider interface is implementable by a mock", func(t *testing.T) {
		mock := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Task from mock"},
			},
		}

		// Compile-time check: mockProvider satisfies Provider.
		var _ Provider = mock

		if mock.Name() != "test" {
			t.Errorf("Name() = %q, want %q", mock.Name(), "test")
		}

		tasks, err := mock.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "Task from mock" {
			t.Errorf("task title = %q, want %q", tasks[0].Title, "Task from mock")
		}
	})

	t.Run("Provider mock can return errors", func(t *testing.T) {
		mock := &mockProvider{
			name: "failing",
			err:  errors.New("source unavailable"),
		}

		var _ Provider = mock

		_, err := mock.Tasks()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "source unavailable" {
			t.Errorf("error = %q, want %q", err.Error(), "source unavailable")
		}
	})
}

// TestResultStruct verifies the Result struct has the expected fields.
func TestResultStruct(t *testing.T) {
	t.Run("Result captures success outcome", func(t *testing.T) {
		r := Result{Title: "My task", Success: true, Err: nil}
		if r.Title != "My task" {
			t.Errorf("Title = %q, want %q", r.Title, "My task")
		}
		if !r.Success {
			t.Error("expected Success to be true")
		}
		if r.Err != nil {
			t.Errorf("expected nil Err, got %v", r.Err)
		}
	})

	t.Run("Result captures failure outcome", func(t *testing.T) {
		err := errors.New("validation failed")
		r := Result{Title: "Bad task", Success: false, Err: err}
		if r.Success {
			t.Error("expected Success to be false")
		}
		if r.Err != err {
			t.Errorf("Err = %v, want %v", r.Err, err)
		}
	})
}

// mockProvider is a test double that satisfies the Provider interface.
type mockProvider struct {
	name  string
	tasks []MigratedTask
	err   error
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Tasks() ([]MigratedTask, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}
