// Package cli implements the tick command-line interface.
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// App is the tick CLI application.
type App struct {
	stdout io.Writer
	stderr io.Writer
	opts   GlobalOpts
}

// GlobalOpts holds parsed global flags.
type GlobalOpts struct {
	Quiet   bool
	Verbose bool
	Toon    bool
	Pretty  bool
	JSON    bool
}

// NewApp creates a new CLI application with the given output writers.
func NewApp(stdout, stderr io.Writer) *App {
	return &App{
		stdout: stdout,
		stderr: stderr,
	}
}

// Run parses arguments and dispatches to the appropriate subcommand.
// workDir is the working directory for the command.
// Returns the exit code (0 for success, 1 for error).
func (a *App) Run(args []string, workDir string) int {
	// Parse global flags and extract subcommand
	remaining := args[1:] // skip program name
	subcmd, cmdArgs := a.parseGlobalFlags(remaining)

	if subcmd == "" {
		a.printUsage()
		return 0
	}

	var err error
	switch subcmd {
	case "init":
		err = a.cmdInit(workDir)
	default:
		fmt.Fprintf(a.stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcmd)
		return 1
	}

	_ = cmdArgs // will be used by future commands

	if err != nil {
		fmt.Fprintf(a.stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}

func (a *App) parseGlobalFlags(args []string) (subcmd string, remaining []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--quiet", "-q":
			a.opts.Quiet = true
		case "--verbose", "-v":
			a.opts.Verbose = true
		case "--toon":
			a.opts.Toon = true
		case "--pretty":
			a.opts.Pretty = true
		case "--json":
			a.opts.JSON = true
		default:
			if strings.HasPrefix(arg, "-") {
				// Unknown flag — pass through to subcommand
				remaining = append(remaining, args[i:]...)
				return subcmd, remaining
			}
			if subcmd == "" {
				subcmd = arg
			} else {
				remaining = append(remaining, arg)
			}
		}
	}
	return subcmd, remaining
}

func (a *App) printUsage() {
	fmt.Fprintln(a.stdout, "Usage: tick <command> [options]")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Commands:")
	fmt.Fprintln(a.stdout, "  init      Initialize tick in current directory")
	fmt.Fprintln(a.stdout, "  create    Create a new task")
	fmt.Fprintln(a.stdout, "  list      List tasks")
	fmt.Fprintln(a.stdout, "  show      Show task details")
	fmt.Fprintln(a.stdout, "  start     Start a task")
	fmt.Fprintln(a.stdout, "  done      Complete a task")
	fmt.Fprintln(a.stdout, "  cancel    Cancel a task")
	fmt.Fprintln(a.stdout, "  reopen    Reopen a closed task")
	fmt.Fprintln(a.stdout, "  update    Update task fields")
	fmt.Fprintln(a.stdout, "  dep       Manage dependencies")
	fmt.Fprintln(a.stdout, "  ready     Show ready tasks")
	fmt.Fprintln(a.stdout, "  blocked   Show blocked tasks")
	fmt.Fprintln(a.stdout, "  stats     Show task statistics")
	fmt.Fprintln(a.stdout, "  rebuild   Force rebuild SQLite cache")
}

func (a *App) cmdInit(workDir string) error {
	tickDir := filepath.Join(workDir, ".tick")

	// Check if already initialized
	if _, err := os.Stat(tickDir); err == nil {
		return fmt.Errorf("Tick already initialized in this directory")
	}

	// Create .tick directory
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		return fmt.Errorf("Could not create .tick/ directory: %w", err)
	}

	// Create empty tasks.jsonl
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
		// Clean up .tick dir on failure
		os.RemoveAll(tickDir)
		return fmt.Errorf("Could not create tasks.jsonl: %w", err)
	}

	if !a.opts.Quiet {
		absDir, _ := filepath.Abs(tickDir)
		fmt.Fprintf(a.stdout, "Initialized tick in %s/\n", absDir)
	}

	return nil
}

// FindTickDir walks up from startDir looking for a .tick directory.
// Returns the path to the .tick directory or an error if not found.
func FindTickDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		tickDir := filepath.Join(dir, ".tick")
		if info, err := os.Stat(tickDir); err == nil && info.IsDir() {
			return tickDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not a tick project (no .tick directory found)")
}
