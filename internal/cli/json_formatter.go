package cli

import (
	"encoding/json"

	"github.com/leeovery/tick/internal/task"
)

// JSONFormatter renders CLI output as standard JSON for compatibility and debugging.
// All keys use snake_case. Output is 2-space indented via json.MarshalIndent.
type JSONFormatter struct{}

// Compile-time interface verification.
var _ Formatter = (*JSONFormatter)(nil)

// jsonTaskListItem represents a task in list output.
type jsonTaskListItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	Type     string `json:"type"`
}

// FormatTaskList renders a list of tasks as a JSON array.
// Empty input produces "[]", never "null".
func (f *JSONFormatter) FormatTaskList(tasks []task.Task) string {
	items := make([]jsonTaskListItem, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, jsonTaskListItem{
			ID:       t.ID,
			Title:    t.Title,
			Status:   string(t.Status),
			Priority: t.Priority,
			Type:     t.Type,
		})
	}
	return marshalIndentJSON(items)
}

// jsonRelatedTask represents a related task (blocker or child) in JSON output.
type jsonRelatedTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// jsonNote represents a note in JSON output.
type jsonNote struct {
	Text    string `json:"text"`
	Created string `json:"created"`
}

// jsonTaskDetail represents the full task detail in JSON output.
// parent and closed use omitempty to omit when zero/nil.
// blocked_by, children, tags, refs, and notes are always present as arrays.
// description is always present (empty string, not null/omitted).
type jsonTaskDetail struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority"`
	Type        string            `json:"type"`
	Tags        []string          `json:"tags"`
	Refs        []string          `json:"refs"`
	Notes       []jsonNote        `json:"notes"`
	Description string            `json:"description"`
	Parent      string            `json:"parent,omitempty"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
	Closed      string            `json:"closed,omitempty"`
	BlockedBy   []jsonRelatedTask `json:"blocked_by"`
	Children    []jsonRelatedTask `json:"children"`
}

// FormatTaskDetail renders a single task with full details as a JSON object.
// parent/closed are omitted when absent. blocked_by/children are always present as arrays.
// description is always present (empty string when not set).
func (f *JSONFormatter) FormatTaskDetail(detail TaskDetail) string {
	t := detail.Task

	var closedStr string
	if t.Closed != nil {
		closedStr = task.FormatTimestamp(*t.Closed)
	}

	tags := make([]string, 0, len(detail.Tags))
	tags = append(tags, detail.Tags...)

	refs := make([]string, 0, len(detail.Refs))
	refs = append(refs, detail.Refs...)

	notes := make([]jsonNote, 0, len(detail.Notes))
	for _, n := range detail.Notes {
		notes = append(notes, jsonNote{
			Text:    n.Text,
			Created: task.FormatTimestamp(n.Created),
		})
	}

	obj := jsonTaskDetail{
		ID:          t.ID,
		Title:       t.Title,
		Status:      string(t.Status),
		Priority:    t.Priority,
		Type:        t.Type,
		Tags:        tags,
		Refs:        refs,
		Notes:       notes,
		Description: t.Description,
		Parent:      t.Parent,
		Created:     task.FormatTimestamp(t.Created),
		Updated:     task.FormatTimestamp(t.Updated),
		Closed:      closedStr,
		BlockedBy:   toJSONRelated(detail.BlockedBy),
		Children:    toJSONRelated(detail.Children),
	}

	return marshalIndentJSON(obj)
}

// toJSONRelated converts a slice of RelatedTask to JSON-serializable structs.
// Always returns a non-nil empty slice to ensure JSON "[]" instead of "null".
func toJSONRelated(related []RelatedTask) []jsonRelatedTask {
	result := make([]jsonRelatedTask, 0, len(related))
	for _, r := range related {
		result = append(result, jsonRelatedTask(r))
	}
	return result
}

// jsonTransition represents a status transition in JSON output.
type jsonTransition struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// FormatTransition renders a status transition as a JSON object with id, from, to.
func (f *JSONFormatter) FormatTransition(id string, oldStatus string, newStatus string) string {
	return marshalIndentJSON(jsonTransition{
		ID:   id,
		From: oldStatus,
		To:   newStatus,
	})
}

// jsonDepChange represents a dependency change in JSON output.
type jsonDepChange struct {
	Action    string `json:"action"`
	TaskID    string `json:"task_id"`
	BlockedBy string `json:"blocked_by"`
}

// FormatDepChange renders a dependency add/remove confirmation as a JSON object.
func (f *JSONFormatter) FormatDepChange(action string, taskID string, depID string) string {
	return marshalIndentJSON(jsonDepChange{
		Action:    action,
		TaskID:    taskID,
		BlockedBy: depID,
	})
}

// jsonStatusCounts represents the by_status section in stats output.
type jsonStatusCounts struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Cancelled  int `json:"cancelled"`
}

// jsonWorkflow represents the workflow section in stats output.
type jsonWorkflow struct {
	Ready   int `json:"ready"`
	Blocked int `json:"blocked"`
}

// jsonPriorityEntry represents a single priority entry in by_priority array.
type jsonPriorityEntry struct {
	Priority int `json:"priority"`
	Count    int `json:"count"`
}

// jsonStats represents the full stats output as a nested JSON object.
type jsonStats struct {
	Total      int                 `json:"total"`
	ByStatus   jsonStatusCounts    `json:"by_status"`
	Workflow   jsonWorkflow        `json:"workflow"`
	ByPriority []jsonPriorityEntry `json:"by_priority"`
}

// FormatStats renders task statistics as a nested JSON object with
// total, by_status, workflow, and by_priority sections.
// by_priority always contains 5 entries (P0-P4), even when counts are zero.
func (f *JSONFormatter) FormatStats(stats Stats) string {
	priorities := make([]jsonPriorityEntry, 5)
	for i := 0; i < 5; i++ {
		priorities[i] = jsonPriorityEntry{
			Priority: i,
			Count:    stats.ByPriority[i],
		}
	}

	obj := jsonStats{
		Total: stats.Total,
		ByStatus: jsonStatusCounts{
			Open:       stats.Open,
			InProgress: stats.InProgress,
			Done:       stats.Done,
			Cancelled:  stats.Cancelled,
		},
		Workflow: jsonWorkflow{
			Ready:   stats.Ready,
			Blocked: stats.Blocked,
		},
		ByPriority: priorities,
	}

	return marshalIndentJSON(obj)
}

// jsonMessage represents a general-purpose message in JSON output.
type jsonMessage struct {
	Message string `json:"message"`
}

// FormatMessage renders a general-purpose message as a JSON object with a "message" key.
func (f *JSONFormatter) FormatMessage(msg string) string {
	return marshalIndentJSON(jsonMessage{Message: msg})
}

// jsonRemovedTask represents a removed task in JSON output.
type jsonRemovedTask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// jsonRemovalResult represents the removal operation result in JSON output.
type jsonRemovalResult struct {
	Removed     []jsonRemovedTask `json:"removed"`
	DepsUpdated []string          `json:"deps_updated"`
}

// FormatRemoval renders a removal result as a JSON object with removed array and deps_updated array.
// Both arrays are always [] not null when empty.
func (f *JSONFormatter) FormatRemoval(result RemovalResult) string {
	removed := make([]jsonRemovedTask, 0, len(result.Removed))
	for _, r := range result.Removed {
		removed = append(removed, jsonRemovedTask(r))
	}
	depsUpdated := make([]string, 0, len(result.DepsUpdated))
	depsUpdated = append(depsUpdated, result.DepsUpdated...)
	return marshalIndentJSON(jsonRemovalResult{
		Removed:     removed,
		DepsUpdated: depsUpdated,
	})
}

// marshalIndentJSON marshals v as 2-space indented JSON.
// Returns "null" on marshal failure (should not happen with controlled types).
func marshalIndentJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "null"
	}
	return string(b)
}
