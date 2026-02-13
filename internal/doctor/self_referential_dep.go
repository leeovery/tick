package doctor

import (
	"context"
	"fmt"
)

// SelfReferentialDepCheck validates that no task's blocked_by list contains the
// task's own ID. Each self-referential task is reported as an individual error.
// It is read-only and never modifies the file.
type SelfReferentialDepCheck struct{}

// Run executes the self-referential dependency check. It reads the tick directory
// path from the context (via TickDirKey), parses task relationships, and checks
// that no task references itself in its blocked_by list. Returns a single passing
// result if no self-references are found, or one failing result per self-referential
// task.
func (c *SelfReferentialDepCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)

	tasks, err := getTaskRelationships(ctx, tickDir)
	if err != nil {
		return []CheckResult{{
			Name:       "Self-referential dependencies",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}

	var failures []CheckResult
	for _, task := range tasks {
		selfRef := false
		for _, depID := range task.BlockedBy {
			if depID == task.ID {
				selfRef = true
				break
			}
		}
		if selfRef {
			failures = append(failures, CheckResult{
				Name:       "Self-referential dependencies",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("%s depends on itself", task.ID),
				Suggestion: "Manual fix required",
			})
		}
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "Self-referential dependencies",
		Passed: true,
	}}
}
