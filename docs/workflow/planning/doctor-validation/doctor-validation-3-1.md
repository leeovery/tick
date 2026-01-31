---
id: doctor-validation-3-1
phase: 3
status: pending
created: 2026-01-31
---

# Orphaned Parent Reference Check

## Goal

Phase 2 checks validate raw file integrity (syntax, ID format, duplicates) but nothing yet validates task relationships. All five relationship error checks (orphaned parents, orphaned dependencies, self-references, cycles, child-blocked-by-parent) and the one warning check (parent done with open children) need to parse `tasks.jsonl` and extract each task's `id`, `parent`, `blocked_by`, and `status` into a structure they can query. Today, each Phase 2 check independently reads and parses the file. If every Phase 3 check did the same, there would be six redundant file reads and six redundant parsing implementations. This task solves both problems at once: it builds a shared task relationship parser/loader that reads `tasks.jsonl` once and extracts the fields all relationship checks need, and it implements the first check that uses it — the orphaned parent reference check. This is specification Error #5: "Task references non-existent parent."

The tick-core specification describes: "Orphaned children (parent reference points to non-existent task) — Task remains valid, treated as root-level task. `tick doctor` flags: 'tick-child references non-existent parent tick-deleted'." Doctor reports this but never modifies data.

## Implementation

### Shared Task Relationship Parser

- Create a `TaskRelationshipData` struct (or similar name) that holds the extracted fields needed by all relationship checks:
  - `ID` (string) — the task's ID
  - `Parent` (string, may be empty) — the parent task ID, empty string if null/absent
  - `BlockedBy` ([]string) — the dependency IDs, empty slice if null/absent
  - `Status` (string) — the task's status (e.g., "open", "in_progress", "done", "cancelled")
  - `Line` (int) — the 1-based line number in the file (for error reporting)

- Create a `ParseTaskRelationships` function (or method) with signature approximately:
  ```go
  func ParseTaskRelationships(tickDir string) ([]TaskRelationshipData, error)
  ```
  This function:
  1. Attempts to open `.tick/tasks.jsonl`. If the file does not exist, returns an error (or a sentinel value) that calling checks can translate into their appropriate `CheckResult` (e.g., "tasks.jsonl not found").
  2. Reads the file line by line. For each line:
     - **Skip blank lines**: If the line is empty or contains only whitespace after trimming, skip it silently. Blank lines still count toward line numbering (consistent with Phase 2 convention).
     - **Attempt JSON parse**: Parse the line into a `map[string]interface{}` or a minimal struct. If parsing fails, skip the line silently — the JSONL syntax check (task 2-1) is responsible for reporting parse errors. The relationship parser does not duplicate syntax error reporting. It simply omits unparseable lines from the result set.
     - **Extract fields**: From the parsed JSON object:
       - `id`: Extract as string. If missing or not a string, skip the line (the ID format check handles those).
       - `parent`: Extract as string if present and non-null. If absent or null, set to empty string. A task with no parent is a root-level task — this is valid and expected.
       - `blocked_by`: Extract as `[]string` if present and non-null. If absent or null, set to empty slice. Handle the JSON array by iterating and extracting string elements.
       - `status`: Extract as string if present. If missing, default to empty string.
       - `line`: Record the 1-based line number.
     - Add the extracted `TaskRelationshipData` to the result slice.
  3. Return the slice of all successfully parsed task records. An empty file produces an empty slice (not an error).

- The parser is **reusable** — it is not coupled to any specific check. Tasks 3-2 through 3-6 will import and call the same function. The parser lives in the same package as the relationship checks (or a shared doctor utilities package) so all checks can access it.

- The parser is **read-only** — it never modifies `tasks.jsonl`.

- Design the parser so that calling code can also build a lookup map (set of known IDs) from the returned slice. This is a one-liner for callers: iterate the slice and collect IDs into a `map[string]bool` or `map[string]struct{}`. The parser itself does not build this map — it returns the raw data and lets each check decide what index structures it needs.

### Orphaned Parent Reference Check

- Create an `OrphanedParentCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path.

- Implement the `Run` method with the following logic:
  1. Call `ParseTaskRelationships(tickDir)` to get the task data. If the parser returns a file-not-found error, return a single failing `CheckResult` with Name `"Orphaned parents"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. This is consistent with the Phase 2 pattern for missing files.
  2. Build a set of all known task IDs from the returned data (iterate the slice, add each `ID` to a `map[string]bool`).
  3. Iterate the task data again. For each task where `Parent` is non-empty:
     - Check if `Parent` exists in the known-ID set.
     - If not found, record a failing `CheckResult` with Name `"Orphaned parents"`, Severity `SeverityError`, Details following the pattern from the tick-core specification: `"tick-{child-id} references non-existent parent tick-{parent-id}"` (e.g., `"tick-a1b2c3 references non-existent parent tick-missing"`), and Suggestion `"Manual fix required"`.
  4. After checking all tasks, if no orphaned parents were found, return a single passing `CheckResult` with Name `"Orphaned parents"` and Passed `true`.
  5. If orphaned parents were found, return all the failing `CheckResult` entries (one per orphaned parent reference). Do not include a passing result alongside failures.

- The check name `"Orphaned parents"` is derived from the specification output format example which shows `"✗ Orphaned reference: tick-a1b2c3 references non-existent parent tick-missing"`. Since the specification uses one output line per error, the Name is the check category label (like `"JSONL syntax"`, `"ID format"`, `"ID uniqueness"` from Phase 2). The Details carry the specific task IDs.

- The check does **not** normalize IDs to lowercase before comparison. IDs in `tasks.jsonl` should already be lowercase (write-time normalization). If a parent reference uses different casing than the target, the ID format check (task 2-2) catches the stored casing issue. The orphaned parent check compares IDs as-is.

- Tasks with `parent` set to `null` or with the `parent` field absent are valid root-level tasks. The parser represents these as empty string for `Parent`. The check skips these — they are not orphaned references.

## Tests

### Parser Tests

- `"ParseTaskRelationships returns empty slice for empty file (zero bytes)"`
- `"ParseTaskRelationships returns error when tasks.jsonl does not exist"`
- `"ParseTaskRelationships extracts id, parent, blocked_by, and status from valid JSON lines"`
- `"ParseTaskRelationships sets parent to empty string when parent field is null"`
- `"ParseTaskRelationships sets parent to empty string when parent field is absent"`
- `"ParseTaskRelationships sets blocked_by to empty slice when field is null"`
- `"ParseTaskRelationships sets blocked_by to empty slice when field is absent"`
- `"ParseTaskRelationships skips blank and whitespace-only lines"`
- `"ParseTaskRelationships skips unparseable JSON lines silently"`
- `"ParseTaskRelationships skips lines with missing or non-string id"`
- `"ParseTaskRelationships records correct 1-based line numbers including blank lines"`
- `"ParseTaskRelationships handles trailing newline without extra entry"`
- `"ParseTaskRelationships extracts multiple blocked_by IDs correctly"`
- `"ParseTaskRelationships does not modify tasks.jsonl (read-only)"`

### Orphaned Parent Check Tests

- `"it returns passing result when all parent references are valid"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns passing result when no tasks have parent references (all root tasks)"`
- `"it returns failing result when a task references a non-existent parent"`
- `"it returns one failing result per orphaned parent reference when multiple exist"`
- `"it includes child ID and missing parent ID in error details"`
- `"it follows specification wording: 'tick-child references non-existent parent tick-missing'"`
- `"it treats parent null as valid root task (not an orphaned reference)"`
- `"it treats absent parent field as valid root task (not an orphaned reference)"`
- `"it skips unparseable lines — does not report them as orphaned parents"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it suggests 'Manual fix required' for orphaned parent errors"`
- `"it uses CheckResult Name 'Orphaned parents' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it does not normalize IDs before comparison (compares as-is)"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Missing `tasks.jsonl`**: The file does not exist in the `.tick/` directory. The parser returns an error, and the check translates it into a single failing `CheckResult` with a suggestion to initialize. This is consistent with the Phase 2 pattern (tasks 2-1, 2-2, 2-3).

- **Empty file (zero bytes)**: The file exists but has no content. The parser returns an empty slice. No tasks means no parent references to validate. The check returns a single passing result. This represents an initialized project with no tasks yet.

- **All parents valid**: Every task that has a non-empty `parent` field references an ID that exists in the file. The check returns a single passing result. This is the happy path confirming no false positives.

- **Multiple orphaned parents**: Several tasks reference parent IDs that do not exist in the file. Each orphaned reference produces its own failing `CheckResult` with the specific child and parent IDs. For example, if `tick-aaa111` references missing parent `tick-xxx` and `tick-bbb222` references missing parent `tick-yyy`, two separate results are returned. This follows the spec: "Doctor lists each error individually."

- **Parent field null (valid root task)**: A task line like `{"id": "tick-abc123", "parent": null, ...}` has an explicit null parent. The parser sets `Parent` to empty string. The check skips it — null parent means root-level task, not an orphaned reference. This is a critical distinction: null/absent parent is intentionally root; a non-null parent referencing a missing ID is an error.

- **Parent field absent (valid root task)**: A task line like `{"id": "tick-abc123", "title": "..."}` has no `parent` key at all. The tick-core specification says optional fields are "omitted when empty/null (not serialized as null)." The parser sets `Parent` to empty string. The check treats this identically to null — a valid root task.

- **Unparseable lines skipped**: Lines that are not valid JSON are skipped by the parser. They do not contribute to the known-ID set, and they do not produce orphaned parent errors. The JSONL syntax check (task 2-1) is responsible for reporting parse errors. If a task's parent references an ID that was on an unparseable line, that parent IS missing from the known-ID set, and the reference IS reported as orphaned. This is correct — from the perspective of parseable data, the parent does not exist.

- **Task references its own ID as parent**: A task like `{"id": "tick-abc123", "parent": "tick-abc123"}` references itself as its parent. The orphaned parent check does NOT flag this — the ID `tick-abc123` exists in the known-ID set, so the reference is not orphaned. Whether self-parenting is an error is a different concern (the specification does not list it as a doctor check; the tick-core spec only rejects self-references for `blocked_by`, not `parent`).

- **Multiple tasks referencing the same non-existent parent**: If `tick-aaa` and `tick-bbb` both have `parent: "tick-missing"`, two separate failing results are produced — one for each child. The error is per-reference, not per-missing-parent.

## Acceptance Criteria

- [ ] `TaskRelationshipData` struct defined with `ID`, `Parent`, `BlockedBy`, `Status`, and `Line` fields
- [ ] `ParseTaskRelationships` function reads `tasks.jsonl` and returns a slice of `TaskRelationshipData`
- [ ] Parser skips blank lines, unparseable lines, and lines with missing/non-string IDs
- [ ] Parser correctly extracts `parent` (string or empty), `blocked_by` (slice or empty), and `status`
- [ ] Parser returns error for missing `tasks.jsonl`; returns empty slice for empty file
- [ ] Parser is reusable — not coupled to the orphaned parent check
- [ ] `OrphanedParentCheck` implements the `Check` interface
- [ ] Passing check returns `CheckResult` with Name `"Orphaned parents"` and Passed `true`
- [ ] Each orphaned parent reference produces its own failing `CheckResult` with child ID and missing parent ID in details
- [ ] Details follow spec wording: `"tick-{child} references non-existent parent tick-{parent}"`
- [ ] Null or absent `parent` field treated as valid root task (not flagged)
- [ ] Unparseable lines skipped by parser — not flagged as orphaned references
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Suggestion is `"Manual fix required"` for orphaned parent errors
- [ ] All failures use `SeverityError`
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] Parser tests written and passing for all parser behaviors
- [ ] Orphaned parent check tests written and passing for all edge cases

## Context

The specification defines orphaned parent references as Error #5: "Task references non-existent parent." The tick-core specification (section "Hierarchy & Dependency Model > Edge Cases") describes the behavior: "Orphaned children (parent reference points to non-existent task) — Task remains valid, treated as root-level task. `tick doctor` flags: 'tick-child references non-existent parent tick-deleted'." Doctor reports this but never fixes it — "No auto-fix — human/agent decides whether to remove parent reference."

The tick-core task schema defines `parent` as: type `string`, required `No`, default `null`. The JSONL format says "Optional fields omitted when empty/null (not serialized as null)." So a root task will either have `"parent": null` or no `parent` key at all. Both are valid root tasks and must not be flagged.

The `blocked_by` field is: type `array`, required `No`, default `[]`. It is extracted by the shared parser for use by tasks 3-2 through 3-6 but is not used by this check (the orphaned parent check only examines `parent`).

The `status` field is: type `enum` (`open`, `in_progress`, `done`, `cancelled`), required `Yes`. It is extracted by the parser for use by tasks 3-5 and 3-6 (child-blocked-by-parent and parent-done-with-open-children) but is not used by this check.

The specification's output format example shows: `"✗ Orphaned reference: tick-a1b2c3 references non-existent parent tick-missing"`. The fix suggestion table maps "All other errors" to `"Manual fix required"`.

The specification requires doctor to report each error individually: "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details."

This is the first Phase 3 task and introduces the shared `ParseTaskRelationships` function that subsequent tasks (3-2 through 3-6) will reuse. The parser is intentionally minimal — it extracts only the fields needed for relationship validation (`id`, `parent`, `blocked_by`, `status`, `line`) and leaves all other fields unread. It follows the same conventions established in Phase 2: blank lines skipped, unparseable lines skipped, 1-based line numbering, read-only file access.

This is a Go project. Use `encoding/json` for JSON parsing and `bufio.Scanner` (or equivalent) for line-by-line reading. The check implements the `Check` interface defined in task 1-1 and will be registered with the `DiagnosticRunner` in task 3-7.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
