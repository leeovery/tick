package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestFormatIntegration tests that all commands output via the resolved Formatter,
// format switching works, and --quiet overrides behave per spec.
func TestFormatIntegration(t *testing.T) {
	// --- init/rebuild: FormatMessage ---

	t.Run("it formats init as message in each format", func(t *testing.T) {
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					// ToonFormatter.FormatMessage outputs plain text with newline
					if !strings.Contains(output, "Initialized tick in") {
						t.Errorf("toon output should contain init message, got:\n%s", output)
					}
				},
			},
			{
				"pretty",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "Initialized tick in") {
						t.Errorf("pretty output should contain init message, got:\n%s", output)
					}
				},
			},
			{
				"json",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					// JSONFormatter.FormatMessage wraps in {"message":"..."}
					var msg struct {
						Message string `json:"message"`
					}
					if err := json.Unmarshal([]byte(output), &msg); err != nil {
						t.Fatalf("json output is not valid JSON: %v\noutput: %s", err, output)
					}
					if !strings.Contains(msg.Message, "Initialized tick in") {
						t.Errorf("json message = %q, want it to contain init message", msg.Message)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := t.TempDir()

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "init"})
				if err != nil {
					t.Fatalf("init returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	// --- create/update: FormatTaskDetail ---

	t.Run("it formats create as full task detail in each format", func(t *testing.T) {
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					// ToonFormatter.FormatTaskDetail starts with "task{..."
					if !strings.Contains(output, "task{") {
						t.Errorf("toon output should contain task{ section header, got:\n%s", output)
					}
					if !strings.Contains(output, "blocked_by[") {
						t.Errorf("toon output should contain blocked_by section, got:\n%s", output)
					}
					if !strings.Contains(output, "children[") {
						t.Errorf("toon output should contain children section, got:\n%s", output)
					}
				},
			},
			{
				"pretty",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "ID:") {
						t.Errorf("pretty output should contain ID: label, got:\n%s", output)
					}
					if !strings.Contains(output, "Title:") {
						t.Errorf("pretty output should contain Title: label, got:\n%s", output)
					}
					if !strings.Contains(output, "Created:") {
						t.Errorf("pretty output should contain Created: label, got:\n%s", output)
					}
				},
			},
			{
				"json",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					var detail map[string]interface{}
					if err := json.Unmarshal([]byte(output), &detail); err != nil {
						t.Fatalf("json output is not valid JSON: %v\noutput: %s", err, output)
					}
					if _, ok := detail["id"]; !ok {
						t.Errorf("json output missing 'id' key")
					}
					if _, ok := detail["title"]; !ok {
						t.Errorf("json output missing 'title' key")
					}
					if _, ok := detail["blocked_by"]; !ok {
						t.Errorf("json output missing 'blocked_by' key")
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupInitializedTickDir(t)

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "create", "Format test task"})
				if err != nil {
					t.Fatalf("create returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	t.Run("it formats update as full task detail in each format", func(t *testing.T) {
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "task{") {
						t.Errorf("toon output should contain task{ section, got:\n%s", output)
					}
				},
			},
			{
				"pretty",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "ID:") {
						t.Errorf("pretty output should contain ID: label, got:\n%s", output)
					}
				},
			},
			{
				"json",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					var detail map[string]interface{}
					if err := json.Unmarshal([]byte(output), &detail); err != nil {
						t.Fatalf("json output not valid JSON: %v\noutput: %s", err, output)
					}
					if detail["title"] != "Updated title" {
						t.Errorf("json title = %v, want 'Updated title'", detail["title"])
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				content := `{"id":"tick-aaa111","title":"Old title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
				dir := setupTickDirWithContent(t, content)

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "update", "tick-aaa111", "--title", "Updated title"})
				if err != nil {
					t.Fatalf("update returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	// --- transitions: FormatTransition ---

	t.Run("it formats transitions in each format", func(t *testing.T) {
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					// Toon/Pretty transitions are plain text: "id: old -> new"
					if !strings.Contains(output, "tick-aaa111:") {
						t.Errorf("toon output should contain task ID, got:\n%s", output)
					}
					if !strings.Contains(output, "\u2192") {
						t.Errorf("toon output should contain arrow, got:\n%s", output)
					}
				},
			},
			{
				"pretty",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "tick-aaa111:") {
						t.Errorf("pretty output should contain task ID, got:\n%s", output)
					}
				},
			},
			{
				"json",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					var trans struct {
						ID   string `json:"id"`
						From string `json:"from"`
						To   string `json:"to"`
					}
					if err := json.Unmarshal([]byte(output), &trans); err != nil {
						t.Fatalf("json output not valid JSON: %v\noutput: %s", err, output)
					}
					if trans.ID != "tick-aaa111" {
						t.Errorf("json id = %q, want 'tick-aaa111'", trans.ID)
					}
					if trans.From != "open" {
						t.Errorf("json from = %q, want 'open'", trans.From)
					}
					if trans.To != "in_progress" {
						t.Errorf("json to = %q, want 'in_progress'", trans.To)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "start", "tick-aaa111"})
				if err != nil {
					t.Fatalf("start returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	// --- dep add/rm: FormatDepChange ---

	t.Run("it formats dep confirmations in each format", func(t *testing.T) {
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon add",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					want := "Dependency added: tick-aaa111 blocked by tick-bbb222"
					if !strings.Contains(output, want) {
						t.Errorf("toon output = %q, want it to contain %q", output, want)
					}
				},
			},
			{
				"pretty add",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					want := "Dependency added: tick-aaa111 blocked by tick-bbb222"
					if !strings.Contains(output, want) {
						t.Errorf("pretty output = %q, want it to contain %q", output, want)
					}
				},
			},
			{
				"json add",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					var dep struct {
						Action    string `json:"action"`
						TaskID    string `json:"task_id"`
						BlockedBy string `json:"blocked_by"`
					}
					if err := json.Unmarshal([]byte(output), &dep); err != nil {
						t.Fatalf("json output not valid JSON: %v\noutput: %s", err, output)
					}
					if dep.Action != "added" {
						t.Errorf("json action = %q, want 'added'", dep.Action)
					}
					if dep.TaskID != "tick-aaa111" {
						t.Errorf("json task_id = %q, want 'tick-aaa111'", dep.TaskID)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupTickDirWithContent(t, twoOpenTasksJSONL())

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "dep", "add", "tick-aaa111", "tick-bbb222"})
				if err != nil {
					t.Fatalf("dep add returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	// --- list/show: FormatTaskList / FormatTaskDetail ---

	t.Run("it formats list in each format", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task one","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Task two","status":"open","priority":2,"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
`
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "tasks[2]") {
						t.Errorf("toon output should contain tasks[2] header, got:\n%s", output)
					}
				},
			},
			{
				"pretty",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") {
						t.Errorf("pretty output should contain column headers, got:\n%s", output)
					}
				},
			},
			{
				"json",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					var rows []map[string]interface{}
					if err := json.Unmarshal([]byte(output), &rows); err != nil {
						t.Fatalf("json output not valid JSON array: %v\noutput: %s", err, output)
					}
					if len(rows) != 2 {
						t.Errorf("json array length = %d, want 2", len(rows))
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupTickDirWithContent(t, content)

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "list"})
				if err != nil {
					t.Fatalf("list returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	t.Run("it formats show in each format", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Show task","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "task{") {
						t.Errorf("toon output should contain task{ section, got:\n%s", output)
					}
				},
			},
			{
				"pretty",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "ID:       tick-aaa111") {
						t.Errorf("pretty output should contain aligned ID, got:\n%s", output)
					}
				},
			},
			{
				"json",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					var detail map[string]interface{}
					if err := json.Unmarshal([]byte(output), &detail); err != nil {
						t.Fatalf("json output not valid JSON: %v\noutput: %s", err, output)
					}
					if detail["id"] != "tick-aaa111" {
						t.Errorf("json id = %v, want 'tick-aaa111'", detail["id"])
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupTickDirWithContent(t, content)

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "show", "tick-aaa111"})
				if err != nil {
					t.Fatalf("show returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	// --- quiet overrides ---

	t.Run("it applies --quiet override for create (ID only)", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "create", "Quiet create"})
		if err != nil {
			t.Fatalf("create returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if !strings.HasPrefix(output, "tick-") {
			t.Errorf("quiet create should output only ID, got: %q", output)
		}
		if len(output) != 11 {
			t.Errorf("quiet create output length = %d, want 11 (just ID)", len(output))
		}
	})

	t.Run("it applies --quiet override for update (ID only)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Old","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "update", "tick-aaa111", "--title", "New"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("quiet update should output only ID, got: %q", output)
		}
	})

	t.Run("it applies --quiet override for transitions (nothing)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "start", "tick-aaa111"})
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}

		if stdout.String() != "" {
			t.Errorf("quiet transition should produce no output, got: %q", stdout.String())
		}
	})

	t.Run("it applies --quiet override for dep add (nothing)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, twoOpenTasksJSONL())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep add returned error: %v", err)
		}

		if stdout.String() != "" {
			t.Errorf("quiet dep add should produce no output, got: %q", stdout.String())
		}
	})

	t.Run("it applies --quiet override for init (nothing)", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		if stdout.String() != "" {
			t.Errorf("quiet init should produce no output, got: %q", stdout.String())
		}
	})

	t.Run("it applies --quiet override for list (IDs only)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("quiet list should output only IDs, got: %q", output)
		}
	})

	t.Run("it applies --quiet override for show (ID only)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Show task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("quiet show should output only ID, got: %q", output)
		}
	})

	// --- quiet + json: quiet wins ---

	t.Run("it applies --quiet even when --json is set (quiet wins)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "--json", "start", "tick-aaa111"})
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}

		if stdout.String() != "" {
			t.Errorf("--quiet + --json should produce nothing for transitions, got: %q", stdout.String())
		}
	})

	// --- empty list per format ---

	t.Run("it handles empty list per format", func(t *testing.T) {
		tests := []struct {
			name  string
			flag  string
			check func(t *testing.T, output string)
		}{
			{
				"toon",
				"--toon",
				func(t *testing.T, output string) {
					t.Helper()
					if !strings.Contains(output, "tasks[0]") {
						t.Errorf("toon empty list should contain tasks[0] header, got:\n%s", output)
					}
				},
			},
			{
				"pretty",
				"--pretty",
				func(t *testing.T, output string) {
					t.Helper()
					trimmed := strings.TrimSpace(output)
					if trimmed != "No tasks found." {
						t.Errorf("pretty empty list = %q, want 'No tasks found.'", trimmed)
					}
				},
			},
			{
				"json",
				"--json",
				func(t *testing.T, output string) {
					t.Helper()
					trimmed := strings.TrimSpace(output)
					if trimmed != "[]" {
						t.Errorf("json empty list = %q, want '[]'", trimmed)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := setupInitializedTickDir(t)

				app := NewApp()
				app.workDir = dir
				var stdout strings.Builder
				app.stdout = &stdout

				err := app.Run([]string{"tick", tt.flag, "list"})
				if err != nil {
					t.Fatalf("list returned error: %v", err)
				}

				tt.check(t, stdout.String())
			})
		}
	})

	// --- TTY auto-detection ---

	t.Run("it defaults to TOON when piped (non-TTY)", func(t *testing.T) {
		// Test environment is non-TTY; without flag, should use TOON
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "list"})
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}

		output := stdout.String()
		// In non-TTY, list should use TOON format: "tasks[N]{...}:"
		if !strings.Contains(output, "tasks[") {
			t.Errorf("non-TTY list output should be TOON format, got:\n%s", output)
		}
	})

	// --- format flag overrides ---

	t.Run("it respects --toon override", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--toon", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		if !strings.Contains(stdout.String(), "task{") {
			t.Errorf("--toon show should use TOON format, got:\n%s", stdout.String())
		}
	})

	t.Run("it respects --pretty override", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--pretty", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		if !strings.Contains(stdout.String(), "ID:       tick-aaa111") {
			t.Errorf("--pretty show should use Pretty format, got:\n%s", stdout.String())
		}
	})

	t.Run("it respects --json override", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--json", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		var detail map[string]interface{}
		if err := json.Unmarshal([]byte(stdout.String()), &detail); err != nil {
			t.Errorf("--json show should produce valid JSON, got:\n%s", stdout.String())
		}
	})

	// --- format resolved once in dispatcher ---

	t.Run("it resolves format once in dispatcher not per command", func(t *testing.T) {
		dir := t.TempDir()

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--json", "init"})
		if err != nil {
			t.Fatalf("init returned error: %v", err)
		}

		// Verify the formatter field was set on App
		if app.formatter == nil {
			t.Error("app.formatter should be set after Run()")
		}
	})
}
