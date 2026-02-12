package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParentDoneWithOpenChildrenCheck(t *testing.T) {
	t.Run("it returns passing result when no parent is done with open children", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"open"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
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

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty file; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when parent is done and all children are done", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-ccc333","parent":"tick-aaa111","status":"done"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when parent is done and all children are cancelled", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"cancelled"}` + "\n" +
			`{"id":"tick-ccc333","parent":"tick-aaa111","status":"cancelled"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when parent is done with mix of done and cancelled children (no open)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-ccc333","parent":"tick-aaa111","status":"cancelled"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when parent is open with open children (not flagged)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"open"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when parent is in_progress with open children (not flagged)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"in_progress"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result when parent is cancelled with open children (not flagged)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"cancelled"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true; details: %s", results[0].Details)
		}
	})

	t.Run("it returns warning result when parent is done with one open child (status open)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for done parent with open child")
		}
		if results[0].Severity != SeverityWarning {
			t.Errorf("expected Severity %q, got %q", SeverityWarning, results[0].Severity)
		}
	})

	t.Run("it returns warning result when parent is done with one in_progress child", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"in_progress"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for done parent with in_progress child")
		}
		if results[0].Severity != SeverityWarning {
			t.Errorf("expected Severity %q, got %q", SeverityWarning, results[0].Severity)
		}
	})

	t.Run("it returns one warning per open child when parent is done with multiple open children", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n" +
			`{"id":"tick-ccc333","parent":"tick-aaa111","status":"in_progress"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
			if r.Severity != SeverityWarning {
				t.Errorf("result %d: expected Severity %q, got %q", i, SeverityWarning, r.Severity)
			}
		}
	})

	t.Run("it returns warnings only for open children - done and cancelled children of same parent not flagged", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n" +
			`{"id":"tick-ccc333","parent":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-ddd444","parent":"tick-aaa111","status":"cancelled"}` + "\n" +
			`{"id":"tick-eee555","parent":"tick-aaa111","status":"in_progress"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 2 {
			t.Fatalf("expected 2 results (open + in_progress), got %d", len(results))
		}
		for i, r := range results {
			if r.Passed {
				t.Errorf("result %d: expected Passed false", i)
			}
		}
	})

	t.Run("it returns warnings across multiple parents - each done parent with open children produces own warnings", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n" +
			`{"id":"tick-ccc333","status":"done"}` + "\n" +
			`{"id":"tick-ddd444","parent":"tick-ccc333","status":"in_progress"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
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

	t.Run("it uses SeverityWarning for parent-done-with-open-children findings", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Severity != SeverityWarning {
			t.Errorf("expected Severity %q, got %q", SeverityWarning, results[0].Severity)
		}
	})

	t.Run("it uses SeverityError (not SeverityWarning) when tasks.jsonl does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for missing file")
		}
		if results[0].Severity != SeverityError {
			t.Errorf("expected Severity %q, got %q", SeverityError, results[0].Severity)
		}
	})

	t.Run("it includes parent ID and child ID in warning details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-parent1","status":"done"}` + "\n" +
			`{"id":"tick-child1","parent":"tick-parent1","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "tick-parent1 is done but has open child tick-child1"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it follows wording 'tick-{parent} is done but has open child tick-{child}' in details", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"in_progress"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "tick-aaa111 is done but has open child tick-bbb222"
		if results[0].Details != expected {
			t.Errorf("expected Details %q, got %q", expected, results[0].Details)
		}
	})

	t.Run("it suggests 'Review whether parent was completed prematurely' for warnings", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "Review whether parent was completed prematurely"
		if results[0].Suggestion != expected {
			t.Errorf("expected Suggestion %q, got %q", expected, results[0].Suggestion)
		}
	})

	t.Run("it uses CheckResult Name 'Parent done with open children' for all results", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "passing",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"tick-aaa111","status":"open"}`+"\n"))
					return tickDir
				},
			},
			{
				name: "warning",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
						`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
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
				check := &ParentDoneWithOpenChildrenCheck{}
				results := check.Run(ctxWithTickDir(tickDir))

				for i, r := range results {
					if r.Name != "Parent done with open children" {
						t.Errorf("result %d: expected Name %q, got %q", i, "Parent done with open children", r.Name)
					}
				}
			})
		}
	})

	t.Run("it skips parent IDs that do not exist as parseable tasks (orphaned parent - handled by task 3-1)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// tick-bbb222 references parent tick-missing which doesn't exist in data
		// The check should skip tick-missing as a parent since it's not in the status map
		content := `{"id":"tick-bbb222","parent":"tick-missing","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true -- orphaned parents skipped; details: %s", results[0].Details)
		}
	})

	t.Run("it skips unparseable lines - does not report them in parent-child analysis", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			"not json\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"done"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
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

		check := &ParentDoneWithOpenChildrenCheck{}
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

	t.Run("it does not normalize IDs before comparison (compares as-is)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// Parent tick-AAA111 (uppercase) exists with status done
		// Child references parent tick-aaa111 (lowercase) — different as-is
		// So parent "tick-aaa111" is not found in status map → skipped
		content := `{"id":"tick-AAA111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		check := &ParentDoneWithOpenChildrenCheck{}
		results := check.Run(ctxWithTickDir(tickDir))

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true -- IDs not normalized, parent not found; details: %s", results[0].Details)
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n")
		writeJSONL(t, tickDir, content)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		before, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before: %v", err)
		}

		check := &ParentDoneWithOpenChildrenCheck{}
		check.Run(ctxWithTickDir(tickDir))

		after, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after: %v", err)
		}
		if string(before) != string(after) {
			t.Error("tasks.jsonl was modified by ParentDoneWithOpenChildrenCheck")
		}
	})

	t.Run("warnings-only report produces exit code 0 when combined with DiagnosticRunner (integration)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := `{"id":"tick-aaa111","status":"done"}` + "\n" +
			`{"id":"tick-bbb222","parent":"tick-aaa111","status":"open"}` + "\n"
		writeJSONL(t, tickDir, []byte(content))

		runner := NewDiagnosticRunner()
		runner.Register(&ParentDoneWithOpenChildrenCheck{})

		ctx := context.WithValue(context.Background(), TickDirKey, tickDir)
		report := runner.RunAll(ctx)

		if report.HasErrors() {
			t.Error("expected HasErrors false for warnings-only report")
		}
		if got := ExitCode(report); got != 0 {
			t.Errorf("expected exit code 0 for warnings-only, got %d", got)
		}
		if report.WarningCount() != 1 {
			t.Errorf("expected 1 warning, got %d", report.WarningCount())
		}
	})
}
