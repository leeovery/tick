package cli

import (
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// openStore discovers the .tick directory from the given dir and opens a Store.
// Callers must defer store.Close() themselves since Go defers are scope-bound.
func openStore(dir string, fc FormatConfig) (*storage.Store, error) {
	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return nil, err
	}
	return storage.NewStore(tickDir, storeOpts(fc)...)
}

// parseCommaSeparatedIDs splits a comma-separated string of task IDs,
// trims whitespace, normalizes to lowercase, and filters empty values.
func parseCommaSeparatedIDs(s string) []string {
	parts := strings.Split(s, ",")
	var ids []string
	for _, part := range parts {
		normalized := task.NormalizeID(strings.TrimSpace(part))
		if normalized != "" {
			ids = append(ids, normalized)
		}
	}
	return ids
}

// applyBlocks iterates tasks and for each task whose ID appears in blockIDs,
// appends sourceID to its BlockedBy slice and sets its Updated timestamp.
// Skips the append if sourceID is already present in BlockedBy.
func applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time) {
	for i := range tasks {
		for _, blockID := range blockIDs {
			if tasks[i].ID == blockID {
				alreadyPresent := false
				for _, dep := range tasks[i].BlockedBy {
					if dep == sourceID {
						alreadyPresent = true
						break
					}
				}
				if !alreadyPresent {
					tasks[i].BlockedBy = append(tasks[i].BlockedBy, sourceID)
					tasks[i].Updated = now
				}
			}
		}
	}
}
