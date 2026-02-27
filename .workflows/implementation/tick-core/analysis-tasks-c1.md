---
topic: tick-core
cycle: 1
total_proposed: 7
---
# Analysis Tasks: Tick Core (Cycle 1)

## Task 1: Add dependency validation to create and update --blocked-by/--blocks
status: approved
severity: high
sources: standards

**Problem**: The spec (line 403) requires validating dependencies at write time before persisting to JSONL. `tick dep add` correctly calls `task.ValidateDependency()` for cycle detection and child-blocked-by-parent checks. However, `tick create --blocked-by` only calls `validateRefs()` which checks existence and self-reference but NOT cycles or child-blocked-by-parent. `tick create --blocks` and `tick update --blocks` append to target tasks' blocked_by arrays with no dependency validation at all. This allows invalid dependency graphs (cycles, child-blocked-by-parent) to be persisted.

**Solution**: Call `task.ValidateDependency()` or `task.ValidateDependencies()` in `RunCreate` for both `--blocked-by` and `--blocks` targets after building the new task, and in `RunUpdate` for `--blocks` targets. The full task list (including the new/modified task) must be passed to enable proper graph analysis.

**Outcome**: All write paths that modify blocked_by arrays enforce the same validation rules as `tick dep add` -- cycle detection and child-blocked-by-parent rejection -- before persisting to JSONL.

**Do**:
1. In `internal/cli/create.go` `RunCreate`, after the new task is built and added to the task list, call `task.ValidateDependencies()` (or loop with `task.ValidateDependency()`) for each entry in `opts.blockedBy` against the full task list
2. In `internal/cli/create.go` `RunCreate`, for each `opts.blocks` target, after appending the new task ID to the target's blocked_by, call `task.ValidateDependency()` to validate the new dependency against the full task list
3. In `internal/cli/update.go` `RunUpdate`, for each `opts.blocks` target, after appending the task ID to the target's blocked_by, call `task.ValidateDependency()` to validate the new dependency against the full task list
4. If any validation fails, return the error before persisting -- the Mutate callback should return early with the validation error

**Acceptance Criteria**:
- `tick create --blocked-by <parent-id>` on a child task returns child-blocked-by-parent error
- `tick create --blocked-by` that would create a cycle returns cycle detection error
- `tick create --blocks <child-id>` on a parent task returns child-blocked-by-parent error
- `tick update --blocks <child-id>` on a parent task returns child-blocked-by-parent error
- `tick create --blocks` that would create a cycle returns cycle detection error
- No invalid dependency graphs can be persisted through any write path

**Tests**:
- Test create with --blocked-by that would create child-blocked-by-parent dependency is rejected
- Test create with --blocked-by that would create a cycle is rejected
- Test create with --blocks that would create child-blocked-by-parent dependency is rejected
- Test create with --blocks that would create a cycle is rejected
- Test update with --blocks that would create child-blocked-by-parent dependency is rejected
- Test update with --blocks that would create a cycle is rejected
- Test that valid dependencies through create --blocked-by and --blocks still work correctly

## Task 2: Move rebuild logic behind Store abstraction
status: approved
severity: high
sources: architecture

**Problem**: `RunRebuild` in `internal/cli/rebuild.go` manually acquires a file lock, reads JSONL, opens cache, and rebuilds -- re-implementing the locking, file-reading, and cache-management responsibilities that `Store` encapsulates. This creates a parallel code path that does not share the same error recovery, verbose logging integration, or corruption handling as Store. If Store's locking or freshness logic changes, RunRebuild will not benefit.

**Solution**: Add a `Rebuild` method to `Store` that encapsulates forced-rebuild semantics: exclusive lock, delete cache, read JSONL, rebuild cache. `RunRebuild` then calls `store.Rebuild()` similar to how other commands call `store.Mutate()` or `store.Query()`.

**Outcome**: All storage operations (read, write, rebuild) flow through the Store API. No CLI code directly manages locks, reads JSONL, or manipulates the cache file.

**Do**:
1. Add a `Rebuild(verbose *VerboseLogger) error` method (or similar signature) to `Store` in `internal/storage/store.go`
2. The method should: acquire exclusive lock, delete the existing cache.db file, read tasks.jsonl, create a new cache, populate it, update the hash in metadata, release lock
3. Refactor `RunRebuild` in `internal/cli/rebuild.go` to instantiate a Store and call `store.Rebuild()` instead of directly using low-level storage primitives
4. Ensure verbose logging is preserved (rebuild should log the same messages as before when --verbose is set)
5. Remove any now-unused imports or helper usage from rebuild.go

**Acceptance Criteria**:
- `tick rebuild` produces the same user-visible output and behavior as before
- `RunRebuild` no longer directly uses `flock`, `ReadJSONL`, `OpenCache`, or other low-level storage functions
- All lock management and file operations for rebuild flow through Store
- Existing rebuild tests continue to pass

**Tests**:
- Test that store.Rebuild() successfully rebuilds cache from JSONL
- Test that store.Rebuild() works when cache.db does not exist
- Test that store.Rebuild() works when cache.db is corrupted
- Test that RunRebuild integration still produces correct output

## Task 3: Consolidate cache freshness/recovery logic
status: approved
severity: high
sources: duplication, architecture

**Problem**: `Store.ensureFresh` (store.go lines 190-233) and the standalone `EnsureFresh` function (cache.go lines 177-209) implement the same ~40-line pattern: open cache (recover on error by deleting and reopening), check freshness (recover on error by closing/deleting/reopening), rebuild if stale. The standalone function is only used in tests. These two implementations could diverge silently.

**Solution**: Either remove the standalone `EnsureFresh` from cache.go entirely (testing freshness through Store) or extract the shared corruption-recovery-and-rebuild logic into a single private helper both can call. Since `Store.ensureFresh` is the runtime path, prefer removing the standalone function and adjusting tests to use the Store API.

**Outcome**: One code path for cache freshness and corruption recovery. No risk of the two implementations diverging.

**Do**:
1. Check which tests use the standalone `EnsureFresh` function in cache.go
2. Migrate those tests to exercise freshness through the Store API (e.g., `store.Query()` which triggers `ensureFresh` internally)
3. Remove the standalone `EnsureFresh` function from cache.go
4. If any non-test code references `EnsureFresh`, refactor to use Store instead
5. Verify all existing tests pass after removal

**Acceptance Criteria**:
- The standalone `EnsureFresh` function no longer exists in cache.go
- All freshness and corruption recovery tests exercise the Store code path
- No test coverage is lost -- every scenario previously tested through standalone EnsureFresh is tested through Store
- All existing tests pass

**Tests**:
- Test Store handles missing cache.db (rebuilds automatically)
- Test Store handles corrupted cache.db (deletes and rebuilds)
- Test Store detects stale cache via hash mismatch and rebuilds
- Test Store handles freshness check errors (corrupted metadata)

## Task 4: Consolidate formatter duplication and fix Unicode arrow
status: approved
severity: medium
sources: duplication, standards

**Problem**: Two issues affecting the same code. (1) `ToonFormatter.FormatTransition` and `PrettyFormatter.FormatTransition` have identical implementations. Same for `FormatDepChange` -- four methods total are exact copies across `internal/cli/toon_formatter.go` and `internal/cli/pretty_formatter.go`. (2) All transition formatters use ASCII `->` but the spec (line 639) specifies Unicode right arrow (U+2192). The codebase is internally inconsistent: dependency.go cycle errors correctly use `\u2192`.

**Solution**: Extract a shared base implementation (e.g., `baseFormatter` embedded struct) providing `FormatTransition` and `FormatDepChange`. Both ToonFormatter and PrettyFormatter embed it. Fix the arrow character to use Unicode `\u2192` in FormatTransition, matching the spec and dependency.go.

**Outcome**: Four duplicate methods consolidated to two. Transition output uses the spec-correct Unicode arrow. Internal consistency with dependency.go cycle error messages.

**Do**:
1. Create a `baseFormatter` struct in an appropriate file (e.g., `internal/cli/format.go`)
2. Move `FormatTransition` and `FormatDepChange` to methods on `baseFormatter`, changing `->` to the Unicode arrow `\u2192` in FormatTransition
3. Embed `baseFormatter` in both `ToonFormatter` and `PrettyFormatter`
4. Remove the now-redundant methods from `toon_formatter.go` and `pretty_formatter.go`
5. Check `JsonFormatter` -- if it also has identical implementations, embed there too
6. Update any tests that assert on the ASCII `->` arrow to use the Unicode arrow

**Acceptance Criteria**:
- `FormatTransition` and `FormatDepChange` exist in one place only
- Transition output uses Unicode right arrow matching spec line 639
- ToonFormatter, PrettyFormatter (and JsonFormatter if applicable) produce correct output
- All existing formatter tests pass

**Tests**:
- Test that FormatTransition output contains the Unicode arrow character
- Test that FormatTransition output matches spec format: `tick-id: old_status \u2192 new_status`
- Test that FormatDepChange output is correct for add and remove cases
- Test that all three formatters produce consistent transition output

## Task 5: Extract shared helpers for --blocks application and ID parsing
status: approved
severity: medium
sources: duplication

**Problem**: Two patterns are duplicated between create.go and update.go. (1) The --blocks application loop (iterate tasks, match by blockID, append new task ID to BlockedBy, set Updated) is structurally identical in both files (~10 lines each). (2) Comma-separated ID parsing with normalize (`strings.Split` -> range -> `NormalizeID(TrimSpace(id))` -> append if non-empty) appears three times across two files.

**Solution**: Extract `applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time)` and `parseCommaSeparatedIDs(s string) []string` as shared helper functions. Both create.go and update.go call these instead of inlining the logic.

**Outcome**: Block application and ID parsing logic exist in one place. Changes to either pattern (e.g., adding validation) only need to be made once.

**Do**:
1. Create a helper function `parseCommaSeparatedIDs(s string) []string` in an appropriate shared file (e.g., `internal/cli/helpers.go` or similar)
2. The function splits on comma, trims whitespace, normalizes IDs, and filters empty values
3. Replace the three inline parsing instances in create.go and update.go with calls to this helper
4. Create a helper function `applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time)` in the same or appropriate file
5. The function iterates tasks, matches by blockID, appends sourceID to BlockedBy, sets Updated timestamp
6. Replace the inline --blocks loops in create.go and update.go with calls to this helper

**Acceptance Criteria**:
- No inline comma-separated ID parsing loops remain in create.go or update.go
- No inline --blocks application loops remain in create.go or update.go
- Both helpers are called from both create and update
- All existing create and update tests pass

**Tests**:
- Test parseCommaSeparatedIDs with single ID, multiple IDs, whitespace, empty strings
- Test parseCommaSeparatedIDs normalizes to lowercase
- Test applyBlocks correctly appends sourceID to matching tasks' BlockedBy
- Test applyBlocks sets Updated timestamp on modified tasks
- Test applyBlocks with non-existent blockIDs (no-op)

## Task 6: Add end-to-end workflow integration test
status: approved
severity: medium
sources: architecture

**Problem**: Tests cover individual commands well but no test exercises the full agent workflow: init -> create tasks with dependencies/hierarchy -> ready (verify correct tasks) -> transition -> ready (verify unblocking). The closest tests are parent_scope_test.go and individual command tests, but none chain multiple mutations and verify emergent behavior of the ready/blocked query logic across a realistic multi-step workflow.

**Solution**: Add one integration test that exercises the primary workflow end-to-end, catching seam issues between mutation and query paths.

**Outcome**: Confidence that the cross-command integration works correctly for the primary agent workflow described in the spec.

**Do**:
1. Create a test (in `internal/cli/` or an appropriate integration test location) that exercises this sequence:
   - Create a parent/epic task
   - Create child tasks with inter-dependencies (some blocked-by others)
   - Call `tick ready` and verify only the correct unblocked leaf tasks appear
   - Transition a blocker to done
   - Call `tick ready` and verify the previously-blocked task now appears
   - Transition remaining tasks
   - Verify the parent/epic task eventually appears in ready (all children closed)
   - Mark parent done
   - Verify `tick stats` reflects the final state correctly
2. Use the existing test infrastructure (tmpdir setup, Store creation, command runners)
3. Assert on both the correct presence AND absence of tasks in ready results at each step

**Acceptance Criteria**:
- Test exercises create, ready, start, done, and stats across a multi-task hierarchy with dependencies
- Test verifies correct ready set at multiple points in the workflow
- Test verifies unblocking behavior when a dependency is completed
- Test verifies parent appears in ready after all children are closed
- Test passes reliably

**Tests**:
- The task itself is a test -- one comprehensive integration test covering the full workflow

## Task 7: Add explanatory second line to child-blocked-by-parent error
status: approved
severity: low
sources: standards

**Problem**: The spec (lines 407-408) defines the child-blocked-by-parent error as a two-line message: "Cannot add dependency - tick-child cannot be blocked by its parent tick-epic" followed by "(would create unworkable task due to leaf-only ready rule)". The implementation in `internal/task/dependency.go:36` only outputs the first line and uses lowercase "cannot" vs the spec's "Cannot".

**Solution**: Update the error message in dependency.go to include the second explanatory line and fix capitalization to match spec.

**Outcome**: Error message matches spec exactly, providing agents and users with the rationale for the constraint.

**Do**:
1. In `internal/task/dependency.go`, find the child-blocked-by-parent error return (around line 36)
2. Update the error message to match spec format: first line "Cannot add dependency - {child} cannot be blocked by its parent {parent}" with capital C
3. Add second line: "(would create unworkable task due to leaf-only ready rule)"
4. Update any tests that assert on the exact error message text

**Acceptance Criteria**:
- Error message matches spec lines 407-408 exactly (two lines, correct capitalization)
- Existing dependency validation tests pass with updated assertions
- The explanatory rationale line is present in the error output

**Tests**:
- Test that child-blocked-by-parent error includes both lines
- Test that error message uses "Cannot" (capital C)
- Test that error message includes the rationale about leaf-only ready rule
