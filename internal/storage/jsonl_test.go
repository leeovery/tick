package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestWriteTasks(t *testing.T) {
	t.Run("it writes tasks as one JSON object per line", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "First task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			{
				ID:       "tick-d4e5f6",
				Title:    "Second task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
		}

		if err := WriteTasks(path, tasks); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}

		content := string(data)
		lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %q", len(lines), content)
		}

		// Each line should be valid JSON (no pretty-printing)
		for i, line := range lines {
			if strings.Contains(line, "\n") || strings.Contains(line, "  ") {
				t.Errorf("line %d appears to be pretty-printed: %q", i, line)
			}
		}

		// Verify first line contains first task
		if !strings.Contains(lines[0], `"tick-a1b2c3"`) {
			t.Errorf("line 0 should contain first task ID, got: %s", lines[0])
		}
		if !strings.Contains(lines[1], `"tick-d4e5f6"`) {
			t.Errorf("line 1 should contain second task ID, got: %s", lines[1])
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
			t.Fatalf("writing file: %v", err)
		}

		tasks, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-a1b2c3" {
			t.Errorf("task 0 ID = %q, want %q", tasks[0].ID, "tick-a1b2c3")
		}
		if tasks[0].Title != "First task" {
			t.Errorf("task 0 Title = %q, want %q", tasks[0].Title, "First task")
		}
		if tasks[1].ID != "tick-d4e5f6" {
			t.Errorf("task 1 ID = %q, want %q", tasks[1].ID, "tick-d4e5f6")
		}
		if tasks[1].Status != task.StatusDone {
			t.Errorf("task 1 Status = %q, want %q", tasks[1].Status, task.StatusDone)
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("it round-trips all task fields without loss", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

		original := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      task.StatusDone,
				Priority:    1,
				Description: "Detailed description\nwith newlines",
				BlockedBy:   []string{"tick-x1y2z3", "tick-m4n5o6"},
				Parent:      "tick-p7q8r9",
				Created:     created,
				Updated:     updated,
				Closed:      &closed,
			},
		}

		if err := WriteTasks(path, original); err != nil {
			t.Fatalf("write error: %v", err)
		}

		restored, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("read error: %v", err)
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
			t.Fatal("Closed should not be nil")
		}
		if !got.Closed.Equal(*original[0].Closed) {
			t.Errorf("Closed = %v, want %v", got.Closed, original[0].Closed)
		}
	})
}

func TestOptionalFieldOmission(t *testing.T) {
	t.Run("it omits optional fields when empty", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Minimal task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
				// No Description, BlockedBy, Parent, or Closed
			},
		}

		if err := WriteTasks(path, tasks); err != nil {
			t.Fatalf("write error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}

		line := strings.TrimSpace(string(data))
		if strings.Contains(line, "description") {
			t.Errorf("description should be omitted, got: %s", line)
		}
		if strings.Contains(line, "blocked_by") {
			t.Errorf("blocked_by should be omitted, got: %s", line)
		}
		if strings.Contains(line, "parent") {
			t.Errorf("parent should be omitted, got: %s", line)
		}
		if strings.Contains(line, "closed") {
			t.Errorf("closed should be omitted, got: %s", line)
		}
		// Ensure required fields are present
		if !strings.Contains(line, `"id"`) {
			t.Errorf("id should be present, got: %s", line)
		}
		if !strings.Contains(line, `"title"`) {
			t.Errorf("title should be present, got: %s", line)
		}
		if !strings.Contains(line, `"status"`) {
			t.Errorf("status should be present, got: %s", line)
		}
		if !strings.Contains(line, `"priority"`) {
			t.Errorf("priority should be present, got: %s", line)
		}
		if !strings.Contains(line, `"created"`) {
			t.Errorf("created should be present, got: %s", line)
		}
		if !strings.Contains(line, `"updated"`) {
			t.Errorf("updated should be present, got: %s", line)
		}
	})
}

func TestEmptyFile(t *testing.T) {
	t.Run("it returns empty list for empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		// Create empty file (0 bytes)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("creating empty file: %v", err)
		}

		tasks, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})
}

func TestMissingFile(t *testing.T) {
	t.Run("it returns error for missing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nonexistent.jsonl")

		_, err := ReadTasks(path)
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})
}

func TestAtomicWrite(t *testing.T) {
	t.Run("it writes atomically via temp file and rename", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// Write initial content
		initial := []task.Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Initial task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}
		if err := WriteTasks(path, initial); err != nil {
			t.Fatalf("initial write error: %v", err)
		}

		// Verify initial content exists
		tasks, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("read error: %v", err)
		}
		if len(tasks) != 1 || tasks[0].ID != "tick-aaaaaa" {
			t.Fatalf("initial content wrong: %v", tasks)
		}

		// Overwrite with new content
		updated := []task.Task{
			{
				ID:       "tick-bbbbbb",
				Title:    "Updated task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
		}
		if err := WriteTasks(path, updated); err != nil {
			t.Fatalf("overwrite error: %v", err)
		}

		// Verify new content replaced old
		tasks, err = ReadTasks(path)
		if err != nil {
			t.Fatalf("read error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-bbbbbb" {
			t.Errorf("expected tick-bbbbbb, got %s", tasks[0].ID)
		}

		// Verify no temp files left behind
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("reading dir: %v", err)
		}
		for _, e := range entries {
			if strings.Contains(e.Name(), ".tmp") {
				t.Errorf("temp file left behind: %s", e.Name())
			}
		}
	})
}

func TestAllFieldsPopulated(t *testing.T) {
	t.Run("it handles tasks with all fields populated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

		full := []task.Task{
			{
				ID:          "tick-ffffff",
				Title:       "Fully populated task",
				Status:      task.StatusDone,
				Priority:    0,
				Description: "A detailed description with special chars: <>&\"'",
				BlockedBy:   []string{"tick-aaaaaa", "tick-bbbbbb"},
				Parent:      "tick-cccccc",
				Created:     created,
				Updated:     updated,
				Closed:      &closed,
			},
		}

		if err := WriteTasks(path, full); err != nil {
			t.Fatalf("write error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}

		line := strings.TrimSpace(string(data))
		// All optional fields should be present
		for _, field := range []string{"description", "blocked_by", "parent", "closed"} {
			if !strings.Contains(line, field) {
				t.Errorf("expected %q in output, got: %s", field, line)
			}
		}

		// Round-trip
		restored, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("read error: %v", err)
		}

		if len(restored) != 1 {
			t.Fatalf("expected 1 task, got %d", len(restored))
		}

		got := restored[0]
		if got.Description != full[0].Description {
			t.Errorf("Description = %q, want %q", got.Description, full[0].Description)
		}
		if len(got.BlockedBy) != 2 {
			t.Fatalf("BlockedBy length = %d, want 2", len(got.BlockedBy))
		}
		if got.Parent != "tick-cccccc" {
			t.Errorf("Parent = %q, want %q", got.Parent, "tick-cccccc")
		}
		if got.Closed == nil || !got.Closed.Equal(closed) {
			t.Errorf("Closed = %v, want %v", got.Closed, closed)
		}
	})
}

func TestOnlyRequiredFields(t *testing.T) {
	t.Run("it handles tasks with only required fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		minimal := []task.Task{
			{
				ID:       "tick-111111",
				Title:    "Minimal task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}

		if err := WriteTasks(path, minimal); err != nil {
			t.Fatalf("write error: %v", err)
		}

		restored, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("read error: %v", err)
		}

		if len(restored) != 1 {
			t.Fatalf("expected 1 task, got %d", len(restored))
		}

		got := restored[0]
		if got.ID != "tick-111111" {
			t.Errorf("ID = %q, want %q", got.ID, "tick-111111")
		}
		if got.Description != "" {
			t.Errorf("Description should be empty, got %q", got.Description)
		}
		if got.BlockedBy != nil {
			t.Errorf("BlockedBy should be nil, got %v", got.BlockedBy)
		}
		if got.Parent != "" {
			t.Errorf("Parent should be empty, got %q", got.Parent)
		}
		if got.Closed != nil {
			t.Errorf("Closed should be nil, got %v", got.Closed)
		}
	})
}

func TestFieldOrdering(t *testing.T) {
	t.Run("it outputs fields in spec order: id, title, status, priority, then optional, then timestamps", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

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
				Closed:      &closed,
			},
		}

		if err := WriteTasks(path, tasks); err != nil {
			t.Fatalf("write error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}

		line := strings.TrimSpace(string(data))

		// Verify field ordering by checking index positions
		fields := []string{`"id"`, `"title"`, `"status"`, `"priority"`, `"description"`, `"blocked_by"`, `"parent"`, `"created"`, `"updated"`, `"closed"`}
		prevIdx := -1
		for _, field := range fields {
			idx := strings.Index(line, field)
			if idx == -1 {
				t.Errorf("field %s not found in output: %s", field, line)
				continue
			}
			if idx <= prevIdx {
				t.Errorf("field %s (at %d) should come after previous field (at %d) in: %s", field, idx, prevIdx, line)
			}
			prevIdx = idx
		}
	})
}

func TestSkipsEmptyLines(t *testing.T) {
	t.Run("it skips empty lines when reading", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("writing file: %v", err)
		}

		tasks, err := ReadTasks(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
	})
}

func TestWriteEmptyTaskList(t *testing.T) {
	t.Run("it writes empty file for empty task list", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		if err := WriteTasks(path, []task.Task{}); err != nil {
			t.Fatalf("write error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}

		if len(data) != 0 {
			t.Errorf("expected empty file, got %d bytes: %q", len(data), string(data))
		}
	})
}
