package cli

import (
	"fmt"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

const maxListTitleLen = 50

// PrettyFormatter renders CLI output in human-readable aligned-column format
// for terminal display. No borders, no colors, no icons.
type PrettyFormatter struct {
	baseFormatter
}

// Compile-time interface verification.
var _ Formatter = (*PrettyFormatter)(nil)

// FormatTaskList renders a list of tasks as an aligned-column table with header.
// Empty input returns "No tasks found." with no headers.
// Long titles are truncated to maxListTitleLen characters with "..." appended.
func (f *PrettyFormatter) FormatTaskList(tasks []task.Task) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}

	// Compute dynamic column widths based on data.
	idWidth := len("ID")
	statusWidth := len("STATUS")
	priWidth := len("PRI")
	typeWidth := len("TYPE")

	for _, t := range tasks {
		if len(t.ID) > idWidth {
			idWidth = len(t.ID)
		}
		s := string(t.Status)
		if len(s) > statusWidth {
			statusWidth = len(s)
		}
		p := fmt.Sprintf("%d", t.Priority)
		if len(p) > priWidth {
			priWidth = len(p)
		}
		tv := typeOrDash(t.Type)
		if len(tv) > typeWidth {
			typeWidth = len(tv)
		}
	}

	// Add gutter spacing (3 spaces between columns).
	idCol := idWidth + 3
	statusCol := statusWidth + 2
	priCol := priWidth + 2
	typeCol := typeWidth + 2

	var b strings.Builder
	// Header
	fmt.Fprintf(&b, "%-*s%-*s%-*s%-*s%s", idCol, "ID", statusCol, "STATUS", priCol, "PRI", typeCol, "TYPE", "TITLE")

	// Rows
	for _, t := range tasks {
		title := truncateTitle(t.Title)
		tv := typeOrDash(t.Type)
		b.WriteString("\n")
		fmt.Fprintf(&b, "%-*s%-*s%-*d%-*s%s",
			idCol, t.ID,
			statusCol, string(t.Status),
			priCol, t.Priority,
			typeCol, tv,
			title,
		)
	}

	return b.String()
}

// FormatTaskDetail renders a single task with full details in key-value format.
// Sections (Blocked by, Children, Description) are omitted when empty.
func (f *PrettyFormatter) FormatTaskDetail(detail TaskDetail) string {
	t := detail.Task
	var b strings.Builder

	fmt.Fprintf(&b, "ID:       %s\n", t.ID)
	fmt.Fprintf(&b, "Title:    %s\n", t.Title)
	fmt.Fprintf(&b, "Status:   %s\n", string(t.Status))
	fmt.Fprintf(&b, "Priority: %d\n", t.Priority)

	fmt.Fprintf(&b, "Type:     %s\n", typeOrDash(t.Type))

	if len(detail.Tags) > 0 {
		fmt.Fprintf(&b, "Tags:     %s\n", strings.Join(detail.Tags, ", "))
	}

	if t.Parent != "" {
		if detail.ParentTitle != "" {
			fmt.Fprintf(&b, "Parent:   %s (%s)\n", t.Parent, detail.ParentTitle)
		} else {
			fmt.Fprintf(&b, "Parent:   %s\n", t.Parent)
		}
	}

	fmt.Fprintf(&b, "Created:  %s\n", task.FormatTimestamp(t.Created))
	fmt.Fprintf(&b, "Updated:  %s", task.FormatTimestamp(t.Updated))

	if t.Closed != nil {
		fmt.Fprintf(&b, "\nClosed:   %s", task.FormatTimestamp(*t.Closed))
	}

	if len(detail.BlockedBy) > 0 {
		b.WriteString("\n\nBlocked by:")
		for _, r := range detail.BlockedBy {
			fmt.Fprintf(&b, "\n  %s  %s (%s)", r.ID, r.Title, r.Status)
		}
	}

	if len(detail.Children) > 0 {
		b.WriteString("\n\nChildren:")
		for _, r := range detail.Children {
			fmt.Fprintf(&b, "\n  %s  %s (%s)", r.ID, r.Title, r.Status)
		}
	}

	if t.Description != "" {
		b.WriteString("\n\nDescription:")
		lines := strings.Split(t.Description, "\n")
		for _, line := range lines {
			fmt.Fprintf(&b, "\n  %s", line)
		}
	}

	return b.String()
}

// FormatStats renders task statistics in grouped sections with right-aligned numbers.
// Numbers right-align to a consistent column within each group.
// Top-level lines align to column 15; indented lines align to column 17
// (accounting for the 2-space indent).
func (f *PrettyFormatter) FormatStats(stats Stats) string {
	var b strings.Builder

	// Total line: "Total: " (7 chars) + %8d = 15 total width.
	fmt.Fprintf(&b, "Total: %8d", stats.Total)

	// Status group: all lines total width 17 from line start.
	b.WriteString("\n\nStatus:")
	fmt.Fprintf(&b, "\n  Open:%10d", stats.Open)
	fmt.Fprintf(&b, "\n  In Progress: %2d", stats.InProgress)
	fmt.Fprintf(&b, "\n  Done:%10d", stats.Done)
	fmt.Fprintf(&b, "\n  Cancelled:%5d", stats.Cancelled)

	// Workflow group: lines total width 17.
	b.WriteString("\n\nWorkflow:")
	fmt.Fprintf(&b, "\n  Ready:%9d", stats.Ready)
	fmt.Fprintf(&b, "\n  Blocked:%7d", stats.Blocked)

	// Priority group: lines total width 19.
	b.WriteString("\n\nPriority:")
	fmt.Fprintf(&b, "\n  P0 (critical): %2d", stats.ByPriority[0])
	fmt.Fprintf(&b, "\n  P1 (high):     %2d", stats.ByPriority[1])
	fmt.Fprintf(&b, "\n  P2 (medium):   %2d", stats.ByPriority[2])
	fmt.Fprintf(&b, "\n  P3 (low):      %2d", stats.ByPriority[3])
	fmt.Fprintf(&b, "\n  P4 (backlog):  %2d", stats.ByPriority[4])

	return b.String()
}

// FormatMessage renders a general-purpose message as plain text.
func (f *PrettyFormatter) FormatMessage(msg string) string {
	return msg
}

// typeOrDash returns the type string or "-" if empty, for Pretty formatter display.
func typeOrDash(typ string) string {
	if typ == "" {
		return "-"
	}
	return typ
}

// truncateTitle truncates a title to maxListTitleLen characters, appending "..." if truncated.
func truncateTitle(title string) string {
	if len(title) <= maxListTitleLen {
		return title
	}
	return title[:maxListTitleLen-3] + "..."
}
