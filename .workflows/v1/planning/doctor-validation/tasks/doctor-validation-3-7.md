---
id: doctor-validation-3-7
phase: 3
status: approved
created: 2026-01-31
---

# Relationship Check Registration

## Goal

Tasks 3-1 through 3-6 implemented six new diagnostic checks (`OrphanedParentCheck`, `OrphanedDependencyCheck`, `SelfReferentialDepCheck`, `DependencyCycleCheck`, `ChildBlockedByParentCheck`, `ParentDoneOpenChildrenCheck`) as standalone units, but they are not yet registered with the `DiagnosticRunner`. The `tick doctor` command currently runs only the four checks from Phases 1 and 2 (cache staleness, JSONL syntax, ID format, duplicate ID). Until these six checks are registered, running `tick doctor` cannot detect orphaned references, self-referential dependencies, dependency cycles, child-blocked-by-parent deadlocks, or the warning condition of a parent marked done with open children. This task wires all six Phase 3 checks into the doctor command's runner alongside the existing four checks, so that all 10 checks execute in a single `tick doctor` invocation. This completes the full doctor check suite — the specification's complete set of 9 error checks and 1 warning check.

This is also the first time a warning-severity check is registered. Prior phases contained only error-severity checks, so the exit code was always 0 (all pass) or 1 (any error). Now the `ParentDoneOpenChildrenCheck` introduces the third scenario: warnings exist but no errors, and exit code must still be 0. This validates the warning/error severity distinction end-to-end through the runner, formatter, and exit code logic.

## Implementation

- Locate the doctor command handler (established in task 1-4, extended in task 2-4) where the `DiagnosticRunner` is created and checks are registered.
- After the existing four check registrations (CacheStalenessCheck, JsonlSyntaxCheck, IdFormatCheck, DuplicateIdCheck), add six additional `Register()` calls:
  1. `runner.Register(OrphanedParentCheck{...})` — orphaned parent references (task 3-1, Error #5)
  2. `runner.Register(OrphanedDependencyCheck{...})` — orphaned dependency references (task 3-2, Error #6)
  3. `runner.Register(SelfReferentialDepCheck{...})` — self-referential dependencies (task 3-3, Error #7)
  4. `runner.Register(DependencyCycleCheck{...})` — dependency cycles (task 3-4, Error #8)
  5. `runner.Register(ChildBlockedByParentCheck{...})` — child blocked by parent (task 3-5, Error #9)
  6. `runner.Register(ParentDoneOpenChildrenCheck{...})` — parent done with open children (task 3-6, Warning #1)
- Pass the same `.tick/` directory path to each new check that the existing checks already receive.
- The registration order determines output order (per task 1-1: "preserves registration order in results"). Register in the order that matches the specification's error numbering:
  1. `CacheStalenessCheck` (Error #1 — already registered)
  2. `JsonlSyntaxCheck` (Error #2 — already registered)
  3. `IdFormatCheck` (Error #4 — already registered)
  4. `DuplicateIdCheck` (Error #3 — already registered)
  5. `OrphanedParentCheck` (Error #5)
  6. `OrphanedDependencyCheck` (Error #6)
  7. `SelfReferentialDepCheck` (Error #7)
  8. `DependencyCycleCheck` (Error #8)
  9. `ChildBlockedByParentCheck` (Error #9)
  10. `ParentDoneOpenChildrenCheck` (Warning #1 — last, since warnings follow errors)
- No new types or interfaces are created in this task. The check structs and the runner already exist. This task is purely wiring — it modifies the doctor command handler to register the six additional checks.
- Verify that the runner still calls all checks regardless of individual check results (the "run all checks" invariant from task 1-1). This was already built into the `DiagnosticRunner`, but the end-to-end tests must confirm it still holds with all 10 checks.
- The exit code logic (task 1-2) already handles any number of results — errors from new checks contribute to `HasErrors()` and `ErrorCount()` exactly like errors from existing checks do. The critical new behavior is that `ParentDoneOpenChildrenCheck` produces `SeverityWarning` results. The `HasErrors()` method returns false when only warnings exist, so exit code remains 0 when the only failures are warnings. No changes needed to the exit code logic — just end-to-end verification.
- The formatter (task 1-2) already handles any number of `CheckResult` entries. Warning-severity results are displayed with `✗` markers just like errors (per task 1-2: "Warnings use the same `✗` marker as errors in output"). Warnings count in the summary issue total. No changes needed to formatting.
- The summary count includes both errors and warnings in the display total (per task 1-2: "Summary counts all failures (errors + warnings) for display"). So if there are 2 errors and 1 warning, the summary says "3 issues found." But exit code depends only on errors — `HasErrors()` checks error-severity only.

## Tests

- `"it registers all 10 checks (4 existing + 6 new relationship/hierarchy checks)"`
- `"it runs all 10 checks in a single tick doctor invocation"`
- `"it exits 0 when all 10 checks pass (healthy store)"`
- `"it exits 1 when only the orphaned parent check fails (other 9 pass)"`
- `"it exits 1 when only the orphaned dependency check fails (other 9 pass)"`
- `"it exits 1 when only the self-referential dependency check fails (other 9 pass)"`
- `"it exits 1 when only the dependency cycle check fails (other 9 pass)"`
- `"it exits 1 when only the child-blocked-by-parent check fails (other 9 pass)"`
- `"it exits 0 when only the parent-done-with-open-children warning fires (all 9 error checks pass)"`
- `"it exits 1 when both error checks and warning check produce failures"`
- `"it exits 1 when a Phase 1 error (cache stale) and a Phase 3 error (orphaned parent) both fire"`
- `"it exits 1 when a Phase 2 error (duplicate ID) and a Phase 3 error (dependency cycle) both fire"`
- `"it reports mixed results correctly — passing checks show checkmark, failing error checks show ✗ with details, warning check shows ✗ with details"`
- `"it displays results for all 10 checks in output (10 check labels visible when all pass)"`
- `"it shows correct summary count reflecting errors and warnings from all checks combined"`
- `"it shows summary count including warnings (e.g., 1 error + 1 warning = '2 issues found.')"`
- `"it runs all 10 checks even when early checks fail (no short-circuit)"`
- `"it handles empty tasks.jsonl — all 10 checks report their respective passing/failing results"`
- `"it does not modify tasks.jsonl or cache.db (read-only invariant preserved with 10 checks)"`
- `"it shows 'No issues found.' summary when all 10 checks pass"`

## Edge Cases

- **All 10 checks pass (healthy store)**: The fully healthy state with all checks registered. Output shows 10 `✓` lines — one for each check name (Cache, JSONL syntax, ID format, ID uniqueness, Orphaned parents, Orphaned dependencies, Self-referential dependencies, Dependency cycles, Child blocked by parent, Parent done with open children) — and the summary says "No issues found." Exit code 0. This is the primary happy path for the complete doctor suite and confirms that adding six checks did not break the existing passing behavior. It also validates that the warning check (`ParentDoneOpenChildrenCheck`) produces a passing `✓` line when there are no parent-done-with-open-children situations.

- **Mixed errors and warnings**: Some error checks fail and the warning check also fires. For example: orphaned parent check finds 1 orphaned reference (error), and parent-done-with-open-children check finds 1 suspicious parent (warning). Other 8 checks pass. Output shows 8 `✓` lines, 1 `✗` error line for orphaned parent, and 1 `✗` warning line for parent done with open children. Summary says "2 issues found." (both errors and warnings count in the display total). Exit code is 1 because at least one error-severity failure exists. This validates that warnings and errors coexist correctly in the output, that the summary counts both, and that exit code is driven solely by errors.

- **Warnings-only exit code 0**: The warning check fires but all 9 error checks pass. For example: a parent task is marked done while a child is still open — the `ParentDoneOpenChildrenCheck` produces a warning-severity result. All other checks return passing. Output shows 9 `✓` lines and 1 `✗` warning line. Summary says "1 issue found." Exit code is 0 because `HasErrors()` returns false — only warning-severity failures exist. This is the critical new behavior introduced by this task. Prior phases had no warnings, so exit code 0 always meant "nothing wrong." Now exit code 0 can mean "warnings only." This tests the full chain: check severity → `DiagnosticReport.HasErrors()` → `ExitCode()` → process exit.

- **Errors from relationship checks combine with earlier phase errors in summary count**: If a Phase 1 check fails (cache stale), a Phase 2 check fails (duplicate ID with 2 duplicates), and a Phase 3 check fails (3 orphaned parent references), plus the warning check fires (1 warning), the summary should count all failures: 1 (cache) + 2 (duplicates) + 3 (orphaned parents) + 1 (warning) = "7 issues found." Exit code 1 because errors exist. This validates that errors from checks across all three phases aggregate correctly in the final report — the runner treats all checks uniformly regardless of which phase they were introduced in. It also validates that multiple results from a single check (2 duplicates, 3 orphaned parents) each count individually in the summary, consistent with the specification's rule: "Doctor lists each error individually."

## Acceptance Criteria

- [ ] `OrphanedParentCheck` registered in the doctor command handler
- [ ] `OrphanedDependencyCheck` registered in the doctor command handler
- [ ] `SelfReferentialDepCheck` registered in the doctor command handler
- [ ] `DependencyCycleCheck` registered in the doctor command handler
- [ ] `ChildBlockedByParentCheck` registered in the doctor command handler
- [ ] `ParentDoneOpenChildrenCheck` registered in the doctor command handler
- [ ] All 10 checks (4 existing + 6 new) run in a single `tick doctor` invocation
- [ ] Each check receives the correct `.tick/` directory path
- [ ] All 10 checks run regardless of individual results (no short-circuit)
- [ ] Exit code 0 when all 10 checks pass
- [ ] Exit code 1 when any check produces an error-severity failure
- [ ] Exit code 0 when the only failures are warning-severity (warnings-only scenario)
- [ ] Summary count reflects total failures (errors + warnings) across all 10 checks
- [ ] Output shows results for all 10 checks (passing and failing)
- [ ] Warning-severity results display with `✗` marker same as errors
- [ ] Doctor remains read-only with 10 checks registered (no data modification)
- [ ] Tests written and passing for all edge cases (all pass, mixed errors/warnings, warnings-only, cross-phase error aggregation)

## Context

The specification requires all checks to run in a single invocation: design principle #4 states "Run all checks — Doctor completes all validations before reporting, never stops early." The `DiagnosticRunner.RunAll()` method (task 1-1) already enforces this — it iterates all registered checks without short-circuiting. This task increases the number of registered checks from 4 to 10, completing the full suite.

The specification defines exit codes: 0 = "All checks passed (no errors, warnings allowed)", 1 = "One or more errors found." The parenthetical "warnings allowed" is the key distinction tested by this task. The `ParentDoneOpenChildrenCheck` (task 3-6) is the only warning-severity check in the entire doctor suite. Its registration here enables the first end-to-end test of the warnings-don't-affect-exit-code behavior.

The specification's output format example shows multiple check labels in sequence (`✓ Cache: OK`, `✓ JSONL syntax: OK`, etc.), demonstrating that all checks appear in output. The formatter (task 1-2) already handles rendering any number of `CheckResult` entries with `✓`/`✗` markers — no formatter changes are needed. The formatter uses `✗` for both errors and warnings (severity distinction is in exit code, not display).

The summary count includes all failures — both errors and warnings — in the display total (per task 1-2). This means warnings are visible to the user even though they don't cause a non-zero exit code. The user sees "1 issue found." for a warnings-only run but the process exits 0.

The Phase 3 acceptance criteria explicitly require: "All 10 checks (9 errors + 1 warning) run in a single `tick doctor` invocation" and "Warnings do not affect exit code (exit 0 if only warnings, no errors)." This task is the verification point for those phase-level criteria — it is where all checks become part of the actual command and can be tested end-to-end.

This task follows the same pattern as task 2-4 (Data Integrity Check Registration): add checks to the runner and test the combined behavior. The only structural difference is the introduction of warning severity, which exercises the error/warning distinction that was built in task 1-1 (`SeverityError` vs `SeverityWarning`) and task 1-2 (`HasErrors()` excludes warnings, `ExitCode()` returns 0 for warnings-only) but never tested end-to-end until now.

This is a Go project. The doctor command handler (task 1-4, extended by task 2-4) is the only file that needs modification. The check types are imported from wherever tasks 3-1 through 3-6 placed them. The runner, formatter, and exit code logic are unchanged.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
