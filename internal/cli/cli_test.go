package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLI_UnknownSubcommand(t *testing.T) {
	t.Run("it routes unknown subcommands to error", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "foobar"})
		if code != 1 {
			t.Errorf("expected exit code 1 for unknown subcommand, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error: ") {
			t.Errorf("expected 'Error: ' prefix, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Unknown command 'foobar'") {
			t.Errorf("expected error to mention unknown command name, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Run 'tick help' for usage") {
			t.Errorf("expected error to suggest 'tick help', got %q", errMsg)
		}
		if stdout.String() != "" {
			t.Errorf("expected no stdout for unknown subcommand, got %q", stdout.String())
		}
	})
}

func TestCLI_NoSubcommand(t *testing.T) {
	t.Run("it prints basic usage with exit code 0 when no subcommand given", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick"})
		if code != 0 {
			t.Errorf("expected exit code 0 for no subcommand, got %d", code)
		}

		output := stdout.String()
		if output == "" {
			t.Error("expected usage output on stdout, got nothing")
		}
		if !strings.Contains(strings.ToLower(output), "usage") {
			t.Errorf("expected output to contain 'usage', got %q", output)
		}
	})
}

func TestCLI_GlobalFlagsParsed(t *testing.T) {
	t.Run("it parses --quiet flag before subcommand", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--quiet", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Quiet flag should suppress output
		if stdout.String() != "" {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it parses --verbose flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		// Just verify it doesn't error; verbose flag is plumbing only
		code := app.Run([]string{"tick", "--verbose", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0 with --verbose, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses -v short flag for verbose", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "-v", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0 with -v, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses --toon flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--toon", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0 with --toon, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses --pretty flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0 with --pretty, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses --json flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--json", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0 with --json, got %d; stderr: %s", code, stderr.String())
		}
	})
}

func TestCLI_TTYDetection(t *testing.T) {
	t.Run("it detects non-TTY and defaults to Toon via Run", func(t *testing.T) {
		var stdout bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &bytes.Buffer{},
			Dir:    t.TempDir(),
		}

		app.Run([]string{"tick", "init"})

		if app.IsTTY {
			t.Error("expected IsTTY to be false when stdout is a bytes.Buffer")
		}
		if app.OutputFormat != FormatToon {
			t.Errorf("expected default format FormatToon for non-TTY, got %v", app.OutputFormat)
		}
	})
}

func TestCLI_OutputFormatFlagOverride(t *testing.T) {
	t.Run("it overrides TTY-detected format with --toon flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		app.Run([]string{"tick", "--toon", "init"})

		if app.OutputFormat != FormatToon {
			t.Errorf("expected format FormatToon with --toon flag, got %v", app.OutputFormat)
		}
	})

	t.Run("it overrides TTY-detected format with --pretty flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		app.Run([]string{"tick", "--pretty", "init"})

		if app.OutputFormat != FormatPretty {
			t.Errorf("expected format FormatPretty with --pretty flag, got %v", app.OutputFormat)
		}
	})

	t.Run("it overrides TTY-detected format with --json flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		app.Run([]string{"tick", "--json", "init"})

		if app.OutputFormat != FormatJSON {
			t.Errorf("expected format FormatJSON with --json flag, got %v", app.OutputFormat)
		}
	})
}
