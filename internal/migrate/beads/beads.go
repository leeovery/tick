// Package beads implements a migration provider that reads tasks from the beads
// JSONL format (.beads/issues.jsonl) and maps them to tick's MigratedTask type.
package beads

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/migrate"
)

// statusMap translates beads status values to tick equivalents.
var statusMap = map[string]string{
	"pending":     "open",
	"in_progress": "in_progress",
	"closed":      "done",
}

// beadsIssue is the intermediate struct for JSON unmarshalling of a single
// beads issue line. Fields with no tick equivalent are parsed but discarded
// during mapping.
type beadsIssue struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Status       string        `json:"status"`
	Priority     int           `json:"priority"`
	IssueType    string        `json:"issue_type"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	ClosedAt     string        `json:"closed_at"`
	CloseReason  string        `json:"close_reason"`
	CreatedBy    string        `json:"created_by"`
	Dependencies []interface{} `json:"dependencies"`
}

// BeadsProvider reads tasks from .beads/issues.jsonl in a given base directory.
type BeadsProvider struct {
	baseDir string
}

// Compile-time check that BeadsProvider satisfies migrate.Provider.
var _ migrate.Provider = (*BeadsProvider)(nil)

// NewBeadsProvider creates a provider that reads from baseDir/.beads/issues.jsonl.
func NewBeadsProvider(baseDir string) *BeadsProvider {
	return &BeadsProvider{baseDir: baseDir}
}

// Name returns the provider identifier.
func (p *BeadsProvider) Name() string {
	return "beads"
}

// Tasks reads .beads/issues.jsonl, parses each line, and returns all entries
// as MigratedTask values. Malformed JSON lines are returned as sentinel entries
// with the title "(malformed entry)" so the engine can report them as failures.
// Empty titles and validation failures are left for the engine to handle.
// Returns an error only if the .beads directory or issues.jsonl file is missing.
func (p *BeadsProvider) Tasks() ([]migrate.MigratedTask, error) {
	beadsDir := filepath.Join(p.baseDir, ".beads")
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf(".beads directory not found in %s", p.baseDir)
	}

	jsonlPath := filepath.Join(beadsDir, "issues.jsonl")
	if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("issues.jsonl not found in %s", beadsDir)
	}

	file, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", jsonlPath, err)
	}
	defer file.Close()

	var tasks []migrate.MigratedTask
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var issue beadsIssue
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			// Malformed JSON — return sentinel entry so the engine reports it as a failure.
			// Status "(invalid)" forces a validation error while the descriptive title
			// makes the failure visible to the user.
			tasks = append(tasks, migrate.MigratedTask{
				Title:  "(malformed entry)",
				Status: "(invalid)",
			})
			continue
		}

		tasks = append(tasks, mapToMigratedTask(issue))
	}

	if err := scanner.Err(); err != nil {
		return tasks, fmt.Errorf("error reading %s: %w", jsonlPath, err)
	}

	return tasks, nil
}

// mapToMigratedTask converts a beadsIssue to a migrate.MigratedTask.
// Empty or whitespace-only titles are preserved; the engine handles validation.
func mapToMigratedTask(issue beadsIssue) migrate.MigratedTask {
	status := statusMap[issue.Status] // unknown/empty maps to "" (zero value)

	priority := issue.Priority

	created, _ := time.Parse(time.RFC3339, issue.CreatedAt)
	updated, _ := time.Parse(time.RFC3339, issue.UpdatedAt)
	closed, _ := time.Parse(time.RFC3339, issue.ClosedAt)

	return migrate.MigratedTask{
		Title:       issue.Title,
		Description: issue.Description,
		Status:      status,
		Priority:    &priority,
		Created:     created,
		Updated:     updated,
		Closed:      closed,
	}
}
