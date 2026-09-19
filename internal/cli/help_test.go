package cli

import (
	"bytes"
	"fmt"
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
		for _, flag := range []string{"--help", "--quiet", "--verbose", "--toon", "--pretty", "--json", "--version"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("stdout missing global flag %q", flag)
			}
		}
	})

	t.Run("tick --help lists --help and --version global flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "--help")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		for _, flag := range []string{"--help", "--version"} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("--help output missing global flag %q", flag)
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
		if !strings.Contains(stdout, "remove") {
			t.Error("stdout missing 'remove'")
		}
	})

	t.Run("it shows <add|remove|tree> in dep help text", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "dep")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "<add|remove|tree>") {
			t.Errorf("stdout should contain '<add|remove|tree>', got %q", stdout)
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
		for _, flag := range []string{"--quiet", "--verbose", "--toon", "--pretty", "--json", "--help", "--version"} {
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

// helpFlagBlock returns the flag lines of a per-command help output.
func helpFlagBlock(t *testing.T, out string) string {
	t.Helper()
	_, block, found := strings.Cut(out, "Flags:\n")
	if !found {
		t.Fatalf("help output has no Flags section:\n%s", out)
	}
	return block
}

// flagLabels returns the rendered label for each flag, in registry order.
func flagLabels(flags []flagInfo) []string {
	labels := make([]string, len(flags))
	for i, f := range flags {
		labels[i] = f.Name
		if f.Arg != "" {
			labels[i] += " " + f.Arg
		}
	}
	return labels
}

func TestHelpFlagColumn(t *testing.T) {
	t.Run("it separates every flag label from its description", func(t *testing.T) {
		for _, cmd := range commands {
			if len(cmd.Flags) == 0 {
				continue
			}
			stdout, _, code := runHelp(t, "help", cmd.Name)
			if code != 0 {
				t.Fatalf("%s: exit code = %d, want 0", cmd.Name, code)
			}
			lines := strings.Split(strings.TrimSuffix(helpFlagBlock(t, stdout), "\n"), "\n")
			if len(lines) != len(cmd.Flags) {
				t.Fatalf("%s: got %d flag lines, want %d", cmd.Name, len(lines), len(cmd.Flags))
			}
			for i, label := range flagLabels(cmd.Flags) {
				rest, ok := strings.CutPrefix(lines[i], "  "+label)
				if !ok {
					t.Errorf("%s: flag line %q does not start with label %q", cmd.Name, lines[i], label)
					continue
				}
				if !strings.HasPrefix(rest, "  ") {
					t.Errorf("%s: flag %q has no two-space gap before its description: %q", cmd.Name, label, lines[i])
				}
				if strings.TrimLeft(rest, " ") != cmd.Flags[i].Desc {
					t.Errorf("%s: flag %q description = %q, want %q", cmd.Name, label, strings.TrimLeft(rest, " "), cmd.Flags[i].Desc)
				}
			}
		}
	})

	t.Run("it prints the show field flag with its description in its own column", func(t *testing.T) {
		want := "  --field, --fields <name,...>  Select fields by name; a section may be narrowed with .N (e.g. notes.2)\n"
		stdout, _, code := runHelp(t, "help", "show")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, want) {
			t.Errorf("tick help show missing line %q, got:\n%s", want, stdout)
		}
		allOut, _, allCode := runHelp(t, "help", "--all")
		if allCode != 0 {
			t.Fatalf("--all exit code = %d, want 0", allCode)
		}
		if !strings.Contains(allOut, want) {
			t.Errorf("tick help --all missing line %q", want)
		}
	})

	t.Run("it renders the same flag block in --all as in per-command help", func(t *testing.T) {
		allOut, _, code := runHelp(t, "help", "--all")
		if code != 0 {
			t.Fatalf("--all exit code = %d, want 0", code)
		}
		for _, cmd := range commands {
			if len(cmd.Flags) == 0 {
				continue
			}
			stdout, _, cmdCode := runHelp(t, "help", cmd.Name)
			if cmdCode != 0 {
				t.Fatalf("%s: exit code = %d, want 0", cmd.Name, cmdCode)
			}
			block := helpFlagBlock(t, stdout)
			if !strings.Contains(allOut, block) {
				t.Errorf("%s: --all flag block differs from per-command block:\n%s", cmd.Name, block)
			}
		}
	})

	t.Run("it sizes the column to the widest label in the command's flag set", func(t *testing.T) {
		for _, tc := range []struct {
			command string
			width   int
		}{
			{"create", 33},
			{"remove", 13},
		} {
			stdout, _, code := runHelp(t, "help", tc.command)
			if code != 0 {
				t.Fatalf("%s: exit code = %d, want 0", tc.command, code)
			}
			cmd := findCommand(tc.command)
			labels := flagLabels(cmd.Flags)
			for i, f := range cmd.Flags {
				want := fmt.Sprintf("  %-*s%s\n", tc.width, labels[i], f.Desc)
				if !strings.Contains(stdout, want) {
					t.Errorf("%s: missing line %q, got:\n%s", tc.command, want, stdout)
				}
			}
		}
	})

	t.Run("it prints no flags section for a command with no flags", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "init")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if strings.Contains(stdout, "Flags:") {
			t.Errorf("tick help init should print no Flags section, got:\n%s", stdout)
		}
	})
}
