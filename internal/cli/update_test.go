package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	t.Run("it updates title with --title flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Original title")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", "Updated title"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "Updated title" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "Updated title")
		}
	})

	t.Run("it updates description with --description flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "open", 2, "Old description", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--description", "New description"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Description != "New description" {
			t.Errorf("description = %q, want %q", tasks[0].Description, "New description")
		}
	})

	t.Run("it clears description with --description \"\"", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "open", 2, "Old description", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--description", ""})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Description != "" {
			t.Errorf("description should be cleared, got %q", tasks[0].Description)
		}
	})

	t.Run("it updates priority with --priority flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--priority", "0"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Priority != 0 {
			t.Errorf("priority = %d, want %d", tasks[0].Priority, 0)
		}
	})

	t.Run("it updates parent with --parent flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-parent", "Parent task")
		setupTask(t, dir, "tick-a1b2c3", "Child task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--parent", "tick-parent"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
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
		for i := range tasks {
			if tasks[i].ID == "tick-a1b2c3" {
				targetTask = &tasks[i]
				break
			}
		}
		if targetTask.Parent != "tick-parent" {
			t.Errorf("parent = %q, want %q", targetTask.Parent, "tick-parent")
		}
	})

	t.Run("it clears parent with --parent \"\"", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-parent", "Parent task")
		setupTaskFull(t, dir, "tick-a1b2c3", "Child task", "open", 2, "", "tick-parent", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--parent", ""})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
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
		for i := range tasks {
			if tasks[i].ID == "tick-a1b2c3" {
				targetTask = &tasks[i]
				break
			}
		}
		if targetTask.Parent != "" {
			t.Errorf("parent should be cleared, got %q", targetTask.Parent)
		}
	})

	t.Run("it updates blocks with --blocks flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Blocker task")
		setupTask(t, dir, "tick-target", "Target task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--blocks", "tick-target"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
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
		for i := range tasks {
			if tasks[i].ID == "tick-target" {
				targetTask = &tasks[i]
				break
			}
		}
		if len(targetTask.BlockedBy) != 1 || targetTask.BlockedBy[0] != "tick-a1b2c3" {
			t.Errorf("target's blocked_by = %v, want [tick-a1b2c3]", targetTask.BlockedBy)
		}
	})

	t.Run("it updates multiple fields in a single command", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Original title")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", "New title", "--priority", "1", "--description", "New description"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Title != "New title" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "New title")
		}
		if tasks[0].Priority != 1 {
			t.Errorf("priority = %d, want %d", tasks[0].Priority, 1)
		}
		if tasks[0].Description != "New description" {
			t.Errorf("description = %q, want %q", tasks[0].Description, "New description")
		}
	})

	t.Run("it refreshes updated timestamp on any change", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Test task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", "Updated title"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Updated == "2026-01-19T10:00:00Z" {
			t.Error("updated timestamp should have been refreshed")
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", "Updated title"})
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
		if !strings.Contains(output, "Updated title") {
			t.Errorf("output should contain updated title, got %q", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "update", "tick-a1b2c3", "--title", "Updated title"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-a1b2c3" {
			t.Errorf("quiet output should be task ID only, got %q", output)
		}
		if strings.Contains(output, "Title:") {
			t.Errorf("quiet output should not contain details, got %q", output)
		}
	})

	t.Run("it errors when no flags are provided", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "At least one flag is required") {
			t.Errorf("error should mention flags required, got %q", errOutput)
		}
		// Should list available options
		if !strings.Contains(errOutput, "--title") {
			t.Errorf("error should list available flags, got %q", errOutput)
		}
	})

	t.Run("it errors when task ID is missing", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "Task ID is required") {
			t.Errorf("error should mention task ID required, got %q", errOutput)
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-nonexistent", "--title", "New title"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "not found") {
			t.Errorf("error should mention 'not found', got %q", errOutput)
		}
	})

	t.Run("it errors on invalid title (empty/500/newlines)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		// Test empty title
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", ""})
		if code != 1 {
			t.Errorf("expected exit code 1 for empty title, got %d", code)
		}

		// Test whitespace-only title
		stderr.Reset()
		code = app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", "   "})
		if code != 1 {
			t.Errorf("expected exit code 1 for whitespace title, got %d", code)
		}

		// Test title with newlines
		stderr.Reset()
		code = app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", "Title\nwith newline"})
		if code != 1 {
			t.Errorf("expected exit code 1 for title with newline, got %d", code)
		}
		if !strings.Contains(stderr.String(), "newline") {
			t.Errorf("error should mention newlines, got %q", stderr.String())
		}

		// Test title over 500 chars
		stderr.Reset()
		longTitle := strings.Repeat("a", 501)
		code = app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", longTitle})
		if code != 1 {
			t.Errorf("expected exit code 1 for title over 500 chars, got %d", code)
		}
		if !strings.Contains(stderr.String(), "500") {
			t.Errorf("error should mention 500 char limit, got %q", stderr.String())
		}
	})

	t.Run("it errors on invalid priority (outside 0-4)", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--priority", "5"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		if !strings.Contains(stderr.String(), "priority") {
			t.Errorf("error should mention priority, got %q", stderr.String())
		}

		// Verify no change was made
		tasks := readTasksFromDir(t, dir)
		if tasks[0].Priority != 2 {
			t.Errorf("priority should not have changed, got %d", tasks[0].Priority)
		}
	})

	t.Run("it errors on non-existent parent/blocks IDs", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Test non-existent parent
		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--parent", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1 for non-existent parent, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("error should mention 'not found', got %q", stderr.String())
		}

		// Test non-existent blocks target
		stderr.Reset()
		code = app.Run([]string{"tick", "update", "tick-a1b2c3", "--blocks", "tick-nonexistent"})
		if code != 1 {
			t.Errorf("expected exit code 1 for non-existent blocks target, got %d", code)
		}
		if !strings.Contains(stderr.String(), "not found") {
			t.Errorf("error should mention 'not found', got %q", stderr.String())
		}
	})

	t.Run("it errors on self-referencing parent", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--parent", "tick-a1b2c3"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		if !strings.Contains(stderr.String(), "own parent") {
			t.Errorf("error should mention self-reference, got %q", stderr.String())
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")
		setupTask(t, dir, "tick-parent", "Parent task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Use uppercase IDs
		code := app.Run([]string{"tick", "update", "TICK-A1B2C3", "--parent", "TICK-PARENT"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
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
		for i := range tasks {
			if tasks[i].ID == "tick-a1b2c3" {
				targetTask = &tasks[i]
				break
			}
		}
		// Parent should be normalized to lowercase
		if targetTask.Parent != "tick-parent" {
			t.Errorf("parent should be normalized to lowercase, got %q", targetTask.Parent)
		}
	})

	t.Run("it persists changes via atomic write", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-a1b2c3", "Test task")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--title", "Updated title"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Create new app to verify persistence
		var stdout2, stderr2 bytes.Buffer
		app2 := &App{
			Stdout: &stdout2,
			Stderr: &stderr2,
			Cwd:    dir,
		}

		// Read fresh from disk
		tasks := readTasksFromDir(t, dir)
		if tasks[0].Title != "Updated title" {
			t.Errorf("persisted title = %q, want %q", tasks[0].Title, "Updated title")
		}

		// Verify via another operation
		code = app2.Run([]string{"tick", "show", "tick-a1b2c3"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr2.String())
		}
		if !strings.Contains(stdout2.String(), "Updated title") {
			t.Errorf("show output should contain updated title, got %q", stdout2.String())
		}
	})

	t.Run("it refreshes target task updated timestamp when --blocks is used", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2c3", "Blocker task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-target", "Target task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-a1b2c3", "--blocks", "tick-target"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
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
		for i := range tasks {
			if tasks[i].ID == "tick-target" {
				targetTask = &tasks[i]
				break
			}
		}
		if targetTask.Updated == "2026-01-19T10:00:00Z" {
			t.Error("target's updated timestamp should have been refreshed")
		}
	})
}
