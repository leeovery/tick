package doctor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// ParseTaskRelationships reads tasks.jsonl from the given tick directory and
// extracts relationship data for each valid task line. Blank lines, unparseable
// JSON, and lines with missing or non-string id fields are silently skipped.
// Returns an error if the file cannot be opened. Returns an empty slice for an
// empty file. This function is read-only and never modifies the file.
func ParseTaskRelationships(tickDir string) ([]TaskRelationshipData, error) {
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("open tasks.jsonl: %w", err)
	}
	defer f.Close()

	var result []TaskRelationshipData
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.TrimSpace(line) == "" {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		idVal, exists := obj["id"]
		if !exists {
			continue
		}
		idStr, ok := idVal.(string)
		if !ok {
			continue
		}

		entry := TaskRelationshipData{
			ID:        idStr,
			Line:      lineNum,
			BlockedBy: []string{},
		}

		if parentVal, exists := obj["parent"]; exists {
			if parentStr, ok := parentVal.(string); ok {
				entry.Parent = parentStr
			}
		}

		if blockedVal, exists := obj["blocked_by"]; exists {
			if blockedArr, ok := blockedVal.([]interface{}); ok {
				for _, item := range blockedArr {
					if s, ok := item.(string); ok {
						entry.BlockedBy = append(entry.BlockedBy, s)
					}
				}
			}
		}

		if statusVal, exists := obj["status"]; exists {
			if statusStr, ok := statusVal.(string); ok {
				entry.Status = statusStr
			}
		}

		result = append(result, entry)
	}

	if result == nil {
		result = []TaskRelationshipData{}
	}

	return result, nil
}
