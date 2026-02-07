# Final Synthesis: Tick Implementation Comparison

## Executive Summary

This report synthesizes the analysis of three independent implementations (V1, V2, V3) of `tick`, a Go CLI task tracker, across 23 tasks organized in 5 phases. Each implementation was built from the same specification by different Claude agent configurations: V1 (sequential, single-pass), V2 (agent-based, second iteration), and V3 (agent-based, third iteration).

**V2 is the decisive overall winner**, taking 21 of 23 tasks outright and sharing credit on one more (task 2-2). V2's dominance is not marginal -- it wins on spec compliance, test thoroughness, Go idioms, error handling, and architectural coherence simultaneously. Its core advantage is a compounding one: early architectural decisions (sub-package layout, `time.Time` timestamps, store-injected logging, composable SQL fragments) created infrastructure that later tasks consumed cleanly. By Phase 5, V2's stats command could reuse Phase 3's SQL constants, its verbose logging fired from inside store operations, and its formatter interface handled nested JSON correctly -- all because foundations laid in Phases 1-2 were sound.

**V3 is a consistent second place**, winning 1 task outright (1-5 CLI framework) and sharing task 2-2. V3 demonstrates that the agent-based approach produces meaningfully better code than sequential generation, but its early design choices -- string timestamps, CLI-level verbose logging, string-returning formatters -- created accumulating friction. By Phase 4, V3's `FormatStats() string` signature forced full output buffering instead of streaming, and by Phase 5, its CLI-level verbose messages described store internals that hadn't yet executed. V3's test suites are strong (second only to V2 in every phase), and its code organization via exported constants (`ReadyCondition`, `BlockedCondition`) shows good separation of concerns.

**V1 is clearly third**, finishing last in 20 of 23 tasks. V1's sequential generation produced working code but with consistently weaker architecture: flat package structure, inline SQL duplication, substring-based test assertions, and a critical dead-code defect in Phase 1 (list/show commands registered in the dispatcher map but never wired to the CLI). V1's strongest showing is task 2-2 (CLI transitions), where its implementation -- though not its tests -- was judged best. V1 also has the lowest test LOC in every phase and the weakest assertion strategies throughout.

## Phase-by-Phase Results

| Phase | Name | V1 | V2 | V3 | Winner |
|-------|------|-----|-----|-----|--------|
| 1 | Walking Skeleton (7 tasks) | 3rd | **1st (6/7)** | 2nd (1/7) | V2 |
| 2 | Task Lifecycle (3 tasks) | 3rd | **1st (2.5/3)** | 2nd (0.5/3) | V2 |
| 3 | Dependencies (5 tasks) | 3rd | **1st (5/5)** | 2nd | V2 |
| 4 | Output Formats (6 tasks) | 3rd | **1st (6/6)** | 2nd | V2 |
| 5 | Stats & Cache (2 tasks) | 3rd | **1st (2/2)** | 2nd | V2 |

V2 wins all 5 phases. V3 is second in all 5. V1 is third in all 5. The margin varies -- Phase 1 is closest (V3 wins task 1-5), while Phases 3-5 are V2 clean sweeps.

## Full Task Scorecard

Rating scale: Strong (clear winner with significant advantages), Moderate (winner with meaningful but narrower advantages), Slight (winner by small margin or mixed dimensions).

| Task | Description | V1 | V2 | V3 | Winner | Margin |
|------|-------------|-----|-----|-----|--------|--------|
| **Phase 1: Walking Skeleton** | | | | | | |
| 1-1 | Task model & ID generation | 3rd | **1st** | 2nd | V2 | Moderate |
| 1-2 | JSONL storage engine | 3rd | **1st** | 2nd | V2 | Moderate |
| 1-3 | SQLite cache layer | 3rd | **1st** | 2nd | V2 | Moderate |
| 1-4 | Store integration | 3rd | **1st** | 2nd | V2 | Strong |
| 1-5 | CLI framework & dispatch | 3rd | 2nd | **1st** | V3 | Moderate |
| 1-6 | `tick create` command | 3rd | **1st** | 2nd | V2 | Moderate |
| 1-7 | `tick list` / `tick show` | 3rd | **1st** | 2nd | V2 | Strong |
| **Phase 2: Task Lifecycle** | | | | | | |
| 2-1 | Status transition validation | 3rd | **1st** | 2nd | V2 | Strong |
| 2-2 | CLI transition commands | **1st** (impl) | 2nd | **1st** (tests) | Mixed | Slight |
| 2-3 | `tick update` command | 3rd | **1st** | 2nd | V2 | Strong |
| **Phase 3: Dependencies** | | | | | | |
| 3-1 | Dependency validation | 3rd | **1st** | 2nd | V2 | Moderate |
| 3-2 | `tick dep add/rm` commands | 3rd | **1st** | 2nd | V2 | Strong |
| 3-3 | Ready query (`tick ready`) | 3rd | **1st** | 2nd | V2 | Moderate |
| 3-4 | Blocked query | 3rd | **1st** | 2nd | V2 | Moderate |
| 3-5 | List filters & composition | 3rd | **1st** | 2nd | V2 | Strong |
| **Phase 4: Output Formats** | | | | | | |
| 4-1 | Formatter abstraction | 3rd | **1st** | 2nd | V2 | Strong |
| 4-2 | TOON formatter | 3rd | **1st** | 2nd | V2 | Moderate |
| 4-3 | Pretty formatter | 3rd | **1st** | 2nd | V2 | Moderate |
| 4-4 | JSON formatter | 3rd | **1st** | 2nd | V2 | Strong |
| 4-5 | Formatter integration | 3rd | **1st** | 2nd | V2 | Strong |
| 4-6 | Verbose output | 3rd | **1st** | 2nd | V2 | Strong |
| **Phase 5: Stats & Cache** | | | | | | |
| 5-1 | `tick stats` command | 3rd | **1st** | 2nd | V2 | Strong |
| 5-2 | `tick rebuild` command | 3rd | **1st** | 2nd | V2 | Moderate |

**Summary: V2 wins 21/23 tasks outright. V3 wins 1/23 outright (1-5). Task 2-2 is split (V1 implementation, V3 tests). V1 wins 0/23 outright.**

## Version Profiles

### V1: Sequential Generation

**Architecture:** Flat package structure with all application code in a single `internal/app` package. Uses `time.Time` for timestamps (the correct choice). CLI dispatch via a hand-rolled `map[string]func()` dispatcher. Persistent `*sql.DB` cache stored on the `Store` struct. Lock helpers (`acquireSharedLock`/`acquireExclusiveLock`) are well-factored and shared across operations.

**Strengths:**
- **Lock helper DRY (Phase 5):** V1's `acquireExclusiveLock()` and `acquireSharedLock()` are the most well-factored lock abstractions across all versions. Reused in `Mutate`, `Query`, and `ForceRebuild` without duplication.
- **Error wrapping consistency:** V1 uses a consistent gerund-prefix style (`"querying status counts: %w"`, `"closing cache: %w"`) throughout. While less informative than V2's approach, it is internally consistent.
- **Task 2-2 implementation:** V1's CLI transition commands were judged the best implementation (though not the best tests) in the only task where V1 placed first on any dimension.
- **`time.Time` timestamps (Phase 1):** V1 correctly uses Go's `time.Time` type for `CreatedAt`/`UpdatedAt`, providing type safety that V3 lacks.

**Weaknesses:**
- **Critical dead-code defect (task 1-7):** The `list` and `show` commands are defined in the dispatcher map but never registered with the CLI flag parser. They exist as unreachable code. This is the single most serious defect found in any version across all 23 tasks. (Phase 1 report, task 1-7 analysis)
- **Flat package structure:** All application code in one package means no compile-time enforcement of layer boundaries. Store, CLI, model, and formatter code can freely cross-reference without import restrictions.
- **Inline SQL duplication (Phase 3, Phase 5):** Ready/blocked SQL logic is duplicated between `list.go` and `stats.go` rather than shared via constants. The Phase 3-to-Phase 5 query reuse pipeline required by the spec is broken in V1.
- **Weak test assertions:** V1 consistently uses `strings.Contains` substring matching where V2 and V3 use JSON parsing or exact string matching. This means V1's tests can pass even when output format is wrong, as long as expected substrings appear somewhere. (Total test LOC: 261 in Phase 5 vs V2's 766)
- **SQL injection vulnerability (Phase 3):** V1 uses `fmt.Sprintf` for SQL query construction in at least one location, creating a potential SQL injection vector. V2 and V3 use parameterized queries exclusively.
- **Stateful cache lifecycle (Phase 5):** The persistent `s.cache` field on `Store` means rebuild must close and reassign it, creating a window where a concurrent stats call on the same Store instance would crash.
- **No TTY detection (task 1-5):** V1 is the only version that completely omits TTY detection, a spec requirement for auto-selecting output format.
- **Blocks dedup bug (task 2-3):** V1's update command does not deduplicate the `blocks` field, allowing duplicate entries to accumulate.

**Test profile:** Lowest test LOC in every phase. 15 tests / 261 LOC in Phase 5 (vs V2's 16 / 766). Test-to-implementation ratio consistently around 1.3-1.7:1. Relies on substring assertions and CLI output checking rather than structured verification.

### V2: Agent-Based (Second Iteration)

**Architecture:** Sub-package organization with clear layer separation: `internal/model`, `internal/storage`, `internal/cli`, `internal/format`. Uses `time.Time` timestamps. CLI dispatch via a `newStore` helper that centralizes store creation, logger wiring, and error wrapping. Per-operation cache lifecycle (local cache instances opened and closed within each `Query`/`Mutate`/`ForceRebuild` call). Store-injected `Logger` interface for verbose output.

**Strengths:**
- **Spec compliance:** V2 is the only version that fully complies with the specification across all 23 tasks. Notable spec-compliance wins: correct JSON nesting with separate `by_status` and `workflow` keys (tasks 4-4, 5-1), exact error message casing matching spec examples (task 1-4), `tick ready` as a true alias for `tick list --ready` (task 3-3), and correct right-aligned stats in pretty format (task 4-3).
- **Sub-package architecture:** Layer boundaries enforced at compile time via Go's import system. `internal/cli` cannot reach `internal/storage` internals without going through the exported API. This prevents the kind of coupling that accumulates in V1's flat structure.
- **Composable SQL fragments (Phase 3, task 3-5):** V2 decomposes queries into reusable WHERE fragments (`readyWhere`, `blockedWhere`) and a pure `buildListQuery` function. The stats command (Phase 5) directly reuses these constants: `SELECT COUNT(*) FROM tasks WHERE ` + `readyWhere`. This is the cleanest DRY chain across the entire codebase.
- **Store-injected verbose logging (task 4-6):** V2's `Logger` interface is injected into the store via `SetLogger()`. Verbose messages fire from inside `ForceRebuild` and `Query` at the exact moment each step occurs (7 distinct messages in rebuild). If an operation fails, only the messages for completed steps appear. This is architecturally correct and the only version where verbose output accurately reflects what happened.
- **Per-operation cache isolation (Phase 5):** Both stats and rebuild open/close their own cache instances. Zero shared mutable state between operations. A failed rebuild cannot corrupt a concurrent stats query's cache reference.
- **Test thoroughness:** Highest test LOC in every phase. 766 LOC in Phase 5 alone. Tests use JSON parsing for structured output verification, direct SQLite queries for rebuild verification, and exact string matching for formatter output. V2 uniquely tests dependency preservation after rebuild (task 5-2) and JSON key structure including the `workflow` nested object (task 5-1).
- **Cross-task refactoring:** V2 is the only version that retroactively improves earlier code when later tasks reveal opportunities. The `unwrapMutationError` helper (task 2-3) was extracted to serve both `update` and `transition` commands, reducing duplicate error-handling code. `ParseTasks` (task 1-4) was refactored from the original JSONL reader to serve both storage and test setup.
- **`*int` priority handling (task 1-6):** V2 uses `*int` for optional priority, distinguishing "not set" (nil) from "set to zero" -- semantically correct for the domain.
- **Rune-aware title validation (task 1-1):** V2 validates title length using `utf8.RuneCountInString()`, correctly handling multi-byte Unicode characters.

**Weaknesses:**
- **Duplicated lock boilerplate (Phase 5):** V2 duplicates the full lock acquisition ceremony (`flock.New()`, `context.WithTimeout()`, `TryLockContext()`) in `Mutate`, `Query`, and `ForceRebuild`. Three copies of essentially the same pattern. V1's helper methods are better factored here.
- **`interface{}` formatter parameter:** V2's formatter methods accept `interface{}` for the data parameter, losing compile-time type safety. A runtime type assertion is needed inside each formatter method. V3's approach (concrete types) is safer here.
- **Higher code volume:** V2's thoroughness comes with more code. This is generally a strength (more tests, better coverage) but does increase maintenance surface area.

**Test profile:** Highest test LOC in every phase. 16 tests / 766 LOC in Phase 5. Test-to-implementation ratio consistently 3.5-4.1:1. Uses JSON-parsed assertions, direct SQLite queries, exact string matching, and structural verification (key existence, nesting depth).

### V3: Agent-Based (Third Iteration)

**Architecture:** Sub-package organization similar to V2 but with different internal conventions. Uses **string timestamps** (a significant early design choice). CLI dispatch returns `int` exit codes rather than `error`. Formatter methods return `string` rather than writing to `io.Writer`. CLI-level verbose logging via `WriteVerbose()` calls in command handlers. Stored `*flock.Flock` on the `Store` struct (one instance, reused across operations). Exported SQL constants (`ReadyCondition`, `BlockedCondition`) in dedicated files.

**Strengths:**
- **CLI framework (task 1-5):** V3 wins the only task where V2 does not place first. V3's testable `IsTTY` function and flexible flag parsing design make it the most testable CLI layer. V3 is also the only version with proper TTY detection integration at the CLI level.
- **Exported SQL constants (Phase 3):** `ReadyCondition` in `ready.go` and `BlockedCondition` in `blocked.go` are well-organized, exported constants that any package can reference. The organization into dedicated files provides clear discoverability.
- **Concrete formatter types:** V3's formatter methods accept concrete Go types rather than `interface{}`, providing compile-time type safety that V2 lacks. If the data structure changes, V3 gets a compile error; V2 gets a runtime panic.
- **SHA256 hash validation test (task 5-2):** V3 uniquely validates that the hash stored after rebuild is exactly 64 hex characters, verifying the hash algorithm choice.
- **Strong test suite (second only to V2):** V3's tests are meaningfully more thorough than V1's in every phase, using structured assertions and comprehensive scenario coverage.

**Weaknesses:**
- **String timestamps (Phase 1, propagating):** V3 stores `CreatedAt`/`UpdatedAt` as `string` rather than `time.Time`. This loses type safety, allows malformed timestamps to exist without detection, and pushes parsing to every consumer. This decision was made in task 1-1 and its cost compounds through all subsequent phases.
- **CLI-level verbose logging (Phase 4-5):** V3's `WriteVerbose()` calls fire from the CLI command handler before the store operation executes. If `store.Rebuild()` fails to acquire its lock, the output will still say "lock acquire exclusive" and "delete existing cache.db" -- describing actions that never happened. This architectural flaw is a direct consequence of not injecting a logger into the store layer. (Phase 5 report, rebuild analysis)
- **String-returning formatters (Phase 4):** `FormatStats() string` requires buffering the entire output in memory before writing, breaking Go's standard `io.Writer` streaming pattern. For small outputs this is not a performance issue, but it is an anti-pattern that would not scale and contradicts Go's composable I/O philosophy.
- **No `workflow` JSON key (tasks 4-4, 5-1):** V3 merges workflow data into the `by_status` object rather than providing a separate `workflow` key as the spec requires. This is the single most significant spec deviation in V3.
- **Implicit table alias coupling (Phase 3-5):** V3's SQL conditions assume a table alias `t` (`FROM tasks t`). Any query using these constants must use the same alias, but this requirement is implicit -- not enforced by the type system or documented in the constant's definition.
- **12+ duplicate `fmt.Fprintf` calls (Phase 2):** V3 distributes error formatting across many call sites rather than centralizing it. The same `fmt.Fprintf(a.Stderr, "Error: %v\n", err)` pattern appears in 12+ locations. V2 centralizes this via `unwrapMutationError`.
- **Bare error returns (Phase 5):** V3's `queryStats` function returns bare errors without wrapping, producing messages like `sql: no rows in result set` with no indication of which operation failed. V1 and V2 wrap errors with context.
- **`int` return convention:** Returning `int` exit codes from command handlers instead of `error` breaks Go's standard error propagation pattern and makes it impossible to wrap or inspect errors programmatically.

**Test profile:** Second highest test LOC in every phase. 16 tests / 682 LOC in Phase 5. Test-to-implementation ratio consistently 3.0-3.5:1. Uses JSON parsing and structured assertions. V3 uniquely tests all 6 verbose step messages (task 5-2) and SHA256 hash length (task 5-2).

## Comparative Patterns

### Where V1 Outperforms Agent-Based Versions

V1's advantages are narrow but real:

1. **Lock helper abstraction (Phase 5):** V1's `acquireSharedLock()` / `acquireExclusiveLock()` helper methods on `Store` are the best-factored lock abstractions across all versions. V2 duplicates the full lock ceremony three times; V3 shares the `flock` instance but duplicates the acquisition ceremony. V1 solves this cleanly with two reusable methods.

2. **Task 2-2 implementation quality:** V1's CLI transition commands were judged to have the best implementation (though V3 had better tests). This is V1's only task-level win on any dimension.

3. **Simplicity of single-package approach (early phases):** In Phases 1-2, V1's flat structure meant less boilerplate -- no package declarations, no import management across packages. This simplicity becomes a liability by Phase 3 when cross-cutting concerns (SQL reuse, error wrapping) need clear boundaries.

4. **Error wrapping consistency:** V1 maintains a consistent gerund-prefix wrapping style throughout. While V2's messages are more informative, V1's are more internally consistent than V3's (which mixes wrapped and bare returns).

These advantages are insufficient to overcome V1's structural deficits. The dead-code bug (task 1-7), SQL injection risk (Phase 3), inline SQL duplication, and weak test assertions collectively place V1 firmly in third.

### Where Agent-Based (V2/V3) Outperforms Sequential (V1)

The agent-based approach produces measurably better code across every dimension:

1. **Test volume and quality:** V2 averages 3.5-4.1x test-to-implementation ratio; V3 averages 3.0-3.5x; V1 averages 1.3-1.7x. This is the single largest measurable gap between approaches. Agent-based versions consistently use JSON-parsed assertions, direct database verification, and structural checks. V1 uses substring matching throughout.

2. **Package organization:** Both V2 and V3 use sub-package layouts with compile-time layer enforcement. V1's flat structure allows unrestricted cross-referencing between layers.

3. **Cross-task refactoring:** V2 (and to a lesser extent V3) retroactively improve earlier code when later tasks reveal opportunities. V1 shows no evidence of revisiting completed work. V2's `unwrapMutationError` (task 2-3) and `ParseTasks` refactoring (task 1-4) are examples of this compounding improvement pattern.

4. **Spec compliance:** V2 matches all spec requirements across 23 tasks. V3 misses the `workflow` JSON key. V1 misses TTY detection, has the dead-code CLI bug, produces flat JSON stats, and has sparse verbose logging. The sequential approach appears to lose track of requirements more readily.

5. **SQL safety:** Both V2 and V3 use parameterized queries exclusively. V1 uses `fmt.Sprintf` for SQL construction in at least one location.

### V2 vs V3: Where the Agent Iterations Diverge

V2 and V3 share the agent-based approach but diverge on several foundational choices made in Phase 1 that compound through later phases:

| Decision Point | V2 | V3 | Impact |
|---------------|-----|-----|--------|
| **Timestamps** | `time.Time` | `string` | V3 loses type safety, pushes parsing to consumers |
| **Verbose logging** | Store-injected `Logger` interface | CLI-level `WriteVerbose()` | V3 produces misleading output on failure |
| **Formatter signature** | `Format(io.Writer, data)` | `Format(data) string` | V3 breaks Go streaming patterns |
| **Error returns** | `error` from commands | `int` exit code from commands | V3 loses error introspection |
| **Formatter data type** | `interface{}` | Concrete types | V3 wins on type safety |
| **CLI testability** | Good | Best (testable `IsTTY`) | V3 wins on CLI testing |
| **SQL constant location** | Same file as usage (`list.go`) | Dedicated files (`ready.go`, `blocked.go`) | V2 proximity aids DRY visibility; V3 separation aids discoverability |

The pattern is clear: V2 makes conventionally idiomatic Go choices (type-safe timestamps, `io.Writer` streaming, `error` returns), while V3 makes unconventional choices that occasionally win on a specific dimension (concrete formatter types, testable TTY) but accumulate costs across the broader system.

V2's single architectural weakness relative to V3 -- the `interface{}` formatter parameter -- is real but contained. It affects one method signature per formatter. V3's weaknesses (string timestamps, CLI verbose, string formatters) affect every consumer of those abstractions across the entire codebase.

## Code Quality Summary

### Spec Adherence

| Dimension | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| JSON stats nesting (`by_status` + `workflow`) | Flat (fails) | Correct | Merged (fails) |
| Error message casing (per spec examples) | Inconsistent | Exact match | Close but not exact |
| `tick ready` as list alias | Separate command | True alias (correct) | Separate query function |
| TTY detection | Missing entirely | Present | Present and testable |
| Verbose output accuracy | Sparse but honest | Detailed and accurate | Detailed but misleading on failure |
| Ready/blocked SQL reuse | Duplicated inline | Shared constants (DRY) | Shared constants (DRY, with alias coupling) |
| Dead code | list/show unreachable | None detected | None detected |

**V2 is the only version with full spec compliance across all 23 tasks.**

### Test Thoroughness

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Phase 5 test LOC | 261 | **766** | 682 |
| Phase 5 test-to-impl ratio | 1.5:1 | **4.0:1** | 3.3:1 |
| Assertion strategy | Substring (`strings.Contains`) | JSON parsing + exact match | JSON parsing + structured |
| Unique test contributions | None notable | Dependency preservation, JSON key validation, right-alignment check | SHA256 hash length, all verbose messages |
| Cross-task interaction tests | None | None | None |
| Coverage gaps | Hash verification, JSON nesting | None identified | `workflow` key verification |

No version tests the interaction between stats and rebuild (e.g., corrupt cache -> rebuild -> verify stats). This is the most significant shared coverage gap.

### Go Idioms

| Pattern | V1 | V2 | V3 |
|---------|-----|-----|-----|
| `io.Writer` for output | Yes | Yes | No (string return) |
| `error` return convention | Yes | Yes | No (`int` exit code) |
| `time.Time` for timestamps | Yes | Yes | No (`string`) |
| `context.Context` propagation | Basic | Full | Full |
| Interface satisfaction | Implicit (standard) | Implicit (standard) | Implicit (standard) |
| Functional options / config | Minimal | Moderate | Moderate |
| Package organization | Flat (anti-pattern for larger projects) | Sub-packages (idiomatic) | Sub-packages (idiomatic) |

V2 is the most idiomatically Go-like. V3 deviates on three fundamental Go patterns (Writer, error, time). V1 gets the types right but the structure wrong.

### Error Handling

| Pattern | V1 | V2 | V3 |
|---------|-----|-----|-----|
| Wrapping style | Gerund prefix (`"closing cache: %w"`) | Past tense (`"failed to query: %w"`) | Mixed / minimal |
| Consistency | High (one style) | High (one style) | Low (some wrapped, some bare) |
| Context in messages | Moderate (operation name) | High (operation + flag context) | Low (bare returns in `queryStats`) |
| Centralized error formatting | No | Yes (`unwrapMutationError`) | No (12+ duplicate `fmt.Fprintf`) |
| `%w` usage for unwrapping | Consistent | Consistent | Inconsistent |

V2's error handling is the most informative and maintainable. V1 is consistent but less informative. V3 is the weakest, with bare returns that lose operational context.

### Architecture Decisions: Cumulative Impact

The most consequential finding of this analysis is that early architectural decisions compound. The table below traces three Phase 1 decisions through their downstream impact:

**Decision: Timestamp representation (task 1-1)**
- V1/V2 (`time.Time`): Clean comparison, sorting, and formatting throughout. No parsing code needed at consumer sites.
- V3 (`string`): Every consumer that needs to compare, sort, or format timestamps must parse the string first. Error handling for malformed timestamps is pushed to every call site.

**Decision: Verbose logging architecture (task 4-6)**
- V2 (store-injected `Logger`): Rebuild verbose output (Phase 5) accurately describes what happened. 7 messages fire at execution time inside the store.
- V3 (CLI-level `WriteVerbose`): Rebuild verbose output describes what the CLI *intends* to happen. If the store operation fails early, messages for uncompleted steps have already been printed.

**Decision: Formatter return type (task 4-1)**
- V2 (`io.Writer` parameter): Stats output streams to any writer. Composable with buffered writers, test capture, network connections.
- V3 (`string` return): Stats output is fully buffered in memory. Cannot stream. Breaks composition with Go's `io.Writer` ecosystem.

Each of these decisions was made once, early, and shaped everything that followed. V2's decisions aligned with Go conventions and scaled well. V3's decisions created increasingly visible friction as the codebase grew.

## Final Ranking

**1st: V2** -- Wins 21/23 tasks. Full spec compliance. Highest test quality. Most idiomatic Go. Best cross-task refactoring. Sound architectural foundations that compound positively through all 5 phases.

**2nd: V3** -- Wins 1/23 tasks (1-5 CLI framework). Strong tests (second in every phase). Good package organization. Undermined by three early design choices (string timestamps, CLI verbose, string formatters) whose costs accumulate through Phases 3-5.

**3rd: V1** -- Wins 0/23 tasks outright. Critical dead-code defect. SQL injection risk. Lowest test quality. Best lock helper abstraction (a narrow win). Sequential generation produces functional but architecturally weaker code compared to agent-based approaches.
