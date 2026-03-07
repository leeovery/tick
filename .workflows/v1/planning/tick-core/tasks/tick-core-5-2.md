---
id: tick-core-5-2
phase: 5
status: completed
created: 2026-01-30
---

# tick rebuild command

## Goal

Implement `tick rebuild` — force a complete SQLite cache rebuild from JSONL, bypassing the freshness check. Diagnostic tool for corrupted cache, debugging, or after manual JSONL edits.

## Implementation

- CLI handler acquires exclusive file lock (same as write path)
- Delete existing `cache.db` if present
- Full rebuild: read JSONL, parse all records, create SQLite schema, insert all tasks
- Update hash in metadata table to current JSONL hash
- Release lock
- Output via `Formatter.FormatMessage()` — confirmation showing count of tasks rebuilt
- `--quiet`: suppress output entirely
- `--verbose`: log each step (delete, read, insert count, hash update) to stderr

## Tests

- `"it rebuilds cache from JSONL"`
- `"it handles missing cache.db (fresh build)"`
- `"it overwrites valid existing cache"`
- `"it updates hash in metadata table after rebuild"`
- `"it acquires exclusive lock during rebuild"`
- `"it outputs confirmation message with task count"`
- `"it suppresses output with --quiet"`
- `"it logs rebuild steps with --verbose"`

## Edge Cases

- Missing cache.db: same as fresh build, no error
- Valid cache overwritten: rebuild replaces regardless of freshness
- Empty JSONL: rebuild creates empty cache with correct schema, 0 tasks
- Concurrent access: exclusive lock prevents reads during rebuild
- Malformed JSONL lines: skip with warning to stderr (behavior defined by JSONL reader from tick-core-1-2 — implementer decides error handling strategy there)

## Acceptance Criteria

- [ ] Rebuilds SQLite from JSONL regardless of current freshness
- [ ] Deletes existing cache before rebuild
- [ ] Updates hash in metadata table
- [ ] Acquires exclusive lock during rebuild
- [ ] Handles missing cache.db without error
- [ ] Handles empty JSONL (0 tasks rebuilt)
- [ ] Outputs confirmation with task count
- [ ] --quiet suppresses output
- [ ] --verbose logs rebuild steps to stderr

## Context

Spec: "tick rebuild forces a complete rebuild of the SQLite cache from JSONL, bypassing the freshness check." Use cases: corrupted SQLite, debugging cache issues, after manual JSONL edits. Reuses the same rebuild logic from the storage engine (Phase 1 task 1-3) but triggered explicitly instead of by freshness mismatch.

Specification reference: `docs/workflow/specification/tick-core.md`
