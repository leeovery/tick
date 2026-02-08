// Package cli provides the command-line interface for Tick.
// It handles subcommand dispatch, global flags, TTY detection, and error formatting.
package cli

import (
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/store"
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
	OutputFormat Format
	IsTTY        bool

	// FormatCfg holds the resolved format configuration passed to command handlers.
	FormatCfg FormatConfig

	// Formatter is the resolved output formatter (set once in Run).
	Formatter Formatter

	// vlog is the verbose logger, created during Run.
	vlog *VerboseLogger
}

// Run parses global flags, determines the subcommand, and dispatches it.
// Returns exit code: 0 for success, 1 for error.
func (a *App) Run(args []string) int {
	// Detect TTY
	a.IsTTY = DetectTTY(a.Stdout)

	// Parse global flags from args[1:] (skip program name)
	remaining, err := a.parseGlobalFlags(args[1:])
	if err != nil {
		a.writeError(err)
		return 1
	}

	// Create verbose logger (writes to stderr when verbose enabled)
	a.vlog = NewVerboseLogger(a.Stderr, a.Verbose)

	// Build FormatConfig for command handlers
	a.FormatCfg = FormatConfig{
		Format:  a.OutputFormat,
		Quiet:   a.Quiet,
		Verbose: a.Verbose,
	}

	// Resolve formatter once based on format
	a.Formatter = resolveFormatter(a.OutputFormat)
	a.vlog.Log("format resolved to %s", formatName(a.OutputFormat))

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
	case "ready":
		if err := a.runReady(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "blocked":
		if err := a.runBlocked(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	case "stats":
		if err := a.runStats(subArgs); err != nil {
			a.writeError(err)
			return 1
		}
		return 0
	default:
		a.writeError(fmt.Errorf("Unknown command '%s'. Run 'tick help' for usage.", subcommand))
		return 1
	}
}

// parseGlobalFlags extracts global flags and returns the remaining arguments.
// It tracks format flag usage and uses ResolveFormat for conflict detection.
func (a *App) parseGlobalFlags(args []string) ([]string, error) {
	var remaining []string
	var toonFlag, prettyFlag, jsonFlag bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--quiet", "-q":
			a.Quiet = true
		case "--verbose", "-v":
			a.Verbose = true
		case "--toon":
			toonFlag = true
		case "--pretty":
			prettyFlag = true
		case "--json":
			jsonFlag = true
		default:
			// Not a global flag; everything from here on is subcommand + subcommand args
			remaining = append(remaining, args[i:]...)
			i = len(args) // exit the loop
		}
	}

	// Resolve format from flags and TTY detection
	format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, a.IsTTY)
	if err != nil {
		return nil, err
	}
	a.OutputFormat = format

	return remaining, nil
}

// openStore creates a Store from the given tick directory and wires up verbose logging.
func (a *App) openStore(tickDir string) (*store.Store, error) {
	s, err := store.NewStore(tickDir)
	if err != nil {
		return nil, err
	}
	if a.vlog != nil {
		s.LogFunc = a.vlog.Log
	}
	return s, nil
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
