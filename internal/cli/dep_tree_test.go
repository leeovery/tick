package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDepTreeWiring(t *testing.T) {
	t.Run("it qualifies dep tree as a two-level command", func(t *testing.T) {
		cmd, rest := qualifyCommand("dep", []string{"tree", "tick-abc123"})
		if cmd != "dep tree" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep tree")
		}
		if len(rest) != 1 || rest[0] != "tick-abc123" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-abc123]", rest)
		}
	})

	t.Run("it qualifies dep tree with no args", func(t *testing.T) {
		cmd, rest := qualifyCommand("dep", []string{"tree"})
		if cmd != "dep tree" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep tree")
		}
		if len(rest) != 0 {
			t.Errorf("qualifyCommand returned rest = %v, want empty", rest)
		}
	})

	t.Run("it rejects unknown flag on dep tree", func(t *testing.T) {
		err := ValidateFlags("dep tree", []string{"--unknown"}, commandFlags)
		if err == nil {
			t.Fatal("expected error for --unknown on dep tree, got nil")
		}
		want := `unknown flag "--unknown" for "dep tree". Run 'tick help dep' for usage.`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it accepts global flags on dep tree", func(t *testing.T) {
		err := ValidateFlags("dep tree", []string{"--quiet"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil for global flag on dep tree, got %v", err)
		}
	})

	t.Run("it dispatches dep tree without error", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "dep", "tree"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
	})

	t.Run("it shows tree in dep help text", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "dep")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "tree") {
			t.Errorf("dep help should mention 'tree', got %q", stdout)
		}
	})

	t.Run("it does not break existing dep add/remove dispatch", func(t *testing.T) {
		// Verify qualifyCommand still works for add and remove
		cmd, rest := qualifyCommand("dep", []string{"add", "tick-aaa", "tick-bbb"})
		if cmd != "dep add" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep add")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "tick-bbb" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa tick-bbb]", rest)
		}

		cmd, rest = qualifyCommand("dep", []string{"remove", "tick-aaa", "tick-bbb"})
		if cmd != "dep remove" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep remove")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "tick-bbb" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa tick-bbb]", rest)
		}
	})
}
