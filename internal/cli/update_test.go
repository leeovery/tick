package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestUpdate(t *testing.T) {
	t.Run("it updates title with --title flag", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Original title")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--title", "New title"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Title != "New title" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "New title")
		}
	})

	t.Run("it updates description with --description flag", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Has description")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--description", "New desc"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Description != "New desc" {
			t.Errorf("description = %q, want %q", tasks[0].Description, "New desc")
		}
	})

	t.Run("it clears description with --description empty string", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Has description")
		tk.Description = "Old desc"
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--description", ""}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Description != "" {
			t.Errorf("description = %q, want empty string", tasks[0].Description)
		}
	})

	t.Run("it updates priority with --priority flag", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Priority task")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--priority", "0"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Priority != 0 {
			t.Errorf("priority = %d, want 0", tasks[0].Priority)
		}
	})

	t.Run("it updates parent with --parent flag", func(t *testing.T) {
		parent := task.NewTask("tick-bbbbbb", "Parent task")
		tk := task.NewTask("tick-aaaaaa", "Child task")
		dir := initTickProjectWithTasks(t, []task.Task{parent, tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--parent", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		// Find tick-aaaaaa
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if tk.Parent != "tick-bbbbbb" {
					t.Errorf("parent = %q, want %q", tk.Parent, "tick-bbbbbb")
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it clears parent with --parent empty string", func(t *testing.T) {
		parent := task.NewTask("tick-bbbbbb", "Parent task")
		tk := task.NewTask("tick-aaaaaa", "Child task")
		tk.Parent = "tick-bbbbbb"
		dir := initTickProjectWithTasks(t, []task.Task{parent, tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--parent", ""}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if tk.Parent != "" {
					t.Errorf("parent = %q, want empty string", tk.Parent)
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it updates blocks with --blocks flag", func(t *testing.T) {
		target := task.NewTask("tick-bbbbbb", "Target task")
		tk := task.NewTask("tick-aaaaaa", "Blocking task")
		dir := initTickProjectWithTasks(t, []task.Task{target, tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--blocks", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-bbbbbb" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-aaaaaa" {
					t.Errorf("target blocked_by = %v, want [tick-aaaaaa]", tk.BlockedBy)
				}
				return
			}
		}
		t.Fatal("task tick-bbbbbb not found")
	})

	t.Run("it updates multiple fields in a single command", func(t *testing.T) {
		parent := task.NewTask("tick-bbbbbb", "Parent")
		tk := task.NewTask("tick-aaaaaa", "Multi update")
		dir := initTickProjectWithTasks(t, []task.Task{parent, tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"tick", "update", "tick-aaaaaa",
			"--title", "Updated title",
			"--description", "Updated desc",
			"--priority", "1",
			"--parent", "tick-bbbbbb",
		}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if tk.Title != "Updated title" {
					t.Errorf("title = %q, want %q", tk.Title, "Updated title")
				}
				if tk.Description != "Updated desc" {
					t.Errorf("description = %q, want %q", tk.Description, "Updated desc")
				}
				if tk.Priority != 1 {
					t.Errorf("priority = %d, want 1", tk.Priority)
				}
				if tk.Parent != "tick-bbbbbb" {
					t.Errorf("parent = %q, want %q", tk.Parent, "tick-bbbbbb")
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it refreshes updated timestamp on any change", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Timestamp test")
		originalUpdated := tk.Updated
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		// Small delay to ensure timestamp differs
		time.Sleep(1100 * time.Millisecond)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--title", "Changed"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if !tasks[0].Updated.After(originalUpdated) {
			t.Errorf("updated timestamp should be refreshed, was %v, original %v", tasks[0].Updated, originalUpdated)
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Output test")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--title", "New output"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("output should contain task ID, got %q", output)
		}
		if !strings.Contains(output, "New output") {
			t.Errorf("output should contain updated title, got %q", output)
		}
		if !strings.Contains(output, "ID:") {
			t.Errorf("output should contain 'ID:' label, got %q", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Quiet test")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "update", "tick-aaaaaa", "--title", "Quiet update"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaaaaa" {
			t.Errorf("with --quiet, expected only task ID %q, got %q", "tick-aaaaaa", output)
		}
	})

	t.Run("it errors when no flags are provided", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "No flags")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "at least one flag is required") {
			t.Errorf("expected 'at least one flag is required' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when task ID is missing", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Task ID is required") {
			t.Errorf("expected 'Task ID is required' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-nonexist", "--title", "Nope"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("expected 'not found' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors on invalid title", func(t *testing.T) {
		tests := []struct {
			name  string
			title string
		}{
			{"empty", ""},
			{"whitespace only", "   "},
			{"exceeds 500 chars", strings.Repeat("a", 501)},
			{"contains newline", "line1\nline2"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tk := task.NewTask("tick-aaaaaa", "Valid title")
				dir := initTickProjectWithTasks(t, []task.Task{tk})

				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", "update", "tick-aaaaaa", "--title", tt.title}, dir, &stdout, &stderr, false)

				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if !strings.Contains(stderr.String(), "Error:") {
					t.Errorf("expected error in stderr, got %q", stderr.String())
				}

				// Verify no mutation occurred
				tasks := readTasksFromFile(t, dir)
				if tasks[0].Title != "Valid title" {
					t.Errorf("title should not have changed, got %q", tasks[0].Title)
				}
			})
		}
	})

	t.Run("it errors on invalid priority", func(t *testing.T) {
		tests := []struct {
			name string
			val  string
		}{
			{"negative", "-1"},
			{"too high", "5"},
			{"way too high", "99"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tk := task.NewTask("tick-aaaaaa", "Priority test")
				dir := initTickProjectWithTasks(t, []task.Task{tk})

				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", "update", "tick-aaaaaa", "--priority", tt.val}, dir, &stdout, &stderr, false)

				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if !strings.Contains(stderr.String(), "Error:") {
					t.Errorf("expected error in stderr, got %q", stderr.String())
				}
			})
		}
	})

	t.Run("it errors on non-existent parent ID", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Orphan parent")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--parent", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("expected 'not found' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors on non-existent blocks ID", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Orphan blocks")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--blocks", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("expected 'not found' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors on self-referencing parent", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Self parent")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "cannot be its own parent") {
			t.Errorf("expected 'cannot be its own parent' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		parent := task.NewTask("tick-bbbbbb", "Parent")
		tk := task.NewTask("tick-aaaaaa", "Normalize test")
		dir := initTickProjectWithTasks(t, []task.Task{parent, tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "TICK-AAAAAA", "--parent", "TICK-BBBBBB"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				if tk.Parent != "tick-bbbbbb" {
					t.Errorf("parent = %q, want %q (lowercase)", tk.Parent, "tick-bbbbbb")
				}
				return
			}
		}
		t.Fatal("task tick-aaaaaa not found")
	})

	t.Run("it persists changes via atomic write", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Persist test")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "update", "tick-aaaaaa", "--title", "Persisted title"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Read back from file
		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "Persisted title" {
			t.Errorf("persisted title = %q, want %q", tasks[0].Title, "Persisted title")
		}
	})
}
