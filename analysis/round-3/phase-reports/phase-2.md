# Phase 2: Task Lifecycle — Status Transitions & Update

## Task Scorecard

| Task | Winner | Key Differentiator |
|------|--------|--------------------|
| 2-1  | Tie    | V5 has more idiomatic data structure and return type; V4 has better error differentiation and more robust time testing. Neither has a clear overall edge. |
| 2-2  | V4 (slight) | V4's tests verify `Updated` timestamp refresh, lowercase output, and task ID in not-found errors -- directly mapping to acceptance criteria V5 leaves unchecked. |
| 2-3  | V5 (slight) | V5 validates title/priority before opening the store, uses O(1) index lookup for blocks, and explicitly rejects unknown flags. Structural improvements outweigh V4's incremental test coverage advantages. |

## Phase-Level Patterns

### Architecture

Both versions share the same high-level flow across all three tasks: parse arguments, discover tick dir, open store, mutate inside a callback, output results. The divergence is in dispatch and context passing.

**V4** uses a method-on-`*App` pattern with a `switch`-case dispatch in `cli.go`. Each new command adds a 5-6 line case block with identical error-handling boilerplate (`if err := ...; err != nil { a.writeError(err); return 1 }`). State flows through the `App` receiver (`a.Stdout`, `a.Quiet`, `a.Dir`).

**V5** uses free functions with a `*Context` parameter and a `commands` map dispatch. Each new command is a single map entry. The closure-returning pattern in 2-2 (`runTransition(command string) func(*Context) error`) eliminates duplication across the four transition commands.

V5's dispatch mechanism is objectively more extensible -- adding a command is 1 line vs 6. V5's `Context` struct bundles request state more cleanly than V4's receiver fields. Across the phase, V5's architecture scales better.

A notable V5 advantage emerges in task 2-3: V5 validates stateless inputs (title length, priority range) **before** opening the store, avoiding unnecessary filesystem access on obvious user errors. V4 validates everything inside the Mutate callback. This is the most significant single architectural difference in the phase.

### Code Quality

**Data structures:** V5 consistently prefers more efficient or idiomatic representations. In 2-1, a nested `map[Status]Status` replaces V4's `[]Status` + linear-scan helper. In 2-3, a `map[string]int` (storing task indices) replaces V4's `map[string]bool`, enabling O(1) index access for blocks instead of O(n) rescans. Both improvements are minor in practice (small data sets) but demonstrate better algorithmic instincts.

**Return types:** V5 consistently uses value types for small structs (`TransitionResult`, `updateOpts`) while V4 uses pointers (`*TransitionResult`, `*updateFlags`). V5's approach is more Go-idiomatic for structs with 2-4 fields -- no heap allocation, no nil checks needed by callers.

**Error messages:** V4 is more consistent. All user-facing errors are capitalized, matching the spec verbatim. V5 is inconsistent -- `"at least one flag is required"` (lowercase) alongside `"Task '%s' not found"` (capitalized). V4 also provides better diagnostic differentiation in 2-1 (separate error messages for unknown commands vs invalid status) and richer no-flags help text in 2-3 (multi-line formatted usage vs a single-line list).

**Defensive coding:** V4 applies `strings.TrimSpace` on ID arguments (2-2, 2-3) and checks for duplicate `blocked_by` entries before appending (2-3). V5 does neither. The TrimSpace is minor (shells strip whitespace), but the duplicate-prevention is a meaningful correctness difference -- running `--blocks Y` twice on V5 produces `blocked_by: [X, X]`.

**Shared helpers:** V5 reuses more code across commands: `validateIDsExist`, `splitCSV`, `normalizeIDs` from `create.go` are shared with `update.go`. V4 only reuses `parseCommaSeparatedIDs`. V5's approach reduces duplication across the codebase.

**Documentation:** V5 is consistently more thorough in doc comments, explaining pointer-vs-nil semantics on option structs (2-3) and describing both success and quiet output paths. V4's docs are adequate but briefer.

### Test Quality

**Organization:** V4 creates separate top-level `Test*` functions per scenario (14 in 2-2, 16 in 2-3). V5 nests all subtests under a single top-level function (1 `TestTransition` in 2-2, 1 `TestUpdate` in 2-3). Both produce the same number of distinct test scenarios. V4's approach allows running individual scenarios more easily (`go test -run TestUpdate_BlocksFlag`); V5's is more compact and groups related tests.

**Coverage breadth:** Effectively equivalent across both versions in all three tasks. Both cover all valid transitions, all invalid transitions, all five update flags, error cases, quiet mode, normalization, and persistence.

**Assertion depth (V4 advantage):** V4 consistently verifies more fields per test. In 2-2, V4 checks `Updated` timestamp refresh after transition (V5 does not), output uses lowercase ID (V5 does not), and not-found error contains the actual task ID (V5 checks only generic "not found" text). In 2-3, V4 checks the blocked target's `Updated` timestamp is refreshed (V5 does not) and tests non-numeric priority input `"abc"` (V5 does not).

**Assertion style (V5 advantage):** V5 uses defensive pre-checks before exact assertions. In 2-1, invalid transition tests add `strings.Contains` checks before exact string match, producing clearer failure messages. In 2-1, the valid transitions table integrates a `wantClosed` column, verifying `Closed` correctness on every valid transition (7 checks) rather than only in dedicated subtests.

**Time testing:** V4 uses `time.Sleep(time.Millisecond)` for timestamp advancement in 2-1 (minimal delay, robust guarantee). V5 uses `time.Sleep(1100 * time.Millisecond)` in 2-3 for the same purpose -- a 1.1-second sleep per test invocation which is unnecessarily slow. V4's approach is strictly better here. In 2-1, V5 skips the sleep entirely and relies on a fixture timestamp being in the past, which is less robust.

**Edge cases unique to each version:** V4 tests child-blocked-by-parent rejection (2-3). V5 tests indirect 3-node dependency cycles and includes a positive acceptance test for valid blocks (2-3). Both are useful; neither is clearly more important.

### Spec Compliance

Both versions pass all acceptance criteria across all three tasks. The only spec-compliance divergence is the `"Error: "` prefix on error messages:

- **V4** includes `"Error: "` in 2-1 error strings, matching the spec verbatim (`"Error: Cannot {command} task tick-{id} -- status is '{status}'"`)
- **V5** omits the prefix in 2-1, producing `"Cannot {command} task tick-{id} -- status is '{status}'"`, which follows Go convention but deviates from the spec's exact format

Both versions produce capitalized error messages for user-facing CLI output in 2-2 and 2-3, which is appropriate since these are displayed directly to users rather than composed programmatically.

V5 has one inconsistency in 2-3: the no-flags error uses lowercase (`"at least one flag is required"`) while other errors use capitalization. V4 is uniformly capitalized throughout the phase.

### golang-pro Skill Compliance

Both versions comply with core skill constraints across all three tasks: explicit error handling, no panics, no hardcoded configuration, no ignored return values, documented exported types.

**Table-driven tests:** Both use table-driven patterns for repetitive scenarios (invalid transitions, invalid titles, invalid priorities) but neither converts all test scenarios to tables. Both are PARTIAL on this constraint -- the individual transition-path tests and individual flag tests are standalone subtests, not table entries. This is a reasonable design choice since each scenario has different setup requirements.

**Error wrapping:** Neither version consistently wraps errors with `fmt.Errorf("%w", err)`. V4 wraps title errors in 2-3 (`"invalid title: %w"`) but not other errors. V5 returns errors directly throughout. Both are PARTIAL. In most cases, direct return is defensible because the underlying errors (from `task.Transition`, `task.ValidateTitle`) already contain sufficient context.

**Functional options:** V5 uses `engine.NewStore(tickDir, ctx.storeOpts()...)` throughout the phase, aligning with the skill's preference for functional options. V4 uses `a.openStore(tickDir)`, a centralized helper method. V5 is more skill-compliant here.

## Phase Verdict
**Winner**: Tie (V4: 1 task, V5: 1 task, Tie: 1 task)

This phase is genuinely balanced. The two versions trade advantages in complementary areas, and no consistent pattern of superiority emerges across all three tasks.

V5 has the stronger architecture. Its command-map dispatch, closure-returning handlers, `Context`-based state passing, and functional options for store creation produce cleaner, more extensible code with less boilerplate. The early validation pattern in task 2-3 -- checking title and priority before opening the store -- is the single best design decision in the entire phase, demonstrating a discipline of "fail fast, fail cheap" that V4 lacks. V5 also consistently uses more idiomatic Go: value-type returns for small structs, nested maps for O(1) lookups, and shared helper functions to reduce duplication.

V4 has the stronger test assertions. Across both 2-2 and 2-3, V4 verifies fields that V5 leaves unchecked: `Updated` timestamp refresh on transition and on blocked targets, lowercase ID in output formatting, and actual task ID in not-found errors. These are not obscure edge cases -- they map directly to acceptance criteria. V4 also handles subtle correctness concerns better: duplicate `blocked_by` prevention, distinct error messages for unknown commands vs invalid status, and `strings.TrimSpace` on input IDs. These small defensive measures accumulate into a more robust implementation.

The net effect is a wash. V5's architectural advantages make it easier to maintain and extend, while V4's assertion thoroughness catches more potential regressions. Neither version has a defect -- both pass all acceptance criteria. The differences are in code quality trade-offs: V5 optimizes for implementation elegance, V4 optimizes for verification rigor. At the phase level, these strengths counterbalance each other.
