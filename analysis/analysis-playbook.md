# Implementation Analysis Playbook

Reusable instructions for running a deep task-by-task code comparison of tick-core implementations. Hand this file to a Claude agent to execute the full analysis.

---

## Prerequisites

1. All implementation branches must be complete (23 tasks each)
2. Worktrees must be set up for each version being compared
3. The `analysis/` directory must exist with `task-reports/` and `phase-reports/` subdirs
4. Existing reports from prior rounds (round-0, round-1, round-2) do NOT need to be regenerated — they are stable baselines
5. Identify all project skills used during implementation (check `docs/workflow/implementation/{topic}.md` for the `project_skills` field and read each skill file)

## Setup

```bash
# Create output directories for this round
# Increment the round number for each new analysis (round-1 = V1/V2/V3, round-2 = V4 vs V2, round-3 = V5 vs V4, round-4 = V6 vs V5)
ROUND="round-4"
mkdir -p /Users/leeovery/Code/tick/analysis/$ROUND/task-reports
mkdir -p /Users/leeovery/Code/tick/analysis/$ROUND/phase-reports

# Create worktrees (adjust branch names for new versions)
SCRATCHPAD="/private/tmp/tick-analysis-worktrees"
mkdir -p "$SCRATCHPAD"

# Baseline (V5 — the current best)
git worktree add "$SCRATCHPAD/v5" implementation-v5 2>/dev/null || echo "v5 exists"

# New version under test (change branch name as needed)
git worktree add "$SCRATCHPAD/v6" implementation-v6 2>/dev/null || echo "v6 exists"
```

**Verify**: `ls $SCRATCHPAD/v5/internal/task/task.go $SCRATCHPAD/v6/internal/task/task.go`

## Permissions

Add these to `.claude/settings.local.json` under `permissions.allow`:
```json
"Bash(git -C /Users/leeovery/Code/tick show:*)",
"Bash(git -C /Users/leeovery/Code/tick diff:*)",
"Bash(git -C /Users/leeovery/Code/tick log:*)",
"Bash(wc:*)",
"Read(/private/tmp/tick-analysis-worktrees/*)",
"Write(/Users/leeovery/Code/tick/analysis/*)",
"Glob(/private/tmp/tick-analysis-worktrees/*)",
"Glob(/Users/leeovery/Code/tick/analysis/*)",
"Grep(/private/tmp/tick-analysis-worktrees/*)"
```

---

## Design Principle: No Summarization Loss

Each layer does its OWN code analysis at the appropriate scope:

| Layer | Reads Code? | Reads Lower Reports? | Unique Perspective |
|-------|-------------|---------------------|-------------------|
| Task reports | Yes (diffs + files) | N/A | Per-task: acceptance criteria, specific functions, test cases |
| Phase reports | Yes (all files for that phase) | Yes (task reports) | Cross-task: architecture patterns, shared code, DRY, integration |
| Final synthesis | Yes (selectively) | Yes (task + phase reports) | Cross-phase: overall architecture, version-level patterns |

---

## Step 1: Task-Level Analysis (23 agents, 8 batches of 3)

Run 3 agents in parallel per batch. Wait for all 3 to complete before starting the next batch.

### Agent Prompt Template

For each task, launch a `general-purpose` subagent with this prompt (fill in the placeholders):

```
You are a Go code reviewer producing an EXHAUSTIVE comparison of two implementations
of the same task specification. Your report must be detailed enough that a reader
can understand exactly what each version did without reading the code themselves.

## Project skills (read FIRST)
Read the skill files that were active during implementation. These define
MUST DO / MUST NOT DO constraints that carry equal weight to spec acceptance criteria.
{List skill file paths, e.g.:}
- /Users/leeovery/Code/tick/.claude/skills/golang-pro/SKILL.md
{If a project-local Go skill exists, include it too}

## Task plan file
Read: /Users/leeovery/Code/tick/docs/workflow/planning/tick-core/tick-core-{P}-{T}.md

## Get the diffs (run via Bash)
- V5: git -C /Users/leeovery/Code/tick show {V5_COMMIT} -- ':!.claude'
- V6: git -C /Users/leeovery/Code/tick show {V6_COMMIT} -- ':!.claude'

## Read resulting source files
After getting diffs, read the FULL source files created/modified by this task:
- V5: /private/tmp/tick-analysis-worktrees/v5/{relevant files}
- V6: /private/tmp/tick-analysis-worktrees/v6/{relevant files}

Read BOTH implementation AND test files for each version.

## Write your report
Write to: /Users/leeovery/Code/tick/analysis/$ROUND/task-reports/tick-core-{P}-{T}.md

Use this structure:

# Task tick-core-{P}-{T}: {Task Name}

## Task Summary
{What this task requires — from the plan file. Include all acceptance criteria.}

## Acceptance Criteria Compliance
| Criterion | V5 | V6 |
|-----------|-----|-----|
{One row per criterion. PASS/FAIL/PARTIAL with specific evidence.}

## Implementation Comparison

### Approach
{Detailed comparison of HOW each version solved this.
 Include code quotes showing key structural differences.
 Explicitly call out what's merely different vs genuinely better/worse.}

### Code Quality
{Go idioms, naming, error handling, DRY, type safety.
 QUOTE specific code from each version.
 Reference specific files and line numbers.}

### Test Quality
{For EACH version:
 - List every test function name
 - List every edge case tested
 - Note table-driven vs individual subtests
 - Note assertion quality

 Then: DIFF of test coverage between versions.}

### Skill Compliance
{Check each version against the MUST DO and MUST NOT DO constraints from the
 project's injected skills (e.g. golang-pro). For each constraint that applies
 to this task's code, note whether each version complies.

 Example constraints to check (adjust per skill):
 - Error wrapping with fmt.Errorf("%w", err)
 - Table-driven tests with subtests
 - Explicit error handling (no ignored errors)
 - Exported function documentation
 - Any other MUST DO / MUST NOT DO from the skill file

 | Constraint | V5 | V6 |
 |------------|-----|-----|
 {One row per applicable constraint. PASS/FAIL with evidence.}}

### Spec-vs-Convention Conflicts
{Identify places where the task specification conflicts with language idioms
 or skill constraints. For each conflict found:
 - What the spec says
 - What the language convention / skill requires
 - What each version chose to do
 - Assessment: was the choice a reasonable judgment call?

 This section prevents penalizing implementations for intelligent spec deviations
 that follow language best practices (e.g. lowercase error messages in Go when
 the spec shows capitalized ones).

 If no conflicts exist for this task, state "No spec-vs-convention conflicts identified."}

## Diff Stats
| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | | |
| Lines added | | |
| Impl LOC | | |
| Test LOC | | |
| Test functions | | |

## Verdict
{Which version implemented this task better and why.
 Must be backed by specific evidence from the sections above.}

## CRITICAL GUIDELINES
- Be EXHAUSTIVE. QUOTE code when comparing approaches.
- List EVERY test function and edge case per version.
- Identify test gaps between versions.
- Distinguish "genuinely better" from "different but equivalent".
- Pure code analysis. Reference specific function names, line numbers, file paths.
- Read the project skill files BEFORE analysing code. Skill constraints carry
  equal weight to spec acceptance criteria when judging implementation quality.
- When spec and language convention conflict, do NOT automatically credit
  spec-verbatim compliance. Assess whether deviating was the right call.
- Do NOT create any git commits or temporary files.
```

### Batch Schedule

| Batch | Tasks | Agents |
|-------|-------|--------|
| 1 | tick-core-1-1, tick-core-1-2, tick-core-1-3 | 3 |
| 2 | tick-core-1-4, tick-core-1-5, tick-core-1-6 | 3 |
| 3 | tick-core-1-7, tick-core-2-1, tick-core-2-2 | 3 |
| 4 | tick-core-2-3, tick-core-3-1, tick-core-3-2 | 3 |
| 5 | tick-core-3-3, tick-core-3-4, tick-core-3-5 | 3 |
| 6 | tick-core-4-1, tick-core-4-2, tick-core-4-3 | 3 |
| 7 | tick-core-4-4, tick-core-4-5, tick-core-4-6 | 3 |
| 8 | tick-core-5-1, tick-core-5-2 | 2 |

**After each batch**: Verify all reports were written before proceeding.

### Prior Baseline Commit Mappings (for reference)

<details>
<summary>V4 Commits</summary>

| Task | Commit |
|------|--------|
| 1-1 | e6443c9 |
| 1-2 | eab0e0d |
| 1-3 | 003b167 |
| 1-4 | 4292323 |
| 1-5 | e8967ef |
| 1-6 | 8216ffc |
| 1-7 | de898f4 |
| 2-1 | d964047 |
| 2-2 | 40beac2 |
| 2-3 | 332939d |
| 3-1 | 0bb85cc |
| 3-2 | 95c0230 |
| 3-3 | 37e69a6 |
| 3-4 | 957b234 |
| 3-5 | 7665c68 |
| 4-1 | 9171403 |
| 4-2 | 67805bd |
| 4-3 | 16ace67 |
| 4-4 | 1a0f941 |
| 4-5 | 5a26a04 |
| 4-6 | e0d31d7 |
| 5-1 | 89b8fd5 |
| 5-2 | 5b27694 |

Additional V4 commits (not per-task):
| Description | Commit |
|-------------|--------|
| Pre-polish checkpoint | ca0ac05 |
| Polish | 199e407 |
| Complete implementation | cbcbfcb |

</details>

<details>
<summary>V2 Commits</summary>

| Task | Commit |
|------|--------|
| 1-1 | 1682a27 |
| 1-2 | 01eeace |
| 1-3 | 09fd787 |
| 1-4 | 352e9d8 |
| 1-5 | c6c6051 |
| 1-6 | 4bf12cf |
| 1-7 | 0326beb |
| 2-1 | a3f0065 |
| 2-2 | 6a43086 |
| 2-3 | 477fe0e |
| 3-1 | e1acede |
| 3-2 | ce6e70a |
| 3-3 | 819aa39 |
| 3-4 | b97f76f |
| 3-5 | 5d325d7 |
| 4-1 | e10ae58 |
| 4-2 | 8f59ce4 |
| 4-3 | 6fc05e2 |
| 4-4 | ef16165 |
| 4-5 | 82c22b4 |
| 4-6 | 424d653 |
| 5-1 | 3055def |
| 5-2 | 7936ab2 |

</details>

### V5 Commit Mapping (Baseline — stable)

| Task | Commit |
|------|--------|
| 1-1 | f6dc11c |
| 1-2 | fa01548 |
| 1-3 | 54ee1b6 |
| 1-4 | 2f697c6 |
| 1-5 | b0f9e25 |
| 1-6 | ad45774 |
| 1-7 | 533ee60 |
| 2-1 | 8573cbf |
| 2-2 | 78495c7 |
| 2-3 | b513b8d |
| 3-1 | 8329ca5 |
| 3-2 | fca9a1f |
| 3-3 | 6fbacef |
| 3-4 | c66ead3 |
| 3-5 | f395505 |
| 3-6 | 730feed |
| 4-1 | 93d777a |
| 4-2 | 7784a71 |
| 4-3 | f13bb9c |
| 4-4 | 8c0ec68 |
| 4-5 | 836b86c |
| 4-6 | 052d542 |
| 5-1 | fa3fa56 |
| 5-2 | 90cad53 |

Additional V5 commits (not per-task):
| Description | Commit |
|-------------|--------|
| Pre-analysis checkpoint | 98eff08 |
| Analysis phase 6 added | 672432e |
| T6-1: Dependency validation gaps | d9df9f8 |
| T6-2: Shared ready/blocked SQL | 41e0947 |
| T6-3: Consolidate JSONL parsing | 245f4cc |
| T6-4: Shared formatter methods | bba1728 |
| T6-5: Remove doctor from help | fc0f395 |
| T6-6: Remove dead StubFormatter | 9a1ae3d |
| T6-7: Type-safe formatter params | 552624d |
| Pre-analysis checkpoint 2 | 3fefed0 |
| Complete implementation | 3d9ca32 |
| Restructure analysis files | 5c6b457 |

### V6 Commit Mapping

_(Fill in after V6 implementation by running: `git log implementation-v6 --oneline | grep "impl(tick-core)"`)_

---

## Step 2: Phase-Level Analysis (5 agents, 2 batches)

Phase agents read task reports AND source code to find cross-task patterns.

### Phase Agent Prompt Template

```
You are analysing Phase {N} ({Phase Name}) across two implementations of a Go task tracker.

Your job is to find CROSS-TASK patterns that individual task analyses miss.

## Read task reports first
{list all task report files for this phase — use the round-4 comparison reports}

## Read the phase description
/Users/leeovery/Code/tick/docs/workflow/planning/tick-core.md (Phase {N} section)

## Read the source files for this phase
- V5: /private/tmp/tick-analysis-worktrees/v5/{files for this phase}
- V6: /private/tmp/tick-analysis-worktrees/v6/{files for this phase}

Use Glob to discover the actual file layout first.

## Write to
/Users/leeovery/Code/tick/analysis/$ROUND/phase-reports/phase-{N}.md

## Structure

# Phase {N}: {Phase Name}

## Task Scorecard
| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|

## Cross-Task Architecture Analysis
{How tasks compose together, shared helpers, DRY across tasks.
 QUOTE code showing cross-task patterns.}

## Code Quality Patterns
{Naming, error handling, DRY, Go idioms across the phase.}

## Test Coverage Analysis
{Aggregate counts, edge case patterns, testing approach differences.}

## Phase Verdict
{Overall winner with thorough reasoning.}

## CRITICAL: Your unique value
- How functions work TOGETHER across tasks
- Cross-task patterns (shared helpers, SQL fragments, error styles)
- Architecture decisions visible only at the phase level
- Don't repeat task reports — ADD to them.
- Do NOT create any git commits or temporary files.
```

### Batch Schedule

| Batch | Phases |
|-------|--------|
| 9 | Phase 1, Phase 2, Phase 3 |
| 10 | Phase 4, Phase 5 |

### Phase-to-File Mapping

**Phase 1** (Walking Skeleton): task/, storage/, cli.go/app.go, create.go, list.go, show.go, init.go
**Phase 2** (Task Lifecycle): task/transition.go, cli/transition.go, cli/update.go
**Phase 3** (Dependencies): task/dependency.go, cli/dep.go, cli/ready.go, cli/blocked.go, cli/list.go (filters)
**Phase 4** (Output Formats): cli/format*.go, cli/formatter*.go, cli/*_formatter.go, cli/verbose.go
**Phase 5** (Stats & Cache): cli/stats.go, cli/rebuild.go

---

## Step 3: Final Synthesis (1 agent)

### Prompt

```
You are producing the definitive synthesis comparing V6 against V5 (the current best
implementation) of a Go task tracker.

You have access to:
- 23 V6-vs-V5 task reports in /Users/leeovery/Code/tick/analysis/$ROUND/task-reports/
- 5 V6-vs-V5 phase reports in /Users/leeovery/Code/tick/analysis/$ROUND/phase-reports/
- Prior round syntheses at /Users/leeovery/Code/tick/analysis/round-{1,2,3}/final-synthesis.md
- The analysis log at /Users/leeovery/Code/tick/analysis/analysis-log.md
- Source code at /private/tmp/tick-analysis-worktrees/{v5,v6}/

Read ALL phase reports first. Then scan each task report's Verdict section.
If you need to verify a claim, read the source code.

Write to: /Users/leeovery/Code/tick/analysis/$ROUND/final-synthesis.md

## Structure

# V6 vs V5 Synthesis

## Executive Summary
{Did V6 match, exceed, or fall short of V5? Back with evidence.}

## Phase-by-Phase Results
| Phase | Winner | Margin | Key Factor |

## Full Task Scorecard
| Task | V5 | V6 | Winner |

## What V6 Did Better Than V5
{Specific improvements with evidence}

## What V6 Did Worse Than V5
{Specific regressions with evidence}

## What Stayed the Same
{Patterns that both versions share}

## Did the Workflow Changes Work?
{Direct assessment of V6's workflow changes against V5. Evaluate whether each change
 achieved its goal and whether any had unintended side effects.}

## Recommendations
{What to change next based on V6 results.}

Do NOT create any git commits or temporary files.
```

---

## Step 4: Update Tracking Files

After the final synthesis is written, update these files to reflect the completed round:

1. **`analysis/index.md`** — Add the new round section with:
   - Status and date
   - Link to final synthesis
   - Full task report table with winners
   - Phase report table with winners
   - Update Key Findings Summary if results change the overall narrative

2. **`analysis/analysis-log.md`** — Update:
   - Current State section (date, what was done)
   - Key Findings (add new round results)
   - Version Inventory table (add new version row)
   - Completed round section (move from "Planned" to "Completed" with results)
   - Planned Next Step (update based on synthesis recommendations)

3. **`analysis/analysis-playbook.md`** — If applicable:
   - Fill in any pending commit mappings
   - Add methodology improvements discovered during this round

**This step is mandatory.** Do not consider the analysis complete until tracking files are updated.

---

## Execution Summary

| Layer | Agents | Batches | Output |
|-------|--------|---------|--------|
| Task analysis | 23 | 8 | 23 task reports in `$ROUND/task-reports/` |
| Phase analysis | 5 | 2 | 5 phase reports in `$ROUND/phase-reports/` |
| Final synthesis | 1 | 1 | `$ROUND/final-synthesis.md` |
| Tracking updates | — | — | `index.md`, `analysis-log.md`, `analysis-playbook.md` |
| **Total** | **29** | **11+1** | **29 files + tracking updates** |

---

## Notes for the Executing Agent

- Run 3 agents max in parallel per batch
- Wait for all agents in a batch to complete before starting the next
- Verify each report was written before proceeding
- If an agent has permission issues reading worktree files, check `.claude/settings.local.json`
- Each round gets its own directory (`round-2/`, `round-3/`, etc.) — no filename suffixes needed
- The V4 and V5 commit mappings are stable and don't change between analysis runs
- Fill in the V6 commit mapping by running: `git log implementation-v6 --oneline | grep "impl(tick-core)"`
- Update the `$ROUND` variable in setup and all path references for each new round
- For round 4, check the analysis log for any V6-specific evaluation criteria
