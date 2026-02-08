package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestStats_CountsTasksByStatusCorrectly(t *testing.T) {
	t.Run("it counts tasks by status correctly", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Open task 2", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb111", Title: "In progress task", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ccc111", Title: "Done task 1", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-ccc222", Title: "Done task 2", Status: task.StatusDone, Priority: 3, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-ccc333", Title: "Done task 3", Status: task.StatusDone, Priority: 0, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-ddd111", Title: "Cancelled task", Status: task.StatusCancelled, Priority: 4, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var result jsonStats
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\noutput: %s", err, stdout.String())
		}

		if result.Total != 7 {
			t.Errorf("expected total 7, got %d", result.Total)
		}
		if result.ByStatus.Open != 2 {
			t.Errorf("expected open 2, got %d", result.ByStatus.Open)
		}
		if result.ByStatus.InProgress != 1 {
			t.Errorf("expected in_progress 1, got %d", result.ByStatus.InProgress)
		}
		if result.ByStatus.Done != 3 {
			t.Errorf("expected done 3, got %d", result.ByStatus.Done)
		}
		if result.ByStatus.Cancelled != 1 {
			t.Errorf("expected cancelled 1, got %d", result.ByStatus.Cancelled)
		}
	})
}

func TestStats_CountsReadyAndBlockedCorrectly(t *testing.T) {
	t.Run("it counts ready and blocked tasks correctly", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			// Ready: open, no blockers, no open children
			{ID: "tick-rdy111", Title: "Ready leaf 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-rdy222", Title: "Ready leaf 2", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			// Blocked: open, has open blocker
			{ID: "tick-blk111", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), BlockedBy: []string{"tick-rdy111"}},
			// Blocked: parent with open child
			{ID: "tick-par111", Title: "Parent blocked", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-chi111", Title: "Open child", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute), Parent: "tick-par111"},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var result jsonStats
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\noutput: %s", err, stdout.String())
		}

		// Ready: tick-rdy111 is a blocker but still ready itself (no blockers, no children);
		// tick-rdy222 is ready; tick-chi111 is ready (open leaf, no blockers)
		// Blocked: tick-blk111 (blocked by open dep); tick-par111 (has open child)
		if result.Workflow.Ready != 3 {
			t.Errorf("expected ready 3, got %d", result.Workflow.Ready)
		}
		if result.Workflow.Blocked != 2 {
			t.Errorf("expected blocked 2, got %d", result.Workflow.Blocked)
		}
	})
}

func TestStats_IncludesAll5PriorityLevelsEvenAtZero(t *testing.T) {
	t.Run("it includes all 5 priority levels even at zero", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			// Only priority 2 tasks
			{ID: "tick-aaa111", Title: "Task P2", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Task P2 again", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var result jsonStats
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\noutput: %s", err, stdout.String())
		}

		if len(result.ByPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(result.ByPriority))
		}

		for i, entry := range result.ByPriority {
			if entry.Priority != i {
				t.Errorf("priority[%d]: expected priority %d, got %d", i, i, entry.Priority)
			}
		}

		// P0=0, P1=0, P2=2, P3=0, P4=0
		expected := []int{0, 0, 2, 0, 0}
		for i, entry := range result.ByPriority {
			if entry.Count != expected[i] {
				t.Errorf("priority[%d]: expected count %d, got %d", i, expected[i], entry.Count)
			}
		}
	})
}

func TestStats_ReturnsAllZerosForEmptyProject(t *testing.T) {
	t.Run("it returns all zeros for empty project", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var result jsonStats
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\noutput: %s", err, stdout.String())
		}

		if result.Total != 0 {
			t.Errorf("expected total 0, got %d", result.Total)
		}
		if result.ByStatus.Open != 0 {
			t.Errorf("expected open 0, got %d", result.ByStatus.Open)
		}
		if result.ByStatus.InProgress != 0 {
			t.Errorf("expected in_progress 0, got %d", result.ByStatus.InProgress)
		}
		if result.ByStatus.Done != 0 {
			t.Errorf("expected done 0, got %d", result.ByStatus.Done)
		}
		if result.ByStatus.Cancelled != 0 {
			t.Errorf("expected cancelled 0, got %d", result.ByStatus.Cancelled)
		}
		if result.Workflow.Ready != 0 {
			t.Errorf("expected ready 0, got %d", result.Workflow.Ready)
		}
		if result.Workflow.Blocked != 0 {
			t.Errorf("expected blocked 0, got %d", result.Workflow.Blocked)
		}
		if len(result.ByPriority) != 5 {
			t.Fatalf("expected 5 priority entries, got %d", len(result.ByPriority))
		}
		for i, entry := range result.ByPriority {
			if entry.Count != 0 {
				t.Errorf("priority[%d]: expected count 0, got %d", i, entry.Count)
			}
		}
	})
}

func TestStats_FormatsToonFormat(t *testing.T) {
	t.Run("it formats stats in TOON format", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--toon", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()

		// Check stats section header
		if !strings.Contains(output, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
			t.Errorf("missing stats header in TOON output, got:\n%s", output)
		}

		// Check stats data row: total=2, open=1, in_progress=0, done=1, cancelled=0, ready=1, blocked=0
		if !strings.Contains(output, "  2,1,0,1,0,1,0") {
			t.Errorf("missing stats data row in TOON output, got:\n%s", output)
		}

		// Check by_priority section header
		if !strings.Contains(output, "by_priority[5]{priority,count}:") {
			t.Errorf("missing by_priority header in TOON output, got:\n%s", output)
		}

		// Check all 5 priority rows present
		if !strings.Contains(output, "  0,0\n") {
			t.Errorf("missing P0 row in TOON output, got:\n%s", output)
		}
		if !strings.Contains(output, "  1,1\n") {
			t.Errorf("missing P1 row in TOON output, got:\n%s", output)
		}
		if !strings.Contains(output, "  2,1\n") {
			t.Errorf("missing P2 row in TOON output, got:\n%s", output)
		}
	})
}

func TestStats_FormatsPrettyFormatWithRightAlignedNumbers(t *testing.T) {
	t.Run("it formats stats in Pretty format with right-aligned numbers", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open 1", Status: task.StatusOpen, Priority: 0, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Open 2", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb111", Title: "In progress", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ccc111", Title: "Done", Status: task.StatusDone, Priority: 3, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--pretty", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()

		// Check Total line
		if !strings.Contains(output, "Total:") {
			t.Errorf("missing Total label in Pretty output, got:\n%s", output)
		}

		// Check Status section
		if !strings.Contains(output, "Status:") {
			t.Errorf("missing Status section in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "Open:") {
			t.Errorf("missing Open label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "In Progress:") {
			t.Errorf("missing In Progress label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "Done:") {
			t.Errorf("missing Done label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "Cancelled:") {
			t.Errorf("missing Cancelled label in Pretty output, got:\n%s", output)
		}

		// Check Workflow section
		if !strings.Contains(output, "Workflow:") {
			t.Errorf("missing Workflow section in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "Ready:") {
			t.Errorf("missing Ready label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "Blocked:") {
			t.Errorf("missing Blocked label in Pretty output, got:\n%s", output)
		}

		// Check Priority section with P0-P4 labels
		if !strings.Contains(output, "Priority:") {
			t.Errorf("missing Priority section in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "P0 (critical):") {
			t.Errorf("missing P0 label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "P1 (high):") {
			t.Errorf("missing P1 label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "P2 (medium):") {
			t.Errorf("missing P2 label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "P3 (low):") {
			t.Errorf("missing P3 label in Pretty output, got:\n%s", output)
		}
		if !strings.Contains(output, "P4 (backlog):") {
			t.Errorf("missing P4 label in Pretty output, got:\n%s", output)
		}
	})
}

func TestStats_FormatsJSONFormatWithNestedStructure(t *testing.T) {
	t.Run("it formats stats in JSON format with nested structure", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := now.Add(time.Hour)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb111", Title: "Done task", Status: task.StatusDone, Priority: 2, Created: now, Updated: now, Closed: &closed},
			{ID: "tick-ccc111", Title: "Cancelled task", Status: task.StatusCancelled, Priority: 4, Created: now, Updated: now, Closed: &closed},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var result jsonStats
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\noutput: %s", err, stdout.String())
		}

		// Verify nested structure
		if result.Total != 3 {
			t.Errorf("expected total 3, got %d", result.Total)
		}
		if result.ByStatus.Open != 1 {
			t.Errorf("expected by_status.open 1, got %d", result.ByStatus.Open)
		}
		if result.ByStatus.Done != 1 {
			t.Errorf("expected by_status.done 1, got %d", result.ByStatus.Done)
		}
		if result.ByStatus.Cancelled != 1 {
			t.Errorf("expected by_status.cancelled 1, got %d", result.ByStatus.Cancelled)
		}
		if result.Workflow.Ready != 1 {
			t.Errorf("expected workflow.ready 1, got %d", result.Workflow.Ready)
		}
		if result.Workflow.Blocked != 0 {
			t.Errorf("expected workflow.blocked 0, got %d", result.Workflow.Blocked)
		}
		if len(result.ByPriority) != 5 {
			t.Fatalf("expected 5 by_priority entries, got %d", len(result.ByPriority))
		}

		// Verify JSON has correct keys by checking raw output
		rawOutput := stdout.String()
		if !strings.Contains(rawOutput, `"total"`) {
			t.Errorf("missing 'total' key in JSON output")
		}
		if !strings.Contains(rawOutput, `"by_status"`) {
			t.Errorf("missing 'by_status' key in JSON output")
		}
		if !strings.Contains(rawOutput, `"workflow"`) {
			t.Errorf("missing 'workflow' key in JSON output")
		}
		if !strings.Contains(rawOutput, `"by_priority"`) {
			t.Errorf("missing 'by_priority' key in JSON output")
		}
	})
}

func TestStats_SuppressesOutputWithQuiet(t *testing.T) {
	t.Run("it suppresses output with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "stats"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if output != "" {
			t.Errorf("expected no output with --quiet, got %q", output)
		}
	})
}
