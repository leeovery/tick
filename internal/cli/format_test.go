package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestDetectTTY(t *testing.T) {
	t.Run("it detects non-TTY when stdout is not an os.File", func(t *testing.T) {
		var buf bytes.Buffer
		isTTY := DetectTTY(&buf)
		if isTTY {
			t.Error("expected non-TTY for bytes.Buffer, got TTY")
		}
	})

	t.Run("it detects non-TTY for pipe file descriptor", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		isTTY := DetectTTY(w)
		if isTTY {
			t.Error("expected non-TTY for pipe, got TTY")
		}
	})

	t.Run("it defaults to non-TTY on stat failure", func(t *testing.T) {
		// Create a file and close it so Stat fails
		f, err := os.CreateTemp(t.TempDir(), "closed-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		name := f.Name()
		f.Close()

		// Reopen and immediately close to get a closed *os.File
		f2, err := os.Open(name)
		if err != nil {
			t.Fatalf("failed to open temp file: %v", err)
		}
		f2.Close() // Close so Stat will fail

		isTTY := DetectTTY(f2)
		if isTTY {
			t.Error("expected non-TTY on stat failure, got TTY")
		}
	})
}

func TestResolveFormat(t *testing.T) {
	t.Run("it defaults to Toon when non-TTY", func(t *testing.T) {
		format, err := ResolveFormat(false, false, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatToon {
			t.Errorf("expected FormatToon for non-TTY default, got %v", format)
		}
	})

	t.Run("it defaults to Pretty when TTY", func(t *testing.T) {
		format, err := ResolveFormat(false, false, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("expected FormatPretty for TTY default, got %v", format)
		}
	})

	t.Run("it returns correct format for --toon flag override", func(t *testing.T) {
		// Even with TTY, --toon overrides
		format, err := ResolveFormat(true, false, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatToon {
			t.Errorf("expected FormatToon with --toon flag, got %v", format)
		}
	})

	t.Run("it returns correct format for --pretty flag override", func(t *testing.T) {
		// Even with non-TTY, --pretty overrides
		format, err := ResolveFormat(false, true, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("expected FormatPretty with --pretty flag, got %v", format)
		}
	})

	t.Run("it returns correct format for --json flag override", func(t *testing.T) {
		format, err := ResolveFormat(false, false, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatJSON {
			t.Errorf("expected FormatJSON with --json flag, got %v", format)
		}
	})

	t.Run("it errors when multiple format flags set", func(t *testing.T) {
		tests := []struct {
			name   string
			toon   bool
			pretty bool
			json   bool
		}{
			{"toon and pretty", true, true, false},
			{"toon and json", true, false, true},
			{"pretty and json", false, true, true},
			{"all three", true, true, true},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := ResolveFormat(tc.toon, tc.pretty, tc.json, false)
				if err == nil {
					t.Error("expected error for conflicting format flags, got nil")
				}
			})
		}
	})
}

func TestCLI_ConflictingFormatFlags(t *testing.T) {
	t.Run("it errors when multiple format flags set via CLI", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "--toon", "--pretty", "init"})
		if code != 1 {
			t.Errorf("expected exit code 1 for conflicting flags, got %d", code)
		}

		errMsg := stderr.String()
		if errMsg == "" {
			t.Error("expected error message on stderr for conflicting flags")
		}
	})
}
