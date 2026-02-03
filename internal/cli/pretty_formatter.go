package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

// maxListTitleLen is the maximum title length before truncation in list output.
const maxListTitleLen = 60

// PrettyFormatter implements the Formatter interface with human-readable,
// column-aligned output. No borders, no colors, no icons.
type PrettyFormatter struct{}

// Compile-time interface verification.
var _ Formatter = &PrettyFormatter{}

// FormatTaskList renders a list of tasks as an aligned column table with header.
// Empty lists produce "No tasks found." with no headers.
// Long titles are truncated with "..." in list output.
func (f *PrettyFormatter) FormatTaskList(w io.Writer, tasks []TaskRow) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(w, "No tasks found.")
		return err
	}

	// Calculate dynamic column widths
	idWidth := len("ID")
	statusWidth := len("STATUS")
	for _, t := range tasks {
		if len(t.ID) > idWidth {
			idWidth = len(t.ID)
		}
		if len(t.Status) > statusWidth {
			statusWidth = len(t.Status)
		}
	}

	// Add gutter spacing (3 spaces between columns for ID/STATUS, 2 for PRI)
	idCol := idWidth + 3
	statusCol := statusWidth + 2
	priCol := 5 // "PRI" + 2 spaces

	// Print header
	headerFmt := fmt.Sprintf("%%-%ds%%-%ds%%-%ds%%s\n", idCol, statusCol, priCol)
	_, err := fmt.Fprintf(w, headerFmt, "ID", "STATUS", "PRI", "TITLE")
	if err != nil {
		return err
	}

	// Print rows
	rowFmt := fmt.Sprintf("%%-%ds%%-%ds%%-%dd%%s\n", idCol, statusCol, priCol)
	for _, t := range tasks {
		title := truncateTitle(t.Title, maxListTitleLen)
		_, err := fmt.Fprintf(w, rowFmt, t.ID, t.Status, t.Priority, title)
		if err != nil {
			return err
		}
	}

	return nil
}

// FormatTaskDetail renders full details for a single task with aligned key-value labels.
// Empty sections (BlockedBy, Children, Description) are omitted entirely.
func (f *PrettyFormatter) FormatTaskDetail(w io.Writer, data *showData) error {
	// Base fields with aligned labels (10-char label width including colon)
	fmt.Fprintf(w, "ID:       %s\n", data.ID)
	fmt.Fprintf(w, "Title:    %s\n", data.Title)
	fmt.Fprintf(w, "Status:   %s\n", data.Status)
	fmt.Fprintf(w, "Priority: %d\n", data.Priority)

	if data.Parent != "" {
		if data.ParentTitle != "" {
			fmt.Fprintf(w, "Parent:   %s  %s\n", data.Parent, data.ParentTitle)
		} else {
			fmt.Fprintf(w, "Parent:   %s\n", data.Parent)
		}
	}

	fmt.Fprintf(w, "Created:  %s\n", data.Created)
	fmt.Fprintf(w, "Updated:  %s\n", data.Updated)

	if data.Closed != "" {
		fmt.Fprintf(w, "Closed:   %s\n", data.Closed)
	}

	// Blocked by section (omit if empty)
	if len(data.BlockedBy) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Blocked by:")
		for _, r := range data.BlockedBy {
			fmt.Fprintf(w, "  %s  %s (%s)\n", r.ID, r.Title, r.Status)
		}
	}

	// Children section (omit if empty)
	if len(data.Children) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Children:")
		for _, r := range data.Children {
			fmt.Fprintf(w, "  %s  %s (%s)\n", r.ID, r.Title, r.Status)
		}
	}

	// Description section (omit if empty)
	if data.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Description:")
		for _, line := range strings.Split(data.Description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	return nil
}

// FormatTransition renders a status transition as plain text.
func (f *PrettyFormatter) FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error {
	_, err := fmt.Fprintf(w, "%s: %s \u2192 %s\n", id, oldStatus, newStatus)
	return err
}

// FormatDepChange renders a dependency add/remove confirmation as plain text.
func (f *PrettyFormatter) FormatDepChange(w io.Writer, action, taskID, blockedByID string) error {
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

// FormatStats renders task statistics in three groups with right-aligned numbers:
// Total, Status breakdown, Workflow counts, Priority with P0-P4 labels.
func (f *PrettyFormatter) FormatStats(w io.Writer, stats interface{}) error {
	sd, ok := stats.(*StatsData)
	if !ok {
		return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
	}

	// Total
	fmt.Fprintf(w, "Total:       %2d\n", sd.Total)

	// Status breakdown
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Status:")
	fmt.Fprintf(w, "  Open:        %2d\n", sd.Open)
	fmt.Fprintf(w, "  In Progress: %2d\n", sd.InProgress)
	fmt.Fprintf(w, "  Done:        %2d\n", sd.Done)
	fmt.Fprintf(w, "  Cancelled:   %2d\n", sd.Cancelled)

	// Workflow
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Workflow:")
	fmt.Fprintf(w, "  Ready:       %2d\n", sd.Ready)
	fmt.Fprintf(w, "  Blocked:     %2d\n", sd.Blocked)

	// Priority with P0-P4 labels
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Priority:")
	fmt.Fprintf(w, "  P0 (critical): %2d\n", sd.ByPriority[0])
	fmt.Fprintf(w, "  P1 (high):     %2d\n", sd.ByPriority[1])
	fmt.Fprintf(w, "  P2 (medium):   %2d\n", sd.ByPriority[2])
	fmt.Fprintf(w, "  P3 (low):      %2d\n", sd.ByPriority[3])
	fmt.Fprintf(w, "  P4 (backlog):  %2d\n", sd.ByPriority[4])

	return nil
}

// FormatMessage renders a simple message as plain text.
func (f *PrettyFormatter) FormatMessage(w io.Writer, message string) error {
	_, err := fmt.Fprintln(w, message)
	return err
}

// truncateTitle truncates a title to maxLen characters, appending "..." if truncated.
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}
