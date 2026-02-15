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

// parseMigrateArgs extracts the --from flag value from migrate subcommand args.
func parseMigrateArgs(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--from" {
			i++
			if i >= len(args) {
				return "", fmt.Errorf("--from requires a value")
			}
			return args[i], nil
		}
		if strings.HasPrefix(args[i], "--from=") {
			return strings.TrimPrefix(args[i], "--from="), nil
		}
	}
	return "", fmt.Errorf("--from flag is required. Usage: tick migrate --from <provider>")
}

// handleMigrate implements the migrate subcommand. It bypasses the format/formatter
// machinery and outputs migration progress directly.
func (a *App) handleMigrate(subArgs []string) int {
	fromValue, err := parseMigrateArgs(subArgs)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	if strings.TrimSpace(fromValue) == "" {
		fmt.Fprintf(a.Stderr, "Error: --from flag is required. Usage: tick migrate --from <provider>\n")
		return 1
	}

	dir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: could not determine working directory: %s\n", err)
		return 1
	}

	provider, err := newMigrateProvider(fromValue, dir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	if err := RunMigrate(dir, provider, a.Stdout); err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	return 0
}

// RunMigrate executes the migration pipeline: opens the store, creates the engine
// with a StoreTaskCreator, runs the provider, and outputs results via the presenter.
func RunMigrate(dir string, provider migrate.Provider, stdout io.Writer) error {
	store, err := openStore(dir, FormatConfig{})
	if err != nil {
		return err
	}
	defer store.Close()

	creator := migrate.NewStoreTaskCreator(store)
	engine := migrate.NewEngine(creator)

	// Print header.
	migrate.WriteHeader(stdout, provider.Name())

	// Run migration.
	results, runErr := engine.Run(provider)

	// Print results and summary regardless of error (partial results on failure).
	for _, r := range results {
		migrate.WriteResult(stdout, r)
	}
	migrate.WriteSummary(stdout, results)

	return runErr
}
