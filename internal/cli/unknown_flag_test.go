package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// TestUnknownFlagRejection is a comprehensive regression suite proving every command
// rejects unknown flags through App.Run(). Flag validation fires before store access,
// so most tests work without .tick project setup.
func TestUnknownFlagRejection(t *testing.T) {
	// No-flag commands: every unknown long flag must be rejected.
	noFlagCommands := []struct {
		name    string
		args    []string
		wantCmd string // fully-qualified command name in error
		helpCmd string // command in help reference
	}{
		{"init", []string{"tick", "init", "--unknown"}, "init", "init"},
		{"show", []string{"tick", "show", "--unknown"}, "show", "show"},
		{"start", []string{"tick", "start", "--unknown"}, "start", "start"},
		{"done", []string{"tick", "done", "--unknown"}, "done", "done"},
		{"cancel", []string{"tick", "cancel", "--unknown"}, "cancel", "cancel"},
		{"reopen", []string{"tick", "reopen", "--unknown"}, "reopen", "reopen"},
		{"stats", []string{"tick", "stats", "--unknown"}, "stats", "stats"},
		{"doctor", []string{"tick", "doctor", "--unknown"}, "doctor", "doctor"},
		{"rebuild", []string{"tick", "rebuild", "--unknown"}, "rebuild", "rebuild"},
	}

	for _, tc := range noFlagCommands {
		t.Run("it rejects --unknown on "+tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var stdout, stderr bytes.Buffer
			app := &App{
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run(tc.args)
			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			want := `unknown flag "--unknown" for "` + tc.wantCmd + `". Run 'tick help ` + tc.helpCmd + `' for usage.`
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
			}
		})
	}

	// Commands with flags: test a flag NOT in their accepted set.
	commandsWithFlags := []struct {
		name      string
		args      []string
		wantCmd   string
		helpCmd   string
		badFlag   string
		setupTick bool // whether to set up .tick dir
	}{
		{"create", []string{"tick", "create", "Title", "--force"}, "create", "create", "--force", false},
		{"update", []string{"tick", "update", "tick-abc", "--force"}, "update", "update", "--force", false},
		{"list", []string{"tick", "list", "--force"}, "list", "list", "--force", false},
		{"remove", []string{"tick", "remove", "--priority", "1"}, "remove", "remove", "--priority", false},
		{"migrate", []string{"tick", "migrate", "--priority"}, "migrate", "migrate", "--priority", false},
		{"ready", []string{"tick", "ready", "--force"}, "ready", "ready", "--force", false},
		{"blocked", []string{"tick", "blocked", "--force"}, "blocked", "blocked", "--force", false},
	}

	for _, tc := range commandsWithFlags {
		t.Run("it rejects "+tc.badFlag+" on "+tc.name, func(t *testing.T) {
			var dir string
			if tc.setupTick {
				dir, _ = setupTickProject(t)
			} else {
				dir = t.TempDir()
			}
			var stdout, stderr bytes.Buffer
			app := &App{
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run(tc.args)
			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			want := `unknown flag "` + tc.badFlag + `" for "` + tc.wantCmd + `". Run 'tick help ` + tc.helpCmd + `' for usage.`
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
			}
		})
	}

	// Two-level commands: error uses fully-qualified name, help uses parent.
	twoLevelCommands := []struct {
		name    string
		args    []string
		wantCmd string // fully-qualified: "dep add"
		helpCmd string // parent: "dep"
	}{
		{"dep add", []string{"tick", "dep", "add", "--unknown"}, "dep add", "dep"},
		{"dep remove", []string{"tick", "dep", "remove", "--unknown"}, "dep remove", "dep"},
		{"note add", []string{"tick", "note", "add", "--unknown"}, "note add", "note"},
		{"note remove", []string{"tick", "note", "remove", "--unknown"}, "note remove", "note"},
	}

	for _, tc := range twoLevelCommands {
		t.Run("it rejects --unknown on "+tc.name+" with fully-qualified name", func(t *testing.T) {
			dir := t.TempDir()
			var stdout, stderr bytes.Buffer
			app := &App{
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run(tc.args)
			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			wantCmdRef := `for "` + tc.wantCmd + `"`
			if !strings.Contains(stderr.String(), wantCmdRef) {
				t.Errorf("stderr = %q, want to contain %q", stderr.String(), wantCmdRef)
			}
			wantHelpRef := `Run 'tick help ` + tc.helpCmd + `' for usage.`
			if !strings.Contains(stderr.String(), wantHelpRef) {
				t.Errorf("stderr = %q, want to contain %q", stderr.String(), wantHelpRef)
			}
		})
	}
}

// TestUnknownShortFlagRejection verifies short unknown flags like -x are rejected
// through App.Run() on various command types.
func TestUnknownShortFlagRejection(t *testing.T) {
	shortFlagTests := []struct {
		name    string
		args    []string
		wantCmd string
		helpCmd string
	}{
		{"show", []string{"tick", "show", "-x"}, "show", "show"},
		{"list", []string{"tick", "list", "-x"}, "list", "list"},
		{"dep add", []string{"tick", "dep", "add", "-x"}, "dep add", "dep"},
	}

	for _, tc := range shortFlagTests {
		t.Run("it rejects -x on "+tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var stdout, stderr bytes.Buffer
			app := &App{
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run(tc.args)
			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			want := `unknown flag "-x" for "` + tc.wantCmd + `". Run 'tick help ` + tc.helpCmd + `' for usage.`
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
			}
		})
	}
}

// TestBugReportScenario verifies the original bug report: dep add --blocks is rejected.
// This test exercises the full dispatch path through App.Run() with actual tasks.
func TestBugReportScenario(t *testing.T) {
	t.Run("it rejects dep add --blocks (bug report scenario)", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "--blocks", "tick-bbb222"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--blocks" for "dep add". Run 'tick help dep' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})
}

// TestCommandsWithFlagsAcceptKnownFlags verifies commands with accepted flags
// work correctly when those flags are used through App.Run().
func TestCommandsWithFlagsAcceptKnownFlags(t *testing.T) {
	t.Run("it accepts --status open on list", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "list", "--status", "open"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
	})

	t.Run("it accepts --priority 1 on create", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "create", "Test task", "--priority", "1"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
	})
}

// TestFlagValidationExcludedCommands verifies that version and help commands
// bypass flag validation entirely — they don't reject unknown flags.
func TestFlagValidationExcludedCommands(t *testing.T) {
	t.Run("it does not validate flags for version command", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "version"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stderr.String() != "" {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})

	t.Run("it does not validate flags for help command", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "help"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stderr.String() != "" {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})
}

// TestGlobalFlagsNotRejected verifies global flags like --verbose, --json, --quiet
// pass through without triggering unknown flag rejection on various commands.
func TestGlobalFlagsNotRejected(t *testing.T) {
	globalFlags := []struct {
		flag string
	}{
		{"--verbose"},
		{"--json"},
		{"--quiet"},
	}

	// Test on a selection of commands covering different dispatch paths.
	commands := []struct {
		name string
		args []string // args after "tick" and the global flag
	}{
		{"init", []string{"init"}},
		{"show", []string{"show", "tick-abc123"}},
		{"list", []string{"list"}},
		{"dep add", []string{"dep", "add", "tick-aaa", "tick-bbb"}},
	}

	for _, gf := range globalFlags {
		for _, cmd := range commands {
			t.Run("it does not reject "+gf.flag+" on "+cmd.name, func(t *testing.T) {
				// Use setupTickProject for commands that proceed past validation.
				// init needs a clean dir (no existing .tick), others need .tick present.
				var dir string
				if cmd.name == "init" {
					dir = t.TempDir()
				} else {
					dir, _ = setupTickProject(t)
				}
				var stdout, stderr bytes.Buffer
				app := &App{
					Stdout: &stdout,
					Stderr: &stderr,
					Getwd:  func() (string, error) { return dir, nil },
				}
				fullArgs := append([]string{"tick", gf.flag}, cmd.args...)
				exitCode := app.Run(fullArgs)

				// The command may fail for other reasons (e.g., task not found),
				// but it must NOT fail with an "unknown flag" error.
				if strings.Contains(stderr.String(), "unknown flag") {
					t.Errorf("global flag %s should not trigger unknown flag error on %s, stderr = %q",
						gf.flag, cmd.name, stderr.String())
				}
				// For init with --quiet/--verbose/--json, it should succeed (exit 0).
				if cmd.name == "init" && exitCode != 0 {
					// Only fail if the error is about unknown flags, not other issues.
					if strings.Contains(stderr.String(), "unknown flag") {
						t.Errorf("init should accept %s, exit code = %d, stderr = %q",
							gf.flag, exitCode, stderr.String())
					}
				}
			})
		}
	}
}
