package cli

import "fmt"

// runRebuild implements the `tick rebuild` command.
// It forces a complete SQLite cache rebuild from JSONL, bypassing freshness checks.
func (a *App) runRebuild(args []string) error {
	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	count, err := s.Rebuild()
	if err != nil {
		return err
	}

	if a.Quiet {
		return nil
	}

	msg := fmt.Sprintf("Rebuilt cache: %d tasks", count)
	return a.Formatter.FormatMessage(a.Stdout, msg)
}
