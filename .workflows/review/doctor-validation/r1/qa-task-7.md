TASK: Duplicate ID Check (doctor-validation-2-3)

ACCEPTANCE CRITERIA:
- DuplicateIdCheck implements the Check interface
- Passing check returns CheckResult with Name "ID uniqueness" and Passed true
- Duplicate detection is case-insensitive (IDs normalized to lowercase before comparison)
- Each distinct duplicate group produces its own failing CheckResult
- Details include line numbers and original-case ID forms for each occurrence in the group
- Groups of more than two are reported as a single result listing all occurrences
- Lines with invalid JSON are skipped (not re-reported as syntax errors)
- Lines with missing or empty id field are skipped (not re-reported as format errors)
- Blank and whitespace-only lines are silently skipped
- Line numbers are 1-based and count blank lines
- Empty file (zero bytes) returns passing result
- Single-task file returns passing result
- Missing tasks.jsonl returns error-severity failure with init suggestion
- Suggestion is "Manual fix required" for duplicate errors
- All failures use SeverityError
- Check is read-only -- never modifies tasks.jsonl
- Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: The specification defines duplicate IDs as Error #3: "Case-insensitive duplicate detection (tick-ABC123 = tick-abc123)." The output format shows "ID uniqueness: OK" for the passing case. The fix suggestion table maps "All other errors" to "Manual fix required." Doctor lists each error individually -- each duplicate group is a separate result.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/duplicate_id.go:1-92
- Notes: Implementation is clean and fully matches all acceptance criteria. The struct implements the Check interface via `Run(ctx context.Context, tickDir string) []CheckResult`. Uses `getJSONLines` (shared JSONL reader from jsonl_reader.go) and `fileNotFoundResult` (shared helper from helpers.go) for consistency with other checks. The `idOccurrence` struct at line 10-13 cleanly captures original-case ID and line number. Case-insensitive normalization uses `strings.ToLower` at line 51. A `keyOrder` slice at line 34 provides deterministic output ordering (goes beyond spec requirement of "does not need to be deterministic" but is a sensible enhancement). Each duplicate group produces its own failing CheckResult (loop at lines 62-82). Passing result returned only when no failures found (lines 84-91). Registered with DiagnosticRunner in /Users/leeovery/Code/tick/internal/cli/doctor.go:22.

TESTS:
- Status: Adequate
- Coverage: All 18 tests from the plan are implemented in /Users/leeovery/Code/tick/internal/doctor/duplicate_id_test.go:1-359. Tests cover:
  - Happy path: no duplicates, single task, empty file
  - Duplicate detection: exact-case, mixed-case, triple duplicates, multiple distinct groups
  - Detail verification: line numbers, original-case forms
  - Skip behavior: blank lines, invalid JSON, missing/empty id
  - Error conditions: missing tasks.jsonl
  - Field verification: Name="ID uniqueness", SeverityError, Suggestion="Manual fix required"
  - Safety: read-only verification via assertReadOnly helper
  - Line numbering: 1-based with blank lines counting toward numbering
- Notes: Tests are well-structured using t.Run subtests. The Name and Severity tests use table-driven pattern, consistent with Go conventions. No over-testing observed -- each test verifies a distinct behavior. Tests would fail if the feature broke (e.g., removing ToLower breaks the mixed-case test, removing line tracking breaks the line number tests). The assertReadOnly helper at line 351-358 provides a solid read-only guarantee.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only (no testify), t.Run subtests, t.TempDir for isolation, t.Helper on helpers, error wrapping with fmt.Errorf. Handler pattern follows the Check interface from task 1-1.
- SOLID principles: Good. Single responsibility (only duplicate detection, no format or syntax checking). Open/closed (implements Check interface, pluggable into runner). Dependency inversion (uses getJSONLines abstraction rather than direct file access).
- Complexity: Low. Single method, linear scan with map-based grouping. Cyclomatic complexity is minimal -- one loop for building groups, one for reporting.
- Modern idioms: Yes. Proper use of context for pre-scanned line injection. Clean struct types. Idiomatic Go map and slice patterns.
- Readability: Good. Well-documented type and method comments. Clear variable names (groups, keyOrder, occurrences, parts). The format string at line 73 is readable and produces helpful output like "Duplicate ID tick-abc123: tick-ABC123 (line 2), tick-abc123 (line 5)".
- Issues: None

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The `keyOrder` slice for deterministic output ordering is a nice touch that goes beyond spec requirements and makes testing more predictable, though the plan explicitly says "iteration order does not need to be deterministic."
- The `idOccurrence` type is unexported and file-scoped, which is appropriate given it is only used within this check.
