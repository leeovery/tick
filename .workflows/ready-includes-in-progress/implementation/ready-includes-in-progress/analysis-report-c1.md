---
topic: ready-includes-in-progress
cycle: 1
total_findings: 1
deduplicated_findings: 1
proposed_tasks: 0
---
# Analysis Report: ready-includes-in-progress (Cycle 1)

## Summary
Across all three analysis agents there was exactly one finding (medium severity, from the duplication agent): the live-status gate literal `t.status IN ('open', 'in_progress')` is hand-copied in both `ReadyConditions()` and `BlockedConditions()` in `query_helpers.go`. The standards and architecture agents reported clean; the architecture agent explicitly considered this same duplication and declined to flag it as the spec's deliberate "flip one shared literal per side" design and a pre-existing package convention, with drift already caught by existing equality assertions. After weighing both perspectives the single medium finding is discarded: it does not reach the actionable threshold, so no tasks are proposed and the cycle is clean.

## Discarded Findings
- Status-gate literal duplicated across ReadyConditions and BlockedConditions (duplication, medium) — Lone medium-severity finding (no high-severity mandate), does not cluster into a pattern. The spec frames the change as "one-line-per-side" (two edit sites are the intended shape, not a single-source mandate), and the duplication matches the package's pre-existing convention of inline SQL-fragment literals (the helpers never interpolate `task.Status*` constants). The drift risk it raises is already structurally guarded by three equality assertions in `query_helpers_test.go` (lines 41, 60, 115) plus the partition and stats-consistency integration tests. The architecture agent independently considered and rejected flagging it for these same reasons. Promoting it would re-litigate an accepted spec decision and an established pattern over an already-test-covered risk.
