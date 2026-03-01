---
topic: cache-schema-versioning
status: planning
format: tick
work_type: bugfix
ext_id: tick-598752
specification: ../specification/cache-schema-versioning/specification.md
spec_commit: 1874c3b2085398bee1254e9773aba96fdd925c24
created: 2026-03-01
updated: 2026-03-01
external_dependencies: []
task_list_gate_mode: gated
author_gate_mode: auto
finding_gate_mode: gated
planning:
  phase: ~
  task: ~
---

# Plan: Cache Schema Versioning

### Phase 1: Schema Version Check and Rebuild
status: approved
ext_id: tick-796ea9
approved_at: 2026-03-01

**Goal**: Add a schema version constant to the cache, store it in metadata during rebuild, and check it early in `ensureFresh()` — triggering a full delete-and-rebuild when the version is missing or mismatched. This is the complete fix for the broken upgrade path.

**Why this order**: This is a single-phase bugfix. The bug has one root cause (no schema-freshness model) and the fix is contained to two files (`cache.go` and `store.go`). There is no incremental value in splitting — the version constant, the storage, and the check are tightly coupled and only meaningful together.

**Acceptance**:
- [ ] `const schemaVersion = 1` exists in `internal/storage/cache.go`
- [ ] `Rebuild()` stores `schema_version` in the `metadata` table alongside `jsonl_hash`
- [ ] `ensureFresh()` checks `schema_version` before calling `IsFresh()`, triggering close + delete + reopen + rebuild on mismatch
- [ ] Test: cache with wrong schema version triggers delete and full rebuild
- [ ] Test: cache with missing schema version (simulating pre-versioning `cache.db`) triggers delete and full rebuild
- [ ] Test: cache with correct schema version is preserved without unnecessary rebuild
- [ ] Test: after a version-triggered rebuild, subsequent queries succeed normally
- [ ] All existing tests in `internal/storage/` continue to pass

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| cache-schema-versioning-1-1 | Add schema version constant and store in metadata during rebuild | none | authored | tick-477711 |
| cache-schema-versioning-1-2 | Check schema version in ensureFresh and delete-rebuild on mismatch | missing schema_version row (pre-versioning cache.db), new empty cache extra rebuild cycle | authored | tick-389db4 |
| cache-schema-versioning-1-3 | End-to-end query success after version-triggered rebuild | none | authored | tick-68d0c3 |
