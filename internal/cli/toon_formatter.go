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

	sections = append(sections, buildTaskSection(detail.Task))

	sections = append(sections, buildRelatedSection("blocked_by", detail.BlockedBy))

	sections = append(sections, buildRelatedSection("children", detail.Children))

	if len(detail.Tags) > 0 {
		sections = append(sections, encodeToonSection("tags", detail.Tags))
	}

	if len(detail.Refs) > 0 {
		sections = append(sections, encodeToonSection("refs", detail.Refs))
	}

	sections = append(sections, buildNotesSection(detail.Notes))

	if detail.Changes != nil {
		sections = append(sections, buildChangedSection(detail.Changes.Rows()))
	}

	if detail.Task.Description != "" {
		sections = append(sections, encodeToonFields(toon.Field{Key: "description", Value: detail.Task.Description}))
	}

	return joinToonSections(sections)
}

// FormatStats renders task statistics in multi-section TOON format.
func (f *ToonFormatter) FormatStats(stats Stats) string {
	var sections []string

	sections = append(sections, encodeToonFields(
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
	sections = append(sections, encodeToonSection("by_priority", rows))

	return joinToonSections(sections)
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
func buildChangedSection(changes []StatusChange) string {
	if len(changes) == 0 {
		return "changed[0]{id,title,from,to,auto}:"
	}
	rows := make([]toonChangedRow, len(changes))
	for i, c := range changes {
		rows[i] = toonChangedRow(c)
	}
	return encodeToonSection("changed", rows)
}

// FormatCascadeTransition renders every status change the command made as one changed table.
func (f *ToonFormatter) FormatCascadeTransition(result CascadeResult) string {
	return buildChangedSection(result.Changed())
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
func (f *ToonFormatter) FormatDepTree(result DepTreeResult) string {
	if result.Target != nil {
		return f.formatFocusedDepTree(result)
	}

	if result.Message != "" {
		return result.Message
	}

	return f.formatFullDepTree(result)
}

// formatFullDepTree renders the full graph as a dep_tree edge list with summary.
func (f *ToonFormatter) formatFullDepTree(result DepTreeResult) string {
	var edges []toonEdgeRow
	for _, root := range result.Roots {
		edges = append(edges, collectDownstreamEdges(root.Task.ID, root.Children)...)
	}

	sections := []string{
		buildEdgeSection("dep_tree", edges),
		encodeToonFields(
			toon.Field{Key: "chains", Value: result.ChainCount},
			toon.Field{Key: "longest", Value: result.LongestChain},
			toon.Field{Key: "blocked", Value: result.BlockedCount},
		),
	}

	return strings.Join(sections, "\n\n")
}

// formatFocusedDepTree renders focused mode as the target's id, title and status
// followed by its blocked_by and blocks sections.
func (f *ToonFormatter) formatFocusedDepTree(result DepTreeResult) string {
	sections := []string{
		encodeToonFields(
			toon.Field{Key: "id", Value: result.Target.ID},
			toon.Field{Key: "title", Value: result.Target.Title},
			toon.Field{Key: "status", Value: result.Target.Status},
		),
		buildEdgeSection("blocked_by", collectUpstreamEdges(result.Target.ID, result.BlockedBy)),
		buildEdgeSection("blocks", collectDownstreamEdges(result.Target.ID, result.Blocks)),
	}

	return strings.Join(sections, "\n\n")
}

// collectDownstreamEdges recursively collects edges from parent to each child node.
// Used for full graph mode and the "blocks" direction of focused mode.
func collectDownstreamEdges(parentID string, nodes []DepTreeNode) []toonEdgeRow {
	var edges []toonEdgeRow
	for _, node := range nodes {
		edges = append(edges, toonEdgeRow{From: parentID, To: node.Task.ID})
		edges = append(edges, collectDownstreamEdges(node.Task.ID, node.Children)...)
	}
	return edges
}

// collectUpstreamEdges recursively collects edges from each blocker node to the blocked task.
// Used for the "blocked_by" direction of focused mode.
func collectUpstreamEdges(blockedID string, nodes []DepTreeNode) []toonEdgeRow {
	var edges []toonEdgeRow
	for _, node := range nodes {
		edges = append(edges, toonEdgeRow{From: node.Task.ID, To: blockedID})
		edges = append(edges, collectUpstreamEdges(node.Task.ID, node.Children)...)
	}
	return edges
}

// buildEdgeSection renders a named toon section of edge rows.
func buildEdgeSection(name string, edges []toonEdgeRow) string {
	if len(edges) == 0 {
		return fmt.Sprintf("%s[0]{from,to}:", name)
	}
	return encodeToonSection(name, edges)
}

// buildTaskSection builds the task's own fields as top-level named fields,
// omitting type, parent and closed when the task does not carry them.
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

	return encodeToonFields(fields...)
}

// encodeToonFields encodes ordered fields as top-level TOON named fields,
// returning "" when the encoder rejects a value.
func encodeToonFields(fields ...toon.Field) string {
	s, err := toon.MarshalString(toon.NewObject(fields...))
	if err != nil {
		return ""
	}
	return s
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

// buildNotesSection builds the notes section as a TOON tabular section.
func buildNotesSection(notes []task.Note) string {
	if len(notes) == 0 {
		return "notes[0]{index,text,created}:"
	}
	rows := make([]toonNoteRow, len(notes))
	for i, n := range notes {
		rows[i] = toonNoteRow{
			Index:   i + 1,
			Text:    n.Text,
			Created: task.FormatTimestamp(n.Created),
		}
	}
	return encodeToonSection("notes", rows)
}

// encodeToonSection encodes a slice as a named TOON section using toon-go: structs
// become a tabular section, scalars an inline list. Quoting is the encoder's.
func encodeToonSection[T any](name string, rows []T) string {
	obj := toon.NewObject(toon.Field{Key: name, Value: rows})
	s, err := toon.MarshalString(obj)
	if err != nil {
		return fmt.Sprintf("%s[0]:", name)
	}
	return s
}
