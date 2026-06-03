# Specification: Ready Includes In-Progress

## Specification

## Overview

### Problem

`tick ready` answers "what should I act on right now?" Today both `ready` and `blocked` gate on `status = 'open'`. An `in_progress` task is therefore in *neither* view — invisible to the very command used to resume interrupted work. Start a task, get interrupted, run `tick ready`, and it points you at *new* open work instead of the started-but-dangling task. The same `status = 'open'` gate also drops a started-but-stuck task out of `blocked`. The real defect is **double invisibility**: in-progress work falls off both actionability views.

### Goal

`ready` means "everything actionable right now" = unblocked `open` **+** `in_progress`. A started task is the most ready thing there is; resuming interrupted work becomes the natural default. `blocked` stays the strict complement so no live task ever falls through the cracks.

### Actor model (decided premise)

tick is single-actor with no ownership concept — a task carries no assignee. `in_progress` means "the task *I* started and got pulled off," not "someone else claimed it." Surfacing it in `ready` means "resume this." The "someone else took it, hands off" reading would require an assignee field tick doesn't have; excluding all `in_progress` to dodge collisions can't tell *my* work from *theirs*, so it would break the single-actor resumption case. Multi-actor claiming is a separate future feature (see Out of Scope).

### Governing invariant

`ready ⊎ blocked = all live tasks`, where live = `status IN ('open','in_progress')`. Every live task is in **exactly one** of `ready`/`blocked` — never both, never neither. `done` and `cancelled` are in neither. `ready` and `blocked` share one identical status gate; `blocked` is the literal De Morgan inverse of `ready`'s `NOT EXISTS` conditions. This invariant is what makes the change small (flip one shared literal per side) and guarantees nothing is dropped.

---

## Ready & Blocked Definitions

### Ready

A task is **ready** when **all** of the following hold:

1. `status IN ('open', 'in_progress')` — the changed gate (was `status = 'open'`).
2. No unclosed blocker — no incomplete task in its blocked-by dependencies.
3. No open or in-progress child — the **leaf gate**: any child with `status IN ('open','in_progress')` disqualifies it.
4. No dependency-blocked ancestor — walking the parent chain, no ancestor is itself blocked by an unclosed dependency.

Conditions 2–4 are the existing `NOT EXISTS` conditions, unchanged. Only the status gate widens.

### Blocked

A task is **blocked** when it is live (`status IN ('open','in_progress')`) **and** at least one of the ready `NOT EXISTS` conditions fails:

- has an unclosed blocker, **OR**
- has an open or in-progress child, **OR**
- has a dependency-blocked ancestor.

`blocked` is the literal De Morgan inverse of `ready` over the same status gate. The inverse/negation machinery is untouched — only the shared status literal changes on each side.

### Consequences that fall out of the definitions

- **Force-started blocked task.** Starting is not gated by blockers (the `start` transition constrains only `from: open`; there is no blocker check in the state machine). So an `open` task with unclosed blockers can be force-started into `in_progress`, becoming blocked-and-in-progress. It shows in `tick blocked` only, **never** `tick ready`. Starting a task does not clear its block — a task that was blocked stays blocked.
- **Leaf gate is symmetric under in-progress.** An `in_progress` parent that exists only because the start-cascade drove it up the ancestor chain does **not** surface in `ready` — it has an `in_progress` child, so the leaf gate excludes it; it surfaces in `blocked` instead. Only the leaf surfaces. Parent and child **never co-occur in `ready`**, regardless of whether rows are `open` or `in_progress`.
- **`done`/`cancelled`** remain in neither view.

---

## Sort Ordering — Resume-First, Ready-View-Only

`in_progress` tasks float to the top of `ready` as a band; within the band — and within the `open` tasks beneath it — the existing `priority ASC, created ASC` ordering holds. This makes resumption the default: `--count 1` naturally returns in-flight work first.

### Scope: the ready *filter* (`f.Ready`), not a literal command

The float applies whenever the ready filter is active — set identically by `tick ready` (a literal alias dispatching as `--ready` into the same `RunList`) and `tick list --ready`. Both are the same ready view, so both float. Plain `tick list` and `tick list --blocked` never set the ready filter and keep the current neutral `priority ASC, created ASC` ordering, unchanged. "Ready-only" means *ready-view-only*, keyed on `f.Ready`.

Rationale: resume-first is a property of `ready`'s "what now?" intent. `tick list` is a neutral browse view where silently floating started work would be surprising; `tick blocked` gains nothing from it.

### Precise promise: "top **unblocked** in-flight work"

Resume-first applies only to *actionable* (unblocked) in-flight tasks. An `in_progress` task that is itself blocked — by a direct blocker or by a blocked ancestor — is not in `ready` at all; by the partition invariant it lives in `tick blocked`. So `--count 1` returns the top **unblocked** in-flight task; force-starting a blocked task does not float it into `ready`.

### Within-band tiebreak

`priority ASC, created ASC` — the existing clause, applied within the in-progress band and within the open tasks beneath it. A "most-recently-started first" variant via transition history is deliberately not adopted (gold-plating — see Out of Scope).

### Accepted consequence

A user with several `in_progress` tasks sees a resumption-heavy `ready`. This is treated as desirable (nudges finishing work-in-progress before pulling new work), not a defect.

---

## Working Notes
