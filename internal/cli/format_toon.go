package cli

import (
	"fmt"
	"io"
	"strings"
)

// ToonFormatter formats output in TOON (Token-Oriented Object Notation).
// Optimized for AI agent consumption with schema headers and compact rows.
type ToonFormatter struct{}

// FormatTaskList formats a list of tasks in TOON array format.
func (f *ToonFormatter) FormatTaskList(w io.Writer, tasks []TaskListItem) error {
	fmt.Fprintf(w, "tasks[%d]{id,title,status,priority}:\n", len(tasks))
	for _, t := range tasks {
		fmt.Fprintf(w, "  %s,%s,%s,%d\n", t.ID, toonEscape(t.Title), t.Status, t.Priority)
	}
	return nil
}

// FormatTaskDetail formats full task detail as a multi-section TOON document.
func (f *ToonFormatter) FormatTaskDetail(w io.Writer, d TaskDetail) error {
	// Build dynamic schema for task section.
	fields := []string{"id", "title", "status", "priority"}
	values := []string{d.ID, toonEscape(d.Title), d.Status, fmt.Sprintf("%d", d.Priority)}

	if d.Parent != nil {
		fields = append(fields, "parent")
		values = append(values, d.Parent.ID)
	}
	fields = append(fields, "created", "updated")
	values = append(values, d.Created, d.Updated)
	if d.Closed != "" {
		fields = append(fields, "closed")
		values = append(values, d.Closed)
	}

	fmt.Fprintf(w, "task{%s}:\n", strings.Join(fields, ","))
	fmt.Fprintf(w, "  %s\n", strings.Join(values, ","))

	// blocked_by section — always present.
	fmt.Fprintf(w, "\nblocked_by[%d]{id,title,status}:\n", len(d.BlockedBy))
	for _, b := range d.BlockedBy {
		fmt.Fprintf(w, "  %s,%s,%s\n", b.ID, toonEscape(b.Title), b.Status)
	}

	// children section — always present.
	fmt.Fprintf(w, "\nchildren[%d]{id,title,status}:\n", len(d.Children))
	for _, c := range d.Children {
		fmt.Fprintf(w, "  %s,%s,%s\n", c.ID, toonEscape(c.Title), c.Status)
	}

	// description section — omitted when empty.
	if d.Description != "" {
		fmt.Fprint(w, "\ndescription:\n")
		for _, line := range strings.Split(d.Description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	return nil
}

// FormatTransition formats a status transition as plain text.
func (f *ToonFormatter) FormatTransition(w io.Writer, data TransitionData) error {
	_, err := fmt.Fprintf(w, "%s: %s → %s\n", data.ID, data.OldStatus, data.NewStatus)
	return err
}

// FormatDepChange formats a dependency change as plain text.
func (f *ToonFormatter) FormatDepChange(w io.Writer, data DepChangeData) error {
	if data.Action == "added" {
		_, err := fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", data.TaskID, data.BlockedBy)
		return err
	}
	_, err := fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", data.TaskID, data.BlockedBy)
	return err
}

// FormatStats formats statistics as a multi-section TOON document.
func (f *ToonFormatter) FormatStats(w io.Writer, data StatsData) error {
	fmt.Fprint(w, "stats{total,open,in_progress,done,cancelled,ready,blocked}:\n")
	fmt.Fprintf(w, "  %d,%d,%d,%d,%d,%d,%d\n",
		data.Total, data.Open, data.InProgress, data.Done, data.Cancelled,
		data.Ready, data.Blocked)

	fmt.Fprint(w, "\nby_priority[5]{priority,count}:\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(w, "  %d,%d\n", i, data.ByPriority[i])
	}

	return nil
}

// FormatMessage formats a simple message as plain text.
func (f *ToonFormatter) FormatMessage(w io.Writer, message string) error {
	_, err := fmt.Fprintln(w, message)
	return err
}

// toonEscape wraps a value in quotes if it contains characters that need escaping
// (commas, newlines, or leading/trailing whitespace). Quotes within the value are doubled.
func toonEscape(s string) string {
	needsQuote := strings.ContainsAny(s, ",\n\r") ||
		(len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t'))

	if !needsQuote {
		return s
	}

	escaped := strings.ReplaceAll(s, `"`, `""`)
	return `"` + escaped + `"`
}
