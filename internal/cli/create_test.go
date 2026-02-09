package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

// initTickProject creates a .tick/ directory with an empty tasks.jsonl in a temp dir.
func initTickProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("tick init failed: %s", stderr.String())
	}
	return dir
}

// initTickProjectWithTasks creates a .tick/ project with pre-existing tasks.
func initTickProjectWithTasks(t *testing.T, tasks []task.Task) string {
	t.Helper()
	dir := initTickProject(t)
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")

	var content []byte
	for _, tk := range tasks {
		data, err := tk.MarshalJSON()
		if err != nil {
			t.Fatalf("marshaling task: %v", err)
		}
		content = append(content, data...)
		content = append(content, '\n')
	}
	if err := os.WriteFile(jsonlPath, content, 0644); err != nil {
		t.Fatalf("writing tasks.jsonl: %v", err)
	}
	return dir
}

// readTasksFromFile reads and parses tasks from the project's tasks.jsonl.
func readTasksFromFile(t *testing.T, dir string) []task.Task {
	t.Helper()
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("reading tasks.jsonl: %v", err)
	}
	var tasks []task.Task
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var tk task.Task
		if err := json.Unmarshal([]byte(line), &tk); err != nil {
			t.Fatalf("parsing task line %q: %v", line, err)
		}
		tasks = append(tasks, tk)
	}
	return tasks
}

func TestCreate(t *testing.T) {
	t.Run("it creates a task with only a title (defaults applied)", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "My first task"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "My first task" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "My first task")
		}
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
		if tasks[0].Priority != 2 {
			t.Errorf("priority = %d, want 2", tasks[0].Priority)
		}
	})

	t.Run("it creates a task with all optional fields specified", func(t *testing.T) {
		existingTask := task.NewTask("tick-aaaaaa", "Existing blocker")
		existingTask2 := task.NewTask("tick-bbbbbb", "Another task")
		parentTask := task.NewTask("tick-cccccc", "Parent task")
		dir := initTickProjectWithTasks(t, []task.Task{existingTask, existingTask2, parentTask})

		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"tick", "create", "Full task",
			"--priority", "1",
			"--description", "A detailed description",
			"--blocked-by", "tick-aaaaaa,tick-bbbbbb",
			"--parent", "tick-cccccc",
		}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		// 3 existing + 1 new
		if len(tasks) != 4 {
			t.Fatalf("expected 4 tasks, got %d", len(tasks))
		}
		newTask := tasks[3]
		if newTask.Title != "Full task" {
			t.Errorf("title = %q, want %q", newTask.Title, "Full task")
		}
		if newTask.Priority != 1 {
			t.Errorf("priority = %d, want 1", newTask.Priority)
		}
		if newTask.Description != "A detailed description" {
			t.Errorf("description = %q, want %q", newTask.Description, "A detailed description")
		}
		if len(newTask.BlockedBy) != 2 || newTask.BlockedBy[0] != "tick-aaaaaa" || newTask.BlockedBy[1] != "tick-bbbbbb" {
			t.Errorf("blocked_by = %v, want [tick-aaaaaa tick-bbbbbb]", newTask.BlockedBy)
		}
		if newTask.Parent != "tick-cccccc" {
			t.Errorf("parent = %q, want %q", newTask.Parent, "tick-cccccc")
		}
	})

	t.Run("it generates a unique ID for the created task", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		Run([]string{"tick", "create", "Task one"}, dir, &stdout, &stderr, false)

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
		if !pattern.MatchString(tasks[0].ID) {
			t.Errorf("ID %q does not match pattern tick-{6 hex}", tasks[0].ID)
		}
	})

	t.Run("it sets status to open on creation", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		Run([]string{"tick", "create", "Open task"}, dir, &stdout, &stderr, false)

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
	})

	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		Run([]string{"tick", "create", "Default priority task"}, dir, &stdout, &stderr, false)

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Priority != 2 {
			t.Errorf("priority = %d, want 2", tasks[0].Priority)
		}
	})

	t.Run("it sets priority from --priority flag", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "High priority", "--priority", "0"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Priority != 0 {
			t.Errorf("priority = %d, want 0", tasks[0].Priority)
		}
	})

	t.Run("it rejects priority outside 0-4 range", func(t *testing.T) {
		dir := initTickProject(t)

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
				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", "create", "Bad priority", "--priority", tt.val}, dir, &stdout, &stderr, false)

				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if !strings.Contains(stderr.String(), "Error:") {
					t.Errorf("expected error in stderr, got %q", stderr.String())
				}
			})
		}
	})

	t.Run("it sets description from --description flag", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Described task", "--description", "Some details"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Description != "Some details" {
			t.Errorf("description = %q, want %q", tasks[0].Description, "Some details")
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (single ID)", func(t *testing.T) {
		existing := task.NewTask("tick-aaaaaa", "Blocker")
		dir := initTickProjectWithTasks(t, []task.Task{existing})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Blocked task", "--blocked-by", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		newTask := tasks[len(tasks)-1]
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaaaaa" {
			t.Errorf("blocked_by = %v, want [tick-aaaaaa]", newTask.BlockedBy)
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)", func(t *testing.T) {
		e1 := task.NewTask("tick-aaaaaa", "Blocker 1")
		e2 := task.NewTask("tick-bbbbbb", "Blocker 2")
		dir := initTickProjectWithTasks(t, []task.Task{e1, e2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Multi blocked", "--blocked-by", "tick-aaaaaa,tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		newTask := tasks[len(tasks)-1]
		if len(newTask.BlockedBy) != 2 {
			t.Fatalf("expected 2 blocked_by entries, got %d", len(newTask.BlockedBy))
		}
		if newTask.BlockedBy[0] != "tick-aaaaaa" || newTask.BlockedBy[1] != "tick-bbbbbb" {
			t.Errorf("blocked_by = %v, want [tick-aaaaaa tick-bbbbbb]", newTask.BlockedBy)
		}
	})

	t.Run("it updates target tasks' blocked_by when --blocks is used", func(t *testing.T) {
		e1 := task.NewTask("tick-aaaaaa", "Task to be blocked")
		dir := initTickProjectWithTasks(t, []task.Task{e1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Blocking task", "--blocks", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		// Find tick-aaaaaa and check its blocked_by now contains the new task's ID
		var targetTask task.Task
		var newTask task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaaaaa" {
				targetTask = tk
			} else {
				newTask = tk
			}
		}

		if len(targetTask.BlockedBy) != 1 {
			t.Fatalf("expected target task to have 1 blocked_by entry, got %d", len(targetTask.BlockedBy))
		}
		if targetTask.BlockedBy[0] != newTask.ID {
			t.Errorf("target blocked_by = %v, want [%s]", targetTask.BlockedBy, newTask.ID)
		}
	})

	t.Run("it sets parent from --parent flag", func(t *testing.T) {
		parentTask := task.NewTask("tick-aaaaaa", "Parent")
		dir := initTickProjectWithTasks(t, []task.Task{parentTask})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Child task", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		newTask := tasks[len(tasks)-1]
		if newTask.Parent != "tick-aaaaaa" {
			t.Errorf("parent = %q, want %q", newTask.Parent, "tick-aaaaaa")
		}
	})

	t.Run("it errors when title is missing", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Title is required") {
			t.Errorf("expected 'Title is required' in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when title is empty string", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", ""}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("expected error in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when title is whitespace only", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "   "}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("expected error in stderr, got %q", stderr.String())
		}
	})

	t.Run("it errors when --blocked-by references non-existent task", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Orphaned dep", "--blocked-by", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("expected error in stderr, got %q", stderr.String())
		}

		// Verify no task was created
		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks (no partial mutation), got %d", len(tasks))
		}
	})

	t.Run("it errors when --blocks references non-existent task", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Orphaned blocks", "--blocks", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("expected error in stderr, got %q", stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks (no partial mutation), got %d", len(tasks))
		}
	})

	t.Run("it errors when --parent references non-existent task", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Orphaned parent", "--parent", "tick-nonexist"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("expected error in stderr, got %q", stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks (no partial mutation), got %d", len(tasks))
		}
	})

	t.Run("it persists the task to tasks.jsonl via atomic write", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		Run([]string{"tick", "create", "Persisted task"}, dir, &stdout, &stderr, false)

		// Read raw file to verify it was written
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("reading tasks.jsonl: %v", err)
		}
		if !strings.Contains(string(data), "Persisted task") {
			t.Error("tasks.jsonl should contain the created task")
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Visible task"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Visible task") {
			t.Errorf("output should contain task title, got %q", output)
		}
		if !strings.Contains(output, "tick-") {
			t.Errorf("output should contain task ID, got %q", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "create", "Quiet task"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
		if !pattern.MatchString(output) {
			t.Errorf("with --quiet, expected only task ID, got %q", output)
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		existing := task.NewTask("tick-aaaaaa", "Existing")
		dir := initTickProjectWithTasks(t, []task.Task{existing})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "Normalized", "--blocked-by", "TICK-AAAAAA"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		newTask := tasks[len(tasks)-1]
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaaaaa" {
			t.Errorf("blocked_by should be lowercase, got %v", newTask.BlockedBy)
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "create", "  Trimmed title  "}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if tasks[0].Title != "Trimmed title" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "Trimmed title")
		}
	})

	t.Run("it rejects --blocked-by that would create child-blocked-by-parent", func(t *testing.T) {
		parent := task.NewTask("tick-aaaaaa", "Parent task")
		dir := initTickProjectWithTasks(t, []task.Task{parent})

		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"tick", "create", "Child task",
			"--parent", "tick-aaaaaa",
			"--blocked-by", "tick-aaaaaa",
		}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "parent") {
			t.Errorf("expected 'parent' in stderr, got %q", stderr.String())
		}

		// Verify no task was created.
		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Errorf("expected 1 task (no new task created), got %d", len(tasks))
		}
	})

	t.Run("it rejects --blocked-by that would create a cycle via --blocks", func(t *testing.T) {
		// A is blocked by B. Create C with --blocked-by A --blocks B.
		// This means: C blocked_by A, and B blocked_by C.
		// Chain: B -> C -> A -> B = cycle.
		taskA := task.NewTask("tick-aaaaaa", "Task A")
		taskA.BlockedBy = []string{"tick-bbbbbb"}
		taskB := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{taskA, taskB})

		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"tick", "create", "Task C",
			"--blocked-by", "tick-aaaaaa",
			"--blocks", "tick-bbbbbb",
		}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "cycle") {
			t.Errorf("expected 'cycle' in stderr, got %q", stderr.String())
		}

		// Verify no task was created.
		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks (no new task created), got %d", len(tasks))
		}
	})

	t.Run("it rejects --blocks that would create a direct cycle", func(t *testing.T) {
		// A exists. Create B with --blocked-by A --blocks A.
		// B blocked_by A, A blocked_by B -> direct cycle.
		taskA := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{taskA})

		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"tick", "create", "Task B",
			"--blocked-by", "tick-aaaaaa",
			"--blocks", "tick-aaaaaa",
		}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "cycle") {
			t.Errorf("expected 'cycle' in stderr, got %q", stderr.String())
		}

		// Verify no task was created.
		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 1 {
			t.Errorf("expected 1 task (no new task created), got %d", len(tasks))
		}
	})

	t.Run("it accepts valid --blocked-by and --blocks dependencies", func(t *testing.T) {
		// A and B exist independently. Create C with --blocked-by A --blocks B.
		// C blocked_by A, B blocked_by C. No cycle.
		taskA := task.NewTask("tick-aaaaaa", "Task A")
		taskB := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{taskA, taskB})

		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"tick", "create", "Task C",
			"--blocked-by", "tick-aaaaaa",
			"--blocks", "tick-bbbbbb",
		}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromFile(t, dir)
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}

		// Verify new task has blocked_by = [tick-aaaaaa].
		newTask := tasks[2]
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaaaaa" {
			t.Errorf("new task blocked_by = %v, want [tick-aaaaaa]", newTask.BlockedBy)
		}

		// Verify tick-bbbbbb has new task in its blocked_by.
		for _, tk := range tasks {
			if tk.ID == "tick-bbbbbb" {
				if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != newTask.ID {
					t.Errorf("tick-bbbbbb blocked_by = %v, want [%s]", tk.BlockedBy, newTask.ID)
				}
				break
			}
		}
	})
}
