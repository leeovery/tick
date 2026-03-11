package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateFlags(t *testing.T) {
	t.Run("it returns nil for args with no flags", func(t *testing.T) {
		err := ValidateFlags("create", []string{"My Task Title"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("it returns nil for args with known command flags", func(t *testing.T) {
		err := ValidateFlags("create", []string{"My Task", "--priority", "3", "--description", "desc"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("it returns error for unknown flag", func(t *testing.T) {
		err := ValidateFlags("list", []string{"--unknown"}, commandFlags)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		want := `unknown flag "--unknown" for "list". Run 'tick help list' for usage.`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it returns error for unknown flag on dep add (bug repro)", func(t *testing.T) {
		err := ValidateFlags("dep add", []string{"tick-aaa", "--blocks", "tick-bbb"}, commandFlags)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		want := `unknown flag "--blocks" for "dep add". Run 'tick help dep' for usage.`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it skips global flags without error", func(t *testing.T) {
		globals := []string{"--quiet", "-q", "--verbose", "-v", "--toon", "--pretty", "--json", "--help", "-h"}
		for _, g := range globals {
			err := ValidateFlags("show", []string{g, "tick-abc123"}, commandFlags)
			if err != nil {
				t.Errorf("global flag %s should be accepted, got error: %v", g, err)
			}
		}
	})

	t.Run("it skips value after value-taking flag", func(t *testing.T) {
		// --priority takes a value; 3 should be skipped; --status also takes a value
		err := ValidateFlags("list", []string{"--priority", "3", "--status", "open"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("it does not skip value after boolean flag", func(t *testing.T) {
		// --ready is boolean; --unknown should not be skipped as a value of --ready
		err := ValidateFlags("list", []string{"--ready", "--unknown"}, commandFlags)
		if err == nil {
			t.Fatal("expected error for --unknown after boolean --ready, got nil")
		}
		if !strings.Contains(err.Error(), "--unknown") {
			t.Errorf("error should mention --unknown, got %q", err.Error())
		}
	})

	t.Run("it rejects short unknown flags", func(t *testing.T) {
		err := ValidateFlags("list", []string{"-x"}, commandFlags)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		want := `unknown flag "-x" for "list". Run 'tick help list' for usage.`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it accepts -f on remove", func(t *testing.T) {
		err := ValidateFlags("remove", []string{"-f", "tick-abc123"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil for -f on remove, got %v", err)
		}
	})

	t.Run("it uses parent command in help hint for two-level commands", func(t *testing.T) {
		twoLevel := []struct {
			command    string
			helpParent string
		}{
			{"dep add", "dep"},
			{"dep remove", "dep"},
			{"note add", "note"},
			{"note remove", "note"},
		}
		for _, tc := range twoLevel {
			err := ValidateFlags(tc.command, []string{"--bogus"}, commandFlags)
			if err == nil {
				t.Errorf("%s: expected error, got nil", tc.command)
				continue
			}
			wantHelpRef := "Run 'tick help " + tc.helpParent + "' for usage."
			if !strings.Contains(err.Error(), wantHelpRef) {
				t.Errorf("%s: error = %q, want to contain %q", tc.command, err.Error(), wantHelpRef)
			}
			wantCmdRef := `for "` + tc.command + `"`
			if !strings.Contains(err.Error(), wantCmdRef) {
				t.Errorf("%s: error = %q, want to contain %q", tc.command, err.Error(), wantCmdRef)
			}
		}
	})

	t.Run("it returns error for flag on command with no flags", func(t *testing.T) {
		noFlagCmds := []string{"init", "show", "start", "done", "cancel", "reopen", "dep add", "dep remove", "note add", "note remove", "stats", "doctor", "rebuild"}
		for _, cmd := range noFlagCmds {
			err := ValidateFlags(cmd, []string{"--anything"}, commandFlags)
			if err == nil {
				t.Errorf("%s: expected error for --anything, got nil", cmd)
				continue
			}
			if !strings.Contains(err.Error(), "--anything") {
				t.Errorf("%s: error = %q, want to contain '--anything'", cmd, err.Error())
			}
		}
	})

	t.Run("it handles global flags interspersed with command args", func(t *testing.T) {
		err := ValidateFlags("create", []string{"--verbose", "My Task", "--priority", "2", "--json"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestCommandFlagsCoversAllCommands(t *testing.T) {
	expected := []string{
		"init", "create", "update", "list", "show",
		"start", "done", "cancel", "reopen",
		"ready", "blocked",
		"dep add", "dep remove", "note add", "note remove",
		"remove", "stats", "doctor", "rebuild", "migrate",
	}
	for _, cmd := range expected {
		if _, ok := commandFlags[cmd]; !ok {
			t.Errorf("commandFlags missing entry for %q", cmd)
		}
	}
	if len(commandFlags) != len(expected) {
		t.Errorf("commandFlags has %d entries, want %d", len(commandFlags), len(expected))
	}
}

func TestHelpCommand(t *testing.T) {
	t.Run("it returns command itself for single-word command", func(t *testing.T) {
		got := helpCommand("list")
		if got != "list" {
			t.Errorf("helpCommand(%q) = %q, want %q", "list", got, "list")
		}
	})

	t.Run("it returns parent for two-level command", func(t *testing.T) {
		got := helpCommand("dep add")
		if got != "dep" {
			t.Errorf("helpCommand(%q) = %q, want %q", "dep add", got, "dep")
		}
	})
}

func TestFlagValidationWiring(t *testing.T) {
	t.Run("it rejects unknown flag on dep add via full dispatch", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "dep", "add", "tick-aaa", "--blocks", "tick-bbb"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--blocks" for "dep add". Run 'tick help dep' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})

	t.Run("it rejects unknown flag before subcommand", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--bogus", "list"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--bogus". Run 'tick help' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})

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

	t.Run("it rejects unknown flag on doctor", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "doctor", "--bogus"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--bogus" for "doctor". Run 'tick help doctor' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})

	t.Run("it rejects unknown flag on migrate", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "migrate", "--bogus", "--from", "beads"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--bogus" for "migrate". Run 'tick help migrate' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})

	t.Run("it rejects unknown flag on list", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "list", "--unknown"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--unknown" for "list". Run 'tick help list' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})

	t.Run("it rejects unknown flag on create", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "create", "My Task", "--bogus"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		want := `unknown flag "--bogus" for "create". Run 'tick help create' for usage.`
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	})

	t.Run("it accepts known flags on create through dispatch", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "create", "My Task", "--priority", "3"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		if stderr.String() != "" {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})
}
