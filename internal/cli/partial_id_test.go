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

	t.Run("it updates a task using partial ID as positional arg", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Old title", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Other task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "a3f", "--title", "New title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		for _, tk := range persisted {
			if tk.ID == "tick-a3f1b2" {
				if tk.Title != "New title" {
					t.Errorf("title = %q, want %q", tk.Title, "New title")
				}
				return
			}
		}
		t.Fatal("task tick-a3f1b2 not found in persisted tasks")
	})

	t.Run("it resolves partial ID in --parent flag on update", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Child task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-b12345", "--parent", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		for _, tk := range persisted {
			if tk.ID == "tick-b12345" {
				if tk.Parent != "tick-a3f1b2" {
					t.Errorf("parent = %q, want %q", tk.Parent, "tick-a3f1b2")
				}
				return
			}
		}
		t.Fatal("task tick-b12345 not found in persisted tasks")
	})

	t.Run("it resolves partial IDs in --blocks flag on update", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Target 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-c99887", Title: "Target 2", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-a3f1b2", "--blocks", "b12,c99", "--title", "Blocker updated")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		for _, tk := range persisted {
			if tk.ID == "tick-b12345" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-a3f1b2" {
					t.Errorf("target1 blocked_by = %v, want [tick-a3f1b2]", tk.BlockedBy)
				}
			}
			if tk.ID == "tick-c99887" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-a3f1b2" {
					t.Errorf("target2 blocked_by = %v, want [tick-a3f1b2]", tk.BlockedBy)
				}
			}
		}
	})

	t.Run("it resolves partial ID in --parent flag on create", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runCreate(t, dir, "Child task", "--parent", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		newTask := persisted[len(persisted)-1]
		if newTask.Parent != "tick-a3f1b2" {
			t.Errorf("parent = %q, want %q", newTask.Parent, "tick-a3f1b2")
		}
	})

	t.Run("it resolves partial IDs in --blocked-by flag on create", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Blocker 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-b12345", Title: "Blocker 2", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runCreate(t, dir, "Blocked task", "--blocked-by", "a3f,b12")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		newTask := persisted[len(persisted)-1]
		if len(newTask.BlockedBy) != 2 {
			t.Fatalf("blocked_by length = %d, want 2", len(newTask.BlockedBy))
		}
		if newTask.BlockedBy[0] != "tick-a3f1b2" || newTask.BlockedBy[1] != "tick-b12345" {
			t.Errorf("blocked_by = %v, want [tick-a3f1b2 tick-b12345]", newTask.BlockedBy)
		}
	})

	t.Run("it resolves partial IDs in --blocks flag on create", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runCreate(t, dir, "Blocking task", "--blocks", "a3f")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		var target task.Task
		var newTask task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-a3f1b2" {
				target = tk
			} else {
				newTask = tk
			}
		}
		if len(target.BlockedBy) != 1 || target.BlockedBy[0] != newTask.ID {
			t.Errorf("target blocked_by = %v, want [%s]", target.BlockedBy, newTask.ID)
		}
	})

	t.Run("it detects self-reference when partial --parent resolves to same task on update", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "a3f", "--parent", "a3f")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "cannot be its own parent") {
			t.Errorf("stderr = %q, want to contain 'cannot be its own parent'", stderr)
		}
	})

	t.Run("it errors on ambiguous partial in --blocked-by on create", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-a3f1b3", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runCreate(t, dir, "New task", "--blocked-by", "a3f")
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

	t.Run("it errors on not-found partial in --parent on update", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-a3f1b2", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-a3f1b2", "--parent", "zzz")
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
