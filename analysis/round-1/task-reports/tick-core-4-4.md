# Task tick-core-4-4: JSON formatter -- list, show, stats output

## Task Summary

Implement a `JSONFormatter` satisfying the `Formatter` interface, producing `--json` output for compatibility and debugging. Universal interchange format for piping to `jq` and tool integration.

**Required methods:**
- **FormatTaskList**: JSON array. Empty produces `[]`, never `null`.
- **FormatTaskDetail**: Object with all fields. `blocked_by`/`children` always `[]` when empty. `parent`/`closed` omitted when null. `description` always present (empty string).
- **FormatStats**: Nested object: `total`, `by_status`, `workflow`, `by_priority` (always 5 entries).
- **FormatTransition**: `{"id", "from", "to"}`
- **FormatDepChange**: `{"action", "task_id", "blocked_by"}`
- **FormatMessage**: `{"message"}`
- All keys `snake_case`. Use `json.MarshalIndent` (2-space).

**Acceptance Criteria:**
1. Implements full Formatter interface
2. Empty list produces `[]`
3. `blocked_by`/`children` always `[]` when empty
4. `parent`/`closed` omitted when null
5. `description` always present
6. `snake_case` keys throughout
7. Stats nested with 5 priority entries
8. All output valid JSON
9. 2-space indented

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Implements full Formatter interface | PASS -- 6 methods matching V1's `Formatter` signatures (`io.Writer`, error return). No compile-time interface check. | PASS -- 6 methods matching V2's `Formatter` signatures. Explicit `var _ Formatter = &JSONFormatter{}` compile-time check. | PASS -- 6 methods matching V3's `Formatter` signatures (returns `string`). Implicit compile-time check via `var _ Formatter = &JSONFormatter{}` in test. |
| 2. Empty list produces `[]` | PASS -- `make([]jsonTaskListItem, len(tasks))` with `len(tasks)==0` produces empty slice, marshals to `[]` | PASS -- `make([]jsonTaskRow, 0, len(tasks))` ensures non-nil empty slice. Also handles `nil` input explicitly. | PASS -- `make([]jsonTaskRow, 0)` initial empty slice, only populates when `data != nil && len(data.Tasks) > 0`. Also handles `nil` `TaskListData` input. |
| 3. `blocked_by`/`children` always `[]` when empty | PASS -- `make([]jsonRelatedTask, len(d.BlockedBy))` produces empty slice when len is 0 | PASS -- `make([]jsonRelatedTask, 0, len(data.BlockedBy))` ensures non-nil empty slice | PASS -- `make([]jsonRelatedTask, 0)` ensures non-nil empty slice |
| 4. `parent`/`closed` omitted when null | PASS -- `Parent *jsonRelatedTask` with `json:",omitempty"` (pointer nil = omitted); `Closed string` with `json:",omitempty"` (empty string = omitted) | PASS -- `Parent *string` with `json:",omitempty"` (pointer nil = omitted); `Closed *string` with `json:",omitempty"` (pointer nil = omitted) | PASS -- `Parent string` with `json:",omitempty"` (empty string = omitted); `Closed string` with `json:",omitempty"` (empty string = omitted) |
| 5. `description` always present | PASS -- `Description string` with `json:"description"` (no omitempty), always serialized | PASS -- same approach | PASS -- same approach |
| 6. `snake_case` keys throughout | PASS -- all JSON tags use snake_case: `id`, `title`, `status`, `priority`, `blocked_by`, `in_progress`, `by_priority` | PASS -- identical snake_case keys | PASS -- identical snake_case keys |
| 7. Stats nested with 5 priority entries | PARTIAL -- flat structure with `total`, `open`, `in_progress`, `done`, `cancelled`, `ready`, `blocked`, `by_priority`. No `by_status` or `workflow` nesting. All status/workflow fields at top level. Priority always 5 entries (hardcoded loop 0..4). | PASS -- properly nested: `total` at top, `by_status` object (`open`, `in_progress`, `done`, `cancelled`), `workflow` object (`ready`, `blocked`), `by_priority` array. Priority always 5 entries (hardcoded loop 0..4). | PARTIAL -- nested with `total`, `by_status` object (contains `open`, `in_progress`, `done`, `cancelled`, `ready`, `blocked` -- workflow fields mixed into by_status). No separate `workflow` object. Priority entries depend on input `[]PriorityCount` length, not hardcoded to 5. |
| 8. All output valid JSON | PASS -- `json.NewEncoder` with `SetIndent` produces valid JSON | PASS -- `json.MarshalIndent` produces valid JSON | PASS -- `json.MarshalIndent` produces valid JSON |
| 9. 2-space indented | PASS -- `enc.SetIndent("", "  ")` | PASS -- `json.MarshalIndent(v, "", "  ")` | PASS -- `json.MarshalIndent(..., "", "  ")` |

## Implementation Comparison

### Approach

#### File Naming

- **V1**: `format_json.go` / `format_json_test.go` -- follows V1's `format_*.go` naming pattern
- **V2**: `json_formatter.go` / `json_formatter_test.go` -- follows V2's `*_formatter.go` pattern
- **V3**: `json_formatter.go` / `json_formatter_test.go` -- same as V2

#### Formatter Interface Signatures

As established in task 4-1, each version uses a different Formatter interface:

**V1**: `io.Writer` parameter, `error` return, custom data structs:
```go
FormatTaskList(w io.Writer, tasks []TaskListItem) error
FormatTaskDetail(w io.Writer, d TaskDetail) error
FormatStats(w io.Writer, data StatsData) error
```

**V2**: `io.Writer` parameter, `error` return, references existing codebase types + `interface{}` for stats:
```go
FormatTaskList(w io.Writer, tasks []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatStats(w io.Writer, stats interface{}) error
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
```

**V3**: Returns `string`, no `io.Writer`, no `error` return:
```go
FormatTaskList(data *TaskListData) string
FormatTaskDetail(data *TaskDetailData) string
FormatStats(data *StatsData) string
FormatTransition(taskID, oldStatus, newStatus string) string
FormatDepChange(action, taskID, blockedByID string) string
```

#### JSON Serialization Strategy

**V1** uses `json.NewEncoder` with `SetIndent` (streaming encoder):
```go
// format_json.go line 145-149
func jsonWrite(w io.Writer, v any) error {
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    return enc.Encode(v)
}
```
This writes directly to `io.Writer` with a trailing newline (added automatically by `Encode`). A single shared helper is used by all methods.

**V2** uses `json.MarshalIndent` (buffer-then-write):
```go
// json_formatter.go line 223-229
func marshalIndentTo(w io.Writer, v interface{}) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return fmt.Errorf("json marshal failed: %w", err)
    }
    _, err = w.Write(append(data, '\n'))
    return err
}
```
Marshals to `[]byte`, appends newline, then writes. Wraps marshal errors with context. Also a single shared helper.

**V3** uses `json.MarshalIndent` directly in each method, returning `string`:
```go
// json_formatter.go line 101-106 (example from FormatTaskList)
result, err := json.MarshalIndent(rows, "", "  ")
if err != nil {
    return "[]"
}
return string(result)
```
No shared helper -- each method calls `json.MarshalIndent` independently. On error, returns a sensible fallback (`"[]"` for lists, `"{}"` for objects) rather than propagating the error.

#### FormatTaskList -- Empty Slice Handling

All three versions solve the Go nil-slice-marshals-to-`null` gotcha, but differently:

**V1** (`format_json.go` lines 21-27):
```go
func (f *JSONFormatter) FormatTaskList(w io.Writer, tasks []TaskListItem) error {
    items := make([]jsonTaskListItem, len(tasks))
    for i, t := range tasks {
        items[i] = jsonTaskListItem{ID: t.ID, Title: t.Title, Status: t.Status, Priority: t.Priority}
    }
    return jsonWrite(w, items)
}
```
Uses `make([]T, len(tasks))`. When `len(tasks)==0`, this produces a non-nil empty slice `[]jsonTaskListItem{}`. However, if `tasks` is `nil`, `len(tasks)` is 0, so it still works. Note: V1 does NOT test the `nil` input case.

**V2** (`json_formatter.go` lines 104-116):
```go
func (f *JSONFormatter) FormatTaskList(w io.Writer, tasks []TaskRow) error {
    rows := make([]jsonTaskRow, 0, len(tasks))
    for _, t := range tasks {
        rows = append(rows, jsonTaskRow{...})
    }
    return marshalIndentTo(w, rows)
}
```
Uses `make([]T, 0, len(tasks))` with append. The initial `0` length with capacity ensures a non-nil empty slice even when input is nil. V2 explicitly tests the `nil` input case.

**V3** (`json_formatter.go` lines 87-107):
```go
func (f *JSONFormatter) FormatTaskList(data *TaskListData) string {
    rows := make([]jsonTaskRow, 0)
    if data != nil && len(data.Tasks) > 0 {
        rows = make([]jsonTaskRow, len(data.Tasks))
        for i, task := range data.Tasks {
            rows[i] = jsonTaskRow{...}
        }
    }
    result, err := json.MarshalIndent(rows, "", "  ")
    if err != nil {
        return "[]"
    }
    return string(result)
}
```
The most defensive: starts with empty slice, then conditionally replaces it. Explicit nil check on `data`. Also has a fallback `return "[]"` on marshal error. V3 tests the `nil` `TaskListData` case.

#### FormatTaskDetail -- parent/closed Omission

The three versions handle the "omit parent/closed when null" requirement with different type strategies:

**V1** (`format_json.go` lines 37-55):
```go
type jsonTaskDetail struct {
    // ...
    Parent  *jsonRelatedTask `json:"parent,omitempty"`  // pointer to struct
    Closed  string           `json:"closed,omitempty"`  // plain string
    // ...
}
```
`Parent` is a pointer to `jsonRelatedTask` (a struct with `id`, `title`, `status`). When the source `TaskDetail.Parent` is `nil` (it's `*RelatedTask`), the JSON parent is nil pointer, so `omitempty` omits it. `Closed` is a string -- when empty, `omitempty` omits it. The parent in JSON is a full object `{"id":..., "title":..., "status":...}`.

**V2** (`json_formatter.go` lines 37-51):
```go
type jsonTaskDetail struct {
    // ...
    Parent  *string          `json:"parent,omitempty"`   // pointer to string
    Closed  *string          `json:"closed,omitempty"`   // pointer to string
    // ...
}
```
Both `Parent` and `Closed` are `*string`. When the source `showData.Parent` is empty string, V2 leaves the pointer nil; when non-empty, it creates a pointer: `parent := data.Parent; detail.Parent = &parent`. The parent in JSON is a bare string ID `"tick-e5f6"`, not a nested object. This is a significant structural difference from V1.

**V3** (`json_formatter.go` lines 22-37):
```go
type jsonTaskDetail struct {
    // ...
    Parent  string  `json:"parent,omitempty"`   // plain string
    Closed  string  `json:"closed,omitempty"`   // plain string
    // ...
}
```
Both are plain strings with `omitempty`. When the source `TaskDetailData.Parent` is empty, `omitempty` omits it. Like V2, parent in JSON is a bare string ID, not a nested object.

**Key difference**: V1 represents `parent` as a nested object `{"id": "...", "title": "...", "status": "..."}` while V2 and V3 represent it as a bare string ID. This reflects the different data structures from task 4-1: V1's `TaskDetail.Parent` is `*RelatedTask` (a struct), while V2/V3's source types have `Parent` as a plain `string`.

#### FormatStats -- Nesting Structure

The task plan specifies: "Nested object: `total`, `by_status`, `workflow`, `by_priority`."

**V1** (`format_json.go` lines 107-130) -- FLAT structure:
```go
obj := struct {
    Total      int                 `json:"total"`
    Open       int                 `json:"open"`
    InProgress int                 `json:"in_progress"`
    Done       int                 `json:"done"`
    Cancelled  int                 `json:"cancelled"`
    Ready      int                 `json:"ready"`
    Blocked    int                 `json:"blocked"`
    ByPriority []jsonPriorityEntry `json:"by_priority"`
}{...}
```
All status and workflow fields are at the same level as `total`. There is no `by_status` or `workflow` nesting. This does NOT match the spec requirement for nested objects.

**V2** (`json_formatter.go` lines 57-79) -- PROPERLY NESTED:
```go
type jsonStatsData struct {
    Total      int                 `json:"total"`
    ByStatus   jsonByStatus        `json:"by_status"`
    Workflow   jsonWorkflow        `json:"workflow"`
    ByPriority []jsonPriorityEntry `json:"by_priority"`
}
type jsonByStatus struct {
    Open       int `json:"open"`
    InProgress int `json:"in_progress"`
    Done       int `json:"done"`
    Cancelled  int `json:"cancelled"`
}
type jsonWorkflow struct {
    Ready   int `json:"ready"`
    Blocked int `json:"blocked"`
}
```
Three separate nested types. Matches the spec: `total` at top level, `by_status` nested object, `workflow` nested object, `by_priority` array.

**V3** (`json_formatter.go` lines 47-61) -- PARTIALLY NESTED:
```go
type jsonStats struct {
    Total      int                 `json:"total"`
    ByStatus   jsonStatusCounts    `json:"by_status"`
    ByPriority []jsonPriorityCount `json:"by_priority"`
}
type jsonStatusCounts struct {
    Open       int `json:"open"`
    InProgress int `json:"in_progress"`
    Done       int `json:"done"`
    Cancelled  int `json:"cancelled"`
    Ready      int `json:"ready"`
    Blocked    int `json:"blocked"`
}
```
Two nesting levels, but `ready` and `blocked` are placed inside `by_status` instead of a separate `workflow` object. There is no `workflow` key at all. This partially matches the spec -- it has nesting, but the structure is wrong.

#### FormatStats -- Priority Count Generation

**V1** and **V2** hardcode the 5-entry loop:
```go
// V1, format_json.go line 113; V2, json_formatter.go line 193
for i := 0; i < 5; i++ {
    bp[i] = jsonPriorityEntry{Priority: i, Count: data.ByPriority[i]}
}
```
Both use `[5]int` as the source type, guaranteeing exactly 5 entries.

**V3** iterates over the input slice:
```go
// json_formatter.go lines 209-214
for _, pc := range data.ByPriority {
    byPriority = append(byPriority, jsonPriorityCount{
        Priority: pc.Priority,
        Count:    pc.Count,
    })
}
```
V3's `StatsData.ByPriority` is `[]PriorityCount`, so the number of entries depends on the caller. The "always 5" guarantee is shifted to the caller rather than being enforced in the formatter.

#### FormatTransition

**V1** (`format_json.go` lines 84-93):
```go
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
```
Uses inline anonymous struct. Takes `TransitionData` value type.

**V2** (`json_formatter.go` lines 161-168):
```go
func (f *JSONFormatter) FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error {
    return marshalIndentTo(w, jsonTransition{
        ID:   id,
        From: string(oldStatus),
        To:   string(newStatus),
    })
}
```
Takes separate parameters with `task.Status` type, requires explicit `string()` conversion. Uses named `jsonTransition` struct.

**V3** (`json_formatter.go` lines 163-175):
```go
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
```
Takes plain strings, returns string. Has error fallback to `"{}"`.

#### FormatDepChange

All three versions produce `{"action", "task_id", "blocked_by"}` -- functionally identical except for the interface signatures (value struct vs separate params vs string return).

#### FormatMessage

All three versions produce `{"message": "..."}` -- functionally identical.

### Code Quality

#### Type System and Struct Design

**V1** defines 4 JSON-specific types: `jsonTaskListItem`, `jsonRelatedTask`, `jsonTaskDetail`, `jsonPriorityEntry`. Uses inline anonymous structs for transition, dep change, stats, and message (lines 84, 96, 107, 139). This is concise but means the structure is less discoverable and not reusable.

**V2** defines 10 named JSON-specific types: `jsonTaskRow`, `jsonRelatedTask`, `jsonTaskDetail`, `jsonStatsData`, `jsonByStatus`, `jsonWorkflow`, `jsonPriorityEntry`, `jsonTransition`, `jsonDepChange`, `jsonMessage`. This is the most explicit and self-documenting. Every JSON structure has a named type, making the code very readable:
```go
// json_formatter.go lines 85-97
type jsonTransition struct {
    ID   string `json:"id"`
    From string `json:"from"`
    To   string `json:"to"`
}
type jsonDepChange struct {
    Action    string `json:"action"`
    TaskID    string `json:"task_id"`
    BlockedBy string `json:"blocked_by"`
}
type jsonMessage struct {
    Message string `json:"message"`
}
```

**V3** defines 8 named JSON-specific types: `jsonTaskRow`, `jsonTaskDetail`, `jsonRelatedTask`, `jsonStats`, `jsonStatusCounts`, `jsonPriorityCount`, `jsonTransition`, `jsonDepChange`, `jsonMessage`. Comparable to V2 in explicitness.

#### Error Handling

**V1**: Returns errors from `jsonWrite` (which returns `enc.Encode` error). Clean pass-through. No wrapping.

**V2**: Wraps marshal errors with context: `fmt.Errorf("json marshal failed: %w", err)`. Also propagates write errors from `w.Write()`. For `FormatStats`, returns explicit error on type assertion failure: `fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)`. Most robust error handling.

**V3**: Swallows errors by returning fallback values (`"[]"`, `"{}"`). This means callers never see an error, which simplifies usage but hides problems. Includes nil-guard at the top of `FormatTaskDetail` and `FormatStats`:
```go
// json_formatter.go line 117
if data == nil {
    return "{}"
}
```

#### DRY Principle

**V1**: Single `jsonWrite` helper used by all methods. Good DRY.

**V2**: Single `marshalIndentTo` helper used by all methods. Good DRY.

**V3**: Each method calls `json.MarshalIndent` independently with the same arguments. `json.MarshalIndent(..., "", "  ")` appears 6 times. NOT DRY. A shared helper would reduce repetition.

#### Compile-Time Interface Check

**V1**: None. The `JSONFormatter` implements the interface, but there is no `var _ Formatter = &JSONFormatter{}` assertion. Correctness is only verified through tests.

**V2**: Explicit at file scope:
```go
// json_formatter.go line 16
var _ Formatter = &JSONFormatter{}
```

**V3**: Only in test file:
```go
// json_formatter_test.go line 10
var _ Formatter = &JSONFormatter{}
```
This is weaker than V2 -- the check exists but only runs during tests, not during normal compilation.

### Test Quality

#### V1 Test Functions (`format_json_test.go`, 289 LOC)

| Top-level Function | Subtests |
|---|---|
| `TestJSONFormatterFormatTaskList` | "it formats list as JSON array", "it formats empty list as [] not null", "it uses snake_case for all keys", "it produces valid parseable JSON" |
| `TestJSONFormatterFormatTaskDetail` | "it formats show with all fields", "it omits parent/closed when null", "it includes blocked_by/children as empty arrays", "it formats description as empty string not null" |
| `TestJSONFormatterFormatStats` | "it formats stats as structured nested object", "it includes 5 priority rows even at zero" |
| `TestJSONFormatterFormatTransition` | "it formats transition as JSON object" |
| `TestJSONFormatterFormatDepChange` | "it formats dep change as JSON object" |
| `TestJSONFormatterFormatMessage` | "it formats message as JSON object" |

**Total: 6 top-level, 12 subtests**

Notable details:
- Snake_case test checks for 4 list keys (`id`, `title`, `status`, `priority`) but not for detail keys.
- Valid JSON test uses `json.Valid()`.
- The "formats show with all fields" test checks `parent` as a nested object (reflecting V1's `*jsonRelatedTask` design).
- Empty blocked_by/children test passes `[]RelatedTask{}` explicitly (non-nil empty slices).
- Stats test only checks `total` field value, not nesting structure.

**Missing**: No nil-input test for `FormatTaskList`. No test verifying stats nesting (only checks `total`). No 2-space indent verification test. No compile-time interface check. No test for valid JSON across all methods.

#### V2 Test Functions (`json_formatter_test.go`, 668 LOC)

| Top-level Function | Subtests |
|---|---|
| `TestJSONFormatterImplementsInterface` | "it implements the full Formatter interface" |
| `TestJSONFormatterFormatTaskList` | "it formats list as JSON array", "it formats empty list as [] not null", "it formats nil list as [] not null" |
| `TestJSONFormatterFormatTaskDetail` | "it formats show with all fields", "it omits parent/closed when null", "it includes blocked_by/children as empty arrays", "it formats description as empty string not null" |
| `TestJSONFormatterSnakeCase` | "it uses snake_case for all keys" |
| `TestJSONFormatterFormatStats` | "it formats stats as structured nested object", "it includes 5 priority rows even at zero", "it returns error for non-StatsData input" |
| `TestJSONFormatterFormatTransitionDepMessage` | "it formats transition as JSON object", "it formats dep change as JSON object", "it formats message as JSON object" |
| `TestJSONFormatterProducesValidJSON` | "it produces valid parseable JSON" (table-driven across all 6 methods) |
| `TestJSONFormatterIndentation` | "it produces 2-space indented JSON" |

**Total: 8 top-level, 14 subtests (with the valid JSON test covering 6 sub-cases internally via table-driven approach)**

Notable details:
- Has a dedicated `TestJSONFormatterImplementsInterface` with compile-time check.
- `nil` list test is unique to V2 and V3 (V1 misses this).
- Snake_case test is the most thorough: checks 11 keys in detail output AND verifies NO camelCase variants:
  ```go
  if strings.Contains(got, `"blockedBy"`) {
      t.Errorf("output contains camelCase key 'blockedBy' instead of snake_case")
  }
  ```
- Stats test verifies the FULL nesting structure: checks `by_status` object with all 4 status fields, `workflow` object with `ready`/`blocked`, and `by_priority` array with each entry's `priority`/`count` values.
- Stats error test for non-StatsData input is unique to V2.
- Valid JSON test is table-driven covering ALL 6 format methods in one test function.
- Indentation test verifies 2-space indent AND explicitly checks for absence of 4-space and tab indentation.
- `blocked_by`/`children` test passes `nil` input (not empty slice), testing the nil-to-empty-array conversion. This is a stronger test than V1 and V3 which pass empty slices.
- Detail "show with all fields" test checks every single field value including `priority` as `float64(1)`.

**Missing**: No special character test (quotes/backslashes in titles).

#### V3 Test Functions (`json_formatter_test.go`, 702 LOC)

| Top-level Function | Subtests |
|---|---|
| `TestJSONFormatter` | "it implements Formatter interface" |
| `TestJSONFormatterFormatTaskList` | "it formats list as JSON array", "it formats empty list as [] not null", "it formats nil TaskListData as []" |
| `TestJSONFormatterFormatTaskDetail` | "it formats show with all fields", "it omits parent/closed when null", "it includes blocked_by/children as empty arrays", "it formats description as empty string not null", "it uses snake_case for all keys" |
| `TestJSONFormatterFormatStats` | "it formats stats as structured nested object", "it includes 5 priority rows even at zero" |
| `TestJSONFormatterFormatTransitionDepMessage` | "it formats transition as JSON object", "it formats dep add as JSON object", "it formats dep rm as JSON object", "it formats message as JSON object" |
| `TestJSONFormatterProducesValidJSON` | "it produces valid parseable JSON for list", "it produces valid parseable JSON for detail", "it produces valid parseable JSON for stats" |
| `TestJSONFormatterUses2SpaceIndent` | "it uses 2-space indent for list", "it uses 2-space indent for detail", "it uses 2-space indent for stats" |

**Total: 7 top-level, 21 subtests**

Notable details:
- Has `nil` `TaskListData` test (unique behavior -- tests wrapper struct nil, not just empty tasks).
- Snake_case test in detail checks for `blockedBy` camelCase absence AND `parentTitle` absence (verifying V3 doesn't leak its `ParentTitle` field).
- Dep change tests both "add" AND "remove" actions separately (V1 tests "added", V2 tests "added" in one test).
- Valid JSON tests check special characters: `"quotes"`, `\\`, and newlines in titles/descriptions:
  ```go
  {ID: "tick-a1b2", Title: "Task with \"quotes\" and \\ backslash", ...}
  ```
  And verifies round-trip preservation:
  ```go
  if parsed[0]["title"] != "Task with \"quotes\" and \\ backslash" {...}
  ```
- Indentation test is the most thorough: tests list (array nesting = 2-space for array item + 4-space for properties), detail (2-space), and stats (2-space). Also checks for tab absence.
- Stats test checks `by_status` nesting with all 6 fields (including `ready`/`blocked` inside `by_status` -- matching V3's non-spec structure).

**Missing**: No test for `blocked_by`/`children` when input is `nil` (only tests empty slice `[]RelatedTaskData{}`).

#### Test Coverage Comparison

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| List with 2 tasks | Yes | Yes | Yes |
| Empty list `[]` | Yes | Yes | Yes |
| Nil list input | **NO** | Yes | Yes |
| Detail all fields | Yes | Yes | Yes |
| Parent omitted when null | Yes | Yes | Yes |
| Closed omitted when null | Yes | Yes | Yes |
| blocked_by `[]` when empty | Yes (empty slice input) | Yes (nil input) | Yes (empty slice input) |
| children `[]` when empty | Yes (empty slice input) | Yes (nil input) | Yes (empty slice input) |
| Description always present | Yes | Yes | Yes |
| Snake_case keys list | Yes | Tested in detail test | Tested in detail test |
| Snake_case keys detail | **NO** | Yes (11 keys + camelCase absence) | Yes (camelCase + parentTitle absence) |
| Stats nested structure | **NO** (checks total only) | Yes (full nesting verification) | Yes (partial nesting verification) |
| 5 priority rows at zero | Yes | Yes | Yes |
| Priority entry values | **NO** | Yes (each entry checked) | Yes (each entry checked) |
| Transition JSON | Yes | Yes | Yes |
| Dep change JSON | Yes (1 action) | Yes (1 action) | Yes (2 actions: add + remove) |
| Message JSON | Yes | Yes | Yes |
| Valid JSON all methods | **NO** (list only) | Yes (table-driven, 6 methods) | Partial (3 methods: list, detail, stats) |
| 2-space indent | **NO** | Yes (1 test, also checks 4-space/tab absence) | Yes (3 tests: list, detail, stats; checks tab absence) |
| Special characters in JSON | **NO** | **NO** | Yes (quotes, backslash, newlines) |
| Interface compile check | **NO** | Yes | Yes |
| Stats type assertion error | N/A (no `interface{}` param) | Yes | N/A (no `interface{}` param) |
| Nil detail input | **NO** | **NO** | **NO** (code handles it but no test) |

**Tests unique to V1**: None.
**Tests unique to V2**: Stats type assertion error test. `blocked_by`/`children` nil-input test (stronger than empty-slice test).
**Tests unique to V3**: Dep change for both add and remove. Special character round-trip (quotes, backslash, newlines). Indentation test for 3 output types separately.
**Tests in all 3**: List as array, empty list `[]`, detail all fields, parent/closed omission, blocked_by/children as `[]`, description present, stats with priority, transition/dep/message as JSON.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 4 | 5 |
| Lines added | 438 | 900 | 965 |
| Impl LOC | 149 | 229 | 244 |
| Test LOC | 289 | 668 | 702 |
| Test functions (top-level) | 6 | 8 | 7 |
| Test subtests | 12 | 14 (+6 table-driven) | 21 |

## Verdict

**V2 is the best implementation.**

**Against V1**: V1 is the most compact (149 impl LOC) but has three significant issues: (1) The stats structure is flat rather than nested, failing acceptance criterion 7 which requires `total`, `by_status`, `workflow`, `by_priority` as distinct nested objects. V1 puts `open`, `in_progress`, `done`, `cancelled`, `ready`, `blocked` all at the top level with no nesting. (2) No compile-time interface check. (3) Weakest test coverage: no nil-input test, no stats nesting verification, no indentation test, no valid-JSON-across-all-methods test, and no snake_case verification for detail output. V1 also lacks the `fmt` import used by V2/V3 for error wrapping, though this is minor since V1 uses `json.NewEncoder` which doesn't need it.

**Against V3**: V3 has the most comprehensive test suite (702 LOC, 21 subtests), uniquely testing special character round-trips and indentation across 3 output types. However, it has two structural issues: (1) The stats structure puts `ready` and `blocked` inside `by_status` rather than in a separate `workflow` object as the spec requires. (2) The `by_priority` count is not guaranteed to be 5 -- it depends on the input `[]PriorityCount` slice length, shifting the invariant to the caller. (3) The `json.MarshalIndent` call is repeated 6 times without a shared helper, violating DRY. (4) Error handling swallows errors by returning fallback strings, which makes debugging harder.

**Why V2 wins**:
1. **Correct stats nesting** -- V2 is the ONLY version that correctly implements the spec's `total`, `by_status`, `workflow`, `by_priority` structure with proper `jsonByStatus` and `jsonWorkflow` nested types. This directly satisfies acceptance criterion 7.
2. **Best test rigor for what matters** -- V2's stats test verifies the full nesting structure (checks `by_status.open`, `workflow.ready`, etc.), tests nil input for lists, verifies all 6 methods produce valid JSON via table-driven test, and includes the strongest snake_case verification (11 keys + camelCase absence check).
3. **Proper error handling** -- wraps errors with context (`"json marshal failed: %w"`) and propagates rather than swallowing them.
4. **DRY** -- single `marshalIndentTo` helper, all named JSON types.
5. **Compile-time interface check** in production code (not just tests).
6. **Strongest blocked_by/children test** -- tests with `nil` input (not just empty slice), which is the actual Go gotcha the spec warns about.

V3 would rank second due to its test thoroughness (special characters, 3-way indent verification, dual dep actions), but the incorrect stats structure and lack of DRY in marshaling calls are meaningful weaknesses. V1 ranks last due to flat stats structure, missing compile-time check, and weakest test coverage.
