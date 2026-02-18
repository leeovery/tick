---
id: doctor-validation-2-4
phase: 2
status: completed
created: 2026-01-31
---

# Data Integrity Check Registration

## Goal

Tasks 2-1, 2-2, and 2-3 implemented three new diagnostic checks (`JsonlSyntaxCheck`, `IdFormatCheck`, `DuplicateIdCheck`) as standalone units, but they are not yet registered with the `DiagnosticRunner`. The `tick doctor` command (task 1-4) currently only runs the `CacheStalenessCheck` from Phase 1. Until these three checks are registered, running `tick doctor` cannot detect JSONL syntax errors, ID format violations, or duplicate IDs -- half the data integrity story is invisible to the user. This task wires the three new checks into the doctor command's runner alongside the existing cache staleness check, so that all four checks execute in a single `tick doctor` invocation. This completes Phase 2's goal: "Validate JSONL file integrity -- syntax, ID uniqueness, and ID format."

## Implementation

- Locate the doctor command handler (established in task 1-4) where the `DiagnosticRunner` is created and `CacheStalenessCheck` is registered.
- After the existing `CacheStalenessCheck` registration, add three additional `Register()` calls:
  1. `runner.Register(JsonlSyntaxCheck{...})` -- JSONL syntax check (task 2-1)
  2. `runner.Register(IdFormatCheck{...})` -- ID format check (task 2-2)
  3. `runner.Register(DuplicateIdCheck{...})` -- Duplicate ID check (task 2-3)
- Pass the same `.tick/` directory path to each new check that the `CacheStalenessCheck` already receives.
- The registration order determines output order (per task 1-1: "preserves registration order in results"). Register in the order that matches the specification's output example and error numbering:
  1. `CacheStalenessCheck` (Error #1 -- already registered)
  2. `JsonlSyntaxCheck` (Error #2)
  3. `IdFormatCheck` (Error #4 -- ID format before duplicate, since format validation is logically prior)
  4. `DuplicateIdCheck` (Error #3)

  Note: The exact registration order is a presentation choice. The key requirement is that all four checks are registered and all run. Follow the order in the specification's output example (`Cache`, `JSONL syntax`, then ID checks) if it provides guidance, otherwise use the error numbering order. The important invariant is consistency -- pick an order and verify it in tests.
- No new types or interfaces are created in this task. The check structs and the runner already exist. This task is purely wiring -- it modifies the doctor command handler to register additional checks.
- Verify that the runner still calls all checks regardless of individual check results (the "run all checks" invariant from task 1-1). This was already built into the `DiagnosticRunner`, but the end-to-end test must confirm it still holds with four checks.
- The exit code logic (task 1-2) already handles any number of results -- errors from new checks contribute to `HasErrors()` and `ErrorCount()` exactly like cache staleness errors do. No changes needed to the exit code logic.
- The formatter (task 1-2) already handles any number of `CheckResult` entries. No changes needed to formatting.

## Tests

- `"it registers all four checks (cache staleness, JSONL syntax, ID format, duplicate ID)"`
- `"it runs all four checks in a single tick doctor invocation"`
- `"it exits 0 when all four checks pass"`
- `"it exits 1 when only the JSONL syntax check fails (other three pass)"`
- `"it exits 1 when only the ID format check fails (other three pass)"`
- `"it exits 1 when only the duplicate ID check fails (other three pass)"`
- `"it exits 1 when all three new checks fail but cache check passes"`
- `"it exits 1 when all four checks fail"`
- `"it exits 1 when cache check fails but all three new checks pass"`
- `"it reports mixed results correctly -- passing checks show checkmark, failing checks show details"`
- `"it displays results for all four checks in output (four check labels visible)"`
- `"it shows correct summary count reflecting errors from all checks combined"`
- `"it runs all four checks even when the first check fails (no short-circuit)"`
- `"it handles empty tasks.jsonl -- cache check and data checks all report their respective results"`
- `"it does not modify tasks.jsonl or cache.db (read-only invariant preserved with four checks)"`

## Edge Cases

- **All new checks pass alongside passing cache check**: The fully healthy state with four checks registered. Output shows four `✓` lines (Cache, JSONL syntax, ID format, ID uniqueness) and the summary says "No issues found." Exit code 0. This is the primary happy path for Phase 2 and confirms that adding three checks did not break the existing passing behavior.
- **All new checks fail alongside passing cache check**: Cache is healthy but `tasks.jsonl` contains malformed lines, invalid IDs, and duplicates. The cache check shows `✓`, while the three data integrity checks each show `✗` with their specific errors. The summary counts all errors from all three failing checks. Exit code 1. This tests that cache passing does not mask data integrity failures, and that multiple failing checks each contribute their errors independently.
- **Mixed results across all four checks**: Some checks pass, some fail, and failing checks may produce multiple error results each. For example: cache passes, JSONL syntax passes, but ID format finds 2 invalid IDs and duplicate check finds 1 duplicate group. Output shows two `✓` lines and three `✗` lines (2 from ID format + 1 from duplicates). Summary shows "3 issues found." Exit code 1. This is the most realistic production scenario -- partial corruption. It validates that the runner collects results from all checks, the formatter renders them all, and the exit code reflects the aggregate.
- **Empty `tasks.jsonl`**: The file exists but has zero bytes. The cache staleness check compares hashes (empty JSONL has a deterministic hash). The JSONL syntax check returns passing (no lines to parse). The ID format check returns passing (no IDs to validate). The duplicate ID check returns passing (no IDs to compare). If the cache hash matches, all four checks pass. This edge case validates that the checks are compatible when operating on empty input -- no check produces spurious errors for the empty-file case, and their results combine cleanly in the runner.

## Acceptance Criteria

- [ ] `JsonlSyntaxCheck` registered in the doctor command handler
- [ ] `IdFormatCheck` registered in the doctor command handler
- [ ] `DuplicateIdCheck` registered in the doctor command handler
- [ ] All four checks (including existing `CacheStalenessCheck`) run in a single `tick doctor` invocation
- [ ] Each check receives the correct `.tick/` directory path
- [ ] All four checks run regardless of individual results (no short-circuit)
- [ ] Exit code 0 when all four checks pass
- [ ] Exit code 1 when any check produces an error-severity failure
- [ ] Summary count reflects total errors across all four checks
- [ ] Output shows results for all four checks (passing and failing)
- [ ] Doctor remains read-only with four checks registered (no data modification)
- [ ] Tests written and passing for all edge cases (all pass, all new fail, mixed, empty file)

## Context

The specification requires all checks to run in a single invocation: design principle #4 states "Run all checks -- Doctor completes all validations before reporting, never stops early." The `DiagnosticRunner.RunAll()` method (task 1-1) already enforces this -- it iterates all registered checks without short-circuiting. This task simply increases the number of registered checks from 1 to 4.

The specification's output format example shows multiple check labels in sequence (`✓ Cache: OK`, `✓ JSONL syntax: OK`, `✓ ID uniqueness: OK`), demonstrating that all checks appear in the output. The formatter (task 1-2) already handles rendering any number of `CheckResult` entries -- no formatter changes are needed.

Exit code logic (task 1-2): exit 0 when no errors (warnings allowed), exit 1 when any error. The `HasErrors()` method on `DiagnosticReport` already aggregates across all results regardless of which check produced them. Adding more checks does not change the exit code logic -- it just increases the number of results that feed into `HasErrors()`.

The Phase 2 acceptance criteria require: "All checks run even if earlier checks find errors" and "Errors from all checks contribute to summary count and exit code." This task is the verification point for those phase-level criteria -- it is where the checks become part of the actual command and can be tested end-to-end.

This is a Go project. The doctor command handler (task 1-4) is the only file that needs modification. The check types are imported from wherever tasks 2-1, 2-2, and 2-3 placed them. The runner, formatter, and exit code logic are unchanged.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
