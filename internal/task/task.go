// Package task provides the core task model and validation for Tick.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Status represents the state of a task.
type Status string

const (
	// StatusOpen indicates the task has not been started.
	StatusOpen Status = "open"
	// StatusInProgress indicates the task is being worked on.
	StatusInProgress Status = "in_progress"
	// StatusDone indicates the task has been completed successfully.
	StatusDone Status = "done"
	// StatusCancelled indicates the task was closed without completion.
	StatusCancelled Status = "cancelled"
)

// Task represents a single task in the Tick system.
// All timestamps use ISO 8601 UTC format (YYYY-MM-DDTHH:MM:SSZ).
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Status      Status    `json:"status"`
	Priority    int       `json:"priority"`
	Description string    `json:"description,omitempty"`
	BlockedBy   []string  `json:"blocked_by,omitempty"`
	Parent      string    `json:"parent,omitempty"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Closed      *time.Time `json:"closed,omitempty"`
}

// TaskOptions holds optional parameters for creating a new task.
type TaskOptions struct {
	Priority    *int
	Description string
	BlockedBy   []string
	Parent      string
}

const (
	maxTitleLength  = 500
	defaultPriority = 2
)

// NewTask creates a new Task with validated fields and generated ID.
// If opts is nil, defaults are used (priority 2, no description, no blockers, no parent).
// The exists function is used by ID generation to check for collisions.
func NewTask(title string, opts *TaskOptions, exists func(id string) bool) (*Task, error) {
	validTitle, err := ValidateTitle(title)
	if err != nil {
		return nil, fmt.Errorf("invalid title: %w", err)
	}

	priority := defaultPriority
	var description string
	var blockedBy []string
	var parent string

	if opts != nil {
		if opts.Priority != nil {
			priority = *opts.Priority
		}
		description = opts.Description
		blockedBy = opts.BlockedBy
		parent = opts.Parent
	}

	if err := ValidatePriority(priority); err != nil {
		return nil, err
	}

	id, err := GenerateID(exists)
	if err != nil {
		return nil, err
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
		Title:       validTitle,
		Status:      StatusOpen,
		Priority:    priority,
		Description: description,
		BlockedBy:   blockedBy,
		Parent:      parent,
		Created:     now,
		Updated:     now,
	}, nil
}

// GenerateID creates a new task ID in the format tick-{6 hex chars}.
// It uses crypto/rand for 3 random bytes, producing 6 lowercase hex characters.
// The exists function checks whether a generated ID already exists.
// On collision, it retries up to 5 times before returning an error.
func GenerateID(exists func(id string) bool) (string, error) {
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		b := make([]byte, 3)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}

		id := "tick-" + hex.EncodeToString(b)

		if !exists(id) {
			return id, nil
		}
	}

	return "", fmt.Errorf("Failed to generate unique ID after 5 attempts - task list may be too large")
}

// NormalizeID converts a task ID to lowercase for case-insensitive matching.
func NormalizeID(id string) string {
	return strings.ToLower(id)
}

// ValidateTitle validates and normalizes a task title.
// It trims whitespace, rejects empty titles, titles exceeding 500 characters,
// and titles containing newlines. Returns the trimmed title on success.
func ValidateTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)

	if trimmed == "" {
		return "", fmt.Errorf("title is required and cannot be empty")
	}

	if strings.ContainsAny(trimmed, "\n\r") {
		return "", fmt.Errorf("title must be a single line (no newlines)")
	}

	if utf8.RuneCountInString(trimmed) > maxTitleLength {
		return "", fmt.Errorf("title exceeds maximum length of %d characters", maxTitleLength)
	}

	return trimmed, nil
}

// ValidatePriority checks that the priority is within the valid range of 0-4.
func ValidatePriority(priority int) error {
	if priority < 0 || priority > 4 {
		return fmt.Errorf("priority must be between 0 and 4, got %d", priority)
	}
	return nil
}

// ValidateBlockedBy checks that the blocked_by list does not contain a self-reference.
func ValidateBlockedBy(taskID string, blockedBy []string) error {
	for _, dep := range blockedBy {
		if dep == taskID {
			return fmt.Errorf("task %s cannot be blocked by itself", taskID)
		}
	}
	return nil
}

// ValidateParent checks that the parent is not the same as the task itself.
func ValidateParent(taskID string, parent string) error {
	if parent == taskID {
		return fmt.Errorf("task %s cannot be its own parent", taskID)
	}
	return nil
}
