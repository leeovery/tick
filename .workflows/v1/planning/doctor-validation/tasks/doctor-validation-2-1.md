---
id: doctor-validation-2-1
phase: 2
status: completed
created: 2026-01-30
---

# JSONL Syntax Check

## Goal

The doctor framework (Phase 1) can run checks and format results, but it has no check that validates the structural integrity of `tasks.jsonl` itself. If `tasks.jsonl` contains malformed JSON lines — due to manual edits, merge conflicts, partial writes, or file corruption — every downstream check (ID format, duplicate detection, relationship validation) operates on incomplete or misleading data. This task implements a `JsonlSyntaxCheck` that reads `tasks.jsonl` line by line, attempts to parse each line as JSON, and reports each unparseable line as an individual error with its line number and content. This is specification Error #2: "Malformed JSON lines that can't be parsed."

## Implementation

- Create a `JsonlSyntaxCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path (provided via context) to locate `tasks.jsonl`.
- Implement the `Run` method with the following logic:
  1. Attempt to open `.tick/tasks.jsonl`. If the file does not exist, return a single failing `CheckResult` with Name `"JSONL syntax"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. A missing JSONL file means the data store is broken — no syntax to validate.
  2. Read the file line by line. For each line:
     - **Skip blank lines**: If the line is empty or contains only whitespace after trimming, skip it silently. Blank lines are not syntax errors — they occur naturally from trailing newlines and are harmless. Do not count them as valid or invalid; simply ignore them.
     - **Attempt JSON parse**: Use `json.Unmarshal` (or equivalent) to parse the line into a generic structure (e.g., `map[string]interface{}` or `json.RawMessage` validity check). The check validates syntax only — it does not inspect field names, types, or values.
     - **If parse succeeds**: The line is syntactically valid. Continue to the next line.
     - **If parse fails**: Record a failing `CheckResult` with Name `"JSONL syntax"`, Severity `SeverityError`, Details including the line number and a truncated preview of the malformed content (e.g., `"Line 7: invalid JSON — {truncated content}"`), and Suggestion `"Manual fix required"`.
  3. After processing all lines, if no syntax errors were found, return a single passing `CheckResult` with Name `"JSONL syntax"` and Passed `true`.
  4. If syntax errors were found, return all the failing `CheckResult` entries (one per malformed line). Do not include a passing result alongside failures.
- Line numbering is 1-based (the first line of the file is line 1) for human readability.
- The line content preview in the Details field should be truncated to a reasonable length (e.g., 80 characters) to prevent excessively long output from a severely corrupted line. Append `"..."` when truncated.
- The check must be read-only — it opens `tasks.jsonl` for reading only and never modifies it.
- An empty file (zero bytes) is a valid state — it contains no lines to parse, so the check returns a single passing result. This is consistent with an initialized project that has no tasks yet.

## Tests

- `"it returns passing result when all lines are valid JSON"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns passing result when file contains only blank lines"`
- `"it returns passing result when file contains only whitespace-only lines"`
- `"it returns failing result for a single malformed line with line number in details"`
- `"it returns one failing result per malformed line when all lines are malformed"`
- `"it returns failing results only for malformed lines when mixed with valid lines"`
- `"it skips blank lines without counting them as valid or invalid"`
- `"it skips trailing newline that produces empty last line"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it suggests 'Manual fix required' for syntax errors"`
- `"it suggests 'Run tick init or verify .tick directory' when file is missing"`
- `"it uses CheckResult Name 'JSONL syntax' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it reports correct 1-based line numbers (skipped blank lines still count in numbering)"`
- `"it truncates long malformed line content in details"`
- `"it does not validate JSON field names or values — only syntax"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Empty file (zero bytes)**: Valid state. No lines to parse, no errors to report. Returns a single passing `CheckResult`. This represents an initialized project with no tasks — `tick init` creates an empty `tasks.jsonl`. The check should not confuse "no lines" with "missing file."
- **Blank/whitespace-only lines**: Lines that are empty or contain only whitespace (spaces, tabs) are silently skipped. They are not syntax errors and are not counted as valid JSON lines. This handles: trailing newlines at end of file, accidental double newlines between entries, lines with only spaces/tabs from careless editing. Important: blank lines still occupy line numbers — if line 3 is blank and line 4 is malformed, the error reports "Line 4", not "Line 3."
- **All lines valid**: The happy path. Every non-blank line parses as valid JSON. Returns a single passing result. This confirms the check doesn't produce false positives.
- **All lines malformed**: Every non-blank line fails to parse. Returns one failing `CheckResult` per malformed line, each with its line number. This tests that the check doesn't stop after the first error — it processes the entire file (consistent with doctor's "run all checks" principle applied within a single check).
- **Single malformed line among many valid**: The common real-world case — one bad line from a merge conflict or partial edit. Returns exactly one failing result identifying the specific line. All valid lines are silently accepted.
- **Trailing newline producing empty last line**: POSIX text files end with a newline, which means the last "line" when splitting is empty. The blank-line skipping logic handles this naturally. The check should not report an error for a trailing newline. This is perhaps the most common edge case in practice.
- **Missing `tasks.jsonl`**: Report as error with a suggestion to initialize. This is the same pattern as the cache staleness check (task 1-3) for missing `tasks.jsonl`. The file might be missing due to accidental deletion or running doctor outside a tick project directory (though the `.tick` directory itself must exist, since the command wiring in task 1-4 checks for that).
- **Line with only `{}` or `[]`**: These are syntactically valid JSON. `{}` is a valid JSON object (empty object), and `[]` is a valid JSON array. The syntax check accepts them — field validation is handled by other checks (ID format, etc.) or at write time per the specification ("Schema validation happens at write time, not in doctor").

## Acceptance Criteria

- [ ] `JsonlSyntaxCheck` implements the `Check` interface
- [ ] Passing check returns `CheckResult` with Name `"JSONL syntax"` and Passed `true`
- [ ] Each malformed line produces its own failing `CheckResult` with line number in details
- [ ] Blank and whitespace-only lines are silently skipped (not errors, not counted as valid)
- [ ] Line numbers are 1-based and count blank lines (blank line on line 3 means next content is line 4)
- [ ] Empty file (zero bytes) returns passing result
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Only JSON syntax is validated — field names, types, and values are not inspected
- [ ] Long malformed lines are truncated in details output
- [ ] Suggestion is `"Manual fix required"` for syntax errors
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] All failures use `SeverityError`
- [ ] Tests written and passing for all edge cases (empty file, blank lines, all valid, all malformed, single malformed among valid, trailing newline, missing file)

## Context

The specification defines JSONL syntax errors as Error #2: "Malformed JSON lines that can't be parsed." The specification also states that "Schema validation (field types, required fields, valid enum values) happens at write time, not in doctor. Doctor catches corruption and edge cases that slipped through." This means the syntax check validates only that each line is parseable JSON — it does not check whether the JSON contains valid task fields.

The JSONL format from the tick-core specification is "one JSON object per line, no trailing commas, no array wrapper." Each line is independently parseable. The check should attempt to parse each line in isolation — it does not need to consider cross-line context.

The specification's output format example shows `✓ JSONL syntax: OK` for the passing case and the fix suggestion table maps "All other errors" to "Manual fix required" — so syntax errors get the generic manual fix suggestion.

The specification requires doctor to report each error individually: "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details." This same principle applies to syntax errors — if 3 lines are malformed, 3 separate `CheckResult` entries are returned, each identifying its line number.

The Phase 2 goal states these checks "operate on raw file content without relationship semantics." The syntax check is the most fundamental of these — it validates that the data can even be parsed before other checks attempt to interpret it.

This is a Go project. Use `encoding/json` from stdlib for JSON parsing. Use `bufio.Scanner` or equivalent for line-by-line file reading. The check implements the `Check` interface defined in task 1-1 and will be registered with the `DiagnosticRunner` in task 2-4.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
