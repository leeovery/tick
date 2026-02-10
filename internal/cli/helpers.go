package cli

import (
	"strings"
	"time"

	"github.com/leeovery/tick/internal/task"
)

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
func applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time) {
	for i := range tasks {
		for _, blockID := range blockIDs {
			if tasks[i].ID == blockID {
				tasks[i].BlockedBy = append(tasks[i].BlockedBy, sourceID)
				tasks[i].Updated = now
			}
		}
	}
}
