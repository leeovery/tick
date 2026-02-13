package doctor

import (
	"context"
	"fmt"
)

// ChildBlockedByParentCheck validates that no child task has its direct parent
// in its blocked_by list. A child blocked by its parent creates an unresolvable
// deadlock with the leaf-only ready rule: the parent cannot complete while it
// has open children, and the child cannot become ready while blocked by the
// parent. Only direct parent-child relationships are checked — grandparent and
// ancestor relationships are not flagged. It is read-only and never modifies
// the file.
type ChildBlockedByParentCheck struct{}

// Run executes the child-blocked-by-parent check. It reads the tick directory
// path from the context (via TickDirKey), parses task relationships, and checks
// that no task's blocked_by list contains its own parent. Returns a single
// passing result if no violations are found, or one failing result per child
// task that has its parent in blocked_by.
func (c *ChildBlockedByParentCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)

	tasks, err := getTaskRelationships(ctx, tickDir)
	if err != nil {
		return []CheckResult{{
			Name:       "Child blocked by parent",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}

	var failures []CheckResult
	for _, task := range tasks {
		if task.Parent == "" || len(task.BlockedBy) == 0 {
			continue
		}

		found := false
		for _, depID := range task.BlockedBy {
			if depID == task.Parent {
				found = true
				break
			}
		}
		if found {
			failures = append(failures, CheckResult{
				Name:       "Child blocked by parent",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("%s is blocked by its parent %s", task.ID, task.Parent),
				Suggestion: "Manual fix required \u2014 child blocked by parent creates deadlock with leaf-only ready rule",
			})
		}
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "Child blocked by parent",
		Passed: true,
	}}
}
