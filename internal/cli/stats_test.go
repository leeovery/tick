package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestStatsCommand(t *testing.T) {
	t.Run("it counts tasks by status correctly", func(t *testing.T) {
		dir := setupTickDir(t)
		// Create tasks in various statuses
		setupTaskFull(t, dir, "tick-open1", "Open task one", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-open2", "Open task two", "open", 1, "", "", nil, "2026-01-19T10:01:00Z", "2026-01-19T10:01:00Z", "")
		setupTaskFull(t, dir, "tick-prog1", "In progress task", "in_progress", 2, "", "", nil, "2026-01-19T10:02:00Z", "2026-01-19T10:02:00Z", "")
		setupTaskFull(t, dir, "tick-done1", "Done task one", "done", 3, "", "", nil, "2026-01-19T10:03:00Z", "2026-01-19T10:03:00Z", "2026-01-19T11:00:00Z")
		setupTaskFull(t, dir, "tick-done2", "Done task two", "done", 3, "", "", nil, "2026-01-19T10:04:00Z", "2026-01-19T10:04:00Z", "2026-01-19T11:00:00Z")
		setupTaskFull(t, dir, "tick-done3", "Done task three", "done", 4, "", "", nil, "2026-01-19T10:05:00Z", "2026-01-19T10:05:00Z", "2026-01-19T11:00:00Z")
		setupTaskFull(t, dir, "tick-canc1", "Cancelled task", "cancelled", 0, "", "", nil, "2026-01-19T10:06:00Z", "2026-01-19T10:06:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Parse JSON output
		var stats struct {
			Total    int `json:"total"`
			ByStatus struct {
				Open       int `json:"open"`
				InProgress int `json:"in_progress"`
				Done       int `json:"done"`
				Cancelled  int `json:"cancelled"`
				Ready      int `json:"ready"`
				Blocked    int `json:"blocked"`
			} `json:"by_status"`
		}
		if err := json.Unmarshal([]byte(stdout.String()), &stats); err != nil {
			t.Fatalf("failed to parse JSON: %v; output: %s", err, stdout.String())
		}

		if stats.Total != 7 {
			t.Errorf("expected total 7, got %d", stats.Total)
		}
		if stats.ByStatus.Open != 2 {
			t.Errorf("expected open 2, got %d", stats.ByStatus.Open)
		}
		if stats.ByStatus.InProgress != 1 {
			t.Errorf("expected in_progress 1, got %d", stats.ByStatus.InProgress)
		}
		if stats.ByStatus.Done != 3 {
			t.Errorf("expected done 3, got %d", stats.ByStatus.Done)
		}
		if stats.ByStatus.Cancelled != 1 {
			t.Errorf("expected cancelled 1, got %d", stats.ByStatus.Cancelled)
		}
	})

	t.Run("it counts ready and blocked tasks correctly", func(t *testing.T) {
		dir := setupTickDir(t)
		// tick-blocker: open, no deps, no children -> ready
		setupTaskFull(t, dir, "tick-blocker", "Blocker task", "open", 1, "", "", nil, "2026-01-19T09:00:00Z", "2026-01-19T09:00:00Z", "")
		// tick-blocked: open, blocked by tick-blocker -> blocked
		setupTaskFull(t, dir, "tick-blocked", "Blocked task", "open", 2, "", "", []string{"tick-blocker"}, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		// tick-parent: open, has open child -> blocked
		setupTaskFull(t, dir, "tick-parent", "Parent task", "open", 1, "", "", nil, "2026-01-19T08:00:00Z", "2026-01-19T08:00:00Z", "")
		// tick-child: open child of tick-parent, no deps -> ready
		setupTaskFull(t, dir, "tick-child", "Child task", "open", 2, "", "tick-parent", nil, "2026-01-19T08:30:00Z", "2026-01-19T08:30:00Z", "")
		// tick-done: done -> not counted in ready/blocked
		setupTaskFull(t, dir, "tick-done1", "Done task", "done", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var stats struct {
			Total    int `json:"total"`
			ByStatus struct {
				Ready   int `json:"ready"`
				Blocked int `json:"blocked"`
			} `json:"by_status"`
		}
		if err := json.Unmarshal([]byte(stdout.String()), &stats); err != nil {
			t.Fatalf("failed to parse JSON: %v; output: %s", err, stdout.String())
		}

		if stats.Total != 5 {
			t.Errorf("expected total 5, got %d", stats.Total)
		}
		// Ready: tick-blocker (open, unblocked, no children), tick-child (open, unblocked, no children)
		if stats.ByStatus.Ready != 2 {
			t.Errorf("expected ready 2, got %d", stats.ByStatus.Ready)
		}
		// Blocked: tick-blocked (blocked by open dep), tick-parent (has open child)
		if stats.ByStatus.Blocked != 2 {
			t.Errorf("expected blocked 2, got %d", stats.ByStatus.Blocked)
		}
	})

	t.Run("it includes all 5 priority levels even at zero", func(t *testing.T) {
		dir := setupTickDir(t)
		// Only create tasks at priority 1 and 3
		setupTaskFull(t, dir, "tick-a1b2", "P1 task", "open", 1, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-c3d4", "P3 task", "open", 3, "", "", nil, "2026-01-19T10:01:00Z", "2026-01-19T10:01:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var stats struct {
			ByPriority []struct {
				Priority int `json:"priority"`
				Count    int `json:"count"`
			} `json:"by_priority"`
		}
		if err := json.Unmarshal([]byte(stdout.String()), &stats); err != nil {
			t.Fatalf("failed to parse JSON: %v; output: %s", err, stdout.String())
		}

		if len(stats.ByPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(stats.ByPriority))
		}

		// Verify all priorities present and correct counts
		expected := map[int]int{0: 0, 1: 1, 2: 0, 3: 1, 4: 0}
		for _, entry := range stats.ByPriority {
			want, ok := expected[entry.Priority]
			if !ok {
				t.Errorf("unexpected priority %d", entry.Priority)
				continue
			}
			if entry.Count != want {
				t.Errorf("priority %d: expected count %d, got %d", entry.Priority, want, entry.Count)
			}
		}
	})

	t.Run("it returns all zeros for empty project", func(t *testing.T) {
		dir := setupTickDir(t)

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var stats struct {
			Total    int `json:"total"`
			ByStatus struct {
				Open       int `json:"open"`
				InProgress int `json:"in_progress"`
				Done       int `json:"done"`
				Cancelled  int `json:"cancelled"`
				Ready      int `json:"ready"`
				Blocked    int `json:"blocked"`
			} `json:"by_status"`
			ByPriority []struct {
				Priority int `json:"priority"`
				Count    int `json:"count"`
			} `json:"by_priority"`
		}
		if err := json.Unmarshal([]byte(stdout.String()), &stats); err != nil {
			t.Fatalf("failed to parse JSON: %v; output: %s", err, stdout.String())
		}

		if stats.Total != 0 {
			t.Errorf("expected total 0, got %d", stats.Total)
		}
		if stats.ByStatus.Open != 0 {
			t.Errorf("expected open 0, got %d", stats.ByStatus.Open)
		}
		if stats.ByStatus.InProgress != 0 {
			t.Errorf("expected in_progress 0, got %d", stats.ByStatus.InProgress)
		}
		if stats.ByStatus.Done != 0 {
			t.Errorf("expected done 0, got %d", stats.ByStatus.Done)
		}
		if stats.ByStatus.Cancelled != 0 {
			t.Errorf("expected cancelled 0, got %d", stats.ByStatus.Cancelled)
		}
		if stats.ByStatus.Ready != 0 {
			t.Errorf("expected ready 0, got %d", stats.ByStatus.Ready)
		}
		if stats.ByStatus.Blocked != 0 {
			t.Errorf("expected blocked 0, got %d", stats.ByStatus.Blocked)
		}
		if len(stats.ByPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(stats.ByPriority))
		}
		for _, entry := range stats.ByPriority {
			if entry.Count != 0 {
				t.Errorf("priority %d: expected count 0, got %d", entry.Priority, entry.Count)
			}
		}
	})

	t.Run("it formats stats in TOON format", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-c3d4", "Done task", "done", 1, "", "", nil, "2026-01-19T10:01:00Z", "2026-01-19T10:01:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--toon", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()

		// Check stats header section
		if !strings.Contains(output, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("output should contain stats header, got %q", output)
		}

		// Check stats data line: total=2, open=1, in_progress=0, done=1, cancelled=0, ready=1, blocked=0
		if !strings.Contains(output, "  2,1,0,1,0,1,0") {
			t.Errorf("output should contain correct stats data, got %q", output)
		}

		// Check by_priority section
		if !strings.Contains(output, "by_priority[5]{priority,count}:") {
			t.Errorf("output should contain by_priority header, got %q", output)
		}

		// Check all 5 priority rows present
		for _, line := range []string{"  0,0", "  1,1", "  2,1", "  3,0", "  4,0"} {
			if !strings.Contains(output, line) {
				t.Errorf("output should contain priority line %q, got %q", line, output)
			}
		}
	})

	t.Run("it formats stats in Pretty format with right-aligned numbers", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Open task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")
		setupTaskFull(t, dir, "tick-c3d4", "Done task", "done", 1, "", "", nil, "2026-01-19T10:01:00Z", "2026-01-19T10:01:00Z", "2026-01-19T11:00:00Z")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--pretty", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()

		// Check total line
		if !strings.Contains(output, "Total:") {
			t.Errorf("output should contain Total:, got %q", output)
		}

		// Check status section headers
		if !strings.Contains(output, "Status:") {
			t.Errorf("output should contain Status: header, got %q", output)
		}
		if !strings.Contains(output, "Open:") {
			t.Errorf("output should contain Open:, got %q", output)
		}
		if !strings.Contains(output, "In Progress:") {
			t.Errorf("output should contain In Progress:, got %q", output)
		}
		if !strings.Contains(output, "Done:") {
			t.Errorf("output should contain Done:, got %q", output)
		}
		if !strings.Contains(output, "Cancelled:") {
			t.Errorf("output should contain Cancelled:, got %q", output)
		}

		// Check workflow section
		if !strings.Contains(output, "Workflow:") {
			t.Errorf("output should contain Workflow: header, got %q", output)
		}
		if !strings.Contains(output, "Ready:") {
			t.Errorf("output should contain Ready:, got %q", output)
		}
		if !strings.Contains(output, "Blocked:") {
			t.Errorf("output should contain Blocked:, got %q", output)
		}

		// Check priority section with labels
		if !strings.Contains(output, "Priority:") {
			t.Errorf("output should contain Priority: header, got %q", output)
		}
		if !strings.Contains(output, "P0 (critical):") {
			t.Errorf("output should contain P0 (critical):, got %q", output)
		}
		if !strings.Contains(output, "P4 (backlog):") {
			t.Errorf("output should contain P4 (backlog):, got %q", output)
		}
	})

	t.Run("it formats stats in JSON format with nested structure", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Open task", "open", 0, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()

		// Verify nested JSON structure
		if !strings.Contains(output, `"total"`) {
			t.Errorf("JSON should contain 'total' key, got %q", output)
		}
		if !strings.Contains(output, `"by_status"`) {
			t.Errorf("JSON should contain 'by_status' key, got %q", output)
		}
		if !strings.Contains(output, `"by_priority"`) {
			t.Errorf("JSON should contain 'by_priority' key, got %q", output)
		}

		// Parse and verify structure
		var stats struct {
			Total    int `json:"total"`
			ByStatus struct {
				Open       int `json:"open"`
				InProgress int `json:"in_progress"`
				Done       int `json:"done"`
				Cancelled  int `json:"cancelled"`
				Ready      int `json:"ready"`
				Blocked    int `json:"blocked"`
			} `json:"by_status"`
			ByPriority []struct {
				Priority int `json:"priority"`
				Count    int `json:"count"`
			} `json:"by_priority"`
		}
		if err := json.Unmarshal([]byte(output), &stats); err != nil {
			t.Fatalf("failed to parse JSON: %v; output: %s", err, output)
		}

		if stats.Total != 1 {
			t.Errorf("expected total 1, got %d", stats.Total)
		}
		if stats.ByStatus.Open != 1 {
			t.Errorf("expected open 1, got %d", stats.ByStatus.Open)
		}
		if stats.ByStatus.Ready != 1 {
			t.Errorf("expected ready 1, got %d", stats.ByStatus.Ready)
		}
		if len(stats.ByPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(stats.ByPriority))
		}
		// P0 should have count 1
		if stats.ByPriority[0].Priority != 0 || stats.ByPriority[0].Count != 1 {
			t.Errorf("P0 should have count 1, got priority=%d count=%d", stats.ByPriority[0].Priority, stats.ByPriority[0].Count)
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTaskFull(t, dir, "tick-a1b2", "Test task", "open", 2, "", "", nil, "2026-01-19T10:00:00Z", "2026-01-19T10:00:00Z", "")

		var stdout, stderr bytes.Buffer
		app := &App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}

		code := app.Run([]string{"tick", "--quiet", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected empty output with --quiet, got %q", stdout.String())
		}
	})
}
