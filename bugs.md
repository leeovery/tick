# Bugs

## BUG-2: Rule 6 parent reopen records `auto=false` in transition history

**Severity:** Medium
**Affects:** `internal/cli/helpers.go:validateAndReopenParent`, transition history accuracy

**Description:** When a child is added to a done parent (via `create --parent` or `update --parent`), Rule 6 automatically reopens the parent by calling `ApplyWithCascades(tasks, &parent, "reopen")`. Since the parent is the primary target of `ApplyWithCascades`, its transition is recorded with `auto=false`. However, this reopen is system-initiated (triggered by adding a child), not a manual `tick reopen` command — it should be `auto=true`.

Further cascades from this reopen (e.g. Rule 5 reopening a done grandparent) are correctly recorded as `auto=true`.

**Example:**

Setup: Epic (done) → Story (done)

Action: `tick create "New task" --parent <Story-ID>`

Resulting transition history:
```
Story:  done → open  auto=false   ← BUG: should be auto=true
Epic:   done → open  auto=true    ← correct
```

The user ran `create`, not `reopen`. The Story reopen was entirely system-initiated.

**Root cause:** `ApplyWithCascades` unconditionally sets `Auto: false` on the primary target (line 43 of `apply_cascades.go`). This is correct when called from `RunTransition` (user explicitly ran a transition command), but incorrect when called from `validateAndReopenParent` where the primary target is also a system-initiated cascade.

**Possible fix:** Either add an `auto` parameter to `ApplyWithCascades` so the caller can specify whether the primary transition is manual or automatic, or have `validateAndReopenParent` patch the transition record after the call.

---

## BUG-3: Cascade output shows unchanged tasks

**Severity:** Low
**Affects:** `internal/cli/transition.go:buildCascadeResult`, all formatters (toon, pretty, JSON)

**Description:** When a status transition triggers cascades, the output includes lines for sibling/descendant tasks that were already terminal and didn't change. These "(unchanged)" lines are noise — if nothing changed, there's nothing to report.

**Example:**

```
$ tick done tick-b15fda
tick-b15fda: in_progress → done
tick-c5a1ff: in_progress → done (auto)
tick-18747f: in_progress → done (auto)
tick-fd039e: done (unchanged)     ← noise
tick-c3e72b: done (unchanged)     ← noise
tick-3d9a7e: done (unchanged)     ← noise
```

**Expected behavior:** Only the primary transition and actual cascaded changes should be displayed. Tasks that didn't change should be omitted.

**Root cause:** `buildCascadeResult` in `transition.go` (lines 117-135) actively collects terminal descendants of involved tasks that weren't cascaded and populates them into `CascadeResult.Unchanged`. All three formatters then render these entries.
