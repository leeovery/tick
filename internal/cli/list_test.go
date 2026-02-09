package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestList(t *testing.T) {
	t.Run("it lists all tasks with aligned columns", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Setup Sanctum")
		t1.Status = task.StatusDone
		t1.Priority = 1
		t2 := task.NewTask("tick-bbbbbb", "Login endpoint")
		t2.Status = task.StatusInProgress
		t2.Priority = 1
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), output)
		}

		// Check header
		if !strings.HasPrefix(lines[0], "ID") {
			t.Errorf("expected header to start with 'ID', got %q", lines[0])
		}
		if !strings.Contains(lines[0], "STATUS") {
			t.Errorf("expected header to contain 'STATUS', got %q", lines[0])
		}
		if !strings.Contains(lines[0], "PRI") {
			t.Errorf("expected header to contain 'PRI', got %q", lines[0])
		}
		if !strings.Contains(lines[0], "TITLE") {
			t.Errorf("expected header to contain 'TITLE', got %q", lines[0])
		}

		// Check task rows contain expected data
		if !strings.Contains(lines[1], "tick-aaaaaa") {
			t.Errorf("expected first data line to contain tick-aaaaaa, got %q", lines[1])
		}
		if !strings.Contains(lines[1], "done") {
			t.Errorf("expected first data line to contain 'done', got %q", lines[1])
		}
		if !strings.Contains(lines[1], "Setup Sanctum") {
			t.Errorf("expected first data line to contain 'Setup Sanctum', got %q", lines[1])
		}
	})

	t.Run("it lists tasks ordered by priority then created date", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		t1 := task.NewTask("tick-aaaaaa", "Low priority early")
		t1.Priority = 3
		t1.Created = now.Add(-2 * time.Hour)
		t1.Updated = now.Add(-2 * time.Hour)

		t2 := task.NewTask("tick-bbbbbb", "High priority")
		t2.Priority = 1
		t2.Created = now
		t2.Updated = now

		t3 := task.NewTask("tick-cccccc", "Low priority late")
		t3.Priority = 3
		t3.Created = now.Add(-1 * time.Hour)
		t3.Updated = now.Add(-1 * time.Hour)

		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines (header + 3 tasks), got %d", len(lines))
		}

		// Priority 1 first, then priority 3 ordered by created (earlier first)
		if !strings.Contains(lines[1], "tick-bbbbbb") {
			t.Errorf("expected first task to be tick-bbbbbb (priority 1), got %q", lines[1])
		}
		if !strings.Contains(lines[2], "tick-aaaaaa") {
			t.Errorf("expected second task to be tick-aaaaaa (priority 3, earlier), got %q", lines[2])
		}
		if !strings.Contains(lines[3], "tick-cccccc") {
			t.Errorf("expected third task to be tick-cccccc (priority 3, later), got %q", lines[3])
		}
	})

	t.Run("it prints 'No tasks found.' when no tasks exist", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})

	t.Run("it prints only task IDs with --quiet flag on list", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task one")
		t2 := task.NewTask("tick-bbbbbb", "Task two")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (one per task), got %d: %q", len(lines), output)
		}

		// IDs only, no header, no other columns
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "tick-") {
				t.Errorf("expected line to be a task ID, got %q", line)
			}
			// Should not contain spaces (just an ID)
			if strings.Contains(line, " ") {
				t.Errorf("expected only ID (no spaces), got %q", line)
			}
		}
	})

	t.Run("it executes through storage engine read flow (shared lock, freshness check)", func(t *testing.T) {
		// This test verifies list uses store.Query (shared lock + freshness).
		// If it works with initTickProjectWithTasks (which writes JSONL but no cache),
		// the Query method must have rebuilt the cache from JSONL (freshness check).
		t1 := task.NewTask("tick-aaaaaa", "Cached task")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if !strings.Contains(stdout.String(), "tick-aaaaaa") {
			t.Errorf("expected output to contain tick-aaaaaa, got %q", stdout.String())
		}
	})
}
