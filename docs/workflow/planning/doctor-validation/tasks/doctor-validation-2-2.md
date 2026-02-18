---
id: doctor-validation-2-2
phase: 2
status: completed
created: 2026-01-31
---

# ID Format Check

## Goal

The doctor framework can detect malformed JSON and cache staleness, but it cannot detect tasks with invalid IDs. If a task has an ID that does not match the required format (`tick-{6 lowercase hex chars}`), it indicates data corruption — manual edits, import errors, or bugs in ID generation. Invalid IDs can cause lookup failures, duplicate-detection false negatives, and confusing UX. This task implements an `IdFormatCheck` that parses each task from `tasks.jsonl`, extracts the `id` field, and validates it against the required pattern. Each invalid ID is reported as an individual error with the offending value. This is specification Error #4: "IDs not matching required format (prefix + 6 hex chars)."

## Implementation

- Create an `IdFormatCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path (provided via context) to locate `tasks.jsonl`.
- Define the ID validation pattern as a compiled regular expression: `^tick-[0-9a-f]{6}$`. This enforces:
  - Prefix is exactly `tick-` (not `TICK-`, not `task-`, not `tck-`)
  - Exactly 6 characters after the dash
  - Characters are lowercase hexadecimal only (0-9, a-f)
  - No extra characters before or after
- Implement the `Run` method with the following logic:
  1. Attempt to open `.tick/tasks.jsonl`. If the file does not exist, return a single failing `CheckResult` with Name `"ID format"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`.
  2. Read the file line by line. For each line:
     - **Skip blank lines**: If the line is empty or contains only whitespace after trimming, skip it silently (same convention as the JSONL syntax check, task 2-1).
     - **Attempt JSON parse**: Parse the line into a generic structure (e.g., `map[string]interface{}`). If parsing fails, skip the line silently — the JSONL syntax check (task 2-1) is responsible for reporting parse errors. The ID format check does not duplicate syntax error reporting.
     - **Extract the `id` field**: Look for a key named `id` in the parsed JSON object.
     - **Missing `id` field**: If the parsed object has no `id` key at all, record a failing `CheckResult` with Name `"ID format"`, Severity `SeverityError`, Details identifying the line number (e.g., `"Line 5: missing id field"`), and Suggestion `"Manual fix required"`.
     - **Empty `id` field**: If the `id` key exists but its value is an empty string (`""`), record a failing `CheckResult` with Name `"ID format"`, Severity `SeverityError`, Details identifying the line number and the empty value (e.g., `"Line 3: invalid ID '' — expected format tick-{6 hex}"`), and Suggestion `"Manual fix required"`.
     - **Non-string `id` field**: If the `id` value is not a string (e.g., a number or null), treat it like an invalid ID. Record a failing result with the actual value shown in details.
     - **Validate against pattern**: Apply the regex `^tick-[0-9a-f]{6}$` to the string value. If it does not match, record a failing `CheckResult` with Name `"ID format"`, Severity `SeverityError`, Details including line number and the actual ID value (e.g., `"Line 7: invalid ID 'TICK-A1B2C3' — expected format tick-{6 hex}"`), and Suggestion `"Manual fix required"`.
     - **If it matches**: The ID is valid. Continue to the next line.
  3. After processing all lines, if no format errors were found, return a single passing `CheckResult` with Name `"ID format"` and Passed `true`.
  4. If format errors were found, return all the failing `CheckResult` entries (one per invalid ID). Do not include a passing result alongside failures.
- The check does NOT normalize IDs to lowercase before validation. The purpose of this check is to detect IDs that are stored incorrectly. IDs should already be lowercase in `tasks.jsonl` — the normalization described in tick-core ("normalize to lowercase on input") happens at write time. If an ID is stored as uppercase, that is a data integrity issue this check should catch.
- Line numbering is 1-based (matching the convention from task 2-1). Blank lines count toward line numbers but are not checked.
- The check must be read-only — it opens `tasks.jsonl` for reading only and never modifies it.
- An empty file (zero bytes) is a valid state — no tasks means no IDs to validate. Returns a single passing result.

## Tests

- `"it returns passing result when all IDs match tick-{6 hex} format"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns failing result for empty ID field with line number in details"`
- `"it returns failing result when id field is missing from JSON object"`
- `"it returns failing result for uppercase hex chars (e.g., tick-A1B2C3)"`
- `"it returns failing result for mixed-case hex chars (e.g., tick-a1B2c3)"`
- `"it returns failing result for extra chars beyond 6 hex (e.g., tick-a1b2c3d4)"`
- `"it returns failing result for fewer than 6 hex chars (e.g., tick-a1b)"`
- `"it returns failing result for wrong prefix (e.g., task-a1b2c3)"`
- `"it returns failing result for missing prefix (e.g., a1b2c3)"`
- `"it returns passing result for numeric-only random part (tick-123456) since 0-9 are valid hex chars"`
- `"it returns failing results for each invalid ID when mixed valid and invalid IDs present"`
- `"it reports correct count of failures — one per invalid ID, not one per check"`
- `"it skips unparseable lines silently (syntax check handles those)"`
- `"it skips blank lines without error"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it shows actual invalid ID value in error details"`
- `"it suggests 'Manual fix required' for all format violations"`
- `"it uses CheckResult Name 'ID format' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it does not normalize IDs to lowercase — uppercase stored IDs are caught as errors"`
- `"it handles non-string id values (null, number) as format violations"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Empty ID field**: A task line like `{"id": "", "title": "test"}` has an `id` key with an empty string value. This is an ID format violation — the empty string does not match `tick-{6 hex}`. Report with the line number and show the empty value in details so the user knows the field exists but is blank.
- **Missing ID field**: A task line like `{"title": "test"}` has no `id` key at all. This is a format violation — every task must have an ID. Report with the line number and indicate the field is absent. This is distinct from an empty ID: missing means the key is not present; empty means the key exists with value `""`.
- **Uppercase hex chars**: An ID like `tick-A1B2C3` or `tick-a1B2c3` uses uppercase hexadecimal characters. The tick-core specification says IDs are normalized to lowercase on input and always stored/output as lowercase. An uppercase ID in `tasks.jsonl` indicates the normalization was bypassed (manual edit, import bug). The check must NOT normalize before validating — it must catch the stored uppercase as an error.
- **Extra chars beyond 6 hex**: An ID like `tick-a1b2c3d4` has 8 hex characters instead of 6. The format requires exactly 6. Any additional characters make the ID invalid regardless of their content.
- **Wrong prefix**: An ID like `task-a1b2c3`, `tck-a1b2c3`, or `TICK-a1b2c3` uses an incorrect prefix. The specification hardcodes the prefix as `tick` (lowercase). Any deviation is a format error.
- **Numeric-only random part**: An ID like `tick-123456` uses only digits (0-9) in the random part. This is actually valid — digits 0-9 are valid hexadecimal characters. The check should pass this case. This is listed as an edge case to ensure the regex is not accidentally restricted to require at least one letter.
- **Mixed valid and invalid IDs**: A file with some valid IDs and some invalid IDs. Only the invalid IDs produce failing results. Valid IDs are silently accepted. The number of failing results equals the number of invalid IDs.
- **Unparseable lines**: Lines that fail JSON parsing are skipped silently by the ID format check. The JSONL syntax check (task 2-1) is responsible for reporting those. The ID format check only operates on successfully parsed JSON objects. This prevents duplicate error reporting between the two checks.
- **Non-string ID values**: A task line like `{"id": 12345}` or `{"id": null}` has an `id` field that is not a string. These are treated as format violations since the ID cannot match the expected pattern.

## Acceptance Criteria

- [ ] `IdFormatCheck` implements the `Check` interface
- [ ] Passing check returns `CheckResult` with Name `"ID format"` and Passed `true`
- [ ] Each invalid ID produces its own failing `CheckResult` with line number and actual value in details
- [ ] Regex pattern `^tick-[0-9a-f]{6}$` used for validation (exact match, no normalization)
- [ ] Empty ID field (`""`) detected as format violation
- [ ] Missing `id` key detected as format violation
- [ ] Uppercase hex chars detected as format violation (no lowercase normalization before check)
- [ ] Extra chars beyond 6 hex detected as format violation
- [ ] Wrong prefix detected as format violation
- [ ] Numeric-only hex part (e.g., `tick-123456`) correctly accepted as valid
- [ ] Unparseable JSON lines skipped silently (not duplicating syntax check)
- [ ] Blank lines skipped silently
- [ ] Empty file returns passing result
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Suggestion is `"Manual fix required"` for format violations
- [ ] All failures use `SeverityError`
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] Tests written and passing for all edge cases

## Context

The specification defines ID format violations as Error #4: "IDs not matching required format (prefix + 6 hex chars)." The tick-core specification (section "ID Generation > Format") defines the exact format: prefix `tick` (hardcoded, no configuration) + dash + 6 lowercase hexadecimal characters. Example: `tick-a3f2b7`.

The tick-core specification states: "IDs are case-insensitive for matching. Normalize to lowercase on input (user can type `TICK-A3F2B7`, stored as `tick-a3f2b7`). Always output lowercase." This normalization happens at write time. Doctor validates what is stored — if uppercase characters appear in `tasks.jsonl`, that is corruption that doctor should catch.

The doctor specification states that "Schema validation (field types, required fields, valid enum values) happens at write time, not in doctor. Doctor catches corruption and edge cases that slipped through." The ID format check is one such corruption detector — it catches IDs that bypassed or pre-date the write-time validation.

The specification's fix suggestion table maps "All other errors" (including ID format) to "Manual fix required." Each error is reported individually per the spec: "Doctor lists each error individually."

The specification also lists duplicate ID detection as a separate check (Error #3, task 2-3) with "Case-insensitive duplicate detection (tick-ABC123 = tick-abc123)." The ID format check and duplicate ID check are independent — a task could have a validly formatted ID that is duplicated, or an invalidly formatted ID that is unique. Both checks run independently.

This is a Go project. Use `regexp` from stdlib for pattern matching. Use `encoding/json` for JSON parsing and `bufio.Scanner` or equivalent for line-by-line reading. The check implements the `Check` interface defined in task 1-1 and will be registered with the `DiagnosticRunner` in task 2-4.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
