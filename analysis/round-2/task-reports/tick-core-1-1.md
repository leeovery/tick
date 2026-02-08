# Task tick-core-1-1: Task Model & ID Generation

## Task Summary

Define the core `Task` struct with all 10 fields, a `Status` string enum with constants (`open`, `in_progress`, `done`, `cancelled`), and a `tick-{6 hex}` ID generator using `crypto/rand` with collision retry. Implement field validation for title (non-empty, max 500 chars, no newlines, trim whitespace), priority (0-4 range, default 2), blocked_by (no self-references), and parent (no self-references). IDs must be normalized to lowercase. Timestamps must be ISO 8601 UTC.

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

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Task struct has all 10 fields with correct Go types | PASS - All 10 fields present: `ID string`, `Title string`, `Status Status`, `Priority int`, `Description string`, `BlockedBy []string`, `Parent string`, `Created time.Time`, `Updated time.Time`, `Closed *time.Time` | PASS - Identical struct definition with all 10 fields and same types |
| ID format matches `tick-{6 hex chars}` pattern | PASS - `GenerateID` produces `idPrefix + hex.EncodeToString(b)` where `b` is 3 bytes | PASS - `GenerateID` produces `"tick-" + hex.EncodeToString(b)` where `b` is 3 bytes |
| IDs are generated using `crypto/rand` | PASS - `crypto/rand.Read(b)` used in `GenerateID` | PASS - `crypto/rand.Read(b)` used in `GenerateID` |
| Collision retry works up to 5 times then errors | PASS - Loop `for attempt := 0; attempt < maxIDRetries; attempt++` where `maxIDRetries = 5` | PASS - Loop `for attempt := 0; attempt < maxRetries; attempt++` where `const maxRetries = 5` (local const) |
| Input IDs are normalized to lowercase | PASS - `NormalizeID` calls `strings.ToLower(id)`, AND `ValidateBlockedBy`/`ValidateParent` use `NormalizeID` for comparison | PARTIAL - `NormalizeID` calls `strings.ToLower(id)`, BUT `ValidateBlockedBy` and `ValidateParent` use direct `==` comparison without normalization |
| Title validation: non-empty, max 500 chars, no newlines, trims whitespace | PASS - `ValidateTitle` trims, checks empty, checks `\n\r`, checks `utf8.RuneCountInString > 500` | PASS - `ValidateTitle` trims, checks empty, checks `\n\r`, checks `utf8.RuneCountInString > 500` |
| Priority validation rejects values outside 0-4 | PASS - `ValidatePriority` checks `< minPriority` or `> maxPriority` (constants 0/4) | PASS - `ValidatePriority` checks `< 0` or `> 4` (inline literals) |
| Self-references in `blocked_by` and `parent` are rejected | PASS - Both validators use `NormalizeID` for case-insensitive self-reference detection | PASS - Both validators use direct `==` comparison (case-sensitive only) |
| Timestamps are ISO 8601 UTC | PASS - `time.Now().UTC().Truncate(time.Second)` | PASS - `time.Now().UTC().Truncate(time.Second)` |

## Implementation Comparison

### Approach

Both versions produce a single file `internal/task/task.go` in package `task` containing the `Task` struct, `TaskOptions` struct, `Status` type with four constants, and the same five exported functions: `GenerateID`, `NormalizeID`, `ValidateTitle`, `ValidatePriority`, `ValidateBlockedBy`, `ValidateParent`, plus the `NewTask` constructor. The overall architecture is effectively identical. The differences are in constants organization, function ordering, error construction, and case-sensitivity handling in self-reference validation.

**Constants organization:**

V2 groups all constants in a single block with named constants for every magic value:
```go
// V2 (task.go lines 30-38)
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

V4 uses a minimal constants block and inlines some values:
```go
// V4 (task.go lines 53-56)
const (
    maxTitleLength  = 500
    defaultPriority = 2
)
```
The retry limit is a local const inside `GenerateID` (`const maxRetries = 5`), the prefix `"tick-"` and byte count `3` are inline literals, and priority bounds `0`/`4` are inline in `ValidatePriority`.

**Function ordering:**

V2 places `GenerateID` first, then validators, then `NewTask` at the bottom.
V4 places `NewTask` first (as the primary API entry point), then `GenerateID`, then validators. V4's ordering is arguably more reader-friendly -- top-down from the public constructor to its helpers.

**Case-insensitive self-reference detection (key difference):**

V2's `ValidateBlockedBy` normalizes both sides:
```go
// V2 (task.go lines 111-118)
func ValidateBlockedBy(taskID string, blockedBy []string) error {
    normalizedTaskID := NormalizeID(taskID)
    for _, dep := range blockedBy {
        if NormalizeID(dep) == normalizedTaskID {
            return fmt.Errorf("task %s cannot block itself", taskID)
        }
    }
    return nil
}
```

V2's `ValidateParent` likewise uses `NormalizeID`:
```go
// V2 (task.go lines 121-129)
func ValidateParent(taskID string, parentID string) error {
    if parentID == "" {
        return nil
    }
    if NormalizeID(parentID) == NormalizeID(taskID) {
        return fmt.Errorf("task %s cannot be its own parent", taskID)
    }
    return nil
}
```

V4 uses direct string equality without normalization:
```go
// V4 (task.go lines 171-177)
func ValidateBlockedBy(taskID string, blockedBy []string) error {
    for _, dep := range blockedBy {
        if dep == taskID {
            return fmt.Errorf("task %s cannot be blocked by itself", taskID)
        }
    }
    return nil
}

// V4 (task.go lines 180-185)
func ValidateParent(taskID string, parent string) error {
    if parent == taskID {
        return fmt.Errorf("task %s cannot be its own parent", taskID)
    }
    return nil
}
```

This is a **genuine behavioral difference**. The task spec states "Normalize IDs to lowercase on input (case-insensitive matching)". V2 correctly applies normalization during self-reference checks, meaning `ValidateBlockedBy("tick-a1b2c3", ["TICK-A1B2C3"])` would correctly detect the self-reference. V4 would miss it because `"TICK-A1B2C3" != "tick-a1b2c3"`. In practice, since `GenerateID` always produces lowercase IDs and `NewTask` validates against the generated ID, this gap only matters if `ValidateBlockedBy`/`ValidateParent` are called independently with mixed-case input.

**Empty parent handling:**

V2's `ValidateParent` has an explicit early return for empty parent:
```go
if parentID == "" {
    return nil
}
```
V4's `ValidateParent` does not -- it relies on the fact that an empty string will never equal a `tick-XXXXXX` ID. Both are functionally correct, but V2 is more defensive and explicit.

**Error construction:**

V2 uses `errors.New()` for static error messages (e.g., in `GenerateID` and `ValidateTitle`), and `fmt.Errorf()` only for parameterized messages. V4 uses `fmt.Errorf()` everywhere, even for static strings. V2's approach is marginally more idiomatic Go (prefer `errors.New` for non-formatted errors).

**Description extraction in NewTask:**

V2 extracts description via a private helper function `optString(opts)`:
```go
// V2 (task.go lines 197-201)
func optString(opts *TaskOptions) string {
    if opts == nil {
        return ""
    }
    return opts.Description
}
```
V4 extracts all optional fields in a single `if opts != nil` block at the top of `NewTask`:
```go
// V4 (task.go lines 70-78)
if opts != nil {
    if opts.Priority != nil {
        priority = *opts.Priority
    }
    description = opts.Description
    blockedBy = opts.BlockedBy
    parent = opts.Parent
}
```
V4's approach is simpler and more direct -- no helper function needed.

**Validation ordering in NewTask:**

V2 validates title, generates ID, validates priority, validates blocked_by/parent.
V4 validates title, extracts all opts, validates priority, generates ID, validates blocked_by/parent.

V4 validates priority *before* generating an ID, which is more efficient -- if priority is invalid, it returns an error without wasting a random ID generation. V2 generates the ID before validating priority, which is less optimal but functionally equivalent.

**Closed field initialization:**

V2 explicitly sets `Closed: nil` in the returned struct literal. V4 omits it (Go zero value for `*time.Time` is already `nil`). Both correct; V2 is more explicit but unnecessary.

### Code Quality

**Go idioms:**

V2 imports `errors` and uses `errors.New()` for static messages -- idiomatic. V4 uses `fmt.Errorf()` for all errors including static ones -- slightly less idiomatic but negligible.

Both use proper Go naming (`StatusOpen`, `StatusInProgress`), proper doc comments on all exported symbols, and proper error wrapping with `%w` for `crypto/rand` failures and title validation wrapping in `NewTask`.

**Named constants vs inline:**

V2 defines `minPriority = 0`, `maxPriority = 4`, `idPrefix = "tick-"`, `idRandomBytes = 3`, `maxIDRetries = 5` as named package-level constants. This is better for maintainability -- a single change point if values need updating.

V4 inlines the prefix string `"tick-"`, byte count `3`, and priority bounds `0`/`4` directly. The retry limit is a function-local const. This is less DRY but keeps the constants closer to their usage.

**Error messages:**

V2's `ValidateBlockedBy` error: `"task %s cannot block itself"`.
V4's `ValidateBlockedBy` error: `"task %s cannot be blocked by itself"`.
V4's wording is more precise (passive voice clarifies the relationship).

V2's `ValidateTitle` newline error: `"title cannot contain newlines"`.
V4's `ValidateTitle` newline error: `"title must be a single line (no newlines)"`.
V4's wording is more descriptive.

**Type safety:**

Both versions use `*int` for optional priority in `TaskOptions` and `*time.Time` for `Closed`. Both use the `Status` type alias properly.

### Test Quality

**V2 Test Functions (internal/task/task_test.go, 362 lines):**

1. `TestGenerateID`
   - `"it generates IDs matching tick-{6 hex} pattern"` -- single ID generation, regex check
   - `"it retries on collision up to 5 times"` -- 4 collisions then success, verifies 5 attempts
   - `"it errors after 5 collision retries"` -- always collides, verifies exact error message

2. `TestNormalizeID`
   - `"it normalizes IDs to lowercase"` -- table-driven: `TICK-A3F2B7`, `Tick-A3f2B7`, `tick-a3f2b7`, `TICK-ABCDEF` (4 cases)

3. `TestValidateTitle`
   - `"it rejects empty title"` -- 4 inputs: `""`, `"   "`, `"\t"`, `"  \t  "`
   - `"it rejects title exceeding 500 characters"` -- 501 chars
   - `"it accepts title at exactly 500 characters"` -- boundary test (500 ASCII chars)
   - `"it accepts multi-byte Unicode title at exactly 500 characters"` -- 500 runes of `'漢'` (multi-byte boundary)
   - `"it rejects multi-byte Unicode title exceeding 500 characters"` -- 501 runes of `'漢'`
   - `"it rejects title with newlines"` -- 3 inputs: `\n`, `\r`, `\r\n`
   - `"it trims whitespace from title"` -- 3 cases: leading+trailing spaces, tabs, internal spaces preserved

4. `TestValidatePriority`
   - `"it rejects priority outside 0-4"` -- 5 inputs: `-1, 5, 10, -100, 100`
   - `"it accepts valid priorities 0-4"` -- all 5 valid values

5. `TestValidateBlockedBy`
   - `"it rejects self-reference in blocked_by"` -- self-ref in list
   - `"it accepts valid blocked_by without self-reference"` -- no self-ref

6. `TestValidateParent`
   - `"it rejects self-reference in parent"` -- self-ref
   - `"it accepts valid parent without self-reference"` -- valid ref
   - `"it accepts empty parent"` -- empty string

7. `TestNewTask`
   - `"it sets default priority to 2 when not specified"` -- nil opts
   - `"it sets created and updated timestamps to current UTC time"` -- range check, equality check, UTC check
   - `"it creates task with all fields properly initialized"` -- all 10 fields verified

8. `TestTaskTimestampFormat`
   - `"timestamps use ISO 8601 UTC format"` -- formats as RFC3339, checks Z suffix

9. `TestStatusConstants`
   - `"status enum has correct values"` -- all 4 status values verified

**Total V2 test subtests: ~20 named subtests across 9 top-level test functions**

**V4 Test Functions (internal/task/task_test.go, 329 lines):**

1. `TestGenerateID`
   - `"it generates IDs matching tick-{6 hex} pattern"` -- generates 20 IDs in a loop, regex check (more robust than V2's single check)
   - `"it retries on collision up to 5 times"` -- 4 collisions then success, verifies 5 attempts
   - `"it errors after 5 collision retries"` -- always collides, verifies exact error message

2. `TestNormalizeID`
   - `"it normalizes IDs to lowercase"` -- table-driven with named struct fields: `uppercase`, `mixed case`, `already lowercase` (3 cases vs V2's 4)

3. `TestValidateTitle`
   - `"it rejects empty title"` -- 3 named inputs: `empty string`, `only spaces`, `only tabs` (V2 has 4 inputs including `"  \t  "`)
   - `"it rejects title exceeding 500 characters"` -- 501 chars
   - `"it counts characters not bytes for title length"` -- 500 runes of `'\u4e16'` accepted, 501 rejected (combines V2's two Unicode tests into one)
   - `"it rejects title with newlines"` -- 3 named inputs: `line feed`, `carriage return`, `crlf`
   - `"it trims whitespace from title"` -- 4 named cases: `leading spaces`, `trailing spaces`, `both sides`, `tabs`

4. `TestValidatePriority`
   - `"it rejects priority outside 0-4"` -- 3 named inputs: `negative (-1)`, `too high (5)`, `way too high (100)` (V2 tests 5 values including `-100, 10`)
   - `"it accepts valid priorities"` -- all 5 valid values (loop without subtests)

5. `TestValidateBlockedBy`
   - `"it rejects self-reference in blocked_by"` -- self-ref in list
   - `"it accepts valid blocked_by references"` -- no self-ref

6. `TestValidateParent`
   - `"it rejects self-reference in parent"` -- self-ref
   - `"it accepts valid parent reference"` -- valid ref
   (missing: V2 has `"it accepts empty parent"` -- V4 does not test empty parent)

7. `TestNewTask`
   - `"it sets default priority to 2 when not specified"` -- nil opts
   - `"it sets created and updated timestamps to current UTC time"` -- range check, equality check, UTC check
   - `"it has all 10 fields with correct types"` -- all 10 fields verified; uses `intPtr` helper

**Total V4 test subtests: ~17 named subtests across 7 top-level test functions**

**Test differences:**

| Gap | V2 | V4 |
|-----|-----|-----|
| ID pattern robustness | Single ID generation | 20 IDs in a loop -- more thorough |
| NormalizeID cases | 4 cases | 3 cases (missing `TICK-ABCDEF`) |
| Empty title inputs | 4 inputs incl. mixed whitespace `"  \t  "` | 3 inputs (missing mixed whitespace) |
| Unicode title tests | 2 separate tests (accept 500, reject 501) | 1 combined test |
| Invalid priority count | 5 values (-1, 5, 10, -100, 100) | 3 values (-1, 5, 100) |
| Empty parent test | Present | Missing |
| `TestTaskTimestampFormat` | Separate test function checking RFC3339 Z suffix | Missing (timestamp tested within NewTask but no format check) |
| `TestStatusConstants` | Present -- verifies all 4 enum values | Missing |
| `intPtr` helper | Uses `&priority` inline | Provides `intPtr(i int) *int` helper |
| Title at exactly 500 chars (ASCII) | Explicit boundary test | Missing (only has multi-byte boundary) |
| Test naming style | `fmt.Sprintf("%q", title)` for unnamed subtest keys | Named struct fields (`name: "..."`) -- more readable |

V2 has more comprehensive edge case coverage (more invalid priority values, mixed whitespace, empty parent, status constants verification, explicit timestamp format test, explicit ASCII boundary test at 500). V4 has better ID generation robustness (20 iterations) and slightly better test naming conventions (named struct fields).

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 5 | 5 |
| Lines added (total) | 570 | 523 |
| Impl LOC (task.go) | 201 | 187 |
| Test LOC (task_test.go) | 362 | 329 |
| Test functions (top-level) | 9 | 7 |
| Test subtests (approx) | 20 | 17 |

## Verdict

**V2 is the better implementation of this task.**

The deciding factors:

1. **Case-insensitive self-reference detection (V2 PASS, V4 PARTIAL):** The spec explicitly requires "Normalize IDs to lowercase on input (case-insensitive matching)". V2's `ValidateBlockedBy` and `ValidateParent` both use `NormalizeID()` for comparison, correctly implementing case-insensitive self-reference detection. V4 uses plain `==` string comparison, which would fail to detect a self-reference like `"TICK-A1B2C3"` vs `"tick-a1b2c3"`. While the gap is unlikely to trigger in normal `NewTask` flow (since `GenerateID` always produces lowercase), the validators are exported functions and the spec is unambiguous.

2. **More comprehensive test coverage:** V2 has 9 top-level test functions with ~20 subtests vs V4's 7 functions with ~17 subtests. V2 includes `TestTaskTimestampFormat` (verifying RFC3339 Z suffix), `TestStatusConstants` (verifying all 4 enum values), an explicit empty-parent test, an explicit ASCII boundary test at exactly 500 chars, more invalid priority values (5 vs 3), and more empty-title variants (4 vs 3).

3. **Better use of named constants:** V2 extracts all magic values into package-level constants (`idPrefix`, `idRandomBytes`, `maxIDRetries`, `minPriority`, `maxPriority`). This is more maintainable and follows the DRY principle more rigorously.

4. **More defensive code:** V2's `ValidateParent` explicitly handles empty parent with an early return, making the intent clear rather than relying on implicit inequality.

V4 has two minor advantages: (a) generating 20 IDs in the pattern test is more thorough than V2's single check, and (b) validating priority before generating the ID is slightly more efficient. V4's function ordering (NewTask first) is also arguably more readable. However, these do not outweigh V2's correctness advantage on case-insensitive matching and its broader test coverage.
