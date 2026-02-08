package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	toon "github.com/toon-format/toon-go"
)

// ToonFormatter implements the Formatter interface using TOON format.
// TOON (Token-Oriented Object Notation) is optimized for AI agent consumption,
// providing 30-60% token savings over JSON.
type ToonFormatter struct{}

// FormatTaskList renders a list of tasks in TOON tabular format.
// Output: tasks[N]{id,title,status,priority}: followed by indented data rows.
// Empty lists produce tasks[0]{id,title,status,priority}: with no rows.
func (f *ToonFormatter) FormatTaskList(w io.Writer, rows []listRow, quiet bool) error {
	if quiet {
		for _, r := range rows {
			fmt.Fprintln(w, r.ID)
		}
		return nil
	}

	if len(rows) == 0 {
		fmt.Fprintln(w, "tasks[0]{id,title,status,priority}:")
		return nil
	}

	// Build toon objects for tabular encoding
	objects := make([]toon.Object, len(rows))
	for i, r := range rows {
		objects[i] = toon.NewObject(
			toon.Field{Key: "id", Value: r.ID},
			toon.Field{Key: "title", Value: r.Title},
			toon.Field{Key: "status", Value: r.Status},
			toon.Field{Key: "priority", Value: r.Priority},
		)
	}

	doc := toon.NewObject(
		toon.Field{Key: "tasks", Value: objects},
	)
	result, err := toon.MarshalString(doc)
	if err != nil {
		return fmt.Errorf("toon marshal error: %w", err)
	}
	fmt.Fprintln(w, result)
	return nil
}

// FormatTaskDetail renders full detail for a single task in TOON multi-section format.
// Sections: task (dynamic schema), blocked_by (always present), children (always present),
// description (omitted when empty).
func (f *ToonFormatter) FormatTaskDetail(w io.Writer, detail TaskDetail) error {
	var sections []string

	// Build task section with dynamic schema
	sections = append(sections, f.buildTaskSection(detail))

	// blocked_by section (always present, even empty)
	sections = append(sections, f.buildRelatedSection("blocked_by", detail.BlockedBy))

	// children section (always present, even empty)
	sections = append(sections, f.buildRelatedSection("children", detail.Children))

	// description section (omitted when empty)
	if detail.Description != "" {
		sections = append(sections, f.buildDescriptionSection(detail.Description))
	}

	fmt.Fprint(w, strings.Join(sections, "\n"))
	return nil
}

// FormatTransition renders a status transition as plain text.
func (f *ToonFormatter) FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error {
	fmt.Fprintf(w, "%s: %s \u2192 %s\n", id, oldStatus, newStatus)
	return nil
}

// FormatDepChange renders a dependency add/remove result as plain text.
func (f *ToonFormatter) FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error {
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

// FormatStats renders task statistics in TOON format.
// Two sections: stats summary (single row) and by_priority (5 rows, priorities 0-4).
func (f *ToonFormatter) FormatStats(w io.Writer, stats StatsData) error {
	var sections []string

	// Stats summary section as tabular single row
	sections = append(sections, f.buildStatsSection(stats))

	// by_priority section with 5 rows
	sections = append(sections, f.buildByPrioritySection(stats.ByPriority))

	fmt.Fprint(w, strings.Join(sections, "\n"))
	return nil
}

// FormatMessage renders a simple message as plain text.
func (f *ToonFormatter) FormatMessage(w io.Writer, msg string) error {
	fmt.Fprintln(w, msg)
	return nil
}

// buildTaskSection constructs the task section with dynamic schema.
// parent and closed are omitted from schema when their values are empty.
func (f *ToonFormatter) buildTaskSection(detail TaskDetail) string {
	fields := []string{"id", "title", "status", "priority"}
	values := []string{
		toonEscapeValue(detail.ID),
		toonEscapeValue(detail.Title),
		toonEscapeValue(detail.Status),
		strconv.Itoa(detail.Priority),
	}

	if detail.Parent != "" {
		fields = append(fields, "parent", "parent_title")
		values = append(values, toonEscapeValue(detail.Parent), toonEscapeValue(detail.ParentTitle))
	}

	fields = append(fields, "created", "updated")
	values = append(values,
		toonEscapeValue(detail.Created),
		toonEscapeValue(detail.Updated),
	)

	if detail.Closed != "" {
		fields = append(fields, "closed")
		values = append(values, toonEscapeValue(detail.Closed))
	}

	header := "task{" + strings.Join(fields, ",") + "}:"
	return header + "\n  " + strings.Join(values, ",") + "\n"
}

// buildRelatedSection constructs a blocked_by or children section.
// Always present with count, even when empty.
func (f *ToonFormatter) buildRelatedSection(name string, tasks []RelatedTask) string {
	count := len(tasks)
	header := fmt.Sprintf("%s[%d]{id,title,status}:", name, count)

	if count == 0 {
		return header + "\n"
	}

	var rows []string
	for _, t := range tasks {
		row := "  " + strings.Join([]string{
			toonEscapeValue(t.ID),
			toonEscapeValue(t.Title),
			toonEscapeValue(t.Status),
		}, ",")
		rows = append(rows, row)
	}

	return header + "\n" + strings.Join(rows, "\n") + "\n"
}

// buildDescriptionSection constructs the description section with indented lines.
func (f *ToonFormatter) buildDescriptionSection(desc string) string {
	var sb strings.Builder
	sb.WriteString("description:\n")
	for _, line := range strings.Split(desc, "\n") {
		sb.WriteString("  " + line + "\n")
	}
	return sb.String()
}

// buildStatsSection constructs the stats summary as a tabular single row.
func (f *ToonFormatter) buildStatsSection(stats StatsData) string {
	header := "stats{total,open,in_progress,done,cancelled,ready,blocked}:"
	values := strings.Join([]string{
		strconv.Itoa(stats.Total),
		strconv.Itoa(stats.Open),
		strconv.Itoa(stats.InProgress),
		strconv.Itoa(stats.Done),
		strconv.Itoa(stats.Cancelled),
		strconv.Itoa(stats.Ready),
		strconv.Itoa(stats.Blocked),
	}, ",")
	return header + "\n  " + values + "\n"
}

// buildByPrioritySection constructs the by_priority section with 5 rows (priorities 0-4).
func (f *ToonFormatter) buildByPrioritySection(byPriority [5]int) string {
	header := "by_priority[5]{priority,count}:"
	var rows []string
	for i := 0; i < 5; i++ {
		rows = append(rows, "  "+strconv.Itoa(i)+","+strconv.Itoa(byPriority[i]))
	}
	return header + "\n" + strings.Join(rows, "\n") + "\n"
}

// toonEscapeValue uses the toon-go library to properly escape a string value
// for use in TOON array context (comma-delimited).
func toonEscapeValue(s string) string {
	// Marshal a single-row tabular array to get proper array-context escaping.
	// In array context, commas and other special characters are properly quoted.
	doc := toon.NewObject(
		toon.Field{Key: "a", Value: []toon.Object{
			toon.NewObject(toon.Field{Key: "v", Value: s}),
		}},
	)
	result, err := toon.MarshalString(doc)
	if err != nil {
		return s
	}
	// Result is "a[1]{v}:\n  <value>" - extract the value from the second line
	lines := strings.SplitN(result, "\n", 2)
	if len(lines) == 2 {
		return strings.TrimSpace(lines[1])
	}
	return s
}
