package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestPartialIDIntegration(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it shows a task using a 3-char partial ID", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Other task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runShow(t, dir, "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "tick-a3f1b2") {
			t.Errorf("stdout should contain full task ID, got %q", stdout)
		}
		if !strings.Contains(stdout, "Target task") {
			t.Errorf("stdout should contain task title, got %q", stdout)
		}
	})

	t.Run("it transitions a task using a partial ID (start)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Other task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runTransition(t, dir, "start", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "tick-a3f1b2") {
			t.Errorf("stdout should contain full task ID, got %q", stdout)
		}

		persisted := readPersistedTasks(t, tickDir)
		for _, tk := range persisted {
			if tk.ID == "tick-a3f1b2" {
				if tk.Status != task.StatusInProgress {
					t.Errorf("status = %q, want %q", tk.Status, task.StatusInProgress)
				}
				return
			}
		}
		t.Fatal("task tick-a3f1b2 not found in persisted tasks")
	})

	t.Run("it transitions a task using a partial ID (done)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Other task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runTransition(t, dir, "done", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		persisted := readPersistedTasks(t, tickDir)
		for _, tk := range persisted {
			if tk.ID == "tick-a3f1b2" {
				if tk.Status != task.StatusDone {
					t.Errorf("status = %q, want %q", tk.Status, task.StatusDone)
				}
				return
			}
		}
		t.Fatal("task tick-a3f1b2 not found in persisted tasks")
	})

	t.Run("it transitions a task using a partial ID (cancel)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Other task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runTransition(t, dir, "cancel", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		persisted := readPersistedTasks(t, tickDir)
		for _, tk := range persisted {
			if tk.ID == "tick-a3f1b2" {
				if tk.Status != task.StatusCancelled {
					t.Errorf("status = %q, want %q", tk.Status, task.StatusCancelled)
				}
				return
			}
		}
		t.Fatal("task tick-a3f1b2 not found in persisted tasks")
	})

	t.Run("it transitions a task using a partial ID (reopen)", func(t *testing.T) {
		closedAt := now
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closedAt},
			{ID: "tick-b12345", Title: "Other task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runTransition(t, dir, "reopen", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		persisted := readPersistedTasks(t, tickDir)
		for _, tk := range persisted {
			if tk.ID == "tick-a3f1b2" {
				if tk.Status != task.StatusOpen {
					t.Errorf("status = %q, want %q", tk.Status, task.StatusOpen)
				}
				return
			}
		}
		t.Fatal("task tick-a3f1b2 not found in persisted tasks")
	})

	t.Run("it removes a task using a partial ID with --force", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Other task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runRemove(t, dir, "a3f", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		persisted := readPersistedTasks(t, tickDir)
		if len(persisted) != 1 {
			t.Fatalf("expected 1 task remaining, got %d", len(persisted))
		}
		if persisted[0].ID != "tick-b12345" {
			t.Errorf("remaining task ID = %q, want %q", persisted[0].ID, "tick-b12345")
		}
	})

	t.Run("it removes multiple tasks using partial IDs", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-c99887", Title: "Task C", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runRemove(t, dir, "a3f", "b12", "--force")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		persisted := readPersistedTasks(t, tickDir)
		if len(persisted) != 1 {
			t.Fatalf("expected 1 task remaining, got %d", len(persisted))
		}
		if persisted[0].ID != "tick-c99887" {
			t.Errorf("remaining task ID = %q, want %q", persisted[0].ID, "tick-c99887")
		}
	})

	t.Run("it errors with ambiguous prefix on show", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-a3f1b3", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runShow(t, dir, "a3f")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "ambiguous") {
			t.Errorf("stderr should contain 'ambiguous', got %q", stderr)
		}
		if !strings.Contains(stderr, "tick-a3f1b2") {
			t.Errorf("stderr should list tick-a3f1b2, got %q", stderr)
		}
		if !strings.Contains(stderr, "tick-a3f1b3") {
			t.Errorf("stderr should list tick-a3f1b3, got %q", stderr)
		}
	})

	t.Run("it errors with not found prefix on start", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runTransition(t, dir, "start", "zzz")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it still works with full IDs on all commands", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		// show with full ID
		dir, _ := setupTickProjectWithTasks(t, tasks)
		stdout, _, exitCode := runShow(t, dir, "tick-a3f1b2")
		if exitCode != 0 {
			t.Fatalf("show with full ID: exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "tick-a3f1b2") {
			t.Errorf("show with full ID: stdout should contain task ID, got %q", stdout)
		}

		// start with full ID
		dir, tickDir := setupTickProjectWithTasks(t, tasks)
		_, _, exitCode = runTransition(t, dir, "start", "tick-a3f1b2")
		if exitCode != 0 {
			t.Fatalf("start with full ID: exit code = %d, want 0", exitCode)
		}
		persisted := readPersistedTasks(t, tickDir)
		if persisted[0].Status != task.StatusInProgress {
			t.Errorf("start with full ID: status = %q, want %q", persisted[0].Status, task.StatusInProgress)
		}

		// remove with full ID
		dir, tickDir = setupTickProjectWithTasks(t, tasks)
		_, _, exitCode = runRemove(t, dir, "tick-a3f1b2", "--force")
		if exitCode != 0 {
			t.Fatalf("remove with full ID: exit code = %d, want 0", exitCode)
		}
		persisted = readPersistedTasks(t, tickDir)
		if len(persisted) != 0 {
			t.Fatalf("remove with full ID: expected 0 tasks, got %d", len(persisted))
		}
	})
}
