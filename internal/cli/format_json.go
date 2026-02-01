package cli

import (
	"encoding/json"
	"io"
)

// JSONFormatter formats output as indented JSON for compatibility and debugging.
type JSONFormatter struct{}

// jsonTaskListItem is the JSON representation of a task list row.
type jsonTaskListItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// FormatTaskList formats tasks as a JSON array. Empty list produces [].
func (f *JSONFormatter) FormatTaskList(w io.Writer, tasks []TaskListItem) error {
	items := make([]jsonTaskListItem, len(tasks))
	for i, t := range tasks {
		items[i] = jsonTaskListItem{ID: t.ID, Title: t.Title, Status: t.Status, Priority: t.Priority}
	}
	return jsonWrite(w, items)
}

// jsonRelatedTask is the JSON representation of a related task.
type jsonRelatedTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// jsonTaskDetail is the JSON representation of full task detail.
// parent and closed use omitempty to omit when zero-valued.
// blocked_by, children, and description are always present.
type jsonTaskDetail struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority"`
	Description string            `json:"description"`
	Parent      *jsonRelatedTask  `json:"parent,omitempty"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
	Closed      string            `json:"closed,omitempty"`
	BlockedBy   []jsonRelatedTask `json:"blocked_by"`
	Children    []jsonRelatedTask `json:"children"`
}

// FormatTaskDetail formats a task as a JSON object.
func (f *JSONFormatter) FormatTaskDetail(w io.Writer, d TaskDetail) error {
	detail := jsonTaskDetail{
		ID:          d.ID,
		Title:       d.Title,
		Status:      d.Status,
		Priority:    d.Priority,
		Description: d.Description,
		Created:     d.Created,
		Updated:     d.Updated,
		Closed:      d.Closed,
		BlockedBy:   make([]jsonRelatedTask, len(d.BlockedBy)),
		Children:    make([]jsonRelatedTask, len(d.Children)),
	}

	if d.Parent != nil {
		detail.Parent = &jsonRelatedTask{ID: d.Parent.ID, Title: d.Parent.Title, Status: d.Parent.Status}
	}

	for i, b := range d.BlockedBy {
		detail.BlockedBy[i] = jsonRelatedTask{ID: b.ID, Title: b.Title, Status: b.Status}
	}
	for i, c := range d.Children {
		detail.Children[i] = jsonRelatedTask{ID: c.ID, Title: c.Title, Status: c.Status}
	}

	return jsonWrite(w, detail)
}

// FormatTransition formats a status transition as a JSON object.
func (f *JSONFormatter) FormatTransition(w io.Writer, data TransitionData) error {
	obj := struct {
		ID   string `json:"id"`
		From string `json:"from"`
		To   string `json:"to"`
	}{
		ID: data.ID, From: data.OldStatus, To: data.NewStatus,
	}
	return jsonWrite(w, obj)
}

// FormatDepChange formats a dependency change as a JSON object.
func (f *JSONFormatter) FormatDepChange(w io.Writer, data DepChangeData) error {
	obj := struct {
		Action    string `json:"action"`
		TaskID    string `json:"task_id"`
		BlockedBy string `json:"blocked_by"`
	}{
		Action: data.Action, TaskID: data.TaskID, BlockedBy: data.BlockedBy,
	}
	return jsonWrite(w, obj)
}

// jsonPriorityEntry is a priority count row in stats output.
type jsonPriorityEntry struct {
	Priority int `json:"priority"`
	Count    int `json:"count"`
}

// FormatStats formats statistics as a nested JSON object.
func (f *JSONFormatter) FormatStats(w io.Writer, data StatsData) error {
	bp := make([]jsonPriorityEntry, 5)
	for i := 0; i < 5; i++ {
		bp[i] = jsonPriorityEntry{Priority: i, Count: data.ByPriority[i]}
	}

	obj := struct {
		Total      int                 `json:"total"`
		Open       int                 `json:"open"`
		InProgress int                 `json:"in_progress"`
		Done       int                 `json:"done"`
		Cancelled  int                 `json:"cancelled"`
		Ready      int                 `json:"ready"`
		Blocked    int                 `json:"blocked"`
		ByPriority []jsonPriorityEntry `json:"by_priority"`
	}{
		Total: data.Total, Open: data.Open, InProgress: data.InProgress,
		Done: data.Done, Cancelled: data.Cancelled,
		Ready: data.Ready, Blocked: data.Blocked,
		ByPriority: bp,
	}
	return jsonWrite(w, obj)
}

// FormatMessage formats a simple message as a JSON object.
func (f *JSONFormatter) FormatMessage(w io.Writer, message string) error {
	obj := struct {
		Message string `json:"message"`
	}{Message: message}
	return jsonWrite(w, obj)
}

// jsonWrite encodes v as 2-space indented JSON and writes to w.
func jsonWrite(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
