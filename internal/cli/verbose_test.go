package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestVerbose(t *testing.T) {
	t.Run("it writes cache/lock/hash/format verbose to stderr", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--verbose", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stderrStr := stderr.String()
		expectedPhrases := []string{
			"verbose: lock acquired",
			"verbose: cache freshness check",
			"verbose: lock released",
			"verbose: format resolved",
		}
		for _, phrase := range expectedPhrases {
			if !strings.Contains(stderrStr, phrase) {
				t.Errorf("expected stderr to contain %q, got:\n%s", phrase, stderrStr)
			}
		}
	})

	t.Run("it writes nothing to stderr when verbose off", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if strings.Contains(stderr.String(), "verbose:") {
			t.Errorf("expected no verbose output when off, got stderr: %q", stderr.String())
		}
	})

	t.Run("it does not write verbose to stdout", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--verbose", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if strings.Contains(stdout.String(), "verbose:") {
			t.Errorf("expected no verbose output on stdout, got: %q", stdout.String())
		}
	})

	t.Run("it allows quiet + verbose simultaneously", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--verbose", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Quiet suppresses formatted output on stdout (IDs only)
		stdoutStr := strings.TrimSpace(stdout.String())
		if stdoutStr != "tick-aaaaaa" {
			t.Errorf("expected quiet to output only ID, got %q", stdoutStr)
		}

		// Verbose still writes to stderr
		if !strings.Contains(stderr.String(), "verbose:") {
			t.Errorf("expected verbose output on stderr even with --quiet, got: %q", stderr.String())
		}
	})

	t.Run("it works with each format flag without contamination", func(t *testing.T) {
		formats := []struct {
			flag    string
			checker func(t *testing.T, stdout string)
		}{
			{"--toon", func(t *testing.T, stdout string) {
				if !strings.Contains(stdout, "tasks[") {
					t.Errorf("expected TOON format, got %q", stdout)
				}
			}},
			{"--pretty", func(t *testing.T, stdout string) {
				if !strings.Contains(stdout, "ID") {
					t.Errorf("expected Pretty format, got %q", stdout)
				}
			}},
			{"--json", func(t *testing.T, stdout string) {
				if !strings.HasPrefix(strings.TrimSpace(stdout), "[") {
					t.Errorf("expected JSON array, got %q", stdout)
				}
			}},
		}

		for _, f := range formats {
			t.Run(f.flag, func(t *testing.T) {
				tk := task.NewTask("tick-aaaaaa", "Task A")
				dir := initTickProjectWithTasks(t, []task.Task{tk})

				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", "--verbose", f.flag, "list"}, dir, &stdout, &stderr, false)

				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				// Verbose output goes to stderr, not stdout
				if strings.Contains(stdout.String(), "verbose:") {
					t.Errorf("verbose contaminated stdout with %s, got: %q", f.flag, stdout.String())
				}

				// Check the format is correct on stdout
				f.checker(t, stdout.String())

				// Verbose output should be on stderr
				if !strings.Contains(stderr.String(), "verbose:") {
					t.Errorf("expected verbose on stderr with %s, got: %q", f.flag, stderr.String())
				}
			})
		}
	})

	t.Run("it produces clean piped output with verbose enabled", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--verbose", "--toon", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// stdout should contain only formatted task output (TOON)
		stdoutStr := stdout.String()
		lines := strings.Split(strings.TrimSpace(stdoutStr), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "verbose:") {
				t.Errorf("verbose line leaked to stdout: %q", line)
			}
		}

		// Verify tasks are present in stdout
		if !strings.Contains(stdoutStr, "tick-aaaaaa") {
			t.Errorf("expected task tick-aaaaaa in stdout, got: %q", stdoutStr)
		}
		if !strings.Contains(stdoutStr, "tick-bbbbbb") {
			t.Errorf("expected task tick-bbbbbb in stdout, got: %q", stdoutStr)
		}
	})

	t.Run("it logs verbose during mutation commands", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--verbose", "create", "Verbose create"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stderrStr := stderr.String()
		// Mutation should log lock, freshness, atomic write
		if !strings.Contains(stderrStr, "verbose: lock acquired (exclusive)") {
			t.Errorf("expected exclusive lock log, got:\n%s", stderrStr)
		}
		if !strings.Contains(stderrStr, "verbose: atomic write") {
			t.Errorf("expected atomic write log, got:\n%s", stderrStr)
		}
	})
}
