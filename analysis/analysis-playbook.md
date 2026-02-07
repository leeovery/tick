# Implementation Analysis Playbook

Reusable instructions for running a deep task-by-task code comparison of tick-core implementations. Hand this file to a Claude agent to execute the full analysis.

---

## Prerequisites

1. All implementation branches must be complete (23 tasks each)
2. Worktrees must be set up for each version being compared
3. The `analysis/` directory must exist with `task-reports/` and `phase-reports/` subdirs
4. Existing reports from prior versions (V1/V2/V3) do NOT need to be regenerated — they are stable baselines

## Setup

```bash
# Create output directories for this round
# Increment the round number for each new analysis (round-1 = V1/V2/V3, round-2 = V4 vs V2, etc.)
ROUND="round-2"
mkdir -p /Users/leeovery/Code/tick/analysis/$ROUND/task-reports
mkdir -p /Users/leeovery/Code/tick/analysis/$ROUND/phase-reports

# Create worktrees (adjust branch names for new versions)
SCRATCHPAD="/private/tmp/tick-analysis-worktrees"
mkdir -p "$SCRATCHPAD"

# Baseline (V2 — the current best)
git worktree add "$SCRATCHPAD/v2" implementation-take-two 2>/dev/null || echo "v2 exists"

# New version under test (change branch name as needed)
git worktree add "$SCRATCHPAD/v4" implementation-v4 2>/dev/null || echo "v4 exists"
```

**Verify**: `ls $SCRATCHPAD/v2/internal/task/task.go $SCRATCHPAD/v4/internal/task/task.go`

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
"Glob(/Users/leeovery/Code/tick/analysis/*)"
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
You are a Go code reviewer producing an EXHAUSTIVE comparison of 2 implementations
of the same task specification. Your report must be detailed enough that a reader
can understand exactly what each version did without reading the code themselves.

## Task plan file
Read: /Users/leeovery/Code/tick/docs/workflow/planning/tick-core/tick-core-{P}-{T}.md

## Get the diffs (run via Bash)
- V2: git -C /Users/leeovery/Code/tick show {V2_COMMIT} -- ':!.claude'
- V4: git -C /Users/leeovery/Code/tick show {V4_COMMIT} -- ':!.claude'

## Read resulting source files
After getting diffs, read the FULL source files created/modified by this task:
- V2: /private/tmp/tick-analysis-worktrees/v2/{relevant files}
- V4: /private/tmp/tick-analysis-worktrees/v4/{relevant files}

Read BOTH implementation AND test files for each version.

## Write your report
Write to: /Users/leeovery/Code/tick/analysis/$ROUND/task-reports/tick-core-{P}-{T}.md

Use this structure:

# Task tick-core-{P}-{T}: {Task Name}

## Task Summary
{What this task requires — from the plan file. Include all acceptance criteria.}

## Acceptance Criteria Compliance
| Criterion | V2 | V4 |
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

## Diff Stats
| Metric | V2 | V4 |
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

### V2 Commit Mapping (Baseline — stable)

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

### V4 Commit Mapping (Fill in after implementation)

| Task | Commit |
|------|--------|
| 1-1 | {TODO} |
| 1-2 | {TODO} |
| 1-3 | {TODO} |
| 1-4 | {TODO} |
| 1-5 | {TODO} |
| 1-6 | {TODO} |
| 1-7 | {TODO} |
| 2-1 | {TODO} |
| 2-2 | {TODO} |
| 2-3 | {TODO} |
| 3-1 | {TODO} |
| 3-2 | {TODO} |
| 3-3 | {TODO} |
| 3-4 | {TODO} |
| 3-5 | {TODO} |
| 4-1 | {TODO} |
| 4-2 | {TODO} |
| 4-3 | {TODO} |
| 4-4 | {TODO} |
| 4-5 | {TODO} |
| 4-6 | {TODO} |
| 5-1 | {TODO} |
| 5-2 | {TODO} |

---

## Step 2: Phase-Level Analysis (5 agents, 2 batches)

Phase agents read task reports AND source code to find cross-task patterns.

### Phase Agent Prompt Template

```
You are analysing Phase {N} ({Phase Name}) across 2 implementations of a Go task tracker.

Your job is to find CROSS-TASK patterns that individual task analyses miss.

## Read task reports first
{list all task report files for this phase — use the v4 comparison reports}

## Read the phase description
/Users/leeovery/Code/tick/docs/workflow/planning/tick-core.md (Phase {N} section)

## Read the source files for this phase
- V2: /private/tmp/tick-analysis-worktrees/v2/{files for this phase}
- V4: /private/tmp/tick-analysis-worktrees/v4/{files for this phase}

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
You are producing the definitive synthesis comparing V4 against V2 (the current best
implementation) of a Go task tracker.

You have access to:
- 23 V4-vs-V2 task reports in /Users/leeovery/Code/tick/analysis/$ROUND/task-reports/
- 5 V4-vs-V2 phase reports in /Users/leeovery/Code/tick/analysis/$ROUND/phase-reports/
- The original V1/V2/V3 synthesis at /Users/leeovery/Code/tick/analysis/round-1/final-synthesis.md
- The analysis log at /Users/leeovery/Code/tick/analysis/analysis-log.md
- Source code at /private/tmp/tick-analysis-worktrees/{v2,v4}/

Read ALL phase reports first. Then scan each task report's Verdict section.
If you need to verify a claim, read the source code.

Write to: /Users/leeovery/Code/tick/analysis/$ROUND/final-synthesis.md

## Structure

# V4 vs V2 Synthesis

## Executive Summary
{Did V4 match, exceed, or fall short of V2? Back with evidence.}

## Phase-by-Phase Results
| Phase | Winner | Margin | Key Factor |

## Full Task Scorecard
| Task | V2 | V4 | Winner |

## What V4 Did Better Than V2
{Specific improvements with evidence}

## What V4 Did Worse Than V2
{Specific regressions with evidence}

## What Stayed the Same
{Patterns that both versions share}

## Did the Workflow Changes Work?
{Direct assessment: did removing PR #79's integration context / adding polish / etc
 produce the expected improvement? Reference the analysis-log.md predictions.}

## Recommendations
{What to change next based on V4 results.}

Do NOT create any git commits or temporary files.
```

---

## Execution Summary

| Layer | Agents | Batches | Output |
|-------|--------|---------|--------|
| Task analysis | 23 | 8 | 23 task reports in `$ROUND/task-reports/` |
| Phase analysis | 5 | 2 | 5 phase reports in `$ROUND/phase-reports/` |
| Final synthesis | 1 | 1 | `$ROUND/final-synthesis.md` |
| **Total** | **29** | **11** | **29 files** |

---

## Notes for the Executing Agent

- Run 3 agents max in parallel per batch
- Wait for all agents in a batch to complete before starting the next
- Verify each report was written before proceeding
- If an agent has permission issues reading worktree files, check `.claude/settings.local.json`
- Each round gets its own directory (`round-2/`, `round-3/`, etc.) — no filename suffixes needed
- The V2 commit mapping is stable and doesn't change between analysis runs
- Fill in the V4 commit mapping by running: `git log implementation-v4 --oneline | grep "impl(tick-core)"`
- Update the `$ROUND` variable in setup and all path references for each new round
