package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/leeovery/tick/internal/jsonl"
)

// JSONLine represents a single line from tasks.jsonl.
type JSONLine struct {
	// LineNum is the 1-based line number in the file.
	LineNum int
	// Raw is the original line text.
	Raw string
	// Parsed is the parsed JSON map, or nil if parsing failed.
	Parsed map[string]any
}

var errIncompleteRead = errors.New("tasks.jsonl could not be read in full")

type tasksOpenerKeyType struct{}

// WithTasksOpener returns a context under which ScanJSONLines opens
// tasks.jsonl with open in place of os.Open.
func WithTasksOpener(ctx context.Context, open func(path string) (io.ReadCloser, error)) context.Context {
	return context.WithValue(ctx, tasksOpenerKeyType{}, open)
}

func openTasks(ctx context.Context, path string) (io.ReadCloser, error) {
	if open, ok := ctx.Value(tasksOpenerKeyType{}).(func(string) (io.ReadCloser, error)); ok {
		return open(path)
	}
	return os.Open(path)
}

// ScanJSONLines reads tasks.jsonl from the given tick directory and returns
// each line the store reads, with its line number and parse result. Lines that
// fail JSON parsing have Parsed set to nil (Raw is still populated). Returns an
// error if the file cannot be opened or read in full.
func ScanJSONLines(ctx context.Context, tickDir string) ([]JSONLine, error) {
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

	f, err := openTasks(ctx, jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("open tasks.jsonl: %w", err)
	}
	defer f.Close()

	lines := []JSONLine{}
	for l, err := range jsonl.Lines(f) {
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errIncompleteRead, err)
		}

		line := JSONLine{
			LineNum: l.Num,
			Raw:     string(l.Text),
		}

		var obj map[string]any
		if err := json.Unmarshal(l.Text, &obj); err == nil {
			line.Parsed = obj
		}

		lines = append(lines, line)
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
	return ScanJSONLines(ctx, tickDir)
}

// getTaskRelationships returns task relationship data derived from JSONLine
// data. It first attempts to get cached lines from the context via
// getJSONLines, then converts them to TaskRelationshipData.
func getTaskRelationships(ctx context.Context, tickDir string) ([]TaskRelationshipData, error) {
	lines, err := getJSONLines(ctx, tickDir)
	if err != nil {
		return nil, err
	}
	return taskRelationshipsFromLines(lines), nil
}
