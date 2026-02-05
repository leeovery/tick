package cli

import (
	"encoding/json"
)

// JSONFormatter implements Formatter for JSON output format.
// Uses snake_case keys, 2-space indentation, and handles null/empty values correctly.
type JSONFormatter struct{}

// jsonTaskRow represents a task row for JSON array marshaling.
type jsonTaskRow struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// jsonTaskDetail represents full task details for JSON marshaling.
// Uses pointers for optional fields that should be omitted when null.
type jsonTaskDetail struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority"`
	Description string            `json:"description"`
	Parent      string            `json:"parent,omitempty"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
	Closed      string            `json:"closed,omitempty"`
	BlockedBy   []jsonRelatedTask `json:"blocked_by"`
	Children    []jsonRelatedTask `json:"children"`
}

// jsonRelatedTask represents a related task (blocker or child) for JSON marshaling.
type jsonRelatedTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// jsonStats represents statistics for JSON marshaling.
type jsonStats struct {
	Total      int                 `json:"total"`
	ByStatus   jsonStatusCounts    `json:"by_status"`
	ByPriority []jsonPriorityCount `json:"by_priority"`
}

// jsonStatusCounts holds status counts for JSON marshaling.
type jsonStatusCounts struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Cancelled  int `json:"cancelled"`
	Ready      int `json:"ready"`
	Blocked    int `json:"blocked"`
}

// jsonPriorityCount holds a priority count for JSON marshaling.
type jsonPriorityCount struct {
	Priority int `json:"priority"`
	Count    int `json:"count"`
}

// jsonTransition represents a status transition for JSON marshaling.
type jsonTransition struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// jsonDepChange represents a dependency change for JSON marshaling.
type jsonDepChange struct {
	Action    string `json:"action"`
	TaskID    string `json:"task_id"`
	BlockedBy string `json:"blocked_by"`
}

// jsonMessage represents a simple message for JSON marshaling.
type jsonMessage struct {
	Message string `json:"message"`
}

// FormatTaskList formats a list of tasks as a JSON array.
// Empty list produces [] not null.
func (f *JSONFormatter) FormatTaskList(data *TaskListData) string {
	// Initialize empty slice to ensure [] output, not null
	rows := make([]jsonTaskRow, 0)

	if data != nil && len(data.Tasks) > 0 {
		rows = make([]jsonTaskRow, len(data.Tasks))
		for i, task := range data.Tasks {
			rows[i] = jsonTaskRow{
				ID:       task.ID,
				Title:    task.Title,
				Status:   task.Status,
				Priority: task.Priority,
			}
		}
	}

	result, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "[]"
	}

	return string(result)
}

// FormatTaskDetail formats full task details as a JSON object.
// Omits parent/closed when empty. Always includes blocked_by/children as arrays.
// Description is always present (empty string, not null).
func (f *JSONFormatter) FormatTaskDetail(data *TaskDetailData) string {
	if data == nil {
		return "{}"
	}

	// Convert blocked_by - initialize as empty slice to ensure [] not null
	blockedBy := make([]jsonRelatedTask, 0)
	for _, b := range data.BlockedBy {
		blockedBy = append(blockedBy, jsonRelatedTask{
			ID:     b.ID,
			Title:  b.Title,
			Status: b.Status,
		})
	}

	// Convert children - initialize as empty slice to ensure [] not null
	children := make([]jsonRelatedTask, 0)
	for _, c := range data.Children {
		children = append(children, jsonRelatedTask{
			ID:     c.ID,
			Title:  c.Title,
			Status: c.Status,
		})
	}

	detail := jsonTaskDetail{
		ID:          data.ID,
		Title:       data.Title,
		Status:      data.Status,
		Priority:    data.Priority,
		Description: data.Description, // Always present, even if empty
		Parent:      data.Parent,      // omitempty handles omission
		Created:     data.Created,
		Updated:     data.Updated,
		Closed:      data.Closed, // omitempty handles omission
		BlockedBy:   blockedBy,
		Children:    children,
	}

	result, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(result)
}

// FormatTransition formats a status transition as a JSON object.
// Fields: id, from, to.
func (f *JSONFormatter) FormatTransition(taskID, oldStatus, newStatus string) string {
	transition := jsonTransition{
		ID:   taskID,
		From: oldStatus,
		To:   newStatus,
	}

	result, err := json.MarshalIndent(transition, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(result)
}

// FormatDepChange formats a dependency change as a JSON object.
// Fields: action, task_id, blocked_by.
func (f *JSONFormatter) FormatDepChange(action, taskID, blockedByID string) string {
	change := jsonDepChange{
		Action:    action,
		TaskID:    taskID,
		BlockedBy: blockedByID,
	}

	result, err := json.MarshalIndent(change, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(result)
}

// FormatStats formats statistics as a nested JSON object.
// Structure: total, by_status (nested object), by_priority (array of 5 entries).
func (f *JSONFormatter) FormatStats(data *StatsData) string {
	if data == nil {
		return "{}"
	}

	// Build by_priority - always 5 entries
	byPriority := make([]jsonPriorityCount, 0, 5)
	for _, pc := range data.ByPriority {
		byPriority = append(byPriority, jsonPriorityCount{
			Priority: pc.Priority,
			Count:    pc.Count,
		})
	}

	stats := jsonStats{
		Total: data.Total,
		ByStatus: jsonStatusCounts{
			Open:       data.Open,
			InProgress: data.InProgress,
			Done:       data.Done,
			Cancelled:  data.Cancelled,
			Ready:      data.Ready,
			Blocked:    data.Blocked,
		},
		ByPriority: byPriority,
	}

	result, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(result)
}

// FormatMessage formats a simple message as a JSON object.
// Field: message.
func (f *JSONFormatter) FormatMessage(msg string) string {
	message := jsonMessage{
		Message: msg,
	}

	result, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(result)
}
