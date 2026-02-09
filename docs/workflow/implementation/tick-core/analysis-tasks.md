---
topic: tick-core
cycle: 1
total_proposed: 7
---
# Analysis Tasks: Tick Core (Cycle 1)

## Task 1: Add dependency validation to create --blocked-by and --blocks paths
status: approved
severity: high
sources: standards

**Problem**: The `create` command's `--blocked-by` path calls `ValidateBlockedBy` (self-reference only) and `validateIDsExist`, but never calls `task.ValidateDependency` or `task.ValidateDependencies` which perform cycle detection and the child-blocked-by-parent check. Similarly, `--blocks` in both `create` and `update` adds the task ID to target tasks' `blocked_by` arrays without any dependency validation. The `dep add` command correctly validates via `task.ValidateDependency` but the same validation was not applied to these equivalent code paths. A user could create a task with `--blocked-by` that forms a cycle or violates the child-blocked-by-parent rule, and it would be silently persisted.

**Solution**: In `create.go`, after validating IDs exist and before building the new task, call `task.ValidateDependencies(tasks, id, blockedBy)` for the `--blocked-by` list. For each `blockID` in the `--blocks` list (in both `create.go` and `update.go`), call `task.ValidateDependency(tasks, blockID, id)` to validate the reverse dependency. This mirrors the validation already performed in `dep.go:runDepAdd`.

**Outcome**: All dependency constraints (cycle detection, child-blocked-by-parent rule) are enforced consistently whether dependencies are added via `create --blocked-by`, `create --blocks`, `update --blocks`, or `dep add`.

**Do**:
1. In `internal/cli/create.go`, inside the Mutate closure, after the `validateIDsExist` calls and before building `newTask`: if `len(blockedBy) > 0`, call `task.ValidateDependencies(tasks, id, blockedBy)` and return error on failure.
2. In `internal/cli/create.go`, inside the Mutate closure, before the `--blocks` loop: for each `blockID` in `blocks`, call `task.ValidateDependency(tasks, blockID, id)` and return error on failure.
3. In `internal/cli/update.go`, inside the Mutate closure, in the `opts.blocks != nil` validation section (around line 95-101): for each `blockID` in `blocks`, call `task.ValidateDependency(tasks, blockID, id)` and return error on failure.
4. Add tests covering: (a) create --blocked-by that would form a cycle is rejected, (b) create --blocked-by where child is blocked by parent is rejected, (c) create --blocks that would form a cycle is rejected, (d) update --blocks that would form a cycle is rejected, (e) valid dependencies still succeed.

**Acceptance Criteria**:
- `tick create "X" --blocked-by <id>` rejects cycles with the same error message as `dep add`
- `tick create "X" --blocked-by <parent>` rejects child-blocked-by-parent with the same error message as `dep add`
- `tick create "X" --blocks <id>` rejects cycles when the reverse dependency would create one
- `tick update <id> --blocks <id2>` rejects cycles when the reverse dependency would create one
- All existing tests continue to pass

**Tests**:
- Test create with --blocked-by forming a direct cycle (A blocks B, create C --blocked-by A,C where C blocks A)
- Test create with --blocked-by where the new task is a child blocked by its parent
- Test create with --blocks forming a cycle
- Test update with --blocks forming a cycle
- Test that valid --blocked-by and --blocks dependencies are still accepted

## Task 2: Extract shared ready/blocked SQL WHERE clauses
status: approved
severity: medium
sources: duplication

**Problem**: `StatsReadyCountQuery` in `stats.go` duplicates the WHERE clause from `ReadyQuery` in `ready.go` (same three conditions: status=open, NOT EXISTS unclosed blockers, NOT EXISTS open children). `StatsBlockedCountQuery` duplicates the WHERE clause from `BlockedQuery` in `blocked.go`. Additionally, `buildReadyFilterQuery` and `buildBlockedFilterQuery` in `list.go` are near-identical functions (~15 lines each) that differ only in which inner query constant they wrap. If the ready/blocked logic changes, both the list query and the stats count query must be updated in sync.

**Solution**: Extract the shared WHERE clause into constants (e.g. `readyWhereClause`, `blockedWhereClause`), then compose both the list query and count query from them. Also extract a shared `buildWrappedFilterQuery(innerQuery, alias string, f listFilters, descendantIDs []string)` function that both filter query builders call.

**Outcome**: The ready and blocked query logic is defined in exactly one place. Changes to readiness/blocked criteria propagate automatically to both list and stats queries.

**Do**:
1. In the appropriate query file(s), define `readyWhereClause` and `blockedWhereClause` as string constants containing just the WHERE conditions.
2. Redefine `ReadyQuery` as `"SELECT ... FROM tasks t WHERE " + readyWhereClause + " ORDER BY ..."` and `StatsReadyCountQuery` as `"SELECT COUNT(*) FROM tasks t WHERE " + readyWhereClause`. Same pattern for blocked.
3. In `internal/cli/list.go`, extract a shared `buildWrappedFilterQuery(innerQuery string, f listFilters, descendantIDs []string) (string, []interface{})` that both `buildReadyFilterQuery` and `buildBlockedFilterQuery` delegate to.
4. Run existing tests to verify no query behavior changes.

**Acceptance Criteria**:
- ReadyQuery and StatsReadyCountQuery share the same WHERE clause constant
- BlockedQuery and StatsBlockedCountQuery share the same WHERE clause constant
- buildReadyFilterQuery and buildBlockedFilterQuery are collapsed into calls to a shared function
- All existing list, ready, blocked, and stats tests pass unchanged

**Tests**:
- All existing tests for list --ready, list --blocked, stats, ready, and blocked commands pass
- No new tests needed -- this is a pure refactor with existing coverage

## Task 3: Consolidate ReadTasks/ParseTasks duplicate JSONL parsing
status: approved
severity: medium
sources: duplication

**Problem**: `ReadTasks` (reads from file path) and `ParseTasks` (reads from `[]byte`) in `internal/storage/jsonl.go` contain identical scanner-based parsing logic: bufio.Scanner loop, empty-line skip, line-by-line json.Unmarshal, lineNum tracking, and error wrapping. The only difference is the io.Reader source (os.File vs bytes.NewReader). This is 20+ lines of duplicated logic that must be kept in sync.

**Solution**: Have `ReadTasks` open the file, read its contents into `[]byte`, then delegate to `ParseTasks`. This collapses the duplicate parsing loop into a single implementation.

**Outcome**: JSONL parsing logic exists in one place (`ParseTasks`). `ReadTasks` becomes a thin wrapper that handles file I/O and delegates.

**Do**:
1. In `internal/storage/jsonl.go`, modify `ReadTasks` to: open the file, read all bytes (e.g. via `io.ReadAll`), close the file, then call `ParseTasks(data)` and return its result.
2. Remove the duplicated scanner loop from `ReadTasks`.
3. Run all storage tests to verify behavior is identical.

**Acceptance Criteria**:
- `ReadTasks` delegates to `ParseTasks` instead of duplicating the parsing loop
- All existing storage/JSONL tests pass unchanged
- Error messages from ReadTasks remain consistent (file-not-found errors still originate from ReadTasks)

**Tests**:
- All existing storage tests pass -- this is a pure refactor
- Verify ReadTasks still returns appropriate error for non-existent file path

## Task 4: Extract shared formatter methods (FormatTransition, FormatDepChange, FormatMessage)
status: approved
severity: medium
sources: duplication, architecture

**Problem**: `ToonFormatter.FormatTransition` and `PrettyFormatter.FormatTransition` produce byte-identical output (same type assertion, same fmt.Fprintf with arrow). `FormatDepChange` is also identical across both formatters. `FormatMessage` is identical across all three formatters (Toon, Pretty, Stub). These are ~50 lines of duplicated logic that will be copied into any future formatter and risk drift if the format changes.

**Solution**: Extract shared implementations into package-level helper functions (e.g. `formatTransitionText`, `formatDepChangeText`) that both ToonFormatter and PrettyFormatter call. For FormatMessage, either embed a common base struct or use a standalone helper.

**Outcome**: Each shared formatting operation is implemented once. Toon and Pretty formatters delegate to the shared implementation for operations where their output is identical.

**Do**:
1. In `internal/cli/format.go` (or a new helpers file in the same package), define `formatTransitionText(w io.Writer, data interface{}) error` containing the shared transition formatting logic currently in both formatters.
2. Define `formatDepChangeText(w io.Writer, data interface{}) error` with the shared dep change formatting logic.
3. Update `ToonFormatter.FormatTransition`, `PrettyFormatter.FormatTransition`, `ToonFormatter.FormatDepChange`, and `PrettyFormatter.FormatDepChange` to delegate to these helpers.
4. For FormatMessage, define a package-level `formatMessageText(w io.Writer, msg string)` and have all formatters call it (or embed a base struct).
5. Run all formatter tests.

**Acceptance Criteria**:
- FormatTransition logic exists in one shared function, called by both Toon and Pretty formatters
- FormatDepChange logic exists in one shared function, called by both Toon and Pretty formatters
- FormatMessage logic exists in one shared function or base struct
- All existing formatter and command tests pass unchanged

**Tests**:
- All existing tests for transition output, dep change output, and message output pass
- No new tests needed -- this is a pure refactor with existing coverage

## Task 5: Remove doctor command from help text
status: approved
severity: medium
sources: standards

**Problem**: The help text in `printUsage` (cli.go line 203) advertises `tick doctor - Run diagnostics and validation` but the `commands` map has no "doctor" entry. Running `tick doctor` produces "Error: Unknown command 'doctor'". The spec lists `tick doctor` for specific diagnostics (orphaned children, parent-done-before-children), but the command is not implemented. This misleads users and agents that parse help output.

**Solution**: Remove the doctor entry from the help text. If doctor is implemented in a future cycle, the help text can be re-added at that time.

**Outcome**: The help text only advertises commands that are actually implemented.

**Do**:
1. In `internal/cli/cli.go`, remove the line `fmt.Fprintln(w, "  doctor    Run diagnostics and validation")` from `printUsage`.
2. Verify `tick help` output no longer mentions doctor.

**Acceptance Criteria**:
- `tick help` does not list the doctor command
- No functional commands are affected
- If a help output test exists, update it to remove the doctor line

**Tests**:
- Verify `tick help` output does not contain "doctor"
- Verify `tick doctor` still returns "Unknown command" error (unchanged behavior)

## Task 6: Remove dead StubFormatter code
status: approved
severity: low
sources: architecture

**Problem**: The `StubFormatter` type in `internal/cli/format.go` (lines 93-127) is annotated as a placeholder that "will be replaced by concrete Toon, Pretty, and JSON formatters" but all three concrete formatters now exist. StubFormatter is not referenced anywhere in production code paths (`newFormatter` only instantiates concrete formatters). It is dead code that adds maintenance noise.

**Solution**: Delete the `StubFormatter` struct and all its methods from `format.go`.

**Outcome**: No dead formatter code in the codebase. The concrete formatters (Toon, Pretty, JSON) are the only implementations.

**Do**:
1. In `internal/cli/format.go`, remove the `StubFormatter` struct definition and all its methods (FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats, FormatMessage).
2. Check if any test files reference StubFormatter -- if so, update them to use a concrete formatter or a test-specific mock.
3. Run all tests.

**Acceptance Criteria**:
- `StubFormatter` type no longer exists in the codebase
- All tests pass
- No compilation errors

**Tests**:
- Full test suite passes after removal
- Grep for "StubFormatter" returns no hits in production code

## Task 7: Replace interface{} Formatter parameters with type-safe signatures
status: approved
severity: medium
sources: architecture

**Problem**: Every Formatter method (FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats) accepts `interface{}` as its data parameter and performs a runtime type assertion at the top of each implementation. With 3 formatters x 5 methods = 15 assertion sites, the surface area for silent type mismatches is significant. A caller passing the wrong type compiles successfully but panics or fails at runtime. Go's type system cannot catch regressions during refactoring.

**Solution**: Replace the Formatter interface methods with type-specific signatures. For example: `FormatTaskList(w io.Writer, rows []TaskRow) error`, `FormatTaskDetail(w io.Writer, data *showData) error`, `FormatTransition(w io.Writer, data *TransitionData) error`, `FormatDepChange(w io.Writer, data *DepChangeData) error`, `FormatStats(w io.Writer, data *StatsData) error`.

**Outcome**: All 15 runtime type assertions are eliminated. Passing the wrong type to a formatter method becomes a compile error.

**Do**:
1. In `internal/cli/format.go`, update the Formatter interface to use concrete types instead of `interface{}` for each method's data parameter.
2. Update `ToonFormatter`, `PrettyFormatter`, and `JSONFormatter` method signatures to match the new interface.
3. Remove the runtime type assertions at the top of each formatter method (the `data.(*SomeType)` calls).
4. Export `showData` and `relatedTask` types if they are used in the Formatter interface signatures (or keep them unexported if the interface and all implementations remain in the same package).
5. Update all call sites if any type adjustments are needed (e.g. pointer vs value).
6. Run all tests.

**Acceptance Criteria**:
- Formatter interface methods use concrete types, not interface{}
- No runtime type assertions in formatter implementations
- All existing tests pass
- Code compiles without errors

**Tests**:
- Full test suite passes after the refactor
- Verify that passing an incorrect type to a formatter method causes a compile error (manual check)
