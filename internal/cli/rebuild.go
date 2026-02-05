package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// runRebuild executes the rebuild subcommand.
// Forces a complete SQLite cache rebuild from JSONL, bypassing the freshness check.
func (a *App) runRebuild() int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Open store
	a.WriteVerbose("store open %s", tickDir)
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	// Perform rebuild
	a.WriteVerbose("lock acquire exclusive")
	a.WriteVerbose("delete existing cache.db")
	a.WriteVerbose("read JSONL tasks")
	count, err := store.Rebuild()
	a.WriteVerbose("insert %d tasks into cache", count)
	a.WriteVerbose("hash updated in metadata table")
	a.WriteVerbose("lock release")

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output confirmation
	if !a.formatConfig.Quiet {
		formatter := a.formatConfig.Formatter()
		msg := fmt.Sprintf("Rebuilt cache: %d tasks", count)
		fmt.Fprint(a.Stdout, formatter.FormatMessage(msg))
	}

	return 0
}
