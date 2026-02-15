package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/leeovery/tick/internal/migrate"
	"github.com/leeovery/tick/internal/migrate/beads"
)

// newMigrateProvider resolves a provider by name, using baseDir for file-based providers.
// Returns an error if the name is not recognized.
func newMigrateProvider(name string, baseDir string) (migrate.Provider, error) {
	switch name {
	case "beads":
		return beads.NewBeadsProvider(baseDir), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

// migrateFlags holds parsed migrate subcommand flags.
type migrateFlags struct {
	from   string
	dryRun bool
}

// parseMigrateArgs extracts flag values from migrate subcommand args.
func parseMigrateArgs(args []string) (migrateFlags, error) {
	var flags migrateFlags
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--from":
			i++
			if i >= len(args) {
				return flags, fmt.Errorf("--from requires a value")
			}
			flags.from = args[i]
		case strings.HasPrefix(args[i], "--from="):
			flags.from = strings.TrimPrefix(args[i], "--from=")
		case args[i] == "--dry-run":
			flags.dryRun = true
		}
	}
	if flags.from == "" {
		return flags, fmt.Errorf("--from flag is required. Usage: tick migrate --from <provider>")
	}
	return flags, nil
}

// handleMigrate implements the migrate subcommand. It bypasses the format/formatter
// machinery and outputs migration progress directly.
func (a *App) handleMigrate(subArgs []string) int {
	mf, err := parseMigrateArgs(subArgs)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	if strings.TrimSpace(mf.from) == "" {
		fmt.Fprintf(a.Stderr, "Error: --from flag is required. Usage: tick migrate --from <provider>\n")
		return 1
	}

	dir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: could not determine working directory: %s\n", err)
		return 1
	}

	provider, err := newMigrateProvider(mf.from, dir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	if err := RunMigrate(dir, provider, mf.dryRun, a.Stdout); err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	return 0
}

// RunMigrate executes the migration pipeline: opens the store, creates the engine
// with a TaskCreator (StoreTaskCreator for real runs, DryRunTaskCreator for dry-run),
// runs the provider, and outputs results via the presenter.
func RunMigrate(dir string, provider migrate.Provider, dryRun bool, stdout io.Writer) error {
	var creator migrate.TaskCreator
	if dryRun {
		creator = &migrate.DryRunTaskCreator{}
	} else {
		store, err := openStore(dir, FormatConfig{})
		if err != nil {
			return err
		}
		defer store.Close()
		creator = migrate.NewStoreTaskCreator(store)
	}

	engine := migrate.NewEngine(creator)

	// Print header.
	migrate.WriteHeader(stdout, provider.Name(), dryRun)

	// Run migration.
	results, runErr := engine.Run(provider)

	// Print results and summary regardless of error (partial results on failure).
	for _, r := range results {
		migrate.WriteResult(stdout, r)
	}
	migrate.WriteSummary(stdout, results)

	return runErr
}
