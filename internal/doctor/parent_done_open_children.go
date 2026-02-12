package doctor

import (
	"context"
	"fmt"
	"sort"
)

// ParentDoneWithOpenChildrenCheck validates that no parent task marked "done"
// has children that are still open (status "open" or "in_progress"). This is
// the only warning-severity check in the doctor suite — it flags suspicious
// but allowed states. It is read-only and never modifies the file.
type ParentDoneWithOpenChildrenCheck struct{}

// Run executes the parent-done-with-open-children check. It reads the tick
// directory path from the context (via TickDirKey), parses task relationships,
// and checks that no done parent has open or in_progress children. Returns a
// single passing result if no issues are found, or one failing result per
// done-parent + open-child pair with SeverityWarning.
func (c *ParentDoneWithOpenChildrenCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)

	tasks, err := ParseTaskRelationships(tickDir)
	if err != nil {
		return []CheckResult{{
			Name:       "Parent done with open children",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}

	statusMap := make(map[string]string, len(tasks))
	childrenMap := make(map[string][]string)

	for _, task := range tasks {
		statusMap[task.ID] = task.Status
		if task.Parent != "" {
			childrenMap[task.Parent] = append(childrenMap[task.Parent], task.ID)
		}
	}

	// Sort parent IDs for deterministic output.
	parentIDs := make([]string, 0, len(childrenMap))
	for parentID := range childrenMap {
		parentIDs = append(parentIDs, parentID)
	}
	sort.Strings(parentIDs)

	var failures []CheckResult
	for _, parentID := range parentIDs {
		parentStatus, exists := statusMap[parentID]
		if !exists {
			continue
		}
		if parentStatus != "done" {
			continue
		}

		children := childrenMap[parentID]
		sort.Strings(children)

		for _, childID := range children {
			childStatus := statusMap[childID]
			if childStatus == "open" || childStatus == "in_progress" {
				failures = append(failures, CheckResult{
					Name:       "Parent done with open children",
					Passed:     false,
					Severity:   SeverityWarning,
					Details:    fmt.Sprintf("%s is done but has open child %s", parentID, childID),
					Suggestion: "Review whether parent was completed prematurely",
				})
			}
		}
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "Parent done with open children",
		Passed: true,
	}}
}
