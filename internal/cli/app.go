package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// OutputFormat represents the selected output format for CLI responses.
type OutputFormat int

const (
	// FormatHuman is the human-readable table output for terminals.
	FormatHuman OutputFormat = iota
	// FormatTOON is the token-oriented format for agent consumption.
	FormatTOON
	// FormatJSON is the standard JSON output format.
	FormatJSON
)

// App is the top-level CLI application, testable via injected writers and working directory.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	// Getwd returns the current working directory. Injected for testability.
	Getwd func() (string, error)
	// IsTTY indicates whether stdout is a terminal. Set during flag parsing.
	IsTTY bool
}

// Run parses args, dispatches subcommands, and returns an exit code (0 = success, 1 = error).
func (a *App) Run(args []string) int {
	// Parse global flags and extract subcommand.
	flags, subcmd, subArgs := parseArgs(args[1:])

	if subcmd == "" {
		a.printUsage()
		return 0
	}

	var err error
	switch subcmd {
	case "init":
		err = a.handleInit(flags, subArgs)
	case "create":
		err = a.handleCreate(flags, subArgs)
	case "list":
		err = a.handleList(flags)
	case "show":
		err = a.handleShow(flags, subArgs)
	case "start", "done", "cancel", "reopen":
		err = a.handleTransition(subcmd, flags, subArgs)
	default:
		fmt.Fprintf(a.Stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcmd)
		return 1
	}

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}

// handleInit implements the init subcommand.
func (a *App) handleInit(flags globalFlags, _ []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunInit(dir, flags.quiet, a.Stdout)
}

// handleCreate implements the create subcommand.
func (a *App) handleCreate(flags globalFlags, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunCreate(dir, flags.quiet, subArgs, a.Stdout)
}

// handleList implements the list subcommand.
func (a *App) handleList(flags globalFlags) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunList(dir, flags.quiet, a.Stdout)
}

// handleShow implements the show subcommand.
func (a *App) handleShow(flags globalFlags, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunShow(dir, flags.quiet, subArgs, a.Stdout)
}

// handleTransition implements the start/done/cancel/reopen subcommands.
func (a *App) handleTransition(command string, flags globalFlags, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunTransition(dir, command, flags.quiet, subArgs, a.Stdout)
}

// printUsage prints basic usage information.
func (a *App) printUsage() {
	fmt.Fprintln(a.Stdout, "Usage: tick <command> [options]")
	fmt.Fprintln(a.Stdout, "")
	fmt.Fprintln(a.Stdout, "Commands:")
	fmt.Fprintln(a.Stdout, "  init    Initialize a new tick project")
}

// globalFlags holds parsed global CLI flags.
type globalFlags struct {
	quiet   bool
	verbose bool
	toon    bool
	pretty  bool
	json    bool
}

// parseArgs separates global flags from the subcommand and its arguments.
// Global flags are extracted from all positions (before and after the subcommand),
// following the pattern of tools like git where "git commit --verbose" works the
// same as "git --verbose commit". Returns the parsed global flags, the subcommand
// name, and remaining subcommand-specific args (non-global arguments only).
func parseArgs(args []string) (globalFlags, string, []string) {
	var flags globalFlags
	var subcmd string
	var rest []string

	foundCmd := false
	for _, arg := range args {
		if applyGlobalFlag(&flags, arg) {
			continue
		}
		if !foundCmd {
			if strings.HasPrefix(arg, "-") {
				// Unknown flag before subcommand — skip
				continue
			}
			subcmd = arg
			foundCmd = true
		} else {
			rest = append(rest, arg)
		}
	}
	return flags, subcmd, rest
}

// applyGlobalFlag checks if arg is a known global flag and applies it to flags.
// Returns true if the arg was a global flag, false otherwise.
func applyGlobalFlag(flags *globalFlags, arg string) bool {
	switch arg {
	case "--quiet", "-q":
		flags.quiet = true
	case "--verbose", "-v":
		flags.verbose = true
	case "--toon":
		flags.toon = true
	case "--pretty":
		flags.pretty = true
	case "--json":
		flags.json = true
	default:
		return false
	}
	return true
}

// IsTerminal checks if the given *os.File is connected to a terminal (TTY).
func IsTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

// ResolveFormat determines the output format based on flags and TTY detection.
func ResolveFormat(flags globalFlags, isTTY bool) OutputFormat {
	if flags.toon {
		return FormatTOON
	}
	if flags.pretty {
		return FormatHuman
	}
	if flags.json {
		return FormatJSON
	}
	if isTTY {
		return FormatHuman
	}
	return FormatTOON
}
