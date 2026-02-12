package doctor

import (
	"context"
	"fmt"
)

// OrphanedDependencyCheck validates that every task's blocked_by entries reference
// task IDs that exist in tasks.jsonl. Each orphaned dependency reference is
// reported as an individual error. It is read-only and never modifies the file.
type OrphanedDependencyCheck struct{}

// Run executes the orphaned dependency check. It reads the tick directory path
// from the context (via TickDirKey), parses task relationships, and checks that
// every blocked_by entry points to a known task ID. Returns a single passing
// result if no orphans are found, or one failing result per orphaned reference.
func (c *OrphanedDependencyCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)

	tasks, err := ParseTaskRelationships(tickDir)
	if err != nil {
		return []CheckResult{{
			Name:       "Orphaned dependencies",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}

	knownIDs := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		knownIDs[task.ID] = struct{}{}
	}

	var failures []CheckResult
	for _, task := range tasks {
		for _, depID := range task.BlockedBy {
			if _, exists := knownIDs[depID]; !exists {
				failures = append(failures, CheckResult{
					Name:       "Orphaned dependencies",
					Passed:     false,
					Severity:   SeverityError,
					Details:    fmt.Sprintf("%s depends on non-existent task %s", task.ID, depID),
					Suggestion: "Manual fix required",
				})
			}
		}
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "Orphaned dependencies",
		Passed: true,
	}}
}
