package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

const (
	overCeilingID     = "tick-bbb222"
	overCeilingLineSz = 100_000
)

// setupOverCeilingProject writes a store holding one task line over 65,536
// bytes between two ordinary tasks.
func setupOverCeilingProject(t *testing.T) string {
	t.Helper()
	dir, _ := setupRawProject(t,
		plainTaskLine("tick-aaa111"),
		sizedTaskLine(t, overCeilingID, overCeilingLineSz),
		plainTaskLine("tick-ccc333"),
	)
	return dir
}

func showIn(t *testing.T, dir, formatFlag, id string) string {
	t.Helper()
	stdout, stderr, code := runTick(t, dir, formatFlag, "show", id)
	if code != 0 {
		t.Fatalf("show %s exit code = %d, want 0; stderr = %q", formatFlag, code, stderr)
	}
	return stdout
}

func overCeilingDescription(t *testing.T) string {
	t.Helper()
	line := sizedTaskLine(t, overCeilingID, overCeilingLineSz)
	head := `{"id":"` + overCeilingID + sizedLinePrefix
	return line[len(head) : len(line)-len(sizedLineSuffix)]
}

func TestOverCeilingStore(t *testing.T) {
	t.Run("it lists every task and shows the oversized task", func(t *testing.T) {
		dir := setupOverCeilingProject(t)

		assertListedIDs(t, dir, "tick-aaa111", overCeilingID, "tick-ccc333")

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", overCeilingID))
		assertToonFields(t, doc, map[string]any{
			"id":          overCeilingID,
			"description": overCeilingDescription(t),
		})
	})

	t.Run("it rebuilds the cache and reports every task rebuilt", func(t *testing.T) {
		dir := setupOverCeilingProject(t)

		got := runToonCommand(t, dir, "rebuild")

		if want := "Cache rebuilt: 3 tasks\n"; got != want {
			t.Errorf("rebuild output = %q, want %q", got, want)
		}
		assertListedIDs(t, dir, "tick-aaa111", overCeilingID, "tick-ccc333")
	})

	t.Run("it updates the oversized task with no repair step first", func(t *testing.T) {
		dir := setupOverCeilingProject(t)

		runToonCommand(t, dir, "update", overCeilingID, "--title", "Renamed")

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", overCeilingID))
		assertToonFields(t, doc, map[string]any{
			"title":       "Renamed",
			"description": overCeilingDescription(t),
		})
		assertListedIDs(t, dir, "tick-aaa111", overCeilingID, "tick-ccc333")
	})

	t.Run("it removes the oversized task with no repair step first", func(t *testing.T) {
		dir := setupOverCeilingProject(t)

		runToonCommand(t, dir, "remove", overCeilingID, "--force")

		assertListedIDs(t, dir, "tick-aaa111", "tick-ccc333")
	})
}

func TestMebibyteTaskRendering(t *testing.T) {
	const id = "tick-ddd444"
	chunk := "0123456789abcdef"
	description := strings.Repeat(chunk, (1<<20)/len(chunk)) + "END"

	setup := func(t *testing.T) string {
		t.Helper()
		encoded, err := json.Marshal(description)
		if err != nil {
			t.Fatalf("encode description: %v", err)
		}
		line := `{"id":"` + id + `","title":"T","status":"open","priority":2,"description":` + string(encoded) +
			`,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		dir, _ := setupRawProject(t, plainTaskLine("tick-aaa111"), line)
		return dir
	}

	t.Run("it shows the full description in toon", func(t *testing.T) {
		dir := setup(t)

		doc := decodeToonDoc(t, showIn(t, dir, "--toon", id))

		if got, _ := doc["description"].(string); got != description {
			t.Errorf("toon description length = %d, want %d", len(got), len(description))
		}
	})

	t.Run("it shows the full description in pretty", func(t *testing.T) {
		dir := setup(t)

		out := showIn(t, dir, "--pretty", id)

		_, block, found := strings.Cut(out, "Description:\n")
		if !found {
			t.Fatalf("pretty output has no Description block:\n%.500s", out)
		}
		if want := "  " + description + "\n"; block != want {
			t.Errorf("pretty description block length = %d, want %d", len(block), len(want))
		}
	})

	t.Run("it shows the full description in JSON", func(t *testing.T) {
		dir := setup(t)

		var doc struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal([]byte(showIn(t, dir, "--json", id)), &doc); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if doc.ID != id {
			t.Errorf("id = %q, want %q", doc.ID, id)
		}
		if doc.Description != description {
			t.Errorf("JSON description length = %d, want %d", len(doc.Description), len(description))
		}
	})
}
