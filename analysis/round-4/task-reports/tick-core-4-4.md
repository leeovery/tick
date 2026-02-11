# Task tick-core-4-4: JSON formatter -- list, show, stats output

## Task Summary

Implement a `JSONFormatter` implementing the `Formatter` interface that produces `--json` output for CLI commands. Requirements:

- **FormatTaskList**: JSON array; empty input produces `[]`, never `null`.
- **FormatTaskDetail**: Object with all fields. `blocked_by`/`children` always `[]` when empty. `parent`/`closed` omitted when null. `description` always present (empty string).
- **FormatStats**: Nested object with `total`, `by_status`, `workflow`, `by_priority` (always 5 entries P0-P4).
- **FormatTransition**: `{"id", "from", "to"}`
- **FormatDepChange**: `{"action", "task_id", "blocked_by"}`
- **FormatMessage**: `{"message"}`
- All keys `snake_case`. Use `json.MarshalIndent` with 2-space indentation.

### Acceptance Criteria

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

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Implements full Formatter interface | PASS -- uses `interface{}` params matching V5's Formatter; runtime type assertions for each method | PASS -- uses typed params matching V6's Formatter; compile-time `var _ Formatter = (*JSONFormatter)(nil)` |
| Empty list produces `[]` | PASS -- `make([]jsonListItem, 0, len(rows))` ensures non-nil slice | PASS -- same `make([]jsonTaskListItem, 0, len(tasks))` pattern |
| `blocked_by`/`children` always `[]` when empty | PASS -- explicit `make([]jsonRelatedTask, 0, ...)` for both | PASS -- `toJSONRelated()` helper always returns `make(..., 0, ...)` |
| `parent`/`closed` omitted when null | PASS -- `omitempty` JSON tags on string fields | PASS -- `omitempty` tags; `Closed` handled via `*time.Time` nil check |
| `description` always present | PASS -- no `omitempty` on description field | PASS -- same approach, no `omitempty` |
| `snake_case` keys throughout | PASS -- all JSON tags use snake_case | PASS -- all JSON tags use snake_case |
| Stats nested with 5 priority entries | PASS -- `make([]jsonPriorityEntry, 5)` loop 0..4 | PASS -- identical pattern |
| All output valid JSON | PASS -- `json.MarshalIndent` for all methods | PASS -- `json.MarshalIndent` via `marshalIndentJSON` |
| 2-space indented | PASS -- `json.MarshalIndent(v, "", "  ")` | PASS -- `json.MarshalIndent(v, "", "  ")` |

## Implementation Comparison

### Approach

The two versions solve the same problem with fundamentally different interface designs, leading to different implementation styles.

**Interface Design (the key divergence)**

V5 implements a Formatter interface using `io.Writer` and `interface{}` parameters:

```go
// V5 Formatter interface (format.go at commit time)
type Formatter interface {
    FormatTaskList(w io.Writer, data interface{}) error
    FormatTaskDetail(w io.Writer, data interface{}) error
    FormatTransition(w io.Writer, data interface{}) error
    FormatDepChange(w io.Writer, data interface{}) error
    FormatStats(w io.Writer, data interface{}) error
    FormatMessage(w io.Writer, msg string)
}
```

V6 implements a Formatter interface using typed parameters and string returns:

```go
// V6 Formatter interface (format.go)
type Formatter interface {
    FormatTaskList(tasks []task.Task) string
    FormatTaskDetail(detail TaskDetail) string
    FormatTransition(id string, oldStatus string, newStatus string) string
    FormatDepChange(action string, taskID string, depID string) string
    FormatStats(stats Stats) string
    FormatMessage(msg string) string
}
```

This is the single most important architectural difference. V6's interface is type-safe at compile time. V5's requires runtime type assertions and returns errors for type mismatches.

**V5's runtime type assertion pattern (repeated in every method):**

```go
func (f *JSONFormatter) FormatTaskList(w io.Writer, data interface{}) error {
    rows, ok := data.([]TaskRow)
    if !ok {
        return fmt.Errorf("FormatTaskList: expected []TaskRow, got %T", data)
    }
    // ...
}
```

**V6's direct typed approach:**

```go
func (f *JSONFormatter) FormatTaskList(tasks []task.Task) string {
    items := make([]jsonTaskListItem, 0, len(tasks))
    for _, t := range tasks {
        items = append(items, jsonTaskListItem{
            ID:       t.ID,
            Title:    t.Title,
            Status:   string(t.Status),
            Priority: t.Priority,
        })
    }
    return marshalIndentJSON(items)
}
```

**Data source types**

V5 uses internal presentation types (`showData` with unexported fields, `TaskRow`, `*TransitionData`, `*DepChangeData`, `*StatsData`). The data has already been pre-formatted into strings before reaching the formatter. For example, `showData.created` is a `string`, not a `time.Time`.

V6 uses domain types directly (`task.Task`, `TaskDetail`, `Stats`). The formatter takes responsibility for converting domain values to strings, e.g.:

```go
// V6: formatter converts time.Time to string
Created: task.FormatTimestamp(t.Created),
Updated: task.FormatTimestamp(t.Updated),
```

**JSON marshaling helper**

V5 uses a `writeJSON` function that writes directly to an `io.Writer` and appends a newline:

```go
func writeJSON(w io.Writer, v interface{}) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return fmt.Errorf("marshaling JSON: %w", err)
    }
    _, err = w.Write(append(data, '\n'))
    return err
}
```

V6 uses a `marshalIndentJSON` function that returns a string, with a silent fallback to `"null"` on error:

```go
func marshalIndentJSON(v interface{}) string {
    b, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return "null"
    }
    return string(b)
}
```

**FormatMessage return type**

V5's `FormatMessage` has no return value (matches `FormatMessage(w io.Writer, msg string)`) and uses `//nolint:errcheck` to suppress the unchecked `writeJSON` error. V6's `FormatMessage` returns `string`, consistent with all other methods.

**RelatedTask conversion**

V5 manually maps each field from unexported `relatedTask` structs:

```go
blockedBy = append(blockedBy, jsonRelatedTask{
    ID:     rt.id,
    Title:  rt.title,
    Status: rt.status,
})
```

V6 extracts this into a reusable `toJSONRelated` helper that uses type conversion (possible because `RelatedTask` and `jsonRelatedTask` have identical field layouts):

```go
func toJSONRelated(related []RelatedTask) []jsonRelatedTask {
    result := make([]jsonRelatedTask, 0, len(related))
    for _, r := range related {
        result = append(result, jsonRelatedTask(r))
    }
    return result
}
```

**Nil slice handling for FormatTaskList**

V6 explicitly handles `nil` input slices because its `FormatTaskList` accepts `[]task.Task` which could be nil. V5 accepts `interface{}` so nil-ness is handled by the type assertion pattern -- a nil `[]TaskRow` would still match the assertion and `make([]jsonListItem, 0, 0)` would produce `[]`.

### Code Quality

**Type Safety**

V6 is genuinely better here. V5 uses `interface{}` for 5 of 6 methods, requiring runtime type assertions with error returns. This means type errors are caught at runtime rather than compile time. V6's typed interface catches all mismatches at compile time.

V6 also includes a compile-time interface check (line 14):
```go
var _ Formatter = (*JSONFormatter)(nil)
```

V5's commit-time code has no compile-time check (the diff shows `var _ Formatter = &JSONFormatter{}` in the test file only).

**Error Handling**

V5 wraps marshal errors properly with `%w`:
```go
return fmt.Errorf("marshaling JSON: %w", err)
```

V6 silently returns `"null"` on marshal failure:
```go
if err != nil {
    return "null"
}
```

V5's approach is better for debugging. V6's comment says "should not happen with controlled types" which is true, but silently returning `"null"` is a weaker error handling pattern. However, since the interface returns `string` not `(string, error)`, V6 has no mechanism to propagate errors. The `"null"` fallback is a pragmatic choice given the constraint.

V5's `FormatMessage` uses `//nolint:errcheck` to suppress the unchecked `writeJSON` error, which is a code smell even with the linter directive.

**Naming**

Both use clear, consistent naming. V5 uses `jsonListItem` while V6 uses `jsonTaskListItem` (slightly more descriptive). V5 uses `jsonByStatus` while V6 uses `jsonStatusCounts` (V6 slightly more self-documenting).

**DRY**

V6 extracts `toJSONRelated()` as a helper, eliminating duplicated blocked_by/children conversion code. V5 repeats the mapping loop inline twice in `FormatTaskDetail`.

**Documentation**

Both versions document all exported types and functions. V5 adds `Data must be []TaskRow` / `Data must be *showData` etc. to doc comments, which is necessary given the `interface{}` params. V6 doesn't need such documentation because the types are self-documenting.

**Field Ordering in jsonTaskDetail**

V5 orders fields: `id, title, status, priority, parent, created, updated, closed, description, blocked_by, children`
V6 orders fields: `id, title, status, priority, description, parent, created, updated, closed, blocked_by, children`

V6 moves `description` before `parent`, grouping core task attributes together before relational/temporal fields. Neither order is specified by the task plan. JSON key ordering from `json.MarshalIndent` follows struct field order, so this affects output aesthetics.

### Test Quality

**V5 Test Functions (682 lines)**

Top-level functions:
1. `TestJSONFormatterImplementsInterface` -- 1 subtest
2. `TestJSONFormatterFormatTaskList` -- 4 subtests
3. `TestJSONFormatterFormatTaskDetail` -- 5 subtests
4. `TestJSONFormatterFormatStats` -- 3 subtests
5. `TestJSONFormatterFormatTransitionDepMessage` -- 4 subtests

Individual subtests (17 total):
- `"it implements the full Formatter interface"` -- compile-time check via assignment
- `"it formats list as JSON array"` -- 2 items, checks field values
- `"it formats empty list as [] not null"` -- empty slice produces `[]`
- `"it produces valid parseable JSON for list"` -- `json.Valid` check
- `"it uses snake_case keys for list"` -- string contains checks
- `"it uses 2-space indentation for list"` -- checks `\n  {` and `\n    "id"` and no tabs
- `"it formats show with all fields"` -- all 11 keys present, specific values checked
- `"it omits parent/closed when null"` -- verifies keys absent
- `"it includes blocked_by/children as empty arrays"` -- string check for `"blocked_by": []` plus parsed check
- `"it formats description as empty string not null"` -- parsed check
- `"it uses snake_case for all keys in show"` -- checks 11 snake_case keys, 5 camelCase exclusions
- `"it formats stats as structured nested object"` -- full nested verification
- `"it includes 5 priority rows even at zero"` -- zero-value stats
- `"it uses snake_case keys for stats"` -- 4 snake_case inclusions, 6 camelCase exclusions
- `"it formats transition as JSON object"` -- 3 field checks
- `"it formats dep change as JSON object"` -- 3 field checks
- `"it formats message as JSON object"` -- 1 field check
- `"it produces valid parseable JSON for all formats"` -- `json.Valid` across all 6 format methods

Edge cases tested:
- Empty list (empty slice)
- No blocked_by/children (nil slice in input, verified as `[]` in output)
- Empty description (zero value string)
- Absent parent/closed (empty string fields omitted)
- Zero-count priority entries
- All format methods produce valid JSON

**V6 Test Functions (570 lines)**

Top-level function:
1. `TestJSONFormatter` -- 12 subtests (single parent function)

Individual subtests (12 total):
- `"it formats list as JSON array"` -- 2 items, field checks, also checks second item
- `"it formats empty list as [] not null"` -- empty slice AND nil slice (V6 tests both!)
- `"it formats show with all fields"` -- all expected fields, uses `task.Task` domain type, verifies blocker/child sub-fields
- `"it omits parent/closed when null"` -- verifies keys absent
- `"it includes blocked_by/children as empty arrays"` -- parsed check
- `"it includes blocked_by/children as empty arrays even with nil input"` -- explicit nil `BlockedBy`/`Children` (extra edge case!)
- `"it formats description as empty string not null"` -- parsed check
- `"it uses snake_case for all keys"` -- 11 expected keys, 8 camelCase exclusions (broader than V5)
- `"it formats stats as structured nested object"` -- full nested verification
- `"it includes 5 priority rows even at zero"` -- zero-value stats
- `"it formats transition/dep/message as JSON objects"` -- combined test: transition, dep-add, dep-remove, message
- `"it produces valid parseable JSON"` -- table-driven across 8 outputs including nil list and dep-remove
- `"it uses 2-space indentation"` -- checks for 2-space indent pattern

Edge cases tested:
- Empty list (empty slice)
- Nil list input (V6 exclusive)
- No blocked_by/children (empty slices)
- Nil blocked_by/children input (V6 exclusive -- tests the Go nil slice gotcha explicitly)
- Empty description
- Absent parent/closed
- Zero-count priority entries
- Dep "removed" action (V6 exclusive -- V5 only tests "added")
- All format methods produce valid JSON

**Test Coverage Diff**

V6 has three edge cases V5 lacks:
1. **Nil slice for `FormatTaskList(nil)`** -- V5 only tests `[]TaskRow{}`, not a nil slice.
2. **Nil `BlockedBy`/`Children` in detail** -- V6 has a dedicated `"even with nil input"` subtest. V5 does not explicitly test nil related task slices (only empty).
3. **Dep "removed" action** -- V6 tests both "added" and "removed" dep actions. V5 only tests "added".

V5 has two subtests V6 lacks:
1. **Explicit `"it produces valid parseable JSON for list"`** -- standalone test. V6 covers this within the combined validity test.
2. **Explicit `"it uses snake_case keys for stats"`** -- standalone test for stats keys. V6's snake_case test focuses on task detail only.

However, V6's combined `"it produces valid parseable JSON"` subtest uses a table-driven pattern testing 8 distinct outputs, which is more comprehensive than V5's equivalent test.

**Assertion Style**

Both versions use raw `testing.T` assertions (no testify or similar). Both parse JSON and check values via type assertions. V5 uses `t.Fatalf`/`t.Errorf` consistently. V6 uses the same pattern.

**Test Organization**

V5 uses 4 top-level test functions grouping by formatter method (List, Detail, Stats, TransitionDepMessage). V6 uses a single `TestJSONFormatter` parent with all subtests nested. V6's flat structure is slightly cleaner but V5's grouping by method provides better test-failure isolation at the top level.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- code is properly formatted | PASS -- code is properly formatted |
| Handle all errors explicitly (no naked returns) | PARTIAL -- `FormatMessage` uses `//nolint:errcheck` to suppress unchecked error from `writeJSON` | PASS -- `marshalIndentJSON` returns `"null"` on error (no error to ignore); `FormatMessage` returns string |
| Write table-driven tests with subtests | PARTIAL -- uses subtests throughout but no table-driven tests; all tests are individual assertions | PARTIAL -- `"it produces valid parseable JSON"` uses table-driven struct with named cases; other tests are individual |
| Document all exported functions, types, and packages | PASS -- all exported symbols documented | PASS -- all exported symbols documented |
| Propagate errors with fmt.Errorf("%w", err) | PASS -- `writeJSON` uses `fmt.Errorf("marshaling JSON: %w", err)` | FAIL -- `marshalIndentJSON` swallows errors, returns `"null"`. No error wrapping anywhere. |
| Use `interface{}` judiciously (avoid reflection without justification) | FAIL -- entire Formatter interface uses `interface{}` params requiring runtime type assertions | PASS -- typed interface, no `interface{}` in signatures |
| Ignore errors (avoid _ assignment without justification) | PARTIAL -- `FormatMessage` has `//nolint:errcheck` which is a documented suppression | PASS -- no ignored errors |
| Use panic for normal error handling | PASS -- no panics | PASS -- no panics |

### Spec-vs-Convention Conflicts

**Conflict 1: `interface{}` vs typed parameters in Formatter interface**

- **Spec says**: The task plan does not specify the Formatter method signatures; it says "JSONFormatter implementing Formatter interface."
- **Go convention / skill**: The golang-pro skill says "avoid reflection without performance justification" and Go idiom strongly favors concrete types in interfaces. Go proverb: "interface{} says nothing."
- **V5 chose**: `interface{}` parameters with runtime type assertions and error returns.
- **V6 chose**: Typed parameters with compile-time safety.
- **Assessment**: V6's choice is the correct Go-idiomatic approach. V5's `interface{}` usage introduces unnecessary runtime risk and verbose type-checking boilerplate. This is not a reasonable judgment call by V5 -- there is no technical justification for using `interface{}` here.

**Conflict 2: Error propagation vs string return**

- **Spec says**: No explicit requirement on error handling for marshaling.
- **Go convention / skill**: "Propagate errors with `fmt.Errorf("%w", err)`" and "Handle all errors explicitly."
- **V5 chose**: Methods return `error`, `writeJSON` wraps with `%w`. But `FormatMessage` ignores the error.
- **V6 chose**: Methods return `string`, `marshalIndentJSON` swallows errors silently.
- **Assessment**: This is a genuine trade-off. V5's error returns are more Go-idiomatic for I/O operations. V6's string returns make the API simpler but lose error propagation. For a formatter that uses `json.MarshalIndent` on controlled struct types (where errors essentially cannot occur), V6's simpler API is a reasonable judgment call. However, the silent `"null"` fallback is less defensible than at least logging a warning.

**Conflict 3: `io.Writer` vs string return**

- **Spec says**: No requirement on output mechanism.
- **Go convention**: `io.Writer` is standard for streaming output. String return is simpler for formatting-only operations.
- **V5 chose**: `io.Writer` parameter on all methods.
- **V6 chose**: String return; caller writes to output.
- **Assessment**: Both are valid Go patterns. `io.Writer` is more flexible (streaming, buffering). String return is simpler and more testable (no buffer setup in tests). For a formatter that produces small, bounded output, V6's approach is reasonable. V6's tests are notably cleaner as a result (no `bytes.Buffer` setup).

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 4 (2 impl + 2 docs) | 4 (2 impl + 2 docs) |
| Lines added | 917 | 783 |
| Impl LOC | 232 | 210 |
| Test LOC | 682 | 570 |
| Test functions | 5 top-level / 17 subtests | 1 top-level / 12 subtests |

## Verdict

**V6 is the better implementation.**

The decisive factor is type safety. V5's `interface{}` Formatter interface is a significant design weakness: it pushes type checking to runtime, adds verbose type-assertion boilerplate to every method, and violates the Go proverb that "interface{} says nothing." V6's typed interface catches all mismatches at compile time, produces cleaner implementation code, and aligns with the golang-pro skill constraint against unnecessary use of `interface{}`.

V6 also demonstrates better edge case coverage in tests, specifically testing nil slices for both `FormatTaskList(nil)` and nil `BlockedBy`/`Children` -- directly addressing the Go nil-slice-to-`null` gotcha called out in the task plan's edge cases section. V6 also tests both "added" and "removed" dep actions, while V5 only tests "added."

V6 extracts the `toJSONRelated()` helper for DRY conversion of related tasks, while V5 duplicates the mapping loop. V6's tests are cleaner (no `bytes.Buffer` boilerplate) due to the string-return interface design.

V5's only advantage is explicit error propagation via `fmt.Errorf("%w", err)` in `writeJSON`, which is more Go-idiomatic. V6 swallows marshal errors with a `"null"` fallback. However, since `json.MarshalIndent` on controlled struct types cannot realistically fail, this advantage is theoretical rather than practical.

V6 produces 134 fewer total lines while covering more edge cases, demonstrating a more focused and efficient implementation.
