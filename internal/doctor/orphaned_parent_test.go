package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrphanedParentCheck(t *testing.T) {
	t.Run("it returns passing result when all parents are valid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result for empty file", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte{})

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when no tasks have parents", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when task references non-existent parent", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-missing"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for orphaned parent")
		}
		if !strings.Contains(results[0].Details, "tick-aaa111") {
			t.Errorf("expected Details to contain child ID, got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "tick-missing") {
			t.Errorf("expected Details to contain parent ID, got: %s", results[0].Details)
		}
	})

	t.Run("it returns one failing result per orphaned parent reference", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-missing1"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-missing2"}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
		}
	})

	t.Run("it includes child ID and missing parent ID in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-child1","parent":"tick-parent1"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !strings.Contains(results[0].Details, "tick-child1") {
			t.Errorf("expected Details to contain child ID 'tick-child1', got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "tick-parent1") {
			t.Errorf("expected Details to contain parent ID 'tick-parent1', got: %s", results[0].Details)
		}
	})

	t.Run("it follows spec wording for details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-child1","parent":"tick-parent1"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "tick-child1 references non-existent parent tick-parent1"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it treats null parent as valid root task (not flagged)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":null}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for null parent; details: %s", results[0].Details)
		}
	})

	t.Run("it treats absent parent as valid root task (not flagged)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for absent parent; details: %s", results[0].Details)
		}
	})

	t.Run("it skips unparseable lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			"not json\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result with init suggestion when file is missing", func(t *testing.T) {
		tickDir := setupTickDir(t)

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for missing file")
		}
		if results[0].Details != "tasks.jsonl not found" {
			t.Errorf("expected Details %q, got %q", "tasks.jsonl not found", results[0].Details)
		}
		if results[0].Suggestion != "Run tick init or verify .tick directory" {
			t.Errorf("expected Suggestion %q, got %q", "Run tick init or verify .tick directory", results[0].Suggestion)
		}
	})

	t.Run("it suggests Manual fix required for orphaned parent errors", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-missing"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Suggestion != "Manual fix required" {
			t.Errorf("expected Suggestion %q, got %q", "Manual fix required", results[0].Suggestion)
		}
	})

	t.Run("it uses Name Orphaned parents for all results", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "passing",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111"}`+"\n"))
					return tickDir
				},
			},
			{
				name: "orphaned parent",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111","parent":"tick-missing"}`+"\n"))
					return tickDir
				},
			},
			{
				name: "missing file",
				setup: func(t *testing.T) string {
					return setupTickDir(t)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tickDir := tc.setup(t)
				check := &OrphanedParentCheck{}
				results := check.Run(ctxWithTickDir(tickDir))

				for i, r := range results {
					if r.Name != "Orphaned parents" {
						t.Errorf("result %d: expected Name %q, got %q", i, "Orphaned parents", r.Name)
					}
				}
			})
		}
	})

	t.Run("it uses SeverityError for all failures", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "orphaned parent",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111","parent":"tick-missing"}`+"\n"))
					return tickDir
				},
			},
			{
				name: "missing file",
				setup: func(t *testing.T) string {
					return setupTickDir(t)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tickDir := tc.setup(t)
				check := &OrphanedParentCheck{}
				results := check.Run(ctxWithTickDir(tickDir))

				for i, r := range results {
					if r.Passed {
						t.Errorf("result %d: expected Passed false", i)
					}
					if r.Severity != SeverityError {
						t.Errorf("result %d: expected Severity %q, got %q", i, SeverityError, r.Severity)
					}
				}
			})
		}
	})

	t.Run("it does not normalize IDs (compares as-is)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Parent is "tick-AAA111" but only "tick-aaa111" exists — should be orphaned
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-AAA111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false — IDs should not be normalized")
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"tick-aaa111","parent":"tick-missing"}` + "\n")
		writeJSONL(t, tickDir, content)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		before, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before: %v", err)
		}

		check := &OrphanedParentCheck{}
		check.Run(ctxWithTickDir(tickDir))

		after, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after: %v", err)
		}
		if string(before) != string(after) {
			t.Error("tasks.jsonl was modified by OrphanedParentCheck")
		}
	})
}
