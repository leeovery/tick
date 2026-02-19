package cli

import (
	"bytes"
	"strings"
	"testing"
)

// runHelp runs the app with the given args and returns stdout, stderr, and exit code.
func runHelp(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	app := &App{
		Stdout: &stdout,
		Stderr: &stderr,
		Getwd:  func() (string, error) { return t.TempDir(), nil },
	}
	full := append([]string{"tick"}, args...)
	code := app.Run(full)
	return stdout.String(), stderr.String(), code
}

func TestHelp(t *testing.T) {
	t.Run("tick help shows all commands", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		for _, name := range []string{
			"init", "create", "list", "show", "update",
			"start", "done", "cancel", "reopen", "remove",
			"dep", "ready", "blocked", "stats", "rebuild",
			"doctor", "migrate", "help",
		} {
			if !strings.Contains(stdout, name) {
				t.Errorf("stdout missing command %q", name)
			}
		}
	})

	t.Run("tick help shows global flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		for _, flag := range []string{"--quiet", "--verbose", "--toon", "--pretty", "--json"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("stdout missing global flag %q", flag)
			}
		}
	})

	t.Run("tick --help matches tick help", func(t *testing.T) {
		helpOut, _, helpCode := runHelp(t, "help")
		flagOut, _, flagCode := runHelp(t, "--help")
		if helpCode != flagCode {
			t.Errorf("exit codes differ: help=%d, --help=%d", helpCode, flagCode)
		}
		if helpOut != flagOut {
			t.Errorf("--help output differs from help output")
		}
	})

	t.Run("tick -h matches tick help", func(t *testing.T) {
		helpOut, _, helpCode := runHelp(t, "help")
		flagOut, _, flagCode := runHelp(t, "-h")
		if helpCode != flagCode {
			t.Errorf("exit codes differ: help=%d, -h=%d", helpCode, flagCode)
		}
		if helpOut != flagOut {
			t.Errorf("-h output differs from help output")
		}
	})

	t.Run("tick with no args shows help", func(t *testing.T) {
		stdout, _, code := runHelp(t)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "Commands:") {
			t.Error("no-args output missing 'Commands:'")
		}
	})

	t.Run("tick help create shows flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "create")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		for _, flag := range []string{"--priority", "--description", "--parent", "--blocked-by", "--blocks"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("stdout missing flag %q", flag)
			}
		}
	})

	t.Run("tick help list shows flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "list")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		for _, flag := range []string{"--status", "--priority", "--ready", "--blocked", "--parent"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("stdout missing flag %q", flag)
			}
		}
	})

	t.Run("tick help migrate shows flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "migrate")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		for _, flag := range []string{"--from", "--dry-run", "--pending-only"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("stdout missing flag %q", flag)
			}
		}
	})

	t.Run("tick help dep shows subcommands", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "dep")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "add") {
			t.Error("stdout missing 'add'")
		}
		if !strings.Contains(stdout, "rm") {
			t.Error("stdout missing 'rm'")
		}
	})

	t.Run("tick help unknown errors", func(t *testing.T) {
		_, stderr, code := runHelp(t, "help", "bogus")
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(stderr, "Unknown command") {
			t.Errorf("stderr = %q, want 'Unknown command'", stderr)
		}
	})

	t.Run("tick create --help shows create help", func(t *testing.T) {
		helpOut, _, _ := runHelp(t, "help", "create")
		flagOut, _, code := runHelp(t, "create", "--help")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if helpOut != flagOut {
			t.Error("create --help output differs from help create output")
		}
	})

	t.Run("tick create -h shows create help", func(t *testing.T) {
		helpOut, _, _ := runHelp(t, "help", "create")
		flagOut, _, code := runHelp(t, "create", "-h")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if helpOut != flagOut {
			t.Error("create -h output differs from help create output")
		}
	})

	t.Run("help exists for every registered command", func(t *testing.T) {
		for _, cmd := range commands {
			t.Run(cmd.Name, func(t *testing.T) {
				stdout, _, code := runHelp(t, "help", cmd.Name)
				if code != 0 {
					t.Fatalf("exit code = %d, want 0", code)
				}
				if stdout == "" {
					t.Error("stdout is empty")
				}
			})
		}
	})

	t.Run("tick help --all shows all commands with flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "--all")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		// Every command's usage line should appear.
		for _, cmd := range commands {
			if !strings.Contains(stdout, cmd.Usage) {
				t.Errorf("--all output missing usage for %q", cmd.Name)
			}
		}
		// Spot-check flags from different commands appear.
		for _, flag := range []string{"--priority", "--status", "--from", "--dry-run"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("--all output missing flag %q", flag)
			}
		}
	})

	t.Run("tick help --all includes global flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "--all")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		for _, flag := range []string{"--quiet", "--verbose", "--toon", "--pretty", "--json", "--help"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("--all output missing global flag %q", flag)
			}
		}
	})

	t.Run("tick help --all is more compact than concatenated per-command help", func(t *testing.T) {
		allOut, _, _ := runHelp(t, "help", "--all")
		// --all should not contain the verbose "Usage:" prefix per command
		// that printCommandHelp uses, instead it uses the bare usage line.
		if strings.Contains(allOut, "Usage: tick") {
			// The top-level "Usage:" header should not appear in --all output.
			t.Error("--all should use compact format without 'Usage:' prefix")
		}
	})

	t.Run("help output goes to stdout not stderr", func(t *testing.T) {
		_, stderr, code := runHelp(t, "help")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if stderr != "" {
			t.Errorf("stderr should be empty, got %q", stderr)
		}
	})

	t.Run("tick help remove shows flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "remove")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "--force") {
			t.Error("stdout missing --force flag")
		}
		if !strings.Contains(stdout, "-f") {
			t.Error("stdout missing -f short flag")
		}
	})

	t.Run("tick help remove mentions cascade", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "remove")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "descendant") {
			t.Error("stdout missing cascade/descendants mention")
		}
	})

	t.Run("tick help remove mentions git recovery", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "remove")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "Git") {
			t.Error("stdout missing Git recovery mention")
		}
	})
}
