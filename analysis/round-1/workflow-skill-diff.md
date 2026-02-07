# Workflow Skill Diff: v2.1.3 (V2 run) vs v2.1.5 (V3 run)

## Version Summary

| Dimension | V2 run (v2.1.2/v2.1.3) | V3 run (v2.1.5) |
|-----------|------------------------|-----------------|
| Plugin version at start | 2.1.2 | 2.1.5 |
| Executor agent | Yes | Yes (changed) |
| Reviewer agent | Yes | Yes (changed) |
| Polish agent | **No** | **Yes (new)** |
| Integration context file | **No** | **Yes (new)** |
| Commands-to-skills migration | Not yet | Completed (PR #76) |
| Fix gate mode | Not present | Present |
| Codebase cohesion review dimension | Not present | Present (6th dimension) |
| Reviewer fix analysis | Not present | Present |

---

## 1. Executor Agent Changes (PR #77, #79)

The executor agent is the file that directly controls what instructions the code-writing agent receives. These changes are the most likely cause of quality differences.

### New in V3's executor (not present in V2's):

#### A. Plan file path as input (PR #79 — holistic-design-gap)

V2 executor received 5 inputs:
1. tdd-workflow.md path
2. code-quality.md path
3. Specification path
4. Project skill paths
5. Task content

V3 executor receives 7 inputs (two new):
6. **Plan file path** — "The implementation plan, for understanding the overall task landscape and where this task fits"
7. **Integration context file path** — "Accumulated notes from prior tasks about established patterns, conventions, and decisions"

**Impact assessment**: The plan file gives the executor awareness of the broader task landscape. The integration context file provides accumulated knowledge from prior tasks. In theory this should improve cohesion. However, the executor also receives the instruction "Use this for awareness, not to build ahead (YAGNI still applies)", which mitigates over-engineering risk.

#### B. Radically expanded "Explore codebase" step (PR #79)

V2's step 5 was brief:
```
5. **Explore codebase** — understand what exists before writing anything:
   - Read files and tests related to the task's domain
   - Identify patterns, conventions, and structures you'll need to follow or extend
   - Check for existing code that the task builds on or integrates with
```

V3's step 5 is dramatically expanded:
```
5. **Explore codebase** — you are weaving into an existing canvas, not creating isolated patches:
   - If an integration context file was provided, read it first — identify helpers,
     patterns, and conventions you must reuse before writing anything new
   - Skim the plan file to understand the task landscape — what's been built,
     what's coming, where your task fits. Use this for awareness, not to build
     ahead (YAGNI still applies)
   - Read files and tests related to the task's domain
   - Search for existing helpers, utilities, and abstractions that solve similar
     problems — reuse, don't duplicate
   - When creating an interface or API boundary, read BOTH sides: the code that
     will consume it AND the code that will implement it
   - Match conventions established in the codebase: error message style, naming
     patterns, file organisation, type placement
   - Your code should read as if the same developer wrote the entire codebase
```

**Key new instructions for V3's executor**:
- "you are weaving into an existing canvas, not creating isolated patches"
- "Search for existing helpers, utilities, and abstractions... reuse, don't duplicate"
- "When creating an interface or API boundary, read BOTH sides"
- "Match conventions established in the codebase"
- "Your code should read as if the same developer wrote the entire codebase"

**Impact assessment**: These instructions directly target cross-task cohesion. They tell the executor to actively look for existing patterns before writing new code, and to maintain consistency. This is a significant behavioral change.

#### C. Statelessness declaration (PR #77)

V3 adds an explicit statement absent from V2:
```
You are stateless — each invocation starts fresh. The full task content is always
provided so you can see what was asked, what was done, and what needs fixing.
```

**Impact assessment**: This clarifies the executor's mental model but doesn't change behavior directly.

#### D. INTEGRATION_NOTES output (PR #79)

V2's output format:
```
STATUS: complete | blocked | failed
TASK: {task name}
SUMMARY: {what was done}
FILES_CHANGED: {list}
TESTS_WRITTEN: {list}
TEST_RESULTS: {results}
ISSUES: {concerns}
```

V3's output adds:
```
INTEGRATION_NOTES:
- {3-5 concise bullet points: key patterns, helpers, conventions, interface
  decisions established by this task. Anchor to concrete file paths where
  applicable}
```

**Impact assessment**: This creates the integration context that subsequent tasks consume. The executor must now document what it established, which feeds forward to future executor invocations.

#### E. Re-invocation with full task content (PR #77)

V2's re-invocation (after review feedback) passed:
- User-approved review notes
- Specific issues to address

V3's re-invocation passes all 7 inputs **plus** review notes:
- All original context (task content, plan path, integration context, etc.)
- User-approved review notes
- Specific issues to address

And the task-loop.md reinforces: "re-invoke the executor with the **full task content** and the reviewer's notes"

**Impact assessment**: V2's executor on re-attempt may have had degraded context. V3 ensures full context is always present. This directly affects fix quality.

---

## 2. Reviewer Agent Changes (PR #78 — review-fix-analysis)

### New in V3's reviewer:

#### A. 6th review dimension: Codebase Cohesion

V2 reviewed 5 dimensions:
1. Spec Conformance
2. Acceptance Criteria
3. Test Adequacy
4. Convention Adherence
5. Architectural Quality

V3 adds a 6th:
6. **Codebase Cohesion** — does new code integrate with existing code?

Specific checks added:
- Duplicated logic that should be extracted
- Existing helpers/patterns being reused
- Naming convention consistency
- Error message convention consistency
- Concrete types vs generic/any where possible
- Related types co-located with interfaces

**Impact assessment**: This is directly relevant to the quality differences observed. V2's analysis found that V2 excelled at cross-task integration (reusing SQL fragments, extracting unwrapMutationError, etc.) while V3 showed drift (mixed error styles, bare returns). The cohesion dimension in V3's reviewer should have caught these issues — but perhaps the instructions weren't calibrated enough, or the executor's exploration wasn't deep enough to act on the feedback.

#### B. Fix recommendations with confidence

V2's reviewer returned issues as bare observations:
```
ISSUES:
- {specific issue with file:line reference}
```

V3's reviewer provides fix analysis:
```
ISSUES:
- {specific issue with file:line reference}
  FIX: {recommended approach}
  ALTERNATIVE: {other valid approach — optional}
  CONFIDENCE: {high | medium | low}
```

**Impact assessment**: This gives the executor concrete actionable guidance when fixing issues, rather than just knowing something is wrong.

#### C. Integration context input

V3's reviewer receives the integration context file and outputs COHESION_NOTES that feed back into the context file.

#### D. COHESION_NOTES output

V3's reviewer produces cohesion notes that are appended to the integration context file, creating a growing knowledge base of established patterns.

---

## 3. Polish Agent (New in V3 — PR #80)

The polish agent (`implementation-polish.md`) is entirely new in v2.1.5. It runs after all tasks complete and performs holistic quality analysis.

### What it does:

1. **Absorbs full context** — reads spec, plan, all project skills, integration context
2. **Identifies implementation scope** — finds all files changed during implementation via git
3. **Runs 2-5 discovery-fix cycles**, each with:
   - 3 parallel analysis sub-agents:
     - **Code Cleanup**: unused code, naming quality, formatting drift
     - **Structural Cohesion**: duplicated logic, over/under-engineering
     - **Cross-Task Integration**: workflow seams, interface mismatches, integration test gaps
   - Optional dynamic analysis passes based on findings
   - Synthesis and prioritization
   - Executor invocation for fixes (with reviewer verification)

### Hard rules:
- No direct code changes (dispatches executor)
- No new features
- Plan scope only
- Existing tests are protected
- Minimum 2 cycles

### V3's polish commit

The V3 branch has a commit `dcdb7a8 impl(tick-core): polish — fix missing dep validation, remove dead code, DRY helpers` — confirming the polish agent ran and produced changes.

**Impact assessment**: The polish agent represents a significant quality improvement mechanism. It specifically targets cross-task issues that individual task executors miss. However, the existence of V3's quality gaps (string timestamps, bare error returns, CLI-level verbose logging) suggests the polish agent either:
- Did not catch these as issues (they were established in task 1-1 and propagated as "conventions")
- Caught them but classified them as out of scope (architectural decisions, not bugs)
- Was limited by its "no new features" and "plan scope only" constraints

---

## 4. Orchestrating Skill Changes (PR #76, #79)

### Commands-to-skills migration (PR #76)

V2's plugin had `commands/` entries. V3's plugin replaces these with additional `skills/` entries:
- `skills/link-dependencies`
- `skills/migrate`
- `skills/start-discussion`
- `skills/start-feature`
- `skills/start-implementation`
- `skills/start-planning`
- `skills/start-research`
- `skills/start-review`
- `skills/start-specification`
- `skills/status`
- `skills/view-plan`

**Impact assessment**: These are convenience entry points. The core implementation skill (`skills/technical-implementation`) is unchanged in structure. This change does not directly affect code quality.

### Integration context accumulation (PR #79)

The task-loop.md gained a new step in "E. Update Progress and Commit":

```
**Append integration context** — extract INTEGRATION_NOTES from the executor's
final output and COHESION_NOTES from the reviewer's output. Append both to
`docs/workflow/implementation/{topic}-context.md`
```

V2's task loop had no equivalent. The integration context mechanism is entirely new.

### Fix gate mode (PR #77/78)

V3's tracking file gains:
- `fix_gate_mode: gated` (alongside existing `task_gate_mode`)
- `fix_attempts: 0` counter

The Review Changes flow changed significantly:

V2's flow: always present to user, user chooses yes/skip/comment.

V3's flow: if `fix_gate_mode: auto` and `fix_attempts < 3`, automatically re-invoke executor without user intervention. Escalate to user only if gated or after 3 failed attempts.

**Impact assessment**: This allows reviewer-flagged issues to be auto-fixed without user gates, speeding up the fix loop. If the user opted into `auto` fix mode during V3's run, the executor could have been repeatedly fixing issues without human review — which could either improve quality (more iteration) or degrade it (auto-pilot without oversight).

---

## 5. Analysis: Which Changes Explain the Quality Differences?

### The paradox

V3's workflow has **more** quality-focused instructions than V2's:
- Integration context file for cross-task awareness
- Expanded "Explore codebase" instructions emphasizing cohesion
- 6th review dimension (Codebase Cohesion)
- Fix analysis with concrete recommendations
- Polish agent for holistic quality
- Auto-fix loop for faster iteration

Yet V2 produced **better** code. How?

### Hypothesis 1: Early architectural decisions precede the integration context

V3's integration context mechanism only helps from task 2 onward. Task 1-1 (Task model & ID generation) has no prior context to draw from. The critical decisions that defined V3's trajectory — `string` timestamps, no `NewTask` constructor, `ValidateTitle` that doesn't trim, `int` exit codes — were all made in Phase 1 tasks where the integration context was empty or minimal.

By the time the integration context was rich enough to enforce patterns, V3's architectural patterns were already established. The context file then **reinforced** the early (flawed) decisions rather than correcting them. Evidence: V3's context file for task 1-1 documents `"TrimTitle is separate from ValidateTitle — call TrimTitle first, then ValidateTitle on trimmed result"` and `"Timestamps use time.RFC3339 format (ISO 8601 UTC) — DefaultTimestamps() returns both created and updated as same value"` — these became the "conventions" that later tasks dutifully followed.

### Hypothesis 2: The expanded exploration instructions add cognitive load without sufficient guidance

V3's executor was told to "skim the plan file to understand the task landscape" and "read BOTH sides" of interface boundaries. These are excellent instructions for a senior developer, but they create additional cognitive load for the agent. More reading before coding means:
- The agent may run into context window limits sooner
- The additional context may dilute focus on the current task
- "Awareness" of future tasks could lead to premature design decisions

V2's executor had simpler instructions: read the task's domain, identify patterns, execute TDD. This focused approach may have produced more internally coherent code per task.

### Hypothesis 3: V2 ran on v2.1.2-v2.1.3 which was a more stable/tested configuration

V2's workflow was simpler (no integration context, no polish, no fix gates). Simpler orchestration means fewer moving parts, less chance of the orchestrator failing to properly pass context, and more predictable agent behavior.

V3's workflow had PRs #76-#80 all landing in quick succession. These represent substantial changes to the orchestration logic, any of which could have subtle interaction effects.

### Hypothesis 4: The polish agent cannot fix architectural decisions

The polish agent's constraints are telling:
- "No new features — only improve what exists"
- "Plan scope only"
- "Existing tests are protected"

The string timestamp choice, the CLI-level verbose logging, the string-returning formatters — these are all architectural decisions deeply woven into V3's codebase. The polish agent cannot change `string` to `time.Time` in the Task struct because that would require rewriting every consumer and every test — it's a "new feature" (different type system) not a cleanup. The polish's V3 commit (`polish — fix missing dep validation, remove dead code, DRY helpers`) shows it caught superficial issues but couldn't address the foundational ones.

### Hypothesis 5: V2's executor was better BECAUSE it had less context

V2's executor had no integration context, no plan awareness, no prior-task notes. It approached each task as a fresh problem, making design decisions based solely on:
- The specification
- The current codebase (by exploring it)
- Go idioms from the golang-pro skill
- The task's acceptance criteria

This forced the executor to make locally optimal decisions that aligned with Go conventions by default — because Go conventions were all it had. V3's executor, armed with prior task notes saying "we use string timestamps" and "error returns are bare", dutifully followed those patterns even when they diverged from Go idioms.

### Most likely explanation

**A combination of Hypotheses 1 and 5.** V3's integration context mechanism is a good idea in principle, but it created a "convention lock-in" effect: early design choices (made without prior context) got documented as established patterns, and later tasks followed them faithfully. V2 had no such mechanism, so each task was free to make locally idiomatic choices. Since the Go conventions are well-established, V2's "clean slate per task" approach naturally converged on idiomatic patterns, while V3's "follow prior conventions" approach locked in Phase 1 deviations.

The expanded exploration instructions in V3's executor may have also pushed the agent to read too much before coding, reducing the attention budget for the actual implementation.

---

## 6. Key Diffs Summary

### Executor: V2 vs V3

| Section | V2 (v2.1.2) | V3 (v2.1.5) | Change |
|---------|-------------|-------------|--------|
| Inputs | 5 paths | 7 paths (+plan, +integration context) | More context |
| Explore codebase | 3 bullets, basic | 7 bullets, emphasis on cohesion and reuse | Significantly expanded |
| Stateless declaration | Absent | Present | Clarifies model |
| Re-invocation context | Review notes only | Full task content + review notes | Fuller context |
| Output format | 6 fields | 7 fields (+INTEGRATION_NOTES) | Feeds forward |

### Reviewer: V2 vs V3

| Section | V2 (v2.1.2) | V3 (v2.1.5) | Change |
|---------|-------------|-------------|--------|
| Review dimensions | 5 | 6 (+Codebase Cohesion) | Broader review |
| Fix recommendations | None | FIX + ALTERNATIVE + CONFIDENCE per issue | Actionable guidance |
| Integration context | Not received | Received | More context |
| Output format | Basic issues list | Issues with fix analysis + COHESION_NOTES | Richer output |

### Skill (Orchestrator): V2 vs V3

| Section | V2 (v2.1.2) | V3 (v2.1.5) | Change |
|---------|-------------|-------------|--------|
| Steps | 6 (no polish) | 7 (+Polish step) | Polish added |
| Fix gate mode | Not present | gated/auto with 3-attempt escalation | Faster fix loop |
| Integration context | Not accumulated | Accumulated per task | Cross-task memory |
| Polish agent | Not invoked | Invoked after all tasks | Holistic quality pass |
| Tracking file fields | task_gate_mode only | +fix_gate_mode, +fix_attempts | More state |

### New file: implementation-polish.md

Entirely new agent. Runs 2-5 discovery-fix cycles with 3 parallel analysis sub-agents (code cleanup, structural cohesion, cross-task integration). Dispatches executor for fixes, reviewer for verification. Cannot make architectural changes, only cleanup/cohesion improvements.

---

## 7. Conclusion

The v2.1.3-to-v2.1.5 changes represent a significant investment in cross-task cohesion and holistic quality. Every change aims to make the implementation more integrated and consistent across tasks. Ironically, this may have contributed to V3's quality gaps by:

1. **Locking in early mistakes as conventions** via the integration context file
2. **Adding cognitive overhead** to the executor's exploration phase
3. **Giving the reviewer more dimensions** but not necessarily making the executor better at foundational design
4. **Making the polish agent unable to fix architectural issues** that were established early

V2's simpler workflow — no integration context, no polish, straightforward executor instructions — produced better results because:
- Each task started fresh with Go conventions as the default
- The executor focused on the current task without distraction
- Early decisions happened to be more idiomatic (possibly due to less overhead competing for the agent's attention)
- There was no mechanism to propagate early mistakes as "established patterns"

The key insight: **cross-task memory is a double-edged sword**. When early decisions are good, it amplifies them. When they're bad, it amplifies those too. V2 got lucky (or benefited from simplicity) by making good early choices. V3's context mechanism then faithfully propagated its less-idiomatic early choices across the entire codebase.
