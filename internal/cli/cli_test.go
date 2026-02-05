package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommand(t *testing.T) {
	t.Run("it creates .tick/ directory in current working directory", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tickDir := filepath.Join(dir, ".tick")
		info, err := os.Stat(tickDir)
		if err != nil {
			t.Fatalf("expected .tick/ to exist: %v", err)
		}
		if !info.IsDir() {
			t.Errorf("expected .tick to be a directory")
		}
	})

	t.Run("it creates empty tasks.jsonl inside .tick/", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		content, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("expected tasks.jsonl to exist: %v", err)
		}
		if len(content) != 0 {
			t.Errorf("expected tasks.jsonl to be empty, got %d bytes", len(content))
		}
	})

	t.Run("it does not create cache.db at init time", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		cachePath := filepath.Join(dir, ".tick", "cache.db")
		_, err := os.Stat(cachePath)
		if !os.IsNotExist(err) {
			t.Errorf("expected cache.db to not exist, got: %v", err)
		}
	})

	t.Run("it prints confirmation with absolute path on success", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		absTickDir := filepath.Join(dir, ".tick")
		expected := "Initialized tick in " + absTickDir + "/\n"
		if output != expected {
			t.Errorf("stdout = %q, want %q", output, expected)
		}
	})

	t.Run("it prints nothing with --quiet flag on success", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "init", "--quiet"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.Len() != 0 {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it prints nothing with -q flag on success", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "-q", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.Len() != 0 {
			t.Errorf("expected no stdout with -q, got %q", stdout.String())
		}
	})

	t.Run("it errors when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.HasPrefix(errOutput, "Error: ") {
			t.Errorf("error message should start with 'Error: ', got %q", errOutput)
		}
		if !strings.Contains(errOutput, "already initialized") {
			t.Errorf("error message should mention 'already initialized', got %q", errOutput)
		}
	})

	t.Run("it returns exit code 1 when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	t.Run("it writes error messages to stderr, not stdout", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		app.Run([]string{"tick", "init"})

		if stdout.Len() != 0 {
			t.Errorf("expected stdout to be empty on error, got %q", stdout.String())
		}
		if stderr.Len() == 0 {
			t.Errorf("expected stderr to have error message, got nothing")
		}
	})

	t.Run("it errors when cannot create .tick directory (unwritable parent)", func(t *testing.T) {
		dir := t.TempDir()
		readOnlyDir := filepath.Join(dir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
			t.Fatalf("failed to create readonly dir: %v", err)
		}

		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    readOnlyDir,
		}

		code := app.Run([]string{"tick", "init"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		if !strings.HasPrefix(errOutput, "Error: ") {
			t.Errorf("error should start with 'Error: ', got %q", errOutput)
		}
		if !strings.Contains(errOutput, "Could not create .tick/ directory") {
			t.Errorf("error should mention 'Could not create .tick/ directory', got %q", errOutput)
		}
	})
}

func TestDiscoverTickDir(t *testing.T) {
	t.Run("it discovers .tick/ directory by walking up from cwd", func(t *testing.T) {
		// Create structure: /tmp/project/.tick/tasks.jsonl with /tmp/project/src/nested as cwd
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		nestedDir := filepath.Join(dir, "src", "nested")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}

		found, err := DiscoverTickDir(nestedDir)
		if err != nil {
			t.Fatalf("expected to find .tick, got error: %v", err)
		}
		if found != tickDir {
			t.Errorf("discovered path = %q, want %q", found, tickDir)
		}
	})

	t.Run("it finds .tick/ in current directory", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create tasks.jsonl: %v", err)
		}

		found, err := DiscoverTickDir(dir)
		if err != nil {
			t.Fatalf("expected to find .tick, got error: %v", err)
		}
		if found != tickDir {
			t.Errorf("discovered path = %q, want %q", found, tickDir)
		}
	})

	t.Run("it errors when no .tick/ directory found (not a tick project)", func(t *testing.T) {
		dir := t.TempDir()

		_, err := DiscoverTickDir(dir)
		if err == nil {
			t.Fatal("expected error when .tick not found")
		}

		expected := "not a tick project (no .tick directory found)"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})
}

func TestUnknownSubcommand(t *testing.T) {
	t.Run("it routes unknown subcommands to error", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick", "unknown"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}

		errOutput := stderr.String()
		expected := "Error: Unknown command 'unknown'. Run 'tick help' for usage.\n"
		if errOutput != expected {
			t.Errorf("stderr = %q, want %q", errOutput, expected)
		}
	})
}

func TestNoSubcommand(t *testing.T) {
	t.Run("it prints usage when no subcommand provided", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Cwd:    dir,
		}

		code := app.Run([]string{"tick"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}

		output := stdout.String()
		if !strings.Contains(output, "Usage:") {
			t.Errorf("expected usage info, got %q", output)
		}
		if !strings.Contains(output, "init") {
			t.Errorf("expected 'init' in usage, got %q", output)
		}
	})
}

func TestTTYDetection(t *testing.T) {
	t.Run("it detects TTY vs non-TTY on stdout", func(t *testing.T) {
		// Non-TTY case: bytes.Buffer is not a TTY
		var buf bytes.Buffer
		if IsTTY(&buf) {
			t.Error("bytes.Buffer should not be detected as TTY")
		}

		// Note: We can't easily test actual TTY detection in unit tests
		// because /dev/tty is not available in CI environments.
		// The important thing is that IsTTY works with os.File.
	})

	t.Run("it sets default output format based on TTY detection", func(t *testing.T) {
		// Non-TTY defaults to TOON
		var buf bytes.Buffer
		app := &App{
			Stdout: &buf,
			Stderr: &buf,
		}

		if app.DefaultOutputFormat() != "toon" {
			t.Errorf("non-TTY should default to 'toon', got %q", app.DefaultOutputFormat())
		}
	})
}

func TestGlobalFlags(t *testing.T) {
	t.Run("it parses --quiet flag", func(t *testing.T) {
		flags, args := ParseGlobalFlags([]string{"tick", "--quiet", "init"})
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if len(args) != 2 || args[1] != "init" {
			t.Errorf("remaining args = %v, want [tick init]", args)
		}
	})

	t.Run("it parses -q flag", func(t *testing.T) {
		flags, args := ParseGlobalFlags([]string{"tick", "-q", "init"})
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if len(args) != 2 || args[1] != "init" {
			t.Errorf("remaining args = %v, want [tick init]", args)
		}
	})

	t.Run("it parses --verbose flag", func(t *testing.T) {
		flags, args := ParseGlobalFlags([]string{"tick", "--verbose", "init"})
		if !flags.Verbose {
			t.Error("expected Verbose to be true")
		}
		if len(args) != 2 || args[1] != "init" {
			t.Errorf("remaining args = %v, want [tick init]", args)
		}
	})

	t.Run("it parses -v flag", func(t *testing.T) {
		flags, args := ParseGlobalFlags([]string{"tick", "-v", "init"})
		if !flags.Verbose {
			t.Error("expected Verbose to be true")
		}
		if len(args) != 2 || args[1] != "init" {
			t.Errorf("remaining args = %v, want [tick init]", args)
		}
	})

	t.Run("it parses --toon flag", func(t *testing.T) {
		flags, _ := ParseGlobalFlags([]string{"tick", "--toon", "list"})
		if flags.OutputFormat != "toon" {
			t.Errorf("OutputFormat = %q, want %q", flags.OutputFormat, "toon")
		}
	})

	t.Run("it parses --pretty flag", func(t *testing.T) {
		flags, _ := ParseGlobalFlags([]string{"tick", "--pretty", "list"})
		if flags.OutputFormat != "pretty" {
			t.Errorf("OutputFormat = %q, want %q", flags.OutputFormat, "pretty")
		}
	})

	t.Run("it parses --json flag", func(t *testing.T) {
		flags, _ := ParseGlobalFlags([]string{"tick", "--json", "list"})
		if flags.OutputFormat != "json" {
			t.Errorf("OutputFormat = %q, want %q", flags.OutputFormat, "json")
		}
	})

	t.Run("flags can appear after subcommand", func(t *testing.T) {
		flags, args := ParseGlobalFlags([]string{"tick", "init", "--quiet"})
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if len(args) != 2 || args[1] != "init" {
			t.Errorf("remaining args = %v, want [tick init]", args)
		}
	})

	t.Run("multiple flags can be combined", func(t *testing.T) {
		flags, _ := ParseGlobalFlags([]string{"tick", "-q", "-v", "--toon", "init"})
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if !flags.Verbose {
			t.Error("expected Verbose to be true")
		}
		if flags.OutputFormat != "toon" {
			t.Errorf("OutputFormat = %q, want %q", flags.OutputFormat, "toon")
		}
	})
}
