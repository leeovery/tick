package cli

import (
	"fmt"
	"strconv"
	"strings"

	toon "github.com/toon-format/toon-go"
)

// ToonFormatter implements Formatter for TOON output format.
// TOON (Token-Oriented Object Notation) is optimized for AI agent consumption,
// providing 30-60% token savings over JSON while improving parsing accuracy.
type ToonFormatter struct{}

// toonTaskRow represents a task row for TOON array marshaling.
type toonTaskRow struct {
	ID       string `toon:"id"`
	Title    string `toon:"title"`
	Status   string `toon:"status"`
	Priority int    `toon:"priority"`
}

// FormatTaskList formats a list of tasks in TOON format.
// Output: tasks[N]{id,title,status,priority}:\n  row1\n  row2\n...
func (f *ToonFormatter) FormatTaskList(data *TaskListData) string {
	if data == nil || len(data.Tasks) == 0 {
		return "tasks[0]{id,title,status,priority}:\n"
	}

	// Convert to toon structs
	rows := make([]toonTaskRow, len(data.Tasks))
	for i, task := range data.Tasks {
		rows[i] = toonTaskRow{
			ID:       task.ID,
			Title:    task.Title,
			Status:   task.Status,
			Priority: task.Priority,
		}
	}

	// Use toon-go to marshal with proper escaping
	result, err := toon.MarshalString(rows, toon.WithIndent(2))
	if err != nil {
		// Fallback to manual format if marshaling fails
		return f.formatTaskListManual(data)
	}

	// toon-go outputs [N]{...}: format, we need tasks[N]{...}:
	// Replace the leading bracket with "tasks"
	result = "tasks" + result

	// Ensure trailing newline for consistent CLI output
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// formatTaskListManual is a fallback for manual formatting if toon-go fails.
func (f *ToonFormatter) formatTaskListManual(data *TaskListData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("tasks[%d]{id,title,status,priority}:\n", len(data.Tasks)))

	for _, task := range data.Tasks {
		title := escapeValue(task.Title)
		sb.WriteString(fmt.Sprintf("  %s,%s,%s,%d\n", task.ID, title, task.Status, task.Priority))
	}

	return sb.String()
}

// FormatTaskDetail formats full task details in TOON format with multiple sections.
// Dynamic schema: parent/closed omitted when null.
// blocked_by/children always present (even with 0 count).
// description omitted when empty.
func (f *ToonFormatter) FormatTaskDetail(data *TaskDetailData) string {
	if data == nil {
		return ""
	}

	var sb strings.Builder

	// Build task section with dynamic schema
	schema := f.buildTaskSchema(data)
	sb.WriteString(fmt.Sprintf("task{%s}:\n", schema))
	sb.WriteString(fmt.Sprintf("  %s\n", f.buildTaskRow(data)))

	// blocked_by section (always present)
	sb.WriteString(fmt.Sprintf("\nblocked_by[%d]{id,title,status}:\n", len(data.BlockedBy)))
	for _, blocker := range data.BlockedBy {
		title := escapeValue(blocker.Title)
		sb.WriteString(fmt.Sprintf("  %s,%s,%s\n", blocker.ID, title, blocker.Status))
	}

	// children section (always present)
	sb.WriteString(fmt.Sprintf("\nchildren[%d]{id,title,status}:\n", len(data.Children)))
	for _, child := range data.Children {
		title := escapeValue(child.Title)
		sb.WriteString(fmt.Sprintf("  %s,%s,%s\n", child.ID, title, child.Status))
	}

	// description section (omitted when empty)
	if data.Description != "" {
		sb.WriteString("\ndescription:\n")
		lines := strings.Split(data.Description, "\n")
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	return sb.String()
}

// buildTaskSchema builds the dynamic schema for task detail.
// Omits parent when empty, omits closed when empty.
func (f *ToonFormatter) buildTaskSchema(data *TaskDetailData) string {
	fields := []string{"id", "title", "status", "priority"}

	if data.Parent != "" {
		fields = append(fields, "parent")
	}

	fields = append(fields, "created", "updated")

	if data.Closed != "" {
		fields = append(fields, "closed")
	}

	return strings.Join(fields, ",")
}

// buildTaskRow builds the task data row matching the dynamic schema.
func (f *ToonFormatter) buildTaskRow(data *TaskDetailData) string {
	title := escapeValue(data.Title)
	values := []string{
		data.ID,
		title,
		data.Status,
		strconv.Itoa(data.Priority),
	}

	if data.Parent != "" {
		values = append(values, data.Parent)
	}

	values = append(values, data.Created, data.Updated)

	if data.Closed != "" {
		values = append(values, data.Closed)
	}

	return strings.Join(values, ",")
}

// FormatTransition formats a status transition message as plain text.
// Non-structured outputs use plain text regardless of format.
func (f *ToonFormatter) FormatTransition(taskID, oldStatus, newStatus string) string {
	return fmt.Sprintf("%s: %s \u2192 %s\n", taskID, oldStatus, newStatus)
}

// FormatDepChange formats a dependency change message as plain text.
// Non-structured outputs use plain text regardless of format.
func (f *ToonFormatter) FormatDepChange(action, taskID, blockedByID string) string {
	if action == "add" {
		return fmt.Sprintf("Dependency added: %s blocked by %s\n", taskID, blockedByID)
	}
	return fmt.Sprintf("Dependency removed: %s no longer blocked by %s\n", taskID, blockedByID)
}

// FormatStats formats statistics in TOON format.
// Output: stats section + by_priority section (always 5 rows).
func (f *ToonFormatter) FormatStats(data *StatsData) string {
	if data == nil {
		return ""
	}

	var sb strings.Builder

	// stats section
	sb.WriteString("stats{total,open,in_progress,done,cancelled,ready,blocked}:\n")
	sb.WriteString(fmt.Sprintf("  %d,%d,%d,%d,%d,%d,%d\n",
		data.Total, data.Open, data.InProgress, data.Done, data.Cancelled, data.Ready, data.Blocked))

	// by_priority section (always 5 rows for P0-P4)
	sb.WriteString(fmt.Sprintf("\nby_priority[%d]{priority,count}:\n", len(data.ByPriority)))
	for _, pc := range data.ByPriority {
		sb.WriteString(fmt.Sprintf("  %d,%d\n", pc.Priority, pc.Count))
	}

	return sb.String()
}

// FormatMessage formats a simple message as plain text.
// Non-structured outputs use plain text regardless of format.
func (f *ToonFormatter) FormatMessage(msg string) string {
	return msg + "\n"
}

// escapeValue escapes a value for TOON format using toon-go.
// If the value contains commas or special characters, it will be quoted.
func escapeValue(s string) string {
	// Use toon-go to escape properly
	escaped, err := toon.MarshalString(s)
	if err != nil {
		// Fallback: quote if contains comma
		if strings.Contains(s, ",") {
			return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
		}
		return s
	}

	// MarshalString adds a newline, trim it
	escaped = strings.TrimSuffix(escaped, "\n")
	return escaped
}
