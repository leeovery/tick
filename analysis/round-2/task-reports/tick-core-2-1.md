# Task tick-core-2-1: Status Transition Validation

## Task Summary

Implement a pure `Transition(task *Task, command string) error` function that enforces the 4 valid status transitions (`start`, `done`, `cancel`, `reopen`), rejects all invalid ones, and manages `closed` and `updated` timestamps as side effects. The function must return old and new status for output formatting.

**Valid transitions:**
- `start`: `open` -> `in_progress`
- `done`: `open` or `in_progress` -> `done`
- `cancel`: `open` or `in_progress` -> `cancelled`
- `reopen`: `done` or `cancelled` -> `open`

**Side effects:** `updated` refreshed on every valid transition; `closed` set on done/cancelled, cleared on reopen. Task must not be mutated on error.

**Error format:** `Error: Cannot {command} task tick-{id} — status is '{current_status}'`

**Acceptance criteria:**
1. All 7 valid status transitions succeed with correct new status
2. All 9 invalid transitions return error
3. Task not modified on invalid transition
4. `closed` set to current UTC on done/cancelled
5. `closed` cleared on reopen
6. `updated` refreshed on every valid transition
7. Error messages include command name and current status
8. Function returns old and new status

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| All 7 valid transitions succeed | PASS — 7 individual subtests each verify status, oldStatus, newStatus | PASS — table-driven test covers all 7 transitions |
| All 9 invalid transitions return error | PASS — 9 individual subtests each verify error returned | PASS — table-driven test covers all 9 invalid cases |
| Task not modified on invalid transition | PASS — table-driven test checks Status, Updated, Closed unchanged for all 9 | PARTIAL — only tests one case (`start` on `StatusDone`), not all 9 |
| `closed` set to current UTC on done/cancelled | PASS — explicit tests for both done and cancel | PASS — explicit tests for both done and cancel |
| `closed` cleared on reopen | PASS — explicit test with precondition check | PASS — explicit test with precondition check |
| `updated` refreshed on every valid transition | PASS — table-driven test covers all 7 transitions | PASS — table-driven test covers all 7 transitions |
| Error messages include command and status | PASS — each invalid test asserts exact error message | PASS — each invalid test asserts exact error message |
| Function returns old and new status | PASS — returns `(oldStatus, newStatus, err)` tuple | PASS — returns `*TransitionResult` struct with OldStatus/NewStatus |

## Implementation Comparison

### Approach

**File organization:**

V2 adds the transition logic directly to the existing `internal/task/task.go` file (51 new lines appended). Tests go into the existing `internal/task/task_test.go` (390 new lines appended).

V4 creates a dedicated `internal/task/transition.go` file (68 lines) and a corresponding `internal/task/transition_test.go` file (226 lines).

V4's approach is genuinely better here — separating transition logic into its own file follows Go's convention of organizing by responsibility within a package, making the code easier to navigate as the codebase grows.

**Return type:**

V2 returns a bare tuple:
```go
func Transition(task *Task, command string) (oldStatus Status, newStatus Status, err error) {
```

V4 returns a result struct:
```go
type TransitionResult struct {
    OldStatus Status
    NewStatus Status
}

func Transition(t *Task, command string) (*TransitionResult, error) {
```

V4's `TransitionResult` struct is more extensible and idiomatic Go for functions returning multiple related values. However, V2's named return values make the signature self-documenting without needing to look up the struct definition. Both approaches are valid; V4 is marginally better for future extensibility.

**Transition map — both identical in structure:**

V2:
```go
var validTransitions = map[string]struct {
    from []Status
    to   Status
}{
    "start":  {from: []Status{StatusOpen}, to: StatusInProgress},
    "done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
    "cancel": {from: []Status{StatusOpen, StatusInProgress}, to: StatusCancelled},
    "reopen": {from: []Status{StatusDone, StatusCancelled}, to: StatusOpen},
}
```

V4:
```go
var validTransitions = map[string]struct {
    from []Status
    to   Status
}{
    "start":  {from: []Status{StatusOpen}, to: StatusInProgress},
    "done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
    "cancel": {from: []Status{StatusOpen, StatusInProgress}, to: StatusCancelled},
    "reopen": {from: []Status{StatusDone, StatusCancelled}, to: StatusOpen},
}
```

These are character-for-character identical. Both use an anonymous struct type with `from` (slice of allowed source statuses) and `to` (target status).

**Status membership check:**

V2 inlines the loop:
```go
allowed := false
for _, s := range transition.from {
    if currentStatus == s {
        allowed = true
        break
    }
}
if !allowed {
    return "", "", fmt.Errorf(...)
}
```

V4 extracts a helper:
```go
func statusIn(s Status, list []Status) bool {
    for _, item := range list {
        if s == item {
            return true
        }
    }
    return false
}
```

Then uses it as:
```go
if !statusIn(t.Status, rule.from) {
    return nil, fmt.Errorf(...)
}
```

V4's extracted helper is cleaner and more reusable — a minor but genuine improvement in DRY principle.

**Error message format:**

V2 error for invalid transitions:
```go
fmt.Errorf("Cannot %s task %s — status is '%s'", command, task.ID, currentStatus)
```

V4 error for invalid transitions:
```go
fmt.Errorf("Error: Cannot %s task %s — status is '%s'", command, t.ID, t.Status)
```

V4 prefixes with `"Error: "`. The spec says: `Error: Cannot {command} task tick-{id} — status is '{current_status}'`. **V4 matches the spec exactly. V2 omits the `Error: ` prefix, which is a spec deviation.**

V2 error for unknown command:
```go
fmt.Errorf("unknown command: %s", command)
```

V4 error for unknown command:
```go
fmt.Errorf("Error: Cannot %s task %s — unknown command", command, t.ID)
```

V4 includes the task ID in the unknown command error and follows the `Error:` prefix pattern. V2's is simpler but less consistent with the spec format.

**Timestamp and status mutation — effectively identical logic in both:**

```go
now := time.Now().UTC().Truncate(time.Second)
t.Status = rule.to
t.Updated = now

switch rule.to {
case StatusDone, StatusCancelled:
    t.Closed = &now
case StatusOpen:
    t.Closed = nil
}
```

### Code Quality

**Naming:**

V2 uses `task` as the parameter name, `transition` for the looked-up rule, `currentStatus` for the saved status. Clear and descriptive.

V4 uses `t` as the parameter name (idiomatic Go short receiver-style naming), `rule` for the looked-up transition. Also clear. The short `t` name could cause confusion in test files where `t` typically refers to `*testing.T`, but since this is in the implementation file, it's acceptable.

**Error handling:**

V2 returns `("", "", error)` for zero-value status strings on error. This works because `Status` is a string type, but callers must check error before using old/new status.

V4 returns `(nil, error)` — the nil pointer for `*TransitionResult` makes it impossible to accidentally use the result without checking error, which is slightly safer.

**DRY:**

V4's `statusIn` helper is reusable. V2 inlines the same logic, which is fine for a single use but would require duplication if status membership checks were needed elsewhere.

**Variable naming in the transition map:**

V2: `transition, ok := validTransitions[command]`
V4: `rule, ok := validTransitions[command]`

Both are reasonable. `rule` is slightly more descriptive of what the map entry represents.

### Test Quality

**V2 test functions (within `TestTransition`):**

All tests are within a single `TestTransition` parent function with 20 subtests:

1. `"it transitions open to in_progress via start"` — individual subtest
2. `"it transitions open to done via done"` — individual subtest
3. `"it transitions in_progress to done via done"` — individual subtest
4. `"it transitions open to cancelled via cancel"` — individual subtest
5. `"it transitions in_progress to cancelled via cancel"` — individual subtest
6. `"it transitions done to open via reopen"` — individual subtest
7. `"it transitions cancelled to open via reopen"` — individual subtest
8. `"it rejects start on in_progress task"` — individual subtest
9. `"it rejects start on done task"` — individual subtest
10. `"it rejects start on cancelled task"` — individual subtest
11. `"it rejects done on done task"` — individual subtest
12. `"it rejects done on cancelled task"` — individual subtest
13. `"it rejects cancel on done task"` — individual subtest
14. `"it rejects cancel on cancelled task"` — individual subtest
15. `"it rejects reopen on open task"` — individual subtest
16. `"it rejects reopen on in_progress task"` — individual subtest
17. `"it sets closed timestamp when transitioning to done"` — individual subtest
18. `"it sets closed timestamp when transitioning to cancelled"` — individual subtest
19. `"it clears closed timestamp when reopening"` — individual subtest
20. `"it updates the updated timestamp on every valid transition"` — table-driven with 7 subtests
21. `"it does not modify task on invalid transition"` — table-driven with 9 subtests

Total: **20 top-level subtests** expanding to **36 leaf tests** (20 - 2 table parents + 7 + 9).

V2 uses a mix: individual subtests for the 7 valid transitions and 9 invalid transitions (where each test verifies exact error messages), table-driven for the timestamp-updated and no-modification-on-error tests. Helper function `newTestTask` returns `*Task` (pointer).

**V4 test functions:**

4 top-level test functions:

1. `TestTransition_ValidTransitions` — table-driven, 7 subtests:
   - `"it transitions open to in_progress via start"`
   - `"it transitions open to done via done"`
   - `"it transitions in_progress to done via done"`
   - `"it transitions open to cancelled via cancel"`
   - `"it transitions in_progress to cancelled via cancel"`
   - `"it transitions done to open via reopen"`
   - `"it transitions cancelled to open via reopen"`

2. `TestTransition_InvalidTransitions` — table-driven, 9 subtests:
   - `"it rejects start on in_progress task"`
   - `"it rejects start on done task"`
   - `"it rejects start on cancelled task"`
   - `"it rejects done on done task"`
   - `"it rejects done on cancelled task"`
   - `"it rejects cancel on done task"`
   - `"it rejects cancel on cancelled task"`
   - `"it rejects reopen on open task"`
   - `"it rejects reopen on in_progress task"`

3. `TestTransition_ClosedTimestamp` — 3 subtests:
   - `"it sets closed timestamp when transitioning to done"`
   - `"it sets closed timestamp when transitioning to cancelled"`
   - `"it clears closed timestamp when reopening"`

4. `TestTransition_UpdatedTimestamp` — 1 parent subtest with 7 table-driven subtests:
   - `"it updates the updated timestamp on every valid transition"` (with subtests per command)

5. `TestTransition_NoModificationOnError` — 1 subtest:
   - `"it does not modify task on invalid transition"` (tests only `start` on `StatusDone`)

Total: **5 top-level test functions** expanding to **27 leaf tests**.

**Test helper comparison:**

V2's `newTestTask` returns `*Task`:
```go
func newTestTask(status Status) *Task {
    now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
    t := &Task{
        ID:       "tick-a3f2b7",
        Title:    "Test task",
        Status:   status,
        Priority: 2,
        Created:  now,
        Updated:  now,
    }
    if status == StatusDone || status == StatusCancelled {
        closed := now
        t.Closed = &closed
    }
    return t
}
```

V4's `makeTask` returns `Task` (value type) and takes an ID parameter:
```go
func makeTask(id string, status Status) Task {
    now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
    t := Task{
        ID:      id,
        Title:   "Test task",
        Status:  status,
        Created: now,
        Updated: now,
    }
    if status == StatusDone || status == StatusCancelled {
        closed := now
        t.Closed = &closed
    }
    return t
}
```

V4's `makeTask` accepting an `id` parameter is slightly more flexible, though in practice both always use `"tick-a3f2b7"`. V4 returns a value type, requiring `&task` at call sites — this is slightly less convenient but means the task is always freshly allocated at the call site.

**Critical test coverage gap in V4:**

V4's `TestTransition_NoModificationOnError` only tests a single case (`start` on a `StatusDone` task). V2 tests all 9 invalid transitions in a table-driven test to verify no mutation occurs. This is a significant coverage gap in V4 — it only proves immutability for 1 of 9 error paths.

**Assertion quality:**

V2 valid transition tests check 3 things each: `oldStatus`, `newStatus`, and `task.Status`. Error messages use `t.Errorf` with `%q` formatting for clear output.

V4 valid transition tests also check 3 things: `task.Status`, `result.OldStatus`, `result.NewStatus`. Similar quality.

V2 invalid transition tests: each test individually verifies the exact error message string. This is redundant but extremely explicit.

V4 invalid transition tests: table-driven with constructed expected error messages. More concise, same coverage.

V2 no-modification test checks: `Status`, `Updated`, `Closed` (with nil-safe comparison). Covers all 9 invalid cases.

V4 no-modification test checks: `Status`, `Updated`, `Closed` (with nil-safe comparison). But only for 1 case.

**UTC timezone verification:**

V2 includes an explicit UTC timezone check in the "sets closed timestamp when transitioning to done" test:
```go
if tk.Closed.Location() != time.UTC {
    t.Errorf("task.Closed timezone = %v, want UTC", tk.Closed.Location())
}
```

V4 does not explicitly verify the timezone is UTC. Since the implementation uses `.UTC()`, this would pass anyway, but V2's explicit assertion documents the requirement.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 4 (task.go, task_test.go, 2 workflow docs) | 4 (transition.go, transition_test.go, 2 workflow docs) |
| Lines added | 444 | 298 |
| Impl LOC | 51 (added to task.go) | 68 (new transition.go) |
| Test LOC | 390 (added to task_test.go) | 226 (new transition_test.go) |
| Test functions | 1 top-level / 20 subtests / 36 leaf tests | 5 top-level / 20 subtests / 27 leaf tests |

## Verdict

**V2 is the better implementation of this task**, though V4 has some structural advantages.

**Where V4 wins:**
- Separate `transition.go` file is better organization
- `TransitionResult` struct is more extensible than a bare tuple
- Extracted `statusIn` helper is cleaner
- Error messages match the spec format exactly (with `Error:` prefix)
- More concise table-driven test style (226 vs 390 lines)

**Where V2 wins:**
- **Complete no-modification coverage**: V2 tests all 9 invalid transitions for immutability; V4 tests only 1. This is the most significant difference.
- **Explicit UTC timezone assertion**: V2 verifies the closed timestamp is in UTC; V4 does not.
- **Spec error format**: While V4's `Error:` prefix matches the spec, V2's format without the prefix is arguably better Go style (errors should not be capitalized or prefixed with "Error:" per Go conventions). However, since the spec explicitly requires this format, V4 is technically more compliant here.

The decisive factor is test completeness. V4's `TestTransition_NoModificationOnError` testing only 1 of 9 invalid cases is a meaningful gap — if a future refactor introduced a bug where one specific invalid transition path accidentally mutated the task, V4's tests would not catch it. V2's exhaustive coverage of all 9 cases is the correct approach for a spec that explicitly lists "each invalid must return error and leave task unmodified" as an edge case requirement.

Overall, V2 delivers more thorough test coverage despite being more verbose, while V4 has better code organization and spec compliance on error format. In a codebase where correctness guarantees matter, V2's test thoroughness outweighs V4's structural elegance.
