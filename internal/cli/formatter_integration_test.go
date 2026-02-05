package cli

import (
	"bytes"
	"strings"
	"testing"
)

// Test: "it formats create/update as full task detail in each format"
func TestFormatterIntegration_CreateUpdateFormatsTaskDetail(t *testing.T) {
	tests := []struct {
		name       string
		formatFlag string
		wantFormat string // "toon", "pretty", "json"
	}{
		{"create uses TOON format with --toon flag", "--toon", "toon"},
		{"create uses Pretty format with --pretty flag", "--pretty", "pretty"},
		{"create uses JSON format with --json flag", "--json", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTickDir(t)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
			exitCode := app.Run([]string{"tick", tt.formatFlag, "create", "Test task"})

			if exitCode != 0 {
				t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
			}

			output := stdout.String()
			switch tt.wantFormat {
			case "toon":
				if !strings.HasPrefix(output, "task{") {
					t.Errorf("expected TOON task detail format, got: %s", output)
				}
			case "pretty":
				if !strings.Contains(output, "ID:") || !strings.Contains(output, "Title:") {
					t.Errorf("expected Pretty task detail format, got: %s", output)
				}
			case "json":
				if !strings.HasPrefix(output, "{") || !strings.Contains(output, `"id"`) {
					t.Errorf("expected JSON task detail format, got: %s", output)
				}
			}
		})
	}

	// Test update separately
	t.Run("update uses Pretty format with --pretty flag", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Original title")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--pretty", "update", "tick-abc123", "--title", "Updated title"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID:") || !strings.Contains(output, "Updated title") {
			t.Errorf("expected Pretty task detail format with updated title, got: %s", output)
		}
	})
}

// Test: "it formats transitions in each format"
func TestFormatterIntegration_TransitionsFormatted(t *testing.T) {
	tests := []struct {
		name       string
		formatFlag string
		wantFormat string
	}{
		{"start uses TOON plain text with --toon", "--toon", "toon"},
		{"start uses Pretty plain text with --pretty", "--pretty", "pretty"},
		{"start uses JSON object with --json", "--json", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTickDir(t)
			setupTask(t, dir, "tick-abc123", "Test task")

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
			exitCode := app.Run([]string{"tick", tt.formatFlag, "start", "tick-abc123"})

			if exitCode != 0 {
				t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
			}

			output := stdout.String()
			switch tt.wantFormat {
			case "toon", "pretty":
				// Plain text: "tick-abc123: open -> in_progress"
				if !strings.Contains(output, "tick-abc123") || !strings.Contains(output, "\u2192") {
					t.Errorf("expected plain text transition format, got: %s", output)
				}
			case "json":
				// JSON object: {"id": "tick-abc123", "from": "open", "to": "in_progress"}
				if !strings.Contains(output, `"id"`) || !strings.Contains(output, `"from"`) || !strings.Contains(output, `"to"`) {
					t.Errorf("expected JSON transition format, got: %s", output)
				}
			}
		})
	}
}

// Test: "it formats dep confirmations in each format"
func TestFormatterIntegration_DepConfirmationsFormatted(t *testing.T) {
	tests := []struct {
		name       string
		formatFlag string
		wantFormat string
	}{
		{"dep add uses TOON plain text with --toon", "--toon", "toon"},
		{"dep add uses Pretty plain text with --pretty", "--pretty", "pretty"},
		{"dep add uses JSON object with --json", "--json", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTickDir(t)
			setupTask(t, dir, "tick-abc123", "Task A")
			setupTask(t, dir, "tick-def456", "Task B")

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
			exitCode := app.Run([]string{"tick", tt.formatFlag, "dep", "add", "tick-abc123", "tick-def456"})

			if exitCode != 0 {
				t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
			}

			output := stdout.String()
			switch tt.wantFormat {
			case "toon", "pretty":
				// Plain text: "Dependency added: tick-abc123 blocked by tick-def456"
				if !strings.Contains(output, "Dependency added") {
					t.Errorf("expected plain text dep confirmation, got: %s", output)
				}
			case "json":
				// JSON object with action, task_id, blocked_by
				if !strings.Contains(output, `"action"`) || !strings.Contains(output, `"task_id"`) {
					t.Errorf("expected JSON dep change format, got: %s", output)
				}
			}
		})
	}
}

// Test: "it formats list/show in each format"
func TestFormatterIntegration_ListShowFormatted(t *testing.T) {
	t.Run("list uses TOON format with --toon", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--toon", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.HasPrefix(output, "tasks[") {
			t.Errorf("expected TOON list format, got: %s", output)
		}
	})

	t.Run("list uses Pretty format with --pretty", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--pretty", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") || !strings.Contains(output, "TITLE") {
			t.Errorf("expected Pretty list format with headers, got: %s", output)
		}
	})

	t.Run("list uses JSON format with --json", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--json", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.HasPrefix(output, "[") || !strings.Contains(output, `"id"`) {
			t.Errorf("expected JSON array format, got: %s", output)
		}
	})

	t.Run("show uses TOON format with --toon", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--toon", "show", "tick-abc123"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.HasPrefix(output, "task{") {
			t.Errorf("expected TOON task detail format, got: %s", output)
		}
	})
}

// Test: "it formats init/rebuild in each format"
func TestFormatterIntegration_InitRebuildFormatted(t *testing.T) {
	t.Run("init uses TOON plain text with --toon", func(t *testing.T) {
		dir := t.TempDir() // Use fresh dir without .tick
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--toon", "init"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Initialized") {
			t.Errorf("expected init message, got: %s", output)
		}
	})

	t.Run("init uses JSON object with --json", func(t *testing.T) {
		dir := t.TempDir()
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--json", "init"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, `"message"`) {
			t.Errorf("expected JSON message format, got: %s", output)
		}
	})
}

// Test: "it applies --quiet override for each command type"
func TestFormatterIntegration_QuietOverride(t *testing.T) {
	t.Run("create with --quiet outputs ID only", func(t *testing.T) {
		dir := setupTickDir(t)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "create", "Test task"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		// Should be just the ID
		if !strings.HasPrefix(output, "tick-") || strings.Contains(output, "Title") {
			t.Errorf("expected only task ID, got: %s", output)
		}
	})

	t.Run("update with --quiet outputs ID only", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Original")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "update", "tick-abc123", "--title", "Updated"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-abc123" {
			t.Errorf("expected only task ID 'tick-abc123', got: %s", output)
		}
	})

	t.Run("start with --quiet outputs nothing", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "start", "tick-abc123"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected empty output, got: %s", stdout.String())
		}
	})

	t.Run("dep add with --quiet outputs nothing", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Task A")
		setupTask(t, dir, "tick-def456", "Task B")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "dep", "add", "tick-abc123", "tick-def456"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected empty output, got: %s", stdout.String())
		}
	})

	t.Run("list with --quiet outputs IDs only", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Task A")
		setupTask(t, dir, "tick-def456", "Task B")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "tick-") {
				t.Errorf("expected only task IDs, got line: %s", line)
			}
		}
	})

	t.Run("init with --quiet outputs nothing", func(t *testing.T) {
		dir := t.TempDir()
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "init"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected empty output, got: %s", stdout.String())
		}
	})

	t.Run("show with --quiet outputs ID only", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "show", "tick-abc123"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-abc123" {
			t.Errorf("expected only task ID, got: %s", output)
		}
	})
}

// Test: "it handles empty list per format"
func TestFormatterIntegration_EmptyListHandling(t *testing.T) {
	t.Run("empty list with --toon shows zero count", func(t *testing.T) {
		dir := setupTickDir(t)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--toon", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tasks[0]") {
			t.Errorf("expected TOON zero count format, got: %s", output)
		}
	})

	t.Run("empty list with --pretty shows message", func(t *testing.T) {
		dir := setupTickDir(t)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--pretty", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "No tasks found") {
			t.Errorf("expected 'No tasks found' message, got: %s", output)
		}
	})

	t.Run("empty list with --json shows empty array", func(t *testing.T) {
		dir := setupTickDir(t)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--json", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "[]" {
			t.Errorf("expected empty JSON array '[]', got: %s", output)
		}
	})
}

// Test: "it defaults to TOON when piped, Pretty when TTY"
func TestFormatterIntegration_TTYAutoDetection(t *testing.T) {
	// Note: In tests, stdout is a bytes.Buffer which is not a TTY,
	// so we expect TOON format by default
	t.Run("non-TTY stdout defaults to TOON", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{} // Not a TTY
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		// With non-TTY (bytes.Buffer), should default to TOON
		if !strings.HasPrefix(output, "tasks[") {
			t.Errorf("expected TOON format for non-TTY, got: %s", output)
		}
	})
}

// Test: "it respects --toon/--pretty/--json overrides"
func TestFormatterIntegration_FormatOverrides(t *testing.T) {
	t.Run("--toon forces TOON format", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--toon", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		if !strings.HasPrefix(stdout.String(), "tasks[") {
			t.Errorf("--toon should force TOON format, got: %s", stdout.String())
		}
	})

	t.Run("--pretty forces Pretty format", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--pretty", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") {
			t.Errorf("--pretty should force Pretty format with headers, got: %s", output)
		}
	})

	t.Run("--json forces JSON format", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--json", "list"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		if !strings.HasPrefix(stdout.String(), "[") {
			t.Errorf("--json should force JSON format, got: %s", stdout.String())
		}
	})
}

// Edge case: --quiet + --json: quiet wins, no JSON wrapping
func TestFormatterIntegration_QuietOverridesJson(t *testing.T) {
	t.Run("--quiet + --json on create outputs ID only, not JSON", func(t *testing.T) {
		dir := setupTickDir(t)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "--json", "create", "Test task"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		// Should be plain ID, not JSON wrapped
		if strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[") {
			t.Errorf("--quiet should suppress JSON formatting, got: %s", output)
		}
		if !strings.HasPrefix(output, "tick-") {
			t.Errorf("expected plain task ID, got: %s", output)
		}
	})

	t.Run("--quiet + --json on transition outputs nothing", func(t *testing.T) {
		dir := setupTickDir(t)
		setupTask(t, dir, "tick-abc123", "Test task")

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--quiet", "--json", "start", "tick-abc123"})

		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("--quiet should suppress all output for transitions, got: %s", stdout.String())
		}
	})
}

// Edge case: Errors always plain text to stderr regardless of format
func TestFormatterIntegration_ErrorsAlwaysPlainText(t *testing.T) {
	t.Run("error with --json is still plain text to stderr", func(t *testing.T) {
		dir := setupTickDir(t)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		app := &App{Stdout: stdout, Stderr: stderr, Cwd: dir}
		exitCode := app.Run([]string{"tick", "--json", "show", "tick-nonexistent"})

		if exitCode != 1 {
			t.Fatalf("expected exit 1 for not found, got %d", exitCode)
		}

		errOutput := stderr.String()
		// Error should be plain text, not JSON
		if strings.HasPrefix(errOutput, "{") || strings.HasPrefix(errOutput, "[") {
			t.Errorf("errors should be plain text, not JSON, got: %s", errOutput)
		}
		if !strings.Contains(errOutput, "Error:") {
			t.Errorf("error should contain 'Error:', got: %s", errOutput)
		}
	})
}
