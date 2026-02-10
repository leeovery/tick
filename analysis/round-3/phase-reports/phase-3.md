# Phase 3: Hierarchy & Dependencies

## Task Scorecard

| Task | Winner | Key Differentiator |
|------|--------|--------------------|
| 3-1  | V4 wins | Spec-exact error messages (`"Error: "` prefix), stronger test assertions (exact match vs substring), better function decomposition (6 focused helpers vs 1 monolithic function) |
| 3-2  | V5 wins slightly | Stale-ref removal test covering a spec-documented edge case V4 misses entirely, defensive ID normalization in duplicate/rm checks, table-driven test patterns |
| 3-3  | V5 wins | Cleaner Context + free-function + map-dispatch architecture, correct unexported fields on internal types, shared rendering evolution into 3-line alias delegation |
| 3-4  | V4 wins | Blocked query reuses `readyConditions` from ready.go (spec requirement), DRY rendering via shared `renderListOutput`, more granular test isolation |
| 3-5  | V5 wins | Explicit contradictory-filter test (plan-specified edge case V4 omits), delegation pattern eliminating ~80 lines of duplicated ready/blocked code, subquery wrapping avoiding alias refactoring |
| 3-6  | V5 only (Excellent) | Recursive CTE with pre-filter/post-filter architecture, refactoring of runReady/runBlocked into 3-line delegators, 15/15 plan tests covered |

## Phase-Level Patterns

### Architecture

The two implementations follow fundamentally different CLI architectures that become increasingly significant as the phase progresses:

**V4** uses an `*App` struct with methods (`func (a *App) runReady(args []string) error`) and a growing `switch` statement in `cli.go` for dispatch. Each command (ready, blocked, list) has its own fully independent handler with separate store opening, query execution, row scanning, and rendering. This creates clean isolation per command but accumulates duplication -- by task 3-5, V4 has three parallel implementations of the same query-execute-render pipeline in `runReady`, `runBlocked`, and `runList`.

**V5** uses free functions with a `*Context` parameter (`func runReady(ctx *Context) error`) and a `commands` map for dispatch. The pivotal architectural decision emerges in tasks 3-3 through 3-5: V5 progressively refactors `runReady` and `runBlocked` into 3-line delegators that prepend `--ready`/`--blocked` to args and call `runList`. This consolidation eliminates ~100 lines of duplicated code and guarantees that `tick ready` and `tick list --ready` are literally the same code path.

V4's approach is initially cleaner (task 3-1's well-decomposed helper functions, task 3-4's extracted `readyConditions` const), but V5's architecture scales better across the phase. By task 3-5, V5's delegation pattern pays significant dividends in maintainability and correctness assurance.

### Code Quality

Both implementations maintain consistently high code quality throughout the phase. Common strengths include: explicit error handling with `fmt.Errorf("%w", ...)` wrapping, `defer` for resource cleanup, thorough doc comments on all functions, and proper parameterized SQL.

**V4 code quality strengths:** Superior function decomposition in task 3-1 (6 small, focused helpers with single responsibilities). Extracted shared rendering via `renderListOutput` from the start in task 3-3. Pre-allocated maps in `buildBlockedByMap`. More memory-efficient BFS path reconstruction using parent-map tracing.

**V5 code quality strengths:** More idiomatic Go conventions -- lowercase error message prefixes (`"querying ready tasks:"` vs `"failed to query ready tasks:"`), unexported fields on internal structs (`listRow{id, status}` vs `listRow{ID, Status}`), extracted `isValidStatus` helper in task 3-5. V5 also provides distinct error messages for non-numeric vs out-of-range priority input, giving better user feedback. Defensive ID normalization (`task.NormalizeID(dep)`) in duplicate and removal checks handles potential data inconsistencies.

**Shared weakness:** Neither version consistently uses table-driven tests, which is a golang-pro skill requirement. V4 uses them in one place (task 3-5 error validation); V5 uses them in another (task 3-2 missing-args scenarios). Both default to individual subtests for most scenarios.

### Test Quality

Test coverage is comprehensive in both versions, but they diverge in assertion strategy and edge case selection.

**V4 test strengths:** Consistently uses exact string matching for error messages (`err.Error() != want`), which catches formatting regressions that substring checks would miss. V4's task 3-1 tests verify self-reference error message content (not just that an error occurred). V4's task 3-2 includes a no-mutation verification on duplicate add attempts. V4 avoids `time.Sleep` in timestamp tests by using fixed past timestamps. Test helper `depTask` produces complete, valid Task structs with all required fields populated.

**V5 test strengths:** Covers spec-documented edge cases that V4 omits entirely -- notably the stale-ref removal test in task 3-2 (a spec requirement that V4 never tests) and the contradictory-filter test in task 3-5 (`--status done --ready` yielding empty result, explicitly called out in the plan). V5's rm persistence test verifies partial removal (removing one of two deps), which is more realistic. V5 uses `task.NewTask()` constructors producing more concise test setup. V5 organizes tests under logical parent functions (e.g., `TestBlockedQuery`, `TestBlockedCommand`, `TestCancelUnblocksDependents`), improving readability.

**Overall:** V4's individual assertions are tighter but V5's coverage breadth is wider. V5 catches more categories of behavioral errors; V4 catches more categories of formatting regressions.

### Spec Compliance

Spec compliance is the most decisive differentiator across the phase, and the two versions split clearly.

**V4 spec compliance wins:**
- Task 3-1: Error messages include the `"Error: "` prefix that the spec explicitly defines. V5 omits it.
- Task 3-1: Error messages match the spec format exactly, without added unsolicited text. V5 appends `"(would create unworkable task due to leaf-only ready rule)"` which the spec does not include.
- Task 3-4: Blocked query reuses ready query logic via extracted `readyConditions`, directly satisfying the plan's "Reuses ready query logic" acceptance criterion. V5 writes an independent blocked query that could drift out of sync.

**V5 spec compliance wins:**
- Task 3-2: Tests the stale-ref removal behavior documented in the spec. V4 has no test for this.
- Task 3-5: Tests the contradictory-filter edge case explicitly listed in the plan. V4 omits this test.
- Task 3-5: Delegation pattern guarantees `tick ready` and `tick list --ready` are identical code paths, which is the strongest possible interpretation of the "alias" requirement.

Both versions have PARTIAL compliance on the `list --ready`/`list --blocked` alias requirement at their respective commit points, though both evolve toward full compliance in later commits.

### golang-pro Skill Compliance

Skill compliance is similar across both implementations:

- **Error handling**: Both PASS throughout. All errors explicitly checked and propagated with `%w`.
- **Table-driven tests**: Both PARTIAL. V4 uses table-driven once (task 3-5 error validation). V5 uses it twice (task 3-2 missing-args). Neither adopts it as the default pattern.
- **Documentation**: Both PASS. All exported and key unexported functions are documented.
- **No panics, no ignored errors, no hardcoded config**: Both PASS throughout.
- **Error message capitalization**: V5 is more Go-idiomatic (lowercase for routing/argument errors, capitalized for user-facing data errors). V4 uniformly capitalizes, matching spec but deviating from Go convention.
- **Exported fields on internal types**: V5 correctly uses unexported fields on `listRow`; V4 exports fields on an unexported struct (a minor anti-pattern).

Overall skill compliance is a slight V5 advantage due to more idiomatic Go conventions, but neither version fully embraces the table-driven test requirement.

## V5-Only Feature: Parent Scoping (3-6)

Task 3-6 rated **Excellent** as a standalone assessment. The implementation uses a recursive CTE to collect descendant IDs as a pre-filter, composing cleanly with all existing filters (ready, blocked, status, priority) via an `AND id IN (...)` clause. The core architectural insight -- the pre-filter/post-filter separation -- means `--parent` works automatically with `tick ready --parent X` and `tick blocked --parent X` with zero additional code, enabled by the runReady/runBlocked delegation pattern.

All 13 acceptance criteria are satisfied and all 15 plan-specified tests are implemented. The recursive CTE is correct (uses `UNION ALL` appropriately for tree structures, starts from children to exclude the parent naturally, leverages the `idx_tasks_parent` index). Error handling is thorough, with proper `%w` wrapping on all error paths. The only minor weaknesses are: the implicit nil-vs-empty contract in `appendDescendantFilter`, use of `[]interface{}` instead of the modern `any` alias, and theoretical scalability limits of the `IN (...)` clause for very large subtrees.

The refactoring of `runReady` and `runBlocked` into 3-line delegators as part of this task was an outstanding design decision that reduced ~100 lines of duplicated code while enabling the feature. This refactoring elevated the overall codebase quality beyond what the task strictly required.

## Phase Verdict

**Winner**: V5 wins 3/5 comparable tasks (3-2, 3-3, 3-5 vs V4's 3-1, 3-4)

V5 demonstrates a stronger architectural trajectory across the phase. While V4 starts with advantages in tasks 3-1 (superior function decomposition, spec-exact error formatting) and 3-4 (ready query reuse for blocked, DRY rendering), V5's architectural choices compound in value as the phase progresses. The delegation pattern that consolidates `runReady`, `runBlocked`, and `runList` into a single code path is the single most impactful design decision in the phase -- it eliminates ~100 lines of duplicated code, guarantees behavioral consistency between alias commands and their flag equivalents, and enables the task 3-6 parent scoping feature to work across all three commands with zero additional code.

V4's wins are real and significant. Task 3-1's spec-exact error messages and six well-decomposed helper functions represent genuinely better craftsmanship for that specific task. Task 3-4's reuse of ready conditions for the blocked query is architecturally sounder than V5's independent query at commit time, and directly satisfies a plan acceptance criterion that V5 fails. These are not negligible advantages -- they show V4 taking spec compliance seriously and decomposing problems thoughtfully.

However, V5's advantages are more systemic. Its Context + free-function architecture is more testable and composable than V4's App-method pattern. Its test coverage captures spec-documented edge cases (stale-ref removal, contradictory filters) that V4 misses entirely. Its subquery-wrapping approach in task 3-5 treats ready/blocked queries as opaque building blocks rather than requiring alias refactoring across three files. And the Excellent-rated task 3-6 provides additional evidence of V5's architectural quality -- the parent scoping implementation is clean, thoroughly tested, and demonstrates how the delegation pattern enables feature composition with minimal incremental complexity. On balance, V5 is the stronger implementation across this phase.
