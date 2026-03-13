TASK: ID Format Check (doctor-validation-2-2)

ACCEPTANCE CRITERIA:
- [x] `IdFormatCheck` implements the `Check` interface
- [x] Passing check returns `CheckResult` with Name `"ID format"` and Passed `true`
- [x] Each invalid ID produces its own failing `CheckResult` with line number and actual value in details
- [x] Regex pattern `^tick-[0-9a-f]{6}$` used for validation (exact match, no normalization)
- [x] Empty ID field (`""`) detected as format violation
- [x] Missing `id` key detected as format violation
- [x] Uppercase hex chars detected as format violation (no lowercase normalization before check)
- [x] Extra chars beyond 6 hex detected as format violation
- [x] Wrong prefix detected as format violation
- [x] Numeric-only hex part (e.g., `tick-123456`) correctly accepted as valid
- [x] Unparseable JSON lines skipped silently (not duplicating syntax check)
- [x] Blank lines skipped silently
- [x] Empty file returns passing result
- [x] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [x] Suggestion is `"Manual fix required"` for format violations
- [x] All failures use `SeverityError`
- [x] Check is read-only -- never modifies `tasks.jsonl`
- [x] Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: Specification Error #4 defines "IDs not matching required format (prefix + 6 hex chars)." The tick-core spec mandates format `tick-{6 lowercase hex}`. IDs are normalized to lowercase at write time; doctor validates what is stored and catches any corruption. Fix suggestion for all non-cache errors is "Manual fix required." Each error reported individually.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/id_format.go:1-98
- Notes:
  - `IdFormatCheck` struct implements `Check` interface with correct `Run(ctx context.Context, tickDir string) []CheckResult` signature
  - Regex `^tick-[0-9a-f]{6}$` compiled at package level (line 10) -- correct pattern
  - Uses shared `getJSONLines()` to read/parse JSONL, delegates to `fileNotFoundResult()` helper for missing file case
  - Missing ID, non-string ID, and regex-failing ID each handled as separate branches with appropriate error details
  - `formatNonStringID()` helper (line 85-98) handles null and numeric values cleanly
  - Returns single passing result when no failures; returns only failures (no mixed pass/fail) -- matches spec
  - No normalization before validation -- uppercase stored IDs caught as errors
  - Read-only: only reads via `getJSONLines()`, never writes
  - Registered in `/Users/leeovery/Code/tick/internal/cli/doctor.go:22` as third check

TESTS:
- Status: Adequate
- Coverage: All 22 test cases from the plan are implemented in `/Users/leeovery/Code/tick/internal/doctor/id_format_test.go:1-452`
  - Happy path: all valid IDs, empty file, numeric-only hex part
  - Empty ID field with line number and value in details
  - Missing id key with line number
  - Uppercase hex chars (full uppercase and mixed case -- two separate tests)
  - Extra chars beyond 6 hex, fewer than 6 hex
  - Wrong prefix, missing prefix
  - Mixed valid/invalid with correct failure count
  - Failure count = one per invalid ID
  - Unparseable lines skipped silently
  - Blank lines skipped
  - Missing tasks.jsonl with correct details and suggestion
  - Actual invalid ID value shown in details
  - "Manual fix required" suggestion on all violations
  - Name "ID format" on all results (table-driven across pass/fail/missing)
  - SeverityError on all failures (table-driven across violation/missing-key/missing-file)
  - No lowercase normalization (explicit test)
  - Non-string id values (null, number) caught as violations with values shown
  - Read-only verification via `assertReadOnly` helper
- Notes: Tests use `t.Run` subtests as per project conventions. Table-driven tests used where appropriate (Name check, Severity check). Each test is focused on a single concern. No over-testing detected.

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run subtests, t.TempDir via setupTickDir, t.Helper on helpers, error wrapping with fmt.Errorf, no testify.
- SOLID principles: Good. Single responsibility (one check, one concern). Implements Check interface cleanly. Uses shared helpers (getJSONLines, fileNotFoundResult) via dependency inversion through context.
- Complexity: Low. Linear scan through lines with straightforward branching (missing -> non-string -> regex fail). No nested loops or complex conditionals.
- Modern idioms: Yes. Package-level compiled regex, type assertion for non-string values, context-based data sharing.
- Readability: Good. Clear doc comments on struct and Run method. Each branch in Run is well-commented. formatNonStringID is self-explanatory.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- No test verifies line numbering correctness when blank lines are interspersed (e.g., that a task on file line 4 after two blank lines is reported as "Line 4"). However, this is the responsibility of `ScanJSONLines` in the shared reader, not this check, so it is not a gap in this task's scope.
