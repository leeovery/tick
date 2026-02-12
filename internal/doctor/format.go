package doctor

import (
	"fmt"
	"io"
)

// FormatReport writes a human-readable representation of the DiagnosticReport
// to the provided writer. Each result is formatted as a pass (✓) or fail (✗)
// line, followed by a summary count of issues.
func FormatReport(w io.Writer, report DiagnosticReport) {
	issueCount := 0

	for _, r := range report.Results {
		if r.Passed {
			fmt.Fprintf(w, "✓ %s: OK\n", r.Name)
		} else {
			fmt.Fprintf(w, "✗ %s: %s\n", r.Name, r.Details)
			if r.Suggestion != "" {
				fmt.Fprintf(w, "  → %s\n", r.Suggestion)
			}
			issueCount++
		}
	}

	if len(report.Results) > 0 {
		fmt.Fprint(w, "\n")
	}

	switch issueCount {
	case 0:
		fmt.Fprint(w, "No issues found.\n")
	case 1:
		fmt.Fprint(w, "1 issue found.\n")
	default:
		fmt.Fprintf(w, "%d issues found.\n", issueCount)
	}
}

// ExitCode returns the process exit code for a diagnostic run.
// It returns 0 when the report has no error-severity failures (warnings allowed),
// and 1 when any error-severity failure exists.
func ExitCode(report DiagnosticReport) int {
	if report.HasErrors() {
		return 1
	}
	return 0
}
