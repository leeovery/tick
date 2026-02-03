package cli

import (
	"strings"
	"testing"
)

func TestShowCommand(t *testing.T) {
	t.Run("it shows full task details by ID", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Setup Sanctum","status":"in_progress","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:30:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()

		// Check all core fields
		if !strings.Contains(output, "ID:       tick-aaa111") {
			t.Errorf("output missing ID field:\n%s", output)
		}
		if !strings.Contains(output, "Title:    Setup Sanctum") {
			t.Errorf("output missing Title field:\n%s", output)
		}
		if !strings.Contains(output, "Status:   in_progress") {
			t.Errorf("output missing Status field:\n%s", output)
		}
		if !strings.Contains(output, "Priority: 1") {
			t.Errorf("output missing Priority field:\n%s", output)
		}
		if !strings.Contains(output, "Created:  2026-01-19T10:00:00Z") {
			t.Errorf("output missing Created field:\n%s", output)
		}
		if !strings.Contains(output, "Updated:  2026-01-19T14:30:00Z") {
			t.Errorf("output missing Updated field:\n%s", output)
		}
	})

	t.Run("it shows blocked_by section with ID, title, and status of each blocker", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Setup Sanctum","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Login endpoint","status":"open","priority":1,"blocked_by":["tick-aaa111"],"created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-bbb222"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Blocked by:") {
			t.Errorf("output missing 'Blocked by:' section:\n%s", output)
		}
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("output missing blocker ID:\n%s", output)
		}
		if !strings.Contains(output, "Setup Sanctum") {
			t.Errorf("output missing blocker title:\n%s", output)
		}
		if !strings.Contains(output, "(done)") {
			t.Errorf("output missing blocker status:\n%s", output)
		}
	})

	t.Run("it shows children section with ID, title, and status of each child", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Auth System","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Sub-task one","status":"open","priority":1,"parent":"tick-aaa111","created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Children:") {
			t.Errorf("output missing 'Children:' section:\n%s", output)
		}
		if !strings.Contains(output, "tick-bbb222") {
			t.Errorf("output missing child ID:\n%s", output)
		}
		if !strings.Contains(output, "Sub-task one") {
			t.Errorf("output missing child title:\n%s", output)
		}
		if !strings.Contains(output, "(open)") {
			t.Errorf("output missing child status:\n%s", output)
		}
	})

	t.Run("it shows description section when description is present", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task with desc","status":"open","priority":2,"description":"Implement the login endpoint with validation...","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Description:") {
			t.Errorf("output missing 'Description:' section:\n%s", output)
		}
		if !strings.Contains(output, "Implement the login endpoint with validation...") {
			t.Errorf("output missing description text:\n%s", output)
		}
	})

	t.Run("it omits blocked_by section when task has no dependencies", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"No deps","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "Blocked by:") {
			t.Errorf("output should omit 'Blocked by:' when no dependencies:\n%s", output)
		}
	})

	t.Run("it omits children section when task has no children", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"No children","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "Children:") {
			t.Errorf("output should omit 'Children:' when no children:\n%s", output)
		}
	})

	t.Run("it omits description section when description is empty", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"No desc","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "Description:") {
			t.Errorf("output should omit 'Description:' when empty:\n%s", output)
		}
	})

	t.Run("it shows parent field with ID and title when parent is set", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Parent task","status":"open","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Child task","status":"open","priority":1,"parent":"tick-aaa111","created":"2026-01-19T11:00:00Z","updated":"2026-01-19T11:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-bbb222"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Parent:") {
			t.Errorf("output missing 'Parent:' field:\n%s", output)
		}
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("output missing parent ID:\n%s", output)
		}
		if !strings.Contains(output, "Parent task") {
			t.Errorf("output missing parent title:\n%s", output)
		}
	})

	t.Run("it omits parent field when parent is null", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Root task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "Parent:") {
			t.Errorf("output should omit 'Parent:' when not set:\n%s", output)
		}
	})

	t.Run("it shows closed timestamp when task is done or cancelled", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Done task","status":"done","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T16:00:00Z","closed":"2026-01-19T16:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Closed:") {
			t.Errorf("output missing 'Closed:' field:\n%s", output)
		}
		if !strings.Contains(output, "2026-01-19T16:00:00Z") {
			t.Errorf("output missing closed timestamp:\n%s", output)
		}
	})

	t.Run("it omits closed field when task is open or in_progress", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Open task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "show", "tick-aaa111"})
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "Closed:") {
			t.Errorf("output should omit 'Closed:' for open tasks:\n%s", output)
		}
	})

	t.Run("it errors when task ID not found", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "show", "tick-xyz123"})
		if err == nil {
			t.Fatal("expected error for non-existent task, got nil")
		}
		if !strings.Contains(err.Error(), "tick-xyz123") {
			t.Errorf("error = %q, want it to contain 'tick-xyz123'", err.Error())
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want it to contain 'not found'", err.Error())
		}
	})

	t.Run("it errors when no ID argument provided to show", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "show"})
		if err == nil {
			t.Fatal("expected error when no ID argument, got nil")
		}
		if !strings.Contains(err.Error(), "Task ID is required") {
			t.Errorf("error = %q, want it to contain 'Task ID is required'", err.Error())
		}
		if !strings.Contains(err.Error(), "tick show <id>") {
			t.Errorf("error = %q, want it to contain usage hint 'tick show <id>'", err.Error())
		}
	})

	t.Run("it normalizes input ID to lowercase for show lookup", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Lowercase task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		// Pass uppercase ID
		err := app.Run([]string{"tick", "show", "TICK-AAA111"})
		if err != nil {
			t.Fatalf("show returned error: %v (should find task with case-insensitive lookup)", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("output should contain lowercase ID:\n%s", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag on show", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Quiet show","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
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
			t.Errorf("output = %q, want %q (just the task ID)", output, "tick-aaa111")
		}
	})
}
