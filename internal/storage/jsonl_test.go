package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	return string(data)
}

func sampleTask(id, title string) task.Task {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	return task.Task{
		ID:       id,
		Title:    title,
		Status:   task.StatusOpen,
		Priority: 2,
		Created:  now,
		Updated:  now,
	}
}

func TestWriteJSONL(t *testing.T) {
	t.Run("writes tasks as one JSON object per line", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		writeFile(t, path, "") // create empty file

		tasks := []task.Task{
			sampleTask("tick-a1b2c3", "First task"),
			sampleTask("tick-d4e5f6", "Second task"),
		}

		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL() error: %v", err)
		}

		content := readFile(t, path)
		lines := strings.Split(strings.TrimSpace(content), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %q", len(lines), content)
		}

		// Each line should be valid JSON
		for i, line := range lines {
			if !json.Valid([]byte(line)) {
				t.Errorf("line %d is not valid JSON: %q", i, line)
			}
		}
	})

	t.Run("omits optional fields when empty", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		writeFile(t, path, "")

		tasks := []task.Task{sampleTask("tick-a1b2c3", "Minimal task")}

		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL() error: %v", err)
		}

		content := readFile(t, path)
		if strings.Contains(content, `"description"`) {
			t.Error("empty description should be omitted")
		}
		if strings.Contains(content, `"blocked_by"`) {
			t.Error("empty blocked_by should be omitted")
		}
		if strings.Contains(content, `"parent"`) {
			t.Error("empty parent should be omitted")
		}
		if strings.Contains(content, `"closed"`) {
			t.Error("nil closed should be omitted")
		}
	})

	t.Run("includes optional fields when populated", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		writeFile(t, path, "")

		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		t2 := task.Task{
			ID:          "tick-a1b2c3",
			Title:       "Full task",
			Status:      task.StatusDone,
			Priority:    1,
			Description: "Detailed description",
			BlockedBy:   []string{"tick-d4e5f6"},
			Parent:      "tick-g7h8i9",
			Created:     time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC),
			Updated:     time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC),
			Closed:      &closed,
		}

		if err := WriteJSONL(path, []task.Task{t2}); err != nil {
			t.Fatalf("WriteJSONL() error: %v", err)
		}

		content := readFile(t, path)
		for _, field := range []string{`"description"`, `"blocked_by"`, `"parent"`, `"closed"`} {
			if !strings.Contains(content, field) {
				t.Errorf("populated field %s should be present in output", field)
			}
		}
	})

	t.Run("writes empty list as empty file", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		writeFile(t, path, "")

		if err := WriteJSONL(path, []task.Task{}); err != nil {
			t.Fatalf("WriteJSONL() error: %v", err)
		}

		content := readFile(t, path)
		if content != "" {
			t.Errorf("expected empty file, got %q", content)
		}
	})
}

func TestReadJSONL(t *testing.T) {
	t.Run("reads tasks from JSONL file", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		content := `{"id":"tick-a1b2c3","title":"First","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Second","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		writeFile(t, path, content)

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL() error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-a1b2c3" {
			t.Errorf("task[0].ID = %q, want %q", tasks[0].ID, "tick-a1b2c3")
		}
		if tasks[1].Priority != 1 {
			t.Errorf("task[1].Priority = %d, want 1", tasks[1].Priority)
		}
	})

	t.Run("returns empty list for empty file", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		writeFile(t, path, "")

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL() error: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := ReadJSONL("/nonexistent/tasks.jsonl")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("skips empty lines", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		content := `{"id":"tick-a1b2c3","title":"First","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

{"id":"tick-d4e5f6","title":"Second","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		writeFile(t, path, content)

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL() error: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks (skipping blank lines), got %d", len(tasks))
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("round-trips all task fields without loss", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		writeFile(t, path, "")

		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		original := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      task.StatusDone,
				Priority:    1,
				Description: "Detailed description",
				BlockedBy:   []string{"tick-d4e5f6", "tick-g7h8i9"},
				Parent:      "tick-j0k1l2",
				Created:     time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC),
				Updated:     time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC),
				Closed:      &closed,
			},
			sampleTask("tick-minimal", "Minimal task"),
		}

		if err := WriteJSONL(path, original); err != nil {
			t.Fatalf("WriteJSONL() error: %v", err)
		}

		loaded, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL() error: %v", err)
		}

		if len(loaded) != len(original) {
			t.Fatalf("expected %d tasks, got %d", len(original), len(loaded))
		}

		// Full task round-trip
		full := loaded[0]
		if full.ID != "tick-a1b2c3" {
			t.Errorf("ID = %q, want %q", full.ID, "tick-a1b2c3")
		}
		if full.Title != "Full task" {
			t.Errorf("Title = %q, want %q", full.Title, "Full task")
		}
		if full.Status != task.StatusDone {
			t.Errorf("Status = %q, want %q", full.Status, task.StatusDone)
		}
		if full.Priority != 1 {
			t.Errorf("Priority = %d, want 1", full.Priority)
		}
		if full.Description != "Detailed description" {
			t.Errorf("Description = %q, want %q", full.Description, "Detailed description")
		}
		if len(full.BlockedBy) != 2 || full.BlockedBy[0] != "tick-d4e5f6" || full.BlockedBy[1] != "tick-g7h8i9" {
			t.Errorf("BlockedBy = %v, want [tick-d4e5f6 tick-g7h8i9]", full.BlockedBy)
		}
		if full.Parent != "tick-j0k1l2" {
			t.Errorf("Parent = %q, want %q", full.Parent, "tick-j0k1l2")
		}
		if full.Closed == nil || !full.Closed.Equal(closed) {
			t.Errorf("Closed = %v, want %v", full.Closed, closed)
		}

		// Minimal task round-trip
		min := loaded[1]
		if min.Description != "" {
			t.Errorf("minimal Description = %q, want empty", min.Description)
		}
		if min.BlockedBy != nil {
			t.Errorf("minimal BlockedBy = %v, want nil", min.BlockedBy)
		}
		if min.Parent != "" {
			t.Errorf("minimal Parent = %q, want empty", min.Parent)
		}
		if min.Closed != nil {
			t.Errorf("minimal Closed = %v, want nil", min.Closed)
		}
	})
}

func TestAtomicWrite(t *testing.T) {
	t.Run("writes atomically via temp file and rename", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "tasks.jsonl")
		writeFile(t, path, `{"id":"tick-old","title":"Old","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`)

		tasks := []task.Task{sampleTask("tick-new", "New task")}
		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL() error: %v", err)
		}

		content := readFile(t, path)
		if strings.Contains(content, "tick-old") {
			t.Error("old content should be replaced")
		}
		if !strings.Contains(content, "tick-new") {
			t.Error("new content should be present")
		}

		// No temp files left behind
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.Name() != "tasks.jsonl" {
				t.Errorf("unexpected file left behind: %s", e.Name())
			}
		}
	})

	t.Run("original file survives write to nonexistent directory", func(t *testing.T) {
		// Writing to a path where the parent doesn't exist should error
		err := WriteJSONL("/nonexistent/dir/tasks.jsonl", []task.Task{})
		if err == nil {
			t.Fatal("expected error writing to nonexistent directory")
		}
	})
}
