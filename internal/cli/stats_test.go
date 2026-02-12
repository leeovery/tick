package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runStats runs the tick stats command with the given args and returns stdout, stderr, and exit code.
func runStats(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  false, // default to TOON format
	}
	fullArgs := append([]string{"tick", "stats"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestStats(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	closedTime := now.Add(time.Hour)

	t.Run("it counts tasks by status correctly", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Open task 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-bbb111", Title: "In progress", Status: task.StatusInProgress, Priority: 1, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-ccc111", Title: "Done task 1", Status: task.StatusDone, Priority: 2, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second), Closed: &closedTime},
			{ID: "tick-ccc222", Title: "Done task 2", Status: task.StatusDone, Priority: 3, Created: now.Add(4 * time.Second), Updated: now.Add(4 * time.Second), Closed: &closedTime},
			{ID: "tick-ccc333", Title: "Done task 3", Status: task.StatusDone, Priority: 0, Created: now.Add(5 * time.Second), Updated: now.Add(5 * time.Second), Closed: &closedTime},
			{ID: "tick-ddd111", Title: "Cancelled", Status: task.StatusCancelled, Priority: 4, Created: now.Add(6 * time.Second), Updated: now.Add(6 * time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Use --json for structured parsing.
		stdout, stderr, exitCode := runStats(t, dir, "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout)
		}

		if parsed["total"] != float64(7) {
			t.Errorf("total = %v, want 7", parsed["total"])
		}

		byStatus := parsed["by_status"].(map[string]interface{})
		if byStatus["open"] != float64(2) {
			t.Errorf("open = %v, want 2", byStatus["open"])
		}
		if byStatus["in_progress"] != float64(1) {
			t.Errorf("in_progress = %v, want 1", byStatus["in_progress"])
		}
		if byStatus["done"] != float64(3) {
			t.Errorf("done = %v, want 3", byStatus["done"])
		}
		if byStatus["cancelled"] != float64(1) {
			t.Errorf("cancelled = %v, want 1", byStatus["cancelled"])
		}
	})

	t.Run("it counts ready and blocked tasks correctly", func(t *testing.T) {
		// Setup:
		// tick-aaa111: open, no blockers, no children => READY
		// tick-aaa222: open, blocked by tick-bbb111 (in_progress) => BLOCKED
		// tick-bbb111: in_progress => neither ready nor blocked (not open)
		// tick-ccc111: open, has open child tick-ccc222 => BLOCKED (has open children)
		// tick-ccc222: open, child of tick-ccc111, no blockers => READY
		// tick-ddd111: done => neither
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Ready leaf", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Blocked by dep", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-bbb111", Title: "In progress blocker", Status: task.StatusInProgress, Priority: 1, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-ccc111", Title: "Parent with open child", Status: task.StatusOpen, Priority: 2, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
			{ID: "tick-ccc222", Title: "Open child leaf", Status: task.StatusOpen, Priority: 2, Parent: "tick-ccc111", Created: now.Add(4 * time.Second), Updated: now.Add(4 * time.Second)},
			{ID: "tick-ddd111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(5 * time.Second), Updated: now.Add(5 * time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runStats(t, dir, "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout)
		}

		workflow := parsed["workflow"].(map[string]interface{})
		// Ready: tick-aaa111 (open, unblocked, no children) + tick-ccc222 (open, unblocked, no children) = 2
		if workflow["ready"] != float64(2) {
			t.Errorf("ready = %v, want 2", workflow["ready"])
		}
		// Blocked: tick-aaa222 (blocked by dep) + tick-ccc111 (has open child) = 2
		if workflow["blocked"] != float64(2) {
			t.Errorf("blocked = %v, want 2", workflow["blocked"])
		}
	})

	t.Run("it includes all 5 priority levels even at zero", func(t *testing.T) {
		// Only priority 2 tasks, others should be 0.
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Task 2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runStats(t, dir, "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout)
		}

		byPriority := parsed["by_priority"].([]interface{})
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}

		expectedCounts := []float64{0, 0, 2, 0, 0}
		for i, expected := range expectedCounts {
			entry := byPriority[i].(map[string]interface{})
			if entry["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %v", i, entry["priority"], i)
			}
			if entry["count"] != expected {
				t.Errorf("by_priority[%d].count = %v, want %v", i, entry["count"], expected)
			}
		}
	})

	t.Run("it returns all zeros for empty project", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, stderr, exitCode := runStats(t, dir, "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout)
		}

		if parsed["total"] != float64(0) {
			t.Errorf("total = %v, want 0", parsed["total"])
		}

		byStatus := parsed["by_status"].(map[string]interface{})
		for _, key := range []string{"open", "in_progress", "done", "cancelled"} {
			if byStatus[key] != float64(0) {
				t.Errorf("by_status.%s = %v, want 0", key, byStatus[key])
			}
		}

		workflow := parsed["workflow"].(map[string]interface{})
		if workflow["ready"] != float64(0) {
			t.Errorf("workflow.ready = %v, want 0", workflow["ready"])
		}
		if workflow["blocked"] != float64(0) {
			t.Errorf("workflow.blocked = %v, want 0", workflow["blocked"])
		}

		byPriority := parsed["by_priority"].([]interface{})
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}
		for i := 0; i < 5; i++ {
			entry := byPriority[i].(map[string]interface{})
			if entry["count"] != float64(0) {
				t.Errorf("by_priority[%d].count = %v, want 0", i, entry["count"])
			}
		}
	})

	t.Run("it formats stats in TOON format", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// IsTTY=false defaults to TOON
		stdout, stderr, exitCode := runStats(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := strings.TrimRight(stdout, "\n")
		sections := strings.Split(output, "\n\n")
		if len(sections) != 2 {
			t.Fatalf("expected 2 sections, got %d: %q", len(sections), output)
		}

		// Section 1: stats summary
		summaryLines := strings.Split(sections[0], "\n")
		expectedHeader := "stats{total,open,in_progress,done,cancelled,ready,blocked}:"
		if summaryLines[0] != expectedHeader {
			t.Errorf("stats header = %q, want %q", summaryLines[0], expectedHeader)
		}
		// Total=2, Open=1, InProgress=0, Done=1, Cancelled=0, Ready=1, Blocked=0
		expectedRow := "  2,1,0,1,0,1,0"
		if summaryLines[1] != expectedRow {
			t.Errorf("stats row = %q, want %q", summaryLines[1], expectedRow)
		}

		// Section 2: by_priority
		priorityLines := strings.Split(sections[1], "\n")
		expectedPriorityHeader := "by_priority[5]{priority,count}:"
		if priorityLines[0] != expectedPriorityHeader {
			t.Errorf("by_priority header = %q, want %q", priorityLines[0], expectedPriorityHeader)
		}
		if len(priorityLines) != 6 {
			t.Fatalf("expected 6 priority lines (header + 5 rows), got %d: %q", len(priorityLines), sections[1])
		}
		expectedPriRows := []string{"  0,0", "  1,1", "  2,1", "  3,0", "  4,0"}
		for i, expected := range expectedPriRows {
			if priorityLines[i+1] != expected {
				t.Errorf("priority row %d = %q, want %q", i, priorityLines[i+1], expected)
			}
		}
	})

	t.Run("it formats stats in Pretty format with right-aligned numbers", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runStats(t, dir, "--pretty")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := strings.TrimRight(stdout, "\n")
		// Should contain all groups
		if !strings.Contains(output, "Total:") {
			t.Error("missing Total: section")
		}
		if !strings.Contains(output, "Status:") {
			t.Error("missing Status: section")
		}
		if !strings.Contains(output, "Workflow:") {
			t.Error("missing Workflow: section")
		}
		if !strings.Contains(output, "Priority:") {
			t.Error("missing Priority: section")
		}
		// Numbers should be right-aligned (check exact format for known values)
		if !strings.Contains(output, "Total:        2") {
			t.Errorf("Total line not right-aligned as expected in:\n%s", output)
		}
		if !strings.Contains(output, "P0 (critical):  0") {
			t.Errorf("P0 line not present as expected in:\n%s", output)
		}
		if !strings.Contains(output, "P4 (backlog):   0") {
			t.Errorf("P4 line not present as expected in:\n%s", output)
		}
	})

	t.Run("it formats stats in JSON format with nested structure", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second), Closed: &closedTime},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runStats(t, dir, "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout)
		}

		// Verify nested structure has correct keys.
		if _, ok := parsed["total"]; !ok {
			t.Error("missing 'total' key")
		}
		if _, ok := parsed["by_status"]; !ok {
			t.Error("missing 'by_status' key")
		}
		if _, ok := parsed["workflow"]; !ok {
			t.Error("missing 'workflow' key")
		}
		if _, ok := parsed["by_priority"]; !ok {
			t.Error("missing 'by_priority' key")
		}

		// Verify values
		if parsed["total"] != float64(2) {
			t.Errorf("total = %v, want 2", parsed["total"])
		}

		byStatus := parsed["by_status"].(map[string]interface{})
		if byStatus["open"] != float64(1) {
			t.Errorf("by_status.open = %v, want 1", byStatus["open"])
		}
		if byStatus["done"] != float64(1) {
			t.Errorf("by_status.done = %v, want 1", byStatus["done"])
		}

		workflow := parsed["workflow"].(map[string]interface{})
		if workflow["ready"] != float64(1) {
			t.Errorf("workflow.ready = %v, want 1", workflow["ready"])
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runStats(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if stdout != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout)
		}
	})
}
