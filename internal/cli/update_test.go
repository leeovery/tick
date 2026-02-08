package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestUpdate_TitleFlag(t *testing.T) {
	t.Run("it updates title with --title flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Original title", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "Updated title"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "Updated title" {
			t.Errorf("expected title %q, got %q", "Updated title", tasks[0].Title)
		}
	})
}

func TestUpdate_DescriptionFlag(t *testing.T) {
	t.Run("it updates description with --description flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--description", "New description"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Description != "New description" {
			t.Errorf("expected description %q, got %q", "New description", tasks[0].Description)
		}
	})
}

func TestUpdate_ClearDescription(t *testing.T) {
	t.Run("it clears description with --description \"\"", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Description: "Has a desc", Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--description", ""})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Description != "" {
			t.Errorf("expected empty description, got %q", tasks[0].Description)
		}
	})
}

func TestUpdate_PriorityFlag(t *testing.T) {
	t.Run("it updates priority with --priority flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--priority", "0"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if tasks[0].Priority != 0 {
			t.Errorf("expected priority 0, got %d", tasks[0].Priority)
		}
	})
}

func TestUpdate_ParentFlag(t *testing.T) {
	t.Run("it updates parent with --parent flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Child task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-bbb222", "--parent", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var child *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-bbb222" {
				child = &tasks[i]
				break
			}
		}
		if child == nil {
			t.Fatal("could not find child task")
		}
		if child.Parent != "tick-aaa111" {
			t.Errorf("expected parent %q, got %q", "tick-aaa111", child.Parent)
		}
	})
}

func TestUpdate_ClearParent(t *testing.T) {
	t.Run("it clears parent with --parent \"\"", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Child task", Status: task.StatusOpen, Priority: 2, Parent: "tick-aaa111", Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-bbb222", "--parent", ""})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var child *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-bbb222" {
				child = &tasks[i]
				break
			}
		}
		if child == nil {
			t.Fatal("could not find child task")
		}
		if child.Parent != "" {
			t.Errorf("expected empty parent, got %q", child.Parent)
		}
	})
}

func TestUpdate_BlocksFlag(t *testing.T) {
	t.Run("it updates blocks with --blocks flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Blocker task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Target task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--blocks", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var target *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-bbb222" {
				target = &tasks[i]
				break
			}
		}
		if target == nil {
			t.Fatal("could not find target task")
		}
		if len(target.BlockedBy) != 1 || target.BlockedBy[0] != "tick-aaa111" {
			t.Errorf("expected target blocked_by [tick-aaa111], got %v", target.BlockedBy)
		}
		// Target's updated should have changed
		if !target.Updated.After(now) {
			t.Errorf("expected target updated timestamp to be refreshed")
		}
	})
}

func TestUpdate_MultipleFields(t *testing.T) {
	t.Run("it updates multiple fields in a single command", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Original", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111",
			"--title", "New title",
			"--description", "New desc",
			"--priority", "1",
		})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		tk := tasks[0]
		if tk.Title != "New title" {
			t.Errorf("expected title %q, got %q", "New title", tk.Title)
		}
		if tk.Description != "New desc" {
			t.Errorf("expected description %q, got %q", "New desc", tk.Description)
		}
		if tk.Priority != 1 {
			t.Errorf("expected priority 1, got %d", tk.Priority)
		}
	})
}

func TestUpdate_RefreshesUpdatedTimestamp(t *testing.T) {
	t.Run("it refreshes updated timestamp on any change", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "Changed"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		if !tasks[0].Updated.After(now) {
			t.Errorf("expected updated timestamp to be refreshed, was %v (original: %v)", tasks[0].Updated, now)
		}
		// Created should not change
		if !tasks[0].Created.Equal(now) {
			t.Errorf("expected created timestamp to remain %v, got %v", now, tasks[0].Created)
		}
	})
}

func TestUpdate_OutputsFullTaskDetails(t *testing.T) {
	t.Run("it outputs full task details on success", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "Updated"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// TOON format: task{id,title,status,priority,...}: header with data
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("expected output to contain task ID, got %q", output)
		}
		if !strings.Contains(output, "Updated") {
			t.Errorf("expected output to contain new title, got %q", output)
		}
		if !strings.Contains(output, "open") {
			t.Errorf("expected output to contain status 'open', got %q", output)
		}
		if !strings.Contains(output, "task{") {
			t.Errorf("expected output to contain TOON task header, got %q", output)
		}
	})
}

func TestUpdate_QuietFlag(t *testing.T) {
	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "update", "tick-aaa111", "--title", "Updated"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("expected only task ID in quiet output, got %q", output)
		}
	})
}

func TestUpdate_ErrorNoFlags(t *testing.T) {
	t.Run("it errors when no flags are provided", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "No flags provided") {
			t.Errorf("expected 'No flags provided' in error, got %q", errMsg)
		}
		// Should list available options
		if !strings.Contains(errMsg, "--title") {
			t.Errorf("expected available options list in error, got %q", errMsg)
		}
	})
}

func TestUpdate_ErrorMissingID(t *testing.T) {
	t.Run("it errors when task ID is missing", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "--title", "Something"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Task ID is required") {
			t.Errorf("expected 'Task ID is required' in error, got %q", errMsg)
		}
	})
}

func TestUpdate_ErrorIDNotFound(t *testing.T) {
	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-nonexist", "--title", "Something"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "not found") {
			t.Errorf("expected 'not found' in error, got %q", errMsg)
		}
	})
}

func TestUpdate_ErrorInvalidTitle(t *testing.T) {
	t.Run("it errors on invalid title (empty/500/newlines)", func(t *testing.T) {
		tests := []struct {
			name  string
			title string
		}{
			{"empty", ""},
			{"whitespace only", "   "},
			{"too long", strings.Repeat("a", 501)},
			{"contains newline", "line one\nline two"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
				existing := []task.Task{
					{ID: "tick-aaa111", Title: "Original", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
				}
				dir := setupInitializedDirWithTasks(t, existing)
				var stdout, stderr bytes.Buffer

				app := &App{
					Stdout: &stdout,
					Stderr: &stderr,
					Dir:    dir,
				}

				code := app.Run([]string{"tick", "update", "tick-aaa111", "--title", tt.title})
				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if stderr.String() == "" {
					t.Error("expected error on stderr")
				}

				// Verify no mutation occurred
				tasks := readTasksFromDir(t, dir)
				if tasks[0].Title != "Original" {
					t.Errorf("expected title to remain %q, got %q", "Original", tasks[0].Title)
				}
			})
		}
	})
}

func TestUpdate_ErrorInvalidPriority(t *testing.T) {
	t.Run("it errors on invalid priority (outside 0-4)", func(t *testing.T) {
		tests := []struct {
			name     string
			priority string
		}{
			{"negative", "-1"},
			{"too high", "5"},
			{"way too high", "100"},
			{"not a number", "abc"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
				existing := []task.Task{
					{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
				}
				dir := setupInitializedDirWithTasks(t, existing)
				var stdout, stderr bytes.Buffer

				app := &App{
					Stdout: &stdout,
					Stderr: &stderr,
					Dir:    dir,
				}

				code := app.Run([]string{"tick", "update", "tick-aaa111", "--priority", tt.priority})
				if code != 1 {
					t.Errorf("expected exit code 1, got %d", code)
				}
				if stderr.String() == "" {
					t.Error("expected error on stderr")
				}
			})
		}
	})
}

func TestUpdate_ErrorNonExistentParentBlocks(t *testing.T) {
	t.Run("it errors on non-existent parent/blocks IDs", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		t.Run("non-existent parent", func(t *testing.T) {
			dir := setupInitializedDirWithTasks(t, existing)
			var stdout, stderr bytes.Buffer

			app := &App{
				Stdout: &stdout,
				Stderr: &stderr,
				Dir:    dir,
			}

			code := app.Run([]string{"tick", "update", "tick-aaa111", "--parent", "tick-nonexist"})
			if code != 1 {
				t.Errorf("expected exit code 1, got %d", code)
			}
			errMsg := stderr.String()
			if !strings.Contains(errMsg, "not found") {
				t.Errorf("expected 'not found' in error, got %q", errMsg)
			}
		})

		t.Run("non-existent blocks target", func(t *testing.T) {
			dir := setupInitializedDirWithTasks(t, existing)
			var stdout, stderr bytes.Buffer

			app := &App{
				Stdout: &stdout,
				Stderr: &stderr,
				Dir:    dir,
			}

			code := app.Run([]string{"tick", "update", "tick-aaa111", "--blocks", "tick-nonexist"})
			if code != 1 {
				t.Errorf("expected exit code 1, got %d", code)
			}
			errMsg := stderr.String()
			if !strings.Contains(errMsg, "not found") {
				t.Errorf("expected 'not found' in error, got %q", errMsg)
			}
		})
	})
}

func TestUpdate_ErrorSelfReferencingParent(t *testing.T) {
	t.Run("it errors on self-referencing parent", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--parent", "tick-aaa111"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "cannot be its own parent") {
			t.Errorf("expected self-reference error, got %q", errMsg)
		}
	})
}

func TestUpdate_NormalizesInputIDs(t *testing.T) {
	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		// Use uppercase IDs
		code := app.Run([]string{"tick", "update", "TICK-AAA111", "--parent", "TICK-BBB222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var tk *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				tk = &tasks[i]
				break
			}
		}
		if tk == nil {
			t.Fatal("could not find task")
		}
		if tk.Parent != "tick-bbb222" {
			t.Errorf("expected parent %q (lowercase), got %q", "tick-bbb222", tk.Parent)
		}
	})
}

func TestUpdate_PersistsChanges(t *testing.T) {
	t.Run("it persists changes via atomic write", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Original", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "Persisted title"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Read raw file and verify it's valid JSONL with the updated data
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line in JSONL, got %d", len(lines))
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
			t.Fatalf("failed to parse JSONL line as JSON: %v", err)
		}
		if parsed["title"] != "Persisted title" {
			t.Errorf("expected title 'Persisted title', got %v", parsed["title"])
		}

		// Verify cache.db was updated
		cacheDB := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cacheDB); os.IsNotExist(err) {
			t.Error("expected cache.db to exist after update")
		}
	})
}
