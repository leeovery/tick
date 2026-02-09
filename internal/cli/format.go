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

// newFormatter returns the concrete Formatter for the given output format.
func newFormatter(format OutputFormat) Formatter {
	switch format {
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

// formatTransitionText writes a plain-text status transition line.
// Data must be *TransitionData. Shared by ToonFormatter and PrettyFormatter.
func formatTransitionText(w io.Writer, data interface{}) error {
	d, ok := data.(*TransitionData)
	if !ok {
		return fmt.Errorf("FormatTransition: expected *TransitionData, got %T", data)
	}
	_, err := fmt.Fprintf(w, "%s: %s \u2192 %s\n", d.ID, d.OldStatus, d.NewStatus)
	return err
}

// formatDepChangeText writes a plain-text dependency change confirmation.
// Data must be *DepChangeData. Shared by ToonFormatter and PrettyFormatter.
func formatDepChangeText(w io.Writer, data interface{}) error {
	d, ok := data.(*DepChangeData)
	if !ok {
		return fmt.Errorf("FormatDepChange: expected *DepChangeData, got %T", data)
	}
	switch d.Action {
	case "added":
		_, err := fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", d.TaskID, d.BlockedByID)
		return err
	case "removed":
		_, err := fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", d.TaskID, d.BlockedByID)
		return err
	default:
		return fmt.Errorf("FormatDepChange: unknown action %q", d.Action)
	}
}

// formatMessageText writes a message followed by a newline.
// Shared by ToonFormatter and PrettyFormatter.
func formatMessageText(w io.Writer, msg string) {
	fmt.Fprintln(w, msg)
}
