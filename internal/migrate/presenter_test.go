package migrate

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteHeader(t *testing.T) {
	t.Run("prints Importing from <provider>... with provider name", func(t *testing.T) {
		var buf bytes.Buffer
		WriteHeader(&buf, "beads")

		got := buf.String()
		want := "Importing from beads...\n"
		if got != want {
			t.Errorf("WriteHeader() = %q, want %q", got, want)
		}
	})

	t.Run("provider name in header matches what provider.Name() returns", func(t *testing.T) {
		var buf bytes.Buffer
		providerName := "jira"
		WriteHeader(&buf, providerName)

		got := buf.String()
		want := "Importing from jira...\n"
		if got != want {
			t.Errorf("WriteHeader() = %q, want %q", got, want)
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

	t.Run("does not print checkmark for unsuccessful result", func(t *testing.T) {
		var buf bytes.Buffer
		r := Result{Title: "Failed task", Success: false}
		WriteResult(&buf, r)

		got := buf.String()
		if strings.Contains(got, "\u2713") {
			t.Errorf("expected no checkmark for unsuccessful result, got %q", got)
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
}

func TestPresent(t *testing.T) {
	t.Run("renders full output: header, per-task lines, blank line, summary", func(t *testing.T) {
		var buf bytes.Buffer
		results := []Result{
			{Title: "Implement login flow", Success: true},
			{Title: "Fix database connection", Success: true},
			{Title: "Add unit tests", Success: true},
		}
		Present(&buf, "beads", results)

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
		Present(&buf, "beads", []Result{})

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
		Present(&buf, "beads", results)

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
		Present(&buf, "test", results)

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
		Present(&buf, "beads", results)

		got := buf.String()
		// The output should contain the pattern: task line\n\nDone:
		if !strings.Contains(got, "Task A\n\nDone:") {
			t.Errorf("expected blank line between task lines and summary, got:\n%s", got)
		}
	})
}
