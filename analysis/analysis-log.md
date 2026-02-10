# Implementation Analysis Log

Tracking document for the ongoing comparison of tick-core implementations produced by different versions of the claude-technical-workflows implementation skill.

---

## Current State (Feb 10, 2026)

### What We've Done

**Round 1 (Feb 6)**:
1. **23 task-level reports** comparing V1, V2, V3 code for every plan task (`round-1/task-reports/`)
2. **5 phase-level reports** analysing cross-task patterns per phase (`round-1/phase-reports/`)
3. **Final synthesis** aggregating all findings (`round-1/final-synthesis.md`)
4. **Polish impact analysis** isolating V3's polish commit effects (`round-1/polish-impact.md`)
5. **Workflow skill diff** comparing executor/reviewer prompts between V2 and V3 runs (`round-1/workflow-skill-diff.md`)
6. **External version analysis** from the workflows repo (`claude-technical-workflows/implementation-version-analysis.md`)

**Round 2 (Feb 8)**:
7. **23 task-level reports** comparing V2 vs V4 (`round-2/task-reports/`)
8. **5 phase-level reports** for V2 vs V4 (`round-2/phase-reports/`)
9. **Final synthesis** — V4 vs V2 definitive comparison (`round-2/final-synthesis.md`)
10. **Playbook update** — added Skill Compliance and Spec-vs-Convention Conflicts dimensions

**Round 3 (Feb 10)**:
11. **23 comparable task-level reports** comparing V4 vs V5 (`round-3/task-reports/`)
12. **8 V5-only task assessments** — 1 extra feature (3-6) + 7 Phase 6 refinements (6-1 through 6-7) (`round-3/task-reports/`)
13. **5 comparable phase-level reports** for V4 vs V5 (`round-3/phase-reports/`)
14. **1 V5-only phase assessment** — Phase 6 analysis refinements (`round-3/phase-reports/phase-6.md`)
15. **Final synthesis** — V5 vs V4 definitive comparison (`round-3/final-synthesis.md`)

### Key Findings

**Round 1**: V2 wins 21/23 tasks, all 5 phases. V3 wins 1/23 (task 1-5). V1 wins 0/23.

**Round 2**: V4 wins 15/23 tasks, all 5 phases. V2 wins 7/23. 1 close call (1-3).

**Round 3**: V5 wins 16/23 comparable tasks, 3/5 comparable phases. V4 wins 6/23. 1 tie. Plus 8 V5-only tasks all rated Excellent.

**Root cause of V3's regression**: PR #79 (integration context + codebase cohesion) created a "convention gravity well" where V3's task 1-1 made unconventional Go choices (string timestamps, bare error returns, no NewTask factory) that got documented as established patterns in the integration context file. Every subsequent executor faithfully propagated these choices because it was instructed to "match conventions" and the reviewer's cohesion dimension actively enforced consistency with them.

**The Polish agent (PR #80) was purely beneficial**: removed dead code, extracted shared helpers, fixed a missing dependency validation bug. Zero regressions. Did not cause any of V3's quality issues.

**PR #77 (fix executor re-attempts)** was a correct bugfix with no negative effects.

**PR #78 (fix recommendations + fix_gate_mode)** was neutral-to-positive. The structured FIX/ALTERNATIVE/CONFIDENCE output from reviewers is the single best V3 addition.

### The Problem in One Sentence

V3's integration context mechanism amplifies whatever direction the first few tasks set — good or bad — by documenting early decisions as constraints and having the reviewer enforce consistency with them.

---

## Version Inventory

| Version | Branch | Workflow Version | Dates | Tasks Won (vs prev best) |
|---------|--------|-----------------|-------|--------------------------|
| V1 | `implementation-v1` | Pre-#73 (monolithic) | Pre-Feb 2 | 0/23 (vs V2) |
| V2 | `implementation-v2` | v2.1.3 (PR #73) | Feb 3 | 21/23 (vs V3), 7/23 (vs V4) |
| V3 | `implementation-v3` | v2.1.5 (PRs #77-80) | Feb 5 | 1/23 (vs V2) |
| V4 | `implementation-v4` | V2 base + PRs #77/#78/#80 (no #79) | Feb 7-8 | 15/23 (vs V2), 6/23 (vs V5) |
| V5 | `implementation-v5` | V4 base + analysis refinement phase | Feb 9 | 16/23 (vs V4) + 8 V5-only (all Excellent) |

### Commit Mapping

All commit SHAs for each version's 23 tasks are documented in the plan that produced this analysis. See the Appendix in the original plan or the task-level reports for specific commits.

### Worktree Paths (for future analysis)

```
git worktree add /private/tmp/tick-analysis-worktrees/v1 implementation-v1
git worktree add /private/tmp/tick-analysis-worktrees/v2 implementation-v2
git worktree add /private/tmp/tick-analysis-worktrees/v3 implementation-v3
git worktree add /private/tmp/tick-analysis-worktrees/v4 implementation-v4
git worktree add /private/tmp/tick-analysis-worktrees/v5 implementation-v5
```

---

## File Structure

```
analysis/
  index.md                    <- Start here. Directory of everything.
  analysis-log.md             <- This file. State tracking across rounds.
  analysis-playbook.md        <- Reusable instructions for running analysis rounds.
  round-1/                    <- V1 vs V2 vs V3 (3-way comparison, Feb 6 2026)
    task-reports/             <- 23 per-task reports
      tick-core-{P}-{T}.md
    phase-reports/            <- 5 cross-task phase reports
      phase-{N}.md
    final-synthesis.md        <- Definitive V1/V2/V3 comparison
    polish-impact.md          <- V3 polish commit analysis
    workflow-skill-diff.md    <- Executor/reviewer prompt diffs between V2 and V3
    root-cause-task-1-1.md    <- Why V3 chose string timestamps
    course-correction-evidence.md <- V2's 6 retroactive fixes vs V3's zero
  round-2/                    <- V4 vs V2 (2-way comparison, Feb 8 2026)
    task-reports/             <- 23 per-task reports
      tick-core-{P}-{T}.md
    phase-reports/            <- 5 cross-task phase reports
      phase-{N}.md
    final-synthesis.md        <- Definitive V4 vs V2 comparison
  round-3/                    <- V5 vs V4 (2-way comparison, Feb 10 2026)
    task-reports/             <- 31 task reports (23 comparable + 8 V5-only)
      tick-core-{P}-{T}.md
    phase-reports/            <- 6 phase reports (5 comparable + 1 V5-only)
      phase-{N}.md
    final-synthesis.md        <- Definitive V5 vs V4 comparison
```

External:
| File | Location |
|------|----------|
| `implementation-version-analysis.md` | `claude-technical-workflows` repo — PR-level analysis with actionable recommendations |

---

## What Changed Between V2 and V3 (PR Summary)

| PR | Change | Impact | Keep? |
|----|--------|--------|-------|
| #76 | Commands-to-skills migration | Neutral (structural) | N/A |
| #77 | Fix executor re-attempt context | **Positive** (bugfix) | Yes |
| #78 | Fix recommendations + fix_gate_mode | **Positive** (better reviewer output, stop gates) | Yes |
| #79 | Integration context + cohesion review + prescriptive exploration | **Negative** (convention lock-in, attention dilution) | Rework |
| #80 | Polish agent | **Positive** (dead code removal, DRY, bug fix) | Yes |

---

## Completed: V4 (Round 2)

### Approach Used: Option A (Rollback to V2 + cherry-pick)

**Base**: V2's executor and reviewer (simpler, less prescriptive)

**Kept from V3**: PRs #77 (bugfix), #78 (fix recs + stop gates), #80 (polish)

**Removed**: PR #79 entirely (integration context, cohesion review, prescriptive exploration)

### Result: V4 Exceeds V2

V4 wins 15/23 tasks, all 5 phases. The rollback strategy is validated — removing PR #79 eliminated the convention gravity well while the cherry-picked additions (structured fix recommendations, polish agent) contributed positively.

**V4's key advantages over V2**: type-safe formatter interface (concrete `StatsData` vs `interface{}`), single-source SQL reuse via `readyConditionsFor(alias)`, cleaner storage error flow, type-safe test infrastructure, more concise implementations.

**V2's remaining advantages**: binary-level integration tests (V4 has zero), defensive `NormalizeID()` at every comparison, spec-verbatim error messages, exact-string formatter test assertions, compile-time interface checks.

### Methodology Gap Discovered

Round 2 did not evaluate project skill compliance (golang-pro MUST DO/MUST NOT DO) or handle spec-vs-convention conflicts (e.g. Go's lowercase error convention vs spec's capitalized messages). Some verdicts crediting V2 for "spec-verbatim" error messages should have been neutral or pro-V4, since V4 followed Go idioms correctly. Playbook updated with two new analysis dimensions for future rounds.

---

## Completed: V5 (Round 3)

### What Changed in V5's Workflow

V5 built on V4's base (V2 + PRs #77/#78/#80) and added a post-implementation analysis refinement phase. The analysis process identified 7 findings across 4 categories:
- **Validation gap** (6-1): `create --blocked-by/--blocks` and `update --blocks` bypassed cycle detection
- **Code duplication** (6-2, 6-3, 6-4): Duplicate SQL WHERE clauses, JSONL parsing, formatter methods
- **Dead code** (6-5, 6-6): Phantom doctor help entry, unreachable StubFormatter
- **Type safety** (6-7): 15 `interface{}` formatter parameters replaced with concrete types

V5 also implemented one additional plan task (3-6: parent scoping with --parent flag) not present in the V4 plan.

### Result: V5 Exceeds V4

V5 wins 16/23 comparable tasks, 3/5 comparable phases. The margin is the largest in any round. Key V5 advantages: function-based architecture, DRY helpers, spec-compliant ISO 8601 timestamps, exhaustive write-error handling, and self-correction via Phase 6.

V4's remaining advantages: deeper test assertions (timestamp verification, typed JSON deserialization, exact string matching), correctness invariants (blocked = open - ready), spec-exact error messages with "Error:" prefix, and superior function decomposition in dependency validation.

### The Self-Correction Arc

The most significant finding is V5's `interface{}` Formatter — introduced in Phase 4 as a significant anti-pattern, then systematically eliminated in Phase 6 task 6-7. This demonstrates that V5's workflow can identify and fix its own mistakes, a capability no previous version exhibited at this structural level.

---

## Planned Next Step: V6

V5 is now the baseline. Recommendations from Round 3 synthesis for V6:
1. Restore V4-level test assertion depth (timestamp verification, typed JSON deserialization, exact string matching)
2. Enforce spec-exact error messages where spec prescribes them
3. Keep Phase 6 analysis refinement process (proven beneficial)
4. Fix FormatMessage error swallowing (return error, don't discard)
5. Derive correctness invariants where possible (blocked = open - ready)
6. Mandate compile-time type safety from Phase 4 onward (no interface{} Formatter params)

---

## Future Ideas (from discussion, not yet actioned)

These emerged from analysing why V3 regressed. Not for implementation now — the tick-core spec/plan must NOT change (to keep experiments controlled). All future experiments use the same planning and specification documents. The goal is making the implementation skill robust to ambiguous plans, because that's the nature of planning.

### The Core Insight: Feedback Loop Direction

The V3 failure wasn't "bad plan" or "bad executor" — it was the feedback loop direction. V3 created a **positive** feedback loop (early decisions reinforced by integration context + cohesion review). V2 had **no** feedback loop (stateless executors). The ideal might be a **negative** feedback loop — one that actively challenges early decisions rather than reinforcing or ignoring them.

### 1. Language-Specific Type Guidance at Implementation Time

The spec says "ISO 8601 UTC format" — a perfectly good spec-level statement. The problem is the translation from spec to code. A Go developer reads that and thinks `time.Time` internally, serialise to string at boundaries. V3's executor read the same words and thought "strings throughout." Both defensible — one is idiomatically wrong.

The gap isn't in planning/specification (those should stay language-agnostic, focused on *what* not *how*). It's in the executor's context. The executor already has access to language skills (golang-pro) but doesn't consult them for foundational type decisions — it pattern-matches on spec text instead.

**Possible approach**: Executor prompts could instruct early tasks (1-3) to explicitly consult the language skill for type/idiom guidance on foundational structures. Or the reviewer could flag "this looks like a string type where the language idiom would be a dedicated type" on early tasks.

Note: We already have internal/external perspective analysis at planning and specification stages, but these deliberately avoid code-level discussion. That was intentional to keep those stages clean. Introducing code examples or language-specific sections at those stages risks polluting them. Better to handle this at the implementation boundary.

### 2. Mid-Implementation Course Correction (Phase Checkpoint)

V2 self-corrected naturally (6 retroactive fixes across 23 tasks). V3 couldn't because of convention lock-in. V4 (removing integration context) should restore natural course correction, but it's still luck-dependent — V2 *happened* to get task 1-1 right. We don't know if a V2 executor would have gone back and changed a type-level architectural choice vs just adding a helper function.

**Possible approach**: A periodic checkpoint agent — say after phase 1 completes — that asks: "Here are the foundational types and patterns established so far. Do any of these violate language idioms or create problems for upcoming tasks?" If yes, fix them *before* building 16 more tasks on top.

**Key distinction from integration context**: The checkpoint's job is to *challenge* early decisions, not *document* them as conventions. It's adversarial rather than conformist. It should produce fix instructions, not a "patterns to match" file.

**Risk**: This starts looking like integration context again if not carefully scoped. Must be framed as "what's wrong?" not "what exists?"

### 3. Making Implementation Robust to Ambiguous Plans

Rather than perfecting plans (impossible), the implementation skill needs to handle ambiguity well:
- Executors should bring language expertise to ambiguous areas (not just pattern-match the spec)
- Reviewers should catch idiom violations early (tasks 1-3 especially), not just check acceptance criteria
- Course correction should be possible at any point, not locked by convention conformity
- V4's Option A already addresses the third point by removing convention lock-in; the first two are future improvements

---

## Open Questions

1. Should V5 test on a different project type to control for tick-specific factors?
2. Should the foundational design review (phase checkpoint agent — see Future Ideas §2) be included in V5?
3. Agent teams vs sub-agents for cross-task review — depends on experimental feature stability
4. How to define the boundary between "light polish during implementation" and "full review after" (Option A from v4-implementation-review.md §6.4)?
5. Is the project-local Go skill sufficient, or should golang-pro be replaced entirely with a more comprehensive skill?
