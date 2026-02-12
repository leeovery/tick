package doctor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// idFormatRegex matches valid tick IDs: tick- followed by exactly 6 lowercase hex chars.
var idFormatRegex = regexp.MustCompile(`^tick-[0-9a-f]{6}$`)

// IdFormatCheck validates that every task in tasks.jsonl has an id field matching
// the required pattern tick-{6 hex}. Each invalid ID is reported as an individual
// error with its 1-based line number. It is read-only and never modifies the file.
type IdFormatCheck struct{}

// Run executes the ID format check. It reads the tick directory path from
// the context (via TickDirKey), opens tasks.jsonl, and validates each task's id field.
// Blank and whitespace-only lines are silently skipped but still count in
// line numbering. Unparseable JSON lines are skipped silently (syntax check handles those).
// Returns a single passing result if all IDs are valid, or one failing result per invalid ID.
func (c *IdFormatCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

	f, err := os.Open(jsonlPath)
	if err != nil {
		return []CheckResult{{
			Name:       "ID format",
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

		// Attempt JSON parse to map[string]interface{}.
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// Unparseable JSON — skip silently (syntax check handles those).
			continue
		}

		// Extract the id field.
		idVal, exists := obj["id"]
		if !exists {
			failures = append(failures, CheckResult{
				Name:       "ID format",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("Line %d: missing id field", lineNum),
				Suggestion: "Manual fix required",
			})
			continue
		}

		// Check if id is a string.
		idStr, ok := idVal.(string)
		if !ok {
			// Non-string value (null, number, etc.)
			display := formatNonStringID(idVal)
			failures = append(failures, CheckResult{
				Name:       "ID format",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("Line %d: invalid ID '%s' — expected format tick-{6 hex}", lineNum, display),
				Suggestion: "Manual fix required",
			})
			continue
		}

		// Validate against regex.
		if !idFormatRegex.MatchString(idStr) {
			failures = append(failures, CheckResult{
				Name:       "ID format",
				Passed:     false,
				Severity:   SeverityError,
				Details:    fmt.Sprintf("Line %d: invalid ID '%s' — expected format tick-{6 hex}", lineNum, idStr),
				Suggestion: "Manual fix required",
			})
			continue
		}
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "ID format",
		Passed: true,
	}}
}

// formatNonStringID returns a display string for non-string id values.
func formatNonStringID(val interface{}) string {
	if val == nil {
		return "<null>"
	}
	// For numbers, json.Unmarshal decodes to float64.
	if num, ok := val.(float64); ok {
		// If it's a whole number, show without decimal.
		if num == float64(int64(num)) {
			return fmt.Sprintf("%d", int64(num))
		}
		return fmt.Sprintf("%g", num)
	}
	return fmt.Sprintf("%v", val)
}
