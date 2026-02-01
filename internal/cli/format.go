package cli

import (
	"fmt"
	"io"
	"os"
)

// Format represents an output format.
type Format int

const (
	// FormatToon is the TOON format for agent consumption.
	FormatToon Format = iota
	// FormatPretty is the human-readable format for terminals.
	FormatPretty
	// FormatJSON is the JSON format for compatibility and debugging.
	FormatJSON
)

// TaskListItem holds data for a single row in a task list.
type TaskListItem struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

// RelatedTask holds context about a related task (blocker, child, parent).
type RelatedTask struct {
	ID     string
	Title  string
	Status string
}

// TaskDetail holds full task detail for show/create/update output.
type TaskDetail struct {
	ID          string
	Title       string
	Status      string
	Priority    int
	Description string
	Parent      *RelatedTask
	Created     string
	Updated     string
	Closed      string
	BlockedBy   []RelatedTask
	Children    []RelatedTask
}

// TransitionData holds status transition output data.
type TransitionData struct {
	ID        string
	OldStatus string
	NewStatus string
}

// DepChangeData holds dependency change output data.
type DepChangeData struct {
	Action    string // "added" or "removed"
	TaskID    string
	BlockedBy string
}

// StatsData holds statistics output data.
type StatsData struct {
	Total       int
	Open        int
	InProgress  int
	Done        int
	Cancelled   int
	Ready       int
	Blocked     int
	ByPriority  [5]int // index 0-4
}

// Formatter defines the interface for formatting command output.
type Formatter interface {
	FormatTaskList(w io.Writer, tasks []TaskListItem) error
	FormatTaskDetail(w io.Writer, detail TaskDetail) error
	FormatTransition(w io.Writer, data TransitionData) error
	FormatDepChange(w io.Writer, data DepChangeData) error
	FormatStats(w io.Writer, data StatsData) error
	FormatMessage(w io.Writer, message string) error
}

// FormatConfig holds the resolved format configuration for a command invocation.
type FormatConfig struct {
	Format  Format
	Quiet   bool
	Verbose bool
}

// DetectTTY returns true if the given file is a terminal (TTY).
// Returns false on stat failure or nil file.
func DetectTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// ResolveFormat determines the output format from flags and TTY state.
// Returns an error if more than one format flag is set.
func ResolveFormat(toon, pretty, json, isTTY bool) (Format, error) {
	count := 0
	if toon {
		count++
	}
	if pretty {
		count++
	}
	if json {
		count++
	}

	if count > 1 {
		return 0, fmt.Errorf("only one of --toon, --pretty, --json may be specified")
	}

	switch {
	case toon:
		return FormatToon, nil
	case pretty:
		return FormatPretty, nil
	case json:
		return FormatJSON, nil
	default:
		if isTTY {
			return FormatPretty, nil
		}
		return FormatToon, nil
	}
}

// newFormatter creates the appropriate Formatter for the given format.
func newFormatter(f Format) Formatter {
	switch f {
	case FormatToon:
		return &ToonFormatter{}
	case FormatPretty:
		return &PrettyFormatter{}
	case FormatJSON:
		return &JSONFormatter{}
	default:
		return &ToonFormatter{}
	}
}

// StubFormatter is a placeholder that produces minimal output.
// Used during testing and as a fallback before concrete formatters are wired in.
type StubFormatter struct{}

// FormatTaskList outputs a minimal task list.
func (s *StubFormatter) FormatTaskList(w io.Writer, tasks []TaskListItem) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(w, "No tasks found.")
		return err
	}
	for _, t := range tasks {
		if _, err := fmt.Fprintf(w, "%s %s %d %s\n", t.ID, t.Status, t.Priority, t.Title); err != nil {
			return err
		}
	}
	return nil
}

// FormatTaskDetail outputs minimal task detail.
func (s *StubFormatter) FormatTaskDetail(w io.Writer, detail TaskDetail) error {
	_, err := fmt.Fprintf(w, "ID: %s\nTitle: %s\nStatus: %s\n", detail.ID, detail.Title, detail.Status)
	return err
}

// FormatTransition outputs a status transition.
func (s *StubFormatter) FormatTransition(w io.Writer, data TransitionData) error {
	_, err := fmt.Fprintf(w, "%s: %s → %s\n", data.ID, data.OldStatus, data.NewStatus)
	return err
}

// FormatDepChange outputs a dependency change confirmation.
func (s *StubFormatter) FormatDepChange(w io.Writer, data DepChangeData) error {
	if data.Action == "added" {
		_, err := fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", data.TaskID, data.BlockedBy)
		return err
	}
	_, err := fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", data.TaskID, data.BlockedBy)
	return err
}

// FormatStats outputs statistics.
func (s *StubFormatter) FormatStats(w io.Writer, data StatsData) error {
	_, err := fmt.Fprintf(w, "Total: %d\n", data.Total)
	return err
}

// FormatMessage outputs a simple message.
func (s *StubFormatter) FormatMessage(w io.Writer, message string) error {
	_, err := fmt.Fprintln(w, message)
	return err
}
