package jsonl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// Helper to create a time from string for deterministic tests.
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", s, err)
	}
	return ts
}

// Helper to create a time pointer.
func timePtr(t time.Time) *time.Time {
	return &t
}

func TestWriteTasks(t *testing.T) {
	t.Run("it writes tasks as one JSON object per line", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "First task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
			{
				ID:       "tick-d4e5f6",
				Title:    "Second task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}

		err := WriteTasks(path, tasks)
		if err != nil {
			t.Fatalf("WriteTasks() returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		// Each line must be valid JSON
		for i, line := range lines {
			if !json.Valid([]byte(line)) {
				t.Errorf("line %d is not valid JSON: %q", i, line)
			}
		}
	})
}

func TestReadTasks(t *testing.T) {
	t.Run("it reads tasks from JSONL file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		tasks, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("ReadTasks() returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-a1b2c3" {
			t.Errorf("tasks[0].ID = %q, want %q", tasks[0].ID, "tick-a1b2c3")
		}
		if tasks[1].ID != "tick-d4e5f6" {
			t.Errorf("tasks[1].ID = %q, want %q", tasks[1].ID, "tick-d4e5f6")
		}
	})
}

func TestParseTasks(t *testing.T) {
	t.Run("it parses tasks from raw JSONL bytes", func(t *testing.T) {
		data := []byte(`{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`)

		tasks, err := ParseTasks(data)
		if err != nil {
			t.Fatalf("ParseTasks() returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-a1b2c3" {
			t.Errorf("tasks[0].ID = %q, want %q", tasks[0].ID, "tick-a1b2c3")
		}
		if tasks[1].ID != "tick-d4e5f6" {
			t.Errorf("tasks[1].ID = %q, want %q", tasks[1].ID, "tick-d4e5f6")
		}
	})

	t.Run("it returns empty list for empty bytes", func(t *testing.T) {
		tasks, err := ParseTasks([]byte(""))
		if err != nil {
			t.Fatalf("ParseTasks() returned error: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("it round-trips all task fields without loss", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		updated := mustParseTime(t, "2026-01-19T14:00:00Z")
		closed := mustParseTime(t, "2026-01-19T16:00:00Z")

		original := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      task.StatusDone,
				Priority:    1,
				Description: "A detailed description",
				BlockedBy:   []string{"tick-x1y2z3", "tick-m4n5o6"},
				Parent:      "tick-p7q8r9",
				Created:     created,
				Updated:     updated,
				Closed:      timePtr(closed),
			},
		}

		err := WriteTasks(path, original)
		if err != nil {
			t.Fatalf("WriteTasks() returned error: %v", err)
		}

		restored, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("ReadTasks() returned error: %v", err)
		}

		if len(restored) != 1 {
			t.Fatalf("expected 1 task, got %d", len(restored))
		}

		got := restored[0]
		if got.ID != original[0].ID {
			t.Errorf("ID = %q, want %q", got.ID, original[0].ID)
		}
		if got.Title != original[0].Title {
			t.Errorf("Title = %q, want %q", got.Title, original[0].Title)
		}
		if got.Status != original[0].Status {
			t.Errorf("Status = %q, want %q", got.Status, original[0].Status)
		}
		if got.Priority != original[0].Priority {
			t.Errorf("Priority = %d, want %d", got.Priority, original[0].Priority)
		}
		if got.Description != original[0].Description {
			t.Errorf("Description = %q, want %q", got.Description, original[0].Description)
		}
		if len(got.BlockedBy) != len(original[0].BlockedBy) {
			t.Fatalf("BlockedBy length = %d, want %d", len(got.BlockedBy), len(original[0].BlockedBy))
		}
		for i, dep := range got.BlockedBy {
			if dep != original[0].BlockedBy[i] {
				t.Errorf("BlockedBy[%d] = %q, want %q", i, dep, original[0].BlockedBy[i])
			}
		}
		if got.Parent != original[0].Parent {
			t.Errorf("Parent = %q, want %q", got.Parent, original[0].Parent)
		}
		if !got.Created.Equal(original[0].Created) {
			t.Errorf("Created = %v, want %v", got.Created, original[0].Created)
		}
		if !got.Updated.Equal(original[0].Updated) {
			t.Errorf("Updated = %v, want %v", got.Updated, original[0].Updated)
		}
		if got.Closed == nil {
			t.Fatal("Closed is nil, want non-nil")
		}
		if !got.Closed.Equal(*original[0].Closed) {
			t.Errorf("Closed = %v, want %v", got.Closed, original[0].Closed)
		}
	})
}

func TestOmitOptionalFields(t *testing.T) {
	t.Run("it omits optional fields when empty", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Minimal task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
				// Description, BlockedBy, Parent, Closed all empty/nil
			},
		}

		err := WriteTasks(path, tasks)
		if err != nil {
			t.Fatalf("WriteTasks() returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		line := strings.TrimSpace(string(data))

		// Optional fields must NOT appear in output
		if strings.Contains(line, `"description"`) {
			t.Error("line contains 'description' key, expected it to be omitted")
		}
		if strings.Contains(line, `"blocked_by"`) {
			t.Error("line contains 'blocked_by' key, expected it to be omitted")
		}
		if strings.Contains(line, `"parent"`) {
			t.Error("line contains 'parent' key, expected it to be omitted")
		}
		if strings.Contains(line, `"closed"`) {
			t.Error("line contains 'closed' key, expected it to be omitted")
		}
		if strings.Contains(line, "null") {
			t.Error("line contains 'null', expected optional fields to be omitted entirely")
		}
	})
}

func TestEmptyFile(t *testing.T) {
	t.Run("it returns empty list for empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		// Create empty file
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatalf("failed to write empty file: %v", err)
		}

		tasks, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("ReadTasks() returned error: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})
}

func TestMissingFile(t *testing.T) {
	t.Run("it returns error for missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nonexistent.jsonl")

		_, err := ReadTasks(path)
		if err == nil {
			t.Fatal("ReadTasks() expected error for missing file, got nil")
		}
	})
}

func TestAtomicWrite(t *testing.T) {
	t.Run("it writes atomically via temp file and rename", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := mustParseTime(t, "2026-01-19T10:00:00Z")

		// Write initial content
		initialTasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Initial task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		if err := WriteTasks(path, initialTasks); err != nil {
			t.Fatalf("initial WriteTasks() returned error: %v", err)
		}

		// Read initial content to verify it was written
		initialData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read initial file: %v", err)
		}
		if !strings.Contains(string(initialData), "tick-a1b2c3") {
			t.Fatal("initial file doesn't contain expected task")
		}

		// Overwrite with new content (full rewrite)
		newTasks := []task.Task{
			{
				ID:       "tick-d4e5f6",
				Title:    "Replacement task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}
		if err := WriteTasks(path, newTasks); err != nil {
			t.Fatalf("second WriteTasks() returned error: %v", err)
		}

		// Verify new content replaced old content entirely
		newData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read new file: %v", err)
		}
		content := string(newData)
		if strings.Contains(content, "tick-a1b2c3") {
			t.Error("file still contains old task after rewrite")
		}
		if !strings.Contains(content, "tick-d4e5f6") {
			t.Error("file doesn't contain new task after rewrite")
		}

		// Verify no temp files are left behind
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".tasks.jsonl.tmp") {
				t.Errorf("temp file left behind: %s", entry.Name())
			}
		}
	})
}

func TestHandleAllFieldsPopulated(t *testing.T) {
	t.Run("it handles tasks with all fields populated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		updated := mustParseTime(t, "2026-01-19T14:00:00Z")
		closed := mustParseTime(t, "2026-01-19T16:00:00Z")

		tasks := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      task.StatusDone,
				Priority:    0,
				Description: "A detailed description\nwith newlines",
				BlockedBy:   []string{"tick-x1y2z3"},
				Parent:      "tick-p7q8r9",
				Created:     created,
				Updated:     updated,
				Closed:      timePtr(closed),
			},
		}

		err := WriteTasks(path, tasks)
		if err != nil {
			t.Fatalf("WriteTasks() returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		line := strings.TrimSpace(string(data))

		// All fields must be present
		for _, field := range []string{`"id"`, `"title"`, `"status"`, `"priority"`, `"description"`, `"blocked_by"`, `"parent"`, `"created"`, `"updated"`, `"closed"`} {
			if !strings.Contains(line, field) {
				t.Errorf("line missing field %s", field)
			}
		}

		// Each task on exactly one line (no pretty-printing)
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 1 {
			t.Errorf("expected 1 line, got %d", len(lines))
		}
	})
}

func TestHandleOnlyRequiredFields(t *testing.T) {
	t.Run("it handles tasks with only required fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := mustParseTime(t, "2026-01-19T10:00:00Z")

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Minimal task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}

		if err := WriteTasks(path, tasks); err != nil {
			t.Fatalf("WriteTasks() returned error: %v", err)
		}

		restored, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("ReadTasks() returned error: %v", err)
		}

		if len(restored) != 1 {
			t.Fatalf("expected 1 task, got %d", len(restored))
		}

		got := restored[0]
		if got.ID != "tick-a1b2c3" {
			t.Errorf("ID = %q, want %q", got.ID, "tick-a1b2c3")
		}
		if got.Title != "Minimal task" {
			t.Errorf("Title = %q, want %q", got.Title, "Minimal task")
		}
		if got.Status != task.StatusOpen {
			t.Errorf("Status = %q, want %q", got.Status, task.StatusOpen)
		}
		if got.Priority != 2 {
			t.Errorf("Priority = %d, want 2", got.Priority)
		}
		if got.Description != "" {
			t.Errorf("Description = %q, want empty", got.Description)
		}
		if len(got.BlockedBy) != 0 {
			t.Errorf("BlockedBy = %v, want empty", got.BlockedBy)
		}
		if got.Parent != "" {
			t.Errorf("Parent = %q, want empty", got.Parent)
		}
		if got.Closed != nil {
			t.Errorf("Closed = %v, want nil", got.Closed)
		}
	})
}

func TestFieldOrdering(t *testing.T) {
	t.Run("it outputs fields in spec order: id, title, status, priority, optional, created, updated, closed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := mustParseTime(t, "2026-01-19T10:00:00Z")
		updated := mustParseTime(t, "2026-01-19T14:00:00Z")
		closed := mustParseTime(t, "2026-01-19T16:00:00Z")

		tasks := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      task.StatusDone,
				Priority:    1,
				Description: "Details",
				BlockedBy:   []string{"tick-x1y2z3"},
				Parent:      "tick-p7q8r9",
				Created:     created,
				Updated:     updated,
				Closed:      timePtr(closed),
			},
		}

		if err := WriteTasks(path, tasks); err != nil {
			t.Fatalf("WriteTasks() returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		line := strings.TrimSpace(string(data))

		// Verify field ordering by checking index positions
		fields := []string{`"id"`, `"title"`, `"status"`, `"priority"`, `"description"`, `"blocked_by"`, `"parent"`, `"created"`, `"updated"`, `"closed"`}
		lastIdx := -1
		for _, field := range fields {
			idx := strings.Index(line, field)
			if idx == -1 {
				t.Fatalf("field %s not found in output", field)
			}
			if idx <= lastIdx {
				t.Errorf("field %s (at %d) appears before previous field (at %d) - wrong order", field, idx, lastIdx)
			}
			lastIdx = idx
		}
	})
}

func TestReadSkipsEmptyLines(t *testing.T) {
	t.Run("it skips empty lines when reading", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"First","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

{"id":"tick-d4e5f6","title":"Second","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		tasks, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("ReadTasks() returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(tasks))
		}
	})
}
