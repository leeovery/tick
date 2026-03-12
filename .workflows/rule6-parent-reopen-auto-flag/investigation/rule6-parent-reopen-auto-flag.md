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

*Pending code analysis*

### Root Cause

*Pending*

### Contributing Factors

*Pending*

### Why It Wasn't Caught

*Pending*

### Blast Radius

*Pending*

---

## Fix Direction

*Pending findings review*

---

## Notes

*None yet*
