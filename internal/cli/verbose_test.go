package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestVerbose_WritesToStderr(t *testing.T) {
	t.Run("it writes cache/lock/hash/format verbose to stderr", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--verbose", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stderrStr := stderr.String()
		// Should contain verbose lines about key operations
		if !strings.Contains(stderrStr, "verbose:") {
			t.Errorf("expected verbose output on stderr, got %q", stderrStr)
		}

		// Check for key operations
		hasLock := strings.Contains(stderrStr, "lock")
		hasHash := strings.Contains(stderrStr, "hash")
		hasCache := strings.Contains(stderrStr, "cache") || strings.Contains(stderrStr, "fresh")
		hasFormat := strings.Contains(stderrStr, "format")

		if !hasLock {
			t.Errorf("expected verbose to mention lock operations, got %q", stderrStr)
		}
		if !hasHash {
			t.Errorf("expected verbose to mention hash operations, got %q", stderrStr)
		}
		if !hasCache {
			t.Errorf("expected verbose to mention cache/freshness operations, got %q", stderrStr)
		}
		if !hasFormat {
			t.Errorf("expected verbose to mention format resolution, got %q", stderrStr)
		}
	})
}

func TestVerbose_WritesNothingWhenOff(t *testing.T) {
	t.Run("it writes nothing to stderr when verbose off", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stderr.String() != "" {
			t.Errorf("expected no stderr when verbose off, got %q", stderr.String())
		}
	})
}

func TestVerbose_DoesNotWriteToStdout(t *testing.T) {
	t.Run("it does not write verbose to stdout", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--verbose", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stdoutStr := stdout.String()
		if strings.Contains(stdoutStr, "verbose:") {
			t.Errorf("expected no verbose output on stdout, got %q", stdoutStr)
		}
	})
}

func TestVerbose_AllowsQuietPlusVerbose(t *testing.T) {
	t.Run("it allows quiet + verbose simultaneously", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "--verbose", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Quiet wins on stdout: only IDs, no format headers
		stdoutStr := strings.TrimSpace(stdout.String())
		if strings.Contains(stdoutStr, "tasks[") {
			t.Errorf("expected quiet to suppress format headers on stdout, got %q", stdoutStr)
		}
		// Verbose still on stderr
		stderrStr := stderr.String()
		if !strings.Contains(stderrStr, "verbose:") {
			t.Errorf("expected verbose output on stderr even with --quiet, got %q", stderrStr)
		}
	})
}

func TestVerbose_WorksWithEachFormatFlag(t *testing.T) {
	t.Run("it works with each format flag without contamination", func(t *testing.T) {
		formats := []string{"--toon", "--pretty", "--json"}

		for _, format := range formats {
			t.Run(format, func(t *testing.T) {
				now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
				tasks := []task.Task{
					{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
				}
				dir := setupInitializedDirWithTasks(t, tasks)
				var stdout, stderr bytes.Buffer

				app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
				code := app.Run([]string{"tick", "--verbose", format, "list"})
				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				// stdout should not contain any verbose: lines
				stdoutStr := stdout.String()
				if strings.Contains(stdoutStr, "verbose:") {
					t.Errorf("expected no verbose on stdout with %s, got %q", format, stdoutStr)
				}

				// stderr should contain verbose lines
				stderrStr := stderr.String()
				if !strings.Contains(stderrStr, "verbose:") {
					t.Errorf("expected verbose on stderr with %s, got %q", format, stderrStr)
				}
			})
		}
	})
}

func TestVerbose_CleanPipedOutput(t *testing.T) {
	t.Run("it produces clean piped output with verbose enabled", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Minute), Updated: now.Add(time.Minute)},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--verbose", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Count non-empty lines on stdout (simulates `tick list --verbose | wc -l`)
		// Should only have task data lines, no verbose lines
		stdoutLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		for _, line := range stdoutLines {
			if strings.HasPrefix(strings.TrimSpace(line), "verbose:") {
				t.Errorf("verbose line leaked to stdout: %q", line)
			}
		}

		// TOON format: 1 header + 2 data = 3 lines
		if len(stdoutLines) != 3 {
			t.Errorf("expected 3 stdout lines (header + 2 tasks), got %d: %q", len(stdoutLines), stdout.String())
		}
	})
}

func TestVerbose_AllLinesPrefixed(t *testing.T) {
	t.Run("it prefixes all verbose lines with verbose:", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--verbose", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		stderrStr := stderr.String()
		if stderrStr == "" {
			t.Fatal("expected verbose output on stderr, got nothing")
		}

		lines := strings.Split(strings.TrimSpace(stderrStr), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if !strings.HasPrefix(trimmed, "verbose:") {
				t.Errorf("expected all stderr lines to start with 'verbose:', got %q", line)
			}
		}
	})
}
