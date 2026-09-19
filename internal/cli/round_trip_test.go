package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

// The awkward fixture: free text carrying a leading dash, a comma, embedded
// quotes, blank lines, a header-shaped line and trailing spaces on an interior
// line, spread across all three free-text carriers, plus the tag and ref
// sections the detail document renders alongside them.
const fixtureTitle = `- read the header, carefully`

const fixtureDescription = "Fix the parser.\n\nSteps:\n  - read the header   \n  - validate \"strictly\", then stop\nDone."

const fixtureNoteText = `- retried "twice", then it stuck`

var (
	fixtureTags = []string{"round-trip", "fixture"}
	fixtureRefs = []string{"https://example.com/issues/42"}
)

// setupFixtureTask creates the awkward fixture task and its note, returning the
// project directory, the .tick directory and the new task's ID.
func setupFixtureTask(t *testing.T) (dir string, tickDir string, id string) {
	t.Helper()
	dir, tickDir = setupTickProject(t)

	stdout, stderr, exitCode := runTick(t, dir, "create", "--quiet",
		"--description", fixtureDescription,
		"--tags", strings.Join(fixtureTags, ","),
		"--refs", strings.Join(fixtureRefs, ","),
		"--", fixtureTitle)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, want 0; stderr = %q", exitCode, stderr)
	}
	id = strings.TrimSuffix(stdout, "\n")

	_, stderr, exitCode = runTick(t, dir, "note", "add", id, "--", fixtureNoteText)
	if exitCode != 0 {
		t.Fatalf("note add exit code = %d, want 0; stderr = %q", exitCode, stderr)
	}
	return dir, tickDir, id
}

// storedFixture returns the single persisted task.
func storedFixture(t *testing.T, tickDir string) task.Task {
	t.Helper()
	tasks := readPersistedTasks(t, tickDir)
	if len(tasks) != 1 {
		t.Fatalf("persisted task count = %d, want 1", len(tasks))
	}
	return tasks[0]
}

// toonString returns a decoded document's string value for key.
func toonString(t *testing.T, doc map[string]any, key string) string {
	t.Helper()
	raw, ok := doc[key]
	if !ok {
		t.Fatalf("key %q missing from decoded document", key)
	}
	s, ok := raw.(string)
	if !ok {
		t.Fatalf("key %q = %#v, want a string", key, raw)
	}
	return s
}

func TestAwkwardFixtureRoundTrip(t *testing.T) {
	t.Run("it stores the awkward title byte-identically", func(t *testing.T) {
		_, tickDir, _ := setupFixtureTask(t)

		if got := storedFixture(t, tickDir).Title; got != fixtureTitle {
			t.Errorf("stored title = %q, want %q", got, fixtureTitle)
		}
	})

	t.Run("it stores the awkward description byte-identically", func(t *testing.T) {
		_, tickDir, _ := setupFixtureTask(t)

		if got := storedFixture(t, tickDir).Description; got != fixtureDescription {
			t.Errorf("stored description = %q, want %q", got, fixtureDescription)
		}
	})

	t.Run("it stores the awkward note byte-identically", func(t *testing.T) {
		_, tickDir, _ := setupFixtureTask(t)

		notes := storedFixture(t, tickDir).Notes
		if len(notes) != 1 {
			t.Fatalf("stored note count = %d, want 1", len(notes))
		}
		if notes[0].Text != fixtureNoteText {
			t.Errorf("stored note text = %q, want %q", notes[0].Text, fixtureNoteText)
		}
	})

	t.Run("it stores the fixture tags and refs", func(t *testing.T) {
		_, tickDir, _ := setupFixtureTask(t)

		stored := storedFixture(t, tickDir)
		if !slices.Equal(stored.Tags, fixtureTags) {
			t.Errorf("stored tags = %#v, want %#v", stored.Tags, fixtureTags)
		}
		if !slices.Equal(stored.Refs, fixtureRefs) {
			t.Errorf("stored refs = %#v, want %#v", stored.Refs, fixtureRefs)
		}
	})

	t.Run("it decodes the stored title out of the show document", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		if got := toonString(t, doc, "title"); got != storedFixture(t, tickDir).Title {
			t.Errorf("decoded title = %q, want %q", got, storedFixture(t, tickDir).Title)
		}
	})

	t.Run("it decodes the stored description out of the show document", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		want := storedFixture(t, tickDir).Description
		if got := toonString(t, doc, "description"); got != want {
			t.Errorf("decoded description = %q, want %q", got, want)
		}
	})

	t.Run("it decodes the stored note text out of the show document", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		rows := toonRows(t, doc, "notes")
		if len(rows) != 1 {
			t.Fatalf("decoded note count = %d, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{
			"index": float64(1),
			"text":  storedFixture(t, tickDir).Notes[0].Text,
		})
	})

	t.Run("it decodes the stored tags out of the show document", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		// The detail query orders tags alphabetically.
		assertToonStringList(t, doc, "tags", slices.Sorted(slices.Values(storedFixture(t, tickDir).Tags)))
	})

	t.Run("it decodes the stored refs out of the show document", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		assertToonStringList(t, doc, "refs", storedFixture(t, tickDir).Refs)
	})

	t.Run("it returns the stored title bare from --field", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		assertBareField(t, dir, id, "title", storedFixture(t, tickDir).Title)
	})

	t.Run("it returns the stored description bare from --field", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		assertBareField(t, dir, id, "description", storedFixture(t, tickDir).Description)
	})

	t.Run("it returns the stored note text bare from --field notes.1", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		assertBareField(t, dir, id, "notes.1", storedFixture(t, tickDir).Notes[0].Text)
	})

	t.Run("it writes the decoded title back unchanged", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		decoded := toonString(t, doc, "title")

		_, stderr, exitCode := runTick(t, dir, "update", id, "--title", decoded)
		if exitCode != 0 {
			t.Fatalf("update exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if got := storedFixture(t, tickDir).Title; got != fixtureTitle {
			t.Errorf("stored title after write-back = %q, want %q", got, fixtureTitle)
		}
	})

	t.Run("it writes the decoded description back unchanged", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		decoded := toonString(t, doc, "description")

		_, stderr, exitCode := runTick(t, dir, "update", id, "--description", decoded)
		if exitCode != 0 {
			t.Fatalf("update exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if got := storedFixture(t, tickDir).Description; got != fixtureDescription {
			t.Errorf("stored description after write-back = %q, want %q", got, fixtureDescription)
		}
	})

	t.Run("it writes the decoded note back as a second note with identical text", func(t *testing.T) {
		dir, tickDir, id := setupFixtureTask(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		rows := toonRows(t, doc, "notes")
		decoded, ok := rows[0]["text"].(string)
		if !ok {
			t.Fatalf("notes[0].text = %#v, want a string", rows[0]["text"])
		}

		_, stderr, exitCode := runTick(t, dir, "note", "add", id, "--", decoded)
		if exitCode != 0 {
			t.Fatalf("note add exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		notes := storedFixture(t, tickDir).Notes
		if len(notes) != 2 {
			t.Fatalf("stored note count = %d, want 2", len(notes))
		}
		if notes[1].Text != notes[0].Text {
			t.Errorf("second note text = %q, want %q", notes[1].Text, notes[0].Text)
		}
	})

	t.Run("it trims edge whitespace on the way in and keeps it trimmed", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		const wantTitle = "- padded, title"
		const wantDescription = "padded description"

		stdout, stderr, exitCode := runTick(t, dir, "create", "--quiet",
			"--description", "  \n"+wantDescription+"\n  ",
			"--", "  "+wantTitle+"  ")
		if exitCode != 0 {
			t.Fatalf("create exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		id := strings.TrimSuffix(stdout, "\n")

		stored := storedFixture(t, tickDir)
		if stored.Title != wantTitle {
			t.Errorf("stored title = %q, want %q", stored.Title, wantTitle)
		}
		if stored.Description != wantDescription {
			t.Errorf("stored description = %q, want %q", stored.Description, wantDescription)
		}

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
		_, stderr, exitCode = runTick(t, dir, "update", id,
			"--title", toonString(t, doc, "title"),
			"--description", toonString(t, doc, "description"))
		if exitCode != 0 {
			t.Fatalf("update exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		stored = storedFixture(t, tickDir)
		if stored.Title != wantTitle {
			t.Errorf("stored title after write-back = %q, want %q", stored.Title, wantTitle)
		}
		if stored.Description != wantDescription {
			t.Errorf("stored description after write-back = %q, want %q", stored.Description, wantDescription)
		}
	})
}

// assertBareField asserts that `tick show <id> --field <field>` prints want
// followed by exactly one newline.
func assertBareField(t *testing.T, dir, id, field, want string) {
	t.Helper()
	stdout, stderr, exitCode := runTick(t, dir, "show", id, "--field", field)
	if exitCode != 0 {
		t.Fatalf("show --field %s exit code = %d, want 0; stderr = %q", field, exitCode, stderr)
	}
	if stdout != want+"\n" {
		t.Errorf("show --field %s stdout = %q, want %q", field, stdout, want+"\n")
	}
}
