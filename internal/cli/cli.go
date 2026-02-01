// Package cli implements the tick command-line interface.
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/leeovery/tick/internal/storage"
)

// App is the tick CLI application.
type App struct {
	stdout  io.Writer
	stderr  io.Writer
	opts    GlobalOpts
	isTTY   bool
	fmtr    Formatter
	verbose *VerboseLogger
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
// TTY detection is based on whether stdout is an *os.File with a character device.
func NewApp(stdout, stderr io.Writer) *App {
	isTTY := false
	if f, ok := stdout.(*os.File); ok {
		isTTY = DetectTTY(f)
	}
	return &App{
		stdout: stdout,
		stderr: stderr,
		isTTY:  isTTY,
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

	// Initialize verbose logger.
	a.verbose = NewVerboseLogger(a.stderr, a.opts.Verbose)

	// Resolve output format once before dispatch.
	format, err := ResolveFormat(a.opts.Toon, a.opts.Pretty, a.opts.JSON, a.isTTY)
	if err != nil {
		fmt.Fprintf(a.stderr, "Error: %s\n", err)
		return 1
	}
	a.fmtr = newFormatter(format)

	formatName := [...]string{"toon", "pretty", "json"}[format]
	if a.opts.Toon || a.opts.Pretty || a.opts.JSON {
		a.verbose.Log("format=%s (flag override)", formatName)
	} else {
		a.verbose.Log("format=%s (auto-detected, tty=%v)", formatName, a.isTTY)
	}
	switch subcmd {
	case "init":
		err = a.cmdInit(workDir)
	case "create":
		err = a.cmdCreate(workDir, cmdArgs)
	case "list":
		err = a.cmdList(workDir, cmdArgs)
	case "show":
		err = a.cmdShow(workDir, cmdArgs)
	case "start", "done", "cancel", "reopen":
		err = a.cmdTransition(workDir, cmdArgs, subcmd)
	case "update":
		err = a.cmdUpdate(workDir, cmdArgs)
	case "dep":
		err = a.cmdDep(workDir, cmdArgs)
	case "ready":
		err = a.cmdReady(workDir, cmdArgs)
	case "blocked":
		err = a.cmdBlocked(workDir, cmdArgs)
	case "stats":
		err = a.cmdStats(workDir)
	default:
		fmt.Fprintf(a.stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcmd)
		return 1
	}

	if err != nil {
		fmt.Fprintf(a.stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}

func (a *App) parseGlobalFlags(args []string) (subcmd string, remaining []string) {
	foundSubcmd := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Only parse global flags before the subcommand
		if !foundSubcmd {
			switch arg {
			case "--quiet", "-q":
				a.opts.Quiet = true
				continue
			case "--verbose", "-v":
				a.opts.Verbose = true
				continue
			case "--toon":
				a.opts.Toon = true
				continue
			case "--pretty":
				a.opts.Pretty = true
				continue
			case "--json":
				a.opts.JSON = true
				continue
			}
		}

		if !foundSubcmd && !strings.HasPrefix(arg, "-") {
			subcmd = arg
			foundSubcmd = true
		} else if foundSubcmd {
			remaining = append(remaining, arg)
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
		a.fmtr.FormatMessage(a.stdout, fmt.Sprintf("Initialized tick in %s/", absDir))
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

// openStore creates a Store for the given tick directory with verbose logging wired in.
func (a *App) openStore(tickDir string) (*storage.Store, error) {
	store, err := storage.NewStore(tickDir)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}
	if a.verbose != nil {
		store.SetLogger(a.verbose.Log)
	}
	return store, nil
}
