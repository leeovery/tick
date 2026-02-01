package cli

import "fmt"

func (a *App) cmdRebuild(workDir string) error {
	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	count, err := store.ForceRebuild()
	if err != nil {
		return err
	}

	if a.opts.Quiet {
		return nil
	}

	return a.fmtr.FormatMessage(a.stdout, fmt.Sprintf("Cache rebuilt: %d tasks", count))
}
