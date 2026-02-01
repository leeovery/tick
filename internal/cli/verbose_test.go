package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVerboseLogger(t *testing.T) {
	t.Run("writes to writer when enabled", func(t *testing.T) {
		var buf bytes.Buffer
		v := NewVerboseLogger(&buf, true)
		v.Log("test message %s", "hello")
		if !strings.Contains(buf.String(), "verbose: test message hello") {
			t.Errorf("expected verbose output, got %q", buf.String())
		}
	})

	t.Run("writes nothing when disabled", func(t *testing.T) {
		var buf bytes.Buffer
		v := NewVerboseLogger(&buf, false)
		v.Log("should not appear")
		if buf.Len() != 0 {
			t.Errorf("expected no output, got %q", buf.String())
		}
	})

	t.Run("prefixes all lines with verbose:", func(t *testing.T) {
		var buf bytes.Buffer
		v := NewVerboseLogger(&buf, true)
		v.Log("first")
		v.Log("second")
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if !strings.HasPrefix(line, "verbose: ") {
				t.Errorf("line should start with 'verbose: ', got %q", line)
			}
		}
	})
}

func TestVerboseIntegration(t *testing.T) {
	t.Run("writes verbose to stderr not stdout", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		app.Run([]string{"tick", "--verbose", "list"}, dir)

		if strings.Contains(outBuf.String(), "verbose:") {
			t.Errorf("verbose should not appear on stdout, got %q", outBuf.String())
		}
		if !strings.Contains(errBuf.String(), "verbose:") {
			t.Errorf("expected verbose output on stderr, got %q", errBuf.String())
		}
	})

	t.Run("writes nothing to stderr when verbose off", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		app.Run([]string{"tick", "list"}, dir)

		if strings.Contains(errBuf.String(), "verbose:") {
			t.Errorf("should not have verbose output, got %q", errBuf.String())
		}
	})

	t.Run("allows quiet + verbose simultaneously", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		code := app.Run([]string{"tick", "--quiet", "--verbose", "list"}, dir)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		// Verbose still goes to stderr
		if !strings.Contains(errBuf.String(), "verbose:") {
			t.Errorf("verbose should still write to stderr with --quiet, got %q", errBuf.String())
		}

		// Quiet: only IDs on stdout
		lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "tick-") {
				t.Errorf("--quiet should output only IDs, got %q", line)
			}
		}
	})

	t.Run("verbose with --json keeps stdout clean", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		app.Run([]string{"tick", "--verbose", "--json", "list"}, dir)

		stdout := outBuf.String()
		if strings.Contains(stdout, "verbose:") {
			t.Errorf("verbose should not contaminate JSON stdout")
		}
		if !strings.Contains(stdout, "[") {
			t.Errorf("expected JSON array output, got %q", stdout)
		}

		if !strings.Contains(errBuf.String(), "verbose:") {
			t.Errorf("verbose should write to stderr")
		}
	})

	t.Run("logs format resolution", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		app.Run([]string{"tick", "--verbose", "list"}, dir)

		stderr := errBuf.String()
		if !strings.Contains(stderr, "format=") {
			t.Errorf("expected format resolution in verbose, got %q", stderr)
		}
	})

	t.Run("logs lock and cache operations", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		app.Run([]string{"tick", "--verbose", "list"}, dir)

		stderr := errBuf.String()
		if !strings.Contains(stderr, "lock") {
			t.Errorf("expected lock info in verbose, got %q", stderr)
		}
		if !strings.Contains(stderr, "cache") {
			t.Errorf("expected cache info in verbose, got %q", stderr)
		}
	})

	t.Run("piped output is clean with verbose", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Test task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		app.Run([]string{"tick", "--verbose", "list"}, dir)

		for _, line := range strings.Split(outBuf.String(), "\n") {
			if strings.HasPrefix(line, "verbose:") {
				t.Errorf("verbose leaked to stdout: %q", line)
			}
		}
	})
}
