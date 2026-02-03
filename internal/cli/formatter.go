package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/leeovery/tick/internal/task"
)

// errStatFailure is a sentinel error used in tests for stat failure simulation.
var errStatFailure = errors.New("stat failure")

// FileInfo is the interface returned by file stat operations.
// Matches the os.FileInfo interface's Mode method for TTY detection.
type FileInfo interface {
	Mode() os.FileMode
}

// FormatConfig holds the resolved output configuration passed to command handlers.
type FormatConfig struct {
	Format  OutputFormat
	Quiet   bool
	Verbose bool
}

// Formatter defines the interface for rendering command output in different formats.
// Concrete implementations (Toon, Pretty, JSON) are provided in subsequent tasks.
type Formatter interface {
	// FormatTaskList renders a list of tasks (for tick list, tick ready, tick blocked).
	FormatTaskList(w io.Writer, tasks []TaskRow) error
	// FormatTaskDetail renders full details for a single task (for tick show, tick create, tick update).
	FormatTaskDetail(w io.Writer, data *showData) error
	// FormatTransition renders a status transition (for tick start, tick done, tick cancel, tick reopen).
	FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
	// FormatDepChange renders a dependency add/remove confirmation (for tick dep add, tick dep rm).
	FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
	// FormatStats renders task statistics (for tick stats).
	FormatStats(w io.Writer, stats interface{}) error
	// FormatMessage renders a simple message (for tick init, tick rebuild, and similar).
	FormatMessage(w io.Writer, message string) error
}

// TaskRow holds the minimal fields for list output rendering.
type TaskRow struct {
	ID       string
	Status   string
	Priority int
	Title    string
}

// StubFormatter is a placeholder formatter that delegates to fmt.Fprint.
// It satisfies the Formatter interface and will be replaced by concrete
// implementations (Toon, Pretty, JSON) in tasks 4-2 through 4-4.
type StubFormatter struct{}

// FormatTaskList writes a basic task list output.
func (f *StubFormatter) FormatTaskList(w io.Writer, tasks []TaskRow) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(w, "No tasks found.")
		return err
	}
	_, err := fmt.Fprintf(w, "%-12s%-12s%-4s%s\n", "ID", "STATUS", "PRI", "TITLE")
	if err != nil {
		return err
	}
	for _, r := range tasks {
		_, err := fmt.Fprintf(w, "%-12s%-12s%-4d%s\n", r.ID, r.Status, r.Priority, r.Title)
		if err != nil {
			return err
		}
	}
	return nil
}

// FormatTaskDetail writes basic task detail output.
func (f *StubFormatter) FormatTaskDetail(w io.Writer, data *showData) error {
	_, err := fmt.Fprintf(w, "ID:       %s\n", data.ID)
	return err
}

// FormatTransition writes a status transition message.
func (f *StubFormatter) FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error {
	_, err := fmt.Fprintf(w, "%s: %s \u2192 %s\n", id, oldStatus, newStatus)
	return err
}

// FormatDepChange writes a dependency change confirmation.
func (f *StubFormatter) FormatDepChange(w io.Writer, action, taskID, blockedByID string) error {
	_, err := fmt.Fprintf(w, "Dependency %s: %s %s %s\n", action, taskID, action, blockedByID)
	return err
}

// FormatStats writes basic stats output.
func (f *StubFormatter) FormatStats(w io.Writer, stats interface{}) error {
	_, err := fmt.Fprintf(w, "%v\n", stats)
	return err
}

// FormatMessage writes a simple message.
func (f *StubFormatter) FormatMessage(w io.Writer, message string) error {
	_, err := fmt.Fprintln(w, message)
	return err
}

// DetectTTY checks whether os.Stdout is a terminal (TTY).
// Returns false on stat failure (safe default for non-TTY/agent environments).
func DetectTTY() bool {
	return DetectTTYFrom(func() (FileInfo, error) {
		fi, err := os.Stdout.Stat()
		if err != nil {
			return nil, err
		}
		return fi, nil
	})
}

// DetectTTYFrom checks TTY status using a provided stat function.
// This allows testing with custom stat functions, including simulating failures.
func DetectTTYFrom(statFn func() (FileInfo, error)) bool {
	fi, err := statFn()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// ResolveFormat determines the output format from flags and TTY status.
// If more than one format flag is set, it returns an error.
// If exactly one flag is set, that format is returned.
// If no flags are set, TTY -> Pretty, non-TTY -> Toon.
func ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY bool) (OutputFormat, error) {
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
		return "", fmt.Errorf("only one format flag allowed: --toon, --pretty, or --json")
	}

	switch {
	case toonFlag:
		return FormatTOON, nil
	case prettyFlag:
		return FormatPretty, nil
	case jsonFlag:
		return FormatJSON, nil
	default:
		// Auto-detect based on TTY
		if isTTY {
			return FormatPretty, nil
		}
		return FormatTOON, nil
	}
}
