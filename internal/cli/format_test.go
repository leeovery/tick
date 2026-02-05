package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	t.Run("it has three format constants: Toon, Pretty, JSON", func(t *testing.T) {
		// Verify format constants exist and have expected values
		if FormatToon != "toon" {
			t.Errorf("FormatToon = %q, want %q", FormatToon, "toon")
		}
		if FormatPretty != "pretty" {
			t.Errorf("FormatPretty = %q, want %q", FormatPretty, "pretty")
		}
		if FormatJSON != "json" {
			t.Errorf("FormatJSON = %q, want %q", FormatJSON, "json")
		}
	})
}

func TestDetectTTY(t *testing.T) {
	t.Run("it detects non-TTY for bytes.Buffer", func(t *testing.T) {
		var buf bytes.Buffer
		if DetectTTY(&buf) {
			t.Error("bytes.Buffer should not be detected as TTY")
		}
	})

	t.Run("it detects non-TTY for nil writer", func(t *testing.T) {
		if DetectTTY(nil) {
			t.Error("nil writer should not be detected as TTY")
		}
	})

	t.Run("it defaults to non-TTY on stat failure", func(t *testing.T) {
		// Create a closed file to cause stat failure
		f, err := os.CreateTemp("", "test")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		name := f.Name()
		f.Close()
		os.Remove(name)

		// Opening a deleted file handle should fail stat
		// But we can't easily test this without access to internal behavior
		// The safest test is to verify the function doesn't panic on closed file
		closedFile, err := os.Open(os.DevNull)
		if err != nil {
			t.Fatalf("failed to open /dev/null: %v", err)
		}
		closedFile.Close()

		// Calling DetectTTY on a closed file should return false (non-TTY default)
		if DetectTTY(closedFile) {
			t.Error("closed file should default to non-TTY")
		}
	})
}

func TestResolveFormat(t *testing.T) {
	t.Run("it defaults to Toon when non-TTY and no flags", func(t *testing.T) {
		format, err := ResolveFormat(false, false, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatToon {
			t.Errorf("format = %q, want %q", format, FormatToon)
		}
	})

	t.Run("it defaults to Pretty when TTY and no flags", func(t *testing.T) {
		format, err := ResolveFormat(false, false, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("format = %q, want %q", format, FormatPretty)
		}
	})

	t.Run("it returns Toon when --toon flag is set", func(t *testing.T) {
		// Even with TTY, --toon overrides
		format, err := ResolveFormat(true, false, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatToon {
			t.Errorf("format = %q, want %q", format, FormatToon)
		}
	})

	t.Run("it returns Pretty when --pretty flag is set", func(t *testing.T) {
		// Even without TTY, --pretty overrides
		format, err := ResolveFormat(false, true, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("format = %q, want %q", format, FormatPretty)
		}
	})

	t.Run("it returns JSON when --json flag is set", func(t *testing.T) {
		format, err := ResolveFormat(false, false, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatJSON {
			t.Errorf("format = %q, want %q", format, FormatJSON)
		}
	})

	t.Run("it errors when --toon and --pretty both set", func(t *testing.T) {
		_, err := ResolveFormat(true, true, false, false)
		if err == nil {
			t.Fatal("expected error when multiple format flags set")
		}
		expected := "cannot specify multiple format flags (--toon, --pretty, --json)"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it errors when --toon and --json both set", func(t *testing.T) {
		_, err := ResolveFormat(true, false, true, false)
		if err == nil {
			t.Fatal("expected error when multiple format flags set")
		}
	})

	t.Run("it errors when --pretty and --json both set", func(t *testing.T) {
		_, err := ResolveFormat(false, true, true, false)
		if err == nil {
			t.Fatal("expected error when multiple format flags set")
		}
	})

	t.Run("it errors when all three format flags set", func(t *testing.T) {
		_, err := ResolveFormat(true, true, true, false)
		if err == nil {
			t.Fatal("expected error when multiple format flags set")
		}
	})
}

func TestFormatConfig(t *testing.T) {
	t.Run("it propagates format, quiet, and verbose", func(t *testing.T) {
		cfg := FormatConfig{
			Format:  FormatJSON,
			Quiet:   true,
			Verbose: true,
		}

		if cfg.Format != FormatJSON {
			t.Errorf("Format = %q, want %q", cfg.Format, FormatJSON)
		}
		if !cfg.Quiet {
			t.Error("expected Quiet to be true")
		}
		if !cfg.Verbose {
			t.Error("expected Verbose to be true")
		}
	})

	t.Run("it has zero values for fields when not set", func(t *testing.T) {
		cfg := FormatConfig{}

		if cfg.Format != "" {
			t.Errorf("Format = %q, want empty string", cfg.Format)
		}
		if cfg.Quiet {
			t.Error("expected Quiet to be false by default")
		}
		if cfg.Verbose {
			t.Error("expected Verbose to be false by default")
		}
	})
}

func TestNewFormatConfig(t *testing.T) {
	t.Run("it creates config from flags with TTY detection", func(t *testing.T) {
		var buf bytes.Buffer

		// Non-TTY, no flags -> Toon format
		cfg, err := NewFormatConfig(false, false, false, false, false, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Format != FormatToon {
			t.Errorf("Format = %q, want %q", cfg.Format, FormatToon)
		}
		if cfg.Quiet {
			t.Error("expected Quiet to be false")
		}
		if cfg.Verbose {
			t.Error("expected Verbose to be false")
		}
	})

	t.Run("it propagates quiet and verbose flags", func(t *testing.T) {
		var buf bytes.Buffer

		cfg, err := NewFormatConfig(false, false, false, true, true, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.Quiet {
			t.Error("expected Quiet to be true")
		}
		if !cfg.Verbose {
			t.Error("expected Verbose to be true")
		}
	})

	t.Run("it returns error when multiple format flags set", func(t *testing.T) {
		var buf bytes.Buffer

		_, err := NewFormatConfig(true, true, false, false, false, &buf)
		if err == nil {
			t.Fatal("expected error when multiple format flags set")
		}
	})
}

func TestFormatter(t *testing.T) {
	t.Run("it defines interface with all required methods", func(t *testing.T) {
		// Verify that concrete formatters implement the Formatter interface
		var _ Formatter = &ToonFormatter{}
		var _ Formatter = &PrettyFormatter{}
		var _ Formatter = &JSONFormatter{}
	})
}

func TestCLIFormatConflictHandling(t *testing.T) {
	t.Run("it errors before dispatch when multiple format flags set", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Try to run list with conflicting flags
		code := app.Run([]string{"tick", "--toon", "--pretty", "list"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.Contains(errOutput, "cannot specify multiple format flags") {
			t.Errorf("expected error about multiple format flags, got %q", errOutput)
		}

		// stdout should be empty (error goes to stderr)
		if stdout.Len() != 0 {
			t.Errorf("expected stdout to be empty, got %q", stdout.String())
		}
	})

	t.Run("it errors with --toon and --json", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--toon", "--json", "list"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	t.Run("it errors with --pretty and --json", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "--pretty", "--json", "list"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	t.Run("it allows single format flag to proceed", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Single flag should succeed (even though no tasks exist)
		code := app.Run([]string{"tick", "--toon", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})
}

func TestFormatConfigWiredIntoApp(t *testing.T) {
	t.Run("it stores format config in App after flag parsing", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Run with format flag - config should be set
		app.Run([]string{"tick", "--json", "list"})

		// Verify formatConfig was set
		if app.formatConfig.Format != FormatJSON {
			t.Errorf("formatConfig.Format = %q, want %q", app.formatConfig.Format, FormatJSON)
		}
	})

	t.Run("it sets quiet in format config", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		app.Run([]string{"tick", "--quiet", "list"})

		if !app.formatConfig.Quiet {
			t.Error("expected formatConfig.Quiet to be true")
		}
	})

	t.Run("it sets verbose in format config", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		app.Run([]string{"tick", "--verbose", "list"})

		if !app.formatConfig.Verbose {
			t.Error("expected formatConfig.Verbose to be true")
		}
	})
}

func TestVerboseOutput(t *testing.T) {
	t.Run("it writes verbose output to stderr, not stdout", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Run with --verbose flag
		app.Run([]string{"tick", "--verbose", "list"})

		// Verbose output should go to stderr (if any)
		// This test ensures that WriteVerbose writes to stderr
		// For now, main output (list) still goes to stdout
		// Verbose debug info should NOT contaminate stdout

		stdoutContent := stdout.String()
		// If there's any verbose-style debug info, it should NOT be in stdout
		// (It should be in stderr instead)
		if strings.Contains(stdoutContent, "verbose:") || strings.Contains(stdoutContent, "[debug]") {
			t.Errorf("verbose/debug output should not appear in stdout, got %q", stdoutContent)
		}
	})

	t.Run("verbose is orthogonal to format selection", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Run with both --verbose and --json
		app.Run([]string{"tick", "--verbose", "--json", "list"})

		// Format should still be JSON
		if app.formatConfig.Format != FormatJSON {
			t.Errorf("formatConfig.Format = %q, want %q", app.formatConfig.Format, FormatJSON)
		}
		if !app.formatConfig.Verbose {
			t.Error("expected formatConfig.Verbose to be true")
		}
	})

	t.Run("quiet is orthogonal to format selection", func(t *testing.T) {
		dir := setupTickDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		// Run with both --quiet and --toon
		app.Run([]string{"tick", "--quiet", "--toon", "list"})

		// Format should still be TOON
		if app.formatConfig.Format != FormatToon {
			t.Errorf("formatConfig.Format = %q, want %q", app.formatConfig.Format, FormatToon)
		}
		if !app.formatConfig.Quiet {
			t.Error("expected formatConfig.Quiet to be true")
		}
	})
}

func TestWriteVerbose(t *testing.T) {
	t.Run("it writes to stderr when verbose is enabled", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout:       &stdout,
			Stderr:       &stderr,
			formatConfig: FormatConfig{Verbose: true},
		}

		app.WriteVerbose("debug message")

		if stderr.String() != "verbose: debug message\n" {
			t.Errorf("stderr = %q, want %q", stderr.String(), "verbose: debug message\n")
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout should be empty, got %q", stdout.String())
		}
	})

	t.Run("it does nothing when verbose is disabled", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout:       &stdout,
			Stderr:       &stderr,
			formatConfig: FormatConfig{Verbose: false},
		}

		app.WriteVerbose("debug message")

		if stderr.Len() != 0 {
			t.Errorf("stderr should be empty when verbose disabled, got %q", stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout should be empty, got %q", stdout.String())
		}
	})

	t.Run("it formats messages with arguments", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout:       &stdout,
			Stderr:       &stderr,
			formatConfig: FormatConfig{Verbose: true},
		}

		app.WriteVerbose("loaded %d tasks from %s", 5, "tasks.jsonl")

		expected := "verbose: loaded 5 tasks from tasks.jsonl\n"
		if stderr.String() != expected {
			t.Errorf("stderr = %q, want %q", stderr.String(), expected)
		}
	})
}
