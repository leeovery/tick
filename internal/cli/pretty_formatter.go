package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PrettyFormatter implements the Formatter interface for human-readable
// terminal output. It produces aligned columns with no borders, colors,
// or icons — minimalist and clean.
type PrettyFormatter struct{}

// FormatTaskList renders a list of tasks as an aligned column table with
// a header row. Dynamic column widths adapt to data. Empty lists produce
// "No tasks found." with no headers. Data must be []TaskRow.
func (f *PrettyFormatter) FormatTaskList(w io.Writer, data interface{}) error {
	rows, ok := data.([]TaskRow)
	if !ok {
		return fmt.Errorf("FormatTaskList: expected []TaskRow, got %T", data)
	}

	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "No tasks found.")
		return err
	}

	// Calculate dynamic column widths.
	idW := len("ID")
	statusW := len("STATUS")
	priW := len("PRI")

	for _, r := range rows {
		if len(r.ID) > idW {
			idW = len(r.ID)
		}
		if len(r.Status) > statusW {
			statusW = len(r.Status)
		}
		ps := strconv.Itoa(r.Priority)
		if len(ps) > priW {
			priW = len(ps)
		}
	}

	// Build format string: left-aligned ID/STATUS, right-aligned PRI, then TITLE.
	fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%%ds  %%s\n", idW, statusW, priW)
	priFmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%%dd  %%s\n", idW, statusW, priW)

	// Header.
	if _, err := fmt.Fprintf(w, fmtStr, "ID", "STATUS", "PRI", "TITLE"); err != nil {
		return err
	}

	// Data rows.
	for _, r := range rows {
		title := truncateTitle(r.Title, maxTitleWidth)
		if _, err := fmt.Fprintf(w, priFmtStr, r.ID, r.Status, r.Priority, title); err != nil {
			return err
		}
	}

	return nil
}

// FormatTaskDetail renders full details of a single task as key-value pairs
// with aligned labels. Empty sections (blocked by, children, description)
// are omitted entirely. Data must be *showData.
func (f *PrettyFormatter) FormatTaskDetail(w io.Writer, data interface{}) error {
	d, ok := data.(*showData)
	if !ok {
		return fmt.Errorf("FormatTaskDetail: expected *showData, got %T", data)
	}

	// Base fields with aligned labels (10-char label column).
	fmt.Fprintf(w, "%-10s%s\n", "ID:", d.id)
	fmt.Fprintf(w, "%-10s%s\n", "Title:", d.title)
	fmt.Fprintf(w, "%-10s%s\n", "Status:", d.status)
	fmt.Fprintf(w, "%-10s%d\n", "Priority:", d.priority)

	if d.parent != "" {
		if d.parentTitle != "" {
			fmt.Fprintf(w, "%-10s%s  %s\n", "Parent:", d.parent, d.parentTitle)
		} else {
			fmt.Fprintf(w, "%-10s%s\n", "Parent:", d.parent)
		}
	}

	fmt.Fprintf(w, "%-10s%s\n", "Created:", d.created)
	fmt.Fprintf(w, "%-10s%s\n", "Updated:", d.updated)

	if d.closed != "" {
		fmt.Fprintf(w, "%-10s%s\n", "Closed:", d.closed)
	}

	// Blocked by section (omit when empty).
	if len(d.blockedBy) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Blocked by:")
		for _, rt := range d.blockedBy {
			fmt.Fprintf(w, "  %s  %s (%s)\n", rt.id, rt.title, rt.status)
		}
	}

	// Children section (omit when empty).
	if len(d.children) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Children:")
		for _, rt := range d.children {
			fmt.Fprintf(w, "  %s  %s (%s)\n", rt.id, rt.title, rt.status)
		}
	}

	// Description section (omit when empty).
	if strings.TrimSpace(d.description) != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Description:")
		for _, line := range strings.Split(d.description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	return nil
}

// FormatTransition renders a status transition result as plain text.
// Data must be *TransitionData.
func (f *PrettyFormatter) FormatTransition(w io.Writer, data interface{}) error {
	d, ok := data.(*TransitionData)
	if !ok {
		return fmt.Errorf("FormatTransition: expected *TransitionData, got %T", data)
	}
	_, err := fmt.Fprintf(w, "%s: %s \u2192 %s\n", d.ID, d.OldStatus, d.NewStatus)
	return err
}

// FormatDepChange renders a dependency change confirmation as plain text.
// Data must be *DepChangeData.
func (f *PrettyFormatter) FormatDepChange(w io.Writer, data interface{}) error {
	d, ok := data.(*DepChangeData)
	if !ok {
		return fmt.Errorf("FormatDepChange: expected *DepChangeData, got %T", data)
	}
	switch d.Action {
	case "added":
		_, err := fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", d.TaskID, d.BlockedByID)
		return err
	case "removed":
		_, err := fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", d.TaskID, d.BlockedByID)
		return err
	default:
		return fmt.Errorf("FormatDepChange: unknown action %q", d.Action)
	}
}

// FormatStats renders task statistics in three groups: total, status/workflow
// breakdown, and priority counts with P0-P4 labels. Numbers are right-aligned.
// Data must be *StatsData.
func (f *PrettyFormatter) FormatStats(w io.Writer, data interface{}) error {
	d, ok := data.(*StatsData)
	if !ok {
		return fmt.Errorf("FormatStats: expected *StatsData, got %T", data)
	}

	// Compute number width for summary group (Total + Status + Workflow).
	summaryNums := []int{d.Total, d.Open, d.InProgress, d.Done, d.Cancelled, d.Ready, d.Blocked}
	summaryNumW := numWidth(summaryNums)

	// Summary label width: max of "Total:", "In Progress:", "Cancelled:", "Ready:", "Blocked:", etc.
	// "In Progress:" = 12 is always the widest.
	const summaryLabelW = 12

	// Total line (no indent).
	summaryFmt := fmt.Sprintf("%%-%ds%%%dd\n", summaryLabelW, summaryNumW)
	indentFmt := fmt.Sprintf("  %%-%ds%%%dd\n", summaryLabelW, summaryNumW)

	fmt.Fprintf(w, summaryFmt, "Total:", d.Total)

	// Status group.
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Status:")
	fmt.Fprintf(w, indentFmt, "Open:", d.Open)
	fmt.Fprintf(w, indentFmt, "In Progress:", d.InProgress)
	fmt.Fprintf(w, indentFmt, "Done:", d.Done)
	fmt.Fprintf(w, indentFmt, "Cancelled:", d.Cancelled)

	// Workflow group.
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Workflow:")
	fmt.Fprintf(w, indentFmt, "Ready:", d.Ready)
	fmt.Fprintf(w, indentFmt, "Blocked:", d.Blocked)

	// Priority group (independent alignment).
	priLabels := [5]string{
		"P0 (critical):",
		"P1 (high):",
		"P2 (medium):",
		"P3 (low):",
		"P4 (backlog):",
	}
	priLabelW := 0
	for _, l := range priLabels {
		if len(l) > priLabelW {
			priLabelW = len(l)
		}
	}
	priNumW := numWidth(d.ByPriority[:])
	priFmt := fmt.Sprintf("  %%-%ds%%%dd\n", priLabelW, priNumW)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Priority:")
	for i, label := range priLabels {
		fmt.Fprintf(w, priFmt, label, d.ByPriority[i])
	}

	return nil
}

// FormatMessage writes the message followed by a newline.
func (f *PrettyFormatter) FormatMessage(w io.Writer, msg string) {
	fmt.Fprintln(w, msg)
}

// maxTitleWidth is the maximum title length in list output before truncation.
const maxTitleWidth = 50

// numWidth returns the width needed to display the widest number in the slice.
// Returns at least 3 to ensure consistent spacing with right-aligned numbers.
func numWidth(nums []int) int {
	w := 3
	for _, n := range nums {
		s := strconv.Itoa(n)
		if len(s) > w {
			w = len(s)
		}
	}
	return w
}

// truncateTitle truncates a title to maxTitleWidth characters, appending "..."
// if it exceeds the limit.
func truncateTitle(title string, maxWidth int) string {
	if len(title) <= maxWidth {
		return title
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	return title[:maxWidth-3] + "..."
}
