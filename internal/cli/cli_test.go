package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	t.Run("it creates .tick/ directory in current working directory", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tickDir := filepath.Join(dir, ".tick")
		info, err := os.Stat(tickDir)
		if err != nil {
			t.Fatalf("expected .tick/ to exist: %v", err)
		}
		if !info.IsDir() {
			t.Fatal("expected .tick to be a directory")
		}
	})

	t.Run("it creates empty tasks.jsonl inside .tick/", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		info, err := os.Stat(jsonlPath)
		if err != nil {
			t.Fatalf("expected tasks.jsonl to exist: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("expected tasks.jsonl to be empty, got %d bytes", info.Size())
		}
	})

	t.Run("it does not create cache.db at init time", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		cacheDB := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cacheDB); err == nil {
			t.Fatal("expected cache.db not to exist at init time")
		}
	})

	t.Run("it prints confirmation with absolute path on success", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		absTickDir := filepath.Join(dir, ".tick")
		expected := "Initialized tick in " + absTickDir + "/"
		got := strings.TrimSpace(stdout.String())
		if got != expected {
			t.Errorf("stdout = %q, want %q", got, expected)
		}
	})

	t.Run("it prints nothing with --quiet flag on success", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it errors when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		// Pre-create .tick/ directory
		if err := os.Mkdir(filepath.Join(dir, ".tick"), 0755); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if stdout.String() != "" {
			t.Errorf("expected no stdout on error, got %q", stdout.String())
		}
	})

	t.Run("it returns exit code 1 when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".tick"), 0755); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	t.Run("it writes error messages to stderr, not stdout", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".tick"), 0755); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		if stdout.String() != "" {
			t.Errorf("expected no stdout on error, got %q", stdout.String())
		}
		if !strings.HasPrefix(stderr.String(), "Error: ") {
			t.Errorf("expected stderr to start with 'Error: ', got %q", stderr.String())
		}
	})

	t.Run("it errors even when .tick/ exists but is corrupted (missing tasks.jsonl)", func(t *testing.T) {
		dir := t.TempDir()
		// Create .tick/ directory but without tasks.jsonl (corrupted)
		if err := os.Mkdir(filepath.Join(dir, ".tick"), 0755); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Fatalf("expected exit code 1 for corrupted .tick/, got %d", code)
		}
	})

	t.Run("it accepts -q shorthand for --quiet", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "-q", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
		if stdout.String() != "" {
			t.Errorf("expected no stdout with -q, got %q", stdout.String())
		}
	})
}

func TestDiscoverTickDir(t *testing.T) {
	t.Run("it discovers .tick/ directory by walking up from cwd", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create a subdirectory to start from
		subdir := filepath.Join(dir, "sub", "deep")
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatal(err)
		}

		found, err := DiscoverTickDir(subdir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != tickDir {
			t.Errorf("found = %q, want %q", found, tickDir)
		}
	})

	t.Run("it errors when no .tick/ directory found (not a tick project)", func(t *testing.T) {
		dir := t.TempDir()

		_, err := DiscoverTickDir(dir)
		if err == nil {
			t.Fatal("expected error when no .tick/ found")
		}
		expected := "Not a tick project (no .tick directory found)"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it finds .tick/ in the starting directory itself", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatal(err)
		}

		found, err := DiscoverTickDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != tickDir {
			t.Errorf("found = %q, want %q", found, tickDir)
		}
	})
}

func TestSubcommandRouting(t *testing.T) {
	t.Run("it routes unknown subcommands to error", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "nosuchcommand"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		expected := "Error: Unknown command 'nosuchcommand'. Run 'tick help' for usage.\n"
		if stderr.String() != expected {
			t.Errorf("stderr = %q, want %q", stderr.String(), expected)
		}
	})

	t.Run("it prints basic usage with no subcommand and exits 0", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
		if stdout.String() == "" {
			t.Error("expected usage output, got empty stdout")
		}
	})

	t.Run("it does not advertise doctor command in help output", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		if strings.Contains(stdout.String(), "doctor") {
			t.Error("help output should not advertise the unimplemented doctor command")
		}
	})

	t.Run("it returns unknown command error for doctor", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "doctor"}, dir, &stdout, &stderr, false)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		expected := "Error: Unknown command 'doctor'. Run 'tick help' for usage.\n"
		if stderr.String() != expected {
			t.Errorf("stderr = %q, want %q", stderr.String(), expected)
		}
	})
}

func TestTTYDetection(t *testing.T) {
	t.Run("it detects TTY vs non-TTY on stdout", func(t *testing.T) {
		dir := t.TempDir()

		// isTTY=false (non-TTY / pipe) -> default format should be toon
		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick"}, dir, &stdout, &stderr, false)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}

		// isTTY=true (terminal) -> default format should be pretty
		var stdout2, stderr2 bytes.Buffer
		code2 := Run([]string{"tick"}, dir, &stdout2, &stderr2, true)
		if code2 != 0 {
			t.Fatalf("expected exit code 0, got %d", code2)
		}
		// Both should succeed; format selection is stored internally.
		// We test that the isTTY parameter is accepted and doesn't crash.
	})
}

func TestGlobalFlags(t *testing.T) {
	t.Run("it parses --verbose flag", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--verbose", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses -v shorthand for --verbose", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "-v", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses --toon flag", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses --pretty flag", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("it parses --json flag", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})
}
