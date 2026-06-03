# Review Report — Task 1-3: Correct the stats blocked-count derivation to the live set

- **Task ID**: tick-6e6a9c (ref `ready-includes-in-progress-1-3`)
- **Status**: ✅ Complete
- **Blocking issues**: 0
- **Spec ACs**: #9

## Implementation — Correct

`internal/cli/stats.go`:
- Line 85: `stats.Blocked = (stats.Open + stats.InProgress) - stats.Ready` — the spec's canonical arithmetic route over the live set.
- Line 78: ready-count comment refreshed to `// Ready count: open or in_progress, no unclosed blockers, no open/in-progress children, no blocked ancestor.`
- Line 84: blocked-count comment refreshed to `// Blocked count: live (open or in_progress) AND NOT ready, derived as (Open + InProgress) − Ready.`
- Ready count flows from `ReadyWhereClause()`, whose gate is already `t.status IN ('open', 'in_progress')` (task 1-1) — in_progress inclusion is automatic.
- **No `BlockedWhereClause()` helper added** — confirmed by grep; blocked stays derived arithmetically (the deliberate "Do NOT add" constraint).

## Tests — Adequate

- `stats_test.go` **"it counts ready and blocked tasks correctly"** — updated. Trace: Open=4 {aaa111, aaa222, ccc111, ccc222}, InProgress=1 {bbb111}. Ready=3 {aaa111, ccc222, bbb111}; aaa222 excluded (unclosed in_progress blocker bbb111 ∉ done/cancelled), ccc111 excluded (open child). Blocked=(4+1)−3=2. Inline fixture comment for bbb111 corrected to a ready leaf. Non-zero InProgress genuinely exercises the new derivation.
- `stats_test.go` **"it derives a non-negative blocked count when ready exceeds open"** — the negative-guard test **is present**: 1 open + 3 in_progress ready leaves → Open=1, InProgress=3, Ready=4. Asserts `ready>open` (guards the regression actually fires), `blocked==(1+3)−4==0`, and `blocked>=0`. The old `Open−Ready` would have been −3. Directly satisfies AC #9 "never negative."
- Formatting tests (TOON/Pretty/JSON, InProgress=0) unchanged — remain green.

Reverting stats.go:85 to `Open − Ready` would fail both new tests.

## Code Quality

Follows project conventions. Low complexity (one arithmetic expression). Reuses `ReadyWhereClause()` rather than adding parallel blocked-query surface, per the spec's deliberate decision.

## Live verification (orchestrator-run)

- End-to-end CLI smoke (local build): with 2 open + 2 in_progress (one started-but-blocked), `tick stats` reported `ready=3, blocked=1` where blocked = (Open 2 + InProgress 2) − Ready 3 = 1, non-negative, and ready includes the unblocked in_progress task (AC #9). Partition holds: 4 live = 3 ready + 1 blocked.
- `go test ./internal/cli -run 'TestStats|TestBlocked' -count=1` → ok; vet/gofmt clean.

## Non-blocking notes

- **[idea]** The `(Open + InProgress) − Ready` derivation is correct only while the shared-gate partition invariant holds; its drift guard lives in `query_helpers_test.go` (task 1-1). Accepted, canonical tradeoff — no action.
