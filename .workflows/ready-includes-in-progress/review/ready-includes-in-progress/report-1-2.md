# Review Report — Task 1-2: Float unblocked in-progress to the top of the ready view

- **Task ID**: tick-2f0d2a (ref `ready-includes-in-progress-1-2`)
- **Status**: ✅ Complete
- **Blocking issues**: 0
- **Spec ACs**: #5, #6, #7

## Implementation — Correct

`internal/cli/list.go` `buildListQuery` (lines 311–324):
- `f.Ready` true → `ORDER BY (t.status = 'in_progress') DESC, t.priority ASC, t.created ASC` (line 316)
- else → `ORDER BY t.priority ASC, t.created ASC` (line 318), unchanged
- `LIMIT` appended **after** the ORDER BY block (line 322) — so `--count` applies after resume-first ordering.

Verified:
- **Band term correct**: SQLite evaluates `t.status = 'in_progress'` to 1/0; `DESC` floats the in_progress band to the top, with `priority ASC, created ASC` within each band.
- **No narrowing-filter guard**: the ORDER BY branch keys solely on `f.Ready`; `--parent/--tag/--type/--priority/--status` append to WHERE only. Matches spec.
- **Zero-in_progress no-op**: with no in_progress rows the band term is uniformly 0 → byte-identical to the neutral clause.
- **`t.created` sort sound**: stored as RFC3339 UTC TEXT, so `t.created ASC` is lexically == chronologically correct.

## Tests — Adequate (not under/over-tested)

- `ready_test.go` **"it floats unblocked in_progress to the top of ready"** — the discriminating test: in_progress `prog01` priority **3 (worst)**, open `open01` priority **0 (best)**; asserts the best-priority open row sorts *below* both in_progress rows. Genuinely separates band term from priority term.
- `ready_test.go` **"it floats in_progress identically for list --ready"** — drives `tick list --ready` via `runList`; locks the `f.Ready` scope decision.
- `ready_test.go` **"it returns the top unblocked in-flight task with --count 1"** — `--count 1 --quiet` returns the floated in_progress task over a better-priority open task (AC #7 in-flight branch); plus a zero-in_progress branch returning top open (AC #7 open branch).
- **Unchanged-ordering proofs kept**: `blocked_test.go` (blocked retains neutral order, AC #6), `list_filter_test.go` (plain list neutral order, AC #6), `ready_test.go` (all-open ready = byte-identical, no-regression).

Tests bite: band removal, dropping `f.Ready`, or mis-ordering LIMIT each flip an assertion.

## Code Quality — Clean

Localized single-branch change; LIMIT parameterized via `?`; the `'in_progress'` literal is a hardcoded constant (no injection surface). Comment documents both band semantics and the zero-in_progress no-op.

## Live verification (orchestrator-run)

- End-to-end CLI smoke (local build): `tick ready` floats `prog-worst` (in_progress, priority 3) **above** `open-best` (open, priority 0) — proves band beats priority (AC #5). `tick list` keeps neutral order with no float (AC #6). `tick ready --count 1 -q` returns the top unblocked in-flight task (AC #7).
- Feature tests pass under `-race`; vet/gofmt clean.

## Non-blocking notes

- **[idea]** No single test asserts the float *together with* a narrowing filter (e.g. mixed in_progress/open under `--type`). The property holds by construction (ordering keyed solely on `f.Ready`); optional hardening, not a gap.
