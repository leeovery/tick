package doctor

import (
	"context"
	"testing"
)

// stubCheck is a test double that returns preconfigured results.
type stubCheck struct {
	results []CheckResult
	called  bool
}

func (s *stubCheck) Run(_ context.Context, _ string) []CheckResult {
	s.called = true
	return s.results
}

func newPassingCheck(name string) *stubCheck {
	return &stubCheck{
		results: []CheckResult{
			{Name: name, Passed: true, Severity: SeverityError},
		},
	}
}

func newFailingCheck(name string, severity Severity) *stubCheck {
	return &stubCheck{
		results: []CheckResult{
			{Name: name, Passed: false, Severity: severity, Details: name + " failed", Suggestion: "Fix " + name},
		},
	}
}

func newMultiResultCheck(name string, count int) *stubCheck {
	results := make([]CheckResult, count)
	for i := range count {
		results[i] = CheckResult{
			Name:       name,
			Passed:     false,
			Severity:   SeverityError,
			Details:    name + " issue",
			Suggestion: "Fix it",
		}
	}
	return &stubCheck{results: results}
}

func TestDiagnosticRunner(t *testing.T) {
	t.Run("it returns empty report when zero checks are registered", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		report := runner.RunAll(context.Background(), "")

		if len(report.Results) != 0 {
			t.Errorf("expected 0 results, got %d", len(report.Results))
		}
		if report.HasErrors() {
			t.Error("expected HasErrors() to be false for empty report")
		}
		if report.ErrorCount() != 0 {
			t.Errorf("expected ErrorCount() 0, got %d", report.ErrorCount())
		}
		if report.WarningCount() != 0 {
			t.Errorf("expected WarningCount() 0, got %d", report.WarningCount())
		}
	})

	t.Run("it runs a single passing check and returns one result with Passed true", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		runner.Register(newPassingCheck("Cache"))
		report := runner.RunAll(context.Background(), "")

		if len(report.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(report.Results))
		}
		if !report.Results[0].Passed {
			t.Error("expected result to have Passed true")
		}
		if report.Results[0].Name != "Cache" {
			t.Errorf("expected name %q, got %q", "Cache", report.Results[0].Name)
		}
	})

	t.Run("it runs a single failing check and returns one result with Passed false", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		runner.Register(newFailingCheck("JSONL syntax", SeverityError))
		report := runner.RunAll(context.Background(), "")

		if len(report.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(report.Results))
		}
		if report.Results[0].Passed {
			t.Error("expected result to have Passed false")
		}
		if report.Results[0].Details == "" {
			t.Error("expected non-empty Details on failing check")
		}
		if report.Results[0].Suggestion == "" {
			t.Error("expected non-empty Suggestion on failing check")
		}
	})

	t.Run("it runs all checks when all pass — report has no errors", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		runner.Register(newPassingCheck("Cache"))
		runner.Register(newPassingCheck("JSONL syntax"))
		runner.Register(newPassingCheck("ID uniqueness"))
		report := runner.RunAll(context.Background(), "")

		if len(report.Results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(report.Results))
		}
		if report.HasErrors() {
			t.Error("expected HasErrors() to be false when all pass")
		}
	})

	t.Run("it runs all checks when all fail — report collects all failures", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		runner.Register(newFailingCheck("Cache", SeverityError))
		runner.Register(newFailingCheck("JSONL syntax", SeverityError))
		runner.Register(newFailingCheck("ID uniqueness", SeverityError))
		report := runner.RunAll(context.Background(), "")

		if len(report.Results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(report.Results))
		}
		if !report.HasErrors() {
			t.Error("expected HasErrors() to be true when all fail")
		}
		if report.ErrorCount() != 3 {
			t.Errorf("expected ErrorCount() 3, got %d", report.ErrorCount())
		}
	})

	t.Run("it runs all checks with mixed pass/fail — report contains both", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		runner.Register(newPassingCheck("Cache"))
		runner.Register(newFailingCheck("JSONL syntax", SeverityError))
		runner.Register(newPassingCheck("ID uniqueness"))
		report := runner.RunAll(context.Background(), "")

		if len(report.Results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(report.Results))
		}

		passCount := 0
		failCount := 0
		for _, r := range report.Results {
			if r.Passed {
				passCount++
			} else {
				failCount++
			}
		}
		if passCount != 2 {
			t.Errorf("expected 2 passing results, got %d", passCount)
		}
		if failCount != 1 {
			t.Errorf("expected 1 failing result, got %d", failCount)
		}
	})

	t.Run("it does not short-circuit — failing check does not prevent subsequent checks from running", func(t *testing.T) {
		first := newFailingCheck("Cache", SeverityError)
		second := newPassingCheck("JSONL syntax")
		third := newPassingCheck("ID uniqueness")

		runner := NewDiagnosticRunner()
		runner.Register(first)
		runner.Register(second)
		runner.Register(third)
		runner.RunAll(context.Background(), "")

		if !first.called {
			t.Error("expected first check to be called")
		}
		if !second.called {
			t.Error("expected second check to be called after first failure")
		}
		if !third.called {
			t.Error("expected third check to be called after first failure")
		}
	})

	t.Run("it preserves registration order in results", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		runner.Register(newPassingCheck("Alpha"))
		runner.Register(newPassingCheck("Beta"))
		runner.Register(newPassingCheck("Gamma"))
		report := runner.RunAll(context.Background(), "")

		expected := []string{"Alpha", "Beta", "Gamma"}
		if len(report.Results) != len(expected) {
			t.Fatalf("expected %d results, got %d", len(expected), len(report.Results))
		}
		for i, name := range expected {
			if report.Results[i].Name != name {
				t.Errorf("result[%d].Name = %q, want %q", i, report.Results[i].Name, name)
			}
		}
	})

	t.Run("it collects multiple results from a single check", func(t *testing.T) {
		runner := NewDiagnosticRunner()
		runner.Register(newMultiResultCheck("Orphaned references", 3))
		report := runner.RunAll(context.Background(), "")

		if len(report.Results) != 3 {
			t.Fatalf("expected 3 results from single check, got %d", len(report.Results))
		}
		for i, r := range report.Results {
			if r.Name != "Orphaned references" {
				t.Errorf("result[%d].Name = %q, want %q", i, r.Name, "Orphaned references")
			}
		}
	})
}

func TestDiagnosticReport(t *testing.T) {
	t.Run("HasErrors returns true when any error-severity result has Passed false", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
				{Name: "JSONL", Passed: false, Severity: SeverityError, Details: "bad"},
			},
		}
		if !report.HasErrors() {
			t.Error("expected HasErrors() true when error-severity result has Passed false")
		}
	})

	t.Run("HasErrors returns false when only warnings exist", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: true, Severity: SeverityError},
				{Name: "Parent done", Passed: false, Severity: SeverityWarning, Details: "suspicious"},
			},
		}
		if report.HasErrors() {
			t.Error("expected HasErrors() false when only warning-severity failures exist")
		}
	})

	t.Run("ErrorCount counts only error-severity failures", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: false, Severity: SeverityError, Details: "stale"},
				{Name: "Parent done", Passed: false, Severity: SeverityWarning, Details: "suspicious"},
				{Name: "JSONL", Passed: false, Severity: SeverityError, Details: "bad"},
				{Name: "IDs", Passed: true, Severity: SeverityError},
			},
		}
		if report.ErrorCount() != 2 {
			t.Errorf("expected ErrorCount() 2, got %d", report.ErrorCount())
		}
	})

	t.Run("WarningCount counts only warning-severity failures", func(t *testing.T) {
		report := DiagnosticReport{
			Results: []CheckResult{
				{Name: "Cache", Passed: false, Severity: SeverityError, Details: "stale"},
				{Name: "Parent done", Passed: false, Severity: SeverityWarning, Details: "suspicious"},
				{Name: "IDs", Passed: true, Severity: SeverityWarning},
			},
		}
		if report.WarningCount() != 1 {
			t.Errorf("expected WarningCount() 1, got %d", report.WarningCount())
		}
	})

	t.Run("HasErrors returns false for empty report", func(t *testing.T) {
		report := DiagnosticReport{}
		if report.HasErrors() {
			t.Error("expected HasErrors() false for empty report")
		}
		if report.ErrorCount() != 0 {
			t.Errorf("expected ErrorCount() 0, got %d", report.ErrorCount())
		}
		if report.WarningCount() != 0 {
			t.Errorf("expected WarningCount() 0, got %d", report.WarningCount())
		}
	})
}
