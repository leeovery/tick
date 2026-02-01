package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransitionCommands(t *testing.T) {
	t.Run("transitions task to in_progress via tick start", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "start", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "open") || !strings.Contains(stdout, "in_progress") {
			t.Errorf("expected transition output, got %q", stdout)
		}
	})

	t.Run("transitions task to done via tick done from open", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "done", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "done") {
			t.Errorf("expected 'done' in output, got %q", stdout)
		}
	})

	t.Run("transitions task to done via tick done from in_progress", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "start", id)

		stdout, _, code := runCmd(t, dir, "tick", "done", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "done") {
			t.Errorf("expected 'done' in output, got %q", stdout)
		}
	})

	t.Run("transitions task to cancelled via tick cancel from open", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "cancel", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "cancelled") {
			t.Errorf("expected 'cancelled' in output, got %q", stdout)
		}
	})

	t.Run("transitions task to cancelled via tick cancel from in_progress", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "start", id)

		stdout, _, code := runCmd(t, dir, "tick", "cancel", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "cancelled") {
			t.Errorf("expected 'cancelled' in output, got %q", stdout)
		}
	})

	t.Run("transitions task to open via tick reopen from done", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "done", id)

		stdout, _, code := runCmd(t, dir, "tick", "reopen", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "open") {
			t.Errorf("expected 'open' in output, got %q", stdout)
		}
	})

	t.Run("transitions task to open via tick reopen from cancelled", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "cancel", id)

		stdout, _, code := runCmd(t, dir, "tick", "reopen", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "open") {
			t.Errorf("expected 'open' in output, got %q", stdout)
		}
	})

	t.Run("outputs status transition line on success", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "start", id)
		// Expected format: tick-XXXXXX: open → in_progress
		if !strings.Contains(stdout, id) {
			t.Errorf("output should contain task ID, got %q", stdout)
		}
		if !strings.Contains(stdout, "→") {
			t.Errorf("output should contain arrow, got %q", stdout)
		}
	})

	t.Run("suppresses output with --quiet flag", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "start", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Errorf("--quiet should suppress output, got %q", stdout)
		}
	})

	t.Run("errors when task ID argument is missing", func(t *testing.T) {
		dir := initTickDir(t)
		for _, cmd := range []string{"start", "done", "cancel", "reopen"} {
			_, stderr, code := runCmd(t, dir, "tick", cmd)
			if code != 1 {
				t.Errorf("%s: expected exit 1, got %d", cmd, code)
			}
			if !strings.Contains(stderr, "Task ID is required") {
				t.Errorf("%s: expected usage error, got %q", cmd, stderr)
			}
		}
	})

	t.Run("errors when task ID is not found", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "start", "tick-nonex")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("expected 'not found' error, got %q", stderr)
		}
	})

	t.Run("errors on invalid transition", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "done", id)

		_, stderr, code := runCmd(t, dir, "tick", "start", id)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "Cannot") {
			t.Errorf("expected transition error, got %q", stderr)
		}
	})

	t.Run("normalizes task ID to lowercase", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		upperID := strings.ToUpper(id)

		_, _, code := runCmd(t, dir, "tick", "start", upperID)
		if code != 0 {
			t.Errorf("expected exit 0 with uppercase ID, got %d", code)
		}
	})

	t.Run("persists status change via atomic write", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "start", id)

		content, err := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if err != nil {
			t.Fatalf("reading JSONL: %v", err)
		}
		if !strings.Contains(string(content), `"status":"in_progress"`) {
			t.Errorf("expected in_progress in JSONL, got %q", string(content))
		}
	})

	t.Run("sets closed timestamp on done", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "done", id)

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"closed"`) {
			t.Errorf("expected closed timestamp in JSONL, got %q", string(content))
		}
	})

	t.Run("sets closed timestamp on cancel", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "cancel", id)

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"closed"`) {
			t.Errorf("expected closed timestamp in JSONL, got %q", string(content))
		}
	})

	t.Run("clears closed timestamp on reopen", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task")
		id := extractID(t, out)
		runCmd(t, dir, "tick", "done", id)
		runCmd(t, dir, "tick", "reopen", id)

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if strings.Contains(string(content), `"closed"`) {
			t.Errorf("closed should be cleared after reopen, got %q", string(content))
		}
	})
}
