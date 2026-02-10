package cli

import (
	"fmt"
	"io"
	"os"

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
// including the task itself plus related context (blockers, children, parent title).
type TaskDetail struct {
	Task        task.Task
	BlockedBy   []RelatedTask
	Children    []RelatedTask
	ParentTitle string
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

// Formatter defines the interface for rendering CLI output in different formats.
// Concrete implementations (Toon, Pretty, JSON) are provided by tasks 4-2 through 4-4.
type Formatter interface {
	// FormatTaskList renders a list of tasks.
	FormatTaskList(tasks []task.Task) string
	// FormatTaskDetail renders a single task with full details including related context.
	FormatTaskDetail(detail TaskDetail) string
	// FormatTransition renders a status transition (e.g., "open -> in_progress").
	FormatTransition(id string, oldStatus string, newStatus string) string
	// FormatDepChange renders a dependency add/remove confirmation.
	FormatDepChange(action string, taskID string, depID string) string
	// FormatStats renders task statistics.
	FormatStats(stats Stats) string
	// FormatMessage renders a general-purpose message.
	FormatMessage(msg string) string
}

// StubFormatter is a placeholder implementation of Formatter.
// It returns empty strings for all methods. Replaced by concrete formatters in tasks 4-2 through 4-4.
type StubFormatter struct{}

// Compile-time interface verification.
var _ Formatter = (*StubFormatter)(nil)

// FormatTaskList returns an empty string (stub).
func (s *StubFormatter) FormatTaskList(_ []task.Task) string { return "" }

// FormatTaskDetail returns an empty string (stub).
func (s *StubFormatter) FormatTaskDetail(_ TaskDetail) string { return "" }

// FormatTransition returns an empty string (stub).
func (s *StubFormatter) FormatTransition(_, _, _ string) string { return "" }

// FormatDepChange returns an empty string (stub).
func (s *StubFormatter) FormatDepChange(_, _, _ string) string { return "" }

// FormatStats returns an empty string (stub).
func (s *StubFormatter) FormatStats(_ Stats) string { return "" }

// FormatMessage returns an empty string (stub).
func (s *StubFormatter) FormatMessage(_ string) string { return "" }

// VerboseLog writes a verbose message to the given writer (intended for stderr).
// Only writes if verbose is true.
func VerboseLog(w io.Writer, verbose bool, msg string) {
	if verbose {
		fmt.Fprintln(w, msg)
	}
}
