package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/testutil"
)

const (
	readmeListSampleAnchor      = "tasks[3]{id,title,status,priority,type}:"
	readmeFormatListAnchor      = "tasks[2]{id,title,status,priority,type}:"
	readmeShowSampleAnchor      = "id: tick-a1b2"
	readmeEmptyTypeRow          = `tick-d5c6,Update docs,open,3,""`
	readmeMissingAnchorFixture  = "tasks[9]{nothing}:"
	readmeShellPromptLinePrefix = "$ tick"
)

// readmeToonBlocks returns the README's fenced blocks that carry no info
// string, keyed by their first line once any shell prompt line is stripped.
func readmeToonBlocks(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join(testutil.FindRepoRoot(t), "README.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}

	blocks := make(map[string]string)
	var current []string
	inBlock, capture := false, false
	for line := range strings.SplitSeq(string(content), "\n") {
		if !strings.HasPrefix(line, "```") {
			if capture {
				current = append(current, line)
			}
			continue
		}
		if !inBlock {
			inBlock = true
			capture = strings.TrimSpace(line) == "```"
			current = nil
			continue
		}
		if capture {
			if anchor, block, ok := anchorToonBlock(current); ok {
				blocks[anchor] = block
			}
		}
		inBlock, capture = false, false
	}
	return blocks
}

func anchorToonBlock(lines []string) (anchor, block string, ok bool) {
	if len(lines) > 0 && strings.HasPrefix(lines[0], readmeShellPromptLinePrefix) {
		lines = lines[1:]
	}
	if len(lines) == 0 {
		return "", "", false
	}
	return lines[0], strings.Join(lines, "\n"), true
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
