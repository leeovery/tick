# Phase 5: Stats & Cache Management

## Task Scorecard

| Task | Winner | Key Differentiator |
|------|--------|--------------------|
| 5-1 (tick stats) | V4 | Mathematically sound blocked-count derivation; significantly more thorough format and JSON tests |
| 5-2 (tick rebuild) | V5 | DRY lock management via `acquireExclusive()`; functional options; superior package separation and dispatch architecture |

## Phase-Level Patterns

### Architecture

The two tasks reveal a consistent architectural split between V4 and V5. V4 uses method-on-struct (`*App`) with a switch-based dispatcher, while V5 uses package-level functions with a map-based dispatcher (`*Context`). V5's map dispatch is objectively more extensible (one-line additions vs. new case blocks).

V5 demonstrates better DRY principles at the store layer: the `acquireExclusive()` helper eliminates ~15 lines of lock boilerplate duplicated across three methods in V4. This is the strongest architectural win across the phase. However, V5's persistent `*cache.Cache` design introduces lifecycle complexity in rebuild (close/delete/reopen/reassign), whereas V4's open-per-operation model makes rebuild trivially safe.

V5 consistently separates concerns better at the package level: `internal/engine` for the store, `internal/storage` for serialization, `internal/task` for the domain model. V4 mixes serialization into the `task` package.

On the stats side, V4's choice to derive `Blocked = Open - readyCount` is architecturally stronger than V5's two independent queries. V4's approach guarantees `Ready + Blocked = Open` as a mathematical tautology, eliminating an entire class of consistency bugs. V5's separate `readyWhereClause`/`blockedWhereClause` can silently diverge.

### Code Quality

Both implementations produce clean, idiomatic Go. Error wrapping is consistent in both, though V5 uses the slightly more idiomatic lowercase gerund form (`"querying status counts: %w"`) versus V4's `"failed to query status counts: %w"`.

V5 has one concrete defect: in the rebuild CLI handler, the `FormatMessage` return value is silently discarded. V4 correctly propagates this error. This is a real bug (broken pipe scenarios) and a direct violation of the golang-pro skill.

V5 uses functional options (`WithVerbose`, `WithLockTimeout`) for store construction, which is more idiomatic than V4's public field injection (`s.LogFunc = a.vlog.Log`). V5's `VerboseLogger` separates `Log(string)` from `Logf(string, ...interface{})`, which is cleaner than V4's single variadic method that always runs through `fmt.Sprintf`.

V4 has a minor organizational advantage: `StatsData` lives in `format.go` alongside the `Formatter` interface. V5 places it in `toon_formatter.go`, which is the wrong home for a format-agnostic data struct.

### Test Quality

This is where the two tasks diverge most sharply and where the phase verdict becomes nuanced.

**Task 5-1 (stats)**: V4's tests are materially stronger. Format-specific tests verify actual output values (exact TOON data rows, all P0-P4 labels, typed JSON deserialization with compile-time safety). V5's format tests only check that headers/labels exist, missing value-level bugs entirely. V4's status test uses 7 tasks (multiple per status) versus V5's 5, better exercising GROUP BY aggregation. V4's typed `jsonStats` struct for JSON deserialization catches structural changes at compile time; V5 uses `map[string]interface{}` with fragile `float64` assertions.

**Task 5-2 (rebuild)**: The picture is mixed. V4's lock test uses an actual external flock to prove mutual exclusion fails under contention -- a genuine concurrency test. V5 only checks verbose log output, proving logging works but not that locking actually happens. V4's empty JSONL test verifies actual database state; V5 only checks output strings. However, V5's verbose log verification is more precise (exact phrase list in a loop vs. ad-hoc lowercase substring checks), and V5 uses cleaner task construction via `NewTask()`.

Across both tasks, neither version uses true table-driven testing. V4 splits tests into separate top-level functions; V5 groups under a single parent with subtests (closer to golang-pro guidance). Both approaches are defensible given the varied setup logic per scenario.

### Spec Compliance

Both V4 and V5 achieve full spec compliance across both tasks. All acceptance criteria are met. Output formats, quiet/verbose flags, edge cases, and data correctness all pass. No spec deviations in either version.

### golang-pro Skill Compliance

Both versions are largely compliant with partial marks on table-driven testing (neither uses true table-driven patterns, though the varied setup logic per test justifies individual subtests).

V5 has one clear violation: the swallowed `FormatMessage` error in the rebuild CLI handler violates "Handle all errors explicitly" and "No ignored errors without justification." V4 has no such violations.

Neither version accepts `context.Context` on the `Rebuild()` method itself, though both use context internally for lock acquisition. This is a minor shared gap.

## Phase Verdict

**Winner: Tie (V4: 1, V5: 1)**

Phase 5 splits evenly between two tasks that reward fundamentally different engineering priorities. Task 5-1 (stats) goes to V4 on the strength of its mathematically sound blocked-count derivation and significantly more thorough test coverage -- advantages that directly impact correctness guarantees and defect detection. Task 5-2 (rebuild) goes to V5 for its superior DRY architecture, functional options, and cleaner package separation -- advantages that impact long-term maintainability and extensibility.

The phase reveals a consistent tension: V4 tends to produce more rigorous tests and safer correctness invariants, while V5 tends to produce better-structured, more maintainable production code. V4's blocked derivation (`Blocked = Open - readyCount`) eliminates an entire class of consistency bugs by construction, which is a meaningful safety advantage. V5's `acquireExclusive()` helper eliminates real boilerplate duplication across three store methods, which is a meaningful maintenance advantage.

Neither version is without flaws. V5's swallowed `FormatMessage` error is a concrete bug and golang-pro violation. V4's lock boilerplate duplication across `Mutate`, `Query`, and `Rebuild` is a DRY violation that compounds as the codebase grows. V5's weak lock test (log checking instead of actual contention testing) is a testing gap. V4's format-agnostic `StatsData` is better placed than V5's, but V5's package-level separation (`engine`/`storage`/`task`) is cleaner overall. These cross-cutting strengths and weaknesses balance out to a genuine tie at the phase level.
