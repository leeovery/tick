package doctor

import (
	"context"
	"fmt"
)

// OrphanedParentCheck validates that every task with a parent field references
// a parent ID that exists in tasks.jsonl. Each orphaned parent reference is
// reported as an individual error. It is read-only and never modifies the file.
type OrphanedParentCheck struct{}

// Run executes the orphaned parent check. It parses task relationships from the
// given tick directory and checks that every non-empty parent reference points
// to a known task ID. Returns a single passing result if no orphans are found,
// or one failing result per orphaned reference.
func (c *OrphanedParentCheck) Run(ctx context.Context, tickDir string) []CheckResult {
	tasks, err := getTaskRelationships(ctx, tickDir)
	if err != nil {
		return []CheckResult{{
			Name:       "Orphaned parents",
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
		if task.Parent == "" {
			continue
		}
		if _, exists := knownIDs[task.Parent]; !exists {
			failures = append(failures, CheckResult{
				Name:       "Orphaned parents",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("%s references non-existent parent %s", task.ID, task.Parent),
				Suggestion: "Manual fix required",
			})
		}
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "Orphaned parents",
		Passed: true,
	}}
}
