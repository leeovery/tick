package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateFlags(t *testing.T) {
	t.Run("it returns nil for args with no flags", func(t *testing.T) {
		err := ValidateFlags("create", []string{"My Task Title"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("it returns nil for args with known command flags", func(t *testing.T) {
		err := ValidateFlags("create", []string{"My Task", "--priority", "3", "--description", "desc"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("it returns error for unknown flag on dep add (bug repro)", func(t *testing.T) {
		err := ValidateFlags("dep add", []string{"tick-aaa", "--blocks", "tick-bbb"}, commandFlags)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		want := `unknown flag "--blocks" for "dep add". Run 'tick help dep' for usage.`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it accepts -f on remove", func(t *testing.T) {
		err := ValidateFlags("remove", []string{"-f", "tick-abc123"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil for -f on remove, got %v", err)
		}
	})
}

func TestHelpCommand(t *testing.T) {
	t.Run("it returns command itself for single-word command", func(t *testing.T) {
		got := helpCommand("list")
		if got != "list" {
			t.Errorf("helpCommand(%q) = %q, want %q", "list", got, "list")
		}
	})

	t.Run("it returns parent for two-level command", func(t *testing.T) {
		got := helpCommand("dep add")
		if got != "dep" {
			t.Errorf("helpCommand(%q) = %q, want %q", "dep add", got, "dep")
		}
	})
}

func TestFlagValidationWiring(t *testing.T) {
	t.Run("it rejects unknown flag before subcommand", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--bogus", "list"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--bogus". Run 'tick help' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})
}
