# Bugs

## BUG-1: Unknown flags silently ignored across all commands

**Severity:** Low
**Affects:** All commands (not specific to any single command)

**Description:** CLI commands silently discard unrecognised flags instead of erroring. Any argument starting with `-` that isn't a known flag is stripped without warning, which can lead to confusing behavior when a user mistakenly passes a flag that doesn't apply to the command they're running.

**Example:** `tick dep add tick-aaa --blocks tick-bbb` silently ignores `--blocks` (which is only valid on `create`/`update`) and treats it as `tick dep add tick-aaa tick-bbb`, producing a result the user didn't intend — with no indication anything was wrong.

**Expected behavior:** All commands should reject unrecognised flags with a clear error, e.g. `Error: unknown flag --blocks. Run 'tick help dep' for usage.`

**Scope:** This is a general CLI parsing concern. Flag parsing across all commands should be audited to ensure unknown flags produce errors rather than being silently discarded.

---

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
