package migrate

import (
	"errors"
	"strings"
	"testing"
)

// mockTaskCreator is a test double satisfying TaskCreator.
type mockTaskCreator struct {
	calls   []MigratedTask
	ids     []string // IDs to return in order
	err     error    // if set, CreateTask returns this error
	callIdx int
}

func (m *mockTaskCreator) CreateTask(t MigratedTask) (string, error) {
	m.calls = append(m.calls, t)
	if m.err != nil {
		return "", m.err
	}
	id := ""
	if m.callIdx < len(m.ids) {
		id = m.ids[m.callIdx]
	}
	m.callIdx++
	return id, nil
}

func TestEngineRun(t *testing.T) {
	t.Run("it calls Validate on each MigratedTask before insertion", func(t *testing.T) {
		// One valid task and one invalid (bad status). The invalid task should
		// never reach the creator because Validate is called first.
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Valid task"},
				{Title: "Bad status", Status: "completed"}, // invalid status
				{Title: "Another valid"},
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1", "id-2"},
		}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		// Creator should only have been called for the 2 valid tasks
		if len(creator.calls) != 2 {
			t.Fatalf("expected 2 CreateTask calls, got %d", len(creator.calls))
		}
		if creator.calls[0].Title != "Valid task" {
			t.Errorf("first call Title = %q, want %q", creator.calls[0].Title, "Valid task")
		}
		if creator.calls[1].Title != "Another valid" {
			t.Errorf("second call Title = %q, want %q", creator.calls[1].Title, "Another valid")
		}
		// 3 results total: 2 success, 1 failure
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		if results[1].Success {
			t.Error("expected results[1] to be a failure")
		}
	})

	t.Run("it returns a successful Result for each valid task inserted", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Task A"},
				{Title: "Task B"},
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1", "id-2"},
		}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
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
	})

	t.Run("it skips tasks that fail validation and records failure Result with error", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Good"},
				{Title: "Bad status", Status: "invalid_status"},
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1"},
		}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		// First result should be success
		if !results[0].Success {
			t.Error("results[0].Success = false, want true")
		}
		// Second result should be failure with error
		if results[1].Success {
			t.Error("results[1].Success = true, want false")
		}
		if results[1].Err == nil {
			t.Fatal("results[1].Err = nil, want validation error")
		}
		if !strings.Contains(results[1].Err.Error(), "invalid status") {
			t.Errorf("results[1].Err = %q, want to contain 'invalid status'", results[1].Err.Error())
		}
		if results[1].Title != "Bad status" {
			t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Bad status")
		}
	})

	t.Run("it returns error immediately when provider.Tasks() fails", func(t *testing.T) {
		providerErr := errors.New("source unavailable")
		provider := &mockProvider{
			name: "broken",
			err:  providerErr,
		}
		creator := &mockTaskCreator{}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != providerErr {
			t.Errorf("err = %v, want %v", err, providerErr)
		}
		if results != nil {
			t.Errorf("expected nil results, got %v", results)
		}
	})

	t.Run("it returns empty Results slice when provider returns zero tasks", func(t *testing.T) {
		provider := &mockProvider{
			name:  "empty",
			tasks: []MigratedTask{},
		}
		creator := &mockTaskCreator{}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if results == nil {
			t.Fatal("expected non-nil results slice")
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("it returns error immediately when TaskCreator.CreateTask fails", func(t *testing.T) {
		insertErr := errors.New("storage write failed")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Task A"},
				{Title: "Task B"},
			},
		}
		creator := &mockTaskCreator{
			err: insertErr,
		}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != insertErr {
			t.Errorf("err = %v, want %v", err, insertErr)
		}
		// Should have partial results: the failed one
		if len(results) != 1 {
			t.Fatalf("expected 1 partial result, got %d", len(results))
		}
		if results[0].Success {
			t.Error("results[0].Success = true, want false")
		}
		if results[0].Err != insertErr {
			t.Errorf("results[0].Err = %v, want %v", results[0].Err, insertErr)
		}
	})

	t.Run("it processes all tasks in order and returns Results in same order", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "First"},
				{Title: "Second"},
				{Title: "Third"},
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1", "id-2", "id-3"},
		}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		expected := []string{"First", "Second", "Third"}
		for i, want := range expected {
			if results[i].Title != want {
				t.Errorf("results[%d].Title = %q, want %q", i, results[i].Title, want)
			}
		}
		// Verify creator received tasks in same order
		if len(creator.calls) != 3 {
			t.Fatalf("expected 3 CreateTask calls, got %d", len(creator.calls))
		}
		for i, want := range expected {
			if creator.calls[i].Title != want {
				t.Errorf("creator.calls[%d].Title = %q, want %q", i, creator.calls[i].Title, want)
			}
		}
	})

	t.Run("it applies defaults via TaskCreator — empty status becomes open, zero priority becomes 2", func(t *testing.T) {
		// This test verifies that the TaskCreator receives the raw MigratedTask
		// with empty/zero values, and that the TaskCreator is responsible for
		// applying defaults. We verify the engine passes the task as-is.
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Default task"}, // empty status, nil priority
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1"},
		}
		engine := NewEngine(creator)

		_, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(creator.calls) != 1 {
			t.Fatalf("expected 1 CreateTask call, got %d", len(creator.calls))
		}
		// Engine passes task as-is; defaults are the creator's responsibility.
		task := creator.calls[0]
		if task.Status != "" {
			t.Errorf("expected empty status passed to creator, got %q", task.Status)
		}
		if task.Priority != nil {
			t.Errorf("expected nil priority passed to creator, got %v", task.Priority)
		}
	})

	t.Run("it records Result with fallback title when task has empty title and fails validation", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: ""},
			},
		}
		creator := &mockTaskCreator{}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Title != "(untitled)" {
			t.Errorf("results[0].Title = %q, want %q", results[0].Title, "(untitled)")
		}
		if results[0].Success {
			t.Error("expected results[0].Success = false")
		}
		if results[0].Err == nil {
			t.Error("expected results[0].Err to be non-nil")
		}
	})

	t.Run("it continues past validation failures but stops on insertion failures", func(t *testing.T) {
		insertErr := errors.New("disk full")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Bad status", Status: "invalid"}, // fails validation
				{Title: "Good task"},                     // passes validation, fails insertion
				{Title: "Never reached"},                 // should not be processed
			},
		}
		creator := &mockTaskCreator{
			err: insertErr,
		}
		engine := NewEngine(creator)

		results, err := engine.Run(provider)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != insertErr {
			t.Errorf("err = %v, want %v", err, insertErr)
		}
		// Should have 2 results: validation failure + insertion failure
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		// First: validation failure (skipped, continued)
		if results[0].Success {
			t.Error("results[0].Success = true, want false (validation failure)")
		}
		if results[0].Title != "Bad status" {
			t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Bad status")
		}
		// Second: insertion failure (stopped)
		if results[1].Success {
			t.Error("results[1].Success = true, want false (insertion failure)")
		}
		if results[1].Err != insertErr {
			t.Errorf("results[1].Err = %v, want %v", results[1].Err, insertErr)
		}
		// Creator should have been called exactly once (for "Good task")
		if len(creator.calls) != 1 {
			t.Fatalf("expected 1 CreateTask call, got %d", len(creator.calls))
		}
	})
}
