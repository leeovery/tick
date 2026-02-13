package doctor

import (
	"testing"
)

func TestDependencyCycleCheck(t *testing.T) {
	t.Run("it returns passing result when no dependency cycles exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result for empty file (zero bytes)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte{})

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when tasks have dependencies but no cycles (valid DAG)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111"}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n" +
			`{"id":"tick-ccc333","blocked_by":["tick-bbb222"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for valid DAG; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when tasks have no dependencies (all empty blocked_by)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":[]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for no dependencies; details: %s", results[0].Details)
		}
	})

	t.Run("it detects a simple 2-node cycle (A depends on B, B depends on A)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for 2-node cycle")
		}
		expected := "Dependency cycle: tick-aaa111 \u2192 tick-bbb222 \u2192 tick-aaa111"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it detects a 3-node cycle (A\u2192B\u2192C\u2192A)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-ccc333","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for 3-node cycle")
		}
		expected := "Dependency cycle: tick-aaa111 \u2192 tick-bbb222 \u2192 tick-ccc333 \u2192 tick-aaa111"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it detects a longer cycle (4+ nodes)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-ccc333","blocked_by":["tick-ddd444"]}` + "\n" +
			`{"id":"tick-ddd444","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for 4-node cycle")
		}
		expected := "Dependency cycle: tick-aaa111 \u2192 tick-bbb222 \u2192 tick-ccc333 \u2192 tick-ddd444 \u2192 tick-aaa111"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it detects multiple independent cycles in the same graph", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n" +
			`{"id":"tick-ccc333","blocked_by":["tick-ddd444"]}` + "\n" +
			`{"id":"tick-ddd444","blocked_by":["tick-ccc333"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
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

	t.Run("it does not report a chain that is not a cycle (A\u2192B\u2192C with no back-edge)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for chain without cycle; details: %s", results[0].Details)
		}
	})

	t.Run("it does not report self-references (handled by task 3-3)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true -- self-references excluded; details: %s", results[0].Details)
		}
	})

	t.Run("it excludes self-references from adjacency list to avoid false cycle detection", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// A has self-ref and depends on B, B depends on A — only A→B→A is a real cycle
		content := `{"id":"tick-aaa111","blocked_by":["tick-aaa111","tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result (self-ref excluded from graph), got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for the real 2-node cycle")
		}
		expected := "Dependency cycle: tick-aaa111 \u2192 tick-bbb222 \u2192 tick-aaa111"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it excludes orphaned dependency references from adjacency list (non-existent targets)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// A depends on non-existent tick-missing — should not affect cycle detection
		content := `{"id":"tick-aaa111","blocked_by":["tick-missing"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true -- orphaned refs excluded; details: %s", results[0].Details)
		}
	})

	t.Run("it handles complex graph with both cycles and valid chains - reports only cycles", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Cycle: aaa→bbb→ccc→aaa
		// Valid chain: ddd→eee→fff (no cycle)
		// Another cycle: ggg→hhh→ggg
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-ccc333","blocked_by":["tick-aaa111"]}` + "\n" +
			`{"id":"tick-ddd444","blocked_by":["tick-eee555"]}` + "\n" +
			`{"id":"tick-eee555","blocked_by":["tick-fff666"]}` + "\n" +
			`{"id":"tick-fff666"}` + "\n" +
			`{"id":"tick-ggg777","blocked_by":["tick-hhh888"]}` + "\n" +
			`{"id":"tick-hhh888","blocked_by":["tick-ggg777"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 2 {
			t.Fatalf("expected 2 results (two cycles), got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
		}
	})

	t.Run("it reports each unique cycle as a separate failing CheckResult", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n" +
			`{"id":"tick-ccc333","blocked_by":["tick-ddd444"]}` + "\n" +
			`{"id":"tick-ddd444","blocked_by":["tick-ccc333"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
			if r.Name != "Dependency cycles" {
				t.Errorf("result %d: expected Name %q, got %q", i, "Dependency cycles", r.Name)
			}
		}
	})

	t.Run("it deduplicates cycles found from different starting nodes", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// A→B→A is the same cycle whether discovered starting from A or B
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result (deduplicated), got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for cycle")
		}
	})

	t.Run("it formats cycle details as 'Dependency cycle: tick-A \u2192 tick-B \u2192 tick-C \u2192 tick-A'", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-ccc333","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "Dependency cycle: tick-aaa111 \u2192 tick-bbb222 \u2192 tick-ccc333 \u2192 tick-aaa111"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it uses deterministic ordering in cycle output (lexicographically smallest ID first)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Write tasks in non-alphabetical order, cycle should still normalize to tick-aaa111 first
		content := `{"id":"tick-ccc333","blocked_by":["tick-aaa111"]}` + "\n" +
			`{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "Dependency cycle: tick-aaa111 \u2192 tick-bbb222 \u2192 tick-ccc333 \u2192 tick-aaa111"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it skips unparseable lines - does not include them in dependency graph", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			"not json\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
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

		check := &DependencyCycleCheck{}
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

	t.Run("it suggests 'Manual fix required' for cycle errors", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &DependencyCycleCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Suggestion != "Manual fix required" {
			t.Errorf("expected Suggestion %q, got %q", "Manual fix required", results[0].Suggestion)
		}
	})

	t.Run("it uses CheckResult Name 'Dependency cycles' for all results", func(t *testing.T) {
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
				name: "cycle",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
						`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
					writeJSONL(t, tickDir, []byte(content))
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
				check := &DependencyCycleCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

				for i, r := range results {
					if r.Name != "Dependency cycles" {
						t.Errorf("result %d: expected Name %q, got %q", i, "Dependency cycles", r.Name)
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
				name: "cycle",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
						`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n"
					writeJSONL(t, tickDir, []byte(content))
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
				check := &DependencyCycleCheck{}
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

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222","blocked_by":["tick-aaa111"]}` + "\n")
		assertReadOnly(t, tickDir, content, func() {
			check := &DependencyCycleCheck{}
			check.Run(ctxWithTickDir(tickDir), tickDir)
		})
	})
}
