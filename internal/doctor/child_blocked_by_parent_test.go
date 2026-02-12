package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChildBlockedByParentCheck(t *testing.T) {
	t.Run("it returns passing result when no child tasks are blocked by their parent", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

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

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when tasks have parents and blocked_by but no overlap", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-ccc333","tick-ddd444"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n" +
			`{"id":"tick-ddd444"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when task has parent but empty blocked_by (no dependencies)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":[]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when task has blocked_by but no parent (root task)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for root task; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when a child has its direct parent in blocked_by", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for child blocked by parent")
		}
	})

	t.Run("it returns failing result when child is blocked by parent among other valid deps", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-ccc333","tick-bbb222","tick-ddd444"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n" +
			`{"id":"tick-ddd444"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for child blocked by parent among other deps")
		}
	})

	t.Run("it returns one failing result per child when multiple children are blocked by the same parent", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-ccc333","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-ccc333","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
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

	t.Run("it does not flag child blocked by grandparent (only direct parent checked)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// tick-aaa111's parent is tick-bbb222, grandparent is tick-ccc333
		// tick-aaa111 is blocked by tick-ccc333 (grandparent) — should NOT be flagged
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-ccc333"}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true -- grandparent not flagged; details: %s", results[0].Details)
		}
	})

	t.Run("it includes child ID and parent ID in error details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-abc123","parent":"tick-def456","blocked_by":["tick-def456"]}` + "\n" +
			`{"id":"tick-def456"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "tick-abc123 is blocked by its parent tick-def456"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it follows wording 'tick-{child} is blocked by its parent tick-{parent}' in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-111aaa","parent":"tick-222bbb","blocked_by":["tick-222bbb"]}` + "\n" +
			`{"id":"tick-222bbb"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "tick-111aaa is blocked by its parent tick-222bbb"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it reports one error per child even if parent appears multiple times in blocked_by", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-bbb222","tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result (deduplicated), got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for child blocked by parent")
		}
	})

	t.Run("it skips unparseable lines - does not report them as child-blocked-by-parent", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-ccc333"]}` + "\n" +
			"not json\n" +
			`{"id":"tick-bbb222"}` + "\n" +
			`{"id":"tick-ccc333"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when tasks.jsonl does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)

		check := &ChildBlockedByParentCheck{}
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

	t.Run("it suggests fix mentioning deadlock with leaf-only ready rule", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "Manual fix required — child blocked by parent creates deadlock with leaf-only ready rule"
		if results[0].Suggestion != expected {
			t.Errorf("expected Suggestion %q, got %q", expected, results[0].Suggestion)
		}
	})

	t.Run("it uses CheckResult Name 'Child blocked by parent' for all results", func(t *testing.T) {
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
				name: "child blocked by parent",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-bbb222"]}` + "\n" +
						`{"id":"tick-bbb222"}` + "\n"
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
				check := &ChildBlockedByParentCheck{}
				results := check.Run(ctxWithTickDir(tickDir))

				for i, r := range results {
					if r.Name != "Child blocked by parent" {
						t.Errorf("result %d: expected Name %q, got %q", i, "Child blocked by parent", r.Name)
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
				name: "child blocked by parent",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					content := `{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-bbb222"]}` + "\n" +
						`{"id":"tick-bbb222"}` + "\n"
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
				check := &ChildBlockedByParentCheck{}
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

	t.Run("it does not normalize IDs before comparison (compares as-is)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Parent is tick-AAA111 (uppercase), blocked_by has tick-aaa111 (lowercase) — different as-is
		content := `{"id":"tick-aaa111","parent":"tick-AAA222","blocked_by":["tick-aaa222"]}` + "\n" +
			`{"id":"tick-AAA222"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ChildBlockedByParentCheck{}
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
		content := []byte(`{"id":"tick-aaa111","parent":"tick-bbb222","blocked_by":["tick-bbb222"]}` + "\n" +
			`{"id":"tick-bbb222"}` + "\n")
		writeJSONL(t, tickDir, content)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		before, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before: %v", err)
		}

		check := &ChildBlockedByParentCheck{}
		check.Run(ctxWithTickDir(tickDir))

		after, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after: %v", err)
		}
		if string(before) != string(after) {
			t.Error("tasks.jsonl was modified by ChildBlockedByParentCheck")
		}
	})
}
