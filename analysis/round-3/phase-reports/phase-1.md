# Phase 1: Walking Skeleton -- Init, Create, List, Show

## Task Scorecard

| Task | Winner | Key Differentiator |
|------|--------|--------------------|
| 1-1  | V5     | Custom ISO 8601 JSON marshaling, case-insensitive self-reference detection, elimination of all magic values |
| 1-2  | V5     | Separate `storage` package, `json.NewEncoder` idiom, better test organization (10 top-level functions), empty-task-list edge case |
| 1-3  | V5     | DRY timestamp formatting via `task.FormatTimestamp()`, practical `EnsureFresh` return type `(*Cache, error)`, superior transaction rollback test |
| 1-4  | V5     | Spec-correct lock timeout error message, DRY lock acquisition helpers, persistent cache connection, functional options pattern |
| 1-5  | V5     | Map-based command dispatch, proper `os.Getwd` error handling, type-safe `DetectTTY(*os.File)`, corrupted `.tick/` edge case test |
| 1-6  | V4     | Significantly broader test coverage: timestamp verification, cache.db existence, non-numeric priority, `--blocks` timestamp update, two-task uniqueness |
| 1-7  | V5     | Tests verify spec-prescribed `--pretty` format rather than TOON, correct section omission compliance, contradictory filter edge case |

## Phase-Level Patterns

### Architecture

V5 consistently demonstrates stronger architectural decisions across all seven tasks. Three patterns stand out:

**Separation of concerns.** V5 places JSONL storage in `internal/storage/` (separate from the domain model in `internal/task/`), the storage engine in `internal/engine/`, and CLI in `internal/cli/`. V4 co-locates storage functions with the domain model in `internal/task/` and uses `internal/store/` for the engine. V5's separation becomes visible in Tasks 1-2 and 1-3 where the cache package reuses `task.FormatTimestamp()` instead of duplicating the timestamp format string -- a direct benefit of clean package boundaries.

**Thin constructors with explicit composition.** V5's `NewTask(id, title) Task` does one thing: set defaults. Validation, ID generation, and field assignment are independent operations the caller composes. V4's `NewTask(title, opts, exists) (*Task, error)` bundles generation, validation, and construction into a single call. V5's approach aligns with Go's preference for small, composable functions and makes testing each concern independently straightforward.

**Map-based command dispatch.** V5 registers commands in a `map[string]func(*Context) error`, adding each new command with a single line. V4 uses a switch statement that requires 6-12 lines of boilerplate per command. By Task 1-7, V5's cli.go modifications total 6 lines across 3 tasks (init, create, list/show) while V4's total 30+ lines. This advantage compounds.

### Code Quality

**Error message style.** V5 uses gerund-based wrapping (`"creating temp file: %w"`, `"querying task: %w"`) across all tasks. V4 uses `"failed to ..."` prefix style (`"failed to create temp file: %w"`, `"failed to query task: %w"`). V5's style is more Go-idiomatic per Effective Go and standard library conventions. Both versions consistently wrap with `%w`, satisfying the skill constraint.

**Magic value elimination.** V5 extracts all magic values into named constants from Task 1-1 onward (`idPrefix`, `idRetries`, `idRandSize`, `maxTitleLen`, `DefaultPriority`, `TimestampFormat`, `lockTimeoutMsg`). V4 hardcodes several values inline: `"tick-"` in GenerateID, `5` as a local variable for retries, `3` in `make([]byte, 3)`, the full filesystem path in lock timeout messages. V5's approach satisfies the skill's "no hardcoded configuration" constraint more completely.

**DRY timestamp formatting.** V5 defines `TimestampFormat` and `FormatTimestamp()` in the task package and reuses them in the cache package (Task 1-3), storage engine (Task 1-4), and CLI output (Task 1-6, 1-7). V4 hardcodes `"2006-01-02T15:04:05Z"` in each package that needs it. This is V5's most impactful code quality advantage -- it prevents format drift across packages.

**Functional options.** V5's storage engine uses `NewStore(tickDir, opts ...Option)` with `WithLockTimeout` and `WithVerbose` options (Task 1-4). V4 exposes `lockTimeout` as a mutable struct field. V5's pattern is more idiomatic Go for configurable constructors and prevents invalid intermediate state.

### Test Quality

Test quality is the one area where the two versions show mixed results rather than a clear V5 advantage.

**V4 excels at assertion depth.** In Task 1-6 (`tick create`), V4 has dedicated tests for timestamp range/UTC verification, cache.db existence, non-numeric priority input ("abc"), `--blocks` timestamp update confirmation, two-task ID uniqueness, and closed-field nil checks. V5 misses all of these. This is V4's only task win, and it is specifically because `tick create` is the first mutation command where thorough end-to-end verification matters most.

**V5 excels at spec-accurate assertions.** In Task 1-7, V5 tests output format using `--pretty` to match the spec-described key-value format, while V4 tests TOON format output that does not match the spec. V4's tests for section omission (acceptance criterion #9) actually assert the opposite of what the spec requires -- they expect `blocked_by[0]` and `children[0]` to appear when empty, contradicting "omit sections with no data." V5 correctly verifies their absence. Testing the wrong format is worse than testing less.

**V5 covers more boundary conditions.** Across the phase, V5 tests: case-insensitive self-reference detection (1-1), empty blocked_by/parent inputs (1-1), JSON serialization format (1-1), empty task list writing (1-2), transaction rollback on failure with duplicate IDs (1-3), corrupted `.tick/` directory (1-5), contradictory filter combinations (1-7). V4 has deeper coverage in a few tests (20 ID generations vs 1, more whitespace trim cases) but misses more edge case categories.

**Test organization diverges.** V4 uses many top-level `Test*` functions, each with a single `t.Run()`. V5 groups related subtests under one top-level function (e.g., `TestCreate` with 23 subtests). V4's approach gives better `go test -run` targeting for individual tests. V5's approach provides logical grouping. Neither is strictly better. However, V5 occasionally goes too far -- Task 1-3's single `TestCache` function with 11 subtests is monolithic where V4's 4 organized groups are more navigable.

### Spec Compliance

V5 demonstrates stronger spec compliance across the phase on the issues that matter most:

1. **ISO 8601 timestamps (Task 1-1):** V5 implements custom `MarshalJSON`/`UnmarshalJSON` producing exact `YYYY-MM-DDTHH:MM:SSZ` format. V4 relies on Go's default `time.Time` JSON encoding (RFC 3339 with nanoseconds), which does not match the spec's prescribed format. Since this is a persistence layer, exact format control is important.

2. **Lock timeout error message (Task 1-4):** The spec prescribes `"Could not acquire lock on .tick/lock - another process may be using tick"`. V5 matches exactly via a named constant. V4 produces `"could not acquire lock on /full/path/.tick/lock - another process may be using tick"` -- lowercase and with the full filesystem path. V4's test is written to pass against its own implementation rather than the spec.

3. **Section omission in show output (Task 1-7):** Spec says "omit sections with no data." V5's tests verify omission. V4's tests verify TOON format presence of empty sections, which is the opposite of the spec requirement.

4. **Directory creation error message (Task 1-5):** The spec says `"Error: Could not create .tick/ directory: <os error>"`. Neither version matches exactly -- V4 uses `"failed to create .tick/ directory"`, V5 uses `"creating .tick directory"`. Both deviate; V4 is slightly closer.

V4's spec deviations (items 1-3) are more consequential than V5's (item 4). The timestamp format affects data persistence and interoperability. The lock timeout message is user-facing. Section omission is a testable acceptance criterion.

### golang-pro Skill Compliance

Both versions comply with most skill constraints equally: explicit error handling, `%w` error wrapping, no panics, no ignored errors, exported function documentation, and godoc-style package comments. The differentiating constraints:

| Constraint | V4 Pattern | V5 Pattern |
|------------|-----------|-----------|
| No hardcoded configuration | PARTIAL -- Magic values inline in Tasks 1-1, 1-4 | PASS -- Named constants throughout |
| Table-driven tests with subtests | PARTIAL -- Subtests via `t.Run()`, table-driven for specific cases (priority rejection) | PARTIAL -- Same pattern |
| `context.Context` for blocking operations | PASS (Task 1-4 uses `context.WithTimeout` for locks) | PASS (same) |

Neither version uses table-driven tests as extensively as the skill prescribes. Both correctly omit `context.Context` from local file/SQLite operations where it would add complexity without benefit. The hardcoded configuration constraint is where V5 has a clear advantage.

## Phase Verdict

**Winner**: V5 (6-1 task score)

V5 is the stronger implementation across Phase 1 by a significant margin. The advantage is not merely numerical -- it stems from a consistent set of engineering principles that V5 applies more rigorously than V4.

The most consequential difference is **spec fidelity in the persistence layer**. V5's custom JSON timestamp marshaling (Task 1-1) ensures the JSONL file format matches the spec exactly, while V4's default `time.Time` encoding produces a different format. For a file-based tool where `tasks.jsonl` is the source of truth and designed to be human-readable and git-diffable, this is foundational. V5's spec-correct lock timeout message (Task 1-4) and output format testing (Task 1-7) reinforce the same pattern: V5 treats the spec as authoritative for user-facing and persistence-facing behavior.

The second systematic advantage is **architectural composability**. V5's thin `NewTask` constructor, separate `storage` package, persistent cache connection, map-based command dispatch, and reusable `FormatTimestamp` utility all reflect a design philosophy where each component has a single, well-defined responsibility. These decisions compound across tasks -- by Task 1-7, V5's cli.go changes are trivially small while V4's switch statement grows linearly. V5's `validateIDsExist()` helper (Task 1-6) eliminates three repetitions of the same lookup-and-error pattern. V5's `taskToShowData()` converter (Task 1-7) enables show-format output from any command. These are the kind of incremental infrastructure investments that reduce friction in later phases.

V4's sole win (Task 1-6, `tick create`) highlights its one genuine strength: **assertion thoroughness in tests**. V4's test suite for the first mutation command is 358 lines longer and covers edge cases V5 misses (timestamp range verification, cache.db existence, non-numeric priority, `--blocks` timestamp update, two-task uniqueness). For a critical write-path command, this depth matters. However, V4's test advantage is undermined in other tasks by testing the wrong output format (TOON instead of spec-prescribed pretty format in Task 1-7) and writing tests that pass against implementation behavior rather than spec requirements (lock timeout message in Task 1-4). Test volume without spec accuracy provides false confidence. The ideal implementation would combine V5's architecture and spec compliance with V4's assertion depth on mutation commands.
