# Task tick-core-1-1: Task model & ID generation

## Task Summary

This task defines the foundational `Task` struct with all 10 fields (id, title, status, priority, description, blocked_by, parent, created, updated, closed), a `Status` string enum with constants (open, in_progress, done, cancelled), and a `tick-{6 hex}` ID generator using `crypto/rand` with collision retry up to 5 attempts. It also requires input ID normalization to lowercase, title validation (non-empty, max 500 chars, no newlines, trim whitespace), priority validation (0-4 range), self-reference rejection in blocked_by and parent, and ISO 8601 UTC timestamps.

### Acceptance Criteria (from plan)

1. Task struct has all 10 fields with correct Go types
2. ID format matches `tick-{6 hex chars}` pattern
3. IDs are generated using `crypto/rand`
4. Collision retry works up to 5 times then errors
5. Input IDs are normalized to lowercase
6. Title validation enforces non-empty, max 500 chars, no newlines, trims whitespace
7. Priority validation rejects values outside 0-4
8. Self-references in `blocked_by` and `parent` are rejected
9. Timestamps are ISO 8601 UTC

## Acceptance Criteria Compliance

| Criterion | V4 | V5 |
|-----------|-----|-----|
| Task struct has all 10 fields with correct Go types | PASS -- struct at line 29 has ID(string), Title(string), Status(Status), Priority(int), Description(string), BlockedBy([]string), Parent(string), Created(time.Time), Updated(time.Time), Closed(*time.Time) | PASS -- identical struct at line 44, same 10 fields and types |
| ID format matches `tick-{6 hex chars}` pattern | PASS -- `GenerateID` at line 118 produces `"tick-" + hex.EncodeToString(b)` from 3 random bytes | PASS -- `GenerateID` at line 152 produces `idPrefix + hex.EncodeToString(b)` from `idRandSize` (3) bytes |
| IDs are generated using `crypto/rand` | PASS -- imports `crypto/rand`, uses `rand.Read(b)` at line 123 | PASS -- imports `crypto/rand`, uses `rand.Read(b)` at line 155 |
| Collision retry works up to 5 times then errors | PASS -- loop `for attempt := 0; attempt < maxRetries; attempt++` with `maxRetries = 5` at line 119-121 | PASS -- loop `for i := 0; i < idRetries; i++` with `idRetries = 5` at line 153 |
| Input IDs are normalized to lowercase | PASS -- `NormalizeID` at line 138 returns `strings.ToLower(id)` | PASS -- `NormalizeID` at line 167 returns `strings.ToLower(id)` |
| Title validation: non-empty, max 500 chars, no newlines, trims whitespace | PASS -- `ValidateTitle` at line 145 trims, checks empty, checks newlines, checks `utf8.RuneCountInString > 500` | PASS -- `ValidateTitle` at line 170 trims, checks empty, checks rune count > 500, checks newlines |
| Priority validation rejects values outside 0-4 | PASS -- `ValidatePriority` at line 164 rejects `< 0 || > 4` | PASS -- `ValidatePriority` at line 195 rejects `< 0 || > 4` |
| Self-references in `blocked_by` and `parent` rejected | PASS -- `ValidateBlockedBy` (line 172) and `ValidateParent` (line 182) do direct string equality checks | PASS -- `ValidateBlockedBy` (line 204) and `ValidateParent` (line 214) use `NormalizeID` for case-insensitive comparison |
| Timestamps are ISO 8601 UTC | PARTIAL -- timestamps are `time.Time` truncated to seconds in UTC, but no explicit formatting constant or JSON marshaling to ISO 8601 strings; default `time.Time` JSON uses RFC 3339 with nanoseconds | PASS -- defines `TimestampFormat = "2006-01-02T15:04:05Z"`, implements `MarshalJSON`/`UnmarshalJSON` to format timestamps as exact ISO 8601 strings, and provides `FormatTimestamp` helper |

## Implementation Comparison

### Approach

**V4: All-in-one `NewTask` constructor with built-in validation and ID generation.**

V4 couples task creation, validation, and ID generation into a single `NewTask` function (line 58):

```go
func NewTask(title string, opts *TaskOptions, exists func(id string) bool) (*Task, error) {
```

This function accepts a title, optional `TaskOptions` struct (containing Priority *int, Description, BlockedBy, Parent), and an `exists` function for collision checking. It validates title, priority, generates ID, validates blocked_by/parent against the generated ID, and returns a fully-constructed `*Task`. The design is monolithic -- callers cannot construct a Task without going through `NewTask`.

**V5: Decomposed approach with thin `NewTask` and separate concerns.**

V5 takes a fundamentally different architectural approach. Its `NewTask` is a simple constructor (line 132):

```go
func NewTask(id, title string) Task {
    now := time.Now().UTC().Truncate(time.Second)
    return Task{
        ID:       id,
        Title:    title,
        Status:   StatusOpen,
        Priority: DefaultPriority,
        Created:  now,
        Updated:  now,
    }
}
```

This accepts an already-generated ID and title, sets defaults, and returns a value-type `Task` (not a pointer). Validation functions exist separately and are expected to be called by callers before constructing the task. This is a more composable design.

Additionally, V5 includes custom JSON serialization (lines 74-128) with `MarshalJSON`/`UnmarshalJSON` methods and an internal `taskJSON` helper struct to ensure timestamps are formatted as ISO 8601 strings rather than Go's default RFC 3339 with nanoseconds. V5 also exports `DefaultPriority` and `TimestampFormat` as package-level constants, and defines named constants for magic values (`idPrefix`, `idRetries`, `idRandSize`, `maxTitleLen`).

**Key structural difference:** V4's `NewTask` orchestrates everything (validation, generation, construction) in one call. V5 separates generation (`GenerateID`), validation (`ValidateTitle`, `ValidatePriority`, etc.), and construction (`NewTask`) into independent functions the caller composes. This is more Go-idiomatic -- small functions that do one thing.

**Self-reference checking:** V4 uses direct string equality (`dep == taskID`), while V5 normalizes both sides via `NormalizeID` for case-insensitive comparison. This is a genuine improvement -- if IDs can be input in mixed case (which `NormalizeID` exists to handle), self-reference detection should also be case-insensitive.

### Code Quality

**Constants and magic values:**

V4 defines two constants (lines 50-53):
```go
const (
    maxTitleLength  = 500
    defaultPriority = 2
)
```

V5 uses a single `const` block with more named values (lines 14-25):
```go
const (
    idPrefix    = "tick-"
    idRetries   = 5
    idRandSize  = 3 // 3 bytes = 6 hex chars
    maxTitleLen = 500
    DefaultPriority = 2
    TimestampFormat = "2006-01-02T15:04:05Z"
)
```

V5 eliminates all magic values from function bodies. V4 has `"tick-"` hardcoded at line 127, `5` hardcoded as `maxRetries` inside `GenerateID`, and `3` hardcoded in `make([]byte, 3)`. V5 also exports `DefaultPriority` and `TimestampFormat` for downstream use.

**Error messages:**

Both versions use `fmt.Errorf` with descriptive messages. V4 wraps title errors with an extra layer (line 61):
```go
return nil, fmt.Errorf("invalid title: %w", err)
```
V5's validation functions return errors directly without wrapping.

V4 uses `%w` error wrapping for the `rand.Read` failure (line 124):
```go
return "", fmt.Errorf("failed to generate random bytes: %w", err)
```
V5 also wraps but with slightly different phrasing (line 156):
```go
return "", fmt.Errorf("reading random bytes: %w", err)
```

Both comply with the skill's "Propagate errors with fmt.Errorf("%w", err)" requirement.

**Return types:**

V4's `NewTask` returns `(*Task, error)` -- a pointer and error. V5's `NewTask` returns `Task` (value type, no error). The V5 approach is appropriate because its `NewTask` cannot fail -- it just sets defaults. Validation happens elsewhere. The V4 approach bundles validation into construction, making `*Task` the correct choice since the function can fail.

**JSON serialization:**

V4 uses default `json:"created"` and `json:"updated"` tags on `time.Time` fields (lines 37-38), which produces Go's default RFC 3339 format with nanosecond precision (e.g., `"2026-01-19T10:00:00.000000000Z"`). This does NOT match the spec's `YYYY-MM-DDTHH:MM:SSZ` format.

V5 marks timestamp fields with `json:"-"` (lines 52-54) and implements `MarshalJSON`/`UnmarshalJSON` (lines 74-128) via an intermediate `taskJSON` struct. This produces exact ISO 8601 output like `"2026-01-19T10:00:00Z"`.

**Documentation:**

Both versions document all exported types and functions. V4's package doc: `"Package task provides the core task model and validation for Tick."` V5's: `"Package task defines the core Task model and ID generation for Tick."` Both are adequate.

**`ValidateParent` empty-string handling:**

V4 (line 182-185):
```go
func ValidateParent(taskID string, parent string) error {
    if parent == taskID {
        return fmt.Errorf("task %s cannot be its own parent", taskID)
    }
    return nil
}
```

V5 (line 214-222):
```go
func ValidateParent(taskID string, parent string) error {
    if parent == "" {
        return nil
    }
    if NormalizeID(parent) == NormalizeID(taskID) {
        return fmt.Errorf("task %s cannot be its own parent", taskID)
    }
    return nil
}
```

V5 explicitly handles the empty-parent case with an early return, making the contract clearer. V4 happens to work for empty parents (empty string won't equal a `tick-xxx` ID), but the intent is implicit.

### Test Quality

#### V4 Test Functions (7 top-level, 23 subtests)

**`TestGenerateID`** (3 subtests):
- `"it generates IDs matching tick-{6 hex} pattern"` -- generates 20 IDs, checks regex `^tick-[0-9a-f]{6}$`
- `"it retries on collision up to 5 times"` -- collides on first 4 attempts, succeeds on 5th, verifies attempt count
- `"it errors after 5 collision retries"` -- always collides, checks exact error message

**`TestNormalizeID`** (1 subtest with table-driven):
- `"it normalizes IDs to lowercase"` -- 3 cases: uppercase, mixed case, already lowercase

**`TestValidateTitle`** (4 subtests):
- `"it rejects empty title"` -- table: empty string, only spaces, only tabs
- `"it rejects title exceeding 500 characters"` -- 501 char string
- `"it counts characters not bytes for title length"` -- 500 CJK chars (pass), 501 CJK chars (fail)
- `"it rejects title with newlines"` -- table: LF, CR, CRLF
- `"it trims whitespace from title"` -- table: leading, trailing, both, tabs

**`TestValidatePriority`** (2 subtests):
- `"it rejects priority outside 0-4"` -- table: -1, 5, 100
- `"it accepts valid priorities"` -- loop 0-4

**`TestValidateBlockedBy`** (2 subtests):
- `"it rejects self-reference in blocked_by"` -- task ID in list
- `"it accepts valid blocked_by references"` -- no self-reference

**`TestValidateParent`** (2 subtests):
- `"it rejects self-reference in parent"` -- same ID
- `"it accepts valid parent reference"` -- different ID

**`TestNewTask`** (3 subtests):
- `"it sets default priority to 2 when not specified"` -- nil opts
- `"it sets created and updated timestamps to current UTC time"` -- bracket test with before/after, checks equal, checks UTC location
- `"it has all 10 fields with correct types"` -- creates task with all options, verifies every field

Helper: `intPtr(i int) *int`

#### V5 Test Functions (10 top-level, 30 subtests)

**`TestGenerateID`** (3 subtests):
- `"it generates IDs matching tick-{6 hex} pattern"` -- generates 1 ID (vs V4's 20), checks regex
- `"it retries on collision up to 5 times"` -- uses `atomic.Int32` for thread-safe counting
- `"it errors after 5 collision retries"` -- always collides, checks exact error message

**`TestValidateTitle`** (6 subtests):
- `"it rejects empty title"` -- 3 cases (inline slice, no struct names)
- `"it rejects title exceeding 500 characters"` -- 501 char string
- `"it rejects title with newlines"` -- 3 cases (inline slice)
- `"it trims whitespace from title"` -- single case `"  hello world  "`
- `"it accepts valid title at 500 chars"` -- boundary test at exactly 500 (V4 lacks this)
- `"it counts characters not bytes for max length"` -- 200 CJK chars (V4 tests 500/501 boundary more thoroughly)

**`TestValidatePriority`** (2 subtests):
- `"it rejects priority outside 0-4"` -- 5 invalid values: -1, 5, 10, -100, 999 (more than V4's 3)
- `"it accepts valid priorities 0 through 4"` -- loop 0-4

**`TestValidateBlockedBy`** (4 subtests):
- `"it rejects self-reference in blocked_by"` -- same ID in list
- `"it accepts blocked_by without self-reference"` -- valid list
- `"it accepts empty blocked_by"` -- nil slice (V4 lacks this)
- `"it detects self-reference case-insensitively"` -- `"TICK-A1B2C3"` vs `"tick-a1b2c3"` (V4 lacks this)

**`TestValidateParent`** (4 subtests):
- `"it rejects self-reference in parent"` -- same ID
- `"it accepts different parent ID"` -- different ID
- `"it accepts empty parent"` -- empty string (V4 lacks this)
- `"it detects self-reference case-insensitively"` -- `"TICK-A1B2C3"` vs `"tick-a1b2c3"` (V4 lacks this)

**`TestNormalizeID`** (1 subtest with table-driven):
- `"it normalizes IDs to lowercase"` -- 4 cases (V4 has 3)

**`TestTaskStruct`** (3 subtests):
- `"it has all 10 fields with correct types"` -- constructs struct directly (not via NewTask), verifies all fields
- `"it sets default priority to 2 when not specified"` -- uses NewTask
- `"it sets created and updated timestamps to current UTC time"` -- bracket test

**`TestStatusConstants`** (1 subtest with table-driven):
- `"it defines correct status values"` -- verifies all 4 status string values (V4 lacks this)

**`TestTaskTimestampFormat`** (1 subtest):
- `"it formats timestamps as ISO 8601 UTC"` -- checks FormatTimestamp output (V4 lacks this, and lacks FormatTimestamp)

**`TestTaskJSONSerialization`** (3 subtests):
- `"it omits optional fields when empty"` -- checks description, blocked_by, parent, closed absent; id, updated present
- `"it includes optional fields when set"` -- checks all optional fields present in JSON
- `"it formats timestamps as ISO 8601 UTC in JSON"` -- checks `"created":"2026-01-19T10:00:00Z"` in output

#### Test Coverage Diff

| Edge Case | V4 | V5 |
|-----------|-----|-----|
| ID pattern (multiple generations) | 20 IDs | 1 ID |
| Collision retry (exact count) | Yes | Yes (atomic) |
| Collision exhaustion error message | Yes | Yes |
| Normalize ID cases | 3 cases | 4 cases |
| Empty title (empty, spaces, tabs) | Yes | Yes |
| Title > 500 chars | Yes | Yes |
| Title at exactly 500 chars boundary | No | Yes |
| Title CJK rune counting | 500/501 boundary | 200 chars only |
| Title newlines (LF, CR, CRLF) | Yes | Yes |
| Title whitespace trim (4 cases) | Yes | 1 case only |
| Priority invalid (-1, 5, 100) | 3 cases | 5 cases |
| Priority valid (0-4) | Yes | Yes |
| BlockedBy self-reference | Yes | Yes |
| BlockedBy valid | Yes | Yes |
| BlockedBy empty/nil | No | Yes |
| BlockedBy case-insensitive self-ref | No | Yes |
| Parent self-reference | Yes | Yes |
| Parent valid | Yes | Yes |
| Parent empty string | No | Yes |
| Parent case-insensitive self-ref | No | Yes |
| All 10 fields verified | Yes | Yes |
| Default priority | Yes | Yes |
| Timestamps UTC + bracket check | Yes | Yes |
| Status constants verification | No | Yes |
| FormatTimestamp | No | Yes |
| JSON serialization (omit empty) | No | Yes |
| JSON serialization (include all) | No | Yes |
| JSON ISO 8601 format | No | Yes |

V5 has broader edge case coverage overall, particularly around case-insensitive self-reference detection, empty input handling, boundary conditions, status constants, and JSON serialization. V4 has deeper coverage on a few specific tests (20 ID generations vs 1; 4 whitespace trim cases vs 1; CJK at exact 500/501 boundary vs 200).

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- code is formatted | PASS -- code is formatted |
| Add context.Context to all blocking operations | N/A -- no blocking operations in this task | N/A -- no blocking operations in this task |
| Handle all errors explicitly (no naked returns) | PASS -- all errors checked | PASS -- all errors checked |
| Write table-driven tests with subtests | PASS -- uses t.Run subtests; many tests use table patterns | PASS -- uses t.Run subtests; many tests use table patterns |
| Document all exported functions, types, and packages | PASS -- all exported items documented | PASS -- all exported items documented |
| Propagate errors with fmt.Errorf("%w", err) | PASS -- `rand.Read` error wrapped with %w | PASS -- `rand.Read` error wrapped with %w; UnmarshalJSON errors wrapped with %w |
| Run race detector on tests (-race flag) | UNKNOWN -- not verifiable from code | UNKNOWN -- not verifiable from code |
| Avoid _ assignment without justification | PASS -- no ignored errors | PASS -- no ignored errors |
| No panic for normal error handling | PASS -- no panics | PASS -- no panics |
| No hardcoded configuration | PARTIAL -- `"tick-"`, `5`, `3` are inline in GenerateID | PASS -- all values extracted to named constants (idPrefix, idRetries, idRandSize) |

### Spec-vs-Convention Conflicts

**1. ISO 8601 Timestamp Format in JSON**

- **Spec says:** "All timestamps use ISO 8601 UTC format (YYYY-MM-DDTHH:MM:SSZ)"
- **Go convention:** Go's `time.Time` marshals to RFC 3339 by default (which includes nanoseconds: `2006-01-02T15:04:05.999999999Z07:00`). RFC 3339 is a profile of ISO 8601, so technically compliant, but the spec specifically calls for second-precision with no sub-second component.
- **V4 chose:** Default `time.Time` JSON marshaling. The JSON tags `json:"created"` and `json:"updated"` produce RFC 3339 with nanoseconds. Does not match the spec's exact format.
- **V5 chose:** Custom `MarshalJSON`/`UnmarshalJSON` with `TimestampFormat = "2006-01-02T15:04:05Z"`, exactly matching the spec format.
- **Assessment:** V5 made the correct call. The spec is explicit about the format, and the `time.Time` default includes unnecessary sub-second precision and timezone offset notation that differs from the spec. Since this is a file-based storage tool, exact format control is important for interoperability and human readability.

**2. `NewTask` function signature -- validation bundled vs. separated**

- **Spec says:** "Define `Task` struct", "Implement ID generation", "Validate title", "Validate priority" -- these are listed as separate implementation items.
- **Go convention:** Functions should do one thing. Constructors should construct; validators should validate.
- **V4 chose:** Single `NewTask` that does generation + validation + construction.
- **V5 chose:** `NewTask` only constructs; `GenerateID`, `ValidateTitle`, `ValidatePriority` etc. are independent.
- **Assessment:** Both are reasonable interpretations. V5's decomposed approach is more Go-idiomatic and more testable/composable. V4's approach is more convenient for callers but harder to extend. Neither violates the spec -- the spec says "implement" these things, not "bundle them into one function."

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 5 (go.mod, task.go, task_test.go, 2 docs) | 5 (go.mod, task.go, task_test.go, 2 docs) |
| Lines added | 523 | 656 |
| Impl LOC | 187 | 218 |
| Test LOC | 329 | 421 |
| Top-level test functions | 7 | 10 |
| Subtests (t.Run) | 23 | 30 |

## Verdict

**V5 is the better implementation.**

The decisive factors:

1. **ISO 8601 compliance (spec criterion #9):** V4 gets PARTIAL on the timestamps acceptance criterion. Its `time.Time` fields with default JSON tags will produce RFC 3339 with nanosecond precision, not the spec-required `YYYY-MM-DDTHH:MM:SSZ`. V5 explicitly implements custom `MarshalJSON`/`UnmarshalJSON` with the exact format string, earning a full PASS. Since this is a persistence layer (tasks are stored as JSON), correct serialization format is critical and not something that can be deferred.

2. **Case-insensitive self-reference detection:** The spec says "Normalize IDs to lowercase on input (case-insensitive matching)" and separately says "no self-references" for blocked_by and parent. V4's `ValidateBlockedBy` and `ValidateParent` use direct string equality (`dep == taskID`), meaning they would miss a self-reference where cases differ (e.g., `"TICK-A1B2C3"` vs `"tick-a1b2c3"`). V5 normalizes both sides via `NormalizeID` before comparison, which is the correct behavior given the spec's case-insensitive matching requirement.

3. **Elimination of magic values:** V5 extracts `idPrefix`, `idRetries`, `idRandSize` into named constants. V4 hardcodes `"tick-"`, `5` (as a local const), and `3` inline. The skill constraint says "Hardcode configuration (use functional options or env vars)" in the MUST NOT DO list. V5 is more compliant by making all values named constants.

4. **Broader test coverage:** V5 has 10 top-level test functions with 30 subtests versus V4's 7/23. V5 covers edge cases V4 misses entirely: case-insensitive self-reference detection, empty blocked_by/parent inputs, status constant verification, FormatTimestamp, JSON serialization (3 tests), and the 500-char boundary condition. V4 has slightly deeper coverage on a few specific tests (20 ID generations, more whitespace trim cases, CJK at exact boundary), but these don't compensate for V5's broader coverage.

5. **Architectural composability:** V5's decomposed `NewTask(id, title) Task` is more Go-idiomatic -- it does one thing (construct with defaults) and is easy to test and compose. V4's monolithic `NewTask(title, opts, exists) (*Task, error)` bundles multiple concerns. Both work, but V5's approach will be easier to extend in subsequent tasks.

V4 is not a bad implementation -- it passes nearly all criteria and has solid tests. But V5 is more thorough, more correct on the ISO 8601 requirement, and better aligned with Go idioms and the skill constraints.
