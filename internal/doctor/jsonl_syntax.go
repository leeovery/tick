package doctor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// JsonlSyntaxCheck validates that every non-blank line in tasks.jsonl is
// syntactically valid JSON. It reports each malformed line individually with
// its 1-based line number. It is read-only and never modifies the file.
type JsonlSyntaxCheck struct{}

// Run executes the JSONL syntax check. It reads the tick directory path from
// the context (via TickDirKey), opens tasks.jsonl, and validates each line.
// Blank and whitespace-only lines are silently skipped but still count in
// line numbering. Returns a single passing result if all non-blank lines are
// valid JSON, or one failing result per malformed line.
func (c *JsonlSyntaxCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

	f, err := os.Open(jsonlPath)
	if err != nil {
		return []CheckResult{{
			Name:       "JSONL syntax",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}
	defer f.Close()

	var failures []CheckResult
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip blank and whitespace-only lines.
		if strings.TrimSpace(line) == "" {
			continue
		}

		if !json.Valid([]byte(line)) {
			preview := line
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			failures = append(failures, CheckResult{
				Name:       "JSONL syntax",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("Line %d: invalid JSON — %s", lineNum, preview),
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
