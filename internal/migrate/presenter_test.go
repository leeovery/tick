package migrate

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestWriteHeader(t *testing.T) {
	t.Run("prints Importing from <provider>... with provider name", func(t *testing.T) {
		var buf bytes.Buffer
		WriteHeader(&buf, "beads", false)

		got := buf.String()
		want := "Importing from beads...\n"
		if got != want {
			t.Errorf("WriteHeader() = %q, want %q", got, want)
		}
	})

	t.Run("provider name in header matches what provider.Name() returns", func(t *testing.T) {
		var buf bytes.Buffer
		providerName := "jira"
		WriteHeader(&buf, providerName, false)

		got := buf.String()
		want := "Importing from jira...\n"
		if got != want {
			t.Errorf("WriteHeader() = %q, want %q", got, want)
		}
	})

	t.Run("dry-run header prints Importing from <provider>... [dry-run]", func(t *testing.T) {
		var buf bytes.Buffer
		WriteHeader(&buf, "beads", true)

		got := buf.String()
		want := "Importing from beads... [dry-run]\n"
		if got != want {
			t.Errorf("WriteHeader() = %q, want %q", got, want)
		}
	})

	t.Run("non-dry-run header does not include [dry-run]", func(t *testing.T) {
		var buf bytes.Buffer
		WriteHeader(&buf, "beads", false)

		got := buf.String()
		if strings.Contains(got, "[dry-run]") {
			t.Errorf("WriteHeader(dryRun=false) should not contain [dry-run], got %q", got)
		}
	})
}

func TestWriteResult(t *testing.T) {
	t.Run("prints checkmark and task title for successful result", func(t *testing.T) {
		var buf bytes.Buffer
		r := Result{Title: "Implement login flow", Success: true}
		WriteResult(&buf, r)

		got := buf.String()
		want := "  \u2713 Task: Implement login flow\n"
		if got != want {
			t.Errorf("WriteResult() = %q, want %q", got, want)
		}
	})

	t.Run("per-task lines are indented with two spaces", func(t *testing.T) {
		var buf bytes.Buffer
		r := Result{Title: "Some task", Success: true}
		WriteResult(&buf, r)

		got := buf.String()
		if !strings.HasPrefix(got, "  ") {
			t.Errorf("expected line to start with two spaces, got %q", got)
		}
	})

	t.Run("prints cross mark and skip reason for failed result", func(t *testing.T) {
		var buf bytes.Buffer
		r := Result{Title: "Broken entry", Success: false, Err: fmt.Errorf("missing title")}
		WriteResult(&buf, r)

		got := buf.String()
		want := "  ✗ Task: Broken entry (skipped: missing title)\n"
		if got != want {
			t.Errorf("WriteResult() = %q, want %q", got, want)
		}
	})

	t.Run("prints checkmark for successful result", func(t *testing.T) {
		var buf bytes.Buffer
		r := Result{Title: "Good task", Success: true}
		WriteResult(&buf, r)

		got := buf.String()
		want := "  ✓ Task: Good task\n"
		if got != want {
			t.Errorf("WriteResult() = %q, want %q", got, want)
		}
	})

	t.Run("prints fallback title when failed result has empty title", func(t *testing.T) {
		var buf bytes.Buffer
		r := Result{Title: "", Success: false, Err: fmt.Errorf("missing title")}
		WriteResult(&buf, r)

		got := buf.String()
		want := "  ✗ Task: (untitled) (skipped: missing title)\n"
		if got != want {
			t.Errorf("WriteResult() = %q, want %q", got, want)
		}
	})

	t.Run("long titles are printed in full without truncation", func(t *testing.T) {
		var buf bytes.Buffer
		longTitle := strings.Repeat("A", 250)
		r := Result{Title: longTitle, Success: true}
		WriteResult(&buf, r)

		got := buf.String()
		if !strings.Contains(got, longTitle) {
			t.Errorf("expected full long title in output, got length %d", len(got))
		}
	})
}

func TestWriteSummary(t *testing.T) {
	t.Run("prints Done: N imported, 0 failed with correct count", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: true},
			{Title: "B", Success: true},
			{Title: "C", Success: true},
		}
		WriteSummary(&buf, results)

		got := buf.String()
		want := "\nDone: 3 imported, 0 failed\n"
		if got != want {
			t.Errorf("WriteSummary() = %q, want %q", got, want)
		}
	})

	t.Run("counts only successful results for imported number", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: true},
			{Title: "B", Success: false},
			{Title: "C", Success: true},
		}
		WriteSummary(&buf, results)

		got := buf.String()
		want := "\nDone: 2 imported, 1 failed\n"
		if got != want {
			t.Errorf("WriteSummary() = %q, want %q", got, want)
		}
	})

	t.Run("counts failures correctly in summary line", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: true},
			{Title: "B", Success: false, Err: fmt.Errorf("bad")},
			{Title: "C", Success: false, Err: fmt.Errorf("bad")},
			{Title: "D", Success: true},
		}
		WriteSummary(&buf, results)

		got := buf.String()
		want := "\nDone: 2 imported, 2 failed\n"
		if got != want {
			t.Errorf("WriteSummary() = %q, want %q", got, want)
		}
	})

	t.Run("prints Done: 2 imported, 1 failed for mixed results", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: true},
			{Title: "B", Success: true},
			{Title: "C", Success: false, Err: fmt.Errorf("bad")},
		}
		WriteSummary(&buf, results)

		got := buf.String()
		want := "\nDone: 2 imported, 1 failed\n"
		if got != want {
			t.Errorf("WriteSummary() = %q, want %q", got, want)
		}
	})

	t.Run("prints Done: 0 imported, 3 failed when all tasks fail", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: false, Err: fmt.Errorf("bad")},
			{Title: "B", Success: false, Err: fmt.Errorf("bad")},
			{Title: "C", Success: false, Err: fmt.Errorf("bad")},
		}
		WriteSummary(&buf, results)

		got := buf.String()
		want := "\nDone: 0 imported, 3 failed\n"
		if got != want {
			t.Errorf("WriteSummary() = %q, want %q", got, want)
		}
	})

	t.Run("prints Done: 3 imported, 0 failed when no failures", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: true},
			{Title: "B", Success: true},
			{Title: "C", Success: true},
		}
		WriteSummary(&buf, results)

		got := buf.String()
		want := "\nDone: 3 imported, 0 failed\n"
		if got != want {
			t.Errorf("WriteSummary() = %q, want %q", got, want)
		}
	})
}

func TestWriteFailures(t *testing.T) {
	t.Run("prints failure detail section with each failure listed", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: true},
			{Title: "foo", Success: false, Err: fmt.Errorf("Missing required field")},
			{Title: "bar", Success: false, Err: fmt.Errorf("Invalid date format")},
		}
		WriteFailures(&buf, results)

		got := buf.String()
		want := "\nFailures:\n" +
			"- Task \"foo\": Missing required field\n" +
			"- Task \"bar\": Invalid date format\n"
		if got != want {
			t.Errorf("WriteFailures() = %q, want %q", got, want)
		}
	})

	t.Run("prints nothing when there are zero failures", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "A", Success: true},
			{Title: "B", Success: true},
		}
		WriteFailures(&buf, results)

		got := buf.String()
		if got != "" {
			t.Errorf("WriteFailures() = %q, want empty string", got)
		}
	})

	t.Run("uses fallback title for failures with empty title", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "", Success: false, Err: fmt.Errorf("missing title")},
		}
		WriteFailures(&buf, results)

		got := buf.String()
		want := "\nFailures:\n" +
			"- Task \"(untitled)\": missing title\n"
		if got != want {
			t.Errorf("WriteFailures() = %q, want %q", got, want)
		}
	})

	t.Run("preserves special characters in failure reason", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "task", Success: false, Err: fmt.Errorf("field <name> has \"quotes\" & symbols")},
		}
		WriteFailures(&buf, results)

		got := buf.String()
		want := "\nFailures:\n" +
			"- Task \"task\": field <name> has \"quotes\" & symbols\n"
		if got != want {
			t.Errorf("WriteFailures() = %q, want %q", got, want)
		}
	})
}

func TestPresent(t *testing.T) {
	t.Run("renders full output: header, per-task lines, blank line, summary", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Implement login flow", Success: true},
			{Title: "Fix database connection", Success: true},
			{Title: "Add unit tests", Success: true},
		}
		Present(&buf, "beads", false, results)

		got := buf.String()
		want := "Importing from beads...\n" +
			"  \u2713 Task: Implement login flow\n" +
			"  \u2713 Task: Fix database connection\n" +
			"  \u2713 Task: Add unit tests\n" +
			"\n" +
			"Done: 3 imported, 0 failed\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("with zero results prints header and summary with zero counts", func(t *testing.T) {
		var buf bytes.Buffer
		Present(&buf, "beads", false, []Result{})

		got := buf.String()
		want := "Importing from beads...\n" +
			"\n" +
			"Done: 0 imported, 0 failed\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("with single result prints one task line and count of 1", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Only task", Success: true},
		}
		Present(&buf, "beads", false, results)

		got := buf.String()
		want := "Importing from beads...\n" +
			"  \u2713 Task: Only task\n" +
			"\n" +
			"Done: 1 imported, 0 failed\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("with multiple results prints each task on its own line", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Task A", Success: true},
			{Title: "Task B", Success: true},
			{Title: "Task C", Success: true},
			{Title: "Task D", Success: true},
		}
		Present(&buf, "test", false, results)

		got := buf.String()
		lines := strings.Split(got, "\n")
		// header, 4 task lines, blank line, summary, trailing newline = 8 parts
		taskLines := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "  \u2713 Task:") {
				taskLines++
			}
		}
		if taskLines != 4 {
			t.Errorf("expected 4 task lines, got %d", taskLines)
		}
	})

	t.Run("summary line is separated from task lines by a blank line", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Task A", Success: true},
		}
		Present(&buf, "beads", false, results)

		got := buf.String()
		// The output should contain the pattern: task line\n\nDone:
		if !strings.Contains(got, "Task A\n\nDone:") {
			t.Errorf("expected blank line between task lines and summary, got:\n%s", got)
		}
	})

	t.Run("renders full output with failures: header, results, summary, failure detail", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Implement login flow", Success: true},
			{Title: "Fix database connection", Success: true},
			{Title: "Broken entry", Success: false, Err: fmt.Errorf("missing title")},
		}
		Present(&buf, "beads", false, results)

		got := buf.String()
		want := "Importing from beads...\n" +
			"  ✓ Task: Implement login flow\n" +
			"  ✓ Task: Fix database connection\n" +
			"  ✗ Task: Broken entry (skipped: missing title)\n" +
			"\n" +
			"Done: 2 imported, 1 failed\n" +
			"\n" +
			"Failures:\n" +
			"- Task \"Broken entry\": missing title\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("omits failure detail section when all results are successful", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Task A", Success: true},
			{Title: "Task B", Success: true},
		}
		Present(&buf, "beads", false, results)

		got := buf.String()
		want := "Importing from beads...\n" +
			"  ✓ Task: Task A\n" +
			"  ✓ Task: Task B\n" +
			"\n" +
			"Done: 2 imported, 0 failed\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("with all failures shows zero imported and failure detail section", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "foo", Success: false, Err: fmt.Errorf("bad data")},
			{Title: "bar", Success: false, Err: fmt.Errorf("invalid format")},
		}
		Present(&buf, "beads", false, results)

		got := buf.String()
		want := "Importing from beads...\n" +
			"  ✗ Task: foo (skipped: bad data)\n" +
			"  ✗ Task: bar (skipped: invalid format)\n" +
			"\n" +
			"Done: 0 imported, 2 failed\n" +
			"\n" +
			"Failures:\n" +
			"- Task \"foo\": bad data\n" +
			"- Task \"bar\": invalid format\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("dry-run with zero tasks prints header with [dry-run] and summary with zero counts", func(t *testing.T) {
		var buf bytes.Buffer
		Present(&buf, "beads", true, []Result{})

		got := buf.String()
		want := "Importing from beads... [dry-run]\n" +
			"\n" +
			"Done: 0 imported, 0 failed\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("dry-run with multiple tasks shows all as successful with checkmark", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Task A", Success: true},
			{Title: "Task B", Success: true},
			{Title: "Task C", Success: true},
		}
		Present(&buf, "beads", true, results)

		got := buf.String()
		want := "Importing from beads... [dry-run]\n" +
			"  \u2713 Task: Task A\n" +
			"  \u2713 Task: Task B\n" +
			"  \u2713 Task: Task C\n" +
			"\n" +
			"Done: 3 imported, 0 failed\n"
		if got != want {
			t.Errorf("Present() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("dry-run summary shows correct imported count matching number of valid tasks", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Task A", Success: true},
			{Title: "Task B", Success: true},
			{Title: "(untitled)", Success: false, Err: fmt.Errorf("title is required")},
		}
		Present(&buf, "beads", true, results)

		got := buf.String()
		if !strings.Contains(got, "Done: 2 imported, 1 failed") {
			t.Errorf("expected summary with 2 imported, 1 failed, got:\n%s", got)
		}
	})
}
