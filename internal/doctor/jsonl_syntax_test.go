package doctor

import (
	"slices"
	"strings"
	"testing"
)

func loadableLine(id string) string {
	return `{"id":"` + id + `","title":"T","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
}

func TestJsonlSyntaxCheck(t *testing.T) {
	t.Run("it returns passing result when every line loads as a task", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\n"+loadableLine("tick-bbb222")+"\n"))

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

	t.Run("it fails each whitespace-only line, naming it by number", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte("   \n\n  \t  \n"))

		check := &JsonlSyntaxCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, wantPrefix := range []string{"Line 1:", "Line 3:"} {
			if results[i].Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
			if !strings.HasPrefix(results[i].Details, wantPrefix) {
				t.Errorf("result %d: Details = %q, want prefix %q", i, results[i].Details, wantPrefix)
			}
		}
	})

	t.Run("it returns failing result for a single malformed line with line number in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\nnot json\n"+loadableLine("tick-bbb222")+"\n"))

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
		writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\nnot json\n"+loadableLine("tick-bbb222")+"\nalso bad\n"))

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
		writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\n\nnot json\n"))

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
		writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\n"))

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
					writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\n"))
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
		writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\n\n\nnot json\n"+loadableLine("tick-bbb222")+"\n"))

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

	t.Run("it fails a line that does not load as a task, giving the loader's reason", func(t *testing.T) {
		tests := []struct {
			name       string
			line       string
			wantReason string
		}{
			{"whitespace-only", "  \t ", "unexpected end of JSON input"},
			{"null", "null", `invalid created timestamp "": parsing time "" as "2006-01-02T15:04:05Z": cannot parse "" as "2006"`},
			{"array", "[]", "json: cannot unmarshal array into Go value of type task.taskJSON"},
			{"empty object", "{}", `invalid created timestamp "": parsing time "" as "2006-01-02T15:04:05Z": cannot parse "" as "2006"`},
			{"malformed JSON", "{\"id\":", "unexpected end of JSON input"},
			{
				"wrong-typed field",
				`{"id":"tick-ccc333","title":"T","priority":"high"}`,
				"json: cannot unmarshal string into Go struct field taskJSON.priority of type int",
			},
			{
				"unparseable created timestamp",
				`{"id":"tick-ccc333","title":"T","created":"yesterday"}`,
				`invalid created timestamp "yesterday": parsing time "yesterday" as "2006-01-02T15:04:05Z": cannot parse "yesterday" as "2006"`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tickDir := setupTickDir(t)
				writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\n"+loadableLine("tick-bbb222")+"\n"+tc.line+"\n"))

				results := (&JsonlSyntaxCheck{}).Run(ctxWithTickDir(tickDir), tickDir)

				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
				}
				want := CheckResult{
					Name:       "JSONL syntax",
					Passed:     false,
					Severity:   SeverityError,
					Details:    "Line 3: " + tc.wantReason + " — " + tc.line,
					Suggestion: "Manual fix required",
				}
				if results[0] != want {
					t.Errorf("result = %+v, want %+v", results[0], want)
				}
			})
		}
	})

	t.Run("it passes a line that loads but carries values the loader does not restrict", func(t *testing.T) {
		tests := []struct {
			name   string
			fields string
		}{
			{"status outside the allowed values", `"status":"bogus","priority":2`},
			{"type outside the allowed values", `"status":"open","priority":2,"type":"weird"`},
			{"priority above 4", `"status":"open","priority":9`},
			{"priority below 0", `"status":"open","priority":-1`},
			{"repeated tag", `"status":"open","priority":2,"tags":["x","x"]`},
			{"repeated ref", `"status":"open","priority":2,"refs":["gh-1","gh-1"]`},
			{"repeated blocker", `"status":"open","priority":2,"blocked_by":["tick-aaa111","tick-aaa111"]`},
			{"field the task does not define", `"status":"open","priority":2,"bogus_field":999`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tickDir := setupTickDir(t)
				line := `{"id":"tick-bbb222","title":"T",` + tc.fields + `,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
				writeJSONL(t, tickDir, []byte(loadableLine("tick-aaa111")+"\n"+line+"\n"))

				results := (&JsonlSyntaxCheck{}).Run(ctxWithTickDir(tickDir), tickDir)

				want := []CheckResult{{Name: "JSONL syntax", Passed: true}}
				if !slices.Equal(results, want) {
					t.Errorf("results = %+v, want %+v", results, want)
				}
			})
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(loadableLine("tick-aaa111") + "\nnot json\n" + loadableLine("tick-bbb222") + "\n")
		assertReadOnly(t, tickDir, content, func() {
			check := &JsonlSyntaxCheck{}
			check.Run(ctxWithTickDir(tickDir), tickDir)
		})
	})
}
