package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStats(t *testing.T) {
	t.Run("it counts tasks by status correctly", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Open one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Open two", "open", 1, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-ccc333", "In progress", "in_progress", 1, nil, "", "2026-01-19T12:00:00Z"),
			taskJSONL("tick-ddd444", "Done one", "done", 2, nil, "", "2026-01-19T13:00:00Z"),
			taskJSONL("tick-eee555", "Done two", "done", 3, nil, "", "2026-01-19T14:00:00Z"),
			taskJSONL("tick-fff666", "Done three", "done", 0, nil, "", "2026-01-19T15:00:00Z"),
			taskJSONL("tick-ggg777", "Cancelled", "cancelled", 4, nil, "", "2026-01-19T16:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--json", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}

		total := int(result["total"].(float64))
		if total != 7 {
			t.Errorf("total = %d, want 7", total)
		}

		byStatus := result["by_status"].(map[string]interface{})
		if int(byStatus["open"].(float64)) != 2 {
			t.Errorf("open = %v, want 2", byStatus["open"])
		}
		if int(byStatus["in_progress"].(float64)) != 1 {
			t.Errorf("in_progress = %v, want 1", byStatus["in_progress"])
		}
		if int(byStatus["done"].(float64)) != 3 {
			t.Errorf("done = %v, want 3", byStatus["done"])
		}
		if int(byStatus["cancelled"].(float64)) != 1 {
			t.Errorf("cancelled = %v, want 1", byStatus["cancelled"])
		}
	})

	t.Run("it counts ready and blocked tasks correctly", func(t *testing.T) {
		// tick-ready1: open, no blockers, no children -> ready
		// tick-ready2: open, no blockers, no children -> ready
		// tick-blk01: open, blocked by tick-ready1 (open) -> blocked
		// tick-parent: open, has open child tick-child1 -> blocked
		// tick-child1: open, no blockers, no children -> ready
		content := strings.Join([]string{
			taskJSONL("tick-ready1", "Ready one", "open", 1, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-ready2", "Ready two", "open", 2, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-blk01", "Blocked", "open", 2, []string{"tick-ready1"}, "", "2026-01-19T12:00:00Z"),
			taskJSONL("tick-parent", "Parent", "open", 2, nil, "", "2026-01-19T13:00:00Z"),
			taskJSONL("tick-child1", "Child", "open", 2, nil, "tick-parent", "2026-01-19T14:00:00Z"),
			taskJSONL("tick-inpr01", "In progress", "in_progress", 1, nil, "", "2026-01-19T15:00:00Z"),
			taskJSONL("tick-done01", "Done", "done", 2, nil, "", "2026-01-19T16:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--json", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}

		workflow := result["workflow"].(map[string]interface{})
		ready := int(workflow["ready"].(float64))
		blocked := int(workflow["blocked"].(float64))

		// Ready: tick-ready1, tick-ready2, tick-child1 = 3
		if ready != 3 {
			t.Errorf("ready = %d, want 3", ready)
		}
		// Blocked: tick-blk01 (has open blocker), tick-parent (has open child) = 2
		if blocked != 2 {
			t.Errorf("blocked = %d, want 2", blocked)
		}
	})

	t.Run("it includes all 5 priority levels even at zero", func(t *testing.T) {
		// Only priority 2 tasks exist; P0, P1, P3, P4 should be 0
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Task two", "open", 2, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--json", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}

		byPriority := result["by_priority"].([]interface{})
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}

		for _, entry := range byPriority {
			e := entry.(map[string]interface{})
			pri := int(e["priority"].(float64))
			count := int(e["count"].(float64))
			if pri == 2 {
				if count != 2 {
					t.Errorf("priority %d count = %d, want 2", pri, count)
				}
			} else {
				if count != 0 {
					t.Errorf("priority %d count = %d, want 0", pri, count)
				}
			}
		}
	})

	t.Run("it returns all zeros for empty project", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--json", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}

		if int(result["total"].(float64)) != 0 {
			t.Errorf("total = %v, want 0", result["total"])
		}

		byStatus := result["by_status"].(map[string]interface{})
		if int(byStatus["open"].(float64)) != 0 {
			t.Errorf("open = %v, want 0", byStatus["open"])
		}
		if int(byStatus["in_progress"].(float64)) != 0 {
			t.Errorf("in_progress = %v, want 0", byStatus["in_progress"])
		}
		if int(byStatus["done"].(float64)) != 0 {
			t.Errorf("done = %v, want 0", byStatus["done"])
		}
		if int(byStatus["cancelled"].(float64)) != 0 {
			t.Errorf("cancelled = %v, want 0", byStatus["cancelled"])
		}

		workflow := result["workflow"].(map[string]interface{})
		if int(workflow["ready"].(float64)) != 0 {
			t.Errorf("ready = %v, want 0", workflow["ready"])
		}
		if int(workflow["blocked"].(float64)) != 0 {
			t.Errorf("blocked = %v, want 0", workflow["blocked"])
		}

		byPriority := result["by_priority"].([]interface{})
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}
		for _, entry := range byPriority {
			e := entry.(map[string]interface{})
			count := int(e["count"].(float64))
			if count != 0 {
				t.Errorf("priority %v count = %d, want 0", e["priority"], count)
			}
		}
	})

	t.Run("it formats stats in TOON format", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Open task", "open", 1, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Done task", "done", 2, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--toon", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		output := stdout.String()

		// Check stats section header
		if !strings.Contains(output, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("output missing stats header, got:\n%s", output)
		}

		// Check stats data line: total=2, open=1, in_progress=0, done=1, cancelled=0, ready=1, blocked=0
		if !strings.Contains(output, "  2,1,0,1,0,1,0") {
			t.Errorf("output missing expected stats data line, got:\n%s", output)
		}

		// Check by_priority section header
		if !strings.Contains(output, "by_priority[5]{priority,count}:") {
			t.Errorf("output missing by_priority header, got:\n%s", output)
		}

		// Check priority lines
		if !strings.Contains(output, "  0,0") {
			t.Errorf("output missing P0 line, got:\n%s", output)
		}
		if !strings.Contains(output, "  1,1") {
			t.Errorf("output missing P1 line, got:\n%s", output)
		}
		if !strings.Contains(output, "  2,1") {
			t.Errorf("output missing P2 line, got:\n%s", output)
		}
		if !strings.Contains(output, "  3,0") {
			t.Errorf("output missing P3 line, got:\n%s", output)
		}
		if !strings.Contains(output, "  4,0") {
			t.Errorf("output missing P4 line, got:\n%s", output)
		}
	})

	t.Run("it formats stats in Pretty format with right-aligned numbers", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Open task", "open", 0, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "In progress", "in_progress", 1, nil, "", "2026-01-19T11:00:00Z"),
			taskJSONL("tick-ccc333", "Done task", "done", 2, nil, "", "2026-01-19T12:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--pretty", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		output := stdout.String()

		// Check all groups present
		if !strings.Contains(output, "Total:") {
			t.Errorf("output missing Total:, got:\n%s", output)
		}
		if !strings.Contains(output, "Status:") {
			t.Errorf("output missing Status:, got:\n%s", output)
		}
		if !strings.Contains(output, "Workflow:") {
			t.Errorf("output missing Workflow:, got:\n%s", output)
		}
		if !strings.Contains(output, "Priority:") {
			t.Errorf("output missing Priority:, got:\n%s", output)
		}

		// Check P0-P4 labels
		if !strings.Contains(output, "P0 (critical)") {
			t.Errorf("output missing P0 (critical), got:\n%s", output)
		}
		if !strings.Contains(output, "P1 (high)") {
			t.Errorf("output missing P1 (high), got:\n%s", output)
		}
		if !strings.Contains(output, "P2 (medium)") {
			t.Errorf("output missing P2 (medium), got:\n%s", output)
		}
		if !strings.Contains(output, "P3 (low)") {
			t.Errorf("output missing P3 (low), got:\n%s", output)
		}
		if !strings.Contains(output, "P4 (backlog)") {
			t.Errorf("output missing P4 (backlog), got:\n%s", output)
		}

		// Check right-aligned numbers by verifying specific lines
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Total:") {
				// Number should be right-aligned (at least 2 chars wide)
				if !strings.Contains(line, " 3") {
					t.Errorf("Total line = %q, want right-aligned 3", line)
				}
			}
		}
	})

	t.Run("it formats stats in JSON format with nested structure", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Open", "open", 0, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Done", "done", 4, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--json", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}

		// Verify nested structure
		if _, ok := result["total"]; !ok {
			t.Error("JSON missing 'total' key")
		}
		if _, ok := result["by_status"]; !ok {
			t.Error("JSON missing 'by_status' key")
		}
		if _, ok := result["workflow"]; !ok {
			t.Error("JSON missing 'workflow' key")
		}
		if _, ok := result["by_priority"]; !ok {
			t.Error("JSON missing 'by_priority' key")
		}

		// Verify by_status nested keys
		byStatus := result["by_status"].(map[string]interface{})
		for _, key := range []string{"open", "in_progress", "done", "cancelled"} {
			if _, ok := byStatus[key]; !ok {
				t.Errorf("by_status missing '%s' key", key)
			}
		}

		// Verify workflow nested keys
		workflow := result["workflow"].(map[string]interface{})
		for _, key := range []string{"ready", "blocked"} {
			if _, ok := workflow[key]; !ok {
				t.Errorf("workflow missing '%s' key", key)
			}
		}

		// Verify by_priority has correct structure
		byPriority := result["by_priority"].([]interface{})
		if len(byPriority) != 5 {
			t.Fatalf("by_priority length = %d, want 5", len(byPriority))
		}
		for i, entry := range byPriority {
			e := entry.(map[string]interface{})
			if _, ok := e["priority"]; !ok {
				t.Errorf("by_priority[%d] missing 'priority' key", i)
			}
			if _, ok := e["count"]; !ok {
				t.Errorf("by_priority[%d] missing 'count' key", i)
			}
			// Verify priority values are 0-4
			pri := int(e["priority"].(float64))
			if pri != i {
				t.Errorf("by_priority[%d].priority = %d, want %d", i, pri, i)
			}
		}

		// Verify actual values
		if int(result["total"].(float64)) != 2 {
			t.Errorf("total = %v, want 2", result["total"])
		}
		if int(byStatus["open"].(float64)) != 1 {
			t.Errorf("open = %v, want 1", byStatus["open"])
		}
		if int(byStatus["done"].(float64)) != 1 {
			t.Errorf("done = %v, want 1", byStatus["done"])
		}
	})

	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		content := strings.Join([]string{
			taskJSONL("tick-aaa111", "Open task", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
			taskJSONL("tick-bbb222", "Done task", "done", 1, nil, "", "2026-01-19T11:00:00Z"),
		}, "\n") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "stats"})
		if err != nil {
			t.Fatalf("stats returned error: %v", err)
		}

		output := stdout.String()
		if output != "" {
			t.Errorf("output = %q, want empty string with --quiet", output)
		}
	})
}
