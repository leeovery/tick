package cli

import (
	"os"
	"strings"
	"testing"
)

func TestDetectTTY(t *testing.T) {
	t.Run("it detects TTY vs non-TTY", func(t *testing.T) {
		// In test environment, stdout is piped, so DetectTTY should return false.
		result := DetectTTY()
		if result {
			t.Error("DetectTTY() = true, expected false in test (pipe) environment")
		}
	})

	t.Run("it defaults to non-TTY on stat failure", func(t *testing.T) {
		// DetectTTYFrom with a statFn that returns an error should default to false.
		result := DetectTTYFrom(func() (FileInfo, error) {
			return nil, errStatFailure
		})
		if result {
			t.Error("DetectTTYFrom() = true, expected false on stat failure")
		}
	})

	t.Run("it returns true when ModeCharDevice is set", func(t *testing.T) {
		fi := &fakeFileInfo{mode: os.ModeCharDevice}
		result := DetectTTYFrom(func() (FileInfo, error) {
			return fi, nil
		})
		if !result {
			t.Error("DetectTTYFrom() = false, expected true when ModeCharDevice is set")
		}
	})

	t.Run("it returns false when ModeCharDevice is not set", func(t *testing.T) {
		fi := &fakeFileInfo{mode: 0}
		result := DetectTTYFrom(func() (FileInfo, error) {
			return fi, nil
		})
		if result {
			t.Error("DetectTTYFrom() = true, expected false when ModeCharDevice is not set")
		}
	})
}

func TestResolveFormat(t *testing.T) {
	t.Run("it defaults to Toon when non-TTY, Pretty when TTY", func(t *testing.T) {
		tests := []struct {
			name  string
			isTTY bool
			want  OutputFormat
		}{
			{"non-TTY defaults to Toon", false, FormatTOON},
			{"TTY defaults to Pretty", true, FormatPretty},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := ResolveFormat(false, false, false, tt.isTTY)
				if err != nil {
					t.Fatalf("ResolveFormat() returned unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("ResolveFormat() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("it returns correct format for each flag override", func(t *testing.T) {
		tests := []struct {
			name   string
			toon   bool
			pretty bool
			json   bool
			isTTY  bool
			want   OutputFormat
		}{
			{"--toon overrides TTY", true, false, false, true, FormatTOON},
			{"--toon overrides non-TTY", true, false, false, false, FormatTOON},
			{"--pretty overrides non-TTY", false, true, false, false, FormatPretty},
			{"--pretty overrides TTY", false, true, false, true, FormatPretty},
			{"--json overrides TTY", false, false, true, true, FormatJSON},
			{"--json overrides non-TTY", false, false, true, false, FormatJSON},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := ResolveFormat(tt.toon, tt.pretty, tt.json, tt.isTTY)
				if err != nil {
					t.Fatalf("ResolveFormat() returned unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("ResolveFormat() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("it errors when multiple format flags set", func(t *testing.T) {
		tests := []struct {
			name   string
			toon   bool
			pretty bool
			json   bool
		}{
			{"--toon and --pretty conflict", true, true, false},
			{"--toon and --json conflict", true, false, true},
			{"--pretty and --json conflict", false, true, true},
			{"all three conflict", true, true, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ResolveFormat(tt.toon, tt.pretty, tt.json, false)
				if err == nil {
					t.Fatal("ResolveFormat() expected error for conflicting flags, got nil")
				}
				want := "only one format flag allowed: --toon, --pretty, or --json"
				if err.Error() != want {
					t.Errorf("error = %q, want %q", err.Error(), want)
				}
			})
		}
	})
}

func TestFormatConfig(t *testing.T) {
	t.Run("it propagates quiet and verbose in FormatConfig", func(t *testing.T) {
		tests := []struct {
			name    string
			format  OutputFormat
			quiet   bool
			verbose bool
		}{
			{"all false", FormatTOON, false, false},
			{"quiet only", FormatPretty, true, false},
			{"verbose only", FormatJSON, false, true},
			{"both quiet and verbose", FormatTOON, true, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fc := FormatConfig{
					Format:  tt.format,
					Quiet:   tt.quiet,
					Verbose: tt.verbose,
				}
				if fc.Format != tt.format {
					t.Errorf("FormatConfig.Format = %q, want %q", fc.Format, tt.format)
				}
				if fc.Quiet != tt.quiet {
					t.Errorf("FormatConfig.Quiet = %v, want %v", fc.Quiet, tt.quiet)
				}
				if fc.Verbose != tt.verbose {
					t.Errorf("FormatConfig.Verbose = %v, want %v", fc.Verbose, tt.verbose)
				}
			})
		}
	})
}

func TestFormatterInterface(t *testing.T) {
	t.Run("it has a stub formatter that satisfies the Formatter interface", func(t *testing.T) {
		var f Formatter = &StubFormatter{}
		_ = f // Verify it compiles as a Formatter
	})
}

func TestAppConflictingFormatFlags(t *testing.T) {
	t.Run("it errors when multiple format flags set via App.Run", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		err := app.Run([]string{"tick", "--toon", "--pretty", "init"})
		if err == nil {
			t.Fatal("Run() expected error for conflicting format flags, got nil")
		}

		want := "only one format flag allowed: --toon, --pretty, or --json"
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})
}

func TestAppFormatConfigWiring(t *testing.T) {
	t.Run("it wires FormatConfig into App from config", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()

		// Run with --quiet --verbose --toon and init to exercise full wiring
		err := app.Run([]string{"tick", "--quiet", "--verbose", "--toon", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		fc := app.formatConfig()
		if fc.Format != FormatTOON {
			t.Errorf("FormatConfig.Format = %q, want %q", fc.Format, FormatTOON)
		}
		if !fc.Quiet {
			t.Error("FormatConfig.Quiet = false, want true")
		}
		if !fc.Verbose {
			t.Error("FormatConfig.Verbose = false, want true")
		}
	})

	t.Run("it stores FormatConfig on App during dispatch", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--toon", "init"})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		// After dispatch, App.FormatCfg should be populated
		if app.FormatCfg.Format != FormatTOON {
			t.Errorf("App.FormatCfg.Format = %q, want %q", app.FormatCfg.Format, FormatTOON)
		}
	})
}

func TestVerboseStderr(t *testing.T) {
	t.Run("it writes verbose output to stderr not stdout", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		// Enable verbose mode
		app.config.Verbose = true
		app.config.OutputFormat = FormatTOON

		// Use the verbose helper
		app.logVerbose("debug: test message")

		if !strings.Contains(stderr.String(), "debug: test message") {
			t.Errorf("stderr = %q, expected it to contain verbose message", stderr.String())
		}
		if strings.Contains(stdout.String(), "debug: test message") {
			t.Error("stdout contains verbose message, expected verbose output only on stderr")
		}
	})

	t.Run("it does not write verbose output when verbose is disabled", func(t *testing.T) {
		app := NewApp()
		app.workDir = t.TempDir()
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		// Verbose is false by default
		app.logVerbose("debug: should not appear")

		if stderr.String() != "" {
			t.Errorf("stderr = %q, expected empty when verbose is disabled", stderr.String())
		}
		if stdout.String() != "" {
			t.Errorf("stdout = %q, expected empty when verbose is disabled", stdout.String())
		}
	})
}

// fakeFileInfo implements the FileInfo interface for testing TTY detection.
type fakeFileInfo struct {
	mode os.FileMode
}

func (f *fakeFileInfo) Mode() os.FileMode {
	return f.mode
}
