// Package cli provides the command-line interface for Tick.
// It handles subcommand dispatch, global flags, TTY detection, and error formatting.
package cli

import (
	"fmt"
	"io"
	"os"
)

// OutputFormat represents the output format for CLI responses.
type OutputFormat string

const (
	// FormatTOON is the default for non-TTY (agent-optimized output).
	FormatTOON OutputFormat = "toon"
	// FormatPretty is the default for TTY (human-readable output).
	FormatPretty OutputFormat = "pretty"
	// FormatJSON forces JSON output.
	FormatJSON OutputFormat = "json"
)

// App is the top-level CLI application.
// It holds global flags, I/O writers, and the working directory.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Dir    string

	// Global flags
	Quiet   bool
	Verbose bool

	// Output format
	OutputFormat OutputFormat
	IsTTY        bool
}

// Run parses global flags, determines the subcommand, and dispatches it.
// Returns exit code: 0 for success, 1 for error.
func (a *App) Run(args []string) int {
	// Detect TTY and set default format
	a.detectTTY()

	// Parse global flags from args[1:] (skip program name)
	remaining, err := a.parseGlobalFlags(args[1:])
	if err != nil {
		a.writeError(err)
		return 1
	}

	// Determine subcommand
	if len(remaining) == 0 {
		a.printUsage()
		return 0
	}

	subcommand := remaining[0]
	subArgs := remaining[1:]

	// Dispatch subcommand
	switch subcommand {
	case "init":
		if err := a.runInit(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "create":
		if err := a.runCreate(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "list":
		if err := a.runList(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "show":
		if err := a.runShow(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "update":
		if err := a.runUpdate(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "start", "done", "cancel", "reopen":
		if err := a.runTransition(subcommand, subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "dep":
		if err := a.runDep(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	default:
		a.writeError(fmt.Errorf("Unknown command '%s'. Run 'tick help' for usage.", subcommand))
		return 1
	}
}

// detectTTY checks if stdout is a terminal device.
// For non-*os.File writers (e.g., bytes.Buffer), it defaults to non-TTY.
func (a *App) detectTTY() {
	a.IsTTY = false
	a.OutputFormat = FormatTOON

	if f, ok := a.Stdout.(*os.File); ok {
		info, err := f.Stat()
		if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
			a.IsTTY = true
			a.OutputFormat = FormatPretty
		}
	}
}

// parseGlobalFlags extracts global flags and returns the remaining arguments.
func (a *App) parseGlobalFlags(args []string) ([]string, error) {
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--quiet", "-q":
			a.Quiet = true
		case "--verbose", "-v":
			a.Verbose = true
		case "--toon":
			a.OutputFormat = FormatTOON
		case "--pretty":
			a.OutputFormat = FormatPretty
		case "--json":
			a.OutputFormat = FormatJSON
		default:
			// Not a global flag; everything from here on is subcommand + subcommand args
			remaining = append(remaining, args[i:]...)
			return remaining, nil
		}
	}

	return remaining, nil
}

// writeError writes a formatted error message to stderr.
func (a *App) writeError(err error) {
	fmt.Fprintf(a.Stderr, "Error: %s\n", err.Error())
}

// printUsage writes basic usage information to stdout.
func (a *App) printUsage() {
	fmt.Fprintln(a.Stdout, "Usage: tick <command> [options]")
	fmt.Fprintln(a.Stdout, "")
	fmt.Fprintln(a.Stdout, "Commands:")
	fmt.Fprintln(a.Stdout, "  init    Initialize a new tick project")
	fmt.Fprintln(a.Stdout, "")
	fmt.Fprintln(a.Stdout, "Global options:")
	fmt.Fprintln(a.Stdout, "  --quiet, -q     Suppress non-essential output")
	fmt.Fprintln(a.Stdout, "  --verbose, -v   More detail for debugging")
	fmt.Fprintln(a.Stdout, "  --toon          Force TOON output format")
	fmt.Fprintln(a.Stdout, "  --pretty        Force human-readable output format")
	fmt.Fprintln(a.Stdout, "  --json          Force JSON output format")
}
