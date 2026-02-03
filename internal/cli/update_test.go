package cli

import (
	"strings"
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	t.Run("it updates title with --title flag", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Old title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "New title"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["title"] != "New title" {
			t.Errorf("title = %q, want %q", tk["title"], "New title")
		}
	})

	t.Run("it updates description with --description flag", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--description", "New description"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["description"] != "New description" {
			t.Errorf("description = %q, want %q", tk["description"], "New description")
		}
	})

	t.Run("it clears description with --description empty string", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"description":"Old desc","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--description", ""})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		// Description should be cleared (omitted from JSONL)
		if _, hasDesc := tk["description"]; hasDesc {
			t.Errorf("description should be cleared (omitted), got %q", tk["description"])
		}
	})

	t.Run("it updates priority with --priority flag", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--priority", "0"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if int(tk["priority"].(float64)) != 0 {
			t.Errorf("priority = %v, want 0", tk["priority"])
		}
	})

	t.Run("it updates parent with --parent flag", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Parent task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--parent", "tick-bbb222"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["parent"] != "tick-bbb222" {
			t.Errorf("parent = %q, want %q", tk["parent"], "tick-bbb222")
		}
	})

	t.Run("it clears parent with --parent empty string", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"parent":"tick-bbb222","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Parent task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--parent", ""})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if _, hasParent := tk["parent"]; hasParent {
			t.Errorf("parent should be cleared (omitted), got %q", tk["parent"])
		}
	})

	t.Run("it updates blocks with --blocks flag", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Blocker task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Target task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--blocks", "tick-bbb222"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		// Target task should now have tick-aaa111 in its blocked_by
		target := readTaskByID(t, dir, "tick-bbb222")
		blockedBy, ok := target["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("target blocked_by is not an array: %T", target["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != "tick-aaa111" {
			t.Errorf("target blocked_by = %v, want [tick-aaa111]", blockedBy)
		}

		// Target task's updated should have changed
		if target["updated"] == "2026-01-19T10:00:00Z" {
			t.Error("target task's updated timestamp should have changed")
		}
	})

	t.Run("it updates multiple fields in a single command", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Old title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "New title", "--priority", "1", "--description", "New desc"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["title"] != "New title" {
			t.Errorf("title = %q, want %q", tk["title"], "New title")
		}
		if int(tk["priority"].(float64)) != 1 {
			t.Errorf("priority = %v, want 1", tk["priority"])
		}
		if tk["description"] != "New desc" {
			t.Errorf("description = %q, want %q", tk["description"], "New desc")
		}
	})

	t.Run("it refreshes updated timestamp on any change", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "Updated title"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["updated"] == "2026-01-19T10:00:00Z" {
			t.Error("updated timestamp should have changed")
		}
	})

	t.Run("it outputs full task details on success", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "Updated task"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("output should contain task ID:\n%s", output)
		}
		if !strings.Contains(output, "Updated task") {
			t.Errorf("output should contain updated title:\n%s", output)
		}
		if !strings.Contains(output, "open") {
			t.Errorf("output should contain status:\n%s", output)
		}
	})

	t.Run("it outputs only task ID with --quiet flag", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "update", "tick-aaa111", "--title", "Quiet update"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "tick-aaa111" {
			t.Errorf("output = %q, want %q (just the task ID)", output, "tick-aaa111")
		}
	})

	t.Run("it errors when no flags are provided", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-aaa111"})
		if err == nil {
			t.Fatal("expected error when no flags provided, got nil")
		}
		// Should mention available options
		if !strings.Contains(err.Error(), "--title") {
			t.Errorf("error = %q, want it to mention --title", err.Error())
		}
		if !strings.Contains(err.Error(), "--description") {
			t.Errorf("error = %q, want it to mention --description", err.Error())
		}
	})

	t.Run("it errors when task ID is missing", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update"})
		if err == nil {
			t.Fatal("expected error when task ID missing, got nil")
		}
		if !strings.Contains(err.Error(), "Task ID is required") {
			t.Errorf("error = %q, want it to contain 'Task ID is required'", err.Error())
		}
	})

	t.Run("it errors when task ID is not found", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-nonexist", "--title", "New"})
		if err == nil {
			t.Fatal("expected error for non-existent task, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want it to contain 'not found'", err.Error())
		}
	})

	t.Run("it errors on invalid title (empty after trim)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "   "})
		if err == nil {
			t.Fatal("expected error for whitespace-only title, got nil")
		}
		if !strings.Contains(err.Error(), "title") {
			t.Errorf("error = %q, want it to mention 'title'", err.Error())
		}

		// Verify no mutation happened
		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["title"] != "Test task" {
			t.Errorf("title should be unchanged, got %q", tk["title"])
		}
	})

	t.Run("it errors on invalid title (over 500 chars)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		longTitle := strings.Repeat("a", 501)
		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", longTitle})
		if err == nil {
			t.Fatal("expected error for title exceeding 500 chars, got nil")
		}
		if !strings.Contains(err.Error(), "title") || !strings.Contains(err.Error(), "500") {
			t.Errorf("error = %q, want it to mention title and 500", err.Error())
		}
	})

	t.Run("it errors on invalid title (contains newlines)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "line1\nline2"})
		if err == nil {
			t.Fatal("expected error for title with newlines, got nil")
		}
		if !strings.Contains(err.Error(), "newline") {
			t.Errorf("error = %q, want it to mention 'newline'", err.Error())
		}
	})

	t.Run("it errors on invalid priority (outside 0-4)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--priority", "5"})
		if err == nil {
			t.Fatal("expected error for invalid priority, got nil")
		}
		if !strings.Contains(err.Error(), "priority") {
			t.Errorf("error = %q, want it to mention 'priority'", err.Error())
		}

		// Also test negative
		app2 := NewApp()
		app2.workDir = dir
		err = app2.Run([]string{"tick", "update", "tick-aaa111", "--priority", "-1"})
		if err == nil {
			t.Fatal("expected error for negative priority, got nil")
		}
	})

	t.Run("it errors on non-existent parent ID", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--parent", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error for non-existent parent, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}
	})

	t.Run("it errors on non-existent blocks ID", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--blocks", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error for non-existent blocks ID, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}
	})

	t.Run("it errors on self-referencing parent", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--parent", "tick-aaa111"})
		if err == nil {
			t.Fatal("expected error for self-referencing parent, got nil")
		}
		if !strings.Contains(err.Error(), "cannot be its own parent") {
			t.Errorf("error = %q, want it to contain 'cannot be its own parent'", err.Error())
		}
	})

	t.Run("it normalizes input IDs to lowercase", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Parent task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		// Use uppercase IDs
		err := app.Run([]string{"tick", "update", "TICK-AAA111", "--parent", "TICK-BBB222"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["parent"] != "tick-bbb222" {
			t.Errorf("parent = %q, want %q (lowercase)", tk["parent"], "tick-bbb222")
		}
	})

	t.Run("it persists changes via atomic write", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Original","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "Persisted"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		// Read back directly from file
		tasks := readTasksJSONL(t, dir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0]["title"] != "Persisted" {
			t.Errorf("persisted title = %q, want %q", tasks[0]["title"], "Persisted")
		}
	})

	t.Run("it silently skips duplicate when --blocks target already has source in blocked_by", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Blocker task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Target task","status":"open","priority":2,"blocked_by":["tick-aaa111"],"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		// tick-bbb222 already has tick-aaa111 in blocked_by; running --blocks again should not duplicate
		err := app.Run([]string{"tick", "update", "tick-aaa111", "--blocks", "tick-bbb222"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		target := readTaskByID(t, dir, "tick-bbb222")
		blockedBy, ok := target["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("target blocked_by is not an array: %T", target["blocked_by"])
		}
		if len(blockedBy) != 1 {
			t.Errorf("blocked_by has %d items, want 1 (no duplicate)", len(blockedBy))
		}
		if blockedBy[0] != "tick-aaa111" {
			t.Errorf("blocked_by[0] = %v, want tick-aaa111", blockedBy[0])
		}
	})

	t.Run("it updates multiple targets with comma-separated --blocks", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Blocker task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Target one","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-ccc333","title":"Target two","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--blocks", "tick-bbb222,tick-ccc333"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		// Both targets should have tick-aaa111 in blocked_by
		target1 := readTaskByID(t, dir, "tick-bbb222")
		blockedBy1, ok := target1["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("target1 blocked_by is not an array: %T", target1["blocked_by"])
		}
		if len(blockedBy1) != 1 || blockedBy1[0] != "tick-aaa111" {
			t.Errorf("target1 blocked_by = %v, want [tick-aaa111]", blockedBy1)
		}

		target2 := readTaskByID(t, dir, "tick-ccc333")
		blockedBy2, ok := target2["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("target2 blocked_by is not an array: %T", target2["blocked_by"])
		}
		if len(blockedBy2) != 1 || blockedBy2[0] != "tick-aaa111" {
			t.Errorf("target2 blocked_by = %v, want [tick-aaa111]", blockedBy2)
		}

		// Both targets' updated timestamps should have changed
		if target1["updated"] == "2026-01-19T10:00:00Z" {
			t.Error("target1 updated timestamp should have changed")
		}
		if target2["updated"] == "2026-01-19T10:00:00Z" {
			t.Error("target2 updated timestamp should have changed")
		}
	})

	t.Run("it applies --blocks combined with --title atomically", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Old title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-bbb222","title":"Target task","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
`
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "update", "tick-aaa111", "--title", "New title", "--blocks", "tick-bbb222"})
		if err != nil {
			t.Fatalf("update returned error: %v", err)
		}

		// Source task should have new title
		source := readTaskByID(t, dir, "tick-aaa111")
		if source["title"] != "New title" {
			t.Errorf("source title = %q, want %q", source["title"], "New title")
		}

		// Target task should have tick-aaa111 in blocked_by
		target := readTaskByID(t, dir, "tick-bbb222")
		blockedBy, ok := target["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("target blocked_by is not an array: %T", target["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != "tick-aaa111" {
			t.Errorf("target blocked_by = %v, want [tick-aaa111]", blockedBy)
		}
	})
}
