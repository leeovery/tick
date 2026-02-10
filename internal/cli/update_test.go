package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runUpdate runs the tick update command with the given args and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runUpdate(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "update"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestUpdate(t *testing.T) {
	t.Run("it updates title with --title flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Old title", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--title", "New title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		if len(persisted) != 1 {
			t.Fatalf("expected 1 task, got %d", len(persisted))
		}
		if persisted[0].Title != "New title" {
			t.Errorf("title = %q, want %q", persisted[0].Title, "New title")
		}
	})

	t.Run("it updates description with --description flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--description", "New description")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		if persisted[0].Description != "New description" {
			t.Errorf("description = %q, want %q", persisted[0].Description, "New description")
		}
	})

	t.Run("it clears description with --description empty string", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Description: "Existing desc", Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--description", "")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		if persisted[0].Description != "" {
			t.Errorf("description = %q, want empty", persisted[0].Description)
		}
	})

	t.Run("it updates priority with --priority flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--priority", "0")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		if persisted[0].Priority != 0 {
			t.Errorf("priority = %d, want 0", persisted[0].Priority)
		}
	})

	t.Run("it updates parent with --parent flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Child", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-bbb222", "--parent", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		var child task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-bbb222" {
				child = tk
			}
		}
		if child.Parent != "tick-aaa111" {
			t.Errorf("parent = %q, want %q", child.Parent, "tick-aaa111")
		}
	})

	t.Run("it clears parent with --parent empty string", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Child", Status: task.StatusOpen, Priority: 2, Parent: "tick-aaa111", Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-bbb222", "--parent", "")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		var child task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-bbb222" {
				child = tk
			}
		}
		if child.Parent != "" {
			t.Errorf("parent = %q, want empty", child.Parent)
		}
	})

	t.Run("it updates blocks with --blocks flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Target", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--blocks", "tick-bbb222", "--title", "Blocker updated")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		var target task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-bbb222" {
				target = tk
			}
		}
		if len(target.BlockedBy) != 1 || target.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("target blocked_by = %v, want [tick-aaa111]", target.BlockedBy)
		}
		// Target's updated timestamp should be refreshed
		if !target.Updated.After(now) {
			t.Error("target's updated timestamp should be refreshed")
		}
	})

	t.Run("it updates multiple fields in a single command", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-ppp111", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa111", Title: "Old title", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111",
			"--title", "New title",
			"--description", "New desc",
			"--priority", "1",
			"--parent", "tick-ppp111",
		)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		var updated task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-aaa111" {
				updated = tk
			}
		}
		if updated.Title != "New title" {
			t.Errorf("title = %q, want %q", updated.Title, "New title")
		}
		if updated.Description != "New desc" {
			t.Errorf("description = %q, want %q", updated.Description, "New desc")
		}
		if updated.Priority != 1 {
			t.Errorf("priority = %d, want 1", updated.Priority)
		}
		if updated.Parent != "tick-ppp111" {
			t.Errorf("parent = %q, want %q", updated.Parent, "tick-ppp111")
		}
	})

	t.Run("it refreshes updated timestamp on any change", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runUpdate(t, dir, "tick-aaa111", "--title", "Updated task")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		persisted := readPersistedTasks(t, tickDir)
		if !persisted[0].Updated.After(now) {
			t.Errorf("updated = %v, should be after %v", persisted[0].Updated, now)
		}
		// Created should not change
		if !persisted[0].Created.Equal(now) {
			t.Errorf("created = %v, should remain %v", persisted[0].Created, now)
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runUpdate(t, dir, "tick-aaa111", "--title", "Updated task")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Output should contain key fields like show format
		if !strings.Contains(stdout, "ID:") {
			t.Errorf("stdout should contain 'ID:' field, got %q", stdout)
		}
		if !strings.Contains(stdout, "tick-aaa111") {
			t.Errorf("stdout should contain task ID, got %q", stdout)
		}
		if !strings.Contains(stdout, "Updated task") {
			t.Errorf("stdout should contain updated title, got %q", stdout)
		}
		if !strings.Contains(stdout, "Status:") {
			t.Errorf("stdout should contain 'Status:' field, got %q", stdout)
		}
		if !strings.Contains(stdout, "Priority:") {
			t.Errorf("stdout should contain 'Priority:' field, got %q", stdout)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, _, exitCode := runUpdate(t, dir, "--quiet", "tick-aaa111", "--title", "Quiet update")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "tick-aaa111\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it errors when no flags are provided", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should contain 'Error:', got %q", stderr)
		}
		// Should mention available flags
		if !strings.Contains(stderr, "--title") {
			t.Errorf("stderr should mention available flags, got %q", stderr)
		}
	})

	t.Run("it errors when task ID is missing", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runUpdate(t, dir)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "task ID is required") {
			t.Errorf("stderr = %q, want to contain 'task ID is required'", stderr)
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runUpdate(t, dir, "tick-nonexist", "--title", "New title")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr = %q, want to contain 'not found'", stderr)
		}
	})

	t.Run("it errors on invalid title (empty/500/newlines)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		tests := []struct {
			name  string
			title string
		}{
			{"empty", ""},
			{"whitespace only", "   "},
			{"exceeds 500 chars", strings.Repeat("x", 501)},
			{"contains newline", "line1\nline2"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir, _ := setupTickProjectWithTasks(t, tasks)
				_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--title", tt.title)
				if exitCode != 1 {
					t.Errorf("exit code = %d, want 1", exitCode)
				}
				if !strings.Contains(stderr, "Error:") {
					t.Errorf("stderr should contain 'Error:', got %q", stderr)
				}
			})
		}
	})

	t.Run("it errors on invalid priority (outside 0-4)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		tests := []struct {
			name     string
			priority string
		}{
			{"negative", "-1"},
			{"above max", "5"},
			{"way above", "100"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir, _ := setupTickProjectWithTasks(t, tasks)
				_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--priority", tt.priority)
				if exitCode != 1 {
					t.Errorf("exit code = %d, want 1", exitCode)
				}
				if !strings.Contains(stderr, "Error:") {
					t.Errorf("stderr should contain 'Error:', got %q", stderr)
				}
			})
		}
	})

	t.Run("it errors on non-existent parent/blocks IDs", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		tests := []struct {
			name string
			args []string
		}{
			{"non-existent parent", []string{"tick-aaa111", "--parent", "tick-nonexist"}},
			{"non-existent blocks", []string{"tick-aaa111", "--blocks", "tick-nonexist", "--title", "Updated"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir, _ := setupTickProjectWithTasks(t, tasks)
				_, stderr, exitCode := runUpdate(t, dir, tt.args...)
				if exitCode != 1 {
					t.Errorf("exit code = %d, want 1", exitCode)
				}
				if !strings.Contains(stderr, "not found") {
					t.Errorf("stderr = %q, want to contain 'not found'", stderr)
				}
			})
		}
	})

	t.Run("it errors on self-referencing parent", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--parent", "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "cannot be its own parent") {
			t.Errorf("stderr = %q, want to contain 'cannot be its own parent'", stderr)
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runUpdate(t, dir, "TICK-BBB222", "--parent", "TICK-AAA111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		persisted := readPersistedTasks(t, tickDir)
		var child task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-bbb222" {
				child = tk
			}
		}
		if child.Parent != "tick-aaa111" {
			t.Errorf("parent = %q, want %q", child.Parent, "tick-aaa111")
		}
	})

	t.Run("it persists changes via atomic write", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Old title", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runUpdate(t, dir, "tick-aaa111", "--title", "New title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Read back from disk to verify persistence
		persisted := readPersistedTasks(t, tickDir)
		if len(persisted) != 1 {
			t.Fatalf("expected 1 task, got %d", len(persisted))
		}
		if persisted[0].Title != "New title" {
			t.Errorf("persisted title = %q, want %q", persisted[0].Title, "New title")
		}
	})

	t.Run("it rejects --blocks that would create child-blocked-by-parent dependency", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		parentTask := task.Task{
			ID: "tick-ppp111", Title: "Parent", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		childTask := task.Task{
			ID: "tick-ccc111", Title: "Child", Status: task.StatusOpen,
			Priority: 2, Parent: "tick-ppp111",
			Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{parentTask, childTask})

		// Parent --blocks child => child.blocked_by += [parent_id] => child blocked by parent => rejected
		_, stderr, exitCode := runUpdate(t, dir, "tick-ppp111", "--blocks", "tick-ccc111", "--title", "Updated parent")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "parent") {
			t.Errorf("stderr should contain 'parent', got %q", stderr)
		}

		// Verify child's blocked_by was not modified
		persisted := readPersistedTasks(t, tickDir)
		var child task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-ccc111" {
				child = tk
				break
			}
		}
		if len(child.BlockedBy) != 0 {
			t.Errorf("child blocked_by = %v, want empty (no changes persisted)", child.BlockedBy)
		}
	})

	t.Run("it rejects --blocks that would create a cycle", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		// taskA is blocked by taskB. Updating taskB --blocks taskA would create:
		// taskA.blocked_by += [taskB] => but taskA already has taskB in blocked_by.
		// That's a duplicate, not a cycle.
		// Better: taskA blocked by taskB. Update taskA --blocks taskB.
		// => taskB.blocked_by += [taskA] => cycle: taskA -> taskB -> taskA
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, BlockedBy: []string{"tick-bbb222"},
			Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--blocks", "tick-bbb222", "--title", "Updated A")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "cycle") {
			t.Errorf("stderr should contain 'cycle', got %q", stderr)
		}

		// Verify taskB's blocked_by was not modified
		persisted := readPersistedTasks(t, tickDir)
		var targetB task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-bbb222" {
				targetB = tk
				break
			}
		}
		if len(targetB.BlockedBy) != 0 {
			t.Errorf("taskB blocked_by = %v, want empty (no changes persisted)", targetB.BlockedBy)
		}
	})
}
