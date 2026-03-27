package cli

import (
	"fmt"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

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

// FormatTaskDetail renders a single task with full details in key-value format.
// Sections (Blocked by, Children, Description) are omitted when empty.
func (f *PrettyFormatter) FormatTaskDetail(detail TaskDetail) string {
	t := detail.Task
	var b strings.Builder

	fmt.Fprintf(&b, "ID:       %s\n", t.ID)
	fmt.Fprintf(&b, "Title:    %s\n", t.Title)
	fmt.Fprintf(&b, "Status:   %s\n", string(t.Status))
	fmt.Fprintf(&b, "Priority: %d\n", t.Priority)

	fmt.Fprintf(&b, "Type:     %s\n", typeOrDash(t.Type))

	if len(detail.Tags) > 0 {
		fmt.Fprintf(&b, "Tags:     %s\n", strings.Join(detail.Tags, ", "))
	}

	if t.Parent != "" {
		if detail.ParentTitle != "" {
			fmt.Fprintf(&b, "Parent:   %s (%s)\n", t.Parent, detail.ParentTitle)
		} else {
			fmt.Fprintf(&b, "Parent:   %s\n", t.Parent)
		}
	}

	fmt.Fprintf(&b, "Created:  %s\n", task.FormatTimestamp(t.Created))
	fmt.Fprintf(&b, "Updated:  %s", task.FormatTimestamp(t.Updated))

	if t.Closed != nil {
		fmt.Fprintf(&b, "\nClosed:   %s", task.FormatTimestamp(*t.Closed))
	}

	if len(detail.BlockedBy) > 0 {
		b.WriteString("\n\nBlocked by:")
		for _, r := range detail.BlockedBy {
			fmt.Fprintf(&b, "\n  %s  %s (%s)", r.ID, r.Title, r.Status)
		}
	}

	if len(detail.Children) > 0 {
		b.WriteString("\n\nChildren:")
		for _, r := range detail.Children {
			fmt.Fprintf(&b, "\n  %s  %s (%s)", r.ID, r.Title, r.Status)
		}
	}

	if len(detail.Refs) > 0 {
		b.WriteString("\n\nRefs:")
		for _, ref := range detail.Refs {
			fmt.Fprintf(&b, "\n  %s", ref)
		}
	}

	if len(detail.Notes) > 0 {
		b.WriteString("\n\nNotes:")
		for _, note := range detail.Notes {
			fmt.Fprintf(&b, "\n  %s  %s", note.Created.Format("2006-01-02 15:04"), note.Text)
		}
	}

	if t.Description != "" {
		b.WriteString("\n\nDescription:")
		lines := strings.Split(t.Description, "\n")
		for _, line := range lines {
			fmt.Fprintf(&b, "\n  %s", line)
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

// writeCascadeTree recursively renders tree nodes with box-drawing characters.
func writeCascadeTree(b *strings.Builder, nodes []*cascadeNode, prefix string) {
	for i, n := range nodes {
		isLast := i == len(nodes)-1
		connector := "\u251c\u2500"
		if isLast {
			connector = "\u2514\u2500"
		}
		fmt.Fprintf(b, "\n%s%s %s", prefix, connector, n.text)

		if len(n.children) > 0 {
			childPrefix := prefix + "\u2502  "
			if isLast {
				childPrefix = prefix + "   "
			}
			writeCascadeTree(b, n.children, childPrefix)
		}
	}
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

// formatFullDepTree renders root tasks with their downstream dependency trees and a summary line.
func (f *PrettyFormatter) formatFullDepTree(result DepTreeResult) string {
	if len(result.Roots) == 0 {
		return ""
	}

	var b strings.Builder
	for i, root := range result.Roots {
		if i > 0 {
			b.WriteString("\n")
		}
		writeDepTreeTaskLine(&b, root.Task, "", 0)
		writeDepTreeNodes(&b, root.Children, "", 1)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(result.Summary)
	return b.String()
}

// formatFocusedDepTree renders the target task header followed by labeled
// "Blocked by:" and "Blocks:" sections, omitting empty sections.
func (f *PrettyFormatter) formatFocusedDepTree(result DepTreeResult) string {
	var b strings.Builder
	writeDepTreeTaskLine(&b, *result.Target, "", 0)

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
func writeDepTreeTaskLine(b *strings.Builder, task DepTreeTask, prefix string, depth int) {
	title := truncateDepTreeTitle(task.Title, depth)
	fmt.Fprintf(b, "%s%s  %s (%s)", prefix, task.ID, title, task.Status)
}

// writeDepTreeNodes recursively renders tree nodes with box-drawing characters.
// Each node is written as a new line (prefixed with \n).
func writeDepTreeNodes(b *strings.Builder, nodes []DepTreeNode, prefix string, depth int) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		b.WriteString("\n")
		writeDepTreeTaskLine(b, node.Task, prefix+connector, depth)

		if len(node.Children) > 0 {
			childPrefix := prefix + "│   "
			if isLast {
				childPrefix = prefix + "    "
			}
			writeDepTreeNodes(b, node.Children, childPrefix, depth+1)
		}
	}
}

// truncateDepTreeTitle truncates a title to fit the available width in dep tree output.
// Available width accounts for indentation (depth * 4), ID length, status, and formatting.
// Ensures at minimum depTreeMinTitle characters are shown (or full title if shorter).
func truncateDepTreeTitle(title string, depth int) string {
	// Overhead: prefix (depth*4) + ID (~11 chars "tick-XXXXXX") + 2 spaces + " (" + status (~11 chars max "in_progress") + ")"
	// = depth*4 + 11 + 2 + 2 + 11 + 1 = depth*4 + 27
	overhead := depth*4 + 27
	available := depTreeLineWidth - overhead
	if available < depTreeMinTitle {
		available = depTreeMinTitle
	}
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
	if typ == "" {
		return "-"
	}
	return typ
}

// truncateTitle truncates a title to maxListTitleLen characters, appending "..." if truncated.
func truncateTitle(title string) string {
	if len(title) <= maxListTitleLen {
		return title
	}
	return title[:maxListTitleLen-3] + "..."
}
