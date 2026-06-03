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

  Discussion Map — Ready Includes In-Progress (5 subtopics — 1 decided · 1 exploring · 3 pending)

  ┌─ ✓ Ready semantics: does in-progress belong? [decided]
  ├─ ◐ Blocked consistency under the new definition [exploring]
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

## Summary

### Key Insights
*(to be captured)*

### Open Threads
*(to be captured)*

### Current State
- **Decided:** `ready` includes `in_progress` (single-actor model; multi-actor handled later via an assignee field + "ready excludes tasks assigned to others").
- **Exploring:** how `blocked` reconciles with the new `ready` definition.
