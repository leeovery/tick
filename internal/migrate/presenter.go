package migrate

import (
	"fmt"
	"io"
)

// WriteHeader prints the migration header line identifying the source provider.
// When dryRun is true, a [dry-run] indicator is appended to the header.
func WriteHeader(w io.Writer, providerName string, dryRun bool) {
	if dryRun {
		fmt.Fprintf(w, "Importing from %s... [dry-run]\n", providerName)
		return
	}
	fmt.Fprintf(w, "Importing from %s...\n", providerName)
}

// WriteResult prints a single migration result as an indented task line.
// Successful results display a checkmark; failed results display a cross mark
// with the skip reason inline.
func WriteResult(w io.Writer, r Result) {
	if r.Success {
		fmt.Fprintf(w, "  \u2713 Task: %s\n", r.Title)
		return
	}
	title := r.Title
	if title == "" {
		title = "(unknown)"
	}
	fmt.Fprintf(w, "  \u2717 Task: %s (skipped: %s)\n", title, r.Err.Error())
}

// WriteSummary prints the summary line showing imported and failed counts,
// preceded by a blank line to separate it from the per-task output.
func WriteSummary(w io.Writer, results []Result) {
	imported := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			imported++
		} else {
			failed++
		}
	}
	fmt.Fprintf(w, "\nDone: %d imported, %d failed\n", imported, failed)
}

// WriteFailures prints the failure detail section after the summary.
// Each failed result is listed with its title and error reason.
// If there are no failures, nothing is printed.
func WriteFailures(w io.Writer, results []Result) {
	var failures []Result
	for _, r := range results {
		if !r.Success {
			failures = append(failures, r)
		}
	}
	if len(failures) == 0 {
		return
	}
	fmt.Fprintf(w, "\nFailures:\n")
	for _, r := range failures {
		title := r.Title
		if title == "" {
			title = "(unknown)"
		}
		fmt.Fprintf(w, "- Task %q: %s\n", title, r.Err.Error())
	}
}

// Present renders the complete migration output: header, per-task lines, summary,
// and failure detail section (when failures exist). When dryRun is true, the
// header includes a [dry-run] indicator.
func Present(w io.Writer, providerName string, dryRun bool, results []Result) {
	WriteHeader(w, providerName, dryRun)
	for _, r := range results {
		WriteResult(w, r)
	}
	WriteSummary(w, results)
	WriteFailures(w, results)
}
