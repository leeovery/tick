package cli

import (
	"bytes"
	"strings"
	"testing"
)

func runCmd(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	app := NewApp(&outBuf, &errBuf)
	c := app.Run(args, dir)
	return outBuf.String(), errBuf.String(), c
}

func TestListCommand(t *testing.T) {
	t.Run("lists all tasks", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "First task")
		createTask(t, dir, "Second task")

		stdout, _, code := runCmd(t, dir, "tick", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "First task") {
			t.Error("expected First task in output")
		}
		if !strings.Contains(stdout, "Second task") {
			t.Error("expected Second task in output")
		}
	})

	t.Run("lists tasks ordered by priority then created", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Low priority", "--priority", "3")
		createTask(t, dir, "High priority", "--priority", "0")
		createTask(t, dir, "Medium priority", "--priority", "2")

		stdout, _, _ := runCmd(t, dir, "tick", "list")
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		// TOON: header line + 3 data lines
		if len(lines) < 4 {
			t.Fatalf("expected at least 4 lines (header + 3 tasks), got %d", len(lines))
		}
		if !strings.Contains(lines[1], "High priority") {
			t.Errorf("first task should be High priority, got %q", lines[1])
		}
		if !strings.Contains(lines[3], "Low priority") {
			t.Errorf("last task should be Low priority, got %q", lines[3])
		}
	})

	t.Run("prints empty result when no tasks", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, code := runCmd(t, dir, "tick", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		// TOON format: zero-count section
		if !strings.Contains(stdout, "tasks[0]") {
			t.Errorf("expected empty list indicator, got %q", stdout)
		}
	})

	t.Run("prints only task IDs with --quiet flag", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task one")
		createTask(t, dir, "Task two")

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "tick-") {
				t.Errorf("--quiet should output only IDs, got %q", line)
			}
		}
	})
}

func TestShowCommand(t *testing.T) {
	t.Run("shows full task details by ID", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test task", "--priority", "1")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "show", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, id) {
			t.Error("output should contain task ID")
		}
		if !strings.Contains(stdout, "Test task") {
			t.Error("output should contain title")
		}
		if !strings.Contains(stdout, "open") {
			t.Error("output should contain status")
		}
	})

	t.Run("shows blocked_by section with context", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker task")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Blocked task", "--blocked-by", id1)
		id2 := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id2)
		// TOON: blocked_by[1]{id,title,status}:
		if !strings.Contains(stdout, "blocked_by[1]") {
			t.Error("expected blocked_by section with count 1")
		}
		if !strings.Contains(stdout, id1) {
			t.Error("expected blocker ID in output")
		}
		if !strings.Contains(stdout, "Blocker task") {
			t.Error("expected blocker title in output")
		}
	})

	t.Run("shows children section", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent task")
		id1 := extractID(t, out1)
		createTask(t, dir, "Child task", "--parent", id1)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id1)
		// TOON: children[1]{id,title,status}:
		if !strings.Contains(stdout, "children[1]") {
			t.Error("expected children section with count 1")
		}
		if !strings.Contains(stdout, "Child task") {
			t.Error("expected child title in output")
		}
	})

	t.Run("shows description when present", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Desc task", "--description", "Detailed info here")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		if !strings.Contains(stdout, "description") {
			t.Error("expected description section")
		}
		if !strings.Contains(stdout, "Detailed info here") {
			t.Error("expected description content")
		}
	})

	t.Run("shows blocked_by with count 0 when no dependencies", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No deps")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		// TOON always shows blocked_by, even empty
		if !strings.Contains(stdout, "blocked_by[0]") {
			t.Error("expected blocked_by[0] section")
		}
	})

	t.Run("shows children with count 0 when no children", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No children")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		if !strings.Contains(stdout, "children[0]") {
			t.Error("expected children[0] section")
		}
	})

	t.Run("omits description section when empty", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No desc")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		if strings.Contains(stdout, "description:") {
			t.Error("should not show description section when empty")
		}
	})

	t.Run("shows parent ID when set", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent task")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Child task", "--parent", id1)
		id2 := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id2)
		if !strings.Contains(stdout, "parent") {
			t.Error("expected parent in schema")
		}
		if !strings.Contains(stdout, id1) {
			t.Error("expected parent ID in output")
		}
	})

	t.Run("omits parent from schema when not set", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No parent")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		// In TOON, parent should not appear in the schema header
		if strings.Contains(stdout, ",parent,") || strings.Contains(stdout, ",parent}") {
			t.Error("should not show parent in schema when not set")
		}
	})

	t.Run("errors when task ID not found", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "show", "tick-nonexist")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("expected 'not found' error, got %q", stderr)
		}
	})

	t.Run("errors when no ID argument provided", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "show")
		if code != 1 {
			t.Errorf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "Task ID is required") {
			t.Errorf("expected usage error, got %q", stderr)
		}
	})

	t.Run("normalizes input ID to lowercase", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test")
		id := extractID(t, out)
		upperId := strings.ToUpper(id)

		_, _, code := runCmd(t, dir, "tick", "show", upperId)
		if code != 0 {
			t.Errorf("expected exit 0 with uppercase ID, got %d", code)
		}
	})

	t.Run("outputs only task ID with --quiet flag", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Test")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "show", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		output := strings.TrimSpace(stdout)
		if output != id {
			t.Errorf("--quiet should output only ID %q, got %q", id, output)
		}
	})
}
