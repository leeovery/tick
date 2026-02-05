package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// PrettyFormatter implements Formatter for human-readable terminal output.
// Uses aligned columns, no borders, no colors, no icons.
type PrettyFormatter struct{}

// maxTitleLength is the maximum length for titles in list output before truncation.
const maxTitleLength = 50

// priorityLabels maps priority levels to their human-readable labels.
var priorityLabels = map[int]string{
	0: "P0 (critical)",
	1: "P1 (high)",
	2: "P2 (medium)",
	3: "P3 (low)",
	4: "P4 (backlog)",
}

// FormatTaskList formats a list of tasks as aligned columns with header.
// Returns "No tasks found." for empty list (no headers).
func (f *PrettyFormatter) FormatTaskList(data *TaskListData) string {
	if data == nil || len(data.Tasks) == 0 {
		return "No tasks found.\n"
	}

	// Calculate column widths
	idWidth := len("ID")
	statusWidth := len("STATUS")
	priWidth := len("PRI")

	for _, task := range data.Tasks {
		if len(task.ID) > idWidth {
			idWidth = len(task.ID)
		}
		if len(task.Status) > statusWidth {
			statusWidth = len(task.Status)
		}
		priStr := strconv.Itoa(task.Priority)
		if len(priStr) > priWidth {
			priWidth = len(priStr)
		}
	}

	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("%-*s  %-*s  %-*s  %s\n",
		idWidth, "ID",
		statusWidth, "STATUS",
		priWidth, "PRI",
		"TITLE"))

	// Data rows
	for _, task := range data.Tasks {
		title := truncateTitle(task.Title, maxTitleLength)
		sb.WriteString(fmt.Sprintf("%-*s  %-*s  %-*d  %s\n",
			idWidth, task.ID,
			statusWidth, task.Status,
			priWidth, task.Priority,
			title))
	}

	return sb.String()
}

// truncateTitle truncates a title to maxLen, adding "..." if truncated.
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

// FormatTaskDetail formats full task details as key-value pairs with aligned labels.
// Omits empty sections (Blocked by, Children, Description).
func (f *PrettyFormatter) FormatTaskDetail(data *TaskDetailData) string {
	if data == nil {
		return ""
	}

	var sb strings.Builder

	// Key-value pairs with aligned labels
	// Label width is 10 to align values (longest label is "Priority:" at 9)
	labelWidth := 10

	sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "ID:", data.ID))
	sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "Title:", data.Title))
	sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "Status:", data.Status))
	sb.WriteString(fmt.Sprintf("%-*s%d\n", labelWidth, "Priority:", data.Priority))
	sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "Created:", data.Created))

	if data.Updated != "" && data.Updated != data.Created {
		sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "Updated:", data.Updated))
	}

	if data.Closed != "" {
		sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "Closed:", data.Closed))
	}

	if data.Parent != "" {
		sb.WriteString(fmt.Sprintf("%-*s%s", labelWidth, "Parent:", data.Parent))
		if data.ParentTitle != "" {
			sb.WriteString(fmt.Sprintf("  %s", data.ParentTitle))
		}
		sb.WriteString("\n")
	}

	// Blocked by section (omit if empty)
	if len(data.BlockedBy) > 0 {
		sb.WriteString("\nBlocked by:\n")
		for _, blocker := range data.BlockedBy {
			sb.WriteString(fmt.Sprintf("  %s  %s (%s)\n", blocker.ID, blocker.Title, blocker.Status))
		}
	}

	// Children section (omit if empty)
	if len(data.Children) > 0 {
		sb.WriteString("\nChildren:\n")
		for _, child := range data.Children {
			sb.WriteString(fmt.Sprintf("  %s  %s (%s)\n", child.ID, child.Title, child.Status))
		}
	}

	// Description section (omit if empty)
	if data.Description != "" {
		sb.WriteString("\nDescription:\n")
		lines := strings.Split(data.Description, "\n")
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	return sb.String()
}

// FormatTransition formats a status transition message as plain text.
// Non-structured outputs use plain text regardless of format.
func (f *PrettyFormatter) FormatTransition(taskID, oldStatus, newStatus string) string {
	return fmt.Sprintf("%s: %s \u2192 %s\n", taskID, oldStatus, newStatus)
}

// FormatDepChange formats a dependency change message as plain text.
// Non-structured outputs use plain text regardless of format.
func (f *PrettyFormatter) FormatDepChange(action, taskID, blockedByID string) string {
	if action == "add" {
		return fmt.Sprintf("Dependency added: %s blocked by %s\n", taskID, blockedByID)
	}
	return fmt.Sprintf("Dependency removed: %s no longer blocked by %s\n", taskID, blockedByID)
}

// FormatStats formats statistics with three groups: total, status breakdown, workflow, priority.
// Numbers are right-aligned. All rows present including zeros.
func (f *PrettyFormatter) FormatStats(data *StatsData) string {
	if data == nil {
		return ""
	}

	var sb strings.Builder

	// Total - right-align number to position after "Total:"
	// Spec: "Total:       47" (number right-aligned in ~8 char field)
	sb.WriteString(fmt.Sprintf("Total:%8d\n", data.Total))

	// Status section - all values right-aligned to same column
	sb.WriteString("\nStatus:\n")
	sb.WriteString(fmt.Sprintf("  Open:%9d\n", data.Open))
	sb.WriteString(fmt.Sprintf("  In Progress:%2d\n", data.InProgress))
	sb.WriteString(fmt.Sprintf("  Done:%9d\n", data.Done))
	sb.WriteString(fmt.Sprintf("  Cancelled:%4d\n", data.Cancelled))

	// Workflow section
	sb.WriteString("\nWorkflow:\n")
	sb.WriteString(fmt.Sprintf("  Ready:%8d\n", data.Ready))
	sb.WriteString(fmt.Sprintf("  Blocked:%6d\n", data.Blocked))

	// Priority section - all values right-aligned to same column
	sb.WriteString("\nPriority:\n")
	for i := 0; i <= 4; i++ {
		label := priorityLabels[i]
		count := 0
		for _, pc := range data.ByPriority {
			if pc.Priority == i {
				count = pc.Count
				break
			}
		}
		// Right-align numbers: "  P0 (critical):  2" format
		// Priority labels vary in length, right-align numbers to column ~18
		sb.WriteString(fmt.Sprintf("  %s:%*d\n", label, 17-len(label), count))
	}

	return sb.String()
}

// FormatMessage formats a simple message as plain text.
// Non-structured outputs use plain text regardless of format.
func (f *PrettyFormatter) FormatMessage(msg string) string {
	return msg + "\n"
}
