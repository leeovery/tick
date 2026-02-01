package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatIntegration(t *testing.T) {
	t.Run("it defaults to TOON when piped (non-TTY)", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		stdout, _, code := runCmd(t, dir, "tick", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		// In tests, stdout is a buffer (non-TTY) → should default to TOON format.
		if !strings.HasPrefix(stdout, "tasks[") {
			t.Errorf("expected TOON format (tasks[...]), got %q", stdout)
		}
	})

	t.Run("it respects --pretty override", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		stdout, _, code := runCmd(t, dir, "tick", "--pretty", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		// Pretty format has column headers.
		if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "STATUS") {
			t.Errorf("expected Pretty format with column headers, got %q", stdout)
		}
	})

	t.Run("it respects --json override", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		stdout, _, code := runCmd(t, dir, "tick", "--json", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !json.Valid([]byte(stdout)) {
			t.Errorf("expected valid JSON, got %q", stdout)
		}
	})

	t.Run("it respects --toon override", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		stdout, _, code := runCmd(t, dir, "tick", "--toon", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.HasPrefix(stdout, "tasks[") {
			t.Errorf("expected TOON format, got %q", stdout)
		}
	})

	t.Run("it formats create as full task detail in TOON", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, code := createTask(t, dir, "New task")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		// TOON show format has task{...}: section.
		if !strings.Contains(stdout, "task{") {
			t.Errorf("expected TOON task detail, got %q", stdout)
		}
	})

	t.Run("it formats create as JSON with --json", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, code := runCmd(t, dir, "tick", "--json", "create", "JSON task")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !json.Valid([]byte(stdout)) {
			t.Errorf("expected valid JSON from create, got %q", stdout)
		}
	})

	t.Run("it formats transition in each format", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Trans task")
		id := extractID(t, out)

		// TOON (default in tests)
		stdout, _, code := runCmd(t, dir, "tick", "start", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "→") {
			t.Errorf("expected transition arrow in TOON output, got %q", stdout)
		}

		// JSON
		stdout, _, code = runCmd(t, dir, "tick", "--json", "done", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !json.Valid([]byte(stdout)) {
			t.Errorf("expected valid JSON from transition, got %q", stdout)
		}
	})

	t.Run("it formats show in TOON", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Show task")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "show", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "task{") {
			t.Errorf("expected TOON task detail, got %q", stdout)
		}
	})

	t.Run("it applies --quiet override for create (ID only)", func(t *testing.T) {
		dir := initTickDir(t)
		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "create", "Quiet task")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		output := strings.TrimSpace(stdout)
		if !strings.HasPrefix(output, "tick-") || len(output) != 11 {
			t.Errorf("--quiet create should output only ID, got %q", output)
		}
	})

	t.Run("it applies --quiet override for transition (nothing)", func(t *testing.T) {
		dir := initTickDir(t)
		out, _, _ := createTask(t, dir, "Task for quiet trans")
		id := extractID(t, out)

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "start", id)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if stdout != "" {
			t.Errorf("--quiet transition should produce no output, got %q", stdout)
		}
	})

	t.Run("it applies --quiet override for list (IDs only)", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task for quiet list")

		stdout, _, code := runCmd(t, dir, "tick", "--quiet", "list")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "tick-") {
				t.Errorf("--quiet list should output only IDs, got %q", line)
			}
		}
	})

	t.Run("it handles empty list per format", func(t *testing.T) {
		dir := initTickDir(t)

		// TOON: zero-count
		stdout, _, _ := runCmd(t, dir, "tick", "list")
		if !strings.Contains(stdout, "tasks[0]") {
			t.Errorf("expected TOON zero-count, got %q", stdout)
		}

		// Pretty: message
		stdout, _, _ = runCmd(t, dir, "tick", "--pretty", "list")
		if !strings.Contains(stdout, "No tasks found.") {
			t.Errorf("expected Pretty empty message, got %q", stdout)
		}

		// JSON: []
		stdout, _, _ = runCmd(t, dir, "tick", "--json", "list")
		if strings.TrimSpace(stdout) != "[]" {
			t.Errorf("expected JSON [], got %q", stdout)
		}
	})

	t.Run("it formats dep confirmations in TOON", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Task A")
		id1 := extractID(t, out1)
		out2, _, _ := createTask(t, dir, "Task B")
		id2 := extractID(t, out2)

		stdout, _, code := runCmd(t, dir, "tick", "dep", "add", id2, id1)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "Dependency added") {
			t.Errorf("expected dep confirmation, got %q", stdout)
		}
	})

	t.Run("it formats init message in TOON", func(t *testing.T) {
		dir := t.TempDir()
		stdout, _, code := runCmd(t, dir, "tick", "init")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "Initialized tick") {
			t.Errorf("expected init message, got %q", stdout)
		}
	})

	t.Run("it errors on conflicting format flags", func(t *testing.T) {
		dir := initTickDir(t)
		_, stderr, code := runCmd(t, dir, "tick", "--toon", "--json", "list")
		if code != 1 {
			t.Errorf("expected exit 1 on conflicting flags, got %d", code)
		}
		if !strings.Contains(stderr, "only one") {
			t.Errorf("expected conflict error, got %q", stderr)
		}
	})
}
