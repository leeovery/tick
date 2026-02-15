// Package migrate defines the contract types for importing tasks from external tools into tick.
package migrate

import (
	"fmt"
	"strings"
	"time"
)

// validStatuses defines the tick status values a migrated task may use.
var validStatuses = map[string]bool{
	"open":        true,
	"in_progress": true,
	"done":        true,
	"cancelled":   true,
}

const (
	minPriority = 0
	maxPriority = 4
)

// MigratedTask represents a normalized task ready for insertion into tick.
// Title is the only required field; all others use defaults when absent.
type MigratedTask struct {
	Title       string
	Status      string
	Priority    *int // nil means "not provided"; defaults applied at insertion time
	Description string
	Created     time.Time
	Updated     time.Time
	Closed      time.Time
}

// Validate checks that a MigratedTask satisfies tick's constraints.
// It returns an error if the title is empty, the status is unrecognized,
// or the priority is outside the 0-4 range.
func (mt MigratedTask) Validate() error {
	if strings.TrimSpace(mt.Title) == "" {
		return fmt.Errorf("title is required and cannot be empty")
	}
	if mt.Status != "" && !validStatuses[mt.Status] {
		return fmt.Errorf("invalid status %q: must be open, in_progress, done, or cancelled", mt.Status)
	}
	if mt.Priority != nil && (*mt.Priority < minPriority || *mt.Priority > maxPriority) {
		return fmt.Errorf("priority must be between %d and %d, got %d", minPriority, maxPriority, *mt.Priority)
	}
	return nil
}

// Provider abstracts a source system from which tasks can be imported.
type Provider interface {
	// Name returns the provider identifier (e.g., "beads") used in output.
	Name() string
	// Tasks returns all normalized tasks from the source, or an error if the source cannot be read.
	Tasks() ([]MigratedTask, error)
}

// Result records the outcome of importing a single task.
type Result struct {
	Title   string
	Success bool
	Err     error // nil on success
}
