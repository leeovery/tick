---
topic: auto-cascade-parent-status
cycle: 1
total_proposed: 6
---
# Analysis Tasks: Auto-Cascade Parent Status (Cycle 1)

## Task 1: Pretty format cascade tree rendering
status: approved
severity: high
sources: standards, architecture

**Problem**: The spec explicitly shows pretty format rendering cascades as a nested tree with box-drawing characters (grandchildren indented under parent children), but the implementation renders all cascade entries as a flat list at the same indentation level. Additionally, unchanged terminal entries are only collected for direct children, not deeper descendants. The CascadeEntry type lacks parent/hierarchy information, making nested rendering impossible with the current data model.

**Solution**: Add a Parent field to CascadeEntry (or add hierarchy information to CascadeResult), then update PrettyFormatter.FormatCascadeTransition to render nested tree output matching the spec examples. Also update buildCascadeResult to collect unchanged terminal entries recursively for all descendants, not just direct children. JSON format remains flat per spec.

**Outcome**: Pretty format cascade output matches the spec's nested tree structure with proper box-drawing indentation for grandchildren and deeper descendants. Unchanged terminal descendants at all levels appear in the output.

**Do**:
1. Add a `ParentID` field to `CascadeEntry` in `internal/cli/format.go` to carry hierarchy information
2. Update `buildCascadeResult` in `internal/cli/transition.go` to populate `ParentID` on each cascade entry and to collect unchanged terminal entries recursively (all descendants, not just direct children)
3. Update `PrettyFormatter.FormatCascadeTransition` in `internal/cli/pretty_formatter.go` to build a tree from the flat entries using `ParentID`, then render with nested box-drawing characters matching the spec examples
4. Verify ToonFormatter and JSONFormatter remain unaffected (flat output)

**Acceptance Criteria**:
- Pretty format downward cascade output shows grandchildren indented under their parent child entries with nested box-drawing characters
- Pretty format upward cascade output renders correctly for multi-level ancestor chains
- Unchanged terminal descendants at all hierarchy levels appear in the output
- Toon format remains flat with `(auto)` and `(unchanged)` markers
- JSON format remains a flat array structure

**Tests**:
- Test pretty format downward cascade with 3-level hierarchy (parent -> children -> grandchildren) matches spec example
- Test pretty format upward cascade with grandparent chain matches spec example
- Test unchanged terminal grandchildren appear in pretty output
- Test toon and JSON formats unchanged by the refactor

## Task 2: Move Rule 9 out of Transition into ApplyWithCascades
status: approved
severity: medium
sources: standards, architecture

**Problem**: The spec defines `Transition(t *Task, action string)` but the implementation uses `Transition(t *Task, action string, tasks []Task)` because Rule 9 (block reopen under cancelled parent) needs the full task list. This leaks a contextual concern into the core transition method. The backward-compatible wrapper `task.Transition()` passes nil for tasks, silently skipping Rule 9 validation -- a correctness trap for any caller not using ApplyWithCascades.

**Solution**: Move Rule 9 checking into `ApplyWithCascades` (which already has the task list) and restore `Transition` to its original two-parameter signature `(t *Task, action string)`. Rule 9 is a contextual pre-condition like cascade rules, not an intrinsic status transition rule.

**Outcome**: Transition signature matches the spec. Rule 9 is enforced consistently for all callers going through ApplyWithCascades. No nil-tasks escape hatch exists.

**Do**:
1. In `internal/task/state_machine.go`, remove the `tasks []Task` parameter from the `Transition` method on `StateMachine`
2. Move the Rule 9 check (block reopen under cancelled parent) into `ApplyWithCascades`, before calling `Transition`
3. Remove or update the backward-compatible `Transition` wrapper in `internal/task/transition.go` so it no longer needs to pass nil
4. Update all call sites that pass tasks to `Transition` directly
5. Verify Rule 9 is still enforced via ApplyWithCascades in all CLI commands that perform transitions

**Acceptance Criteria**:
- `StateMachine.Transition` accepts only `(t *Task, action string)` matching the spec
- Rule 9 (block reopen under cancelled parent) is enforced in ApplyWithCascades
- No caller can silently skip Rule 9 by using a different entry point
- All existing transition tests pass

**Tests**:
- Test that reopening a task under a cancelled parent returns error when going through ApplyWithCascades
- Test that Transition alone does not check parent status (it is a pure transition validator)
- Test backward-compatible wrapper still works for simple transitions

## Task 3: Extract Rule 3 completion evaluation into shared function
status: approved
severity: medium
sources: duplication, architecture

**Problem**: `evaluateRule3()` in `internal/cli/update.go` re-implements the "are all children terminal? done vs cancelled?" detection logic that already lives in `cascadeUpwardCompletion` in `internal/task/cascades.go`. Both build the same allTerminal/anyDone boolean pattern, determine the action string, and derive the new status. If Rule 3 criteria ever change, two places need updating.

**Solution**: Extract the "should this parent auto-complete and with what action" evaluation into a shared function in the task package (e.g., `EvaluateParentCompletion(tasks []Task, parentID string) (action string, shouldComplete bool)`). Both `cascadeUpwardCompletion` and `evaluateRule3` call it.

**Outcome**: Rule 3 completion detection logic exists in exactly one place in `internal/task/`. Changes to Rule 3 criteria only require one update.

**Do**:
1. Add `EvaluateParentCompletion(tasks []Task, parentID string) (action string, shouldComplete bool)` to `internal/task/cascades.go` (or a new file in the task package)
2. Refactor `cascadeUpwardCompletion` to call `EvaluateParentCompletion` instead of inline child-scanning
3. Refactor `evaluateRule3` in `internal/cli/update.go` to call `EvaluateParentCompletion` instead of reimplementing the logic
4. Ensure both callers handle the result identically to their current behavior

**Acceptance Criteria**:
- Child-scanning logic for Rule 3 exists in one place only
- cascadeUpwardCompletion behavior unchanged
- evaluateRule3 / reparenting behavior unchanged
- All existing tests pass

**Tests**:
- Test EvaluateParentCompletion with all children done returns ("done", true)
- Test EvaluateParentCompletion with all children cancelled returns ("cancel", true)
- Test EvaluateParentCompletion with mixed terminal children returns ("done", true)
- Test EvaluateParentCompletion with non-terminal children returns ("", false)
- Test reparenting still triggers Rule 3 on original parent

## Task 4: Extract cascade output helper function
status: approved
severity: medium
sources: duplication

**Problem**: The same conditional pattern for rendering cascade output appears 4 times across 3 files (`transition.go`, `create.go`, `update.go`): check if cascades slice is empty, call FormatTransition for simple output or buildCascadeResult + FormatCascadeTransition for cascade output. Each instance is 5-7 lines with identical structure differing only in variable names.

**Solution**: Extract a helper function like `outputTransitionOrCascade(stdout io.Writer, fmtr Formatter, id, title string, result TransitionResult, cascades []CascadeChange, tasks []Task) error` that encapsulates the conditional. All four call sites reduce to a single function call.

**Outcome**: Cascade output rendering logic exists in one place. Adding new commands that trigger cascades requires one function call instead of duplicating the pattern.

**Do**:
1. Add `outputTransitionOrCascade` function in `internal/cli/transition.go` (or a shared helpers file in the cli package)
2. Replace the 4 duplicate blocks in `transition.go:56-63`, `create.go:293-298`, `update.go:447-453`, and `update.go:457-463` with calls to the new helper
3. Verify output is identical before and after

**Acceptance Criteria**:
- All 4 call sites use the extracted helper
- No duplicate cascade output pattern remains
- CLI output is identical for all transition/create/update commands

**Tests**:
- Existing CLI integration tests for transition, create, and update commands pass unchanged
- Test the helper directly with empty cascades (should call FormatTransition)
- Test the helper with non-empty cascades (should call FormatCascadeTransition)

## Task 5: Extract parent validation and reopen helper
status: approved
severity: medium
sources: duplication

**Problem**: Both `RunCreate` and `RunUpdate` contain nearly identical 15-18 line blocks that iterate tasks to find the parent by ID, call `ValidateAddChild` to enforce Rule 7, check if parent status is done, and call `ApplyWithCascades` with "reopen" to trigger Rule 6. The only differences are variable names.

**Solution**: Extract a function like `validateAndReopenParent(tasks []Task, parentID string, sm *StateMachine) (TransitionResult, []CascadeChange, bool, error)` that encapsulates the validate-then-reopen pattern.

**Outcome**: Parent validation and Rule 6 reopen logic exists in one place. Both create.go and update.go call it.

**Do**:
1. Add `validateAndReopenParent` function in a shared location in the cli package (e.g., `internal/cli/parent_helpers.go` or in an existing shared file)
2. Replace the duplicate blocks in `create.go:228-246` and `update.go:338-357` with calls to the new function
3. Return values should indicate whether a reopen occurred and provide the transition result and cascade changes for output rendering

**Acceptance Criteria**:
- Both RunCreate and RunUpdate use the extracted helper
- Rule 7 (block add child to cancelled parent) still enforced
- Rule 6 (reopen done parent on new child) still triggered
- No behavioral change in create or update commands

**Tests**:
- Existing tests for creating a task under a done parent pass unchanged
- Existing tests for reparenting to a done parent pass unchanged
- Existing tests for creating/reparenting to a cancelled parent (error) pass unchanged

## Task 6: Defensive copy of task data for cascade display output
status: approved
severity: medium
sources: architecture

**Problem**: The `allTasks` variable is set to the `tasks` slice from inside the `Mutate` callback, then used after `Mutate` returns to build cascade display output. The CLI layer depends on an implementation detail of how Mutate manages its buffer lifecycle. If `Store.Mutate` ever reuses or clears the slice buffer, display code reads stale/corrupt data. Currently safe but fragile coupling.

**Solution**: Copy the display-relevant data inside the Mutate closure into dedicated output structs, rather than holding a reference to the mutable tasks slice. The CascadeChange structs already contain task pointers, so at minimum copy the unchanged children data needed for display.

**Outcome**: Display output code does not depend on Mutate's internal buffer lifecycle. Safe against future Store implementation changes.

**Do**:
1. In `internal/cli/transition.go`, `create.go`, and `update.go`, identify the data needed for display output after Mutate returns
2. Inside the Mutate closure, copy the needed data (task IDs, titles, statuses for unchanged children) into local structs rather than capturing the full tasks slice reference
3. Update `buildCascadeResult` to work with the copied data instead of the full tasks slice
4. Verify all cascade display output remains identical

**Acceptance Criteria**:
- No reference to the Mutate callback's tasks slice is held after Mutate returns
- Display output is built from copied data
- All existing CLI output tests pass unchanged

**Tests**:
- Existing transition, create, and update tests with cascades pass unchanged
- Verify output correctness for cascades with unchanged terminal children
