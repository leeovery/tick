---
topic: blocked-ancestor-ready
status: planning
format: tick
specification: ../specification/blocked-ancestor-ready/specification.md
spec_commit: e5c5d2995477fa95787af0a959e7162d1bb39cc3
created: 2026-02-20
updated: 2026-02-20
external_dependencies: []
task_list_gate_mode: gated
author_gate_mode: gated
finding_gate_mode: gated
planning:
  phase: 1
  task: ~
---

# Plan: Blocked Ancestor Ready

### Phase 1: Blocked Ancestor Ready Check
status: draft
ext_id:

**Goal**: Tasks with dependency-blocked ancestors are excluded from ready results and included in blocked results across all code paths (list --ready, list --blocked, tick ready, stats).

**Why this order**: Single-phase feature. The entire scope is a self-contained change to `query_helpers.go` and its integration tests. No meaningful intermediate checkpoint exists — the walking skeleton IS the complete feature because there is no partial ancestor check that delivers independent value.

**Acceptance**:
- [ ] `ReadyNoBlockedAncestor()` helper returns a NOT EXISTS subquery with recursive CTE that walks the full ancestor chain checking for unclosed dependency blockers
- [ ] `ReadyConditions()` includes the ancestor check as the 4th condition
- [ ] `BlockedConditions()` includes the EXISTS inverse of the ancestor check in its OR clause
- [ ] Child of a dependency-blocked parent does not appear in ready results and does appear in blocked results
- [ ] Grandchild of a dependency-blocked grandparent (with clean intermediate parent) does not appear in ready results
- [ ] Descendant behind an intermediate grouping task (no own blockers) under a blocked ancestor is correctly excluded from ready
- [ ] When the ancestor's blocker is resolved (done/cancelled), descendants become ready again
- [ ] Root tasks with no parent remain unaffected by the ancestor check
- [ ] Stats ready count matches `list --ready` output for mixed scenarios with blocked ancestors
