# Review Report — Task 1-1: Widen the shared ready/blocked status gate to live tasks

- **Task ID**: tick-fe7e70 (ref `ready-includes-in-progress-1-1`)
- **Status**: ✅ Complete
- **Blocking issues**: 0
- **Spec ACs**: #1, #2, #3, #4, #8, #10

## Implementation — Correct

`internal/cli/query_helpers.go`:
- `ReadyConditions()` line 53: `conditions[0]` = `` `t.status IN ('open', 'in_progress')` ``
- `BlockedConditions()` line 78: first slice element = `` `t.status IN ('open', 'in_progress')` ``
- `negateNotExists()`, `ReadyNoUnclosedBlockers()`, `ReadyNoOpenChildren()` (already matched `IN ('open','in_progress')` at line 24), `ReadyNoBlockedAncestor()`, and the `parts`/`strings.Join` disjunction are **untouched** — the De Morgan inverse machinery is preserved.
- Grep for `status = 'open'` across `internal/cli` returns **zero** matches — no stale literal remains.

Wiring: `buildListQuery` appends the conditions; `--status` composes as an additional `t.status = ?` AND term, producing the `ready --status open` / `--status in_progress` intersections.

## Tests — Adequate, assert the right things

- **query_helpers_test.go**: `conditions[0]` literals updated to the new gate; `len()` (Ready==4, Blocked==2) and EXISTS-count (==3) and negateNotExists-derivation assertions unchanged.
- **ready_test.go**: old "it excludes in_progress tasks" correctly **inverted and renamed** to `"it includes unblocked in_progress leaf"`. KEEP tests preserved (`excludes task with in_progress blocker`, `excludes parent with in_progress children`). New `"it partitions an in_progress task into exactly one of ready/blocked"` is a genuine partition assertion. New `"it returns only unstarted work for ready --status open"` and `"it returns only resumptions for ready --status in_progress"` assert presence AND absence of the complementary status.
- **blocked_test.go**: old "it excludes in_progress tasks from output" **rewritten per spec option (b)** to `"it includes blocked in_progress task"` — gives the in_progress task an unclosed blocker and asserts it APPEARS in blocked (a real assertion, not passing for the wrong reason).
- **list_filter_test.go**: stale comment on "contradictory filters return empty result no error" refreshed.

## Code Quality — No concerns

Project conventions followed. The SQL literal duplicated identically on both sides is intentional (the symmetry is load-bearing for the De Morgan derivation, not a DRY violation).

## Live verification (orchestrator-run)

- `go test ./internal/cli -count=1` → **ok**; full `go test ./...` → **all packages ok**; `-race` on feature tests → **ok**.
- `go vet ./...` → clean; `gofmt -l ./internal ./cmd` → clean.
- End-to-end CLI smoke (local build): unblocked in_progress leaf appears in `ready` (AC #1); blocked in_progress appears only in `blocked` (AC #3); live tasks partition exactly between ready/blocked (AC #4); `ready --status open`/`in_progress`/`done` compose correctly (AC #8).

## Non-blocking notes

- **[idea]** AC #10's exact scenario (in_progress *parent* via start-cascade) is covered by the same SQL path (`ReadyNoOpenChildren` excludes regardless of parent status) but the kept test uses an *open* parent; an in_progress-parent fixture would assert AC #10 literally. Minor coverage nicety, not a correctness gap.
- **[quickfix]** Two query_helpers_test.go subtest names still read "...status open..." / "...beyond status check" — cosmetic; the assertions are already correct for the widened gate.
