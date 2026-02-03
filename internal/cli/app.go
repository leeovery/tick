package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/leeovery/tick/internal/storage"
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
	config    Config
	FormatCfg FormatConfig
	formatter Formatter
	verbose   *VerboseLogger
	workDir   string
	stdout    io.Writer
	stderr    io.Writer
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
	// Parse global flags (including format resolution via ResolveFormat)
	subcmd, cmdArgs, err := a.parseGlobalFlags(args[1:])
	if err != nil {
		return err
	}

	// Resolve working directory
	if a.workDir == "" {
		a.workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine working directory: %w", err)
		}
	}

	// Initialize verbose logger (always created; no-op when disabled)
	a.verbose = NewVerboseLogger(a.stderr, a.config.Verbose)

	// Resolve FormatConfig and store on App for handlers to use
	a.FormatCfg = a.formatConfig()
	a.verbose.Log(fmt.Sprintf("format resolved: %s", a.FormatCfg.Format))

	// Resolve formatter once based on FormatConfig
	a.formatter = newFormatter(a.FormatCfg.Format)

	// Dispatch subcommand
	switch subcmd {
	case "":
		return a.printUsage()
	case "init":
		return a.runInit()
	case "create":
		return a.runCreate(cmdArgs)
	case "update":
		return a.runUpdate(cmdArgs)
	case "list":
		return a.runList(cmdArgs)
	case "ready":
		return a.runList([]string{"--ready"})
	case "blocked":
		return a.runList([]string{"--blocked"})
	case "show":
		return a.runShow(cmdArgs)
	case "start", "done", "cancel", "reopen":
		return a.runTransition(subcmd, cmdArgs)
	case "dep":
		return a.runDep(cmdArgs)
	case "stats":
		return a.runStats()
	case "rebuild":
		return a.runRebuild()
	default:
		return fmt.Errorf("Unknown command '%s'. Run 'tick help' for usage.", subcmd)
	}
}

// parseGlobalFlags parses global flags from args and returns the subcommand name
// and remaining arguments after the subcommand.
// Format flags are tracked individually so ResolveFormat can detect conflicts.
func (a *App) parseGlobalFlags(args []string) (string, []string, error) {
	var toonFlag, prettyFlag, jsonFlag bool

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--quiet", "-q":
			a.config.Quiet = true
		case "--verbose", "-v":
			a.config.Verbose = true
		case "--toon":
			toonFlag = true
		case "--pretty":
			prettyFlag = true
		case "--json":
			jsonFlag = true
		default:
			// First non-flag argument is the subcommand; rest are command args
			// Resolve format before returning so conflicts are detected early
			format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, DetectTTY())
			if err != nil {
				return "", nil, err
			}
			a.config.OutputFormat = format
			return arg, args[i+1:], nil
		}
	}

	// No subcommand found; still resolve format
	format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, DetectTTY())
	if err != nil {
		return "", nil, err
	}
	a.config.OutputFormat = format
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

// unwrapMutationError extracts the inner error from store.Mutate's wrapping.
// store.Mutate wraps callback errors with "mutation failed: <inner>", and this
// function returns the inner error for cleaner user-facing messages.
// If the error has no wrapped inner error, it is returned as-is.
func unwrapMutationError(err error) error {
	if inner := errors.Unwrap(err); inner != nil {
		return inner
	}
	return err
}

// formatConfig returns the FormatConfig derived from the App's parsed config.
// This is the entry point for passing output configuration to command handlers.
func (a *App) formatConfig() FormatConfig {
	return FormatConfig{
		Format:  a.config.OutputFormat,
		Quiet:   a.config.Quiet,
		Verbose: a.config.Verbose,
	}
}

// logVerbose writes a message to stderr only when verbose mode is enabled.
// Verbose output always goes to stderr so it never contaminates stdout.
// Delegates to the VerboseLogger which handles the "verbose: " prefix.
// If the VerboseLogger has not been initialized yet (pre-Run), falls back
// to direct stderr write when config.Verbose is true.
func (a *App) logVerbose(msg string) {
	if a.verbose != nil {
		a.verbose.Log(msg)
		return
	}
	if a.config.Verbose {
		fmt.Fprintf(a.stderr, "verbose: %s\n", msg)
	}
}

// newStore creates a Store for the given .tick/ directory and wires verbose logging.
func (a *App) newStore(tickDir string) (*storage.Store, error) {
	store, err := storage.NewStore(tickDir)
	if err != nil {
		return nil, err
	}
	if a.verbose != nil {
		store.SetLogger(a.verbose)
	}
	return store, nil
}

// newFormatter returns the concrete Formatter for the given output format.
func newFormatter(format OutputFormat) Formatter {
	switch format {
	case FormatPretty:
		return &PrettyFormatter{}
	case FormatJSON:
		return &JSONFormatter{}
	default:
		return &ToonFormatter{}
	}
}
