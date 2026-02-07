package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_CreatesTickDirectory(t *testing.T) {
	t.Run("it creates .tick/ directory in current working directory", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tickDir := filepath.Join(dir, ".tick")
		info, err := os.Stat(tickDir)
		if err != nil {
			t.Fatalf("expected .tick/ directory to exist: %v", err)
		}
		if !info.IsDir() {
			t.Fatalf("expected .tick to be a directory")
		}
	})
}

func TestInit_CreatesEmptyTasksJSONL(t *testing.T) {
	t.Run("it creates empty tasks.jsonl inside .tick/", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		info, err := os.Stat(jsonlPath)
		if err != nil {
			t.Fatalf("expected tasks.jsonl to exist: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("expected tasks.jsonl to be empty (0 bytes), got %d bytes", info.Size())
		}
	})
}

func TestInit_DoesNotCreateCacheDB(t *testing.T) {
	t.Run("it does not create cache.db at init time", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		cacheDBPath := filepath.Join(dir, ".tick", "cache.db")
		_, err := os.Stat(cacheDBPath)
		if err == nil {
			t.Error("expected cache.db to NOT exist at init time, but it does")
		}
		if !os.IsNotExist(err) {
			t.Errorf("unexpected error checking for cache.db: %v", err)
		}
	})
}

func TestInit_PrintsConfirmationWithAbsolutePath(t *testing.T) {
	t.Run("it prints confirmation with absolute path on success", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		absDir, _ := filepath.Abs(dir)
		expected := "Initialized tick in " + absDir + "/.tick/\n"
		if stdout.String() != expected {
			t.Errorf("expected stdout %q, got %q", expected, stdout.String())
		}
	})
}

func TestInit_QuietFlagProducesNoOutput(t *testing.T) {
	t.Run("it prints nothing with --quiet flag on success", func(t *testing.T) {
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

		if stdout.String() != "" {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
		if stderr.String() != "" {
			t.Errorf("expected no stderr with --quiet on success, got %q", stderr.String())
		}
	})

	t.Run("it also accepts -q short flag", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "-q", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no stdout with -q, got %q", stdout.String())
		}
	})
}

func TestInit_ErrorWhenAlreadyExists(t *testing.T) {
	t.Run("it errors when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		// Pre-create .tick/ directory
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to pre-create .tick/: %v", err)
		}

		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}

		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error: ") {
			t.Errorf("expected error prefix 'Error: ', got %q", errMsg)
		}
		if !strings.Contains(errMsg, "already initialized") {
			t.Errorf("expected error to mention 'already initialized', got %q", errMsg)
		}
		if stdout.String() != "" {
			t.Errorf("expected no stdout on error, got %q", stdout.String())
		}
	})

	t.Run("it returns exit code 1 when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to pre-create .tick/: %v", err)
		}

		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})
}

func TestInit_WritesErrorsToStderr(t *testing.T) {
	t.Run("it writes error messages to stderr, not stdout", func(t *testing.T) {
		dir := t.TempDir()
		// Pre-create .tick/ to trigger error
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to pre-create .tick/: %v", err)
		}

		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Dir:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}

		if stdout.String() != "" {
			t.Errorf("expected no stdout on error, got %q", stdout.String())
		}
		if stderr.String() == "" {
			t.Error("expected error message on stderr, got nothing")
		}
		if !strings.HasPrefix(stderr.String(), "Error: ") {
			t.Errorf("expected error message to start with 'Error: ', got %q", stderr.String())
		}
	})
}
