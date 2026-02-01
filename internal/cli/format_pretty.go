package cli

import (
	"fmt"
	"io"
	"strings"
)

const maxTitleWidth = 60

// PrettyFormatter formats output as human-readable aligned text for terminals.
// No borders, no colors, no icons — plain fmt.Print output.
type PrettyFormatter struct{}

// FormatTaskList formats a task list with aligned columns.
// Empty list produces "No tasks found." with no headers.
func (f *PrettyFormatter) FormatTaskList(w io.Writer, tasks []TaskListItem) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(w, "No tasks found.")
		return err
	}

	// Compute dynamic column widths.
	idW := len("ID")
	statusW := len("STATUS")
	for _, t := range tasks {
		if len(t.ID) > idW {
			idW = len(t.ID)
		}
		if len(t.Status) > statusW {
			statusW = len(t.Status)
		}
	}

	fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%-4s %%s\n", idW, statusW)

	fmt.Fprintf(w, fmtStr, "ID", "STATUS", "PRI", "TITLE")
	for _, t := range tasks {
		title := truncateTitle(t.Title, maxTitleWidth)
		fmt.Fprintf(w, fmtStr, t.ID, t.Status, fmt.Sprintf("%d", t.Priority), title)
	}
	return nil
}

// FormatTaskDetail formats full task detail with aligned key-value labels.
// Empty sections are omitted.
func (f *PrettyFormatter) FormatTaskDetail(w io.Writer, d TaskDetail) error {
	fmt.Fprintf(w, "ID:       %s\n", d.ID)
	fmt.Fprintf(w, "Title:    %s\n", d.Title)
	fmt.Fprintf(w, "Status:   %s\n", d.Status)
	fmt.Fprintf(w, "Priority: %d\n", d.Priority)
	if d.Parent != nil {
		fmt.Fprintf(w, "Parent:   %s  %s (%s)\n", d.Parent.ID, d.Parent.Title, d.Parent.Status)
	}
	fmt.Fprintf(w, "Created:  %s\n", d.Created)
	fmt.Fprintf(w, "Updated:  %s\n", d.Updated)
	if d.Closed != "" {
		fmt.Fprintf(w, "Closed:   %s\n", d.Closed)
	}

	if len(d.BlockedBy) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Blocked by:")
		for _, b := range d.BlockedBy {
			fmt.Fprintf(w, "  %s  %s (%s)\n", b.ID, b.Title, b.Status)
		}
	}

	if len(d.Children) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Children:")
		for _, c := range d.Children {
			fmt.Fprintf(w, "  %s  %s (%s)\n", c.ID, c.Title, c.Status)
		}
	}

	if d.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Description:")
		for _, line := range strings.Split(d.Description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	return nil
}

// FormatTransition formats a status transition as plain text.
func (f *PrettyFormatter) FormatTransition(w io.Writer, data TransitionData) error {
	_, err := fmt.Fprintf(w, "%s: %s → %s\n", data.ID, data.OldStatus, data.NewStatus)
	return err
}

// FormatDepChange formats a dependency change as plain text.
func (f *PrettyFormatter) FormatDepChange(w io.Writer, data DepChangeData) error {
	if data.Action == "added" {
		_, err := fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", data.TaskID, data.BlockedBy)
		return err
	}
	_, err := fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", data.TaskID, data.BlockedBy)
	return err
}

// FormatStats formats statistics in three groups with right-aligned numbers.
func (f *PrettyFormatter) FormatStats(w io.Writer, data StatsData) error {
	fmt.Fprintf(w, "Total:       %d\n", data.Total)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Status:")
	fmt.Fprintf(w, "  Open:        %d\n", data.Open)
	fmt.Fprintf(w, "  In Progress: %d\n", data.InProgress)
	fmt.Fprintf(w, "  Done:        %d\n", data.Done)
	fmt.Fprintf(w, "  Cancelled:   %d\n", data.Cancelled)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Workflow:")
	fmt.Fprintf(w, "  Ready:       %d\n", data.Ready)
	fmt.Fprintf(w, "  Blocked:     %d\n", data.Blocked)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Priority:")
	labels := [5]string{"P0 (critical)", "P1 (high)", "P2 (medium)", "P3 (low)", "P4 (backlog)"}
	for i, label := range labels {
		fmt.Fprintf(w, "  %-14s %d\n", label+":", data.ByPriority[i])
	}

	return nil
}

// FormatMessage formats a simple message.
func (f *PrettyFormatter) FormatMessage(w io.Writer, message string) error {
	_, err := fmt.Fprintln(w, message)
	return err
}

// truncateTitle shortens a title to maxLen characters, appending "..." if truncated.
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}
