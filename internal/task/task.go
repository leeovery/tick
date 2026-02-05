// Package task provides the core Task model and validation logic for Tick.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// Status represents the state of a task.
type Status string

// Status constants for task lifecycle.
const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

// Task represents a single task in Tick.
// All timestamps use ISO 8601 UTC format (YYYY-MM-DDTHH:MM:SSZ).
type Task struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      Status   `json:"status"`
	Priority    int      `json:"priority"`
	Description string   `json:"description,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Closed      string   `json:"closed,omitempty"`
}

const (
	idPrefix   = "tick-"
	idHexLen   = 6
	maxRetries = 5
)

// ExistsFunc checks if an ID already exists. Returns true if ID exists (collision).
type ExistsFunc func(id string) (bool, error)

// GenerateID creates a new task ID in the format tick-{6 hex chars}.
// Uses crypto/rand for cryptographically secure random generation.
// Retries up to 5 times on collision, then returns an error.
func GenerateID(exists ExistsFunc) (string, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Generate 3 random bytes (6 hex chars)
		bytes := make([]byte, 3)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}

		id := idPrefix + hex.EncodeToString(bytes)

		collision, err := exists(id)
		if err != nil {
			return "", err
		}
		if !collision {
			return id, nil
		}
	}

	return "", errors.New("failed to generate unique ID after 5 attempts - task list may be too large")
}

// NormalizeID converts an ID to lowercase for case-insensitive matching.
func NormalizeID(id string) string {
	return strings.ToLower(id)
}

// ValidateTitle checks that a title is valid.
// Title must be non-empty, max 500 chars, and contain no newlines.
func ValidateTitle(title string) error {
	if title == "" {
		return errors.New("title is required")
	}
	if len(title) > 500 {
		return errors.New("title exceeds 500 characters")
	}
	if strings.ContainsAny(title, "\n\r") {
		return errors.New("title cannot contain newlines")
	}
	return nil
}

// TrimTitle removes leading and trailing whitespace from a title.
func TrimTitle(title string) string {
	return strings.TrimSpace(title)
}

// ValidatePriority checks that priority is in valid range 0-4.
func ValidatePriority(priority int) error {
	if priority < 0 || priority > 4 {
		return errors.New("priority must be between 0 and 4")
	}
	return nil
}

// ValidateBlockedBy checks that a task doesn't block itself.
// Uses case-insensitive comparison.
func ValidateBlockedBy(taskID string, blockedBy []string) error {
	normalizedTaskID := NormalizeID(taskID)
	for _, blockerID := range blockedBy {
		if NormalizeID(blockerID) == normalizedTaskID {
			return errors.New("task cannot block itself")
		}
	}
	return nil
}

// ValidateParent checks that a task isn't its own parent.
// Uses case-insensitive comparison.
func ValidateParent(taskID string, parent string) error {
	if parent == "" {
		return nil
	}
	if NormalizeID(parent) == NormalizeID(taskID) {
		return errors.New("task cannot be its own parent")
	}
	return nil
}

// DefaultPriority returns the default priority value (2 = medium).
func DefaultPriority() int {
	return 2
}

// DefaultTimestamps returns the current UTC time formatted as ISO 8601
// for both created and updated fields.
func DefaultTimestamps() (created, updated string) {
	now := time.Now().UTC().Format(time.RFC3339)
	return now, now
}
