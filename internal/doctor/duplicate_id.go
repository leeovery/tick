package doctor

import (
	"context"
	"fmt"
	"strings"
)

// idOccurrence records a single occurrence of an ID with its original case and line number.
type idOccurrence struct {
	originalID string
	lineNumber int
}

// DuplicateIdCheck validates that no two tasks in tasks.jsonl share the same ID
// when compared case-insensitively. Each group of duplicates is reported as an
// individual error with line numbers and original-case forms. It is read-only
// and never modifies the file.
type DuplicateIdCheck struct{}

// Run executes the duplicate ID check. It reads the tick directory path from
// the context (via TickDirKey), calls ScanJSONLines, and groups IDs by their
// lowercase-normalized form. Any group with more than one entry produces a
// failing result. Unparseable JSON and missing/empty IDs are silently skipped.
func (c *DuplicateIdCheck) Run(ctx context.Context) []CheckResult {
	tickDir, _ := ctx.Value(TickDirKey).(string)

	lines, err := getJSONLines(ctx, tickDir)
	if err != nil {
		return []CheckResult{{
			Name:       "ID uniqueness",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "tasks.jsonl not found",
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}

	// Map from lowercase(id) to list of occurrences.
	groups := make(map[string][]idOccurrence)
	// Track insertion order of keys for deterministic output.
	var keyOrder []string

	for _, line := range lines {
		if line.Parsed == nil {
			continue
		}

		idVal, exists := line.Parsed["id"]
		if !exists {
			continue
		}

		idStr, ok := idVal.(string)
		if !ok || idStr == "" {
			continue
		}

		key := strings.ToLower(idStr)
		if _, seen := groups[key]; !seen {
			keyOrder = append(keyOrder, key)
		}
		groups[key] = append(groups[key], idOccurrence{
			originalID: idStr,
			lineNumber: line.LineNum,
		})
	}

	var failures []CheckResult
	for _, key := range keyOrder {
		occurrences := groups[key]
		if len(occurrences) <= 1 {
			continue
		}

		parts := make([]string, len(occurrences))
		for i, occ := range occurrences {
			parts[i] = fmt.Sprintf("%s (line %d)", occ.originalID, occ.lineNumber)
		}

		details := fmt.Sprintf("Duplicate ID %s: %s", key, strings.Join(parts, ", "))

		failures = append(failures, CheckResult{
			Name:       "ID uniqueness",
			Passed:     false,
			Severity:   SeverityError,
			Details:    details,
			Suggestion: "Manual fix required",
		})
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   "ID uniqueness",
		Passed: true,
	}}
}
