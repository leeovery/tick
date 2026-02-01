package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func initTickDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	app := NewApp(&stdout, &stderr)
	code := app.Run([]string{"tick", "init"}, dir)
	if code != 0 {
		t.Fatalf("init failed: %s", stderr.String())
	}
	return dir
}

func createTask(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	app := NewApp(&outBuf, &errBuf)
	cmdArgs := append([]string{"tick", "create"}, args...)
	c := app.Run(cmdArgs, dir)
	return outBuf.String(), errBuf.String(), c
}

func TestCreateCommand(t *testing.T) {
	t.Run("creates a task with only a title", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, stderr, code := createTask(t, dir, "Setup authentication")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "tick-") {
			t.Errorf("output should contain task ID, got %q", stdout)
		}
		if !strings.Contains(stdout, "Setup authentication") {
			t.Errorf("output should contain title, got %q", stdout)
		}
	})

	t.Run("sets status to open on creation", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, code := createTask(t, dir, "Test task")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "open") {
			t.Errorf("output should show status 'open', got %q", stdout)
		}
	})

	t.Run("sets default priority to 2", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, _ := createTask(t, dir, "Test task")
		// Check JSONL file for priority
		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"priority":2`) {
			t.Errorf("expected priority 2 in JSONL, got %q", string(content))
		}
		_ = stdout
	})

	t.Run("sets priority from --priority flag", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := createTask(t, dir, "Urgent task", "--priority", "0")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
		}
		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"priority":0`) {
			t.Errorf("expected priority 0 in JSONL, got %q", string(content))
		}
	})

	t.Run("rejects priority outside 0-4 range", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := createTask(t, dir, "Test", "--priority", "5")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "priority") {
			t.Errorf("stderr should mention priority, got %q", stderr)
		}
	})

	t.Run("sets description from --description flag", func(t *testing.T) {
		dir := initTickDir(t)
		_, _, code := createTask(t, dir, "Test", "--description", "Detailed info")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"description":"Detailed info"`) {
			t.Errorf("expected description in JSONL, got %q", string(content))
		}
	})

	t.Run("sets blocked_by from --blocked-by flag", func(t *testing.T) {
		dir := initTickDir(t)
		// Create first task
		out1, _, _ := createTask(t, dir, "First task")
		id1 := extractID(t, out1)

		// Create second task blocked by first
		_, stderr, code := createTask(t, dir, "Second task", "--blocked-by", id1)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
		}
		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"blocked_by":["`) {
			t.Errorf("expected blocked_by in JSONL, got %q", string(content))
		}
	})

	t.Run("sets blocked_by from --blocked-by with multiple IDs", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "First")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Second")
		id2 := extractID(t, out2)

		_, stderr, code := createTask(t, dir, "Third", "--blocked-by", id1+","+id2)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
		}
	})

	t.Run("updates target tasks' blocked_by when --blocks is used", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Target task")
		id1 := extractID(t, out1)

		_, stderr, code := createTask(t, dir, "Blocker task", "--blocks", id1)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
		}

		// Target task should now have the blocker in its blocked_by
		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		contentStr := string(content)
		// Check that target task has blocked_by
		if !strings.Contains(contentStr, `"blocked_by":[`) {
			t.Errorf("target should have blocked_by, content: %q", contentStr)
		}
	})

	t.Run("sets parent from --parent flag", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent task")
		id1 := extractID(t, out1)

		_, stderr, code := createTask(t, dir, "Child task", "--parent", id1)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
		}
		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"parent":"`+id1+`"`) {
			t.Errorf("expected parent in JSONL, got %q", string(content))
		}
	})

	t.Run("errors when title is missing", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := createTask(t, dir)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "Title is required") {
			t.Errorf("stderr should mention title required, got %q", stderr)
		}
	})

	t.Run("errors when title is empty string", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := createTask(t, dir, "")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(strings.ToLower(stderr), "title") {
			t.Errorf("stderr should mention title, got %q", stderr)
		}
	})

	t.Run("errors when --blocked-by references non-existent task", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := createTask(t, dir, "Test", "--blocked-by", "tick-nonexist")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should mention not found, got %q", stderr)
		}
	})

	t.Run("errors when --parent references non-existent task", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := createTask(t, dir, "Test", "--parent", "tick-nonexist")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should mention not found, got %q", stderr)
		}
	})

	t.Run("persists task to tasks.jsonl", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Persisted task")

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), "Persisted task") {
			t.Errorf("task should be persisted in JSONL, got %q", string(content))
		}
	})

	t.Run("outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := initTickDir(t)
		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		code := app.Run([]string{"tick", "--quiet", "create", "Quiet task"}, dir)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, errBuf.String())
		}
		output := strings.TrimSpace(outBuf.String())
		if !strings.HasPrefix(output, "tick-") || len(output) != 11 {
			t.Errorf("--quiet should output only task ID (tick-XXXXXX), got %q", output)
		}
	})

	t.Run("normalizes input IDs to lowercase", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "First")
		id1 := extractID(t, out1)
		upperId := strings.ToUpper(id1)

		_, stderr, code := createTask(t, dir, "Second", "--blocked-by", upperId)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
		}
	})

	t.Run("trims whitespace from title", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, code := createTask(t, dir, "  Spaced Title  ")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "Spaced Title") {
			t.Errorf("title should be trimmed, got %q", stdout)
		}
		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if strings.Contains(string(content), "  Spaced") {
			t.Error("title in JSONL should be trimmed")
		}
	})
}

// extractID pulls the tick-XXXXXX ID from command output.
// Handles multiple output formats: key-value (ID: tick-xxx), TOON (tick-xxx,...), JSON, etc.
func extractID(t *testing.T, output string) string {
	t.Helper()
	// Look for tick-XXXXXX pattern anywhere in the output.
	for _, line := range strings.Split(output, "\n") {
		// Split by common delimiters: spaces, commas, colons.
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == ' ' || r == ',' || r == ':' || r == '\t' || r == '"'
		})
		for _, p := range parts {
			cleaned := strings.Trim(p, "(),[]{}:")
			if strings.HasPrefix(cleaned, "tick-") && len(cleaned) == 11 {
				return cleaned
			}
		}
	}
	t.Fatalf("could not extract task ID from output: %q", output)
	return ""
}
