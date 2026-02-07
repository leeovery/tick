# Task tick-core-1-1: Task model & ID generation

## Task Summary

Define the core `Task` struct with all 10 fields (`id`, `title`, `status`, `priority`, `description`, `blocked_by`, `parent`, `created`, `updated`, `closed`), implement `Status` as a string enum (`open`, `in_progress`, `done`, `cancelled`), and build a `tick-{6 hex}` ID generator using `crypto/rand` with collision retry.

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

### Required Tests (from plan)

- "it generates IDs matching tick-{6 hex} pattern"
- "it retries on collision up to 5 times"
- "it errors after 5 collision retries"
- "it normalizes IDs to lowercase"
- "it rejects empty title"
- "it rejects title exceeding 500 characters"
- "it rejects title with newlines"
- "it trims whitespace from title"
- "it rejects priority outside 0-4"
- "it rejects self-reference in blocked_by"
- "it rejects self-reference in parent"
- "it sets default priority to 2 when not specified"
- "it sets created and updated timestamps to current UTC time"

### Required Error Messages

- Collision: "Failed to generate unique ID after 5 attempts - task list may be too large"

---

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Task struct has all 10 fields with correct Go types | PASS -- All 10 fields present. Uses `time.Time` for Created/Updated, `*time.Time` for Closed. | PASS -- All 10 fields present. Same types as V1 plus uses `*time.Time` for Closed. | PARTIAL -- All 10 fields present but uses `string` for Created/Updated/Closed instead of `time.Time`. This is a defensible design choice (ISO 8601 strings) but deviates from typical Go idiom of storing time as `time.Time` internally. |
| 2. ID format matches `tick-{6 hex chars}` | PASS -- `"tick-" + hex.EncodeToString(b)` where `b` is 3 bytes. | PASS -- Identical approach: `idPrefix + hex.EncodeToString(b)`. | PASS -- Identical approach: `idPrefix + hex.EncodeToString(bytes)`. |
| 3. IDs generated using `crypto/rand` | PASS -- `crypto/rand.Read(b)`. | PASS -- `crypto/rand.Read(b)`. | PASS -- `crypto/rand.Read(bytes)`. |
| 4. Collision retry up to 5 then error | PASS -- Loop `maxRetries=5` times, returns error with spec-matching message. | PASS -- Loop `maxIDRetries=5` times, returns error. | PASS -- Loop `maxRetries=5` times, returns error with spec-matching message. |
| 5. IDs normalized to lowercase | PASS -- `NormalizeID()` returns `strings.ToLower(id)`. | PASS -- Identical. | PASS -- Identical. |
| 6. Title validation: non-empty, max 500 chars, no newlines, trims whitespace | PASS -- `ValidateTitle()` trims first, then validates. Separate `TrimTitle()` function. | PASS -- `ValidateTitle()` returns cleaned title, uses `utf8.RuneCountInString` for length (rune-aware). | PARTIAL -- `ValidateTitle()` does NOT trim before validating. `TrimTitle()` is separate. Caller must trim before validating. Uses `len()` (byte-count) not rune-count for 500-char check. |
| 7. Priority rejects outside 0-4 | PASS -- `ValidatePriority()` checks `p < 0 || p > 4`. | PASS -- Uses named constants `minPriority`/`maxPriority`. | PASS -- Same logic, simpler error message. |
| 8. Self-references in `blocked_by`/`parent` rejected | PASS -- Uses case-insensitive comparison via `NormalizeID()`. | PASS -- Same approach. | PASS -- Same approach. |
| 9. Timestamps are ISO 8601 UTC | PASS -- `FormatTimestamp()` uses `"2006-01-02T15:04:05Z"` format. `NewTask()` stores `time.Time` truncated to seconds. | PASS -- `NewTask()` stores `time.Time` truncated to seconds. No separate `FormatTimestamp()` function. | PASS -- `DefaultTimestamps()` formats with `time.RFC3339`. Timestamps stored as strings directly. |

### Collision Error Message Compliance

| Version | Error Message | Matches Spec? |
|---------|--------------|---------------|
| V1 | `"failed to generate unique ID after 5 attempts - task list may be too large"` | YES (lowercase "failed") |
| V2 | `"Failed to generate unique ID after 5 attempts - task list may be too large"` | PARTIAL -- capitalised "Failed" differs from spec which says lowercase in edge cases section but capitalised in no specific place. Inconsistent with Go convention (errors should not be capitalised). |
| V3 | `"failed to generate unique ID after 5 attempts - task list may be too large"` | YES (lowercase "failed") |

---

## Implementation Comparison

### Approach

#### File Organization

All three versions use the same file structure: `internal/task/task.go` and `internal/task/task_test.go` in the `task` package.

#### Task Struct

V1 and V2 use `time.Time` for timestamp fields, which is the idiomatic Go approach:

```go
// V1 & V2 (identical struct definition)
type Task struct {
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Status      Status     `json:"status"`
    Priority    int        `json:"priority"`
    Description string     `json:"description,omitempty"`
    BlockedBy   []string   `json:"blocked_by,omitempty"`
    Parent      string     `json:"parent,omitempty"`
    Created     time.Time  `json:"created"`
    Updated     time.Time  `json:"updated"`
    Closed      *time.Time `json:"closed,omitempty"`
}
```

V3 stores timestamps as strings, which loses type safety:

```go
// V3
type Task struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Status      Status   `json:"status"`
    Priority    int      `json:"priority"`
    Description string   `json:"description,omitempty"`
    BlockedBy   []string `json:"blocked_by,omitempty"`
    Parent      string   `json:"parent,omitempty"`
    Created     string   `json:"created"`
    Updated     string   `json:"updated"`
    Closed      string   `json:"closed,omitempty"`
}
```

V3's `Closed` field is `string` (empty string for unset) rather than `*time.Time` (nil for unset). This means V3 cannot distinguish between "never closed" and "explicitly set to empty string" at the type level, and JSON serialization will include `"closed":""` rather than omitting it entirely (the `omitempty` on string only omits `""`).

#### ExistsFunc Signature

This is a significant design divergence:

```go
// V1: Simple boolean callback
func GenerateID(exists func(string) bool) (string, error)

// V2: Same simple boolean callback
func GenerateID(existsFn func(id string) bool) (string, error)

// V3: Callback that can return its own error
type ExistsFunc func(id string) (bool, error)
func GenerateID(exists ExistsFunc) (string, error)
```

V3's approach is **genuinely better** for production use. A real existence check queries a file/database and can fail. V1/V2 swallow the possibility of lookup errors. V3 defines a named type `ExistsFunc` for clarity, and propagates errors from the lookup:

```go
// V3: error propagation from exists check
collision, err := exists(id)
if err != nil {
    return "", err
}
if !collision {
    return id, nil
}
```

#### NewTask / Task Creation

This is the largest architectural divergence across all three versions.

**V1** takes pre-generated ID and raw values, returns a value type, does zero validation:

```go
// V1: Simple constructor, no validation, no ID generation
func NewTask(id string, title string, priority int) Task {
    if priority < 0 {
        priority = 2
    }
    now := time.Now().UTC().Truncate(time.Second)
    return Task{
        ID: id, Title: title, Status: StatusOpen,
        Priority: priority, Created: now, Updated: now,
    }
}
```

V1 uses a sentinel value (`priority < 0`) to trigger default priority. This means callers must know to pass `-1` for "default", which is non-obvious and error-prone. There is no validation in `NewTask`; the caller must call `ValidateTitle()`, `ValidatePriority()`, etc. separately.

**V2** is a fully integrated factory that generates ID, validates everything, and returns a pointer:

```go
// V2: Full factory with integrated validation and ID generation
func NewTask(title string, opts *TaskOptions, existsFn func(id string) bool) (*Task, error) {
    cleanTitle, err := ValidateTitle(title)  // returns trimmed title
    if err != nil {
        return nil, fmt.Errorf("invalid title: %w", err)
    }
    id, err := GenerateID(existsFn)
    if err != nil {
        return nil, err
    }
    priority := defaultPriority
    if opts != nil && opts.Priority != nil {
        priority = *opts.Priority
    }
    if err := ValidatePriority(priority); err != nil {
        return nil, err
    }
    // ... validates blocked_by and parent too ...
    now := time.Now().UTC().Truncate(time.Second)
    return &Task{
        ID: id, Title: cleanTitle, Status: StatusOpen,
        Priority: priority, Description: optString(opts),
        BlockedBy: blockedBy, Parent: parent,
        Created: now, Updated: now, Closed: nil,
    }, nil
}
```

V2 uses an options struct with a pointer for priority (`*int`) to distinguish "not set" from "set to 0":

```go
type TaskOptions struct {
    Priority    *int
    Description string
    BlockedBy   []string
    Parent      string
}
```

V2's `ValidateTitle()` also returns the cleaned (trimmed) title, eliminating the separate `TrimTitle()` function:

```go
// V2: ValidateTitle returns cleaned title
func ValidateTitle(title string) (string, error) {
    trimmed := strings.TrimSpace(title)
    if trimmed == "" { return "", errors.New("title is required and cannot be empty") }
    if strings.ContainsAny(trimmed, "\n\r") { return "", errors.New("title cannot contain newlines") }
    if utf8.RuneCountInString(trimmed) > maxTitleLength { ... }
    return trimmed, nil
}
```

**V3** has no `NewTask` function at all. It provides only standalone validation functions and helpers:

```go
// V3: Only utility functions, no constructor
func DefaultPriority() int { return 2 }
func DefaultTimestamps() (created, updated string) {
    now := time.Now().UTC().Format(time.RFC3339)
    return now, now
}
```

This is the most "toolkit" approach -- callers construct `Task` structs directly and call validation functions as needed.

#### Title Length Validation: Bytes vs Runes

V1 and V3 use `len()` which counts bytes:

```go
// V1: byte-length check
if len(trimmed) > 500 {

// V3: byte-length check
if len(title) > 500 {
```

V2 uses `utf8.RuneCountInString()` which counts Unicode characters:

```go
// V2: rune-count check
if utf8.RuneCountInString(trimmed) > maxTitleLength {
```

V2's approach is **genuinely better** because it correctly handles multi-byte UTF-8 characters. A 500-character Chinese title would be 1500 bytes and be rejected by V1/V3 but correctly accepted by V2. V2 also tests this explicitly.

#### Title Validation and Trimming Integration

V1 trims inside `ValidateTitle` then validates the trimmed result, but returns `error` only (the trimmed value is lost):

```go
// V1: trims internally but discards trimmed result
func ValidateTitle(title string) error {
    trimmed := TrimTitle(title)
    if trimmed == "" { ... }
    if len(trimmed) > 500 { ... }
    if strings.ContainsAny(trimmed, "\n\r") { ... }
    return nil
}
```

V2 returns the trimmed title alongside the error:

```go
// V2: returns cleaned title
func ValidateTitle(title string) (string, error) {
    trimmed := strings.TrimSpace(title)
    // ... validations ...
    return trimmed, nil
}
```

V3 does NOT trim in `ValidateTitle` at all -- it validates the raw input:

```go
// V3: no trimming in validation
func ValidateTitle(title string) error {
    if title == "" {
        return errors.New("title is required")
    }
    if len(title) > 500 { ... }
    if strings.ContainsAny(title, "\n\r") { ... }
    return nil
}
```

This means in V3, a title of `"   "` (spaces only) passes `ValidateTitle` (it's not empty), but `TrimTitle` would reduce it to `""`. The caller must trim THEN validate, but nothing enforces this ordering. This is a **bug** -- whitespace-only titles are not rejected by `ValidateTitle`.

#### Constants and Magic Numbers

V2 is most disciplined about named constants:

```go
// V2: all magic numbers named
const (
    idPrefix        = "tick-"
    idRandomBytes   = 3
    maxIDRetries    = 5
    maxTitleLength  = 500
    minPriority     = 0
    maxPriority     = 4
    defaultPriority = 2
)
```

V1 uses minimal constants:

```go
// V1: only retry count named
const maxRetries = 5
```

V3 names some but not all:

```go
// V3
const (
    idPrefix   = "tick-"
    idHexLen   = 6  // defined but never used in code
    maxRetries = 5
)
```

Note: V3 defines `idHexLen = 6` but never uses it -- dead code.

#### FormatTimestamp / Timestamp Handling

V1 provides an explicit `FormatTimestamp()` utility:

```go
// V1
func FormatTimestamp(t time.Time) string {
    return t.UTC().Format("2006-01-02T15:04:05Z")
}
```

V2 has no timestamp formatting function. Since timestamps are stored as `time.Time`, formatting happens at serialization time.

V3 provides `DefaultTimestamps()` which generates both created and updated as pre-formatted strings:

```go
// V3
func DefaultTimestamps() (created, updated string) {
    now := time.Now().UTC().Format(time.RFC3339)
    return now, now
}
```

V3 also provides `DefaultPriority()` as a function rather than a constant, which is unnecessary indirection:

```go
// V3
func DefaultPriority() int { return 2 }
```

### Code Quality

#### Error Handling

V1 uses `fmt.Errorf` throughout:

```go
return "", fmt.Errorf("failed to generate unique ID after %d attempts - task list may be too large", maxRetries)
return fmt.Errorf("title is required")
return fmt.Errorf("priority must be between 0 and 4, got %d", p)
return fmt.Errorf("task cannot be blocked by itself")
```

V2 mixes `errors.New` and `fmt.Errorf`, using `fmt.Errorf` only when formatting is needed:

```go
return "", errors.New("Failed to generate unique ID after 5 attempts - task list may be too large")
return "", errors.New("title is required and cannot be empty")
return fmt.Errorf("priority must be between %d and %d, got %d", minPriority, maxPriority, priority)
return fmt.Errorf("task %s cannot block itself", taskID)  // includes taskID in message
```

V3 uses `errors.New` consistently:

```go
return "", errors.New("failed to generate unique ID after 5 attempts - task list may be too large")
return errors.New("title is required")
return errors.New("priority must be between 0 and 4")
return errors.New("task cannot block itself")
```

V2's error messages include the task ID in self-reference errors (`"task %s cannot block itself"`), which is more useful for debugging. V1 and V3 do not include the ID.

V1 wraps the collision error with `fmt.Errorf` using `%d` for the retry count (parameterized), while V2 and V3 hardcode the number "5" in the string. V1's approach is more maintainable if `maxRetries` changes.

V2's collision error message starts with capital "F" (`"Failed to generate..."`), which violates the Go convention that error strings should not be capitalised (per `go vet` and the Go Code Review Comments guide). V1 and V3 correctly use lowercase.

#### Go Idioms

V2 uses the functional options pattern (`*TaskOptions`) which is idiomatic Go for optional parameters. The `*int` pointer for priority elegantly distinguishes "unset" from "zero".

V1's sentinel value approach (`priority < 0` means "use default") is less idiomatic and could be confusing -- priority 0 is valid (highest priority), so the sentinel must be negative.

V3's "toolkit of functions" approach is the most minimal but shifts complexity to callers.

#### Import Hygiene

V1 imports: `crypto/rand`, `encoding/hex`, `fmt`, `strings`, `time` (5 imports).
V2 imports: `crypto/rand`, `encoding/hex`, `errors`, `fmt`, `strings`, `time`, `unicode/utf8` (7 imports).
V3 imports: `crypto/rand`, `encoding/hex`, `errors`, `strings`, `time` (5 imports).

V2's additional imports (`unicode/utf8` for rune counting, `errors` for `errors.New`) reflect its more thorough implementation.

#### Documentation

All three have package-level doc comments. V2 has the most thorough per-function documentation with detailed parameter descriptions. V3's comments are concise but adequate. V1 is in between.

---

## Test Quality

### V1 Test Functions

File: `internal/task/task_test.go` (347 lines, commit ba060d9)

| Test Function | Subtests | Edge Cases Covered |
|---|---|---|
| `TestGenerateID` | "generates IDs matching tick-{6 hex} pattern", "retries on collision up to 5 times", "errors after 5 collision retries", "generates unique IDs across 100 calls" | Pattern matching, retry counting, error message exact match, uniqueness across 100 calls |
| `TestNormalizeID` | Table-driven: "all uppercase", "mixed case", "already lowercase" | Three case variants |
| `TestValidateTitle` | "valid titles" (table: simple, exactly 500, internal spaces), "invalid titles" (table: empty, whitespace-only, >500, newline, carriage return) | Boundary at 500, whitespace-only, both \n and \r |
| `TestTrimTitle` | Table-driven: leading/trailing spaces, tabs, no whitespace, internal spaces | Four trim scenarios |
| `TestValidatePriority` | "valid priorities" (loop 0-4), "invalid priorities" (table: -1, 5, 100, -100) | All valid values, four invalid boundary/extreme values |
| `TestValidateBlockedBy` | Table-driven: valid reference, self-reference, case-insensitive self, empty list, self among others | Case-insensitive detection, self hidden among valid refs |
| `TestValidateParent` | Table-driven: valid, self-reference, case-insensitive self, empty parent | Four cases including case-insensitive |
| `TestNewTask` | "default values" (checks all fields), "explicit priority" (table: 0, 1, 4, -1), "timestamps are current UTC" | Default priority sentinel, timestamp range check, created==updated |
| `TestStatusConstants` | Table-driven: all 4 status values | Verifies string values |
| `TestFormatTimestamp` | Table-driven: specific time, midnight, end of day; "matches ISO 8601 pattern" | Boundary times, regex pattern check |

**Total: 10 test functions, ~25 subtests/table entries**

### V2 Test Functions

File: `internal/task/task_test.go` (362 lines, commit 1682a27)

| Test Function | Subtests | Edge Cases Covered |
|---|---|---|
| `TestGenerateID` | "it generates IDs matching tick-{6 hex} pattern", "it retries on collision up to 5 times", "it errors after 5 collision retries" | Pattern, retry, error message exact match |
| `TestNormalizeID` | "it normalizes IDs to lowercase" (table: 4 entries) | Four case variants including all-uppercase prefix |
| `TestValidateTitle` | "it rejects empty title" (table: 4 whitespace variants), "it rejects title exceeding 500 characters", "it accepts title at exactly 500 characters", **"it accepts multi-byte Unicode title at exactly 500 characters"**, **"it rejects multi-byte Unicode title exceeding 500 characters"**, "it rejects title with newlines" (table: \n, \r, \r\n), "it trims whitespace from title" (table: 3 entries) | **Unicode rune boundary testing** unique to V2, whitespace variants, CRLF combo |
| `TestValidatePriority` | "it rejects priority outside 0-4" (table: 5 invalid), "it accepts valid priorities 0-4" (loop) | Five invalid values including 10, -100, 100 |
| `TestValidateBlockedBy` | "it rejects self-reference in blocked_by", "it accepts valid blocked_by without self-reference" | Basic self-ref and valid case |
| `TestValidateParent` | "it rejects self-reference in parent", "it accepts valid parent without self-reference", "it accepts empty parent" | Three basic cases |
| `TestNewTask` | "it sets default priority to 2 when not specified", "it sets created and updated timestamps to current UTC time", "it creates task with all fields properly initialized" | Default priority via nil opts, timestamp range/equality/timezone, full field verification with all options set |
| `TestTaskTimestampFormat` | "timestamps use ISO 8601 UTC format" | Z suffix check on formatted timestamp |
| `TestStatusConstants` | "status enum has correct values" | All 4 values individually checked |

**Total: 9 test functions, ~22 subtests/table entries**

### V3 Test Functions

File: `internal/task/task_test.go` (399 lines, commit 861cff5)

| Test Function | Subtests | Edge Cases Covered |
|---|---|---|
| `TestGenerateID` | "it generates IDs matching tick-{6 hex} pattern" (generates 10 IDs), "it retries on collision up to 5 times", "it errors after 5 collision retries" | Multi-ID pattern check, retry counting with exact attempt verification, error message exact match |
| `TestNormalizeID` | "it normalizes IDs to lowercase" (table: 4 entries) | Four case variants |
| `TestTitleValidation` | "it rejects empty title" (checks error message text), "it rejects title exceeding 500 characters" (checks message), "it rejects title with newlines" (both \n and \r, checks messages), "it trims whitespace from title" (2 entries), "it accepts valid title at boundary" (500 chars) | Error message exact verification, 500-char boundary |
| `TestPriorityValidation` | "it rejects priority outside 0-4" (table: 8 entries mixing valid and invalid, checks messages) | Full range -1 through 5 plus 100 |
| `TestSelfReferenceValidation` | "it rejects self-reference in blocked_by", "it accepts blocked_by without self-reference", **"it accepts empty blocked_by"** (both nil and empty slice), "it rejects self-reference in parent", "it accepts parent without self-reference", "it accepts empty parent", **"blocked_by self-reference check is case-insensitive"**, **"parent self-reference check is case-insensitive"** | Case-insensitive tests separated out, nil vs empty slice |
| `TestTaskDefaults` | "it sets default priority to 2 when not specified", "it sets created and updated timestamps to current UTC time" | Timestamp parsing, UTC verification, range check, equality |
| `TestStatusEnum` | "status constants are defined" | All 4 values |
| `TestTaskStruct` | **"Task struct has all 10 fields with correct types"** (sets and verifies all 10 fields), **"optional fields can be zero values"** | Full field coverage, zero-value verification |

**Total: 8 test functions, ~24 subtests**

### Test Coverage Comparison

#### Tests Present in All Three Versions
- ID pattern matching (`tick-{6 hex}`)
- Collision retry up to 5 times
- Error after 5 collisions with exact message
- ID normalization to lowercase (multiple case variants)
- Empty title rejection
- Title exceeding 500 chars rejection
- Title with newlines rejection (both \n and \r)
- Whitespace trimming
- Title at exactly 500 chars boundary
- Priority 0-4 valid, outside rejected
- Self-reference in blocked_by rejected
- Self-reference in parent rejected
- Empty parent accepted
- Default priority is 2
- Timestamps are UTC and created==updated
- Status constants have correct string values

#### Tests Unique to V1
- **Uniqueness across 100 calls**: V1 generates 100 IDs and checks for duplicates -- a statistical uniqueness test absent from V2/V3.
- **FormatTimestamp exact values**: V1 tests `FormatTimestamp()` with specific dates (midnight, end of day) and ISO 8601 regex pattern.
- **Self among others in blocked_by**: V1 tests a self-reference hidden within a list of valid references (`["tick-d4e5f6", "tick-a1b2c3"]`).
- **Case-insensitive blocked_by/parent self-reference**: V1 tests this inline in the table-driven tests.

#### Tests Unique to V2
- **Multi-byte Unicode title at 500 runes**: V2 tests that 500 Chinese characters (`strings.Repeat("漢", 500)`) are accepted. This is only meaningful because V2 uses `utf8.RuneCountInString`.
- **Multi-byte Unicode title at 501 runes rejected**: Corresponding rejection test.
- **Full NewTask integration test**: V2's `TestNewTask` has "it creates task with all fields properly initialized" which creates a task with ALL options set and verifies every field.
- **CRLF combination**: V2 tests `\r\n` as a newline variant (in addition to `\n` and `\r`).
- **Timestamp timezone assertion**: `task.Created.Location() != time.UTC` -- explicitly checks the Location, not just the formatted output.

#### Tests Unique to V3
- **10-iteration pattern check**: V3 generates 10 IDs in the pattern test (more thorough than V1/V2's single ID test, less than V1's 100-uniqueness test).
- **Explicit attempt counting on failure**: V3 counts `attempts` even in the error path and verifies exactly 5 attempts were made.
- **Nil vs empty slice for blocked_by**: V3 tests both `nil` and `[]string{}` as valid empty blocked_by values.
- **Separate case-insensitive subtests**: V3 has dedicated subtests for case-insensitive self-reference detection (separate from the basic self-reference tests).
- **Task struct field verification**: V3's `TestTaskStruct` explicitly constructs a Task with all 10 fields and reads them back, plus tests zero-value optional fields.
- **Error message text verification**: V3 checks exact error message strings for most validation errors (title, priority, blocked_by, parent).

#### Notable Test Gaps

| Gap | V1 | V2 | V3 |
|-----|-----|-----|-----|
| No `NewTask` integration validation test | MISSING -- NewTask does no validation | PRESENT -- tests validation errors through NewTask | N/A -- no NewTask function |
| No multi-byte Unicode boundary test | MISSING | PRESENT | MISSING |
| No uniqueness/statistical test | PRESENT (100 IDs) | MISSING | MISSING |
| No error message exactness checks for validation | MISSING (only checks err != nil) | MISSING (only checks err != nil for validation) | PRESENT (checks most error strings) |
| No exists-function error propagation test | N/A (exists returns bool) | N/A (exists returns bool) | MISSING (exists returns error but not tested) |
| Case-insensitive blocked_by self-ref test | PRESENT | MISSING | PRESENT |
| Case-insensitive parent self-ref test | PRESENT | MISSING | PRESENT |

V3's `ExistsFunc` returns `(bool, error)` but no test verifies that an error from `exists()` is properly propagated by `GenerateID`. This is a notable gap given that error propagation was V3's key design differentiator.

V2 lacks case-insensitive self-reference tests for blocked_by and parent -- a significant gap since the implementation does use `NormalizeID()`.

---

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 3 (go.mod, task.go, task_test.go) | 3 (go.mod, task.go, task_test.go) | 3 (go.mod, task.go, task_test.go) |
| Impl LOC (task.go) | 130 | 201 | 139 |
| Test LOC (task_test.go) | 347 | 362 | 399 |
| Total LOC | 477 | 563 | 538 |
| Test functions | 10 | 9 | 8 |
| Approximate subtests | 25 | 22 | 24 |
| Exported functions | 8 (`GenerateID`, `NormalizeID`, `TrimTitle`, `ValidateTitle`, `ValidatePriority`, `ValidateBlockedBy`, `ValidateParent`, `NewTask`, `FormatTimestamp`) | 7 (`GenerateID`, `NormalizeID`, `ValidateTitle`, `ValidatePriority`, `ValidateBlockedBy`, `ValidateParent`, `NewTask`) | 8 (`GenerateID`, `NormalizeID`, `ValidateTitle`, `TrimTitle`, `ValidatePriority`, `ValidateBlockedBy`, `ValidateParent`, `DefaultPriority`, `DefaultTimestamps`) |
| Types defined | 2 (`Status`, `Task`) | 3 (`Status`, `Task`, `TaskOptions`) | 3 (`Status`, `Task`, `ExistsFunc`) |

---

## Verdict

**V2 is the best implementation**, followed by V1, then V3.

### V2 Wins Because:

1. **Integrated validation in `NewTask`**: V2 is the only version where constructing a task guarantees validity. It validates title, priority, blocked_by, and parent all within `NewTask`, returning an error if anything fails. V1 and V3 leave validation to the caller, creating a gap where invalid tasks can be constructed.

2. **Rune-aware title length checking**: V2 uses `utf8.RuneCountInString()` which correctly counts characters rather than bytes. A 500-character title in Chinese would be 1500 bytes and incorrectly rejected by V1/V3. V2 tests this explicitly.

3. **ValidateTitle returns the cleaned title**: V2's `ValidateTitle(title string) (string, error)` signature eliminates the coordination problem where callers must remember to trim before (or after) validating. V1 trims internally but discards the result. V3 doesn't trim at all in validation.

4. **Options struct with `*int` priority**: The `TaskOptions.Priority *int` pattern elegantly distinguishes "unset" (nil, use default 2) from "explicitly set to 0" (highest priority). V1's sentinel approach (`priority < 0`) is less clear. V3 has no constructor at all.

5. **Most complete `NewTask` test**: V2's "it creates task with all fields properly initialized" test exercises every field through the full creation pipeline with custom options, verifying the complete integration.

6. **Named constants for all magic numbers**: `maxTitleLength`, `minPriority`, `maxPriority`, `defaultPriority`, `idPrefix`, `idRandomBytes`, `maxIDRetries`.

### V2 Weaknesses:

- Capitalised error message "Failed to generate..." violates Go conventions.
- Missing case-insensitive self-reference tests for blocked_by/parent.
- Missing exists-error propagation (though this is not applicable since V2's exists function returns only `bool`).
- Slightly more verbose (201 LOC impl vs V1's 130).

### V1 Strengths:

- Clean, minimal code. Easy to understand.
- `FormatTimestamp()` utility is practical for serialization.
- 100-ID uniqueness test is the most thorough statistical check.
- Correct lowercase error messages (Go convention).
- Most comprehensive table-driven tests for blocked_by/parent including case-insensitive cases.

### V1 Weaknesses:

- `NewTask` does no validation -- invalid tasks can be constructed.
- Sentinel value `priority < 0` for default is non-obvious.
- Byte-length (`len()`) for title validation.
- `ValidateTitle` trims but discards the trimmed result.

### V3 Weaknesses (most significant):

- **`string` timestamps** lose type safety and make time arithmetic impossible without parsing.
- **`ValidateTitle` does not trim** -- whitespace-only titles pass validation. This is a specification violation.
- **No `NewTask` function** -- callers must manually coordinate ID generation, validation, and defaults.
- **`ExistsFunc` error path untested** despite being V3's key design improvement.
- **Dead code**: `idHexLen = 6` is defined but never used.
- **`DefaultPriority()` as a function** returning a constant adds unnecessary indirection vs a named constant.
