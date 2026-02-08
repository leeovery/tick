# V4 vs V2 Synthesis

## Executive Summary

V4 exceeds V2. Across 23 tasks, V4 wins 15, V2 wins 7, and 1 is a tie. V4 wins 4 of 5 phases and ties none. The margin is moderate, not decisive -- V2 remains stronger on spec fidelity, defensive coding, integration testing, and edge-case test coverage, while V4 wins on architecture, type safety, code concision, and test infrastructure quality.

V4 corrects V2's two most criticized design weaknesses from the round-1 analysis: the `interface{}` formatter parameter (replaced with concrete `StatsData`) and the `taskJSON` intermediate struct (eliminated entirely, saving 84 lines). V4 also introduces genuinely new architectural advantages that V2 never had: `readyConditionsFor(alias)` for single-source-of-truth SQL reuse, `SerializeJSONL` for in-memory hash computation, and pointer-based flag optionality for the update command.

V2's remaining advantages are real but narrow. Its integration tests (6 binary-level tests building the actual `tick` binary) verify a boundary V4 never touches. Its pervasive `NormalizeID()` at every comparison point is more defensively correct. Its exact-string test assertions catch formatting regressions that V4's `strings.Contains` checks would miss. And its `EnsureFresh` returning `*Cache` avoids a double-open inefficiency that V4's cache lifecycle introduces.

**Overall task scorecard: V4 15, V2 7, Tie 1.**

## Phase-by-Phase Results

| Phase | Winner | Margin | Key Factor |
|-------|--------|--------|------------|
| 1: Walking Skeleton (7 tasks) | V4 | 4-2-1 | 18% less impl code, type-safe test infra, better package layout |
| 2: Task Lifecycle (3 tasks) | Tie | 1-1-1 (V4 phase verdict) | V4 wins on storage-layer error design; V2 wins on domain-layer test exhaustiveness |
| 3: Dependencies (5 tasks) | V4 | 3-2 | `readyConditionsFor(alias)` -- single source of truth for ready/blocked SQL |
| 4: Output Formats (6 tasks) | V4 | 4-2 | iota enum, concrete StatsData type, dynamic alignment in PrettyFormatter |
| 5: Stats & Cache (2 tasks) | V4 | 2-0 | Type-safe formatter, blocked = open - ready, real lock exclusion test |

## Full Task Scorecard

| Task | V2 | V4 | Winner | Key Reason |
|------|-----|-----|--------|------------|
| **Phase 1** | | | | |
| 1-1 Task Model & ID | Better | -- | **V2** | `NormalizeID()` in self-reference checks; broader test coverage (20 vs 17 subtests) |
| 1-2 JSONL Storage | -- | Better | **V4** | Eliminates 84 lines of `taskJSON` boilerplate; non-nil empty slice guarantee |
| 1-3 SQLite Cache | ~ | ~ | **Tie** | V2 wins API (`EnsureFresh` returns `*Cache`); V4 wins code quality (NULL handling, error wrapping) |
| 1-4 Storage Engine | -- | Better | **V4** | `SerializeJSONL` avoids post-write re-read; 20 vs 16 test functions; cleaner package separation |
| 1-5 CLI Framework & init | Better | -- | **V2** | 6 integration tests building real binary; tests corrupted `.tick/`, unwritable dirs |
| 1-6 tick create | -- | Better | **V4** | Type-safe test setup; 24 vs 23 tests; `parseCommaSeparatedIDs` reusable helper |
| 1-7 tick list & show | -- | Better | **V4** | Type-safe test data; multiline description handling; exit code + stderr testing |
| **Phase 2** | | | | |
| 2-1 Transition validation | Better | -- | **V2** | Tests all 9 invalid transitions for no-mutation; V4 tests only 1 |
| 2-2 CLI transition cmds | -- | Better | **V4** | Type-safe test infra; exit code verification; stderr testing |
| 2-3 Update command | Better | -- | **V2** | Validate-then-apply atomicity; shared `unwrapMutationError`; broader --blocks coverage |
| **Phase 3** | | | | |
| 3-1 Dependency validation | -- | Better | **V4** | 5 focused functions vs 2 monolithic; dedicated `dependency.go`; 12 vs 11 subtests |
| 3-2 dep add/rm CLI | Better | -- | **V2** | Tests 4 edge cases V4 misses (stale ref, rm persistence, one-arg rm, unknown subcmd) |
| 3-3 Ready query | Better | -- | **V2** | Tests `"No tasks found."` per spec; tests `list --ready` alias; V4 checks TOON format instead |
| 3-4 Blocked query | -- | Better | **V4** | Genuinely reuses ready logic via `readyConditions`; V2 duplicates inverse SQL manually |
| 3-5 List filter flags | -- | Better | **V4** | `readyConditionsFor(alias)` avoids SQL ambiguity; fast-path optimization; unknown-flag error handling |
| **Phase 4** | | | | |
| 4-1 Formatter Abstraction | -- | Better | **V4** | iota enum; `DetectTTY(io.Writer)`; single `ResolveFormat` call site |
| 4-2 TOON Formatter | Better | -- | **V2** | Exact spec fidelity; strict test assertions; error propagation from `fmt.Fprintf` |
| 4-3 Pretty Formatter | -- | Better | **V4** | Dynamic alignment everywhere; more test cases; defensive `truncateTitle` |
| 4-4 JSON Formatter | -- | Better | **V4** | Concrete `StatsData` type; cleaner domain types; no `interface{}` assertion boilerplate |
| 4-5 Integration | -- | Better | **V4** | Error-to-stderr testing; formatter type assertions; existing test updates for TOON default |
| 4-6 Verbose | Better | -- | **V2** | More tests (10 vs 7); mutation verbose coverage; unit tests for logger in isolation |
| **Phase 5** | | | | |
| 5-1 Stats command | -- | Better | **V4** | Type-safe formatter; decomposed query helpers; blocked = open - ready |
| 5-2 Rebuild command | -- | Better | **V4** | Real lock exclusion test via external flock; proper ENOENT guard on `os.Remove` |

## What V4 Did Better Than V2

### 1. Type-safe formatter interface (eliminates `interface{}`)

V2's `FormatStats(w io.Writer, stats interface{})` forces a runtime type assertion in every formatter implementation -- 3 copies of identical boilerplate across TOON, Pretty, and JSON formatters. V4's `FormatStats(w io.Writer, stats StatsData)` uses a concrete type with compile-time safety. This eliminates 9 lines of duplicated assertion code and removes an entire class of runtime failure.

### 2. Eliminated `taskJSON` intermediate struct (84 fewer lines)

V2's JSONL layer requires a `taskJSON` bridge struct with `toJSON`/`fromJSON` conversion functions, duplicating the `Task` struct's field definitions. V4 co-locates JSONL serialization with the Task model in `internal/task/jsonl.go`, using the struct's own JSON tags directly. Result: 99 vs 183 implementation LOC for the same functionality.

### 3. `readyConditionsFor(alias)` -- single source of truth for SQL reuse

V4 defines ready conditions once in a parameterized function that accepts a table alias. This function is used in 4 files: `ready.go`, `blocked.go`, `list.go`, and `stats.go`. The blocked query is mathematically derived via set subtraction (`open AND NOT IN ready_set`), so any change to what makes a task "ready" automatically propagates. V2 maintains manually-inverted `readyWhere` and `blockedWhere` string constants that must be kept in sync -- a DRY violation and maintenance hazard.

### 4. `SerializeJSONL` for in-memory hash computation

V4 introduced `task.SerializeJSONL` in task 1-4, enabling hash computation from in-memory bytes without re-reading from disk. V2 must re-read the JSONL file after writing to compute the hash -- an unnecessary I/O operation on every mutation.

### 5. Pointer-based flag optionality for update

V4's `updateFlags` uses uniform `*string`/`*int` fields. V2 mixes boolean sentinels (`titleProvided bool`) with pointer fields (`priority *int`), creating an inconsistency where a sentinel-desync bug is structurally possible (value set but `Provided` flag false). V4's nil-check pattern makes this class of bug impossible.

### 6. Type-safe test infrastructure

Every V4 CLI test uses `task.Task` structs for setup and `task.ReadJSONL` for assertions. V2 writes raw JSONL strings and parses output into `map[string]interface{}` with fragile runtime type assertions like `tk["priority"].(float64)`. V4's approach catches structural errors at compile time and uses domain constants (`task.StatusOpen`) instead of string literals.

### 7. `blocked = open - ready` stats computation

V4 runs 3 database queries for stats; V2 runs 4. V4 computes `Blocked = Open - ReadyCount`, which is mathematically correct (every open task is either ready or blocked) and eliminates the need for a separate `blockedWhere` SQL fragment.

### 8. Dynamic alignment in PrettyFormatter

V4 computes column widths dynamically for stats, detail, and list output. V2 hardcodes `%2d` for stats numbers, which produces misaligned output when task counts reach 100+. This is a real correctness issue in V2.

### 9. Cleaner package architecture

V4's `task/`, `cache/`, `store/` layout follows standard Go conventions with clean one-directional imports. V2 nests `storage/jsonl/` and `storage/sqlite/` under a parent `storage/` package that imports them -- a structurally unusual Go pattern where a parent package imports its own children.

### 10. Per-concern test functions

V4's `TestCreate_PriorityFlag`, `TestShow_BlockedBySection` pattern enables targeted test execution with `go test -run`. V2's monolithic `TestCreateCommand` with 23 subtests requires `-run "TestCreateCommand/it sets priority"` -- harder to type and remember.

## What V4 Did Worse Than V2

### 1. No integration tests

V2 includes 6 binary-level integration tests (`main_test.go`, 168 lines) that build the actual `tick` binary via `exec.Command("go", "build", ...)` and test real exit codes, real stderr separation, and the complete error formatting pipeline. V4 never verifies the `main()` -> `os.Exit()` boundary. This is the single most important thing V4 is missing.

### 2. Weaker defensive ID normalization

V2 applies `task.NormalizeID()` at both sides of every ID comparison: `task.NormalizeID(tasks[i].ID) == id`. V4 normalizes only the input and assumes stored IDs are already lowercase: `tasks[i].ID == id`. V2's approach handles the case where a JSONL file is manually edited with mixed-case IDs. V4's relies on an unstated and unenforced invariant.

### 3. Spec fidelity gaps

V2 matches spec error messages verbatim (`"Could not acquire lock on .tick/lock"`). V4 uses lowercase and dynamic paths (`"failed to create .tick/ directory: %w"`), deviating from spec. V4 also introduces a `parent_title` field in TOON show output that the spec does not include, and in task 3-3 V4's tests check TOON format (`tasks[0]`) instead of the spec-required `"No tasks found."` message.

### 4. Missing edge-case test coverage

Across all phases, V2 uniquely tests: all 9 invalid transitions for no-mutation (V4 tests only 1), stale ref removal via `dep rm`, `dep rm` atomic write persistence, unknown dep subcommand handling, `tick list --ready` alias verification, contradictory filter edge case (`--status done --ready`), dependency preservation during rebuild, and mutation verbose output. These are all spec-mandated or architecturally important edge cases.

### 5. Exact-string vs substring test assertions

V2 uses exact string matching for formatter output (`want := "stats{total,...}:\n  5,3,1,..."`). V4 uses `strings.Contains` checks. V2's approach catches any formatting regression (extra whitespace, wrong newlines). V4's approach is more resilient to minor changes but could miss formatting drift. This is a systematic philosophy difference across all formatter tests.

### 6. Missing compile-time interface checks

V2 includes `var _ Formatter = &PrettyFormatter{}` in implementation files -- a standard Go idiom that catches interface violations at compile time without running tests. V4 relies on test-time checks only.

### 7. Inconsistent quiet-mode handling

V4 pushes `quiet bool` into `FormatTaskList` and `FormatDepChange` but handles it in the command handler for transitions, init, create, update, and show. V2 is consistent: quiet is always checked in the handler, formatters are never invoked in quiet mode. V4's split approach creates a maintenance burden where developers must know which pattern to follow.

### 8. Write error propagation

V2 consistently captures and returns `fmt.Fprintf` write errors in formatters. V4 systematically ignores them (`fmt.Fprintf(w, ...); return nil`). V2's approach is more correct for broken pipes or full disks, though write errors to stdout buffers are rare in practice.

### 9. Exported App fields break encapsulation

V2 keeps App fields private (`a.config.Quiet`, `a.stdout`, `a.workDir`) with a `NewApp()` constructor. V4 exports everything (`a.Quiet`, `a.Stdout`, `a.Dir`), allowing direct struct literal construction in tests but exposing internals to external callers.

### 10. `EnsureFresh` API design

V2's `EnsureFresh` returns `*Cache` for reuse, avoiding the double-open problem where V4 must open the SQLite database once for freshness check and again after the check closes it. V2's is the more efficient cache lifecycle API.

## What Stayed the Same

Both V2 and V4 share these patterns consistently:

1. **Same 5-step mutation skeleton**: Parse args -> discover tick dir -> open store -> `store.Mutate(fn)` -> format output. Neither extracts the duplicated "discover + open + defer close" preamble into a helper, despite it appearing at 9+ call sites.

2. **Same "find task by index" duplication**: Both copy-paste the `idx := -1; for i := range tasks { if tasks[i].ID == id { idx = i; break } }` loop into every Mutate callback (transition, update, dep add, dep rm). Neither extracts a `findTaskIndex` helper.

3. **Same lock acquisition duplication**: Both duplicate the full lock ceremony (`flock.New()`, `context.WithTimeout()`, `TryLockContext()`) in `Mutate`, `Query`, and `Rebuild`/`ForceRebuild` -- 3 copies each. Neither extracts a `withExclusiveLock`/`withSharedLock` helper.

4. **Same Formatter interface shape**: Both define `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage` -- the same 6 methods.

5. **Same `DiscoverTickDir` walk-up algorithm**: Both walk the filesystem upward looking for `.tick/` directories, with identical semantics.

6. **Same test volume**: Phase-level test LOC is remarkably close (V2: ~10,654 estimated; V4: ~10,347 estimated). Both maintain test-to-impl ratios above 2.5:1 consistently.

7. **Same missing cross-task interaction tests**: Neither tests the stats-rebuild interaction (corrupt cache -> rebuild -> verify stats) or concurrent mutation scenarios.

## Did the Workflow Changes Work?

### V3 regressed because of convention gravity well. Did V4 avoid this?

**Yes, definitively.** V3's regression was caused by PR #79's integration context mechanism, which documented task 1-1's unconventional choices (string timestamps, bare error returns) as "established patterns" and had the reviewer enforce consistency with them. V4 removed PR #79 entirely. The result: V4's task 1-1 uses `time.Time` timestamps, proper `NewTask` factory function, and `error` returns -- all idiomatic Go choices. No evidence of convention lock-in appears anywhere in V4's codebase. V4's early decisions are sound, and subsequent tasks build on them cleanly.

### V2 had natural course correction (6 retroactive fixes). Did V4 show similar behavior?

**Yes.** The phase reports document at least two clear examples of V4 course-correcting:

1. V4's `unwrapMutationError` pattern is absent entirely -- not because V4 forgot to add it, but because V4's storage layer was designed correctly from the start (Mutate passes callback errors through without wrapping). This means the correction was architectural rather than retroactive, which is arguably better.

2. V4 refactored `WriteJSONL` to use `SerializeJSONL` after introducing the helper in task 1-4, removing internal duplication. This mirrors V2's `ParseTasks` refactoring pattern.

V2's `unwrapMutationError` was a workaround for a Phase 1 storage design flaw, applied retroactively across 4 callers in Phase 2. V4 avoided needing the workaround at all. This suggests V4's foundational decisions were sounder than V2's in at least this dimension, reducing the need for retroactive correction.

### The polish agent was beneficial in V3. Was it beneficial in V4?

**The reports do not show clear evidence of polish agent activity in V4**, but V4's codebase exhibits the same patterns that the polish agent produced in V3: no dead code, shared helpers extracted (`parseCommaSeparatedIDs`, `taskToDetail`, `readyConditionsFor`), and consistent formatting. Whether this came from the polish agent or from better initial generation is not distinguishable from the task reports alone. The polish agent was kept in V4's workflow per the analysis-log plan, but its incremental effects are harder to isolate when the base code quality is already high.

### PR #78's structured fix recommendations -- any visible effect?

**Indirectly, yes.** PR #78 introduced FIX/ALTERNATIVE/CONFIDENCE structured output from reviewers and fix_gate_mode stop gates. V4's code quality is consistently high across all 23 tasks with no systematic defects propagating across tasks. The absence of propagating defects is consistent with the reviewer catching issues early and the stop gates preventing flawed code from merging. However, this is circumstantial -- V2 also lacked propagating defects without these mechanisms. The clearest evidence for PR #78's value is that V4 maintains quality while having PR #79 removed; the structured reviewer output from #78 may have partially compensated for the loss of integration context by providing better per-task feedback.

## Recommendations

### 1. Adopt V4 as the new baseline

V4 wins 15/23 tasks and all 5 phases. Its architectural improvements (concrete formatter types, `readyConditionsFor`, `SerializeJSONL`, pointer-based flags) are structural advantages that compound across the codebase. V2's advantages (integration tests, defensive normalization, spec fidelity) are point fixes that can be grafted onto V4's architecture.

### 2. Backport V2's integration tests

V4's most significant gap is the absence of binary-level integration tests. Add a `main_test.go` that builds the real binary and verifies exit codes, stderr separation, and the `Error:` prefix pipeline. This is V2's single strongest unique asset.

### 3. Add defensive ID normalization

V4 should normalize both sides of ID comparisons in Mutate callbacks, not just the input. This costs almost nothing and protects against manually edited JSONL files with mixed-case IDs.

### 4. Adopt exact-string test assertions for formatters

V4's `strings.Contains` approach for formatter output is too permissive. Switch to exact string matching (V2's approach) for TOON, Pretty, and JSON formatter tests. Substring checks should be reserved for tests that intentionally need format flexibility.

### 5. Add compile-time interface checks

Add `var _ Formatter = &ToonFormatter{}` (etc.) to formatter implementation files. Standard Go idiom, zero cost, catches interface drift at compile time.

### 6. Resolve quiet-mode inconsistency

Pick one approach: either always check `quiet` in the command handler before calling the formatter (V2's consistent approach) or always pass it to the formatter (V4's partial approach). Do not mix. V2's approach is simpler and recommended.

### 7. Investigate spec fidelity gaps

V4's lowercase error messages, `parent_title` TOON field, and TOON-format empty-list output deviate from the spec. Review whether these are intentional improvements or accidental drift. If the spec is authoritative, fix them.

### 8. The workflow changes are validated

The V4 experiment confirms that removing PR #79 (integration context + cohesion review) restores quality to V2 levels or better, while keeping PRs #77, #78, and #80. No further workflow changes are needed for the next implementation. The convention gravity well problem is solved by removal, not by guardrails.
