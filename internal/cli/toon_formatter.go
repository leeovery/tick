package cli

import (
	"fmt"
	"strings"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/task"
)

// ToonFormatter renders CLI output in TOON (Token-Oriented Object Notation) format,
// optimized for AI agent consumption with 30-60% token savings over JSON.
type ToonFormatter struct {
	baseFormatter
}

// Compile-time interface verification.
var _ Formatter = (*ToonFormatter)(nil)

// toonTaskRow is a TOON-serializable row for task list output.
type toonTaskRow struct {
	ID       string `toon:"id"`
	Title    string `toon:"title"`
	Status   string `toon:"status"`
	Priority int    `toon:"priority"`
	Type     string `toon:"type"`
}

// toonRelatedRow is a TOON-serializable row for blocked_by/children sections.
type toonRelatedRow struct {
	ID     string `toon:"id"`
	Title  string `toon:"title"`
	Status string `toon:"status"`
}

// toonStatsSummary is a TOON-serializable row for the stats summary section.
type toonStatsSummary struct {
	Total      int `toon:"total"`
	Open       int `toon:"open"`
	InProgress int `toon:"in_progress"`
	Done       int `toon:"done"`
	Cancelled  int `toon:"cancelled"`
	Ready      int `toon:"ready"`
	Blocked    int `toon:"blocked"`
}

// toonNoteRow is a TOON-serializable row for the notes section in show output.
type toonNoteRow struct {
	Text    string `toon:"text"`
	Created string `toon:"created"`
}

// toonPriorityRow is a TOON-serializable row for the by_priority section.
type toonPriorityRow struct {
	Priority int `toon:"priority"`
	Count    int `toon:"count"`
}

// FormatTaskList renders a list of tasks in TOON tabular format.
func (f *ToonFormatter) FormatTaskList(tasks []task.Task) string {
	if len(tasks) == 0 {
		return "tasks[0]{id,title,status,priority,type}:"
	}
	rows := make([]toonTaskRow, len(tasks))
	for i, t := range tasks {
		rows[i] = toonTaskRow{
			ID:       t.ID,
			Title:    t.Title,
			Status:   string(t.Status),
			Priority: t.Priority,
			Type:     t.Type,
		}
	}
	return encodeToonSection("tasks", rows)
}

// FormatTaskDetail renders a single task with full details in multi-section TOON format.
func (f *ToonFormatter) FormatTaskDetail(detail TaskDetail) string {
	var sections []string

	// Section 1: task (single-object scope with dynamic schema)
	sections = append(sections, buildTaskSection(detail.Task))

	// Section 2: blocked_by (always present, even with count 0)
	sections = append(sections, buildRelatedSection("blocked_by", detail.BlockedBy))

	// Section 3: children (always present, even with count 0)
	sections = append(sections, buildRelatedSection("children", detail.Children))

	// Section 4: tags (omitted when empty)
	if len(detail.Tags) > 0 {
		sections = append(sections, buildTagsSection(detail.Tags))
	}

	// Section 5: refs (omitted when empty)
	if len(detail.Refs) > 0 {
		sections = append(sections, buildRefsSection(detail.Refs))
	}

	// Section 6: notes (always present, even with count 0)
	sections = append(sections, buildNotesSection(detail.Notes))

	// Section 7: description (omitted when empty)
	if detail.Task.Description != "" {
		sections = append(sections, buildDescriptionSection(detail.Task.Description))
	}

	return strings.Join(sections, "\n\n")
}

// FormatStats renders task statistics in multi-section TOON format.
func (f *ToonFormatter) FormatStats(stats Stats) string {
	var sections []string

	// Section 1: stats summary (single-object scope)
	summary := toonStatsSummary{
		Total:      stats.Total,
		Open:       stats.Open,
		InProgress: stats.InProgress,
		Done:       stats.Done,
		Cancelled:  stats.Cancelled,
		Ready:      stats.Ready,
		Blocked:    stats.Blocked,
	}
	sections = append(sections, encodeToonSingleObject("stats", summary))

	// Section 2: by_priority (always 5 rows, 0-4)
	rows := make([]toonPriorityRow, 5)
	for i := 0; i < 5; i++ {
		rows[i] = toonPriorityRow{Priority: i, Count: stats.ByPriority[i]}
	}
	sections = append(sections, encodeToonSection("by_priority", rows))

	return strings.Join(sections, "\n\n")
}

// FormatMessage renders a general-purpose message as plain text.
func (f *ToonFormatter) FormatMessage(msg string) string {
	return msg
}

// buildTaskSection builds the task section with dynamic schema (omitting parent/closed when null).
func buildTaskSection(t task.Task) string {
	var fields []toon.Field

	fields = append(fields,
		toon.Field{Key: "id", Value: t.ID},
		toon.Field{Key: "title", Value: t.Title},
		toon.Field{Key: "status", Value: string(t.Status)},
		toon.Field{Key: "priority", Value: t.Priority},
	)

	if t.Type != "" {
		fields = append(fields, toon.Field{Key: "type", Value: t.Type})
	}

	if t.Parent != "" {
		fields = append(fields, toon.Field{Key: "parent", Value: t.Parent})
	}

	fields = append(fields,
		toon.Field{Key: "created", Value: task.FormatTimestamp(t.Created)},
		toon.Field{Key: "updated", Value: task.FormatTimestamp(t.Updated)},
	)

	if t.Closed != nil {
		fields = append(fields, toon.Field{Key: "closed", Value: task.FormatTimestamp(*t.Closed)})
	}

	row := toon.NewObject(fields...)
	// Encode as 1-element array under "task" key, then strip "[1]" for single-object scope
	wrapper := toon.NewObject(toon.Field{Key: "task", Value: []toon.Object{row}})
	s, err := toon.MarshalString(wrapper)
	if err != nil {
		return "task:"
	}
	return strings.Replace(s, "task[1]", "task", 1)
}

// buildRelatedSection builds a blocked_by or children section.
func buildRelatedSection(name string, related []RelatedTask) string {
	if len(related) == 0 {
		return fmt.Sprintf("%s[0]{id,title,status}:", name)
	}
	rows := make([]toonRelatedRow, len(related))
	for i, r := range related {
		rows[i] = toonRelatedRow(r)
	}
	return encodeToonSection(name, rows)
}

// buildTagsSection builds the tags section as a TOON array of strings.
func buildTagsSection(tags []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "tags[%d]:", len(tags))
	for _, tag := range tags {
		b.WriteString("\n  ")
		b.WriteString(tag)
	}
	return b.String()
}

// buildRefsSection builds the refs section as a TOON array of strings.
func buildRefsSection(refs []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "refs[%d]:", len(refs))
	for _, ref := range refs {
		b.WriteString("\n  ")
		b.WriteString(ref)
	}
	return b.String()
}

// buildNotesSection builds the notes section as a TOON tabular section.
func buildNotesSection(notes []task.Note) string {
	if len(notes) == 0 {
		return "notes[0]{text,created}:"
	}
	rows := make([]toonNoteRow, len(notes))
	for i, n := range notes {
		rows[i] = toonNoteRow{
			Text:    n.Text,
			Created: task.FormatTimestamp(n.Created),
		}
	}
	return encodeToonSection("notes", rows)
}

// buildDescriptionSection builds the description section with indented lines.
func buildDescriptionSection(desc string) string {
	var b strings.Builder
	b.WriteString("description:")
	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		b.WriteString("\n  ")
		b.WriteString(line)
	}
	return b.String()
}

// encodeToonSection encodes an array of structs as a TOON tabular section using toon-go.
// It wraps the rows in a named field, marshals via toon-go, and returns the result.
func encodeToonSection[T any](name string, rows []T) string {
	// Build an Object with the named array field
	obj := toon.NewObject(toon.Field{Key: name, Value: rows})
	s, err := toon.MarshalString(obj)
	if err != nil {
		return fmt.Sprintf("%s[0]:", name)
	}
	return s
}

// encodeToonSingleObject encodes a single struct as a TOON object scope (no [N] count).
// It encodes as a 1-element array via toon-go, then strips the "[1]" from the header.
func encodeToonSingleObject[T any](name string, value T) string {
	obj := toon.NewObject(toon.Field{Key: name, Value: []T{value}})
	s, err := toon.MarshalString(obj)
	if err != nil {
		return name + ":"
	}
	// Replace "name[1]" with "name" to get single-object scope format
	return strings.Replace(s, name+"[1]", name, 1)
}
