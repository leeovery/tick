package doctor

import "errors"

// buildKnownIDs returns a set of all task IDs from the given relationship data.
func buildKnownIDs(tasks []TaskRelationshipData) map[string]struct{} {
	knownIDs := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		knownIDs[task.ID] = struct{}{}
	}
	return knownIDs
}

// fileNotFoundResult returns the standard CheckResult for when tasks.jsonl
// cannot be found. The checkName parameter sets the Name field.
func fileNotFoundResult(checkName string) []CheckResult {
	return []CheckResult{{
		Name:       checkName,
		Passed:     false,
		Severity:   SeverityError,
		Details:    "tasks.jsonl not found",
		Suggestion: "Run tick init or verify .tick directory",
	}}
}

// linesUnavailableResult returns the failure a check that consumes the lines
// of tasks.jsonl reports in place of its verdict when it cannot get them: the
// read error when the file opened but could not be read in full, otherwise
// not found.
func linesUnavailableResult(checkName string, err error) []CheckResult {
	if !errors.Is(err, errIncompleteRead) {
		return fileNotFoundResult(checkName)
	}
	return []CheckResult{{
		Name:       checkName,
		Passed:     false,
		Severity:   SeverityError,
		Details:    err.Error(),
		Suggestion: "Verify tasks.jsonl is readable",
	}}
}
