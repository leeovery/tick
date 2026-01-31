---
id: doctor-validation-2-3
phase: 2
status: pending
created: 2026-01-31
---

# Duplicate ID Check

## Goal

The doctor framework can run checks and the JSONL syntax check (task 2-1) validates that lines are parseable JSON, but nothing yet detects duplicate task IDs in `tasks.jsonl`. Duplicate IDs cause ambiguous lookups -- when two tasks share the same ID, commands like `tick start tick-a1b2c3` cannot determine which task to act on, leading to silent data corruption or unpredictable behavior. The specification defines this as Error #3: "Case-insensitive duplicate detection (tick-ABC123 = tick-abc123)." IDs are normalized to lowercase before comparison, so `tick-ABC123` and `tick-abc123` are considered the same ID. This task implements a `DuplicateIdCheck` that reads all task IDs from `tasks.jsonl`, groups them by their lowercase-normalized form, and reports each group that contains more than one entry as an individual error.

## Implementation

- Create a `DuplicateIdCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path (provided via context) to locate `tasks.jsonl`.
- Implement the `Run` method with the following logic:
  1. Attempt to open `.tick/tasks.jsonl`. If the file does not exist, return a single failing `CheckResult` with Name `"ID uniqueness"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. This is consistent with the pattern established in task 2-1 for missing files.
  2. Read the file line by line. For each line:
     - **Skip blank lines**: If the line is empty or contains only whitespace after trimming, skip it. Consistent with the blank-line handling established in task 2-1.
     - **Attempt JSON parse**: Parse the line into a structure sufficient to extract the `id` field. If the line is not valid JSON, skip it silently -- syntax errors are the responsibility of `JsonlSyntaxCheck` (task 2-1), not this check. This avoids duplicate error reporting between checks.
     - **Extract the `id` field**: Read the `id` field from the parsed JSON object. If the `id` field is missing or empty, skip the line -- ID format validation is the responsibility of `IdFormatCheck` (task 2-2). This check only concerns itself with duplicate detection among IDs that exist.
  3. Build a map of normalized (lowercase) IDs to a list of their original-case occurrences and line numbers. For example, if line 2 has `tick-ABC123` and line 5 has `tick-abc123`, the map entry for `tick-abc123` contains both entries.
  4. After processing all lines, iterate the map. For each normalized ID that has more than one occurrence:
     - Produce a failing `CheckResult` with Name `"ID uniqueness"`, Severity `SeverityError`, Details describing the duplicate group (e.g., `"Duplicate ID tick-abc123: found on lines 2, 5"` or `"Duplicate ID tick-abc123: tick-ABC123 (line 2), tick-abc123 (line 5)"` -- include original-case forms to help the user identify which entries need fixing), and Suggestion `"Manual fix required"`.
  5. If no duplicate groups were found, return a single passing `CheckResult` with Name `"ID uniqueness"` and Passed `true`.
  6. If duplicate groups were found, return one failing `CheckResult` per duplicate group. Each group is a separate error. Do not include a passing result alongside failures.
- The iteration order of duplicate groups in the output does not need to be deterministic, but each group must be reported exactly once.
- Line numbering is 1-based and counts blank lines (consistent with task 2-1).
- The check must be read-only -- it never modifies `tasks.jsonl`.

## Tests

- `"it returns passing result when no duplicate IDs exist"`
- `"it returns passing result for a single task (cannot have duplicates)"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it detects exact-case duplicates (tick-abc123 appears twice)"`
- `"it detects mixed-case duplicates (tick-ABC123 and tick-abc123)"`
- `"it reports more than two duplicates of the same ID in a single result"`
- `"it reports multiple distinct duplicate groups as separate results"`
- `"it includes line numbers in the details for each duplicate occurrence"`
- `"it includes original-case ID forms in the details"`
- `"it skips blank and whitespace-only lines without counting them as entries"`
- `"it skips lines with invalid JSON silently (syntax errors are task 2-1's responsibility)"`
- `"it skips lines with missing or empty id field (format errors are task 2-2's responsibility)"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it suggests 'Manual fix required' for duplicate ID errors"`
- `"it uses CheckResult Name 'ID uniqueness' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it reports correct 1-based line numbers (blank lines still count in numbering)"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Exact-case duplicates**: Two or more lines with identically-cased IDs (e.g., `tick-abc123` on lines 3 and 7). The simplest duplicate scenario. The normalized map groups them together and reports them.
- **Mixed-case duplicates (tick-ABC123 vs tick-abc123)**: IDs that differ only in case. The specification explicitly requires case-insensitive detection: "Case-insensitive duplicate detection (tick-ABC123 = tick-abc123)." Both normalize to `tick-abc123`. The details should show original-case forms so the user can see which entries have which casing.
- **More than two duplicates of the same ID**: Three or more lines sharing the same normalized ID (e.g., `tick-abc123` on lines 2, 5, and 9). This produces a single `CheckResult` for the group listing all occurrences, not pairwise comparisons. The check must handle groups of arbitrary size, not just pairs.
- **Multiple distinct duplicate groups**: Two or more different IDs each duplicated independently (e.g., `tick-aaa111` duplicated on lines 1 and 3, `tick-bbb222` duplicated on lines 4 and 6). Each group produces its own `CheckResult`. The total number of failing results equals the number of distinct duplicate groups, not the total number of duplicate lines.
- **No duplicates**: The happy path. All IDs are unique after normalization. Returns a single passing result. Confirms the check does not produce false positives.
- **Single task**: A file with exactly one task. Cannot possibly have duplicates. Returns a single passing result. This is a boundary condition -- the smallest non-empty valid input.
- **Unparseable lines**: Lines that are not valid JSON are skipped silently. The `JsonlSyntaxCheck` (task 2-1) handles syntax errors. This check should not re-report them. If a file has 3 valid lines (2 duplicates) and 2 unparseable lines, the check reports one duplicate group and ignores the unparseable lines.
- **Lines with missing/empty `id` field**: Lines that parse as valid JSON but have no `id` field or an empty `id` value are skipped. The `IdFormatCheck` (task 2-2) handles invalid IDs. A line without an ID cannot participate in duplicate detection.
- **Missing `tasks.jsonl`**: Same pattern as tasks 2-1 and 2-2 -- report as error with an init suggestion.

## Acceptance Criteria

- [ ] `DuplicateIdCheck` implements the `Check` interface
- [ ] Passing check returns `CheckResult` with Name `"ID uniqueness"` and Passed `true`
- [ ] Duplicate detection is case-insensitive (IDs normalized to lowercase before comparison)
- [ ] Each distinct duplicate group produces its own failing `CheckResult`
- [ ] Details include line numbers and original-case ID forms for each occurrence in the group
- [ ] Groups of more than two are reported as a single result listing all occurrences
- [ ] Lines with invalid JSON are skipped (not re-reported as syntax errors)
- [ ] Lines with missing or empty `id` field are skipped (not re-reported as format errors)
- [ ] Blank and whitespace-only lines are silently skipped
- [ ] Line numbers are 1-based and count blank lines
- [ ] Empty file (zero bytes) returns passing result
- [ ] Single-task file returns passing result
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Suggestion is `"Manual fix required"` for duplicate errors
- [ ] All failures use `SeverityError`
- [ ] Check is read-only -- never modifies `tasks.jsonl`
- [ ] Tests written and passing for all edge cases

## Context

The specification defines duplicate IDs as Error #3: "Case-insensitive duplicate detection (tick-ABC123 = tick-abc123)." The specification output format example shows `✓ ID uniqueness: OK` for the passing case, establishing the check's Name as `"ID uniqueness"`.

The specification's fix suggestion table maps "All other errors" (everything except cache staleness) to "Manual fix required" -- so duplicate ID errors get the generic manual fix suggestion.

The specification requires doctor to report each error individually: "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details." Applied to duplicates, this means each distinct duplicate group is a separate `CheckResult`, not one combined result for all duplicates.

IDs in tick follow the format defined in tick-core: a prefix (`tick-`) followed by 6 hex characters. However, this check does not validate format -- it only detects duplicates among whatever IDs are present. Format validation is handled by `IdFormatCheck` (task 2-2). The checks are intentionally separated to keep each TDD cycle focused and to avoid coupling their concerns.

This check operates on the raw file content, consistent with the Phase 2 goal: "These checks operate on raw file content without relationship semantics." It reads `tasks.jsonl` directly rather than querying the SQLite cache, because the cache might be stale and the JSONL file is the source of truth.

This is a Go project. Use `encoding/json` from stdlib for JSON parsing. Use `bufio.Scanner` or equivalent for line-by-line reading. Use `strings.ToLower` for case normalization. The check implements the `Check` interface defined in task 1-1 and will be registered with the `DiagnosticRunner` in task 2-4.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
