# Implementation Review: Ready Includes In-Progress

**Plan**: ready-includes-in-progress
**QA Verdict**: Approve

## Summary

A tight, well-scoped feature, implemented exactly as specified. The "double invisibility" defect — `in_progress` tasks falling out of both `ready` and `blocked` — is fixed by widening one shared SQL literal per side, with the De Morgan inverse machinery left untouched. Resume-first ordering floats unblocked in-progress work to the top of the ready view (keyed solely on `f.Ready`, so `tick ready` and `tick list --ready` behave identically and plain `list`/`list --blocked` are unchanged), and the `stats` blocked count is corrected to the live-set derivation `(Open + InProgress) − Ready`. All three plan tasks are Complete with zero blocking issues. All ten spec acceptance criteria were verified both by static review and by an end-to-end CLI smoke test against a fresh build. The full suite passes (including `-race`), `go vet` and `gofmt` are clean.

## QA Verification

### Specification Compliance

Implementation aligns with the specification on every point:

- **Status gate** (`query_helpers.go`): `t.status = 'open'` → `t.status IN ('open','in_progress')` in both `ReadyConditions()` and `BlockedConditions()`; `negateNotExists` and the three `ReadyNo*()` helpers untouched. The governing invariant `ready ⊎ blocked = all live tasks` holds.
- **Resume-first ordering** (`list.go` `buildListQuery`): conditional `ORDER BY (t.status = 'in_progress') DESC, t.priority ASC, t.created ASC` when `f.Ready`, neutral clause otherwise; `LIMIT` applied after, no narrowing-filter guard.
- **Stats derivation** (`stats.go`): `Blocked = (Open + InProgress) − Ready`; both inline comments refreshed; no `BlockedWhereClause()` helper added (the deliberate "do NOT add" constraint honored).

No deviations.

### Plan Completion

- [x] Phase 1 acceptance criteria met (all 10 spec ACs verified)
- [x] All tasks completed (1-1, 1-2, 1-3 — all Complete)
- [x] No scope creep — the only file beyond the three named production files + their tests is `workflow_integration_test.go`, a pre-existing `tick-core` integration test whose old-semantics assertion (`child A should NOT be ready`) was correctly flipped to the new behavior. A required consequence of the gate change, not new scope.

### Code Quality

No issues found. Changes are localized and idiomatic; the SQL literal duplicated identically on both ready/blocked sides is intentional (the symmetry is load-bearing for the De Morgan derivation). The `'in_progress'` band literal is a hardcoded constant — no injection surface. `t.created ASC` is a sound chronological sort (RFC3339 UTC TEXT). Follows project conventions and the golang-pro project skill.

### Test Quality

Tests adequately verify requirements, with good discrimination:

- The resume-first test gives the in_progress row a **worse** priority (3) than the open row (0), proving the band term wins independently of priority — it would fail if the band were absent.
- The blocked-side rewrite uses spec option (b) (genuine unclosed blocker) rather than a bare-absence assertion that would pass for the wrong reason.
- The stats negative-guard test constructs `Ready > Open` and asserts `blocked >= 0`, regression-proofing the old `Open − Ready`.
- No over-testing observed; kept tests (`excludes parent with in_progress children`, neutral-ordering proofs, InProgress=0 formatting tests) remain valid.

Orchestrator-run verification: `go test ./...` all green; feature tests green under `-race`; `go vet ./...` and `gofmt -l ./internal ./cmd` clean. End-to-end CLI smoke confirmed ACs #1, #3, #4, #5, #6, #7, #8, #9 at runtime.

### Required Changes (if any)

None.

## Recommendations

### Quick-fixes

1. **(1-1)** Two `query_helpers_test.go` subtest names still read "...returns status open..." / "...beyond status check" — the names now slightly under-describe the widened `IN ('open','in_progress')` gate. Cosmetic; assertions are already correct.

### Ideas

2. **(1-1)** AC #10's exact scenario (an `in_progress` *parent* driven up by the start-cascade, with an `in_progress` child) is covered by the same SQL path (`ReadyNoOpenChildren` excludes the parent regardless of its own status), but the kept test uses an *open* parent. A fixture with an in_progress parent would assert AC #10 literally — minor coverage nicety, not a correctness gap.
3. **(1-2)** No single test asserts the float *together with* a narrowing filter (e.g. mixed in_progress/open under `--type`). The property holds by construction (ordering keyed solely on `f.Ready`); optional hardening.
