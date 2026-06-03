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

  Discussion Map — Ready Includes In-Progress (5 subtopics · 5 pending)

  ┌─ ○ Ready semantics: does in-progress belong? [pending]
  ├─ ○ Blocked consistency under the new definition [pending]
  ├─ ○ Presentation of in-progress in ready output [pending]
  ├─ ○ Sort ordering (resume-first vs priority) [pending]
  └─ ○ Edge cases & scope (filters, stats, --count) [pending]

---

*Subtopics are documented below as they reach `decided` or accumulate enough exploration to capture.*

---

## Summary

### Key Insights
*(to be captured)*

### Open Threads
*(to be captured)*

### Current State
- Discussion just initialized; no subtopics decided yet.
