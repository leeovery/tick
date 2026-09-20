package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestAttachedFlagValue(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
	existingTask := task.Task{
		ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
		Priority: 2, Created: now, Updated: now,
	}

	t.Run("it stores a title of exactly two dashes given in the attached form", func(t *testing.T) {
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runUpdate(t, dir, existingTask.ID, "--title=--")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if got := readPersistedTasks(t, tickDir)[0].Title; got != "--" {
			t.Errorf("stored title = %q, want %q", got, "--")
		}
	})

	t.Run("it stores a description of exactly two dashes on create", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		_, stderr, exitCode := runCreate(t, dir, "--description=--", "--", "title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		stored := readPersistedTasks(t, tickDir)[0]
		if stored.Description != "--" {
			t.Errorf("stored description = %q, want %q", stored.Description, "--")
		}
		if stored.Title != "title" {
			t.Errorf("stored title = %q, want %q", stored.Title, "title")
		}
	})

	t.Run("it stores a value that spells a global flag", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		stdout, stderr, exitCode := runCreate(t, dir, "--description=--json", "--", "title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		created := readPersistedTasks(t, tickDir)[0]
		if created.Description != "--json" {
			t.Errorf("stored description = %q, want %q", created.Description, "--json")
		}
		assertNotJSON(t, "create", stdout)

		stdout, stderr, exitCode = runUpdate(t, dir, created.ID, "--title=--json", "--description=--json")
		if exitCode != 0 {
			t.Fatalf("update exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		updated := readPersistedTasks(t, tickDir)[0]
		if updated.Title != "--json" {
			t.Errorf("stored title = %q, want %q", updated.Title, "--json")
		}
		if updated.Description != "--json" {
			t.Errorf("stored description = %q, want %q", updated.Description, "--json")
		}
		assertNotJSON(t, "update", stdout)
	})

	t.Run("it still reports a missing value for a trailing separated flag", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runUpdate(t, dir, existingTask.ID, "--title")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		if stderr != "Error: --title requires a value\n" {
			t.Errorf("stderr = %q, want %q", stderr, "Error: --title requires a value\n")
		}
	})

	t.Run("it keeps a positional containing an equals sign whole", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		_, stderr, exitCode := runCreate(t, dir, "--", "a=b")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if got := readPersistedTasks(t, tickDir)[0].Title; got != "a=b" {
			t.Errorf("stored title = %q, want %q", got, "a=b")
		}

		_, stderr, exitCode = runShow(t, dir, "a=b")
		if exitCode != 1 {
			t.Fatalf("show exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		if stderr != "Error: task 'a=b' not found\n" {
			t.Errorf("stderr = %q, want %q", stderr, "Error: task 'a=b' not found\n")
		}
	})

	t.Run("it rejects an unknown flag given in the attached form", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--stauts=open")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		want := "Error: unknown flag \"--stauts=open\" for \"list\". Run 'tick help list' for usage.\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})

	t.Run("it still rejects an unknown flag following an attached one", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--status=open", "--stauts=x")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		want := "Error: unknown flag \"--stauts=x\" for \"list\". Run 'tick help list' for usage.\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})

	t.Run("it rejects the attached form on a flag that takes no value", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runUpdate(t, dir, existingTask.ID, "--clear-tags=x")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		want := "Error: unknown flag \"--clear-tags=x\" for \"update\". Run 'tick help update' for usage.\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})

	t.Run("it rejects an attached value on a global flag", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--json=x")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		want := "Error: unknown flag \"--json=x\" for \"list\". Run 'tick help list' for usage.\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})

	t.Run("it accepts the attached form on create and update", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		stdout, stderr, exitCode := runTick(t, dir, "create", "--quiet", "--priority=0", "title")
		if exitCode != 0 {
			t.Fatalf("create exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		id := strings.TrimSuffix(stdout, "\n")
		if got := readPersistedTasks(t, tickDir)[0].Priority; got != 0 {
			t.Errorf("stored priority = %d, want 0", got)
		}

		_, stderr, exitCode = runUpdate(t, dir, id, "--tags=a,b")
		if exitCode != 0 {
			t.Fatalf("update exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if got := readPersistedTasks(t, tickDir)[0].Tags; len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Errorf("stored tags = %#v, want [a b]", got)
		}
	})

	t.Run("it accepts the attached form on list, show and migrate", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		stdout, stderr, exitCode := runList(t, dir, "--status=open")
		if exitCode != 0 {
			t.Fatalf("list exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, existingTask.ID) {
			t.Errorf("list stdout = %q, want it to contain %q", stdout, existingTask.ID)
		}

		stdout, stderr, exitCode = runShow(t, dir, existingTask.ID, "--field=title")
		if exitCode != 0 {
			t.Fatalf("show exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if stdout != existingTask.Title+"\n" {
			t.Errorf("show stdout = %q, want %q", stdout, existingTask.Title+"\n")
		}

		migrateDir, _ := setupTickProject(t)
		setupBeadsFixture(t, migrateDir, beadsPendingFixture)
		_, stderr, exitCode = runMigrate(t, migrateDir, "--from=beads", "--dry-run")
		if exitCode != 0 {
			t.Fatalf("migrate exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
	})
}

// assertNotJSON asserts that the named command's output is not a JSON document.
func assertNotJSON(t *testing.T, command, stdout string) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err == nil {
		t.Errorf("%s output parsed as JSON: %q", command, stdout)
	}
}
