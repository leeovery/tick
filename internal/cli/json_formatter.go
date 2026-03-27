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

// jsonCascadeTransition represents the primary transition in cascade JSON output.
type jsonCascadeTransition struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// jsonCascadeEntry represents a cascaded status change in JSON output.
type jsonCascadeEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// jsonCascadeResult represents the full cascade result in JSON output.
type jsonCascadeResult struct {
	Transition jsonCascadeTransition `json:"transition"`
	Cascaded   []jsonCascadeEntry    `json:"cascaded"`
}

// FormatCascadeTransition renders a cascade transition as structured JSON.
// cascaded is always [] not null.
func (f *JSONFormatter) FormatCascadeTransition(result CascadeResult) string {
	if result.TaskID == "" {
		return ""
	}
	cascaded := make([]jsonCascadeEntry, 0, len(result.Cascaded))
	for _, c := range result.Cascaded {
		cascaded = append(cascaded, jsonCascadeEntry{
			ID:    c.ID,
			Title: c.Title,
			From:  c.OldStatus,
			To:    c.NewStatus,
		})
	}

	return marshalIndentJSON(jsonCascadeResult{
		Transition: jsonCascadeTransition{
			ID:   result.TaskID,
			From: result.OldStatus,
			To:   result.NewStatus,
		},
		Cascaded: cascaded,
	})
}

// jsonDepTreeTask represents a task in dep tree JSON output.
type jsonDepTreeTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// jsonDepTreeNode represents a node in the dep tree with nested children.
type jsonDepTreeNode struct {
	Task     jsonDepTreeTask   `json:"task"`
	Children []jsonDepTreeNode `json:"children"`
}

// jsonDepTreeFull represents the full graph mode JSON output.
type jsonDepTreeFull struct {
	Mode    string            `json:"mode"`
	Roots   []jsonDepTreeNode `json:"roots"`
	Chains  int               `json:"chains"`
	Longest int               `json:"longest"`
	Blocked int               `json:"blocked"`
}

// jsonDepTreeFocused represents the focused mode JSON output.
// BlockedBy and Blocks use omitempty to omit empty directions entirely.
// Message is included when the target has no dependencies.
type jsonDepTreeFocused struct {
	Mode      string            `json:"mode"`
	Target    jsonDepTreeTask   `json:"target"`
	BlockedBy []jsonDepTreeNode `json:"blocked_by,omitempty"`
	Blocks    []jsonDepTreeNode `json:"blocks,omitempty"`
	Message   string            `json:"message,omitempty"`
}

// toJSONDepTreeNodes recursively converts []DepTreeNode to []jsonDepTreeNode.
// Leaf nodes get an empty non-nil children slice to render as [] not null.
func toJSONDepTreeNodes(nodes []DepTreeNode) []jsonDepTreeNode {
	result := make([]jsonDepTreeNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, jsonDepTreeNode{
			Task: jsonDepTreeTask{
				ID:     n.Task.ID,
				Title:  n.Task.Title,
				Status: n.Task.Status,
			},
			Children: toJSONDepTreeNodes(n.Children),
		})
	}
	return result
}

// FormatDepTree renders a dependency tree as structured JSON.
// Full graph: {mode, roots, chains, longest, blocked}.
// Focused: {mode, target, blocked_by?, blocks?, message?} with omitempty on directions.
// Message-only (no target): {message}.
func (f *JSONFormatter) FormatDepTree(result DepTreeResult) string {
	if result.Target != nil {
		return f.formatFocusedDepTreeJSON(result)
	}

	if result.Message != "" {
		return marshalIndentJSON(jsonMessage{Message: result.Message})
	}

	return f.formatFullDepTreeJSON(result)
}

// formatFullDepTreeJSON renders the full graph as nested JSON.
func (f *JSONFormatter) formatFullDepTreeJSON(result DepTreeResult) string {
	return marshalIndentJSON(jsonDepTreeFull{
		Mode:    "full",
		Roots:   toJSONDepTreeNodes(result.Roots),
		Chains:  result.ChainCount,
		Longest: result.LongestChain,
		Blocked: result.BlockedCount,
	})
}

// formatFocusedDepTreeJSON renders focused mode as JSON with optional directions.
// When both BlockedBy and Blocks are empty, includes the message field.
func (f *JSONFormatter) formatFocusedDepTreeJSON(result DepTreeResult) string {
	obj := jsonDepTreeFocused{
		Mode: "focused",
		Target: jsonDepTreeTask{
			ID:     result.Target.ID,
			Title:  result.Target.Title,
			Status: result.Target.Status,
		},
	}

	if len(result.BlockedBy) > 0 {
		obj.BlockedBy = toJSONDepTreeNodes(result.BlockedBy)
	}

	if len(result.Blocks) > 0 {
		obj.Blocks = toJSONDepTreeNodes(result.Blocks)
	}

	if len(result.BlockedBy) == 0 && len(result.Blocks) == 0 {
		obj.Message = result.Message
	}

	return marshalIndentJSON(obj)
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
