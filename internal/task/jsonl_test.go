package task

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWriteJSONL(t *testing.T) {
	t.Run("it omits optional fields when empty", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Minimal task",
				Status:   StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
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

		// Optional fields must NOT appear in the output
		optionalFields := []string{`"description"`, `"blocked_by"`, `"parent"`, `"closed"`}
		for _, field := range optionalFields {
			if strings.Contains(line, field) {
				t.Errorf("expected optional field %s to be omitted, but found in: %s", field, line)
			}
		}

		// Required fields must appear
		requiredFields := []string{`"id"`, `"title"`, `"status"`, `"priority"`, `"created"`, `"updated"`}
		for _, field := range requiredFields {
			if !strings.Contains(line, field) {
				t.Errorf("expected required field %s to be present, but not found in: %s", field, line)
			}
		}
	})

	t.Run("it writes atomically via temp file and rename", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

		// Write initial content
		initial := []Task{
			{
				ID:       "tick-aaaaaa",
				Title:    "Original task",
				Status:   StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}
		if err := WriteJSONL(path, initial); err != nil {
			t.Fatalf("initial write failed: %v", err)
		}

		// Overwrite with new content (full rewrite, not append)
		updated := []Task{
			{
				ID:       "tick-bbbbbb",
				Title:    "Replacement task",
				Status:   StatusDone,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
		}
		if err := WriteJSONL(path, updated); err != nil {
			t.Fatalf("overwrite failed: %v", err)
		}

		// Verify only new content exists (full rewrite, not append)
		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task after overwrite, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-bbbbbb" {
			t.Errorf("expected task ID tick-bbbbbb, got %s", tasks[0].ID)
		}

		// Verify no temp files left behind
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read directory: %v", err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".tasks-") && strings.HasSuffix(entry.Name(), ".tmp") {
				t.Errorf("temp file left behind: %s", entry.Name())
			}
		}
	})

	t.Run("it outputs fields in spec order", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		tasks := []Task{
			{
				ID:          "tick-aabbcc",
				Title:       "Ordered fields",
				Status:      StatusDone,
				Priority:    1,
				Description: "desc",
				BlockedBy:   []string{"tick-112233"},
				Parent:      "tick-445566",
				Created:     now,
				Updated:     now,
				Closed:      &closed,
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

		// Verify field ordering: id, title, status, priority, then optional fields
		// The JSON keys should appear in this order
		fields := []string{`"id"`, `"title"`, `"status"`, `"priority"`, `"description"`, `"blocked_by"`, `"parent"`, `"created"`, `"updated"`, `"closed"`}
		lastIdx := -1
		for _, field := range fields {
			idx := strings.Index(line, field)
			if idx == -1 {
				t.Errorf("field %s not found in output: %s", field, line)
				continue
			}
			if idx <= lastIdx {
				t.Errorf("field %s (at %d) should appear after previous field (at %d) in: %s", field, idx, lastIdx, line)
			}
			lastIdx = idx
		}
	})

	t.Run("it writes tasks as one JSON object per line", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "First task",
				Status:   StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			{
				ID:       "tick-d4e5f6",
				Title:    "Second task",
				Status:   StatusDone,
				Priority: 1,
				Created:  now,
				Updated:  now,
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

		// Each line should be a single JSON object (no pretty-printing)
		for i, line := range lines {
			if strings.Contains(line, "\n") {
				t.Errorf("line %d contains newline (pretty-printed)", i)
			}
			if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
				t.Errorf("line %d is not a JSON object: %s", i, line)
			}
		}
	})
}

func TestReadJSONL(t *testing.T) {
	t.Run("it round-trips all task fields without loss", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		original := []Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      StatusInProgress,
				Priority:    1,
				Description: "Details here",
				BlockedBy:   []string{"tick-x1y2z3"},
				Parent:      "tick-e5f6a7",
				Created:     now,
				Updated:     now,
				Closed:      &closed,
			},
		}

		if err := WriteJSONL(path, original); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		readBack, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(readBack) != 1 {
			t.Fatalf("expected 1 task, got %d", len(readBack))
		}

		got := readBack[0]
		want := original[0]

		if got.ID != want.ID {
			t.Errorf("ID: got %q, want %q", got.ID, want.ID)
		}
		if got.Title != want.Title {
			t.Errorf("Title: got %q, want %q", got.Title, want.Title)
		}
		if got.Status != want.Status {
			t.Errorf("Status: got %q, want %q", got.Status, want.Status)
		}
		if got.Priority != want.Priority {
			t.Errorf("Priority: got %d, want %d", got.Priority, want.Priority)
		}
		if got.Description != want.Description {
			t.Errorf("Description: got %q, want %q", got.Description, want.Description)
		}
		if !reflect.DeepEqual(got.BlockedBy, want.BlockedBy) {
			t.Errorf("BlockedBy: got %v, want %v", got.BlockedBy, want.BlockedBy)
		}
		if got.Parent != want.Parent {
			t.Errorf("Parent: got %q, want %q", got.Parent, want.Parent)
		}
		if !got.Created.Equal(want.Created) {
			t.Errorf("Created: got %v, want %v", got.Created, want.Created)
		}
		if !got.Updated.Equal(want.Updated) {
			t.Errorf("Updated: got %v, want %v", got.Updated, want.Updated)
		}
		if got.Closed == nil {
			t.Fatal("Closed: got nil, want non-nil")
		}
		if !got.Closed.Equal(*want.Closed) {
			t.Errorf("Closed: got %v, want %v", got.Closed, want.Closed)
		}
	})

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

		// Ensure it's an empty slice, not nil
		if tasks == nil {
			t.Error("expected empty slice, got nil")
		}
	})

	t.Run("it returns error for missing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nonexistent.jsonl")

		_, err := ReadJSONL(path)
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("it handles tasks with all fields populated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		tasks := []Task{
			{
				ID:          "tick-c3d4e5",
				Title:       "With all fields",
				Status:      StatusDone,
				Priority:    0,
				Description: "Full description with details",
				BlockedBy:   []string{"tick-a1b2c3", "tick-x9y8z7"},
				Parent:      "tick-e5f6a7",
				Created:     now,
				Updated:     updated,
				Closed:      &closed,
			},
		}

		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		readBack, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(readBack) != 1 {
			t.Fatalf("expected 1 task, got %d", len(readBack))
		}

		got := readBack[0]
		if got.Description != "Full description with details" {
			t.Errorf("Description: got %q, want %q", got.Description, "Full description with details")
		}
		if len(got.BlockedBy) != 2 {
			t.Fatalf("expected 2 blocked_by entries, got %d", len(got.BlockedBy))
		}
		if got.BlockedBy[0] != "tick-a1b2c3" || got.BlockedBy[1] != "tick-x9y8z7" {
			t.Errorf("BlockedBy: got %v, want [tick-a1b2c3 tick-x9y8z7]", got.BlockedBy)
		}
		if got.Parent != "tick-e5f6a7" {
			t.Errorf("Parent: got %q, want %q", got.Parent, "tick-e5f6a7")
		}
		if got.Closed == nil || !got.Closed.Equal(closed) {
			t.Errorf("Closed: got %v, want %v", got.Closed, closed)
		}

		// Verify the JSON contains all expected fields
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		line := strings.TrimSpace(string(data))
		expectedFields := []string{`"description"`, `"blocked_by"`, `"parent"`, `"closed"`}
		for _, field := range expectedFields {
			if !strings.Contains(line, field) {
				t.Errorf("expected field %s to be present in: %s", field, line)
			}
		}
	})

	t.Run("it handles tasks with only required fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []Task{
			{
				ID:       "tick-f6a7b8",
				Title:    "Minimal task",
				Status:   StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}

		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		readBack, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(readBack) != 1 {
			t.Fatalf("expected 1 task, got %d", len(readBack))
		}

		got := readBack[0]
		if got.ID != "tick-f6a7b8" {
			t.Errorf("ID: got %q, want %q", got.ID, "tick-f6a7b8")
		}
		if got.Title != "Minimal task" {
			t.Errorf("Title: got %q, want %q", got.Title, "Minimal task")
		}
		if got.Status != StatusOpen {
			t.Errorf("Status: got %q, want %q", got.Status, StatusOpen)
		}
		if got.Priority != 2 {
			t.Errorf("Priority: got %d, want %d", got.Priority, 2)
		}
		if got.Description != "" {
			t.Errorf("Description: got %q, want empty", got.Description)
		}
		if got.BlockedBy != nil {
			t.Errorf("BlockedBy: got %v, want nil", got.BlockedBy)
		}
		if got.Parent != "" {
			t.Errorf("Parent: got %q, want empty", got.Parent)
		}
		if got.Closed != nil {
			t.Errorf("Closed: got %v, want nil", got.Closed)
		}
	})

	t.Run("it skips empty lines in JSONL file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks (skipping empty lines), got %d", len(tasks))
		}
	})

	t.Run("it reads tasks from JSONL file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-a1b2c3" {
			t.Errorf("expected first task ID tick-a1b2c3, got %s", tasks[0].ID)
		}
		if tasks[1].ID != "tick-d4e5f6" {
			t.Errorf("expected second task ID tick-d4e5f6, got %s", tasks[1].ID)
		}
	})
}

func TestReadJSONLFromBytes(t *testing.T) {
	t.Run("it parses tasks from in-memory bytes", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		content := []byte(`{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`)

		tasks, err := ReadJSONLFromBytes(content)
		if err != nil {
			t.Fatalf("ReadJSONLFromBytes returned error: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-a1b2c3" {
			t.Errorf("expected first task ID tick-a1b2c3, got %s", tasks[0].ID)
		}
		if !tasks[0].Created.Equal(now) {
			t.Errorf("expected first task Created %v, got %v", now, tasks[0].Created)
		}
		if tasks[1].ID != "tick-d4e5f6" {
			t.Errorf("expected second task ID tick-d4e5f6, got %s", tasks[1].ID)
		}
	})

	t.Run("it returns empty list for empty bytes", func(t *testing.T) {
		tasks, err := ReadJSONLFromBytes([]byte{})
		if err != nil {
			t.Fatalf("ReadJSONLFromBytes returned error for empty bytes: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks for empty bytes, got %d", len(tasks))
		}
		if tasks == nil {
			t.Error("expected empty slice, got nil")
		}
	})

	t.Run("it skips empty lines", func(t *testing.T) {
		content := []byte(`{"id":"tick-a1b2c3","title":"First","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

{"id":"tick-d4e5f6","title":"Second","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

`)
		tasks, err := ReadJSONLFromBytes(content)
		if err != nil {
			t.Fatalf("ReadJSONLFromBytes returned error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks (skipping empty lines), got %d", len(tasks))
		}
	})

	t.Run("it returns error for invalid JSON", func(t *testing.T) {
		content := []byte(`{"id":"tick-a1b2c3","title":"First","status":"open"
not-json
`)
		_, err := ReadJSONLFromBytes(content)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})

	t.Run("it produces same results as ReadJSONL for same content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		original := []Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task",
				Status:      StatusInProgress,
				Priority:    1,
				Description: "Details here",
				BlockedBy:   []string{"tick-x1y2z3"},
				Parent:      "tick-e5f6a7",
				Created:     now,
				Updated:     now,
				Closed:      &closed,
			},
		}

		if err := WriteJSONL(path, original); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		fromFile, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL returned error: %v", err)
		}

		fromBytes, err := ReadJSONLFromBytes(data)
		if err != nil {
			t.Fatalf("ReadJSONLFromBytes returned error: %v", err)
		}

		if !reflect.DeepEqual(fromFile, fromBytes) {
			t.Errorf("ReadJSONL and ReadJSONLFromBytes produced different results:\nfromFile:  %+v\nfromBytes: %+v", fromFile, fromBytes)
		}
	})
}

func TestSerializeJSONL(t *testing.T) {
	t.Run("it serializes tasks to JSONL bytes", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "First task",
				Status:   StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			{
				ID:       "tick-d4e5f6",
				Title:    "Second task",
				Status:   StatusDone,
				Priority: 1,
				Created:  now,
				Updated:  now,
			},
		}

		data, err := SerializeJSONL(tasks)
		if err != nil {
			t.Fatalf("SerializeJSONL returned error: %v", err)
		}

		// Verify the bytes can be parsed back
		parsed, err := ReadJSONLFromBytes(data)
		if err != nil {
			t.Fatalf("ReadJSONLFromBytes returned error: %v", err)
		}
		if len(parsed) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(parsed))
		}
		if parsed[0].ID != "tick-a1b2c3" {
			t.Errorf("expected first task ID tick-a1b2c3, got %s", parsed[0].ID)
		}
		if parsed[1].ID != "tick-d4e5f6" {
			t.Errorf("expected second task ID tick-d4e5f6, got %s", parsed[1].ID)
		}
	})

	t.Run("it produces bytes identical to WriteJSONL output", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Test task",
				Status:   StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		}

		if err := WriteJSONL(path, tasks); err != nil {
			t.Fatalf("WriteJSONL returned error: %v", err)
		}

		fileData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		serialized, err := SerializeJSONL(tasks)
		if err != nil {
			t.Fatalf("SerializeJSONL returned error: %v", err)
		}

		if string(fileData) != string(serialized) {
			t.Errorf("SerializeJSONL output differs from WriteJSONL file content:\nfile:       %q\nserialized: %q", string(fileData), string(serialized))
		}
	})

	t.Run("it returns empty bytes for empty task list", func(t *testing.T) {
		data, err := SerializeJSONL([]Task{})
		if err != nil {
			t.Fatalf("SerializeJSONL returned error: %v", err)
		}
		if len(data) != 0 {
			t.Errorf("expected empty bytes for empty task list, got %d bytes", len(data))
		}
	})
}
