---
id: tick-core-1-3
phase: 1
status: pending
created: 2026-01-30
---

# SQLite cache with freshness detection

## Goal

Queries against JSONL are too slow for filtered/sorted results at scale, and JSONL can't express normalized relationships (dependencies). Tick uses SQLite as an auto-rebuilding cache — expendable, gitignored, always rebuildable from JSONL. This task implements the SQLite schema, the full rebuild-from-JSONL pipeline, and SHA256 hash-based freshness detection so the cache self-heals on every operation without explicit sync commands.

## Implementation

- Define SQLite schema with three tables and three indexes:
  - `tasks` table: `id TEXT PRIMARY KEY`, `title TEXT NOT NULL`, `status TEXT NOT NULL DEFAULT 'open'`, `priority INTEGER NOT NULL DEFAULT 2`, `description TEXT`, `parent TEXT`, `created TEXT NOT NULL`, `updated TEXT NOT NULL`, `closed TEXT`
  - `dependencies` table: `task_id TEXT NOT NULL`, `blocked_by TEXT NOT NULL`, `PRIMARY KEY (task_id, blocked_by)` — normalizes the `blocked_by` array from JSONL
  - `metadata` table: `key TEXT PRIMARY KEY`, `value TEXT` — stores `jsonl_hash` for freshness detection
  - Indexes: `idx_tasks_status` on `tasks(status)`, `idx_tasks_priority` on `tasks(priority)`, `idx_tasks_parent` on `tasks(parent)`
- Implement cache initialization: create `cache.db` at `.tick/cache.db`, create all tables and indexes if not present
- Implement full rebuild from JSONL: accept a `[]Task` slice (already parsed by the JSONL reader from tick-core-1-2), clear all existing rows, insert all tasks and dependencies in a single transaction, compute and store the SHA256 hash of the raw JSONL file content in `metadata` as key `jsonl_hash`
- Implement freshness check: compute SHA256 hash of JSONL file contents, compare with `jsonl_hash` stored in `metadata` table, return whether cache is fresh or stale
- Implement `EnsureFresh` function: takes raw JSONL bytes and parsed `[]Task`, checks freshness, triggers rebuild if stale, no-ops if fresh. This is the gatekeeper called on every operation.
- Handle missing `cache.db`: if the file doesn't exist or can't be opened, create it from scratch and do a full rebuild
- Handle corrupted `cache.db`: if any query errors (schema mismatch, disk corruption), delete the file, recreate, and rebuild. Log a warning but don't fail the operation.
- Use `github.com/mattn/go-sqlite3` as the SQLite driver
- Hash computation: `crypto/sha256` from Go stdlib, hash the entire raw file bytes (not the parsed structs)

## Tests

- `"it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)"`
- `"it rebuilds cache from parsed tasks — all fields round-trip correctly"`
- `"it normalizes blocked_by array into dependencies table rows"`
- `"it stores JSONL content hash in metadata table after rebuild"`
- `"it detects fresh cache (hash matches) and skips rebuild"`
- `"it detects stale cache (hash mismatch) and triggers rebuild"`
- `"it rebuilds from scratch when cache.db is missing"`
- `"it deletes and recreates cache.db when corrupted"`
- `"it handles empty task list (zero rows, hash still stored)"`
- `"it replaces all existing data on rebuild (no stale rows)"`
- `"it rebuilds within a single transaction (all-or-nothing)"`

## Edge Cases

- Missing `cache.db`: create fresh database, run full rebuild. This is the normal path on first operation after `tick init` (init does not create the cache).
- Corrupted `cache.db` (invalid schema, disk corruption, partial write): delete the file, recreate from scratch, rebuild from JSONL. Log warning to stderr but continue — the operation must succeed if JSONL is intact.
- Hash in metadata table: stored under key `jsonl_hash` in the `metadata` table. If the key is missing (e.g., empty metadata table), treat as stale and rebuild.
- Empty JSONL (0 bytes, 0 tasks): valid state. Cache should have zero rows in `tasks` and `dependencies`, but still store the hash of the empty file content in `metadata`.
- Rebuild atomicity: the full rebuild (clear + insert all + update hash) must happen in a single SQLite transaction. If any step fails, the transaction rolls back and the error is surfaced.

## Acceptance Criteria

- [ ] SQLite schema matches spec exactly (3 tables, 3 indexes)
- [ ] Full rebuild from `[]Task` populates tasks and dependencies tables correctly
- [ ] SHA256 hash of JSONL content stored in metadata table after rebuild
- [ ] Freshness check correctly identifies fresh vs stale cache
- [ ] Missing cache.db triggers automatic creation and rebuild
- [ ] Corrupted cache.db is deleted, recreated, and rebuilt without failing the operation
- [ ] Empty task list handled (zero rows, hash still stored)
- [ ] Rebuild is transactional (all-or-nothing within single SQLite transaction)

## Context

SQLite is a cache, not a peer. It can always be rebuilt from JSONL. The specification explicitly states: "Mismatches self-heal on next read." The freshness detection sequence on every operation is: (1) read `tasks.jsonl` into memory, (2) compute SHA256 hash, (3) compare with hash in SQLite metadata, (4) if mismatch, rebuild from JSONL data already in memory (no double-read). SQLite is created on first operation, not at `tick init`. Cache file lives at `.tick/cache.db` and is gitignored.

Rebuild triggers: hash mismatch (primary), SQLite missing, SQLite query errors, explicit `tick rebuild` command. This task covers the first three; the explicit command is Phase 5.

The hash update is always part of the same transaction as the data update — this prevents a scenario where data is written but the hash is not, which would cause an unnecessary rebuild on next read.

Specification reference: `docs/workflow/specification/tick-core.md` (for ambiguity resolution)
