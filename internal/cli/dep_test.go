package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runDep runs a tick dep command and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runDep(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "dep"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestDepAdd(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it adds a dependency between two existing tasks", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, _, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.BlockedBy) != 1 || found.BlockedBy[0] != "tick-bbb222" {
			t.Errorf("blocked_by = %v, want [tick-bbb222]", found.BlockedBy)
		}
	})

	t.Run("it outputs confirmation on success (add)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		stdout, _, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "Dependency added: tick-aaa111 blocked by tick-bbb222\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it updates task's updated timestamp (add)", func(t *testing.T) {
		pastTime := now.Add(-1 * time.Hour)
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: pastTime, Updated: pastTime,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: pastTime, Updated: pastTime,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		before := time.Now().UTC().Truncate(time.Second)
		_, _, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if found.Updated.Before(before) || found.Updated.After(after) {
			t.Errorf("updated = %v, want between %v and %v", found.Updated, before, after)
		}
	})

	t.Run("it errors when task_id not found (add)", func(t *testing.T) {
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskB})

		_, stderr, exitCode := runDep(t, dir, "add", "tick-nonexist", "tick-bbb222")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it errors when blocked_by_id not found (add)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-nonexist")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it errors on duplicate dependency (add)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, BlockedBy: []string{"tick-bbb222"},
			Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, stderr, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-bbb222")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "already") {
			t.Errorf("stderr should contain 'already', got %q", stderr)
		}
	})

	t.Run("it errors on self-reference (add)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "cycle") {
			t.Errorf("stderr should contain 'cycle', got %q", stderr)
		}
	})

	t.Run("it errors when add creates cycle", func(t *testing.T) {
		// A blocked by B, now trying to add B blocked by A => cycle
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, BlockedBy: []string{"tick-bbb222"},
			Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, stderr, exitCode := runDep(t, dir, "add", "tick-bbb222", "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "cycle") {
			t.Errorf("stderr should contain 'cycle', got %q", stderr)
		}
	})

	t.Run("it errors when add creates child-blocked-by-parent", func(t *testing.T) {
		parentTask := task.Task{
			ID: "tick-ppp111", Title: "Parent", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		childTask := task.Task{
			ID: "tick-ccc111", Title: "Child", Status: task.StatusOpen,
			Priority: 2, Parent: "tick-ppp111",
			Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{parentTask, childTask})

		_, stderr, exitCode := runDep(t, dir, "add", "tick-ccc111", "tick-ppp111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "parent") {
			t.Errorf("stderr should contain 'parent', got %q", stderr)
		}
	})

	t.Run("it normalizes IDs to lowercase (add)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		stdout, _, exitCode := runDep(t, dir, "add", "TICK-AAA111", "TICK-BBB222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.BlockedBy) != 1 || found.BlockedBy[0] != "tick-bbb222" {
			t.Errorf("blocked_by = %v, want [tick-bbb222]", found.BlockedBy)
		}
		// Output should use lowercase IDs
		if !strings.Contains(stdout, "tick-aaa111") || !strings.Contains(stdout, "tick-bbb222") {
			t.Errorf("stdout should contain lowercase IDs, got %q", stdout)
		}
	})

	t.Run("it suppresses output with --quiet (add)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		stdout, stderr, exitCode := runDep(t, dir, "add", "--quiet", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if stdout != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("stderr should be empty on success, got %q", stderr)
		}
	})

	t.Run("it errors when fewer than two IDs provided (add)", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runDep(t, dir, "add", "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("stderr should contain usage hint, got %q", stderr)
		}
	})

	t.Run("it errors when no IDs provided (add)", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runDep(t, dir, "add")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("stderr should contain usage hint, got %q", stderr)
		}
	})

	t.Run("it persists via atomic write (add)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		_, _, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Read back from disk to confirm persistence
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.BlockedBy) != 1 || found.BlockedBy[0] != "tick-bbb222" {
			t.Errorf("persisted blocked_by = %v, want [tick-bbb222]", found.BlockedBy)
		}
	})
}

func TestDepRm(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it removes an existing dependency", func(t *testing.T) {
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

		_, _, exitCode := runDep(t, dir, "rm", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.BlockedBy) != 0 {
			t.Errorf("blocked_by = %v, want empty", found.BlockedBy)
		}
	})

	t.Run("it outputs confirmation on success (rm)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, BlockedBy: []string{"tick-bbb222"},
			Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		stdout, _, exitCode := runDep(t, dir, "rm", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		expected := "Dependency removed: tick-aaa111 no longer blocked by tick-bbb222\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it updates task's updated timestamp (rm)", func(t *testing.T) {
		pastTime := now.Add(-1 * time.Hour)
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, BlockedBy: []string{"tick-bbb222"},
			Created: pastTime, Updated: pastTime,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: pastTime, Updated: pastTime,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		before := time.Now().UTC().Truncate(time.Second)
		_, _, exitCode := runDep(t, dir, "rm", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if found.Updated.Before(before) || found.Updated.After(after) {
			t.Errorf("updated = %v, want between %v and %v", found.Updated, before, after)
		}
	})

	t.Run("it errors when task_id not found (rm)", func(t *testing.T) {
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskB})

		_, stderr, exitCode := runDep(t, dir, "rm", "tick-nonexist", "tick-bbb222")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it errors when dependency not found in blocked_by (rm)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runDep(t, dir, "rm", "tick-aaa111", "tick-bbb222")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not a dependency") {
			t.Errorf("stderr should contain 'not a dependency', got %q", stderr)
		}
	})

	t.Run("it normalizes IDs to lowercase (rm)", func(t *testing.T) {
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

		stdout, _, exitCode := runDep(t, dir, "rm", "TICK-AAA111", "TICK-BBB222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.BlockedBy) != 0 {
			t.Errorf("blocked_by = %v, want empty", found.BlockedBy)
		}
		// Output should use lowercase IDs
		if !strings.Contains(stdout, "tick-aaa111") || !strings.Contains(stdout, "tick-bbb222") {
			t.Errorf("stdout should contain lowercase IDs, got %q", stdout)
		}
	})

	t.Run("it suppresses output with --quiet (rm)", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, BlockedBy: []string{"tick-bbb222"},
			Created: now, Updated: now,
		}
		taskB := task.Task{
			ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})

		stdout, stderr, exitCode := runDep(t, dir, "rm", "--quiet", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if stdout != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("stderr should be empty on success, got %q", stderr)
		}
	})

	t.Run("it errors when fewer than two IDs provided (rm)", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runDep(t, dir, "rm", "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("stderr should contain usage hint, got %q", stderr)
		}
	})

	t.Run("it errors when no IDs provided (rm)", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runDep(t, dir, "rm")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("stderr should contain usage hint, got %q", stderr)
		}
	})

	t.Run("it persists via atomic write (rm)", func(t *testing.T) {
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

		_, _, exitCode := runDep(t, dir, "rm", "tick-aaa111", "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Read back from disk to confirm persistence
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.BlockedBy) != 0 {
			t.Errorf("persisted blocked_by = %v, want empty", found.BlockedBy)
		}
	})

	t.Run("rm does not validate blocked_by_id exists as a task", func(t *testing.T) {
		// Task has a stale ref in blocked_by that no longer exists as a task
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, BlockedBy: []string{"tick-deleted"},
			Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runDep(t, dir, "rm", "tick-aaa111", "tick-deleted")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.BlockedBy) != 0 {
			t.Errorf("blocked_by = %v, want empty", found.BlockedBy)
		}
	})
}

func TestDepNoSubcommand(t *testing.T) {
	t.Run("it errors when no sub-subcommand given", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runDep(t, dir)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("stderr should contain usage hint, got %q", stderr)
		}
	})
}
