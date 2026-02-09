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

	t.Run("it filters to ready tasks with --ready", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		// ready task: open, no blockers, no children
		ready := task.NewTask("tick-aaaaaa", "Ready task")

		// blocked task: open, has open blocker
		blocker := task.NewTask("tick-bbbbbb", "Open blocker")
		blocked := task.NewTask("tick-cccccc", "Blocked task")
		blocked.BlockedBy = []string{"tick-bbbbbb"}

		// done task
		done := task.NewTask("tick-dddddd", "Done task")
		done.Status = task.StatusDone
		done.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{ready, blocker, blocked, done})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Ready tasks: tick-aaaaaa and tick-bbbbbb (both open, unblocked, no children)
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected ready task tick-aaaaaa in output, got %q", output)
		}
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected ready task tick-bbbbbb in output, got %q", output)
		}
		// Not ready: blocked and done
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected blocked task tick-cccccc NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected done task tick-dddddd NOT in output, got %q", output)
		}
	})

	t.Run("it filters to blocked tasks with --blocked", func(t *testing.T) {
		blocker := task.NewTask("tick-aaaaaa", "Open blocker")
		blocked := task.NewTask("tick-bbbbbb", "Blocked task")
		blocked.BlockedBy = []string{"tick-aaaaaa"}
		ready := task.NewTask("tick-cccccc", "Ready task")

		dir := initTickProjectWithTasks(t, []task.Task{blocker, blocked, ready})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--blocked"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected blocked task tick-bbbbbb in output, got %q", output)
		}
		if strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected unblocked task tick-aaaaaa NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected ready task tick-cccccc NOT in output, got %q", output)
		}
	})

	t.Run("it filters by --status (all 4 values)", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		open := task.NewTask("tick-aaaaaa", "Open task")
		ip := task.NewTask("tick-bbbbbb", "In progress task")
		ip.Status = task.StatusInProgress
		done := task.NewTask("tick-cccccc", "Done task")
		done.Status = task.StatusDone
		done.Closed = &now
		cancelled := task.NewTask("tick-dddddd", "Cancelled task")
		cancelled.Status = task.StatusCancelled
		cancelled.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{open, ip, done, cancelled})

		tests := []struct {
			status   string
			expected string
			excluded []string
		}{
			{"open", "tick-aaaaaa", []string{"tick-bbbbbb", "tick-cccccc", "tick-dddddd"}},
			{"in_progress", "tick-bbbbbb", []string{"tick-aaaaaa", "tick-cccccc", "tick-dddddd"}},
			{"done", "tick-cccccc", []string{"tick-aaaaaa", "tick-bbbbbb", "tick-dddddd"}},
			{"cancelled", "tick-dddddd", []string{"tick-aaaaaa", "tick-bbbbbb", "tick-cccccc"}},
		}

		for _, tt := range tests {
			t.Run(tt.status, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", "list", "--status", tt.status}, dir, &stdout, &stderr, false)

				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				output := stdout.String()
				if !strings.Contains(output, tt.expected) {
					t.Errorf("expected %s in output, got %q", tt.expected, output)
				}
				for _, ex := range tt.excluded {
					if strings.Contains(output, ex) {
						t.Errorf("expected %s NOT in output, got %q", ex, output)
					}
				}
			})
		}
	})

	t.Run("it filters by --priority", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Priority 1")
		t1.Priority = 1
		t2 := task.NewTask("tick-bbbbbb", "Priority 2")
		t2.Priority = 2
		t3 := task.NewTask("tick-cccccc", "Priority 3")
		t3.Priority = 3

		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--priority", "2"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected priority 2 task in output, got %q", output)
		}
		if strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected priority 1 task NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected priority 3 task NOT in output, got %q", output)
		}
	})

	t.Run("it combines --ready with --priority", func(t *testing.T) {
		blocker := task.NewTask("tick-aaaaaa", "Blocker")

		ready1 := task.NewTask("tick-bbbbbb", "Ready P1")
		ready1.Priority = 1
		ready2 := task.NewTask("tick-cccccc", "Ready P3")
		ready2.Priority = 3

		blocked := task.NewTask("tick-dddddd", "Blocked P1")
		blocked.Priority = 1
		blocked.BlockedBy = []string{"tick-aaaaaa"}

		dir := initTickProjectWithTasks(t, []task.Task{blocker, ready1, ready2, blocked})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--ready", "--priority", "1"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-bbbbbb is ready AND priority 1
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected ready P1 task in output, got %q", output)
		}
		// tick-aaaaaa is ready but priority 2 (default)
		if strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected blocker (priority 2) NOT in output, got %q", output)
		}
		// tick-cccccc is ready but priority 3
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected ready P3 task NOT in output, got %q", output)
		}
		// tick-dddddd is blocked
		if strings.Contains(output, "tick-dddddd") {
			t.Errorf("expected blocked task NOT in output, got %q", output)
		}
	})

	t.Run("it combines --status with --priority", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Open P1")
		t1.Priority = 1
		t2 := task.NewTask("tick-bbbbbb", "Open P3")
		t2.Priority = 3
		t3 := task.NewTask("tick-cccccc", "IP P1")
		t3.Status = task.StatusInProgress
		t3.Priority = 1

		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--status", "open", "--priority", "1"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected open P1 task in output, got %q", output)
		}
		if strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected open P3 task NOT in output, got %q", output)
		}
		if strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected IP P1 task NOT in output, got %q", output)
		}
	})

	t.Run("it errors when --ready and --blocked both set", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--ready", "--blocked"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "ready") || !strings.Contains(stderr.String(), "blocked") {
			t.Errorf("expected error mentioning --ready and --blocked, got %q", stderr.String())
		}
	})

	t.Run("it errors for invalid status value", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--status", "invalid"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "open") || !strings.Contains(errMsg, "done") {
			t.Errorf("expected error listing valid status values, got %q", errMsg)
		}
	})

	t.Run("it errors for invalid priority value", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--priority", "9"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "0") || !strings.Contains(errMsg, "4") {
			t.Errorf("expected error mentioning valid priority range 0-4, got %q", errMsg)
		}
	})

	t.Run("it errors for non-numeric priority value", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--priority", "abc"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
	})

	t.Run("it returns 'No tasks found.' when no matches", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		done := task.NewTask("tick-aaaaaa", "Done task")
		done.Status = task.StatusDone
		done.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{done})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--status", "open"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})

	t.Run("it outputs IDs only with --quiet after filtering", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Open P1")
		t1.Priority = 1
		t2 := task.NewTask("tick-bbbbbb", "Open P2")
		t2.Priority = 2

		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "list", "--priority", "1"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d: %q", len(lines), output)
		}
		if strings.TrimSpace(lines[0]) != "tick-aaaaaa" {
			t.Errorf("expected tick-aaaaaa, got %q", lines[0])
		}
	})

	t.Run("it returns all tasks with no filters", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		open := task.NewTask("tick-aaaaaa", "Open")
		ip := task.NewTask("tick-bbbbbb", "In progress")
		ip.Status = task.StatusInProgress
		done := task.NewTask("tick-cccccc", "Done")
		done.Status = task.StatusDone
		done.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{open, ip, done})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected tick-aaaaaa in output, got %q", output)
		}
		if !strings.Contains(output, "tick-bbbbbb") {
			t.Errorf("expected tick-bbbbbb in output, got %q", output)
		}
		if !strings.Contains(output, "tick-cccccc") {
			t.Errorf("expected tick-cccccc in output, got %q", output)
		}
	})

	t.Run("it maintains deterministic ordering", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		t1 := task.NewTask("tick-aaaaaa", "P3 early")
		t1.Priority = 3
		t1.Created = now.Add(-2 * time.Hour)
		t1.Updated = now.Add(-2 * time.Hour)

		t2 := task.NewTask("tick-bbbbbb", "P1")
		t2.Priority = 1
		t2.Created = now
		t2.Updated = now

		t3 := task.NewTask("tick-cccccc", "P3 late")
		t3.Priority = 3
		t3.Created = now.Add(-1 * time.Hour)
		t3.Updated = now.Add(-1 * time.Hour)

		dir := initTickProjectWithTasks(t, []task.Task{t1, t2, t3})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--status", "open"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines (header + 3 tasks), got %d", len(lines))
		}
		if !strings.Contains(lines[1], "tick-bbbbbb") {
			t.Errorf("expected first task to be tick-bbbbbb (P1), got %q", lines[1])
		}
		if !strings.Contains(lines[2], "tick-aaaaaa") {
			t.Errorf("expected second task to be tick-aaaaaa (P3, earlier), got %q", lines[2])
		}
		if !strings.Contains(lines[3], "tick-cccccc") {
			t.Errorf("expected third task to be tick-cccccc (P3, later), got %q", lines[3])
		}
	})

	t.Run("it handles contradictory filters with empty result not error", func(t *testing.T) {
		// --status done --ready is contradictory (ready requires open),
		// but should return empty result, not error
		now := time.Now().UTC().Truncate(time.Second)
		done := task.NewTask("tick-aaaaaa", "Done task")
		done.Status = task.StatusDone
		done.Closed = &now
		open := task.NewTask("tick-bbbbbb", "Open task")

		dir := initTickProjectWithTasks(t, []task.Task{done, open})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list", "--status", "done", "--ready"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0 for contradictory filters, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})
}
