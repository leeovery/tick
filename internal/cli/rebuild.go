package cli

import (
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/storage"
)

// RunRebuild executes the rebuild command: forces a complete SQLite cache rebuild
// from JSONL, bypassing the freshness check. All lock management and file operations
// are delegated to the Store.
func RunRebuild(dir string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir, storeOpts(fc)...)
	if err != nil {
		return err
	}
	defer store.Close()

	count, err := store.Rebuild()
	if err != nil {
		return err
	}

	// Output confirmation.
	if !fc.Quiet {
		msg := fmt.Sprintf("Cache rebuilt: %d tasks", count)
		fmt.Fprintln(stdout, fmtr.FormatMessage(msg))
	}

	return nil
}
