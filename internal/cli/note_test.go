package cli

import (
	"bytes"
	"encoding/json"
	"slices"
	"strconv"
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

	t.Run("it errors when text exceeds 2000 chars", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		longText := strings.Repeat("a", 2001)
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

func TestNoteRemove(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it removes note at index 1", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
				{Text: "Second note", Created: now},
			},
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runNote(t, dir, "remove", "tick-aaa111", "1")
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
		if found.Notes[0].Text != "Second note" {
			t.Errorf("remaining note text = %q, want %q", found.Notes[0].Text, "Second note")
		}
	})

	t.Run("it removes note at last index", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
				{Text: "Second note", Created: now},
				{Text: "Third note", Created: now},
			},
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runNote(t, dir, "remove", "tick-aaa111", "3")
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
		if len(found.Notes) != 2 {
			t.Fatalf("notes count = %d, want 2", len(found.Notes))
		}
		if found.Notes[0].Text != "First note" {
			t.Errorf("notes[0] text = %q, want %q", found.Notes[0].Text, "First note")
		}
		if found.Notes[1].Text != "Second note" {
			t.Errorf("notes[1] text = %q, want %q", found.Notes[1].Text, "Second note")
		}
	})

	t.Run("it removes middle note preserving order", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
				{Text: "Second note", Created: now},
				{Text: "Third note", Created: now},
			},
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runNote(t, dir, "remove", "tick-aaa111", "2")
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
		if len(found.Notes) != 2 {
			t.Fatalf("notes count = %d, want 2", len(found.Notes))
		}
		if found.Notes[0].Text != "First note" {
			t.Errorf("notes[0] text = %q, want %q", found.Notes[0].Text, "First note")
		}
		if found.Notes[1].Text != "Third note" {
			t.Errorf("notes[1] text = %q, want %q", found.Notes[1].Text, "Third note")
		}
	})

	t.Run("it errors on index 0", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
			},
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runNote(t, dir, "remove", "tick-aaa111", "0")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "must be >= 1") {
			t.Errorf("stderr should contain 'must be >= 1', got %q", stderr)
		}
	})

	t.Run("it errors on negative index", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
			},
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runNote(t, dir, "remove", "tick-aaa111", "-1")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "must be >= 1") {
			t.Errorf("stderr should contain 'must be >= 1', got %q", stderr)
		}
	})

	t.Run("it errors when index exceeds note count", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
				{Text: "Second note", Created: now},
			},
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runNote(t, dir, "remove", "tick-aaa111", "3")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "out of range") {
			t.Errorf("stderr should contain 'out of range', got %q", stderr)
		}
		if !strings.Contains(stderr, "has 2 note(s)") {
			t.Errorf("stderr should contain 'has 2 note(s)', got %q", stderr)
		}
	})

	t.Run("it errors on non-integer index", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
			},
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runNote(t, dir, "remove", "tick-aaa111", "abc")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "invalid index") {
			t.Errorf("stderr should contain 'invalid index', got %q", stderr)
		}
	})

	t.Run("it errors when task has no notes", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, stderr, exitCode := runNote(t, dir, "remove", "tick-aaa111", "1")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "task has no notes to remove") {
			t.Errorf("stderr should contain 'task has no notes to remove', got %q", stderr)
		}
	})

	t.Run("it errors when task not found", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runNote(t, dir, "remove", "tick-nonexist", "1")
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it resolves partial ID for note remove", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-a3f1b2", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "First note", Created: now},
				{Text: "Second note", Created: now},
			},
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		_, _, exitCode := runNote(t, dir, "remove", "a3f", "1")
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
		if found.Notes[0].Text != "Second note" {
			t.Errorf("remaining note text = %q, want %q", found.Notes[0].Text, "Second note")
		}
	})

	t.Run("it updates task Updated timestamp on remove", func(t *testing.T) {
		pastTime := now.Add(-1 * time.Hour)
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: pastTime, Updated: pastTime,
			Notes: []task.Note{
				{Text: "First note", Created: pastTime},
			},
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		before := time.Now().UTC().Truncate(time.Second)
		_, _, exitCode := runNote(t, dir, "remove", "tick-aaa111", "1")
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

func TestNoteTreeRejection(t *testing.T) {
	t.Run("it rejects tree as unknown note sub-command", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
			IsTTY:  true,
		}
		exitCode := app.Run([]string{"tick", "note", "tree"})
		if exitCode == 0 {
			t.Fatal("expected non-zero exit code for tick note tree")
		}
		stderr := stderrBuf.String()
		if !strings.Contains(stderr, "unknown note sub-command 'tree'") {
			t.Errorf("stderr should contain \"unknown note sub-command 'tree'\", got %q", stderr)
		}
	})

	t.Run("it does not reference note tree in flag error", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
			IsTTY:  true,
		}
		exitCode := app.Run([]string{"tick", "note", "tree", "--foo"})
		if exitCode == 0 {
			t.Fatal("expected non-zero exit code for tick note tree --foo")
		}
		stderr := stderrBuf.String()
		if !strings.Contains(stderr, "note") {
			t.Errorf("stderr should reference \"note\", got %q", stderr)
		}
		if strings.Contains(stderr, "note tree") {
			t.Errorf("stderr must not reference \"note tree\", got %q", stderr)
		}
	})
}

func TestNoteIndex(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("it matches the index note remove accepts", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA})

		for _, text := range []string{"First note", "Second note", "Third note"} {
			if _, _, code := runNote(t, dir, "add", "tick-aaa111", text); code != 0 {
				t.Fatalf("note add %q exit code = %d, want 0", text, code)
			}
		}

		stdout, _, code := runShow(t, dir, "tick-aaa111", "--toon")
		if code != 0 {
			t.Fatalf("show exit code = %d, want 0", code)
		}
		notes := decodeToonNotes(t, stdout)
		if len(notes) != 3 {
			t.Fatalf("notes length = %d, want 3", len(notes))
		}
		var secondIndex float64
		for _, note := range notes {
			if note["text"] == "Second note" {
				secondIndex, _ = note["index"].(float64)
			}
		}
		if secondIndex == 0 {
			t.Fatalf("second note not found in %#v", notes)
		}

		if _, _, code := runNote(t, dir, "remove", "tick-aaa111", strconv.Itoa(int(secondIndex))); code != 0 {
			t.Fatalf("note remove exit code = %d, want 0", code)
		}

		remaining := notesTextsOf(t, readPersistedTasks(t, tickDir), "tick-aaa111")
		want := []string{"First note", "Third note"}
		if !slices.Equal(remaining, want) {
			t.Errorf("remaining notes = %#v, want %#v", remaining, want)
		}
	})

	t.Run("it keeps insertion order for notes sharing a created timestamp", func(t *testing.T) {
		taskA := task.Task{
			ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
			Notes: []task.Note{
				{Text: "Zulu first", Created: now},
				{Text: "Alpha second", Created: now},
			},
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA})

		stdout, _, code := runShow(t, dir, "tick-aaa111", "--toon")
		if code != 0 {
			t.Fatalf("show exit code = %d, want 0", code)
		}
		notes := decodeToonNotes(t, stdout)
		if len(notes) != 2 {
			t.Fatalf("notes length = %d, want 2", len(notes))
		}
		wantRows := []struct {
			index float64
			text  string
		}{{1, "Zulu first"}, {2, "Alpha second"}}
		for i, want := range wantRows {
			if notes[i]["index"] != want.index || notes[i]["text"] != want.text {
				t.Errorf("notes[%d] = %#v, want index %v text %q", i, notes[i], want.index, want.text)
			}
		}
	})
}

// notesTextsOf returns the note texts of the named task, in stored order.
func notesTextsOf(t *testing.T, tasks []task.Task, id string) []string {
	t.Helper()
	for _, tk := range tasks {
		if tk.ID != id {
			continue
		}
		texts := make([]string, 0, len(tk.Notes))
		for _, n := range tk.Notes {
			texts = append(texts, n.Text)
		}
		return texts
	}
	t.Fatalf("task %q not found", id)
	return nil
}

func TestNoteAddFreeText(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	existingTask := task.Task{
		ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
		Priority: 2, Created: now, Updated: now,
	}

	addedNoteText := func(t *testing.T, tickDir string) string {
		t.Helper()
		tasks := readPersistedTasks(t, tickDir)
		for _, tk := range tasks {
			if tk.ID == existingTask.ID {
				if len(tk.Notes) != 1 {
					t.Fatalf("notes count = %d, want 1", len(tk.Notes))
				}
				return tk.Notes[0].Text
			}
		}
		t.Fatalf("task %s not found", existingTask.ID)
		return ""
	}

	t.Run("it accepts dash-leading note text without the marker", func(t *testing.T) {
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runNote(t, dir, "add", existingTask.ID, "- read the header")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if got := addedNoteText(t, tickDir); got != "- read the header" {
			t.Errorf("note text = %q, want %q", got, "- read the header")
		}
	})

	t.Run("it accepts note text beginning with two dashes", func(t *testing.T) {
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runNote(t, dir, "add", existingTask.ID, "--not-a-flag")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if got := addedNoteText(t, tickDir); got != "--not-a-flag" {
			t.Errorf("note text = %q, want %q", got, "--not-a-flag")
		}
	})

	t.Run("it still rejects a flag before the task id", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runNote(t, dir, "add", "--bogus", existingTask.ID, "text")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
		want := "Error: unknown flag \"--bogus\" for \"note add\". Run 'tick help note' for usage.\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})

	t.Run("it still resolves a global flag after the text", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		stdout, stderr, exitCode := runNote(t, dir, "add", existingTask.ID, "text", "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("stdout is not JSON: %v (stdout = %q)", err, stdout)
		}
	})

	t.Run("it needs the marker for text that spells a global flag", func(t *testing.T) {
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runNote(t, dir, "add", existingTask.ID, "--", "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if got := addedNoteText(t, tickDir); got != "--json" {
			t.Errorf("note text = %q, want %q", got, "--json")
		}
	})

	t.Run("it keeps full validation on note remove", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runNote(t, dir, "remove", existingTask.ID, "--bogus", "1")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
		want := "Error: unknown flag \"--bogus\" for \"note remove\". Run 'tick help note' for usage.\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})
}
