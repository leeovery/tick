package doctor

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
