package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

// App is the top-level CLI application, testable via injected writers and working directory.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	// Stdin is the standard input reader. Injected for testability (confirmation prompts).
	Stdin io.Reader
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
		printTopLevelHelp(a.Stdout)
		return 0
	}

	// Help bypasses format/formatter machinery — always plain text.
	if subcmd == "help" {
		return a.handleHelp(subArgs)
	}
	if flags.help {
		return a.handleHelp([]string{subcmd})
	}

	// Doctor and migrate bypass format/formatter machinery — always human-readable text.
	if subcmd == "doctor" {
		return a.handleDoctor()
	}
	if subcmd == "migrate" {
		return a.handleMigrate(subArgs)
	}

	// Resolve format once in dispatcher.
	fc, err := NewFormatConfig(flags, a.IsTTY)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Create verbose logger when --verbose is set.
	if fc.Verbose {
		fc.Logger = NewVerboseLogger(a.Stderr)
	}

	fmtr := NewFormatter(fc.Format)

	// Log format resolution if verbose.
	if fc.Logger != nil {
		var formatName string
		switch fc.Format {
		case FormatToon:
			formatName = "toon"
		case FormatPretty:
			formatName = "pretty"
		case FormatJSON:
			formatName = "json"
		}
		fc.Logger.Log("format resolved: " + formatName)
	}

	switch subcmd {
	case "init":
		err = a.handleInit(fc, fmtr, subArgs)
	case "create":
		err = a.handleCreate(fc, fmtr, subArgs)
	case "list":
		err = a.handleList(fc, fmtr, subArgs)
	case "show":
		err = a.handleShow(fc, fmtr, subArgs)
	case "update":
		err = a.handleUpdate(fc, fmtr, subArgs)
	case "start", "done", "cancel", "reopen":
		err = a.handleTransition(subcmd, fc, fmtr, subArgs)
	case "ready":
		err = a.handleReady(fc, fmtr, subArgs)
	case "blocked":
		err = a.handleBlocked(fc, fmtr, subArgs)
	case "dep":
		err = a.handleDep(fc, fmtr, subArgs)
	case "remove":
		err = a.handleRemove(fc, fmtr, subArgs)
	case "stats":
		err = a.handleStats(fc, fmtr)
	case "rebuild":
		err = a.handleRebuild(fc, fmtr)
	default:
		fmt.Fprintf(a.Stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcmd)
		return 1
	}

	if err != nil {
		if errors.Is(err, errAborted) {
			return 1
		}
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}

// handleInit implements the init subcommand.
func (a *App) handleInit(fc FormatConfig, fmtr Formatter, _ []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunInit(dir, fc, fmtr, a.Stdout)
}

// handleCreate implements the create subcommand.
func (a *App) handleCreate(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunCreate(dir, fc, fmtr, subArgs, a.Stdout)
}

// handleList implements the list subcommand.
func (a *App) handleList(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	filter, err := parseListFlags(subArgs)
	if err != nil {
		return err
	}
	return RunList(dir, fc, fmtr, filter, a.Stdout)
}

// handleShow implements the show subcommand.
func (a *App) handleShow(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunShow(dir, fc, fmtr, subArgs, a.Stdout)
}

// handleUpdate implements the update subcommand.
func (a *App) handleUpdate(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunUpdate(dir, fc, fmtr, subArgs, a.Stdout)
}

// handleReady implements the ready subcommand (alias for list --ready).
func (a *App) handleReady(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	filter, err := parseListFlags(append([]string{"--ready"}, subArgs...))
	if err != nil {
		return err
	}
	return RunList(dir, fc, fmtr, filter, a.Stdout)
}

// handleBlocked implements the blocked subcommand (alias for list --blocked).
func (a *App) handleBlocked(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	filter, err := parseListFlags(append([]string{"--blocked"}, subArgs...))
	if err != nil {
		return err
	}
	return RunList(dir, fc, fmtr, filter, a.Stdout)
}

// handleStats implements the stats subcommand.
func (a *App) handleStats(fc FormatConfig, fmtr Formatter) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunStats(dir, fc, fmtr, a.Stdout)
}

// handleRebuild implements the rebuild subcommand.
func (a *App) handleRebuild(fc FormatConfig, fmtr Formatter) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunRebuild(dir, fc, fmtr, a.Stdout)
}

// handleRemove implements the remove subcommand.
// It parses args, computes the blast radius for non-force removals,
// runs the interactive confirmation prompt using App's Stdin/Stderr,
// then delegates to RunRemove for the actual mutation and formatting.
func (a *App) handleRemove(fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}

	ids, force := parseRemoveArgs(subArgs)

	if len(ids) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick remove <id> [<id>...]")
	}

	if !force {
		store, err := openStore(dir, fc)
		if err != nil {
			return err
		}
		var br blastRadius
		err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
			var execErr error
			tasks, br, _, execErr = executeRemoval(tasks, ids, true)
			return tasks, execErr
		})
		store.Close()
		if err != nil {
			return err
		}
		if err := confirmRemovalWithCascade(br, a.Stdin, a.Stderr); err != nil {
			return err
		}
	}

	return RunRemove(dir, fc, fmtr, subArgs, a.Stdout)
}

// handleTransition implements the start/done/cancel/reopen subcommands.
func (a *App) handleTransition(command string, fc FormatConfig, fmtr Formatter, subArgs []string) error {
	dir, err := a.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	return RunTransition(dir, command, fc, fmtr, subArgs, a.Stdout)
}

// handleHelp implements the help command and --help/-h flag.
func (a *App) handleHelp(args []string) int {
	if len(args) == 0 {
		printTopLevelHelp(a.Stdout)
		return 0
	}
	if args[0] == "--all" {
		printAllHelp(a.Stdout)
		return 0
	}
	cmd := findCommand(args[0])
	if cmd == nil {
		fmt.Fprintf(a.Stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", args[0])
		return 1
	}
	printCommandHelp(a.Stdout, cmd)
	return 0
}

// globalFlags holds parsed global CLI flags.
type globalFlags struct {
	quiet   bool
	verbose bool
	toon    bool
	pretty  bool
	json    bool
	help    bool
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
	case "--help", "-h":
		flags.help = true
	default:
		return false
	}
	return true
}
