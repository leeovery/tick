package cli

import (
	"bytes"
	"strings"
	"testing"
)

// Test: "it writes cache/lock/hash/format verbose to stderr"
func TestVerbose_WritesDebugToStderr(t *testing.T) {
	dir := setupTickDir(t)
	setupTask(t, dir, "tick-abc123", "Test task")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
	exitCode := app.Run([]string{"tick", "--verbose", "--pretty", "list"})

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	errOutput := stderr.String()

	// Should contain verbose lines about key operations
	if !strings.Contains(errOutput, "verbose:") {
		t.Errorf("expected verbose output on stderr, got: %s", errOutput)
	}

	// Check for format resolution
	if !strings.Contains(errOutput, "format") {
		t.Errorf("expected verbose line about format resolution, got: %s", errOutput)
	}

	// Check for lock/store operations
	if !strings.Contains(errOutput, "lock") || !strings.Contains(errOutput, "store") {
		t.Errorf("expected verbose lines about lock/store operations, got: %s", errOutput)
	}

	// Check for cache/hash/freshness
	if !strings.Contains(errOutput, "cache") {
		t.Errorf("expected verbose line about cache operations, got: %s", errOutput)
	}
}

// Test: "it writes nothing to stderr when verbose off"
func TestVerbose_WritesNothingWhenOff(t *testing.T) {
	dir := setupTickDir(t)
	setupTask(t, dir, "tick-abc123", "Test task")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
	exitCode := app.Run([]string{"tick", "--pretty", "list"})

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	errOutput := stderr.String()
	if strings.Contains(errOutput, "verbose:") {
		t.Errorf("expected no verbose output when verbose off, got: %s", errOutput)
	}
}

// Test: "it does not write verbose to stdout"
func TestVerbose_NeverWritesToStdout(t *testing.T) {
	dir := setupTickDir(t)
	setupTask(t, dir, "tick-abc123", "Test task")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
	exitCode := app.Run([]string{"tick", "--verbose", "--pretty", "list"})

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	stdoutOutput := stdout.String()
	if strings.Contains(stdoutOutput, "verbose:") {
		t.Errorf("verbose output must never appear on stdout, got: %s", stdoutOutput)
	}
}

// Test: "it allows quiet + verbose simultaneously"
func TestVerbose_QuietPlusVerboseSimultaneously(t *testing.T) {
	dir := setupTickDir(t)
	setupTask(t, dir, "tick-abc123", "Test task")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
	exitCode := app.Run([]string{"tick", "--quiet", "--verbose", "--pretty", "list"})

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	// --quiet: stdout should have IDs only (no headers, no format chrome)
	stdoutOutput := stdout.String()
	lines := strings.Split(strings.TrimSpace(stdoutOutput), "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "tick-") {
			t.Errorf("quiet stdout should only have task IDs, got line: %s", line)
		}
	}

	// --verbose: stderr should still have debug output
	errOutput := stderr.String()
	if !strings.Contains(errOutput, "verbose:") {
		t.Errorf("verbose should still write to stderr when quiet is active, got: %s", errOutput)
	}
}

// Test: "it works with each format flag without contamination"
func TestVerbose_WorksWithEachFormatFlag(t *testing.T) {
	formats := []struct {
		name       string
		flag       string
		stdoutHas  string
		stdoutNot  string
	}{
		{"toon format", "--toon", "tasks[", "verbose:"},
		{"pretty format", "--pretty", "ID", "verbose:"},
		{"json format", "--json", `"id"`, "verbose:"},
	}

	for _, tt := range formats {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTickDir(t)
			setupTask(t, dir, "tick-abc123", "Test task")

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
			exitCode := app.Run([]string{"tick", "--verbose", tt.flag, "list"})

			if exitCode != 0 {
				t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
			}

			stdoutOutput := stdout.String()
			if !strings.Contains(stdoutOutput, tt.stdoutHas) {
				t.Errorf("expected stdout to contain %q, got: %s", tt.stdoutHas, stdoutOutput)
			}
			if strings.Contains(stdoutOutput, tt.stdoutNot) {
				t.Errorf("stdout must not contain %q, got: %s", tt.stdoutNot, stdoutOutput)
			}

			errOutput := stderr.String()
			if !strings.Contains(errOutput, "verbose:") {
				t.Errorf("expected verbose output on stderr, got: %s", errOutput)
			}
		})
	}
}

// Test: "it produces clean piped output with verbose enabled"
func TestVerbose_CleanPipedOutput(t *testing.T) {
	dir := setupTickDir(t)
	setupTask(t, dir, "tick-abc123", "First task")
	setupTask(t, dir, "tick-def456", "Second task")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Non-TTY (bytes.Buffer) so default is TOON format
	app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
	exitCode := app.Run([]string{"tick", "--verbose", "list"})

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	// stdout should contain only formatted output, no verbose lines
	stdoutLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range stdoutLines {
		if strings.HasPrefix(line, "verbose:") {
			t.Errorf("piped stdout must not contain verbose lines, got: %s", line)
		}
	}

	// stderr should have verbose output
	if !strings.Contains(stderr.String(), "verbose:") {
		t.Errorf("expected verbose output on stderr, got: %s", stderr.String())
	}
}

// Test: "it prefixes all verbose lines with verbose:"
func TestVerbose_AllLinesPrefixed(t *testing.T) {
	dir := setupTickDir(t)
	setupTask(t, dir, "tick-abc123", "Test task")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
	exitCode := app.Run([]string{"tick", "--verbose", "--pretty", "list"})

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	// Every non-empty line in stderr should start with "verbose:"
	errLines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
	for _, line := range errLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "verbose:") {
			t.Errorf("verbose line not prefixed correctly, got: %s", line)
		}
	}
}

// Test: verbose on mutate commands (create) instruments key operations
func TestVerbose_CreateCommandInstrumented(t *testing.T) {
	dir := setupTickDir(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
	exitCode := app.Run([]string{"tick", "--verbose", "--pretty", "create", "Test task"})

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	errOutput := stderr.String()

	// Should contain verbose lines about key operations
	if !strings.Contains(errOutput, "verbose:") {
		t.Errorf("expected verbose output on stderr for create, got: %s", errOutput)
	}

	// Should mention store/write operations
	if !strings.Contains(errOutput, "store") {
		t.Errorf("expected verbose line about store operations, got: %s", errOutput)
	}

	// No verbose on stdout
	if strings.Contains(stdout.String(), "verbose:") {
		t.Errorf("verbose must not appear on stdout, got: %s", stdout.String())
	}
}
