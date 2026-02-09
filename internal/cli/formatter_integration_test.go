package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestFormatterIntegration(t *testing.T) {
	// --- Create/Update formatted as task detail ---

	t.Run("it formats create as full task detail in toon format", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "create", "TOON task"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// TOON task detail starts with "task{" schema header
		if !strings.HasPrefix(output, "task{") {
			t.Errorf("expected TOON task detail starting with 'task{', got %q", output)
		}
		if !strings.Contains(output, "TOON task") {
			t.Errorf("expected title in output, got %q", output)
		}
	})

	t.Run("it formats create as full task detail in pretty format", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "create", "Pretty task"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		// Pretty task detail has "ID:" label
		if !strings.Contains(output, "ID:") {
			t.Errorf("expected Pretty 'ID:' label, got %q", output)
		}
		if !strings.Contains(output, "Pretty task") {
			t.Errorf("expected title in output, got %q", output)
		}
	})

	t.Run("it formats create as full task detail in json format", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "create", "JSON task"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(output), &obj); err != nil {
			t.Fatalf("expected valid JSON, got %q: %v", output, err)
		}
		if obj["title"] != "JSON task" {
			t.Errorf("expected title 'JSON task', got %v", obj["title"])
		}
		if _, ok := obj["id"]; !ok {
			t.Error("expected 'id' field in JSON output")
		}
	})

	t.Run("it formats update as full task detail in each format", func(t *testing.T) {
		formats := []struct {
			flag    string
			checker func(t *testing.T, output string)
		}{
			{"--toon", func(t *testing.T, output string) {
				if !strings.HasPrefix(output, "task{") {
					t.Errorf("expected TOON format, got %q", output)
				}
			}},
			{"--pretty", func(t *testing.T, output string) {
				if !strings.Contains(output, "ID:") {
					t.Errorf("expected Pretty format, got %q", output)
				}
			}},
			{"--json", func(t *testing.T, output string) {
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(output), &obj); err != nil {
					t.Errorf("expected valid JSON, got %q", output)
				}
			}},
		}

		for _, f := range formats {
			t.Run(f.flag, func(t *testing.T) {
				tk := task.NewTask("tick-aaaaaa", "Update me")
				dir := initTickProjectWithTasks(t, []task.Task{tk})

				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", f.flag, "update", "tick-aaaaaa", "--title", "Updated"}, dir, &stdout, &stderr, false)

				if code != 0 {
					t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
				}

				f.checker(t, stdout.String())
			})
		}
	})

	// --- Transitions ---

	t.Run("it formats transitions in toon format (plain text)", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Transition toon")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaaaaa") {
			t.Errorf("expected task ID in output, got %q", output)
		}
		if !strings.Contains(output, "\u2192") {
			t.Errorf("expected arrow in output, got %q", output)
		}
	})

	t.Run("it formats transitions in json format (structured)", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Transition json")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &obj); err != nil {
			t.Fatalf("expected valid JSON for transition, got %q: %v", stdout.String(), err)
		}
		if obj["id"] != "tick-aaaaaa" {
			t.Errorf("expected id 'tick-aaaaaa', got %v", obj["id"])
		}
		if obj["from"] != "open" {
			t.Errorf("expected from 'open', got %v", obj["from"])
		}
		if obj["to"] != "in_progress" {
			t.Errorf("expected to 'in_progress', got %v", obj["to"])
		}
	})

	// --- Dep confirmations ---

	t.Run("it formats dep add confirmation in toon format", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if !strings.Contains(stdout.String(), "Dependency added") {
			t.Errorf("expected dep confirmation, got %q", stdout.String())
		}
	})

	t.Run("it formats dep add confirmation in json format", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &obj); err != nil {
			t.Fatalf("expected valid JSON for dep change, got %q: %v", stdout.String(), err)
		}
		if obj["action"] != "added" {
			t.Errorf("expected action 'added', got %v", obj["action"])
		}
	})

	t.Run("it formats dep rm confirmation in json format", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t1.BlockedBy = []string{"tick-bbbbbb"}
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "dep", "rm", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &obj); err != nil {
			t.Fatalf("expected valid JSON for dep rm, got %q: %v", stdout.String(), err)
		}
		if obj["action"] != "removed" {
			t.Errorf("expected action 'removed', got %v", obj["action"])
		}
	})

	// --- List/Show ---

	t.Run("it formats list in toon format", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tasks[") {
			t.Errorf("expected TOON list format with 'tasks[', got %q", output)
		}
	})

	t.Run("it formats list in pretty format", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") {
			t.Errorf("expected Pretty list with headers, got %q", output)
		}
	})

	t.Run("it formats list in json format", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var arr []interface{}
		if err := json.Unmarshal(stdout.Bytes(), &arr); err != nil {
			t.Fatalf("expected valid JSON array for list, got %q: %v", stdout.String(), err)
		}
		if len(arr) != 1 {
			t.Errorf("expected 1 item in list, got %d", len(arr))
		}
	})

	t.Run("it formats show in toon format", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Show toon")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.HasPrefix(output, "task{") {
			t.Errorf("expected TOON task detail, got %q", output)
		}
	})

	t.Run("it formats show in json format", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Show json")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &obj); err != nil {
			t.Fatalf("expected valid JSON for show, got %q: %v", stdout.String(), err)
		}
		if obj["title"] != "Show json" {
			t.Errorf("expected title 'Show json', got %v", obj["title"])
		}
	})

	// --- Init/Rebuild ---

	t.Run("it formats init message in toon format", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Initialized tick in") {
			t.Errorf("expected init message, got %q", output)
		}
	})

	t.Run("it formats init message in json format", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &obj); err != nil {
			t.Fatalf("expected valid JSON for init message, got %q: %v", stdout.String(), err)
		}
		if _, ok := obj["message"]; !ok {
			t.Error("expected 'message' key in JSON output")
		}
	})

	// --- Quiet overrides ---

	t.Run("it applies --quiet override for create (ID only)", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--json", "create", "Quiet create"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		// --quiet should output just the ID, no JSON wrapping
		if strings.HasPrefix(output, "{") {
			t.Errorf("--quiet should suppress JSON wrapping, got %q", output)
		}
		if !strings.HasPrefix(output, "tick-") {
			t.Errorf("expected task ID, got %q", output)
		}
	})

	t.Run("it applies --quiet override for update (ID only)", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Quiet update")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--json", "update", "tick-aaaaaa", "--title", "Changed"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaaaaa" {
			t.Errorf("expected only task ID, got %q", output)
		}
	})

	t.Run("it applies --quiet override for show (ID only)", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Quiet show")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--json", "show", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaaaaa" {
			t.Errorf("expected only task ID, got %q", output)
		}
	})

	t.Run("it applies --quiet override for transitions (nothing)", func(t *testing.T) {
		tk := task.NewTask("tick-aaaaaa", "Quiet transition")
		dir := initTickProjectWithTasks(t, []task.Task{tk})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--json", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet for transitions, got %q", stdout.String())
		}
	})

	t.Run("it applies --quiet override for dep (nothing)", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--json", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet for dep, got %q", stdout.String())
		}
	})

	t.Run("it applies --quiet override for list (IDs only)", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Task A")
		t2 := task.NewTask("tick-bbbbbb", "Task B")
		dir := initTickProjectWithTasks(t, []task.Task{t1, t2})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--json", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 ID lines, got %d: %q", len(lines), output)
		}
		// Should be plain IDs, not JSON
		if strings.HasPrefix(output, "[") {
			t.Errorf("--quiet should suppress JSON wrapping, got %q", output)
		}
	})

	t.Run("it applies --quiet override for init (nothing)", func(t *testing.T) {
		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--quiet", "--json", "init"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no output with --quiet for init, got %q", stdout.String())
		}
	})

	// --- Empty list per format ---

	t.Run("it handles empty list in toon format (zero count)", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--toon", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tasks[0]") {
			t.Errorf("expected TOON zero-count 'tasks[0]', got %q", output)
		}
	})

	t.Run("it handles empty list in pretty format (message)", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--pretty", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "No tasks found." {
			t.Errorf("expected 'No tasks found.', got %q", output)
		}
	})

	t.Run("it handles empty list in json format (empty array)", func(t *testing.T) {
		dir := initTickProject(t)

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		if output != "[]" {
			t.Errorf("expected '[]', got %q", output)
		}
	})

	// --- TTY auto-detection ---

	t.Run("it defaults to toon when piped (non-TTY)", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "TTY test")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		// isTTY=false -> should default to TOON
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tasks[") {
			t.Errorf("expected TOON format when non-TTY, got %q", output)
		}
	})

	t.Run("it defaults to pretty when TTY", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "TTY test")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		// isTTY=true -> should default to Pretty
		code := Run([]string{"tick", "list"}, dir, &stdout, &stderr, true)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") {
			t.Errorf("expected Pretty format when TTY, got %q", output)
		}
	})

	// --- Flag overrides ---

	t.Run("it respects --toon override when TTY", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Override test")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		// isTTY=true but --toon should force TOON
		code := Run([]string{"tick", "--toon", "list"}, dir, &stdout, &stderr, true)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "tasks[") {
			t.Errorf("expected TOON format with --toon override, got %q", output)
		}
	})

	t.Run("it respects --pretty override when non-TTY", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Override test")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		// isTTY=false but --pretty should force Pretty
		code := Run([]string{"tick", "--pretty", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") {
			t.Errorf("expected Pretty format with --pretty override, got %q", output)
		}
	})

	t.Run("it respects --json override", func(t *testing.T) {
		t1 := task.NewTask("tick-aaaaaa", "Override test")
		dir := initTickProjectWithTasks(t, []task.Task{t1})

		var stdout, stderr bytes.Buffer
		code := Run([]string{"tick", "--json", "list"}, dir, &stdout, &stderr, false)

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		var arr []interface{}
		if err := json.Unmarshal(stdout.Bytes(), &arr); err != nil {
			t.Fatalf("expected valid JSON with --json override, got %q: %v", stdout.String(), err)
		}
	})

	// --- Errors ---

	t.Run("errors remain plain text stderr regardless of format", func(t *testing.T) {
		dir := initTickProject(t)

		formats := []string{"--toon", "--pretty", "--json"}
		for _, f := range formats {
			t.Run(f, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				code := Run([]string{"tick", f, "show", "tick-nonexist"}, dir, &stdout, &stderr, false)

				if code != 1 {
					t.Fatalf("expected exit code 1, got %d", code)
				}

				errOutput := stderr.String()
				if !strings.HasPrefix(errOutput, "Error:") {
					t.Errorf("expected plain text error starting with 'Error:', got %q", errOutput)
				}
				// Should NOT be JSON even with --json flag
				if strings.HasPrefix(errOutput, "{") {
					t.Errorf("errors should not be JSON, got %q", errOutput)
				}
			})
		}
	})

	// --- Formatter resolved once in dispatcher ---

	t.Run("it resolves formatter in Context for handlers to use", func(t *testing.T) {
		// Verify that Context has a Formatter field set after parseArgs
		var stdout, stderr bytes.Buffer
		ctx, _, err := parseArgs(
			[]string{"tick", "--json", "list"},
			t.TempDir(), &stdout, &stderr, false,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ctx.Fmt == nil {
			t.Error("expected Fmt to be set in Context after parseArgs")
		}

		// Verify it is a JSONFormatter
		if _, ok := ctx.Fmt.(*JSONFormatter); !ok {
			t.Errorf("expected JSONFormatter, got %T", ctx.Fmt)
		}
	})

	t.Run("it sets ToonFormatter when non-TTY default", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		ctx, _, err := parseArgs(
			[]string{"tick", "list"},
			t.TempDir(), &stdout, &stderr, false,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := ctx.Fmt.(*ToonFormatter); !ok {
			t.Errorf("expected ToonFormatter for non-TTY, got %T", ctx.Fmt)
		}
	})

	t.Run("it sets PrettyFormatter when TTY default", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		ctx, _, err := parseArgs(
			[]string{"tick", "list"},
			t.TempDir(), &stdout, &stderr, true,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := ctx.Fmt.(*PrettyFormatter); !ok {
			t.Errorf("expected PrettyFormatter for TTY, got %T", ctx.Fmt)
		}
	})
}
