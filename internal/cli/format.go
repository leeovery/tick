package cli

import (
	"fmt"
	"io"
	"os"
)

// Formatter defines the interface for rendering command output in different
// formats (Toon, Pretty, JSON). Concrete implementations are provided in
// tasks 4-2 through 4-4.
type Formatter interface {
	// FormatTaskList renders a list of tasks (for tick list, ready, blocked).
	FormatTaskList(w io.Writer, data interface{}) error
	// FormatTaskDetail renders full details of a single task (for tick show, create, update).
	FormatTaskDetail(w io.Writer, data interface{}) error
	// FormatTransition renders a status transition result (for tick start, done, cancel, reopen).
	FormatTransition(w io.Writer, data interface{}) error
	// FormatDepChange renders a dependency change confirmation (for tick dep add/rm).
	FormatDepChange(w io.Writer, data interface{}) error
	// FormatStats renders task statistics (for tick stats).
	FormatStats(w io.Writer, data interface{}) error
	// FormatMessage renders a simple text message (for tick init, rebuild, etc.).
	FormatMessage(w io.Writer, msg string)
}

// FormatConfig holds the resolved output configuration for a CLI invocation.
// It is derived from global flags and TTY detection, then passed to handlers.
type FormatConfig struct {
	Format  OutputFormat
	Quiet   bool
	Verbose bool
}

// DetectTTY checks whether the given file is connected to a terminal (TTY).
// Returns false on stat failure, defaulting to non-TTY.
func DetectTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ResolveFormat determines the output format from explicit flag overrides and
// TTY status. If more than one format flag is set, an error is returned.
// With no flags, TTY defaults to Pretty and non-TTY defaults to Toon.
func ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY bool) (OutputFormat, error) {
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

	if count > 1 {
		return 0, fmt.Errorf("only one format flag may be specified (--toon, --pretty, --json)")
	}

	switch {
	case toonFlag:
		return FormatToon, nil
	case prettyFlag:
		return FormatPretty, nil
	case jsonFlag:
		return FormatJSON, nil
	default:
		if isTTY {
			return FormatPretty, nil
		}
		return FormatToon, nil
	}
}

// StubFormatter is a placeholder implementation of the Formatter interface.
// It will be replaced by concrete Toon, Pretty, and JSON formatters in
// tasks 4-2 through 4-4.
type StubFormatter struct{}

// FormatTaskList is a stub that returns nil.
func (f *StubFormatter) FormatTaskList(w io.Writer, data interface{}) error {
	return nil
}

// FormatTaskDetail is a stub that returns nil.
func (f *StubFormatter) FormatTaskDetail(w io.Writer, data interface{}) error {
	return nil
}

// FormatTransition is a stub that returns nil.
func (f *StubFormatter) FormatTransition(w io.Writer, data interface{}) error {
	return nil
}

// FormatDepChange is a stub that returns nil.
func (f *StubFormatter) FormatDepChange(w io.Writer, data interface{}) error {
	return nil
}

// FormatStats is a stub that returns nil.
func (f *StubFormatter) FormatStats(w io.Writer, data interface{}) error {
	return nil
}

// FormatMessage writes the message followed by a newline.
func (f *StubFormatter) FormatMessage(w io.Writer, msg string) {
	fmt.Fprintln(w, msg)
}
