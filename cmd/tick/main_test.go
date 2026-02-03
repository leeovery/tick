package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildTickBinary builds the tick binary for testing and returns the path to it.
func buildTickBinary(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "tick")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = filepath.Join(findModuleRoot(t), "cmd", "tick")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build tick binary: %v\n%s", err, out)
	}
	return binPath
}

// findModuleRoot walks up from cwd to find the go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}
}

func TestMainIntegration(t *testing.T) {
	binPath := buildTickBinary(t)

	t.Run("it returns exit code 1 when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		cmd := exec.Command(binPath, "init")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected exit code 1, got 0")
		}

		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected ExitError, got %T: %v", err, err)
		}
		if exitErr.ExitCode() != 1 {
			t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
		}

		output := string(out)
		if !strings.HasPrefix(output, "Error: ") {
			t.Errorf("output = %q, want it to start with 'Error: '", output)
		}
	})

	t.Run("it returns exit code 0 on successful init", func(t *testing.T) {
		dir := t.TempDir()

		cmd := exec.Command(binPath, "init")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected exit code 0, got error: %v\n%s", err, out)
		}

		output := string(out)
		if !strings.Contains(output, "Initialized tick in") {
			t.Errorf("output = %q, want it to contain 'Initialized tick in'", output)
		}
	})

	t.Run("it writes error messages to stderr, not stdout", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		cmd := exec.Command(binPath, "init")
		cmd.Dir = dir
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		_ = cmd.Run()

		if stdout.String() != "" {
			t.Errorf("stdout = %q, want empty (errors should go to stderr)", stdout.String())
		}
		if !strings.HasPrefix(stderr.String(), "Error: ") {
			t.Errorf("stderr = %q, want it to start with 'Error: '", stderr.String())
		}
	})

	t.Run("it returns exit code 1 for unknown subcommand", func(t *testing.T) {
		dir := t.TempDir()

		cmd := exec.Command(binPath, "nonexistent")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected exit code 1, got 0")
		}

		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected ExitError, got %T: %v", err, err)
		}
		if exitErr.ExitCode() != 1 {
			t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
		}

		output := string(out)
		if !strings.Contains(output, "Error: Unknown command 'nonexistent'") {
			t.Errorf("output = %q, want it to contain \"Error: Unknown command 'nonexistent'\"", output)
		}
	})

	t.Run("it returns exit code 0 with no subcommand (prints usage)", func(t *testing.T) {
		dir := t.TempDir()

		cmd := exec.Command(binPath)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected exit code 0, got error: %v\n%s", err, out)
		}

		output := string(out)
		if !strings.Contains(output, "Usage:") {
			t.Errorf("output = %q, want it to contain 'Usage:'", output)
		}
	})

	t.Run("it prefixes all error messages with Error:", func(t *testing.T) {
		dir := t.TempDir()

		cmd := exec.Command(binPath, "badcommand")
		cmd.Dir = dir
		var stderr strings.Builder
		cmd.Stderr = &stderr

		_ = cmd.Run()

		got := stderr.String()
		if !strings.HasPrefix(got, "Error: ") {
			t.Errorf("stderr = %q, want prefix 'Error: '", got)
		}
	})
}
