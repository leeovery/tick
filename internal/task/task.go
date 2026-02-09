// Package task defines the core Task model and ID generation for Tick.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	idPrefix    = "tick-"
	idRetries   = 5
	idRandSize  = 3 // 3 bytes = 6 hex chars
	maxTitleLen = 500

	// DefaultPriority is the default priority for new tasks.
	DefaultPriority = 2

	// TimestampFormat is the ISO 8601 UTC format used for all timestamps.
	TimestampFormat = "2006-01-02T15:04:05Z"
)

// Status represents the current state of a task.
type Status string

const (
	// StatusOpen indicates a task that has not been started.
	StatusOpen Status = "open"
	// StatusInProgress indicates a task that is being worked on.
	StatusInProgress Status = "in_progress"
	// StatusDone indicates a task that has been completed successfully.
	StatusDone Status = "done"
	// StatusCancelled indicates a task that was closed without completion.
	StatusCancelled Status = "cancelled"
)

// Task represents a single task in the Tick system. It has 10 fields as defined
// by the task schema. Optional fields use Go zero values or nil and are omitted
// from JSON when empty.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Status      Status     `json:"status"`
	Priority    int        `json:"priority"`
	Description string     `json:"description,omitempty"`
	BlockedBy   []string   `json:"blocked_by,omitempty"`
	Parent      string     `json:"parent,omitempty"`
	Created     time.Time  `json:"-"`
	Updated     time.Time  `json:"-"`
	Closed      *time.Time `json:"-"`
}

// taskJSON is the JSON-serializable representation of a Task with timestamps
// formatted as ISO 8601 UTC strings.
type taskJSON struct {
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

// MarshalJSON implements custom JSON marshaling for Task to format timestamps
// as ISO 8601 UTC strings and omit optional fields when empty.
func (t Task) MarshalJSON() ([]byte, error) {
	j := taskJSON{
		ID:          t.ID,
		Title:       t.Title,
		Status:      t.Status,
		Priority:    t.Priority,
		Description: t.Description,
		BlockedBy:   t.BlockedBy,
		Parent:      t.Parent,
		Created:     FormatTimestamp(t.Created),
		Updated:     FormatTimestamp(t.Updated),
	}
	if t.Closed != nil {
		j.Closed = FormatTimestamp(*t.Closed)
	}
	return json.Marshal(j)
}

// UnmarshalJSON implements custom JSON unmarshaling for Task to parse ISO 8601
// UTC timestamp strings.
func (t *Task) UnmarshalJSON(data []byte) error {
	var j taskJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshaling task JSON: %w", err)
	}

	created, err := time.Parse(TimestampFormat, j.Created)
	if err != nil {
		return fmt.Errorf("parsing created timestamp: %w", err)
	}
	updated, err := time.Parse(TimestampFormat, j.Updated)
	if err != nil {
		return fmt.Errorf("parsing updated timestamp: %w", err)
	}

	t.ID = j.ID
	t.Title = j.Title
	t.Status = j.Status
	t.Priority = j.Priority
	t.Description = j.Description
	t.BlockedBy = j.BlockedBy
	t.Parent = j.Parent
	t.Created = created
	t.Updated = updated

	if j.Closed != "" {
		closed, err := time.Parse(TimestampFormat, j.Closed)
		if err != nil {
			return fmt.Errorf("parsing closed timestamp: %w", err)
		}
		t.Closed = &closed
	}

	return nil
}

// NewTask creates a new Task with the given ID and title, setting default values
// for status (open), priority (2), and timestamps (current UTC time).
func NewTask(id, title string) Task {
	now := time.Now().UTC().Truncate(time.Second)
	return Task{
		ID:       id,
		Title:    title,
		Status:   StatusOpen,
		Priority: DefaultPriority,
		Created:  now,
		Updated:  now,
	}
}

// FormatTimestamp formats a time.Time as an ISO 8601 UTC string (YYYY-MM-DDTHH:MM:SSZ).
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(TimestampFormat)
}

// GenerateID creates a new task ID in the format tick-{6 hex chars} using
// crypto/rand. The exists function checks whether a generated ID is already
// in use; if so, generation is retried up to 5 times before returning an error.
func GenerateID(exists func(id string) bool) (string, error) {
	for i := 0; i < idRetries; i++ {
		b := make([]byte, idRandSize)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("reading random bytes: %w", err)
		}
		id := idPrefix + hex.EncodeToString(b)
		if !exists(id) {
			return id, nil
		}
	}
	return "", fmt.Errorf("Failed to generate unique ID after %d attempts - task list may be too large", idRetries)
}

// NormalizeID converts a task ID to lowercase for case-insensitive matching.
func NormalizeID(id string) string {
	return strings.ToLower(id)
}

// TransitionResult holds the old and new status after a successful transition,
// enabling output formatting like "tick-a3f2b7: open -> in_progress".
type TransitionResult struct {
	OldStatus Status
	NewStatus Status
}

// validTransitions maps each command to its allowed (from -> to) status pairs.
var validTransitions = map[string]map[Status]Status{
	"start":  {StatusOpen: StatusInProgress},
	"done":   {StatusOpen: StatusDone, StatusInProgress: StatusDone},
	"cancel": {StatusOpen: StatusCancelled, StatusInProgress: StatusCancelled},
	"reopen": {StatusDone: StatusOpen, StatusCancelled: StatusOpen},
}

// Transition applies a status transition to the given task based on the command.
// Valid commands are "start", "done", "cancel", and "reopen". On success the
// task's status, updated, and closed fields are mutated and a TransitionResult
// is returned. On failure the task is not modified and an error is returned.
func Transition(t *Task, command string) (TransitionResult, error) {
	transitions, ok := validTransitions[command]
	if !ok {
		return TransitionResult{}, fmt.Errorf("Cannot %s task %s \u2014 status is '%s'", command, t.ID, t.Status)
	}

	newStatus, ok := transitions[t.Status]
	if !ok {
		return TransitionResult{}, fmt.Errorf("Cannot %s task %s \u2014 status is '%s'", command, t.ID, t.Status)
	}

	oldStatus := t.Status
	now := time.Now().UTC().Truncate(time.Second)

	t.Status = newStatus
	t.Updated = now

	switch newStatus {
	case StatusDone, StatusCancelled:
		t.Closed = &now
	case StatusOpen:
		t.Closed = nil
	}

	return TransitionResult{OldStatus: oldStatus, NewStatus: newStatus}, nil
}

// ValidateTitle validates and normalizes a task title. It trims whitespace,
// then rejects empty titles, titles exceeding 500 characters, and titles
// containing newlines. Returns the trimmed title or an error.
func ValidateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("title is required and cannot be empty")
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return "", fmt.Errorf("title exceeds maximum length of %d characters", maxTitleLen)
	}
	if strings.ContainsAny(title, "\n\r") {
		return "", fmt.Errorf("title must be a single line (no newlines)")
	}
	return title, nil
}

// ValidatePriority checks that a priority value is within the valid range 0-4.
func ValidatePriority(priority int) error {
	if priority < 0 || priority > 4 {
		return fmt.Errorf("priority must be between 0 and 4, got %d", priority)
	}
	return nil
}

// ValidateBlockedBy checks that none of the blocked_by IDs reference the task itself.
// Comparison is case-insensitive.
func ValidateBlockedBy(taskID string, blockedBy []string) error {
	normalizedID := NormalizeID(taskID)
	for _, dep := range blockedBy {
		if NormalizeID(dep) == normalizedID {
			return fmt.Errorf("task %s cannot be blocked by itself", taskID)
		}
	}
	return nil
}

// ValidateParent checks that the parent ID does not reference the task itself.
// Comparison is case-insensitive. An empty parent is valid.
func ValidateParent(taskID string, parent string) error {
	if parent == "" {
		return nil
	}
	if NormalizeID(parent) == NormalizeID(taskID) {
		return fmt.Errorf("task %s cannot be its own parent", taskID)
	}
	return nil
}
