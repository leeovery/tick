package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestStats(t *testing.T) {
	t.Run("it counts tasks by status correctly", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		open1 := task.NewTask("tick-aaaaaa", "Open one")
		open2 := task.NewTask("tick-bbbbbb", "Open two")

		ip := task.NewTask("tick-cccccc", "In progress")
		ip.Status = task.StatusInProgress

		done := task.NewTask("tick-dddddd", "Done task")
		done.Status = task.StatusDone
		done.Closed = &now

		cancelled := task.NewTask("tick-eeeeee", "Cancelled task")
		cancelled.Status = task.StatusCancelled
		cancelled.Closed = &now

		dir := initTickProjectWithTasks(t, []task.Task{open1, open2, ip, done, cancelled})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// TOON stats header: stats{total,open,in_progress,done,cancelled,ready,blocked}:
		// Values: total=5, open=2, in_progress=1, done=1, cancelled=1
		if !strings.Contains(output, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("expected stats header, got %q", output)
		}

		// Parse the stats line to check values.
		lines := strings.Split(output, "\n")
		if len(lines) < 2 {
			t.Fatalf("expected at least 2 lines, got %d", len(lines))
		}
		statsLine := strings.TrimSpace(lines[1])
		// Format: total,open,in_progress,done,cancelled,ready,blocked
		// Expect: 5,2,1,1,1,...
		if !strings.HasPrefix(statsLine, "5,2,1,1,1,") {
			t.Errorf("expected stats to start with '5,2,1,1,1,' got %q", statsLine)
		}
	})

	t.Run("it counts ready and blocked tasks correctly", func(t *testing.T) {
		// Setup: 3 open tasks
		// - ready1: open, no blockers, no children (ready)
		// - ready2: open, no blockers, no children (ready)
		// - blocked1: open, blocked by ready1 (blocked)
		// - parent1: open, has open child (blocked)
		// - child1: open, child of parent1, no blockers (ready)
		ready1 := task.NewTask("tick-aaaaaa", "Ready one")
		ready2 := task.NewTask("tick-bbbbbb", "Ready two")

		blocked1 := task.NewTask("tick-cccccc", "Blocked one")
		blocked1.BlockedBy = []string{"tick-aaaaaa"}

		parent1 := task.NewTask("tick-dddddd", "Parent")
		child1 := task.NewTask("tick-eeeeee", "Child")
		child1.Parent = "tick-dddddd"

		dir := initTickProjectWithTasks(t, []task.Task{ready1, ready2, blocked1, parent1, child1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
		}

		workflow, ok := parsed["workflow"].(map[string]interface{})
		if !ok {
			t.Fatalf("workflow is not an object: %T", parsed["workflow"])
		}

		// Ready: ready1, ready2, child1 = 3
		if workflow["ready"] != float64(3) {
			t.Errorf("expected workflow.ready 3, got %v", workflow["ready"])
		}
		// Blocked: blocked1, parent1 = 2
		if workflow["blocked"] != float64(2) {
			t.Errorf("expected workflow.blocked 2, got %v", workflow["blocked"])
		}
	})

	t.Run("it includes all 5 priority levels even at zero", func(t *testing.T) {
		// Only create priority 2 tasks, others should be 0.
		t1 := task.NewTask("tick-aaaaaa", "P2 task")
		t1.Priority = 2

		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
		}

		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority is not an array: %T", parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(byPriority))
		}

		for i, entry := range byPriority {
			obj, ok := entry.(map[string]interface{})
			if !ok {
				t.Fatalf("by_priority[%d] is not an object", i)
			}
			if obj["priority"] != float64(i) {
				t.Errorf("by_priority[%d].priority = %v, want %d", i, obj["priority"], i)
			}
			expectedCount := float64(0)
			if i == 2 {
				expectedCount = 1
			}
			if obj["count"] != expectedCount {
				t.Errorf("by_priority[%d].count = %v, want %v", i, obj["count"], expectedCount)
			}
		}
	})

	t.Run("it returns all zeros for empty project", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
		}

		if parsed["total"] != float64(0) {
			t.Errorf("expected total 0, got %v", parsed["total"])
		}

		byStatus, ok := parsed["by_status"].(map[string]interface{})
		if !ok {
			t.Fatalf("by_status is not an object: %T", parsed["by_status"])
		}
		for _, key := range []string{"open", "in_progress", "done", "cancelled"} {
			if byStatus[key] != float64(0) {
				t.Errorf("expected by_status.%s 0, got %v", key, byStatus[key])
			}
		}

		workflow, ok := parsed["workflow"].(map[string]interface{})
		if !ok {
			t.Fatalf("workflow is not an object: %T", parsed["workflow"])
		}
		if workflow["ready"] != float64(0) {
			t.Errorf("expected workflow.ready 0, got %v", workflow["ready"])
		}
		if workflow["blocked"] != float64(0) {
			t.Errorf("expected workflow.blocked 0, got %v", workflow["blocked"])
		}

		byPriority, ok := parsed["by_priority"].([]interface{})
		if !ok {
			t.Fatalf("by_priority is not an array: %T", parsed["by_priority"])
		}
		if len(byPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(byPriority))
		}
		for i, entry := range byPriority {
			obj := entry.(map[string]interface{})
			if obj["count"] != float64(0) {
				t.Errorf("by_priority[%d].count = %v, want 0", i, obj["count"])
			}
		}
	})

	t.Run("it formats stats in TOON format", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Open task")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("expected TOON stats header, got %q", output)
		}
		if !strings.Contains(output, "by_priority[5]{priority,count}:") {
			t.Errorf("expected TOON by_priority header, got %q", output)
		}
	})

	t.Run("it formats stats in Pretty format with right-aligned numbers", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Open task")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Total:") {
			t.Errorf("expected Total label, got %q", output)
		}
		if !strings.Contains(output, "Status:") {
			t.Errorf("expected Status group, got %q", output)
		}
		if !strings.Contains(output, "Workflow:") {
			t.Errorf("expected Workflow group, got %q", output)
		}
		if !strings.Contains(output, "Priority:") {
			t.Errorf("expected Priority group, got %q", output)
		}
		if !strings.Contains(output, "P0 (critical):") {
			t.Errorf("expected P0 label, got %q", output)
		}
	})

	t.Run("it formats stats in JSON format with nested structure", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Open task")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
		}

		// Check nested structure exists.
		if _, ok := parsed["total"]; !ok {
			t.Error("expected 'total' key")
		}
		if _, ok := parsed["by_status"]; !ok {
			t.Error("expected 'by_status' key")
		}
		if _, ok := parsed["workflow"]; !ok {
			t.Error("expected 'workflow' key")
		}
		if _, ok := parsed["by_priority"]; !ok {
			t.Error("expected 'by_priority' key")
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Open task")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "stats"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet, got %q", stdout.String())
		}
	})
}
