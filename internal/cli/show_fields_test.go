package cli

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// parseSelection parses args and fails the test if parsing returns an error.
func parseSelection(t *testing.T, args ...string) (string, *FieldSelection) {
	t.Helper()
	id, sel, err := parseShowArgs(args)
	if err != nil {
		t.Fatalf("parseShowArgs(%v) returned error: %v", args, err)
	}
	return id, sel
}

// selectedNames returns the selected names in first-seen order.
func selectedNames(sel *FieldSelection) []string {
	if sel == nil {
		return nil
	}
	return sel.names
}

func TestParseShowArgs(t *testing.T) {
	t.Run("it returns a nil selection when no flag is present", func(t *testing.T) {
		id, sel := parseSelection(t, "tick-a1b2")
		if id != "tick-a1b2" {
			t.Errorf("id = %q, want %q", id, "tick-a1b2")
		}
		if sel != nil {
			t.Errorf("selection = %v, want nil", sel)
		}
	})

	t.Run("it accepts a comma-separated list of field names", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "title,status")
		want := []string{"title", "status"}
		if got := selectedNames(sel); !slices.Equal(got, want) {
			t.Errorf("names = %v, want %v", got, want)
		}
	})

	t.Run("it accepts the plural spelling", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--fields", "title")
		if got := selectedNames(sel); !slices.Equal(got, []string{"title"}) {
			t.Errorf("names = %v, want [title]", got)
		}
	})

	t.Run("it composes repeated flags", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "title", "--fields", "status")
		want := []string{"title", "status"}
		if got := selectedNames(sel); !slices.Equal(got, want) {
			t.Errorf("names = %v, want %v", got, want)
		}
	})

	t.Run("it trims whitespace around names", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "title, status")
		want := []string{"title", "status"}
		if got := selectedNames(sel); !slices.Equal(got, want) {
			t.Errorf("names = %v, want %v", got, want)
		}
	})

	t.Run("it collapses a repeated name", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "title,title")
		if sel.Len() != 1 {
			t.Errorf("Len() = %d, want 1", sel.Len())
		}
		only, ok := sel.Only()
		if !ok || only != "title" {
			t.Errorf("Only() = %q, %v; want \"title\", true", only, ok)
		}
	})

	t.Run("it collapses a repeated position", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "notes.2,notes.3,notes.2")
		if got := sel.Positions("notes"); !slices.Equal(got, []int{2, 3}) {
			t.Errorf("Positions(notes) = %v, want [2 3]", got)
		}
	})

	t.Run("it collapses a repeated position across flags", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "notes.3", "--field", "notes.3")
		if got := sel.Positions("notes"); !slices.Equal(got, []int{3}) {
			t.Errorf("Positions(notes) = %v, want [3]", got)
		}
	})

	t.Run("it reports several names as not a single selection", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "title,status")
		if sel.Len() != 2 {
			t.Errorf("Len() = %d, want 2", sel.Len())
		}
		if _, ok := sel.Only(); ok {
			t.Error("Only() reported a single name for a two-name selection")
		}
	})

	t.Run("it reads the task ID past the flag value", func(t *testing.T) {
		id, sel := parseSelection(t, "--field", "title", "tick-a1b2")
		if id != "tick-a1b2" {
			t.Errorf("id = %q, want %q", id, "tick-a1b2")
		}
		if !sel.Selected("title") {
			t.Error("title should be selected")
		}
	})

	t.Run("it recognises every registered name", func(t *testing.T) {
		names := []string{
			"id", "title", "status", "priority", "type", "parent",
			"created", "updated", "closed", "description",
			"notes", "tags", "refs", "children", "blocked_by",
		}
		for _, name := range names {
			_, sel, err := parseShowArgs([]string{"tick-a1b2", "--field", name})
			if err != nil {
				t.Errorf("parseShowArgs for %q returned error: %v", name, err)
				continue
			}
			if !sel.Selected(name) {
				t.Errorf("%q should be selected", name)
			}
		}
	})

	t.Run("it narrows a list section to a position", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "notes.2")
		if !sel.Selected("notes") {
			t.Fatal("notes should be selected")
		}
		if got := sel.Positions("notes"); !slices.Equal(got, []int{2}) {
			t.Errorf("Positions(notes) = %v, want [2]", got)
		}
	})

	t.Run("it takes a section whole when named both whole and by position", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "notes,notes.2")
		if got := sel.Positions("notes"); got != nil {
			t.Errorf("Positions(notes) = %v, want nil", got)
		}
		if sel.Len() != 1 {
			t.Errorf("Len() = %d, want 1", sel.Len())
		}
	})

	t.Run("it discards positions recorded before the section is named whole", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "notes.2,notes")
		if got := sel.Positions("notes"); got != nil {
			t.Errorf("Positions(notes) = %v, want nil", got)
		}
	})

	t.Run("it reports nil positions for a name taken whole", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "title")
		if got := sel.Positions("title"); got != nil {
			t.Errorf("Positions(title) = %v, want nil", got)
		}
		if sel.Selected("status") {
			t.Error("status should not be selected")
		}
	})

	t.Run("it accepts out-of-range positions", func(t *testing.T) {
		_, sel := parseSelection(t, "tick-a1b2", "--field", "notes.0,notes.-1")
		if got := sel.Positions("notes"); !slices.Equal(got, []int{0, -1}) {
			t.Errorf("Positions(notes) = %v, want [0 -1]", got)
		}
	})

	t.Run("it rejects unrecognised names", func(t *testing.T) {
		cases := []struct {
			args []string
			name string
		}{
			{[]string{"tick-a1b2", "--field", "titel"}, "titel"},
			{[]string{"tick-a1b2", "--field", ""}, ""},
			{[]string{"tick-a1b2", "--field", "title,,status"}, ""},
			{[]string{"tick-a1b2", "--field", "title,"}, ""},
			{[]string{"tick-a1b2", "--field", "notes.x"}, "notes.x"},
			{[]string{"tick-a1b2", "--field", "notes.1.2"}, "notes.1.2"},
			{[]string{"tick-a1b2", "--field", "title.1"}, "title.1"},
			{[]string{"tick-a1b2", "--field", "titel.1"}, "titel.1"},
		}
		for _, tc := range cases {
			_, _, err := parseShowArgs(tc.args)
			if err == nil {
				t.Errorf("parseShowArgs(%v) returned nil error", tc.args)
				continue
			}
			want := `unknown field "` + tc.name + `" for "show". Run 'tick help show' for usage.`
			if err.Error() != want {
				t.Errorf("error = %q, want %q", err.Error(), want)
			}
		}
	})

	t.Run("it rejects a flag with no value", func(t *testing.T) {
		for _, flag := range []string{"--field", "--fields"} {
			_, _, err := parseShowArgs([]string{"tick-a1b2", flag})
			if err == nil {
				t.Fatalf("parseShowArgs with bare %s returned nil error", flag)
			}
			if err.Error() != "--field requires a value" {
				t.Errorf("error = %q, want %q", err.Error(), "--field requires a value")
			}
		}
	})

	t.Run("it returns an empty ID when none is given", func(t *testing.T) {
		id, sel := parseSelection(t, "--field", "title")
		if id != "" {
			t.Errorf("id = %q, want empty", id)
		}
		if sel == nil {
			t.Fatal("selection should not be nil")
		}
	})
}

func TestShowFieldFlag(t *testing.T) {
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	newProject := func(t *testing.T) string {
		t.Helper()
		dir, _ := setupTickProjectWithTasks(t, []task.Task{
			{ID: "tick-a1b2c3", Title: "Add login", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		})
		return dir
	}

	t.Run("it renders the named fields for a recognised selection", func(t *testing.T) {
		dir := newProject(t)

		selected, stderr, code := runShow(t, dir, "tick-a1b2c3", "--field", "title,status")
		if code != 0 {
			t.Fatalf("exit code with --field = %d, want 0; stderr = %q", code, stderr)
		}
		want := "Title:    Add login\nStatus:   open\n"
		if selected != want {
			t.Errorf("output with --field = %q, want %q", selected, want)
		}
	})

	t.Run("it resolves the task with the flag before the ID", func(t *testing.T) {
		dir := newProject(t)

		stdout, stderr, code := runShow(t, dir, "--field", "title", "tick-a1b2c3")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		if stdout != "Add login\n" {
			t.Errorf("stdout = %q, want the resolved task's title", stdout)
		}
	})

	t.Run("it recognises a name the task does not carry", func(t *testing.T) {
		dir := newProject(t)

		_, stderr, code := runShow(t, dir, "tick-a1b2c3", "--field", "closed")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
	})

	t.Run("it rejects an unrecognised field name", func(t *testing.T) {
		dir := newProject(t)

		stdout, stderr, code := runShow(t, dir, "tick-a1b2c3", "--field", "titel")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if !strings.Contains(stderr, `unknown field "titel"`) {
			t.Errorf("stderr should name titel, got %q", stderr)
		}
	})

	t.Run("it rejects a blank name from an empty value", func(t *testing.T) {
		dir := newProject(t)

		stdout, stderr, code := runShow(t, dir, "tick-a1b2c3", "--field", "")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if !strings.Contains(stderr, `unknown field ""`) {
			t.Errorf("stderr should carry the unrecognised-name error, got %q", stderr)
		}
	})

	t.Run("it rejects a blank name from a doubled comma", func(t *testing.T) {
		dir := newProject(t)

		stdout, _, code := runShow(t, dir, "tick-a1b2c3", "--field", "title,,status")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("it rejects a trailing comma", func(t *testing.T) {
		dir := newProject(t)

		stdout, _, code := runShow(t, dir, "tick-a1b2c3", "--field", "title,")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("it rejects a flag with no value", func(t *testing.T) {
		dir := newProject(t)

		stdout, stderr, code := runShow(t, dir, "tick-a1b2c3", "--field")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if !strings.Contains(stderr, "--field requires a value") {
			t.Errorf("stderr = %q, want the requires-a-value error", stderr)
		}
	})

	t.Run("it reports a missing value when a global flag follows the flag", func(t *testing.T) {
		dir := newProject(t)

		_, stderr, code := runShow(t, dir, "tick-a1b2c3", "--field", "--json")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if !strings.Contains(stderr, "--field requires a value") {
			t.Errorf("stderr = %q, want the requires-a-value error", stderr)
		}
	})

	t.Run("it refuses quiet alongside a selection", func(t *testing.T) {
		dir := newProject(t)

		stdout, stderr, code := runShow(t, dir, "tick-a1b2c3", "--quiet", "--field", "title")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if !strings.Contains(stderr, "--quiet cannot be combined with --field") {
			t.Errorf("stderr = %q, want the quiet refusal", stderr)
		}
	})

	t.Run("it still prints the ID under quiet with no selection", func(t *testing.T) {
		dir := newProject(t)

		stdout, stderr, code := runShow(t, dir, "tick-a1b2c3", "--quiet")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		if stdout != "tick-a1b2c3\n" {
			t.Errorf("stdout = %q, want %q", stdout, "tick-a1b2c3\n")
		}
	})

	t.Run("it requires a task ID", func(t *testing.T) {
		dir := newProject(t)

		_, stderr, code := runShow(t, dir, "--field", "title")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if !strings.Contains(stderr, "task ID is required") {
			t.Errorf("stderr = %q, want the missing-ID error", stderr)
		}
	})

	t.Run("it keeps the flag off other commands", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		_, stderr, code := runCreate(t, dir, "New task", "--field", "title")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if !strings.Contains(stderr, `unknown flag "--field"`) {
			t.Errorf("stderr = %q, want the unknown-flag error", stderr)
		}
	})
}

func TestBareFieldValue(t *testing.T) {
	created := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 2, 11, 9, 30, 0, 0, time.UTC)
	closed := time.Date(2026, 2, 12, 8, 0, 0, 0, time.UTC)
	detail := TaskDetail{
		Task: task.Task{
			ID:          "tick-a1b2c3",
			Title:       "Add login",
			Status:      task.StatusDone,
			Priority:    2,
			Type:        "feature",
			Description: "Fix the parser.\n\nSteps:\n  - read the header",
			Parent:      "tick-ffee00",
			Created:     created,
			Updated:     updated,
			Closed:      &closed,
		},
		Tags:  []string{"api", "ui"},
		Refs:  []string{"https://example.com", "https://example.org"},
		Notes: []task.Note{{Text: "looked at it", Created: created}, {Text: "and again", Created: created}},
	}

	bare := func(t *testing.T, field string) (string, bool) {
		t.Helper()
		_, sel := parseSelection(t, "tick-a1b2c3", "--field", field)
		return bareFieldValue(detail, sel)
	}

	scalars := []struct {
		field string
		want  string
	}{
		{"id", "tick-a1b2c3"},
		{"title", "Add login"},
		{"status", "done"},
		{"priority", "2"},
		{"type", "feature"},
		{"parent", "tick-ffee00"},
		{"created", task.FormatTimestamp(created)},
		{"updated", task.FormatTimestamp(updated)},
		{"closed", task.FormatTimestamp(closed)},
		{"description", "Fix the parser.\n\nSteps:\n  - read the header"},
	}
	for _, tc := range scalars {
		t.Run("it returns the bare value for "+tc.field, func(t *testing.T) {
			got, ok := bare(t, tc.field)
			if !ok {
				t.Fatalf("bareFieldValue(%q) reported not bare", tc.field)
			}
			if got != tc.want {
				t.Errorf("bareFieldValue(%q) = %q, want %q", tc.field, got, tc.want)
			}
		})
	}

	t.Run("it returns an empty value for an absent optional", func(t *testing.T) {
		open := detail
		open.Task.Closed = nil
		_, sel := parseSelection(t, "tick-a1b2c3", "--field", "closed")
		got, ok := bareFieldValue(open, sel)
		if !ok {
			t.Fatal("bareFieldValue(closed) reported not bare")
		}
		if got != "" {
			t.Errorf("bareFieldValue(closed) = %q, want empty", got)
		}
	})

	t.Run("it treats a repeated name as one field", func(t *testing.T) {
		got, ok := bare(t, "title,title")
		if !ok || got != "Add login" {
			t.Errorf("bareFieldValue = %q, %v; want \"Add login\", true", got, ok)
		}
	})

	for _, field := range []string{"notes", "tags", "refs", "children", "blocked_by"} {
		t.Run("it is not bare for the list section "+field, func(t *testing.T) {
			if _, ok := bare(t, field); ok {
				t.Errorf("bareFieldValue(%q) reported bare, want not bare", field)
			}
		})
	}

	positions := []struct {
		field string
		want  string
	}{
		{"notes.2", "and again"},
		{"tags.2", "ui"},
		{"refs.1", "https://example.com"},
	}
	for _, tc := range positions {
		t.Run("it returns the bare value for "+tc.field, func(t *testing.T) {
			got, ok := bare(t, tc.field)
			if !ok {
				t.Fatalf("bareFieldValue(%q) reported not bare", tc.field)
			}
			if got != tc.want {
				t.Errorf("bareFieldValue(%q) = %q, want %q", tc.field, got, tc.want)
			}
		})
	}

	t.Run("it is bare for a repeated position", func(t *testing.T) {
		got, ok := bare(t, "notes.2,notes.2")
		if !ok || got != "and again" {
			t.Errorf("bareFieldValue = %q, %v; want \"and again\", true", got, ok)
		}
	})

	t.Run("it is not bare for two positions in one section", func(t *testing.T) {
		if _, ok := bare(t, "notes.1,notes.2"); ok {
			t.Error("bareFieldValue(notes.1,notes.2) reported bare, want not bare")
		}
	})

	for _, field := range []string{"children.1", "blocked_by.1"} {
		t.Run("it is not bare for the row position "+field, func(t *testing.T) {
			if _, ok := bare(t, field); ok {
				t.Errorf("bareFieldValue(%q) reported bare, want not bare", field)
			}
		})
	}

	t.Run("it is not bare for a position outside the section", func(t *testing.T) {
		if _, ok := bare(t, "notes.9"); ok {
			t.Error("bareFieldValue(notes.9) reported bare, want not bare")
		}
	})

	t.Run("it is not bare for two names", func(t *testing.T) {
		if _, ok := bare(t, "title,status"); ok {
			t.Error("bareFieldValue(title,status) reported bare, want not bare")
		}
	})

	t.Run("it is not bare without a selection", func(t *testing.T) {
		if _, ok := bareFieldValue(detail, nil); ok {
			t.Error("bareFieldValue(nil) reported bare, want not bare")
		}
	})
}

func TestSelectedItems(t *testing.T) {
	items := []string{"a", "b", "c"}

	t.Run("it returns every item with its position for no positions", func(t *testing.T) {
		got, positions := selectedItems(items, nil)
		if !slices.Equal(got, items) {
			t.Errorf("items = %#v, want %#v", got, items)
		}
		if !slices.Equal(positions, []int{1, 2, 3}) {
			t.Errorf("positions = %#v, want [1 2 3]", positions)
		}
	})

	t.Run("it returns the named item with its position", func(t *testing.T) {
		got, positions := selectedItems(items, []int{2})
		if !slices.Equal(got, []string{"b"}) {
			t.Errorf("items = %#v, want [b]", got)
		}
		if !slices.Equal(positions, []int{2}) {
			t.Errorf("positions = %#v, want [2]", positions)
		}
	})

	t.Run("it orders several positions ascending", func(t *testing.T) {
		got, positions := selectedItems(items, []int{3, 1})
		if !slices.Equal(got, []string{"a", "c"}) {
			t.Errorf("items = %#v, want [a c]", got)
		}
		if !slices.Equal(positions, []int{1, 3}) {
			t.Errorf("positions = %#v, want [1 3]", positions)
		}
	})

	t.Run("it collapses a repeated position", func(t *testing.T) {
		got, positions := selectedItems(items, []int{2, 2})
		if !slices.Equal(got, []string{"b"}) {
			t.Errorf("items = %#v, want [b]", got)
		}
		if !slices.Equal(positions, []int{2}) {
			t.Errorf("positions = %#v, want [2]", positions)
		}
	})

	t.Run("it skips a position outside the range", func(t *testing.T) {
		got, positions := selectedItems(items, []int{0, 2, 4, -1})
		if !slices.Equal(got, []string{"b"}) {
			t.Errorf("items = %#v, want [b]", got)
		}
		if !slices.Equal(positions, []int{2}) {
			t.Errorf("positions = %#v, want [2]", positions)
		}
	})

	t.Run("it returns nothing for an empty section", func(t *testing.T) {
		got, positions := selectedItems([]string{}, []int{1})
		if len(got) != 0 || len(positions) != 0 {
			t.Errorf("selectedItems = %#v, %#v; want empty", got, positions)
		}
	})
}
