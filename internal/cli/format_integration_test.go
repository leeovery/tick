package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestFormatIntegration_CreateFormatsAsTaskDetail(t *testing.T) {
	t.Run("it formats create as full task detail in TOON (default non-TTY)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "create", "My task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// TOON task detail starts with "task{"
		if !strings.Contains(output, "task{") {
			t.Errorf("expected TOON task detail format, got %q", output)
		}
		if !strings.Contains(output, "My task") {
			t.Errorf("expected title in output, got %q", output)
		}
	})

	t.Run("it formats create as task detail in Pretty with --pretty", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--pretty", "create", "My task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Pretty format uses "ID:" label
		if !strings.Contains(output, "ID:") {
			t.Errorf("expected Pretty task detail format with 'ID:', got %q", output)
		}
		if !strings.Contains(output, "Title:") {
			t.Errorf("expected 'Title:' label, got %q", output)
		}
	})

	t.Run("it formats create as task detail in JSON with --json", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "create", "My task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("expected valid JSON output, got error: %v; output: %q", err, output)
		}
		if _, ok := parsed["id"]; !ok {
			t.Errorf("expected 'id' key in JSON, got %v", parsed)
		}
		if parsed["title"] != "My task" {
			t.Errorf("expected title 'My task', got %v", parsed["title"])
		}
	})
}

func TestFormatIntegration_UpdateFormatsAsTaskDetail(t *testing.T) {
	t.Run("it formats update as full task detail in each format", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Original", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		tests := []struct {
			name       string
			flag       string
			assertToon bool
			assertJSON bool
		}{
			{"toon", "--toon", true, false},
			{"json", "--json", false, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupInitializedDirWithTasks(t, existing)
				var stdout, stderr bytes.Buffer

				app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
				code := app.Run([]string{"tick", tt.flag, "update", "tick-aaa111", "--title", "Updated"})
				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				output := stdout.String()
				if tt.assertToon {
					if !strings.Contains(output, "task{") {
						t.Errorf("expected TOON format, got %q", output)
					}
				}
				if tt.assertJSON {
					var parsed map[string]interface{}
					if err := json.Unmarshal([]byte(output), &parsed); err != nil {
						t.Fatalf("expected valid JSON, got error: %v; output: %q", err, output)
					}
				}
			})
		}
	})
}

func TestFormatIntegration_TransitionsInEachFormat(t *testing.T) {
	t.Run("it formats transitions in TOON/Pretty as plain text", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		expected := "tick-aaa111: open \u2192 in_progress"
		if output != expected {
			t.Errorf("expected %q, got %q", expected, output)
		}
	})

	t.Run("it formats transitions in JSON as structured object", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &parsed); err != nil {
			t.Fatalf("expected valid JSON, got error: %v; output: %q", err, stdout.String())
		}
		if parsed["id"] != "tick-aaa111" {
			t.Errorf("expected id 'tick-aaa111', got %v", parsed["id"])
		}
		if parsed["from"] != "open" {
			t.Errorf("expected from 'open', got %v", parsed["from"])
		}
		if parsed["to"] != "in_progress" {
			t.Errorf("expected to 'in_progress', got %v", parsed["to"])
		}
	})
}

func TestFormatIntegration_DepConfirmationsInEachFormat(t *testing.T) {
	t.Run("it formats dep add in TOON as plain text", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		expected := "Dependency added: tick-aaa111 blocked by tick-bbb222"
		if output != expected {
			t.Errorf("expected %q, got %q", expected, output)
		}
	})

	t.Run("it formats dep add in JSON as structured object", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &parsed); err != nil {
			t.Fatalf("expected valid JSON, got error: %v; output: %q", err, stdout.String())
		}
		if parsed["action"] != "added" {
			t.Errorf("expected action 'added', got %v", parsed["action"])
		}
	})

	t.Run("it formats dep rm in JSON as structured object", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &parsed); err != nil {
			t.Fatalf("expected valid JSON, got error: %v; output: %q", err, stdout.String())
		}
		if parsed["action"] != "removed" {
			t.Errorf("expected action 'removed', got %v", parsed["action"])
		}
	})
}

func TestFormatIntegration_ListShowInEachFormat(t *testing.T) {
	t.Run("it formats list in TOON (default non-TTY)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// TOON list format starts with "tasks["
		if !strings.Contains(output, "tasks[") {
			t.Errorf("expected TOON list format, got %q", output)
		}
	})

	t.Run("it formats list in Pretty with --pretty", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--pretty", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Pretty list has header row with "ID"
		if !strings.Contains(output, "ID") {
			t.Errorf("expected Pretty list header with 'ID', got %q", output)
		}
		if !strings.Contains(output, "STATUS") {
			t.Errorf("expected Pretty list header with 'STATUS', got %q", output)
		}
	})

	t.Run("it formats list in JSON with --json", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &parsed); err != nil {
			t.Fatalf("expected valid JSON array, got error: %v; output: %q", err, stdout.String())
		}
		if len(parsed) != 1 {
			t.Fatalf("expected 1 item, got %d", len(parsed))
		}
		if parsed[0]["id"] != "tick-aaa111" {
			t.Errorf("expected id 'tick-aaa111', got %v", parsed[0]["id"])
		}
	})

	t.Run("it formats show in TOON (default non-TTY)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "task{") {
			t.Errorf("expected TOON task detail format, got %q", output)
		}
	})

	t.Run("it formats show in JSON with --json", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &parsed); err != nil {
			t.Fatalf("expected valid JSON, got error: %v; output: %q", err, stdout.String())
		}
		if parsed["id"] != "tick-aaa111" {
			t.Errorf("expected id 'tick-aaa111', got %v", parsed["id"])
		}
	})
}

func TestFormatIntegration_InitRebuildInEachFormat(t *testing.T) {
	t.Run("it formats init as message in TOON", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		absDir, _ := filepath.Abs(dir)
		expected := "Initialized tick in " + absDir + "/.tick/"
		output := strings.TrimSpace(stdout.String())
		if output != expected {
			t.Errorf("expected %q, got %q", expected, output)
		}
	})

	t.Run("it formats init as message in JSON", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &parsed); err != nil {
			t.Fatalf("expected valid JSON, got error: %v; output: %q", err, stdout.String())
		}
		if _, ok := parsed["message"]; !ok {
			t.Errorf("expected 'message' key in JSON, got %v", parsed)
		}
	})
}

func TestFormatIntegration_QuietOverridePerCommandType(t *testing.T) {
	t.Run("it outputs only ID for create with --quiet", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "create", "My task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		// Should be just a task ID
		if !strings.HasPrefix(output, "tick-") {
			t.Errorf("expected task ID only, got %q", output)
		}
		// Should not contain format labels
		if strings.Contains(output, "task{") || strings.Contains(output, "Title:") {
			t.Errorf("expected quiet to suppress detail, got %q", output)
		}
	})

	t.Run("it outputs nothing for transition with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet for transition, got %q", stdout.String())
		}
	})

	t.Run("it outputs nothing for dep with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet for dep, got %q", stdout.String())
		}
	})

	t.Run("it outputs nothing for init with --quiet", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "init"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet for init, got %q", stdout.String())
		}
	})

	t.Run("it outputs IDs only for list with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("expected only task ID, got %q", output)
		}
	})

	t.Run("it outputs only ID for show with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, tasks)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "show", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("expected only task ID, got %q", output)
		}
	})

	t.Run("it outputs only ID for update with --quiet", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "update", "tick-aaa111", "--title", "Updated"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("expected only task ID, got %q", output)
		}
	})
}

func TestFormatIntegration_QuietPlusJsonQuietWins(t *testing.T) {
	t.Run("it outputs nothing with --quiet + --json for transition (quiet wins)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "--json", "start", "tick-aaa111"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet + --json, got %q", stdout.String())
		}
	})

	t.Run("it outputs only ID with --quiet + --json for create (quiet wins, no JSON wrapping)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "--json", "create", "My task"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		// Should be plain ID, not JSON-wrapped
		if strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[") {
			t.Errorf("expected plain ID (no JSON wrapping), got %q", output)
		}
		if !strings.HasPrefix(output, "tick-") {
			t.Errorf("expected task ID, got %q", output)
		}
	})
}

func TestFormatIntegration_EmptyListPerFormat(t *testing.T) {
	t.Run("it outputs TOON zero-count for empty list (default non-TTY)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tasks[0]{id,title,status,priority}:" {
			t.Errorf("expected TOON empty list format, got %q", output)
		}
	})

	t.Run("it outputs Pretty message for empty list", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--pretty", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected 'No tasks found.', got %q", output)
		}
	})

	t.Run("it outputs JSON empty array for empty list", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--json", "list"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "[]" {
			t.Errorf("expected JSON empty array '[]', got %q", output)
		}
	})
}

func TestFormatIntegration_DefaultsToonWhenPiped(t *testing.T) {
	t.Run("it defaults to TOON when stdout is bytes.Buffer (non-TTY)", func(t *testing.T) {
		var stdout bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &bytes.Buffer{},
			Dir:    t.TempDir(),
		}

		app.Run([]string{"tick", "init"})

		if app.OutputFormat != FormatToon {
			t.Errorf("expected FormatToon for non-TTY, got %v", app.OutputFormat)
		}
	})
}

func TestFormatIntegration_FlagOverrides(t *testing.T) {
	t.Run("it respects --toon override", func(t *testing.T) {
		var stdout bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &bytes.Buffer{},
			Dir:    t.TempDir(),
		}

		app.Run([]string{"tick", "--toon", "init"})

		if app.OutputFormat != FormatToon {
			t.Errorf("expected FormatToon, got %v", app.OutputFormat)
		}
	})

	t.Run("it respects --pretty override", func(t *testing.T) {
		var stdout bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &bytes.Buffer{},
			Dir:    t.TempDir(),
		}

		app.Run([]string{"tick", "--pretty", "init"})

		if app.OutputFormat != FormatPretty {
			t.Errorf("expected FormatPretty, got %v", app.OutputFormat)
		}
	})

	t.Run("it respects --json override", func(t *testing.T) {
		var stdout bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &bytes.Buffer{},
			Dir:    t.TempDir(),
		}

		app.Run([]string{"tick", "--json", "init"})

		if app.OutputFormat != FormatJSON {
			t.Errorf("expected FormatJSON, got %v", app.OutputFormat)
		}
	})
}

func TestFormatIntegration_ErrorsRemainPlainTextStderr(t *testing.T) {
	t.Run("it writes errors to stderr as plain text regardless of format", func(t *testing.T) {
		formats := []string{"--toon", "--pretty", "--json"}

		for _, flag := range formats {
			t.Run(flag, func(t *testing.T) {
				dir := setupInitializedDir(t)
				var stdout, stderr bytes.Buffer

				app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
				code := app.Run([]string{"tick", flag, "show", "tick-nonexist"})
				if code != 1 {
					t.Fatalf("expected exit code 1, got %d", code)
				}

				errMsg := stderr.String()
				if !strings.HasPrefix(errMsg, "Error: ") {
					t.Errorf("expected plain text error starting with 'Error: ', got %q", errMsg)
				}
				// Should NOT be JSON
				if strings.HasPrefix(errMsg, "{") {
					t.Errorf("expected plain text error, got JSON: %q", errMsg)
				}
				if stdout.String() != "" {
					t.Errorf("expected no stdout on error, got %q", stdout.String())
				}
			})
		}
	})
}

func TestFormatIntegration_FormatterResolvedOnceInDispatcher(t *testing.T) {
	t.Run("it stores formatter on App after Run resolves format", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		app.Run([]string{"tick", "init"})

		if app.Formatter == nil {
			t.Error("expected Formatter to be set after Run()")
		}
	})

	t.Run("it uses ToonFormatter for non-TTY", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		app.Run([]string{"tick", "init"})

		if _, ok := app.Formatter.(*ToonFormatter); !ok {
			t.Errorf("expected ToonFormatter for non-TTY, got %T", app.Formatter)
		}
	})

	t.Run("it uses PrettyFormatter with --pretty", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		app.Run([]string{"tick", "--pretty", "init"})

		if _, ok := app.Formatter.(*PrettyFormatter); !ok {
			t.Errorf("expected PrettyFormatter with --pretty, got %T", app.Formatter)
		}
	})

	t.Run("it uses JSONFormatter with --json", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		app.Run([]string{"tick", "--json", "init"})

		if _, ok := app.Formatter.(*JSONFormatter); !ok {
			t.Errorf("expected JSONFormatter with --json, got %T", app.Formatter)
		}
	})
}
