package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupInitializedTickDir creates a .tick/ directory with an empty tasks.jsonl.
func setupInitializedTickDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick dir: %v", err)
	}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return dir
}

// setupTickDirWithContent creates a .tick/ directory with the given JSONL content.
func setupTickDirWithContent(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick dir: %v", err)
	}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return dir
}

// readTasksJSONL reads and parses the tasks.jsonl file, returning a slice of maps.
func readTasksJSONL(t *testing.T, dir string) []map[string]interface{} {
	t.Helper()
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}
	var tasks []map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("failed to parse JSONL line: %v", err)
		}
		tasks = append(tasks, m)
	}
	return tasks
}

func TestCreateCommand(t *testing.T) {
	t.Run("it creates a task with only a title (defaults applied)", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "My first task"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		tk := tasks[0]
		if tk["title"] != "My first task" {
			t.Errorf("title = %q, want %q", tk["title"], "My first task")
		}
		if tk["status"] != "open" {
			t.Errorf("status = %q, want %q", tk["status"], "open")
		}
		if int(tk["priority"].(float64)) != 2 {
			t.Errorf("priority = %v, want 2", tk["priority"])
		}
	})

	t.Run("it creates a task with all optional fields specified", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Blocker task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Parent task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{
			"tick", "create", "Full task",
			"--priority", "1",
			"--description", "Some detailed description",
			"--blocked-by", "tick-aaa111",
			"--parent", "tick-bbb222",
		})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		// Original 2 + new 1
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}
		newTask := tasks[2]
		if newTask["title"] != "Full task" {
			t.Errorf("title = %q, want %q", newTask["title"], "Full task")
		}
		if int(newTask["priority"].(float64)) != 1 {
			t.Errorf("priority = %v, want 1", newTask["priority"])
		}
		if newTask["description"] != "Some detailed description" {
			t.Errorf("description = %q, want %q", newTask["description"], "Some detailed description")
		}
		blockedBy, ok := newTask["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", newTask["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != "tick-aaa111" {
			t.Errorf("blocked_by = %v, want [tick-aaa111]", blockedBy)
		}
		if newTask["parent"] != "tick-bbb222" {
			t.Errorf("parent = %q, want %q", newTask["parent"], "tick-bbb222")
		}
	})

	t.Run("it generates a unique ID for the created task", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "ID test"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		id, ok := tasks[0]["id"].(string)
		if !ok || id == "" {
			t.Fatal("task has no id")
		}
		if !strings.HasPrefix(id, "tick-") {
			t.Errorf("id = %q, want prefix 'tick-'", id)
		}
		// tick- (5 chars) + 6 hex chars = 11 total
		if len(id) != 11 {
			t.Errorf("id length = %d, want 11", len(id))
		}
	})

	t.Run("it sets status to open on creation", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Status test"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if tasks[0]["status"] != "open" {
			t.Errorf("status = %q, want %q", tasks[0]["status"], "open")
		}
	})

	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Priority default test"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if int(tasks[0]["priority"].(float64)) != 2 {
			t.Errorf("priority = %v, want 2", tasks[0]["priority"])
		}
	})

	t.Run("it sets priority from --priority flag", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Priority flag test", "--priority", "0"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if int(tasks[0]["priority"].(float64)) != 0 {
			t.Errorf("priority = %v, want 0", tasks[0]["priority"])
		}
	})

	t.Run("it rejects priority outside 0-4 range", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "create", "Bad priority", "--priority", "5"})
		if err == nil {
			t.Fatal("expected error for priority outside range, got nil")
		}
		if !strings.Contains(err.Error(), "priority") {
			t.Errorf("error = %q, want it to contain 'priority'", err.Error())
		}

		// Also test negative
		app2 := NewApp()
		app2.workDir = dir
		err = app2.Run([]string{"tick", "create", "Bad priority neg", "--priority", "-1"})
		if err == nil {
			t.Fatal("expected error for negative priority, got nil")
		}
	})

	t.Run("it sets description from --description flag", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Desc test", "--description", "A detailed description"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if tasks[0]["description"] != "A detailed description" {
			t.Errorf("description = %q, want %q", tasks[0]["description"], "A detailed description")
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (single ID)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Blocker","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Blocked task", "--blocked-by", "tick-aaa111"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		newTask := tasks[1]
		blockedBy, ok := newTask["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", newTask["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != "tick-aaa111" {
			t.Errorf("blocked_by = %v, want [tick-aaa111]", blockedBy)
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Blocker 1","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Blocker 2","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Multi blocked", "--blocked-by", "tick-aaa111,tick-bbb222"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		newTask := tasks[2]
		blockedBy, ok := newTask["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", newTask["blocked_by"])
		}
		if len(blockedBy) != 2 {
			t.Fatalf("blocked_by has %d items, want 2", len(blockedBy))
		}
		if blockedBy[0] != "tick-aaa111" || blockedBy[1] != "tick-bbb222" {
			t.Errorf("blocked_by = %v, want [tick-aaa111 tick-bbb222]", blockedBy)
		}
	})

	t.Run("it updates target tasks' blocked_by when --blocks is used", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Target task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Blocking task", "--blocks", "tick-aaa111"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		// The target task (tick-aaa111) should now have the new task in blocked_by
		targetTask := tasks[0]
		newTask := tasks[1]
		newID := newTask["id"].(string)

		blockedBy, ok := targetTask["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("target blocked_by is not an array: %T", targetTask["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != newID {
			t.Errorf("target blocked_by = %v, want [%s]", blockedBy, newID)
		}

		// Target task's updated timestamp should have changed
		if targetTask["updated"] == "2026-01-19T10:00:00Z" {
			t.Error("target task's updated timestamp should have changed")
		}
	})

	t.Run("it sets parent from --parent flag", func(t *testing.T) {
		content := `{"id":"tick-parent1","title":"Parent","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Child task", "--parent", "tick-parent1"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		newTask := tasks[1]
		if newTask["parent"] != "tick-parent1" {
			t.Errorf("parent = %q, want %q", newTask["parent"], "tick-parent1")
		}
	})

	t.Run("it errors when title is missing", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "create"})
		if err == nil {
			t.Fatal("expected error when title is missing, got nil")
		}
		if !strings.Contains(err.Error(), "Title is required") {
			t.Errorf("error = %q, want it to contain 'Title is required'", err.Error())
		}
	})

	t.Run("it errors when title is empty string", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "create", ""})
		if err == nil {
			t.Fatal("expected error for empty title, got nil")
		}
		if !strings.Contains(err.Error(), "Title cannot be empty") {
			t.Errorf("error = %q, want it to contain 'Title cannot be empty'", err.Error())
		}
	})

	t.Run("it errors when title is whitespace only", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "create", "   "})
		if err == nil {
			t.Fatal("expected error for whitespace-only title, got nil")
		}
		if !strings.Contains(err.Error(), "Title cannot be empty") {
			t.Errorf("error = %q, want it to contain 'Title cannot be empty'", err.Error())
		}
	})

	t.Run("it errors when --blocked-by references non-existent task", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "create", "Test", "--blocked-by", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error for non-existent blocked-by ID, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}

		// Verify no task was created
		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks (no partial mutation), got %d", len(tasks))
		}
	})

	t.Run("it errors when --blocks references non-existent task", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "create", "Test", "--blocks", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error for non-existent blocks ID, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}

		// Verify no task was created
		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks (no partial mutation), got %d", len(tasks))
		}
	})

	t.Run("it errors when --parent references non-existent task", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "create", "Test", "--parent", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error for non-existent parent ID, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}

		// Verify no task was created
		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks (no partial mutation), got %d", len(tasks))
		}
	})

	t.Run("it persists the task to tasks.jsonl via atomic write", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Persisted task"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		// Read back and verify
		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0]["title"] != "Persisted task" {
			t.Errorf("title = %q, want %q", tasks[0]["title"], "Persisted task")
		}
		// Verify timestamps exist
		if _, ok := tasks[0]["created"]; !ok {
			t.Error("created timestamp missing")
		}
		if _, ok := tasks[0]["updated"]; !ok {
			t.Error("updated timestamp missing")
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "Output test"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		output := stdout.String()
		// Should contain the task ID
		if !strings.Contains(output, "tick-") {
			t.Errorf("output = %q, want it to contain a task ID (tick-...)", output)
		}
		// Should contain the title
		if !strings.Contains(output, "Output test") {
			t.Errorf("output = %q, want it to contain 'Output test'", output)
		}
		// Should contain status
		if !strings.Contains(output, "open") {
			t.Errorf("output = %q, want it to contain 'open'", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "create", "Quiet test"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		// Output should be only the task ID
		if !strings.HasPrefix(output, "tick-") {
			t.Errorf("output = %q, want it to start with 'tick-'", output)
		}
		// Should be exactly 11 chars (tick- + 6 hex)
		if len(output) != 11 {
			t.Errorf("output length = %d, want 11 (just the task ID)", len(output))
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Existing","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		// Use uppercase ID in --blocked-by
		err := app.Run([]string{"tick", "create", "Normalized", "--blocked-by", "TICK-AAA111"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		newTask := tasks[1]
		blockedBy, ok := newTask["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", newTask["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != "tick-aaa111" {
			t.Errorf("blocked_by = %v, want [tick-aaa111] (lowercase)", blockedBy)
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "create", "  Trimmed title  "})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		tasks := readTasksJSONL(t, dir)
		if tasks[0]["title"] != "Trimmed title" {
			t.Errorf("title = %q, want %q", tasks[0]["title"], "Trimmed title")
		}
	})
}
