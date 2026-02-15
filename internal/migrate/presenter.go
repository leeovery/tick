package migrate

import (
	"fmt"
	"io"
)

// WriteHeader prints the migration header line identifying the source provider.
func WriteHeader(w io.Writer, providerName string) {
	fmt.Fprintf(w, "Importing from %s...\n", providerName)
}

// WriteResult prints a single migration result as an indented task line.
// Successful results display a checkmark; unsuccessful results are skipped
// (Phase 2 will render failure markers).
func WriteResult(w io.Writer, r Result) {
	if !r.Success {
		return
	}
	fmt.Fprintf(w, "  \u2713 Task: %s\n", r.Title)
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

// Present renders the complete migration output: header, per-task lines, and summary.
func Present(w io.Writer, providerName string, results []Result) {
	WriteHeader(w, providerName)
	for _, r := range results {
		WriteResult(w, r)
	}
	WriteSummary(w, results)
}
