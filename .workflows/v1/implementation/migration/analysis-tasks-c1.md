---
topic: migration
cycle: 1
total_proposed: 4
---
# Analysis Tasks: Migration (Cycle 1)

## Task 1: Replace manual presenter calls in RunMigrate with Present function
status: approved
severity: high
sources: duplication, standards, architecture

**Problem**: `RunMigrate` in `internal/cli/migrate.go:123-133` manually calls `WriteHeader`, loops over results calling `WriteResult`, then calls `WriteSummary` -- but never calls `WriteFailures`. The `Present` function in `internal/migrate/presenter.go:74-81` does this exact sequence plus `WriteFailures`. The CLI therefore silently drops the spec-mandated "Failures:" detail section from output when tasks fail. All three analysis agents independently identified this.

**Solution**: Replace the manual presenter orchestration in `RunMigrate` with a single call to `migrate.Present()`. This requires restructuring `RunMigrate` slightly since the header is currently printed before `engine.Run()` while `Present` prints it after. Move the `Present` call to after `engine.Run()` completes, passing all results.

**Outcome**: The CLI renders the complete output including the failure detail section. No duplicated presenter orchestration. The `Present` function becomes the single point of output composition.

**Do**:
1. In `internal/cli/migrate.go`, remove lines 123-133 (the manual `WriteHeader`, result loop, and `WriteSummary` calls).
2. After `engine.Run()` returns results, call `migrate.Present(stdout, provider.Name(), dryRun, results)`.
3. The function should still return `runErr` after calling `Present` so partial results are displayed even on provider error.
4. Update CLI integration tests that assert on migration output to verify the failure detail section appears when failures exist.

**Acceptance Criteria**:
- `RunMigrate` calls `migrate.Present` instead of individual `Write*` functions.
- When tasks fail, the "Failures:" section appears in CLI output after the summary line.
- Existing passing tests continue to pass.
- A test case with failing tasks asserts the failure detail block is present in output.

**Tests**:
- Test `RunMigrate` with a provider that returns tasks causing validation failures; assert output contains both the cross-mark lines and the "Failures:" detail section with per-task error messages.
- Test `RunMigrate` with all-successful tasks; assert no "Failures:" section appears.

## Task 2: Surface beads provider parse/validation errors as failed results instead of silently dropping
status: approved
severity: high
sources: standards, architecture

**Problem**: The beads provider's `Tasks()` method in `internal/migrate/beads/beads.go:90-107` silently skips malformed JSON lines, empty-title entries, and validation failures. These skipped entries never reach the engine and never appear in the output. A user with 100 entries and 20 malformed ones sees "Done: 80 imported, 0 failed" with no indication that 20 entries were dropped. The spec requires "Continue on error, report failures at end."

**Solution**: Return entries with empty titles and invalid field values from the provider so the engine's validation can catch them and report them as failed Results with explanations in the output. Truly unparseable JSON lines (where no MigratedTask can be constructed) should be returned as sentinel MigratedTask values with a title indicating the parse error so they fail validation visibly.

**Outcome**: All entries in the JSONL file are accounted for in the migration output -- either as successful imports or as visible failures with explanations.

**Do**:
1. In `internal/migrate/beads/beads.go`, in the `Tasks()` method:
   - For malformed JSON lines (line 91-93): create a `MigratedTask` with a descriptive title like `"(malformed entry)"` and empty other fields. The engine's validation will reject it for the empty-title check or it will appear as a named failure. Alternatively, collect these as MigratedTask entries with a title describing the line number/error so the user sees them.
   - For entries with empty titles (line 96-99): remove the empty-title skip. Return the `MigratedTask` as-is with the empty title. The engine already handles empty-title validation at `engine.go:67-72` and produces a "(untitled)" result with the validation error.
   - For validation failures (line 102-104): remove the `Validate()` call from the provider. The engine already calls `Validate()` at `engine.go:67`. Double-validation in the provider pre-filters entries the engine should report.
2. Update beads provider tests to reflect that invalid entries are now returned rather than skipped.
3. Verify CLI integration tests show failures for invalid entries.

**Acceptance Criteria**:
- Empty-title entries from the JSONL file appear as failed results in migration output with a validation error message.
- Entries that fail validation (e.g., out-of-range priority) appear as failed results rather than being silently dropped.
- Malformed JSON lines produce visible failures in the output.
- Valid entries continue to import successfully.

**Tests**:
- Test beads provider with a JSONL file containing a mix of valid entries, empty-title entries, and malformed JSON lines; assert all are returned from `Tasks()`.
- Integration test: run migration against a fixture with invalid entries; assert the output shows the correct failed count and failure detail lines.

## Task 3: Consolidate inconsistent empty-title fallback strings
status: approved
severity: medium
sources: duplication

**Problem**: Three locations use different fallback strings for empty titles: `engine.go:69` uses `"(untitled)"`, while `presenter.go:27-29` (`WriteResult`) and `presenter.go:64-66` (`WriteFailures`) both use `"(unknown)"`. These serve the same purpose but drifted because they were written independently.

**Solution**: Define a single exported constant (e.g., `FallbackTitle`) in `internal/migrate/migrate.go` and reference it from `engine.go`, `WriteResult`, and `WriteFailures`. Use `"(untitled)"` as the canonical value since it more accurately describes the situation (the title is missing, not the task's identity).

**Outcome**: One consistent fallback string used everywhere, defined in one place.

**Do**:
1. In `internal/migrate/migrate.go`, add: `const FallbackTitle = "(untitled)"`.
2. In `internal/migrate/engine.go:70`, replace `"(untitled)"` with `FallbackTitle`.
3. In `internal/migrate/presenter.go:28`, replace `"(unknown)"` with `FallbackTitle`.
4. In `internal/migrate/presenter.go:64`, replace `"(unknown)"` with `FallbackTitle`.
5. Update any tests that assert on the `"(unknown)"` string to use the new constant value.

**Acceptance Criteria**:
- A single `FallbackTitle` constant exists in `migrate.go`.
- All three usage sites reference the constant.
- No hardcoded fallback title strings remain in the migrate package.
- Tests pass with the consolidated value.

**Tests**:
- Existing tests for `WriteResult`, `WriteFailures`, and `Engine.Run` with empty-title tasks should pass after updating expected strings.

## Task 4: Use task.Status type and constants instead of raw status strings
status: approved
severity: medium
sources: architecture, standards

**Problem**: The `task` package defines `type Status string` with constants (`StatusOpen`, `StatusDone`, `StatusCancelled`, `StatusInProgress`). The migrate package re-declares the same values as raw string literals in three independent maps: `validStatuses` in `migrate.go:11`, `completedStatuses` in `engine.go:6`, and `statusMap` in `beads/beads.go:18`. If a status value changes in the `task` package, these maps would silently produce wrong behavior. The `MigratedTask.Status` field is `string` rather than `task.Status`, so the type system cannot catch mismatches. In `store_creator.go:52`, a raw cast `task.Status(mt.Status)` converts the untyped string. Additionally, unknown beads status values (e.g., "wontfix") silently map to an empty string with no user visibility.

**Solution**: Change `MigratedTask.Status` to `task.Status` and replace the string-literal maps with references to `task` package constants. This makes the type system enforce correctness and eliminates the raw cast in `store_creator.go`.

**Outcome**: Status values are type-safe throughout the migration pipeline. Any drift between the task package and migration package becomes a compile error.

**Do**:
1. In `internal/migrate/migrate.go`, change `MigratedTask.Status` from `string` to `task.Status`. Add import for `task` package.
2. Replace `validStatuses` map with references to task constants: `task.StatusOpen`, `task.StatusInProgress`, `task.StatusDone`, `task.StatusCancelled`.
3. In `internal/migrate/engine.go`, replace `completedStatuses` map entries with `task.StatusDone` and `task.StatusCancelled`. The map key type becomes `task.Status`.
4. In `internal/migrate/beads/beads.go`, change `statusMap` values from `string` to `task.Status`: `task.StatusOpen`, `task.StatusInProgress`, `task.StatusDone`.
5. In `internal/migrate/store_creator.go:52`, remove the `task.Status(mt.Status)` cast since `mt.Status` is now already `task.Status`. The empty-string check (`mt.Status == ""`) still works since `task.Status` is an underlying string.
6. Update all tests that construct `MigratedTask` with string status values to use `task.Status` constants.

**Acceptance Criteria**:
- `MigratedTask.Status` is `task.Status` not `string`.
- No raw status string literals remain in `migrate.go`, `engine.go`, or `beads.go` maps.
- No `task.Status()` cast in `store_creator.go`.
- All tests compile and pass.

**Tests**:
- Existing unit tests for `Validate`, `Engine.Run`, `StoreTaskCreator.CreateTask`, and beads provider should pass after updating status values to use constants.
- Verify that providing an invalid status to `MigratedTask.Validate()` still returns an error.
