package doctor

import (
	"context"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// JsonlSyntaxCheck validates that every line tasks.jsonl holds loads as a task
// exactly as the store loads it. It reports each line that does not load
// individually with its 1-based line number and the loader's reason. It is
// read-only and never modifies the file.
type JsonlSyntaxCheck struct{}

// Run executes the JSONL syntax check. It reads tasks.jsonl from the given
// tick directory and loads each line. Returns a single passing result if every
// line loads, or one failing result per line that does not.
func (c *JsonlSyntaxCheck) Run(ctx context.Context, tickDir string) []CheckResult {
	lines, err := getJSONLines(ctx, tickDir)
	if err != nil {
		return fileNotFoundResult("JSONL syntax")
	}

	var failures []CheckResult
	for _, line := range lines {
		if _, err := storage.DecodeTaskLine([]byte(line.Raw)); err != nil {
			failures = append(failures, loadFailure(line, err))
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

func loadFailure(line JSONLine, err error) CheckResult {
	preview := line.Raw
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	return CheckResult{
		Name:       "JSONL syntax",
		Passed:     false,
		Severity:   SeverityError,
		Details:    fmt.Sprintf("Line %d: %v — %s", line.LineNum, err, preview),
		Suggestion: "Manual fix required",
	}
}
