package migrate

import (
	"errors"
	"strings"
	"testing"
)

// mockTaskCreator is a test double satisfying TaskCreator.
// It supports per-call error control via the errs map (keyed by call index).
type mockTaskCreator struct {
	calls   []MigratedTask
	ids     []string      // IDs to return in order
	errs    map[int]error // per-call errors keyed by call index (0-based)
	callIdx int
}

func (m *mockTaskCreator) CreateTask(t MigratedTask) (string, error) {
	idx := m.callIdx
	m.calls = append(m.calls, t)
	m.callIdx++
	if m.errs != nil {
		if err, ok := m.errs[idx]; ok {
			return "", err
		}
	}
	id := ""
	if idx < len(m.ids) {
		id = m.ids[idx]
	}
	return id, nil
}

func TestFilterPending(t *testing.T) {
	t.Run("filterPending removes tasks with status done", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "Done task", Status: "done"},
			{Title: "Open task", Status: "open"},
		}
		got := filterPending(tasks)
		if len(got) != 1 {
			t.Fatalf("expected 1 task, got %d", len(got))
		}
		if got[0].Title != "Open task" {
			t.Errorf("got[0].Title = %q, want %q", got[0].Title, "Open task")
		}
	})

	t.Run("filterPending removes tasks with status cancelled", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "Cancelled task", Status: "cancelled"},
			{Title: "Open task", Status: "open"},
		}
		got := filterPending(tasks)
		if len(got) != 1 {
			t.Fatalf("expected 1 task, got %d", len(got))
		}
		if got[0].Title != "Open task" {
			t.Errorf("got[0].Title = %q, want %q", got[0].Title, "Open task")
		}
	})

	t.Run("filterPending retains tasks with status open", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "Open task", Status: "open"},
		}
		got := filterPending(tasks)
		if len(got) != 1 {
			t.Fatalf("expected 1 task, got %d", len(got))
		}
		if got[0].Title != "Open task" {
			t.Errorf("got[0].Title = %q, want %q", got[0].Title, "Open task")
		}
	})

	t.Run("filterPending retains tasks with status in_progress", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "In progress task", Status: "in_progress"},
		}
		got := filterPending(tasks)
		if len(got) != 1 {
			t.Fatalf("expected 1 task, got %d", len(got))
		}
		if got[0].Title != "In progress task" {
			t.Errorf("got[0].Title = %q, want %q", got[0].Title, "In progress task")
		}
	})

	t.Run("filterPending retains tasks with empty status", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "No status task", Status: ""},
		}
		got := filterPending(tasks)
		if len(got) != 1 {
			t.Fatalf("expected 1 task, got %d", len(got))
		}
		if got[0].Title != "No status task" {
			t.Errorf("got[0].Title = %q, want %q", got[0].Title, "No status task")
		}
	})

	t.Run("filterPending returns empty slice when all tasks are completed", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "Done", Status: "done"},
			{Title: "Cancelled", Status: "cancelled"},
		}
		got := filterPending(tasks)
		if len(got) != 0 {
			t.Fatalf("expected 0 tasks, got %d", len(got))
		}
	})

	t.Run("filterPending returns all tasks when none are completed", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "Open", Status: "open"},
			{Title: "In progress", Status: "in_progress"},
			{Title: "Empty status", Status: ""},
		}
		got := filterPending(tasks)
		if len(got) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(got))
		}
	})

	t.Run("filterPending with mixed statuses returns only non-completed tasks", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "Open", Status: "open"},
			{Title: "Done", Status: "done"},
			{Title: "In progress", Status: "in_progress"},
			{Title: "Cancelled", Status: "cancelled"},
			{Title: "Empty", Status: ""},
		}
		got := filterPending(tasks)
		if len(got) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(got))
		}
		wantTitles := []string{"Open", "In progress", "Empty"}
		for i, want := range wantTitles {
			if got[i].Title != want {
				t.Errorf("got[%d].Title = %q, want %q", i, got[i].Title, want)
			}
		}
	})

	t.Run("filterPending preserves task order", func(t *testing.T) {
		tasks := []MigratedTask{
			{Title: "C", Status: "open"},
			{Title: "A", Status: "done"},
			{Title: "B", Status: "in_progress"},
			{Title: "D", Status: "cancelled"},
			{Title: "E", Status: ""},
		}
		got := filterPending(tasks)
		if len(got) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(got))
		}
		wantTitles := []string{"C", "B", "E"}
		for i, want := range wantTitles {
			if got[i].Title != want {
				t.Errorf("got[%d].Title = %q, want %q", i, got[i].Title, want)
			}
		}
	})
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
		engine := NewEngine(creator, Options{})

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
		engine := NewEngine(creator, Options{})

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
		engine := NewEngine(creator, Options{})

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
		engine := NewEngine(creator, Options{})

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
		engine := NewEngine(creator, Options{})

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

	t.Run("it continues processing after CreateTask fails and records failure Result", func(t *testing.T) {
		insertErr := errors.New("storage write failed")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Task A"},
				{Title: "Task B"},
				{Title: "Task C"},
			},
		}
		creator := &mockTaskCreator{
			ids:  []string{"id-1", "", "id-3"},
			errs: map[int]error{1: insertErr}, // Task B fails insertion
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		// Task A: success
		if !results[0].Success {
			t.Error("results[0].Success = false, want true")
		}
		// Task B: insertion failure
		if results[1].Success {
			t.Error("results[1].Success = true, want false")
		}
		if results[1].Err != insertErr {
			t.Errorf("results[1].Err = %v, want %v", results[1].Err, insertErr)
		}
		// Task C: success (engine continued after Task B failure)
		if !results[2].Success {
			t.Error("results[2].Success = false, want true")
		}
		// All 3 tasks should have been sent to creator
		if len(creator.calls) != 3 {
			t.Fatalf("expected 3 CreateTask calls, got %d", len(creator.calls))
		}
	})

	t.Run("it returns nil error when all tasks fail insertion", func(t *testing.T) {
		insertErr := errors.New("storage unavailable")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Task A"},
				{Title: "Task B"},
			},
		}
		creator := &mockTaskCreator{
			errs: map[int]error{0: insertErr, 1: insertErr},
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Success {
				t.Errorf("results[%d].Success = true, want false", i)
			}
			if r.Err != insertErr {
				t.Errorf("results[%d].Err = %v, want %v", i, r.Err, insertErr)
			}
		}
	})

	t.Run("it returns nil error with mixed validation and insertion failures", func(t *testing.T) {
		insertErr := errors.New("write failed")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Valid A"},                         // succeeds
				{Title: "Bad status", Status: "completed"}, // fails validation
				{Title: "Valid B"},                         // fails insertion
				{Title: "Valid C"},                         // succeeds
			},
		}
		creator := &mockTaskCreator{
			ids:  []string{"id-1", "", "id-3"},
			errs: map[int]error{1: insertErr}, // second CreateTask call (Valid B) fails
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 4 {
			t.Fatalf("expected 4 results, got %d", len(results))
		}
		// Valid A: success
		if !results[0].Success {
			t.Error("results[0].Success = false, want true")
		}
		// Bad status: validation failure
		if results[1].Success {
			t.Error("results[1].Success = true, want false")
		}
		if !strings.Contains(results[1].Err.Error(), "invalid status") {
			t.Errorf("results[1].Err = %q, want validation error", results[1].Err.Error())
		}
		// Valid B: insertion failure
		if results[2].Success {
			t.Error("results[2].Success = true, want false")
		}
		if results[2].Err != insertErr {
			t.Errorf("results[2].Err = %v, want %v", results[2].Err, insertErr)
		}
		// Valid C: success
		if !results[3].Success {
			t.Error("results[3].Success = false, want true")
		}
	})

	t.Run("failure Result from insertion contains the original CreateTask error", func(t *testing.T) {
		originalErr := errors.New("unique constraint violated: task already exists")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Duplicate task"},
			},
		}
		creator := &mockTaskCreator{
			errs: map[int]error{0: originalErr},
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Err != originalErr {
			t.Errorf("results[0].Err = %v, want exact error %v", results[0].Err, originalErr)
		}
		if results[0].Title != "Duplicate task" {
			t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Duplicate task")
		}
	})

	t.Run("results slice contains entries for all tasks in provider order regardless of success or failure", func(t *testing.T) {
		insertErr := errors.New("write failed")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Alpha"},
				{Title: "Beta"},
				{Title: "Gamma"},
				{Title: "Delta"},
			},
		}
		creator := &mockTaskCreator{
			ids:  []string{"id-1", "", "id-3", "id-4"},
			errs: map[int]error{1: insertErr}, // Beta fails insertion
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 4 {
			t.Fatalf("expected 4 results, got %d", len(results))
		}
		expectedTitles := []string{"Alpha", "Beta", "Gamma", "Delta"}
		for i, want := range expectedTitles {
			if results[i].Title != want {
				t.Errorf("results[%d].Title = %q, want %q", i, results[i].Title, want)
			}
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
		engine := NewEngine(creator, Options{})

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
		engine := NewEngine(creator, Options{})

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
		engine := NewEngine(creator, Options{})

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

	t.Run("successful tasks are persisted even when later tasks fail insertion", func(t *testing.T) {
		insertErr := errors.New("disk full")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "First valid"},
				{Title: "Fails insertion"},
				{Title: "Second valid"},
			},
		}
		creator := &mockTaskCreator{
			ids:  []string{"id-1", "", "id-3"},
			errs: map[int]error{1: insertErr},
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		// All 3 tasks were sent to creator
		if len(creator.calls) != 3 {
			t.Fatalf("expected 3 CreateTask calls, got %d", len(creator.calls))
		}
		// First and third succeeded
		if !results[0].Success {
			t.Error("results[0].Success = false, want true")
		}
		if !results[2].Success {
			t.Error("results[2].Success = false, want true")
		}
		// Second failed
		if results[1].Success {
			t.Error("results[1].Success = true, want false")
		}
	})

	t.Run("all tasks fail insertion returns results with all failures and nil error", func(t *testing.T) {
		errA := errors.New("fail A")
		errB := errors.New("fail B")
		errC := errors.New("fail C")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Task A"},
				{Title: "Task B"},
				{Title: "Task C"},
			},
		}
		creator := &mockTaskCreator{
			errs: map[int]error{0: errA, 1: errB, 2: errC},
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		expectedErrs := []error{errA, errB, errC}
		for i, wantErr := range expectedErrs {
			if results[i].Success {
				t.Errorf("results[%d].Success = true, want false", i)
			}
			if results[i].Err != wantErr {
				t.Errorf("results[%d].Err = %v, want %v", i, results[i].Err, wantErr)
			}
		}
	})

	t.Run("mixed failures: validation then insertion then success produces three Results in order", func(t *testing.T) {
		insertErr := errors.New("insert failed")
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Bad status", Status: "invalid"}, // validation failure
				{Title: "Insert fail"},                   // insertion failure
				{Title: "Success task"},                  // success
			},
		}
		creator := &mockTaskCreator{
			ids:  []string{"", "id-2"},
			errs: map[int]error{0: insertErr}, // first CreateTask call fails
		}
		engine := NewEngine(creator, Options{})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		// Result 0: validation failure
		if results[0].Success {
			t.Error("results[0].Success = true, want false (validation)")
		}
		if results[0].Title != "Bad status" {
			t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Bad status")
		}
		if !strings.Contains(results[0].Err.Error(), "invalid status") {
			t.Errorf("results[0].Err = %q, want validation error", results[0].Err.Error())
		}
		// Result 1: insertion failure
		if results[1].Success {
			t.Error("results[1].Success = true, want false (insertion)")
		}
		if results[1].Title != "Insert fail" {
			t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Insert fail")
		}
		if results[1].Err != insertErr {
			t.Errorf("results[1].Err = %v, want %v", results[1].Err, insertErr)
		}
		// Result 2: success
		if !results[2].Success {
			t.Error("results[2].Success = false, want true")
		}
		if results[2].Title != "Success task" {
			t.Errorf("results[2].Title = %q, want %q", results[2].Title, "Success task")
		}
		// Creator called twice: "Insert fail" and "Success task" (validation failure skips creator)
		if len(creator.calls) != 2 {
			t.Fatalf("expected 2 CreateTask calls, got %d", len(creator.calls))
		}
		if creator.calls[0].Title != "Insert fail" {
			t.Errorf("creator.calls[0].Title = %q, want %q", creator.calls[0].Title, "Insert fail")
		}
		if creator.calls[1].Title != "Success task" {
			t.Errorf("creator.calls[1].Title = %q, want %q", creator.calls[1].Title, "Success task")
		}
	})
}

func TestEnginePendingOnly(t *testing.T) {
	t.Run("engine with PendingOnly true filters completed tasks before processing", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Open task", Status: "open"},
				{Title: "Done task", Status: "done"},
				{Title: "In progress task", Status: "in_progress"},
				{Title: "Cancelled task", Status: "cancelled"},
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1", "id-2"},
		}
		engine := NewEngine(creator, Options{PendingOnly: true})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].Title != "Open task" {
			t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Open task")
		}
		if results[1].Title != "In progress task" {
			t.Errorf("results[1].Title = %q, want %q", results[1].Title, "In progress task")
		}
		if len(creator.calls) != 2 {
			t.Fatalf("expected 2 CreateTask calls, got %d", len(creator.calls))
		}
	})

	t.Run("engine with PendingOnly false does not filter any tasks", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Open task", Status: "open"},
				{Title: "Done task", Status: "done"},
				{Title: "Cancelled task", Status: "cancelled"},
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1", "id-2", "id-3"},
		}
		engine := NewEngine(creator, Options{PendingOnly: false})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		if len(creator.calls) != 3 {
			t.Fatalf("expected 3 CreateTask calls, got %d", len(creator.calls))
		}
	})

	t.Run("engine with PendingOnly true and all tasks completed returns empty Results and nil error", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Done", Status: "done"},
				{Title: "Cancelled", Status: "cancelled"},
			},
		}
		creator := &mockTaskCreator{}
		engine := NewEngine(creator, Options{PendingOnly: true})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if results == nil {
			t.Fatal("expected non-nil results slice")
		}
		if len(results) != 0 {
			t.Fatalf("expected 0 results, got %d", len(results))
		}
		if len(creator.calls) != 0 {
			t.Fatalf("expected 0 CreateTask calls, got %d", len(creator.calls))
		}
	})

	t.Run("engine with PendingOnly true and no completed tasks returns all Results", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Open", Status: "open"},
				{Title: "In progress", Status: "in_progress"},
				{Title: "Empty", Status: ""},
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1", "id-2", "id-3"},
		}
		engine := NewEngine(creator, Options{PendingOnly: true})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		wantTitles := []string{"Open", "In progress", "Empty"}
		for i, want := range wantTitles {
			if results[i].Title != want {
				t.Errorf("results[%d].Title = %q, want %q", i, results[i].Title, want)
			}
		}
	})

	t.Run("engine with PendingOnly true still validates remaining tasks", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			tasks: []MigratedTask{
				{Title: "Valid open", Status: "open"},
				{Title: "Done task", Status: "done"},
				{Title: "", Status: "open"}, // invalid: empty title
			},
		}
		creator := &mockTaskCreator{
			ids: []string{"id-1"},
		}
		engine := NewEngine(creator, Options{PendingOnly: true})

		results, err := engine.Run(provider)
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		// "Done task" filtered out, 2 remaining: "Valid open" (success) and "" (validation failure)
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if !results[0].Success {
			t.Error("results[0].Success = false, want true")
		}
		if results[0].Title != "Valid open" {
			t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Valid open")
		}
		if results[1].Success {
			t.Error("results[1].Success = true, want false")
		}
		if results[1].Err == nil {
			t.Error("results[1].Err = nil, want validation error")
		}
		// Only 1 task sent to creator (the valid one)
		if len(creator.calls) != 1 {
			t.Fatalf("expected 1 CreateTask call, got %d", len(creator.calls))
		}
	})
}
