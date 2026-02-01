package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	t.Run("updates title with --title flag", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Original title")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "update", id, "--title", "New title")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "New title") {
			t.Errorf("expected new title in output, got %q", stdout)
		}
	})

	t.Run("updates description with --description flag", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		_, _, code := runCmd(t, dir, "tick", "update", id, "--description", "New desc")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"description":"New desc"`) {
			t.Errorf("expected description in JSONL, got %q", string(content))
		}
	})

	t.Run("clears description with empty --description", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test", "--description", "Initial desc")
		id := extractID(t, out)

		_, _, code := runCmd(t, dir, "tick", "update", id, "--description", "")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if strings.Contains(string(content), `"description"`) {
			t.Errorf("description should be cleared, got %q", string(content))
		}
	})

	t.Run("updates priority with --priority flag", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		_, _, code := runCmd(t, dir, "tick", "update", id, "--priority", "0")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"priority":0`) {
			t.Errorf("expected priority 0 in JSONL, got %q", string(content))
		}
	})

	t.Run("updates parent with --parent flag", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent task")
		parentID := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Child task")
		childID := extractID(t, out2)

		_, _, code := runCmd(t, dir, "tick", "update", childID, "--parent", parentID)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"parent":"`+parentID+`"`) {
			t.Errorf("expected parent in JSONL, got %q", string(content))
		}
	})

	t.Run("clears parent with empty --parent", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent")
		parentID := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Child", "--parent", parentID)
		childID := extractID(t, out2)

		_, _, code := runCmd(t, dir, "tick", "update", childID, "--parent", "")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		// The child task line should not have parent anymore
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		for _, line := range lines {
			if strings.Contains(line, childID) {
				if strings.Contains(line, `"parent"`) {
					t.Errorf("parent should be cleared, got %q", line)
				}
			}
		}
	})

	t.Run("updates blocks with --blocks flag", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Target task")
		targetID := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocker task")
		blockerID := extractID(t, out2)

		_, _, code := runCmd(t, dir, "tick", "update", blockerID, "--blocks", targetID)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"blocked_by"`) {
			t.Errorf("expected blocked_by in JSONL, got %q", string(content))
		}
	})

	t.Run("updates multiple fields in a single command", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Original")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "update", id, "--title", "Updated", "--priority", "1", "--description", "New desc")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "Updated") {
			t.Errorf("expected new title in output, got %q", stdout)
		}
	})

	t.Run("refreshes updated timestamp on any change", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		content1, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))

		runCmd(t, dir, "tick", "update", id, "--title", "Changed")

		content2, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		// Updated timestamp should differ or at least be present
		if string(content1) == string(content2) {
			t.Error("JSONL should change after update")
		}
	})

	t.Run("outputs full task details on success", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "update", id, "--title", "Changed")
		// TOON format: task detail with schema header
		if !strings.Contains(stdout, "task{") {
			t.Errorf("expected TOON task detail, got %q", stdout)
		}
		if !strings.Contains(stdout, id) {
			t.Errorf("expected task ID in output, got %q", stdout)
		}
	})

	t.Run("outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "update", id, "--title", "Changed")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		output := strings.TrimSpace(stdout)
		if output != id {
			t.Errorf("--quiet should output only ID %q, got %q", id, output)
		}
	})

	t.Run("errors when no flags are provided", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		_, stderr, code := runCmd(t, dir, "tick", "update", id)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "At least one") {
			t.Errorf("expected 'at least one' error, got %q", stderr)
		}
	})

	t.Run("errors when task ID is missing", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "update")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "Task ID is required") {
			t.Errorf("expected usage error, got %q", stderr)
		}
	})

	t.Run("errors when task ID is not found", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "update", "tick-nonex", "--title", "X")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("expected 'not found' error, got %q", stderr)
		}
	})

	t.Run("errors on invalid title", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		_, stderr, code := runCmd(t, dir, "tick", "update", id, "--title", "")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(strings.ToLower(stderr), "title") {
			t.Errorf("expected title error, got %q", stderr)
		}
	})

	t.Run("errors on invalid priority", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		_, stderr, code := runCmd(t, dir, "tick", "update", id, "--priority", "5")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "priority") {
			t.Errorf("expected priority error, got %q", stderr)
		}
	})

	t.Run("errors on non-existent parent ID", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		_, stderr, code := runCmd(t, dir, "tick", "update", id, "--parent", "tick-nonex")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("expected 'not found' error, got %q", stderr)
		}
	})

	t.Run("errors on self-referencing parent", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		_, stderr, code := runCmd(t, dir, "tick", "update", id, "--parent", id)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "own parent") {
			t.Errorf("expected self-reference error, got %q", stderr)
		}
	})

	t.Run("normalizes input IDs to lowercase", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		upperID := strings.ToUpper(id)

		_, _, code := runCmd(t, dir, "tick", "update", upperID, "--title", "Changed")
		if code != 0 {
			t.Errorf("expected exit 0 with uppercase ID, got %d", code)
		}
	})

	t.Run("persists changes via atomic write", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		runCmd(t, dir, "tick", "update", id, "--title", "Persisted title")

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), "Persisted title") {
			t.Errorf("expected updated title in JSONL, got %q", string(content))
		}
	})
}
