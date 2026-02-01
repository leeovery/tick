package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDepCommands(t *testing.T) {
	t.Run("adds a dependency between two existing tasks", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent")
		id2 := extractID(t, out2)

		_, _, code := runCmd(t, dir, "tick", "dep", "add", id2, id1)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		if !strings.Contains(string(content), `"blocked_by"`) {
			t.Errorf("expected blocked_by in JSONL, got %q", string(content))
		}
	})

	t.Run("removes an existing dependency", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent", "--blocked-by", id1)
		id2 := extractID(t, out2)

		_, _, code := runCmd(t, dir, "tick", "dep", "rm", id2, id1)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		// The dependent task line should no longer have blocked_by
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		for _, line := range lines {
			if strings.Contains(line, id2) {
				if strings.Contains(line, `"blocked_by"`) {
					t.Errorf("blocked_by should be removed, got %q", line)
				}
			}
		}
	})

	t.Run("outputs confirmation on add", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent")
		id2 := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "dep", "add", id2, id1)
		if !strings.Contains(stdout, "added") || !strings.Contains(stdout, id2) {
			t.Errorf("expected add confirmation, got %q", stdout)
		}
	})

	t.Run("outputs confirmation on rm", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent", "--blocked-by", id1)
		id2 := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "dep", "rm", id2, id1)
		if !strings.Contains(stdout, "removed") || !strings.Contains(stdout, id2) {
			t.Errorf("expected rm confirmation, got %q", stdout)
		}
	})

	t.Run("updates task updated timestamp", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent")
		id2 := extractID(t, out2)

		content1, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
		runCmd(t, dir, "tick", "dep", "add", id2, id1)
		content2, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))

		if string(content1) == string(content2) {
			t.Error("JSONL should change after dep add")
		}
	})

	t.Run("errors when task_id not found on add", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "add", "tick-nonex", id1)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("expected not found error, got %q", stderr)
		}
	})

	t.Run("errors when blocked_by_id not found on add", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Task")
		id1 := extractID(t, out1)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "add", id1, "tick-nonex")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("expected not found error, got %q", stderr)
		}
	})

	t.Run("errors on duplicate dependency", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent", "--blocked-by", id1)
		id2 := extractID(t, out2)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "add", id2, id1)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "already") {
			t.Errorf("expected duplicate error, got %q", stderr)
		}
	})

	t.Run("errors when dependency not found on rm", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent")
		id2 := extractID(t, out2)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "rm", id2, id1)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not blocked") {
			t.Errorf("expected not blocked error, got %q", stderr)
		}
	})

	t.Run("errors on self-reference", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Task")
		id1 := extractID(t, out1)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "add", id1, id1)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "itself") {
			t.Errorf("expected self-reference error, got %q", stderr)
		}
	})

	t.Run("errors when add creates cycle", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Task A")
		idA := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Task B", "--blocked-by", idA)
		idB := extractID(t, out2)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "add", idA, idB)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "cycle") {
			t.Errorf("expected cycle error, got %q", stderr)
		}
	})

	t.Run("errors when add creates child-blocked-by-parent", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent")
		idP := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Child", "--parent", idP)
		idC := extractID(t, out2)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "add", idC, idP)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "parent") {
			t.Errorf("expected parent error, got %q", stderr)
		}
	})

	t.Run("normalizes IDs to lowercase", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent")
		id2 := extractID(t, out2)

		_, _, code := runCmd(t, dir, "tick", "dep", "add", strings.ToUpper(id2), strings.ToUpper(id1))
		if code != 0 {
			t.Errorf("expected exit 0 with uppercase IDs, got %d", code)
		}
	})

	t.Run("suppresses output with --quiet", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Dependent")
		id2 := extractID(t, out2)

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "dep", "add", id2, id1)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Errorf("--quiet should suppress output, got %q", stdout)
		}
	})

	t.Run("errors when fewer than two IDs provided", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Task")
		id1 := extractID(t, out1)

		_, stderr, code := runCmd(t, dir, "tick", "dep", "add", id1)
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "requires") {
			t.Errorf("expected usage error, got %q", stderr)
		}

		_, stderr2, code2 := runCmd(t, dir, "tick", "dep", "add")
		if code2 != 1 {
			t.Errorf("expected exit 1, got %d", code2)
		}
		if !strings.Contains(stderr2, "requires") {
			t.Errorf("expected usage error, got %q", stderr2)
		}
	})

	t.Run("errors when no subcommand provided", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "dep")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "add") && !strings.Contains(stderr, "rm") {
			t.Errorf("expected usage hint, got %q", stderr)
		}
	})
}
