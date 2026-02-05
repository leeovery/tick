package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/storage"
)

// setupTickDir creates a temp directory with .tick/tasks.jsonl initialized
func setupTickDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick dir: %v", err)
	}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	return dir
}

// readTasksFromDir reads tasks from .tick/tasks.jsonl in given directory
func readTasksFromDir(t *testing.T, dir string) []struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Description string   `json:"description,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Closed      string   `json:"closed,omitempty"`
} {
	t.Helper()
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	tasks, err := storage.ReadJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}

	// Convert to struct slice
	result := make([]struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Status      string   `json:"status"`
		Priority    int      `json:"priority"`
		Description string   `json:"description,omitempty"`
		BlockedBy   []string `json:"blocked_by,omitempty"`
		Parent      string   `json:"parent,omitempty"`
		Created     string   `json:"created"`
		Updated     string   `json:"updated"`
		Closed      string   `json:"closed,omitempty"`
	}, len(tasks))

	for i, task := range tasks {
		result[i].ID = task.ID
		result[i].Title = task.Title
		result[i].Status = string(task.Status)
		result[i].Priority = task.Priority
		result[i].Description = task.Description
		result[i].BlockedBy = task.BlockedBy
		result[i].Parent = task.Parent
		result[i].Created = task.Created
		result[i].Updated = task.Updated
		result[i].Closed = task.Closed
	}
	return result
}

// setupTask adds a task directly to tasks.jsonl for test setup
func setupTask(t *testing.T, dir, id, title string) {
	t.Helper()
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}

	// Append task line
	taskLine := fmt.Sprintf(`{"id":"%s","title":"%s","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`, id, title)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	content = append(content, []byte(taskLine+"\n")...)

	if err := os.WriteFile(jsonlPath, content, 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
}

// setupTaskWithPriority creates a task with a specific priority and created timestamp
func setupTaskWithPriority(t *testing.T, dir, id, title string, priority int, created string) {
	t.Helper()
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}

	taskLine := `{"id":"` + id + `","title":"` + title + `","status":"open","priority":` + strconv.Itoa(priority) + `,"created":"` + created + `","updated":"` + created + `"}`
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	content = append(content, []byte(taskLine+"\n")...)

	if err := os.WriteFile(jsonlPath, content, 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
}

// setupTaskFull creates a task with all fields for testing show command
func setupTaskFull(t *testing.T, dir, id, title, status string, priority int, description, parent string, blockedBy []string, created, updated, closed string) {
	t.Helper()
	jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}

	// Build JSON manually to handle optional fields
	taskJSON := `{"id":"` + id + `","title":"` + escapeJSON(title) + `","status":"` + status + `","priority":` + strconv.Itoa(priority)

	if description != "" {
		taskJSON += `,"description":"` + escapeJSON(description) + `"`
	}
	if len(blockedBy) > 0 {
		taskJSON += `,"blocked_by":["` + strings.Join(blockedBy, `","`) + `"]`
	}
	if parent != "" {
		taskJSON += `,"parent":"` + parent + `"`
	}
	taskJSON += `,"created":"` + created + `","updated":"` + updated + `"`
	if closed != "" {
		taskJSON += `,"closed":"` + closed + `"`
	}
	taskJSON += "}"

	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	content = append(content, []byte(taskJSON+"\n")...)

	if err := os.WriteFile(jsonlPath, content, 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
}

// escapeJSON escapes special characters in JSON strings
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
