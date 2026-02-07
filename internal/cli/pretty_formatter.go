package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// maxListTitleLen is the maximum title length in list output before truncation.
const maxListTitleLen = 50

// PrettyFormatter implements the Formatter interface for human-readable terminal output.
// It renders aligned columns with no borders, colors, or icons.
type PrettyFormatter struct{}

// FormatTaskList renders a list of tasks as column-aligned table with header.
// Dynamic column widths adapt to data. Empty lists produce "No tasks found." with no headers.
// Long titles are truncated with "..." to maxListTitleLen characters.
func (f *PrettyFormatter) FormatTaskList(w io.Writer, rows []listRow, quiet bool) error {
	if quiet {
		for _, r := range rows {
			fmt.Fprintln(w, r.ID)
		}
		return nil
	}

	if len(rows) == 0 {
		fmt.Fprintln(w, "No tasks found.")
		return nil
	}

	// Compute dynamic column widths
	idWidth := len("ID")
	statusWidth := len("STATUS")
	priWidth := len("PRI")

	for _, r := range rows {
		if len(r.ID) > idWidth {
			idWidth = len(r.ID)
		}
		if len(r.Status) > statusWidth {
			statusWidth = len(r.Status)
		}
		ps := strconv.Itoa(r.Priority)
		if len(ps) > priWidth {
			priWidth = len(ps)
		}
	}

	// Print header
	fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%s\n", idWidth, statusWidth, priWidth)
	fmt.Fprintf(w, fmtStr, "ID", "STATUS", "PRI", "TITLE")

	// Print data rows
	for _, r := range rows {
		title := truncateTitle(r.Title, maxListTitleLen)
		fmt.Fprintf(w, fmtStr, r.ID, r.Status, strconv.Itoa(r.Priority), title)
	}

	return nil
}

// FormatTaskDetail renders full detail for a single task as key-value pairs with aligned labels.
// Sections: base fields, Blocked by (indented), Children (indented), Description (indented block).
// Empty sections are omitted entirely. Titles are never truncated in show output.
func (f *PrettyFormatter) FormatTaskDetail(w io.Writer, detail TaskDetail) error {
	// Determine the longest label for alignment.
	// Base labels: ID, Title, Status, Priority, Created, Updated
	// Conditional: Parent, Closed
	labelWidth := len("Priority") // "Priority" is the longest base label (8 chars)
	if detail.Parent != "" && len("Parent") > labelWidth {
		labelWidth = len("Parent")
	}
	if detail.Closed != "" && len("Closed") > labelWidth {
		labelWidth = len("Closed")
	}

	fmtStr := fmt.Sprintf("%%-%ds  %%s\n", labelWidth+1) // +1 for the colon

	fmt.Fprintf(w, fmtStr, "ID:", detail.ID)
	fmt.Fprintf(w, fmtStr, "Title:", detail.Title)
	fmt.Fprintf(w, fmtStr, "Status:", detail.Status)
	fmt.Fprintf(w, fmtStr, "Priority:", strconv.Itoa(detail.Priority))

	if detail.Parent != "" {
		fmt.Fprintf(w, fmtStr, "Parent:", detail.Parent)
	}

	fmt.Fprintf(w, fmtStr, "Created:", detail.Created)
	fmt.Fprintf(w, fmtStr, "Updated:", detail.Updated)

	if detail.Closed != "" {
		fmt.Fprintf(w, fmtStr, "Closed:", detail.Closed)
	}

	// Blocked by section
	if len(detail.BlockedBy) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Blocked by:")
		for _, b := range detail.BlockedBy {
			fmt.Fprintf(w, "  %s  %s (%s)\n", b.ID, b.Title, b.Status)
		}
	}

	// Children section
	if len(detail.Children) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Children:")
		for _, c := range detail.Children {
			fmt.Fprintf(w, "  %s  %s (%s)\n", c.ID, c.Title, c.Status)
		}
	}

	// Description section
	if detail.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Description:")
		for _, line := range strings.Split(detail.Description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	return nil
}

// FormatTransition renders a status transition as plain text.
func (f *PrettyFormatter) FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error {
	fmt.Fprintf(w, "%s: %s \u2192 %s\n", id, oldStatus, newStatus)
	return nil
}

// FormatDepChange renders a dependency add/remove result as plain text.
func (f *PrettyFormatter) FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error {
	if quiet {
		return nil
	}
	switch action {
	case "added":
		fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", taskID, blockedByID)
	case "removed":
		fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", taskID, blockedByID)
	}
	return nil
}

// FormatStats renders task statistics in three groups with right-aligned numbers.
// Groups: total, status breakdown + workflow counts, priority with P0-P4 labels.
// All rows are always present, even when values are zero.
func (f *PrettyFormatter) FormatStats(w io.Writer, stats StatsData) error {
	// Priority labels with descriptors
	priorityLabels := [5]string{
		"P0 (critical):",
		"P1 (high):",
		"P2 (medium):",
		"P3 (low):",
		"P4 (backlog):",
	}

	// Find the maximum number width for right-alignment across all values
	allValues := []int{
		stats.Total,
		stats.Open, stats.InProgress, stats.Done, stats.Cancelled,
		stats.Ready, stats.Blocked,
	}
	for _, v := range stats.ByPriority {
		allValues = append(allValues, v)
	}

	maxNumWidth := 1
	for _, v := range allValues {
		w := len(strconv.Itoa(v))
		if w > maxNumWidth {
			maxNumWidth = w
		}
	}

	// Find the longest label in each section for alignment
	// Status section labels
	statusLabels := []string{"Open:", "In Progress:", "Done:", "Cancelled:"}
	workflowLabels := []string{"Ready:", "Blocked:"}

	// For the status+workflow sections, find the max label width
	maxStatusLabel := 0
	for _, l := range statusLabels {
		if len(l) > maxStatusLabel {
			maxStatusLabel = len(l)
		}
	}
	for _, l := range workflowLabels {
		if len(l) > maxStatusLabel {
			maxStatusLabel = len(l)
		}
	}

	// For priority labels, find the max
	maxPriLabel := 0
	for _, l := range priorityLabels {
		if len(l) > maxPriLabel {
			maxPriLabel = len(l)
		}
	}

	// Total line - use same alignment as status section
	totalLabelWidth := len("Total:")
	if totalLabelWidth < maxStatusLabel {
		totalLabelWidth = maxStatusLabel
	}

	// Render total
	fmt.Fprintf(w, "%-*s  %*d\n", totalLabelWidth, "Total:", maxNumWidth, stats.Total)

	// Status section
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Status:")
	statusValues := []int{stats.Open, stats.InProgress, stats.Done, stats.Cancelled}
	for i, label := range statusLabels {
		fmt.Fprintf(w, "  %-*s  %*d\n", maxStatusLabel, label, maxNumWidth, statusValues[i])
	}

	// Workflow section
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Workflow:")
	workflowValues := []int{stats.Ready, stats.Blocked}
	for i, label := range workflowLabels {
		fmt.Fprintf(w, "  %-*s  %*d\n", maxStatusLabel, label, maxNumWidth, workflowValues[i])
	}

	// Priority section
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Priority:")
	for i, label := range priorityLabels {
		fmt.Fprintf(w, "  %-*s  %*d\n", maxPriLabel, label, maxNumWidth, stats.ByPriority[i])
	}

	return nil
}

// FormatMessage renders a simple message as plain text.
func (f *PrettyFormatter) FormatMessage(w io.Writer, msg string) error {
	fmt.Fprintln(w, msg)
	return nil
}

// truncateTitle shortens a title to maxLen characters, appending "..." if truncated.
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	if maxLen <= 3 {
		return "..."
	}
	return title[:maxLen-3] + "..."
}
