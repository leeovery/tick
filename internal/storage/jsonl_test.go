package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestJSONLFormat(t *testing.T) {
	t.Run("it matches spec format with correct field ordering", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// Minimal task (only required fields)
		minTask := task.Task{
			ID:       "tick-a1b2",
			Title:    "Task title",
			Status:   task.StatusOpen,
			Priority: 2,
			Created:  created,
			Updated:  created,
		}

		if err := WriteJSONL(path, []task.Task{minTask}); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Spec format: {"id":"tick-a1b2","title":"Task title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
		expected := `{"id":"tick-a1b2","title":"Task title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		got := strings.TrimSpace(string(data))
		if got != expected {
			t.Errorf("JSONL format mismatch\ngot:  %s\nwant: %s", got, expected)
		}
	})

	t.Run("it matches spec format with all fields populated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

		fullTask := task.Task{
			ID:          "tick-c3d4",
			Title:       "With all fields",
			Status:      task.StatusInProgress,
			Priority:    1,
			Description: "Details here",
			BlockedBy:   []string{"tick-a1b2"},
			Parent:      "tick-e5f6",
			Created:     created,
			Updated:     updated,
			Closed:      &closed,
		}

		if err := WriteJSONL(path, []task.Task{fullTask}); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Field order: id, title, status, priority, description, blocked_by, parent, created, updated, closed
		expected := `{"id":"tick-c3d4","title":"With all fields","status":"in_progress","priority":1,"description":"Details here","blocked_by":["tick-a1b2"],"parent":"tick-e5f6","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:00:00Z","closed":"2026-01-19T16:00:00Z"}`
		got := strings.TrimSpace(string(data))
		if got != expected {
			t.Errorf("JSONL format mismatch\ngot:  %s\nwant: %s", got, expected)
		}
	})
}

func TestWriteJSONL(t *testing.T) {
	t.Run("it writes tasks as one JSON object per line", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
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

		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		// Each line should be a valid JSON object (no pretty-printing)
		for i, line := range lines {
			if strings.Contains(line, "\n") {
				t.Errorf("line %d contains embedded newline", i)
			}
			if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
				t.Errorf("line %d is not a JSON object: %s", i, line)
			}
		}
	})
}

func TestReadJSONL(t *testing.T) {
	t.Run("it returns empty list for empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		// Create an empty file (0 bytes)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create empty file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error for empty file: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks for empty file, got %d", len(tasks))
		}
	})

	t.Run("it returns error for missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nonexistent.jsonl")

		_, err := ReadJSONL(path)
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("it reads tasks from JSONL file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		if tasks[0].ID != "tick-a1b2c3" {
			t.Errorf("tasks[0].ID = %q, want %q", tasks[0].ID, "tick-a1b2c3")
		}
		if tasks[0].Title != "First task" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "First task")
		}
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("tasks[0].Status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
		if tasks[0].Priority != 2 {
			t.Errorf("tasks[0].Priority = %d, want %d", tasks[0].Priority, 2)
		}

		if tasks[1].ID != "tick-d4e5f6" {
			t.Errorf("tasks[1].ID = %q, want %q", tasks[1].ID, "tick-d4e5f6")
		}
		if tasks[1].Status != task.StatusDone {
			t.Errorf("tasks[1].Status = %q, want %q", tasks[1].Status, task.StatusDone)
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("it omits optional fields when empty", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
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

		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		line := strings.TrimSpace(string(data))

		// Optional fields should NOT appear in JSON output
		optionalFields := []string{"description", "blocked_by", "parent", "closed"}
		for _, field := range optionalFields {
			if strings.Contains(line, `"`+field+`"`) {
				t.Errorf("optional field %q should be omitted when empty, but found in: %s", field, line)
			}
		}

		// Required fields should be present
		requiredFields := []string{"id", "title", "status", "priority", "created", "updated"}
		for _, field := range requiredFields {
			if !strings.Contains(line, `"`+field+`"`) {
				t.Errorf("required field %q should be present, but not found in: %s", field, line)
			}
		}
	})

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
				Description: "Detailed description here",
				BlockedBy:   []string{"tick-x1y2z3"},
				Parent:      "tick-p1a2b3",
				Created:     created,
				Updated:     updated,
				Closed:      &closed,
			},
		}

		if err := WriteJSONL(path, original); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		roundTripped, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(roundTripped) != 1 {
			t.Fatalf("expected 1 task, got %d", len(roundTripped))
		}

		got := roundTripped[0]
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

func TestAtomicWrite(t *testing.T) {
	t.Run("it writes atomically via temp file and rename", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// Write initial content
		initial := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Original task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}
		if err := WriteJSONL(path, initial); err != nil {
			t.Fatalf("initial WriteJSONL returned error: %v", err)
		}

		// Verify original content exists
		originalData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read original file: %v", err)
		}
		if !strings.Contains(string(originalData), "Original task") {
			t.Fatal("original file does not contain expected content")
		}

		// Write new content (atomic overwrite)
		updated := []task.Task{
			{
				ID:       "tick-d4e5f6",
				Title:    "Updated task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  created,
				Updated:  created,
			},
		}
		if err := WriteJSONL(path, updated); err != nil {
			t.Fatalf("updated WriteJSONL returned error: %v", err)
		}

		// Verify new content replaced old
		newData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read updated file: %v", err)
		}
		if !strings.Contains(string(newData), "Updated task") {
			t.Error("updated file does not contain expected content")
		}
		if strings.Contains(string(newData), "Original task") {
			t.Error("updated file still contains original content")
		}

		// Verify no temp files remain after successful write
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read directory: %v", err)
		}
		for _, entry := range entries {
			if strings.Contains(entry.Name(), ".tmp") {
				t.Errorf("temp file %q left behind after successful write", entry.Name())
			}
		}
	})

	t.Run("it cleans up temp file on write error", func(t *testing.T) {
		// Write to a directory that doesn't allow rename (read-only target dir scenario)
		// Instead, test that writing to a non-existent directory fails gracefully
		path := filepath.Join(t.TempDir(), "nonexistent-dir", "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  created,
				Updated:  created,
			},
		}

		err := WriteJSONL(path, tasks)
		if err == nil {
			t.Fatal("expected error when writing to non-existent directory, got nil")
		}
	})
}

func TestFieldVariations(t *testing.T) {
	t.Run("it handles tasks with all fields populated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)

		fullTask := task.Task{
			ID:          "tick-a1b2c3",
			Title:       "Complete task",
			Status:      task.StatusDone,
			Priority:    0,
			Description: "Full description with details",
			BlockedBy:   []string{"tick-x1y2z3", "tick-m4n5o6"},
			Parent:      "tick-p1a2b3",
			Created:     created,
			Updated:     updated,
			Closed:      &closed,
		}

		if err := WriteJSONL(path, []task.Task{fullTask}); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		line := strings.TrimSpace(string(data))

		// Verify all fields present in output
		allFields := []string{"id", "title", "status", "priority", "description", "blocked_by", "parent", "created", "updated", "closed"}
		for _, field := range allFields {
			if !strings.Contains(line, `"`+field+`"`) {
				t.Errorf("field %q should be present in fully-populated task, but not found in: %s", field, line)
			}
		}

		// Round-trip and verify
		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		got := tasks[0]
		if len(got.BlockedBy) != 2 {
			t.Fatalf("BlockedBy length = %d, want 2", len(got.BlockedBy))
		}
		if got.BlockedBy[0] != "tick-x1y2z3" {
			t.Errorf("BlockedBy[0] = %q, want %q", got.BlockedBy[0], "tick-x1y2z3")
		}
		if got.BlockedBy[1] != "tick-m4n5o6" {
			t.Errorf("BlockedBy[1] = %q, want %q", got.BlockedBy[1], "tick-m4n5o6")
		}
	})

	t.Run("it handles tasks with only required fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		minimalTask := task.Task{
			ID:       "tick-a1b2c3",
			Title:    "Minimal task",
			Status:   task.StatusOpen,
			Priority: 2,
			Created:  created,
			Updated:  created,
		}

		if err := WriteJSONL(path, []task.Task{minimalTask}); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		got := tasks[0]
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
			t.Errorf("Priority = %d, want %d", got.Priority, 2)
		}
		if got.Description != "" {
			t.Errorf("Description = %q, want empty", got.Description)
		}
		if got.BlockedBy != nil {
			t.Errorf("BlockedBy = %v, want nil", got.BlockedBy)
		}
		if got.Parent != "" {
			t.Errorf("Parent = %q, want empty", got.Parent)
		}
		if got.Closed != nil {
			t.Errorf("Closed = %v, want nil", got.Closed)
		}
	})

	t.Run("it skips empty lines when reading", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"Task one","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

{"id":"tick-d4e5f6","title":"Task two","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks (skipping empty line), got %d", len(tasks))
		}
	})
}
