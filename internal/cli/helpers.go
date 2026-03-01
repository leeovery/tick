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
// trims whitespace, lowercases, and filters empty values.
// Does not normalize to full IDs — callers resolve via store.ResolveID.
func parseCommaSeparatedIDs(s string) []string {
	parts := strings.Split(s, ",")
	var ids []string
	for _, part := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}

// validateTypeFlag normalizes the type value, checks it is non-empty, and validates
// it against the allowed type set. Returns the normalized value or an error.
func validateTypeFlag(value string) (string, error) {
	normalized := task.NormalizeType(value)
	if err := task.ValidateTypeNotEmpty(normalized); err != nil {
		return "", err
	}
	if err := task.ValidateType(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

// validateTagsFlag deduplicates tags, checks the result is non-empty (using emptyErr
// as the error message), and validates all tag values. Returns the deduplicated slice
// or an error.
func validateTagsFlag(tags []string, emptyErr string) ([]string, error) {
	deduped := task.DeduplicateTags(tags)
	if len(deduped) == 0 {
		return nil, fmt.Errorf("%s", emptyErr)
	}
	if err := task.ValidateTags(deduped); err != nil {
		return nil, err
	}
	return deduped, nil
}

// validateRefsFlag deduplicates refs, checks the result is non-empty (using emptyErr
// as the error message), and validates all ref values. Returns the deduplicated slice
// or an error.
func validateRefsFlag(refs []string, emptyErr string) ([]string, error) {
	deduped := task.DeduplicateRefs(refs)
	if len(deduped) == 0 {
		return nil, fmt.Errorf("%s", emptyErr)
	}
	if err := task.ValidateRefs(deduped); err != nil {
		return nil, err
	}
	return deduped, nil
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
