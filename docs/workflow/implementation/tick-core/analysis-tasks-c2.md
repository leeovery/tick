---
topic: tick-core
cycle: 2
total_proposed: 5
---
# Analysis Tasks: Tick Core (Cycle 2)

## Task 1: Extract shared ready-query SQL conditions
status: approved
severity: medium
sources: duplication, architecture

**Problem**: The "ready" SQL WHERE conditions (status='open' AND NOT EXISTS unclosed blockers AND NOT EXISTS open children) are independently authored in list.go (buildListQuery, lines 211-224) and stats.go (RunStats, lines 86-99). The blocked query in list.go (lines 227-245) is the De Morgan inverse, making three total locations encoding ready/blocked semantics. If the definition of "ready" changes (e.g., adding a new condition), all locations must be updated in sync -- a drift risk flagged by both the duplication and architecture agents.

**Solution**: Extract the ready NOT EXISTS subquery clauses as named SQL constants or a helper function (e.g., `readyConditions() []string` or `const readyBlockerCondition` and `readyChildCondition`) in a shared location such as a new `query_helpers.go` file. Both `buildListQuery` and `RunStats` compose their queries from these shared fragments. The blocked conditions can be derived as the negation.

**Outcome**: The ready/blocked SQL definition exists in exactly one place. Changes to readiness semantics require updating a single location. Both list and stats queries stay in sync automatically.

**Do**:
1. Create `internal/cli/query_helpers.go` (or add to an existing shared file)
2. Define shared SQL condition fragments for the two NOT EXISTS subqueries that define "ready": (a) no unclosed blockers, (b) no open children
3. Refactor `buildListQuery` in `internal/cli/list.go` to use the shared conditions for both the `--ready` and `--blocked` filters
4. Refactor the ready count query in `internal/cli/stats.go` (RunStats) to use the same shared conditions
5. Ensure the blocked filter in list.go derives from the negation of the ready conditions rather than independently re-implementing them
6. Run all existing tests to verify no behavioral changes

**Acceptance Criteria**:
- The ready NOT EXISTS subqueries appear in exactly one location (the shared helper)
- buildListQuery uses the shared helper for both ready and blocked filters
- RunStats uses the shared helper for the ready count query
- All existing list and stats tests pass unchanged
- `tick ready`, `tick blocked`, and `tick stats` produce identical output to before

**Tests**:
- Test that the shared ready conditions produce correct SQL fragments
- Test that list --ready still returns correct results after refactor
- Test that list --blocked still returns correct results after refactor
- Test that stats ready/blocked counts remain accurate after refactor

## Task 2: Add relationship context to create command output
status: approved
severity: medium
sources: standards

**Problem**: The spec (line 631) states create output should be "full task details (same format as tick show), TTY-aware." The implementation constructs a TaskDetail with only the raw Task struct and empty BlockedBy/Children slices (create.go line 183), without querying SQLite for relationship context (blocker titles/statuses, parent title). If a task is created with `--blocked-by` or `--parent`, those relationships will not appear in the output. In contrast, `tick update` (update.go lines 214-220) correctly calls `queryShowData` to populate all relationship context before formatting.

**Solution**: After the Mutate call succeeds in RunCreate, call `queryShowData(store, createdTask.ID)` to retrieve full relationship context (matching the approach used by RunUpdate), then pass that populated TaskDetail to `FormatTaskDetail`.

**Outcome**: Create output is truly "same format as tick show" -- blocked_by entries show with titles and statuses, parent title is included, and children (if any exist due to --blocks creating reverse relationships) are shown.

**Do**:
1. In `internal/cli/create.go` RunCreate, after the successful Mutate call and before the output block
2. Replace the manual `TaskDetail{Task: createdTask}` construction (line 183) with a call to `queryShowData(store, createdTask.ID)`
3. Use the returned showData to build the TaskDetail via `showDataToTaskDetail` (same approach as update.go)
4. Ensure the store is available at that point in the function (it should be, from the openStore call)
5. Handle the error case from queryShowData appropriately

**Acceptance Criteria**:
- `tick create "test" --blocked-by tick-abc` output includes the blocker's title and status in the blocked_by section
- `tick create "test" --parent tick-abc` output includes the parent's title
- Create output matches show output for the same task (when viewed immediately after creation)
- `--quiet` mode still outputs only the task ID (no change)
- All existing create tests pass

**Tests**:
- Test create with --blocked-by shows blocker title and status in output
- Test create with --parent shows parent title in output
- Test create with --blocks shows the created task's relationship context
- Test create with --quiet still outputs only the task ID
- Test create without relationships still produces correct output (empty blocked_by/children sections)

## Task 3: Extract store-opening boilerplate into shared helper
status: approved
severity: medium
sources: duplication

**Problem**: Every Run* function that accesses the store repeats the same 8-line sequence: `DiscoverTickDir(dir)`, `storage.NewStore(tickDir, storeOpts(fc)...)`, `defer store.Close()`. This identical block appears 9 times across 8 files (dep.go has it twice). Each instance uses the same arguments (dir, fc) and produces the same local variables (tickDir, store, err). The pattern is mechanical boilerplate with no variation.

**Solution**: Extract an `openStore(dir string, fc FormatConfig) (*storage.Store, func(), error)` helper that encapsulates DiscoverTickDir + NewStore. Return a cleanup function or let callers defer store.Close() themselves. Callers reduce from 8 lines to approximately 3 lines.

**Outcome**: Store opening logic exists in one place. If the initialization sequence changes (e.g., adding a new option, changing DiscoverTickDir behavior), only one location needs updating. Approximately 45 lines of boilerplate eliminated across 9 call sites.

**Do**:
1. Create a helper function `openStore(dir string, fc FormatConfig) (*storage.Store, error)` in an appropriate shared file (e.g., `internal/cli/helpers.go` or `internal/cli/store_helpers.go`)
2. The function calls `DiscoverTickDir(dir)` and `storage.NewStore(tickDir, storeOpts(fc)...)`, returning the store or any error
3. Replace the boilerplate in all 9 call sites: create.go, dep.go (twice), list.go, rebuild.go, show.go, stats.go, transition.go, update.go
4. Each call site retains its own `defer store.Close()` since Go defers are scope-bound
5. Run all tests to verify no behavioral changes

**Acceptance Criteria**:
- No inline DiscoverTickDir + NewStore sequence remains in any Run* function
- All 9 call sites use the shared openStore helper
- Each call site still has its own defer store.Close()
- All existing tests pass unchanged

**Tests**:
- Test openStore returns a valid store for a valid tick directory
- Test openStore returns appropriate error when no .tick directory exists
- Test that all commands still function correctly after refactor (covered by existing integration tests)

## Task 4: Remove dead VerboseLog function
status: approved
severity: low
sources: architecture

**Problem**: The standalone function `VerboseLog(w io.Writer, verbose bool, msg string)` in format.go (lines 186-192) is defined and tested (format_test.go lines 381-400) but never called in any production code path. All production verbose logging uses the `VerboseLogger` struct (verbose.go) instead. This is dead code left over from before VerboseLogger was introduced.

**Solution**: Remove VerboseLog from format.go and its associated tests from format_test.go.

**Outcome**: No dead code in the verbose logging surface. All verbose logging flows through VerboseLogger.

**Do**:
1. Remove the `VerboseLog` function from `internal/cli/format.go` (lines 186-192)
2. Remove the associated test(s) from `internal/cli/format_test.go` (lines 381-400)
3. Remove any unused imports that result from the removal
4. Verify no production code references VerboseLog (it should only be referenced from its own test)
5. Run all tests to confirm nothing breaks

**Acceptance Criteria**:
- VerboseLog function no longer exists in format.go
- No test references to VerboseLog remain
- All existing tests pass
- No production code is affected

**Tests**:
- Verify via grep that no production code calls VerboseLog (validation step, not a new test)
- All existing tests pass after removal

## Task 5: Consolidate duplicate relatedTask struct into RelatedTask
status: approved
severity: low
sources: duplication

**Problem**: show.go defines an unexported `relatedTask{id, title, status string}` struct (lines 30-34) used in `queryShowData`, while format.go defines the exported `RelatedTask{ID, Title, Status string}` (lines 88-92) with identical fields. The `showDataToTaskDetail` function (show.go lines 176-191) loops through each `relatedTask` and converts it to a `RelatedTask` by copying field by field. The two structs are structurally identical -- the only difference is export visibility and field naming.

**Solution**: Use `RelatedTask` directly in `queryShowData` instead of the unexported `relatedTask`. SQL Scan calls can target `RelatedTask` fields directly (`&r.ID`, `&r.Title`, `&r.Status`). This eliminates the `relatedTask` struct and the two conversion loops in `showDataToTaskDetail`.

**Outcome**: One struct for related task data. Approximately 15 lines of mapping code removed. No intermediate type conversion needed between query and formatting layers.

**Do**:
1. In `internal/cli/show.go`, change the `showData` struct to use `[]RelatedTask` instead of `[]relatedTask` for the `blockedBy` and `children` fields
2. Update `queryShowData` to scan directly into `RelatedTask` fields (`&r.ID`, `&r.Title`, `&r.Status`)
3. Remove the unexported `relatedTask` struct definition from show.go
4. Simplify `showDataToTaskDetail` to directly assign the `[]RelatedTask` slices instead of converting field by field
5. Run all tests to verify no behavioral changes

**Acceptance Criteria**:
- The unexported `relatedTask` struct no longer exists in show.go
- `queryShowData` populates `RelatedTask` directly
- `showDataToTaskDetail` no longer has field-by-field conversion loops
- All existing show and format tests pass unchanged

**Tests**:
- Test that tick show output is unchanged after refactor (covered by existing show tests)
- Test that queryShowData correctly populates RelatedTask fields
