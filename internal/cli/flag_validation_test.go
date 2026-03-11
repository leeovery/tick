package cli

import (
	"strings"
	"testing"
)

func TestFlagValidationAllCommands(t *testing.T) {
	type commandTestCase struct {
		command   string
		validArgs []string // args containing all valid flags with values
		flagCount int      // expected number of flag entries in commandFlags
	}

	commandsWithFlags := []commandTestCase{
		{
			command: "create",
			validArgs: []string{
				"My Task",
				"--priority", "1",
				"--description", "desc",
				"--blocked-by", "tick-aaa111",
				"--blocks", "tick-bbb222",
				"--parent", "tick-ccc333",
				"--type", "bug",
				"--tags", "frontend,backend",
				"--refs", "https://example.com",
			},
			flagCount: 8,
		},
		{
			command: "update",
			validArgs: []string{
				"tick-abc123",
				"--title", "New Title",
				"--description", "new desc",
				"--priority", "3",
				"--parent", "tick-aaa111",
				"--clear-description",
				"--type", "feature",
				"--clear-type",
				"--tags", "api",
				"--clear-tags",
				"--refs", "https://example.com",
				"--clear-refs",
				"--blocks", "tick-bbb222",
			},
			flagCount: 12,
		},
		{
			command: "list",
			validArgs: []string{
				"--ready",
				"--blocked",
				"--status", "open",
				"--priority", "1",
				"--parent", "tick-aaa111",
				"--type", "bug",
				"--tag", "frontend",
				"--count", "10",
			},
			flagCount: 8,
		},
		{
			command: "ready",
			validArgs: []string{
				"--blocked",
				"--status", "open",
				"--priority", "1",
				"--parent", "tick-aaa111",
				"--type", "bug",
				"--tag", "frontend",
				"--count", "10",
			},
			flagCount: 7,
		},
		{
			command: "blocked",
			validArgs: []string{
				"--ready",
				"--status", "open",
				"--priority", "1",
				"--parent", "tick-aaa111",
				"--type", "bug",
				"--tag", "frontend",
				"--count", "10",
			},
			flagCount: 7,
		},
		{
			command: "remove",
			validArgs: []string{
				"--force",
				"tick-abc123",
			},
			flagCount: 2,
		},
		{
			command: "migrate",
			validArgs: []string{
				"--from", "beads",
				"--dry-run",
				"--pending-only",
			},
			flagCount: 3,
		},
	}

	for _, tc := range commandsWithFlags {
		t.Run("it accepts all valid flags for "+tc.command, func(t *testing.T) {
			err := ValidateFlags(tc.command, tc.validArgs, commandFlags)
			if err != nil {
				t.Errorf("expected nil for all valid flags on %s, got %v", tc.command, err)
			}
		})

		t.Run("it has correct flag count for "+tc.command, func(t *testing.T) {
			got := len(commandFlags[tc.command])
			if got != tc.flagCount {
				t.Errorf("commandFlags[%q] has %d flags, want %d", tc.command, got, tc.flagCount)
			}
		})

		t.Run("it rejects unknown flag for "+tc.command, func(t *testing.T) {
			err := ValidateFlags(tc.command, []string{"--unknown-flag"}, commandFlags)
			if err == nil {
				t.Fatalf("expected error for --unknown-flag on %s, got nil", tc.command)
			}
			want := `unknown flag "--unknown-flag" for "` + tc.command + `"`
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), want)
			}
		})
	}

	// Commands with no flags — every unknown flag must be rejected.
	noFlagCommands := []string{
		"init", "show", "start", "done", "cancel", "reopen",
		"dep add", "dep remove", "note add", "note remove",
		"stats", "doctor", "rebuild",
	}

	for _, cmd := range noFlagCommands {
		t.Run("it rejects unknown flag for "+cmd, func(t *testing.T) {
			err := ValidateFlags(cmd, []string{"--unknown"}, commandFlags)
			if err == nil {
				t.Fatalf("expected error for --unknown on %s, got nil", cmd)
			}
			want := `unknown flag "--unknown" for "` + cmd + `"`
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), want)
			}
		})

		t.Run("it has zero flags for "+cmd, func(t *testing.T) {
			got := len(commandFlags[cmd])
			if got != 0 {
				t.Errorf("commandFlags[%q] has %d flags, want 0", cmd, got)
			}
		})
	}
}

func TestReadyRejectsReady(t *testing.T) {
	t.Run("ready rejects --ready", func(t *testing.T) {
		err := ValidateFlags("ready", []string{"--ready"}, commandFlags)
		if err == nil {
			t.Fatal("expected error for --ready on ready command, got nil")
		}
		want := `unknown flag "--ready" for "ready"`
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
	})
}

func TestBlockedRejectsBlocked(t *testing.T) {
	t.Run("blocked rejects --blocked", func(t *testing.T) {
		err := ValidateFlags("blocked", []string{"--blocked"}, commandFlags)
		if err == nil {
			t.Fatal("expected error for --blocked on blocked command, got nil")
		}
		want := `unknown flag "--blocked" for "blocked"`
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
	})
}

func TestGlobalFlagsAcceptedOnAnyCommand(t *testing.T) {
	globalFlags := []string{"--quiet", "-q", "--verbose", "-v", "--toon", "--pretty", "--json", "--help", "-h"}
	commands := []string{"create", "list", "show", "dep add", "update", "remove", "ready", "blocked", "migrate", "start", "done", "cancel", "reopen", "init", "stats", "doctor", "rebuild", "note add", "note remove", "dep remove"}

	for _, cmd := range commands {
		for _, gf := range globalFlags {
			t.Run("global flag "+gf+" accepted on "+cmd, func(t *testing.T) {
				err := ValidateFlags(cmd, []string{gf}, commandFlags)
				if err != nil {
					t.Errorf("global flag %s should be accepted on %s, got error: %v", gf, cmd, err)
				}
			})
		}
	}
}

func TestGlobalFlagsMixedWithCommandFlags(t *testing.T) {
	t.Run("global flags mixed with command flags on create", func(t *testing.T) {
		args := []string{"--json", "title", "--priority", "1", "--verbose"}
		err := ValidateFlags("create", args, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("global flags mixed with command flags on list", func(t *testing.T) {
		args := []string{"--pretty", "--status", "open", "--tag", "api", "-q"}
		err := ValidateFlags("list", args, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("global flags mixed with command flags on update", func(t *testing.T) {
		args := []string{"--verbose", "tick-abc123", "--title", "New Title", "--json", "--clear-description"}
		err := ValidateFlags("update", args, commandFlags)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestValueTakingFlagSkipping(t *testing.T) {
	t.Run("value that looks like a flag is skipped", func(t *testing.T) {
		// --description takes a value; --not-a-flag should be treated as the value of --description
		err := ValidateFlags("create", []string{"--description", "--not-a-flag"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil (--not-a-flag should be treated as value of --description), got %v", err)
		}
	})

	t.Run("consecutive value-taking flags work correctly", func(t *testing.T) {
		args := []string{"--priority", "1", "--description", "desc", "--type", "bug"}
		err := ValidateFlags("create", args, commandFlags)
		if err != nil {
			t.Errorf("expected nil for consecutive value-taking flags, got %v", err)
		}
	})

	t.Run("value that looks like a known flag is skipped as value", func(t *testing.T) {
		// --description takes a value; even --priority (a known flag) in value position is consumed as value
		err := ValidateFlags("create", []string{"--description", "--priority"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil (--priority consumed as value of --description), got %v", err)
		}
	})

	t.Run("value that looks like a global flag is skipped as value", func(t *testing.T) {
		// --description takes a value; even --json (a global flag) in value position is consumed as value
		err := ValidateFlags("create", []string{"--description", "--json"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil (--json consumed as value of --description), got %v", err)
		}
	})

	t.Run("all value-taking flags on create skip their values", func(t *testing.T) {
		args := []string{
			"--priority", "--fake-value-1",
			"--description", "--fake-value-2",
			"--blocked-by", "--fake-value-3",
			"--blocks", "--fake-value-4",
			"--parent", "--fake-value-5",
			"--type", "--fake-value-6",
			"--tags", "--fake-value-7",
			"--refs", "--fake-value-8",
		}
		err := ValidateFlags("create", args, commandFlags)
		if err != nil {
			t.Errorf("expected nil (all values should be skipped), got %v", err)
		}
	})

	t.Run("all value-taking flags on update skip their values", func(t *testing.T) {
		args := []string{
			"--title", "--fake-1",
			"--description", "--fake-2",
			"--priority", "--fake-3",
			"--parent", "--fake-4",
			"--type", "--fake-5",
			"--tags", "--fake-6",
			"--refs", "--fake-7",
			"--blocks", "--fake-8",
		}
		err := ValidateFlags("update", args, commandFlags)
		if err != nil {
			t.Errorf("expected nil (all values should be skipped), got %v", err)
		}
	})

	t.Run("boolean flags on update do not consume next arg", func(t *testing.T) {
		// --clear-description is boolean; --unknown should be checked as a flag
		err := ValidateFlags("update", []string{"--clear-description", "--unknown"}, commandFlags)
		if err == nil {
			t.Fatal("expected error for --unknown after boolean --clear-description, got nil")
		}
		if !strings.Contains(err.Error(), "--unknown") {
			t.Errorf("error should mention --unknown, got %q", err.Error())
		}
	})
}

func TestRemoveAcceptsShortFlag(t *testing.T) {
	t.Run("remove accepts -f short flag", func(t *testing.T) {
		err := ValidateFlags("remove", []string{"-f", "tick-abc123"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil for -f on remove, got %v", err)
		}
	})

	t.Run("remove accepts --force long flag", func(t *testing.T) {
		err := ValidateFlags("remove", []string{"--force", "tick-abc123"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil for --force on remove, got %v", err)
		}
	})
}
