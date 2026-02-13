package doctor

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
