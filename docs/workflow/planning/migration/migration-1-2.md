---
id: migration-1-2
phase: 1
status: pending
created: 2026-01-31
---

# Beads Provider - Read & Map

## Goal

The migration system has types and a provider interface (migration-1-1) but no concrete provider. Without a beads provider, there is nothing to feed the migration engine. This task implements the first (and currently only) provider: reading `.beads/issues.jsonl` from the filesystem and mapping each JSON line to a `MigratedTask`. It must handle every filesystem and data-quality edge case so the engine receives either a clean slice of normalized tasks or a clear error explaining why the source could not be read.

## Implementation

- Create `internal/migrate/beads/` package with a `BeadsProvider` struct.
- `BeadsProvider` takes a base directory path in its constructor (the directory expected to contain a `.beads/` folder). This makes the provider testable without relying on the real working directory.
  ```go
  func NewBeadsProvider(baseDir string) *BeadsProvider
  ```
- Implement `Name() string` returning `"beads"`.
- Implement `Tasks() ([]MigratedTask, error)`:
  1. Construct path: `filepath.Join(baseDir, ".beads", "issues.jsonl")`
  2. Check the `.beads` directory exists — if not, return an error: `".beads directory not found in <path>"`.
  3. Check `issues.jsonl` exists inside `.beads/` — if not, return an error: `"issues.jsonl not found in <path>/.beads"`.
  4. Open and read the file line by line (use `bufio.Scanner`).
  5. Skip blank lines.
  6. For each non-blank line, unmarshal JSON into a raw beads struct (unexported), then map to `MigratedTask`. If a line is malformed JSON, skip it and collect it as a warning/error (do NOT abort the whole file — other lines may be valid). Return all successfully parsed tasks plus an error only if zero tasks could be parsed from a non-empty file.
  7. An empty file (zero lines, or all blank lines) returns an empty slice and nil error — this is not an error condition.

- Define an unexported `beadsIssue` struct for JSON unmarshalling with fields matching the beads JSONL format:
  ```go
  type beadsIssue struct {
      ID           string        `json:"id"`
      Title        string        `json:"title"`
      Description  string        `json:"description"`
      Status       string        `json:"status"`
      Priority     int           `json:"priority"`
      IssueType    string        `json:"issue_type"`
      CreatedAt    string        `json:"created_at"`
      UpdatedAt    string        `json:"updated_at"`
      ClosedAt     string        `json:"closed_at"`
      CloseReason  string        `json:"close_reason"`
      CreatedBy    string        `json:"created_by"`
      Dependencies []interface{} `json:"dependencies"`
  }
  ```

- Implement a `mapToMigratedTask(issue beadsIssue) (MigratedTask, error)` function (unexported) that performs the mapping:

  **Status mapping** (beads status to tick status):
  | Beads | Tick |
  |-------|------|
  | `"pending"` | `"open"` |
  | `"in_progress"` | `"in_progress"` |
  | `"closed"` | `"done"` |
  | empty/unknown | `""` (let MigratedTask defaults handle it) |

  **Priority mapping** (beads 0-3 to tick 0-4):
  - Beads uses 0-3; tick uses 0-4. Map directly (0→0, 1→1, 2→2, 3→3). Beads priority values already fall within tick's valid range, so no scaling needed. If beads priority is outside 0-3, pass it through and let `MigratedTask.Validate()` catch invalid values.

  **Timestamp mapping**:
  - Parse `created_at`, `updated_at`, `closed_at` as ISO 8601 (`time.Parse(time.RFC3339, ...)`)
  - If parsing fails, leave the `time.Time` as zero value (defaults applied by MigratedTask)

  **Discarded fields** (no tick equivalent):
  - `id` — tick generates its own IDs
  - `issue_type` — tick has no epic/task distinction
  - `close_reason` — tick has no equivalent
  - `created_by` — tick has no multi-user model
  - `dependencies` — tick uses a different dependency model

  **Title validation**:
  - If `title` is empty after trimming, this line cannot produce a valid task. Skip it and record it as a parse failure (include the beads `id` if available, or the line number, in the error message).

- Validate each mapped `MigratedTask` using its `Validate()` method before including it in the returned slice. Invalid tasks are skipped and recorded.

## Tests

- `"Name returns beads"`
- `"Tasks reads valid JSONL and returns MigratedTasks"`
- `"Tasks maps beads pending status to tick open"`
- `"Tasks maps beads in_progress status to tick in_progress"`
- `"Tasks maps beads closed status to tick done"`
- `"Tasks maps unknown status to empty string"`
- `"Tasks maps beads priority values directly (0-3)"`
- `"Tasks parses ISO 8601 timestamps into time.Time"`
- `"Tasks returns error when .beads directory is missing"`
- `"Tasks returns error when issues.jsonl is missing"`
- `"Tasks returns empty slice and nil error for empty file"`
- `"Tasks returns empty slice and nil error for file with only blank lines"`
- `"Tasks skips malformed JSON lines and returns valid tasks"`
- `"Tasks skips lines with empty title and returns valid tasks"`
- `"Tasks skips lines with whitespace-only title and returns valid tasks"`
- `"Tasks discards id, issue_type, close_reason, created_by, dependencies fields"`
- `"Tasks maps description field to MigratedTask Description"`
- `"Tasks handles closed_at timestamp for closed tasks"`
- `"Tasks leaves timestamp fields as zero when parsing fails"`
- `"mapToMigratedTask produces valid MigratedTask from fully populated beadsIssue"`

## Edge Cases

**Missing .beads directory**: `Tasks()` returns a descriptive error. The engine will handle this as a provider-level failure — no tasks to process.

**Missing issues.jsonl**: `.beads/` exists but `issues.jsonl` does not. Same treatment — descriptive error returned.

**Empty file**: Zero lines or all blank lines. Returns empty slice, nil error. The engine treats this as zero tasks (not a failure).

**Malformed JSON lines**: A line that is not valid JSON is skipped. Other valid lines in the same file are still processed and returned. This is a line-level error, not a file-level error.

**Missing title**: A JSON line that parses successfully but has an empty or whitespace-only title is skipped. The beads `id` (or line number if id is empty) is included in the skip message for debugging.

**Discarded fields**: Fields like `issue_type`, `close_reason`, `created_by`, and `dependencies` are present in the beads format but have no tick equivalent. They are parsed into the intermediate struct but never transferred to `MigratedTask`. Tests verify they do not appear in the output.

**Status mapping**: Beads uses `"pending"` / `"in_progress"` / `"closed"`. Tick uses `"open"` / `"in_progress"` / `"done"` / `"cancelled"`. The mapping is: pending→open, in_progress→in_progress, closed→done. Unknown/empty status maps to empty string (MigratedTask default of `"open"` applies later).

**Priority mapping**: Beads uses 0-3, tick uses 0-4. Values map 1:1 since beads range is a subset of tick's range. No scaling or transformation needed.

## Acceptance Criteria

- [ ] `BeadsProvider` struct implements the `Provider` interface (compile-time verified)
- [ ] `Name()` returns `"beads"`
- [ ] `Tasks()` reads `.beads/issues.jsonl` from the configured base directory
- [ ] Beads status values are correctly mapped to tick equivalents (pending→open, in_progress→in_progress, closed→done)
- [ ] Priority values 0-3 are passed through directly
- [ ] ISO 8601 timestamps are parsed into `time.Time` fields
- [ ] Unparseable timestamps result in zero-value `time.Time` (not an error)
- [ ] Fields with no tick equivalent (id, issue_type, close_reason, created_by, dependencies) are discarded
- [ ] Missing `.beads` directory returns a descriptive error
- [ ] Missing `issues.jsonl` returns a descriptive error
- [ ] Empty file returns empty slice and nil error
- [ ] Malformed JSON lines are skipped; valid lines in the same file are still returned
- [ ] Lines with empty/whitespace-only title are skipped
- [ ] All tests written and passing

## Context

The specification states: "Map all available fields from source to tick equivalents. Missing data uses sensible defaults or is left empty. Extra source fields with no tick equivalent are discarded." Title is the only required field. The provider is file-based (no authentication). The beads JSONL format stores one issue per line with fields: id, title, description, status, priority, issue_type, created_at, updated_at, closed_at, close_reason, created_by, dependencies. The spec's error strategy is "continue on error, report failures at end" — this applies at the line level within the provider: malformed lines don't abort the file read.

The `MigratedTask` struct (from migration-1-1) has: Title, Status (open/in_progress/done/cancelled), Priority (0-4), Description, Created, Updated, Closed. The `Provider` interface requires `Name() string` and `Tasks() ([]MigratedTask, error)`.

Specification reference: `docs/workflow/specification/migration.md`
