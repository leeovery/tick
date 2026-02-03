package cli

import "fmt"

// runRebuild implements the `tick rebuild` command.
// It forces a complete rebuild of the SQLite cache from JSONL,
// bypassing the freshness check.
func (a *App) runRebuild() error {
	tickDir, err := DiscoverTickDir(a.workDir)
	if err != nil {
		return err
	}

	store, err := a.newStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	count, err := store.ForceRebuild()
	if err != nil {
		return err
	}

	if a.config.Quiet {
		return nil
	}

	msg := fmt.Sprintf("Rebuilt cache: %d tasks", count)
	return a.formatter.FormatMessage(a.stdout, msg)
}
