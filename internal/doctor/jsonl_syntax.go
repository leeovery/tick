package doctor

import (
	"context"
	"encoding/json"
	"fmt"
)

// JsonlSyntaxCheck validates that every non-blank line in tasks.jsonl is
// syntactically valid JSON. It reports each malformed line individually with
// its 1-based line number. It is read-only and never modifies the file.
type JsonlSyntaxCheck struct{}

// Run executes the JSONL syntax check. It reads tasks.jsonl from the given
// tick directory and validates each line. Blank and whitespace-only lines are
// silently skipped but still count in line numbering. Returns a single passing
// result if all non-blank lines are valid JSON, or one failing result per
// malformed line.
func (c *JsonlSyntaxCheck) Run(ctx context.Context, tickDir string) []CheckResult {
	lines, err := getJSONLines(ctx, tickDir)
	if err != nil {
		return fileNotFoundResult("JSONL syntax")
	}

	var failures []CheckResult
	for _, line := range lines {
		if line.Parsed != nil {
			continue
		}
		// Parsed is nil — check if the raw line is syntactically valid JSON.
		// ScanJSONLines only parses into map[string]interface{}, so valid JSON
		// arrays or primitives will have nil Parsed. Use json.Valid to confirm
		// actual syntax errors.
		if !json.Valid([]byte(line.Raw)) {
			preview := line.Raw
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			failures = append(failures, CheckResult{
				Name:       "JSONL syntax",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("Line %d: invalid JSON — %s", line.LineNum, preview),
				Suggestion: "Manual fix required",
			})
		}
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "JSONL syntax",
		Passed: true,
	}}
}
