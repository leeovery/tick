# Task tick-core-1-1: Task model & ID generation

## Task Summary

Define the core `Task` struct with all 10 fields (`id`, `title`, `status`, `priority`, `description`, `blocked_by`, `parent`, `created`, `updated`, `closed`), a `Status` string enum (`open`, `in_progress`, `done`, `cancelled`), and a `tick-{6 hex}` ID generator using `crypto/rand` with collision retry. Includes field validation logic for title, priority, blocked_by, and parent fields, plus ID normalization.

**Acceptance Criteria:**
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

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Task struct has all 10 fields with correct Go types | PASS -- Struct defined with all 10 fields: `ID string`, `Title string`, `Status Status`, `Priority int`, `Description string`, `BlockedBy []string`, `Parent string`, `Created time.Time`, `Updated time.Time`, `Closed *time.Time` | PASS -- Identical struct definition with all 10 fields and same types |
| ID format matches `tick-{6 hex chars}` pattern | PASS -- `GenerateID` uses `idPrefix + hex.EncodeToString(b)` with 3-byte random input | PASS -- Identical approach: `idPrefix + hex.EncodeToString(b)` with 3-byte input |
| IDs are generated using `crypto/rand` | PASS -- `crypto/rand.Read(b)` | PASS -- `crypto/rand.Read(b)` |
| Collision retry works up to 5 times then errors | PASS -- Loop `for i := 0; i < idRetries; i++` with `idRetries = 5`, error message matches spec exactly: "Failed to generate unique ID after 5 attempts..." | PASS -- Loop `for attempt := 0; attempt < maxIDRetries; attempt++` with `maxIDRetries = 5`, error message differs in casing: "failed to generate unique ID after 5 attempts..." |
| Input IDs are normalized to lowercase | PASS -- `NormalizeID` returns `strings.ToLower(id)` | PASS -- Identical `NormalizeID` implementation |
| Title validation: non-empty, max 500 chars, no newlines, trims whitespace | PASS -- `ValidateTitle` trims, checks empty, checks rune count > 500, checks `\n\r`; returns trimmed title | PARTIAL -- `ValidateTitle` trims internally and checks same constraints, but returns only `error` (no trimmed title returned). Separate `TrimTitle` function exists, but caller must remember to call it before or after validation. The spec says "trim whitespace" which V6 achieves via a two-function approach. |
| Priority validation rejects values outside 0-4 | PASS -- `ValidatePriority` checks `priority < 0 \|\| priority > 4` | PASS -- Same logic using named constants `minPriority`/`maxPriority` |
| Self-references in `blocked_by` and `parent` rejected | PASS -- `ValidateBlockedBy` and `ValidateParent` use case-insensitive comparison via `NormalizeID` | PASS -- Identical approach in both functions |
| Timestamps are ISO 8601 UTC | PASS -- `TimestampFormat = "2006-01-02T15:04:05Z"`, `FormatTimestamp` forces UTC | PASS -- Identical format constant and `FormatTimestamp` implementation |

## Implementation Comparison

### Approach

Both versions solve the task by creating a single `internal/task/task.go` file with the `Task` struct, `Status` type, ID generation, normalization, validation functions, and timestamp formatting. The structural differences are:

**NewTask constructor:**

V5 takes pre-existing ID and title, returning a value type:
```go
// V5: internal/task/task.go line 130
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

V6 integrates ID generation and title validation into the constructor, returning a pointer and error:
```go
// V6: internal/task/task.go line 132
func NewTask(title string, exists func(id string) bool) (*Task, error) {
    trimmed := TrimTitle(title)
    if err := ValidateTitle(trimmed); err != nil {
        return nil, err
    }
    if exists == nil {
        exists = func(id string) bool { return false }
    }
    id, err := GenerateID(exists)
    if err != nil {
        return nil, err
    }
    now := time.Now().UTC().Truncate(time.Second)
    return &Task{
        ID:       id,
        Title:    trimmed,
        Status:   StatusOpen,
        Priority: defaultPriority,
        Created:  now,
        Updated:  now,
    }, nil
}
```

V6's approach is more self-contained -- a single call creates a fully validated task with a generated ID. V5's approach is more composable -- ID generation and validation are separate concerns the caller orchestrates. Both are valid Go patterns. V5's simpler constructor is more flexible for testing (you can construct tasks with known IDs directly), while V6's integrated constructor enforces invariants at creation time.

**Title validation return type:**

V5's `ValidateTitle` returns `(string, error)`, giving back the trimmed title:
```go
// V5: internal/task/task.go line 149
func ValidateTitle(title string) (string, error) {
    title = strings.TrimSpace(title)
    if title == "" {
        return "", fmt.Errorf("title is required and cannot be empty")
    }
    ...
    return title, nil
}
```

V6 splits this into two functions -- `TrimTitle` (returns `string`) and `ValidateTitle` (returns `error`):
```go
// V6: internal/task/task.go line 85
func ValidateTitle(title string) error {
    trimmed := strings.TrimSpace(title)
    if trimmed == "" {
        return errors.New("title is required and cannot be empty")
    }
    ...
}

func TrimTitle(title string) string {
    return strings.TrimSpace(title)
}
```

V5's combined approach is more idiomatic for a "validate-and-normalize" operation in Go. V6's separation follows single-responsibility principle but introduces a subtle issue: `ValidateTitle` trims internally for checking, but the caller doesn't get the trimmed result back, so they must also call `TrimTitle` separately. This means whitespace is trimmed twice (once in `ValidateTitle` for validation, once by the caller via `TrimTitle`).

**JSON serialization:**

V5 implements custom `MarshalJSON`/`UnmarshalJSON` methods via a shadow `taskJSON` struct to control timestamp formatting and omit optional fields:
```go
// V5: internal/task/task.go line 60-67
type taskJSON struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    ...
    Created     string   `json:"created"`
    Updated     string   `json:"updated"`
    Closed      string   `json:"closed,omitempty"`
}
```

V6 does not implement custom JSON marshaling. Its `Task` struct uses standard `json` tags with `time.Time` fields tagged directly:
```go
// V6: internal/task/task.go line 49-50
Created     time.Time  `json:"created"`
Updated     time.Time  `json:"updated"`
Closed      *time.Time `json:"closed,omitempty"`
```

This means V6's JSON output will use Go's default `time.Time` marshaling (RFC 3339 with nanoseconds, e.g., `"2026-01-19T10:00:00Z"`), which happens to be compatible with ISO 8601 but includes nanosecond precision when present. V5's approach guarantees exact `YYYY-MM-DDTHH:MM:SSZ` format. V5 also properly handles the `Closed` omission (nil pointer -> empty string -> omitempty), while V6 relies on `*time.Time` with `omitempty` which works correctly for JSON but doesn't control the format.

V5's JSON support is genuinely superior here -- it's production-ready with round-trip serialization, while V6 defers this concern.

**Error creation:**

V5 consistently uses `fmt.Errorf(...)` for all errors. V6 uses `errors.New(...)` for static strings and `fmt.Errorf(...)` for formatted strings. V6's approach is marginally more correct in Go style (no need for formatting when the string is static), though both compile to essentially the same thing.

**Constants:**

V5 exports `DefaultPriority = 2`. V6 keeps `defaultPriority = 2` unexported, along with `minPriority = 0` and `maxPriority = 4` as named constants rather than magic numbers.

V6's approach of named bounds (`minPriority`, `maxPriority`) is slightly more maintainable but the 0-4 range is defined by the spec and unlikely to change.

### Code Quality

**Go idioms:**

V5's error message for collision matches the spec exactly: `"Failed to generate unique ID after 5 attempts - task list may be too large"` (capital F). V6 uses lowercase: `"failed to generate unique ID after 5 attempts..."`. Go convention is lowercase error messages (per the Go Code Review Comments guide), so V6 is more idiomatic here, though V5 is spec-verbatim.

V5 wraps errors with `%w`:
```go
// V5
return "", fmt.Errorf("reading random bytes: %w", err)
```

V6 also wraps:
```go
// V6
return "", fmt.Errorf("failed to generate random bytes: %w", err)
```

Both comply with the skill constraint on `fmt.Errorf("%w", err)`.

**Naming:**

V5 uses `idRetries`, `idRandSize`. V6 uses `maxIDRetries`, `idByteLength`. V6's naming is slightly more descriptive.

V5's loop variable: `for i := 0; i < idRetries; i++`. V6's: `for attempt := 0; attempt < maxIDRetries; attempt++`. V6's `attempt` is more readable than `i`.

**Type safety:**

Both use the same `Status` string type. Neither implements a `Valid() bool` method on `Status`, meaning invalid status values could be assigned. This is acceptable for this task scope.

**DRY:**

V6 has a redundancy issue: `TrimTitle` is just `strings.TrimSpace`, and `ValidateTitle` internally calls `strings.TrimSpace` again. The trim happens twice when used as designed (in `NewTask`: `TrimTitle` then `ValidateTitle`).

V5 avoids this by having `ValidateTitle` do the trim and return the trimmed result.

**Documentation:**

Both document all exported types and functions. V5 is slightly more thorough with doc comments on `taskJSON` and the marshal/unmarshal methods.

### Test Quality

**V5 Test Functions (10 top-level, 23 subtests):**

1. `TestGenerateID`
   - `it generates IDs matching tick-{6 hex} pattern` -- regex validation
   - `it retries on collision up to 5 times` -- uses `sync/atomic.Int32` for thread-safe call counting
   - `it errors after 5 collision retries` -- checks exact error message

2. `TestValidateTitle`
   - `it rejects empty title` -- tests `""`, `"   "`, `"\t"` (3 empty variants)
   - `it rejects title exceeding 500 characters` -- 501-char string
   - `it rejects title with newlines` -- tests `\n`, `\r`, `\r\n` (3 newline variants)
   - `it trims whitespace from title` -- checks returned trimmed string
   - `it accepts valid title at 500 chars` -- boundary test at exactly 500
   - `it counts characters not bytes for max length` -- 200 CJK chars (600 bytes) accepted

3. `TestValidatePriority`
   - `it rejects priority outside 0-4` -- tests -1, 5, 10, -100, 999
   - `it accepts valid priorities 0 through 4` -- loop 0-4

4. `TestValidateBlockedBy`
   - `it rejects self-reference in blocked_by` -- self in list with other IDs
   - `it accepts blocked_by without self-reference` -- two other IDs
   - `it accepts empty blocked_by` -- nil slice
   - `it detects self-reference case-insensitively` -- `TICK-A1B2C3` vs `tick-a1b2c3`

5. `TestValidateParent`
   - `it rejects self-reference in parent` -- exact match
   - `it accepts different parent ID` -- different ID
   - `it accepts empty parent` -- empty string
   - `it detects self-reference case-insensitively` -- mixed case

6. `TestNormalizeID`
   - `it normalizes IDs to lowercase` -- table-driven: 4 cases (all-upper, mixed, already-lower, all-upper-hex)

7. `TestTaskStruct`
   - `it has all 10 fields with correct types` -- constructs full struct, verifies each field
   - `it sets default priority to 2 when not specified` -- via `NewTask`
   - `it sets created and updated timestamps to current UTC time` -- time-range check + UTC location check

8. `TestStatusConstants`
   - `it defines correct status values` -- table-driven: 4 statuses

9. `TestTaskTimestampFormat`
   - `it formats timestamps as ISO 8601 UTC` -- known date to expected string

10. `TestTaskJSONSerialization`
    - `it omits optional fields when empty` -- marshal, check absence of optional keys
    - `it includes optional fields when set` -- marshal, check presence of all values
    - `it formats timestamps as ISO 8601 UTC in JSON` -- verifies exact timestamp format in JSON output

**V6 Test Functions (8 top-level, 20 subtests):**

1. `TestGenerateID`
   - `it generates IDs matching tick-{6 hex} pattern` -- regex validation
   - `it retries on collision up to 5 times` -- plain int counter (not atomic)
   - `it errors after 5 collision retries` -- checks error message (lowercase variant)
   - `it normalizes IDs to lowercase` -- single test case for `NormalizeID`

2. `TestValidateTitle`
   - `it rejects empty title` -- only tests `""` (one variant)
   - `it rejects title exceeding 500 characters` -- 501-char string
   - `it rejects title with newlines` -- only tests `\n` (one variant)
   - `it trims whitespace from title` -- tests `TrimTitle` separately
   - `it rejects whitespace-only title` -- `"   "`
   - `it accepts valid title at 500 characters` -- boundary test
   - `it counts multi-byte Unicode characters as single characters` -- 500 CJK accepted, 501 CJK rejected

3. `TestValidatePriority`
   - `it rejects priority outside 0-4` -- table-driven: -1, 5, 100
   - `it accepts valid priorities` -- loop 0-4 with subtests

4. `TestValidateBlockedBy`
   - `it rejects self-reference in blocked_by` -- single self-ref
   - `it accepts valid blocked_by references` -- single other ID

5. `TestValidateParent`
   - `it rejects self-reference in parent` -- exact match
   - `it accepts valid parent reference` -- different ID
   - `it accepts empty parent` -- empty string

6. `TestNewTask`
   - `it sets default priority to 2 when not specified` -- via `NewTask`
   - `it sets created and updated timestamps to current UTC time` -- time-range + equality + UTC check
   - `it has all 10 fields with correct Go types` -- accesses all fields, checks defaults

7. `TestStatus`
   - `it defines all four status constants` -- direct comparison

8. `TestTimestampFormat`
   - `it formats timestamps as ISO 8601 UTC` -- known date to expected string

**Test Coverage Gaps (V6 relative to V5):**

- **Missing JSON serialization tests**: V5 has 3 JSON subtests (omit empty, include set, timestamp format in JSON). V6 has zero. This is the biggest gap -- V6 doesn't test serialization at all, partly because it doesn't implement custom marshaling.
- **Missing case-insensitive self-reference tests**: V5 tests case-insensitive detection in both `ValidateBlockedBy` and `ValidateParent`. V6 does not test this edge case in either.
- **Missing empty blocked_by test**: V5 tests `nil` blocked_by slice. V6 does not.
- **Missing `\r` and `\r\n` newline variants**: V5 tests 3 newline types (`\n`, `\r`, `\r\n`). V6 only tests `\n`.
- **Missing multiple empty title variants**: V5 tests `""`, `"   "`, `"\t"`. V6 tests `""` and `"   "` as separate subtests but misses `"\t"`.
- **NormalizeID testing**: V5 has a dedicated `TestNormalizeID` with 4 table-driven cases. V6 tests NormalizeID with a single case inside `TestGenerateID`, which is less thorough.
- **Thread safety**: V5 uses `sync/atomic.Int32` for the collision counter, making the test theoretically safe even though `GenerateID` is synchronous. V6 uses a plain `int`.

**Test Coverage Gaps (V5 relative to V6):**

- **501 multi-byte rejection**: V6 tests that 501 CJK characters are rejected (validating the upper boundary for multi-byte). V5 only tests that 200 CJK chars are accepted.
- **Whitespace-only as separate test**: V6 has a dedicated `it rejects whitespace-only title` subtest. V5 includes this in the empty title test but not as a named case.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| **MUST: Use gofmt and golangci-lint** | PASS -- Code follows standard gofmt formatting | PASS -- Code follows standard gofmt formatting |
| **MUST: Handle all errors explicitly** | PASS -- All errors from `rand.Read`, `json.Unmarshal`, `time.Parse` are checked | PASS -- All errors from `rand.Read` are checked. No JSON code to evaluate. |
| **MUST: Write table-driven tests with subtests** | PASS -- `TestNormalizeID`, `TestStatusConstants`, `TestValidatePriority` (valid range), `TestValidateTitle` (empty, newlines) use table-driven approach with subtests | PARTIAL -- `TestValidatePriority` (invalid) and `TestValidatePriority` (valid) use table-driven with subtests. Less table-driven usage overall; `TestStatus` uses inline assertions rather than table. |
| **MUST: Document all exported functions, types, and packages** | PASS -- Package doc, all exported types (`Task`, `Status`, `StatusOpen`, etc.), all exported functions documented | PASS -- Package doc, all exported types, all exported functions documented |
| **MUST: Propagate errors with fmt.Errorf("%w", err)** | PASS -- `rand.Read` error: `fmt.Errorf("reading random bytes: %w", err)`. JSON errors: `fmt.Errorf("unmarshaling task JSON: %w", err)` etc. | PASS -- `rand.Read` error: `fmt.Errorf("failed to generate random bytes: %w", err)` |
| **MUST NOT: Ignore errors** | PASS -- No ignored errors | PASS -- No ignored errors |
| **MUST NOT: Use panic for normal error handling** | PASS -- No panics | PASS -- No panics |
| **MUST NOT: Hardcode configuration** | PASS -- Uses named constants for all magic numbers | PASS -- Uses named constants including `minPriority`/`maxPriority` |

### Spec-vs-Convention Conflicts

**1. Error message casing for collision failure**

- **Spec says**: Error message should be "Failed to generate unique ID after 5 attempts - task list may be too large" (capital F)
- **Go convention**: Error strings should not be capitalized (Go Code Review Comments)
- **V5 chose**: Spec-verbatim capital F: `fmt.Errorf("Failed to generate unique ID after 5 attempts...")`
- **V6 chose**: Go convention lowercase: `errors.New("failed to generate unique ID after 5 attempts...")`
- **Assessment**: The spec provides an exact error message to match. V5's literal compliance is defensible since test assertions check the exact string. V6's deviation follows Go idiom. Both are reasonable judgment calls. If downstream code uses `errors.Is` or substring matching, the casing difference could matter. V6's choice is slightly more idiomatic.

**2. ValidateTitle return type -- trim-and-validate vs separate functions**

- **Spec says**: "Validate title: required, non-empty, max 500 chars, no newlines, trim whitespace" -- implies a single validation operation that also trims.
- **Go convention**: Functions should do one thing. Returning a modified value from a validation function is idiomatic in Go (e.g., `url.Parse` validates and returns).
- **V5 chose**: Combined `ValidateTitle(title string) (string, error)` -- validates and returns trimmed title.
- **V6 chose**: Separated into `TrimTitle` + `ValidateTitle(title string) error` -- single responsibility.
- **Assessment**: V5's approach more naturally matches the spec's description of a single operation. V6's separation is a reasonable design choice but introduces the double-trim issue mentioned above.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 5 (go.mod, task.go, task_test.go, tracking.md, plan) | 6 (go.mod, task.go, task_test.go, tracking.md, plan, binary delete) |
| Lines added | 656 (total), 642 (source) | 485 (total), 469 (source) |
| Impl LOC | 218 | 161 |
| Test LOC | 421 | 305 |
| Test functions (top-level) | 10 | 8 |
| Test subtests | 23 | 20 |

## Verdict

**V5 is the better implementation of this task.**

The core reasoning:

1. **JSON serialization (V5 only)**: V5 implements custom `MarshalJSON`/`UnmarshalJSON` with a shadow struct to ensure timestamps are formatted as exact ISO 8601 `YYYY-MM-DDTHH:MM:SSZ` strings. V6 relies on Go's default `time.Time` marshaling, which produces RFC 3339 (compatible but not identical format control). For a task management tool that stores data as JSON, this is production-relevant functionality. V5 also tests JSON round-tripping with 3 dedicated subtests; V6 has zero JSON tests.

2. **Test thoroughness**: V5 has 23 subtests across 10 test functions vs V6's 20 subtests across 8 functions. V5 covers edge cases V6 misses: case-insensitive self-reference detection in both `blocked_by` and `parent`, nil `blocked_by` slice, `\r` and `\r\n` newline variants, and all JSON serialization paths. V6's only test coverage advantage is the 501-multi-byte rejection boundary test.

3. **ValidateTitle API design**: V5's `(string, error)` return is more ergonomic and avoids the double-trim issue in V6 where `TrimTitle` + `ValidateTitle` both call `strings.TrimSpace`.

4. **Spec compliance**: V5 matches the spec's exact error message for collision failure. V6 deviates to follow Go convention (lowercase). While V6's choice is more idiomatic, the spec explicitly defines this message string.

V6 has two minor advantages: more descriptive constant names (`maxIDRetries`, `idByteLength`, `minPriority`, `maxPriority`), and using `errors.New` for static error strings. These are style improvements that don't outweigh V5's functional advantages.

Both implementations fully satisfy the core acceptance criteria. V5 goes further with JSON serialization support and more comprehensive test coverage, making it the stronger implementation overall.
