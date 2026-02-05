// Package cli provides the command-line interface for tick.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// App represents the CLI application with configurable I/O.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Cwd    string
	flags  GlobalFlags
}

// GlobalFlags holds parsed global command-line flags.
type GlobalFlags struct {
	Quiet        bool
	Verbose      bool
	OutputFormat string // "toon", "pretty", "json", or "" for auto-detect
}

// Run executes the CLI with the given arguments.
// Returns exit code 0 for success, 1 for errors.
func (a *App) Run(args []string) int {
	// Parse global flags
	a.flags, args = ParseGlobalFlags(args)

	// Get subcommand
	if len(args) < 2 {
		a.printUsage()
		return 0
	}

	subcommand := args[1]

	// Route to handler
	switch subcommand {
	case "init":
		return a.runInit()
	case "create":
		return a.runCreate(args)
	default:
		fmt.Fprintf(a.Stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcommand)
		return 1
	}
}

// runInit executes the init subcommand.
func (a *App) runInit() int {
	tickDir := filepath.Join(a.Cwd, ".tick")

	// Check if .tick/ already exists
	if _, err := os.Stat(tickDir); err == nil {
		fmt.Fprintf(a.Stderr, "Error: Tick already initialized in this directory\n")
		return 1
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(a.Stderr, "Error: Could not check .tick/ directory: %v\n", err)
		return 1
	}

	// Create .tick/ directory
	if err := os.Mkdir(tickDir, 0755); err != nil {
		fmt.Fprintf(a.Stderr, "Error: Could not create .tick/ directory: %v\n", err)
		return 1
	}

	// Create empty tasks.jsonl
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	f, err := os.OpenFile(jsonlPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: Could not create tasks.jsonl: %v\n", err)
		return 1
	}
	f.Close()

	// Print confirmation (unless --quiet)
	if !a.flags.Quiet {
		fmt.Fprintf(a.Stdout, "Initialized tick in %s/\n", tickDir)
	}

	return 0
}

// printUsage prints basic usage information.
func (a *App) printUsage() {
	fmt.Fprintln(a.Stdout, "Usage: tick <command> [options]")
	fmt.Fprintln(a.Stdout, "")
	fmt.Fprintln(a.Stdout, "Commands:")
	fmt.Fprintln(a.Stdout, "  init    Initialize tick in current directory")
	fmt.Fprintln(a.Stdout, "  create  Create a new task")
	fmt.Fprintln(a.Stdout, "")
	fmt.Fprintln(a.Stdout, "Global Options:")
	fmt.Fprintln(a.Stdout, "  -q, --quiet    Suppress non-essential output")
	fmt.Fprintln(a.Stdout, "  -v, --verbose  More detail for debugging")
	fmt.Fprintln(a.Stdout, "  --toon         Force TOON output format")
	fmt.Fprintln(a.Stdout, "  --pretty       Force human-readable output format")
	fmt.Fprintln(a.Stdout, "  --json         Force JSON output format")
}

// DefaultOutputFormat returns the default output format based on TTY detection.
func (a *App) DefaultOutputFormat() string {
	if a.flags.OutputFormat != "" {
		return a.flags.OutputFormat
	}
	if IsTTY(a.Stdout) {
		return "pretty"
	}
	return "toon"
}

// ParseGlobalFlags extracts global flags from args and returns the remaining args.
func ParseGlobalFlags(args []string) (GlobalFlags, []string) {
	var flags GlobalFlags
	var remaining []string

	for _, arg := range args {
		switch arg {
		case "-q", "--quiet":
			flags.Quiet = true
		case "-v", "--verbose":
			flags.Verbose = true
		case "--toon":
			flags.OutputFormat = "toon"
		case "--pretty":
			flags.OutputFormat = "pretty"
		case "--json":
			flags.OutputFormat = "json"
		default:
			remaining = append(remaining, arg)
		}
	}

	return flags, remaining
}

// IsTTY checks if the given writer is a terminal.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// DiscoverTickDir walks up from the given directory looking for .tick/.
// Returns the path to .tick/ or error if not found.
func DiscoverTickDir(startDir string) (string, error) {
	dir := startDir

	for {
		tickDir := filepath.Join(dir, ".tick")
		if info, err := os.Stat(tickDir); err == nil && info.IsDir() {
			return tickDir, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", errors.New("not a tick project (no .tick directory found)")
}
