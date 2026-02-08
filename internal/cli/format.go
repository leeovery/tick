package cli

import (
	"fmt"
	"io"
	"os"
)

// Format represents the output format for CLI responses.
type Format int

const (
	// FormatToon is the default for non-TTY (agent-optimized output).
	FormatToon Format = iota
	// FormatPretty is the default for TTY (human-readable output).
	FormatPretty
	// FormatJSON forces JSON output.
	FormatJSON
)

// FormatConfig holds format-related configuration passed to all command handlers.
type FormatConfig struct {
	Format  Format
	Quiet   bool
	Verbose bool
}

// RelatedTask holds a related task's summary (used in blocked_by and children lists).
type RelatedTask struct {
	ID     string
	Title  string
	Status string
}

// TaskDetail holds data for rendering a single task's full detail.
type TaskDetail struct {
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
	BlockedBy   []RelatedTask
	Children    []RelatedTask
}

// StatsData holds data for rendering task statistics.
type StatsData struct {
	Total      int
	Open       int
	InProgress int
	Done       int
	Cancelled  int
	Ready      int
	Blocked    int
	ByPriority [5]int // index 0-4 maps to priority 0-4
}

// Formatter is the interface for rendering command output in different formats.
// Concrete implementations (toon, pretty, json) are provided in tasks 4-2 through 4-4.
type Formatter interface {
	// FormatTaskList renders a list of tasks (used by list, ready, blocked commands).
	FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
	// FormatTaskDetail renders full detail for a single task (used by show, create, update).
	FormatTaskDetail(w io.Writer, detail TaskDetail) error
	// FormatTransition renders a status transition (used by start, done, cancel, reopen).
	FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
	// FormatDepChange renders a dependency add/remove result (used by dep add, dep rm).
	FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
	// FormatStats renders task statistics (used by stats command).
	FormatStats(w io.Writer, stats StatsData) error
	// FormatMessage renders a simple message (used by init, rebuild, etc.).
	FormatMessage(w io.Writer, msg string) error
}

// StubFormatter is a placeholder that implements the Formatter interface.
// It will be replaced by concrete formatters (toon, pretty, json) in tasks 4-2 through 4-4.
type StubFormatter struct{}

// FormatTaskList is a stub implementation.
func (f *StubFormatter) FormatTaskList(w io.Writer, rows []listRow, quiet bool) error {
	return nil
}

// FormatTaskDetail is a stub implementation.
func (f *StubFormatter) FormatTaskDetail(w io.Writer, detail TaskDetail) error {
	return nil
}

// FormatTransition is a stub implementation.
func (f *StubFormatter) FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error {
	return nil
}

// FormatDepChange is a stub implementation.
func (f *StubFormatter) FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error {
	return nil
}

// FormatStats is a stub implementation.
func (f *StubFormatter) FormatStats(w io.Writer, stats StatsData) error {
	return nil
}

// FormatMessage is a stub implementation.
func (f *StubFormatter) FormatMessage(w io.Writer, msg string) error {
	return nil
}

// resolveFormatter returns the concrete Formatter for the given format.
func resolveFormatter(f Format) Formatter {
	switch f {
	case FormatPretty:
		return &PrettyFormatter{}
	case FormatJSON:
		return &JSONFormatter{}
	default:
		return &ToonFormatter{}
	}
}

// formatName returns a human-readable name for the given Format.
func formatName(f Format) string {
	switch f {
	case FormatToon:
		return "toon"
	case FormatPretty:
		return "pretty"
	case FormatJSON:
		return "json"
	default:
		return "unknown"
	}
}

// DetectTTY checks if the given writer is connected to a terminal device.
// Returns false for non-*os.File writers and on Stat failure.
func DetectTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// ResolveFormat determines the output format from flag overrides and TTY state.
// If more than one format flag is set, returns an error.
// If no flags are set, defaults to Pretty for TTY and Toon for non-TTY.
func ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY bool) (Format, error) {
	flagCount := 0
	if toonFlag {
		flagCount++
	}
	if prettyFlag {
		flagCount++
	}
	if jsonFlag {
		flagCount++
	}

	if flagCount > 1 {
		return 0, fmt.Errorf("only one format flag allowed; got multiple of --toon, --pretty, --json")
	}

	if toonFlag {
		return FormatToon, nil
	}
	if prettyFlag {
		return FormatPretty, nil
	}
	if jsonFlag {
		return FormatJSON, nil
	}

	// No flags: auto-detect from TTY
	if isTTY {
		return FormatPretty, nil
	}
	return FormatToon, nil
}
