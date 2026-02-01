package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCommand(t *testing.T) {
	t.Run("creates .tick directory in target directory", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		code := app.Run([]string{"tick", "init"}, dir)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tickDir := filepath.Join(dir, ".tick")
		info, err := os.Stat(tickDir)
		if err != nil {
			t.Fatalf(".tick directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error(".tick should be a directory")
		}
	})

	t.Run("creates empty tasks.jsonl inside .tick", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		app.Run([]string{"tick", "init"}, dir)

		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		info, err := os.Stat(jsonlPath)
		if err != nil {
			t.Fatalf("tasks.jsonl not created: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("tasks.jsonl should be empty, size = %d", info.Size())
		}
	})

	t.Run("does not create cache.db at init time", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		app.Run([]string{"tick", "init"}, dir)

		cachePath := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cachePath); err == nil {
			t.Error("cache.db should not be created at init")
		}
	})

	t.Run("prints confirmation with absolute path", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		app.Run([]string{"tick", "init"}, dir)

		absDir, _ := filepath.Abs(dir)
		expected := "Initialized tick in " + filepath.Join(absDir, ".tick") + "/\n"
		if stdout.String() != expected {
			t.Errorf("stdout = %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("prints nothing with --quiet flag", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		code := app.Run([]string{"tick", "--quiet", "init"}, dir)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout.String())
		}
	})

	t.Run("errors when .tick already exists", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".tick"), 0755)

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		code := app.Run([]string{"tick", "init"}, dir)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !bytes.Contains(stderr.Bytes(), []byte("Tick already initialized")) {
			t.Errorf("stderr should contain 'Tick already initialized', got %q", stderr.String())
		}
	})

	t.Run("writes error messages to stderr not stdout", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".tick"), 0755)

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		app.Run([]string{"tick", "init"}, dir)

		if stdout.Len() != 0 {
			t.Errorf("stdout should be empty on error, got %q", stdout.String())
		}
		if stderr.Len() == 0 {
			t.Error("stderr should contain error message")
		}
	})
}

func TestFindTickDir(t *testing.T) {
	t.Run("discovers .tick directory by walking up from cwd", func(t *testing.T) {
		root := t.TempDir()
		tickDir := filepath.Join(root, ".tick")
		os.MkdirAll(tickDir, 0755)
		os.WriteFile(filepath.Join(tickDir, "tasks.jsonl"), []byte(""), 0644)

		// Create nested directory
		nested := filepath.Join(root, "sub", "deep")
		os.MkdirAll(nested, 0755)

		found, err := FindTickDir(nested)
		if err != nil {
			t.Fatalf("FindTickDir() error: %v", err)
		}
		if found != tickDir {
			t.Errorf("found = %q, want %q", found, tickDir)
		}
	})

	t.Run("errors when no .tick directory found", func(t *testing.T) {
		dir := t.TempDir()
		_, err := FindTickDir(dir)
		if err == nil {
			t.Fatal("expected error for missing .tick directory")
		}
	})
}

func TestUnknownSubcommand(t *testing.T) {
	t.Run("routes unknown subcommands to error", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		code := app.Run([]string{"tick", "nonexistent"}, dir)

		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !bytes.Contains(stderr.Bytes(), []byte("Unknown command")) {
			t.Errorf("stderr should contain 'Unknown command', got %q", stderr.String())
		}
	})
}

func TestGlobalFlags(t *testing.T) {
	t.Run("parses --quiet flag", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		code := app.Run([]string{"tick", "--quiet", "init"}, dir)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})

	t.Run("parses -q short form", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		app := NewApp(&stdout, &stderr)
		code := app.Run([]string{"tick", "-q", "init"}, dir)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}
	})
}
