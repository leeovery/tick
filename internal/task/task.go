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

// ValidateDependency checks that adding newBlockedByID as a blocker for taskID
// does not create a cycle or violate the child-blocked-by-parent rule.
// It takes the full set of tasks to traverse the dependency graph.
// This is a pure validation function with no I/O.
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
	normalizedTaskID := NormalizeID(taskID)
	normalizedBlockedByID := NormalizeID(newBlockedByID)

	// Self-reference is a trivial cycle
	if normalizedTaskID == normalizedBlockedByID {
		return fmt.Errorf("Cannot add dependency - creates cycle: %s \u2192 %s", taskID, taskID)
	}

	// Build lookup map: id -> Task
	taskMap := make(map[string]Task, len(tasks))
	for _, t := range tasks {
		taskMap[NormalizeID(t.ID)] = t
	}

	// Child-blocked-by-parent check
	if task, ok := taskMap[normalizedTaskID]; ok {
		if NormalizeID(task.Parent) == normalizedBlockedByID {
			return fmt.Errorf("Cannot add dependency - %s cannot be blocked by its parent %s", taskID, newBlockedByID)
		}
	}

	// Cycle detection: DFS from newBlockedByID following blocked_by edges
	// to see if we can reach taskID. If so, adding the edge taskID -> newBlockedByID creates a cycle.
	// We also track the path to reconstruct the full cycle.
	path, hasCycle := detectCycle(taskMap, normalizedTaskID, normalizedBlockedByID)
	if hasCycle {
		// Build path string: taskID -> newBlockedByID -> ... -> taskID
		parts := make([]string, 0, len(path)+1)
		parts = append(parts, taskID)
		for _, id := range path {
			// Use original-cased ID from the task map
			if t, ok := taskMap[id]; ok {
				parts = append(parts, t.ID)
			} else {
				parts = append(parts, id)
			}
		}
		parts = append(parts, taskID)
		return fmt.Errorf("Cannot add dependency - creates cycle: %s", strings.Join(parts, " \u2192 "))
	}

	return nil
}

// detectCycle performs a DFS from startID following blocked_by edges
// to determine if targetID is reachable. If so, it returns the path from
// startID to targetID (inclusive of startID, exclusive of targetID) and true.
func detectCycle(taskMap map[string]Task, targetID, startID string) ([]string, bool) {
	type frame struct {
		id   string
		path []string
	}

	visited := make(map[string]bool)
	stack := []frame{{id: startID, path: []string{startID}}}

	for len(stack) > 0 {
		// Pop
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current.id] {
			continue
		}
		visited[current.id] = true

		task, ok := taskMap[current.id]
		if !ok {
			continue
		}

		for _, dep := range task.BlockedBy {
			normalizedDep := NormalizeID(dep)
			if normalizedDep == targetID {
				// Found target - return the path
				return current.path, true
			}
			if !visited[normalizedDep] {
				newPath := make([]string, len(current.path)+1)
				copy(newPath, current.path)
				newPath[len(current.path)] = normalizedDep
				stack = append(stack, frame{id: normalizedDep, path: newPath})
			}
		}
	}

	return nil, false
}

// ValidateDependencies validates multiple blocked_by IDs sequentially,
// failing on the first error. This is a pure validation function with no I/O.
func ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error {
	for _, blockedByID := range blockedByIDs {
		if err := ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return err
		}
	}
	return nil
}

// optString extracts the description from TaskOptions, returning empty string if opts is nil.
func optString(opts *TaskOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Description
}

// validTransitions maps each command to a set of valid source statuses and the resulting target status.
var validTransitions = map[string]struct {
	from []Status
	to   Status
}{
	"start":  {from: []Status{StatusOpen}, to: StatusInProgress},
	"done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
	"cancel": {from: []Status{StatusOpen, StatusInProgress}, to: StatusCancelled},
	"reopen": {from: []Status{StatusDone, StatusCancelled}, to: StatusOpen},
}

// Transition applies a status transition to the given task by command name.
// Valid commands are: start, done, cancel, reopen.
// On success it mutates the task's Status, Updated, and Closed fields,
// and returns the old and new status. On failure it returns an error
// without modifying the task.
func Transition(task *Task, command string) (oldStatus Status, newStatus Status, err error) {
	transition, ok := validTransitions[command]
	if !ok {
		return "", "", fmt.Errorf("unknown command: %s", command)
	}

	currentStatus := task.Status
	allowed := false
	for _, s := range transition.from {
		if currentStatus == s {
			allowed = true
			break
		}
	}

	if !allowed {
		return "", "", fmt.Errorf("Cannot %s task %s \u2014 status is '%s'", command, task.ID, currentStatus)
	}

	now := time.Now().UTC().Truncate(time.Second)

	task.Status = transition.to
	task.Updated = now

	switch transition.to {
	case StatusDone, StatusCancelled:
		task.Closed = &now
	case StatusOpen:
		// reopen clears closed
		task.Closed = nil
	}

	return currentStatus, transition.to, nil
}
