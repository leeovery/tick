package cli

import (
	"errors"
	"io"
	"os"
)

// Format represents the output format type.
type Format string

// Format constants for output selection.
const (
	FormatToon   Format = "toon"
	FormatPretty Format = "pretty"
	FormatJSON   Format = "json"
)

// FormatConfig holds output configuration passed to handlers.
type FormatConfig struct {
	Format  Format
	Quiet   bool
	Verbose bool
}

// TaskListData holds data for formatting a list of tasks.
type TaskListData struct {
	Tasks []TaskRowData
}

// TaskRowData holds basic task info for list display.
type TaskRowData struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

// TaskDetailData holds full task info for detail display.
type TaskDetailData struct {
	ID          string
	Title       string
	Status      string
	Priority    int
	Description string
	Parent      string
	ParentTitle string
	Created     string
	Updated     string
	Closed      string
	BlockedBy   []RelatedTaskData
	Children    []RelatedTaskData
}

// RelatedTaskData holds info for related tasks (blockers, children).
type RelatedTaskData struct {
	ID     string
	Title  string
	Status string
}

// StatsData holds statistics for formatting.
type StatsData struct {
	Total      int
	Open       int
	InProgress int
	Done       int
	Cancelled  int
	Ready      int
	Blocked    int
	ByPriority []PriorityCount
}

// PriorityCount holds count for a priority level.
type PriorityCount struct {
	Priority int
	Count    int
}

// Formatter defines the interface for output formatting.
// All commands use a Formatter to produce output strings.
type Formatter interface {
	// FormatTaskList formats a list of tasks.
	FormatTaskList(data *TaskListData) string

	// FormatTaskDetail formats full task details.
	FormatTaskDetail(data *TaskDetailData) string

	// FormatTransition formats a status transition message.
	// action is "add" or "remove", taskID and blockedByID are the IDs involved.
	FormatTransition(taskID, oldStatus, newStatus string) string

	// FormatDepChange formats a dependency change message.
	FormatDepChange(action, taskID, blockedByID string) string

	// FormatStats formats statistics output.
	FormatStats(data *StatsData) string

	// FormatMessage formats a simple message (e.g., "No tasks found.").
	FormatMessage(msg string) string
}

// DetectTTY checks if the given writer is a terminal (TTY).
// Returns false if writer is not an *os.File, if Stat() fails,
// or if the file is not a character device.
func DetectTTY(w io.Writer) bool {
	if w == nil {
		return false
	}

	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		// Stat failure -> default to non-TTY
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// ResolveFormat determines the output format from flags and TTY status.
// Returns error if more than one format flag is set.
// If no flags set, returns Pretty for TTY, Toon for non-TTY.
func ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY bool) (Format, error) {
	// Count how many format flags are set
	count := 0
	if toonFlag {
		count++
	}
	if prettyFlag {
		count++
	}
	if jsonFlag {
		count++
	}

	// Error if more than one flag set
	if count > 1 {
		return "", errors.New("cannot specify multiple format flags (--toon, --pretty, --json)")
	}

	// Return format based on flag
	if toonFlag {
		return FormatToon, nil
	}
	if prettyFlag {
		return FormatPretty, nil
	}
	if jsonFlag {
		return FormatJSON, nil
	}

	// No flags - auto-detect based on TTY
	if isTTY {
		return FormatPretty, nil
	}
	return FormatToon, nil
}

// NewFormatConfig creates a FormatConfig from flags and TTY detection.
// Returns error if conflicting format flags are set.
func NewFormatConfig(toonFlag, prettyFlag, jsonFlag, quiet, verbose bool, stdout io.Writer) (FormatConfig, error) {
	isTTY := DetectTTY(stdout)

	format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)
	if err != nil {
		return FormatConfig{}, err
	}

	return FormatConfig{
		Format:  format,
		Quiet:   quiet,
		Verbose: verbose,
	}, nil
}

// Formatter returns the appropriate Formatter for the configured format.
// This is the single point where format is resolved to a concrete formatter.
func (c FormatConfig) Formatter() Formatter {
	switch c.Format {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatPretty:
		return &PrettyFormatter{}
	default:
		return &ToonFormatter{}
	}
}

// StubFormatter is a placeholder formatter that returns empty strings.
// Used as a placeholder until concrete formatters (TOON, Pretty, JSON) are implemented.
type StubFormatter struct{}

// FormatTaskList returns empty string (stub implementation).
func (f *StubFormatter) FormatTaskList(data *TaskListData) string {
	return ""
}

// FormatTaskDetail returns empty string (stub implementation).
func (f *StubFormatter) FormatTaskDetail(data *TaskDetailData) string {
	return ""
}

// FormatTransition returns empty string (stub implementation).
func (f *StubFormatter) FormatTransition(taskID, oldStatus, newStatus string) string {
	return ""
}

// FormatDepChange returns empty string (stub implementation).
func (f *StubFormatter) FormatDepChange(action, taskID, blockedByID string) string {
	return ""
}

// FormatStats returns empty string (stub implementation).
func (f *StubFormatter) FormatStats(data *StatsData) string {
	return ""
}

// FormatMessage returns empty string (stub implementation).
func (f *StubFormatter) FormatMessage(msg string) string {
	return ""
}
