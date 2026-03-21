---
status: complete
created: 2026-03-21
cycle: 1
phase: Plan Integrity Review
topic: Cascade Unchanged Noise
---

# Review Tracking: Cascade Unchanged Noise - Integrity

## Findings

### 1. Task 1 produces a deliberately failing test, violating One Task = One TDD Cycle

**Severity**: Important
**Plan Reference**: Phase 1, Task 1-1 (cascade-unchanged-noise-1-1) and Task 1-2 (cascade-unchanged-noise-1-2)
**Category**: Vertical Slicing / Scope and Granularity
**Change Type**: remove-task

**Details**:
Task-design.md defines "One Task = One TDD Cycle: Write test -> implement -> pass -> commit." Task 1 explicitly states its test "fails against the current code" and acceptance criterion 6 reads "Test compiles and passes after Task 2 is applied (it is expected to fail before Task 2)." This means Task 1 cannot be independently verified -- it produces a red test that only goes green after Task 2. The two tasks are not two TDD cycles; they are one cycle split across two tasks.

The independence test from task-design.md asks: "Can I write a test for this task that passes without any other task being complete?" For Task 1, the answer is no. This means an implementer cannot commit after Task 1 (failing test in the suite), cannot verify Task 1 is correct in isolation, and the phase checkpoint ("all existing tests pass") is violated between tasks.

Since all the changes are pure deletion in a single package and the test + deletion together form one natural TDD cycle, Task 1 should be merged into Task 2. Task 2 already covers the deletion; adding the negative-case test as a first step is a natural fit and keeps the Do section conceptually cohesive (all steps serve the same goal).

**Current**:
The plan has two separate tasks:

Task 1-1 (tick-4b03c8): "Add negative-case test confirming terminal siblings excluded from cascade output" -- writes a test that deliberately fails until Task 2 is applied.

Task 1-2 (tick-f9cb01): "Remove UnchangedEntry type and all unchanged collection and rendering" -- performs the deletion that makes Task 1's test pass.

**Proposed**:
Remove Task 1-1 as a standalone task. Merge its content into Task 1-2 (which becomes the sole task in the phase). See Finding 2 for the updated Task 1-2 content.

**Resolution**: Fixed
**Notes**: If the user prefers to keep two tasks, Task 1 needs restructuring so the test passes on current code (e.g., asserting the current behavior, then Task 2 rewrites it). But that adds unnecessary churn. Merging is cleaner.

---

### 2. Merged task content for single-task phase

**Severity**: Important
**Plan Reference**: Phase 1, Task 1-2 (cascade-unchanged-noise-1-2) -- updated to absorb Task 1-1
**Category**: Vertical Slicing / Scope and Granularity
**Change Type**: update-task

**Details**:
This is the companion to Finding 1. If Task 1-1 is removed, Task 1-2 must absorb the negative-case test as its first Do step. The planning.md task table, phase acceptance criteria, and task detail all need updating. The merged task remains a single TDD cycle: write the negative-case test (red), perform the deletion (green), verify all tests pass.

Note on scope: the merged Do section has 9 steps, which exceeds the soft guideline of 5. However, all steps are mechanical deletions of a single feature across files in one package, plus one test addition. The conceptual complexity is low. Splitting would create horizontal slices (e.g., "delete from formatters" vs "delete from types") that can't be independently verified.

**Current**:
Planning.md task table:
```markdown
#### Tasks
status: approved
approved_at: 2026-03-21

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| cascade-unchanged-noise-1-1 | Add negative-case test confirming terminal siblings excluded from cascade output | terminal child with done status, terminal child with cancelled status |
| cascade-unchanged-noise-1-2 | Remove UnchangedEntry type and all unchanged collection and rendering | pretty formatter tree node count with only cascaded entries, JSON output no longer includes unchanged key, empty cascade result still renders correctly |
```

Phase 1 task detail for cascade-unchanged-noise-1-2 (current Do section):
```markdown
**Do**:

1. **`internal/cli/format.go`** -- Remove type and field:
   - Delete the `UnchangedEntry` struct (lines 139-145).
   - Remove the `Unchanged []UnchangedEntry` field from `CascadeResult` (line 155).

2. **`internal/cli/transition.go`** -- Remove collection logic from `buildCascadeResult()`:
   - Delete the `involvedIDs` map construction (lines 110-115): the map that merges primary + cascaded IDs exists solely to support the unchanged loop and is dead code without it.
   - Delete the unchanged collection loop (lines 117-135): the `for i := range tasks` block that walks descendants of involved tasks and appends terminal ones to `cr.Unchanged`.
   - Keep the `cascadedIDs` map (lines 93-108) -- it is used to build the `Cascaded` entries and detect upward cascades. Only `involvedIDs` and the final loop are removed.

3. **`internal/cli/toon_formatter.go`** -- Remove unchanged rendering in `FormatCascadeTransition()` (lines 154-156):
   - Delete the `for _, u := range result.Unchanged` loop that appends `"(unchanged)"` lines.

4. **`internal/cli/pretty_formatter.go`** -- Remove unchanged rendering in `FormatCascadeTransition()`:
   - Delete the `totalEntries` calculation on line 206 that sums `len(result.Cascaded) + len(result.Unchanged)`. Replace the `totalEntries == 0` guard (line 207) with `len(result.Cascaded) == 0`.
   - Delete the unchanged node construction loop (lines 224-231): the `for _, u := range result.Unchanged` block that creates `cascadeNode` entries with `"(unchanged)"` text.
   - Delete the `parentIDOf` population for unchanged entries (lines 239-241): the `for _, u := range result.Unchanged` block.

5. **`internal/cli/json_formatter.go`** -- Remove unchanged from JSON output:
   - Delete the `jsonUnchangedEntry` struct (lines 275-279).
   - Remove the `Unchanged []jsonUnchangedEntry` field from `jsonCascadeResult` (line 285).
   - In `FormatCascadeTransition()`, delete the unchanged slice construction (lines 304-311): the `unchanged := make(...)` and `for _, u := range result.Unchanged` block.
   - Remove `Unchanged: unchanged` from the `jsonCascadeResult` construction (line 320 in the `marshalIndentJSON` call).

6. **`internal/cli/cascade_formatter_test.go`** -- Update tests:
   - `TestToonFormatterCascadeTransition`:
     - `"it renders downward cancel cascade flat with ParentID present"` (line 13): Remove `Unchanged` field from `CascadeResult` construction. Update `expected` to remove the `"tick-child3: done (unchanged)"` line.
     - `"it renders upward start cascade"` (line 37): Remove `Unchanged: nil` field.
     - `"it renders single cascade entry"` (line 58): Remove `Unchanged: nil` field.
   - `TestPrettyFormatterCascadeTransition`:
     - `"it renders downward cancel cascade with tree"` (line 79): Remove `Unchanged` field. Update `expected` to remove the `tick-child3 "Logout": done (unchanged)` tree line, and update the last cascaded entry's connector from `├─` to `└─` (it becomes the last entry).
     - `"it renders mixed cascaded and unchanged children"` (line 105): **Delete this entire subtest** -- it exists solely to test mixed cascaded+unchanged rendering.
     - `"it renders downward cascade with 3-level hierarchy"` (line 137): Remove `Unchanged` field. Update `expected` to remove the `tick-child3 "Logout": done (unchanged)` line, and update the last cascaded entry's tree connector accordingly (tick-child2 becomes the last root, so `├─` for tick-child1 stays, `└─` for tick-child2).
     - `"it renders upward cascade with grandparent chain"` (line 167): Remove `Unchanged: nil` field.
     - `"it renders unchanged terminal grandchildren in tree"` (line 195): **Delete this entire subtest** -- it exists solely to test unchanged grandchild tree rendering.
   - `TestJSONFormatterCascadeTransition`:
     - `"it renders cascade as structured object"` (line 223): Remove `Unchanged` field from construction. Delete the entire "Verify unchanged array" assertion block (lines 280-297).
     - `"it renders empty cascaded array as []"` (line 300): Remove `Unchanged` field. Delete the unchanged array assertion. This test now only verifies empty cascaded renders as `[]`.
   - `TestBuildCascadeResult`:
     - `"it populates ParentID on cascade entries"` (line 331): No changes needed (does not assert on Unchanged).
     - `"it collects unchanged terminal descendants recursively"` (line 356): **Delete this entire subtest**.
     - `"it populates ParentID on unchanged entries for direct children"` (line 383): **Delete this entire subtest**.
   - `TestAllFormattersCascadeEmptyArrays` (line 405):
     - Remove `Unchanged: nil` from the `CascadeResult` construction.
     - Delete the JSON unchanged array assertions (lines 444-450).

7. **`internal/cli/format_test.go`** -- Update tests:
   - `TestCascadeTypes`:
     - `"it compiles with FormatCascadeTransition on all formatter types"` (line 365): Remove the `Unchanged` field from the `CascadeResult` construction (line 388-390).
     - `"it returns empty string from stub implementation"` (line 396): No `Unchanged` field to remove (not present).
   - `TestCascadeResultStruct`:
     - `"it holds all cascade data fields"` (line 430): Remove the `Unchanged` field from construction (lines 440-442). Delete the assertions on `result.Unchanged` (lines 462-468).

8. **`internal/cli/transition_test.go`** -- Update tests:
   - Delete the `"it includes unchanged terminal children in cascade output"` subtest (lines 514-546) entirely. Task 1's negative-case test replaces it.
```

**Proposed**:
Planning.md task table (single task):
```markdown
#### Tasks
status: approved
approved_at: 2026-03-21

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| cascade-unchanged-noise-1-1 | Remove UnchangedEntry type and all unchanged collection and rendering, with negative-case test | terminal child with done status excluded, terminal child with cancelled status excluded, pretty formatter tree node count with only cascaded entries, JSON output no longer includes unchanged key, empty cascade result still renders correctly |
```

Phase 1 task detail (full replacement of phase-1-tasks.md):
```markdown
# Phase 1: Remove unchanged cascade output

## cascade-unchanged-noise-1-1 | approved

### Task 1: Remove UnchangedEntry type and all unchanged collection and rendering, with negative-case test

**Problem**: `buildCascadeResult()` in `internal/cli/transition.go` deliberately collects terminal descendants not affected by a cascade into `CascadeResult.Unchanged`, and all three formatters render these entries. This produces "(unchanged)" lines that carry no actionable information and create visual noise scaling with hierarchy depth. Additionally, the current test suite asserts unchanged terminal siblings appear in cascade output (line 514 of `transition_test.go`), and no negative-case test exists to confirm the desired behavior after removal.

**Solution**: Remove the `UnchangedEntry` type, the `Unchanged` field from `CascadeResult`, the collection logic in `buildCascadeResult()`, the rendering code in all three formatters, and update all tests that reference unchanged behavior. Add a negative-case integration test confirming terminal siblings do NOT appear in cascade output. This is a pure deletion with one new test to lock in the fixed behavior.

**Outcome**: Cascade output shows only the primary transition and actually-cascaded entries. No "(unchanged)" lines appear in any format. A negative-case test confirms terminal siblings are excluded. All tests pass.

**Do**:

1. **`internal/cli/transition_test.go`** -- Add negative-case test and remove old test:
   - In `TestTransitionCommands`, add a new subtest: `"it excludes terminal siblings from cascade output"`.
   - Set up three children under a parent: one open child (will cascade), one done child, one cancelled child. The parent is `in_progress`.
   - Cancel the parent via `runTransition(t, dir, "cancel", parentID)`.
   - Assert `exitCode == 0`.
   - Assert stdout contains the cascaded child's ID (the open child that was cancelled via cascade).
   - Assert stdout does NOT contain the done child's ID.
   - Assert stdout does NOT contain the cancelled child's ID.
   - Assert stdout does NOT contain the string `"unchanged"`.
   - Delete the `"it includes unchanged terminal children in cascade output"` subtest (lines 514-546) entirely.

2. **`internal/cli/format.go`** -- Remove type and field:
   - Delete the `UnchangedEntry` struct (lines 139-145).
   - Remove the `Unchanged []UnchangedEntry` field from `CascadeResult` (line 155).

3. **`internal/cli/transition.go`** -- Remove collection logic from `buildCascadeResult()`:
   - Delete the `involvedIDs` map construction (lines 110-115): the map that merges primary + cascaded IDs exists solely to support the unchanged loop and is dead code without it.
   - Delete the unchanged collection loop (lines 117-135): the `for i := range tasks` block that walks descendants of involved tasks and appends terminal ones to `cr.Unchanged`.
   - Keep the `cascadedIDs` map (lines 93-108) -- it is used to build the `Cascaded` entries and detect upward cascades. Only `involvedIDs` and the final loop are removed.

4. **`internal/cli/toon_formatter.go`** -- Remove unchanged rendering in `FormatCascadeTransition()` (lines 154-156):
   - Delete the `for _, u := range result.Unchanged` loop that appends `"(unchanged)"` lines.

5. **`internal/cli/pretty_formatter.go`** -- Remove unchanged rendering in `FormatCascadeTransition()`:
   - Delete the `totalEntries` calculation on line 206 that sums `len(result.Cascaded) + len(result.Unchanged)`. Replace the `totalEntries == 0` guard (line 207) with `len(result.Cascaded) == 0`.
   - Delete the unchanged node construction loop (lines 224-231): the `for _, u := range result.Unchanged` block that creates `cascadeNode` entries with `"(unchanged)"` text.
   - Delete the `parentIDOf` population for unchanged entries (lines 239-241): the `for _, u := range result.Unchanged` block.

6. **`internal/cli/json_formatter.go`** -- Remove unchanged from JSON output:
   - Delete the `jsonUnchangedEntry` struct (lines 275-279).
   - Remove the `Unchanged []jsonUnchangedEntry` field from `jsonCascadeResult` (line 285).
   - In `FormatCascadeTransition()`, delete the unchanged slice construction (lines 304-311): the `unchanged := make(...)` and `for _, u := range result.Unchanged` block.
   - Remove `Unchanged: unchanged` from the `jsonCascadeResult` construction (line 320 in the `marshalIndentJSON` call).

7. **`internal/cli/cascade_formatter_test.go`** -- Update tests:
   - `TestToonFormatterCascadeTransition`:
     - `"it renders downward cancel cascade flat with ParentID present"` (line 13): Remove `Unchanged` field from `CascadeResult` construction. Update `expected` to remove the `"tick-child3: done (unchanged)"` line.
     - `"it renders upward start cascade"` (line 37): Remove `Unchanged: nil` field.
     - `"it renders single cascade entry"` (line 58): Remove `Unchanged: nil` field.
   - `TestPrettyFormatterCascadeTransition`:
     - `"it renders downward cancel cascade with tree"` (line 79): Remove `Unchanged` field. Update `expected` to remove the `tick-child3 "Logout": done (unchanged)` tree line, and update the last cascaded entry's connector from `├─` to `└─` (it becomes the last entry).
     - `"it renders mixed cascaded and unchanged children"` (line 105): **Delete this entire subtest** -- it exists solely to test mixed cascaded+unchanged rendering.
     - `"it renders downward cascade with 3-level hierarchy"` (line 137): Remove `Unchanged` field. Update `expected` to remove the `tick-child3 "Logout": done (unchanged)` line, and update the last cascaded entry's tree connector accordingly (tick-child2 becomes the last root, so `├─` for tick-child1 stays, `└─` for tick-child2).
     - `"it renders upward cascade with grandparent chain"` (line 167): Remove `Unchanged: nil` field.
     - `"it renders unchanged terminal grandchildren in tree"` (line 195): **Delete this entire subtest** -- it exists solely to test unchanged grandchild tree rendering.
   - `TestJSONFormatterCascadeTransition`:
     - `"it renders cascade as structured object"` (line 223): Remove `Unchanged` field from construction. Delete the entire "Verify unchanged array" assertion block (lines 280-297).
     - `"it renders empty cascaded array as []"` (line 300): Remove `Unchanged` field. Delete the unchanged array assertion. This test now only verifies empty cascaded renders as `[]`.
   - `TestBuildCascadeResult`:
     - `"it populates ParentID on cascade entries"` (line 331): No changes needed (does not assert on Unchanged).
     - `"it collects unchanged terminal descendants recursively"` (line 356): **Delete this entire subtest**.
     - `"it populates ParentID on unchanged entries for direct children"` (line 383): **Delete this entire subtest**.
   - `TestAllFormattersCascadeEmptyArrays` (line 405):
     - Remove `Unchanged: nil` from the `CascadeResult` construction.
     - Delete the JSON unchanged array assertions (lines 444-450).

8. **`internal/cli/format_test.go`** -- Update tests:
   - `TestCascadeTypes`:
     - `"it compiles with FormatCascadeTransition on all formatter types"` (line 365): Remove the `Unchanged` field from the `CascadeResult` construction (line 388-390).
     - `"it returns empty string from stub implementation"` (line 396): No `Unchanged` field to remove (not present).
   - `TestCascadeResultStruct`:
     - `"it holds all cascade data fields"` (line 430): Remove the `Unchanged` field from construction (lines 440-442). Delete the assertions on `result.Unchanged` (lines 462-468).

**Acceptance Criteria**:
- [ ] Test `"it excludes terminal siblings from cascade output"` exists in `TestTransitionCommands` in `internal/cli/transition_test.go`
- [ ] Test asserts terminal sibling with `done` status is absent from output
- [ ] Test asserts terminal sibling with `cancelled` status is absent from output
- [ ] Test asserts the string "unchanged" is absent from output
- [ ] Test asserts the cascaded (non-terminal) child IS present in output
- [ ] `UnchangedEntry` type no longer exists in `internal/cli/format.go`
- [ ] `CascadeResult` struct no longer has an `Unchanged` field
- [ ] `buildCascadeResult()` no longer contains the `involvedIDs` map or the unchanged collection loop
- [ ] `ToonFormatter.FormatCascadeTransition()` does not iterate over or render unchanged entries
- [ ] `PrettyFormatter.FormatCascadeTransition()` uses `len(result.Cascaded) == 0` as the empty guard, does not create nodes for unchanged entries
- [ ] `JSONFormatter.FormatCascadeTransition()` does not include `jsonUnchangedEntry` type, does not produce an `"unchanged"` key in output
- [ ] `"it includes unchanged terminal children in cascade output"` subtest is removed from `transition_test.go`
- [ ] All deleted subtests (`"it renders mixed cascaded and unchanged children"`, `"it renders unchanged terminal grandchildren in tree"`, `"it collects unchanged terminal descendants recursively"`, `"it populates ParentID on unchanged entries for direct children"`) are gone
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes (all packages)

**Tests**:
- `"it excludes terminal siblings from cascade output"` -- negative-case integration test confirming terminal siblings absent
- `"it renders downward cancel cascade flat with ParentID present"` -- updated, no unchanged line in expected output
- `"it renders upward start cascade"` -- compiles without Unchanged field
- `"it renders single cascade entry"` -- compiles without Unchanged field
- `"it renders downward cancel cascade with tree"` -- updated, tree connectors adjusted for removed unchanged entry
- `"it renders downward cascade with 3-level hierarchy"` -- updated, tree connectors adjusted
- `"it renders upward cascade with grandparent chain"` -- compiles without Unchanged field
- `"it renders cascade as structured object"` -- JSON output no longer has unchanged key
- `"it renders empty cascaded array as []"` -- only cascaded array checked
- `"it populates ParentID on cascade entries"` -- unchanged, still passes
- `"all formatters handle both empty cascaded and unchanged"` -- renamed/updated, no unchanged assertions
- `"it compiles with FormatCascadeTransition on all formatter types"` -- no Unchanged field
- `"it holds all cascade data fields"` -- no Unchanged assertions

**Edge Cases**:
- Terminal child with `done` status must not appear in cascade output -- covered by the negative-case test including a done sibling
- Terminal child with `cancelled` status must not appear in cascade output -- covered by including a cancelled sibling in the same test
- Pretty formatter tree node count with only cascaded entries: the `totalEntries` guard changes from `len(Cascaded) + len(Unchanged) == 0` to `len(Cascaded) == 0`. When there are zero cascaded entries, no "Cascaded:" header or tree is rendered -- just the primary transition line. This matches the existing behavior for the `TestAllFormattersCascadeEmptyArrays` test.
- JSON output no longer includes unchanged key: the `jsonCascadeResult` struct loses the `Unchanged` field, so `json.MarshalIndent` will not emit an `"unchanged"` key at all. The existing test `"it renders cascade as structured object"` is updated to not look for it.
- Empty cascade result still renders correctly: the `TestAllFormattersCascadeEmptyArrays` test continues to verify that all three formatters handle a `CascadeResult` with nil `Cascaded` (and no `Unchanged` field at all). Toon and Pretty render just the primary transition; JSON renders `{"transition":..., "cascaded":[]}`.

**Context**:
> The specification is explicit: "Remove the unchanged feature entirely. No flags, no conditional logic -- clean deletion." The `involvedIDs` map on lines 110-115 of `transition.go` exists solely to power the unchanged collection loop. The `cascadedIDs` map (lines 93-108) is **not** removed -- it supports the `Cascaded` entry construction and the upward-cascade detection logic.
>
> The specification notes: `internal/cli/helpers_test.go` "Uses `buildCascadeResult` but does not assert on `Unchanged` directly; should compile cleanly after type changes with no modifications needed." This means `helpers_test.go` requires no edits.
>
> For the JSON formatter, the specification says to "Remove unchanged array from JSON output." After removal, the `jsonCascadeResult` struct will have only `Transition` and `Cascaded` fields. This means the JSON output shape changes from `{"transition":..., "cascaded":[], "unchanged":[]}` to `{"transition":..., "cascaded":[]}`. This is acceptable because the tool is pre-v1 (no backwards compatibility concerns).
>
> The test uses `runTransition` helper (line 14 of `transition_test.go`) which sets `IsTTY: true`, defaulting to PrettyFormatter. Since PrettyFormatter renders unchanged entries in the tree, the negative assertion (no unchanged IDs in output) covers the pretty format path. The toon format is covered by the formatter-level tests updated in the same task.

**Spec Reference**: `.workflows/cascade-unchanged-noise/specification/cascade-unchanged-noise/specification.md` -- "Fix" section (all 8 numbered items) and "Testing" section
```

Planning.md phase acceptance criteria update -- add the negative-case test criterion:
```markdown
**Acceptance**:
- [ ] `UnchangedEntry` type and `Unchanged` field no longer exist in `CascadeResult`
- [ ] `buildCascadeResult()` no longer collects unchanged terminal descendants (involvedIDs map and collection loop removed)
- [ ] Toon, Pretty, and JSON formatters no longer render unchanged entries
- [ ] A test confirms that terminal siblings are NOT included in cascade output (the negative case)
- [ ] All existing tests in `internal/cli/` pass (updated to remove Unchanged references)
- [ ] `go vet ./...` and full test suite `go test ./...` pass with no failures
```

**Resolution**: Fixed
**Notes**: This finding is paired with Finding 1. Together they propose collapsing from 2 tasks to 1 task. The internal ID changes from cascade-unchanged-noise-1-2 to cascade-unchanged-noise-1-1 (renumbered as sole task). The tick task management entries (tick-4b03c8 removed, tick-f9cb01 becomes the sole child of tick-dcb013) would need updating by the orchestrator.

---
