package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	toon "github.com/toon-format/toon-go"
)

// TaskRow holds a single row of task list output data.
type TaskRow struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

// StatsData holds all statistics data for formatting.
type StatsData struct {
	Total      int
	Open       int
	InProgress int
	Done       int
	Cancelled  int
	Ready      int
	Blocked    int
	ByPriority [5]int // index 0-4 = priority 0-4
}

// TransitionData holds the result of a status transition for formatting.
type TransitionData struct {
	ID        string
	OldStatus string
	NewStatus string
}

// DepChangeData holds the result of a dependency change for formatting.
type DepChangeData struct {
	Action      string // "added" or "removed"
	TaskID      string
	BlockedByID string
}

// toonListRow is the toon-go struct-tagged type for list row marshaling.
type toonListRow struct {
	ID       string `toon:"id"`
	Title    string `toon:"title"`
	Status   string `toon:"status"`
	Priority int    `toon:"priority"`
}

// toonListWrapper wraps list rows for toon-go marshaling.
type toonListWrapper struct {
	Tasks []toonListRow `toon:"tasks"`
}

// ToonFormatter implements the Formatter interface using TOON output format.
// It produces token-optimized output for AI agent consumption with schema
// headers, correct counts, and field escaping via the toon-go library.
type ToonFormatter struct{}

// FormatTaskList renders a list of tasks in TOON tabular format.
// Data must be []TaskRow.
func (f *ToonFormatter) FormatTaskList(w io.Writer, data interface{}) error {
	rows, ok := data.([]TaskRow)
	if !ok {
		return fmt.Errorf("FormatTaskList: expected []TaskRow, got %T", data)
	}

	if len(rows) == 0 {
		_, err := fmt.Fprint(w, "tasks[0]{id,title,status,priority}:\n")
		return err
	}

	tRows := make([]toonListRow, len(rows))
	for i, r := range rows {
		tRows[i] = toonListRow(r)
	}

	out, err := toon.MarshalString(toonListWrapper{Tasks: tRows}, toon.WithIndent(2))
	if err != nil {
		return fmt.Errorf("marshaling task list: %w", err)
	}

	_, err = fmt.Fprint(w, out+"\n")
	return err
}

// FormatTaskDetail renders full details of a single task in TOON multi-section
// format. Data must be *showData.
func (f *ToonFormatter) FormatTaskDetail(w io.Writer, data interface{}) error {
	d, ok := data.(*showData)
	if !ok {
		return fmt.Errorf("FormatTaskDetail: expected *showData, got %T", data)
	}

	// Build dynamic schema and values for task section.
	schema := []string{"id", "title", "status", "priority"}
	values := []string{
		d.id,
		escapeField(d.title),
		d.status,
		strconv.Itoa(d.priority),
	}

	if d.parent != "" {
		schema = append(schema, "parent")
		values = append(values, d.parent)
	}

	schema = append(schema, "created", "updated")
	values = append(values, d.created, d.updated)

	if d.closed != "" {
		schema = append(schema, "closed")
		values = append(values, d.closed)
	}

	// Write task section.
	if _, err := fmt.Fprintf(w, "task{%s}:\n", strings.Join(schema, ",")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %s\n", strings.Join(values, ",")); err != nil {
		return err
	}

	// Write blocked_by section (always present).
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if err := writeRelatedSection(w, "blocked_by", d.blockedBy); err != nil {
		return err
	}

	// Write children section (always present).
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if err := writeRelatedSection(w, "children", d.children); err != nil {
		return err
	}

	// Write description section (conditional — omit when empty).
	if strings.TrimSpace(d.description) != "" {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "description:"); err != nil {
			return err
		}
		for _, line := range strings.Split(d.description, "\n") {
			if _, err := fmt.Fprintf(w, "  %s\n", line); err != nil {
				return err
			}
		}
	}

	return nil
}

// FormatTransition renders a status transition result as plain text.
// Data must be *TransitionData.
func (f *ToonFormatter) FormatTransition(w io.Writer, data interface{}) error {
	return formatTransitionText(w, data)
}

// FormatDepChange renders a dependency change confirmation as plain text.
// Data must be *DepChangeData.
func (f *ToonFormatter) FormatDepChange(w io.Writer, data interface{}) error {
	return formatDepChangeText(w, data)
}

// FormatStats renders task statistics in TOON format with a summary section
// and a 5-row by_priority section. Data must be *StatsData.
func (f *ToonFormatter) FormatStats(w io.Writer, data interface{}) error {
	d, ok := data.(*StatsData)
	if !ok {
		return fmt.Errorf("FormatStats: expected *StatsData, got %T", data)
	}

	// Write stats summary section.
	if _, err := fmt.Fprintln(w, "stats{total,open,in_progress,done,cancelled,ready,blocked}:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d,%d,%d,%d,%d,%d,%d\n",
		d.Total, d.Open, d.InProgress, d.Done, d.Cancelled, d.Ready, d.Blocked); err != nil {
		return err
	}

	// Write by_priority section (always 5 rows, 0-4).
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "by_priority[5]{priority,count}:"); err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		if _, err := fmt.Fprintf(w, "  %d,%d\n", i, d.ByPriority[i]); err != nil {
			return err
		}
	}

	return nil
}

// FormatMessage writes the message followed by a newline.
func (f *ToonFormatter) FormatMessage(w io.Writer, msg string) {
	formatMessageText(w, msg)
}

// writeRelatedSection writes a TOON tabular section for related tasks
// (blocked_by or children). Empty slices produce a zero-count header with
// the schema still present.
func writeRelatedSection(w io.Writer, name string, tasks []relatedTask) error {
	if _, err := fmt.Fprintf(w, "%s[%d]{id,title,status}:\n", name, len(tasks)); err != nil {
		return err
	}
	for _, rt := range tasks {
		if _, err := fmt.Fprintf(w, "  %s,%s,%s\n", rt.id, escapeField(rt.title), rt.status); err != nil {
			return err
		}
	}
	return nil
}

// escapeField applies TOON escaping to a string field value using the toon-go
// library. Fields that don't contain special characters are returned as-is for
// efficiency.
func escapeField(s string) string {
	if !strings.ContainsAny(s, ",\"\n\\:[]{}") {
		return s
	}
	type wrapper struct {
		V string `toon:"v"`
	}
	out, err := toon.MarshalString(wrapper{V: s})
	if err != nil {
		return s
	}
	return strings.TrimPrefix(out, "v: ")
}
