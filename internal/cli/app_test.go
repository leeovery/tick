package cli

import (
	"strings"
	"testing"
)

func TestAppRouting(t *testing.T) {
	t.Run("it routes unknown subcommands to error", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "foobar"})
		if err == nil {
			t.Fatal("Run() expected error for unknown subcommand, got nil")
		}

		want := "Unknown command 'foobar'. Run 'tick help' for usage."
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it prints basic usage with no subcommand", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		got := stdout.String()
		if !strings.Contains(got, "Usage:") {
			t.Errorf("stdout = %q, want it to contain 'Usage:'", got)
		}
		if !strings.Contains(got, "init") {
			t.Errorf("stdout = %q, want it to contain 'init'", got)
		}
	})

	t.Run("it parses --quiet global flag", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		// --quiet before subcommand init
		err := app.Run([]string{"tick", "--quiet", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		if !app.config.Quiet {
			t.Error("expected Quiet to be true after --quiet flag")
		}
	})

	t.Run("it parses -q global flag", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "-q", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		if !app.config.Quiet {
			t.Error("expected Quiet to be true after -q flag")
		}
	})

	t.Run("it parses --verbose global flag", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "--verbose", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		if !app.config.Verbose {
			t.Error("expected Verbose to be true after --verbose flag")
		}
	})

	t.Run("it parses -v global flag", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "-v", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		if !app.config.Verbose {
			t.Error("expected Verbose to be true after -v flag")
		}
	})

	t.Run("it parses --toon global flag", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "--toon", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		if app.config.OutputFormat != FormatTOON {
			t.Errorf("OutputFormat = %q, want %q", app.config.OutputFormat, FormatTOON)
		}
	})

	t.Run("it parses --pretty global flag", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "--pretty", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		if app.config.OutputFormat != FormatPretty {
			t.Errorf("OutputFormat = %q, want %q", app.config.OutputFormat, FormatPretty)
		}
	})

	t.Run("it parses --json global flag", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "--json", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		if app.config.OutputFormat != FormatJSON {
			t.Errorf("OutputFormat = %q, want %q", app.config.OutputFormat, FormatJSON)
		}
	})
}

func TestTTYDetection(t *testing.T) {
	t.Run("it detects TTY vs non-TTY on stdout", func(t *testing.T) {
		// In tests, stdout is never a TTY (it's a pipe).
		// So detectTTY should return false.
		isTTY := DetectTTY()

		// We can only verify the non-TTY case in tests since we're piped
		if isTTY {
			t.Error("detectTTY() = true, expected false in test (pipe) environment")
		}
	})

	t.Run("it defaults to TOON format when not a TTY", func(t *testing.T) {
		app := NewApp()
		// In test environment (not TTY), default should be TOON
		// The format will be set to default since no flags override it
		app.workDir = t.TempDir()

		// Don't set any format flags — let the default apply
		err := app.Run([]string{"tick", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		// When no format flag is set and not TTY, default is TOON
		if app.config.OutputFormat != FormatTOON {
			t.Errorf("OutputFormat = %q, want %q (non-TTY default)", app.config.OutputFormat, FormatTOON)
		}
	})
}
