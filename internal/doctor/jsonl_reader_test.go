package doctor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/jsonl"
)

func TestScanJSONLines(t *testing.T) {
	t.Run("it returns error for missing file", func(t *testing.T) {
		tickDir := setupTickDir(t)

		_, err := ScanJSONLines(context.Background(), tickDir)

		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("it returns empty slice for empty file", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte{})

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lines) != 0 {
			t.Errorf("expected 0 lines, got %d", len(lines))
		}
	})

	t.Run("it skips blank lines and maintains correct line numbers", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Line 1: valid JSON, Line 2: blank, Line 3: blank, Line 4: valid JSON
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\n\n\n{\"id\":\"def\"}\n"))

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
		if lines[0].LineNum != 1 {
			t.Errorf("expected first line number 1, got %d", lines[0].LineNum)
		}
		if lines[1].LineNum != 4 {
			t.Errorf("expected second line number 4, got %d", lines[1].LineNum)
		}
	})

	t.Run("it returns parsed map for valid JSON lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\",\"status\":\"open\"}\n"))

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if lines[0].Parsed == nil {
			t.Fatal("expected Parsed to be non-nil for valid JSON")
		}
		if lines[0].Parsed["id"] != "abc" {
			t.Errorf("expected Parsed[\"id\"] = \"abc\", got %v", lines[0].Parsed["id"])
		}
		if lines[0].Parsed["status"] != "open" {
			t.Errorf("expected Parsed[\"status\"] = \"open\", got %v", lines[0].Parsed["status"])
		}
	})

	t.Run("it returns nil Parsed for invalid JSON lines with Raw still populated", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("not json\n"))

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if lines[0].Parsed != nil {
			t.Error("expected Parsed to be nil for invalid JSON")
		}
		if lines[0].Raw != "not json" {
			t.Errorf("expected Raw = %q, got %q", "not json", lines[0].Raw)
		}
		if lines[0].LineNum != 1 {
			t.Errorf("expected LineNum = 1, got %d", lines[0].LineNum)
		}
	})

	t.Run("it populates Raw for valid JSON lines too", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\n"))

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if lines[0].Raw != "{\"id\":\"abc\"}" {
			t.Errorf("expected Raw = %q, got %q", "{\"id\":\"abc\"}", lines[0].Raw)
		}
	})

	t.Run("it keeps whitespace-only lines, numbered, with nil Parsed", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("   \n\t\n{\"id\":\"abc\"}\n"))

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []JSONLine{
			{LineNum: 1, Raw: "   "},
			{LineNum: 2, Raw: "\t"},
			{LineNum: 3, Raw: `{"id":"abc"}`, Parsed: map[string]any{"id": "abc"}},
		}
		assertJSONLines(t, lines, want)
	})

	t.Run("it reads a line over 64 KiB and the lines after it", func(t *testing.T) {
		tickDir := setupTickDir(t)
		long := `{"id":"abc","description":"` + strings.Repeat("d", 70000) + `"}`
		writeJSONL(t, tickDir, []byte(long+"\n{\"id\":\"def\"}\n"))

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
		if lines[0].Raw != long || lines[0].Parsed == nil {
			t.Errorf("long line not read whole: len(Raw) = %d, Parsed nil = %t", len(lines[0].Raw), lines[0].Parsed == nil)
		}
		if lines[1].LineNum != 2 || lines[1].Raw != `{"id":"def"}` {
			t.Errorf("second line = %d %q, want 2 %q", lines[1].LineNum, lines[1].Raw, `{"id":"def"}`)
		}
	})

	t.Run("it strips CRLF terminators and skips a final lone carriage return", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\r\n\r\n{\"id\":\"def\"}\n\r"))

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []JSONLine{
			{LineNum: 1, Raw: `{"id":"abc"}`, Parsed: map[string]any{"id": "abc"}},
			{LineNum: 3, Raw: `{"id":"def"}`, Parsed: map[string]any{"id": "def"}},
		}
		assertJSONLines(t, lines, want)
	})

	t.Run("it reads the same lines with the same numbers as the store's line reader", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte("\r\n{\"id\":\"abc\"}\r\n\n  \n{\"id\":\"def\"}\n\r\n\t\r\nnot json")
		writeJSONL(t, tickDir, content)

		lines, err := ScanJSONLines(context.Background(), tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var want []JSONLine
		for line, err := range jsonl.Lines(bytes.NewReader(content)) {
			if err != nil {
				t.Fatalf("store reader: %v", err)
			}
			want = append(want, JSONLine{LineNum: line.Num, Raw: string(line.Text)})
		}
		if len(lines) != len(want) {
			t.Fatalf("doctor read %d lines, store read %d", len(lines), len(want))
		}
		for i := range want {
			if lines[i].LineNum != want[i].LineNum || lines[i].Raw != want[i].Raw {
				t.Errorf("line %d: doctor = %d %q, store = %d %q", i, lines[i].LineNum, lines[i].Raw, want[i].LineNum, want[i].Raw)
			}
		}
	})
}

// ctxWithoutOpening returns a context whose tasks.jsonl opener fails the test.
func ctxWithoutOpening(t *testing.T) context.Context {
	t.Helper()
	return WithTasksOpener(context.Background(), func(path string) (io.ReadCloser, error) {
		t.Errorf("opened %s, want the stored scan used", path)
		return nil, errors.New("unexpected open")
	})
}

func assertJSONLines(t *testing.T, got, want []JSONLine) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %#v, want %#v", got, want)
	}
}

func TestGetJSONLines(t *testing.T) {
	t.Run("it returns the stored lines without opening tasks.jsonl", func(t *testing.T) {
		tickDir := setupTickDir(t)
		preloaded := []JSONLine{
			{LineNum: 1, Raw: "{\"id\":\"abc\"}", Parsed: map[string]any{"id": "abc"}},
		}
		ctx := WithScan(ctxWithoutOpening(t), preloaded, nil)

		lines, err := getJSONLines(ctx, tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertJSONLines(t, lines, preloaded)
	})

	t.Run("it returns the stored read error without opening tasks.jsonl", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"def\"}\n"))
		stored := fmt.Errorf("%w: read line 3: device went away", errIncompleteRead)
		ctx := WithScan(ctxWithoutOpening(t), nil, stored)

		lines, err := getJSONLines(ctx, tickDir)

		if !errors.Is(err, stored) {
			t.Fatalf("error = %v, want the stored error %v", err, stored)
		}
		if lines != nil {
			t.Errorf("lines = %#v, want nil", lines)
		}
	})

	t.Run("it falls back to ScanJSONLines when no scan is stored", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"def\"}\n"))

		ctx := ctxWithTickDir(tickDir)

		lines, err := getJSONLines(ctx, tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if lines[0].Raw != "{\"id\":\"def\"}" {
			t.Errorf("expected Raw %q, got %q", "{\"id\":\"def\"}", lines[0].Raw)
		}
	})
}

func TestGetTaskRelationships(t *testing.T) {
	t.Run("it derives data from the stored lines without opening tasks.jsonl", func(t *testing.T) {
		tickDir := setupTickDir(t)
		preloaded := []JSONLine{
			{LineNum: 1, Raw: `{"id":"tick-aaa111","status":"open"}`, Parsed: map[string]any{"id": "tick-aaa111", "status": "open"}},
		}
		ctx := WithScan(ctxWithoutOpening(t), preloaded, nil)

		tasks, err := getTaskRelationships(ctx, tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-aaa111" {
			t.Errorf("expected ID %q, got %q", "tick-aaa111", tasks[0].ID)
		}
		if tasks[0].Status != "open" {
			t.Errorf("expected Status %q, got %q", "open", tasks[0].Status)
		}
	})

	t.Run("it returns the stored read error without opening tasks.jsonl", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-bbb222\",\"status\":\"open\"}\n"))
		stored := fmt.Errorf("%w: read line 3: device went away", errIncompleteRead)
		ctx := WithScan(ctxWithoutOpening(t), nil, stored)

		tasks, err := getTaskRelationships(ctx, tickDir)

		if !errors.Is(err, stored) {
			t.Fatalf("error = %v, want the stored error %v", err, stored)
		}
		if tasks != nil {
			t.Errorf("tasks = %#v, want nil", tasks)
		}
	})

	t.Run("it falls back to ScanJSONLines when no scan is stored", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-bbb222\",\"status\":\"open\"}\n"))

		ctx := ctxWithTickDir(tickDir)

		tasks, err := getTaskRelationships(ctx, tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "tick-bbb222" {
			t.Errorf("expected ID %q, got %q", "tick-bbb222", tasks[0].ID)
		}
	})
}
