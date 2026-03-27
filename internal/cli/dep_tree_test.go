package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestDepTreeWiring(t *testing.T) {
	t.Run("it qualifies dep tree as a two-level command", func(t *testing.T) {
		cmd, rest := qualifyCommand("dep", []string{"tree", "tick-abc123"})
		if cmd != "dep tree" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep tree")
		}
		if len(rest) != 1 || rest[0] != "tick-abc123" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-abc123]", rest)
		}
	})

	t.Run("it qualifies dep tree with no args", func(t *testing.T) {
		cmd, rest := qualifyCommand("dep", []string{"tree"})
		if cmd != "dep tree" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep tree")
		}
		if len(rest) != 0 {
			t.Errorf("qualifyCommand returned rest = %v, want empty", rest)
		}
	})

	t.Run("it rejects unknown flag on dep tree", func(t *testing.T) {
		err := ValidateFlags("dep tree", []string{"--unknown"}, commandFlags)
		if err == nil {
			t.Fatal("expected error for --unknown on dep tree, got nil")
		}
		want := `unknown flag "--unknown" for "dep tree". Run 'tick help dep' for usage.`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it accepts global flags on dep tree", func(t *testing.T) {
		err := ValidateFlags("dep tree", []string{"--quiet"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil for global flag on dep tree, got %v", err)
		}
	})

	t.Run("it dispatches dep tree without error", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "dep", "tree"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
	})

	t.Run("it shows tree in dep help text", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "dep")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "tree") {
			t.Errorf("dep help should mention 'tree', got %q", stdout)
		}
	})

	t.Run("it does not break existing dep add/remove dispatch", func(t *testing.T) {
		// Verify qualifyCommand still works for add and remove
		cmd, rest := qualifyCommand("dep", []string{"add", "tick-aaa", "tick-bbb"})
		if cmd != "dep add" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep add")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "tick-bbb" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa tick-bbb]", rest)
		}

		cmd, rest = qualifyCommand("dep", []string{"remove", "tick-aaa", "tick-bbb"})
		if cmd != "dep remove" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep remove")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "tick-bbb" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa tick-bbb]", rest)
		}
	})
}

// runDepTree runs the tick dep tree command with the given args and returns stdout, stderr, and exit code.
func runDepTree(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	fullArgs := append([]string{"tick", "--pretty", "dep", "tree"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestRunDepTree(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	t.Run("it outputs no dependencies found for empty project", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := strings.TrimSpace(stdout)
		if output != "No dependencies found." {
			t.Errorf("output = %q, want %q", output, "No dependencies found.")
		}
	})

	t.Run("it outputs no dependencies found for project with no tasks", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := strings.TrimSpace(stdout)
		if output != "No dependencies found." {
			t.Errorf("output = %q, want %q", output, "No dependencies found.")
		}
	})

	t.Run("it outputs dep tree for project with dependencies", func(t *testing.T) {
		// A blocks B blocks C: chain A -> B -> C
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusInProgress, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ccc333", Title: "Task C", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := strings.TrimSpace(stdout)
		// Handler should NOT take the "no dependencies" path when deps exist.
		// FormatDepTree is currently a stub returning empty, so output will be empty,
		// but it must not be the "No dependencies found." message.
		if output == "No dependencies found." {
			t.Error("output should not be the 'no dependencies' message when dependencies exist")
		}
	})

	t.Run("it outputs focused view for task with dependencies", func(t *testing.T) {
		// A -> B -> C, focus on B (has both upstream and downstream)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusInProgress, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ccc333", Title: "Task C", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runDepTree(t, dir, "tick-bbb222")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := strings.TrimSpace(stdout)
		// Focused view with dependencies should NOT show the "No dependencies." message
		if output == "No dependencies." {
			t.Error("output should not be 'No dependencies.' for task with deps")
		}
		// Should not contain the isolated-task format (ID + title + status + no deps message)
		if strings.Contains(output, "No dependencies.") {
			t.Error("output should not contain 'No dependencies.' for task with deps")
		}
	})

	t.Run("it outputs no dependencies for isolated task in focused mode", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runDepTree(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := stdout
		// Should show the task itself: ID, title, and status
		if !strings.Contains(output, "tick-aaa111") {
			t.Error("output should contain task ID")
		}
		if !strings.Contains(output, "Task A") {
			t.Error("output should contain task title")
		}
		if !strings.Contains(output, "open") {
			t.Error("output should contain task status")
		}
		// Should show "No dependencies." message
		if !strings.Contains(output, "No dependencies.") {
			t.Errorf("output should contain 'No dependencies.', got %q", output)
		}
	})

	t.Run("it returns error for nonexistent task ID", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runDepTree(t, dir, "tick-zzz999")
		if exitCode == 0 {
			t.Fatal("expected non-zero exit code for nonexistent task ID")
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it resolves partial task ID", func(t *testing.T) {
		// Isolated task — focused mode with no deps shows task info directly
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Use partial ID "aaa111" (6 hex chars without tick- prefix) — should resolve to tick-aaa111
		stdout, stderr, exitCode := runDepTree(t, dir, "aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := stdout
		// Isolated task outputs task ID + title + status directly via handler
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("partial ID should resolve to tick-aaa111; output = %q", output)
		}
		if !strings.Contains(output, "Task A") {
			t.Errorf("output should contain task title; output = %q", output)
		}
	})

	t.Run("it suppresses output in quiet mode", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runDepTree(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if stdout != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout)
		}
	})

	t.Run("it returns error for ambiguous partial ID", func(t *testing.T) {
		// Two tasks with IDs that share a prefix: tick-aaa111 and tick-aaa222
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Task A2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runDepTree(t, dir, "aaa")
		if exitCode == 0 {
			t.Fatal("expected non-zero exit code for ambiguous partial ID")
		}
		if !strings.Contains(stderr, "ambiguous") {
			t.Errorf("stderr should contain 'ambiguous', got %q", stderr)
		}
	})

	t.Run("it handles focused view via full App.Run dispatch", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "dep", "tree", "tick-aaa111"})
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}
	})
}
