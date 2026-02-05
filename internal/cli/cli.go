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
	Stdout       io.Writer
	Stderr       io.Writer
	Cwd          string
	flags        GlobalFlags
	formatConfig FormatConfig
}

// GlobalFlags holds parsed global command-line flags.
type GlobalFlags struct {
	Quiet        bool
	Verbose      bool
	OutputFormat string // "toon", "pretty", "json", or "" for auto-detect
	// Individual format flags for conflict detection
	ToonFlag   bool
	PrettyFlag bool
	JSONFlag   bool
}

// Run executes the CLI with the given arguments.
// Returns exit code 0 for success, 1 for errors.
func (a *App) Run(args []string) int {
	// Parse global flags
	a.flags, args = ParseGlobalFlags(args)

	// Resolve format config early to catch conflicting flags before dispatch
	formatConfig, err := NewFormatConfig(
		a.flags.ToonFlag,
		a.flags.PrettyFlag,
		a.flags.JSONFlag,
		a.flags.Quiet,
		a.flags.Verbose,
		a.Stdout,
	)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	a.formatConfig = formatConfig
	a.WriteVerbose("format resolved to %s", formatConfig.Format)

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
	case "update":
		return a.runUpdate(args)
	case "list":
		return a.runList(args)
	case "show":
		return a.runShow(args)
	case "start", "done", "cancel", "reopen":
		return a.runTransition(subcommand, args)
	case "dep":
		return a.runDep(args)
	case "ready":
		return a.runReady()
	case "blocked":
		return a.runBlocked()
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
	if !a.formatConfig.Quiet {
		formatter := a.formatConfig.Formatter()
		msg := fmt.Sprintf("Initialized tick in %s/", tickDir)
		fmt.Fprint(a.Stdout, formatter.FormatMessage(msg))
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
	fmt.Fprintln(a.Stdout, "  update  Update task fields")
	fmt.Fprintln(a.Stdout, "  list    List all tasks")
	fmt.Fprintln(a.Stdout, "  show    Show task details")
	fmt.Fprintln(a.Stdout, "  start   Mark task as in-progress")
	fmt.Fprintln(a.Stdout, "  done    Mark task as completed")
	fmt.Fprintln(a.Stdout, "  cancel  Mark task as cancelled")
	fmt.Fprintln(a.Stdout, "  reopen  Reopen a closed task")
	fmt.Fprintln(a.Stdout, "  dep     Manage dependencies (add/rm)")
	fmt.Fprintln(a.Stdout, "  ready   Show tasks that are ready to work on")
	fmt.Fprintln(a.Stdout, "  blocked Show tasks that are blocked")
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
	if DetectTTY(a.Stdout) {
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
			flags.ToonFlag = true
			flags.OutputFormat = "toon"
		case "--pretty":
			flags.PrettyFlag = true
			flags.OutputFormat = "pretty"
		case "--json":
			flags.JSONFlag = true
			flags.OutputFormat = "json"
		default:
			remaining = append(remaining, arg)
		}
	}

	return flags, remaining
}

// WriteVerbose writes a verbose message to stderr if verbose mode is enabled.
// Verbose output is always written to stderr to avoid contaminating stdout.
// All lines are prefixed with "verbose:" for grep-ability.
func (a *App) WriteVerbose(format string, args ...interface{}) {
	if !a.formatConfig.Verbose {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(a.Stderr, "verbose: %s\n", msg)
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
