package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelfReferentialDepCheck(t *testing.T) {
	t.Run("it returns passing result when no tasks have self-referential deps", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
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

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when tasks have deps but none self-referential", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when task lists itself in blocked_by", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for self-referential dependency")
		}
	})

	t.Run("it detects self-ref among other valid deps", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111","tick-bbb222"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for self-referential dependency among valid deps")
		}
	})

	t.Run("it returns one result per self-referential task when multiple tasks self-ref", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-aaa111"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
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

	t.Run("it includes task ID in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-abc123","blocked_by":["tick-abc123"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "tick-abc123 depends on itself"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it reports one error per task even with duplicate self-refs in blocked_by", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-aaa111","tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result (deduplicated), got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for self-referential dependency")
		}
	})

	t.Run("it skips unparseable lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			"not json\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
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

		check := &SelfReferentialDepCheck{}
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

	t.Run("it suggests Manual fix required for self-referential errors", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Suggestion != "Manual fix required" {
			t.Errorf("expected Suggestion %q, got %q", "Manual fix required", results[0].Suggestion)
		}
	})

	t.Run("it uses Name Self-referential dependencies for all results", func(t *testing.T) {
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
				name: "self-ref",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111","blocked_by":["tick-aaa111"]}`+"\n"))
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
				check := &SelfReferentialDepCheck{}
				results := check.Run(ctxWithTickDir(tickDir))

				for i, r := range results {
					if r.Name != "Self-referential dependencies" {
						t.Errorf("result %d: expected Name %q, got %q", i, "Self-referential dependencies", r.Name)
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
				name: "self-ref",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111","blocked_by":["tick-aaa111"]}`+"\n"))
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
				check := &SelfReferentialDepCheck{}
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

	t.Run("it skips empty blocked_by", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":[]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty blocked_by; details: %s", results[0].Details)
		}
	})

	t.Run("it does not normalize IDs (compares as-is)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-AAA111","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &SelfReferentialDepCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true -- IDs should not be normalized; details: %s", results[0].Details)
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"tick-aaa111","blocked_by":["tick-aaa111"]}` + "\n")
		writeJSONL(t, tickDir, content)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		before, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before: %v", err)
		}

		check := &SelfReferentialDepCheck{}
		check.Run(ctxWithTickDir(tickDir))

		after, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after: %v", err)
		}
		if string(before) != string(after) {
			t.Error("tasks.jsonl was modified by SelfReferentialDepCheck")
		}
	})
}
