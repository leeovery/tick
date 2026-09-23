package doctor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type seqOccurrence struct {
	id         string
	lineNumber int
}

// DuplicateSeqCheck warns when more than one record in tasks.jsonl carries the
// same creation sequence. Records with an absent or zero sequence carry none
// and are not compared. It is read-only and never modifies the file.
type DuplicateSeqCheck struct{}

// Run executes the duplicate sequence check against tasks.jsonl in tickDir.
// It returns one SeverityWarning result per shared sequence, in order of first
// appearance, or a single passing result when no sequence is shared.
// Unparseable lines are skipped.
func (c *DuplicateSeqCheck) Run(ctx context.Context, tickDir string) []CheckResult {
	const checkName = "Sequence uniqueness"

	lines, err := getJSONLines(ctx, tickDir)
	if err != nil {
		return fileNotFoundResult(checkName)
	}

	groups := make(map[float64][]seqOccurrence)
	var seqOrder []float64

	for _, line := range lines {
		if line.Parsed == nil {
			continue
		}

		// Sequences decode as float64; zero means the record carries none.
		seq, ok := line.Parsed["seq"].(float64)
		if !ok || seq == 0 {
			continue
		}

		if _, seen := groups[seq]; !seen {
			seqOrder = append(seqOrder, seq)
		}
		id, _ := line.Parsed["id"].(string)
		groups[seq] = append(groups[seq], seqOccurrence{id: id, lineNumber: line.LineNum})
	}

	var failures []CheckResult
	for _, seq := range seqOrder {
		occurrences := groups[seq]
		if len(occurrences) <= 1 {
			continue
		}

		parts := make([]string, len(occurrences))
		for i, occ := range occurrences {
			parts[i] = fmt.Sprintf("%s (line %d)", occ.id, occ.lineNumber)
		}

		failures = append(failures, CheckResult{
			Name:       checkName,
			Passed:     false,
			Severity:   SeverityWarning,
			Details:    fmt.Sprintf("Duplicate sequence %s: %s", strconv.FormatFloat(seq, 'f', -1, 64), strings.Join(parts, ", ")),
			Suggestion: "Edit the seq values in tasks.jsonl so they differ",
		})
	}

	if len(failures) > 0 {
		return failures
	}

	return []CheckResult{{
		Name:   checkName,
		Passed: true,
	}}
}
