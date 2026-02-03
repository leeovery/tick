// Package task defines the core Task data model, ID generation, and field validation for Tick.
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

// Status represents the state of a task.
type Status string

const (
	// StatusOpen indicates a task that has not been started.
	StatusOpen Status = "open"
	// StatusInProgress indicates a task currently being worked on.
	StatusInProgress Status = "in_progress"
	// StatusDone indicates a task that has been completed successfully.
	StatusDone Status = "done"
	// StatusCancelled indicates a task that was closed without completion.
	StatusCancelled Status = "cancelled"
)

const (
	idPrefix        = "tick-"
	idRandomBytes   = 3
	maxIDRetries    = 5
	maxTitleLength  = 500
	minPriority     = 0
	maxPriority     = 4
	defaultPriority = 2
)

// Task represents a single work item in Tick.
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

// TaskOptions holds optional fields for creating a new task.
type TaskOptions struct {
	Priority    *int
	Description string
	BlockedBy   []string
	Parent      string
}

// GenerateID creates a new task ID in the format tick-{6 hex chars}.
// It uses crypto/rand for randomness and retries up to 5 times on collision.
// The existsFn parameter checks whether a generated ID already exists.
func GenerateID(existsFn func(id string) bool) (string, error) {
	for attempt := 0; attempt < maxIDRetries; attempt++ {
		b := make([]byte, idRandomBytes)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}

		id := idPrefix + hex.EncodeToString(b)

		if !existsFn(id) {
			return id, nil
		}
	}

	return "", errors.New("Failed to generate unique ID after 5 attempts - task list may be too large")
}

// NormalizeID converts an ID to lowercase for case-insensitive matching.
func NormalizeID(id string) string {
	return strings.ToLower(id)
}

// ValidateTitle validates and normalizes a task title.
// It trims whitespace, rejects empty titles, titles exceeding 500 characters,
// and titles containing newlines. Returns the cleaned title or an error.
func ValidateTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)

	if trimmed == "" {
		return "", errors.New("title is required and cannot be empty")
	}

	if strings.ContainsAny(trimmed, "\n\r") {
		return "", errors.New("title cannot contain newlines")
	}

	if utf8.RuneCountInString(trimmed) > maxTitleLength {
		return "", fmt.Errorf("title exceeds maximum length of %d characters", maxTitleLength)
	}

	return trimmed, nil
}

// ValidatePriority checks that the priority value is within the valid range of 0-4.
func ValidatePriority(priority int) error {
	if priority < minPriority || priority > maxPriority {
		return fmt.Errorf("priority must be between %d and %d, got %d", minPriority, maxPriority, priority)
	}
	return nil
}

// ValidateBlockedBy checks that the blocked_by list does not contain a self-reference.
func ValidateBlockedBy(taskID string, blockedBy []string) error {
	normalizedTaskID := NormalizeID(taskID)
	for _, dep := range blockedBy {
		if NormalizeID(dep) == normalizedTaskID {
			return fmt.Errorf("task %s cannot block itself", taskID)
		}
	}
	return nil
}

// ValidateParent checks that the parent is not a self-reference.
func ValidateParent(taskID string, parentID string) error {
	if parentID == "" {
		return nil
	}
	if NormalizeID(parentID) == NormalizeID(taskID) {
		return fmt.Errorf("task %s cannot be its own parent", taskID)
	}
	return nil
}

// NewTask creates a new Task with the given title and options.
// It generates a unique ID, sets defaults, validates all fields,
// and returns the initialized task or an error.
func NewTask(title string, opts *TaskOptions, existsFn func(id string) bool) (*Task, error) {
	cleanTitle, err := ValidateTitle(title)
	if err != nil {
		return nil, fmt.Errorf("invalid title: %w", err)
	}

	id, err := GenerateID(existsFn)
	if err != nil {
		return nil, err
	}

	priority := defaultPriority
	if opts != nil && opts.Priority != nil {
		priority = *opts.Priority
	}

	if err := ValidatePriority(priority); err != nil {
		return nil, err
	}

	var blockedBy []string
	var parent string

	if opts != nil {
		blockedBy = opts.BlockedBy
		parent = opts.Parent
	}

	if len(blockedBy) > 0 {
		if err := ValidateBlockedBy(id, blockedBy); err != nil {
			return nil, err
		}
	}

	if parent != "" {
		if err := ValidateParent(id, parent); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC().Truncate(time.Second)

	return &Task{
		ID:          id,
		Title:       cleanTitle,
		Status:      StatusOpen,
		Priority:    priority,
		Description: optString(opts),
		BlockedBy:   blockedBy,
		Parent:      parent,
		Created:     now,
		Updated:     now,
		Closed:      nil,
	}, nil
}

// optString extracts the description from TaskOptions, returning empty string if opts is nil.
func optString(opts *TaskOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Description
}
