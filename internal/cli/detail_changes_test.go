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

// toonSectionKeys returns the key of each section of a TOON document in order.
func toonSectionKeys(t *testing.T, doc string) []string {
	t.Helper()
	blocks := strings.Split(doc, "\n\n")
	keys := make([]string, 0, len(blocks))
	for _, block := range blocks {
		first, _, _ := strings.Cut(block, "\n")
		key, _, _ := strings.Cut(first, ":")
		key, _, _ = strings.Cut(key, "[")
		keys = append(keys, key)
	}
	return keys
}

func detailWithChanges(changes *StatusChanges) TaskDetail {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	return TaskDetail{
		Task: task.Task{
			ID:          "tick-aaa111",
			Title:       "Parent task",
			Status:      task.StatusInProgress,
			Priority:    2,
			Description: "Some description",
			Created:     now,
			Updated:     now,
		},
		Notes:   []task.Note{{Text: "A note", Created: now}},
		Changes: changes,
	}
}

func TestTaskDetailChangedSection(t *testing.T) {
	rows := []StatusChange{
		{ID: "tick-aaa111", Title: "Parent task", From: "open", To: "in_progress", Auto: false},
		{ID: "tick-bbb222", Title: "Child task", From: "open", To: "in_progress", Auto: true},
	}

	t.Run("it carries a changed section when the detail carries changes", func(t *testing.T) {
		f := &ToonFormatter{}

		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWithChanges(&StatusChanges{Rows: rows})))

		got := toonRows(t, doc, "changed")
		if len(got) != 2 {
			t.Fatalf("changed has %d rows, want 2", len(got))
		}
		assertToonFields(t, got[0], map[string]any{
			"id": "tick-aaa111", "title": "Parent task", "from": "open", "to": "in_progress", "auto": false,
		})
		assertToonFields(t, got[1], map[string]any{
			"id": "tick-bbb222", "title": "Child task", "from": "open", "to": "in_progress", "auto": true,
		})
	})

	t.Run("it carries a count-zero changed section when the change set is empty", func(t *testing.T) {
		f := &ToonFormatter{}

		rendered := f.FormatTaskDetail(detailWithChanges(&StatusChanges{}))

		if !strings.Contains(rendered, "changed[0]{id,title,from,to,auto}:") {
			t.Fatalf("document does not carry a count-zero changed header:\n%s", rendered)
		}
		if got := toonRows(t, decodeToonDoc(t, rendered), "changed"); len(got) != 0 {
			t.Errorf("changed has %d rows, want 0", len(got))
		}
	})

	t.Run("it omits the changed section entirely when the detail carries no changes", func(t *testing.T) {
		f := &ToonFormatter{}

		doc := decodeToonDoc(t, f.FormatTaskDetail(detailWithChanges(nil)))

		assertToonKeysAbsent(t, doc, "changed")
	})

	t.Run("it places the changed section after notes and before description", func(t *testing.T) {
		f := &ToonFormatter{}

		keys := toonSectionKeys(t, f.FormatTaskDetail(detailWithChanges(&StatusChanges{Rows: rows})))

		want := []string{"id", "blocked_by", "children", "notes", "changed", "description"}
		if !slices.Equal(keys, want) {
			t.Errorf("section keys = %#v, want %#v", keys, want)
		}
	})

	t.Run("it emits changed as [] and never null in json", func(t *testing.T) {
		f := &JSONFormatter{}

		var parsed map[string]any
		if err := json.Unmarshal([]byte(f.FormatTaskDetail(detailWithChanges(&StatusChanges{}))), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		changed, ok := parsed["changed"]
		if !ok {
			t.Fatal("key \"changed\" missing from JSON document")
		}
		list, ok := changed.([]any)
		if !ok {
			t.Fatalf("changed = %#v, want a list", changed)
		}
		if len(list) != 0 {
			t.Errorf("changed has %d entries, want 0", len(list))
		}
	})

	t.Run("it carries the changed rows in json", func(t *testing.T) {
		f := &JSONFormatter{}

		var parsed map[string]any
		if err := json.Unmarshal([]byte(f.FormatTaskDetail(detailWithChanges(&StatusChanges{Rows: rows}))), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		list, ok := parsed["changed"].([]any)
		if !ok {
			t.Fatalf("changed = %#v, want a list", parsed["changed"])
		}
		if len(list) != len(rows) {
			t.Fatalf("changed has %d entries, want %d", len(list), len(rows))
		}
		for i, want := range rows {
			got, ok := list[i].(map[string]any)
			if !ok {
				t.Fatalf("changed[%d] = %#v, want an object", i, list[i])
			}
			wantFields := map[string]any{
				"id": want.ID, "title": want.Title, "from": want.From, "to": want.To, "auto": want.Auto,
			}
			for key, wantValue := range wantFields {
				if got[key] != wantValue {
					t.Errorf("changed[%d][%q] = %#v, want %#v", i, key, got[key], wantValue)
				}
			}
		}
	})

	t.Run("it omits changed from json when the detail carries no changes", func(t *testing.T) {
		f := &JSONFormatter{}

		var parsed map[string]any
		if err := json.Unmarshal([]byte(f.FormatTaskDetail(detailWithChanges(nil))), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if _, ok := parsed["changed"]; ok {
			t.Errorf("key \"changed\" = %#v, want absent", parsed["changed"])
		}
	})
}

func TestTaskDetailChangedSectionPretty(t *testing.T) {
	f := &PrettyFormatter{}
	blockOne := CascadeResult{TaskID: "tick-aaa111", TaskTitle: "Parent task", OldStatus: "done", NewStatus: "open"}
	blockTwo := CascadeResult{TaskID: "tick-bbb222", TaskTitle: "Other task", OldStatus: "open", NewStatus: "done"}

	t.Run("it appends nothing in pretty when the detail carries no blocks", func(t *testing.T) {
		want := f.FormatTaskDetail(detailWithChanges(nil))

		got := f.FormatTaskDetail(detailWithChanges(&StatusChanges{Rows: []StatusChange{
			{ID: "tick-aaa111", Title: "Parent task", From: "open", To: "in_progress"},
		}}))

		if got != want {
			t.Errorf("result = %q, want %q", got, want)
		}
	})

	t.Run("it appends one block in pretty exactly as the handler printed it", func(t *testing.T) {
		body := f.FormatTaskDetail(detailWithChanges(nil))

		got := f.FormatTaskDetail(detailWithChanges(&StatusChanges{Blocks: []CascadeResult{blockOne}}))

		want := body + "\n" + f.FormatCascadeTransition(blockOne)
		if got != want {
			t.Errorf("result = %q, want %q", got, want)
		}
	})

	t.Run("it appends two blocks in pretty in order", func(t *testing.T) {
		body := f.FormatTaskDetail(detailWithChanges(nil))

		got := f.FormatTaskDetail(detailWithChanges(&StatusChanges{Blocks: []CascadeResult{blockOne, blockTwo}}))

		want := body + "\n" + f.FormatCascadeTransition(blockOne) + "\n" + f.FormatCascadeTransition(blockTwo)
		if got != want {
			t.Errorf("result = %q, want %q", got, want)
		}
	})
}

func TestDetailCommandsCarryNoChangedSection(t *testing.T) {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	seed := []task.Task{{ID: "tick-aaa111", Title: "Solo task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}}

	t.Run("it carries no changed section on note add", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "note", "add", "tick-aaa111", "A note"))

		assertToonKeysAbsent(t, doc, "changed")
	})

	t.Run("it carries no changed section on note remove", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)
		runToonCommand(t, dir, "note", "add", "tick-aaa111", "A note")

		doc := decodeToonDoc(t, runToonCommand(t, dir, "note", "remove", "tick-aaa111", "1"))

		assertToonKeysAbsent(t, doc, "changed")
	})

	t.Run("it carries no changed section on show", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", "tick-aaa111"))

		assertToonKeysAbsent(t, doc, "changed")
	})
}

func TestOutputMutationResultChanges(t *testing.T) {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	seed := []task.Task{{ID: "tick-aaa111", Title: "Solo task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}}

	t.Run("it passes the changes through to the rendered detail", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)
		store, err := openStore(dir, FormatConfig{})
		if err != nil {
			t.Fatalf("openStore: %v", err)
		}
		defer store.Close()

		var buf bytes.Buffer
		changes := &StatusChanges{Rows: []StatusChange{
			{ID: "tick-aaa111", Title: "Solo task", From: "open", To: "in_progress"},
		}}
		if err := outputMutationResult(store, "tick-aaa111", FormatConfig{}, &ToonFormatter{}, &buf, changes); err != nil {
			t.Fatalf("outputMutationResult: %v", err)
		}

		rows := toonRows(t, decodeToonDoc(t, buf.String()), "changed")
		if len(rows) != 1 {
			t.Fatalf("changed has %d rows, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{"id": "tick-aaa111", "to": "in_progress"})
	})
}
