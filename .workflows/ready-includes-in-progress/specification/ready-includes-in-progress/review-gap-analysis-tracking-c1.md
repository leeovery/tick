---
status: complete
created: 2026-06-03
cycle: 1
phase: Gap Analysis
topic: ready-includes-in-progress
---

# Review Tracking: ready-includes-in-progress - Gap Analysis

## Findings

### 1. Stats test "correct counts" — new expected numbers not stated

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Test Impact → "Tests asserting the OLD semantics — MUST change" (`stats_test.go` bullet); Acceptance Criteria #9

**Details**:
The spec says of `stats_test.go` "it counts ready and blocked tasks correctly": "expected counts change; this test exercises the `Blocked = (Open + InProgress) − Ready` derivation." It does not state the *new* expected values, so an implementer must re-derive them — and the derivation is non-obvious enough to be worth pinning, because the existing fixture contains a trap.

In that fixture, `tick-bbb111` is `in_progress`, has no blockers and no children of its own (it is only *referenced as* a blocker by `tick-aaa222`). Under the new gate it becomes a ready leaf. The fixture's six tasks therefore re-partition as:
- Open: 4 (aaa111, aaa222, ccc111, ccc222), InProgress: 1 (bbb111), Done: 1.
- Ready = aaa111 + ccc222 + bbb111 = 3 (bbb111 now joins).
- Blocked = (Open + InProgress) − Ready = (4 + 1) − 3 = 2 (aaa222, ccc111) — unchanged at 2, but for a different reason.

So `ready` changes 2 → 3 while `blocked` stays 2. The current inline comment ("=> neither ready nor blocked (not open)" for bbb111) becomes wrong. Stating the expected post-change numbers (and the corrected comment) prevents an implementer from "fixing" the test to the wrong value or leaving the misleading comment in place — and it makes Acceptance Criteria #9 ("ready count includes unblocked in_progress tasks") concretely verifiable against this exact fixture.

**Proposed Addition**:
Applied to the specification (see the affected sections above and git history).

**Resolution**: Approved
**Notes**: Auto-approved (cycle 1). All claims verified against the actual source files before applying.

---

### 2. blocked_test.go "excludes in_progress" — spec admits assertion passes "for the wrong reason" but leaves the fix-vs-keep call open

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Test Impact → "Tests asserting the OLD semantics — MUST change" (`blocked_test.go` bullet)

**Details**:
The spec says the `blocked_test.go` test "it excludes in_progress tasks from output" (line ~126) is "now misleading... the assertion as written may still pass for the wrong reason" and directs "Update the rationale." But it does not resolve the real question an implementer faces: a lone unblocked `in_progress` task is now *ready*, not *blocked-and-excluded*. Verifying it is absent from `blocked` no longer tests anything meaningful about `blocked` — it tests the partition only incidentally. The spec should decide whether to (a) just reword the comment and keep the weak assertion, or (b) strengthen it into a real partition assertion (the same `in_progress` task asserted present in `ready` and absent in `blocked`). Leaving "update the rationale" as the only instruction risks shipping a test that documents new behavior while asserting nothing about it. (Note: the proposed "new test to ADD" — "unblocked in_progress leaf appears in ready; a blocked in_progress appears in blocked" — may already cover (b); if so, the spec should say this old test is then redundant or downgraded, to avoid two tests drifting.)

**Proposed Addition**:
Applied to the specification (see the affected sections above and git history).

**Resolution**: Approved
**Notes**: Auto-approved (cycle 1). All claims verified against the actual source files before applying.

---

### 3. `--ready` + `--parent` ordering not stated (sort scope keyed on f.Ready, but combination unmentioned)

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Sort Ordering → "Scope: the ready filter (f.Ready)"; Affected Code Surface → `list.go` conditional ORDER BY

**Details**:
The sort decision is keyed purely on `f.Ready`. `tick list --ready --parent X` (and `--ready --tag`, `--ready --type`, etc.) all set `f.Ready` and so will float `in_progress` to the top. This is a reasonable and consistent consequence of "keyed on f.Ready," but the spec never states it — it only contrasts `tick ready`/`tick list --ready` against plain `tick list` and `tick list --blocked`. An implementer reading "resume-first is a property of ready's intent" might wonder whether scoping to a parent (a narrowing browse) should still float. Confirming explicitly that *any* f.Ready query floats regardless of additional filters removes the ambiguity and prevents an implementer from adding a special-case guard. (Recommend a one-line confirmation, not new behavior — the current code path already does this.)

**Proposed Addition**:
Applied to the specification (see the affected sections above and git history).

**Resolution**: Approved
**Notes**: Auto-approved (cycle 1). All claims verified against the actual source files before applying.

---

### 4. "First, otherwise" wording of AC #7 / `--count` not reflected in the ORDER BY description

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Acceptance Criteria #7; `--count` section; Affected Code Surface → `list.go` ORDER BY example

**Details**:
AC #7 states `--count 1` "returns the top unblocked in-flight task when one exists, otherwise the top unblocked open task." The proposed ORDER BY (`(t.status = 'in_progress') DESC, t.priority ASC, t.created ASC`) does produce this. But there is an unstated tie-case worth pinning for the implementer: when there are *zero* in_progress rows in the result set, the `(t.status = 'in_progress') DESC` term is uniformly false (0) across all rows and is a no-op — so the open-only ordering is byte-identical to today's `priority, created`. This is the basis for AC #2 / AC #6 "no regression," and confirming it explicitly (the band term collapses to identity when the band is empty) lets the implementer assert it directly and reassures the reviewer that a no-in_progress ready list is unchanged. Currently this only falls out implicitly.

**Proposed Addition**:
Applied to the specification (see the affected sections above and git history).

**Resolution**: Approved
**Notes**: Auto-approved (cycle 1). All claims verified against the actual source files before applying.

---

### 5. New resume-first ordering test — within-band tiebreak underspecified for the open band

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Test Impact → "New tests to ADD" (resume-first ordering bullet); Acceptance Criteria #5

**Details**:
The new ordering test is described as "float above open regardless of priority; within band priority, created." AC #5 says "within each band, priority ASC, created ASC holds." To make this test unambiguous and genuinely catch a regression, it needs a fixture that distinguishes the band term from the priority term — i.e. an `in_progress` task with a *worse* (higher number) priority than an `open` task, proving the in_progress row still sorts first. The spec says "float above open regardless of priority" which implies this, but the test inventory does not specify the discriminating fixture, so an implementer could write a test where in_progress happens to also have better priority (passing for the wrong reason, mirroring finding #2). Pinning the minimal discriminating fixture (e.g. in_progress priority 3 above open priority 0) would make the test load-bearing.

**Proposed Addition**:
Applied to the specification (see the affected sections above and git history).

**Resolution**: Approved
**Notes**: Auto-approved (cycle 1). All claims verified against the actual source files before applying.

---

### 6. stats.go:78 comment refresh — target text not given, and the line reference will drift

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Stats Counts → "Ready count tracks the new semantics"; Affected Code Surface → `stats.go`

**Details**:
The spec twice instructs "refresh the stale comment at `stats.go:78`" but does not say what the corrected comment should read. The current comment is `// Ready count: open, no unclosed blockers, no open children.` — it omits the in_progress gate *and* the no-blocked-ancestor condition (it was already incomplete pre-feature). An implementer left to invent replacement prose may reproduce the same incompleteness. Also, citing the literal line number (78) is brittle: any earlier edit shifts it, and there is in fact a *second* comment to update — line 84/85 `// Blocked count: open AND NOT ready (derived from ready).` — which the spec mentions only as "blocked derivation + comment" without separating it from the line-78 ready comment. Suggest specifying the intended replacement text for both comments and referencing them by content rather than line number.

**Proposed Addition**:
Applied to the specification (see the affected sections above and git history).

**Resolution**: Approved
**Notes**: Auto-approved (cycle 1). All claims verified against the actual source files before applying.

---

### 7. Quiet-mode and empty-result behavior for the widened ready view not stated

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Filters, --count, and Presentation → "Presentation — no change"

**Details**:
The presentation section asserts "no change" and is correct that formatting is untouched, but it speaks only of toon/JSON/pretty row rendering. Two adjacent behaviors that the widened gate now exercises are not mentioned: (1) `--quiet` mode (`RunList` prints bare IDs when `fc.Quiet`) — an in_progress task will now appear in `tick ready --quiet` ID output where it previously did not; this is the intended consequence but is unstated, and an agent piping `tick ready -q` is a primary consumer. (2) The empty-list case — `tick ready --status done` is documented to "return a silent empty list," but the spec does not state how an empty ready list renders across formats (it presumably uses the existing FormatTaskList empty path, unchanged). Confirming both are pre-existing, untouched paths (no new work) closes the loop so an implementer/reviewer does not treat them as undefined. If they are genuinely no-change, a single sentence suffices.

**Proposed Addition**:
Applied to the specification (see the affected sections above and git history).

**Resolution**: Approved
**Notes**: Auto-approved (cycle 1). All claims verified against the actual source files before applying.

---
