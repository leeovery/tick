package cli

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// runTick runs the full tick argument vector (including the program name) and
// returns stdout, stderr, and exit code.
func runTick(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	code := app.Run(append([]string{"tick"}, args...))
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestEndOfFlagsMarker(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	existingTask := task.Task{
		ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
		Priority: 2, Created: now, Updated: now,
	}

	t.Run("it accepts a dash-leading title after the marker", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		_, stderr, exitCode := runCreate(t, dir, "--", "- title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "- title" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "- title")
		}
	})

	t.Run("it accepts dash-leading note text after the marker", func(t *testing.T) {
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runNote(t, dir, "add", existingTask.ID, "--", "- text")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks[0].Notes) != 1 {
			t.Fatalf("expected 1 note, got %d", len(tasks[0].Notes))
		}
		if tasks[0].Notes[0].Text != "- text" {
			t.Errorf("note text = %q, want %q", tasks[0].Notes[0].Text, "- text")
		}
	})

	t.Run("it accepts the marker on a command with no flags", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		want, _, wantCode := runShow(t, dir, existingTask.ID)
		got, stderr, exitCode := runShow(t, dir, existingTask.ID, "--")
		if exitCode != wantCode {
			t.Fatalf("exit code = %d, want %d; stderr = %q", exitCode, wantCode, stderr)
		}
		if got != want {
			t.Errorf("output with marker = %q, want %q", got, want)
		}
	})

	t.Run("it accepts the marker on doctor", func(t *testing.T) {
		dir, _ := setupDoctorProject(t)

		wantOut, _, wantCode := runDoctor(t, dir)
		gotOut, stderr, exitCode := runDoctor(t, dir, "--", "--bogus")
		if exitCode != wantCode {
			t.Fatalf("exit code = %d, want %d; stderr = %q", exitCode, wantCode, stderr)
		}
		if gotOut != wantOut {
			t.Errorf("output with marker = %q, want %q", gotOut, wantOut)
		}
		if strings.Contains(stderr, "unknown flag") {
			t.Errorf("stderr should not report an unknown flag, got %q", stderr)
		}
	})

	t.Run("it accepts the marker on migrate", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, stderr, _ := runMigrate(t, dir, "--from", "beads", "--", "--bogus")
		if strings.Contains(stderr, "unknown flag") || strings.Contains(stdout, "unknown flag") {
			t.Errorf("should not report an unknown flag, stdout = %q stderr = %q", stdout, stderr)
		}
	})

	t.Run("it treats a post-marker global flag as text", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		_, stderr, exitCode := runCreate(t, dir, "--", "--json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "--json" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "--json")
		}
	})

	t.Run("it applies a global flag placed before the marker", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, stderr, exitCode := runTick(t, dir, "--json", "create", "--", "- title")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		var doc map[string]any
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("stdout is not JSON: %v (stdout = %q)", err, stdout)
		}
		if doc["title"] != "- title" {
			t.Errorf("title field = %v, want %q", doc["title"], "- title")
		}
	})

	t.Run("it drops the marker from the arguments", func(t *testing.T) {
		dir, tickDir := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runNote(t, dir, "add", existingTask.ID, "--", "text")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks[0].Notes) != 1 {
			t.Fatalf("expected 1 note, got %d", len(tasks[0].Notes))
		}
		if tasks[0].Notes[0].Text != "text" {
			t.Errorf("note text = %q, want %q", tasks[0].Notes[0].Text, "text")
		}
	})

	t.Run("it treats a second marker as text", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		_, stderr, exitCode := runCreate(t, dir, "--", "--")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "--" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "--")
		}
	})

	t.Run("it accepts a marker with nothing after it", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		want, _, wantCode := runList(t, dir)
		got, stderr, exitCode := runList(t, dir, "--")
		if exitCode != wantCode {
			t.Fatalf("exit code = %d, want %d; stderr = %q", exitCode, wantCode, stderr)
		}
		if got != want {
			t.Errorf("output with marker = %q, want %q", got, want)
		}
	})

	t.Run("it resolves the subcommand after a leading marker", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		_, stderr, exitCode := runTick(t, dir, "--", "create", "x")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "x" {
			t.Errorf("title = %q, want %q", tasks[0].Title, "x")
		}
	})

	t.Run("it rejects an unknown flag before the marker", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

		_, stderr, exitCode := runList(t, dir, "--stauts", "open", "--", "x")
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stderr, `unknown flag "--stauts" for "list"`) {
			t.Errorf("stderr = %q, want it to report the unknown flag", stderr)
		}
	})

	t.Run("it accepts the marker on every registered command", func(t *testing.T) {
		for command := range commandFlags {
			t.Run(command, func(t *testing.T) {
				dir, _ := setupTickProjectWithTasks(t, []task.Task{existingTask})

				_, stderr, _ := runTick(t, dir, append(strings.Split(command, " "), "--")...)
				if strings.Contains(stderr, `unknown flag "--"`) {
					t.Errorf("stderr reports the marker as an unknown flag: %q", stderr)
				}
			})
		}
	})

	t.Run("it registers the marker as neither a command flag nor a global flag", func(t *testing.T) {
		for command, flags := range commandFlags {
			if _, ok := flags[endOfFlagsMarker]; ok {
				t.Errorf("commandFlags[%q] contains %q", command, endOfFlagsMarker)
			}
		}
		if globalFlagSet[endOfFlagsMarker] {
			t.Errorf("globalFlagSet contains %q", endOfFlagsMarker)
		}
		for _, cmd := range commands {
			for _, f := range cmd.Flags {
				if f.Name == endOfFlagsMarker {
					t.Errorf("help entry for %q lists %q as a flag", cmd.Name, endOfFlagsMarker)
				}
			}
		}
	})
}

func TestParseArgsLiterals(t *testing.T) {
	t.Run("it counts only post-marker arguments", func(t *testing.T) {
		flags, subcmd, rest, err := parseArgs([]string{"--quiet", "create", "a", "--", "b", "c"})
		if err != nil {
			t.Fatalf("parseArgs returned unexpected error: %v", err)
		}
		if subcmd != "create" {
			t.Errorf("subcmd = %q, want %q", subcmd, "create")
		}
		want := []string{"a", "b", "c"}
		if !slices.Equal(rest, want) {
			t.Errorf("rest = %v, want %v", rest, want)
		}
		if flags.literals != 2 {
			t.Errorf("literals = %d, want 2", flags.literals)
		}
	})

	t.Run("it reports no literals without a marker", func(t *testing.T) {
		flags, subcmd, rest, err := parseArgs([]string{"--quiet", "create", "a", "b"})
		if err != nil {
			t.Fatalf("parseArgs returned unexpected error: %v", err)
		}
		if subcmd != "create" {
			t.Errorf("subcmd = %q, want %q", subcmd, "create")
		}
		want := []string{"a", "b"}
		if !slices.Equal(rest, want) {
			t.Errorf("rest = %v, want %v", rest, want)
		}
		if flags.literals != 0 {
			t.Errorf("literals = %d, want 0", flags.literals)
		}
	})

	t.Run("it does not count the subcommand after a leading marker", func(t *testing.T) {
		flags, subcmd, rest, err := parseArgs([]string{"--", "create", "a"})
		if err != nil {
			t.Fatalf("parseArgs returned unexpected error: %v", err)
		}
		if subcmd != "create" {
			t.Errorf("subcmd = %q, want %q", subcmd, "create")
		}
		if !slices.Equal(rest, []string{"a"}) {
			t.Errorf("rest = %v, want [a]", rest)
		}
		if flags.literals != 1 {
			t.Errorf("literals = %d, want 1", flags.literals)
		}
	})

	t.Run("it rejects a flag-shaped subcommand after the marker", func(t *testing.T) {
		_, _, _, err := parseArgs([]string{"--", "--bogus"})
		if err == nil {
			t.Fatal("parseArgs should reject a flag-shaped subcommand")
		}
		if !strings.Contains(err.Error(), `unknown flag "--bogus"`) {
			t.Errorf("error = %q, want it to name the unknown flag", err)
		}
	})
}

func TestSplitLiteralArgs(t *testing.T) {
	t.Run("it splits the trailing arguments off", func(t *testing.T) {
		flagArgs, literals := splitLiteralArgs([]string{"a", "b", "c"}, 2)
		if !slices.Equal(flagArgs, []string{"a"}) {
			t.Errorf("flagArgs = %v, want [a]", flagArgs)
		}
		if !slices.Equal(literals, []string{"b", "c"}) {
			t.Errorf("literals = %v, want [b c]", literals)
		}
	})

	t.Run("it returns all arguments as flags when the count is zero", func(t *testing.T) {
		flagArgs, literals := splitLiteralArgs([]string{"a", "b"}, 0)
		if !slices.Equal(flagArgs, []string{"a", "b"}) {
			t.Errorf("flagArgs = %v, want [a b]", flagArgs)
		}
		if len(literals) != 0 {
			t.Errorf("literals = %v, want empty", literals)
		}
	})

	t.Run("it clamps a literal count larger than the slice", func(t *testing.T) {
		flagArgs, literals := splitLiteralArgs([]string{"x"}, 3)
		if len(flagArgs) != 0 {
			t.Errorf("flagArgs = %v, want empty", flagArgs)
		}
		if !slices.Equal(literals, []string{"x"}) {
			t.Errorf("literals = %v, want [x]", literals)
		}
	})
}

func TestFormatConfigSplitLiterals(t *testing.T) {
	t.Run("it splits using the literal count carried on the config", func(t *testing.T) {
		fc, err := NewFormatConfig(globalFlags{literals: 2}, false)
		if err != nil {
			t.Fatalf("NewFormatConfig returned unexpected error: %v", err)
		}
		if fc.Literals != 2 {
			t.Fatalf("fc.Literals = %d, want 2", fc.Literals)
		}
		flagArgs, literals := fc.SplitLiterals([]string{"a", "b", "c"})
		if !slices.Equal(flagArgs, []string{"a"}) {
			t.Errorf("flagArgs = %v, want [a]", flagArgs)
		}
		if !slices.Equal(literals, []string{"b", "c"}) {
			t.Errorf("literals = %v, want [b c]", literals)
		}
	})
}
