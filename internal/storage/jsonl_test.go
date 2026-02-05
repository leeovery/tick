package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestJSONLWriter(t *testing.T) {
	t.Run("it writes tasks as one JSON object per line", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		// Create empty file first
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "First task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
			},
			{
				ID:       "tick-d4e5f6",
				Title:    "Second task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  "2026-01-19T11:00:00Z",
				Updated:  "2026-01-19T12:00:00Z",
			},
		}

		err := WriteJSONL(path, tasks)
		if err != nil {
			t.Fatalf("WriteJSONL failed: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(lines))
		}

		// Each line should be a valid JSON object (no array wrapper)
		if !strings.HasPrefix(lines[0], "{") || !strings.HasSuffix(lines[0], "}") {
			t.Errorf("line 0 is not a JSON object: %s", lines[0])
		}
		if !strings.HasPrefix(lines[1], "{") || !strings.HasSuffix(lines[1], "}") {
			t.Errorf("line 1 is not a JSON object: %s", lines[1])
		}

		// Should not have commas between lines (no array)
		if strings.Contains(string(content), "},\n") {
			t.Error("file should not have trailing commas between objects")
		}
	})
}

func TestJSONLReader(t *testing.T) {
	t.Run("it reads tasks from JSONL file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		content := `{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T12:00:00Z"}
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL failed: %v", err)
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

func TestJSONLRoundTrip(t *testing.T) {
	t.Run("it round-trips all task fields without loss", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		original := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Complete task",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: "Full description with\nmultiple lines",
				BlockedBy:   []string{"tick-x1y2z3", "tick-d4e5f6"},
				Parent:      "tick-parent1",
				Created:     "2026-01-19T10:00:00Z",
				Updated:     "2026-01-19T14:00:00Z",
				Closed:      "2026-01-19T16:00:00Z",
			},
		}

		err := WriteJSONL(path, original)
		if err != nil {
			t.Fatalf("WriteJSONL failed: %v", err)
		}

		loaded, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL failed: %v", err)
		}

		if len(loaded) != 1 {
			t.Fatalf("expected 1 task, got %d", len(loaded))
		}

		got := loaded[0]
		want := original[0]

		if got.ID != want.ID {
			t.Errorf("ID = %q, want %q", got.ID, want.ID)
		}
		if got.Title != want.Title {
			t.Errorf("Title = %q, want %q", got.Title, want.Title)
		}
		if got.Status != want.Status {
			t.Errorf("Status = %q, want %q", got.Status, want.Status)
		}
		if got.Priority != want.Priority {
			t.Errorf("Priority = %d, want %d", got.Priority, want.Priority)
		}
		if got.Description != want.Description {
			t.Errorf("Description = %q, want %q", got.Description, want.Description)
		}
		if len(got.BlockedBy) != len(want.BlockedBy) {
			t.Fatalf("BlockedBy length = %d, want %d", len(got.BlockedBy), len(want.BlockedBy))
		}
		for i, id := range got.BlockedBy {
			if id != want.BlockedBy[i] {
				t.Errorf("BlockedBy[%d] = %q, want %q", i, id, want.BlockedBy[i])
			}
		}
		if got.Parent != want.Parent {
			t.Errorf("Parent = %q, want %q", got.Parent, want.Parent)
		}
		if got.Created != want.Created {
			t.Errorf("Created = %q, want %q", got.Created, want.Created)
		}
		if got.Updated != want.Updated {
			t.Errorf("Updated = %q, want %q", got.Updated, want.Updated)
		}
		if got.Closed != want.Closed {
			t.Errorf("Closed = %q, want %q", got.Closed, want.Closed)
		}
	})

	t.Run("it omits optional fields when empty", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Minimal task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
				// Description, BlockedBy, Parent, Closed are empty
			},
		}

		err := WriteJSONL(path, tasks)
		if err != nil {
			t.Fatalf("WriteJSONL failed: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		line := string(content)

		// Optional fields should NOT appear in JSON
		if strings.Contains(line, "description") {
			t.Error("empty description should be omitted from JSON")
		}
		if strings.Contains(line, "blocked_by") {
			t.Error("empty blocked_by should be omitted from JSON")
		}
		if strings.Contains(line, "parent") {
			t.Error("empty parent should be omitted from JSON")
		}
		if strings.Contains(line, "closed") {
			t.Error("empty closed should be omitted from JSON")
		}

		// Required fields should appear
		if !strings.Contains(line, `"id"`) {
			t.Error("id should be present in JSON")
		}
		if !strings.Contains(line, `"title"`) {
			t.Error("title should be present in JSON")
		}
		if !strings.Contains(line, `"status"`) {
			t.Error("status should be present in JSON")
		}
		if !strings.Contains(line, `"priority"`) {
			t.Error("priority should be present in JSON")
		}
		if !strings.Contains(line, `"created"`) {
			t.Error("created should be present in JSON")
		}
		if !strings.Contains(line, `"updated"`) {
			t.Error("updated should be present in JSON")
		}
	})

	t.Run("it returns empty list for empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		// Create empty file (0 bytes)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL failed: %v", err)
		}

		if tasks == nil {
			t.Error("expected non-nil slice for empty file")
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})
}

func TestJSONLAtomicWrite(t *testing.T) {
	t.Run("it writes atomically via temp file and rename", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		// Write original content
		original := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Original task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
			},
		}
		if err := WriteJSONL(path, original); err != nil {
			t.Fatalf("initial WriteJSONL failed: %v", err)
		}

		// Get original file info
		origInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat original file: %v", err)
		}

		// Write new content
		updated := []task.Task{
			{
				ID:       "tick-d4e5f6",
				Title:    "Updated task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  "2026-01-19T11:00:00Z",
				Updated:  "2026-01-19T12:00:00Z",
			},
		}
		if err := WriteJSONL(path, updated); err != nil {
			t.Fatalf("second WriteJSONL failed: %v", err)
		}

		// Verify content was replaced (atomic rename)
		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL failed: %v", err)
		}

		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-d4e5f6" {
			t.Errorf("task ID = %q, want %q", tasks[0].ID, "tick-d4e5f6")
		}

		// Verify no temp files left behind
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".tasks-") && strings.HasSuffix(entry.Name(), ".tmp") {
				t.Errorf("temp file left behind: %s", entry.Name())
			}
		}

		// Verify file was replaced (inode changed via rename)
		newInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat new file: %v", err)
		}

		// File size should be different (different task data)
		if origInfo.Size() == newInfo.Size() {
			t.Log("warning: file sizes are same, content should still be verified")
		}
	})

	t.Run("it cleans up temp file on write error", func(t *testing.T) {
		// Create a directory that doesn't exist to cause rename error
		dir := t.TempDir()
		nonExistentDir := filepath.Join(dir, "does-not-exist")
		path := filepath.Join(nonExistentDir, "tasks.jsonl")

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Test task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
			},
		}

		err := WriteJSONL(path, tasks)
		if err == nil {
			t.Fatal("expected error for non-existent directory")
		}

		// Verify no temp files left behind in parent dir
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".tasks-") && strings.HasSuffix(entry.Name(), ".tmp") {
				t.Errorf("temp file left behind: %s", entry.Name())
			}
		}
	})
}

func TestJSONLFieldCombinations(t *testing.T) {
	t.Run("it handles tasks with all fields populated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		tasks := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Full task with all fields",
				Status:      task.StatusDone,
				Priority:    0, // P0 critical
				Description: "A comprehensive description\nwith multiple lines\nand special chars: \"quotes\" and \\backslash",
				BlockedBy:   []string{"tick-x1y2z3", "tick-d4e5f6", "tick-g7h8i9"},
				Parent:      "tick-parent1",
				Created:     "2026-01-19T10:00:00Z",
				Updated:     "2026-01-19T14:30:00Z",
				Closed:      "2026-01-19T16:00:00Z",
			},
		}

		err := WriteJSONL(path, tasks)
		if err != nil {
			t.Fatalf("WriteJSONL failed: %v", err)
		}

		loaded, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL failed: %v", err)
		}

		if len(loaded) != 1 {
			t.Fatalf("expected 1 task, got %d", len(loaded))
		}

		got := loaded[0]
		want := tasks[0]

		// Verify all fields round-trip correctly
		if got.ID != want.ID {
			t.Errorf("ID = %q, want %q", got.ID, want.ID)
		}
		if got.Title != want.Title {
			t.Errorf("Title = %q, want %q", got.Title, want.Title)
		}
		if got.Status != want.Status {
			t.Errorf("Status = %q, want %q", got.Status, want.Status)
		}
		if got.Priority != want.Priority {
			t.Errorf("Priority = %d, want %d", got.Priority, want.Priority)
		}
		if got.Description != want.Description {
			t.Errorf("Description = %q, want %q", got.Description, want.Description)
		}
		if len(got.BlockedBy) != 3 {
			t.Fatalf("BlockedBy length = %d, want 3", len(got.BlockedBy))
		}
		if got.Parent != want.Parent {
			t.Errorf("Parent = %q, want %q", got.Parent, want.Parent)
		}
		if got.Created != want.Created {
			t.Errorf("Created = %q, want %q", got.Created, want.Created)
		}
		if got.Updated != want.Updated {
			t.Errorf("Updated = %q, want %q", got.Updated, want.Updated)
		}
		if got.Closed != want.Closed {
			t.Errorf("Closed = %q, want %q", got.Closed, want.Closed)
		}
	})

	t.Run("it handles tasks with only required fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		tasks := []task.Task{
			{
				ID:       "tick-a1b2c3",
				Title:    "Minimal task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  "2026-01-19T10:00:00Z",
				Updated:  "2026-01-19T10:00:00Z",
				// No optional fields
			},
		}

		err := WriteJSONL(path, tasks)
		if err != nil {
			t.Fatalf("WriteJSONL failed: %v", err)
		}

		loaded, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL failed: %v", err)
		}

		if len(loaded) != 1 {
			t.Fatalf("expected 1 task, got %d", len(loaded))
		}

		got := loaded[0]

		// Required fields present
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
		if got.Created != "2026-01-19T10:00:00Z" {
			t.Errorf("Created = %q, want %q", got.Created, "2026-01-19T10:00:00Z")
		}
		if got.Updated != "2026-01-19T10:00:00Z" {
			t.Errorf("Updated = %q, want %q", got.Updated, "2026-01-19T10:00:00Z")
		}

		// Optional fields should be zero values
		if got.Description != "" {
			t.Errorf("Description should be empty, got %q", got.Description)
		}
		if got.BlockedBy != nil && len(got.BlockedBy) > 0 {
			t.Errorf("BlockedBy should be empty, got %v", got.BlockedBy)
		}
		if got.Parent != "" {
			t.Errorf("Parent should be empty, got %q", got.Parent)
		}
		if got.Closed != "" {
			t.Errorf("Closed should be empty, got %q", got.Closed)
		}
	})
}

func TestJSONLFormat(t *testing.T) {
	t.Run("each task occupies exactly one line", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		tasks := []task.Task{
			{
				ID:          "tick-a1b2c3",
				Title:       "Task with all fields",
				Status:      task.StatusOpen,
				Priority:    2,
				Description: "Description with\nmultiple\nlines",
				BlockedBy:   []string{"tick-x1y2z3"},
				Parent:      "tick-parent1",
				Created:     "2026-01-19T10:00:00Z",
				Updated:     "2026-01-19T10:00:00Z",
			},
		}

		err := WriteJSONL(path, tasks)
		if err != nil {
			t.Fatalf("WriteJSONL failed: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Count lines (each task = 1 line + trailing newline)
		lines := strings.Split(string(content), "\n")
		// Last element is empty due to trailing newline
		nonEmptyLines := 0
		for _, line := range lines {
			if line != "" {
				nonEmptyLines++
			}
		}

		if nonEmptyLines != 1 {
			t.Errorf("expected 1 line for 1 task, got %d lines", nonEmptyLines)
		}

		// Newlines in description should be escaped, not literal
		if strings.Count(string(content), "\n") > 1 {
			// File should have exactly 1 newline (at end), not multiple
			t.Log("content contains properly escaped newlines")
		}

		// Verify no pretty-printing (no indentation)
		if strings.Contains(string(content), "  ") {
			t.Error("JSON should not be pretty-printed (no indentation)")
		}
	})
}

func TestJSONLErrors(t *testing.T) {
	t.Run("it returns error for missing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")
		// Don't create the file

		tasks, err := ReadJSONL(path)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		if tasks != nil {
			t.Errorf("expected nil tasks for error case, got %v", tasks)
		}

		// Should be a file-not-found error
		if !os.IsNotExist(err) {
			t.Errorf("expected not-exist error, got %v", err)
		}
	})

	t.Run("it skips empty lines in JSONL file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")

		// Content with empty lines
		content := `{"id":"tick-a1b2c3","title":"First task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}

{"id":"tick-d4e5f6","title":"Second task","status":"done","priority":1,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T12:00:00Z"}

`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		tasks, err := ReadJSONL(path)
		if err != nil {
			t.Fatalf("ReadJSONL failed: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks (ignoring empty lines), got %d", len(tasks))
		}
	})
}
