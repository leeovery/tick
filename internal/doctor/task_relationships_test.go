package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTaskRelationships(t *testing.T) {
	t.Run("it returns empty slice for empty file", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte{})

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 0 {
			t.Errorf("expected 0 entries, got %d", len(data))
		}
	})

	t.Run("it returns error for missing file", func(t *testing.T) {
		tickDir := setupTickDir(t)

		_, err := ParseTaskRelationships(tickDir)

		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("it extracts id, parent, blocked_by, and status correctly", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-ccc333"],"status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(data))
		}
		if data[0].ID != "tick-aaa111" {
			t.Errorf("expected ID %q, got %q", "tick-aaa111", data[0].ID)
		}
		if data[0].Parent != "tick-bbb222" {
			t.Errorf("expected Parent %q, got %q", "tick-bbb222", data[0].Parent)
		}
		if len(data[0].BlockedBy) != 1 || data[0].BlockedBy[0] != "tick-ccc333" {
			t.Errorf("expected BlockedBy [tick-ccc333], got %v", data[0].BlockedBy)
		}
		if data[0].Status != "open" {
			t.Errorf("expected Status %q, got %q", "open", data[0].Status)
		}
	})

	t.Run("it sets parent to empty string when parent is null", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":null}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(data))
		}
		if data[0].Parent != "" {
			t.Errorf("expected empty Parent for null, got %q", data[0].Parent)
		}
	})

	t.Run("it sets parent to empty string when parent is absent", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(data))
		}
		if data[0].Parent != "" {
			t.Errorf("expected empty Parent for absent, got %q", data[0].Parent)
		}
	})

	t.Run("it sets blocked_by to empty slice when blocked_by is null", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":null}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(data))
		}
		if data[0].BlockedBy == nil {
			t.Fatal("expected non-nil empty slice for null blocked_by")
		}
		if len(data[0].BlockedBy) != 0 {
			t.Errorf("expected empty BlockedBy for null, got %v", data[0].BlockedBy)
		}
	})

	t.Run("it sets blocked_by to empty slice when blocked_by is absent", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(data))
		}
		if data[0].BlockedBy == nil {
			t.Fatal("expected non-nil empty slice for absent blocked_by")
		}
		if len(data[0].BlockedBy) != 0 {
			t.Errorf("expected empty BlockedBy for absent, got %v", data[0].BlockedBy)
		}
	})

	t.Run("it skips blank lines, unparseable JSON, and missing or non-string id", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			"\n" +
			"   \n" +
			"not json\n" +
			`{"title":"no id"}` + "\n" +
			`{"id":42}` + "\n" +
			`{"id":null}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(data))
		}
		if data[0].ID != "tick-aaa111" {
			t.Errorf("expected first ID %q, got %q", "tick-aaa111", data[0].ID)
		}
		if data[1].ID != "tick-bbb222" {
			t.Errorf("expected second ID %q, got %q", "tick-bbb222", data[1].ID)
		}
	})

	t.Run("it reports correct 1-based line numbers including blank lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			"\n" +
			"\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(data))
		}
		if data[0].Line != 1 {
			t.Errorf("expected first entry line 1, got %d", data[0].Line)
		}
		if data[1].Line != 4 {
			t.Errorf("expected second entry line 4, got %d", data[1].Line)
		}
	})

	t.Run("it handles trailing newline", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(data))
		}
	})

	t.Run("it extracts multiple blocked_by IDs correctly", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222","tick-ccc333","tick-ddd444"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		data, err := ParseTaskRelationships(tickDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(data))
		}
		if len(data[0].BlockedBy) != 3 {
			t.Fatalf("expected 3 blocked_by entries, got %d", len(data[0].BlockedBy))
		}
		expected := []string{"tick-bbb222", "tick-ccc333", "tick-ddd444"}
		for i, want := range expected {
			if data[0].BlockedBy[i] != want {
				t.Errorf("BlockedBy[%d] = %q, want %q", i, data[0].BlockedBy[i], want)
			}
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"tick-aaa111","parent":"tick-bbb222"}` + "\n")
		writeJSONL(t, tickDir, content)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		before, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before: %v", err)
		}

		_, _ = ParseTaskRelationships(tickDir)

		after, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after: %v", err)
		}
		if string(before) != string(after) {
			t.Error("tasks.jsonl was modified by ParseTaskRelationships")
		}
	})
}

func TestTaskRelationshipsFromLines(t *testing.T) {
	t.Run("it skips entries where Parsed is nil", func(t *testing.T) {
		lines := []JSONLine{
			{LineNum: 1, Raw: `{"id":"tick-aaa111","status":"open"}`, Parsed: map[string]interface{}{"id": "tick-aaa111", "status": "open"}},
			{LineNum: 2, Raw: "not json", Parsed: nil},
			{LineNum: 3, Raw: `{"id":"tick-bbb222"}`, Parsed: map[string]interface{}{"id": "tick-bbb222"}},
		}

		result := taskRelationshipsFromLines(lines)

		if len(result) != 2 {
			t.Fatalf("expected 2 entries (skipping nil Parsed), got %d", len(result))
		}
		if result[0].ID != "tick-aaa111" {
			t.Errorf("expected first ID %q, got %q", "tick-aaa111", result[0].ID)
		}
		if result[0].Line != 1 {
			t.Errorf("expected first Line 1, got %d", result[0].Line)
		}
		if result[0].Status != "open" {
			t.Errorf("expected first Status %q, got %q", "open", result[0].Status)
		}
		if result[1].ID != "tick-bbb222" {
			t.Errorf("expected second ID %q, got %q", "tick-bbb222", result[1].ID)
		}
		if result[1].Line != 3 {
			t.Errorf("expected second Line 3, got %d", result[1].Line)
		}
	})

	t.Run("it skips entries with missing id field", func(t *testing.T) {
		lines := []JSONLine{
			{LineNum: 1, Raw: `{"title":"no id"}`, Parsed: map[string]interface{}{"title": "no id"}},
		}

		result := taskRelationshipsFromLines(lines)

		if len(result) != 0 {
			t.Fatalf("expected 0 entries for missing id, got %d", len(result))
		}
	})

	t.Run("it skips entries with non-string id", func(t *testing.T) {
		lines := []JSONLine{
			{LineNum: 1, Raw: `{"id":42}`, Parsed: map[string]interface{}{"id": float64(42)}},
		}

		result := taskRelationshipsFromLines(lines)

		if len(result) != 0 {
			t.Fatalf("expected 0 entries for non-string id, got %d", len(result))
		}
	})

	t.Run("it extracts all fields correctly", func(t *testing.T) {
		lines := []JSONLine{
			{
				LineNum: 5,
				Raw:     `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-ccc333"],"status":"open"}`,
				Parsed: map[string]interface{}{
					"id":         "tick-aaa111",
					"parent":     "tick-bbb222",
					"blocked_by": []interface{}{"tick-ccc333"},
					"status":     "open",
				},
			},
		}

		result := taskRelationshipsFromLines(lines)

		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		if result[0].ID != "tick-aaa111" {
			t.Errorf("expected ID %q, got %q", "tick-aaa111", result[0].ID)
		}
		if result[0].Parent != "tick-bbb222" {
			t.Errorf("expected Parent %q, got %q", "tick-bbb222", result[0].Parent)
		}
		if len(result[0].BlockedBy) != 1 || result[0].BlockedBy[0] != "tick-ccc333" {
			t.Errorf("expected BlockedBy [tick-ccc333], got %v", result[0].BlockedBy)
		}
		if result[0].Status != "open" {
			t.Errorf("expected Status %q, got %q", "open", result[0].Status)
		}
		if result[0].Line != 5 {
			t.Errorf("expected Line 5, got %d", result[0].Line)
		}
	})

	t.Run("it returns empty slice for empty input", func(t *testing.T) {
		result := taskRelationshipsFromLines([]JSONLine{})

		if result == nil {
			t.Fatal("expected non-nil empty slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("expected 0 entries, got %d", len(result))
		}
	})

	t.Run("it initializes BlockedBy to empty slice when absent", func(t *testing.T) {
		lines := []JSONLine{
			{LineNum: 1, Raw: `{"id":"tick-aaa111"}`, Parsed: map[string]interface{}{"id": "tick-aaa111"}},
		}

		result := taskRelationshipsFromLines(lines)

		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		if result[0].BlockedBy == nil {
			t.Fatal("expected non-nil empty slice for absent blocked_by")
		}
		if len(result[0].BlockedBy) != 0 {
			t.Errorf("expected empty BlockedBy, got %v", result[0].BlockedBy)
		}
	})
}
