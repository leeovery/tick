# Implementation Analysis — Index

Comparison of tick-core implementations produced by different versions of the claude-technical-workflows implementation skill.

---

## Quick Links

| Document | Purpose |
|----------|---------|
| [Analysis Log](analysis-log.md) | Current state, findings, decisions, next steps |
| [Analysis Playbook](analysis-playbook.md) | Reusable instructions for running new analysis rounds |

## External

| Document | Location | Purpose |
|----------|----------|---------|
| implementation-version-analysis.md | `claude-technical-workflows` repo | PR-level analysis of what changed between V2 and V3 skill versions, with actionable recommendations |

---

## Analysis Rounds

### Round 0: V1 vs V2 (Feb 4, 2026)

Earlier, lighter comparison before V3 existed. Superseded by round-1 but preserved for historical context and baseline LOC stats.

| Document | Description |
|----------|-------------|
| [round-0/v1-vs-v2-comparison.md](round-0/v1-vs-v2-comparison.md) | Detailed V1 vs V2 with LOC stats, layer-by-layer verdicts |
| [round-0/v1-v2-v3-overview.md](round-0/v1-v2-v3-overview.md) | Lighter 3-way overview (superseded by round-1) |
| [round-0/original-analysis-instructions.md](round-0/original-analysis-instructions.md) | Original comparison instructions (ancestor of analysis-playbook.md) |

---

### Round 1: V1 vs V2 vs V3 (Feb 6, 2026)

3-way comparison of all implementations. **Result: V2 wins 21/23 tasks.**

| Document | Description |
|----------|-------------|
| [round-1/final-synthesis.md](round-1/final-synthesis.md) | **Start here.** Executive summary, full scorecard, version profiles, comparative patterns |
| [round-1/polish-impact.md](round-1/polish-impact.md) | Did V3's polish agent help or hurt? (Answer: helped, but couldn't fix foundational issues) |
| [round-1/workflow-skill-diff.md](round-1/workflow-skill-diff.md) | Exact diff of executor/reviewer prompts between V2 and V3 runs |
#### Task Reports (23)

| Task | File | Winner |
|------|------|--------|
| 1-1: Task model & ID generation | [tick-core-1-1.md](round-1/task-reports/tick-core-1-1.md) | V2 |
| 1-2: JSONL storage with atomic writes | [tick-core-1-2.md](round-1/task-reports/tick-core-1-2.md) | V2 |
| 1-3: SQLite cache with freshness detection | [tick-core-1-3.md](round-1/task-reports/tick-core-1-3.md) | V2 |
| 1-4: Storage engine with file locking | [tick-core-1-4.md](round-1/task-reports/tick-core-1-4.md) | V2 |
| 1-5: CLI framework & tick init | [tick-core-1-5.md](round-1/task-reports/tick-core-1-5.md) | V3 |
| 1-6: tick create command | [tick-core-1-6.md](round-1/task-reports/tick-core-1-6.md) | V2 |
| 1-7: tick list & tick show commands | [tick-core-1-7.md](round-1/task-reports/tick-core-1-7.md) | V2 |
| 2-1: Status transition validation | [tick-core-2-1.md](round-1/task-reports/tick-core-2-1.md) | V2 |
| 2-2: tick start, done, cancel, reopen | [tick-core-2-2.md](round-1/task-reports/tick-core-2-2.md) | Tie (V1 impl / V3 tests) |
| 2-3: tick update command | [tick-core-2-3.md](round-1/task-reports/tick-core-2-3.md) | V2 |
| 3-1: Dependency validation | [tick-core-3-1.md](round-1/task-reports/tick-core-3-1.md) | V2 |
| 3-2: tick dep add & tick dep rm | [tick-core-3-2.md](round-1/task-reports/tick-core-3-2.md) | V2 |
| 3-3: Ready query & tick ready | [tick-core-3-3.md](round-1/task-reports/tick-core-3-3.md) | V2 |
| 3-4: Blocked query & tick blocked | [tick-core-3-4.md](round-1/task-reports/tick-core-3-4.md) | V2 |
| 3-5: tick list filter flags | [tick-core-3-5.md](round-1/task-reports/tick-core-3-5.md) | V2 |
| 4-1: Formatter abstraction & TTY | [tick-core-4-1.md](round-1/task-reports/tick-core-4-1.md) | V2 |
| 4-2: TOON formatter | [tick-core-4-2.md](round-1/task-reports/tick-core-4-2.md) | V2 |
| 4-3: Human-readable formatter | [tick-core-4-3.md](round-1/task-reports/tick-core-4-3.md) | V2 |
| 4-4: JSON formatter | [tick-core-4-4.md](round-1/task-reports/tick-core-4-4.md) | V2 |
| 4-5: Integrate formatters | [tick-core-4-5.md](round-1/task-reports/tick-core-4-5.md) | V2 |
| 4-6: Verbose output | [tick-core-4-6.md](round-1/task-reports/tick-core-4-6.md) | V2 |
| 5-1: tick stats command | [tick-core-5-1.md](round-1/task-reports/tick-core-5-1.md) | V2 |
| 5-2: tick rebuild command | [tick-core-5-2.md](round-1/task-reports/tick-core-5-2.md) | V2 |

#### Phase Reports (5)

| Phase | File | Winner |
|-------|------|--------|
| Phase 1: Walking Skeleton | [phase-1.md](round-1/phase-reports/phase-1.md) | V2 (6/7 tasks) |
| Phase 2: Task Lifecycle | [phase-2.md](round-1/phase-reports/phase-2.md) | V2 |
| Phase 3: Dependencies | [phase-3.md](round-1/phase-reports/phase-3.md) | V2 (5/5 tasks) |
| Phase 4: Output Formats | [phase-4.md](round-1/phase-reports/phase-4.md) | V2 (6/6 tasks) |
| Phase 5: Stats & Cache | [phase-5.md](round-1/phase-reports/phase-5.md) | V2 (2/2 tasks) |

---

### Round 2: V4 vs V2 (Feb 8, 2026)

2-way comparison. V4 = workflow changes based on round-1 findings. V2 = baseline (current best). **Result: V4 wins 15/23 tasks, all 5 phases.**

| Document | Description |
|----------|-------------|
| [round-2/final-synthesis.md](round-2/final-synthesis.md) | **Start here.** Executive summary, full scorecard, workflow validation |

#### Task Reports (23)

| Task | File | Winner |
|------|------|--------|
| 1-1: Task model & ID generation | [tick-core-1-1.md](round-2/task-reports/tick-core-1-1.md) | V2 |
| 1-2: JSONL storage with atomic writes | [tick-core-1-2.md](round-2/task-reports/tick-core-1-2.md) | V4 |
| 1-3: SQLite cache with freshness detection | [tick-core-1-3.md](round-2/task-reports/tick-core-1-3.md) | Close |
| 1-4: Storage engine with file locking | [tick-core-1-4.md](round-2/task-reports/tick-core-1-4.md) | V4 |
| 1-5: CLI framework & tick init | [tick-core-1-5.md](round-2/task-reports/tick-core-1-5.md) | V2 |
| 1-6: tick create command | [tick-core-1-6.md](round-2/task-reports/tick-core-1-6.md) | V4 |
| 1-7: tick list & tick show commands | [tick-core-1-7.md](round-2/task-reports/tick-core-1-7.md) | V4 |
| 2-1: Status transition validation | [tick-core-2-1.md](round-2/task-reports/tick-core-2-1.md) | Mixed |
| 2-2: tick start, done, cancel, reopen | [tick-core-2-2.md](round-2/task-reports/tick-core-2-2.md) | V4 |
| 2-3: tick update command | [tick-core-2-3.md](round-2/task-reports/tick-core-2-3.md) | V2 |
| 3-1: Dependency validation | [tick-core-3-1.md](round-2/task-reports/tick-core-3-1.md) | V4 |
| 3-2: tick dep add & tick dep rm | [tick-core-3-2.md](round-2/task-reports/tick-core-3-2.md) | V2 |
| 3-3: Ready query & tick ready | [tick-core-3-3.md](round-2/task-reports/tick-core-3-3.md) | V2 |
| 3-4: Blocked query & tick blocked | [tick-core-3-4.md](round-2/task-reports/tick-core-3-4.md) | V4 |
| 3-5: tick list filter flags | [tick-core-3-5.md](round-2/task-reports/tick-core-3-5.md) | V4 |
| 4-1: Formatter abstraction & TTY | [tick-core-4-1.md](round-2/task-reports/tick-core-4-1.md) | V4 |
| 4-2: TOON formatter | [tick-core-4-2.md](round-2/task-reports/tick-core-4-2.md) | V2 |
| 4-3: Human-readable formatter | [tick-core-4-3.md](round-2/task-reports/tick-core-4-3.md) | V4 |
| 4-4: JSON formatter | [tick-core-4-4.md](round-2/task-reports/tick-core-4-4.md) | V4 |
| 4-5: Integrate formatters | [tick-core-4-5.md](round-2/task-reports/tick-core-4-5.md) | V4 |
| 4-6: Verbose output | [tick-core-4-6.md](round-2/task-reports/tick-core-4-6.md) | V2 |
| 5-1: tick stats command | [tick-core-5-1.md](round-2/task-reports/tick-core-5-1.md) | V4 |
| 5-2: tick rebuild command | [tick-core-5-2.md](round-2/task-reports/tick-core-5-2.md) | V4 |

#### Phase Reports (5)

| Phase | File | Winner |
|-------|------|--------|
| Phase 1: Walking Skeleton | [phase-1.md](round-2/phase-reports/phase-1.md) | V4 (4/7 tasks) |
| Phase 2: Task Lifecycle | [phase-2.md](round-2/phase-reports/phase-2.md) | V4 (narrow) |
| Phase 3: Dependencies | [phase-3.md](round-2/phase-reports/phase-3.md) | V4 (3/5 tasks) |
| Phase 4: Output Formats | [phase-4.md](round-2/phase-reports/phase-4.md) | V4 (4/6 tasks) |
| Phase 5: Stats & Cache | [phase-5.md](round-2/phase-reports/phase-5.md) | V4 (2/2 tasks) |

---

### Round 3: V5 vs V4 (Feb 10, 2026)

2-way comparison. V5 = workflow changes based on V4 implementation review findings + analysis refinement phase. V4 = baseline (current best). **Result: V5 wins 16/23 comparable tasks, 3/5 comparable phases, plus 8 V5-only tasks all rated Excellent.**

| Document | Description |
|----------|-------------|
| [round-3/final-synthesis.md](round-3/final-synthesis.md) | **Start here.** Executive summary, full scorecard, workflow validation |

#### Task Reports (23 Comparable + 8 V5-Only)

| Task | File | Winner |
|------|------|--------|
| 1-1: Task model & ID generation | [tick-core-1-1.md](round-3/task-reports/tick-core-1-1.md) | V5 |
| 1-2: JSONL storage with atomic writes | [tick-core-1-2.md](round-3/task-reports/tick-core-1-2.md) | V5 |
| 1-3: SQLite cache with freshness detection | [tick-core-1-3.md](round-3/task-reports/tick-core-1-3.md) | V5 |
| 1-4: Storage engine with file locking | [tick-core-1-4.md](round-3/task-reports/tick-core-1-4.md) | V5 |
| 1-5: CLI framework & tick init | [tick-core-1-5.md](round-3/task-reports/tick-core-1-5.md) | V5 |
| 1-6: tick create command | [tick-core-1-6.md](round-3/task-reports/tick-core-1-6.md) | V4 |
| 1-7: tick list & tick show commands | [tick-core-1-7.md](round-3/task-reports/tick-core-1-7.md) | V5 |
| 2-1: Status transition validation | [tick-core-2-1.md](round-3/task-reports/tick-core-2-1.md) | Tie |
| 2-2: tick start, done, cancel, reopen | [tick-core-2-2.md](round-3/task-reports/tick-core-2-2.md) | V4 |
| 2-3: tick update command | [tick-core-2-3.md](round-3/task-reports/tick-core-2-3.md) | V5 |
| 3-1: Dependency validation | [tick-core-3-1.md](round-3/task-reports/tick-core-3-1.md) | V4 |
| 3-2: tick dep add & tick dep rm | [tick-core-3-2.md](round-3/task-reports/tick-core-3-2.md) | V5 |
| 3-3: Ready query & tick ready | [tick-core-3-3.md](round-3/task-reports/tick-core-3-3.md) | V5 |
| 3-4: Blocked query & tick blocked | [tick-core-3-4.md](round-3/task-reports/tick-core-3-4.md) | V4 |
| 3-5: tick list filter flags | [tick-core-3-5.md](round-3/task-reports/tick-core-3-5.md) | V5 |
| 3-6: Parent scoping (V5 only) | [tick-core-3-6.md](round-3/task-reports/tick-core-3-6.md) | V5 only (Excellent) |
| 4-1: Formatter abstraction & TTY | [tick-core-4-1.md](round-3/task-reports/tick-core-4-1.md) | V5 |
| 4-2: TOON formatter | [tick-core-4-2.md](round-3/task-reports/tick-core-4-2.md) | V5 |
| 4-3: Human-readable formatter | [tick-core-4-3.md](round-3/task-reports/tick-core-4-3.md) | V5 |
| 4-4: JSON formatter | [tick-core-4-4.md](round-3/task-reports/tick-core-4-4.md) | V4 |
| 4-5: Integrate formatters | [tick-core-4-5.md](round-3/task-reports/tick-core-4-5.md) | V5 |
| 4-6: Verbose output | [tick-core-4-6.md](round-3/task-reports/tick-core-4-6.md) | V5 |
| 5-1: tick stats command | [tick-core-5-1.md](round-3/task-reports/tick-core-5-1.md) | V4 |
| 5-2: tick rebuild command | [tick-core-5-2.md](round-3/task-reports/tick-core-5-2.md) | V5 |
| 6-1: Dependency validation gaps (V5 only) | [tick-core-6-1.md](round-3/task-reports/tick-core-6-1.md) | V5 only (Excellent) |
| 6-2: Shared ready/blocked SQL (V5 only) | [tick-core-6-2.md](round-3/task-reports/tick-core-6-2.md) | V5 only (Excellent) |
| 6-3: Consolidate JSONL parsing (V5 only) | [tick-core-6-3.md](round-3/task-reports/tick-core-6-3.md) | V5 only (Excellent) |
| 6-4: Shared formatter methods (V5 only) | [tick-core-6-4.md](round-3/task-reports/tick-core-6-4.md) | V5 only (Excellent) |
| 6-5: Remove doctor from help (V5 only) | [tick-core-6-5.md](round-3/task-reports/tick-core-6-5.md) | V5 only (Excellent) |
| 6-6: Remove dead StubFormatter (V5 only) | [tick-core-6-6.md](round-3/task-reports/tick-core-6-6.md) | V5 only (Excellent) |
| 6-7: Type-safe formatter params (V5 only) | [tick-core-6-7.md](round-3/task-reports/tick-core-6-7.md) | V5 only (Excellent) |

#### Phase Reports (5 Comparable + 1 V5-Only)

| Phase | File | Winner |
|-------|------|--------|
| Phase 1: Walking Skeleton | [phase-1.md](round-3/phase-reports/phase-1.md) | V5 (6/7 tasks) |
| Phase 2: Task Lifecycle | [phase-2.md](round-3/phase-reports/phase-2.md) | Tie |
| Phase 3: Dependencies | [phase-3.md](round-3/phase-reports/phase-3.md) | V5 (3/5 tasks) |
| Phase 4: Output Formats | [phase-4.md](round-3/phase-reports/phase-4.md) | V5 (5/6 tasks) |
| Phase 5: Stats & Cache | [phase-5.md](round-3/phase-reports/phase-5.md) | Tie |
| Phase 6: Analysis Refinements | [phase-6.md](round-3/phase-reports/phase-6.md) | V5 only (Excellent) |

---

## Key Findings Summary

### Round 1: Why V2 Won (vs V3)

- Full spec compliance across all 23 tasks
- Sub-package architecture, composable SQL, store-injected logging
- Each executor independently converged on Go-idiomatic patterns
- Simpler, less prescriptive executor instructions left more room for good decisions

### Round 1: Why V3 Regressed

- PR #79's integration context file created a "convention gravity well"
- Task 1-1's unconventional choices (string timestamps, bare errors) got documented as established patterns
- Reviewer's cohesion dimension enforced consistency with early mistakes
- Prescriptive exploration instructions diluted attention from actual implementation

### Round 2: V4 Exceeds V2

- V4 wins 15/23 tasks across all 5 phases
- Type-safe formatter interface, single-source SQL reuse, cleaner error flow
- Removing PR #79 eliminated convention gravity well — validation of rollback strategy
- V2 retains edge in integration tests, defensive normalization, spec-verbatim messages, edge-case test coverage

### Polish Agent: Beneficial But Disproportionate

- V3: Removed dead code, extracted shared helpers, fixed dependency validation bug. Net -181 lines. Zero regressions.
- V4: One genuinely valuable finding (--blocks cycle detection gap, ~18 LOC). But: 75 minutes, 152 tool calls, violated its own protocol (made direct changes instead of dispatching executor/reviewer sub-agents). 505 of 533 insertions were test code that should have been a plan task. See `v4-implementation-review.md` sections 5.4 and 7.4 for quantitative analysis and structural recommendations.

### What to Keep from V3

| Addition | Keep? | Reason |
|----------|-------|--------|
| PR #77: Fix executor re-attempts | Yes | Pure bugfix |
| PR #78: Fix recommendations (FIX/ALT/CONF) | Yes | Best V3 addition |
| PR #78: fix_gate_mode stop gates | Yes | Good human-in-the-loop |
| PR #80: Polish agent | Yes | Proven beneficial |
| PR #79: Integration context | No | Convention lock-in |
| PR #79: Cohesion review dimension | No | Enforced early mistakes |
| PR #79: Prescriptive exploration | No | Attention dilution |
| PR #79: Plan-file access for executor | No | Over-engineering pressure |
| PR #79: "Same developer" instruction | No | Conflicts with quality standards |

### Round 3: V5 Exceeds V4

- V5 wins 16/23 comparable tasks, 3/5 comparable phases — largest margin in any round
- Function-based architecture, DRY helpers, spec-compliant timestamps, exhaustive write-error handling
- Phase 6 analysis refinements (7 tasks, all Excellent) demonstrate self-correction capability unique to V5's workflow
- V5's `interface{}` Formatter was its biggest early weakness — self-corrected in Phase 6 task 6-7
- V4 retains edge in test assertion depth, correctness invariants (blocked count derivation), spec-exact error messages, and function decomposition
- The ideal V6 would combine V5's architecture with V4's verification rigor

### Methodology Note (Round 2)

Round 2 analysis did not account for project skill compliance (golang-pro) or spec-vs-convention conflict resolution. The playbook has been updated with two additional dimensions for future rounds. Some task verdicts (particularly around error message format) may have shifted slightly in V4's favour if skill adherence had been evaluated.
