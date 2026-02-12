package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestVerboseLogger(t *testing.T) {
	t.Run("it writes cache/lock/hash/format verbose to stderr", func(t *testing.T) {
		var stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr)

		vl.Log("cache is fresh")
		vl.Log("acquiring exclusive lock")
		vl.Log("lock acquired")
		vl.Log("hash match: yes")
		vl.Log("format resolved: toon")

		output := stderr.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 5 {
			t.Fatalf("expected 5 verbose lines, got %d: %q", len(lines), output)
		}

		expected := []string{
			"verbose: cache is fresh",
			"verbose: acquiring exclusive lock",
			"verbose: lock acquired",
			"verbose: hash match: yes",
			"verbose: format resolved: toon",
		}
		for i, want := range expected {
			if lines[i] != want {
				t.Errorf("line %d = %q, want %q", i, lines[i], want)
			}
		}
	})

	t.Run("it writes nothing to stderr when verbose off", func(t *testing.T) {
		// A nil VerboseLogger represents verbose off.
		var vl *VerboseLogger
		// Should be a no-op, no panic.
		vl.Log("should not appear")
		// No stderr to check — nil logger produces no output.
	})

	t.Run("it does not write verbose to stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr)
		vl.Log("debug message")

		if stdout.String() != "" {
			t.Errorf("stdout should be empty, got %q", stdout.String())
		}
		if stderr.String() == "" {
			t.Error("stderr should have verbose output")
		}
	})

	t.Run("it allows quiet + verbose simultaneously", func(t *testing.T) {
		// quiet affects stdout, verbose affects stderr — they are orthogonal.
		now := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		// --quiet + --verbose: quiet silences stdout, verbose writes debug to stderr.
		exitCode := app.Run([]string{"tick", "--quiet", "--verbose", "list"})
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}

		// Quiet: stdout should only have task IDs (quiet list behavior).
		if stdoutBuf.String() != "tick-aaa111\n" {
			t.Errorf("stdout = %q, want %q", stdoutBuf.String(), "tick-aaa111\n")
		}

		// Verbose: stderr should contain verbose-prefixed lines.
		stderrStr := stderrBuf.String()
		if stderrStr == "" {
			t.Error("stderr should contain verbose output when --verbose is set")
		}
		for _, line := range strings.Split(strings.TrimSpace(stderrStr), "\n") {
			if !strings.HasPrefix(line, "verbose: ") {
				t.Errorf("verbose line %q does not have 'verbose: ' prefix", line)
			}
		}
	})

	t.Run("it works with each format flag without contamination", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}

		formats := []string{"--toon", "--pretty", "--json"}
		for _, flag := range formats {
			t.Run(flag, func(t *testing.T) {
				dir, _ := setupTickProjectWithTasks(t, tasks)
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", "--verbose", flag, "list"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}

				// Stdout should contain only formatted output, no "verbose:" lines.
				if strings.Contains(stdoutBuf.String(), "verbose:") {
					t.Errorf("stdout should not contain verbose lines, got %q", stdoutBuf.String())
				}

				// Stderr should have verbose lines.
				stderrStr := stderrBuf.String()
				if stderrStr == "" {
					t.Errorf("stderr should contain verbose output with %s flag", flag)
				}
				for _, line := range strings.Split(strings.TrimSpace(stderrStr), "\n") {
					if !strings.HasPrefix(line, "verbose: ") {
						t.Errorf("verbose line %q does not have 'verbose: ' prefix", line)
					}
				}
			})
		}
	})

	t.Run("it produces clean piped output with verbose enabled", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "First", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
			IsTTY:  false, // Simulates pipe
		}
		exitCode := app.Run([]string{"tick", "--verbose", "list"})
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}

		// Piped stdout should be pure TOON format — no verbose contamination.
		stdoutStr := stdoutBuf.String()
		if strings.Contains(stdoutStr, "verbose:") {
			t.Errorf("piped stdout should not contain verbose lines, got %q", stdoutStr)
		}

		// Count non-empty lines in stdout — these are the task lines only.
		stdoutLines := strings.Split(strings.TrimSpace(stdoutStr), "\n")
		// TOON list format: header line + 2 data lines = 3 lines.
		if len(stdoutLines) < 2 {
			t.Errorf("expected at least 2 stdout lines (header + tasks), got %d: %q", len(stdoutLines), stdoutStr)
		}

		// Verbose goes to stderr.
		if stderrBuf.String() == "" {
			t.Error("stderr should contain verbose output")
		}
	})

	t.Run("it writes nothing to stderr when verbose off", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		// No --verbose flag.
		exitCode := app.Run([]string{"tick", "list"})
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}

		// Stderr should be completely empty when verbose is off.
		if stderrBuf.String() != "" {
			t.Errorf("stderr should be empty when verbose off, got %q", stderrBuf.String())
		}
	})

	t.Run("it prefixes all lines with verbose:", func(t *testing.T) {
		var stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr)
		vl.Log("test message 1")
		vl.Log("test message 2")
		vl.Log("test message 3")

		for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
			if !strings.HasPrefix(line, "verbose: ") {
				t.Errorf("line %q missing 'verbose: ' prefix", line)
			}
		}
	})
}
