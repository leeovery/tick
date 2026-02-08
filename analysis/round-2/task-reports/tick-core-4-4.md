# Task tick-core-4-4: JSON Formatter

## Task Summary

Implement a `JSONFormatter` for `--json` output across list, show, stats, transition, dep-change, and message commands. The formatter must produce universal interchange output suitable for piping to `jq` or tool integration.

**Requirements from plan:**
- `JSONFormatter` implementing `Formatter` interface
- `FormatTaskList`: JSON array; empty produces `[]`, never `null`
- `FormatTaskDetail`: Object with all fields; `blocked_by`/`children` always `[]` when empty; `parent`/`closed` omitted when null; `description` always present (empty string)
- `FormatStats`: Nested object with `total`, `by_status`, `workflow`, `by_priority` (always 5 entries)
- `FormatTransition`: `{"id", "from", "to"}`
- `FormatDepChange`: `{"action", "task_id", "blocked_by"}`
- `FormatMessage`: `{"message"}`
- All keys `snake_case`; uses `json.MarshalIndent` (2-space)

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

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Implements full Formatter interface | PASS -- compile-time check `var _ Formatter = &JSONFormatter{}` | PASS -- verified via test `var _ Formatter = &JSONFormatter{}` |
| Empty list produces `[]` | PASS -- `make([]jsonTaskRow, 0, len(tasks))` ensures `[]` for nil and empty input | PASS -- `make([]jsonListRow, 0, len(rows))` ensures `[]` for nil and empty input |
| `blocked_by`/`children` always `[]` when empty | PASS -- explicit `make([]jsonRelatedTask, 0, ...)` in FormatTaskDetail | PASS -- explicit `make([]jsonRelatedTask, 0, ...)` in FormatTaskDetail |
| `parent`/`closed` omitted when null | PASS -- uses `*string` pointer fields with `omitempty`; only sets pointer when non-empty | PASS -- uses `string` fields with `omitempty`; Go zero-value `""` triggers omitempty |
| `description` always present | PASS -- no `omitempty` on Description field | PASS -- no `omitempty` on Description field |
| `snake_case` keys throughout | PASS -- all json tags use snake_case | PASS -- all json tags use snake_case |
| Stats nested with 5 priority entries | PASS -- explicit loop `for i := 0; i < 5; i++` builds 5 entries | PASS -- explicit loop `for i := 0; i < 5; i++` builds 5 entries |
| All output valid JSON | PASS -- uses `json.MarshalIndent` | PASS -- uses `json.MarshalIndent` |
| 2-space indented | PASS -- `json.MarshalIndent(v, "", "  ")` | PASS -- `json.MarshalIndent(v, "", "  ")` |

## Implementation Comparison

### Approach

Both versions follow the same fundamental approach: define dedicated JSON struct types with `json:"snake_case"` tags, map domain types to JSON types, and serialize via `json.MarshalIndent` with 2-space indentation.

**Key structural differences:**

**1. Formatter interface signatures differ between versions**

V2 (`internal/cli/formatter.go`) uses these signatures:
```go
FormatTaskList(w io.Writer, tasks []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
FormatStats(w io.Writer, stats interface{}) error
```

V4 (`internal/cli/format.go`) uses these signatures:
```go
FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
FormatTaskDetail(w io.Writer, detail TaskDetail) error
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
FormatStats(w io.Writer, stats StatsData) error
```

Notable differences:
- V4 adds a `quiet bool` parameter to `FormatTaskList` and `FormatDepChange`, supporting quiet mode at the interface level. V2 has no quiet mode awareness in the JSON formatter.
- V4's `FormatTransition` takes `string` parameters; V2 takes `task.Status` typed parameters (stronger type safety).
- V4's `FormatStats` takes a concrete `StatsData` value type; V2 takes `interface{}` and does a type assertion at runtime.
- V4's `FormatTaskDetail` takes a value `TaskDetail`; V2 takes a pointer `*showData`.

**2. Parent/Closed omission strategy**

V2 uses pointer types for nullable fields:
```go
type jsonTaskDetail struct {
    Parent      *string           `json:"parent,omitempty"`
    Closed      *string           `json:"closed,omitempty"`
}
```
And explicitly assigns pointers only when non-empty:
```go
if data.Parent != "" {
    parent := data.Parent
    detail.Parent = &parent
}
```

V4 uses plain strings with `omitempty`:
```go
type jsonTaskDetail struct {
    Parent      string            `json:"parent,omitempty"`
    Closed      string            `json:"closed,omitempty"`
}
```
And simply copies the value:
```go
obj := jsonTaskDetail{
    Parent:      detail.Parent,
    Closed:      detail.Closed,
}
```

Both approaches produce the correct behavior (omitted when empty). V2's pointer approach is the more "canonical" Go JSON pattern for distinguishing absent vs zero-value, though in this case both work equivalently since an empty string is the desired "absent" sentinel. V4's approach is simpler.

**3. Stats type safety**

V2 uses `interface{}` in the Formatter interface and does a runtime type assertion:
```go
func (f *JSONFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }
```

V4 uses a concrete type in the interface:
```go
func (f *JSONFormatter) FormatStats(w io.Writer, stats StatsData) error {
```

V4's approach is genuinely better -- it catches type mismatches at compile time rather than runtime. V2 adds a runtime error test for the wrong type, but this is a workaround for the loose interface.

**4. Quiet mode handling**

V4 adds quiet mode support in `FormatTaskList`:
```go
func (f *JSONFormatter) FormatTaskList(w io.Writer, rows []listRow, quiet bool) error {
    if quiet {
        ids := make([]string, len(rows))
        for i, r := range rows {
            ids[i] = r.ID
        }
        return f.writeJSON(w, ids)
    }
```

And in `FormatDepChange`:
```go
func (f *JSONFormatter) FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error {
    if quiet {
        return nil
    }
```

V2 has no quiet mode support at all in its JSON formatter. This is functionality beyond the task spec, but it's a reasonable design decision since the interface forces it.

**5. Marshal helper**

V2 uses a package-level function:
```go
func marshalIndentTo(w io.Writer, v interface{}) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return fmt.Errorf("json marshal failed: %w", err)
    }
    _, err = w.Write(append(data, '\n'))
    return err
}
```

V4 uses a method on the struct:
```go
func (f *JSONFormatter) writeJSON(w io.Writer, v interface{}) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return fmt.Errorf("json marshal error: %w", err)
    }
    _, err = fmt.Fprintf(w, "%s\n", data)
    return err
}
```

V2's `w.Write(append(data, '\n'))` is slightly more efficient (no format string parsing); V4's `fmt.Fprintf(w, "%s\n", data)` is slightly more readable. V4 making it a method is unnecessary since it doesn't use the receiver, but is a reasonable organizational choice. V2's package-level function could be reused by other formatters if needed.

**6. Domain types**

V2 uses `showData`/`relatedTask`/`TaskRow` (defined in `show.go` and `formatter.go`).
V4 uses `TaskDetail`/`RelatedTask`/`listRow` (defined in `format.go`).

V4's types are cleaner -- `TaskDetail` and `RelatedTask` are exported and semantically named. V2's `showData` is unexported and tied to the "show" command name. V4's `listRow` is unexported, which is less consistent with its exported `TaskDetail`/`RelatedTask`.

### Code Quality

**Go idioms:**

V2's compile-time interface check is a standalone line:
```go
var _ Formatter = &JSONFormatter{}
```
This is idiomatic Go. V4 lacks a compile-time check in the implementation file, relying only on the test for verification.

**Error handling:**

V2's `FormatStats` handles invalid input with a runtime error:
```go
sd, ok := stats.(*StatsData)
if !ok {
    return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
}
```
This is defensive but indicates a design smell (the `interface{}` parameter). V4 eliminates this class of error entirely via the concrete type signature.

**Naming:**

V2: `jsonTaskRow`, `jsonRelatedTask`, `jsonTaskDetail`, `jsonStatsData`, `jsonByStatus`, `jsonWorkflow`, `jsonPriorityEntry`, `jsonTransition`, `jsonDepChange`, `jsonMessage`, `marshalIndentTo`

V4: `jsonListRow`, `jsonRelatedTask`, `jsonTaskDetail`, `jsonStats`, `jsonByStatus`, `jsonWorkflow`, `jsonPriorityEntry`, `jsonTransition`, `jsonDepChange`, `jsonMessage`, `writeJSON`

Both use the `json` prefix convention consistently. V4's `jsonListRow` better matches the "list" context vs V2's `jsonTaskRow`. V4's `jsonStats` is slightly more concise than V2's `jsonStatsData`.

**Documentation:**

Both versions provide thorough doc comments on all types and methods. V2's comments are slightly more detailed (e.g., explaining the nil slice gotcha inline). V4's comments are adequate but more terse.

### Test Quality

**V2 Test Functions:**

| Function | Subtests |
|----------|----------|
| `TestJSONFormatterImplementsInterface` | "it implements the full Formatter interface" |
| `TestJSONFormatterFormatTaskList` | "it formats list as JSON array", "it formats empty list as [] not null", "it formats nil list as [] not null" |
| `TestJSONFormatterFormatTaskDetail` | "it formats show with all fields", "it omits parent/closed when null", "it includes blocked_by/children as empty arrays", "it formats description as empty string not null" |
| `TestJSONFormatterSnakeCase` | "it uses snake_case for all keys" |
| `TestJSONFormatterFormatStats` | "it formats stats as structured nested object", "it includes 5 priority rows even at zero", "it returns error for non-StatsData input" |
| `TestJSONFormatterFormatTransitionDepMessage` | "it formats transition as JSON object", "it formats dep change as JSON object", "it formats message as JSON object" |
| `TestJSONFormatterProducesValidJSON` | table-driven subtest: "list", "detail", "stats", "transition", "dep", "message" |
| `TestJSONFormatterIndentation` | "it produces 2-space indented JSON" |

Total: 8 top-level test functions, 15 subtests.

**V4 Test Functions:**

| Function | Subtests |
|----------|----------|
| `TestJSONFormatter_ImplementsFormatter` | "it implements the full Formatter interface" |
| `TestJSONFormatter_FormatTaskList` | "it formats list as JSON array", "it formats empty list as [] not null" |
| `TestJSONFormatter_FormatTaskDetail` | "it formats show with all fields", "it omits parent/closed when null", "it includes blocked_by/children as empty arrays", "it formats description as empty string not null" |
| `TestJSONFormatter_SnakeCaseKeys` | "it uses snake_case for all keys" |
| `TestJSONFormatter_FormatStats` | "it formats stats as structured nested object", "it includes 5 priority rows even at zero" |
| `TestJSONFormatter_FormatTransitionDepMessage` | "it formats transition/dep/message as JSON objects" |
| `TestJSONFormatter_ValidJSON` | table-driven subtest: "task list", "task detail", "stats", "transition", "dep change", "message" |
| `TestJSONFormatter_Indentation` | "it produces 2-space indented JSON" |

Total: 8 top-level test functions, 12 subtests.

**Edge cases compared:**

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Empty list (`[]TaskRow{}`) | Tested | Tested (within empty list test) |
| Nil list (nil input) | Tested separately | Tested (nil is the primary test; empty slice also tested within same subtest) |
| Nil blocked_by/children | Tested (passes nil, asserts `[]`) | Tested (passes nil via zero-value, asserts `[]`) |
| Empty description | Tested | Tested |
| Non-StatsData input error | Tested | N/A (impossible -- concrete type) |
| Special chars in JSON (quotes, newlines) | Not tested | Tested in valid JSON test: `"Task with \"quotes\" and special chars"`, `"Line 1\nLine 2\n\"Quoted\""` |
| Quiet mode for list | N/A (not in interface) | Not tested |
| Quiet mode for dep change | N/A (not in interface) | Not tested |
| Snake_case in nested objects | Tested (checks parent-level keys only) | Tested (also checks nested blocked_by entry keys) |
| 4-space / tab indentation rejected | Tested (explicit negative check) | Not tested (only checks indent is multiple of 2) |

**Test structure:**

Both use a mix of individual subtests and table-driven approaches (for the "valid JSON" cross-format test). V2 uses `t.Fatalf`/`t.Errorf` directly. V4 also uses `t.Fatalf`/`t.Errorf` directly. Neither uses a test helper library.

**Assertion quality:**

V2 tests are slightly more granular -- for instance, the indentation test checks for the absence of 4-space and tab indentation explicitly:
```go
if strings.Contains(got, "    \"message\"") {
    t.Errorf("output should use 2-space indent, not 4-space:\n%s", got)
}
if strings.Contains(got, "\t\"message\"") {
    t.Errorf("output should use 2-space indent, not tabs:\n%s", got)
}
```

V4's indentation test is more programmatic but less precise -- it checks that indent is a multiple of 2 but doesn't reject 4-space:
```go
indent := len(line) - len(trimmed)
if indent%2 != 0 {
    t.Errorf("indent %d is not a multiple of 2: %q", indent, s)
}
```
This would pass for 4-space indentation, which is technically wrong.

V4 tests special characters in JSON (quotes, newlines), which V2 does not -- this is a genuinely better edge case to cover.

**Test gaps:**

- V2 is missing: special character handling in JSON strings
- V4 is missing: explicit nil list test (separate from empty), non-StatsData input (N/A), explicit negative indentation checks, quiet mode behavior tests

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 4 | 4 |
| Lines added | 900 | 927 |
| Impl LOC | 229 | 225 |
| Test LOC | 668 | 701 |
| Test functions | 8 (15 subtests) | 8 (12 subtests) |

## Verdict

**V4 is marginally better overall**, primarily due to two genuine design advantages:

1. **Type-safe `FormatStats` signature**: V4's `FormatStats(w io.Writer, stats StatsData)` eliminates an entire class of runtime errors that V2 must handle via type assertion. This is the single most meaningful architectural difference.

2. **Cleaner domain types**: V4's exported `TaskDetail`/`RelatedTask` types are better named and more reusable than V2's unexported `showData`/`relatedTask`.

V2 has some countervailing strengths:
- The compile-time interface check `var _ Formatter = &JSONFormatter{}` in the implementation file (not just tests)
- More test subtests (15 vs 12), including a dedicated nil-list test and explicit negative indentation assertions
- V2's pointer-based `*string` approach for parent/closed is more canonically correct for JSON nullable fields, even though plain `omitempty` on strings works identically here

V4's additional quiet-mode support goes beyond the task spec and shows awareness of the broader system, but it lacks test coverage for that behavior, which somewhat undermines the advantage.

Both versions are well-implemented, produce correct output for all acceptance criteria, and have thorough test coverage. The differences are modest; neither implementation has bugs or significant quality issues.
