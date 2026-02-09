package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestDepAdd(t *testing.T) {
	t.Run("it adds a dependency between two existing tasks", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-bbbbbb" {
					t.Errorf("blocked_by = %v, want [tick-bbbbbb]", tk.BlockedBy)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it outputs confirmation on success", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Dependency added: tick-aaaaaa blocked by tick-bbbbbb") {
			t.Errorf("expected confirmation message, got %q", output)
		}
	})

	t.Run("it updates task's updated timestamp", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		originalUpdated := t1.Updated
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		time.Sleep(1100 * time.Millisecond)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if !tk.Updated.After(originalUpdated) {
					t.Errorf("updated timestamp should be refreshed, was %v, original %v", tk.Updated, originalUpdated)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it errors when task_id not found", func(t *testing.T) {
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-nonexist", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("expected 'not found' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when blocked_by_id not found", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("expected 'not found' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors on duplicate dependency", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "already") {
			t.Errorf("expected 'already' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors on self-reference", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "cycle") {
			t.Errorf("expected 'cycle' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when add creates cycle", func(t *testing.T) {
		// t1 blocked by t2, t2 blocked by t3, now adding t3 blocked by t1 -> cycle
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		t2.BlockedBy = []string{"tick-cccccc"}
		t3 := task.NewTask("tick-cccccc", "Task C")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-cccccc", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "cycle") {
			t.Errorf("expected 'cycle' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when add creates child-blocked-by-parent", func(t *testing.T) {
		parent := task.NewTask("tick-bbbbbb", "Parent task")
		child := task.NewTask("tick-aaaaaa", "Child task")
		child.Parent = "tick-bbbbbb"
		dir := initTickProjectWithTasks(t, []task.Task{parent, child})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "parent") {
			t.Errorf("expected 'parent' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it normalizes IDs to lowercase", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "TICK-AAAAAA", "TICK-BBBBBB"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-bbbbbb" {
					t.Errorf("blocked_by = %v, want [tick-bbbbbb]", tk.BlockedBy)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it errors when fewer than two IDs provided", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		tests := []struct {
			name string
			args []string
		}{
			{"no IDs", []string{"tick", "dep", "add"}},
			{"one ID", []string{"tick", "dep", "add", "tick-aaaaaa"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				code := Run(tt.args, dir, &stdout, &stderr, false)

				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if !strings.Contains(stderr.String(), "requires two") {
					t.Errorf("expected 'requires two' in stderr, got %q", stderr.String())
				}
			})
		}
	})

	t.Run("it persists via atomic write", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify by re-reading from file
		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-bbbbbb" {
					t.Errorf("persisted blocked_by = %v, want [tick-bbbbbb]", tk.BlockedBy)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})
}

func TestDepRm(t *testing.T) {
	t.Run("it removes an existing dependency", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if len(tk.BlockedBy) != 0 {
					t.Errorf("blocked_by = %v, want empty", tk.BlockedBy)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it outputs confirmation on success", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Dependency removed: tick-aaaaaa no longer blocked by tick-bbbbbb") {
			t.Errorf("expected confirmation message, got %q", output)
		}
	})

	t.Run("it updates task's updated timestamp", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		originalUpdated := t1.Updated
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		time.Sleep(1100 * time.Millisecond)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if !tk.Updated.After(originalUpdated) {
					t.Errorf("updated timestamp should be refreshed, was %v, original %v", tk.Updated, originalUpdated)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it errors when task_id not found", func(t *testing.T) {
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "tick-nonexist", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("expected 'not found' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when dependency not found in blocked_by", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not blocked by") {
			t.Errorf("expected 'not blocked by' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it removes stale dependency without validating blocked_by_id exists", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-deleted"}
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-deleted"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify the stale dependency was removed.
		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if len(tk.BlockedBy) != 0 {
					t.Errorf("blocked_by = %v, want empty", tk.BlockedBy)
				}
				// Verify confirmation output is printed.
				output := stdout.String()
				if !strings.Contains(output, "Dependency removed: tick-aaaaaa no longer blocked by tick-deleted") {
					t.Errorf("expected confirmation message, got %q", output)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it normalizes IDs to lowercase", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "TICK-AAAAAA", "TICK-BBBBBB"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if len(tk.BlockedBy) != 0 {
					t.Errorf("blocked_by = %v, want empty", tk.BlockedBy)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it errors when fewer than two IDs provided", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		tests := []struct {
			name string
			args []string
		}{
			{"no IDs", []string{"tick", "dep", "rm"}},
			{"one ID", []string{"tick", "dep", "rm", "tick-aaaaaa"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				code := Run(tt.args, dir, &stdout, &stderr, false)

				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if !strings.Contains(stderr.String(), "requires two") {
					t.Errorf("expected 'requires two' in stderr, got %q", stderr.String())
				}
			})
		}
	})

	t.Run("it persists via atomic write", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb", "tick-cccccc"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		t3 := task.NewTask("tick-cccccc", "Task C")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify by re-reading from file
		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-cccccc" {
					t.Errorf("persisted blocked_by = %v, want [tick-cccccc]", tk.BlockedBy)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})
}
