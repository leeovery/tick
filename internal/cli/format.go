package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

// Format represents the selected output format for CLI responses.
type Format int

const (
	// FormatToon is the token-oriented format for agent consumption.
	FormatToon Format = iota
	// FormatPretty is the human-readable table output for terminals.
	FormatPretty
	// FormatJSON is the standard JSON output format.
	FormatJSON
)

// DetectTTY checks if the given *os.File is connected to a terminal (TTY).
// Returns false on stat failure (defaulting to non-TTY).
func DetectTTY(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

// ResolveFormat determines the output format based on flags and TTY detection.
// Returns an error if more than one format flag is set.
func ResolveFormat(flags globalFlags, isTTY bool) (Format, error) {
	count := 0
	if flags.toon {
		count++
	}
	if flags.pretty {
		count++
	}
	if flags.json {
		count++
	}
	if count > 1 {
		return 0, fmt.Errorf("only one format flag (--toon, --pretty, --json) may be specified")
	}

	if flags.toon {
		return FormatToon, nil
	}
	if flags.pretty {
		return FormatPretty, nil
	}
	if flags.json {
		return FormatJSON, nil
	}
	if isTTY {
		return FormatPretty, nil
	}
	return FormatToon, nil
}

// FormatConfig holds resolved formatting configuration passed to all command handlers.
type FormatConfig struct {
	Format  Format
	Quiet   bool
	Verbose bool
	// Logger is the verbose logger. Nil when verbose is disabled.
	Logger *VerboseLogger
}

// NewFormatConfig builds a FormatConfig from parsed global flags and TTY state.
func NewFormatConfig(flags globalFlags, isTTY bool) (FormatConfig, error) {
	f, err := ResolveFormat(flags, isTTY)
	if err != nil {
		return FormatConfig{}, err
	}
	return FormatConfig{
		Format:  f,
		Quiet:   flags.quiet,
		Verbose: flags.verbose,
	}, nil
}

// RelatedTask represents a task referenced in blocked_by or children sections of show output.
type RelatedTask struct {
	ID     string
	Title  string
	Status string
}

// TaskDetail holds all data needed to render the show command output,
// including the task itself plus related context (blockers, children, parent title, tags, refs, notes).
type TaskDetail struct {
	Task        task.Task
	BlockedBy   []RelatedTask
	Children    []RelatedTask
	ParentTitle string
	Tags        []string
	Refs        []string
	Notes       []task.Note
	// Fields narrows the document to the named fields; nil means the whole document.
	Fields *FieldSelection
	// Changes is nil when the document carries no changed section, and non-nil —
	// possibly with no rows — when it always carries one.
	Changes *StatusChanges
}

// StatusChanges holds a command's status changes as the per-transition cascade results
// rendered by pretty; Rows flattens them into the merged table rendered by toon and JSON.
type StatusChanges struct {
	Blocks []CascadeResult
}

// Rows merges the blocks into one table in which each task appears at most once.
func (c StatusChanges) Rows() []StatusChange {
	return mergeStatusChanges(c.Blocks...)
}

// Stats holds typed task statistics for rendering by formatters.
type Stats struct {
	Total      int
	Open       int
	InProgress int
	Done       int
	Cancelled  int
	Ready      int
	Blocked    int
	ByPriority [5]int // index 0-4 maps to priority P0-P4
}

// RemovedTask holds the ID and title of a task that was removed.
type RemovedTask struct {
	ID    string
	Title string
}

// RemovalResult holds the outcome of a remove operation for rendering by formatters.
type RemovalResult struct {
	Removed     []RemovedTask
	DepsUpdated []string
}

// CascadeEntry holds a single cascaded status change for display.
type CascadeEntry struct {
	ID        string
	Title     string
	ParentID  string
	OldStatus string
	NewStatus string
}

// StatusChange is one row of a command's status-change table: a task that moved,
// read from the status it held before the command to the status it holds after.
// Auto is false only for a change the caller asked for.
type StatusChange struct {
	ID    string
	Title string
	From  string
	To    string
	Auto  bool
}

// CascadeResult holds all data needed to render a cascade transition. PrimaryAuto is
// false only when the caller asked for the primary transition.
type CascadeResult struct {
	TaskID      string
	TaskTitle   string
	OldStatus   string
	NewStatus   string
	PrimaryAuto bool
	Cascaded    []CascadeEntry
}

// Changed flattens the result into one merged row per task that moved.
func (c CascadeResult) Changed() []StatusChange {
	var set statusChangeSet
	set.add(StatusChange{
		ID:    c.TaskID,
		Title: c.TaskTitle,
		From:  c.OldStatus,
		To:    c.NewStatus,
		Auto:  c.PrimaryAuto,
	})
	for _, e := range c.Cascaded {
		set.add(StatusChange{
			ID:    e.ID,
			Title: e.Title,
			From:  e.OldStatus,
			To:    e.NewStatus,
			Auto:  true,
		})
	}
	return set.rows()
}

// DepTreeTask holds the minimal task data needed for dependency tree rendering.
type DepTreeTask struct {
	ID     string
	Title  string
	Status string
}

// DepTreeEdge is one stored dependency: the blocker From blocks the task To.
type DepTreeEdge struct {
	From string
	To   string
}

// DepTreeNode represents a node in the dependency tree with its children.
type DepTreeNode struct {
	Task     DepTreeTask
	Children []DepTreeNode
}

// DepTreeResult holds all data needed to render a dep tree command output.
// For full graph mode: Trees holds the drawn trees, Edges holds one entry per stored
// dependency, and summary stats are populated.
// For focused mode: BlockedBy and Blocks contain upstream/downstream trees, and
// BlockedByEdges and BlocksEdges hold one entry per stored dependency inside the
// target's upstream and downstream neighbourhood.
type DepTreeResult struct {
	// Full graph mode fields
	Trees        []DepTreeNode
	Edges        []DepTreeEdge
	Summary      string
	ChainCount   int
	LongestChain int
	BlockedCount int

	// Focused mode fields
	Target         *DepTreeTask
	BlockedBy      []DepTreeNode
	Blocks         []DepTreeNode
	BlockedByEdges []DepTreeEdge
	BlocksEdges    []DepTreeEdge

	// Message is the no-dependencies sentence; the machine formats answer an empty
	// graph with the emptied document instead.
	Message string
}

// Formatter defines the interface for rendering CLI output in different formats.
type Formatter interface {
	// FormatTaskList renders a list of tasks.
	FormatTaskList(tasks []task.Task) (string, error)
	// FormatTaskDetail renders a single task with its related context, narrowed
	// to detail.Fields when the caller set a selection.
	FormatTaskDetail(detail TaskDetail) (string, error)
	// FormatDepChange renders a dependency add/remove confirmation.
	FormatDepChange(action string, taskID string, depID string) string
	// FormatStats renders task statistics.
	FormatStats(stats Stats) (string, error)
	// FormatMessage renders a general-purpose message.
	FormatMessage(msg string) string
	// FormatRemoval renders the result of a task removal operation.
	FormatRemoval(result RemovalResult) string
	// FormatCascadeTransition renders the status changes a command made. It
	// returns an error when the format cannot carry a value it was handed,
	// rather than a document that contradicts the data.
	FormatCascadeTransition(result CascadeResult) (string, error)
	// FormatDepTree renders a dependency tree visualization.
	FormatDepTree(result DepTreeResult) (string, error)
}

// baseFormatter provides shared implementations of FormatDepChange and FormatRemoval
// for text-based formatters (Toon and Pretty).
type baseFormatter struct{}

// FormatDepChange renders a dependency add/remove confirmation as plain text.
func (b *baseFormatter) FormatDepChange(action string, taskID string, depID string) string {
	if action == "removed" {
		return fmt.Sprintf("Dependency removed: %s no longer blocked by %s", taskID, depID)
	}
	return fmt.Sprintf("Dependency added: %s blocked by %s", taskID, depID)
}

// FormatRemoval renders the result of a task removal as plain text.
// One line per removed task as 'Removed {id} "{title}"', plus an optional
// dependency update line if DepsUpdated is non-empty.
func (b *baseFormatter) FormatRemoval(result RemovalResult) string {
	var lines []string
	for _, r := range result.Removed {
		lines = append(lines, fmt.Sprintf("Removed %s %q", r.ID, r.Title))
	}
	if len(result.DepsUpdated) > 0 {
		lines = append(lines, fmt.Sprintf("Updated dependencies on %s", strings.Join(result.DepsUpdated, ", ")))
	}
	return strings.Join(lines, "\n")
}

// StubFormatter is a placeholder implementation of Formatter.
// It returns empty strings for all methods.
type StubFormatter struct{}

// Compile-time interface verification.
var _ Formatter = (*StubFormatter)(nil)

// FormatTaskList returns an empty string (stub).
func (s *StubFormatter) FormatTaskList(_ []task.Task) (string, error) { return "", nil }

// FormatTaskDetail returns an empty string (stub).
func (s *StubFormatter) FormatTaskDetail(_ TaskDetail) (string, error) { return "", nil }

// FormatDepChange returns an empty string (stub).
func (s *StubFormatter) FormatDepChange(_, _, _ string) string { return "" }

// FormatStats returns an empty string (stub).
func (s *StubFormatter) FormatStats(_ Stats) (string, error) { return "", nil }

// FormatMessage returns an empty string (stub).
func (s *StubFormatter) FormatMessage(_ string) string { return "" }

// FormatRemoval returns an empty string (stub).
func (s *StubFormatter) FormatRemoval(_ RemovalResult) string { return "" }

// FormatCascadeTransition returns an empty string (stub).
func (s *StubFormatter) FormatCascadeTransition(_ CascadeResult) (string, error) { return "", nil }

// FormatDepTree returns an empty string (stub).
func (s *StubFormatter) FormatDepTree(_ DepTreeResult) (string, error) { return "", nil }

// NewFormatter creates a Formatter for the given Format.
func NewFormatter(f Format) Formatter {
	switch f {
	case FormatPretty:
		return &PrettyFormatter{}
	case FormatJSON:
		return &JSONFormatter{}
	default:
		return &ToonFormatter{}
	}
}
