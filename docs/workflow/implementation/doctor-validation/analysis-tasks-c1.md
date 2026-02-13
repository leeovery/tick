---
topic: doctor-validation
cycle: 1
total_proposed: 3
---
# Analysis Tasks: Doctor Validation (Cycle 1)

## Task 1: Extract shared JSONL line iterator and parse tasks.jsonl once per doctor run
status: approved
severity: high
sources: duplication, architecture

**Problem**: Three line-level checks (JsonlSyntaxCheck, DuplicateIdCheck, IdFormatCheck) each independently implement the same ~20-line JSONL file-scanning loop: open tasks.jsonl, handle file-not-found, create bufio.Scanner, maintain lineNum counter, skip blank lines via strings.TrimSpace, and json.Unmarshal each line. Additionally, six relationship checks each independently call ParseTaskRelationships which re-opens, re-reads, and re-parses the entire tasks.jsonl file. The file is opened and fully parsed ~10 times per doctor run, wasting I/O and risking inconsistent snapshots if the file changes mid-run.

**Solution**: (1) Extract a shared JSONL line iterator (e.g., `ScanJSONLines(tickDir string) ([]JSONLine, error)` in a new `jsonl_reader.go`) that opens the file, scans lines, skips blanks, parses JSON, and returns (lineNum, raw line, parsed map) tuples. The three line-level checks use this instead of duplicating the scanning loop. (2) In `RunDoctor` (internal/cli/doctor.go), call `ParseTaskRelationships` once and store the result in the context (add a new context key `TaskRelationshipsKey` in doctor.go) so the six relationship checks pull pre-parsed data from context rather than re-parsing. Alternatively, add a `SetTaskRelationships` method or pass the data via a struct field on each relationship check.

**Outcome**: tasks.jsonl is opened and scanned at most twice per doctor run (once for line-level checks via the shared iterator, once for relationship data via ParseTaskRelationships). Line-level checks contain only their validation logic. Relationship checks no longer independently parse the file.

**Do**:
1. Create `internal/doctor/jsonl_reader.go` with a `JSONLine` struct (`LineNum int`, `Raw string`, `Parsed map[string]interface{}`) and a `ScanJSONLines(tickDir string) ([]JSONLine, error)` function that implements the shared open/scan/skip-blank/parse loop
2. Refactor `JsonlSyntaxCheck.Run` to call `ScanJSONLines` -- note: syntax check needs the raw line for `json.Valid` on unparsed lines, so `ScanJSONLines` should return raw lines even when JSON parsing fails (or provide a lower-level variant that yields raw lines and parse errors)
3. Refactor `DuplicateIdCheck.Run` to call `ScanJSONLines` and iterate the returned slice
4. Refactor `IdFormatCheck.Run` to call `ScanJSONLines` and iterate the returned slice
5. Add a `TaskRelationshipsKey` context key (similar pattern to `TickDirKey`) in `internal/doctor/doctor.go`
6. In `RunDoctor` (internal/cli/doctor.go), call `ParseTaskRelationships(tickDir)` once before `runner.RunAll`, store the result in the context via `context.WithValue`
7. Refactor all six relationship checks (orphaned_parent, orphaned_dependency, self_referential_dep, dependency_cycle, child_blocked_by_parent, parent_done_open_children) to extract task relationships from context instead of calling `ParseTaskRelationships` directly. Include a fallback: if the context key is missing, call `ParseTaskRelationships` directly (defensive coding)
8. Run all existing tests to ensure no regressions

**Acceptance Criteria**:
- tasks.jsonl is opened at most twice per full doctor run (once for line-level scanning, once for relationship parsing)
- No line-level check contains file-open, bufio.Scanner, or line-counting boilerplate
- No relationship check calls ParseTaskRelationships directly in its Run method (it reads from context)
- All existing doctor tests pass without modification (or with minimal test setup changes)
- JsonlSyntaxCheck still correctly reports line numbers for malformed JSON

**Tests**:
- Existing test suite passes (jsonl_syntax_test, duplicate_id_test, id_format_test, all relationship check tests)
- Verify ScanJSONLines correctly skips blank lines and maintains accurate line numbers
- Verify relationship checks work both with pre-parsed context data and with fallback (missing context key)

## Task 2: Make tickDir an explicit parameter on the Check interface
status: approved
severity: medium
sources: architecture

**Problem**: The tick directory path is passed to checks via `context.WithValue` using `TickDirKey`. Every check starts with `tickDir, _ := ctx.Value(TickDirKey).(string)` -- a runtime type assertion that silently returns empty string on failure. This is the "untyped parameters when concrete types are known" anti-pattern. CacheStalenessCheck guards against empty tickDir but 9 other checks do not, meaning they would construct paths like `filepath.Join("", "tasks.jsonl")` which resolves to a relative path. This inconsistency means checks fail with different error messages (or silently read wrong files) when the context key is missing.

**Solution**: Change the Check interface from `Run(ctx context.Context) []CheckResult` to `Run(ctx context.Context, tickDir string) []CheckResult`. Update DiagnosticRunner.RunAll to accept tickDir and pass it to each check. Remove the TickDirKey context value pattern. Remove the empty-tickDir guard from CacheStalenessCheck since the runner now provides a validated string. Keep the context parameter for future extensibility (cancellation, timeouts).

**Outcome**: The primary configuration dependency is explicit and type-safe. No check needs a runtime type assertion for tickDir. The inconsistent empty-tickDir guarding problem is eliminated structurally. If tickDir is ever empty, it fails at a single point (the runner) rather than inconsistently across 10 checks.

**Do**:
1. Change the `Check` interface in `internal/doctor/doctor.go` from `Run(ctx context.Context) []CheckResult` to `Run(ctx context.Context, tickDir string) []CheckResult`
2. Update `DiagnosticRunner.RunAll` signature to accept `tickDir string` and pass it to each `check.Run(ctx, tickDir)` call
3. Update all 10 check implementations to accept `tickDir string` as the second parameter instead of extracting it from context
4. Remove `tickDirKeyType`, `TickDirKey` from `internal/doctor/doctor.go` (unless still needed for TaskRelationshipsKey from Task 1 -- if so, keep the type but remove TickDirKey specifically)
5. Update `RunDoctor` in `internal/cli/doctor.go` to pass `tickDir` to `runner.RunAll(ctx, tickDir)` instead of embedding it in context
6. Remove the empty-tickDir guard from CacheStalenessCheck (lines 26-34) since the caller is now responsible
7. Remove all `tickDir, _ := ctx.Value(TickDirKey).(string)` lines from all 10 checks
8. Update all test files that create contexts with TickDirKey to instead pass tickDir directly
9. Run all tests

**Acceptance Criteria**:
- Check interface signature is `Run(ctx context.Context, tickDir string) []CheckResult`
- No check implementation contains `ctx.Value(TickDirKey)`
- TickDirKey is removed from the package (or clearly deprecated if Task 1 adds other context keys that reuse the key type)
- All existing tests pass
- DiagnosticRunner.RunAll accepts and forwards tickDir

**Tests**:
- All existing doctor and CLI doctor tests pass
- Verify that passing empty string to RunAll produces consistent error behavior across all checks

## Task 3: Extract fileNotFoundResult helper for repeated tasks.jsonl-not-found error
status: approved
severity: medium
sources: duplication

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
