package doctor

import (
	"testing"
)

func TestDuplicateSeqCheck(t *testing.T) {
	runSeqCheck := func(t *testing.T, content string) []CheckResult {
		t.Helper()
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(content))
		check := &DuplicateSeqCheck{}
		return check.Run(ctxWithTickDir(tickDir), tickDir)
	}

	assertSinglePass := func(t *testing.T, results []CheckResult) {
		t.Helper()
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
		}
		want := CheckResult{Name: "Sequence uniqueness", Passed: true}
		if results[0] != want {
			t.Errorf("expected %+v, got %+v", want, results[0])
		}
	}

	duplicateResult := func(details string) CheckResult {
		return CheckResult{
			Name:       "Sequence uniqueness",
			Passed:     false,
			Severity:   SeverityWarning,
			Details:    details,
			Suggestion: "Edit the seq values in tasks.jsonl so they differ",
		}
	}

	t.Run("it warns once naming both tasks and their lines when two records share a sequence", func(t *testing.T) {
		content := "{\"id\":\"tick-aaa111\",\"seq\":1}\n" +
			"{\"id\":\"tick-bbb222\",\"seq\":2}\n" +
			"{\"id\":\"tick-ccc333\",\"seq\":3}\n" +
			"{\"id\":\"tick-ddd444\",\"seq\":2}\n" +
			"{\"id\":\"tick-eee555\",\"seq\":4}\n"

		results := runSeqCheck(t, content)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
		}
		want := duplicateResult("Duplicate sequence 2: tick-bbb222 (line 2), tick-ddd444 (line 4)")
		if results[0] != want {
			t.Errorf("expected %+v, got %+v", want, results[0])
		}
	})

	t.Run("it reports three records sharing a sequence in a single result", func(t *testing.T) {
		content := "{\"id\":\"tick-aaa111\",\"seq\":5}\n" +
			"{\"id\":\"tick-bbb222\",\"seq\":5}\n" +
			"{\"id\":\"tick-ccc333\",\"seq\":6}\n" +
			"{\"id\":\"tick-ddd444\",\"seq\":5}\n"

		results := runSeqCheck(t, content)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
		}
		want := duplicateResult("Duplicate sequence 5: tick-aaa111 (line 1), tick-bbb222 (line 2), tick-ddd444 (line 4)")
		if results[0] != want {
			t.Errorf("expected %+v, got %+v", want, results[0])
		}
	})

	t.Run("it reports each shared sequence as its own warning with only its members' lines", func(t *testing.T) {
		content := "{\"id\":\"tick-aaa111\",\"seq\":3}\n" +
			"{\"id\":\"tick-bbb222\",\"seq\":7}\n" +
			"{\"id\":\"tick-ccc333\",\"seq\":3}\n" +
			"{\"id\":\"tick-ddd444\",\"seq\":1}\n" +
			"{\"id\":\"tick-eee555\",\"seq\":7}\n"

		results := runSeqCheck(t, content)

		want := []CheckResult{
			duplicateResult("Duplicate sequence 3: tick-aaa111 (line 1), tick-ccc333 (line 3)"),
			duplicateResult("Duplicate sequence 7: tick-bbb222 (line 2), tick-eee555 (line 5)"),
		}
		if len(results) != len(want) {
			t.Fatalf("expected %d results, got %d: %+v", len(want), len(results), results)
		}
		for i := range want {
			if results[i] != want[i] {
				t.Errorf("result %d: expected %+v, got %+v", i, want[i], results[i])
			}
		}
	})

	t.Run("it returns a single passing result when every sequence is distinct", func(t *testing.T) {
		content := "{\"id\":\"tick-aaa111\",\"seq\":1}\n" +
			"{\"id\":\"tick-bbb222\",\"seq\":2}\n" +
			"{\"id\":\"tick-ccc333\",\"seq\":3}\n"

		assertSinglePass(t, runSeqCheck(t, content))
	})

	t.Run("it returns a single passing result when no record carries a sequence", func(t *testing.T) {
		content := "{\"id\":\"tick-aaa111\"}\n" +
			"{\"id\":\"tick-bbb222\"}\n" +
			"{\"id\":\"tick-ccc333\"}\n"

		assertSinglePass(t, runSeqCheck(t, content))
	})

	t.Run("it does not compare records carrying a zero or absent sequence", func(t *testing.T) {
		content := "{\"id\":\"tick-aaa111\",\"seq\":0}\n" +
			"{\"id\":\"tick-bbb222\",\"seq\":1}\n" +
			"{\"id\":\"tick-ccc333\"}\n" +
			"{\"id\":\"tick-ddd444\",\"seq\":0}\n" +
			"{\"id\":\"tick-eee555\",\"seq\":2}\n" +
			"{\"id\":\"tick-fff666\"}\n"

		assertSinglePass(t, runSeqCheck(t, content))
	})

	t.Run("it returns a single passing result for an empty file", func(t *testing.T) {
		assertSinglePass(t, runSeqCheck(t, ""))
	})

	t.Run("it skips lines with invalid JSON", func(t *testing.T) {
		content := "{\"id\":\"tick-aaa111\",\"seq\":1}\n" +
			"not json {\"seq\":1}\n" +
			"{\"id\":\"tick-bbb222\",\"seq\":2}\n"

		assertSinglePass(t, runSeqCheck(t, content))
	})

	t.Run("it returns a failing error result when tasks.jsonl does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		check := &DuplicateSeqCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		want := CheckResult{
			Name:       "Sequence uniqueness",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}
		if results[0] != want {
			t.Errorf("expected %+v, got %+v", want, results[0])
		}
	})

	t.Run("it does not modify tasks.jsonl (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte("{\"id\":\"tick-aaa111\",\"seq\":4}\n{\"id\":\"tick-bbb222\",\"seq\":4}\n")
		assertReadOnly(t, tickDir, content, func() {
			check := &DuplicateSeqCheck{}
			results := check.Run(ctxWithTickDir(tickDir), tickDir)
			if len(results) != 1 || results[0].Passed {
				t.Fatalf("expected one failing result, got %+v", results)
			}
		})
	})
}
