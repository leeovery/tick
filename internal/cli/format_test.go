package cli

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestDetectTTY(t *testing.T) {
	t.Run("it detects TTY vs non-TTY", func(t *testing.T) {
		// A pipe (os.Pipe) is not a TTY.
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		defer w.Close()

		result := DetectTTY(w)
		if result != false {
			t.Errorf("expected pipe to be non-TTY, got TTY")
		}
	})

	t.Run("it defaults to non-TTY on stat failure", func(t *testing.T) {
		// A closed file will fail Stat.
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		r.Close()
		w.Close()

		result := DetectTTY(w)
		if result != false {
			t.Errorf("expected closed file to default to non-TTY, got TTY")
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
			t.Errorf("expected FormatToon, got %v", format)
		}
	})

	t.Run("it defaults to Pretty when TTY", func(t *testing.T) {
		format, err := ResolveFormat(false, false, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("expected FormatPretty, got %v", format)
		}
	})

	t.Run("it returns FormatToon when toon flag set", func(t *testing.T) {
		format, err := ResolveFormat(true, false, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatToon {
			t.Errorf("expected FormatToon, got %v", format)
		}
	})

	t.Run("it returns FormatPretty when pretty flag set", func(t *testing.T) {
		format, err := ResolveFormat(false, true, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("expected FormatPretty, got %v", format)
		}
	})

	t.Run("it returns FormatJSON when json flag set", func(t *testing.T) {
		format, err := ResolveFormat(false, false, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatJSON {
			t.Errorf("expected FormatJSON, got %v", format)
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
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ResolveFormat(tt.toon, tt.pretty, tt.json, false)
				if err == nil {
					t.Fatal("expected error for conflicting format flags")
				}
				expected := "only one format flag may be specified (--toon, --pretty, --json)"
				if err.Error() != expected {
					t.Errorf("error = %q, want %q", err.Error(), expected)
				}
			})
		}
	})
}

func TestFormatConfig(t *testing.T) {
	t.Run("it propagates quiet and verbose in FormatConfig", func(t *testing.T) {
		cfg := FormatConfig{
			Format:  FormatJSON,
			Quiet:   true,
			Verbose: true,
		}
		if cfg.Format != FormatJSON {
			t.Errorf("Format = %v, want FormatJSON", cfg.Format)
		}
		if !cfg.Quiet {
			t.Error("expected Quiet to be true")
		}
		if !cfg.Verbose {
			t.Error("expected Verbose to be true")
		}
	})
}

func TestStubFormatter(t *testing.T) {
	t.Run("it implements Formatter interface", func(t *testing.T) {
		var f Formatter = &StubFormatter{}
		_ = f
	})

	t.Run("it returns placeholder output for FormatMessage", func(t *testing.T) {
		var buf bytes.Buffer
		f := &StubFormatter{}
		f.FormatMessage(&buf, "hello")
		if buf.String() != "hello\n" {
			t.Errorf("FormatMessage output = %q, want %q", buf.String(), "hello\n")
		}
	})
}

func TestConflictingFormatFlagsIntegration(t *testing.T) {
	t.Run("it errors when multiple format flags passed via CLI", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "--pretty", "init"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if stderr.String() == "" {
			t.Fatal("expected error message in stderr")
		}
	})
}

func TestFormatConfigWiredIntoContext(t *testing.T) {
	t.Run("it creates FormatConfig from Context", func(t *testing.T) {
		ctx := &Context{
			Format:  FormatJSON,
			Quiet:   true,
			Verbose: true,
			Stdout:  io.Discard,
			Stderr:  io.Discard,
		}

		cfg := ctx.FormatCfg()
		if cfg.Format != FormatJSON {
			t.Errorf("Format = %v, want FormatJSON", cfg.Format)
		}
		if !cfg.Quiet {
			t.Error("expected Quiet to be true")
		}
		if !cfg.Verbose {
			t.Error("expected Verbose to be true")
		}
	})
}
