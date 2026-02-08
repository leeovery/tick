# V4 Implementation Review — Findings & Recommendations for V5

Analysis of what V4 did well, what it missed, and what workflow changes could close the gap for V5.

---

## Context

V4 was implemented using the `claude-technical-workflows` implementation skill with V2-base workflow + PRs #77/#78/#80 (PR #79 removed). It was analysed against V2 (the previous best) in round 2 of the implementation analysis.

**Result**: V4 wins 15/23 tasks, all 5 phases. V4 is the new baseline.

**Full analysis**: See `analysis/round-2/final-synthesis.md` for the complete comparison.

---

## 1. What V4 Got Right

These patterns should be preserved in V5:

- **Type-safe formatter interface** — concrete `StatsData` instead of `interface{}`
- **Single-source SQL reuse** — `readyConditionsFor(alias)` used across 4 files
- **Eliminated intermediate structs** — no `taskJSON` bridge layer (84 fewer lines)
- **In-memory hash computation** — `SerializeJSONL` avoids post-write re-read
- **Pointer-based flag optionality** — uniform `*string`/`*int` for update flags
- **Type-safe test infrastructure** — `task.Task` structs for setup, `task.ReadJSONL` for assertions
- **Per-concern test functions** — `TestCreate_PriorityFlag` pattern for targeted `go test -run`
- **Dynamic alignment** — PrettyFormatter scales to 100+ task counts
- **Dedicated files per concern** — `transition.go`, `dependency.go` instead of cramming into `task.go`

---

## 2. What V4 Missed (V2 Did Better)

These are specific gaps to close in V5:

### 2.1 No integration tests

V2 has 6 binary-level integration tests in `main_test.go` (168 LOC) that build the actual `tick` binary via `exec.Command("go", "build", ...)` and verify:
- Real exit codes
- Real stderr/stdout separation
- Complete `Error:` prefix pipeline

V4 has zero. This is the single most important gap.

### 2.2 Weak defensive ID normalization

V2 normalizes both sides of every ID comparison: `task.NormalizeID(tasks[i].ID) == id`. V4 normalizes only input and trusts stored IDs are lowercase. V2's approach handles manually-edited JSONL files.

### 2.3 Substring test assertions for formatters

V4 uses `strings.Contains` for formatter output verification. V2 uses exact string matching. V2's approach catches formatting regressions (extra whitespace, wrong newlines) that V4 would miss.

### 2.4 Missing compile-time interface checks

V2 includes `var _ Formatter = &PrettyFormatter{}` in implementation files. Standard Go idiom, zero cost, catches interface drift at compile time. V4 relies on test-time checks only.

### 2.5 Inconsistent quiet-mode handling

V4 pushes `quiet bool` into some `Formatter` methods but handles it in the command handler for others. V2 is consistent: quiet is always checked in the handler, formatters never invoked in quiet mode.

### 2.6 Write error propagation

V2 consistently captures and returns `fmt.Fprintf` write errors in formatters. V4 discards them (`fmt.Fprintf(w, ...); return nil`).

### 2.7 Edge-case test gaps

V4 misses these spec-mandated or architecturally important edge cases that V2 tests:
- All 9 invalid transitions for no-mutation (V4 tests only 1 of 9)
- Stale ref removal via `dep rm`
- `dep rm` atomic write persistence
- Unknown dep subcommand handling
- `tick list --ready` alias verification
- Contradictory filter edge case (`--status done --ready`)
- Dependency preservation during rebuild
- Mutation verbose output

### 2.8 EnsureFresh API design

V2's `EnsureFresh` returns `*Cache` for reuse. V4's closes the connection, forcing a double-open per operation.

---

## 3. Cross-Task Duplication (Neither Version Fixed)

Both V2 and V4 have these duplicated patterns that no implementation extracted into helpers:

### 3.1 "Find task by index" loop — 5 copies

```go
idx := -1
for i := range tasks {
    if tasks[i].ID == id {
        idx = i
        break
    }
}
```

Appears in: transition.go, update.go, dep.go (add path), dep.go (rm path), and at least one more.

### 3.2 "Discover + open + defer close" preamble — 9 copies

```go
tickDir, err := DiscoverTickDir(a.Dir)
if err != nil { return err }
store := storage.NewStore(tickDir, a.Verbose)
defer store.Close()
```

Appears in every command handler. Could be `a.withStore(func(store *Store) error { ... })`.

### 3.3 Lock acquisition ceremony — 3 copies

Full `flock.New()`, `context.WithTimeout()`, `TryLockContext()` sequence duplicated in `Mutate`, `Query`, and `Rebuild`/`ForceRebuild`. Could be `withExclusiveLock`/`withSharedLock` helpers.

---

## 4. Analysis Methodology Gap — Skill Compliance

Round 2 analysis didn't evaluate compliance with the `golang-pro` skill injected during implementation. Key findings:

### 4.1 Error message casing

The spec uses capitalized error messages (`"Error: Cannot add dependency..."`). Go convention (and the golang-pro skill's implicit standards) requires lowercase error strings. V4 correctly used lowercase. Several task verdicts credited V2 for "spec-verbatim" messages when V4 made the right language-idiomatic choice.

### 4.2 Error wrapping

golang-pro MUST DO: `Propagate errors with fmt.Errorf("%w", err)`. V4 does this consistently. V2 often returns bare errors. This wasn't weighted in the analysis.

### 4.3 Table-driven tests

golang-pro MUST DO: `Write table-driven tests with subtests`. V4 follows this more consistently than V2.

### 4.4 Playbook updated

The analysis playbook now includes two new dimensions for future rounds:
- **Skill Compliance** — check MUST DO/MUST NOT DO constraints from injected skills
- **Spec-vs-Convention Conflicts** — when spec and language idiom conflict, assess whether the implementation made a reasonable judgment call

---

## 5. Why the Polish Agent Didn't Catch These Issues

### 5.1 The polish agent's instructions are correct

Sub-agent 2 (Structural Cohesion) explicitly asks for:
- "duplicated logic across task boundaries that should be extracted"
- "design patterns that are now obvious with the full picture"

Sub-agent 3 (Cross-Task Integration) asks for:
- "gaps in integration test coverage for cross-task workflows"

### 5.2 But the framing and context are insufficient

Three factors explain why the same model with the right instructions still missed issues:

**No external reference standard**: The analysis agents had the golang-pro skill, the spec, and a second implementation to compare against. Polish sub-agents get only "the list of implementation files." Without an external standard for what "ideal" looks like, patterns that appear 5 times may look intentional.

**No adversarial framing**: Analysis asked "which is better and WHY?" — forces finding fault. Polish asks "improve what exists" — constructive framing biased toward accepting status quo.

**Sub-agent scope too broad**: Each polish sub-agent gets ALL implementation files and a paragraph of instructions. Analysis agents got narrow scope (one task or one phase) with detailed structured output templates. Narrower scope + structured templates = more thorough analysis.

### 5.3 Sub-agent dispatch may be failing

The polish process took 75 minutes and likely hit context compactions. There is suspicion that the polish agent (which dispatches sub-agents itself) may have been unable to successfully dispatch and coordinate its sub-agents. A sub-agent trying to dispatch further sub-agents is architecturally fragile.

### 5.4 Quantitative cost/benefit breakdown

The polish commit (`ca0ac05..199e407`) produced:

| Category | Files | Insertions | Deletions | Net |
|---|---|---|---|---|
| Production code | 8 | +25 | -58 | **-33** |
| Test code | 6 | +505 | -47 | **+458** |
| **Total** | **16** | **+533** | **-106** | **+427** |

Production changes were net **negative 33 lines** — mostly removing dead code and adding a few validation calls. The bulk (505 of 533 insertions) was the 344-line `workflow_integration_test.go` file.

Actual findings by value:

| Finding | Production LOC | Value | Needed an agent? |
|---|---|---|---|
| `--blocks` missing cycle detection | ~18 lines | **High** — real cross-task bug | Yes — only cross-task analysis catches this |
| Dead code removal (`renderListOutput`, format helpers) | ~-58 lines | **Low** | No — `staticcheck` / `deadcode` finds this in seconds |
| Error message prefix consistency | ~4 lines | **Trivial** | No — linter or convention check |
| 5 integration tests | ~505 lines (test only) | **Medium** | Debatable — should've been a plan task |

**Conclusion**: 75 minutes of opus runtime for one genuinely valuable finding (~18 LOC). The rest was either deterministic tooling territory or work that belonged in the plan.

---

## 6. Proposed Changes for V5

### 6.1 Move polish orchestration up to the main orchestrator

**Problem**: The polish agent acts as a nested orchestrator — it dispatches sub-agents that dispatch executors that dispatch reviewers. This creates a 3-level deep agent hierarchy that is fragile, slow (75 minutes), and likely hits context limits.

**Solution**: Move the polish agent's orchestration responsibilities to the main implementation orchestrator. Instead of dispatching a single "polish agent" that internally manages sub-agents, the orchestrator directly dispatches the review/fix agents at the same level as task executors and reviewers.

### 6.2 Use agent teams for the review pass

Anthropic released [agent teams](https://code.claude.com/docs/en/agent-teams) (Feb 2026, experimental). Key differences from sub-agents:

| Feature | Sub-agents | Agent Teams |
|---------|-----------|-------------|
| Context | Shared with caller, results return | Own context window, fully independent |
| Communication | Report back to caller only | Teammates message each other directly |
| Visibility | Hidden from user | Each gets own tmux pane |
| Coordination | Caller manages all | Shared task list, self-coordination |

Agent teams solve the exact problem: independent reviewers that can each focus deeply on one concern (like the analysis agents did) and share findings with each other.

**Proposed team structure for post-implementation review:**

| Teammate | Focus | External References |
|----------|-------|-------------------|
| Code Cleanup | Dead code, naming drift, formatting | Project skills |
| Duplication Hunter | Cross-file pattern duplication, helper extraction opportunities | Full file list + grep for repeated patterns |
| Standards Auditor | Skill compliance (MUST DO/MUST NOT DO), spec-vs-convention conflicts | Skill files + spec |
| Test Gap Analyst | Missing edge cases, integration test gaps, assertion quality | Plan acceptance criteria + spec |
| Architecture Reviewer | Package structure, API design, consistency patterns | Spec + Go idioms |

Each reviewer reads the full codebase with a narrow, focused lens and structured output template. The adversarial framing: "find what's wrong" not "clean this up."

After review findings are collected, the orchestrator synthesizes and dispatches executors to fix prioritised issues.

### 6.3 Separate review from fix

The current polish agent discovers AND fixes in a loop. This conflates two concerns:

- **Discovery** should be thorough, adversarial, and read-only
- **Fixing** should be targeted, constructive, and test-guarded

Separating them means the review pass can be comprehensive without being constrained by "can I fix this in one pass?" thinking.

### 6.4 Consider the existing review stage

The technical workflow has a 6th stage: `technical-review`. This runs AFTER implementation and is designed to review the code. Options:

**Option A — Light polish during implementation + full review after:**
- Implementation orchestrator does a single focused pass (duplication + obvious issues)
- The review stage handles deeper analysis (standards, architecture, test gaps)
- Pro: clear separation of concerns, review has fresh eyes
- Con: more total processing, review may duplicate some polish work

**Option B — No polish during implementation + comprehensive review after:**
- Implementation just runs the 23 tasks and stops
- Review stage handles ALL quality analysis (what polish + review currently do)
- Pro: simplest implementation flow, review gets full ownership
- Con: review stage needs to both find AND fix issues, which is what polish does now

**Option C — Full review during implementation + light review after:**
- Implementation does the comprehensive agent-team review described in 6.2
- The review stage does a lighter verification pass
- Pro: issues caught closest to where they were created
- Con: heavier implementation process

**Recommendation**: Start with Option A. The implementation orchestrator dispatches a focused agent team for duplication/DRY/obvious issues (the mechanical stuff). The review stage handles the adversarial architectural analysis with skill compliance, test gaps, and spec-vs-convention assessment. This gives each stage a clear, non-overlapping mandate.

### 6.5 Leverage deterministic static analysis before agent work

**Problem**: The polish agent uses an opus-powered sub-agent for "Code Cleanup" — dead code, unused imports, naming drift. These are solved problems with deterministic tools that run in seconds.

**Solution**: Run language-appropriate static analysis tools as a mandatory step before any agent-based review. For Go: `staticcheck`, `golangci-lint`, `deadcode`. The output feeds directly into a fix pass — no agent analysis needed for these categories.

This could live in several places:
- **During implementation**: Each executor runs linters after their TDD cycle (cheapest — catches issues at source)
- **During planning**: Plan tasks include "configure and pass linters" as acceptance criteria
- **Post-implementation**: A single deterministic pass before any agent review

The first option is strongest — issues caught at source cost the least to fix and prevent accumulation. The executor's TDD cycle could include a lint step: RED → GREEN → LINT → REFACTOR.

This completely removes "Code Cleanup" from the agent review scope, letting agents focus on what they're uniquely good at: cross-task reasoning.

### 6.6 Time-boxing and model selection for review agents

**Problem**: The polish agent ran for 152 tool calls over 75 minutes with no guardrails.

**Solutions**:
- **`max_turns` on agent invocations**: Cap analysis agents at ~30 turns. If an agent can't complete analysis in 30 turns, the scope is too broad.
- **Lighter models for analysis passes**: Analysis sub-agents scan for patterns and produce findings — they don't need opus. Sonnet handles pattern recognition and structured output well at a fraction of the cost and latency.
- **Reserve opus for synthesis**: The orchestrator (which triages findings and decides what to fix) benefits from opus-level reasoning. The scanners don't.

### 6.7 Integration tests as plan tasks, not review findings

**Problem**: The polish agent generated 344 lines of integration tests. This was the bulk of its output and could have been written during normal implementation.

**Note**: Phases are a planning structure only — during implementation, tasks are picked up sequentially regardless of phase boundaries. The orchestrator doesn't know or care when a phase ends.

**Solution**: Include integration test tasks explicitly in the plan — e.g., "tick-core-3-6: cross-command integration tests" placed after the tasks they exercise. Benefits:
- Tests go through normal executor+reviewer cycle with full TDD workflow
- Tests are committed incrementally, not dumped in a single polish commit
- Review agents can focus on *gaps* in existing integration coverage rather than writing tests from scratch
- Reduces polish scope significantly

### 6.8 Project-local skill supplements

The `golang-pro` skill is third-party (from Vercel skills library). Create a project-local skill (e.g. `.claude/skills/go-project-standards/SKILL.md`) that supplements it with project-specific MUST DOs:

- `var _ Interface = &ConcreteType{}` compile-time checks when implementing interfaces
- Normalize both sides of ID/key comparisons (defensive equality)
- Use exact-string assertions for output-format tests, not substring checks
- Integration tests for CLI binaries — at least one test that builds and executes the real binary
- When spec conflicts with Go convention, follow Go convention and add a code comment documenting the deviation

This skill would be passed to both executors and reviewers alongside golang-pro.

---

## 7. Observations on This Document

Commentary on the proposals above — agreements, pushbacks, and a gap.

### 7.1 Agreements

**Section 2.1 (no integration tests)** — Correctly identified as the single most important gap. Section 6.7 proposes making these plan tasks rather than polish/review output, which is the right structural fix. Integration tests written during normal implementation go through TDD and get committed incrementally — better than a 344-line dump during polish.

**Section 3 (cross-task duplication)** — The "find task by index" loop appearing 5 times is exactly what per-task reviewers can't catch. But the polish agent *also* didn't catch it — reinforcing that polish needs a fundamentally different approach, not just better instructions for the same architecture.

**Section 5.2 (framing and context)** — The "no adversarial framing" insight is sharp. "Improve what exists" vs "find what's wrong" produces meaningfully different results from LLMs. Review agents should be told to *critique*, not *polish*. This is a low-cost change (just prompt wording) with potentially high impact.

**Section 6.3 (separate review from fix)** — The single most important structural change in this document. Discovery and fixing have fundamentally different objectives: discovery should be thorough, adversarial, read-only; fixing should be targeted, constructive, test-guarded. Conflating them constrains discovery ("can I fix this in one pass?") and encourages fixing trivial issues just because they're easy.

### 7.2 Pushbacks

**Section 6.2 (5 agent teammates for review)** — Five dedicated teammates risks overcorrecting. The polish agent was too heavy with 3 analysis sub-agents + executor + reviewer. Replacing that with 5 teammates could reproduce the same cost problem at higher token spend (agent teams use significantly more tokens than sub-agents, as noted in open question #2). With static analysis absorbing the "Code Cleanup" teammate's entire scope (6.5), and integration tests moved to plan tasks (6.7), the remaining review concerns could be covered by 2-3 focused reviewers. Start lean, add teammates only if gaps persist.

**Section 2.6 (write error propagation)** — Capturing `fmt.Fprintf` write errors in formatters is technically correct but borderline over-engineering for a CLI tool. If stdout is broken, the process has bigger problems. This is the kind of finding that looks meaningful in a comparative analysis but doesn't materially improve the codebase. Deprioritise vs the other gaps.

**Section 2.8 (EnsureFresh double-open)** — Worth noting as a design observation, but may not be worth the API complexity of returning `*Cache` for reuse. Tick is a CLI tool — each invocation is a short-lived process. The cost of opening SQLite twice (once for freshness check, once for query) is negligible in practice. If this were a long-running server, the calculus changes. For CLI, the simpler API is likely better.

### 7.3 Nuances

**Section 6.4 Option A recommendation** — Agree Option A is the right starting point, but the boundary between "light polish during implementation" and "full review after" needs to be explicitly defined. Without crisp scope definitions, both stages will do overlapping work. The doc acknowledges this ("clear, non-overlapping mandate") but doesn't define where the line is yet. Suggested boundary:

- **Implementation polish scope**: Deterministic tooling (linters, static analysis) + mechanical issues (dead code, unused imports). No agent judgment calls.
- **Review stage scope**: Cross-task reasoning, architectural coherence, spec compliance, test gap analysis. Agent-driven, adversarial framing.

This makes the boundary: "Can a tool catch it?" → implementation. "Does it require judgment?" → review.

### 7.4 Gap: context compaction resilience

Section 5.3 identifies context compaction as a likely cause of the polish agent's protocol violation (152 tool calls, never dispatched sub-agents). But none of the proposed solutions in section 6 explicitly address context compaction resilience.

Time-boxing (6.6) helps indirectly — fewer turns means less compaction pressure. But the deeper structural issue is: any agent that must hold the full codebase in context while also orchestrating sub-agents is fragile. A single compaction event can erase the procedural instructions that govern sub-agent dispatch, and the agent falls back to doing work directly.

Agent teams (6.2) actually solve this — each teammate gets its own context window, so compaction in one doesn't affect others. This is arguably the strongest argument for teams over sub-agents, stronger than the parallelism benefit. The doc could emphasise this: agent teams aren't just about parallel analysis, they're about **context isolation**. A reviewer that compacts only loses its own analysis state, not the orchestration protocol.

If agent teams aren't viable yet (experimental status, token cost), the fallback is: keep review agents small and focused enough that they complete well within context limits. The `max_turns` cap (6.6) serves this purpose. An agent that finishes in 30 turns won't hit compaction.

---

## 8. Open Questions

1. **Agent teams maturity**: The feature is experimental (requires `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`). Is it stable enough for production use in the workflow?

2. **Token cost**: Agent teams use significantly more tokens than sub-agents. Is the quality improvement worth the cost?

3. **Review stage scope**: Should the `technical-review` stage be redesigned to include the adversarial architectural analysis, or should that stay in implementation?

4. **Utility index**: Should the orchestrator maintain a lightweight "available helpers" index (function signatures + file paths) that executors can optionally reference? This addresses the helper reuse gap without convention lock-in. Risk: even a utility index could subtly create a gravity well.

5. **golang-pro alternatives**: Is there a more comprehensive Go skill available, or is project-local supplementation the right approach?

6. **Different test project**: Should V5 be tested on a different project type to control for tick-specific factors?

---

## 9. Action Items

| Priority | Action | Owner |
|----------|--------|-------|
| 1 | Restructure polish to move orchestration to main orchestrator | Workflow skill |
| 2 | Add deterministic static analysis (staticcheck/golangci-lint) to executor TDD cycle | Workflow skill |
| 3 | Design agent-team-based review pass with focused reviewers | Workflow skill |
| 4 | Create project-local Go skill with project-specific standards | This repo |
| 5 | Add integration test tasks to plans (end-of-phase or dedicated phase) | Planning convention |
| 6 | Add max_turns + model selection (sonnet for analysis, opus for synthesis) | Workflow skill |
| 7 | Update analysis playbook (already done — skill compliance + spec-vs-convention) | Done |
| 8 | Evaluate agent teams stability for workflow integration | Research needed |
| 9 | Decide Option A/B/C for implementation vs review stage split | Design decision |
| 10 | Run V5 implementation and round 3 analysis | After above changes |

---

## 10. References

- [Round 2 Analysis — Final Synthesis](analysis/round-2/final-synthesis.md)
- [Analysis Playbook](analysis/analysis-playbook.md)
- [Analysis Log](analysis/analysis-log.md)
- [Polish Agent Prompt](node_modules/@leeovery/claude-technical-workflows/agents/implementation-polish.md)
- [golang-pro Skill](.claude/skills/golang-pro/SKILL.md)
- [Agent Teams Documentation](https://code.claude.com/docs/en/agent-teams)

Sources:
- [Orchestrate teams of Claude Code sessions — Claude Code Docs](https://code.claude.com/docs/en/agent-teams)
- [Anthropic releases Opus 4.6 with new 'agent teams' | TechCrunch](https://techcrunch.com/2026/02/05/anthropic-releases-opus-4-6-with-new-agent-teams/)
- [Claude Code Agent Teams: Multi-Session Orchestration](https://claudefa.st/blog/guide/agents/agent-teams)
