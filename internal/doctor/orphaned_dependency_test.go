package doctor

import (
	"strings"
	"testing"
)

func TestOrphanedDependencyCheck(t *testing.T) {
	t.Run("it returns passing result when all blocked_by refs are valid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

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

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when no tasks have blocked_by", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when task refs non-existent dep", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-missing"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for orphaned dependency")
		}
	})

	t.Run("it returns one result per orphaned dep when single task has multiple orphaned deps", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-missing1","tick-missing2"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
		}
	})

	t.Run("it returns one result per orphaned dep across multiple tasks", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-missing1"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-missing2"]}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
		}
	})

	t.Run("it includes task ID and missing dep ID in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-child1","blocked_by":["tick-dep1"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !strings.Contains(results[0].Details, "tick-child1") {
			t.Errorf("expected Details to contain task ID 'tick-child1', got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "tick-dep1") {
			t.Errorf("expected Details to contain dep ID 'tick-dep1', got: %s", results[0].Details)
		}
	})

	t.Run("it follows spec wording for details pattern", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-child1","blocked_by":["tick-dep1"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "tick-child1 depends on non-existent task tick-dep1"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it treats empty blocked_by array as valid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":[]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty blocked_by array; details: %s", results[0].Details)
		}
	})

	t.Run("it treats null blocked_by as valid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":null}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for null blocked_by; details: %s", results[0].Details)
		}
	})

	t.Run("it treats absent blocked_by as valid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for absent blocked_by; details: %s", results[0].Details)
		}
	})

	t.Run("it only flags invalid refs in mix of valid and invalid", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111","tick-missing"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result (only invalid ref), got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for orphaned dependency")
		}
		if !strings.Contains(results[0].Details, "tick-missing") {
			t.Errorf("expected Details to contain 'tick-missing', got: %s", results[0].Details)
		}
		if !strings.Contains(results[0].Details, "tick-bbb222") {
			t.Errorf("expected Details to contain 'tick-bbb222', got: %s", results[0].Details)
		}
	})

	t.Run("it skips unparseable lines", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			"not json\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result with init suggestion when file is missing", func(t *testing.T) {
		tickDir := setupTickDir(t)

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

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

	t.Run("it suggests Manual fix required for orphaned dependency errors", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-missing"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Suggestion != "Manual fix required" {
			t.Errorf("expected Suggestion %q, got %q", "Manual fix required", results[0].Suggestion)
		}
	})

	t.Run("it uses Name Orphaned dependencies for all results", func(t *testing.T) {
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
				name: "orphaned dep",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111","blocked_by":["tick-missing"]}`+"\n"))
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
				check := &OrphanedDependencyCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

				for i, r := range results {
					if r.Name != "Orphaned dependencies" {
						t.Errorf("result %d: expected Name %q, got %q", i, "Orphaned dependencies", r.Name)
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
				name: "orphaned dep",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111","blocked_by":["tick-missing"]}`+"\n"))
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
				check := &OrphanedDependencyCheck{}
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

	t.Run("it does not normalize IDs (compares as-is)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-AAA111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &OrphanedDependencyCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false -- IDs should not be normalized")
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"tick-aaa111","blocked_by":["tick-missing"]}` + "\n")
		assertReadOnly(t, tickDir, content, func() {
			check := &OrphanedDependencyCheck{}
			check.Run(ctxWithTickDir(tickDir), tickDir)
		})
	})
}
