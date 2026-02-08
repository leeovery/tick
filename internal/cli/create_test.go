package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// setupInitializedDir creates a temp directory with .tick/ and empty tasks.jsonl.
func setupInitializedDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick dir: %v", err)
	}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return dir
}

// setupInitializedDirWithTasks creates a temp directory with .tick/ and pre-populated tasks.
func setupInitializedDirWithTasks(t *testing.T, tasks []task.Task) string {
	t.Helper()
	dir := setupInitializedDir(t)
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	if err := task.WriteJSONL(jsonlPath, tasks); err != nil {
		t.Fatalf("failed to write tasks: %v", err)
	}
	return dir
}

// readTasksFromDir reads tasks from .tick/tasks.jsonl in the given directory.
func readTasksFromDir(t *testing.T, dir string) []task.Task {
	t.Helper()
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	tasks, err := task.ReadJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}
	return tasks
}

func TestCreate_WithOnlyTitle(t *testing.T) {
	t.Run("it creates a task with only a title (defaults applied)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "My first task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		tk := tasks[0]
		if tk.Title != "My first task" {
			t.Errorf("expected title %q, got %q", "My first task", tk.Title)
		}
		if tk.Status != task.StatusOpen {
			t.Errorf("expected status %q, got %q", task.StatusOpen, tk.Status)
		}
		if tk.Priority != 2 {
			t.Errorf("expected default priority 2, got %d", tk.Priority)
		}
		if tk.Description != "" {
			t.Errorf("expected empty description, got %q", tk.Description)
		}
		if len(tk.BlockedBy) != 0 {
			t.Errorf("expected empty blocked_by, got %v", tk.BlockedBy)
		}
		if tk.Parent != "" {
			t.Errorf("expected empty parent, got %q", tk.Parent)
		}
		if tk.Closed != nil {
			t.Errorf("expected nil closed, got %v", tk.Closed)
		}
	})
}

func TestCreate_WithAllOptionalFields(t *testing.T) {
	t.Run("it creates a task with all optional fields specified", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existingTasks := []task.Task{
			{
				ID:       "tick-aaa111",
				Title:    "Blocker task",
				Status:   task.StatusOpen,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
			{
				ID:       "tick-bbb222",
				Title:    "Parent task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			{
				ID:       "tick-ccc333",
				Title:    "Blocks target",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}
		dir := setupInitializedDirWithTasks(t, existingTasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Full task",
			"--priority", "1",
			"--description", "A detailed description",
			"--blocked-by", "tick-aaa111",
			"--blocks", "tick-ccc333",
			"--parent", "tick-bbb222",
		})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// 3 original + 1 new
		if len(tasks) != 4 {
			t.Fatalf("expected 4 tasks, got %d", len(tasks))
		}

		// Find the new task (not one of the originals)
		var newTask *task.Task
		for i := range tasks {
			if tasks[i].ID != "tick-aaa111" && tasks[i].ID != "tick-bbb222" && tasks[i].ID != "tick-ccc333" {
				newTask = &tasks[i]
				break
			}
		}
		if newTask == nil {
			t.Fatal("could not find newly created task")
		}

		if newTask.Priority != 1 {
			t.Errorf("expected priority 1, got %d", newTask.Priority)
		}
		if newTask.Description != "A detailed description" {
			t.Errorf("expected description %q, got %q", "A detailed description", newTask.Description)
		}
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("expected blocked_by [tick-aaa111], got %v", newTask.BlockedBy)
		}
		if newTask.Parent != "tick-bbb222" {
			t.Errorf("expected parent %q, got %q", "tick-bbb222", newTask.Parent)
		}
	})
}

func TestCreate_GeneratesUniqueID(t *testing.T) {
	t.Run("it generates a unique ID for the created task", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Task one"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
		if !pattern.MatchString(tasks[0].ID) {
			t.Errorf("ID %q does not match tick-{6 hex} pattern", tasks[0].ID)
		}

		// Create a second task and verify IDs are different
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "create", "Task two"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks = readTasksFromDir(t, dir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID == tasks[1].ID {
			t.Errorf("expected unique IDs, both are %q", tasks[0].ID)
		}
	})
}

func TestCreate_SetsStatusOpen(t *testing.T) {
	t.Run("it sets status to open on creation", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("expected status %q, got %q", task.StatusOpen, tasks[0].Status)
		}
	})
}

func TestCreate_DefaultPriority(t *testing.T) {
	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Priority != 2 {
			t.Errorf("expected default priority 2, got %d", tasks[0].Priority)
		}
	})
}

func TestCreate_PriorityFlag(t *testing.T) {
	t.Run("it sets priority from --priority flag", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--priority", "0"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Priority != 0 {
			t.Errorf("expected priority 0, got %d", tasks[0].Priority)
		}
	})
}

func TestCreate_RejectsPriorityOutsideRange(t *testing.T) {
	t.Run("it rejects priority outside 0-4 range", func(t *testing.T) {
		tests := []struct {
			name     string
			priority string
		}{
			{"negative", "-1"},
			{"too high", "5"},
			{"way too high", "100"},
			{"not a number", "abc"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupInitializedDir(t)
				var stdout, stderr bytes.Buffer

				app := &App{
					Stdout: &stdout,
					Stderr: &stderr,
					Dir:    dir,
				}

				code := app.Run([]string{"tick", "create", "Test task", "--priority", tt.priority})
				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if stderr.String() == "" {
					t.Error("expected error on stderr")
				}
			})
		}
	})
}

func TestCreate_DescriptionFlag(t *testing.T) {
	t.Run("it sets description from --description flag", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--description", "This is a description"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Description != "This is a description" {
			t.Errorf("expected description %q, got %q", "This is a description", tasks[0].Description)
		}
	})
}

func TestCreate_BlockedBySingleID(t *testing.T) {
	t.Run("it sets blocked_by from --blocked-by flag (single ID)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Existing", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Blocked task", "--blocked-by", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// Find the new task
		var newTask *task.Task
		for i := range tasks {
			if tasks[i].ID != "tick-aaa111" {
				newTask = &tasks[i]
				break
			}
		}
		if newTask == nil {
			t.Fatal("could not find new task")
		}
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("expected blocked_by [tick-aaa111], got %v", newTask.BlockedBy)
		}
	})
}

func TestCreate_BlockedByMultipleIDs(t *testing.T) {
	t.Run("it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "First", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Multi-blocked", "--blocked-by", "tick-aaa111,tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var newTask *task.Task
		for i := range tasks {
			if tasks[i].ID != "tick-aaa111" && tasks[i].ID != "tick-bbb222" {
				newTask = &tasks[i]
				break
			}
		}
		if newTask == nil {
			t.Fatal("could not find new task")
		}
		if len(newTask.BlockedBy) != 2 {
			t.Fatalf("expected 2 blocked_by entries, got %d", len(newTask.BlockedBy))
		}
		// Verify both are present
		blockedBySet := map[string]bool{}
		for _, id := range newTask.BlockedBy {
			blockedBySet[id] = true
		}
		if !blockedBySet["tick-aaa111"] || !blockedBySet["tick-bbb222"] {
			t.Errorf("expected blocked_by to contain tick-aaa111 and tick-bbb222, got %v", newTask.BlockedBy)
		}
	})
}

func TestCreate_BlocksUpdatesTargetTasks(t *testing.T) {
	t.Run("it updates target tasks' blocked_by when --blocks is used", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Blocking task", "--blocks", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// Find the target task and verify it was updated
		var targetTask *task.Task
		var newTask *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				targetTask = &tasks[i]
			} else {
				newTask = &tasks[i]
			}
		}
		if targetTask == nil {
			t.Fatal("could not find target task")
		}
		if newTask == nil {
			t.Fatal("could not find new task")
		}

		// Target task should now have the new task in its blocked_by
		if len(targetTask.BlockedBy) != 1 || targetTask.BlockedBy[0] != newTask.ID {
			t.Errorf("expected target blocked_by [%s], got %v", newTask.ID, targetTask.BlockedBy)
		}

		// Target task's updated timestamp should have changed
		if !targetTask.Updated.After(now) {
			t.Errorf("expected target updated timestamp to be refreshed, got %v (original: %v)", targetTask.Updated, now)
		}
	})
}

func TestCreate_ParentFlag(t *testing.T) {
	t.Run("it sets parent from --parent flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Child task", "--parent", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var newTask *task.Task
		for i := range tasks {
			if tasks[i].ID != "tick-aaa111" {
				newTask = &tasks[i]
				break
			}
		}
		if newTask == nil {
			t.Fatal("could not find new task")
		}
		if newTask.Parent != "tick-aaa111" {
			t.Errorf("expected parent %q, got %q", "tick-aaa111", newTask.Parent)
		}
	})
}

func TestCreate_ErrorMissingTitle(t *testing.T) {
	t.Run("it errors when title is missing", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Title is required") {
			t.Errorf("expected 'Title is required' in error, got %q", errMsg)
		}
	})
}

func TestCreate_ErrorEmptyTitle(t *testing.T) {
	t.Run("it errors when title is empty string", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", ""})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
	})
}

func TestCreate_ErrorWhitespaceOnlyTitle(t *testing.T) {
	t.Run("it errors when title is whitespace only", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "   "})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
	})
}

func TestCreate_ErrorBlockedByNonExistent(t *testing.T) {
	t.Run("it errors when --blocked-by references non-existent task", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--blocked-by", "tick-nonexist"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}

		// No tasks should have been written
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after error, got %d", len(tasks))
		}
	})
}

func TestCreate_ErrorBlocksNonExistent(t *testing.T) {
	t.Run("it errors when --blocks references non-existent task", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--blocks", "tick-nonexist"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}

		// No tasks should have been written
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks after error, got %d", len(tasks))
		}
	})
}

func TestCreate_ErrorParentNonExistent(t *testing.T) {
	t.Run("it errors when --parent references non-existent task", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--parent", "tick-nonexist"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
	})
}

func TestCreate_PersistsViaAtomicWrite(t *testing.T) {
	t.Run("it persists the task to tasks.jsonl via atomic write", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Persisted task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Read raw file and verify it's valid JSONL
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		// Verify it's valid JSON on a single line
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line in JSONL, got %d", len(lines))
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
			t.Fatalf("failed to parse JSONL line as JSON: %v", err)
		}
		if parsed["title"] != "Persisted task" {
			t.Errorf("expected title 'Persisted task', got %v", parsed["title"])
		}

		// Verify cache.db was updated (SQLite cache updated as part of write flow)
		cacheDB := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cacheDB); os.IsNotExist(err) {
			t.Error("expected cache.db to exist after create")
		}
	})
}

func TestCreate_OutputsTaskDetails(t *testing.T) {
	t.Run("it outputs full task details on success", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "Output test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if output == "" {
			t.Error("expected output on stdout, got nothing")
		}

		// Output should include the task ID
		idPattern := regexp.MustCompile(`tick-[0-9a-f]{6}`)
		if !idPattern.MatchString(output) {
			t.Errorf("expected output to contain task ID, got %q", output)
		}

		// Output should include the title
		if !strings.Contains(output, "Output test task") {
			t.Errorf("expected output to contain title, got %q", output)
		}

		// Output should include status
		if !strings.Contains(output, "open") {
			t.Errorf("expected output to contain status 'open', got %q", output)
		}
	})
}

func TestCreate_QuietFlag(t *testing.T) {
	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "create", "Quiet test"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		idPattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
		if !idPattern.MatchString(output) {
			t.Errorf("expected only task ID in output with --quiet, got %q", output)
		}
	})
}

func TestCreate_NormalizesInputIDs(t *testing.T) {
	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Existing", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		// Use uppercase IDs in flags
		code := app.Run([]string{"tick", "create", "Normalized task",
			"--blocked-by", "TICK-AAA111",
		})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var newTask *task.Task
		for i := range tasks {
			if tasks[i].ID != "tick-aaa111" {
				newTask = &tasks[i]
				break
			}
		}
		if newTask == nil {
			t.Fatal("could not find new task")
		}
		// blocked_by should be stored as lowercase
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("expected blocked_by [tick-aaa111] (lowercase), got %v", newTask.BlockedBy)
		}
	})
}

func TestCreate_TrimsWhitespaceFromTitle(t *testing.T) {
	t.Run("it trims whitespace from title", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "create", "  Trimmed title  "})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Title != "Trimmed title" {
			t.Errorf("expected title %q, got %q", "Trimmed title", tasks[0].Title)
		}
	})
}

func TestCreate_TimestampsSetToUTC(t *testing.T) {
	t.Run("it sets timestamps to current UTC ISO 8601", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		before := time.Now().UTC().Truncate(time.Second)
		code := app.Run([]string{"tick", "create", "Timestamp test"})
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		tk := tasks[0]

		if tk.Created.Before(before) || tk.Created.After(after) {
			t.Errorf("created timestamp %v not within expected range [%v, %v]", tk.Created, before, after)
		}
		if tk.Updated.Before(before) || tk.Updated.After(after) {
			t.Errorf("updated timestamp %v not within expected range [%v, %v]", tk.Updated, before, after)
		}
		if !tk.Created.Equal(tk.Updated) {
			t.Errorf("expected created and updated to be equal")
		}
		if tk.Created.Location() != time.UTC {
			t.Errorf("expected UTC timezone, got %v", tk.Created.Location())
		}
	})
}

func TestCreate_BlocksRejectsCycle(t *testing.T) {
	t.Run("it rejects --blocks that would create a dependency cycle", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		// A is blocked by B (A.blocked_by = [B])
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		// Create C with --blocked-by A --blocks B
		// This means: C.blocked_by=[A], B.blocked_by=[C]
		// Cycle: B→C→A→B
		code := app.Run([]string{"tick", "create", "Task C", "--blocked-by", "tick-aaa111", "--blocks", "tick-bbb222"})
		if code != 1 {
			t.Errorf("expected exit code 1 (cycle), got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "cycle") {
			t.Errorf("expected cycle error, got %q", errMsg)
		}

		// No new task should have been created
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks (unchanged), got %d", len(tasks))
		}
	})
}

func TestCreate_BlocksRejectsChildBlockedByParent(t *testing.T) {
	t.Run("it rejects --blocked-by parent for a child task", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-ppp111", Title: "Parent P", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		// Create child C with --parent P --blocked-by P
		// C.parent=P, C.blocked_by=[P] — child blocked by its own parent is invalid
		code := app.Run([]string{"tick", "create", "Child C", "--parent", "tick-ppp111", "--blocked-by", "tick-ppp111"})
		if code != 1 {
			t.Errorf("expected exit code 1 (child-blocked-by-parent), got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "cannot") {
			t.Errorf("expected rejection error, got %q", errMsg)
		}

		// No new task should have been created
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Errorf("expected 1 task (unchanged), got %d", len(tasks))
		}
	})
}
