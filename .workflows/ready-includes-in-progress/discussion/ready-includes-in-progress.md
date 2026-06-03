# Discussion: Ready Includes In-Progress

## Context

`tick ready` surfaces the next actionable tasks to answer "what should I be doing right now?". Today it returns only tasks with `status = 'open'` that are unblocked — it skips tasks already in `in_progress`. The result: you start a task, get interrupted, and when you run `tick ready` to resume it points you at *new* open work instead of the started-but-dangling task. The interrupted task becomes invisible to the very command meant to orient you.

The intuition from the seed: "ready" should mean "work available and unblocked to act on right now," and a task you've already started is arguably the *most* ready thing there is. Including in-progress items would close the loop so resuming interrupted work is the natural default.

Two things must be settled before spec/code:
- **Semantics** — should `ready` include `in_progress` at all, and how does that reconcile with `ready` also serving the "what new work can I pull?" question?
- **Presentation** — should in-progress items appear inline, be sorted to the top, or be visually distinguished as resumptions vs fresh starts?

A hard constraint: `blocked` is currently defined as the De Morgan inverse of `ready`'s `NOT EXISTS` conditions (`query_helpers.go`), so any change to "ready" must keep "blocked" consistent.

### Current State (code)

- `ReadyConditions()` = `t.status = 'open'` AND no unclosed blockers AND no open/in-progress children AND no dependency-blocked ancestor.
- `BlockedConditions()` = `t.status = 'open'` AND (has unclosed blocker OR has open/in-progress child OR has blocked ancestor) — the inverse `EXISTS` set, sharing the `status = 'open'` gate.
- **Key wrinkle:** both `ready` and `blocked` require `status = 'open'`. An `in_progress` task is therefore currently *neither* ready nor blocked.
- Sort order: `ORDER BY t.priority ASC, t.created ASC` (shared by all list-family queries).
- `ReadyWhereClause()` also feeds the `stats` ready count.

### References

- [Seed: tick ready should include in-progress work](../seeds/2026-06-02-ready-includes-in-progress.md)
- [Discovery session 001](../discovery/session-001.md)
- `internal/cli/query_helpers.go` — `ReadyConditions()` / `BlockedConditions()`
- `internal/cli/list.go:262` — `buildListQuery` (sort, filters)
- `internal/cli/stats.go:79` — ready count consumer

## Discussion Map

### States

- **pending** (`○`) — identified but not yet explored
- **exploring** (`◐`) — actively being discussed
- **converging** (`→`) — narrowing toward a decision
- **decided** (`✓`) — decision reached with rationale documented

### Map

  Discussion Map — Ready Includes In-Progress (5 subtopics — 2 decided · 3 pending)

  ┌─ ✓ Ready semantics: does in-progress belong? [decided]
  ├─ ✓ Blocked consistency under the new definition [decided]
  ├─ ○ Presentation of in-progress in ready output [pending]
  ├─ ○ Sort ordering (resume-first vs priority) [pending]
  └─ ○ Edge cases & scope (filters, stats, --count) [pending]

---

*Subtopics are documented below as they reach `decided` or accumulate enough exploration to capture.*

---

## Ready Semantics: Does In-Progress Belong?

### Context

`tick ready` answers "what should I act on right now?". Today both `ready` and `blocked` gate on `status = 'open'`, so an `in_progress` task is in *neither* view — invisible to the very command you'd use to resume interrupted work. That invisibility is the gap. The question is whether `in_progress` belongs in `ready` at all, and the answer turns out to hinge on tick's actor model.

### Options Considered

**Option A — Single-actor / resumption model.** `in_progress` = "the task *I* started and got pulled off." Surfacing it in `ready` means "resume this." `ready` becomes "everything actionable right now" = unblocked `open` **+** `in_progress`.
- Pros: closes the resumption gap directly; matches the everyday use of `ready` ("where was I / what's next").
- Cons: dilutes the secondary "what *new* work can I pull?" reading of `ready` (started work mixes with fresh-start candidates).

**Option B — Multi-actor / claim model.** `in_progress` = "someone already took it, hands off." Then it's *not* ready for me, and showing it risks two actors colliding on one task. `in_progress` would be *excluded* from `ready` (a soft lock) — stronger than today.
- Pros: prevents collision in a concurrent team setting.
- Cons: tick has **no assignee field** — `in_progress` cannot distinguish *my* interrupted work from *someone else's* claimed work. So exclusion also kills my own resumptions; it's a crude proxy for ownership.

### Journey

Initial lean was Option A (the seed's framing). The user raised the sharp counter: if a task is "taken by another developer," it's being handled — not ready — so it shouldn't appear. That's a genuinely different worldview, and it gives the *opposite* answer.

The resolver: tick has no ownership concept. Task carries Title/Status/Priority/Parent/Deps/Type/Tags/Refs/Notes/Transitions — nothing about *who*. The whole design reads single-actor: cascades assume one will (starting one task drives the whole ancestor chain to `in_progress`), and there's no claim/lease/lock-by-owner machinery. The multi-actor collision problem is real, but its correct fix is an **assignee + claim** mechanism, not "exclude all `in_progress`." Excluding `in_progress` to dodge collisions can't tell *my* work from *theirs*, so it breaks the single-actor case to half-serve a model tick doesn't yet implement.

The "aha": the multi-actor concern doesn't argue *against* this feature — it points at a *future* feature. Once an assignee field exists, the right rule is **"`ready` excludes tasks assigned to others"** — precise, collision-safe, and it preserves resumption of your own work. That reframing satisfied the user's concern without compromising the decision here.

### Decision

**Include `in_progress` in `ready`.** Decide for the tool tick is today: single-actor, no ownership → `ready` = "everything actionable now" = unblocked `open` + `in_progress`.

- **Deciding factor:** no assignee field exists, so the multi-actor exclusion argument can't be implemented correctly anyway; the single-actor resumption case is the real, present need.
- **Trade-off accepted:** the "pull only new work" reading of `ready` is diluted; anyone wanting strictly unstarted work can use `tick list --status open`.
- **Future path (noted, out of scope):** if/when multi-actor claiming is pursued, add an assignee field and make `ready` exclude tasks assigned to *others* — revisit then, but do **not** solve it now by excluding all `in_progress`.
- **Confidence:** high.

---

## Blocked Consistency Under The New Definition

### Context

`blocked` is the De Morgan inverse of `ready` in the code: `BlockedConditions()` negates each `ReadyNo*()` helper and ORs them, sharing the *same* `t.status = 'open'` literal as the gate. Today that means `ready` and `blocked` partition the `open` set, and `in_progress`/`done`/`cancelled` are in neither. Now that `ready` admits `in_progress`, we must decide what `blocked`'s status gate does — and whether the partition survives. (This is the discussion's stated hard constraint, and the load-bearing gap flagged by the first review's F1/F7.)

### Options Considered

**Option A — Symmetric.** `blocked` gates on `(open OR in_progress)`, identical to `ready`. The two stay strict De Morgan complements, now over the *live* (non-terminal) set instead of just `open`.
- Pros: closes the invisibility hole completely (a started-but-stuck task lands in `blocked` instead of vanishing); preserves the clean inverse; minimal code change (flip one shared literal).
- Cons: an `in_progress` task can be labeled "blocked" — semantically loose for a started task, but defensible (it genuinely can't proceed).

**Option B — Asymmetric.** `ready` admits `in_progress`; `blocked` stays `open`-only.
- Cons: breaks the complement (no longer a partition; code needs special-casing) and **reopens the exact hole** — an `in_progress` task with an open child or an unclosed blocker would be in *neither* view. Common case (start a parent, then add subtasks) goes invisible.

**Option C — "blocked = can't even start."** Once started, a task is past "blocked," so `in_progress` is never blocked. Same invisibility problem as B for started-but-stuck work.

### Journey

The user chose A immediately and articulated the governing semantic crisply: a task that's blocked *stays* blocked even if you force-start it. You can always look a task up and start it directly (starting isn't gated by blockers) — but ignoring the fact that `ready` never offered it doesn't change its nature. It was blocked; now it's blocked-and-in-progress; it still reports as blocked.

The user's verification question — "a blocked-in-progress task won't *also* show as ready, will it?" — is the crux of why A is clean. Answer: **no, never.** Because A keeps `ready` and `blocked` sharing one identical status gate, `blocked` is the exact logical negation of `ready`'s three `NOT EXISTS` conditions. Over the live set, every task is *exactly one* of ready/blocked — mutual exclusivity and exhaustiveness both hold. The blocked-in-progress task fails the ready test and surfaces only in `tick blocked`.

### Decision

**Option A — keep `ready` and `blocked` as strict complements.** Both gate on `status IN ('open','in_progress')`; `blocked` remains the De Morgan inverse.

- **Invariant:** `ready ⊎ blocked = all live tasks` (open + in_progress). Never both, never neither, for any live task. `done`/`cancelled` are in neither.
- **Force-started blocked task:** shows in `tick blocked` only, never `tick ready`. Starting a task does not clear its block.
- **Implementation shape (captured, not a plan):** the shared `t.status = 'open'` literal in *both* `ReadyConditions()` and `BlockedConditions()` becomes `t.status IN ('open','in_progress')`. The `negateNotExists` / inverse machinery is untouched — the symmetry is exactly why this is a one-line-per-side change. The fact that the gate is a single shared string is evidence A is the option the code "wants."
- **To confirm in edge-cases subtopic:** that the state machine genuinely allows `open → in_progress` regardless of unclosed blockers (assumed true — blockers are a query concept, not a transition guard).
- **Confidence:** high.

This resolves review F1 (De Morgan reconciliation) and F7 (in-progress + unclosed blocker → blocked).

---

## Summary

### Key Insights
*(to be captured)*

### Open Threads
*(to be captured)*

### Current State
- **Decided:** `ready` includes `in_progress` (single-actor model; multi-actor handled later via an assignee field + "ready excludes tasks assigned to others").
- **Decided:** `blocked` stays the strict De Morgan complement — both gate on `(open OR in_progress)`; `ready ⊎ blocked = all live tasks`; a task is never in both.
- **Pending:** presentation, sort ordering, and edge cases/scope (filters, stats count, `--count`).
