// Package cli implements the tick command-line interface including subcommand
// dispatch, global flag parsing, TTY detection, and error handling.
package cli

import (
	"fmt"
	"io"
	"strings"
)

// OutputFormat represents the output format for CLI commands.
type OutputFormat int

const (
	// FormatToon is the token-oriented output format for AI agents.
	FormatToon OutputFormat = iota
	// FormatPretty is the human-readable output format for terminals.
	FormatPretty
	// FormatJSON is the JSON output format for compatibility.
	FormatJSON
)

// Context holds the parsed CLI state for a single invocation.
type Context struct {
	WorkDir string
	Stdout  io.Writer
	Stderr  io.Writer
	Quiet   bool
	Verbose bool
	Format  OutputFormat
	Fmt     Formatter // resolved once in dispatcher from Format
	Args    []string  // remaining args after global flags and subcommand
}

// FormatCfg returns a FormatConfig derived from this Context's fields.
func (c *Context) FormatCfg() FormatConfig {
	return FormatConfig{
		Format:  c.Format,
		Quiet:   c.Quiet,
		Verbose: c.Verbose,
	}
}

// Run executes the tick CLI with the given arguments, working directory,
// output writers, and TTY detection flag. It returns an exit code (0 for
// success, 1 for errors).
func Run(args []string, workDir string, stdout, stderr io.Writer, isTTY bool) int {
	ctx, subcmd, err := parseArgs(args, workDir, stdout, stderr, isTTY)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	if subcmd == "" {
		printUsage(ctx.Stdout)
		return 0
	}

	handler, ok := commands[subcmd]
	if !ok {
		fmt.Fprintf(stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcmd)
		return 1
	}

	if err := handler(ctx); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	return 0
}

// commands maps subcommand names to their handler functions.
var commands = map[string]func(*Context) error{
	"init":    runInit,
	"create":  runCreate,
	"update":  runUpdate,
	"list":    runList,
	"show":    runShow,
	"start":   runTransition("start"),
	"done":    runTransition("done"),
	"cancel":  runTransition("cancel"),
	"reopen":  runTransition("reopen"),
	"dep":     runDep,
	"ready":   runReady,
	"blocked": runBlocked,
}

// parseArgs parses global flags from args and returns the context, subcommand
// name, and any error. Global flags are extracted first; the first non-flag
// argument is the subcommand.
func parseArgs(args []string, workDir string, stdout, stderr io.Writer, isTTY bool) (*Context, string, error) {
	ctx := &Context{
		WorkDir: workDir,
		Stdout:  stdout,
		Stderr:  stderr,
	}

	// Skip program name
	remaining := args[1:]

	var subcmd string
	var cmdArgs []string
	foundCmd := false
	var toonFlag, prettyFlag, jsonFlag bool

	for _, arg := range remaining {
		if foundCmd {
			cmdArgs = append(cmdArgs, arg)
			continue
		}

		switch {
		case arg == "--quiet" || arg == "-q":
			ctx.Quiet = true
		case arg == "--verbose" || arg == "-v":
			ctx.Verbose = true
		case arg == "--toon":
			toonFlag = true
		case arg == "--pretty":
			prettyFlag = true
		case arg == "--json":
			jsonFlag = true
		case strings.HasPrefix(arg, "-"):
			return nil, "", fmt.Errorf("unknown flag '%s'", arg)
		default:
			subcmd = arg
			foundCmd = true
		}
	}

	// Resolve format from flags and TTY status.
	format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)
	if err != nil {
		return nil, "", err
	}
	ctx.Format = format
	ctx.Fmt = newFormatter(format)

	ctx.Args = cmdArgs
	return ctx, subcmd, nil
}

// printUsage writes basic usage information to the given writer.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "tick - A minimal, deterministic task tracker for AI coding agents")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage: tick <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init      Initialize .tick/ directory in current project")
	fmt.Fprintln(w, "  create    Create a new task")
	fmt.Fprintln(w, "  update    Update task fields")
	fmt.Fprintln(w, "  start     Mark task as in-progress")
	fmt.Fprintln(w, "  done      Mark task as completed")
	fmt.Fprintln(w, "  cancel    Mark task as cancelled")
	fmt.Fprintln(w, "  reopen    Reopen a closed task")
	fmt.Fprintln(w, "  list      List tasks with optional filters")
	fmt.Fprintln(w, "  show      Show detailed task information")
	fmt.Fprintln(w, "  ready     Show workable tasks")
	fmt.Fprintln(w, "  blocked   Show blocked tasks")
	fmt.Fprintln(w, "  dep       Manage task dependencies")
	fmt.Fprintln(w, "  stats     Show task statistics")
	fmt.Fprintln(w, "  doctor    Run diagnostics and validation")
	fmt.Fprintln(w, "  rebuild   Force rebuild SQLite cache")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  -q, --quiet    Suppress non-essential output")
	fmt.Fprintln(w, "  -v, --verbose  More detail (useful for debugging)")
	fmt.Fprintln(w, "  --toon         Force TOON output format")
	fmt.Fprintln(w, "  --pretty       Force human-readable output format")
	fmt.Fprintln(w, "  --json         Force JSON output format")
}
