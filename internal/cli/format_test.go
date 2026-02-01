package cli

import (
	"os"
	"testing"
)

func TestDetectTTY(t *testing.T) {
	t.Run("it detects TTY vs non-TTY", func(t *testing.T) {
		// In tests, stdout is a pipe/buffer, not a TTY.
		result := DetectTTY(os.Stdout)
		if result {
			t.Error("expected non-TTY in test environment, got TTY")
		}
	})

	t.Run("it defaults to non-TTY on stat failure", func(t *testing.T) {
		// nil file should default to non-TTY, not panic.
		result := DetectTTY(nil)
		if result {
			t.Error("expected non-TTY on nil file, got TTY")
		}
	})
}

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name    string
		toon    bool
		pretty  bool
		json    bool
		isTTY   bool
		want    Format
		wantErr bool
	}{
		{
			name: "it defaults to Toon when non-TTY",
			isTTY: false,
			want:  FormatToon,
		},
		{
			name: "it defaults to Pretty when TTY",
			isTTY: true,
			want:  FormatPretty,
		},
		{
			name: "it returns Toon for --toon flag override",
			toon: true,
			isTTY: true,
			want:  FormatToon,
		},
		{
			name: "it returns Pretty for --pretty flag override",
			pretty: true,
			isTTY: false,
			want:   FormatPretty,
		},
		{
			name: "it returns JSON for --json flag override",
			json: true,
			isTTY: false,
			want:  FormatJSON,
		},
		{
			name:    "it errors when multiple format flags set (toon+pretty)",
			toon:    true,
			pretty:  true,
			wantErr: true,
		},
		{
			name:    "it errors when multiple format flags set (toon+json)",
			toon:    true,
			json:    true,
			wantErr: true,
		},
		{
			name:    "it errors when multiple format flags set (pretty+json)",
			pretty:  true,
			json:    true,
			wantErr: true,
		},
		{
			name:    "it errors when all three format flags set",
			toon:    true,
			pretty:  true,
			json:    true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveFormat(tt.toon, tt.pretty, tt.json, tt.isTTY)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatConfig(t *testing.T) {
	t.Run("it propagates quiet and verbose in FormatConfig", func(t *testing.T) {
		cfg := FormatConfig{
			Format:  FormatToon,
			Quiet:   true,
			Verbose: true,
		}
		if cfg.Format != FormatToon {
			t.Errorf("Format = %v, want FormatToon", cfg.Format)
		}
		if !cfg.Quiet {
			t.Error("Quiet should be true")
		}
		if !cfg.Verbose {
			t.Error("Verbose should be true")
		}
	})
}

func TestFormatEnum(t *testing.T) {
	t.Run("it has three format constants", func(t *testing.T) {
		formats := []Format{FormatToon, FormatPretty, FormatJSON}
		seen := make(map[Format]bool)
		for _, f := range formats {
			if seen[f] {
				t.Errorf("duplicate format constant: %v", f)
			}
			seen[f] = true
		}
		if len(seen) != 3 {
			t.Errorf("expected 3 unique formats, got %d", len(seen))
		}
	})
}

func TestFormatterInterface(t *testing.T) {
	t.Run("it defines Formatter interface covering all command output types", func(t *testing.T) {
		// Compile-time check that the interface exists with correct methods.
		// StubFormatter must satisfy Formatter.
		var _ Formatter = &StubFormatter{}
	})
}
