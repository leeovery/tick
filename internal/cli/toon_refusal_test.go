package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

// createRefusedTitleTask creates a task whose title carries a character TOON
// cannot encode and returns its ID.
func createRefusedTitleTask(t *testing.T, dir string) string {
	t.Helper()
	return createCarryingValue(t, dir, "title", refusedTitle)
}

// createRefusedTitleTaskAtPriority creates a task whose title carries a refused
// character at the given priority, which fixes its position in list order.
func createRefusedTitleTaskAtPriority(t *testing.T, dir, priority string) string {
	t.Helper()
	stdout, stderr, exitCode := runToon(t, dir, "create", "--quiet", "--priority", priority, "--", refusedTitle)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, want 0; stderr = %q", exitCode, stderr)
	}
	return strings.TrimSuffix(stdout, "\n")
}

// blockAgainstOrdinaryTask blocks id on a newly created encodable task and
// asserts the dependency was stored, the document confirming it being refused.
func blockAgainstOrdinaryTask(t *testing.T, dir, id string) {
	t.Helper()
	blocker := createCarryingValue(t, dir, "title", "an ordinary blocker")
	runToon(t, dir, "dep", "add", id, blocker)
	stdout, stderr, exitCode := runToon(t, dir, "blocked", "--quiet")
	if exitCode != 0 {
		t.Fatalf("blocked --quiet exit code = %d, want 0; stderr = %q", exitCode, stderr)
	}
	if got := strings.TrimSuffix(stdout, "\n"); got != id {
		t.Fatalf("blocked task IDs = %q, want %q", got, id)
	}
}

func runToon(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	return runTick(t, dir, append([]string{"--toon"}, args...)...)
}

// assertUnnamed asserts that a refusal diagnostic leaves a task it does not
// concern out of its message.
func assertUnnamed(t *testing.T, stderr, id string) {
	t.Helper()
	if strings.Contains(stderr, id) {
		t.Errorf("stderr = %q, want it to leave task %q unnamed", stderr, id)
	}
}

func assertRefused(t *testing.T, stdout, stderr string, exitCode int, wants ...string) {
	t.Helper()
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stdout = %q, stderr = %q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	for _, want := range wants {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to name %q", stderr, want)
		}
	}
}

// refusalEnv is a project holding one task whose title carries a refused character.
type refusalEnv struct {
	dir     string
	tickDir string
	id      string
}

// refusedDocumentCommand is a command that prints a TOON document covering the
// refused task. namedSection is the section its diagnostic names alongside the
// task ID, for a document covering more than one task.
type refusedDocumentCommand struct {
	name         string
	args         func(env refusalEnv) []string
	setup        func(t *testing.T, env refusalEnv)
	namedSection string
}

func refusedDocumentCommands() []refusedDocumentCommand {
	return []refusedDocumentCommand{
		{name: "show", args: func(env refusalEnv) []string { return []string{"show", env.id} }},
		{name: "list", args: func(refusalEnv) []string { return []string{"list"} }, namedSection: "tasks"},
		{name: "dep tree", args: func(env refusalEnv) []string { return []string{"dep", "tree", env.id} }},
		{name: "update", args: func(env refusalEnv) []string { return []string{"update", env.id, "--priority", "1"} }},
		{name: "note add", args: func(env refusalEnv) []string {
			return []string{"note", "add", env.id, "--", "an ordinary note"}
		}},
		{
			name:  "note remove",
			args:  func(env refusalEnv) []string { return []string{"note", "remove", env.id, "1"} },
			setup: addStoredNote,
		},
		{name: "start", args: func(env refusalEnv) []string { return []string{"start", env.id} }},
		{name: "done", args: func(env refusalEnv) []string { return []string{"done", env.id} }},
		{name: "cancel", args: func(env refusalEnv) []string { return []string{"cancel", env.id} }},
		{
			name:  "reopen",
			args:  func(env refusalEnv) []string { return []string{"reopen", env.id} },
			setup: closeTask,
		},
	}
}

// addStoredNote adds a note and asserts it was stored, the document confirming
// it being refused.
func addStoredNote(t *testing.T, env refusalEnv) {
	t.Helper()
	runToon(t, env.dir, "note", "add", env.id, "--", "an ordinary note")
	if got := len(storedFixture(t, env.tickDir).Notes); got != 1 {
		t.Fatalf("stored note count = %d, want 1", got)
	}
}

// closeTask moves the task to done and asserts it got there, the document
// confirming the transition being refused.
func closeTask(t *testing.T, env refusalEnv) {
	t.Helper()
	runToon(t, env.dir, "done", env.id)
	if got := storedFixture(t, env.tickDir).Status; got != task.StatusDone {
		t.Fatalf("stored status = %q, want %q", got, task.StatusDone)
	}
}

// createChildCarryingRefusedTitle creates a child of parentID whose title
// carries a refused character and returns its ID.
func createChildCarryingRefusedTitle(t *testing.T, dir, parentID string) string {
	t.Helper()
	stdout, stderr, exitCode := runToon(t, dir, "create", "--quiet", "--parent", parentID, "--", refusedTitle)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, want 0; stderr = %q", exitCode, stderr)
	}
	return strings.TrimSuffix(stdout, "\n")
}

func TestToonRefusesUnencodableValues(t *testing.T) {
	t.Run("it fails naming the task and the field when the stored title cannot be encoded", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		id := createRefusedTitleTask(t, dir)

		stdout, stderr, exitCode := runToon(t, dir, "show", id)

		assertRefused(t, stdout, stderr, exitCode, id, "title")
	})

	t.Run("it fails rather than printing an empty task list when one task's title cannot be encoded", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		encodable := createCarryingValue(t, dir, "title", "an ordinary title")
		refused := createRefusedTitleTask(t, dir)

		stdout, stderr, exitCode := runToon(t, dir, "list")

		assertRefused(t, stdout, stderr, exitCode, "tasks", refused)
		assertUnnamed(t, stderr, encodable)
	})

	t.Run("it names the first offending task when two tasks in the list carry refused values", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		first := createRefusedTitleTaskAtPriority(t, dir, "1")
		second := createRefusedTitleTaskAtPriority(t, dir, "3")

		stdout, stderr, exitCode := runToon(t, dir, "list")

		assertRefused(t, stdout, stderr, exitCode, "tasks", first)
		assertUnnamed(t, stderr, second)
	})

	for _, command := range []string{"ready", "blocked"} {
		t.Run("it names the offending task for "+command+" as well as list", func(t *testing.T) {
			dir, _ := setupTickProject(t)
			refused := createRefusedTitleTask(t, dir)
			if command == "blocked" {
				blockAgainstOrdinaryTask(t, dir, refused)
			}

			stdout, stderr, exitCode := runToon(t, dir, command)

			assertRefused(t, stdout, stderr, exitCode, "tasks", refused)
		})
	}

	t.Run("it fails rather than printing a count-zero notes section when a note's text cannot be encoded", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		id := createCarryingValue(t, dir, "title", "an ordinary title")
		runToon(t, dir, "note", "add", id, "--", refusedNoteText)

		stdout, stderr, exitCode := runToon(t, dir, "show", id)

		assertRefused(t, stdout, stderr, exitCode, id, "notes")
	})

	t.Run("it fails when the focused dep tree target's title cannot be encoded", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		id := createRefusedTitleTask(t, dir)

		stdout, stderr, exitCode := runToon(t, dir, "dep", "tree", id)

		assertRefused(t, stdout, stderr, exitCode, id, "title")
	})

	t.Run("it writes nothing to stdout when the document is refused", func(t *testing.T) {
		for _, command := range refusedDocumentCommands() {
			t.Run(command.name, func(t *testing.T) {
				dir, tickDir := setupTickProject(t)
				env := refusalEnv{dir: dir, tickDir: tickDir, id: createRefusedTitleTask(t, dir)}
				if command.setup != nil {
					command.setup(t, env)
				}

				stdout, stderr, exitCode := runToon(t, env.dir, command.args(env)...)

				wants := []string{"cannot encode", env.id}
				if command.namedSection != "" {
					wants = append(wants, command.namedSection)
				}
				assertRefused(t, stdout, stderr, exitCode, wants...)
			})
		}
	})

	t.Run("it names the cascaded task carrying the refused value", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		parent := createCarryingValue(t, dir, "title", "an ordinary parent")
		child := createChildCarryingRefusedTitle(t, dir, parent)

		stdout, stderr, exitCode := runToon(t, dir, "done", parent)

		assertRefused(t, stdout, stderr, exitCode, "cannot encode section changed", child)
		assertUnnamed(t, stderr, parent)
	})

	t.Run("it names the child when a child's title cannot be encoded", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		parent := createCarryingValue(t, dir, "title", "an ordinary parent")
		child := createChildCarryingRefusedTitle(t, dir, parent)

		stdout, stderr, exitCode := runToon(t, dir, "show", parent)

		assertRefused(t, stdout, stderr, exitCode, "section children", child)
		assertUnnamed(t, stderr, parent)
	})

	t.Run("it names the blocker when a blocker's title cannot be encoded", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		blocked := createCarryingValue(t, dir, "title", "an ordinary blocked task")
		blocker := createRefusedTitleTask(t, dir)
		runToon(t, dir, "dep", "add", blocked, blocker)

		stdout, stderr, exitCode := runToon(t, dir, "show", blocked)

		assertRefused(t, stdout, stderr, exitCode, "section blocked_by", blocker)
		assertUnnamed(t, stderr, blocked)
	})

	t.Run("it fails on create while storing the task whose title cannot be encoded", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)

		stdout, stderr, exitCode := runToon(t, dir, "create", "--", refusedTitle)

		assertRefused(t, stdout, stderr, exitCode, "title")
		if got := len(readPersistedTasks(t, tickDir)); got != 1 {
			t.Errorf("persisted task count = %d, want 1", got)
		}
	})

	t.Run("it stores the note when the document confirming it is refused", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		id := createCarryingValue(t, dir, "title", "an ordinary title")

		stdout, stderr, exitCode := runToon(t, dir, "note", "add", id, "--", refusedNoteText)

		assertRefused(t, stdout, stderr, exitCode, "notes")
		notes := storedFixture(t, tickDir).Notes
		if len(notes) != 1 {
			t.Fatalf("stored note count = %d, want 1", len(notes))
		}
		if notes[0].Text != refusedNoteText {
			t.Errorf("stored note text = %q, want %q", notes[0].Text, refusedNoteText)
		}
	})

	t.Run("it returns the refused title bare from --field at exit zero", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		id := createRefusedTitleTask(t, dir)

		assertBareField(t, dir, id, "title", storedFixture(t, tickDir).Title)
	})

	t.Run("it returns the refused description intact under --json at exit zero", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		id := createCarryingValue(t, dir, "description", refusedDescription)

		stdout, stderr, exitCode := runTick(t, dir, "--json", "show", id)
		if exitCode != 0 {
			t.Fatalf("show --json exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		var doc struct {
			Description string `json:"description"`
		}
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("decoding %q failed: %v", stdout, err)
		}
		if doc.Description != refusedDescription {
			t.Errorf("json description = %q, want %q", doc.Description, refusedDescription)
		}
	})

	t.Run("it prints only the ID under --quiet for a task whose title cannot be encoded", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		id := createRefusedTitleTask(t, dir)

		for _, args := range [][]string{
			{"show", "--quiet", id},
			{"list", "--quiet"},
			{"update", "--quiet", id, "--priority", "1"},
			{"start", "--quiet", id},
		} {
			stdout, stderr, exitCode := runToon(t, dir, args...)
			if exitCode != 0 {
				t.Fatalf("%v exit code = %d, want 0; stderr = %q", args, exitCode, stderr)
			}
			want := id + "\n"
			if args[0] == "start" {
				want = ""
			}
			if stdout != want {
				t.Errorf("%v stdout = %q, want %q", args, stdout, want)
			}
		}
	})

	t.Run("it prints the refused title under --pretty at exit zero", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		id := createRefusedTitleTask(t, dir)

		stdout, stderr, exitCode := runTick(t, dir, "--pretty", "show", id)
		if exitCode != 0 {
			t.Fatalf("show --pretty exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, refusedTitle) {
			t.Errorf("stdout = %q, want it to carry %q", stdout, refusedTitle)
		}
	})

	for _, command := range [][]string{{"stats"}, {"dep", "tree"}} {
		t.Run("it leaves "+strings.Join(command, " ")+" output unchanged since it carries no free text", func(t *testing.T) {
			plainDir, _ := setupTickProject(t)
			createCarryingValue(t, plainDir, "title", "an ordinary title")
			refusedDir, _ := setupTickProject(t)
			createRefusedTitleTask(t, refusedDir)

			want, stderr, exitCode := runToon(t, plainDir, command...)
			if exitCode != 0 {
				t.Fatalf("%v exit code = %d, want 0; stderr = %q", command, exitCode, stderr)
			}
			got, stderr, exitCode := runToon(t, refusedDir, command...)
			if exitCode != 0 {
				t.Fatalf("%v exit code = %d, want 0; stderr = %q", command, exitCode, stderr)
			}
			if got != want {
				t.Errorf("%v output = %q, want %q", command, got, want)
			}
		})
	}
}
