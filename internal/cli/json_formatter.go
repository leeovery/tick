package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONFormatter implements the Formatter interface using JSON output.
// All keys use snake_case. Output is 2-space indented via json.MarshalIndent.
type JSONFormatter struct{}

// jsonListRow is the JSON representation of a task in list output.
type jsonListRow struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// jsonRelatedTask is the JSON representation of a related task (blocked_by/children entry).
type jsonRelatedTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// jsonTaskDetail is the JSON representation of a full task detail.
// Parent and Closed use omitempty to be omitted when empty (zero value).
// BlockedBy and Children are initialized to empty slices to produce [] not null.
// Description is always present (no omitempty).
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

// jsonStats is the JSON representation of task statistics.
type jsonStats struct {
	Total      int                `json:"total"`
	ByStatus   jsonByStatus       `json:"by_status"`
	Workflow   jsonWorkflow       `json:"workflow"`
	ByPriority []jsonPriorityEntry `json:"by_priority"`
}

// jsonByStatus holds the status breakdown in stats output.
type jsonByStatus struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Cancelled  int `json:"cancelled"`
}

// jsonWorkflow holds the workflow counts in stats output.
type jsonWorkflow struct {
	Ready   int `json:"ready"`
	Blocked int `json:"blocked"`
}

// jsonPriorityEntry holds a single priority row in stats output.
type jsonPriorityEntry struct {
	Priority int `json:"priority"`
	Count    int `json:"count"`
}

// FormatTaskList renders a list of tasks as a JSON array.
// Empty and nil slices produce [] (never null).
func (f *JSONFormatter) FormatTaskList(w io.Writer, rows []listRow, quiet bool) error {
	if quiet {
		ids := make([]string, len(rows))
		for i, r := range rows {
			ids[i] = r.ID
		}
		return f.writeJSON(w, ids)
	}

	// Initialize to empty slice so nil input produces [] not null
	jsonRows := make([]jsonListRow, 0, len(rows))
	for _, r := range rows {
		jsonRows = append(jsonRows, jsonListRow{
			ID:       r.ID,
			Title:    r.Title,
			Status:   r.Status,
			Priority: r.Priority,
		})
	}

	return f.writeJSON(w, jsonRows)
}

// FormatTaskDetail renders full detail for a single task as a JSON object.
// blocked_by and children are always [] when empty (never null).
// parent and closed are omitted when empty.
// description is always present (empty string when no description).
func (f *JSONFormatter) FormatTaskDetail(w io.Writer, detail TaskDetail) error {
	// Initialize slices to empty (not nil) for proper [] serialization
	blockedBy := make([]jsonRelatedTask, 0, len(detail.BlockedBy))
	for _, b := range detail.BlockedBy {
		blockedBy = append(blockedBy, jsonRelatedTask{
			ID:     b.ID,
			Title:  b.Title,
			Status: b.Status,
		})
	}

	children := make([]jsonRelatedTask, 0, len(detail.Children))
	for _, c := range detail.Children {
		children = append(children, jsonRelatedTask{
			ID:     c.ID,
			Title:  c.Title,
			Status: c.Status,
		})
	}

	obj := jsonTaskDetail{
		ID:          detail.ID,
		Title:       detail.Title,
		Status:      detail.Status,
		Priority:    detail.Priority,
		Description: detail.Description,
		Parent:      detail.Parent,
		Created:     detail.Created,
		Updated:     detail.Updated,
		Closed:      detail.Closed,
		BlockedBy:   blockedBy,
		Children:    children,
	}

	return f.writeJSON(w, obj)
}

// FormatTransition renders a status transition as a JSON object with id, from, to keys.
func (f *JSONFormatter) FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error {
	return f.writeJSON(w, jsonTransition{
		ID:   id,
		From: oldStatus,
		To:   newStatus,
	})
}

// FormatDepChange renders a dependency add/remove result as a JSON object.
// In quiet mode, no output is produced.
func (f *JSONFormatter) FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error {
	if quiet {
		return nil
	}
	return f.writeJSON(w, jsonDepChange{
		Action:    action,
		TaskID:    taskID,
		BlockedBy: blockedByID,
	})
}

// FormatStats renders task statistics as a nested JSON object.
// Contains total, by_status, workflow, and by_priority (always 5 entries).
func (f *JSONFormatter) FormatStats(w io.Writer, stats StatsData) error {
	byPriority := make([]jsonPriorityEntry, 5)
	for i := 0; i < 5; i++ {
		byPriority[i] = jsonPriorityEntry{
			Priority: i,
			Count:    stats.ByPriority[i],
		}
	}

	obj := jsonStats{
		Total: stats.Total,
		ByStatus: jsonByStatus{
			Open:       stats.Open,
			InProgress: stats.InProgress,
			Done:       stats.Done,
			Cancelled:  stats.Cancelled,
		},
		Workflow: jsonWorkflow{
			Ready:   stats.Ready,
			Blocked: stats.Blocked,
		},
		ByPriority: byPriority,
	}

	return f.writeJSON(w, obj)
}

// FormatMessage renders a simple message as a JSON object with a "message" key.
func (f *JSONFormatter) FormatMessage(w io.Writer, msg string) error {
	return f.writeJSON(w, jsonMessage{Message: msg})
}

// writeJSON marshals the value as 2-space indented JSON and writes it to w with a trailing newline.
func (f *JSONFormatter) writeJSON(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}
