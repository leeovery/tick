# Task tick-core-2-1: Status transition validation logic

## Task Summary

This task implements pure domain logic for status transitions in the task lifecycle. A `Transition(task *Task, command string) error` function must apply one of 4 commands (`start`, `done`, `cancel`, `reopen`) to move tasks through the lifecycle: `open -> in_progress -> done/cancelled`, with `reopen` to reverse closures. Valid transitions:

- `start`: open -> in_progress
- `done`: open or in_progress -> done
- `cancel`: open or in_progress -> cancelled
- `reopen`: done or cancelled -> open

On valid transition: set new status, refresh `updated` to current UTC, set `closed` on done/cancelled, clear `closed` on reopen. On invalid transition: return error without modifying task. Error format: `Error: Cannot {command} task tick-{id} -- status is '{current_status}'`. The function must return old and new status for output formatting. Pure domain logic with no I/O.

### Acceptance Criteria (from plan)

1. All 7 valid status transitions succeed with correct new status
2. All 9 invalid transitions return error
3. Task not modified on invalid transition
4. `closed` set to current UTC on done/cancelled
5. `closed` cleared on reopen
6. `updated` refreshed on every valid transition
7. Error messages include command name and current status
8. Function returns old and new status

## Acceptance Criteria Compliance

| Criterion | V4 | V5 |
|-----------|-----|-----|
| All 7 valid status transitions succeed with correct new status | PASS -- `TestTransition_ValidTransitions` covers all 7 pairs (open->start, open->done, in_progress->done, open->cancel, in_progress->cancel, done->reopen, cancelled->reopen); transition logic in `Transition()` uses `validTransitions` map with `from []Status` and `to Status` | PASS -- `TestTransition` "valid transitions" subtest covers same 7 pairs; transition logic uses `map[string]map[Status]Status` for direct from->to lookup |
| All 9 invalid transitions return error | PASS -- `TestTransition_InvalidTransitions` covers all 9 invalid pairs with exact error message assertions | PASS -- `TestTransition` "invalid transitions" subtest covers all 9 invalid pairs with exact error message assertions |
| Task not modified on invalid transition | PASS -- `TestTransition_NoModificationOnError` creates original copy and verifies status, updated, and closed are unchanged after failed transition | PASS -- `TestTransition` "it does not modify task on invalid transition" subtest verifies same three fields unchanged |
| `closed` set to current UTC on done/cancelled | PASS -- `TestTransition_ClosedTimestamp` has dedicated subtests for done and cancel; both verify `Closed` is set and within time bracket | PASS -- dedicated subtests "it sets closed timestamp when transitioning to done" and "it sets closed timestamp when transitioning to cancelled"; additionally the valid transitions table includes `wantClosed` column checking this on every valid transition |
| `closed` cleared on reopen | PASS -- `TestTransition_ClosedTimestamp` "it clears closed timestamp when reopening" verifies `Closed` is nil after reopen | PASS -- "it clears closed timestamp when reopening" subtest verifies same |
| `updated` refreshed on every valid transition | PASS -- `TestTransition_UpdatedTimestamp` iterates all 7 valid transitions with `time.Sleep(time.Millisecond)` to ensure time advances, verifies `Updated` is after original and within bracket | PASS -- "it updates the updated timestamp on every valid transition" iterates all 7 valid transitions, verifies `Updated` is after original or equal to `before` bracket |
| Error messages include command name and current status | PASS -- `TestTransition_InvalidTransitions` asserts exact error string `"Error: Cannot {command} task tick-a3f2b7 -- status is '{status}'"` | PASS -- "invalid transitions" asserts exact error string `"Cannot {command} task tick-a3f2b7 -- status is '{status}'"` (note: no "Error: " prefix) |
| Function returns old and new status | PASS -- `Transition` returns `*TransitionResult` with `OldStatus` and `NewStatus`; `TestTransition_ValidTransitions` verifies both fields | PASS -- `Transition` returns `TransitionResult` (value type) with same fields; "valid transitions" subtest verifies both |

## Implementation Comparison

### Approach

**V4: Separate file with slice-based transition lookup.**

V4 places transition logic in a dedicated `internal/task/transition.go` file (68 lines) with its own test file `internal/task/transition_test.go` (226 lines). The transition map uses a struct with a slice of allowed source statuses:

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

The lookup is two-step: first check if the command exists in the map, then call `statusIn()` to linear-scan the `from` slice:

```go
func Transition(t *Task, command string) (*TransitionResult, error) {
	rule, ok := validTransitions[command]
	if !ok {
		return nil, fmt.Errorf("Error: Cannot %s task %s — unknown command", command, t.ID)
	}
	if !statusIn(t.Status, rule.from) {
		return nil, fmt.Errorf("Error: Cannot %s task %s — status is '%s'", command, t.ID, t.Status)
	}
	// ...
}
```

This requires a helper function `statusIn`:

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

The function returns `*TransitionResult` (pointer type) and produces two distinct error messages: one for unknown commands (`"unknown command"`) and one for invalid status (`"status is '{s}'"`)

**V5: Inline in task.go with nested map lookup.**

V5 adds the transition logic directly into the existing `internal/task/task.go` file (46 lines added) and appends tests to `internal/task/task_test.go` (236 lines added). The transition map uses a nested map for O(1) lookup:

```go
var validTransitions = map[string]map[Status]Status{
	"start":  {StatusOpen: StatusInProgress},
	"done":   {StatusOpen: StatusDone, StatusInProgress: StatusDone},
	"cancel": {StatusOpen: StatusCancelled, StatusInProgress: StatusCancelled},
	"reopen": {StatusDone: StatusOpen, StatusCancelled: StatusOpen},
}
```

The lookup is also two-step but both are O(1) map lookups with no helper needed:

```go
func Transition(t *Task, command string) (TransitionResult, error) {
	transitions, ok := validTransitions[command]
	if !ok {
		return TransitionResult{}, fmt.Errorf("Cannot %s task %s — status is '%s'", command, t.ID, t.Status)
	}
	newStatus, ok := transitions[t.Status]
	if !ok {
		return TransitionResult{}, fmt.Errorf("Cannot %s task %s — status is '%s'", command, t.ID, t.Status)
	}
	// ...
}
```

The function returns `TransitionResult` (value type, not pointer) and produces the same error message for both unknown commands and invalid status transitions.

**Key structural differences:**

1. **File organization:** V4 creates a dedicated file pair (`transition.go` / `transition_test.go`). V5 adds to the existing `task.go` / `task_test.go`. Both are reasonable; V4's approach provides cleaner separation of concerns, while V5 keeps related domain types together.

2. **Data structure:** V4 uses `[]Status` requiring linear scan via `statusIn()`. V5 uses `map[Status]Status` for direct O(1) lookup with no helper function. V5's approach is more efficient (though with only 2-3 items, performance is irrelevant) and eliminates 9 lines of helper code.

3. **Return type:** V4 returns `*TransitionResult` (pointer). V5 returns `TransitionResult` (value). Since `TransitionResult` is a small struct (two string-typed fields), returning by value is more Go-idiomatic -- no heap allocation, no nil-check needed by callers.

4. **Error message prefix:** V4 prefixes errors with `"Error: "` (matching the spec's error format `"Error: Cannot {command} task tick-{id} — status is '{current_status}'"` verbatim). V5 omits the `"Error: "` prefix, producing `"Cannot {command} task tick-{id} — status is '{current_status}'"`.

5. **Unknown command handling:** V4 has a distinct error for unknown commands (`"unknown command"`) versus invalid status. V5 produces the same error format for both cases, which means an unknown command would still show `"status is '{current_status}'"` even though the real issue is the command itself. V4 is more diagnostically precise here.

### Code Quality

**Transition map design:**

V4's anonymous struct approach:
```go
var validTransitions = map[string]struct {
	from []Status
	to   Status
}{ ... }
```
This is readable but verbose and requires the `statusIn` helper. The struct fields `from` and `to` are self-documenting.

V5's nested map approach:
```go
var validTransitions = map[string]map[Status]Status{ ... }
```
This is more concise and eliminates the helper function entirely. The inner map directly encodes the from->to relationship. However, the triple-nested map type signature `map[string]map[Status]Status` is slightly less self-documenting than V4's named struct fields.

**Core mutation logic (identical):**

Both versions share the same mutation pattern:
```go
oldStatus := t.Status
now := time.Now().UTC().Truncate(time.Second)
t.Status = rule.to  // V4: rule.to, V5: newStatus
t.Updated = now
switch rule.to {     // V4: rule.to, V5: newStatus
case StatusDone, StatusCancelled:
    t.Closed = &now
case StatusOpen:
    t.Closed = nil
}
```

The mutation logic is functionally identical between both versions. Both correctly use `time.Now().UTC().Truncate(time.Second)` and both handle the `Closed` field via a switch statement.

**Error handling:**

V4 provides two distinct error messages:
```go
// Unknown command
return nil, fmt.Errorf("Error: Cannot %s task %s — unknown command", command, t.ID)
// Invalid status
return nil, fmt.Errorf("Error: Cannot %s task %s — status is '%s'", command, t.ID, t.Status)
```

V5 uses one unified message for both error paths:
```go
return TransitionResult{}, fmt.Errorf("Cannot %s task %s — status is '%s'", command, t.ID, t.Status)
```

V4's dual-message approach is better for debugging (callers can distinguish "bad command" from "wrong status"). However, the spec only defines one error format, and V5's approach is simpler.

**Documentation:**

V4 (transition.go):
```go
// TransitionResult holds the old and new status after a successful transition.
// Transition applies a status transition to the given task by command name.
// Valid commands: start, done, cancel, reopen.
// On success, it updates the task's status, updated timestamp, and closed timestamp
// (set on done/cancelled, cleared on reopen). Returns old and new status.
// On invalid transition, the task is not modified and an error is returned.
// statusIn checks whether s is in the given list of statuses.
```

V5 (task.go):
```go
// TransitionResult holds the old and new status after a successful transition,
// enabling output formatting like "tick-a3f2b7: open -> in_progress".
// Transition applies a status transition to the given task based on the command.
// Valid commands are "start", "done", "cancel", and "reopen". On success the
// task's status, updated, and closed fields are mutated and a TransitionResult
// is returned. On failure the task is not modified and an error is returned.
```

Both are well-documented. V5's `TransitionResult` doc includes a usage example which is slightly more helpful.

### Test Quality

#### V4 Test Functions (`transition_test.go`, 226 lines)

**Package-level helper:**
```go
func makeTask(id string, status Status) Task
```
Creates a task with a fixed ID, the given status, and pre-set `Created`/`Updated` to `2026-01-19T10:00:00Z`. Automatically sets `Closed` for done/cancelled statuses.

**`TestTransition_ValidTransitions`** (1 top-level, 7 subtests via table):
- `"it transitions open to in_progress via start"`
- `"it transitions open to done via done"`
- `"it transitions in_progress to done via done"`
- `"it transitions open to cancelled via cancel"`
- `"it transitions in_progress to cancelled via cancel"`
- `"it transitions done to open via reopen"`
- `"it transitions cancelled to open via reopen"`

Each subtest checks: no error, task.Status == expected, result.OldStatus == from, result.NewStatus == expected.

**`TestTransition_InvalidTransitions`** (1 top-level, 9 subtests via table):
- `"it rejects start on in_progress task"`
- `"it rejects start on done task"`
- `"it rejects start on cancelled task"`
- `"it rejects done on done task"`
- `"it rejects done on cancelled task"`
- `"it rejects cancel on done task"`
- `"it rejects cancel on cancelled task"`
- `"it rejects reopen on open task"`
- `"it rejects reopen on in_progress task"`

Each subtest checks: error is non-nil, exact error message matches `"Error: Cannot {command} task tick-a3f2b7 — status is '{status}'"`.

**`TestTransition_ClosedTimestamp`** (1 top-level, 3 subtests):
- `"it sets closed timestamp when transitioning to done"` -- precondition check (Closed nil), time bracket assertion
- `"it sets closed timestamp when transitioning to cancelled"` -- same pattern
- `"it clears closed timestamp when reopening"` -- precondition check (Closed not nil), asserts Closed becomes nil

**`TestTransition_UpdatedTimestamp`** (1 top-level, 1 subtest with 7 inner subtests):
- `"it updates the updated timestamp on every valid transition"` -- iterates all 7 valid transitions, uses `time.Sleep(time.Millisecond)` to ensure time advances, checks `Updated` is after original AND within time bracket

**`TestTransition_NoModificationOnError`** (1 top-level, 1 subtest):
- `"it does not modify task on invalid transition"` -- creates two identical tasks (original + task), runs invalid transition on `task`, compares all mutable fields (Status, Updated, Closed) against `original`

**V4 total: 5 top-level test functions, 21 subtests (including 7 inner subtests for UpdatedTimestamp).**

#### V5 Test Functions (added to `task_test.go`, 236 lines added)

**Local helper (inside TestTransition):**
```go
makeTask := func(status Status, closed bool) Task
```
Takes status and an explicit `closed` boolean parameter (rather than deriving it from status like V4). Uses different dates: `Created=2026-01-01`, `Updated=2026-01-02`, `Closed=2026-01-03` (when set). Sets `Priority: DefaultPriority`.

**`TestTransition`** (1 top-level, 7 subtests):

*`"valid transitions"`* (1 subtest, 7 inner subtests via table):
- `"it transitions open to in_progress via start"`
- `"it transitions open to done via done"`
- `"it transitions in_progress to done via done"`
- `"it transitions open to cancelled via cancel"`
- `"it transitions in_progress to cancelled via cancel"`
- `"it transitions done to open via reopen"`
- `"it transitions cancelled to open via reopen"`

Each subtest checks: no error, result.OldStatus, result.NewStatus, task.Status, Updated within bracket, AND Closed timestamp correctness (set or nil based on `wantClosed` column). This is more thorough per-subtest -- V4 checks Closed separately.

*`"invalid transitions"`* (1 subtest, 9 inner subtests via table):
- Same 9 cases as V4

Each subtest checks: error non-nil, error contains command name, error contains status string, AND exact error message match. V5 adds the `strings.Contains` assertions as pre-checks before the exact match, which is good defensive testing.

*`"it does not modify task on invalid transition"`* (1 subtest):
- Same pattern as V4. Checks Status, Updated, Closed.

*`"it sets closed timestamp when transitioning to done"`* (1 subtest):
- Same pattern as V4 with time bracket.

*`"it sets closed timestamp when transitioning to cancelled"`* (1 subtest):
- Same pattern as V4.

*`"it clears closed timestamp when reopening"`* (1 subtest):
- Same pattern as V4.

*`"it updates the updated timestamp on every valid transition"`* (1 subtest, 7 inner subtests):
- Same 7 transitions as V4 but WITHOUT `time.Sleep`. Uses only `!tk.Updated.After(origUpdated) && !tk.Updated.Equal(before)` check.

**V5 total: 1 top-level test function, 7 second-level subtests, 23 inner subtests.**

#### Test Coverage Diff

| Edge Case | V4 | V5 |
|-----------|-----|-----|
| 7 valid transitions (status, result) | Yes -- dedicated table | Yes -- integrated table with Closed check |
| 9 invalid transitions (error message) | Yes -- exact match | Yes -- contains + exact match |
| Closed set on done | Yes -- dedicated subtest | Yes -- dedicated subtest + table column |
| Closed set on cancel | Yes -- dedicated subtest | Yes -- dedicated subtest + table column |
| Closed cleared on reopen | Yes -- dedicated subtest | Yes -- dedicated subtest |
| Updated refreshed (all 7) | Yes -- with time.Sleep | Yes -- without time.Sleep |
| Task unmodified on error | Yes -- compares 3 fields | Yes -- compares 3 fields |
| Unknown command error | Distinct message tested implicitly (not a subtest) | Not distinctly tested |
| Error contains command name | Implicit via exact match | Explicit `strings.Contains` + exact match |
| Error contains status | Implicit via exact match | Explicit `strings.Contains` + exact match |
| Time bracket assertions | Yes | Yes |
| Precondition checks | Yes (Closed nil/non-nil before transition) | Yes (same) |

The test coverage is effectively equivalent. V5's valid transition table is slightly more thorough by integrating Closed timestamp verification into the table-driven test (via `wantClosed` column), so the Closed timestamp is checked on every valid transition, not just in dedicated subtests. V5's invalid transition tests add explicit `strings.Contains` pre-checks before the exact match, which is marginally better for debugging assertion failures. V4 uses `time.Sleep(time.Millisecond)` to guarantee time advancement for the Updated timestamp test, which is more robust than V5's approach that relies on the original timestamp being in the past.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- code is properly formatted | PASS -- code is properly formatted |
| Add context.Context to all blocking operations | N/A -- no blocking operations | N/A -- no blocking operations |
| Handle all errors explicitly (no naked returns) | PASS -- all errors checked, no ignored return values | PASS -- all errors checked, no ignored return values |
| Write table-driven tests with subtests | PASS -- `ValidTransitions` and `InvalidTransitions` use table-driven pattern with `t.Run` | PASS -- "valid transitions" and "invalid transitions" use table-driven pattern with `t.Run` |
| Document all exported functions, types, and packages | PASS -- `TransitionResult`, `Transition` documented; unexported `statusIn` also documented | PASS -- `TransitionResult`, `Transition` documented |
| Propagate errors with fmt.Errorf("%w", err) | N/A -- this task creates new errors, does not wrap existing ones | N/A -- same |
| No panic for normal error handling | PASS -- no panics | PASS -- no panics |
| No hardcoded configuration | PASS -- transition rules are in a declarative map, not inline conditionals | PASS -- same |
| Avoid _ assignment without justification | PASS -- no ignored errors | PASS -- no ignored errors |

### Spec-vs-Convention Conflicts

**1. Error message prefix: "Error: " in format string.**

- **Spec says:** `Error: Cannot {command} task tick-{id} — status is '{current_status}'`
- **Go convention:** Error strings should not be capitalized or start with punctuation. The `"Error: "` prefix is redundant -- the caller already knows it's an error because the function returned a non-nil `error` value. The Go proverb is "don't decorate errors with context that's already available."
- **V4 chose:** Verbatim spec compliance. Error messages start with `"Error: Cannot ..."`.
- **V5 chose:** Dropped the `"Error: "` prefix. Messages start with `"Cannot ..."`.
- **Assessment:** This is a genuine spec-vs-convention conflict. The `"Error: "` prefix is unusual in Go -- it duplicates the type information. V5's choice to drop it follows Go convention. However, if this error message is displayed directly to CLI users (which it likely is, given the format includes the task ID), the spec format may be intentional for user-facing output. Both are reasonable judgment calls. V5's approach is more Go-idiomatic; V4's approach is more spec-faithful.

**2. Return type: pointer vs value for TransitionResult.**

- **Spec says:** "Return old and new status for output formatting" -- no guidance on pointer vs value.
- **Go convention:** Small structs (two fields) should be returned by value, not pointer. Pointer returns are appropriate for large structs or when nil is a meaningful return value.
- **V4 chose:** `*TransitionResult` (pointer). Returns `nil` on error.
- **V5 chose:** `TransitionResult` (value). Returns zero-value on error.
- **Assessment:** V5 follows Go convention more closely. `TransitionResult` has two `Status` (string) fields -- it's small enough that value semantics are preferable. V4's pointer return forces callers to nil-check. However, V4's `nil` return on error is a common Go pattern (return nil, err) and not incorrect.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 4 (transition.go, transition_test.go, 2 docs) | 6 (task.go, task_test.go, .gitignore, tick binary removed, 2 docs) |
| Lines added (code only) | 294 (68 impl + 226 test) | 282 (46 impl + 236 test) |
| Impl LOC | 68 | 46 |
| Test LOC | 226 | 236 |
| Top-level test functions | 5 | 1 |
| Total subtests (all levels) | 21 | 23 |

## Verdict

**Effectively a tie, with minor advantages to each version in different areas.**

Both implementations are correct, well-tested, and pass all acceptance criteria. The differences are stylistic and organizational rather than substantive:

**V5 advantages:**
1. **More efficient data structure:** The nested `map[Status]Status` eliminates the `statusIn` helper and provides O(1) lookup (though irrelevant at this scale). The result is 22 fewer implementation lines (46 vs 68).
2. **Value-type return:** Returning `TransitionResult` by value rather than pointer is more Go-idiomatic for a small two-field struct.
3. **Integrated Closed testing:** The valid transitions table includes a `wantClosed` column, so Closed timestamp correctness is verified on every valid transition (7 checks) in addition to the dedicated subtests.
4. **Error assertion defense:** Invalid transition tests add `strings.Contains` pre-checks before exact match, producing clearer failure messages.

**V4 advantages:**
1. **File separation:** Dedicating `transition.go` and `transition_test.go` provides cleaner separation of concerns. As the codebase grows, this keeps files smaller and easier to navigate.
2. **Spec-faithful error format:** Including `"Error: "` prefix matches the spec verbatim. While Go convention suggests omitting it, the spec is explicit.
3. **Distinct unknown-command error:** V4 differentiates between "unknown command" and "invalid status" errors, providing better diagnostics. V5 produces the same message for both, which could be confusing if an unknown command is passed.
4. **Robust time testing:** Using `time.Sleep(time.Millisecond)` in the Updated timestamp test guarantees the clock has advanced, making the assertion more reliable than V5's approach which depends on the fixture timestamp being sufficiently in the past.

Neither version has a clear overall advantage. V5 is marginally more Go-idiomatic in its data structure and return type. V4 is marginally more thorough in error differentiation and test robustness. Both fully satisfy the acceptance criteria and skill constraints.
