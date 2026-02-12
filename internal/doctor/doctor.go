// Package doctor provides diagnostic checks for the tick data store.
// It defines the check interface, result types, and a runner that executes
// all registered checks without short-circuiting.
package doctor

import "context"

// Severity indicates whether a check failure is an error or a warning.
// Errors affect exit code; warnings do not.
type Severity string

const (
	// SeverityError indicates a failure that breaks tick and affects exit code.
	SeverityError Severity = "error"
	// SeverityWarning indicates a suspicious but allowed state that does not affect exit code.
	SeverityWarning Severity = "warning"
)

// CheckResult holds the outcome of a single diagnostic check evaluation.
// A passing check has Passed true with empty Details and Suggestion.
// A failing check has Passed false with a human-readable Details description
// and an actionable Suggestion for remediation.
type CheckResult struct {
	// Name is the check's display label (e.g. "Cache", "JSONL syntax").
	Name string
	// Passed indicates whether this check evaluation passed.
	Passed bool
	// Severity indicates whether this result is an error or warning.
	Severity Severity
	// Details is a human-readable description of what is wrong. Empty when passed.
	Details string
	// Suggestion is actionable fix text. Empty when passed or when no suggestion applies.
	Suggestion string
}

// Check is the interface that all diagnostic checks implement.
// Run executes the check and returns one or more results.
// A passing check returns exactly one result with Passed true.
// A failing check returns one or more results with Passed false.
type Check interface {
	Run(ctx context.Context) []CheckResult
}

// DiagnosticReport collects all check results from a diagnostic run.
type DiagnosticReport struct {
	// Results contains all CheckResult entries in registration order.
	Results []CheckResult
}

// HasErrors returns true if any result has Passed false with SeverityError.
func (r *DiagnosticReport) HasErrors() bool {
	for _, result := range r.Results {
		if !result.Passed && result.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of results with Passed false and SeverityError.
func (r *DiagnosticReport) ErrorCount() int {
	count := 0
	for _, result := range r.Results {
		if !result.Passed && result.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of results with Passed false and SeverityWarning.
func (r *DiagnosticReport) WarningCount() int {
	count := 0
	for _, result := range r.Results {
		if !result.Passed && result.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// DiagnosticRunner holds an ordered slice of Check implementations
// and executes all of them, collecting results into a DiagnosticReport.
type DiagnosticRunner struct {
	checks []Check
}

// NewDiagnosticRunner creates a DiagnosticRunner with no registered checks.
func NewDiagnosticRunner() *DiagnosticRunner {
	return &DiagnosticRunner{}
}

// Register appends a check to the runner's ordered slice.
func (d *DiagnosticRunner) Register(check Check) {
	d.checks = append(d.checks, check)
}

// RunAll executes every registered check and collects all CheckResult entries
// into a DiagnosticReport. It never short-circuits — all checks run regardless
// of prior failures. With zero registered checks, it returns an empty report.
func (d *DiagnosticRunner) RunAll(ctx context.Context) DiagnosticReport {
	var results []CheckResult
	for _, check := range d.checks {
		results = append(results, check.Run(ctx)...)
	}
	return DiagnosticReport{Results: results}
}
