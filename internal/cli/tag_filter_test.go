package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestTagFilter(t *testing.T) {
	now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

	t.Run("it filters list by single tag", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI task", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-api111", Title: "API task", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-notag1", Title: "No tag task", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("task tagged ui should appear with --tag ui")
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("task tagged api should not appear with --tag ui")
		}
		if strings.Contains(stdout, "tick-notag1") {
			t.Error("untagged task should not appear with --tag ui")
		}
	})

	t.Run("it filters list by AND (comma-separated tags)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-both11", Title: "Both tags", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui", "backend"}, Created: now, Updated: now},
			{ID: "tick-uionly", Title: "UI only", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-beonly", Title: "Backend only", Status: task.StatusOpen, Priority: 2, Tags: []string{"backend"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui,backend")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-both11") {
			t.Error("task with both tags should appear with --tag ui,backend")
		}
		if strings.Contains(stdout, "tick-uionly") {
			t.Error("task with only ui should not appear with --tag ui,backend (AND)")
		}
		if strings.Contains(stdout, "tick-beonly") {
			t.Error("task with only backend should not appear with --tag ui,backend (AND)")
		}
	})

	t.Run("it filters list by OR (multiple --tag flags)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI task", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-api111", Title: "API task", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-other1", Title: "Other task", Status: task.StatusOpen, Priority: 2, Tags: []string{"docs"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui", "--tag", "api")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("task tagged ui should appear with --tag ui --tag api (OR)")
		}
		if !strings.Contains(stdout, "tick-api111") {
			t.Error("task tagged api should appear with --tag ui --tag api (OR)")
		}
		if strings.Contains(stdout, "tick-other1") {
			t.Error("task tagged docs should not appear with --tag ui --tag api")
		}
	})

	t.Run("it filters list by AND/OR composition (--tag ui,backend --tag api)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-both11", Title: "Both ui+backend", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui", "backend"}, Created: now, Updated: now},
			{ID: "tick-uionly", Title: "UI only", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-api111", Title: "API task", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-docs11", Title: "Docs task", Status: task.StatusOpen, Priority: 2, Tags: []string{"docs"}, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui,backend", "--tag", "api")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-both11") {
			t.Error("task with both ui+backend should appear (matches AND group)")
		}
		if strings.Contains(stdout, "tick-uionly") {
			t.Error("task with only ui should not appear (does not satisfy AND group)")
		}
		if !strings.Contains(stdout, "tick-api111") {
			t.Error("task with api should appear (matches OR group)")
		}
		if strings.Contains(stdout, "tick-docs11") {
			t.Error("task with docs should not appear (matches neither group)")
		}
	})

	t.Run("it returns empty list when no tasks match tag filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI task", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "api")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		expected := "No tasks found.\n"
		if stdout != expected {
			t.Errorf("stdout = %q, want %q", stdout, expected)
		}
	})

	t.Run("it rejects invalid kebab-case tag in filter", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, exitCode := runList(t, dir, "--tag", "not valid")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}

		if !strings.Contains(stderr, "kebab-case") {
			t.Errorf("stderr = %q, want to contain 'kebab-case'", stderr)
		}
	})

	t.Run("it normalizes tag filter input to lowercase", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI task", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-api111", Title: "API task", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "UI")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("task tagged ui should appear with --tag UI (normalized)")
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("task tagged api should not appear with --tag UI")
		}
	})

	t.Run("it filters ready tasks by tag", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI ready", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-api111", Title: "API ready", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runReady(t, dir, "--tag", "ui")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("ready task tagged ui should appear with --tag ui")
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("ready task tagged api should not appear with --tag ui")
		}
	})

	t.Run("it filters blocked tasks by tag", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-blk000", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ui1111", Title: "UI blocked", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, BlockedBy: []string{"tick-blk000"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-api111", Title: "API blocked", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, BlockedBy: []string{"tick-blk000"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runBlocked(t, dir, "--tag", "ui")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("blocked task tagged ui should appear with --tag ui")
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("blocked task tagged api should not appear with --tag ui")
		}
	})

	t.Run("it combines --tag with --status filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI open", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-ui2222", Title: "UI in progress", Status: task.StatusInProgress, Priority: 2, Tags: []string{"ui"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-api111", Title: "API open", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui", "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("open ui task should appear with --tag ui --status open")
		}
		if strings.Contains(stdout, "tick-ui2222") {
			t.Error("in_progress ui task should not appear with --status open")
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("api task should not appear with --tag ui")
		}
	})

	t.Run("it combines --tag with --priority filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI p1", Status: task.StatusOpen, Priority: 1, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-ui2222", Title: "UI p2", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-api111", Title: "API p1", Status: task.StatusOpen, Priority: 1, Tags: []string{"api"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui", "--priority", "1")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("ui priority 1 task should appear with --tag ui --priority 1")
		}
		if strings.Contains(stdout, "tick-ui2222") {
			t.Error("ui priority 2 task should not appear with --priority 1")
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("api task should not appear with --tag ui")
		}
	})

	t.Run("it combines --tag with --parent filter", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent", Title: "Parent task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ui1111", Title: "UI child", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Parent: "tick-parent", Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-api111", Title: "API child", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Parent: "tick-parent", Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-ui2222", Title: "UI non-child", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui", "--parent", "tick-parent")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("ui child should appear with --tag ui --parent tick-parent")
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("api child should not appear with --tag ui")
		}
		if strings.Contains(stdout, "tick-ui2222") {
			t.Error("ui non-child should not appear with --parent tick-parent")
		}
	})

	t.Run("it combines --tag with --count flag", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI 1", Status: task.StatusOpen, Priority: 1, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-ui2222", Title: "UI 2", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-ui3333", Title: "UI 3", Status: task.StatusOpen, Priority: 3, Tags: []string{"ui"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
			{ID: "tick-api111", Title: "API task", Status: task.StatusOpen, Priority: 1, Tags: []string{"api"}, Created: now.Add(3 * time.Second), Updated: now.Add(3 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir, "--tag", "ui", "--count", "2")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		// header + 2 data rows
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 tasks), got %d: %q", len(lines), stdout)
		}
		if !strings.HasPrefix(lines[1], "tick-ui1111") {
			t.Errorf("row 1 should start with tick-ui1111, got %q", lines[1])
		}
		if !strings.HasPrefix(lines[2], "tick-ui2222") {
			t.Errorf("row 2 should start with tick-ui2222, got %q", lines[2])
		}
		if strings.Contains(stdout, "tick-api111") {
			t.Error("api task should not appear with --tag ui")
		}
	})

	t.Run("it returns all tasks when --tag not specified", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-ui1111", Title: "UI task", Status: task.StatusOpen, Priority: 2, Tags: []string{"ui"}, Created: now, Updated: now},
			{ID: "tick-api111", Title: "API task", Status: task.StatusOpen, Priority: 2, Tags: []string{"api"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			{ID: "tick-notag1", Title: "No tag task", Status: task.StatusOpen, Priority: 2, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runList(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if !strings.Contains(stdout, "tick-ui1111") {
			t.Error("ui task should appear with no --tag filter")
		}
		if !strings.Contains(stdout, "tick-api111") {
			t.Error("api task should appear with no --tag filter")
		}
		if !strings.Contains(stdout, "tick-notag1") {
			t.Error("untagged task should appear with no --tag filter")
		}
	})
}
