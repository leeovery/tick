# Task tick-core-2-1: Status transition validation logic

## Task Summary

Implement a pure validation function `Transition(task *Task, command string) error` that enforces the 4 valid status transition commands (`start`, `done`, `cancel`, `reopen`) across 7 valid from-to pairs, rejects 9 invalid transitions with descriptive errors, and manages `closed` and `updated` timestamps as side effects. The function must return old and new status for output formatting.

### Acceptance Criteria

1. All 7 valid status transitions succeed with correct new status
2. All 9 invalid transitions return error
3. Task not modified on invalid transition
4. `closed` set to current UTC on done/cancelled
5. `closed` cleared on reopen
6. `updated` refreshed on every valid transition
7. Error messages include command name and current status
8. Function returns old and new status

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 7 valid transitions succeed | PASS - all 7 tested via table-driven subtests | PASS - all 7 tested as individual subtests | PASS - all 7 tested as individual subtests |
| 9 invalid transitions return error | PASS - all 9 tested via table-driven subtests | PASS - all 9 tested as individual subtests with exact message matching | PASS - all 9 tested as individual subtests with exact message matching |
| Task not modified on invalid transition | PASS - single test checks status, updated, closed | PASS - table-driven test checks all 9 invalid cases for status, updated, closed | PASS - single test checks one invalid case for status, updated, closed |
| `closed` set to UTC on done/cancelled | PASS - two separate tests with time-window assertion | PASS - two separate tests with time-window assertion and UTC location check | PASS - two separate tests, parses RFC3339 string with time-window |
| `closed` cleared on reopen | PASS - sets closed then reopens, asserts nil | PASS - uses precondition check then asserts nil | PASS - uses makeClosedTask helper, asserts empty string |
| `updated` refreshed on every valid transition | PASS - tests one transition (start) with time.Sleep | PASS - table-driven test covering all 7 transitions with time.Sleep | PASS - table-driven test covering all 7 transitions |
| Error messages include command and current status | PASS - checks error contains "start", "tick-abc123", "done" substrings | PASS - exact string match against full error message for all 9 cases | PASS - exact string match against full error message for all 9 cases |
| Returns old and new status | PASS - TransitionResult struct with OldStatus, NewStatus | PASS - returns (oldStatus Status, newStatus Status, err error) as bare values | PASS - TransitionResult struct with OldStatus, NewStatus |

## Implementation Comparison

### Approach

#### File Organization

**V1** and **V3** place the transition logic in a dedicated `internal/task/transition.go` file with a corresponding `internal/task/transition_test.go`. This is clean separation of concerns.

**V2** appends the transition logic directly to the existing `internal/task/task.go` file (lines 201-252) and adds tests to `internal/task/task_test.go` (lines 363-752). This keeps related domain logic co-located but makes both files substantially larger.

#### Transition Lookup Strategy

**V1** uses a **map with composite struct keys**:

```go
type key struct {
    status  Status
    command string
}

transitions := map[key]Status{
    {StatusOpen, "start"}:        StatusInProgress,
    {StatusInProgress, "done"}:   StatusDone,
    // ...
}
```

The map is **rebuilt on every call** (local variable inside `resolveTransition`). This is a minor performance concern -- the map allocation happens per invocation.

**V2** uses a **package-level map** with slice-based source validation:

```go
var validTransitions = map[string]struct {
    from []Status
    to   Status
}{
    "start":  {from: []Status{StatusOpen}, to: StatusInProgress},
    "done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
    // ...
}
```

The lookup first checks the command exists in the map, then iterates the `from` slice to validate the current status. This is more efficient (single allocation at init) and more readable: each command maps naturally to its valid source states.

**V3** uses a **switch statement** with inline boolean validation:

```go
switch command {
case "start":
    valid = oldStatus == StatusOpen
    newStatus = StatusInProgress
case "done":
    valid = oldStatus == StatusOpen || oldStatus == StatusInProgress
    newStatus = StatusDone
// ...
default:
    return TransitionResult{}, fmt.Errorf("unknown command: %s", command)
}
```

This is the most explicit and readable approach. Each case is self-documenting. The `default` branch also handles unknown commands, which V1 does not.

#### Return Type

**V1** and **V3** return `(TransitionResult, error)` where `TransitionResult` is a struct:

```go
type TransitionResult struct {
    OldStatus Status
    NewStatus Status
}
```

**V2** returns bare values `(oldStatus Status, newStatus Status, err error)`. This is simpler but loses the self-documenting nature of a named struct. On error, V2 returns `("", "", err)` which means callers must know that empty-string Status values indicate error.

#### Timestamp Type Handling

This is the most significant architectural difference.

**V1** and **V2** work with `time.Time` / `*time.Time` for the `Created`, `Updated`, and `Closed` fields:

```go
// V1 (same pattern as V2)
now := time.Now().UTC().Truncate(time.Second)
t.Updated = now
t.Closed = &now   // pointer to time
t.Closed = nil     // clear on reopen
```

**V3** uses **string timestamps** in RFC3339 format:

```go
// V3's Task struct
Created     string   `json:"created"`
Updated     string   `json:"updated"`
Closed      string   `json:"closed,omitempty"`

// V3's transition code
now := time.Now().UTC().Format(time.RFC3339)
task.Updated = now
task.Closed = now    // string assignment
task.Closed = ""     // clear on reopen
```

V3's approach means timestamps are stored as strings throughout the domain model. This simplifies JSON serialization but loses type safety -- the compiler cannot prevent invalid timestamp strings from being assigned to these fields. V1/V2's `time.Time` approach provides compile-time type safety and eliminates the possibility of malformed timestamps in the struct.

#### Error Message Format

The spec requires: `Error: Cannot {command} task tick-{id} -- status is '{current_status}'`

**V1**: `"Cannot %s task %s -- status is '%s'"` -- uses em dash (--), matches spec exactly.

**V2**: `"Cannot %s task %s -- status is '%s'"` -- uses em dash (\u2014), matches spec exactly.

**V3**: `"cannot %s task %s - status is '%s'"` -- uses **lowercase** "cannot" and a **regular hyphen** instead of em dash. This deviates from the spec. The spec says "Cannot" (capital C) and uses an em dash.

#### Unknown Command Handling

**V2** and **V3** both handle unknown commands explicitly:

```go
// V2
transition, ok := validTransitions[command]
if !ok {
    return "", "", fmt.Errorf("unknown command: %s", command)
}

// V3
default:
    return TransitionResult{}, fmt.Errorf("unknown command: %s", command)
```

**V1** does **not** handle unknown commands -- `resolveTransition` simply returns `false` and the error message will say "Cannot {unknown_command} task ..." which is acceptable but less precise.

#### Closed-on-Reopen Guard

**V1** has a subtle extra guard on reopen:

```go
case StatusOpen:
    if command == "reopen" {
        t.Closed = nil
    }
```

This checks `command == "reopen"` before clearing `Closed`, which is technically unnecessary since the only way to reach `StatusOpen` as `newStatus` is via the `reopen` command. V2 and V3 unconditionally clear closed when transitioning to `StatusOpen`, which is simpler and equally correct.

### Code Quality

#### Go Idioms

**V1** uses a helper function `resolveTransition` that cleanly separates lookup logic from mutation logic. The composite key pattern (`type key struct{...}`) is idiomatic Go for multi-key maps but the map is allocated per call.

**V2** uses a package-level `var validTransitions` with an anonymous struct type, which is idiomatic for configuration-like data. The `for _, s := range transition.from` loop for membership testing is standard Go. V2 also handles `unknown command` as a separate error class.

**V3** uses a switch statement, which is the most natural Go control flow for a fixed set of known commands. The explicit boolean `valid` variable is clear. However, the string-based timestamp model is non-idiomatic for Go domain types.

#### Naming

All three use `Transition` as the function name. V1 names the task parameter `t`, V2 and V3 use `task`. The longer name `task` is clearer, especially since `t` is conventionally used for `*testing.T` in Go.

V1's `resolveTransition` helper is well-named. V2's `validTransitions` package variable is descriptive. V3 has no extracted helper -- everything is inline.

#### Error Handling

All three follow the same pattern: validate first, mutate only on success, return error without mutation on failure. This is correct.

V2 and V3 distinguish between "unknown command" and "invalid transition" errors. V1 lumps them together.

#### DRY

**V1**: Map-based lookup is DRY -- no repeated logic per command.

**V2**: Map-based lookup with anonymous struct is DRY. The `from` slice pattern eliminates repeated `||` chains.

**V3**: The switch statement has some repetition (`valid = oldStatus == ... || oldStatus == ...`) but it is minimal and the explicitness has value.

### Test Quality

#### V1 Test Functions

File: `internal/task/transition_test.go` (192 lines)

1. `TestTransition/valid transitions` -- table-driven, 7 subtests:
   - `start: open -> in_progress`
   - `done: open -> done`
   - `done: in_progress -> done`
   - `cancel: open -> cancelled`
   - `cancel: in_progress -> cancelled`
   - `reopen: done -> open`
   - `reopen: cancelled -> open`
   - Checks: task.Status, result.OldStatus, result.NewStatus

2. `TestTransition/invalid transitions` -- table-driven, 9 subtests:
   - `start on in_progress`, `start on done`, `start on cancelled`
   - `done on done`, `done on cancelled`
   - `cancel on done`, `cancel on cancelled`
   - `reopen on open`, `reopen on in_progress`
   - Checks: error is non-nil (does NOT check error message content)

3. `TestTransition/sets closed timestamp on done` -- time-window assertion

4. `TestTransition/sets closed timestamp on cancel` -- time-window assertion

5. `TestTransition/clears closed timestamp on reopen` -- sets closed, reopens, asserts nil

6. `TestTransition/updates timestamp on every valid transition` -- single test (start only), uses time.Sleep

7. `TestTransition/does not modify task on invalid transition` -- single invalid case (start on done), checks status, updated, closed

Helper functions:
- `makeTask(status Status) Task` -- creates task by value
- `containsAll(s string, subs ...string) bool` -- custom string search
- `contains(s, sub string) bool` -- wrapper
- `searchString(s, sub string) bool` -- manual substring search (reimplements `strings.Contains`)

**V1 test gaps:**
- Invalid transition tests do NOT verify error message content (only non-nil)
- `updated` timestamp test covers only 1 of 7 transitions
- `does not modify task` test covers only 1 of 9 invalid cases
- Custom `containsAll`/`searchString` reimplements stdlib `strings.Contains` unnecessarily

#### V2 Test Functions

File: `internal/task/task_test.go` (390 lines added, lines 363-752)

1. `TestTransition/it transitions open to in_progress via start` -- individual test, checks oldStatus, newStatus, task.Status
2. `TestTransition/it transitions open to done via done`
3. `TestTransition/it transitions in_progress to done via done`
4. `TestTransition/it transitions open to cancelled via cancel`
5. `TestTransition/it transitions in_progress to cancelled via cancel`
6. `TestTransition/it transitions done to open via reopen`
7. `TestTransition/it transitions cancelled to open via reopen`

8-16. Nine individual invalid transition tests, each with **exact error message matching**:
   - `it rejects start on in_progress task` -- asserts exact string `"Cannot start task tick-a3f2b7 -- status is 'in_progress'"`
   - `it rejects start on done task`
   - `it rejects start on cancelled task`
   - `it rejects done on done task`
   - `it rejects done on cancelled task`
   - `it rejects cancel on done task`
   - `it rejects cancel on cancelled task`
   - `it rejects reopen on open task`
   - `it rejects reopen on in_progress task`

17. `TestTransition/it sets closed timestamp when transitioning to done` -- time-window + UTC location check
18. `TestTransition/it sets closed timestamp when transitioning to cancelled` -- time-window assertion
19. `TestTransition/it clears closed timestamp when reopening` -- precondition check + nil assertion

20. `TestTransition/it updates the updated timestamp on every valid transition` -- **table-driven across all 7 transitions**, each with time.Sleep, before/after window, and comparison to original

21. `TestTransition/it does not modify task on invalid transition` -- **table-driven across all 9 invalid cases**, checks status, updated, closed (including nil vs non-nil comparison)

Helper function:
- `newTestTask(status Status) *Task` -- creates task as pointer, sets Closed for done/cancelled states

**V2 strengths:**
- Matches every spec test name exactly (e.g., "it transitions open to in_progress via start")
- Exact error message validation for all 9 invalid cases
- `updated` timestamp verified across all 7 transitions
- `does not modify task` verified across all 9 invalid cases
- `Closed` timestamp test includes UTC location assertion
- Done/cancelled test tasks get a pre-set Closed timestamp (realistic setup)

#### V3 Test Functions

File: `internal/task/transition_test.go` (413 lines)

1-7. Seven individual valid transition tests (same pattern as V2):
   - `it transitions open to in_progress via start`
   - `it transitions open to done via done`
   - `it transitions in_progress to done via done`
   - `it transitions open to cancelled via cancel`
   - `it transitions in_progress to cancelled via cancel`
   - `it transitions done to open via reopen`
   - `it transitions cancelled to open via reopen`

8-16. Nine individual invalid transition tests with **exact error message matching**:
   - Same set as V2, but error messages use lowercase "cannot" and hyphen: `"cannot start task tick-a3f2b7 - status is 'in_progress'"`

17. `it sets closed timestamp when transitioning to done` -- parses RFC3339 string, time-window assertion
18. `it sets closed timestamp when transitioning to cancelled` -- same pattern
19. `it clears closed timestamp when reopening` -- precondition check, asserts empty string

20. `it updates the updated timestamp on every valid transition` -- table-driven across all 7 transitions with RFC3339 parsing, before/after window

21. `it does not modify task on invalid transition` -- **single test case only** (start on done), checks status, updated, closed

Helper functions:
- `makeTask(id string, status Status) *Task` -- takes id parameter (more flexible)
- `makeClosedTask(id string, status Status) *Task` -- separate helper for closed tasks

**V3 test gaps:**
- `does not modify task` only tests 1 of 9 invalid cases (same gap as V1)
- Error format deviates from spec (lowercase, hyphen instead of em dash)

#### Test Coverage Diff

| Test Aspect | V1 | V2 | V3 |
|-------------|-----|-----|-----|
| Valid transitions (7) | All 7 (table) | All 7 (individual) | All 7 (individual) |
| Invalid transitions (9) | All 9 (table) | All 9 (individual) | All 9 (individual) |
| Error message content validated | Partial (1 test, substring) | Full (all 9, exact match) | Full (all 9, exact match) |
| Closed on done | Yes | Yes | Yes |
| Closed on cancel | Yes | Yes | Yes |
| Closed cleared on reopen | Yes | Yes | Yes |
| Updated on all valid transitions | 1 of 7 | 7 of 7 | 7 of 7 |
| Task unmodified on all invalid | 1 of 9 | 9 of 9 | 1 of 9 |
| UTC location assertion | No | Yes (done test) | No |
| Unknown command handling | Not tested | Not tested | Not tested |
| Spec test name matching | No | Yes (exact) | Yes (exact) |

Tests unique to V2 only:
- UTC location assertion on closed timestamp
- All 9 invalid cases verified for non-mutation (table-driven)
- All 7 transitions verified for updated timestamp change

Tests unique to V1 only:
- Custom `containsAll` helper (but reimplements stdlib)

Tests unique to V3 only:
- Separate `makeClosedTask` helper (good pattern)
- RFC3339 string parsing in timestamp assertions

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 4 (2 code, 2 docs) | 6 (2 code, 3 docs, 1 config) |
| Lines added | 257 | 444 (441 code + 3 docs) | 503 (484 code + 16 docs + 3 config) |
| Impl LOC | 65 | 51 | 71 |
| Test LOC | 192 | 390 | 413 |
| Test functions | 7 subtests + 3 helpers | 21 subtests + 1 helper | 21 subtests + 2 helpers |

## Verdict

**V2 is the best implementation.**

### Evidence

1. **Most thorough test coverage.** V2 is the only version that validates task non-mutation across all 9 invalid cases and verifies the `updated` timestamp across all 7 valid transitions. V1 and V3 test only 1 of 9 and 1 of 7 respectively for these critical criteria.

2. **Exact spec compliance.** V2 matches every test name from the spec verbatim ("it transitions open to in_progress via start"), validates exact error message strings for all 9 invalid cases (including the spec-required capital "Cannot" and em dash), and includes a UTC location assertion on the closed timestamp.

3. **Clean implementation design.** The package-level `validTransitions` map with `from []Status` slices is both DRY and efficient (single allocation at package init). The anonymous struct pattern is idiomatic Go. At 51 implementation lines, it is the most concise.

4. **Correct timestamp types.** V2 uses `time.Time` and `*time.Time` for the task fields, matching V1. V3's string-based timestamps lose type safety and deviate from the V1/V2 base model.

5. **Error format correctness.** V2 matches the spec format exactly: `"Cannot %s task %s -- status is '%s'"`. V3 deviates with lowercase "cannot" and a regular hyphen.

**V2's only weakness** is the return signature `(oldStatus, newStatus, err)` instead of a `TransitionResult` struct. Bare return values are slightly less self-documenting than a named struct, but this is a minor stylistic preference and does not affect correctness or usability.

**V1** is a solid, compact implementation but has notably weaker test coverage: error messages are only checked via substring for a single case, `updated` is verified for only 1 transition, and non-mutation is checked for only 1 invalid case. The custom `containsAll`/`searchString` helpers unnecessarily reimplement `strings.Contains`.

**V3** has comprehensive individual tests but falls short on spec compliance (error message format) and makes a debatable architectural choice with string timestamps. Its `does not modify task` test covers only a single case.

### Ranking: V2 > V3 > V1
