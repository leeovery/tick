package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/task"
)

// JSONFormatter implements the Formatter interface using JSON output format.
// It produces indented JSON with snake_case keys for compatibility and debugging.
type JSONFormatter struct{}

// Compile-time interface verification.
var _ Formatter = &JSONFormatter{}

// jsonTaskRow is the JSON representation of a task list row.
type jsonTaskRow struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// jsonRelatedTask is the JSON representation of a related task (blocker or child).
type jsonRelatedTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// jsonTaskDetail is the JSON representation of full task details.
// Parent and Closed use pointer types so they are omitted when nil.
// BlockedBy and Children are initialized to empty slices to produce [] not null.
// Description is always present (empty string, never omitted).
type jsonTaskDetail struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority"`
	Description string            `json:"description"`
	Parent      *string           `json:"parent,omitempty"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
	Closed      *string           `json:"closed,omitempty"`
	BlockedBy   []jsonRelatedTask `json:"blocked_by"`
	Children    []jsonRelatedTask `json:"children"`
}

// jsonStatsData is the JSON representation of task statistics.
type jsonStatsData struct {
	Total      int                 `json:"total"`
	ByStatus   jsonByStatus        `json:"by_status"`
	Workflow   jsonWorkflow        `json:"workflow"`
	ByPriority []jsonPriorityEntry `json:"by_priority"`
}

// jsonByStatus holds status breakdown counts.
type jsonByStatus struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Cancelled  int `json:"cancelled"`
}

// jsonWorkflow holds workflow counts.
type jsonWorkflow struct {
	Ready   int `json:"ready"`
	Blocked int `json:"blocked"`
}

// jsonPriorityEntry holds a single priority count.
type jsonPriorityEntry struct {
	Priority int `json:"priority"`
	Count    int `json:"count"`
}

// jsonTransition is the JSON representation of a status transition.
type jsonTransition struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// jsonDepChange is the JSON representation of a dependency change.
type jsonDepChange struct {
	Action    string `json:"action"`
	TaskID    string `json:"task_id"`
	BlockedBy string `json:"blocked_by"`
}

// jsonMessage is the JSON representation of a simple message.
type jsonMessage struct {
	Message string `json:"message"`
}

// FormatTaskList renders a list of tasks as a JSON array.
// Empty or nil input produces [] (never null).
func (f *JSONFormatter) FormatTaskList(w io.Writer, tasks []TaskRow) error {
	// Ensure empty slice (not nil) to produce [] instead of null
	rows := make([]jsonTaskRow, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, jsonTaskRow{
			ID:       t.ID,
			Title:    t.Title,
			Status:   t.Status,
			Priority: t.Priority,
		})
	}

	return marshalIndentTo(w, rows)
}

// FormatTaskDetail renders full details for a single task as a JSON object.
// blocked_by and children are always [] when empty. parent/closed omitted when null.
// description is always present (empty string when not set).
func (f *JSONFormatter) FormatTaskDetail(w io.Writer, data *showData) error {
	detail := jsonTaskDetail{
		ID:          data.ID,
		Title:       data.Title,
		Status:      data.Status,
		Priority:    data.Priority,
		Description: data.Description,
		Created:     data.Created,
		Updated:     data.Updated,
	}

	// Parent: omit when empty (pointer nil = omitempty)
	if data.Parent != "" {
		parent := data.Parent
		detail.Parent = &parent
	}

	// Closed: omit when empty
	if data.Closed != "" {
		closed := data.Closed
		detail.Closed = &closed
	}

	// Ensure blocked_by is always [] not null
	detail.BlockedBy = make([]jsonRelatedTask, 0, len(data.BlockedBy))
	for _, r := range data.BlockedBy {
		detail.BlockedBy = append(detail.BlockedBy, jsonRelatedTask{
			ID:     r.ID,
			Title:  r.Title,
			Status: r.Status,
		})
	}

	// Ensure children is always [] not null
	detail.Children = make([]jsonRelatedTask, 0, len(data.Children))
	for _, r := range data.Children {
		detail.Children = append(detail.Children, jsonRelatedTask{
			ID:     r.ID,
			Title:  r.Title,
			Status: r.Status,
		})
	}

	return marshalIndentTo(w, detail)
}

// FormatTransition renders a status transition as a JSON object with id, from, to.
func (f *JSONFormatter) FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error {
	return marshalIndentTo(w, jsonTransition{
		ID:   id,
		From: string(oldStatus),
		To:   string(newStatus),
	})
}

// FormatDepChange renders a dependency add/remove confirmation as a JSON object.
func (f *JSONFormatter) FormatDepChange(w io.Writer, action, taskID, blockedByID string) error {
	return marshalIndentTo(w, jsonDepChange{
		Action:    action,
		TaskID:    taskID,
		BlockedBy: blockedByID,
	})
}

// FormatStats renders task statistics as a nested JSON object.
// by_priority always contains 5 entries (priorities 0-4).
func (f *JSONFormatter) FormatStats(w io.Writer, stats interface{}) error {
	sd, ok := stats.(*StatsData)
	if !ok {
		return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
	}

	priorities := make([]jsonPriorityEntry, 5)
	for i := 0; i < 5; i++ {
		priorities[i] = jsonPriorityEntry{
			Priority: i,
			Count:    sd.ByPriority[i],
		}
	}

	data := jsonStatsData{
		Total: sd.Total,
		ByStatus: jsonByStatus{
			Open:       sd.Open,
			InProgress: sd.InProgress,
			Done:       sd.Done,
			Cancelled:  sd.Cancelled,
		},
		Workflow: jsonWorkflow{
			Ready:   sd.Ready,
			Blocked: sd.Blocked,
		},
		ByPriority: priorities,
	}

	return marshalIndentTo(w, data)
}

// FormatMessage renders a simple message as a JSON object with a "message" key.
func (f *JSONFormatter) FormatMessage(w io.Writer, message string) error {
	return marshalIndentTo(w, jsonMessage{Message: message})
}

// marshalIndentTo marshals v as 2-space indented JSON and writes to w with a trailing newline.
func marshalIndentTo(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	_, err = w.Write(append(data, '\n'))
	return err
}
