---
id: doctor-validation-4-3
phase: 4
status: approved
created: 2026-02-13
---

# Extract fileNotFoundResult helper for repeated tasks.jsonl-not-found error

**Problem**: Nine check files construct a nearly identical `CheckResult` for the "tasks.jsonl not found" case: `Details: "tasks.jsonl not found"`, `Suggestion: "Run tick init or verify .tick directory"`, `Passed: false`, `Severity: SeverityError`. Only the `Name` field varies. This is 6-8 lines repeated 9 times. If the suggestion wording changes, all 9 files must be updated.

**Solution**: Extract a helper function `fileNotFoundResult(checkName string) []CheckResult` in the doctor package that returns the standard not-found error result. Each check calls this instead of constructing the literal.

**Outcome**: The "tasks.jsonl not found" error message and suggestion are defined in exactly one place. Adding or changing the wording requires a single edit.

**Do**:
1. Add a `fileNotFoundResult(checkName string) []CheckResult` function in `internal/doctor/helpers.go` (new file) that returns `[]CheckResult{{Name: checkName, Passed: false, Severity: SeverityError, Details: "tasks.jsonl not found", Suggestion: "Run tick init or verify .tick directory"}}`
2. Replace the file-not-found CheckResult literal in all 9 check files (jsonl_syntax.go, duplicate_id.go, id_format.go, orphaned_parent.go, orphaned_dependency.go, self_referential_dep.go, dependency_cycle.go, child_blocked_by_parent.go, parent_done_open_children.go) with a call to `fileNotFoundResult("CheckName")`
3. Note: CacheStalenessCheck has a slightly different message ("tasks.jsonl not found or unreadable") so it may not use this helper -- evaluate and keep its custom message if it includes the error detail
4. Run all tests

**Acceptance Criteria**:
- A single `fileNotFoundResult` function exists in the doctor package
- At least 8 of the 9 check files use this helper instead of constructing the literal inline
- The helper is unexported (lowercase) since it is internal to the doctor package
- All existing tests pass

**Tests**:
- All existing doctor tests pass
- Unit test for fileNotFoundResult verifying it returns the expected CheckResult with correct fields
