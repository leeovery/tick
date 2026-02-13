package doctor

import (
	"strings"
	"testing"
)

func TestIdFormatCheck(t *testing.T) {
	t.Run("it returns passing result when all IDs match tick-{6 hex} format", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-a1b2c3\"}\n{\"id\":\"tick-d4e5f6\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true, got false; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result for empty file (zero bytes)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte{})

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for empty ID field with line number in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for empty ID")
		}
		if results[0].Details != "Line 1: invalid ID '' — expected format tick-{6 hex}" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when id field is missing from JSON object", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"title\":\"no id here\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for missing id key")
		}
		if results[0].Details != "Line 1: missing id field" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for uppercase hex chars (e.g., tick-A1B2C3)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-A1B2C3\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for uppercase hex")
		}
		if results[0].Details != "Line 1: invalid ID 'tick-A1B2C3' — expected format tick-{6 hex}" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for mixed-case hex chars (e.g., tick-a1B2c3)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-a1B2c3\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for mixed-case hex")
		}
		if results[0].Details != "Line 1: invalid ID 'tick-a1B2c3' — expected format tick-{6 hex}" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for extra chars beyond 6 hex (e.g., tick-a1b2c3d4)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-a1b2c3d4\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for extra hex chars")
		}
		if results[0].Details != "Line 1: invalid ID 'tick-a1b2c3d4' — expected format tick-{6 hex}" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for fewer than 6 hex chars (e.g., tick-a1b)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-a1b\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for fewer hex chars")
		}
		if results[0].Details != "Line 1: invalid ID 'tick-a1b' — expected format tick-{6 hex}" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for wrong prefix (e.g., task-a1b2c3)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"task-a1b2c3\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for wrong prefix")
		}
		if results[0].Details != "Line 1: invalid ID 'task-a1b2c3' — expected format tick-{6 hex}" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for missing prefix (e.g., a1b2c3)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"a1b2c3\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for missing prefix")
		}
		if results[0].Details != "Line 1: invalid ID 'a1b2c3' — expected format tick-{6 hex}" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result for numeric-only random part (tick-123456)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-123456\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for numeric-only hex; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing results for each invalid ID when mixed valid and invalid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-a1b2c3\"}\n{\"id\":\"bad\"}\n{\"id\":\"tick-d4e5f6\"}\n{\"id\":\"also-bad\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 2 {
			t.Fatalf("expected 2 failing results, got %d", len(results))
		}
		for _, r := range results {
			if r.Passed {
				t.Error("expected all results to be failures")
			}
		}
	})

	t.Run("it reports correct count of failures — one per invalid ID", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"bad1\"}\n{\"id\":\"bad2\"}\n{\"id\":\"bad3\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 3 {
			t.Fatalf("expected 3 results (one per invalid ID), got %d", len(results))
		}
	})

	t.Run("it skips unparseable lines silently", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-a1b2c3\"}\nnot json at all\n{\"id\":\"tick-d4e5f6\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true (unparseable lines should be skipped); details: %s", results[0].Details)
		}
	})

	t.Run("it skips blank lines without error", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"tick-a1b2c3\"}\n\n\n{\"id\":\"tick-d4e5f6\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when tasks.jsonl does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// No tasks.jsonl created

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

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

	t.Run("it shows actual invalid ID value in error details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"WRONG-FORMAT\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !strings.Contains(results[0].Details, "WRONG-FORMAT") {
			t.Errorf("expected Details to contain the actual invalid ID value, got: %s", results[0].Details)
		}
	})

	t.Run("it suggests Manual fix required for all format violations", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":\"bad1\"}\n{\"id\":\"bad2\"}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		for i, r := range results {
			if r.Suggestion != "Manual fix required" {
				t.Errorf("result %d: expected Suggestion %q, got %q", i, "Manual fix required", r.Suggestion)
			}
		}
	})

	t.Run("it uses CheckResult Name ID format for all results", func(t *testing.T) {
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
				name: "format violation",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("{\"id\":\"bad\"}\n"))
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
				check := &IdFormatCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

				for i, r := range results {
					if r.Name != "ID format" {
						t.Errorf("result %d: expected Name %q, got %q", i, "ID format", r.Name)
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
				name: "format violation",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("{\"id\":\"bad\"}\n"))
					return tickDir
				},
			},
			{
				name: "missing id key",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("{\"title\":\"no id\"}\n"))
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
				check := &IdFormatCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

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

	t.Run("it does not normalize IDs to lowercase", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// tick-A1b2C3 has uppercase — should fail even though lowercase version would be valid
		writeJSONL(t, tickDir, []byte("{\"id\":\"tick-A1b2C3\"}\n"))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false — ID should not be normalized to lowercase")
		}
	})

	t.Run("it handles non-string id values (null, number) as format violations", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := "{\"id\":null}\n{\"id\":42}\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &IdFormatCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false for non-string id", i)
			}
		}
		// Verify actual value is shown
		if !strings.Contains(results[0].Details, "<null>") {
			t.Errorf("expected Details to show null value, got: %s", results[0].Details)
		}
		if !strings.Contains(results[1].Details, "42") {
			t.Errorf("expected Details to show numeric value, got: %s", results[1].Details)
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte("{\"id\":\"tick-a1b2c3\"}\n{\"id\":\"bad\"}\n{\"id\":\"tick-d4e5f6\"}\n")
		assertReadOnly(t, tickDir, content, func() {
			check := &IdFormatCheck{}
			check.Run(ctxWithTickDir(tickDir), tickDir)
		})
	})
}
