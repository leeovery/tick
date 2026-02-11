# Task tick-core-4-2: TOON formatter — list, show, stats output

## Task Summary

This task requires implementing a concrete `ToonFormatter` that satisfies the `Formatter` interface, producing TOON (Token-Oriented Object Notation) output optimized for AI agent consumption with 30-60% token savings over JSON. Key requirements:

- `FormatTaskList`: schema header with count and field names, indented data rows; zero tasks produce `tasks[0]{...}:` with no rows
- `FormatTaskDetail`: multi-section output with dynamic schema (parent/closed omitted when null); blocked_by/children always present (even `[0]`); description omitted when empty, multiline as indented lines
- `FormatStats`: summary section + `by_priority` with always 5 rows (0-4)
- `FormatTransition`, `FormatDepChange`, `FormatMessage`: plain text passthrough
- Escaping delegated to `github.com/toon-format/toon-go`

Acceptance criteria:
1. Implements full Formatter interface
2. List output matches spec TOON format exactly
3. Show output multi-section with dynamic schema
4. blocked_by/children always present, description conditional
5. Stats produces summary + 5-row by_priority
6. Escaping handled by toon-go
7. All output matches spec examples

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Implements full Formatter interface | PASS — compile-time check via `var _ Formatter = &ToonFormatter{}` in test | PASS — compile-time check via `var _ Formatter = (*ToonFormatter)(nil)` at package level |
| List output matches spec TOON format exactly | PASS — `tasks[2]{id,title,status,priority}:` with indented rows | PASS — identical output format |
| Show output multi-section with dynamic schema | PASS — dynamic schema/values built manually, sections separated by blank lines | PASS — dynamic schema via `toon.Field` slices, sections joined with `\n\n` |
| blocked_by/children always present, description conditional | PASS — always written; description guarded by `strings.TrimSpace(d.description) != ""` | PASS — always written; description guarded by `detail.Task.Description != ""` |
| Stats produces summary + 5-row by_priority | PASS — manual `fmt.Fprintf` with correct format | PASS — uses `encodeToonSingleObject` and `encodeToonSection` |
| Escaping handled by toon-go | PASS — `escapeField()` helper uses toon-go; list uses `toon.MarshalString` | PASS — all encoding delegated to `toon.MarshalString` via generic helpers |
| All output matches spec examples | PARTIAL — timestamps in show output are unquoted (raw), matching spec examples verbatim | PARTIAL — timestamps in show output are quoted (toon-go adds quotes around strings containing colons), deviating from spec example format |

## Implementation Comparison

### Approach

**V5: Manual string building with selective toon-go usage**

V5 takes a hybrid approach. It uses `toon.MarshalString` for the task list (via a `toonListWrapper` struct) and a manual `escapeField()` helper for everything else. The task detail section is built by manually assembling schema and value slices, then joining them with commas:

```go
// V5 toon_formatter.go lines 88-116
schema := []string{"id", "title", "status", "priority"}
values := []string{
    d.id,
    escapeField(d.title),
    d.status,
    strconv.Itoa(d.priority),
}
// ...
if _, err := fmt.Fprintf(w, "task{%s}:\n", strings.Join(schema, ",")); err != nil {
    return err
}
if _, err := fmt.Fprintf(w, "  %s\n", strings.Join(values, ",")); err != nil {
    return err
}
```

V5 defines its own formatter-specific data types (`TaskRow`, `StatsData`, `TransitionData`, `DepChangeData`) as exported structs in the toon_formatter.go file, plus uses `showData` and `relatedTask` (unexported) from show.go.

The `escapeField` function is a custom wrapper that only invokes toon-go when special characters are detected:

```go
// V5 toon_formatter.go lines 213-224
func escapeField(s string) string {
    if !strings.ContainsAny(s, ",\"\n\\:[]{}") {
        return s
    }
    type wrapper struct {
        V string `toon:"v"`
    }
    out, err := toon.MarshalString(wrapper{V: s})
    if err != nil {
        return s
    }
    return strings.TrimPrefix(out, "v: ")
}
```

**V6: Full toon-go delegation with generic helpers**

V6 delegates all encoding to toon-go through two generic helper functions:

```go
// V6 toon_formatter.go (commit) lines 187-213
func encodeToonSection[T any](name string, rows []T) string {
    obj := toon.NewObject(toon.Field{Key: name, Value: rows})
    s, err := toon.MarshalString(obj)
    if err != nil {
        return fmt.Sprintf("%s[0]:", name)
    }
    return s
}

func encodeToonSingleObject[T any](name string, value T) string {
    obj := toon.NewObject(toon.Field{Key: name, Value: []T{value}})
    s, err := toon.MarshalString(obj)
    if err != nil {
        return name + ":"
    }
    return strings.Replace(s, name+"[1]", name, 1)
}
```

For the task detail section, V6 uses `toon.Field` and `toon.NewObject` directly:

```go
// V6 toon_formatter.go (commit) lines 135-153
func buildTaskSection(t task.Task) string {
    var fields []toon.Field
    fields = append(fields,
        toon.Field{Key: "id", Value: t.ID},
        toon.Field{Key: "title", Value: t.Title},
        toon.Field{Key: "status", Value: string(t.Status)},
        toon.Field{Key: "priority", Value: t.Priority},
    )
    // ...
    row := toon.NewObject(fields...)
    wrapper := toon.NewObject(toon.Field{Key: "task", Value: []toon.Object{row}})
    s, err := toon.MarshalString(wrapper)
    if err != nil {
        return "task:"
    }
    return strings.Replace(s, "task[1]", "task", 1)
}
```

V6 uses shared domain types (`task.Task`, `TaskDetail`, `RelatedTask`, `Stats`) from format.go rather than formatter-specific data types, and accepts `[]task.Task` directly.

**Key structural differences:**

1. **Interface signatures**: V5 uses `(w io.Writer, data *Type) error` pattern (writer + error return). V6 uses `(data Type) string` pattern (returns string). This is a fundamental design difference established by task 4-1 (Formatter interface definition), not by this task.

2. **Data types**: V5 defines its own `TaskRow`, `StatsData`, `TransitionData`, `DepChangeData` exported structs alongside the formatter. V6 uses shared types (`task.Task`, `Stats`, `TaskDetail`, `RelatedTask`) from the format.go layer, meaning the formatter works directly with domain types.

3. **Escaping strategy**: V5 uses a custom `escapeField` function with a fast-path check. V6 relies entirely on toon-go's marshaling to handle escaping automatically.

4. **Single-object scope hack**: V6 uses `strings.Replace(s, "task[1]", "task", 1)` to convert toon-go's array output into single-object scope. This is a brittle workaround for toon-go not natively supporting single-object scope. V5 avoids this by manually building the header string.

### Code Quality

**Error handling:**

V5 handles every `fmt.Fprintf` error explicitly, propagating via `error` return:

```go
// V5 toon_formatter.go lines 118-124
if _, err := fmt.Fprintln(w); err != nil {
    return err
}
if err := writeRelatedSection(w, "blocked_by", d.blockedBy); err != nil {
    return err
}
```

V6 returns strings, so write errors are impossible (they occur later at the call site). However, V6 silently swallows toon-go marshal errors by returning fallback strings:

```go
// V6 toon_formatter.go (commit) lines 149-152
s, err := toon.MarshalString(wrapper)
if err != nil {
    return "task:"
}
```

This is a tradeoff: V6 is cleaner at the method level but hides potential marshaling failures. V5 would propagate such errors. In practice, marshaling failures with well-formed data are extremely unlikely.

**Type safety:**

V5's commit-time code (the original diff) used `interface{}` parameters:
```go
// V5 original commit
func (f *ToonFormatter) FormatTaskList(w io.Writer, data interface{}) error {
    rows, ok := data.([]TaskRow)
    if !ok {
        return fmt.Errorf("FormatTaskList: expected []TaskRow, got %T", data)
    }
```

This was later refactored in task T6-7 to use typed signatures. The V5 worktree shows the final typed version:
```go
// V5 worktree (post-T6-7)
func (f *ToonFormatter) FormatTaskList(w io.Writer, rows []TaskRow) error {
```

V6's commit already used typed parameters from the start:
```go
// V6 commit
func (f *ToonFormatter) FormatTaskList(tasks []task.Task) string {
```

V6 is genuinely better here: it accepted domain types directly from day one, avoiding the need for a later refactoring task.

**Go generics usage:**

V6 uses Go generics for the `encodeToonSection[T any]` and `encodeToonSingleObject[T any]` helpers, which is idiomatic Go 1.18+ and eliminates code duplication across section types. V5 has no generic functions.

**DRY principle:**

V5's commit defines `FormatTransition`, `FormatDepChange`, and `FormatMessage` inline with full implementations. These were later extracted to shared functions (`formatTransitionText`, `formatDepChangeText`, `formatMessageText`) in task T6-4.

V6's commit also defines these inline, but they were later extracted to `baseFormatter` via embedding in task 6-4. The V6 worktree shows `ToonFormatter` embedding `baseFormatter` and only defining `FormatMessage` directly.

Both versions eventually achieved DRY for shared text methods, but through different mechanisms (V5: package-level helper functions; V6: struct embedding).

**Naming:**

V5 uses `TaskRow`, `StatsData`, `TransitionData`, `DepChangeData` -- generic but clear. V6 uses `Stats`, `TaskDetail`, `RelatedTask` -- more concise and closer to domain language. V6's naming is marginally better as it avoids the `Data` suffix pattern.

### Test Quality

**V5 test functions (6 top-level, 16 subtests):**

1. `TestToonFormatterImplementsInterface`
   - "it implements the full Formatter interface"

2. `TestToonFormatterFormatTaskList`
   - "it formats list with correct header count and schema"
   - "it formats zero tasks as empty section"

3. `TestToonFormatterFormatTaskDetail`
   - "it formats show with all sections"
   - "it omits parent/closed from schema when null"
   - "it renders blocked_by/children with count 0 when empty"
   - "it omits description section when empty"
   - "it renders multiline description as indented lines"
   - "it includes closed in schema when present"

4. `TestToonFormatterEscaping`
   - "it escapes commas in titles"

5. `TestToonFormatterFormatStats`
   - "it formats stats with all counts"
   - "it formats by_priority with 5 rows including zeros"

6. `TestToonFormatterFormatTransitionAndDep`
   - "it formats transition as plain text"
   - "it formats dep add as plain text"
   - "it formats dep removed as plain text"
   - "it formats message as plain text"

**V6 test functions (1 top-level, 16 subtests):**

1. `TestToonFormatter`
   - "it formats list with correct header count and schema"
   - "it formats zero tasks as empty section"
   - "it formats zero tasks from nil slice as empty section" (UNIQUE to V6)
   - "it formats show with all sections"
   - "it omits parent and closed from schema when null"
   - "it renders blocked_by and children with count 0 when empty"
   - "it omits description section when empty"
   - "it renders multiline description as indented lines"
   - "it escapes commas in titles"
   - "it formats stats with all counts"
   - "it formats by_priority with 5 rows including zeros"
   - "it formats transition as plain text"
   - "it formats dep change as plain text"
   - "it formats message as plain text"
   - "it includes closed in show schema when present"
   - "it includes both parent and closed in show schema when both present" (UNIQUE to V6)

**Edge case coverage comparison:**

| Edge Case | V5 | V6 |
|-----------|-----|-----|
| Zero tasks (empty slice) | YES | YES |
| Zero tasks (nil slice) | NO | YES |
| Dynamic schema with parent present | YES | YES |
| Dynamic schema without parent/closed | YES | YES |
| Closed field present | YES | YES |
| Both parent AND closed present | NO | YES |
| blocked_by/children zero count | YES | YES |
| Description omitted when empty | YES | YES |
| Multiline description | YES | YES |
| Comma escaping in titles | YES | YES |
| Stats with all counts | YES | YES |
| by_priority with zero counts | YES | YES |
| Transition plain text | YES | YES |
| Dep add plain text | YES | YES |
| Dep remove plain text | YES | YES |
| Message plain text | YES | YES |

V6 has two additional edge cases: nil slice handling and the combined parent+closed scenario.

**Assertion approach:**

V5 uses `bytes.Buffer` and compares full output strings with exact `expected` values. Tests are written with `t.Fatalf` for fatal failures and `t.Errorf` for non-fatal. The pattern is: write to buffer, compare entire output.

V6 returns strings and splits them into sections/lines for targeted assertions. Tests verify individual sections rather than the whole output string, making them more granular. For example:

```go
// V6 test — verifies sections individually
sections := strings.Split(result, "\n\n")
if len(sections) != 4 {
    t.Fatalf("expected 4 sections, got %d: %q", len(sections), result)
}
taskLines := strings.Split(sections[0], "\n")
expectedTaskHeader := "task{id,title,status,priority,parent,created,updated}:"
if taskLines[0] != expectedTaskHeader {
    t.Errorf("task header = %q, want %q", taskLines[0], expectedTaskHeader)
}
```

V5 tests are more brittle (entire output must match exactly) but also more thorough as exact-match catches any deviation. V6 tests are more maintainable but could miss unexpected extra content within sections.

**Test organization:**

V5 uses 6 top-level `Test*` functions organized by feature area. V6 uses a single `TestToonFormatter` function with all subtests nested inside. The skill file recommends "table-driven tests with subtests" -- neither version uses table-driven tests, but both use subtests. V6's single-function approach is less idiomatic for Go, which typically organizes by tested function.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS — code is properly formatted | PASS — code is properly formatted |
| Add context.Context to all blocking operations | N/A — no blocking operations | N/A — no blocking operations |
| Handle all errors explicitly (no naked returns) | PASS — all errors from fmt.Fprint* checked and returned | PASS — toon-go errors checked with fallback returns; no error returns in interface |
| Write table-driven tests with subtests | PARTIAL — uses subtests but no table-driven tests | PARTIAL — uses subtests but no table-driven tests |
| Document all exported functions, types, and packages | PASS — all exported types and functions have godoc comments | PASS — all exported types and functions have godoc comments |
| Propagate errors with fmt.Errorf("%w", err) | PASS — `fmt.Errorf("marshaling task list: %w", err)` on line 78 | PARTIAL — errors are swallowed with fallback returns rather than propagated |
| Ignore errors (avoid _ assignment without justification) | PASS — no ignored errors | PASS — no ignored errors, though marshal errors are silently absorbed |
| Use panic for normal error handling | PASS — no panics | PASS — no panics |
| Use reflection without performance justification | PASS — no reflection used | PASS — no reflection used |

### Spec-vs-Convention Conflicts

**1. Transition arrow character: Unicode `→` vs ASCII `->`**

- **Spec says**: `tick-a3f2b7: open → in_progress` (Unicode right arrow U+2192)
- **V5 chose**: Unicode arrow `\u2192` — matches spec exactly
- **V6 chose** (at commit time): ASCII `->` in `FormatTransition`; later changed to Unicode `\u2192` in baseFormatter (task 6-4)
- **Assessment**: V5 matched spec from the start. V6 initially deviated but was corrected later. At the commit level, V5 is spec-compliant; V6 is not.

**2. Timestamp quoting in show output**

- **Spec example shows**: `tick-a1b2,Setup Sanctum,in_progress,1,tick-e5f6,2026-01-19T10:00:00Z,2026-01-19T14:30:00Z` (unquoted timestamps)
- **TOON format convention**: Strings containing colons should be quoted per TOON escaping rules
- **V5 chose**: Unquoted timestamps (manual string building bypasses toon-go quoting for the task detail section), matching spec example verbatim
- **V6 chose**: Quoted timestamps (`"2026-01-19T10:00:00Z"`) because toon-go's marshaler correctly quotes strings with colons
- **Assessment**: This is a genuine spec-vs-format conflict. The spec examples show unquoted timestamps, but TOON format rules require quoting strings with special characters. V6 follows correct TOON encoding; V5 follows the spec example literally. V6's approach is arguably more correct since the spec says "escaping via toon-go library" (which would quote these), but V5 matches the spec examples as-given. Both are reasonable judgment calls.

**3. Interface design: `io.Writer` + `error` vs `string` returns**

- **Spec says**: Nothing specific about Formatter method signatures
- **Go convention**: `io.Writer` is standard for output rendering; returning strings is simpler but less flexible
- **V5 chose**: Writer pattern with error returns — more Go-idiomatic for output formatting
- **V6 chose**: String returns — simpler, easier to test, but requires caller to handle writing
- **Assessment**: Both are valid Go patterns. The writer pattern composes better with `io.Writer` implementations; the string pattern is easier to test and more functional. Neither is wrong; this was a task 4-1 decision that task 4-2 inherited.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 6 (2 impl + go.mod/sum + 2 docs) | 6 (2 impl + go.mod/sum + 2 docs) |
| Lines added | 660 | 620 |
| Impl LOC | 261 | 213 |
| Test LOC | 393 | 401 |
| Test functions | 6 top-level, 16 subtests | 1 top-level, 16 subtests |

## Verdict

**V6 is the better implementation overall**, with meaningful advantages in several areas:

1. **Type safety from day one**: V6 accepted domain types (`task.Task`, `TaskDetail`, `Stats`) directly in method signatures, avoiding the `interface{}` anti-pattern that V5 used and had to fix in a later task (T6-7). This is a significant design advantage.

2. **Better toon-go integration**: V6's generic `encodeToonSection[T any]` and `encodeToonSingleObject[T any]` helpers demonstrate proper use of Go generics to eliminate duplication. V5 used toon-go only for list encoding and a custom `escapeField` wrapper, manually building most output.

3. **More thorough edge case coverage**: V6 tests nil slice handling and the combined parent+closed schema scenario -- both real-world edge cases that V5 misses.

4. **More concise implementation**: 213 vs 261 implementation LOC, while achieving the same functionality with more delegation to the library.

**V5's advantages are more narrow:**

1. **Spec-literal timestamp format**: V5 produces unquoted timestamps matching spec examples exactly, while V6 produces quoted timestamps (technically more correct per TOON rules, but diverging from spec examples).

2. **Explicit error propagation**: V5's writer+error pattern catches every write failure, while V6 silently swallows toon-go errors with fallback strings.

3. **Unicode arrow from day one**: V5 used the spec-correct Unicode `→` immediately; V6 initially used ASCII `->`.

The type safety and generic helper advantages of V6 outweigh V5's spec-literal output format, as the task plan itself states "escaping via toon-go library" -- and delegating fully to toon-go (which quotes timestamps) is a reasonable interpretation of that requirement. V5's need for a subsequent refactoring task to fix type safety is evidence that its initial design was not as well considered.
