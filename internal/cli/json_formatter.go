package cli

import (
	"bytes"
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
	Index   int    `json:"index"`
	Text    string `json:"text"`
	Created string `json:"created"`
}

// jsonMember is one key of a JSON object, paired with the value it carries.
type jsonMember struct {
	key   string
	value any
}

// jsonObject is a JSON object that marshals its members in slice order rather
// than the alphabetical order a map yields.
type jsonObject []jsonMember

func (o jsonObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, member := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(member.key)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(member.value)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// FormatTaskDetail renders a single task as a JSON object, narrowed to
// detail.Fields when it is set, and "" when a selection leaves no key. Of the
// keys it carries: parent/closed are omitted when absent, the list sections are
// arrays, and description is an empty string when not set.
func (f *JSONFormatter) FormatTaskDetail(detail TaskDetail) string {
	obj := taskDetailJSONObject(detail)
	if len(obj) == 0 {
		return ""
	}
	return marshalIndentJSON(obj)
}

// taskDetailJSONObject lays out the detail document's keys in the order it
// renders them, each gated through the selection. A nil selection keeps every
// key and every item of the list sections.
func taskDetailJSONObject(detail TaskDetail) jsonObject {
	t := detail.Task
	fields := detail.Fields

	obj := make(jsonObject, 0, 16)
	add := func(key string, value any) {
		if fields.includes(key) {
			obj = append(obj, jsonMember{key: key, value: value})
		}
	}

	tags, _ := selectedItems(detail.Tags, fields.Positions(fieldTags))
	refs, _ := selectedItems(detail.Refs, fields.Positions(fieldRefs))
	blockedBy, _ := selectedItems(detail.BlockedBy, fields.Positions(fieldBlockedBy))
	children, _ := selectedItems(detail.Children, fields.Positions(fieldChildren))

	add(fieldID, t.ID)
	add(fieldTitle, t.Title)
	add(fieldStatus, string(t.Status))
	add(fieldPriority, t.Priority)
	add(fieldType, t.Type)
	add(fieldTags, toJSONStrings(tags))
	add(fieldRefs, toJSONStrings(refs))
	add(fieldNotes, toJSONNotes(selectedItems(detail.Notes, fields.Positions(fieldNotes))))
	add(fieldDescription, t.Description)
	if t.Parent != "" {
		add(fieldParent, t.Parent)
	}
	add(fieldCreated, task.FormatTimestamp(t.Created))
	add(fieldUpdated, task.FormatTimestamp(t.Updated))
	if closed := jsonClosedTimestamp(t); closed != "" {
		add(fieldClosed, closed)
	}
	add(fieldBlockedBy, toJSONRelated(blockedBy))
	add(fieldChildren, toJSONRelated(children))
	if detail.Changes != nil {
		add("changed", toJSONStatusChanges(detail.Changes.Rows()))
	}

	return obj
}

// jsonClosedTimestamp formats the task's closed time, empty when it is not closed.
func jsonClosedTimestamp(t task.Task) string {
	if t.Closed == nil {
		return ""
	}
	return task.FormatTimestamp(*t.Closed)
}

// toJSONStrings copies a string list, always returning a non-nil empty slice to
// ensure JSON "[]" instead of "null".
func toJSONStrings(values []string) []string {
	result := make([]string, 0, len(values))
	return append(result, values...)
}

// toJSONNotes converts notes to JSON-serializable structs, each carrying the
// position it holds in the whole section. Always returns a non-nil empty slice.
func toJSONNotes(notes []task.Note, positions []int) []jsonNote {
	result := make([]jsonNote, 0, len(notes))
	for i, n := range notes {
		result = append(result, jsonNote{
			Index:   positions[i],
			Text:    n.Text,
			Created: task.FormatTimestamp(n.Created),
		})
	}
	return result
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

// jsonDepChange represents a dependency change in JSON output.
type jsonDepChange struct {
	Action  string `json:"action"`
	TaskID  string `json:"task_id"`
	Blocker string `json:"blocker"`
}

// FormatDepChange renders a dependency add/remove confirmation as a JSON object.
func (f *JSONFormatter) FormatDepChange(action string, taskID string, depID string) string {
	return marshalIndentJSON(jsonDepChange{
		Action:  action,
		TaskID:  taskID,
		Blocker: depID,
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
	for i := range 5 {
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

// jsonStatusChange represents one row of a command's status-change list in JSON output.
type jsonStatusChange struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	From  string `json:"from"`
	To    string `json:"to"`
	Auto  bool   `json:"auto"`
}

// toJSONStatusChanges converts status changes to JSON-serializable structs.
// Always returns a non-nil empty slice to ensure JSON "[]" instead of "null".
func toJSONStatusChanges(changes []StatusChange) []jsonStatusChange {
	result := make([]jsonStatusChange, 0, len(changes))
	for _, c := range changes {
		result = append(result, jsonStatusChange(c))
	}
	return result
}

// jsonChangedList represents every status change a command made in JSON output.
type jsonChangedList struct {
	Changed []jsonStatusChange `json:"changed"`
}

// FormatCascadeTransition renders every status change the command made as one changed list.
func (f *JSONFormatter) FormatCascadeTransition(result CascadeResult) string {
	return marshalIndentJSON(jsonChangedList{Changed: toJSONStatusChanges(result.Changed())})
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
	Trees   []jsonDepTreeNode `json:"trees"`
	Chains  int               `json:"chains"`
	Longest int               `json:"longest"`
	Blocked int               `json:"blocked"`
}

// jsonDepTreeFocused represents the focused mode JSON output.
// Both directions are always present, empty ones rendering as [].
type jsonDepTreeFocused struct {
	Mode      string            `json:"mode"`
	Target    jsonDepTreeTask   `json:"target"`
	BlockedBy []jsonDepTreeNode `json:"blocked_by"`
	Blocks    []jsonDepTreeNode `json:"blocks"`
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
// Full graph: {mode, trees, chains, longest, blocked}.
// Focused: {mode, target, blocked_by, blocks}.
func (f *JSONFormatter) FormatDepTree(result DepTreeResult) string {
	if result.Target != nil {
		return f.formatFocusedDepTreeJSON(result)
	}

	return f.formatFullDepTreeJSON(result)
}

func (f *JSONFormatter) formatFullDepTreeJSON(result DepTreeResult) string {
	return marshalIndentJSON(jsonDepTreeFull{
		Mode:    "full",
		Trees:   toJSONDepTreeNodes(result.Trees),
		Chains:  result.ChainCount,
		Longest: result.LongestChain,
		Blocked: result.BlockedCount,
	})
}

func (f *JSONFormatter) formatFocusedDepTreeJSON(result DepTreeResult) string {
	return marshalIndentJSON(jsonDepTreeFocused{
		Mode: "focused",
		Target: jsonDepTreeTask{
			ID:     result.Target.ID,
			Title:  result.Target.Title,
			Status: result.Target.Status,
		},
		BlockedBy: toJSONDepTreeNodes(result.BlockedBy),
		Blocks:    toJSONDepTreeNodes(result.Blocks),
	})
}

// marshalIndentJSON marshals v as 2-space indented JSON.
// Returns "null" on marshal failure (should not happen with controlled types).
func marshalIndentJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "null"
	}
	return string(b)
}
