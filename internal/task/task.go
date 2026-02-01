// Package task defines the core task model, ID generation, and field validation
// for the Tick task tracker.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Status represents a task's lifecycle state.
type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

// Task represents a single task in the tracker.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Status      Status     `json:"status"`
	Priority    int        `json:"priority"`
	Description string     `json:"description,omitempty"`
	BlockedBy   []string   `json:"blocked_by,omitempty"`
	Parent      string     `json:"parent,omitempty"`
	Created     time.Time  `json:"created"`
	Updated     time.Time  `json:"updated"`
	Closed      *time.Time `json:"closed,omitempty"`
}

const maxRetries = 5

// GenerateID creates a new task ID in the format tick-{6 hex chars}.
// The exists function checks if an ID is already in use.
func GenerateID(exists func(string) bool) (string, error) {
	for i := 0; i < maxRetries; i++ {
		b := make([]byte, 3)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}
		id := "tick-" + hex.EncodeToString(b)
		if !exists(id) {
			return id, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique ID after %d attempts - task list may be too large", maxRetries)
}

// NormalizeID converts an ID to lowercase for case-insensitive matching.
func NormalizeID(id string) string {
	return strings.ToLower(id)
}

// TrimTitle removes leading and trailing whitespace from a title.
func TrimTitle(title string) string {
	return strings.TrimSpace(title)
}

// ValidateTitle checks that a title meets all constraints.
func ValidateTitle(title string) error {
	trimmed := TrimTitle(title)
	if trimmed == "" {
		return fmt.Errorf("title is required")
	}
	if len(trimmed) > 500 {
		return fmt.Errorf("title exceeds maximum length of 500 characters")
	}
	if strings.ContainsAny(trimmed, "\n\r") {
		return fmt.Errorf("title must not contain newlines")
	}
	return nil
}

// ValidatePriority checks that priority is in the valid range 0-4.
func ValidatePriority(p int) error {
	if p < 0 || p > 4 {
		return fmt.Errorf("priority must be between 0 and 4, got %d", p)
	}
	return nil
}

// ValidateBlockedBy checks for self-references in blocked_by.
func ValidateBlockedBy(taskID string, blockedBy []string) error {
	normalized := NormalizeID(taskID)
	for _, dep := range blockedBy {
		if NormalizeID(dep) == normalized {
			return fmt.Errorf("task cannot be blocked by itself")
		}
	}
	return nil
}

// ValidateParent checks for self-reference in parent.
func ValidateParent(taskID string, parentID string) error {
	if parentID == "" {
		return nil
	}
	if NormalizeID(taskID) == NormalizeID(parentID) {
		return fmt.Errorf("task cannot be its own parent")
	}
	return nil
}

// NewTask creates a new Task with defaults applied.
// Pass priority < 0 to use the default priority of 2.
func NewTask(id string, title string, priority int) Task {
	if priority < 0 {
		priority = 2
	}
	now := time.Now().UTC().Truncate(time.Second)
	return Task{
		ID:       id,
		Title:    title,
		Status:   StatusOpen,
		Priority: priority,
		Created:  now,
		Updated:  now,
	}
}

// FormatTimestamp formats a time as ISO 8601 UTC (YYYY-MM-DDTHH:MM:SSZ).
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
