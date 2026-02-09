package cli

import (
	"fmt"

	"github.com/leeovery/tick/internal/engine"
)

// runRebuild implements the "tick rebuild" command. It forces a complete SQLite
// cache rebuild from JSONL, bypassing the freshness check.
func runRebuild(ctx *Context) error {
	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
	if err != nil {
		return err
	}
	defer store.Close()

	count, err := store.Rebuild()
	if err != nil {
		return err
	}

	if !ctx.Quiet {
		ctx.Fmt.FormatMessage(ctx.Stdout, fmt.Sprintf("Cache rebuilt: %d tasks", count))
	}

	return nil
}
