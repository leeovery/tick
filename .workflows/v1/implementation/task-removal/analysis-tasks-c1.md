---
topic: task-removal
cycle: 1
total_proposed: 2
---
# Analysis Tasks: Task Removal (Cycle 1)

## Task 1: Consolidate blast radius computation into Mutate callback
status: approved
severity: high
sources: duplication, architecture

**Problem**: `computeBlastRadius` (lines 58-141) and the `Mutate` callback in `RunRemove` (lines 172-223) both independently (1) validate all IDs exist with all-or-nothing semantics, (2) build a removeSet from explicit targets, and (3) expand that set with transitive descendants. The blast radius function does this via SQL queries with an iterative fixed-point loop; the Mutate callback does it via the in-memory task slice using `collectDescendants`. This creates ~30 lines of duplicated logic, a maintenance burden (changes must be synced across both paths), a theoretical TOCTOU gap between the Query read and the Mutate write, a duplicate type (`idTitle` in remove.go mirrors `RemovedTask` in format.go), and a non-standard inline interface parameter on `computeBlastRadius`.

**Solution**: Restructure so the Mutate callback is the single source of truth for validation and descendant expansion. Introduce a "dry run" mode: the Mutate callback computes the full blast radius (removed tasks, cascaded descendants, affected deps) and returns it without persisting when a flag indicates dry-run. For the non-force path, call Mutate in dry-run mode to get the blast radius for the confirmation prompt, then call Mutate again for the real removal if the user confirms. For the force path, call Mutate once (real mode). This eliminates `computeBlastRadius` entirely along with its SQL queries, the `idTitle` type (use `RemovedTask` for prompt data too), and the inline interface.

**Outcome**: One code path for validation and descendant expansion. No duplicate types. No inline interface. The confirmation prompt shows data computed by the same algorithm that performs the actual removal. The TOCTOU gap is narrowed to an inherent property of the confirmation pattern rather than being amplified by divergent code paths.

**Do**:
1. In `internal/cli/remove.go`, replace the `idTitle` type with `RemovedTask` from `format.go` in the `blastRadius` struct fields (`targetTasks`, `cascadedTasks`, `affectedDeps`).
2. Update `confirmRemovalWithCascade` to accept a `blastRadius` that uses `RemovedTask` fields (access `.ID` and `.Title` instead of `.id` and `.title`).
3. Refactor the Mutate callback into a named function (e.g., `executeRemoval`) that takes the task slice and target IDs, performs validation, descendant expansion via `collectDescendants`, filtering, and dep cleanup, and returns the filtered slice, a `blastRadius` (populated with `RemovedTask` values), a `RemovalResult`, and an error.
4. Add a `computeOnly bool` parameter (or split into two functions): when true, the function validates and computes the blast radius but returns the original task slice unmodified; when false, it performs the full removal and returns the filtered slice.
5. In `RunRemove`, for the non-force path: call `store.Mutate` with `computeOnly=true` to get the blast radius, show the confirmation prompt, then call `store.Mutate` again with `computeOnly=false` to perform the removal. For the force path: call `store.Mutate` once with `computeOnly=false`.
6. Delete the `computeBlastRadius` function, the `idTitle` type, and the `database/sql` import (if no longer needed).
7. Verify all existing tests pass without modification (the external behavior is unchanged).

**Acceptance Criteria**:
- `computeBlastRadius` function is removed
- `idTitle` type is removed
- No `database/sql` import in `remove.go` unless needed elsewhere
- ID validation and descendant expansion exist in exactly one code path
- The confirmation prompt (non-force path) still shows target tasks, cascaded descendants, and affected dependencies
- All existing tests in `internal/cli` pass unchanged
- Force and non-force paths produce identical removal results for the same inputs

**Tests**:
- Run `go test ./internal/cli -run TestRemove -count=1` -- all existing remove tests pass
- Run `go test ./internal/cli -count=1` -- no regressions in other commands
- Verify force and non-force paths produce same output by comparing formatter output in existing test cases that test both paths

## Task 2: Align RunRemove signature with handler convention
status: approved
severity: medium
sources: architecture

**Problem**: Every `Run*` handler in the cli package follows the signature pattern `Run*(dir, fc, fmtr, args, stdout)` with 4-5 parameters. `RunRemove` takes 7 parameters (`dir`, `fc`, `fmtr`, `args`, `stdin`, `stderr`, `stdout`), breaking the uniform handler contract. The extra `stdin` and `stderr` parameters are needed for the interactive confirmation prompt but create an inconsistency that makes the handler harder to call and understand at a glance.

**Solution**: Move the confirmation prompt logic into `handleRemove` on the `App` struct (which already has access to `a.Stdin` and `a.Stderr`), and slim `RunRemove` down to the standard 5-parameter signature. `RunRemove` should only handle the mutation and formatting -- not the interactive prompt. The `handleRemove` method computes the blast radius, runs the confirmation prompt using `a.Stdin`/`a.Stderr`, then calls `RunRemove` for the actual removal.

**Outcome**: `RunRemove` conforms to the established `Run*(dir, fc, fmtr, args, stdout)` signature. The interactive prompt concern is separated from the mutation concern. The handler dispatch pattern in `app.go` remains consistent.

**Do**:
1. In `internal/cli/app.go`, expand `handleRemove` to: parse args (call `parseRemoveArgs`), open the store, compute the blast radius (if not force), run the confirmation prompt using `a.Stdin` and `a.Stderr`, then call `RunRemove`.
2. Change `RunRemove` signature to `RunRemove(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error` -- matching other handlers. It receives pre-validated IDs (the force flag has already been handled), performs the Mutate, and formats output.
3. Alternatively, if Task 1 has not been applied yet and `computeBlastRadius` still exists, `handleRemove` calls `computeBlastRadius` + `confirmRemovalWithCascade` before calling `RunRemove`. If Task 1 has been applied, `handleRemove` calls the dry-run Mutate + confirmation before calling `RunRemove` for the real Mutate.
4. Update all call sites of `RunRemove` (only `handleRemove` in app.go).
5. Update any tests that call `RunRemove` directly to use the new 5-parameter signature. Tests that need confirmation behavior should test via `App.Run` instead.
6. Move `confirmRemovalWithCascade` call site to `handleRemove` or keep it in remove.go as a helper but remove `stdin`/`stderr` from `RunRemove`.

**Acceptance Criteria**:
- `RunRemove` signature matches `Run*(dir, fc, fmtr, args, stdout)` pattern
- `RunRemove` has no `stdin` or `stderr` parameters
- Interactive confirmation logic lives in `handleRemove` or a helper called from `handleRemove`
- All existing tests pass
- Force and non-force behavior unchanged

**Tests**:
- Run `go test ./internal/cli -run TestRemove -count=1` -- all existing remove tests pass
- Run `go test ./internal/cli -count=1` -- no regressions
- Verify `RunRemove` can be called without stdin/stderr in tests that only need mutation behavior
