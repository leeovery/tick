TASK: Migration Engine - Iterate & Insert

ACCEPTANCE CRITERIA:
- [ ] `Engine` struct exists in `internal/migrate/` with a `Run(Provider) ([]Result, error)` method
- [ ] `TaskCreator` interface abstracts tick-core task creation
- [ ] Each `MigratedTask` is validated before insertion; validation failures produce a failure `Result` without stopping the engine
- [ ] Provider error (from `Tasks()`) is returned immediately
- [ ] Insertion error (from `CreateTask`) is returned immediately with partial results (Phase 1 fail-fast)
- [ ] Zero tasks from provider returns empty `[]Result` and nil error
- [ ] Results are collected in provider order
- [ ] Engine is testable with mock `Provider` and mock `TaskCreator` (no real tick-core dependency in tests)

STATUS: Complete

SPEC CONTEXT:
The specification defines an architecture of Provider -> Normalize -> Insert, where the engine is the "Insert" step. It receives `[]MigratedTask` (normalized format) and creates tick entries. Error handling strategy from the spec: "Continue on error, report failures at end. No rollback." Phase 1 was planned as a simpler version with fail-fast on insertion errors, but the actual implementation has evolved to Phase 2 behavior (continue-on-error for insertion failures as well). The task plan document explicitly states this Phase 1 fail-fast behavior for insertion errors, but the implementation follows the Phase 2 continue-on-error model -- which is the more complete and correct behavior per the specification's overall intent.

IMPLEMENTATION:
- Status: Implemented (with deliberate Phase 2 evolution)
- Location: `/Users/leeovery/Code/tick/internal/migrate/engine.go` (lines 1-89)
- Notes:
  - `Engine` struct exists at line 43 with `creator TaskCreator` and `opts Options` fields.
  - `TaskCreator` interface exists at line 35 with `CreateTask(t MigratedTask) (string, error)` signature.
  - `NewEngine(creator TaskCreator, opts Options) *Engine` constructor at line 50.
  - `Run(provider Provider) ([]Result, error)` method at line 59.
  - Provider error from `Tasks()` is returned immediately at line 62 (`return nil, err`).
  - Validation is called before insertion at line 71. Validation failures produce a failure `Result` with fallback title handling (lines 72-77) and `continue` to next task.
  - Insertion failures produce a failure `Result` and `continue` (lines 80-83), which is the Phase 2 continue-on-error behavior rather than the Phase 1 fail-fast originally planned. This is noted as intentional evolution per the plan (Phase 2 task migration-2-1 upgrades to continue-on-error).
  - Results are collected in provider iteration order.
  - `FallbackTitle` constant `"(untitled)"` defined in `migrate.go` line 22.
  - `StoreTaskCreator` (real implementation) in `store_creator.go` applies all defaults: empty status -> `open`, nil priority -> `2`, zero Created -> `time.Now()`, zero Updated -> Created, zero Closed -> nil.
  - `Options` struct with `PendingOnly` field and `filterPending` function also reside in `engine.go` (Phase 2 concern, but cleanly integrated).

TESTS:
- Status: Adequate
- Coverage:
  - `TestEngineRun` in `/Users/leeovery/Code/tick/internal/migrate/engine_test.go` covers all 10 test cases specified in the task plan:
    1. "it calls Validate on each MigratedTask before insertion" (line 169)
    2. "it returns a successful Result for each valid task inserted" (line 208)
    3. "it skips tasks that fail validation and records failure Result with error" (line 244)
    4. "it returns error immediately when provider.Tasks() fails" (line 283)
    5. "it returns empty Results slice when provider returns zero tasks" (line 304)
    6. "it continues processing after CreateTask fails and records failure Result" (line 324) -- tests Phase 2 continue behavior instead of Phase 1 fail-fast
    7. "it processes all tasks in order and returns Results in same order" (line 507)
    8. "it applies defaults via TaskCreator -- empty status becomes open, zero priority becomes 2" (line 545)
    9. "it records Result with fallback title when task has empty title and fails validation" (line 577)
    10. "successful tasks are persisted even when later tasks fail insertion" (line 605) -- continues-past-failure variant
  - Additional tests beyond the plan:
    - "it returns nil error when all tasks fail insertion" (line 368)
    - "it returns nil error with mixed validation and insertion failures" (line 399)
    - "failure Result from insertion contains the original CreateTask error" (line 447)
    - "results slice contains entries for all tasks in provider order regardless of success or failure" (line 475)
    - "all tasks fail insertion returns results with all failures and nil error" (line 645)
    - "mixed failures: validation then insertion then success produces three Results in order" (line 680)
  - `TestFilterPending` covers pending-only filtering (Phase 2) with 9 subtests.
  - `TestEnginePendingOnly` covers engine-level pending-only behavior with 5 subtests.
  - Edge cases covered: empty provider (zero tasks), insertion failure, empty title fallback, provider error, mixed validation/insertion failures, all-failures case, ordering.
  - Tests use mock `Provider` and mock `TaskCreator` -- no real tick-core dependency.
  - `StoreTaskCreator` tests in `store_creator_test.go` separately cover all default-application logic (8 subtests).
- Notes:
  - The test "it returns error immediately when TaskCreator.CreateTask fails" from the plan (Phase 1 fail-fast) has been replaced with the continue-on-error variant. This is consistent with the implementation's Phase 2 evolution.
  - There are some tests that overlap in what they verify (e.g., ordering is checked in multiple places, and multiple tests verify continue-on-insertion-failure). This is borderline over-testing but each test does exercise a meaningfully different scenario, so it remains acceptable.

CODE QUALITY:
- Project conventions: Followed
  - stdlib `testing` only (no testify), `t.Run()` subtests, proper error wrapping
  - Handler/interface patterns consistent with project conventions
  - `Engine` uses DI via constructor (consistent with project pattern)
- SOLID principles: Good
  - Single responsibility: Engine orchestrates; TaskCreator handles persistence; Provider handles source reading
  - Open/closed: New providers can be added without modifying Engine
  - Dependency inversion: Engine depends on `TaskCreator` and `Provider` interfaces, not concrete types
  - Interface segregation: `TaskCreator` has a single method; `Provider` has two focused methods
- Complexity: Low
  - `Engine.Run` is a simple linear loop with two branch points (validation, insertion)
  - No nested complexity, no complex state management
- Modern idioms: Yes
  - Proper use of interfaces for decoupling
  - Slice pre-allocation with `make([]Result, 0, len(tasks))`
  - `strings.TrimSpace` for whitespace-only title check
- Readability: Good
  - Clear variable names, well-documented exported types
  - Engine flow is easy to follow: fetch -> filter -> iterate -> validate -> insert -> collect
  - Comments explain Phase 2 evolution in the Run method doc comment
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The implementation has evolved from Phase 1 fail-fast on insertion errors to Phase 2 continue-on-error. The `Run` method doc comment at line 55-58 accurately documents this behavior. The plan task acceptance criterion "Insertion error (from CreateTask) is returned immediately with partial results (Phase 1 fail-fast)" is technically not met as-is, but this is because the implementation was intentionally upgraded to the more complete Phase 2 behavior as part of task migration-2-1. This is a positive evolution, not a regression.
- The `filterPending` tests (9 subtests) and `TestEnginePendingOnly` (5 subtests) belong more logically to Phase 2 tasks (migration-2-4), but their presence in `engine_test.go` is reasonable since `filterPending` and `Options.PendingOnly` live in `engine.go`.
- Some test overlap exists between "results slice contains entries for all tasks in provider order regardless of success or failure" and "it processes all tasks in order and returns Results in same order" -- both verify ordering. The first adds failure scenarios, making it mildly distinct but overlapping.
