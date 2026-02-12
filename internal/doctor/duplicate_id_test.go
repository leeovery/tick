package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDuplicateIdCheck(t *testing.T) {
	t.Run("it returns passing result when no duplicate IDs exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-a1b2c3\"}\n{\"id\":\"tick-d4e5f6\"}\n"))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true, got false; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result for a single task", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-a1b2c3\"}\n"))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for single task; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result for empty file (zero bytes)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte{})

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it detects exact-case duplicates (tick-abc123 appears twice)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-abc123\"}\n"))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for exact-case duplicates")
		}
	})

	t.Run("it detects mixed-case duplicates (tick-ABC123 and tick-abc123)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-ABC123\"}\n{\"id\":\"tick-abc123\"}\n"))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for mixed-case duplicates")
		}
	})

	t.Run("it reports more than two duplicates of the same ID in a single result", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-abc123\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result for single duplicate group, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for triple duplicate")
		}
		// All three occurrences should be listed
		if !strings.Contains(results[0].Details, "line 1") {
			t.Errorf("expected Details to mention line 1, got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "line 2") {
			t.Errorf("expected Details to mention line 2, got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "line 3") {
			t.Errorf("expected Details to mention line 3, got: %s", results[0].Details)
		}
	})

	t.Run("it reports multiple distinct duplicate groups as separate results", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-aaa111\"}\n{\"id\":\"tick-bbb222\"}\n{\"id\":\"tick-aaa111\"}\n{\"id\":\"tick-bbb222\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 2 {
			t.Fatalf("expected 2 results (one per duplicate group), got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
		}
	})

	t.Run("it includes line numbers in the details for each duplicate occurrence", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-d4e5f6\"}\n{\"id\":\"tick-abc123\"}\n"))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !strings.Contains(results[0].Details, "line 1") {
			t.Errorf("expected Details to contain 'line 1', got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "line 3") {
			t.Errorf("expected Details to contain 'line 3', got: %s", results[0].Details)
		}
	})

	t.Run("it includes original-case ID forms in the details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-ABC123\"}\n{\"id\":\"tick-abc123\"}\n"))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !strings.Contains(results[0].Details, "tick-ABC123") {
			t.Errorf("expected Details to contain original-case 'tick-ABC123', got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "tick-abc123") {
			t.Errorf("expected Details to contain original-case 'tick-abc123', got: %s", results[0].Details)
		}
	})

	t.Run("it skips blank and whitespace-only lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-a1b2c3\"}\n\n   \n{\"id\":\"tick-d4e5f6\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it skips lines with invalid JSON silently", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-a1b2c3\"}\nnot json at all\n{\"id\":\"tick-d4e5f6\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true (invalid JSON skipped); details: %s", results[0].Details)
		}
	})

	t.Run("it skips lines with missing or empty id field", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-a1b2c3\"}\n{\"title\":\"no id\"}\n{\"id\":\"\"}\n{\"id\":\"tick-d4e5f6\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true (missing/empty id skipped); details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when tasks.jsonl does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// No tasks.jsonl created

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false when tasks.jsonl missing")
		}
		if results[0].Details != "tasks.jsonl not found" {
			t.Errorf("expected Details %q, got %q", "tasks.jsonl not found", results[0].Details)
		}
		if results[0].Suggestion != "Run tick init or verify .tick directory" {
			t.Errorf("expected Suggestion %q, got %q", "Run tick init or verify .tick directory", results[0].Suggestion)
		}
	})

	t.Run("it suggests Manual fix required for duplicate ID errors", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-abc123\"}\n"))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Suggestion != "Manual fix required" {
			t.Errorf("expected Suggestion %q, got %q", "Manual fix required", results[0].Suggestion)
		}
	})

	t.Run("it uses CheckResult Name ID uniqueness for all results", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "passing",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("{\"id\":\"tick-a1b2c3\"}\n"))
					return tickDir
				},
			},
			{
				name: "duplicate",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-abc123\"}\n"))
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
				check := &DuplicateIdCheck{}
				results := check.Run(ctxWithTickDir(tickDir))

				for i, r := range results {
					if r.Name != "ID uniqueness" {
						t.Errorf("result %d: expected Name %q, got %q", i, "ID uniqueness", r.Name)
					}
				}
			})
		}
	})

	t.Run("it uses SeverityError for all failure cases", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "duplicate",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-abc123\"}\n"))
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
				check := &DuplicateIdCheck{}
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

	t.Run("it reports correct 1-based line numbers", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Line 1: unique, Line 2: blank, Line 3: blank, Line 4: duplicate of line 1's id, Line 5: unique
		content := "{\"id\":\"tick-abc123\"}\n\n\n{\"id\":\"tick-abc123\"}\n{\"id\":\"tick-d4e5f6\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DuplicateIdCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !strings.Contains(results[0].Details, "line 1") {
			t.Errorf("expected Details to reference line 1, got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "line 4") {
			t.Errorf("expected Details to reference line 4, got: %s", results[0].Details)
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte("{\"id\":\"tick-a1b2c3\"}\n{\"id\":\"tick-a1b2c3\"}\n{\"id\":\"tick-d4e5f6\"}\n")
		writeJSONL(t, tickDir, content)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		before, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before check: %v", err)
		}

		check := &DuplicateIdCheck{}
		check.Run(ctxWithTickDir(tickDir))

		after, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after check: %v", err)
		}

		if string(before) != string(after) {
			t.Error("tasks.jsonl was modified by the check")
		}
	})
}
