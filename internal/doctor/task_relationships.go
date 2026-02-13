package doctor

// TaskRelationshipData holds the fields extracted from a single task line
// that are needed by relationship and hierarchy checks.
type TaskRelationshipData struct {
	// ID is the task's unique identifier.
	ID string
	// Parent is the parent task ID, or empty string if null/absent.
	Parent string
	// BlockedBy is the list of blocking task IDs, or empty slice if null/absent.
	BlockedBy []string
	// Status is the task's status string, or empty string if absent.
	Status string
	// Line is the 1-based line number in tasks.jsonl.
	Line int
}

// taskRelationshipsFromLines converts a slice of JSONLine into
// TaskRelationshipData entries. Lines where Parsed is nil, or where the id
// field is missing or non-string, are skipped. This is a pure function with
// no I/O.
func taskRelationshipsFromLines(lines []JSONLine) []TaskRelationshipData {
	var result []TaskRelationshipData

	for _, jl := range lines {
		if jl.Parsed == nil {
			continue
		}

		idVal, exists := jl.Parsed["id"]
		if !exists {
			continue
		}
		idStr, ok := idVal.(string)
		if !ok {
			continue
		}

		entry := TaskRelationshipData{
			ID:        idStr,
			Line:      jl.LineNum,
			BlockedBy: []string{},
		}

		if parentVal, exists := jl.Parsed["parent"]; exists {
			if parentStr, ok := parentVal.(string); ok {
				entry.Parent = parentStr
			}
		}

		if blockedVal, exists := jl.Parsed["blocked_by"]; exists {
			if blockedArr, ok := blockedVal.([]interface{}); ok {
				for _, item := range blockedArr {
					if s, ok := item.(string); ok {
						entry.BlockedBy = append(entry.BlockedBy, s)
					}
				}
			}
		}

		if statusVal, exists := jl.Parsed["status"]; exists {
			if statusStr, ok := statusVal.(string); ok {
				entry.Status = statusStr
			}
		}

		result = append(result, entry)
	}

	if result == nil {
		result = []TaskRelationshipData{}
	}

	return result
}

// ParseTaskRelationships reads tasks.jsonl from the given tick directory and
// extracts relationship data for each valid task line. Blank lines, unparseable
// JSON, and lines with missing or non-string id fields are silently skipped.
// Returns an error if the file cannot be opened. Returns an empty slice for an
// empty file. This function is read-only and never modifies the file.
func ParseTaskRelationships(tickDir string) ([]TaskRelationshipData, error) {
	lines, err := ScanJSONLines(tickDir)
	if err != nil {
		return nil, err
	}

	return taskRelationshipsFromLines(lines), nil
}
