package cli

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/testutil"
)

const (
	readmeListSampleAnchor      = "tasks[3]{id,title,status,priority,type}:"
	readmeFormatListAnchor      = "tasks[2]{id,title,status,priority,type}:"
	readmeShowSampleAnchor      = "id: tick-a1b2"
	readmeTransitionAnchor      = "changed[1]{id,title,from,to,auto}:"
	readmeCascadeAnchor         = "changed[2]{id,title,from,to,auto}:"
	readmeDepTreeAnchor         = "dep_tree[2]{from,to}:"
	readmeEmptyTypeRow          = `tick-d5c6,Update docs,open,3,""`
	readmeMissingAnchorFixture  = "tasks[9]{nothing}:"
	readmeShellPromptLinePrefix = "$ tick"
	readmeUnchangedMarker       = "(unchanged)"
	readmeObjectHeaderMarker    = "summary{"
)

type readmeFence struct {
	info string
	body string
}

func readmeContent(t *testing.T) string {
	t.Helper()
	path := filepath.Join(testutil.FindRepoRoot(t), "README.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	return string(content)
}

// readmeFences returns the README's fenced blocks with their info string, each
// body stripped of any leading shell prompt line.
func readmeFences(t *testing.T) []readmeFence {
	t.Helper()

	var fences []readmeFence
	var current []string
	info := ""
	inBlock := false
	for line := range strings.SplitSeq(readmeContent(t), "\n") {
		if !strings.HasPrefix(line, "```") {
			if inBlock {
				current = append(current, line)
			}
			continue
		}
		if !inBlock {
			inBlock = true
			info = strings.TrimSpace(strings.TrimPrefix(line, "```"))
			current = nil
			continue
		}
		if body, ok := fenceBody(current); ok {
			fences = append(fences, readmeFence{info: info, body: body})
		}
		inBlock = false
	}
	return fences
}

// readmeToonBlocks returns the README's fenced blocks that carry no info
// string, keyed by their first line once any shell prompt line is stripped.
func readmeToonBlocks(t *testing.T) map[string]string {
	t.Helper()

	blocks := make(map[string]string)
	for _, fence := range readmeFences(t) {
		if fence.info != "" {
			continue
		}
		anchor, _, _ := strings.Cut(fence.body, "\n")
		blocks[anchor] = fence.body
	}
	return blocks
}

func fenceBody(lines []string) (string, bool) {
	if len(lines) > 0 && strings.HasPrefix(lines[0], readmeShellPromptLinePrefix) {
		lines = lines[1:]
	}
	if len(lines) == 0 {
		return "", false
	}
	return strings.Join(lines, "\n"), true
}

func readmeChangedRows(t *testing.T, blocks map[string]string, anchor string) []map[string]any {
	t.Helper()
	block, ok := blocks[anchor]
	if !ok {
		t.Fatalf("no README toon sample anchored on %q", anchor)
	}
	decoded, err := toon.DecodeString(block)
	if err != nil {
		t.Fatalf("README toon sample anchored on %q does not decode: %v", anchor, err)
	}
	return changedRowsOf(t, decoded, anchor)
}

func changedRowsOf(t *testing.T, decoded any, source string) []map[string]any {
	t.Helper()
	document, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("%s sample is %T, want an object", source, decoded)
	}
	if keys := slices.Sorted(maps.Keys(document)); !slices.Equal(keys, []string{"changed"}) {
		t.Fatalf("%s sample has keys %v, want only [changed]", source, keys)
	}
	entries, ok := document["changed"].([]any)
	if !ok {
		t.Fatalf("%s sample changed is %T, want a list", source, document["changed"])
	}
	rows := make([]map[string]any, 0, len(entries))
	for i, entry := range entries {
		row, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("%s sample row %d is %T, want an object", source, i, entry)
		}
		rows = append(rows, row)
	}
	return rows
}

func decodeAnchoredBlock(blocks map[string]string, anchor string) error {
	block, ok := blocks[anchor]
	if !ok {
		return fmt.Errorf("no README toon sample anchored on %q", anchor)
	}
	if _, err := toon.DecodeString(block); err != nil {
		return fmt.Errorf("README toon sample anchored on %q does not decode: %w", anchor, err)
	}
	return nil
}

func TestREADMEToonSamplesDecode(t *testing.T) {
	blocks := readmeToonBlocks(t)

	t.Run("it decodes the README show sample", func(t *testing.T) {
		if err := decodeAnchoredBlock(blocks, readmeShowSampleAnchor); err != nil {
			t.Error(err)
		}
	})

	t.Run("it decodes the README list samples", func(t *testing.T) {
		for _, anchor := range []string{readmeListSampleAnchor, readmeFormatListAnchor} {
			if err := decodeAnchoredBlock(blocks, anchor); err != nil {
				t.Error(err)
			}
		}
	})

	t.Run("it decodes the README simple transition sample", func(t *testing.T) {
		if err := decodeAnchoredBlock(blocks, readmeTransitionAnchor); err != nil {
			t.Error(err)
		}
	})

	t.Run("it decodes the README cascade sample", func(t *testing.T) {
		if err := decodeAnchoredBlock(blocks, readmeCascadeAnchor); err != nil {
			t.Error(err)
		}
	})

	t.Run("it decodes the README dep tree sample", func(t *testing.T) {
		if err := decodeAnchoredBlock(blocks, readmeDepTreeAnchor); err != nil {
			t.Error(err)
		}
	})

	t.Run("it has no single-object section header in the README", func(t *testing.T) {
		if strings.Contains(readmeContent(t), readmeObjectHeaderMarker) {
			t.Errorf("README contains %q", readmeObjectHeaderMarker)
		}
	})

	t.Run("it carries the title column in the README samples", func(t *testing.T) {
		for _, anchor := range []string{readmeTransitionAnchor, readmeCascadeAnchor} {
			for i, row := range readmeChangedRows(t, blocks, anchor) {
				if title, _ := row["title"].(string); title == "" {
					t.Errorf("sample %q row %d has no title: %v", anchor, i, row)
				}
			}
		}
	})

	t.Run("it marks only cascaded rows as automatic", func(t *testing.T) {
		for anchor, want := range map[string][]bool{
			readmeTransitionAnchor: {false},
			readmeCascadeAnchor:    {false, true},
		} {
			rows := readmeChangedRows(t, blocks, anchor)
			if len(rows) != len(want) {
				t.Fatalf("sample %q has %d rows, want %d", anchor, len(rows), len(want))
			}
			for i, row := range rows {
				if row["auto"] != want[i] {
					t.Errorf("sample %q row %d auto = %#v, want %v", anchor, i, row["auto"], want[i])
				}
			}
		}
	})

	t.Run("it fails when an anchored sample is missing", func(t *testing.T) {
		if err := decodeAnchoredBlock(blocks, readmeMissingAnchorFixture); err == nil {
			t.Errorf("expected an error for anchor %q, got nil", readmeMissingAnchorFixture)
		}
	})

	t.Run("it renders an empty type as a quoted empty string", func(t *testing.T) {
		block, ok := blocks[readmeListSampleAnchor]
		if !ok {
			t.Fatalf("no README toon sample anchored on %q", readmeListSampleAnchor)
		}
		lines := strings.Split(block, "\n")
		if len(lines) != 4 {
			t.Fatalf("list sample has %d lines, want 4:\n%s", len(lines), block)
		}
		if got := strings.TrimSpace(lines[3]); got != readmeEmptyTypeRow {
			t.Errorf("empty-type row = %q, want %q", got, readmeEmptyTypeRow)
		}
	})
}

func TestREADMETransitionSamples(t *testing.T) {
	t.Run("it parses the README JSON transition sample", func(t *testing.T) {
		var objects []string
		for _, fence := range readmeFences(t) {
			if fence.info == "json" && strings.HasPrefix(fence.body, "{") {
				objects = append(objects, fence.body)
			}
		}
		if len(objects) != 1 {
			t.Fatalf("README has %d JSON object samples, want 1", len(objects))
		}

		var decoded any
		if err := json.Unmarshal([]byte(objects[0]), &decoded); err != nil {
			t.Fatalf("README JSON transition sample does not parse: %v", err)
		}
		rows := changedRowsOf(t, decoded, "README JSON transition")
		if len(rows) != 1 {
			t.Fatalf("JSON transition sample has %d rows, want 1", len(rows))
		}
		if _, ok := rows[0]["auto"].(bool); !ok {
			t.Errorf("JSON transition auto = %#v, want an unquoted boolean", rows[0]["auto"])
		}
	})

	t.Run("it has no unchanged marker in the README", func(t *testing.T) {
		if strings.Contains(readmeContent(t), readmeUnchangedMarker) {
			t.Errorf("README contains %q", readmeUnchangedMarker)
		}
	})
}
