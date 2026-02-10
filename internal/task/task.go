// Package task defines the core task data model, ID generation, and field validation for Tick.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Status represents the lifecycle state of a task.
type Status string

const (
	// StatusOpen indicates a task that has not been started.
	StatusOpen Status = "open"
	// StatusInProgress indicates a task currently being worked on.
	StatusInProgress Status = "in_progress"
	// StatusDone indicates a task completed successfully.
	StatusDone Status = "done"
	// StatusCancelled indicates a task closed without completion.
	StatusCancelled Status = "cancelled"
)

const (
	maxIDRetries    = 5
	idByteLength    = 3
	idPrefix        = "tick-"
	maxTitleLen     = 500
	defaultPriority = 2
	minPriority     = 0
	maxPriority     = 4
)

// TimestampFormat is the ISO 8601 UTC format used for all task timestamps.
const TimestampFormat = "2006-01-02T15:04:05Z"

// Task represents a work item in Tick with all 10 schema fields.
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

// GenerateID creates a new task ID in the format tick-{6 hex chars} using crypto/rand.
// The exists function is called to check for collisions; up to 5 retries are attempted.
func GenerateID(exists func(id string) bool) (string, error) {
	for attempt := 0; attempt < maxIDRetries; attempt++ {
		b := make([]byte, idByteLength)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}

		id := idPrefix + hex.EncodeToString(b)

		if !exists(id) {
			return id, nil
		}
	}

	return "", errors.New("failed to generate unique ID after 5 attempts - task list may be too large")
}

// NormalizeID converts a task ID to lowercase for case-insensitive matching.
func NormalizeID(id string) string {
	return strings.ToLower(id)
}

// ValidateTitle checks that a title is non-empty, has no newlines, and is at most 500 characters.
// The title should be trimmed before validation using TrimTitle.
func ValidateTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return errors.New("title is required and cannot be empty")
	}
	if strings.ContainsAny(trimmed, "\n\r") {
		return errors.New("title must be a single line (no newlines)")
	}
	if utf8.RuneCountInString(trimmed) > maxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", maxTitleLen)
	}
	return nil
}

// TrimTitle removes leading and trailing whitespace from a title.
func TrimTitle(title string) string {
	return strings.TrimSpace(title)
}

// ValidatePriority checks that a priority value is in the range 0-4.
func ValidatePriority(priority int) error {
	if priority < minPriority || priority > maxPriority {
		return fmt.Errorf("priority must be between %d and %d, got %d", minPriority, maxPriority, priority)
	}
	return nil
}

// ValidateBlockedBy checks that a task does not reference itself in its blocked_by list.
func ValidateBlockedBy(taskID string, blockedBy []string) error {
	for _, dep := range blockedBy {
		if NormalizeID(dep) == NormalizeID(taskID) {
			return fmt.Errorf("task %s cannot be blocked by itself", taskID)
		}
	}
	return nil
}

// ValidateParent checks that a task does not reference itself as its own parent.
func ValidateParent(taskID string, parent string) error {
	if parent == "" {
		return nil
	}
	if NormalizeID(parent) == NormalizeID(taskID) {
		return fmt.Errorf("task %s cannot be its own parent", taskID)
	}
	return nil
}

// FormatTimestamp formats a time.Time as ISO 8601 UTC string.
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(TimestampFormat)
}

// NewTask creates a new Task with the given title and default values.
// The exists function is used for ID collision detection; pass nil to skip collision checks.
func NewTask(title string, exists func(id string) bool) (*Task, error) {
	trimmed := TrimTitle(title)
	if err := ValidateTitle(trimmed); err != nil {
		return nil, err
	}

	if exists == nil {
		exists = func(id string) bool { return false }
	}

	id, err := GenerateID(exists)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Truncate(time.Second)

	return &Task{
		ID:       id,
		Title:    trimmed,
		Status:   StatusOpen,
		Priority: defaultPriority,
		Created:  now,
		Updated:  now,
	}, nil
}
