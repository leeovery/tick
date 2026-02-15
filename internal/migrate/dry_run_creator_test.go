package migrate

import (
	"testing"
	"time"
)

// Compile-time check that DryRunTaskCreator satisfies the TaskCreator interface.
var _ TaskCreator = (*DryRunTaskCreator)(nil)

func TestDryRunTaskCreator(t *testing.T) {
	t.Run("CreateTask returns empty string and nil error", func(t *testing.T) {
		creator := &DryRunTaskCreator{}
		id, err := creator.CreateTask(MigratedTask{Title: "Some task"})
		if id != "" {
			t.Errorf("CreateTask() id = %q, want empty string", id)
		}
		if err != nil {
			t.Errorf("CreateTask() err = %v, want nil", err)
		}
	})

	t.Run("CreateTask never returns an error regardless of input", func(t *testing.T) {
		creator := &DryRunTaskCreator{}
		p := 3
		tasks := []MigratedTask{
			{Title: "Simple task"},
			{Title: "Full task", Status: "done", Priority: &p, Description: "desc",
				Created: time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
				Updated: time.Date(2026, 1, 12, 14, 0, 0, 0, time.UTC),
				Closed:  time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
			{Title: "Empty fields"},
		}
		for _, mt := range tasks {
			id, err := creator.CreateTask(mt)
			if id != "" {
				t.Errorf("CreateTask(%q) id = %q, want empty string", mt.Title, id)
			}
			if err != nil {
				t.Errorf("CreateTask(%q) err = %v, want nil", mt.Title, err)
			}
		}
	})
}

func TestEngineWithDryRunTaskCreator(t *testing.T) {
	t.Run("engine with DryRunTaskCreator produces successful Result for each valid task", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Task A"},
				{Title: "Task B"},
				{Title: "Task C"},
			},
		}
		creator := &DryRunTaskCreator{}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		for i, r := range results {
			if !r.Success {
				t.Errorf("results[%d].Success = false, want true", i)
			}
			if r.Err != nil {
				t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
			}
		}
		if results[0].Title != "Task A" {
			t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Task A")
		}
		if results[1].Title != "Task B" {
			t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Task B")
		}
		if results[2].Title != "Task C" {
			t.Errorf("results[2].Title = %q, want %q", results[2].Title, "Task C")
		}
	})

	t.Run("engine with DryRunTaskCreator still fails validation for tasks with empty title", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Valid task"},
				{Title: ""},
			},
		}
		creator := &DryRunTaskCreator{}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if !results[0].Success {
			t.Error("results[0].Success = false, want true")
		}
		if results[1].Success {
			t.Error("results[1].Success = true, want false")
		}
		if results[1].Err == nil {
			t.Fatal("results[1].Err = nil, want validation error")
		}
	})
}
