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

func TestList_StatusFilter(t *testing.T) {
	t.Run("it filters by --status (all 4 values)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "In progress task", Status: task.StatusInProgress, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-ccc333", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), Closed: &closed},
			{ID: "tick-ddd444", Title: "Cancelled task", Status: task.StatusCancelled, Priority: 2, Created: now.Add(3 * time.Minute), Updated: now.Add(3 * time.Minute), Closed: &closed},
		}

		tests := []struct {
			status   string
			expected string
			excluded []string
		}{
			{"open", "tick-aaa111", []string{"tick-bbb222", "tick-ccc333", "tick-ddd444"}},
			{"in_progress", "tick-bbb222", []string{"tick-aaa111", "tick-ccc333", "tick-ddd444"}},
			{"done", "tick-ccc333", []string{"tick-aaa111", "tick-bbb222", "tick-ddd444"}},
			{"cancelled", "tick-ddd444", []string{"tick-aaa111", "tick-bbb222", "tick-ccc333"}},
		}

		for _, tt := range tests {
			t.Run(tt.status, func(t *testing.T) {
				dir := setupInitializedDirWithTasks(t, tasks)
				var stdout, stderr bytes.Buffer

				app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
				code := app.Run([]string{"tick", "list", "--status", tt.status})
				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				output := stdout.String()
				if !strings.Contains(output, tt.expected) {
					t.Errorf("expected %s in output, got %q", tt.expected, output)
				}
				for _, ex := range tt.excluded {
					if strings.Contains(output, ex) {
						t.Errorf("expected %s to be excluded, got %q", ex, output)
					}
				}
			})
		}
	})
}

func TestList_PriorityFilter(t *testing.T) {
	t.Run("it filters by --priority", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "P0 task", Status: task.StatusOpen, Priority: 0, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "P1 task", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-ccc333", Title: "P2 task", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--priority", "1"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in output, got %q", output)
		}
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 excluded, got %q", output)
		}
		if strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 excluded, got %q", output)
		}
	})
}

func TestList_CombineReadyWithPriority(t *testing.T) {
	t.Run("it combines --ready with --priority", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked P2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Ready P2", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute)},
			{ID: "tick-ddd444", Title: "Ready P1", Status: task.StatusOpen, Priority: 1, Created: now.Add(3 * time.Minute), Updated: now.Add(3 * time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--ready", "--priority", "2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Only tick-ccc333 is ready AND priority 2
		if !strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 in output, got %q", output)
		}
		// tick-aaa111 is ready but P1
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 excluded (P1), got %q", output)
		}
		// tick-ddd444 is ready but P1
		if strings.Contains(output, "tick-ddd444") {
			t.Errorf("expected tick-ddd444 excluded (P1), got %q", output)
		}
		// tick-bbb222 is blocked
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 excluded (blocked), got %q", output)
		}
	})
}

func TestList_CombineStatusWithPriority(t *testing.T) {
	t.Run("it combines --status with --priority", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open P1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Open P2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-ccc333", Title: "Done P1", Status: task.StatusDone, Priority: 1, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--status", "open", "--priority", "1"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Only tick-aaa111 matches both open AND P1
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 in output, got %q", output)
		}
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 excluded (P2), got %q", output)
		}
		if strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 excluded (done), got %q", output)
		}
	})
}

func TestList_ErrorReadyAndBlocked(t *testing.T) {
	t.Run("it errors when --ready and --blocked both set", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--ready", "--blocked"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "mutually exclusive") {
			t.Errorf("expected mutually exclusive error, got %q", errMsg)
		}
	})
}

func TestList_ErrorInvalidStatusPriority(t *testing.T) {
	t.Run("it errors for invalid status/priority values", func(t *testing.T) {
		tests := []struct {
			name    string
			args    []string
			errText string
		}{
			{
				name:    "invalid status",
				args:    []string{"tick", "list", "--status", "invalid"},
				errText: "invalid status",
			},
			{
				name:    "invalid priority negative",
				args:    []string{"tick", "list", "--priority", "-1"},
				errText: "invalid priority",
			},
			{
				name:    "invalid priority too high",
				args:    []string{"tick", "list", "--priority", "5"},
				errText: "invalid priority",
			},
			{
				name:    "invalid priority not a number",
				args:    []string{"tick", "list", "--priority", "abc"},
				errText: "invalid priority",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupInitializedDir(t)
				var stdout, stderr bytes.Buffer

				app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
				code := app.Run(tt.args)
				if code != 1 {
					t.Fatalf("expected exit code 1, got %d", code)
				}

				errMsg := stderr.String()
				if !strings.Contains(strings.ToLower(errMsg), tt.errText) {
					t.Errorf("expected error containing %q, got %q", tt.errText, errMsg)
				}
			})
		}
	})
}

func TestList_NoMatchesReturnsNoTasksFound(t *testing.T) {
	t.Run("it returns 'No tasks found.' when no matches", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--priority", "4"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected %q, got %q", "No tasks found.", output)
		}
	})
}

func TestList_QuietAfterFiltering(t *testing.T) {
	t.Run("it outputs IDs only with --quiet after filtering", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open P1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Open P2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-ccc333", Title: "Done P1", Status: task.StatusDone, Priority: 1, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "list", "--status", "open"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 IDs, got %d:\n%s", len(lines), output)
		}
		if strings.TrimSpace(lines[0]) != "tick-aaa111" {
			t.Errorf("expected first ID tick-aaa111, got %q", lines[0])
		}
		if strings.TrimSpace(lines[1]) != "tick-bbb222" {
			t.Errorf("expected second ID tick-bbb222, got %q", lines[1])
		}
	})
}

func TestList_AllTasksNoFilters(t *testing.T) {
	t.Run("it returns all tasks with no filters", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "In progress", Status: task.StatusInProgress, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-ccc333", Title: "Done", Status: task.StatusDone, Priority: 3, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111, got %q", output)
		}
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222, got %q", output)
		}
		if !strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333, got %q", output)
		}
	})
}

func TestList_DeterministicOrdering(t *testing.T) {
	t.Run("it maintains deterministic ordering", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-ccc333", Title: "P2 older", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa111", Title: "P1 task", Status: task.StatusOpen, Priority: 1, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
			{ID: "tick-bbb222", Title: "P2 newer", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)

		// Run twice and verify identical output
		for run := 0; run < 2; run++ {
			var stdout, stderr bytes.Buffer
			app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
			code := app.Run([]string{"tick", "list", "--status", "open"})
			if code != 0 {
				t.Fatalf("run %d: expected exit code 0, got %d", run, code)
			}

			lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
			if len(lines) != 4 { // header + 3
				t.Fatalf("run %d: expected 4 lines, got %d", run, len(lines))
			}
			// P1 first, then P2 ordered by created
			if !strings.Contains(lines[1], "tick-aaa111") {
				t.Errorf("run %d: expected tick-aaa111 first (P1), got %q", run, lines[1])
			}
			if !strings.Contains(lines[2], "tick-ccc333") {
				t.Errorf("run %d: expected tick-ccc333 second (P2 older), got %q", run, lines[2])
			}
			if !strings.Contains(lines[3], "tick-bbb222") {
				t.Errorf("run %d: expected tick-bbb222 third (P2 newer), got %q", run, lines[3])
			}
		}
	})
}

func TestList_BlockedFlag(t *testing.T) {
	t.Run("it filters to blocked tasks with --blocked", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Ready task", Status: task.StatusOpen, Priority: 1, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--blocked"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-bbb222 is blocked
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in output, got %q", output)
		}
		// tick-aaa111 is ready, should NOT appear
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 to be excluded (ready), got %q", output)
		}
		// tick-ccc333 is ready, should NOT appear
		if strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 to be excluded (ready), got %q", output)
		}
	})
}

func TestList_CombineBlockedWithPriority(t *testing.T) {
	t.Run("it combines --blocked with --priority", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker P1", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked P2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Blocked P1", Status: task.StatusOpen, Priority: 1, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ddd444", Title: "Ready P2", Status: task.StatusOpen, Priority: 2, Created: now.Add(3 * time.Minute), Updated: now.Add(3 * time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--blocked", "--priority", "2"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Only tick-bbb222 is blocked AND priority 2
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 in output, got %q", output)
		}
		// tick-aaa111 is ready (not blocked), P1
		if strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 excluded (ready, P1), got %q", output)
		}
		// tick-ccc333 is blocked but P1, not P2
		if strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 excluded (P1), got %q", output)
		}
		// tick-ddd444 is ready (not blocked), P2
		if strings.Contains(output, "tick-ddd444") {
			t.Errorf("expected tick-ddd444 excluded (ready), got %q", output)
		}
	})
}

func TestList_ReadyFlag(t *testing.T) {
	t.Run("it filters to ready tasks with --ready", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-aaa111"}},
			{ID: "tick-ccc333", Title: "Ready task", Status: task.StatusOpen, Priority: 1, Created: now.Add(2 * time.Minute), Updated: now.Add(2 * time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list", "--ready"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// tick-aaa111 is ready (open, no blockers, no children)
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected tick-aaa111 in output, got %q", output)
		}
		// tick-ccc333 is ready (open, no blockers, no children)
		if !strings.Contains(output, "tick-ccc333") {
			t.Errorf("expected tick-ccc333 in output, got %q", output)
		}
		// tick-bbb222 is blocked, should NOT appear
		if strings.Contains(output, "tick-bbb222") {
			t.Errorf("expected tick-bbb222 to be excluded (blocked), got %q", output)
		}
	})
}
