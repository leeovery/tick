package doctor

import (
	"strings"
	"testing"
)

func TestJsonlSyntaxCheck(t *testing.T) {
	t.Run("it returns passing result when all lines are valid JSON", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\n{\"id\":\"def\"}\n"))

		check := &JsonlSyntaxCheck{}
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

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when file contains only blank lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("\n\n\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for blank-lines-only file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when file contains only whitespace-only lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("   \n\t\n  \t  \n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for whitespace-only file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result for a single malformed line with line number in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\nnot json\n{\"id\":\"def\"}\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for malformed line")
		}
		if !strings.Contains(results[0].Details, "Line 2") {
			t.Errorf("expected Details to contain 'Line 2', got: %s", results[0].Details)
		}
	})

	t.Run("it returns one failing result per malformed line when all lines are malformed", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("bad1\nbad2\nbad3\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
		}
	})

	t.Run("it returns failing results only for malformed lines when mixed with valid lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\nnot json\n{\"id\":\"def\"}\nalso bad\n"))

		check := &JsonlSyntaxCheck{}
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

	t.Run("it skips blank lines without counting them as valid or invalid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Line 1: valid, Line 2: blank, Line 3: invalid
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\n\nnot json\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		// Only the invalid line should produce a result
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false")
		}
		// Line 3 is the malformed one (blank line at 2 is skipped but still counts)
		if !strings.Contains(results[0].Details, "Line 3") {
			t.Errorf("expected Details to reference Line 3, got: %s", results[0].Details)
		}
	})

	t.Run("it skips trailing newline that produces empty last line", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true (trailing newline should not be an error); details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when tasks.jsonl does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// No tasks.jsonl created

		check := &JsonlSyntaxCheck{}
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
	})

	t.Run("it suggests Manual fix required for syntax errors", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("not json\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Suggestion != "Manual fix required" {
			t.Errorf("expected Suggestion %q, got %q", "Manual fix required", results[0].Suggestion)
		}
	})

	t.Run("it suggests Run tick init or verify .tick directory when file is missing", func(t *testing.T) {
		tickDir := setupTickDir(t)

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "Run tick init or verify .tick directory"
		if results[0].Suggestion != expected {
			t.Errorf("expected Suggestion %q, got %q", expected, results[0].Suggestion)
		}
	})

	t.Run("it uses CheckResult Name JSONL syntax for all results", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "passing",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\n"))
					return tickDir
				},
			},
			{
				name: "syntax error",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("not json\n"))
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
				check := &JsonlSyntaxCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

				for i, r := range results {
					if r.Name != "JSONL syntax" {
						t.Errorf("result %d: expected Name %q, got %q", i, "JSONL syntax", r.Name)
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
				name: "syntax error",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte("not json\n"))
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
				check := &JsonlSyntaxCheck{}
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

	t.Run("it reports correct 1-based line numbers (skipped blank lines still count in numbering)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Line 1: valid, Line 2: blank, Line 3: blank, Line 4: invalid, Line 5: valid
		writeJSONL(t, tickDir, []byte("{\"id\":\"abc\"}\n\n\nnot json\n{\"id\":\"def\"}\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !strings.Contains(results[0].Details, "Line 4") {
			t.Errorf("expected Details to reference Line 4, got: %s", results[0].Details)
		}
	})

	t.Run("it truncates long malformed line content in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		longLine := strings.Repeat("x", 200)
		writeJSONL(t, tickDir, []byte(longLine+"\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		// Should contain truncated content (80 chars) with "..."
		if !strings.Contains(results[0].Details, "...") {
			t.Errorf("expected Details to contain '...' for truncated content, got: %s", results[0].Details)
		}
		// The full 200-char line should NOT appear
		if strings.Contains(results[0].Details, longLine) {
			t.Error("expected long line to be truncated in Details")
		}
	})

	t.Run("it does not validate JSON field names or values — only syntax", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// {} and [] are valid JSON syntax, arbitrary field names are fine
		writeJSONL(t, tickDir, []byte("{}\n[]\n{\"bogus_field\":999}\n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true — only syntax matters; details: %s", results[0].Details)
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte("{\"id\":\"abc\"}\nnot json\n{\"id\":\"def\"}\n")
		assertReadOnly(t, tickDir, content, func() {
			check := &JsonlSyntaxCheck{}
			check.Run(ctxWithTickDir(tickDir), tickDir)
		})
	})
}
