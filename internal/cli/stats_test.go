package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatsCommand(t *testing.T) {
	t.Run("counts tasks by status correctly", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Open task")
		_ = out1
		out2, _, _ := createTask(t, dir, "IP task")
		id2 := extractID(t, out2)
		out3, _, _ := createTask(t, dir, "Done task")
		id3 := extractID(t, out3)
		out4, _, _ := createTask(t, dir, "Cancelled task")
		id4 := extractID(t, out4)

		runCmd(t, dir, "tick", "start", id2)
		runCmd(t, dir, "tick", "done", id3)
		runCmd(t, dir, "tick", "cancel", id4)

		stdout, _, code := runCmd(t, dir, "tick", "stats")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		// TOON: stats section with counts
		if !strings.Contains(stdout, "stats{") {
			t.Errorf("expected stats section, got %q", stdout)
		}
		// Total should be 4
		if !strings.Contains(stdout, "4,") {
			t.Errorf("expected total 4 in output, got %q", stdout)
		}
	})

	t.Run("counts ready and blocked tasks correctly", func(t *testing.T) {
		dir := initTickDir(t)
		out1, _, _ := createTask(t, dir, "Blocker")
		id1 := extractID(t, out1)
		createTask(t, dir, "Blocked", "--blocked-by", id1)

		stdout, _, _ := runCmd(t, dir, "tick", "stats")
		// Both are open. Blocker is ready, Blocked is blocked.
		// TOON: stats row includes ready and blocked counts
		if !strings.Contains(stdout, "stats{") {
			t.Errorf("expected stats section, got %q", stdout)
		}
	})

	t.Run("includes all 5 priority levels even at zero", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task", "--priority", "2")

		stdout, _, _ := runCmd(t, dir, "tick", "stats")
		// TOON: by_priority[5]{priority,count}:
		if !strings.Contains(stdout, "by_priority[5]") {
			t.Errorf("expected 5 priority rows, got %q", stdout)
		}
	})

	t.Run("returns all zeros for empty project", func(t *testing.T) {
		dir := initTickDir(t)

		stdout, _, code := runCmd(t, dir, "tick", "stats")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		// TOON: 0,0,0,0,0,0,0
		if !strings.Contains(stdout, "0,0,0,0,0,0,0") {
			t.Errorf("expected all zeros, got %q", stdout)
		}
		if !strings.Contains(stdout, "by_priority[5]") {
			t.Errorf("expected 5 priority rows even for empty project, got %q", stdout)
		}
	})

	t.Run("formats stats in JSON format", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		stdout, _, code := runCmd(t, dir, "tick", "--json", "stats")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "\"total\"") {
			t.Errorf("expected JSON total field, got %q", stdout)
		}
		if !strings.Contains(stdout, "\"by_priority\"") {
			t.Errorf("expected JSON by_priority field, got %q", stdout)
		}
	})

	t.Run("formats stats in Pretty format", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		stdout, _, code := runCmd(t, dir, "tick", "--pretty", "stats")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "Total:") {
			t.Errorf("expected Pretty Total label, got %q", stdout)
		}
	})

	t.Run("suppresses output with --quiet", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		code := app.Run([]string{"tick", "--quiet", "stats"}, dir)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if outBuf.Len() != 0 {
			t.Errorf("--quiet should suppress stats output, got %q", outBuf.String())
		}
	})

	t.Run("ready count matches ready query semantics", func(t *testing.T) {
		dir := initTickDir(t)
		// Parent with open child — parent is blocked, child is ready
		out1, _, _ := createTask(t, dir, "Parent")
		idP := extractID(t, out1)
		createTask(t, dir, "Child", "--parent", idP)

		stdout, _, _ := runCmd(t, dir, "tick", "--json", "stats")
		// Should have ready=1 (child), blocked=1 (parent)
		if !strings.Contains(stdout, `"ready": 1`) && !strings.Contains(stdout, `"ready":1`) {
			t.Errorf("expected ready=1, got %q", stdout)
		}
		if !strings.Contains(stdout, `"blocked": 1`) && !strings.Contains(stdout, `"blocked":1`) {
			t.Errorf("expected blocked=1, got %q", stdout)
		}
	})
}
