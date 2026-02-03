package cli

import (
	"strings"
	"testing"
)

func TestVerboseLogger(t *testing.T) {
	t.Run("it writes verbose-prefixed messages when enabled", func(t *testing.T) {
		var buf strings.Builder
		logger := NewVerboseLogger(&buf, true)

		logger.Log("cache rebuild triggered")

		got := buf.String()
		if !strings.Contains(got, "verbose:") {
			t.Errorf("output = %q, want it to contain 'verbose:' prefix", got)
		}
		if !strings.Contains(got, "cache rebuild triggered") {
			t.Errorf("output = %q, want it to contain message", got)
		}
	})

	t.Run("it writes nothing when disabled", func(t *testing.T) {
		var buf strings.Builder
		logger := NewVerboseLogger(&buf, false)

		logger.Log("should not appear")

		if buf.String() != "" {
			t.Errorf("output = %q, want empty when verbose disabled", buf.String())
		}
	})

	t.Run("it prefixes every line with verbose:", func(t *testing.T) {
		var buf strings.Builder
		logger := NewVerboseLogger(&buf, true)

		logger.Log("first message")
		logger.Log("second message")

		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		for i, line := range lines {
			if !strings.HasPrefix(line, "verbose: ") {
				t.Errorf("line %d = %q, want 'verbose: ' prefix", i, line)
			}
		}
	})
}

func TestVerboseOutput(t *testing.T) {
	t.Run("it writes cache/lock/hash/format verbose to stderr", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "--verbose", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		stderrOutput := stderr.String()

		// Should contain verbose messages about key operations
		if !strings.Contains(stderrOutput, "verbose:") {
			t.Errorf("stderr should contain verbose: messages, got:\n%s", stderrOutput)
		}

		// Check for key operation categories
		expectations := []string{"format", "lock", "freshness", "cache"}
		for _, exp := range expectations {
			if !strings.Contains(strings.ToLower(stderrOutput), exp) {
				t.Errorf("stderr should contain '%s' verbose message, got:\n%s", exp, stderrOutput)
			}
		}
	})

	t.Run("it writes nothing to stderr when verbose off", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		if strings.Contains(stderr.String(), "verbose:") {
			t.Errorf("stderr should not contain verbose: messages when verbose off, got:\n%s", stderr.String())
		}
	})

	t.Run("it does not write verbose to stdout", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "--verbose", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		if strings.Contains(stdout.String(), "verbose:") {
			t.Errorf("stdout should not contain verbose: messages, got:\n%s", stdout.String())
		}
	})

	t.Run("it allows quiet + verbose simultaneously", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "--quiet", "--verbose", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		// Quiet: stdout should only have IDs (no headers, no format)
		stdoutLines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
		for _, line := range stdoutLines {
			if strings.Contains(line, "verbose:") {
				t.Errorf("stdout should not contain verbose: with --quiet, got line: %q", line)
			}
		}

		// Verbose: stderr should still have verbose messages
		if !strings.Contains(stderr.String(), "verbose:") {
			t.Errorf("stderr should contain verbose: messages even with --quiet, got:\n%s", stderr.String())
		}
	})

	t.Run("it works with each format flag without contamination", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		formats := []struct {
			name string
			flag string
		}{
			{"toon", "--toon"},
			{"pretty", "--pretty"},
			{"json", "--json"},
		}

		for _, f := range formats {
			t.Run(f.name, func(t *testing.T) {
				dir := setupTickDirWithContent(t, content)

				app := NewApp()
				app.workDir = dir
				var stdout, stderr strings.Builder
				app.stdout = &stdout
				app.stderr = &stderr

				err := app.Run([]string{"tick", "--verbose", f.flag, "list"})
				if err != nil {
					t.Fatalf("list returned error: %v", err)
				}

				// stdout should not contain verbose
				if strings.Contains(stdout.String(), "verbose:") {
					t.Errorf("[%s] stdout should not contain verbose:, got:\n%s", f.name, stdout.String())
				}

				// stderr should contain verbose
				if !strings.Contains(stderr.String(), "verbose:") {
					t.Errorf("[%s] stderr should contain verbose:, got:\n%s", f.name, stderr.String())
				}
			})
		}
	})

	t.Run("it produces clean piped output with verbose enabled", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task one","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Task two","status":"open","priority":2,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "--verbose", "--toon", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		// stdout should only contain TOON format output, no verbose lines
		stdoutLines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
		for _, line := range stdoutLines {
			if strings.HasPrefix(line, "verbose:") {
				t.Errorf("stdout contains verbose line: %q", line)
			}
		}

		// Verify it actually has task data
		if !strings.Contains(stdout.String(), "tick-aaa111") {
			t.Errorf("stdout should contain task data, got:\n%s", stdout.String())
		}

		// stderr should have the verbose messages
		if !strings.Contains(stderr.String(), "verbose:") {
			t.Errorf("stderr should contain verbose messages, got:\n%s", stderr.String())
		}
	})

	t.Run("it logs verbose for mutations too", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout, stderr strings.Builder
		app.stdout = &stdout
		app.stderr = &stderr

		err := app.Run([]string{"tick", "--verbose", "create", "Test task"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		stderrOutput := stderr.String()
		if !strings.Contains(stderrOutput, "verbose:") {
			t.Errorf("stderr should contain verbose: messages for mutations, got:\n%s", stderrOutput)
		}

		// Should mention lock and write operations
		if !strings.Contains(strings.ToLower(stderrOutput), "lock") {
			t.Errorf("stderr should mention lock operations, got:\n%s", stderrOutput)
		}
		if !strings.Contains(strings.ToLower(stderrOutput), "write") {
			t.Errorf("stderr should mention write operations, got:\n%s", stderrOutput)
		}
	})
}
