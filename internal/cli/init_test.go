package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommand(t *testing.T) {
	t.Run("it creates .tick/ directory in current working directory", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		err := app.Run([]string{"tick", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		tickDir := filepath.Join(dir, ".tick")
		info, err := os.Stat(tickDir)
		if err != nil {
			t.Fatalf(".tick/ directory not found: %v", err)
		}
		if !info.IsDir() {
			t.Error(".tick is not a directory")
		}
	})

	t.Run("it creates empty tasks.jsonl inside .tick/", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		err := app.Run([]string{"tick", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		info, err := os.Stat(jsonlPath)
		if err != nil {
			t.Fatalf("tasks.jsonl not found: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("tasks.jsonl size = %d, want 0 (empty)", info.Size())
		}
	})

	t.Run("it does not create cache.db at init time", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		err := app.Run([]string{"tick", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		cachePath := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
			t.Error("cache.db should not exist at init time")
		}
	})

	t.Run("it prints confirmation with absolute path on success", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout
		err := app.Run([]string{"tick", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		absDir, _ := filepath.Abs(dir)
		want := "Initialized tick in " + filepath.Join(absDir, ".tick") + "/\n"
		got := stdout.String()
		if got != want {
			t.Errorf("stdout = %q, want %q", got, want)
		}
	})

	t.Run("it prints nothing with --quiet flag on success", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout
		err := app.Run([]string{"tick", "--quiet", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		got := stdout.String()
		if got != "" {
			t.Errorf("stdout = %q, want empty string with --quiet", got)
		}
	})

	t.Run("it prints nothing with -q flag on success", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout
		err := app.Run([]string{"tick", "-q", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		got := stdout.String()
		if got != "" {
			t.Errorf("stdout = %q, want empty string with -q", got)
		}
	})

	t.Run("it errors when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		app := NewApp()
		app.workDir = dir
		err := app.Run([]string{"tick", "init"})
		if err == nil {
			t.Fatal("init expected error when .tick/ already exists, got nil")
		}

		if !strings.Contains(err.Error(), "already initialized") {
			t.Errorf("error = %q, want it to contain 'already initialized'", err.Error())
		}
	})

	t.Run("it errors when .tick/ already exists even if corrupted (missing tasks.jsonl)", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}
		// No tasks.jsonl created - corrupted .tick/

		app := NewApp()
		app.workDir = dir
		err := app.Run([]string{"tick", "init"})
		if err == nil {
			t.Fatal("init expected error for corrupted .tick/, got nil")
		}
	})

	t.Run("it writes error messages to stderr, not stdout", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "init"})
		if err == nil {
			t.Fatal("init expected error, got nil")
		}

		// stdout should be empty on error
		if stdout.String() != "" {
			t.Errorf("stdout = %q, want empty (errors go to stderr)", stdout.String())
		}
	})

	t.Run("it surfaces OS error when directory is not writable", func(t *testing.T) {
		dir := t.TempDir()
		// Make the directory read-only
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0755) })

		app := NewApp()
		app.workDir = dir
		err := app.Run([]string{"tick", "init"})
		if err == nil {
			t.Fatal("init expected error for read-only directory, got nil")
		}

		if !strings.Contains(err.Error(), "Could not create .tick/ directory") {
			t.Errorf("error = %q, want it to contain 'Could not create .tick/ directory'", err.Error())
		}
	})
}
