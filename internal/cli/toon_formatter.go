package cli

import (
	"fmt"
	"io"
	"strings"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/task"
)

// StatsData holds the computed statistics for FormatStats.
type StatsData struct {
	Total      int
	Open       int
	InProgress int
	Done       int
	Cancelled  int
	Ready      int
	Blocked    int
	ByPriority [5]int // index = priority (0-4), value = count
}

// ToonFormatter implements the Formatter interface using TOON output format.
// It produces agent-optimized output with schema headers and compact rows.
type ToonFormatter struct{}

// toonListRow is an internal type used for toon-go escaping of list rows.
type toonListRow struct {
	ID       string `toon:"id"`
	Title    string `toon:"title"`
	Status   string `toon:"status"`
	Priority int    `toon:"priority"`
}

// toonListWrapper wraps a named slice for toon-go Marshal.
type toonListWrapper struct {
	Tasks []toonListRow `toon:"tasks"`
}

// FormatTaskList renders a list of tasks in TOON format.
// Output: tasks[N]{id,title,status,priority}: followed by indented data rows.
// Zero tasks produce tasks[0]{id,title,status,priority}: with no rows.
func (f *ToonFormatter) FormatTaskList(w io.Writer, tasks []TaskRow) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprint(w, "tasks[0]{id,title,status,priority}:\n")
		return err
	}

	rows := make([]toonListRow, len(tasks))
	for i, t := range tasks {
		rows[i] = toonListRow{ID: t.ID, Title: t.Title, Status: t.Status, Priority: t.Priority}
	}

	out, err := toon.MarshalString(toonListWrapper{Tasks: rows})
	if err != nil {
		return fmt.Errorf("toon marshal failed: %w", err)
	}

	_, err = fmt.Fprint(w, out+"\n")
	return err
}

// FormatTaskDetail renders full task details in TOON multi-section format.
// Sections: task (dynamic schema), blocked_by, children (always present), description (conditional).
func (f *ToonFormatter) FormatTaskDetail(w io.Writer, data *showData) error {
	// Build task section with dynamic schema
	schema := []string{"id", "title", "status", "priority"}
	values := []string{
		data.ID,
		escapeField(data.Title),
		data.Status,
		fmt.Sprintf("%d", data.Priority),
	}

	if data.Parent != "" {
		schema = append(schema, "parent")
		values = append(values, data.Parent)
	}

	schema = append(schema, "created", "updated")
	values = append(values, data.Created, data.Updated)

	if data.Closed != "" {
		schema = append(schema, "closed")
		values = append(values, data.Closed)
	}

	// task{schema}: row
	fmt.Fprintf(w, "task{%s}:\n", strings.Join(schema, ","))
	fmt.Fprintf(w, "  %s\n", strings.Join(values, ","))

	// blocked_by section (always present)
	fmt.Fprintln(w)
	writeRelatedSection(w, "blocked_by", data.BlockedBy)

	// children section (always present)
	fmt.Fprintln(w)
	writeRelatedSection(w, "children", data.Children)

	// description section (conditional)
	if data.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "description:")
		for _, line := range strings.Split(data.Description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	return nil
}

// FormatTransition renders a status transition as plain text.
func (f *ToonFormatter) FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error {
	_, err := fmt.Fprintf(w, "%s: %s \u2192 %s\n", id, oldStatus, newStatus)
	return err
}

// FormatDepChange renders a dependency add/remove confirmation as plain text.
func (f *ToonFormatter) FormatDepChange(w io.Writer, action, taskID, blockedByID string) error {
	switch action {
	case "added":
		_, err := fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", taskID, blockedByID)
		return err
	case "removed":
		_, err := fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", taskID, blockedByID)
		return err
	default:
		_, err := fmt.Fprintf(w, "Dependency %s: %s %s %s\n", action, taskID, action, blockedByID)
		return err
	}
}

// FormatStats renders task statistics in TOON format.
// Sections: stats summary + by_priority (always 5 rows, priorities 0-4).
func (f *ToonFormatter) FormatStats(w io.Writer, stats interface{}) error {
	sd, ok := stats.(*StatsData)
	if !ok {
		return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
	}

	// stats summary section
	fmt.Fprintln(w, "stats{total,open,in_progress,done,cancelled,ready,blocked}:")
	fmt.Fprintf(w, "  %d,%d,%d,%d,%d,%d,%d\n", sd.Total, sd.Open, sd.InProgress, sd.Done, sd.Cancelled, sd.Ready, sd.Blocked)

	// by_priority section
	fmt.Fprintln(w)
	fmt.Fprintln(w, "by_priority[5]{priority,count}:")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(w, "  %d,%d\n", i, sd.ByPriority[i])
	}

	return nil
}

// FormatMessage renders a simple message as plain text.
func (f *ToonFormatter) FormatMessage(w io.Writer, message string) error {
	_, err := fmt.Fprintln(w, message)
	return err
}

// writeRelatedSection writes a named section for related tasks (blocked_by or children).
// Always includes the schema header, even with zero items.
func writeRelatedSection(w io.Writer, name string, items []relatedTask) {
	fmt.Fprintf(w, "%s[%d]{id,title,status}:\n", name, len(items))
	for _, item := range items {
		fmt.Fprintf(w, "  %s,%s,%s\n", item.ID, escapeField(item.Title), item.Status)
	}
}

// escapeField uses toon-go to escape a string value that may contain commas or special characters.
// Values requiring escaping are wrapped in double quotes per TOON spec section 7.1.
func escapeField(s string) string {
	// If the value contains no special characters, return as-is
	if !strings.ContainsAny(s, ",\"\n\\") {
		return s
	}

	// Use toon-go to get properly escaped value
	type wrapper struct {
		V string `toon:"v"`
	}
	out, err := toon.MarshalString(wrapper{V: s})
	if err != nil {
		// Fallback: quote manually
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}

	// Output is "v: <value>" - extract value after "v: "
	return strings.TrimPrefix(out, "v: ")
}
