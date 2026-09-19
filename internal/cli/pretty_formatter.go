package cli

import (
	"cmp"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

// treeStyle defines the box-drawing characters used by writeTree.
type treeStyle struct {
	mid       string // connector prefix for non-last siblings (e.g. "├── ")
	last      string // connector prefix for last sibling (e.g. "└── ")
	childMid  string // indentation prefix under non-last siblings (e.g. "│   ")
	childLast string // indentation prefix under last sibling (e.g. "    ")
}

// writeTree renders a generic tree of nodes with box-drawing connectors.
// Each node is preceded by a newline. renderLine writes the node content
// (excluding the newline) given the writer, node, combined prefix, and depth.
// getChildren returns the child nodes for recursion.
func writeTree[T any](w io.Writer, nodes []T, prefix string, depth int, style treeStyle, renderLine func(io.Writer, T, string, int), getChildren func(T) []T) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		connector := style.mid
		if isLast {
			connector = style.last
		}
		fmt.Fprint(w, "\n")
		renderLine(w, node, prefix+connector, depth)

		kids := getChildren(node)
		if len(kids) > 0 {
			childPrefix := prefix + style.childMid
			if isLast {
				childPrefix = prefix + style.childLast
			}
			writeTree(w, kids, childPrefix, depth+1, style, renderLine, getChildren)
		}
	}
}

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

// prettyDetailLine renders one header line of the detail document, padding the
// label to the column the widest label ("Priority:") sets.
func prettyDetailLine(label, value string) string {
	return fmt.Sprintf("%-9s %s", label+":", value)
}

func prettyDetailBlock(label string, entries []string) string {
	var b strings.Builder
	b.WriteString(label + ":")
	for _, entry := range entries {
		fmt.Fprintf(&b, "\n  %s", entry)
	}
	return b.String()
}

func prettyRelatedEntries(related []RelatedTask) []string {
	entries := make([]string, len(related))
	for i, r := range related {
		entries[i] = fmt.Sprintf("%s  %s (%s)", r.ID, r.Title, r.Status)
	}
	return entries
}

// prettyDetailHeader returns the header lines the selection keeps, in document order.
func prettyDetailHeader(detail TaskDetail) []string {
	t := detail.Task
	sel := detail.Fields

	var lines []string
	add := func(name, label, value string) {
		if sel.includes(name) {
			lines = append(lines, prettyDetailLine(label, value))
		}
	}

	add(fieldID, "ID", t.ID)
	add(fieldTitle, "Title", t.Title)
	add(fieldStatus, "Status", string(t.Status))
	add(fieldPriority, "Priority", strconv.Itoa(t.Priority))
	add(fieldType, "Type", typeOrDash(t.Type))

	if len(detail.Tags) > 0 {
		tags, _ := selectedItems(detail.Tags, sel.Positions(fieldTags))
		add(fieldTags, "Tags", strings.Join(tags, ", "))
	}

	if t.Parent != "" {
		parent := t.Parent
		if detail.ParentTitle != "" {
			parent = fmt.Sprintf("%s (%s)", t.Parent, detail.ParentTitle)
		}
		add(fieldParent, "Parent", parent)
	}

	add(fieldCreated, "Created", task.FormatTimestamp(t.Created))
	add(fieldUpdated, "Updated", task.FormatTimestamp(t.Updated))

	if t.Closed != nil {
		add(fieldClosed, "Closed", task.FormatTimestamp(*t.Closed))
	}

	return lines
}

// prettyDetailBlocks returns the blocks the selection keeps, in document order.
// A block the task has nothing for is omitted whether or not it was selected.
func prettyDetailBlocks(detail TaskDetail) []string {
	sel := detail.Fields
	var blocks []string

	if len(detail.BlockedBy) > 0 && sel.includes(fieldBlockedBy) {
		blockedBy, _ := selectedItems(detail.BlockedBy, sel.Positions(fieldBlockedBy))
		blocks = append(blocks, prettyDetailBlock("Blocked by", prettyRelatedEntries(blockedBy)))
	}

	if len(detail.Children) > 0 && sel.includes(fieldChildren) {
		children, _ := selectedItems(detail.Children, sel.Positions(fieldChildren))
		blocks = append(blocks, prettyDetailBlock("Children", prettyRelatedEntries(children)))
	}

	if len(detail.Refs) > 0 && sel.includes(fieldRefs) {
		refs, _ := selectedItems(detail.Refs, sel.Positions(fieldRefs))
		blocks = append(blocks, prettyDetailBlock("Refs", refs))
	}

	if len(detail.Notes) > 0 && sel.includes(fieldNotes) {
		notes, _ := selectedItems(detail.Notes, sel.Positions(fieldNotes))
		entries := make([]string, len(notes))
		for i, note := range notes {
			entries[i] = fmt.Sprintf("%s  %s", note.Created.Format("2006-01-02 15:04"), note.Text)
		}
		blocks = append(blocks, prettyDetailBlock("Notes", entries))
	}

	if detail.Task.Description != "" && sel.includes(fieldDescription) {
		blocks = append(blocks, prettyDetailBlock("Description", strings.Split(detail.Task.Description, "\n")))
	}

	return blocks
}

// FormatTaskDetail renders a single task in key-value format, narrowed to
// detail.Fields when it is set. A section the task has nothing for is omitted,
// and the whole document is empty when the selection keeps nothing.
func (f *PrettyFormatter) FormatTaskDetail(detail TaskDetail) string {
	var groups []string
	if header := prettyDetailHeader(detail); len(header) > 0 {
		groups = append(groups, strings.Join(header, "\n"))
	}
	groups = append(groups, prettyDetailBlocks(detail)...)

	if len(groups) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(strings.Join(groups, "\n\n"))

	if detail.Changes != nil {
		for _, block := range detail.Changes.Blocks {
			fmt.Fprintf(&b, "\n%s", f.FormatCascadeTransition(block))
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

// cascadeNode represents a node in the cascade tree for pretty-format rendering.
type cascadeNode struct {
	id       string
	text     string
	children []*cascadeNode
}

// FormatCascadeTransition renders a cascade transition with box-drawing tree characters.
// Entries are organized into a tree using ParentID. Entries whose ParentID equals the
// primary task ID are top-level; entries whose ParentID matches another entry are nested.
func (f *PrettyFormatter) FormatCascadeTransition(result CascadeResult) string {
	if result.TaskID == "" {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s \u2192 %s", result.TaskID, result.OldStatus, result.NewStatus)

	if len(result.Cascaded) == 0 {
		return b.String()
	}

	// Build a map of nodes keyed by ID for tree construction.
	nodes := make(map[string]*cascadeNode)
	// Ordered list of all node IDs to preserve insertion order.
	var orderedIDs []string

	for _, c := range result.Cascaded {
		n := &cascadeNode{
			id:   c.ID,
			text: fmt.Sprintf("%s %q: %s \u2192 %s", c.ID, c.Title, c.OldStatus, c.NewStatus),
		}
		nodes[c.ID] = n
		orderedIDs = append(orderedIDs, c.ID)
	}
	// Build tree: attach children to parents.
	var roots []*cascadeNode
	parentIDOf := make(map[string]string)
	for _, c := range result.Cascaded {
		parentIDOf[c.ID] = c.ParentID
	}

	for _, id := range orderedIDs {
		pid := parentIDOf[id]
		if parent, ok := nodes[pid]; ok {
			parent.children = append(parent.children, nodes[id])
		} else {
			roots = append(roots, nodes[id])
		}
	}

	b.WriteString("\n\nCascaded:")
	writeCascadeTree(&b, roots, "")

	return b.String()
}

// cascadeTreeStyle defines box-drawing characters for cascade transition trees.
var cascadeTreeStyle = treeStyle{
	mid:       "\u251c\u2500",
	last:      "\u2514\u2500",
	childMid:  "\u2502  ",
	childLast: "   ",
}

// writeCascadeTree recursively renders tree nodes with box-drawing characters.
func writeCascadeTree(b *strings.Builder, nodes []*cascadeNode, prefix string) {
	writeTree(b, nodes, prefix, 0, cascadeTreeStyle,
		func(w io.Writer, n *cascadeNode, linePrefix string, _ int) {
			fmt.Fprintf(w, "%s %s", linePrefix, n.text)
		},
		func(n *cascadeNode) []*cascadeNode {
			return n.children
		},
	)
}

// depTreeLineWidth is the assumed terminal width for title truncation in dep tree output.
const depTreeLineWidth = 80

// depTreeMinTitle is the minimum number of title characters to display before truncating.
const depTreeMinTitle = 10

// FormatDepTree renders a dependency tree visualization with box-drawing characters.
// Supports both full-graph mode (Roots populated) and focused mode (Target populated).
func (f *PrettyFormatter) FormatDepTree(result DepTreeResult) string {
	if result.Target != nil {
		return f.formatFocusedDepTree(result)
	}
	return f.formatFullDepTree(result)
}

// formatFullDepTree renders every full-graph tree with its downstream dependencies and a summary line.
// The no-dependencies message stands in only when the graph holds no participant at all.
func (f *PrettyFormatter) formatFullDepTree(result DepTreeResult) string {
	trees := result.fullGraphTrees()
	if len(trees) == 0 {
		return result.Message
	}

	var b strings.Builder
	for i, tree := range trees {
		if i > 0 {
			b.WriteString("\n")
		}
		writeDepTreeTaskLine(&b, tree.Task, "", 0)
		writeDepTreeNodes(&b, tree.Children, "", 1)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(result.Summary)
	return b.String()
}

// formatFocusedDepTree renders the target task header followed by labeled
// "Blocked by:" and "Blocks:" sections, omitting empty sections.
// When both directions are empty, renders the task info followed by the message.
func (f *PrettyFormatter) formatFocusedDepTree(result DepTreeResult) string {
	var b strings.Builder
	writeDepTreeTaskLine(&b, *result.Target, "", 0)

	if len(result.BlockedBy) == 0 && len(result.Blocks) == 0 && result.Message != "" {
		b.WriteString("\n")
		b.WriteString(result.Message)
		return b.String()
	}

	if len(result.BlockedBy) > 0 {
		b.WriteString("\n\nBlocked by:")
		writeDepTreeNodes(&b, result.BlockedBy, "", 1)
	}

	if len(result.Blocks) > 0 {
		b.WriteString("\n\nBlocks:")
		writeDepTreeNodes(&b, result.Blocks, "", 1)
	}

	return b.String()
}

// writeDepTreeTaskLine writes a single task line: {prefix}{id}  {title} ({status}).
// The title is truncated to fit within depTreeLineWidth.
func writeDepTreeTaskLine(w io.Writer, task DepTreeTask, prefix string, depth int) {
	title := truncateDepTreeTitle(task.Title, depth)
	fmt.Fprintf(w, "%s%s  %s (%s)", prefix, task.ID, title, task.Status)
}

// depTreeStyle defines box-drawing characters for dependency tree rendering.
var depTreeStyle = treeStyle{
	mid:       "├── ",
	last:      "└── ",
	childMid:  "│   ",
	childLast: "    ",
}

// writeDepTreeNodes recursively renders tree nodes with box-drawing characters.
// Each node is written as a new line (prefixed with \n).
func writeDepTreeNodes(b *strings.Builder, nodes []DepTreeNode, prefix string, depth int) {
	writeTree(b, nodes, prefix, depth, depTreeStyle,
		func(w io.Writer, node DepTreeNode, linePrefix string, d int) {
			writeDepTreeTaskLine(w, node.Task, linePrefix, d)
		},
		func(node DepTreeNode) []DepTreeNode {
			return node.Children
		},
	)
}

// truncateDepTreeTitle truncates a title to fit the available width in dep tree output.
// Available width accounts for indentation (depth * 4), ID length, status, and formatting.
// Ensures at minimum depTreeMinTitle characters are shown (or full title if shorter).
func truncateDepTreeTitle(title string, depth int) string {
	// Overhead: prefix (depth*4) + ID (~11 chars "tick-XXXXXX") + 2 spaces + " (" + status (~11 chars max "in_progress") + ")"
	// = depth*4 + 11 + 2 + 2 + 11 + 1 = depth*4 + 27
	overhead := depth*4 + 27
	available := max(depTreeLineWidth-overhead, depTreeMinTitle)
	if len(title) <= available {
		return title
	}
	if available <= 3 {
		return title[:available]
	}
	return title[:available-3] + "..."
}

// typeOrDash returns the type string or "-" if empty, for Pretty formatter display.
func typeOrDash(typ string) string {
	return cmp.Or(typ, "-")
}

// truncateTitle truncates a title to maxListTitleLen characters, appending "..." if truncated.
func truncateTitle(title string) string {
	if len(title) <= maxListTitleLen {
		return title
	}
	return title[:maxListTitleLen-3] + "..."
}
