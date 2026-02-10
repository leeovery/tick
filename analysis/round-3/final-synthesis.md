# Round 3: V5 vs V4 -- Final Synthesis

## Executive Summary

V5 wins Round 3 decisively. Across 23 comparable tasks, V5 wins 16, V4 wins 6, and 1 is a tie. V5 wins 3 of 5 comparable phases and ties 2. Beyond the comparable scope, V5 delivers 8 additional tasks -- 1 feature (parent scoping) and 7 analysis refinements -- all rated Excellent. The combined picture is a version that is architecturally stronger, more spec-compliant, more idiomatic in Go, and uniquely capable of self-correction.

The margin (16-6) is the largest in any round of this analysis. It exceeds Round 2's margin (V4 15, V2 7) and approaches Round 1's (V2 21, V3 2), though the nature of V5's wins is qualitatively different -- V5 does not expose a fundamental design flaw like V3's convention gravity well, but rather accumulates a steady stream of engineering advantages: DRY helpers, correct error handling, efficient escaping, map-based dispatch, struct-parameter interfaces, and exact-output test assertions.

V4's wins are real and concentrated in specific dimensions: test assertion depth on mutation commands (1-6, 5-1), code decomposition and test rigor in dependency validation (3-1), ready-query reuse for blocked semantics (3-4), and type-safe interface design from the start (4-4). These reflect V4's core strength -- disciplined verification rigor. V4 never uses `interface{}` in its Formatter and builds deeper test coverage for individual commands. These are valuable engineering qualities that V5's workflow should preserve. (Note: V4's "spec-exact error messages" — previously counted as a strength — actually embed display formatting inside error values, which is a Go anti-pattern. See "Spec vs Go Idiom" section.)

The most significant V5 development is Phase 6: the analysis refinement phase. Seven tasks found and fixed real issues -- a dependency validation gap (correctness bug), 100 lines of duplicated logic, 50 lines of dead code, and 15 runtime type assertions replaced with compile-time checks. The existence of this phase, and the surgical quality of its execution, demonstrates that V5's workflow has matured beyond feature delivery into retrospective quality assurance. V4 has no equivalent capability. This is the strongest evidence yet that iterating on the workflow produces compounding returns.

## Full Scorecard

### Task-Level Results (23 Comparable Tasks)

| Task | Name | Winner | Key Differentiator |
|------|------|--------|--------------------|
| 1-1 | Task model & ID generation | V5 | ISO 8601 custom JSON marshaling, case-insensitive self-ref detection, broader tests |
| 1-2 | JSONL storage with atomic writes | V5 | Separate `storage` package, `json.NewEncoder` idiom, better test organization |
| 1-3 | SQLite cache with freshness detection | V5 | DRY `FormatTimestamp` reuse, practical `EnsureFresh` return type, rollback testing |
| 1-4 | Storage engine with file locking | V5 | Spec-correct lock error message, DRY lock helpers, persistent cache, functional options |
| 1-5 | CLI framework & tick init | V5 | Map-based command dispatch, `*os.File` TTY signature, corrupted `.tick/` edge case test |
| 1-6 | tick create | V4 | Significantly broader test coverage: timestamp verification, cache.db, non-numeric priority |
| 1-7 | tick list & tick show | V5 | Spec-compliant `--pretty` test format, correct section omission, contradictory filter test |
| 2-1 | Status transition validation | Tie | V5 more idiomatic data structures; V4 better error differentiation and time testing |
| 2-2 | CLI transition commands | V4 (slight) | Updated timestamp check, defensive TrimSpace, task ID in not-found errors |
| 2-3 | tick update | V5 (slight) | Early validation before store open, O(1) index lookup, unknown flag rejection |
| 3-1 | Dependency validation | V4 | 6 decomposed helpers, exact-match test assertions, batch success test (note: "Error:" prefix is a Go anti-pattern — see Spec vs Go Idiom) |
| 3-2 | dep add & dep rm | V5 (slight) | Stale-ref removal test, partial rm persistence, defensive ID normalization |
| 3-3 | Ready query & tick ready | V5 | Cleaner Context + free-function architecture, unexported internal fields, delegation pattern |
| 3-4 | Blocked query & cancel-unblocks | V4 | Spec-compliant `readyConditions` reuse for blocked query, DRY rendering |
| 3-5 | List filter flags | V5 | Contradictory filter test, DRY delegation eliminating ~80 lines, subquery wrapping |
| 4-1 | Formatter abstraction & TTY | V5 | Explicit `*os.File` DetectTTY, working StubFormatter, derived FormatConfig via method |
| 4-2 | TOON formatter | V5 | 37 write-error checks vs 0, efficient `escapeField` fast-path, shared text helpers |
| 4-3 | Pretty formatter | V5 | Right-aligned PRI column, exact string assertions, shared helpers, correct truncateTitle |
| 4-4 | JSON formatter | V4 | Type-safe concrete Formatter interface (no `interface{}`), error-returning FormatMessage, nil-slice tests |
| 4-5 | Formatter integration | V5 | Typed struct params, consistent quiet at command level, TaskRow DTO, shared helpers |
| 4-6 | Verbose output & edge cases | V5 | 17 vs 7 tests across 3 layers, functional options, no-op default logger |
| 5-1 | tick stats | V4 | Blocked = Open - Ready derivation, significantly more thorough format and JSON tests |
| 5-2 | tick rebuild | V5 | DRY `acquireExclusive()` helper, functional options, superior package separation |

**Score: V5 16 -- V4 6 -- Tie 1**

### V5-Only Tasks (8)

| Task | Name | Rating | Category |
|------|------|--------|----------|
| 3-6 | Parent scoping with recursive CTE | Excellent | New feature |
| 6-1 | Dependency validation gap fix | Excellent | Correctness bug |
| 6-2 | Shared ready/blocked SQL WHERE clauses | Excellent | Code duplication |
| 6-3 | Consolidated JSONL parsing | Excellent | Code duplication |
| 6-4 | Shared formatter methods | Excellent | Code duplication |
| 6-5 | Remove doctor from help text | Excellent | Dead code |
| 6-6 | Remove dead StubFormatter | Excellent | Dead code |
| 6-7 | Type-safe Formatter parameters | Excellent | Type safety |

### Phase-Level Results

| Phase | Name | Winner | Task Score |
|-------|------|--------|------------|
| 1 | Walking Skeleton | V5 | 6-1 |
| 2 | Task Lifecycle | Tie | 1-1-1 |
| 3 | Hierarchy & Dependencies | V5 | 3-2 (comparable) + 1 V5-only Excellent |
| 4 | Output Formats | V5 | 5-1 |
| 5 | Stats & Cache Management | Tie | 1-1 |
| 6 | Analysis Refinements | V5 only | 7/7 Excellent |

**Score: V5 3 -- V4 0 -- Tie 2 (+ Phase 6 V5-only Excellent)**

## Version Profiles

### V5 Profile

**Systematic strengths across 31 tasks:**

1. **Architectural composability.** V5 consistently builds small, reusable components that compound across tasks. The map-based command dispatch (1 line per command vs 6-12 lines of switch boilerplate), the `*Context` free-function pattern, the delegation of `runReady`/`runBlocked` into 3-line wrappers over `runList`, the functional options pattern for store construction -- these are infrastructure investments that reduce friction in every subsequent task. By Phase 3, adding a new filter flag or command requires trivially small changes because the architecture absorbs complexity.

2. **DRY discipline.** V5 extracts reusable helpers aggressively: `FormatTimestamp` (used in 4 packages), `validateIDsExist` (used in create, update, dep add), `splitCSV`/`normalizeIDs` (shared across mutation commands), `formatTransitionText`/`formatDepChangeText`/`formatMessageText` (shared across 3 formatters), `readyWhereClause`/`blockedWhereClause` (shared across ready, blocked, list, stats), `acquireExclusive` (shared across Mutate, Query, Rebuild). V4 duplicates most of these patterns.

3. **Spec compliance on persistence and output.** V5's custom ISO 8601 JSON marshaling (1-1), spec-exact lock timeout message (1-4), correct section omission testing (1-7), and strict-spec formatter output (4-2 through 4-5 with no `parent_title` scope creep) demonstrate that V5 treats the spec as authoritative for user-facing and persistence-facing behavior.

4. **Idiomatic Go.** Gerund-based error wrapping (`"querying tasks: %w"` not `"failed to query tasks: %w"`), value-type returns for small structs, unexported fields on internal types, separate `Log`/`Logf` methods mirroring stdlib, struct-tag-based toon-go usage. These are individually minor but collectively signal deeper Go fluency.

5. **Self-correction.** Phase 6 demonstrates V5's unique capability: retrospective analysis that finds real issues and fixes them surgically. The `interface{}` Formatter anti-pattern (V5's biggest weakness in Phases 4-5) was identified and completely eliminated in task 6-7. The dependency validation gap (a genuine correctness bug) was caught and fixed in task 6-1. No other version in any round has demonstrated this capability.

**Systematic weaknesses across 31 tasks:**

1. **Test assertion depth on mutation commands.** V5 consistently provides fewer assertions per test for write-path commands. In 1-6 (create), V5 misses timestamp range verification, cache.db existence checks, non-numeric priority input, and two-task uniqueness. In 5-1 (stats), V5's format tests check only headers/labels, not actual values. In 3-1 (dependency validation), V5 uses `strings.Contains` where V4 uses exact match. This is V5's most persistent weakness -- it tests more edge case categories but with shallower per-test verification.

2. **The `interface{}` arc (Phases 4-5).** V5's Formatter interface used `interface{}` parameters for 6 tasks (4-1 through 5-2), requiring 15 runtime type assertions across 3 formatters. This was a genuine anti-pattern that violated Go's type safety principles. V5 fixed it in 6-7, but the debt existed for the majority of the output formatting phase and directly caused V5's loss on task 4-4.

3. **Occasional error handling gaps.** V5's `FormatMessage` returns no error (void signature), silently losing write failures. The rebuild CLI handler swallows the `FormatMessage` return. These are specific violations of the golang-pro skill's error handling rules that V4 avoids.

4. **Unsolicited error message additions.** V5 appends explanatory text `"(would create unworkable task due to leaf-only ready rule)"` to the child-blocked-by-parent error in task 3-1 — content the spec does not include. Adding unsolicited context to spec-defined messages is a legitimate deviation. (Note: V5's omission of the `"Error: "` prefix from error *values* is separately addressed under "Spec vs Go Idiom" below — that choice is actually correct per Go conventions.)

### V4 Profile

**Systematic strengths across 23 tasks:**

1. **Test assertion thoroughness.** V4 consistently verifies more fields per test. Updated timestamp refresh after transitions (2-2), lowercase ID in output (2-2), exact error message matching via `err.Error() != want` (3-1), typed `jsonStats` struct for compile-time JSON deserialization safety (5-1), timestamp range verification on create (1-6), all 5 P0-P4 priority labels in Pretty format (5-1). This depth catches regressions that V5's broader-but-shallower tests would miss.

2. **Type-safe Formatter interface from day one.** V4 used concrete types (`[]listRow`, `TaskDetail`, `StatsData`) in Formatter method signatures from task 4-1 onward. Every call site and every implementation is statically checked. V4 never incurred the `interface{}` technical debt that V5 accumulated and later cleaned up.

3. **Spec-exact error message text.** V4 matches spec-defined error message templates verbatim across tasks 3-1, 2-1, and 3-4. However, V4 embeds the `"Error: "` prefix directly in `fmt.Errorf()` return values — e.g., `fmt.Errorf("Error: Cannot %s task %s — status is '%s'", ...)` — which is a Go anti-pattern. Error values should not be prefixed with "Error:" because the caller already knows the return is an error; the CLI layer is the correct place to add display formatting. V5's approach of returning clean error values and letting the CLI layer prepend "Error:" is more correct per Go conventions. (See "Spec vs Go Idiom" section below.)

4. **Mathematically sound invariants.** V4's `Blocked = Open - readyCount` in stats (5-1) guarantees `Ready + Blocked = Open` as a tautology. V4's `readyConditionsFor(alias)` function ensures blocked queries are always the exact inverse of ready queries. These are correctness-by-construction patterns that eliminate entire classes of consistency bugs.

5. **Function decomposition in domain logic.** V4's dependency validation (3-1) uses 6 focused helpers (`checkChildBlockedByParent`, `checkCycle`, `buildBlockedByMap`, `reconstructPath`, `formatCyclePath`) each with single responsibilities and doc comments. This is textbook Go code organization.

**Systematic weaknesses across 23 tasks:**

1. **Architecture does not scale.** V4's `*App` method pattern with switch-case dispatch adds 6-12 lines of boilerplate per command. By Phase 3, three parallel implementations of the query-execute-render pipeline exist in `runReady`, `runBlocked`, and `runList` with ~100 lines of duplicated code. V4 cannot add `--parent` scoping without duplicating logic across all three commands.

2. **Write-error handling is systematically absent.** V4's TOON and Pretty formatters ignore all `fmt.Fprintf`/`fmt.Fprintln` return values. The TOON formatter (4-2) has 0 checked write-error sites where V5 has 37. This is a pervasive violation of the golang-pro skill's error handling rules.

3. **DRY violations accumulate.** Lock acquisition boilerplate is duplicated across `Mutate`, `Query`, and `Rebuild` (~15 lines each). Timestamp format strings are hardcoded in each package that needs them. Formatter implementations duplicate identical transition/dep/message formatting logic. These duplications compound into maintenance hazards.

4. **Scope creep.** V4 adds `parent_title` to TOON, Pretty, and JSON task detail output across tasks 4-2 through 4-5 -- a field not specified in any task plan. While arguably useful, this constitutes unilateral spec deviation that could break strict schema validation.

5. **No self-correction mechanism.** V4 has no equivalent of Phase 6. The `interface{}` anti-pattern that V5 introduced and later fixed is exactly the kind of issue that V4's workflow would never surface. V4 ships whatever the initial implementation produces, without retrospective analysis. Any technical debt introduced early persists through the final commit.

## Comparative Patterns

### Where V5 Consistently Excels

**Pattern 1: Infrastructure investment.** V5 makes upfront investments in reusable infrastructure that pay dividends across subsequent tasks. The map-based dispatch (1-5), `FormatTimestamp` constant (1-1), persistent cache connection (1-4), `validateIDsExist` helper (1-6), `runReady`/`runBlocked` delegation pattern (3-3 through 3-5), shared formatter helpers (4-2), and `acquireExclusive` lock helper (5-2) each reduce the incremental cost of later tasks. The parent scoping feature (3-6) is the capstone example: it works automatically with `tick ready --parent X` and `tick blocked --parent X` with zero additional code, because the delegation pattern already unified the three commands into a single code path.

**Pattern 2: Exact-output test assertions.** V5 consistently uses `buf.String() != expected` for formatter output tests (4-2, 4-3, 4-5), catching any deviation in formatting -- extra whitespace, wrong alignment, missing newlines. V4 relies more heavily on `strings.Contains`, which can pass even if the output has extra or malformed content.

**Pattern 3: Error handling in formatters.** V5's TOON formatter (4-2) checks every one of 37 write-error return values. V4's TOON formatter checks zero. This is the single largest code quality gap between the two versions.

**Pattern 4: Spec fidelity on persistence and output format.** V5's custom ISO 8601 JSON marshaling, correct section omission testing, and strict-spec formatter output (no `parent_title`) demonstrate a pattern of treating the spec as authoritative where it directly affects stored data and user-facing output.

### Where V4 Consistently Excels

**Pattern 1: Assertion depth on critical paths.** V4 builds deeper test suites for mutation commands and format verification. Task 1-6 (create) has 358 more lines of tests covering timestamp verification, cache.db existence, non-numeric priority, `--blocks` timestamp update, and two-task uniqueness. Task 5-1 (stats) verifies actual TOON data values, all 5 P0-P4 labels, and typed JSON deserialization. Task 3-1 (dependency validation) uses exact string matching for every error message. V4's tests catch more regression categories per test, even if V5 covers more categories overall.

**Pattern 2: Spec-verbatim error message text.** Across tasks 2-1, 3-1, and 3-4, V4 matches spec-defined error format strings verbatim. V5 adds unsolicited explanatory text in 3-1. However, V4's inclusion of the `"Error: "` prefix inside error *values* is a Go anti-pattern (see "Spec vs Go Idiom" below) — so V4's apparent spec fidelity here actually reflects a convention violation. V5 correctly separates the error value from its display formatting.

**Pattern 3: Correctness invariants by construction.** V4's `Blocked = Open - readyCount` (5-1) and `readyConditionsFor(alias)` (3-4) ensure that blocked query semantics are always the exact inverse of ready query semantics, enforced by the code structure rather than by developer discipline. V5's separate `readyWhereClause`/`blockedWhereClause` constants could drift.

### The interface{} Arc

V5's Formatter interface used `interface{}` parameters from task 4-1 through task 5-2 -- a span of 8 tasks across two phases. Every data-bearing Formatter method accepted `interface{}`, requiring runtime type assertions in all 15 implementation sites (5 methods x 3 formatters). This was V5's most significant architectural weakness: it violated Go's type safety principles, added 20+ lines of boilerplate, created runtime panic risks, and directly caused V5's loss on task 4-4 where V4's concrete-type interface was objectively superior.

What makes this arc instructive is not the mistake but the correction. In task 6-7, V5's post-implementation analysis identified the `interface{}` parameters as a type safety issue, generated a well-scoped remediation plan, and executed it surgically: all 15 `interface{}` parameters were replaced with concrete types (`[]TaskRow`, `*showData`, `*TransitionData`, `*DepChangeData`, `*StatsData`), all 15 runtime type assertion sites were eliminated, and compile-time interface satisfaction checks (`var _ Formatter = &ToonFormatter{}`) were added to each formatter's test file. Zero test modifications were needed because all call sites already passed the correct types.

This arc is the strongest evidence of workflow maturity in any round. V4 avoids the mistake entirely by using concrete types from the start -- which is better engineering -- but V4's workflow has no mechanism to find and fix mistakes that do occur. V5's workflow found and fixed its own worst decision. Over many iterations of a codebase, the ability to self-correct is arguably more valuable than the ability to never err, because the former scales to arbitrary complexity while the latter does not.

### The Analysis Refinement Phase

Phase 6 is V5's most distinctive contribution to the analysis. Seven tasks addressed four categories of post-implementation issues:

**Correctness (1 task):** Task 6-1 found that `create --blocked-by`, `create --blocks`, and `update --blocks` bypassed the cycle detection and child-blocked-by-parent validation enforced by `dep add`. This was a real bug -- users could create dependency cycles through these alternative paths without detection. The fix required a sophisticated stub-task pattern to make `ValidateDependency` work during `create` when the task does not yet exist in the list. V4 has this same bug and no mechanism to find it.

**Code duplication (3 tasks):** Tasks 6-2, 6-3, and 6-4 eliminated ~100 lines of duplicated logic: SQL WHERE clauses shared between listing and counting queries, JSONL scanner loops in `ReadTasks`/`ParseTasks`, and formatter method bodies identical across `ToonFormatter`/`PrettyFormatter`. Each used the same surgical pattern: extract shared logic into a single source of truth, delegate from original call sites, zero test modifications needed (proving behavioral equivalence).

**Dead code (2 tasks):** Task 6-5 removed a phantom `doctor` command from help text that was never implemented. Task 6-6 removed `StubFormatter` -- scaffolding that was unreachable after all concrete formatters were built. Combined: ~50 lines of dead code removed.

**Type safety (1 task):** Task 6-7 replaced all 15 `interface{}` Formatter parameters with concrete types, converting runtime panics into compile-time errors.

The cumulative impact is significant: the codebase is measurably more correct (cycle detection on all paths), safer (compile-time type checking on all formatter calls), more maintainable (single source of truth for shared logic), and cleaner (no dead code). The net code reduction across the phase was ~130 lines, with every deletion improving rather than degrading the codebase.

What Phase 6 tells us about the workflow is that V5's development process does not just build features -- it retrospectively audits the result, generates findings, and remediates them with the same rigor applied to primary development. This is a maturity signal that no previous version has demonstrated.

### Spec vs Go Idiom: Error Message Formatting

A recurring theme across tasks 2-1, 3-1, and 3-4 is the treatment of the `"Error: "` prefix in error messages. The spec defines error formats like `"Error: Cannot add dependency - creates cycle: ..."` and `"Error: Cannot {command} task {id} — status is '{status}'"`. V4 embeds this prefix directly in `fmt.Errorf()` return values. V5 omits it from error values and relies on the CLI layer to add the prefix when displaying errors to users.

**V5 made the correct choice.** Go conventions are explicit on this point:

- Error strings should not be capitalized (per Effective Go and the Go Code Review Comments wiki)
- Error strings should not include redundant context — the caller knows it received an error because `err != nil`
- The `"Error: "` prefix is display-layer formatting, not error-value content

V4's pattern of `fmt.Errorf("Error: Cannot %s task %s — status is '%s'", command, t.ID, t.Status)` produces error values that look wrong when composed with `%w` wrapping, logged by structured loggers, or compared programmatically. The `"Error: "` prefix is meaningful only when displayed to a CLI user — and both V4 and V5 already have a CLI-layer function (`writeError` / `fmt.Fprintf(stderr, "Error: %s\n", err)`) that adds this prefix for display. V4 therefore double-prefixes: the error says "Error:" and then the CLI layer prepends "Error:" again (or V4 has to special-case the display to avoid doubling).

This is a textbook example of the spec-vs-convention conflict identified in Round 2's methodology gap. The spec was written from a user-facing perspective, showing what the CLI output should look like. The `"Error: "` prefix in the spec describes the *display format*, not the *error value format*. V5 correctly interpreted this distinction; V4 took the spec too literally and violated Go idioms in the process.

**Impact on verdicts:** This reframing does not change any individual task verdicts — V4's wins on 3-1 and 3-4 are justified by other factors (code decomposition, test rigor, ready-query reuse). But it does shift the narrative: what the task reports characterized as a V4 strength ("spec-exact error messages") is actually a V4 weakness that happens to produce spec-matching output through incorrect layering. V5's approach is both more Go-idiomatic and architecturally sounder.

The golang-pro project skill supports this interpretation — its MUST DO rules include following Go conventions for error handling, and its MUST NOT DO rules include avoiding patterns that diverge from stdlib idioms without justification.

## Workflow Validation

### What Changed Between V4 and V5 Workflows

Based on the evidence in the task and phase reports, V5's workflow improvements produced visible effects in three areas:

**1. Architectural discipline improved.** V5's consistent use of map-based dispatch, `*Context` free-functions, delegation patterns, and functional options suggests stronger architectural guidance in the workflow. V4's `*App`-method + switch-case pattern was consistent but less scalable. The workflow change likely involved clearer architectural constraints or examples in the planning phase.

**2. DRY extraction became systematic.** V5 extracts reusable helpers at the point of second use, not third or fourth. The `FormatTimestamp` constant appears in task 1-1 and is reused in 1-3. The `validateIDsExist` helper appears in 1-6 and is reused in 2-3. The `formatTransitionText` helper appears in 4-2 and is reused in 4-3. This suggests the workflow now includes explicit checks for extraction opportunities during implementation.

**3. Post-implementation analysis was added.** Phase 6 is entirely new -- V4 had no equivalent. The workflow now includes a retrospective pass that finds validation gaps, duplication, dead code, and type safety issues, then generates well-scoped remediation tasks. This is the most impactful workflow change, producing 8 high-quality tasks that materially improved the codebase.

**4. Test assertion depth slightly degraded.** V5 covers more edge case categories but with shallower per-test verification. This suggests the workflow's test guidance may have shifted toward breadth over depth. V4's deeper assertions on individual tests (timestamp range checks, typed JSON deserialization, exact error message matching) are a genuine loss.

### Recommendations for V6

**1. Restore test assertion depth.** V5's most consistent weakness is shallow per-test assertions on mutation commands. The V6 workflow should include explicit guidance to verify all acceptance criteria fields per test -- not just that the operation succeeded, but that timestamps were refreshed, output format is exact, error messages match the spec, and side effects (cache.db creation, blocked target updates) are verified. The ideal is V5's edge case breadth combined with V4's per-test depth.

**2. Distinguish spec display format from error value format.** The spec's `"Error: ..."` prefix describes CLI *display* output, not error *value* content. V5 correctly keeps error values clean and lets the CLI layer format for display — this should be preserved. However, V5 should avoid adding unsolicited explanatory text to spec-defined messages (as in 3-1's child-blocked-by-parent error). The V6 workflow should enforce that error *message content* (after the prefix) matches the spec, while the `"Error: "` prefix remains a display-layer concern.

**3. Keep the analysis refinement phase.** Phase 6 is the highest-impact workflow innovation in this round. It found a real correctness bug (6-1), eliminated meaningful technical debt (6-2 through 6-4, 6-6, 6-7), and removed dead code (6-5, 6-6). This phase should be a permanent part of the workflow.

**4. Fix FormatMessage error handling.** V5's `FormatMessage` void return is a known gap. The V6 workflow should ensure all Formatter methods return `error`, including `FormatMessage`, to prevent silent write failures.

**5. Add a correctness invariant check to stats.** V4's `Blocked = Open - readyCount` derivation is architecturally superior to V5's two independent queries. The V6 workflow should include guidance to derive computed values from primitives where possible, rather than running independent queries that could drift.

**6. Mandate compile-time interface checks.** V5 added `var _ Formatter = &ToonFormatter{}` in Phase 6 but not in Phase 4 where it was first needed. The V6 workflow should require compile-time interface satisfaction checks in the same task that introduces the implementation.

## Historical Context

### Round 1: V2 vs V3 -- V2 won 21/23

V3 suffered a catastrophic regression caused by PR #79's integration context mechanism, which locked early unconventional decisions (string timestamps, bare error returns) into "established patterns" and had reviewers enforce consistency with them. V2 won overwhelmingly because V3's foundational choices were poor and the workflow amplified rather than corrected them. The lesson: workflow mechanisms that enforce consistency can be harmful when the initial direction is wrong.

### Round 2: V4 vs V2 -- V4 won 15/23

V4 removed PR #79 entirely, eliminating the convention gravity well. V4 introduced concrete Formatter types (fixing V2's `interface{}`), `readyConditionsFor(alias)` for SQL reuse, `SerializeJSONL` for in-memory hashing, and pointer-based flag optionality. V2 retained advantages in integration testing, defensive ID normalization, and spec fidelity. The lesson: removing a harmful workflow mechanism is more effective than adding guardrails around it.

### Round 3: V5 vs V4 -- V5 won 16/23

V5 extends V4's architectural foundation with stronger DRY discipline, map-based dispatch, delegation patterns, and a post-implementation analysis phase. V5 introduced the `interface{}` Formatter anti-pattern but self-corrected in Phase 6. V4 retained advantages in test assertion depth, spec-exact error messages, and correctness-by-construction invariants. The lesson: workflow maturity produces compounding returns -- each iteration improves not just the code but the process's ability to find and fix its own weaknesses.

### Trend Analysis

The winning margins tell a story of convergence and escalation:

- **Round 1 (21-2):** The gap was enormous because V3 had a systemic flaw (convention gravity well). One bad workflow mechanism can dominate all other factors.
- **Round 2 (15-7):** The gap narrowed because V4 fixed the systemic flaw. Wins were driven by targeted architectural improvements, not by avoiding a catastrophe. V2 still won 7 tasks on genuine strengths.
- **Round 3 (16-6):** The gap widened slightly, but the character of the competition changed. V4 had no systemic flaw to exploit; V5 won through accumulated marginal advantages plus the Phase 6 analysis refinement capability. V4's 6 wins are defensible and concentrated in real strengths (test depth, spec compliance, type safety from the start).

The progression suggests that the workflow is approaching a local optimum for the comparable 23-task scope. V5's 16 wins are distributed across all 5 phases, and the remaining V4 wins are in areas where V4 has genuine engineering merit, not workflow artifacts. Further gains will likely come from targeted fixes (restore test depth, enforce spec-exact errors) rather than architectural redesign.

The more interesting development is the expanding scope. V5 delivered 31 tasks where V4 delivered 23 -- an additional feature (parent scoping) and 7 refinement tasks. If the analysis framework evolves to weight total output quality rather than just comparable-task head-to-head results, V5's advantage grows substantially. The Phase 6 tasks alone would give V5 a qualitative edge even if the comparable task score were even.

## Conclusion

V5 is the clear winner of Round 3 and becomes the new baseline for future workflow iterations. Its 16-6 margin on comparable tasks, combined with 8 additional tasks all rated Excellent, represents the strongest performance of any version across all three rounds of analysis.

The most important finding is not V5's task-level superiority but its demonstrated ability to self-correct. The `interface{}` Formatter arc -- introduced in Phase 4, persistent through Phase 5, identified and eliminated in Phase 6 -- is the first time any version has found and fixed its own architectural mistake through a structured process. V4 avoids this mistake entirely, which is admirable, but V4's workflow provides no mechanism to surface mistakes that do occur. As codebases grow in complexity, the ability to retrospectively audit and remediate becomes more valuable than the ability to be right on the first pass, because the former scales and the latter does not.

V4's contributions should not be dismissed. Its test assertion depth, type-safe Formatter interface from the start, correctness-by-construction invariants (blocked = open - ready), and meticulous function decomposition are genuine engineering strengths that V5 should adopt. Notably, V4's "spec-exact error messages" — previously counted as a strength — is actually a Go anti-pattern: embedding `"Error: "` display formatting inside error values violates Go conventions and produces values that compose poorly. V5's approach of returning clean error values and formatting at the CLI layer is both more idiomatic and architecturally sounder. The ideal V6 would combine V5's architecture, DRY discipline, correct error layering, and self-correction capability with V4's verification rigor and test depth. The workflow recommendations above target exactly this synthesis. Three rounds of iterative analysis have progressively identified what works (removing harmful consistency enforcement, adding post-implementation analysis) and what needs preservation (test depth, spec compliance). The workflow is converging on a mature engineering process.
