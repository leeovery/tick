package doctor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// JSONLine represents a single line from tasks.jsonl.
type JSONLine struct {
	// LineNum is the 1-based line number in the file.
	LineNum int
	// Raw is the original line text.
	Raw string
	// Parsed is the parsed JSON map, or nil if parsing failed.
	Parsed map[string]interface{}
}

// ScanJSONLines reads tasks.jsonl from the given tick directory and returns
// all non-blank lines with their line numbers and parse results.
// Lines that fail JSON parsing have Parsed set to nil (Raw is still populated).
// Returns error only for file-open failures.
func ScanJSONLines(tickDir string) ([]JSONLine, error) {
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("open tasks.jsonl: %w", err)
	}
	defer f.Close()

	var lines []JSONLine
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		text := scanner.Text()

		if strings.TrimSpace(text) == "" {
			continue
		}

		line := JSONLine{
			LineNum: lineNum,
			Raw:     text,
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(text), &obj); err == nil {
			line.Parsed = obj
		}

		lines = append(lines, line)
	}

	if lines == nil {
		lines = []JSONLine{}
	}

	return lines, nil
}

// jsonLinesKeyType is an unexported type for the context key used to
// pass pre-scanned JSONL lines to checks.
type jsonLinesKeyType struct{}

// JSONLinesKey is the context key used to pass pre-scanned JSONLine data
// to line-level checks.
var JSONLinesKey = jsonLinesKeyType{}

// getJSONLines returns JSONL line data, first checking the context for
// pre-scanned data and falling back to ScanJSONLines.
func getJSONLines(ctx context.Context, tickDir string) ([]JSONLine, error) {
	if lines, ok := ctx.Value(JSONLinesKey).([]JSONLine); ok {
		return lines, nil
	}
	return ScanJSONLines(tickDir)
}

// taskRelationshipsKeyType is an unexported type for the context key used to
// pass pre-parsed task relationship data to checks.
type taskRelationshipsKeyType struct{}

// TaskRelationshipsKey is the context key used to pass pre-parsed
// TaskRelationshipData to relationship checks.
var TaskRelationshipsKey = taskRelationshipsKeyType{}

// getTaskRelationships returns task relationship data, first checking the
// context for pre-parsed data and falling back to ParseTaskRelationships.
func getTaskRelationships(ctx context.Context, tickDir string) ([]TaskRelationshipData, error) {
	if tasks, ok := ctx.Value(TaskRelationshipsKey).([]TaskRelationshipData); ok {
		return tasks, nil
	}
	return ParseTaskRelationships(tickDir)
}
