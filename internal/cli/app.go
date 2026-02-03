package cli

import (
	"fmt"
	"io"
	"os"
)

// OutputFormat represents the output format for CLI responses.
type OutputFormat string

const (
	// FormatTOON is the TOON output format (default for non-TTY/agents).
	FormatTOON OutputFormat = "toon"
	// FormatPretty is the human-readable output format (default for TTY).
	FormatPretty OutputFormat = "pretty"
	// FormatJSON is the JSON output format.
	FormatJSON OutputFormat = "json"
)

// Config holds the parsed global flags and configuration.
type Config struct {
	Quiet        bool
	Verbose      bool
	OutputFormat OutputFormat
}

// App is the CLI application.
type App struct {
	config  Config
	workDir string
	stdout  io.Writer
	stderr  io.Writer
}

// NewApp creates a new App with default settings.
func NewApp() *App {
	return &App{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// Run parses args and dispatches the subcommand.
// args[0] is the program name (e.g., "tick").
func (a *App) Run(args []string) error {
	// Parse global flags and extract subcommand + remaining args
	subcmd, cmdArgs, err := a.parseGlobalFlags(args[1:])
	if err != nil {
		return err
	}

	// Apply TTY detection for default output format (only if no format flag was set)
	if a.config.OutputFormat == "" {
		if detectTTY() {
			a.config.OutputFormat = FormatPretty
		} else {
			a.config.OutputFormat = FormatTOON
		}
	}

	// Resolve working directory
	if a.workDir == "" {
		a.workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine working directory: %w", err)
		}
	}

	// Dispatch subcommand
	switch subcmd {
	case "":
		return a.printUsage()
	case "init":
		return a.runInit()
	case "create":
		return a.runCreate(cmdArgs)
	case "list":
		return a.runList()
	case "show":
		return a.runShow(cmdArgs)
	default:
		return fmt.Errorf("Unknown command '%s'. Run 'tick help' for usage.", subcmd)
	}
}

// parseGlobalFlags parses global flags from args and returns the subcommand name
// and remaining arguments after the subcommand.
func (a *App) parseGlobalFlags(args []string) (string, []string, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--quiet", "-q":
			a.config.Quiet = true
		case "--verbose", "-v":
			a.config.Verbose = true
		case "--toon":
			a.config.OutputFormat = FormatTOON
		case "--pretty":
			a.config.OutputFormat = FormatPretty
		case "--json":
			a.config.OutputFormat = FormatJSON
		default:
			// First non-flag argument is the subcommand; rest are command args
			return arg, args[i+1:], nil
		}
	}
	return "", nil, nil
}

// printUsage prints basic usage information.
func (a *App) printUsage() error {
	usage := `Usage: tick <command> [options]

Commands:
  init       Initialize a new tick project
  create     Create a new task
  list       List all tasks
  show       Show detailed task information

Global flags:
  -q, --quiet     Suppress non-essential output
  -v, --verbose   More detail for debugging
  --toon          Force TOON output format
  --pretty        Force human-readable output format
  --json          Force JSON output format
`
	fmt.Fprint(a.stdout, usage)
	return nil
}

// detectTTY checks whether stdout is a terminal.
func detectTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
