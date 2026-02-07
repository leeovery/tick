package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestShow_FullTaskDetails(t *testing.T) {
	t.Run("it shows full task details by ID", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusInProgress, Priority: 1, Created: now, Updated: now.Add(4*time.Hour + 30*time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID:") {
			t.Errorf("expected output to contain 'ID:', got %q", output)
		}
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected output to contain task ID, got %q", output)
		}
		if !strings.Contains(output, "Title:") {
			t.Errorf("expected output to contain 'Title:', got %q", output)
		}
		if !strings.Contains(output, "Setup Sanctum") {
			t.Errorf("expected output to contain title, got %q", output)
		}
		if !strings.Contains(output, "Status:") {
			t.Errorf("expected output to contain 'Status:', got %q", output)
		}
		if !strings.Contains(output, "in_progress") {
			t.Errorf("expected output to contain status, got %q", output)
		}
		if !strings.Contains(output, "Priority:") {
			t.Errorf("expected output to contain 'Priority:', got %q", output)
		}
		if !strings.Contains(output, "Created:") {
			t.Errorf("expected output to contain 'Created:', got %q", output)
		}
		if !strings.Contains(output, "2026-01-19T10:00:00Z") {
			t.Errorf("expected output to contain created timestamp, got %q", output)
		}
		if !strings.Contains(output, "Updated:") {
			t.Errorf("expected output to contain 'Updated:', got %q", output)
		}
		if !strings.Contains(output, "2026-01-19T14:30:00Z") {
			t.Errorf("expected output to contain updated timestamp, got %q", output)
		}
	})
}

func TestShow_BlockedBySection(t *testing.T) {
	t.Run("it shows blocked_by section with ID, title, and status of each blocker", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker task", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 1, BlockedBy: []string{"tick-aaa111"}, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Blocked by:") {
			t.Errorf("expected output to contain 'Blocked by:', got %q", output)
		}
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected blocked_by section to contain blocker ID, got %q", output)
		}
		if !strings.Contains(output, "Blocker task") {
			t.Errorf("expected blocked_by section to contain blocker title, got %q", output)
		}
		if !strings.Contains(output, "(done)") {
			t.Errorf("expected blocked_by section to contain blocker status, got %q", output)
		}
	})
}

func TestShow_ChildrenSection(t *testing.T) {
	t.Run("it shows children section with ID, title, and status of each child", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Child task one", Status: task.StatusOpen, Priority: 2, Parent: "tick-aaa111", Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Children:") {
			t.Errorf("expected output to contain 'Children:', got %q", output)
		}
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected children section to contain child ID, got %q", output)
		}
		if !strings.Contains(output, "Child task one") {
			t.Errorf("expected children section to contain child title, got %q", output)
		}
		if !strings.Contains(output, "(open)") {
			t.Errorf("expected children section to contain child status, got %q", output)
		}
	})
}

func TestShow_DescriptionSection(t *testing.T) {
	t.Run("it shows description section when description is present", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task with desc", Status: task.StatusOpen, Priority: 1, Description: "Implement the login endpoint...", Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Description:") {
			t.Errorf("expected output to contain 'Description:', got %q", output)
		}
		if !strings.Contains(output, "Implement the login endpoint...") {
			t.Errorf("expected output to contain description text, got %q", output)
		}
	})
}

func TestShow_OmitsBlockedByWhenEmpty(t *testing.T) {
	t.Run("it omits blocked_by section when task has no dependencies", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No deps", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Blocked by:") {
			t.Errorf("expected output to NOT contain 'Blocked by:' when task has no deps, got %q", output)
		}
	})
}

func TestShow_OmitsChildrenWhenEmpty(t *testing.T) {
	t.Run("it omits children section when task has no children", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No children", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Children:") {
			t.Errorf("expected output to NOT contain 'Children:' when task has no children, got %q", output)
		}
	})
}

func TestShow_OmitsDescriptionWhenEmpty(t *testing.T) {
	t.Run("it omits description section when description is empty", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "No description", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Description:") {
			t.Errorf("expected output to NOT contain 'Description:' when empty, got %q", output)
		}
	})
}

func TestShow_ParentFieldWhenSet(t *testing.T) {
	t.Run("it shows parent field with ID and title when parent is set", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Child task", Status: task.StatusOpen, Priority: 2, Parent: "tick-aaa111", Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Parent:") {
			t.Errorf("expected output to contain 'Parent:', got %q", output)
		}
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected parent field to contain parent ID, got %q", output)
		}
		if !strings.Contains(output, "Parent task") {
			t.Errorf("expected parent field to contain parent title, got %q", output)
		}
	})
}

func TestShow_OmitsParentWhenNull(t *testing.T) {
	t.Run("it omits parent field when parent is null", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Root task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Parent:") {
			t.Errorf("expected output to NOT contain 'Parent:' when parent is null, got %q", output)
		}
	})
}

func TestShow_ClosedTimestampWhenDone(t *testing.T) {
	t.Run("it shows closed timestamp when task is done or cancelled", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closedAt := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Done task", Status: task.StatusDone, Priority: 1, Created: now, Updated: closedAt, Closed: &closedAt},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Closed:") {
			t.Errorf("expected output to contain 'Closed:', got %q", output)
		}
		if !strings.Contains(output, "2026-01-19T16:00:00Z") {
			t.Errorf("expected output to contain closed timestamp, got %q", output)
		}
	})
}

func TestShow_OmitsClosedWhenOpen(t *testing.T) {
	t.Run("it omits closed field when task is open or in_progress", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if strings.Contains(output, "Closed:") {
			t.Errorf("expected output to NOT contain 'Closed:' when task is open, got %q", output)
		}
	})
}

func TestShow_ErrorNotFound(t *testing.T) {
	t.Run("it errors when task ID not found", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show", "tick-xyz123"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "tick-xyz123") {
			t.Errorf("expected error to contain the task ID, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "not found") {
			t.Errorf("expected error to contain 'not found', got %q", errMsg)
		}
	})
}

func TestShow_ErrorNoIDArgument(t *testing.T) {
	t.Run("it errors when no ID argument provided to show", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "show"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Task ID is required") {
			t.Errorf("expected error to contain 'Task ID is required', got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Usage: tick show <id>") {
			t.Errorf("expected error to contain usage hint, got %q", errMsg)
		}
	})
}

func TestShow_NormalizesInputID(t *testing.T) {
	t.Run("it normalizes input ID to lowercase for show lookup", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Some task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		// Use uppercase ID
		code := app.Run([]string{"tick", "show", "TICK-AAA111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected output to contain normalized ID 'tick-aaa111', got %q", output)
		}
	})
}

func TestShow_QuietFlag(t *testing.T) {
	t.Run("it outputs only task ID with --quiet flag on show", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Some task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("expected only task ID in quiet output, got %q", output)
		}
	})
}

func TestShow_UsesStorageEngineReadFlow(t *testing.T) {
	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		// First show call should work (creates cache)
		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Second show should also work (uses existing cache, freshness check)
		stdout.Reset()
		stderr.Reset()
		code = app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0 on second call, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected output to contain task ID, got %q", output)
		}
	})
}
