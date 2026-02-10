# Task 4-4: JSON formatter -- list, show, stats output

## Task Plan Summary

Implement a `JSONFormatter` that satisfies the `Formatter` interface, producing JSON output for all commands. Key requirements:

- **FormatTaskList**: JSON array; empty input produces `[]`, never `null`.
- **FormatTaskDetail**: Object with all fields. `blocked_by`/`children` always `[]` when empty. `parent`/`closed` omitted when null. `description` always present (empty string).
- **FormatStats**: Nested object with `total`, `by_status`, `workflow`, `by_priority` (always 5 entries P0-P4).
- **FormatTransition**: `{"id", "from", "to"}`
- **FormatDepChange**: `{"action", "task_id", "blocked_by"}`
- **FormatMessage**: `{"message"}`
- All keys `snake_case`. Use `json.MarshalIndent` with 2-space indentation.
- 11 specified tests covering empty lists, null/empty handling, snake_case verification, stats structure, valid JSON output, etc.

---

## V4 Implementation

### Architecture & Design

**File**: `internal/cli/json_formatter.go` (223 lines)

V4 defines `JSONFormatter` as a zero-size struct implementing the `Formatter` interface. The Formatter interface in V4 uses **concrete, typed method signatures**:

```go
type Formatter interface {
    FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
    FormatTaskDetail(w io.Writer, detail TaskDetail) error
    FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
    FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
    FormatStats(w io.Writer, stats StatsData) error
    FormatMessage(w io.Writer, msg string) error
}
```

Key design decisions:

1. **Exported data types for formatter input**: V4 uses `TaskDetail` (exported struct with exported fields), `RelatedTask` (exported), `StatsData` (exported), and `listRow` (unexported but with exported fields like `ID`, `Title`, `Status`, `Priority`). This creates a clear contract: the formatter receives well-typed domain data.

2. **Private JSON DTO structs**: Each output shape has a corresponding private struct with `json` tags: `jsonListRow`, `jsonRelatedTask`, `jsonTaskDetail`, `jsonTransition`, `jsonDepChange`, `jsonMessage`, `jsonStats`, `jsonByStatus`, `jsonWorkflow`, `jsonPriorityEntry`. This cleanly separates serialization concerns from domain types.

3. **Method receiver helper**: `writeJSON` is a method on `*JSONFormatter` (`func (f *JSONFormatter) writeJSON(...)`), keeping it scoped to the type.

4. **Quiet mode handling**: `FormatTaskList` and `FormatDepChange` accept a `quiet bool` parameter directly. In quiet mode for list, it outputs just IDs as a JSON string array. In quiet mode for dep change, it produces no output at all.

5. **Extra field**: The `jsonTaskDetail` struct includes `ParentTitle string \`json:"parent_title,omitempty"\`` -- a field not specified in the task plan. This is an addition that goes beyond the spec.

6. **Error propagation in FormatMessage**: Returns `error`, matching the interface.

### Code Quality

- **Idiomatic Go**: Uses `json.MarshalIndent` correctly, `fmt.Errorf` with `%w` for error wrapping.
- **Nil slice handling**: `make([]jsonListRow, 0, len(rows))` correctly ensures `[]` instead of `null` in JSON output. Applied consistently to `blockedBy` and `children` slices too.
- **Comments**: Every exported method and every private type has a godoc comment. The `jsonTaskDetail` type has an especially thorough comment explaining the `omitempty` semantics.
- **Write method**: Uses `fmt.Fprintf(w, "%s\n", data)` to write with trailing newline.
- **FormatTransition signature**: Takes individual parameters (`id string, oldStatus string, newStatus string`) rather than a data struct, which spreads the method signature thin but matches the V4 Formatter interface.

### Test Coverage

**File**: `internal/cli/json_formatter_test.go` (701 lines)

V4 tests are organized into 7 top-level test functions with subtests matching nearly all 11 specified test cases:

| Spec Test | V4 Coverage |
|-----------|-------------|
| "it formats list as JSON array" | `TestJSONFormatter_FormatTaskList/"it formats list as JSON array"` |
| "it formats empty list as [] not null" | `TestJSONFormatter_FormatTaskList/"it formats empty list as [] not null"` -- tests both `nil` and empty slice |
| "it formats show with all fields" | `TestJSONFormatter_FormatTaskDetail/"it formats show with all fields"` |
| "it omits parent/closed when null" | `TestJSONFormatter_FormatTaskDetail/"it omits parent/closed when null"` |
| "it includes blocked_by/children as empty arrays" | `TestJSONFormatter_FormatTaskDetail/"it includes blocked_by/children as empty arrays"` |
| "it formats description as empty string not null" | `TestJSONFormatter_FormatTaskDetail/"it formats description as empty string not null"` |
| "it uses snake_case for all keys" | `TestJSONFormatter_SnakeCaseKeys/"it uses snake_case for all keys"` -- checks nested objects too |
| "it formats stats as structured nested object" | `TestJSONFormatter_FormatStats/"it formats stats as structured nested object"` |
| "it includes 5 priority rows even at zero" | `TestJSONFormatter_FormatStats/"it includes 5 priority rows even at zero"` |
| "it formats transition/dep/message as JSON objects" | `TestJSONFormatter_FormatTransitionDepMessage/"it formats transition/dep/message as JSON objects"` |
| "it produces valid parseable JSON" | `TestJSONFormatter_ValidJSON/"it produces valid parseable JSON"` -- table-driven across all 6 methods |

Additional tests:
- `TestJSONFormatter_ImplementsFormatter` -- compile-time interface check
- `TestJSONFormatter_Indentation/"it produces 2-space indented JSON"` -- checks indent is multiple of 2 on every line

**Test quality observations**:
- The snake_case test is thorough: checks all expected keys are present, checks bad camelCase variants, and verifies nested `blocked_by` entries also use snake_case.
- The valid JSON test uses a table-driven approach with special characters (`"quotes"`, `\n` newlines) in the data -- good edge case coverage.
- The empty list test explicitly tests both `nil` slice and empty `[]listRow{}` -- addresses the core Go nil-slice-to-null gotcha from both angles.
- The indentation test verifies lines with spaces have indent as a multiple of 2 but does not check for the specific `"  "` prefix vs `"    "` prefix pattern.

### Spec Compliance

| Criterion | Status | Notes |
|-----------|--------|-------|
| Implements full Formatter interface | PASS | Compile-time verified |
| Empty list -> `[]` | PASS | Both nil and empty slice tested |
| blocked_by/children always `[]` | PASS | Explicit `make([]..., 0)` |
| parent/closed omitted when null | PASS | `omitempty` on struct tags |
| description always present | PASS | No `omitempty` on description |
| snake_case keys throughout | PASS | All json tags use snake_case |
| Stats nested with 5 priority entries | PASS | Hard-coded loop 0..4 |
| All output valid JSON | PASS | Tested across all methods |
| 2-space indented | PASS | `json.MarshalIndent(v, "", "  ")` |

**Deviation**: V4 adds `ParentTitle` (`"parent_title,omitempty"`) field to `jsonTaskDetail`, which is not in the spec. This is additional data not called for by the plan.

### golang-pro Skill Compliance

| Rule | Status | Notes |
|------|--------|-------|
| Handle all errors explicitly | PASS | All errors checked and returned with wrapping |
| Write table-driven tests with subtests | PARTIAL | `TestJSONFormatter_ValidJSON` uses table-driven; most other tests are individual subtests |
| Document all exported functions/types | PASS | All exported items documented |
| Propagate errors with `fmt.Errorf("%w", err)` | PASS | `fmt.Errorf("json marshal error: %w", err)` |
| No ignored errors | PASS | |
| No panic for error handling | PASS | |

---

## V5 Implementation

### Architecture & Design

**File**: `internal/cli/json_formatter.go` (232 lines at commit 8c0ec68)

V5 also defines `JSONFormatter` as a zero-size struct. The V5 Formatter interface at this commit uses **`interface{}` parameters**:

```go
type Formatter interface {
    FormatTaskList(w io.Writer, data interface{}) error
    FormatTaskDetail(w io.Writer, data interface{}) error
    FormatTransition(w io.Writer, data interface{}) error
    FormatDepChange(w io.Writer, data interface{}) error
    FormatStats(w io.Writer, data interface{}) error
    FormatMessage(w io.Writer, msg string)
}
```

Key design decisions:

1. **`interface{}` method signatures**: All methods except `FormatMessage` accept `interface{}` as the data parameter, requiring runtime type assertions inside each method. This is a deliberate design choice allowing different formatters to receive different underlying types without changing the interface.

2. **Mixed visibility for data types**: The V5 JSON formatter reads from `*showData` (unexported type with unexported fields) for task detail, `[]TaskRow` (exported type) for list, `*StatsData` (exported) for stats, `*TransitionData` (exported) for transition, and `*DepChangeData` (exported) for dep change. The asymmetry between `*showData` (unexported) and `*TransitionData` (exported) is notable.

3. **Package-level helper**: `writeJSON` is a package-level function (`func writeJSON(...)`) rather than a method on `JSONFormatter`. This means other formatters could also call it, improving reusability.

4. **No quiet mode in interface**: V5's `Formatter` interface does not include `quiet` parameters. Quiet mode is presumably handled at a higher level (in the `FormatConfig`).

5. **FormatMessage returns nothing**: V5's `FormatMessage` has no return value (`func (f *JSONFormatter) FormatMessage(w io.Writer, msg string)`), and the implementation calls `writeJSON` with a `//nolint:errcheck` comment to suppress the linter. This is a deliberate API simplification.

6. **No ParentTitle field**: V5's `jsonTaskDetail` does not include `parent_title` -- it stays strictly within the spec.

7. **Shared text formatters**: V5's `format.go` includes `formatTransitionText`, `formatDepChangeText`, and `formatMessageText` functions shared by ToonFormatter and PrettyFormatter. This indicates a more cohesive shared-utility architecture.

### Code Quality

- **Type assertion boilerplate**: Each method that takes `interface{}` starts with a type assertion block:
  ```go
  d, ok := data.(*showData)
  if !ok {
      return fmt.Errorf("FormatTaskDetail: expected *showData, got %T", data)
  }
  ```
  This adds 4 lines of boilerplate per method (20 lines total across 5 methods), loses compile-time safety, and introduces runtime error paths that can only be caught if tested. This is a significant anti-pattern in Go.

- **Write method efficiency**: Uses `w.Write(append(data, '\n'))` instead of `fmt.Fprintf`. The `append` approach is slightly more efficient (avoids format string parsing) but creates a temporary allocation by appending to the byte slice returned by `MarshalIndent`.

- **Error wrapping**: Uses `fmt.Errorf("marshaling JSON: %w", err)` -- slightly more idiomatic phrasing than V4's "json marshal error".

- **Comments**: All types and methods are documented. Comments on the interface methods include "Data must be X" notes documenting the expected concrete type -- a necessary evil when using `interface{}`.

- **Field ordering in jsonTaskDetail**: Places `Description` after `Closed` and before `BlockedBy`. V4 places `Description` after `Priority` and before `Parent`. The spec does not mandate field ordering, but V4's ordering (description near the top, grouped with identifying fields) feels more natural.

### Test Coverage

**File**: `internal/cli/json_formatter_test.go` (682 lines at commit 8c0ec68)

V5 tests are organized into 5 top-level test functions:

| Spec Test | V5 Coverage |
|-----------|-------------|
| "it formats list as JSON array" | `TestJSONFormatterFormatTaskList/"it formats list as JSON array"` |
| "it formats empty list as [] not null" | `TestJSONFormatterFormatTaskList/"it formats empty list as [] not null"` |
| "it formats show with all fields" | `TestJSONFormatterFormatTaskDetail/"it formats show with all fields"` |
| "it omits parent/closed when null" | `TestJSONFormatterFormatTaskDetail/"it omits parent/closed when null"` |
| "it includes blocked_by/children as empty arrays" | `TestJSONFormatterFormatTaskDetail/"it includes blocked_by/children as empty arrays"` |
| "it formats description as empty string not null" | `TestJSONFormatterFormatTaskDetail/"it formats description as empty string not null"` |
| "it uses snake_case for all keys" | Split across multiple subtests: `"it uses snake_case keys for list"`, `"it uses snake_case for all keys in show"`, `"it uses snake_case keys for stats"` |
| "it formats stats as structured nested object" | `TestJSONFormatterFormatStats/"it formats stats as structured nested object"` |
| "it includes 5 priority rows even at zero" | `TestJSONFormatterFormatStats/"it includes 5 priority rows even at zero"` |
| "it formats transition/dep/message as JSON objects" | Split: `"it formats transition as JSON object"`, `"it formats dep change as JSON object"`, `"it formats message as JSON object"` |
| "it produces valid parseable JSON" | `TestJSONFormatterFormatTransitionDepMessage/"it produces valid parseable JSON for all formats"` |

Additional tests:
- `TestJSONFormatterImplementsInterface` -- compile-time interface check
- `TestJSONFormatterFormatTaskList/"it produces valid parseable JSON for list"` -- separate validity check
- `TestJSONFormatterFormatTaskList/"it uses 2-space indentation for list"` -- checks for `"\n  {"` and `"\n    \"id\""` patterns plus no tabs

**Test quality observations**:
- The empty list test uses `strings.TrimSpace(buf.String())` and checks for literal `"[]"`, which is a more direct and readable assertion than V4's approach of parsing through `json.RawMessage`.
- The empty arrays test for `blocked_by`/`children` uses **both** string matching (`strings.Contains(got, "\"blocked_by\": []")`) and JSON parsing -- a belt-and-suspenders approach that catches both the serialization form and the semantic meaning.
- V5 splits the snake_case test across list, show, and stats, providing more granular coverage per output type. V4 only tests snake_case on show detail.
- V5's stats test validates exact `expectedCounts` in a loop, providing tighter correctness checking than V4's stats test which only verifies the zero case.
- V5 does NOT test with `nil` input to `FormatTaskList` (only `[]TaskRow{}`). V4 tests both `nil` and empty slice. This is a gap in V5 since the spec explicitly calls out the Go nil-slice-to-null gotcha.
- V5 does NOT test special characters (quotes, newlines) in the valid-JSON test. V4's `TestJSONFormatter_ValidJSON` includes `"Task with \"quotes\" and special chars"` and `"Line 1\nLine 2\n\"Quoted\""`. This is a notable edge case gap in V5.
- V5's indentation test is more precise: it checks for the exact `"\n  {"` and `"\n    \"id\""` substrings, confirming the 2-level nesting pattern. V4 only checks that indent is a multiple of 2.

### Spec Compliance

| Criterion | Status | Notes |
|-----------|--------|-------|
| Implements full Formatter interface | PASS | Compile-time verified |
| Empty list -> `[]` | PASS | Tested with empty slice (not nil) |
| blocked_by/children always `[]` | PASS | Explicit `make([]..., 0)` |
| parent/closed omitted when null | PASS | `omitempty` on struct tags |
| description always present | PASS | No `omitempty` on description |
| snake_case keys throughout | PASS | All json tags use snake_case |
| Stats nested with 5 priority entries | PASS | Hard-coded loop 0..4 |
| All output valid JSON | PASS | Tested across all methods |
| 2-space indented | PASS | `json.MarshalIndent(v, "", "  ")` |

**No deviations from spec.**

### golang-pro Skill Compliance

| Rule | Status | Notes |
|------|--------|-------|
| Handle all errors explicitly | FAIL | `FormatMessage` ignores the error from `writeJSON` with `//nolint:errcheck` |
| Write table-driven tests with subtests | PARTIAL | No table-driven tests; all individual subtests |
| Document all exported functions/types | PASS | All documented |
| Propagate errors with `fmt.Errorf("%w", err)` | PASS | `fmt.Errorf("marshaling JSON: %w", err)` |
| No ignored errors | FAIL | `//nolint:errcheck` in `FormatMessage` |
| No panic for error handling | PASS | |
| Use reflection without performance justification | BORDERLINE | `interface{}` + runtime type assertions are reflection-adjacent |

---

## Comparative Analysis

### Where V4 is Better

1. **Type-safe Formatter interface**: V4's `Formatter` interface uses concrete types (`[]listRow`, `TaskDetail`, `StatsData`) in method signatures. This provides compile-time safety -- if a caller passes the wrong type, the compiler catches it. V5 uses `interface{}` on all methods (except `FormatMessage`), requiring runtime type assertions and introducing an entire class of bugs that can only be caught at runtime. This is a fundamental Go anti-pattern. The Go proverb "accept interfaces, return structs" does not mean "accept empty interface everywhere." V5's approach is more aligned with dynamically-typed languages than idiomatic Go.

2. **Error handling on FormatMessage**: V4's `FormatMessage` returns `error`, allowing callers to handle write failures. V5's returns nothing and silently ignores the error with `//nolint:errcheck`. This violates the golang-pro skill's "handle all errors explicitly" and "no ignored errors" rules. While message formatting rarely fails, the principle matters.

3. **Nil slice edge case testing**: V4 explicitly tests `FormatTaskList` with a `nil` input (not just empty slice), which is the exact Go nil-slice-to-null gotcha called out in the task plan's Edge Cases section. V5 only tests with `[]TaskRow{}`.

4. **Special character testing**: V4's `TestJSONFormatter_ValidJSON` includes data with embedded quotes, newlines, and escaped characters (`"Task with \"quotes\""`, `"Line 1\nLine 2\n\"Quoted\""`) to verify the JSON encoder handles them correctly. V5 omits this edge case entirely.

5. **Table-driven valid-JSON test**: V4 uses a proper table-driven test in `TestJSONFormatter_ValidJSON` with a `[]struct{name string; fn func() (string, error)}` slice, iterating with `t.Run(tc.name, ...)`. V5's equivalent valid-JSON test is a single large function body that manually creates 6 buffers sequentially. V4's approach is more maintainable and more aligned with the golang-pro skill's "write table-driven tests with subtests" requirement.

6. **Quiet mode support**: V4's `FormatTaskList` and `FormatDepChange` handle quiet mode directly, producing a JSON string array of IDs for quiet list output and no output for quiet dep change. V5's interface has no quiet concept; it is handled elsewhere. While V5's approach separates concerns, it means the JSON formatter cannot produce a JSON-flavored quiet output (e.g., `["tick-a1b2", "tick-c3d4"]`) -- the quiet handling must be done before the formatter is called.

### Where V5 is Better

1. **Strict spec adherence (no extra fields)**: V5's `jsonTaskDetail` matches the spec exactly. V4 adds `ParentTitle` (`parent_title`) which is not specified in the task plan. Extra fields in JSON output can cause problems for consumers that do strict schema validation.

2. **Package-level `writeJSON` function**: V5's `writeJSON` is a free function, not a method on `JSONFormatter`. This is better design -- it carries no implicit state and could be shared across formatters if needed (e.g., a hypothetical JSONL formatter). V4's `writeJSON` is a method on `*JSONFormatter` despite not using the receiver, which is misleading.

3. **More granular snake_case testing**: V5 tests snake_case compliance across three separate output types (list, show, stats), not just show. This catches potential issues where different output types might have different key conventions.

4. **Precise indentation testing**: V5's indentation test checks for the exact expected pattern (`"\n  {"` and `"\n    \"id\""` plus no tabs), which is a more meaningful assertion than V4's "indent is a multiple of 2" check. V4's check would pass for 4-space or 6-space indentation too.

5. **Shared text formatting utilities**: V5's `format.go` includes `formatTransitionText`, `formatDepChangeText`, and `formatMessageText` as shared helpers. This reduces duplication across ToonFormatter and PrettyFormatter. V4 does not show similar shared utilities.

6. **Structured data objects for transitions/deps**: V5 uses `*TransitionData` and `*DepChangeData` structs to bundle parameters, instead of V4's individual parameters (`id string, oldStatus string, newStatus string`). Struct-based parameters are more maintainable -- adding a field does not require changing the interface signature.

7. **`FormatConfig` struct**: V5 introduces `FormatConfig` with `Format`, `Quiet`, and `Verbose` fields, separating formatting configuration from the formatter methods. This is cleaner architecture than threading `quiet` through individual method signatures.

### Differences That Are Neutral

1. **Naming**: V4 uses `jsonListRow`, V5 uses `jsonListItem`. V4 uses `jsonStats`, V5 uses `jsonStats`. Both are clear.

2. **Field ordering in jsonTaskDetail**: V4 puts `Description` after `Priority`; V5 puts it after `Closed`. Neither ordering is specified.

3. **Test function naming**: V4 uses underscores (`TestJSONFormatter_FormatTaskList`), V5 uses PascalCase (`TestJSONFormatterFormatTaskList`). Both are valid Go conventions.

4. **Write approach**: V4 uses `fmt.Fprintf(w, "%s\n", data)`, V5 uses `w.Write(append(data, '\n'))`. Both produce identical output. V5 is marginally more efficient; V4 is arguably more readable.

5. **V4's `listRow` has exported fields, V5's `listRow` has unexported fields**: V4's `listRow` uses `ID`, `Title`, etc. while V5's uses `id`, `title`. The JSON formatter in V4 reads from `listRow` directly; V5's reads from exported `TaskRow`. Both work but represent different layering choices.

---

## Verdict

**Winner: V4**

V4 wins on the most important dimension: **type safety through concrete interface signatures**. The V5 decision to use `interface{}` parameters on the `Formatter` interface is a significant architectural regression that:

- Violates Go's core design philosophy of compile-time type safety
- Introduces 20+ lines of runtime type assertion boilerplate across the implementation
- Creates an entire class of bugs detectable only at runtime
- Forces "Data must be X" documentation on every interface method because the compiler cannot enforce it
- Is borderline reflection usage, which the golang-pro skill says to avoid "without performance justification"

V4 also wins on **error handling discipline** (`FormatMessage` returns `error` vs. V5's `//nolint:errcheck`), **edge case testing** (nil slice + special characters), and **table-driven test structure** (for the valid-JSON test).

V5 has legitimate advantages in spec adherence (no extra `ParentTitle` field), `FormatConfig` separation, package-level `writeJSON`, and structured data objects for transitions. These are good design choices. But they are outweighed by the `interface{}` interface anti-pattern, which affects every single method call and every single test case, and which fundamentally undermines the value of Go's type system.

If V5 had used concrete types in the Formatter interface (as the current on-disk version appears to have been refactored to do in a later commit), the verdict would be much closer and might favor V5 for its cleaner separation of concerns and stricter spec compliance. But at the evaluated commit (8c0ec68), the `interface{}` approach is a clear liability.
