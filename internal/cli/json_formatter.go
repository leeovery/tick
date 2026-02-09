package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONFormatter implements the Formatter interface using JSON output format.
// It produces standard JSON with 2-space indentation, snake_case keys, and
// correct null/empty handling for tool integration and debugging.
type JSONFormatter struct{}

// jsonListItem is the JSON representation of a single task in a list.
type jsonListItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// jsonRelatedTask is the JSON representation of a related task (blocked_by or children).
type jsonRelatedTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// jsonTaskDetail is the JSON representation of full task details.
type jsonTaskDetail struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority"`
	Parent      string            `json:"parent,omitempty"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
	Closed      string            `json:"closed,omitempty"`
	Description string            `json:"description"`
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

// jsonPriorityEntry is a single entry in the by_priority array.
type jsonPriorityEntry struct {
	Priority int `json:"priority"`
	Count    int `json:"count"`
}

// jsonStats is the JSON representation of task statistics.
type jsonStats struct {
	Total      int                 `json:"total"`
	ByStatus   jsonByStatus        `json:"by_status"`
	Workflow   jsonWorkflow        `json:"workflow"`
	ByPriority []jsonPriorityEntry `json:"by_priority"`
}

// jsonByStatus holds the status breakdown for stats.
type jsonByStatus struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Cancelled  int `json:"cancelled"`
}

// jsonWorkflow holds the workflow breakdown for stats.
type jsonWorkflow struct {
	Ready   int `json:"ready"`
	Blocked int `json:"blocked"`
}

// writeJSON marshals v as 2-space indented JSON and writes it to w.
func writeJSON(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

// FormatTaskList renders a list of tasks as a JSON array.
// Empty lists produce `[]`, never `null`.
func (f *JSONFormatter) FormatTaskList(w io.Writer, rows []TaskRow) error {
	// Initialize to empty slice to avoid null in JSON output.
	items := make([]jsonListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonListItem(r))
	}

	return writeJSON(w, items)
}

// FormatTaskDetail renders full details of a single task as a JSON object.
// blocked_by/children are always arrays (never null). parent/closed are omitted
// when empty. description is always present as a string.
func (f *JSONFormatter) FormatTaskDetail(w io.Writer, d *showData) error {
	// Initialize blocked_by to empty slice to ensure [] not null.
	blockedBy := make([]jsonRelatedTask, 0, len(d.blockedBy))
	for _, rt := range d.blockedBy {
		blockedBy = append(blockedBy, jsonRelatedTask{
			ID:     rt.id,
			Title:  rt.title,
			Status: rt.status,
		})
	}

	// Initialize children to empty slice to ensure [] not null.
	children := make([]jsonRelatedTask, 0, len(d.children))
	for _, rt := range d.children {
		children = append(children, jsonRelatedTask{
			ID:     rt.id,
			Title:  rt.title,
			Status: rt.status,
		})
	}

	detail := jsonTaskDetail{
		ID:          d.id,
		Title:       d.title,
		Status:      d.status,
		Priority:    d.priority,
		Parent:      d.parent,
		Created:     d.created,
		Updated:     d.updated,
		Closed:      d.closed,
		Description: d.description,
		BlockedBy:   blockedBy,
		Children:    children,
	}

	return writeJSON(w, detail)
}

// FormatTransition renders a status transition result as a JSON object with
// id, from, and to keys.
func (f *JSONFormatter) FormatTransition(w io.Writer, data *TransitionData) error {
	return writeJSON(w, jsonTransition{
		ID:   data.ID,
		From: data.OldStatus,
		To:   data.NewStatus,
	})
}

// FormatDepChange renders a dependency change confirmation as a JSON object
// with action, task_id, and blocked_by keys.
func (f *JSONFormatter) FormatDepChange(w io.Writer, data *DepChangeData) error {
	return writeJSON(w, jsonDepChange{
		Action:    data.Action,
		TaskID:    data.TaskID,
		BlockedBy: data.BlockedByID,
	})
}

// FormatStats renders task statistics as a nested JSON object with total,
// by_status, workflow, and by_priority sections. by_priority always contains
// 5 entries (P0-P4).
func (f *JSONFormatter) FormatStats(w io.Writer, d *StatsData) error {
	byPriority := make([]jsonPriorityEntry, 5)
	for i := 0; i < 5; i++ {
		byPriority[i] = jsonPriorityEntry{
			Priority: i,
			Count:    d.ByPriority[i],
		}
	}

	stats := jsonStats{
		Total: d.Total,
		ByStatus: jsonByStatus{
			Open:       d.Open,
			InProgress: d.InProgress,
			Done:       d.Done,
			Cancelled:  d.Cancelled,
		},
		Workflow: jsonWorkflow{
			Ready:   d.Ready,
			Blocked: d.Blocked,
		},
		ByPriority: byPriority,
	}

	return writeJSON(w, stats)
}

// FormatMessage writes a message as a JSON object with a "message" key.
func (f *JSONFormatter) FormatMessage(w io.Writer, msg string) {
	writeJSON(w, jsonMessage{Message: msg}) //nolint:errcheck
}
