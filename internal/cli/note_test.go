package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runNote runs a tick note command and returns stdout, stderr, and exit code.
// Uses IsTTY=true to default to PrettyFormatter for consistent test output.
func runNote(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "note"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestNoteAdd(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it adds a note to a task", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runNote(t, dir, "add", "tick-aaa111", "Started investigating")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.Notes) != 1 {
			t.Fatalf("notes count = %d, want 1", len(found.Notes))
		}
		if found.Notes[0].Text != "Started investigating" {
			t.Errorf("note text = %q, want %q", found.Notes[0].Text, "Started investigating")
		}
	})

	t.Run("it collects text from multiple remaining args", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runNote(t, dir, "add", "tick-aaa111", "Started", "investigating", "the", "issue")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.Notes) != 1 {
			t.Fatalf("notes count = %d, want 1", len(found.Notes))
		}
		if found.Notes[0].Text != "Started investigating the issue" {
			t.Errorf("note text = %q, want %q", found.Notes[0].Text, "Started investigating the issue")
		}
	})

	t.Run("it errors when id is missing", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runNote(t, dir, "add")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("stderr should contain 'Usage:', got %q", stderr)
		}
	})

	t.Run("it errors when text is missing", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runNote(t, dir, "add", "tick-aaa111")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "note text is required") {
			t.Errorf("stderr should contain 'note text is required', got %q", stderr)
		}
	})

	t.Run("it errors when text exceeds 500 chars", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		longText := strings.Repeat("a", 501)
		_, stderr, exitCode := runNote(t, dir, "add", "tick-aaa111", longText)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "exceeds maximum length") {
			t.Errorf("stderr should contain 'exceeds maximum length', got %q", stderr)
		}
	})

	t.Run("it errors when task not found", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runNote(t, dir, "add", "tick-nonexist", "some note")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it resolves partial ID for note add", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-a3f1b2", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runNote(t, dir, "add", "a3f", "A partial ID note")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-a3f1b2" {
				found = tk
				break
			}
		}
		if len(found.Notes) != 1 {
			t.Fatalf("notes count = %d, want 1", len(found.Notes))
		}
		if found.Notes[0].Text != "A partial ID note" {
			t.Errorf("note text = %q, want %q", found.Notes[0].Text, "A partial ID note")
		}
	})

	t.Run("it updates task Updated timestamp", func(t *testing.T) {
		pastTime := now.Add(-1 * time.Hour)
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: pastTime, Updated: pastTime,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		before := time.Now().UTC().Truncate(time.Second)
		_, _, exitCode := runNote(t, dir, "add", "tick-aaa111", "Updating timestamp")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if found.Updated.Before(before) || found.Updated.After(after) {
			t.Errorf("updated = %v, want between %v and %v", found.Updated, before, after)
		}
	})

	t.Run("it appends note with current timestamp", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		before := time.Now().UTC().Truncate(time.Second)
		_, _, exitCode := runNote(t, dir, "add", "tick-aaa111", "Timestamped note")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		tasks := readPersistedTasks(t, tickDir)
		var found task.Task
		for _, tk := range tasks {
			if tk.ID == "tick-aaa111" {
				found = tk
				break
			}
		}
		if len(found.Notes) != 1 {
			t.Fatalf("notes count = %d, want 1", len(found.Notes))
		}
		noteCreated := found.Notes[0].Created
		if noteCreated.Before(before) || noteCreated.After(after) {
			t.Errorf("note created = %v, want between %v and %v", noteCreated, before, after)
		}
	})
}

func TestNoteNoSubcommand(t *testing.T) {
	t.Run("it errors when no sub-subcommand given", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runNote(t, dir)
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("stderr should contain usage hint, got %q", stderr)
		}
	})
}
