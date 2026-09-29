package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

const oldLineCeiling = 65536

// escapeInflated repeats '<', which the store's JSON encoding writes as a
// 6-byte escape, so the text outruns its character count sixfold.
func escapeInflated(chars int) string {
	return strings.Repeat("<", chars)
}

func storedLineLen(t *testing.T, dir, id string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
	if err != nil {
		t.Fatalf("read tasks.jsonl: %v", err)
	}
	prefix := `{"id":"` + id + `"`
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return len(line)
		}
	}
	t.Fatalf("tasks.jsonl holds no line for %s", id)
	return 0
}

func assertLineAtLeastOldCeiling(t *testing.T, dir, id string) {
	t.Helper()
	if got := storedLineLen(t, dir, id); got < oldLineCeiling {
		t.Fatalf("stored line for %s is %d bytes, want at least %d", id, got, oldLineCeiling)
	}
}

func assertStoreOperable(t *testing.T, dir, id string, wantIDs ...string) {
	t.Helper()
	assertListedIDs(t, dir, wantIDs...)

	shown := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
	assertToonFields(t, shown, map[string]any{"id": id})

	updated := decodeToonDoc(t, runToonCommand(t, dir, "update", id, "--title", "Retitled"))
	assertToonFields(t, updated, map[string]any{"id": id, "title": "Retitled"})

	if got := runToonCommand(t, dir, "rebuild"); !strings.HasPrefix(got, "Cache rebuilt: ") {
		t.Errorf("rebuild output = %q, want a Cache rebuilt report", got)
	}
	assertListedIDs(t, dir, wantIDs...)
}

func TestWritesCrossingOldCeiling(t *testing.T) {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	seed := []task.Task{
		{ID: "tick-aaa111", Title: "Grows", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		{ID: "tick-bbb222", Title: "Neighbour", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
	}

	t.Run("it reports an update whose description takes the line past the old ceiling", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)
		if got := storedLineLen(t, dir, "tick-aaa111"); got >= oldLineCeiling {
			t.Fatalf("seed line is %d bytes, want under %d", got, oldLineCeiling)
		}
		description := escapeInflated(11_000)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "update", "tick-aaa111", "--description", description))

		assertToonFields(t, doc, map[string]any{"id": "tick-aaa111", "description": description})
		assertLineAtLeastOldCeiling(t, dir, "tick-aaa111")
		assertStoreOperable(t, dir, "tick-aaa111", "tick-aaa111", "tick-bbb222")
	})

	t.Run("it reports a create whose description makes the line past the old ceiling", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)
		description := escapeInflated(11_000)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "create", "Big", "--description", description))

		id, _ := doc["id"].(string)
		assertToonFields(t, doc, map[string]any{"title": "Big", "description": description})
		assertLineAtLeastOldCeiling(t, dir, id)
		assertStoreOperable(t, dir, id, "tick-aaa111", "tick-bbb222", id)
	})

	t.Run("it reports the note add whose note takes the line past the old ceiling", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)
		const maxAdds = 20

		var added []string
		for i := range maxAdds {
			if got := storedLineLen(t, dir, "tick-aaa111"); got >= oldLineCeiling {
				t.Fatalf("line reached %d bytes before add %d", got, i+1)
			}
			text := escapeInflated(1999) + string(rune('a'+i))
			added = append(added, text)

			doc := decodeToonDoc(t, runToonCommand(t, dir, "note", "add", "tick-aaa111", text))

			if storedLineLen(t, dir, "tick-aaa111") < oldLineCeiling {
				continue
			}
			assertToonFields(t, doc, map[string]any{"id": "tick-aaa111"})
			rows := toonRows(t, doc, "notes")
			if len(rows) != len(added) {
				t.Fatalf("notes length = %d, want %d", len(rows), len(added))
			}
			for j, row := range rows {
				assertToonFields(t, row, map[string]any{"index": float64(j + 1), "text": added[j]})
			}
			assertStoreOperable(t, dir, "tick-aaa111", "tick-aaa111", "tick-bbb222")
			return
		}
		t.Fatalf("%d note adds never took the line to %d bytes", maxAdds, oldLineCeiling)
	})
}
