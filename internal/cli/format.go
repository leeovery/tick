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
	FormatTaskList(w io.Writer, rows []TaskRow) error
	// FormatTaskDetail renders full details of a single task (for tick show, create, update).
	FormatTaskDetail(w io.Writer, data *showData) error
	// FormatTransition renders a status transition result (for tick start, done, cancel, reopen).
	FormatTransition(w io.Writer, data *TransitionData) error
	// FormatDepChange renders a dependency change confirmation (for tick dep add/rm).
	FormatDepChange(w io.Writer, data *DepChangeData) error
	// FormatStats renders task statistics (for tick stats).
	FormatStats(w io.Writer, data *StatsData) error
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
// Shared by ToonFormatter and PrettyFormatter.
func formatTransitionText(w io.Writer, data *TransitionData) error {
	_, err := fmt.Fprintf(w, "%s: %s \u2192 %s\n", data.ID, data.OldStatus, data.NewStatus)
	return err
}

// formatDepChangeText writes a plain-text dependency change confirmation.
// Shared by ToonFormatter and PrettyFormatter.
func formatDepChangeText(w io.Writer, data *DepChangeData) error {
	switch data.Action {
	case "added":
		_, err := fmt.Fprintf(w, "Dependency added: %s blocked by %s\n", data.TaskID, data.BlockedByID)
		return err
	case "removed":
		_, err := fmt.Fprintf(w, "Dependency removed: %s no longer blocked by %s\n", data.TaskID, data.BlockedByID)
		return err
	default:
		return fmt.Errorf("FormatDepChange: unknown action %q", data.Action)
	}
}

// formatMessageText writes a message followed by a newline.
// Shared by ToonFormatter and PrettyFormatter.
func formatMessageText(w io.Writer, msg string) {
	fmt.Fprintln(w, msg)
}
