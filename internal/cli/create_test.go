package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// setupTickProject creates a .tick/ directory with an empty tasks.jsonl file
// and returns the temp directory path and the .tick/ path.
func setupTickProject(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.Mkdir(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick/: %v", err)
	}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return dir, tickDir
}

// setupTickProjectWithTasks creates a .tick/ project with pre-existing tasks.
func setupTickProjectWithTasks(t *testing.T, tasks []task.Task) (string, string) {
	t.Helper()
	dir, tickDir := setupTickProject(t)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	data, err := storage.MarshalJSONL(tasks)
	if err != nil {
		t.Fatalf("failed to marshal tasks: %v", err)
	}
	if err := os.WriteFile(jsonlPath, data, 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
	return dir, tickDir
}

// readPersistedTasks reads tasks from the .tick/tasks.jsonl file.
func readPersistedTasks(t *testing.T, tickDir string) []task.Task {
	t.Helper()
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	tasks, err := storage.ReadJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read persisted tasks: %v", err)
	}
	return tasks
}

// runCreate runs the tick create command with the given args and returns stdout, stderr, and exit code.
func runCreate(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	fullArgs := append([]string{"tick", "create"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestCreate(t *testing.T) {
	t.Run("it creates a task with only a title (defaults applied)", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		stdout, stderr, exitCode := runCreate(t, dir, "My first task")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if stderr != "" {
			t.Errorf("stderr should be empty, got %q", stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		tk := tasks[0]
		if tk.Title != "My first task" {
			t.Errorf("title = %q, want %q", tk.Title, "My first task")
		}
		if tk.Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tk.Status, task.StatusOpen)
		}
		if tk.Priority != 2 {
			t.Errorf("priority = %d, want 2", tk.Priority)
		}
		if tk.Description != "" {
			t.Errorf("description = %q, want empty", tk.Description)
		}
		if len(tk.BlockedBy) != 0 {
			t.Errorf("blocked_by = %v, want empty", tk.BlockedBy)
		}
		if tk.Parent != "" {
			t.Errorf("parent = %q, want empty", tk.Parent)
		}

		// Verify output contains the task ID
		if !strings.Contains(stdout, tk.ID) {
			t.Errorf("stdout should contain task ID %q, got %q", tk.ID, stdout)
		}
	})

	t.Run("it creates a task with all optional fields specified", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		existingTask := task.Task{
			ID:       "tick-aaa111",
			Title:    "Blocker task",
			Status:   task.StatusOpen,
			Priority: 2,
			Created:  now,
			Updated:  now,
		}
		parentTask := task.Task{
			ID:       "tick-bbb222",
			Title:    "Parent task",
			Status:   task.StatusOpen,
			Priority: 2,
			Created:  now,
			Updated:  now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask, parentTask})

		stdout, stderr, exitCode := runCreate(t, dir,
			"Full task",
			"--priority", "1",
			"--description", "A detailed description",
			"--blocked-by", "tick-aaa111",
			"--parent", "tick-bbb222",
		)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}
		// New task should be the last one appended
		tk := tasks[2]
		if tk.Title != "Full task" {
			t.Errorf("title = %q, want %q", tk.Title, "Full task")
		}
		if tk.Priority != 1 {
			t.Errorf("priority = %d, want 1", tk.Priority)
		}
		if tk.Description != "A detailed description" {
			t.Errorf("description = %q, want %q", tk.Description, "A detailed description")
		}
		if len(tk.BlockedBy) != 1 || tk.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("blocked_by = %v, want [tick-aaa111]", tk.BlockedBy)
		}
		if tk.Parent != "tick-bbb222" {
			t.Errorf("parent = %q, want %q", tk.Parent, "tick-bbb222")
		}

		if !strings.Contains(stdout, tk.ID) {
			t.Errorf("stdout should contain task ID %q, got %q", tk.ID, stdout)
		}
	})

	t.Run("it generates a unique ID for the created task", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		runCreate(t, dir, "Task one")
		runCreate(t, dir, "Task two")

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID == tasks[1].ID {
			t.Errorf("IDs should be unique, both are %q", tasks[0].ID)
		}
		for _, tk := range tasks {
			if !strings.HasPrefix(tk.ID, "tick-") {
				t.Errorf("ID %q does not have tick- prefix", tk.ID)
			}
			if len(tk.ID) != 11 {
				t.Errorf("ID %q should be 11 chars (tick- + 6 hex), got %d", tk.ID, len(tk.ID))
			}
		}
	})

	t.Run("it sets status to open on creation", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		_, _, exitCode := runCreate(t, dir, "Status check")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
	})

	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		_, _, exitCode := runCreate(t, dir, "Priority check")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Priority != 2 {
			t.Errorf("priority = %d, want 2", tasks[0].Priority)
		}
	})

	t.Run("it sets priority from --priority flag", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		_, _, exitCode := runCreate(t, dir, "High priority", "--priority", "0")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Priority != 0 {
			t.Errorf("priority = %d, want 0", tasks[0].Priority)
		}
	})

	t.Run("it rejects priority outside 0-4 range", func(t *testing.T) {
		dir, _ := setupTickProject(t)
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
				_, stderr, exitCode := runCreate(t, dir, "Bad priority", "--priority", tt.priority)
				if exitCode != 1 {
					t.Errorf("exit code = %d, want 1", exitCode)
				}
				if !strings.Contains(stderr, "Error:") {
					t.Errorf("stderr should contain error, got %q", stderr)
				}
			})
		}
	})

	t.Run("it sets description from --description flag", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		_, _, exitCode := runCreate(t, dir, "With desc", "--description", "My detailed description")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Description != "My detailed description" {
			t.Errorf("description = %q, want %q", tasks[0].Description, "My detailed description")
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (single ID)", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		existingTask := task.Task{
			ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, _, exitCode := runCreate(t, dir, "Blocked task", "--blocked-by", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		tasks := readPersistedTasks(t, tickDir)
		newTask := tasks[len(tasks)-1]
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("blocked_by = %v, want [tick-aaa111]", newTask.BlockedBy)
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocker 2", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, tickDir := setupTickProjectWithTasks(t, tasks)

		_, _, exitCode := runCreate(t, dir, "Multi blocked", "--blocked-by", "tick-aaa111,tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		persisted := readPersistedTasks(t, tickDir)
		newTask := persisted[len(persisted)-1]
		if len(newTask.BlockedBy) != 2 {
			t.Fatalf("blocked_by length = %d, want 2", len(newTask.BlockedBy))
		}
		if newTask.BlockedBy[0] != "tick-aaa111" || newTask.BlockedBy[1] != "tick-bbb222" {
			t.Errorf("blocked_by = %v, want [tick-aaa111 tick-bbb222]", newTask.BlockedBy)
		}
	})

	t.Run("it updates target tasks' blocked_by when --blocks is used", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		existingTask := task.Task{
			ID: "tick-aaa111", Title: "Target task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, _, exitCode := runCreate(t, dir, "Blocking task", "--blocks", "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		persisted := readPersistedTasks(t, tickDir)
		// Find the target task
		var target task.Task
		var newTask task.Task
		for _, tk := range persisted {
			if tk.ID == "tick-aaa111" {
				target = tk
			} else {
				newTask = tk
			}
		}
		// The target's blocked_by should now contain the new task's ID
		if len(target.BlockedBy) != 1 || target.BlockedBy[0] != newTask.ID {
			t.Errorf("target blocked_by = %v, want [%s]", target.BlockedBy, newTask.ID)
		}
		// Target's updated timestamp should be refreshed
		if !target.Updated.After(now.Add(-time.Second)) {
			t.Error("target's updated timestamp should be refreshed")
		}
	})

	t.Run("it sets parent from --parent flag", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		parentTask := task.Task{
			ID: "tick-ppp111", Title: "Parent", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{parentTask})

		_, _, exitCode := runCreate(t, dir, "Child task", "--parent", "tick-ppp111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		persisted := readPersistedTasks(t, tickDir)
		newTask := persisted[len(persisted)-1]
		if newTask.Parent != "tick-ppp111" {
			t.Errorf("parent = %q, want %q", newTask.Parent, "tick-ppp111")
		}
	})

	t.Run("it errors when title is missing", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runCreate(t, dir)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "title is required") {
			t.Errorf("stderr = %q, want to contain 'title is required'", stderr)
		}
	})

	t.Run("it errors when title is empty string", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runCreate(t, dir, "")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should contain error, got %q", stderr)
		}
	})

	t.Run("it errors when title is whitespace only", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runCreate(t, dir, "   ")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should contain error, got %q", stderr)
		}
	})

	t.Run("it errors when --blocked-by references non-existent task", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runCreate(t, dir, "Task", "--blocked-by", "tick-nonexist")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should contain error, got %q", stderr)
		}
	})

	t.Run("it errors when --blocks references non-existent task", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runCreate(t, dir, "Task", "--blocks", "tick-nonexist")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should contain error, got %q", stderr)
		}
	})

	t.Run("it errors when --parent references non-existent task", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runCreate(t, dir, "Task", "--parent", "tick-nonexist")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Error:") {
			t.Errorf("stderr should contain error, got %q", stderr)
		}
	})

	t.Run("it persists the task to tasks.jsonl via atomic write", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		_, _, exitCode := runCreate(t, dir, "Persisted task")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		// Read the raw file to verify it was written
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("tasks.jsonl should not be empty after create")
		}

		// Verify it's valid JSONL
		var raw map[string]interface{}
		lines := strings.TrimSpace(string(data))
		if err := json.Unmarshal([]byte(lines), &raw); err != nil {
			t.Errorf("tasks.jsonl content is not valid JSON: %v", err)
		}
		if raw["title"] != "Persisted task" {
			t.Errorf("persisted title = %v, want %q", raw["title"], "Persisted task")
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		stdout, _, exitCode := runCreate(t, dir, "Output check")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		tk := tasks[0]

		// Output should contain key fields
		if !strings.Contains(stdout, tk.ID) {
			t.Errorf("stdout should contain ID %q, got %q", tk.ID, stdout)
		}
		if !strings.Contains(stdout, "Output check") {
			t.Errorf("stdout should contain title, got %q", stdout)
		}
		if !strings.Contains(stdout, "open") {
			t.Errorf("stdout should contain status 'open', got %q", stdout)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		stdout, _, exitCode := runCreate(t, dir, "--quiet", "Quiet task")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		tk := tasks[0]

		expected := tk.ID + "\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		existingTask := task.Task{
			ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, _, exitCode := runCreate(t, dir, "Normalized", "--blocked-by", "TICK-AAA111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		persisted := readPersistedTasks(t, tickDir)
		newTask := persisted[len(persisted)-1]
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("blocked_by = %v, want [tick-aaa111]", newTask.BlockedBy)
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		_, _, exitCode := runCreate(t, dir, "  Trimmed title  ")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "Trimmed title" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "Trimmed title")
		}
	})
}
