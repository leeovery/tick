# Investigation: Rule 6 Parent Reopen Auto Flag

## Symptoms

### Problem Description

**Expected behavior:**
When a child is added to a done parent (via `create --parent` or `update --parent`), Rule 6 automatically reopens the parent. This transition should be recorded with `auto=true` since it is system-initiated, not a manual user command.

**Actual behavior:**
The parent reopen transition is recorded with `auto=false`, making it appear as if the user manually ran `tick reopen`. Further cascades from this reopen (e.g. Rule 5 reopening a done grandparent) are correctly recorded as `auto=true`.

### Manifestation

- Transition history shows incorrect `auto=false` on system-initiated parent reopens
- Misleading audit trail — appears user ran `reopen` when they ran `create` or `update`
- Only affects Rule 6 (add-child-to-done-parent) path; all other cascade transitions are correct

### Reproduction Steps

1. Create a parent-child hierarchy where both are done: Epic (done) → Story (done)
2. Run `tick create "New task" --parent <Story-ID>`
3. Observe transition history: Story shows `done → open auto=false` (should be `auto=true`)

**Reproducibility:** Always

### Impact

- **Severity:** Medium
- **Scope:** Any user adding children to done tasks
- **Business impact:** Inaccurate transition history, affects reporting/audit accuracy

### References

- Bug tracked in `bugs.md` as BUG-2
- Affected code: `internal/cli/helpers.go:validateAndReopenParent`
- Root area: `internal/task/apply_cascades.go` line 43

---

## Analysis

### Initial Hypotheses

`ApplyWithCascades` unconditionally sets `Auto: false` on the primary target. This is correct when called from `RunTransition` (user explicitly ran a transition command), but incorrect when called from `validateAndReopenParent` where the primary target is also a system-initiated cascade.

### Code Trace

**Entry point:** `internal/cli/helpers.go:114` — `validateAndReopenParent()`

**Execution path:**
1. `helpers.go:114-133` — `validateAndReopenParent` finds parent in tasks, validates Rule 7 (no cancelled parent), checks if parent is done
2. `helpers.go:124` — If parent is done, calls `sm.ApplyWithCascades(tasks, &tasks[i], "reopen")`
3. `apply_cascades.go:33` — `ApplyWithCascades` applies the primary transition via `sm.Transition(target, action)`
4. `apply_cascades.go:39-44` — **BUG HERE**: Records `TransitionRecord{Auto: false}` on the primary target unconditionally
5. `apply_cascades.go:63-95` — Processes cascade queue; all cascaded tasks correctly get `Auto: true` (line 85)

**Callers of `ApplyWithCascades` (3 total):**

| Caller | File | Auto should be | Currently |
|--------|------|---------------|-----------|
| `RunTransition` | `transition.go:37` | `false` (user-initiated) | `false` ✓ |
| `validateAndReopenParent` | `helpers.go:124` | `true` (system-initiated Rule 6) | `false` ✗ |
| `evaluateRule3` | `update.go:151` | `true` (system-initiated Rule 3) | `false` ✗ |

**Second affected caller:** `evaluateRule3` in `update.go:134-165` also calls `ApplyWithCascades` for system-initiated auto-completion when a child is reparented away and remaining children are all terminal. This has the same bug — the parent's done/cancel transition is recorded as `auto=false`.

**Key files involved:**
- `internal/task/apply_cascades.go` — contains the hardcoded `Auto: false` on line 43
- `internal/cli/helpers.go` — `validateAndReopenParent` (Rule 6 caller)
- `internal/cli/update.go` — `evaluateRule3` (Rule 3 reparent caller)
- `internal/cli/transition.go` — `RunTransition` (correct manual caller)
- `internal/task/transition_history.go` — `TransitionRecord` struct definition

### Root Cause

`ApplyWithCascades` hardcodes `Auto: false` on the primary target's `TransitionRecord` (line 43 of `apply_cascades.go`). This assumes the primary target is always a user-initiated transition. However, two callers use `ApplyWithCascades` for system-initiated transitions where the primary target should also be `Auto: true`:

1. **`validateAndReopenParent`** (Rule 6) — reopens a done parent when a child is added
2. **`evaluateRule3`** (Rule 3 via reparent) — auto-completes a parent when all remaining children become terminal after reparenting

### Contributing Factors

- `ApplyWithCascades` was designed with only the manual transition use case in mind (1 caller: `RunTransition`)
- Rule 6 and Rule 3 reparent callers were added later, reusing `ApplyWithCascades` without accounting for the hardcoded `Auto: false`
- The function's doc comment explicitly states "The primary task receives a TransitionRecord with Auto: false" — the behavior is intentional but the contract is too rigid

### Why It Wasn't Caught

- Tests for `ApplyWithCascades` verify the primary always gets `Auto: false` (test at line 67, 370 of `apply_cascades_test.go`) — this was the expected behavior at design time
- No integration-level tests verify that Rule 6 parent reopen produces `auto=true` in the transition history
- No integration-level tests verify that Rule 3 reparent auto-completion produces `auto=true`

### Blast Radius

**Directly affected:**
- Rule 6 parent reopen (via `create --parent` or `update --parent` on a done parent)
- Rule 3 reparent auto-completion (via `update --parent` moving a child away, leaving all remaining children terminal)

**Not affected:**
- Manual transitions via `RunTransition` (start, done, cancel, reopen) — correctly `auto=false`
- All cascade transitions (children of `ApplyWithCascades`) — correctly `auto=true`
- Rule 3 completion triggered by normal cascade (e.g. completing the last open child) — this goes through `Cascades()` not `ApplyWithCascades`, so it's correctly `auto=true`

---

## Fix Direction

*Pending findings review*

---

## Notes

- The bug report only mentions Rule 6 (`validateAndReopenParent`), but `evaluateRule3` in `update.go` has the identical issue
- Fix must preserve `Auto: false` for `RunTransition` (the only manual caller)
