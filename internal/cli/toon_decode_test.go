package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/task"
)

// decodeToonDoc decodes a TOON document and fails the test if it does not
// decode to a single object.
func decodeToonDoc(t *testing.T, doc string) map[string]any {
	t.Helper()
	value, err := toon.DecodeString(doc)
	if err != nil {
		t.Fatalf("decode failed: %v\ndocument:\n%s", err, doc)
	}
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("decoded value is %T, want map[string]any\ndocument:\n%s", value, doc)
	}
	return obj
}

// decodeToonNotes decodes a task detail document and returns its notes rows.
func decodeToonNotes(t *testing.T, doc string) []map[string]any {
	t.Helper()
	return toonRows(t, decodeToonDoc(t, doc), "notes")
}

// toonRows returns the rows of a tabular section of a decoded document.
func toonRows(t *testing.T, doc map[string]any, key string) []map[string]any {
	t.Helper()
	raw, ok := doc[key]
	if !ok {
		t.Fatalf("key %q missing from decoded document", key)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("key %q = %#v, want a list", key, raw)
	}
	rows := make([]map[string]any, 0, len(items))
	for i, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("%s[%d] = %#v, want an object", key, i, item)
		}
		rows = append(rows, row)
	}
	return rows
}

func assertToonFields(t *testing.T, doc map[string]any, want map[string]any) {
	t.Helper()
	for key, wantValue := range want {
		got, ok := doc[key]
		if !ok {
			t.Errorf("key %q missing from decoded document", key)
			continue
		}
		if got != wantValue {
			t.Errorf("key %q = %#v, want %#v", key, got, wantValue)
		}
	}
}

func assertToonKeysPresent(t *testing.T, doc map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := doc[key]; !ok {
			t.Errorf("key %q missing from decoded document", key)
		}
	}
}

func assertToonKeysAbsent(t *testing.T, doc map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := doc[key]; ok {
			t.Errorf("key %q should be absent from decoded document", key)
		}
	}
}

func assertToonStringList(t *testing.T, doc map[string]any, key string, want []string) {
	t.Helper()
	raw, ok := doc[key]
	if !ok {
		t.Fatalf("key %q missing from decoded document", key)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("key %q = %#v, want a list", key, raw)
	}
	got := make([]string, 0, len(items))
	for i, item := range items {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("key %q item %d = %#v, want a string", key, i, item)
		}
		got = append(got, s)
	}
	if !slices.Equal(got, want) {
		t.Errorf("key %q = %#v, want %#v", key, got, want)
	}
}

// runToonCommand runs a tick command under --toon and returns its stdout.
func runToonCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	full := append([]string{"tick", "--toon"}, args...)
	if code := app.Run(full); code != 0 {
		t.Fatalf("%v exit code = %d, want 0; stderr = %q", full, code, stderrBuf.String())
	}
	return stdoutBuf.String()
}

func assertToonNoteRows(t *testing.T, doc map[string]any, want []task.Note) {
	t.Helper()
	rows := toonRows(t, doc, "notes")
	if len(rows) != len(want) {
		t.Fatalf("notes length = %d, want %d", len(rows), len(want))
	}
	for i, row := range rows {
		assertToonFields(t, row, map[string]any{
			"index":   float64(i + 1),
			"text":    want[i].Text,
			"created": task.FormatTimestamp(want[i].Created),
		})
	}
}

func assertToonRelatedRow(t *testing.T, doc map[string]any, key string, want RelatedTask) {
	t.Helper()
	rows := toonRows(t, doc, key)
	if len(rows) != 1 {
		t.Fatalf("key %q has %d rows, want 1", key, len(rows))
	}
	assertToonFields(t, rows[0], map[string]any{
		"id":     want.ID,
		"title":  want.Title,
		"status": want.Status,
	})
}

// assertCountZeroSection asserts that a rendered document carries the given
// count-zero section header verbatim and that the same section decodes to an
// empty list. Decoding collapses a count-zero header's columns, so the column
// list can only be checked as text.
func assertCountZeroSection(t *testing.T, rendered, header string) {
	t.Helper()
	key, _, ok := strings.Cut(header, "[")
	if !ok {
		t.Fatalf("header %q is not a section header", header)
	}
	if !strings.Contains(rendered, header) {
		t.Errorf("document does not carry the count-zero header %q:\n%s", header, rendered)
	}
	if rows := toonRows(t, decodeToonDoc(t, rendered), key); len(rows) != 0 {
		t.Errorf("section %q has %d rows, want 0", key, len(rows))
	}
}

func assertToonRowsEmpty(t *testing.T, doc map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if rows := toonRows(t, doc, key); len(rows) != 0 {
			t.Errorf("key %q has %d rows, want 0", key, len(rows))
		}
	}
}

func TestToonTaskDetailConformance(t *testing.T) {
	created := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 2, 11, 8, 30, 0, 0, time.UTC)
	closed := time.Date(2026, 2, 11, 9, 0, 0, 0, time.UTC)

	notes := []task.Note{
		{Text: "First note", Created: created},
		{Text: "Second note: with a colon", Created: created.Add(time.Hour)},
	}
	fullTask := task.Task{
		ID:          "tick-aaa111",
		Title:       "Full task",
		Status:      task.StatusDone,
		Priority:    1,
		Type:        "bug",
		Parent:      "tick-ccc333",
		Tags:        []string{"backend", "ui"},
		Refs:        []string{"https://x.dev/issues/3"},
		Description: "Line one\nLine two",
		Notes:       notes,
		BlockedBy:   []string{"tick-ddd444"},
		Created:     created,
		Updated:     updated,
		Closed:      &closed,
	}
	blocker := task.Task{ID: "tick-ddd444", Title: "Blocking task", Status: task.StatusDone, Priority: 2, Created: created, Updated: created, Closed: &closed}
	child := task.Task{ID: "tick-eee555", Title: "Child task", Status: task.StatusOpen, Priority: 3, Parent: "tick-aaa111", Created: created, Updated: created}
	parent := task.Task{ID: "tick-ccc333", Title: "Parent task", Status: task.StatusOpen, Priority: 1, Created: created, Updated: created}
	bareTask := task.Task{ID: "tick-bbb222", Title: "Bare task", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created}

	seed := []task.Task{parent, blocker, child, fullTask, bareTask}

	t.Run("it decodes a show document carrying every optional field", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", "tick-aaa111"))

		assertToonFields(t, doc, map[string]any{
			"id":          "tick-aaa111",
			"title":       "Full task",
			"status":      "done",
			"priority":    float64(1),
			"type":        "bug",
			"parent":      "tick-ccc333",
			"created":     task.FormatTimestamp(created),
			"updated":     task.FormatTimestamp(updated),
			"closed":      task.FormatTimestamp(closed),
			"description": "Line one\nLine two",
		})
		assertToonStringList(t, doc, "tags", []string{"backend", "ui"})
		assertToonStringList(t, doc, "refs", []string{"https://x.dev/issues/3"})
		assertToonRelatedRow(t, doc, "blocked_by", RelatedTask{ID: "tick-ddd444", Title: "Blocking task", Status: "done"})
		assertToonRelatedRow(t, doc, "children", RelatedTask{ID: "tick-eee555", Title: "Child task", Status: "open"})
		assertToonNoteRows(t, doc, notes)
	})

	t.Run("it decodes a show document carrying no optional fields", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", "tick-bbb222"))

		assertToonFields(t, doc, map[string]any{
			"id":       "tick-bbb222",
			"title":    "Bare task",
			"status":   "open",
			"priority": float64(2),
			"created":  task.FormatTimestamp(created),
			"updated":  task.FormatTimestamp(created),
		})
		assertToonKeysAbsent(t, doc, "type", "parent", "closed", "tags", "refs", "description")
		assertToonRowsEmpty(t, doc, "blocked_by", "children", "notes")
	})

	t.Run("it decodes note add output as the task detail document", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "note", "add", "tick-aaa111", "Third note"))

		assertToonFields(t, doc, map[string]any{"id": "tick-aaa111"})
		rows := toonRows(t, doc, "notes")
		if len(rows) != 3 {
			t.Fatalf("notes length = %d, want 3", len(rows))
		}
		assertToonFields(t, rows[2], map[string]any{"index": float64(3), "text": "Third note"})
	})

	t.Run("it decodes note remove output as the task detail document", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "note", "remove", "tick-aaa111", "1"))

		assertToonFields(t, doc, map[string]any{"id": "tick-aaa111"})
		assertToonNoteRows(t, doc, notes[1:])
	})
}

// changedHeader is the schema line every status command's changed table carries.
const changedHeader = "{id,title,from,to,auto}:"

func TestToonStatusChangeConformance(t *testing.T) {
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	solo := task.Task{ID: "tick-aaa111", Title: "Solo task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}
	parent := task.Task{ID: "tick-ppp111", Title: "Parse, the header", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now}
	child := task.Task{ID: "tick-ccc111", Title: "Child task", Status: task.StatusOpen, Priority: 2, Parent: "tick-ppp111", Created: now, Updated: now}

	t.Run("it renders a single transition as a one-row changed table", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{solo})

		doc := runToonCommand(t, dir, "start", "tick-aaa111")

		rows := toonRows(t, decodeToonDoc(t, doc), "changed")
		if len(rows) != 1 {
			t.Fatalf("changed has %d rows, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{
			"id":    "tick-aaa111",
			"title": "Solo task",
			"from":  "open",
			"to":    "in_progress",
			"auto":  false,
		})
	})

	t.Run("it renders a cascade as one table with the requested row first", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{parent, child})

		doc := runToonCommand(t, dir, "done", "tick-ppp111")

		rows := toonRows(t, decodeToonDoc(t, doc), "changed")
		if len(rows) != 2 {
			t.Fatalf("changed has %d rows, want 2", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{"id": "tick-ppp111", "from": "in_progress", "to": "done"})
		assertToonFields(t, rows[1], map[string]any{"id": "tick-ccc111", "from": "open", "to": "done"})
	})

	t.Run("it marks cascaded rows auto true", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{parent, child})

		rows := toonRows(t, decodeToonDoc(t, runToonCommand(t, dir, "done", "tick-ppp111")), "changed")

		if len(rows) != 2 {
			t.Fatalf("changed has %d rows, want 2", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{"auto": false})
		assertToonFields(t, rows[1], map[string]any{"auto": true})
	})

	t.Run("it quotes a title containing a comma", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{parent, child})

		rows := toonRows(t, decodeToonDoc(t, runToonCommand(t, dir, "done", "tick-ppp111")), "changed")

		assertToonFields(t, rows[0], map[string]any{"title": "Parse, the header"})
	})

	t.Run("it decodes auto as a boolean", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{solo})

		rows := toonRows(t, decodeToonDoc(t, runToonCommand(t, dir, "start", "tick-aaa111")), "changed")

		if _, ok := rows[0]["auto"].(bool); !ok {
			t.Errorf("auto is %T, want bool", rows[0]["auto"])
		}
	})

	t.Run("it renders an empty changed set as a count-zero header", func(t *testing.T) {
		section := formatted(t).of(buildChangedSection(nil))

		if section != "changed[0]"+changedHeader {
			t.Fatalf("section = %q, want %q", section, "changed[0]"+changedHeader)
		}
		if rows := toonRows(t, decodeToonDoc(t, section), "changed"); len(rows) != 0 {
			t.Errorf("changed has %d rows, want 0", len(rows))
		}
	})

	t.Run("it prints nothing under --quiet", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, []task.Task{parent, child})

		stdout, stderr, exitCode := runTransition(t, dir, "done", "tick-ppp111", "--quiet")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})
}

// decodeToonEdgeRows decodes an edge section into from/to pairs in document order.
func decodeToonEdgeRows(t *testing.T, doc map[string]any, key string) []toonEdgeRow {
	t.Helper()
	var edges []toonEdgeRow
	for i, row := range toonRows(t, doc, key) {
		from, fromOK := row["from"].(string)
		to, toOK := row["to"].(string)
		if !fromOK || !toOK {
			t.Fatalf("%s[%d] = %#v, want string from and to", key, i, row)
		}
		edges = append(edges, toonEdgeRow{From: from, To: to})
	}
	return edges
}

// assertToonEdgeRows asserts that an edge section decodes to the given from/to pairs in order.
func assertToonEdgeRows(t *testing.T, doc map[string]any, key string, want []toonEdgeRow) {
	t.Helper()
	rows := toonRows(t, doc, key)
	if len(rows) != len(want) {
		t.Fatalf("key %q has %d rows, want %d", key, len(rows), len(want))
	}
	for i, row := range rows {
		assertToonFields(t, row, map[string]any{"from": want[i].From, "to": want[i].To})
	}
}

func TestToonDepTreeFocusedConformance(t *testing.T) {
	now := time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)
	upstream := task.Task{ID: "tick-aaa111", Title: "Upstream", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}
	middle := task.Task{ID: "tick-bbb222", Title: "Middle, with a comma", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now, Updated: now}
	downstream := task.Task{ID: "tick-ccc333", Title: "Downstream", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now}
	lone := task.Task{ID: "tick-ddd444", Title: "Lone", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}

	seed := []task.Task{upstream, middle, downstream, lone}

	t.Run("it decodes dep tree output for a task with both directions", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree", "tick-bbb222"))

		assertToonFields(t, doc, map[string]any{
			"id":     "tick-bbb222",
			"title":  "Middle, with a comma",
			"status": "open",
		})
		assertToonEdgeRows(t, doc, "blocked_by", []toonEdgeRow{{From: "tick-aaa111", To: "tick-bbb222"}})
		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{{From: "tick-bbb222", To: "tick-ccc333"}})
	})

	t.Run("it decodes dep tree output for a task with only upstream", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree", "tick-ccc333"))

		assertToonFields(t, doc, map[string]any{"id": "tick-ccc333", "status": "open"})
		assertToonRowsEmpty(t, doc, "blocks")
		if rows := toonRows(t, doc, "blocked_by"); len(rows) == 0 {
			t.Error("blocked_by should carry edges")
		}
	})

	t.Run("it decodes dep tree output for a task with only downstream", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree", "tick-aaa111"))

		assertToonFields(t, doc, map[string]any{"id": "tick-aaa111", "status": "open"})
		assertToonRowsEmpty(t, doc, "blocked_by")
		if rows := toonRows(t, doc, "blocks"); len(rows) == 0 {
			t.Error("blocks should carry edges")
		}
	})

	t.Run("it decodes dep tree output for a task with no dependencies", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, seed)

		out := runToonCommand(t, dir, "dep", "tree", "tick-ddd444")

		if strings.Contains(out, "No dependencies.") {
			t.Errorf("output should carry no prose, got:\n%s", out)
		}
		doc := decodeToonDoc(t, out)
		assertToonFields(t, doc, map[string]any{"id": "tick-ddd444", "title": "Lone", "status": "open"})
		assertToonRowsEmpty(t, doc, "blocked_by", "blocks")
	})
}
