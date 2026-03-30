---
phase: 1
phase_name: Fix Tree Qualification Leak To Note
total: 2
---

# Phase 1: Fix Tree Qualification Leak To Note

## qualify-command-tree-leaks-to-note-1-1 | approved

### Task 1: Reproduce Bug and Fix qualifyCommand

**Problem**: The `qualifyCommand` function in `internal/cli/app.go` (lines 374-376) has a single `case "add", "remove", "tree"` branch that applies to both `"dep"` and `"note"` parents. This causes `qualifyCommand("note", ["tree", "--foo"])` to return `("note tree", ["--foo"])`, which then makes `ValidateFlags` produce the misleading error `unknown flag "--foo" for "note tree"` — implying `"note tree"` is a real command. The subcommand `"tree"` is only valid under `"dep"`, not `"note"`.

**Solution**: Split the switch case inside `qualifyCommand` so that `"tree"` is only qualified when the parent is `"dep"`. The shared `"add"`, `"remove"` case remains unchanged (both parents legitimately have these subcommands). Add a separate `case "tree"` with a guard on `subcmd == "dep"`. When `subcmd == "note"` and `sub == "tree"`, the function falls through to the default return, leaving the command unqualified so `handleNote` produces the correct `"unknown note sub-command 'tree'"` error. Write unit tests first that reproduce the bug before applying the fix.

**Outcome**: `qualifyCommand("note", ["tree"])` returns `("note", ["tree"])` — tree is NOT qualified under note. `qualifyCommand("dep", ["tree"])` continues to return `("dep tree", [])`. Shared subcommands `add` and `remove` continue to qualify under both parents.

**Do**:
1. In `internal/cli/dep_tree_test.go` (or a new `qualify_command_test.go` — either is fine since the existing dep_tree_test.go already tests `qualifyCommand`), add failing tests FIRST:
   - `qualifyCommand("note", ["tree"])` must return `("note", ["tree"])` — not `("note tree", [])`
   - `qualifyCommand("note", ["tree", "--foo"])` must return `("note", ["tree", "--foo"])` — args preserved, not stripped
   - `qualifyCommand("note", ["add", "tick-aaa", "hello"])` must still return `("note add", ["tick-aaa", "hello"])` — shared subcommand unaffected
   - `qualifyCommand("note", ["remove", "tick-aaa", "1"])` must still return `("note remove", ["tick-aaa", "1"])` — shared subcommand unaffected
2. Run the new tests to confirm they fail (the first two should fail, the last two should pass).
3. In `internal/cli/app.go`, modify `qualifyCommand` (around line 374):
   - Change `case "add", "remove", "tree":` to `case "add", "remove":` (shared subcommands only)
   - Add a new `case "tree":` that only qualifies when `subcmd == "dep"`:
     ```go
     case "tree":
         if subcmd == "dep" {
             return subcmd + " " + sub, subArgs[1:]
         }
         return subcmd, subArgs
     ```
4. Run all tests to confirm the new tests pass and no existing tests regress.
5. Also run `go vet ./internal/cli` to ensure no warnings.

**Acceptance Criteria**:
- [ ] `qualifyCommand("note", ["tree"])` returns `("note", ["tree"])`
- [ ] `qualifyCommand("note", ["tree", "--foo"])` returns `("note", ["tree", "--foo"])`
- [ ] `qualifyCommand("dep", ["tree"])` returns `("dep tree", [])` — unchanged
- [ ] `qualifyCommand("dep", ["tree", "tick-abc123"])` returns `("dep tree", ["tick-abc123"])` — unchanged
- [ ] `qualifyCommand("note", ["add", ...])` and `qualifyCommand("note", ["remove", ...])` continue to qualify — shared subcommands unaffected
- [ ] `qualifyCommand("dep", ["add", ...])` and `qualifyCommand("dep", ["remove", ...])` continue to qualify — shared subcommands unaffected
- [ ] All existing tests in `internal/cli` pass (`go test ./internal/cli`)

**Tests**:
- `"it does not qualify tree under note"` — `qualifyCommand("note", ["tree"])` returns `("note", ["tree"])`
- `"it preserves args when tree is not qualified under note"` — `qualifyCommand("note", ["tree", "--foo"])` returns `("note", ["tree", "--foo"])`
- `"it still qualifies add under note"` — `qualifyCommand("note", ["add", "tick-aaa", "hello"])` returns `("note add", ["tick-aaa", "hello"])`
- `"it still qualifies remove under note"` — `qualifyCommand("note", ["remove", "tick-aaa", "1"])` returns `("note remove", ["tick-aaa", "1"])`
- `"it still qualifies tree under dep"` — existing tests in `TestDepTreeWiring` must continue to pass
- `"it still qualifies add and remove under dep"` — existing test `"it does not break existing dep add/remove dispatch"` must continue to pass

**Edge Cases**:
- `qualifyCommand("note", ["tree"])` must return `("note", ["tree"])` not `("note tree", [])` — the core bug
- Shared `add`/`remove` must still qualify under both `dep` and `note` — ensure the refactor does not accidentally break them
- `qualifyCommand("note", [])` — empty args, returns `("note", [])` — existing guard at line 370-372 handles this, unchanged
- `qualifyCommand("note", ["unknown"])` — unknown sub-subcommand, returns `("note", ["unknown"])` — existing default handles this, unchanged

**Context**:
> The root cause is at `app.go:374-376` where a single `case "add", "remove", "tree"` applies to both `"dep"` and `"note"` parents without any parent-specific guard. The `commandFlags` map in `flags.go` has entries for `"dep tree"`, `"note add"`, and `"note remove"` but NOT `"note tree"` — confirming `"note tree"` was never intended as a valid command. The fix is minimal: split `"tree"` into its own case with a `subcmd == "dep"` guard.

**Spec Reference**: `.workflows/qualify-command-tree-leaks-to-note/specification/qualify-command-tree-leaks-to-note/specification.md` — "Root Cause" and "Fix" sections

---

## qualify-command-tree-leaks-to-note-1-2 | approved

### Task 2: Integration Regression Test via App.Run

**Problem**: The unit-level fix to `qualifyCommand` (Task 1) ensures the function returns correct values, but the bug's user-visible symptom is a misleading error message surfaced through the full `App.Run` dispatch pipeline. Without an integration test exercising the complete path (`App.Run` -> `qualifyCommand` -> `ValidateFlags` -> `handleNote`), a future refactor could reintroduce the leak at a different layer (e.g., in flag validation or dispatch routing).

**Solution**: Add integration tests that exercise the full `App.Run` pipeline for `tick note tree` scenarios, verifying the user-visible error messages and exit codes. Also verify that `tick dep tree` behavior remains unchanged. These tests complement the unit tests from Task 1 by covering the end-to-end path.

**Outcome**: Integration tests confirm that (1) `tick note tree --foo` produces an error referencing `"note"` not `"note tree"`, (2) `tick note tree` produces `"unknown note sub-command 'tree'"`, and (3) `tick dep tree` continues to work correctly.

**Do**:
1. In `internal/cli/note_test.go`, add a new test function `TestNoteTreeRejection` (or add subtests to an existing function) with the following cases:
   - **`tick note tree` with no flags**: Run via `App.Run(["tick", "note", "tree"])`. Expect non-zero exit code. Expect stderr to contain `"unknown note sub-command 'tree'"`. This exercises the `handleNote` default case.
   - **`tick note tree --foo` with unknown flag**: Run via `App.Run(["tick", "note", "tree", "--foo"])`. Expect non-zero exit code. Expect stderr error to reference `"note"` (not `"note tree"`). Specifically, the error should be the `ValidateFlags` error for the `"note"` command (since `qualifyCommand` no longer qualifies it as `"note tree"`), or the `handleNote` unknown-subcommand error — either way, `"note tree"` must NOT appear in the error message.
   - **`tick dep tree` unchanged**: Run via the existing `runDepTree` helper or `App.Run(["tick", "--pretty", "dep", "tree"])` against a project with `setupTickProject`. Expect exit code 0. This is a sanity check that the fix did not break dep tree.
   - **`tick dep tree --unknown` unchanged**: Run `ValidateFlags("dep tree", ["--unknown"], commandFlags)`. Expect error containing `"dep tree"`. This confirms dep tree flag validation still references `"dep tree"`.
2. Use `setupTickProject(t)` for tests that don't need task data, `setupTickProjectWithTasks(t, tasks)` for those that do.
3. For the `tick note tree --foo` test, the exact error path depends on which check fires first. After the Task 1 fix, `qualifyCommand("note", ["tree", "--foo"])` returns `("note", ["tree", "--foo"])`. Then `ValidateFlags("note", ["tree", "--foo"], commandFlags)` runs. Since `"note"` is not in `commandFlags` (only `"note add"` and `"note remove"` are), and `"--foo"` is flag-like but not a global flag, this will produce `unknown flag "--foo" for "note"`. The test should verify the error message contains `"note"` and does NOT contain `"note tree"`.

   Note: If `"note"` is not a key in `commandFlags`, `ValidateFlags` will return `cmdFlags` as nil, meaning ALL non-global flags fail. The test should confirm the error references `"note"` regardless.
4. Run `go test ./internal/cli` to confirm all tests pass.

**Acceptance Criteria**:
- [ ] `tick note tree` (via `App.Run`) exits non-zero with stderr containing `"unknown note sub-command 'tree'"`
- [ ] `tick note tree --foo` (via `App.Run`) exits non-zero with stderr error referencing `"note"` — NOT `"note tree"`
- [ ] `tick dep tree` (via `App.Run`) exits zero — behavior unchanged
- [ ] `tick dep tree --unknown` flag validation error still references `"dep tree"` — behavior unchanged
- [ ] Shared subcommands: `tick note add` and `tick note remove` continue to work (existing tests pass)
- [ ] All existing tests in `internal/cli` pass

**Tests**:
- `"it rejects tree as unknown note sub-command"` — `App.Run(["tick", "note", "tree"])` with `setupTickProject`, exits non-zero, stderr contains `"unknown note sub-command 'tree'"`
- `"it does not reference note tree in flag error"` — `App.Run(["tick", "note", "tree", "--foo"])`, exits non-zero, stderr contains `"note"`, stderr does NOT contain `"note tree"`
- `"it preserves dep tree dispatch"` — `App.Run(["tick", "--pretty", "dep", "tree"])` with `setupTickProject`, exits zero
- `"it preserves dep tree flag validation"` — `ValidateFlags("dep tree", ["--unknown"], commandFlags)` returns error containing `"dep tree"`

**Edge Cases**:
- `tick note tree --foo`: error must reference `"note"` not `"note tree"` — the primary regression this test guards against
- `tick note tree` (no flags): must still produce the `handleNote` unknown-subcommand error, not silently succeed
- `tick dep tree` behavior must be completely unchanged — the fix is note-specific

**Context**:
> After the Task 1 fix, `qualifyCommand("note", ["tree", "--foo"])` returns `("note", ["tree", "--foo"])`. The `ValidateFlags` call then validates flags against command `"note"`. Since `"note"` itself is not a key in the `commandFlags` map (only `"note add"` and `"note remove"` are), `cmdFlags` will be nil. Any flag-like arg (e.g., `"--foo"`) will fail with `unknown flag "--foo" for "note"`. This is acceptable — `"note"` is not a leaf command that accepts flags directly; it always delegates to sub-subcommands. The important thing is the error no longer references the non-existent `"note tree"` command. After flag validation passes (or when there are no flags), dispatch goes to `handleNote` with the full `subArgs` including `"tree"`, which correctly hits the unknown-subcommand error.

**Spec Reference**: `.workflows/qualify-command-tree-leaks-to-note/specification/qualify-command-tree-leaks-to-note/specification.md` — "Acceptance Criteria" section, points 1-5
