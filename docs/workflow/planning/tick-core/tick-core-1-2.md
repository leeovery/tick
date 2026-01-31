---
id: tick-core-1-2
phase: 1
status: pending
created: 2026-01-30
---

# JSONL storage with atomic writes

## Goal

Tasks need persistent storage that diffs cleanly in git and survives process crashes. Without atomic writes, a crash mid-write could corrupt the task file. Implement a JSONL reader/writer that serializes tasks one-per-line and uses the temp file + fsync + rename pattern for crash-safe writes.

## Implementation

- Implement JSONL writer: serialize each `Task` as a single JSON line, no array wrapper, no trailing commas
- Omit optional fields when empty/null (don't serialize as `null` — omit the key entirely)
- Implement JSONL reader: parse file line-by-line into `[]Task`, skip empty lines
- Implement atomic write: write to temp file in same directory → `fsync` → `os.Rename(temp, tasks.jsonl)`
- Full file rewrite on every mutation (no append mode)
- Handle empty file (0 bytes) as valid — returns empty task list
- Handle `tasks.jsonl` not existing (pre-init) as an error

## Tests

- `"it writes tasks as one JSON object per line"`
- `"it reads tasks from JSONL file"`
- `"it round-trips all task fields without loss"`
- `"it omits optional fields when empty"`
- `"it returns empty list for empty file"`
- `"it writes atomically via temp file and rename"`
- `"it handles tasks with all fields populated"`
- `"it handles tasks with only required fields"`

## Edge Cases

- Empty file (0 bytes): valid, returns empty task list
- Missing file: error (not initialized)
- Optional fields (`description`, `blocked_by`, `parent`, `closed`): omitted from JSON when zero/nil, not serialized as `null`
- Atomic write: if process crashes mid-write, temp file is left behind but original is intact

## Acceptance Criteria

- [ ] Tasks round-trip through write → read without data loss
- [ ] Optional fields omitted from JSON when empty/null
- [ ] Atomic write uses temp file + fsync + rename
- [ ] Empty file returns empty task list
- [ ] Each task occupies exactly one line (no pretty-printing)
- [ ] JSONL output matches spec format (field ordering: id, title, status, priority, then optional fields)

## Context

JSONL format: one JSON object per line, no trailing commas, no array wrapper. Optional fields (`description`, `blocked_by`, `parent`, `closed`) are omitted when empty — not serialized as `null`. The `updated` field is always present (set to `created` value initially). Full file rewrite for all operations — no append-only mode. Atomic rewrite pattern: write to temp file → fsync → `os.Rename(temp, tasks.jsonl)` — atomic on POSIX.

Specification reference: `docs/workflow/specification/tick-core.md` (for ambiguity resolution)
