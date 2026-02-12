package doctor

import (
	"bytes"
	"testing"
)

func TestFormatReport(t *testing.T) {
	t.Run("it formats zero results as empty output with 'No issues found.' summary", func(t *testing.T) {
		report := DiagnosticReport{}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "No issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it formats a single passing check as '✓ {Name}: OK'", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✓ Cache: OK\n\nNo issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it formats multiple passing checks each on their own line", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
				{Name: "JSONL syntax", Passed: true, Severity: SeverityError},
				{Name: "ID uniqueness", Passed: true, Severity: SeverityError},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✓ Cache: OK\n✓ JSONL syntax: OK\n✓ ID uniqueness: OK\n\nNo issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it formats a single failing check as '✗ {Name}: {Details}' with suggestion on next line", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Orphaned reference", Passed: false, Severity: SeverityError, Details: "tick-a1b2c3 references non-existent parent tick-missing", Suggestion: "Manual fix required"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✗ Orphaned reference: tick-a1b2c3 references non-existent parent tick-missing\n  → Manual fix required\n\n1 issue found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it formats a failing check without suggestion — no suggestion line emitted", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "JSONL syntax", Passed: false, Severity: SeverityError, Details: "line 42: invalid JSON"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✗ JSONL syntax: line 42: invalid JSON\n\n1 issue found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it formats multiple failures from different checks each individually", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: false, Severity: SeverityError, Details: "stale", Suggestion: "Run `tick rebuild`"},
				{Name: "JSONL syntax", Passed: false, Severity: SeverityError, Details: "line 5: bad JSON", Suggestion: "Manual fix required"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✗ Cache: stale\n  → Run `tick rebuild`\n✗ JSONL syntax: line 5: bad JSON\n  → Manual fix required\n\n2 issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it formats multiple failures from one check as separate ✗ lines", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Orphaned reference", Passed: false, Severity: SeverityError, Details: "tick-aaa111 orphaned", Suggestion: "Fix it"},
				{Name: "Orphaned reference", Passed: false, Severity: SeverityError, Details: "tick-bbb222 orphaned", Suggestion: "Fix it"},
				{Name: "Orphaned reference", Passed: false, Severity: SeverityError, Details: "tick-ccc333 orphaned", Suggestion: "Fix it"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✗ Orphaned reference: tick-aaa111 orphaned\n  → Fix it\n✗ Orphaned reference: tick-bbb222 orphaned\n  → Fix it\n✗ Orphaned reference: tick-ccc333 orphaned\n  → Fix it\n\n3 issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it shows summary '1 issue found.' for exactly one failure", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
				{Name: "JSONL syntax", Passed: false, Severity: SeverityError, Details: "bad JSON"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✓ Cache: OK\n✗ JSONL syntax: bad JSON\n\n1 issue found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it shows summary '{N} issues found.' for multiple failures", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "A", Passed: false, Severity: SeverityError, Details: "fail a"},
				{Name: "B", Passed: false, Severity: SeverityError, Details: "fail b"},
				{Name: "C", Passed: false, Severity: SeverityWarning, Details: "fail c"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✗ A: fail a\n✗ B: fail b\n✗ C: fail c\n\n3 issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it shows summary 'No issues found.' when all checks pass", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
				{Name: "JSONL", Passed: true, Severity: SeverityError},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✓ Cache: OK\n✓ JSONL: OK\n\nNo issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it includes both errors and warnings in the summary issue count", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: false, Severity: SeverityError, Details: "stale"},
				{Name: "Parent done", Passed: false, Severity: SeverityWarning, Details: "suspicious"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✗ Cache: stale\n✗ Parent done: suspicious\n\n2 issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it preserves result order from DiagnosticReport in output", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Alpha", Passed: true, Severity: SeverityError},
				{Name: "Beta", Passed: false, Severity: SeverityError, Details: "broken"},
				{Name: "Gamma", Passed: true, Severity: SeverityError},
				{Name: "Delta", Passed: false, Severity: SeverityWarning, Details: "warn"},
			},
		}
		var buf bytes.Buffer

		FormatReport(&buf, report)

		want := "✓ Alpha: OK\n✗ Beta: broken\n✓ Gamma: OK\n✗ Delta: warn\n\n2 issues found.\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestExitCode(t *testing.T) {
	t.Run("exit code returns 0 when report has no errors (empty report)", func(t *testing.T) {
		report := DiagnosticReport{}

		if got := ExitCode(report); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("exit code returns 0 when report has only passing checks", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
				{Name: "JSONL", Passed: true, Severity: SeverityError},
			},
		}

		if got := ExitCode(report); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("exit code returns 0 when report has only warnings (no errors)", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
				{Name: "Parent done", Passed: false, Severity: SeverityWarning, Details: "suspicious"},
			},
		}

		if got := ExitCode(report); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("exit code returns 1 when report has at least one error-severity failure", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: false, Severity: SeverityError, Details: "stale"},
			},
		}

		if got := ExitCode(report); got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})

	t.Run("exit code returns 1 when report has mixed errors and warnings", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: false, Severity: SeverityError, Details: "stale"},
				{Name: "Parent done", Passed: false, Severity: SeverityWarning, Details: "suspicious"},
				{Name: "JSONL", Passed: true, Severity: SeverityError},
			},
		}

		if got := ExitCode(report); got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})
}
