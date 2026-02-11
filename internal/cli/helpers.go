package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// outputMutationResult handles post-mutation output for create and update commands.
// In quiet mode it prints only the task ID; otherwise it queries the full task detail
// from the store and formats it via the Formatter.
func outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
	if fc.Quiet {
		fmt.Fprintln(stdout, id)
		return nil
	}

	data, err := queryShowData(store, id)
	if err != nil {
		return err
	}

	detail := showDataToTaskDetail(data)
	fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
	return nil
}

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
			if task.NormalizeID(tasks[i].ID) == task.NormalizeID(blockID) {
				alreadyPresent := false
				for _, dep := range tasks[i].BlockedBy {
					if task.NormalizeID(dep) == task.NormalizeID(sourceID) {
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
