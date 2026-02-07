package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestList_AllTasksWithAlignedColumns(t *testing.T) {
	t.Run("it lists all tasks with aligned columns", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Login endpoint", Status: task.StatusInProgress, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 2 task rows
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d:\n%s", len(lines), output)
		}

		// Header line
		header := lines[0]
		if !strings.HasPrefix(header, "ID") {
			t.Errorf("expected header to start with 'ID', got %q", header)
		}
		if !strings.Contains(header, "STATUS") {
			t.Errorf("expected header to contain 'STATUS', got %q", header)
		}
		if !strings.Contains(header, "PRI") {
			t.Errorf("expected header to contain 'PRI', got %q", header)
		}
		if !strings.Contains(header, "TITLE") {
			t.Errorf("expected header to contain 'TITLE', got %q", header)
		}

		// Task rows should contain the data
		if !strings.Contains(lines[1], "tick-aaa111") {
			t.Errorf("expected first row to contain task ID, got %q", lines[1])
		}
		if !strings.Contains(lines[1], "done") {
			t.Errorf("expected first row to contain status 'done', got %q", lines[1])
		}
		if !strings.Contains(lines[1], "Setup Sanctum") {
			t.Errorf("expected first row to contain title, got %q", lines[1])
		}
	})
}

func TestList_OrderByPriorityThenCreated(t *testing.T) {
	t.Run("it lists tasks ordered by priority then created date", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-low111", Title: "Low priority", Status: task.StatusOpen, Priority: 3, Created: now, Updated: now},
			{ID: "tick-hi2222", Title: "High priority newer", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-hi1111", Title: "High priority older", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-med111", Title: "Med priority", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Skip header
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines (header + 4 tasks), got %d:\n%s", len(lines), output)
		}

		// Priority 1 tasks first (older then newer), then pri 2, then pri 3
		if !strings.Contains(lines[1], "tick-hi1111") {
			t.Errorf("expected first task to be high priority older, got %q", lines[1])
		}
		if !strings.Contains(lines[2], "tick-hi2222") {
			t.Errorf("expected second task to be high priority newer, got %q", lines[2])
		}
		if !strings.Contains(lines[3], "tick-med111") {
			t.Errorf("expected third task to be med priority, got %q", lines[3])
		}
		if !strings.Contains(lines[4], "tick-low111") {
			t.Errorf("expected fourth task to be low priority, got %q", lines[4])
		}
	})
}

func TestList_NoTasksFound(t *testing.T) {
	t.Run("it prints 'No tasks found.' when no tasks exist", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})
}

func TestList_QuietFlag(t *testing.T) {
	t.Run("it prints only task IDs with --quiet flag on list", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "First task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per task), got %d:\n%s", len(lines), output)
		}

		// Should be only IDs, ordered by priority then created
		if strings.TrimSpace(lines[0]) != "tick-aaa111" {
			t.Errorf("expected first line to be 'tick-aaa111', got %q", lines[0])
		}
		if strings.TrimSpace(lines[1]) != "tick-bbb222" {
			t.Errorf("expected second line to be 'tick-bbb222', got %q", lines[1])
		}
	})
}
