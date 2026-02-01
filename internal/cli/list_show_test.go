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
	t.Run("lists all tasks with aligned columns", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "First task")
		createTask(t, dir, "Second task")

		stdout, _, code := runCmd(t, dir, "tick", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "ID") {
			t.Error("expected header row with ID")
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
		// Skip header
		if len(lines) < 4 {
			t.Fatalf("expected at least 4 lines (header + 3 tasks), got %d", len(lines))
		}
		// First data line should be priority 0 (High)
		if !strings.Contains(lines[1], "High priority") {
			t.Errorf("first task should be High priority, got %q", lines[1])
		}
		// Last data line should be priority 3 (Low)
		if !strings.Contains(lines[3], "Low priority") {
			t.Errorf("last task should be Low priority, got %q", lines[3])
		}
	})

	t.Run("prints 'No tasks found.' when empty", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, code := runCmd(t, dir, "tick", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "No tasks found.") {
			t.Errorf("expected 'No tasks found.', got %q", stdout)
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
		if !strings.Contains(stdout, "Blocked by:") {
			t.Error("expected 'Blocked by:' section")
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
		if !strings.Contains(stdout, "Children:") {
			t.Error("expected 'Children:' section")
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
		if !strings.Contains(stdout, "Description:") {
			t.Error("expected 'Description:' section")
		}
		if !strings.Contains(stdout, "Detailed info here") {
			t.Error("expected description content")
		}
	})

	t.Run("omits blocked_by section when no dependencies", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No deps")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		if strings.Contains(stdout, "Blocked by:") {
			t.Error("should not show 'Blocked by:' when no deps")
		}
	})

	t.Run("omits children section when no children", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No children")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		if strings.Contains(stdout, "Children:") {
			t.Error("should not show 'Children:' when no children")
		}
	})

	t.Run("omits description section when empty", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No desc")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		if strings.Contains(stdout, "Description:") {
			t.Error("should not show 'Description:' when empty")
		}
	})

	t.Run("shows parent with ID and title when set", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Parent task")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Child task", "--parent", id1)
		id2 := extractID(t, out2)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id2)
		if !strings.Contains(stdout, "Parent:") {
			t.Error("expected 'Parent:' field")
		}
		if !strings.Contains(stdout, id1) {
			t.Error("expected parent ID in output")
		}
		if !strings.Contains(stdout, "Parent task") {
			t.Error("expected parent title in output")
		}
	})

	t.Run("omits parent field when not set", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "No parent")
		id := extractID(t, out)

		stdout, _, _ := runCmd(t, dir, "tick", "show", id)
		if strings.Contains(stdout, "Parent:") {
			t.Error("should not show 'Parent:' when not set")
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
