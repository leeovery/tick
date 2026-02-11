# Task tick-core-2-1: Status transition validation logic

## Task Summary

Implement a pure `Transition(task *Task, command string) error` function that enforces 4 valid status transitions (`start`, `done`, `cancel`, `reopen`) across the 4 task statuses (`open`, `in_progress`, `done`, `cancelled`). Must manage `closed` and `updated` timestamps as side effects, reject all 9 invalid transitions with a specific error format, leave the task unmodified on error, and return old/new status for output formatting.

**Acceptance Criteria:**
1. All 7 valid status transitions succeed with correct new status
2. All 9 invalid transitions return error
3. Task not modified on invalid transition
4. `closed` set to current UTC on done/cancelled
5. `closed` cleared on reopen
6. `updated` refreshed on every valid transition
7. Error messages include command name and current status
8. Function returns old and new status

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| All 7 valid status transitions succeed with correct new status | PASS -- `validTransitions` map covers all 7 pairs; tested in "valid transitions" table-driven subtest with all 7 cases | PASS -- `transitionTable` covers all 7 pairs; tested in `TestTransition_ValidTransitions` with all 7 cases |
| All 9 invalid transitions return error | PASS -- Tested in "invalid transitions" table-driven subtest with all 9 cases | PASS -- Tested in `TestTransition_InvalidTransitions` with all 9 cases |
| Task not modified on invalid transition | PASS -- Validated by early-return before mutation; tested in "it does not modify task on invalid transition" subtest checking Status, Updated, and Closed | PASS -- Validated by early-return before mutation; tested in `TestTransition_NoModificationOnInvalid` checking Status, Updated, and Closed |
| `closed` set to current UTC on done/cancelled | PASS -- `switch newStatus { case StatusDone, StatusCancelled: t.Closed = &now }`; dedicated tests for done and cancel | PASS -- `switch rule.to { case StatusDone, StatusCancelled: t.Closed = &now }`; dedicated tests in `TestTransition_ClosedTimestamp` |
| `closed` cleared on reopen | PASS -- `case StatusOpen: t.Closed = nil`; tested in "it clears closed timestamp when reopening" | PASS -- `case StatusOpen: t.Closed = nil`; tested in `TestTransition_ClosedTimestamp/"it clears closed timestamp when reopening"` |
| `updated` refreshed on every valid transition | PASS -- `t.Updated = now` on all valid paths; tested in "it updates the updated timestamp on every valid transition" across all 7 from/command pairs | PASS -- `t.Updated = now` on all valid paths; tested in `TestTransition_UpdatedTimestamp` but only 4 of 7 pairs (missing `in_progress->done`, `open->cancel`, `cancelled->reopen`) |
| Error messages include command name and current status | PASS -- Format `"Cannot %s task %s — status is '%s'"` includes both; tests verify exact format match | PARTIAL -- Two distinct error formats: unknown command uses `"unknown command %q"` (no status), valid command + invalid state uses `"cannot %s task %s — status is '%s'"`. The unknown-command path omits current status. For known commands with invalid states, both command and status are present. |
| Function returns old and new status | PASS -- Returns `TransitionResult{OldStatus, NewStatus}`; tests check both fields in valid transition table | PASS -- Returns `TransitionResult{OldStatus, NewStatus}`; tests check both fields in valid transition table |

## Implementation Comparison

### Approach

Both versions implement the same public API: `Transition(t *Task, command string) (TransitionResult, error)` with an identical `TransitionResult` struct. The core logic is the same: look up the command, validate the current status against allowed source statuses, mutate the task, and return the result. The key structural differences are in file organization and the transition table data structure.

**File organization:**

V5 adds the transition logic directly into the existing `internal/task/task.go` file (lines 171-213) and adds tests to `internal/task/task_test.go`:
```go
// V5: internal/task/task.go — appended after NormalizeID
var validTransitions = map[string]map[Status]Status{
    "start":  {StatusOpen: StatusInProgress},
    "done":   {StatusOpen: StatusDone, StatusInProgress: StatusDone},
    "cancel": {StatusOpen: StatusCancelled, StatusInProgress: StatusCancelled},
    "reopen": {StatusDone: StatusOpen, StatusCancelled: StatusOpen},
}
```

V6 creates a separate `internal/task/transition.go` file (75 lines) with a dedicated `internal/task/transition_test.go` (308 lines):
```go
// V6: internal/task/transition.go — standalone file
var transitionTable = map[string]struct {
    from []Status
    to   Status
}{
    "start":  {from: []Status{StatusOpen}, to: StatusInProgress},
    "done":   {from: []Status{StatusOpen, StatusInProgress}, to: StatusDone},
    "cancel": {from: []Status{StatusOpen, StatusInProgress}, to: StatusCancelled},
    "reopen": {from: []Status{StatusDone, StatusCancelled}, to: StatusOpen},
}
```

**Transition table structure:**

V5 uses a nested `map[string]map[Status]Status` — the outer map keyed by command, the inner map keyed by source status mapping to target status. Lookup is `O(1)` via two map lookups:
```go
// V5: internal/task/task.go
transitions, ok := validTransitions[command]
if !ok { ... }
newStatus, ok := transitions[t.Status]
if !ok { ... }
```

V6 uses `map[string]struct{ from []Status; to Status }` — the outer map keyed by command, with an anonymous struct holding a slice of valid source statuses and a single target status. Lookup requires a linear scan of the `from` slice via a helper function:
```go
// V6: internal/task/transition.go
rule, ok := transitionTable[command]
if !ok { ... }
if !statusIn(t.Status, rule.from) { ... }
```

With the helper:
```go
// V6: internal/task/transition.go lines 69-75
func statusIn(s Status, statuses []Status) bool {
    for _, v := range statuses {
        if s == v {
            return true
        }
    }
    return false
}
```

**Error handling for unknown commands:**

V5 treats an unknown command identically to an invalid transition — both return the same error format:
```go
// V5: both unknown command and invalid from-status produce:
"Cannot %s task %s — status is '%s'"
```

V6 differentiates unknown commands from invalid transitions:
```go
// V6: unknown command (line 41):
fmt.Errorf("unknown command %q", command)

// V6: valid command but invalid from-status (line 44-47):
fmt.Errorf("cannot %s task %s — status is '%s'", command, t.ID, t.Status)
```

**Error message casing:**

V5 uses capitalized `"Cannot"` matching the spec exactly: `"Error: Cannot {command} task tick-{id} — status is '{current_status}'"`. Note: neither version includes the `"Error: "` prefix from the spec.

V6 uses lowercase `"cannot"` which deviates from the spec's capitalized form.

**Spec error format (from task plan):**
```
Error: Cannot {command} task tick-{id} — status is '{current_status}'
```

V5 produces: `Cannot start task tick-a3f2b7 — status is 'done'` (missing "Error: " prefix, correct casing)
V6 produces: `cannot start task tick-a3f2b7 — status is 'done'` (missing "Error: " prefix, lowercase)

Neither version includes the `"Error: "` prefix from the spec, but V5 is closer to the spec's casing.

### Code Quality

**V5:**
- 46 lines added to `task.go` — compact, inline with the rest of the model
- Uses the idiomatic Go pattern of `map[K1]map[K2]V` for a two-key lookup; `O(1)` for both lookups
- No helper functions needed — the nested map handles the lookup directly
- Single error format for all rejection paths — simpler but less informative for truly unknown commands
- All exported types/functions have doc comments: `TransitionResult`, `Transition`, `validTransitions`
- Follows `fmt.Errorf` pattern consistently

**V6:**
- 75 lines in a dedicated file — cleaner separation of concerns
- Uses an anonymous struct in the map value, requiring a `statusIn` helper (7 lines)
- The `statusIn` helper is unexported and well-scoped but adds a linear scan (negligible with max 2 items)
- Two distinct error paths — more precise error reporting for unknown commands vs invalid transitions
- All exported types/functions have doc comments
- Includes inline comment `// reopen clears closed` in the switch — slightly more readable
- Slightly more verbose: `rule.to` instead of direct `newStatus` from map lookup

**Both versions:**
- Use `time.Now().UTC().Truncate(time.Second)` consistently
- Use pointer-based mutation of the `*Task` argument
- Return `TransitionResult` with `OldStatus` and `NewStatus`
- Handle `closed` via switch on new status
- No I/O — pure domain logic as required

### Test Quality

**V5 Tests** (236 lines added to `task_test.go`):

All tests live in `TestTransition` as subtests:

1. `"valid transitions"` — table-driven, 7 sub-cases:
   - `"it transitions open to in_progress via start"`
   - `"it transitions open to done via done"`
   - `"it transitions in_progress to done via done"`
   - `"it transitions open to cancelled via cancel"`
   - `"it transitions in_progress to cancelled via cancel"`
   - `"it transitions done to open via reopen"`
   - `"it transitions cancelled to open via reopen"`
   Each checks: `result.OldStatus`, `result.NewStatus`, `tk.Status`, `tk.Updated` in time range, `tk.Closed` set/nil as appropriate.

2. `"invalid transitions"` — table-driven, 9 sub-cases:
   - `"it rejects start on in_progress task"`
   - `"it rejects start on done task"`
   - `"it rejects start on cancelled task"`
   - `"it rejects done on done task"`
   - `"it rejects done on cancelled task"`
   - `"it rejects cancel on done task"`
   - `"it rejects cancel on cancelled task"`
   - `"it rejects reopen on open task"`
   - `"it rejects reopen on in_progress task"`
   Each checks: error non-nil, error contains command, error contains status, exact error format match.

3. `"it does not modify task on invalid transition"` — creates a done task with closed timestamp, attempts invalid `start`, verifies Status, Updated, and Closed unchanged.

4. `"it sets closed timestamp when transitioning to done"` — verifies precondition (Closed nil), transitions, checks Closed set within time range.

5. `"it sets closed timestamp when transitioning to cancelled"` — same pattern for cancel command.

6. `"it clears closed timestamp when reopening"` — verifies precondition (Closed set), transitions via reopen, checks Closed nil.

7. `"it updates the updated timestamp on every valid transition"` — table-driven across all 7 valid from/command pairs, checks Updated was refreshed beyond original value.

Total distinct test names matching spec: **all 21 tests from the spec are covered**. The valid-transitions table implicitly covers the `closed` and `updated` checks, with dedicated subtests for explicit coverage of each named spec test.

Helper function:
```go
// V5: task_test.go
makeTask := func(status Status, closed bool) Task {
    // returns value type, uses bool for closed presence
}
```

**V6 Tests** (308 lines in `transition_test.go`):

Tests are split across 4 top-level test functions:

1. `TestTransition_ValidTransitions` — table-driven, 7 sub-cases:
   - `"it transitions open to in_progress via start"`
   - `"it transitions open to done via done"`
   - `"it transitions in_progress to done via done"`
   - `"it transitions open to cancelled via cancel"`
   - `"it transitions in_progress to cancelled via cancel"`
   - `"it transitions done to open via reopen"`
   - `"it transitions cancelled to open via reopen"`
   Each checks: `task.Status`, `result.OldStatus`, `result.NewStatus`, `task.Updated` in time range.
   **Note:** Valid transition tests do NOT verify `task.Closed` behavior (no wantClosed field in the table).

2. `TestTransition_InvalidTransitions` — table-driven, 9 sub-cases:
   - `"it rejects start on in_progress task"`
   - `"it rejects start on done task"`
   - `"it rejects start on cancelled task"`
   - `"it rejects done on done task"`
   - `"it rejects done on cancelled task"`
   - `"it rejects cancel on done task"`
   - `"it rejects cancel on cancelled task"`
   - `"it rejects reopen on open task"`
   - `"it rejects reopen on in_progress task"`
   Each checks: error non-nil, exact error format match.
   **Note:** Uses lowercase `"cannot"` in expected message, matching V6's implementation but not the spec.

3. `TestTransition_ClosedTimestamp` — 3 sub-cases:
   - `"it sets closed timestamp when transitioning to done"`
   - `"it sets closed timestamp when transitioning to cancelled"`
   - `"it clears closed timestamp when reopening"`

4. `TestTransition_UpdatedTimestamp` — 1 sub-case with 4 inner iterations:
   - `"it updates the updated timestamp on every valid transition"` with sub-tests: `start`, `done`, `cancel`, `reopen`
   **Note:** Only 4 of 7 valid transitions tested. Missing: `in_progress->done`, `open->cancel` (only tests `in_progress->cancel`), `cancelled->reopen` (only tests `done->reopen`).

5. `TestTransition_NoModificationOnInvalid` — 1 sub-case:
   - `"it does not modify task on invalid transition"`

Total distinct test names matching spec: **all 21 spec tests are covered**, though `TestTransition_UpdatedTimestamp` covers only 4 representative transitions rather than all 7.

Helper functions (package-level):
```go
// V6: transition_test.go
func makeTask(status Status, closed *time.Time) *Task {
    // returns pointer type, uses *time.Time for closed
}

func closedTime() *time.Time {
    // returns pointer to fixed time
}
```

**Test quality comparison:**

| Aspect | V5 | V6 |
|--------|-----|-----|
| Closed timestamp checked in valid transitions | Yes — `wantClosed` field in table | No — valid transition table lacks closed verification |
| Updated timestamp exhaustiveness | All 7 transitions | Only 4 of 7 |
| Error message verification | Contains + exact match | Exact match only |
| Helper returns | Value type + bool for closed | Pointer type + `*time.Time` for closed |
| makeTask scope | Local closure in `TestTransition` | Package-level function (visible to other test files) |
| Helper reuse potential | Low (function-scoped) | High (package-scoped, separate file) |
| Test organization | Single `TestTransition` with subtests | 4 separate top-level functions |

### Skill Compliance

| Skill Constraint | V5 | V6 |
|------------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS — code follows standard formatting | PASS — code follows standard formatting |
| Handle all errors explicitly | PASS — all error paths return errors | PASS — all error paths return errors |
| Write table-driven tests with subtests | PASS — table-driven for valid and invalid transitions | PASS — table-driven for valid and invalid transitions |
| Document all exported functions, types, and packages | PASS — `TransitionResult`, `Transition`, `validTransitions` all documented | PASS — `TransitionResult`, `Transition` documented; `transitionTable` is unexported |
| Propagate errors with fmt.Errorf("%w", err) | N/A — no error wrapping needed (errors are terminal) | N/A — no error wrapping needed |
| MUST NOT ignore errors | PASS | PASS |
| MUST NOT use panic | PASS | PASS |
| MUST NOT hardcode configuration | PASS — transition rules are data-driven | PASS — transition rules are data-driven |

### Spec-vs-Convention Conflicts

1. **Error format prefix:** The spec says `"Error: Cannot {command} task tick-{id} — status is '{current_status}'"`. Neither version includes the `"Error: "` prefix. This is likely intentional — Go convention is to return error values without "Error:" prefixes, since `err.Error()` already contextualizes it. Both versions are reasonable.

2. **Error casing:** The spec capitalizes `"Cannot"`. V5 matches this; V6 uses lowercase `"cannot"`. Go convention (per Go Code Review Comments) says error strings should not be capitalized. V6 follows Go convention; V5 follows the spec.

3. **V6 unknown command error:** V6 produces `"unknown command %q"` for unknown commands, which does not match the spec format at all. V5 produces the spec format even for unknown commands (using the current status as context). The spec doesn't explicitly address unknown commands, but the single error format is more predictable for callers.

4. **V6 valid-transition tests omit closed verification:** The spec test list includes "it sets closed timestamp when transitioning to done" as a separate test. V6 does have this as a separate test in `TestTransition_ClosedTimestamp`, but the valid-transitions table does not cross-verify closed behavior. V5's valid-transitions table includes a `wantClosed` field for comprehensive inline verification.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed (task-relevant) | 2 (task.go, task_test.go) | 2 (transition.go, transition_test.go) |
| New files created | 0 | 2 |
| Implementation lines added | 46 | 75 |
| Test lines added | 236 | 308 |
| Total lines added | 282 | 383 |
| Helper functions | 0 (uses nested map) | 1 (`statusIn`) |
| Test helper functions | 1 (closure `makeTask`) | 2 (package-level `makeTask`, `closedTime`) |
| Exported types added | 1 (`TransitionResult`) | 1 (`TransitionResult`) |
| Exported functions added | 1 (`Transition`) | 1 (`Transition`) |

## Verdict

**V5 is the stronger implementation.**

Both versions produce correct, working transition logic with the same public API and both cover all acceptance criteria. The differences are:

**V5 advantages:**
- More concise: 282 total lines vs 383 — 36% less code for the same functionality
- More elegant data structure: `map[string]map[Status]Status` gives O(1) two-key lookup with no helper function, vs V6's struct+slice requiring a `statusIn` linear scan
- Error format closer to spec: capitalized `"Cannot"` matches the spec; single consistent error format for all rejection paths
- More thorough tests: valid-transition table includes `wantClosed` field verifying closed timestamp inline; updated-timestamp test covers all 7 transitions
- Error message tests include both `strings.Contains` checks (command and status) AND exact format match — defense in depth

**V6 advantages:**
- Better file organization: dedicated `transition.go` separates transition logic from the model definition, following single-responsibility at the file level
- Package-level test helpers (`makeTask`, `closedTime`) are reusable across test files in the package
- Lowercase error strings follow Go convention (Go Code Review Comments)
- Distinct error for unknown commands (`"unknown command %q"`) is more precise, though it deviates from the spec format
- Inline comment `// reopen clears closed` improves readability of the switch case

**V6 weaknesses:**
- `TestTransition_ValidTransitions` does not verify `Closed` field behavior — a gap in coverage
- `TestTransition_UpdatedTimestamp` covers only 4 of 7 valid transitions
- Error format uses lowercase `"cannot"` deviating from spec without documenting the reason
- Unknown-command error path (`"unknown command %q"`) omits the current status, partially failing acceptance criterion 7
- 36% more code with no additional functional benefit

V5's compact nested-map approach is more idiomatic for this particular problem shape (command -> from-status -> to-status), and its test suite is more exhaustive despite being shorter. V6's file separation is a legitimate organizational advantage that would matter more in a larger codebase, but for this focused feature the additional indirection and code volume do not pay off.
