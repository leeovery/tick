package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// twoOpenTasksJSONL returns JSONL content with two open tasks for dep testing.
func twoOpenTasksJSONL() string {
	return openTaskJSONL("tick-aaa111") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
}

// openTaskWithBlockedByJSONL returns a JSONL line for an open task blocked by another.
func openTaskWithBlockedByJSONL(id, blockedByID string) string {
	return `{"id":"` + id + `","title":"Test task","status":"open","priority":2,"blocked_by":["` + blockedByID + `"],"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
}

// openTaskWithParentJSONL returns a JSONL line for an open task with a parent.
func openTaskWithParentJSONL(id, parentID string) string {
	return `{"id":"` + id + `","title":"Test task","status":"open","priority":2,"parent":"` + parentID + `","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
}

func TestDepAddCommand(t *testing.T) {
	t.Run("it adds a dependency between two existing tasks", func(t *testing.T) {
		dir := setupTickDirWithContent(t, twoOpenTasksJSONL())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep add returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		blockedBy, ok := tk["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", tk["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != "tick-bbb222" {
			t.Errorf("blocked_by = %v, want [tick-bbb222]", blockedBy)
		}
	})

	t.Run("it outputs confirmation on success (add)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, twoOpenTasksJSONL())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep add returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		want := "Dependency added: tick-aaa111 blocked by tick-bbb222"
		if output != want {
			t.Errorf("output = %q, want %q", output, want)
		}
	})

	t.Run("it updates task's updated timestamp on add", func(t *testing.T) {
		dir := setupTickDirWithContent(t, twoOpenTasksJSONL())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep add returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["updated"] == "2026-01-19T10:00:00Z" {
			t.Error("updated timestamp should have changed after dep add")
		}
	})

	t.Run("it errors when task_id not found (add)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-bbb222")+"\n")

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "add", "tick-nonexist", "tick-bbb222"})
		if err == nil {
			t.Fatal("expected error for non-existent task_id, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}
	})

	t.Run("it errors when blocked_by_id not found (add)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-nonexist"})
		if err == nil {
			t.Fatal("expected error for non-existent blocked_by_id, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}
	})

	t.Run("it errors on duplicate dependency (add)", func(t *testing.T) {
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if err == nil {
			t.Fatal("expected error for duplicate dependency, got nil")
		}
		if !strings.Contains(err.Error(), "already") {
			t.Errorf("error = %q, want it to contain 'already'", err.Error())
		}

		// Verify no mutation occurred
		tk := readTaskByID(t, dir, "tick-aaa111")
		blockedBy := tk["blocked_by"].([]interface{})
		if len(blockedBy) != 1 {
			t.Errorf("blocked_by should still have 1 entry, got %d", len(blockedBy))
		}
	})

	t.Run("it errors on self-reference (add)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-aaa111"})
		if err == nil {
			t.Fatal("expected error for self-reference, got nil")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("error = %q, want it to contain 'cycle'", err.Error())
		}
	})

	t.Run("it errors when add creates cycle", func(t *testing.T) {
		// tick-aaa111 is blocked by tick-bbb222. Adding tick-bbb222 blocked by tick-aaa111 creates cycle.
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "add", "tick-bbb222", "tick-aaa111"})
		if err == nil {
			t.Fatal("expected error for cycle, got nil")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("error = %q, want it to contain 'cycle'", err.Error())
		}
	})

	t.Run("it errors when add creates child-blocked-by-parent", func(t *testing.T) {
		// tick-child has parent tick-parent. Adding tick-child blocked by tick-parent should error.
		content := openTaskWithParentJSONL("tick-child1", "tick-parent") + "\n" + openTaskJSONL("tick-parent") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "add", "tick-child1", "tick-parent"})
		if err == nil {
			t.Fatal("expected error for child-blocked-by-parent, got nil")
		}
		if !strings.Contains(err.Error(), "parent") {
			t.Errorf("error = %q, want it to contain 'parent'", err.Error())
		}
	})

	t.Run("it normalizes IDs to lowercase (add)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, twoOpenTasksJSONL())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "add", "TICK-AAA111", "TICK-BBB222"})
		if err != nil {
			t.Fatalf("dep add returned error: %v", err)
		}

		// Verify the dependency was added using normalized IDs
		tk := readTaskByID(t, dir, "tick-aaa111")
		blockedBy, ok := tk["blocked_by"].([]interface{})
		if !ok {
			t.Fatalf("blocked_by is not an array: %T", tk["blocked_by"])
		}
		if len(blockedBy) != 1 || blockedBy[0] != "tick-bbb222" {
			t.Errorf("blocked_by = %v, want [tick-bbb222]", blockedBy)
		}

		// Output should use lowercase IDs
		output := strings.TrimSpace(stdout.String())
		want := "Dependency added: tick-aaa111 blocked by tick-bbb222"
		if output != want {
			t.Errorf("output = %q, want %q", output, want)
		}
	})

	t.Run("it suppresses output with --quiet (add)", func(t *testing.T) {
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
			t.Errorf("stdout = %q, want empty with --quiet", stdout.String())
		}
	})

	t.Run("it errors when fewer than two IDs provided (add)", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		// No IDs
		err := app.Run([]string{"tick", "dep", "add"})
		if err == nil {
			t.Fatal("expected error for missing IDs, got nil")
		}
		if !strings.Contains(err.Error(), "Usage") || !strings.Contains(err.Error(), "dep add") {
			t.Errorf("error = %q, want it to contain usage hint with 'dep add'", err.Error())
		}

		// One ID only
		app2 := NewApp()
		app2.workDir = dir
		err = app2.Run([]string{"tick", "dep", "add", "tick-aaa111"})
		if err == nil {
			t.Fatal("expected error for only one ID, got nil")
		}
		if !strings.Contains(err.Error(), "Usage") {
			t.Errorf("error = %q, want it to contain 'Usage'", err.Error())
		}
	})

	t.Run("it persists via atomic write (add)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, twoOpenTasksJSONL())

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep add returned error: %v", err)
		}

		// Read the file directly to verify persistence
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		if !strings.Contains(string(data), "tick-bbb222") {
			t.Error("persisted data should contain the dependency tick-bbb222")
		}
	})
}

func TestDepRmCommand(t *testing.T) {
	t.Run("it removes an existing dependency", func(t *testing.T) {
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep rm returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		// blocked_by should be empty (omitted) or empty array
		if blockedBy, ok := tk["blocked_by"]; ok {
			arr, ok := blockedBy.([]interface{})
			if ok && len(arr) > 0 {
				t.Errorf("blocked_by should be empty after rm, got %v", arr)
			}
		}
	})

	t.Run("it outputs confirmation on success (rm)", func(t *testing.T) {
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep rm returned error: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		want := "Dependency removed: tick-aaa111 no longer blocked by tick-bbb222"
		if output != want {
			t.Errorf("output = %q, want %q", output, want)
		}
	})

	t.Run("it updates task's updated timestamp on rm", func(t *testing.T) {
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep rm returned error: %v", err)
		}

		tk := readTaskByID(t, dir, "tick-aaa111")
		if tk["updated"] == "2026-01-19T10:00:00Z" {
			t.Error("updated timestamp should have changed after dep rm")
		}
	})

	t.Run("it errors when task_id not found (rm)", func(t *testing.T) {
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-bbb222")+"\n")

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "rm", "tick-nonexist", "tick-bbb222"})
		if err == nil {
			t.Fatal("expected error for non-existent task_id, got nil")
		}
		if !strings.Contains(err.Error(), "tick-nonexist") {
			t.Errorf("error = %q, want it to contain 'tick-nonexist'", err.Error())
		}
	})

	t.Run("it errors when dependency not found in blocked_by (rm)", func(t *testing.T) {
		// tick-aaa111 has no dependencies - removing one that doesn't exist should error
		dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n"+openTaskJSONL("tick-bbb222")+"\n")

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if err == nil {
			t.Fatal("expected error for dependency not in blocked_by, got nil")
		}
		if !strings.Contains(err.Error(), "tick-bbb222") {
			t.Errorf("error = %q, want it to contain 'tick-bbb222'", err.Error())
		}
	})

	t.Run("it does not validate blocked_by_id exists as a task on rm (supports stale refs)", func(t *testing.T) {
		// tick-aaa111 has tick-stale1 in blocked_by, but tick-stale1 doesn't exist as a task
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-stale1") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-stale1"})
		if err != nil {
			t.Fatalf("dep rm should succeed for stale refs, got error: %v", err)
		}

		// Verify it was removed
		tk := readTaskByID(t, dir, "tick-aaa111")
		if blockedBy, ok := tk["blocked_by"]; ok {
			arr, ok := blockedBy.([]interface{})
			if ok && len(arr) > 0 {
				t.Errorf("blocked_by should be empty after removing stale ref, got %v", arr)
			}
		}
	})

	t.Run("it normalizes IDs to lowercase (rm)", func(t *testing.T) {
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "rm", "TICK-AAA111", "TICK-BBB222"})
		if err != nil {
			t.Fatalf("dep rm returned error: %v", err)
		}

		// Verify dependency was removed
		tk := readTaskByID(t, dir, "tick-aaa111")
		if blockedBy, ok := tk["blocked_by"]; ok {
			arr, ok := blockedBy.([]interface{})
			if ok && len(arr) > 0 {
				t.Errorf("blocked_by should be empty after rm, got %v", arr)
			}
		}

		// Output should use lowercase IDs
		output := strings.TrimSpace(stdout.String())
		want := "Dependency removed: tick-aaa111 no longer blocked by tick-bbb222"
		if output != want {
			t.Errorf("output = %q, want %q", output, want)
		}
	})

	t.Run("it suppresses output with --quiet (rm)", func(t *testing.T) {
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "--quiet", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep rm returned error: %v", err)
		}

		if stdout.String() != "" {
			t.Errorf("stdout = %q, want empty with --quiet", stdout.String())
		}
	})

	t.Run("it errors when fewer than two IDs provided (rm)", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		// No IDs
		err := app.Run([]string{"tick", "dep", "rm"})
		if err == nil {
			t.Fatal("expected error for missing IDs, got nil")
		}
		if !strings.Contains(err.Error(), "Usage") || !strings.Contains(err.Error(), "dep rm") {
			t.Errorf("error = %q, want it to contain usage hint with 'dep rm'", err.Error())
		}

		// One ID only
		app2 := NewApp()
		app2.workDir = dir
		err = app2.Run([]string{"tick", "dep", "rm", "tick-aaa111"})
		if err == nil {
			t.Fatal("expected error for only one ID, got nil")
		}
		if !strings.Contains(err.Error(), "Usage") {
			t.Errorf("error = %q, want it to contain 'Usage'", err.Error())
		}
	})

	t.Run("it persists via atomic write (rm)", func(t *testing.T) {
		content := openTaskWithBlockedByJSONL("tick-aaa111", "tick-bbb222") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
		dir := setupTickDirWithContent(t, content)

		app := NewApp()
		app.workDir = dir
		var stdout strings.Builder
		app.stdout = &stdout

		err := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if err != nil {
			t.Fatalf("dep rm returned error: %v", err)
		}

		// Read the file directly to verify persistence
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		// The first task should no longer have blocked_by with tick-bbb222
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
		// The first task line should not contain "blocked_by" anymore (omitted when empty)
		if strings.Contains(lines[0], `"blocked_by"`) {
			t.Error("persisted first task should not contain blocked_by after rm")
		}
	})
}

func TestDepSubcommandRouting(t *testing.T) {
	t.Run("it errors for dep with no subcommand", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep"})
		if err == nil {
			t.Fatal("expected error for dep with no subcommand, got nil")
		}
		if !strings.Contains(err.Error(), "Usage") {
			t.Errorf("error = %q, want it to contain 'Usage'", err.Error())
		}
	})

	t.Run("it errors for dep with unknown subcommand", func(t *testing.T) {
		dir := setupInitializedTickDir(t)

		app := NewApp()
		app.workDir = dir

		err := app.Run([]string{"tick", "dep", "unknown"})
		if err == nil {
			t.Fatal("expected error for dep unknown subcommand, got nil")
		}
		if !strings.Contains(err.Error(), "unknown") {
			t.Errorf("error = %q, want it to contain 'unknown'", err.Error())
		}
	})
}
