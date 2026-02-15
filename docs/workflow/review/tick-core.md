# Implementation Review: Tick Core

**Scope**: Single Plan (tick-core)
**QA Verdict**: Request Changes

## Summary

Tick Core delivers a well-architected, thoroughly tested dual-storage task tracker across 38 tasks in 8 phases. 37 of 38 tasks pass all acceptance criteria with comprehensive test coverage. One blocking issue exists: tick-core-6-7 (child-blocked-by-parent error capitalization) was not implemented as specified. The product assessment identified additional non-blocking concerns around Unicode truncation, minimal help text, and `log.Printf` bypassing injected writers.

## QA Verification

### Specification Compliance

Implementation aligns with specification across all major features: JSONL/SQLite dual storage, SHA256 freshness detection, atomic writes, file locking, all CLI commands, three output formats (TOON/Pretty/JSON), TTY auto-detection, dependency validation with cycle detection, and parent scoping via recursive CTE.

**One deviation found**:
- `dependency.go:40` — Error message uses lowercase "cannot" instead of spec-mandated "Cannot" (spec lines 407-408). Task tick-core-6-7 was specifically created to fix this but the fix was not applied.

### Plan Completion

- [x] Phase 1: Walking Skeleton — all 7 tasks complete, all acceptance criteria met
- [x] Phase 2: Task Lifecycle — all 3 tasks complete, all acceptance criteria met
- [x] Phase 3: Hierarchy & Dependencies — all 6 tasks complete, all acceptance criteria met
- [x] Phase 4: Output Formats — all 6 tasks complete, all acceptance criteria met
- [x] Phase 5: Stats & Cache Management — all 2 tasks complete, all acceptance criteria met
- [x] Phase 6: Analysis Fixes (Cycle 1) — 6/7 tasks complete; tick-core-6-7 incomplete
- [x] Phase 7: Analysis Fixes (Cycle 2) — all 5 tasks complete, all acceptance criteria met
- [x] Phase 8: Analysis Fixes (Cycle 3) — all 2 tasks complete, all acceptance criteria met
- [x] No scope creep detected

### Code Quality

No blocking code quality issues. Architecture is clean with well-defined module boundaries:
- `internal/task` — domain model, validation, transitions
- `internal/storage` — JSONL persistence, SQLite cache, Store abstraction
- `internal/cli` — command dispatch, formatters, helpers

Go conventions followed throughout: table-driven tests, interface-based formatters, error wrapping, proper use of `crypto/rand`.

### Test Quality

Tests adequately verify requirements. Coverage includes:
- Unit tests for every domain function (task model, transitions, dependency validation)
- Storage-level tests (JSONL read/write, cache rebuild, freshness detection, corruption recovery)
- CLI integration tests per command with subtests for edge cases
- Cross-format consistency tests (TOON/Pretty/JSON produce equivalent data)
- End-to-end workflow integration test exercising full task lifecycle
- Race detector verification for concurrent access

No under-testing or over-testing detected across the 38 tasks.

### Required Changes

1. **tick-core-6-7**: Fix capitalization in `internal/task/dependency.go:40` — change "cannot add dependency" to "Cannot add dependency" to match spec lines 407-408. Update test assertion at `internal/task/dependency_test.go:87` accordingly.

## Product Assessment

### Robustness

- **`log.Printf` bypasses injected writers** (`store.go:167,273,288`): Three warning messages use Go's global logger instead of the app's injected `Stderr` writer. Untestable and inconsistent with error output formatting.
- **`truncateTitle` uses byte length, not rune count** (`pretty_formatter.go:162-167`): Multi-byte characters (CJK, emoji) could be split mid-rune, producing invalid UTF-8.
- **`dep add/rm` compares IDs without normalization** (`dep.go:75,87,98,154`): Direct `==` comparison against stored IDs could fail on hand-edited JSONL with mixed-case IDs. Other commands consistently use `NormalizeID()`.
- **`Query` parses full JSONL then discards result** (`store.go:230-243`): Read path parses all tasks into `[]task.Task` for freshness check, then discards them. Hash-only check would halve CPU cost on reads.

### Gaps

- **Minimal help text** (`app.go:203-208`): `printUsage()` only lists "init". All other commands undiscoverable from CLI. No `--help`/`-h` support.
- **No per-command help**: Unknown flags silently ignored. Typos like `--priorty` produce no error.
- **`StubFormatter` in production code** (`format.go:148-171`): Phase 1 scaffolding artifact never used in production paths. Should move to test file.
- **No `--blocked-by` on `tick update`**: Asymmetric with `tick create` which supports both `--blocked-by` and `--blocks`.

### Strengthening Opportunities

1. Route `log.Printf` through injected writers (testability + consistency)
2. Fix `truncateTitle` for Unicode (correctness)
3. Add comprehensive help text (human usability)
4. Enable WAL mode for SQLite (concurrent read performance)
5. Add hash-only freshness check for read path (performance)

### What's Next

- Complete the one remaining required change (capitalization fix)
- Consider `tick delete` / `tick archive` for task cleanup over time
- `doctor-validation` plan exists separately for diagnostics
- Help text additions for human usability

## Recommendations

1. Fix the one blocking issue (capitalization) — trivial change, spec compliance
2. Route `log.Printf` through injected writers — small effort, high testability impact
3. Fix Unicode `truncateTitle` — small effort, correctness impact
4. Add help text — moderate effort, essential for human adoption
5. Move `StubFormatter` to test file — trivial cleanup
