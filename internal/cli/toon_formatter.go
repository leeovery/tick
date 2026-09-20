package cli

import (
	"errors"
	"fmt"
	"reflect"
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

// toonNoteRow is a TOON-serializable row for the notes section in show output.
type toonNoteRow struct {
	Index   int    `toon:"index"`
	Text    string `toon:"text"`
	Created string `toon:"created"`
}

// toonPriorityRow is a TOON-serializable row for the by_priority section.
type toonPriorityRow struct {
	Priority int `toon:"priority"`
	Count    int `toon:"count"`
}

// FormatTaskList renders a list of tasks in TOON tabular format.
func (f *ToonFormatter) FormatTaskList(tasks []task.Task) (string, error) {
	if len(tasks) == 0 {
		return emptyToonSection[toonTaskRow]("tasks"), nil
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
	return encodeToonSectionIdentified("tasks", rows, func(r toonTaskRow) string { return r.ID })
}

// FormatTaskDetail renders a single task in multi-section TOON format, narrowed to detail.Fields when it is set.
func (f *ToonFormatter) FormatTaskDetail(detail TaskDetail) (string, error) {
	sel := detail.Fields
	sections := detail.selectedSections()
	var doc toonDoc

	doc.add(buildTaskSection(detail.Task, sel))

	if sel.includes(fieldBlockedBy) {
		doc.add(buildRelatedSection("blocked_by", sections.BlockedBy))
	}

	if sel.includes(fieldChildren) {
		doc.add(buildRelatedSection("children", sections.Children))
	}

	if len(detail.Tags) > 0 && sel.includes(fieldTags) {
		doc.add(encodeToonSection("tags", sections.Tags))
	}

	if len(detail.Refs) > 0 && sel.includes(fieldRefs) {
		doc.add(encodeToonSection("refs", sections.Refs))
	}

	if sel.includes(fieldNotes) {
		doc.add(buildNotesSection(sections.Notes, sections.NotePositions))
	}

	if detail.Changes != nil {
		doc.add(buildChangedSection(detail.Changes.Rows()))
	}

	if detail.Task.Description != "" && sel.includes(fieldDescription) {
		doc.add(encodeToonFields(toon.Field{Key: fieldDescription, Value: detail.Task.Description}))
	}

	return doc.join(detail.Task.ID)
}

// FormatStats renders task statistics in multi-section TOON format.
func (f *ToonFormatter) FormatStats(stats Stats) (string, error) {
	var doc toonDoc

	doc.add(encodeToonFields(
		toon.Field{Key: "total", Value: stats.Total},
		toon.Field{Key: "open", Value: stats.Open},
		toon.Field{Key: "in_progress", Value: stats.InProgress},
		toon.Field{Key: "done", Value: stats.Done},
		toon.Field{Key: "cancelled", Value: stats.Cancelled},
		toon.Field{Key: "ready", Value: stats.Ready},
		toon.Field{Key: "blocked", Value: stats.Blocked},
	))

	rows := make([]toonPriorityRow, 5)
	for i := range 5 {
		rows[i] = toonPriorityRow{Priority: i, Count: stats.ByPriority[i]}
	}
	doc.add(encodeToonSection("by_priority", rows))

	return doc.join("")
}

// FormatMessage renders a general-purpose message as plain text.
func (f *ToonFormatter) FormatMessage(msg string) string {
	return msg
}

// toonChangedRow is a TOON-serializable row of the changed status table.
type toonChangedRow struct {
	ID    string `toon:"id"`
	Title string `toon:"title"`
	From  string `toon:"from"`
	To    string `toon:"to"`
	Auto  bool   `toon:"auto"`
}

// buildChangedSection builds the changed section listing every task whose status moved.
func buildChangedSection(changes []StatusChange) (string, error) {
	if len(changes) == 0 {
		return emptyToonSection[toonChangedRow]("changed"), nil
	}
	rows := make([]toonChangedRow, len(changes))
	for i, c := range changes {
		rows[i] = toonChangedRow(c)
	}
	return encodeToonSection("changed", rows)
}

// FormatCascadeTransition renders every status change the command made as one changed table.
func (f *ToonFormatter) FormatCascadeTransition(result CascadeResult) (string, error) {
	changed, err := buildChangedSection(result.Changed())
	return changed, refusalForTask(err, result.TaskID)
}

// toonEdgeRow is a TOON-serializable row for dep tree edge list output.
type toonEdgeRow struct {
	From string `toon:"from"`
	To   string `toon:"to"`
}

// FormatDepTree renders a dependency tree in TOON edge-list format.
// Full graph: dep_tree[N]{from,to}: section + chains, longest and blocked named fields.
// Focused mode: the target's id, title and status as named fields, followed by
// blocked_by[N]{from,to}: and blocks[N]{from,to}: sections, both always present.
func (f *ToonFormatter) FormatDepTree(result DepTreeResult) (string, error) {
	if result.Target != nil {
		return f.formatFocusedDepTree(result)
	}

	return f.formatFullDepTree(result)
}

func (f *ToonFormatter) formatFullDepTree(result DepTreeResult) (string, error) {
	var doc toonDoc

	doc.add(buildEdgeSection("dep_tree", toonEdgeRows(result.Edges)))
	doc.add(encodeToonFields(
		toon.Field{Key: "chains", Value: result.ChainCount},
		toon.Field{Key: "longest", Value: result.LongestChain},
		toon.Field{Key: "blocked", Value: result.BlockedCount},
	))

	return doc.join("")
}

func (f *ToonFormatter) formatFocusedDepTree(result DepTreeResult) (string, error) {
	var doc toonDoc

	doc.add(encodeToonFields(
		toon.Field{Key: fieldID, Value: result.Target.ID},
		toon.Field{Key: fieldTitle, Value: result.Target.Title},
		toon.Field{Key: fieldStatus, Value: result.Target.Status},
	))
	doc.add(buildEdgeSection("blocked_by", toonEdgeRows(result.BlockedByEdges)))
	doc.add(buildEdgeSection("blocks", toonEdgeRows(result.BlocksEdges)))

	return doc.join(result.Target.ID)
}

func toonEdgeRows(edges []DepTreeEdge) []toonEdgeRow {
	rows := make([]toonEdgeRow, 0, len(edges))
	for _, edge := range edges {
		rows = append(rows, toonEdgeRow(edge))
	}
	return rows
}

// buildEdgeSection renders a named toon section of edge rows.
func buildEdgeSection(name string, edges []toonEdgeRow) (string, error) {
	if len(edges) == 0 {
		return emptyToonSection[toonEdgeRow](name), nil
	}
	return encodeToonSection(name, edges)
}

// buildTaskSection builds the task's own selected fields as top-level named
// fields, omitting type, parent and closed when the task does not carry them
// and returning "" when no field survives.
func buildTaskSection(t task.Task, sel *FieldSelection) (string, error) {
	var fields []toon.Field
	add := func(key string, value any) {
		if sel.includes(key) {
			fields = append(fields, toon.Field{Key: key, Value: value})
		}
	}

	add(fieldID, t.ID)
	add(fieldTitle, t.Title)
	add(fieldStatus, string(t.Status))
	add(fieldPriority, t.Priority)

	if t.Type != "" {
		add(fieldType, t.Type)
	}

	if t.Parent != "" {
		add(fieldParent, t.Parent)
	}

	add(fieldCreated, task.FormatTimestamp(t.Created))
	add(fieldUpdated, task.FormatTimestamp(t.Updated))

	if t.Closed != nil {
		add(fieldClosed, task.FormatTimestamp(*t.Closed))
	}

	if len(fields) == 0 {
		return "", nil
	}

	return encodeToonFields(fields...)
}

// encodeToonFields encodes ordered fields as top-level TOON named fields,
// returning a *toonEncodeError naming the refused field when the encoder
// rejects a value.
func encodeToonFields(fields ...toon.Field) (string, error) {
	s, err := toon.MarshalString(toon.NewObject(fields...))
	if err != nil {
		return "", fieldRefusal(fields, err)
	}
	return s, nil
}

// fieldRefusal names the first field the encoder rejects on its own; the
// library error names none.
func fieldRefusal(fields []toon.Field, err error) error {
	for _, f := range fields {
		if _, single := toon.MarshalString(toon.NewObject(f)); single != nil {
			return &toonEncodeError{part: "field " + f.Key, err: err}
		}
	}
	return &toonEncodeError{part: "the named fields", err: err}
}

// joinToonSections joins non-empty sections with a blank line between them.
func joinToonSections(sections []string) string {
	kept := make([]string, 0, len(sections))
	for _, section := range sections {
		if section != "" {
			kept = append(kept, section)
		}
	}
	return strings.Join(kept, "\n\n")
}

// buildRelatedSection builds a blocked_by or children section.
func buildRelatedSection(name string, related []RelatedTask) (string, error) {
	if len(related) == 0 {
		return emptyToonSection[toonRelatedRow](name), nil
	}
	rows := make([]toonRelatedRow, len(related))
	for i, r := range related {
		rows[i] = toonRelatedRow(r)
	}
	return encodeToonSection(name, rows)
}

// buildNotesSection builds the notes section as a TOON tabular section, each
// row carrying the position the note holds in the whole section.
func buildNotesSection(notes []task.Note, positions []int) (string, error) {
	if len(notes) == 0 {
		return emptyToonSection[toonNoteRow]("notes"), nil
	}
	rows := make([]toonNoteRow, len(notes))
	for i, n := range notes {
		rows[i] = toonNoteRow{
			Index:   positions[i],
			Text:    n.Text,
			Created: task.FormatTimestamp(n.Created),
		}
	}
	return encodeToonSection("notes", rows)
}

// emptyToonSection renders the header a named TOON section of T rows carries when it
// holds none, naming T's toon-tagged fields in declaration order as its columns.
func emptyToonSection[T any](name string) string {
	rowType := reflect.TypeFor[T]()
	cols := make([]string, rowType.NumField())
	for i := range cols {
		cols[i], _, _ = strings.Cut(rowType.Field(i).Tag.Get("toon"), ",")
	}
	return fmt.Sprintf("%s[0]{%s}:", name, strings.Join(cols, ","))
}

// encodeToonSection encodes a slice as a named TOON section using toon-go: structs
// become a tabular section, scalars an inline list. Quoting is the encoder's.
// It returns a *toonEncodeError naming the section when the encoder rejects a value.
func encodeToonSection[T any](name string, rows []T) (string, error) {
	return encodeToonSectionIdentified(name, rows, nil)
}

// encodeToonSectionIdentified encodes a section like encodeToonSection, naming
// the refused row in the refusal when identity resolves a row's task ID.
func encodeToonSectionIdentified[T any](name string, rows []T, identity func(T) string) (string, error) {
	obj := toon.NewObject(toon.Field{Key: name, Value: rows})
	s, err := toon.MarshalString(obj)
	if err != nil {
		return "", sectionRefusal(name, rows, identity, err)
	}
	return s, nil
}

// sectionRefusal names the first row the encoder rejects on its own; the
// library error names none. Without an identity, or with no row refused alone,
// the section name stands by itself.
func sectionRefusal[T any](name string, rows []T, identity func(T) string, err error) error {
	refusal := &toonEncodeError{part: "section " + name, err: err}
	if identity == nil {
		return refusal
	}
	for _, row := range rows {
		if _, single := toon.MarshalString(toon.NewObject(toon.Field{Key: name, Value: []T{row}})); single != nil {
			refusal.taskID = identity(row)
			break
		}
	}
	return refusal
}

// toonEncodeError reports a value TOON cannot carry, naming the field or
// section that holds it and the task the document covers when it covers one.
type toonEncodeError struct {
	part   string
	taskID string
	err    error
}

func (e *toonEncodeError) Error() string {
	if e.taskID == "" {
		return fmt.Sprintf("cannot encode %s as TOON: %v", e.part, e.err)
	}
	return fmt.Sprintf("cannot encode %s of task %s as TOON: %v", e.part, e.taskID, e.err)
}

func (e *toonEncodeError) Unwrap() error { return e.err }

// refusalForTask returns err naming taskID when err is a TOON refusal.
func refusalForTask(err error, taskID string) error {
	var refusal *toonEncodeError
	if taskID == "" || !errors.As(err, &refusal) {
		return err
	}
	return &toonEncodeError{part: refusal.part, taskID: taskID, err: refusal.err}
}

// toonDoc collects a document's sections, keeping the first refusal instead.
type toonDoc struct {
	sections []string
	err      error
}

func (d *toonDoc) add(section string, err error) {
	if d.err != nil {
		return
	}
	if err != nil {
		d.err = err
		return
	}
	d.sections = append(d.sections, section)
}

// join renders the collected sections, or the refusal naming taskID for a
// document covering a single task.
func (d *toonDoc) join(taskID string) (string, error) {
	if d.err != nil {
		return "", refusalForTask(d.err, taskID)
	}
	return joinToonSections(d.sections), nil
}
