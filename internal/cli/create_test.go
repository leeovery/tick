package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCommand(t *testing.T) {
	t.Run("it creates a task with only a title (defaults applied)", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "My first task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Verify task was created with defaults
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		task := tasks[0]
		if task.Title != "My first task" {
			t.Errorf("title = %q, want %q", task.Title, "My first task")
		}
		if task.Status != "open" {
			t.Errorf("status = %q, want %q", task.Status, "open")
		}
		if task.Priority != 2 {
			t.Errorf("priority = %d, want %d", task.Priority, 2)
		}
	})

	t.Run("it creates a task with all optional fields specified", func(t *testing.T) {
		dir := setupTickDir(t)

		// First create a task to reference as parent and blocked-by
		setupTask(t, dir, "tick-parent", "Parent task")
		setupTask(t, dir, "tick-blocker", "Blocker task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{
			"tick", "create", "Full task",
			"--priority", "1",
			"--description", "Detailed description",
			"--blocked-by", "tick-blocker",
			"--parent", "tick-parent",
		})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// We should have 3 tasks now (parent, blocker, new one)
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}

		// Find the new task (not parent or blocker)
		var newTask *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			Description string   `json:"description,omitempty"`
			BlockedBy   []string `json:"blocked_by,omitempty"`
			Parent      string   `json:"parent,omitempty"`
			Created     string   `json:"created"`
			Updated     string   `json:"updated"`
			Closed      string   `json:"closed,omitempty"`
		}
		for i := range tasks {
			if tasks[i].ID != "tick-parent" && tasks[i].ID != "tick-blocker" {
				newTask = &tasks[i]
				break
			}
		}
		if newTask == nil {
			t.Fatal("new task not found")
		}

		if newTask.Title != "Full task" {
			t.Errorf("title = %q, want %q", newTask.Title, "Full task")
		}
		if newTask.Priority != 1 {
			t.Errorf("priority = %d, want %d", newTask.Priority, 1)
		}
		if newTask.Description != "Detailed description" {
			t.Errorf("description = %q, want %q", newTask.Description, "Detailed description")
		}
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-blocker" {
			t.Errorf("blocked_by = %v, want [tick-blocker]", newTask.BlockedBy)
		}
		if newTask.Parent != "tick-parent" {
			t.Errorf("parent = %q, want %q", newTask.Parent, "tick-parent")
		}
	})

	t.Run("it generates a unique ID for the created task", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		// Verify ID follows pattern tick-{6 hex}
		id := tasks[0].ID
		if !strings.HasPrefix(id, "tick-") {
			t.Errorf("ID should start with 'tick-', got %q", id)
		}
		if len(id) != 11 { // "tick-" + 6 hex chars
			t.Errorf("ID should be 11 chars, got %d: %q", len(id), id)
		}
	})

	t.Run("it sets status to open on creation", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Status != "open" {
			t.Errorf("status = %q, want %q", tasks[0].Status, "open")
		}
	})

	t.Run("it sets default priority to 2 when not specified", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Priority != 2 {
			t.Errorf("priority = %d, want %d", tasks[0].Priority, 2)
		}
	})

	t.Run("it sets priority from --priority flag", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--priority", "0"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Priority != 0 {
			t.Errorf("priority = %d, want %d", tasks[0].Priority, 0)
		}
	})

	t.Run("it rejects priority outside 0-4 range", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--priority", "5"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		if !strings.Contains(stderr.String(), "priority") {
			t.Errorf("error should mention priority, got %q", stderr.String())
		}

		// Verify no task was created
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("it sets description from --description flag", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--description", "A detailed description"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Description != "A detailed description" {
			t.Errorf("description = %q, want %q", tasks[0].Description, "A detailed description")
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (single ID)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-blocker", "Blocker task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--blocked-by", "tick-blocker"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// Find the new task
		var newTask *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			Description string   `json:"description,omitempty"`
			BlockedBy   []string `json:"blocked_by,omitempty"`
			Parent      string   `json:"parent,omitempty"`
			Created     string   `json:"created"`
			Updated     string   `json:"updated"`
			Closed      string   `json:"closed,omitempty"`
		}
		for i := range tasks {
			if tasks[i].ID != "tick-blocker" {
				newTask = &tasks[i]
				break
			}
		}

		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-blocker" {
			t.Errorf("blocked_by = %v, want [tick-blocker]", newTask.BlockedBy)
		}
	})

	t.Run("it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-block1", "Blocker 1")
		setupTask(t, dir, "tick-block2", "Blocker 2")
		setupTask(t, dir, "tick-block3", "Blocker 3")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--blocked-by", "tick-block1,tick-block2,tick-block3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// Find the new task
		var newTask *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			Description string   `json:"description,omitempty"`
			BlockedBy   []string `json:"blocked_by,omitempty"`
			Parent      string   `json:"parent,omitempty"`
			Created     string   `json:"created"`
			Updated     string   `json:"updated"`
			Closed      string   `json:"closed,omitempty"`
		}
		for i := range tasks {
			if tasks[i].Title == "Test task" {
				newTask = &tasks[i]
				break
			}
		}

		if len(newTask.BlockedBy) != 3 {
			t.Errorf("blocked_by length = %d, want 3", len(newTask.BlockedBy))
		}
	})

	t.Run("it updates target tasks' blocked_by when --blocks is used", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-target", "Target task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Blocking task", "--blocks", "tick-target"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// Find the target task and check its blocked_by
		var targetTask *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			Description string   `json:"description,omitempty"`
			BlockedBy   []string `json:"blocked_by,omitempty"`
			Parent      string   `json:"parent,omitempty"`
			Created     string   `json:"created"`
			Updated     string   `json:"updated"`
			Closed      string   `json:"closed,omitempty"`
		}
		var newTaskID string
		for i := range tasks {
			if tasks[i].ID == "tick-target" {
				targetTask = &tasks[i]
			} else {
				newTaskID = tasks[i].ID
			}
		}

		if len(targetTask.BlockedBy) != 1 || targetTask.BlockedBy[0] != newTaskID {
			t.Errorf("target's blocked_by = %v, want [%s]", targetTask.BlockedBy, newTaskID)
		}

		// Verify target's updated timestamp was refreshed
		if targetTask.Updated == "2026-01-19T10:00:00Z" {
			t.Error("target's updated timestamp should have been refreshed")
		}
	})

	t.Run("it sets parent from --parent flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-parent", "Parent task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Child task", "--parent", "tick-parent"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// Find the new task
		var newTask *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			Description string   `json:"description,omitempty"`
			BlockedBy   []string `json:"blocked_by,omitempty"`
			Parent      string   `json:"parent,omitempty"`
			Created     string   `json:"created"`
			Updated     string   `json:"updated"`
			Closed      string   `json:"closed,omitempty"`
		}
		for i := range tasks {
			if tasks[i].ID != "tick-parent" {
				newTask = &tasks[i]
				break
			}
		}

		if newTask.Parent != "tick-parent" {
			t.Errorf("parent = %q, want %q", newTask.Parent, "tick-parent")
		}
	})

	t.Run("it errors when title is missing", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Title is required") {
			t.Errorf("error should mention title is required, got %q", errOutput)
		}
	})

	t.Run("it errors when title is empty string", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", ""})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Title") {
			t.Errorf("error should mention title, got %q", errOutput)
		}
	})

	t.Run("it errors when title is whitespace only", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "   "})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Title cannot be empty") {
			t.Errorf("error should mention 'Title cannot be empty', got %q", errOutput)
		}
	})

	t.Run("it errors when --blocked-by references non-existent task", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--blocked-by", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "not found") {
			t.Errorf("error should mention 'not found', got %q", errOutput)
		}

		// Verify no task was created
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("it errors when --blocks references non-existent task", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--blocks", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "not found") {
			t.Errorf("error should mention 'not found', got %q", errOutput)
		}

		// Verify no task was created
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("it errors when --parent references non-existent task", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Test task", "--parent", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "not found") {
			t.Errorf("error should mention 'not found', got %q", errOutput)
		}

		// Verify no task was created
		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("it persists the task to tasks.jsonl via atomic write", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Persisted task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Read raw file to verify persistence
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		content, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		if !strings.Contains(string(content), "Persisted task") {
			t.Errorf("tasks.jsonl should contain 'Persisted task', got %q", string(content))
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Output test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID:") {
			t.Errorf("output should contain ID, got %q", output)
		}
		if !strings.Contains(output, "Title:") {
			t.Errorf("output should contain Title, got %q", output)
		}
		if !strings.Contains(output, "Output test task") {
			t.Errorf("output should contain task title, got %q", output)
		}
		if !strings.Contains(output, "Status:") {
			t.Errorf("output should contain Status, got %q", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "create", "Quiet test task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		// Should only be the ID
		if !strings.HasPrefix(output, "tick-") {
			t.Errorf("output should be task ID only, got %q", output)
		}
		// Should not contain other details
		if strings.Contains(output, "Title:") || strings.Contains(output, "Status:") {
			t.Errorf("quiet output should not contain details, got %q", output)
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-blocker", "Blocker task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Use uppercase ID
		code := app.Run([]string{"tick", "create", "Test task", "--blocked-by", "TICK-BLOCKER"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		// Find the new task
		var newTask *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			Description string   `json:"description,omitempty"`
			BlockedBy   []string `json:"blocked_by,omitempty"`
			Parent      string   `json:"parent,omitempty"`
			Created     string   `json:"created"`
			Updated     string   `json:"updated"`
			Closed      string   `json:"closed,omitempty"`
		}
		for i := range tasks {
			if tasks[i].ID != "tick-blocker" {
				newTask = &tasks[i]
				break
			}
		}

		// blocked_by should be normalized to lowercase
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != "tick-blocker" {
			t.Errorf("blocked_by should be normalized to lowercase, got %v", newTask.BlockedBy)
		}
	})

	t.Run("it trims whitespace from title", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "   Trimmed title   "})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Title != "Trimmed title" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "Trimmed title")
		}
	})

	t.Run("it sets timestamps to current UTC ISO 8601", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "create", "Timestamp test"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		task := tasks[0]

		// Verify timestamps are set and in RFC3339 format
		if task.Created == "" {
			t.Error("created timestamp should be set")
		}
		if task.Updated == "" {
			t.Error("updated timestamp should be set")
		}

		// Created and updated should be equal for new tasks
		if task.Created != task.Updated {
			t.Errorf("created (%s) and updated (%s) should be equal for new task", task.Created, task.Updated)
		}

		// Should contain 'T' and 'Z' as per ISO 8601 UTC format
		if !strings.Contains(task.Created, "T") || !strings.Contains(task.Created, "Z") {
			t.Errorf("timestamp should be ISO 8601 UTC format, got %q", task.Created)
		}
	})
}

