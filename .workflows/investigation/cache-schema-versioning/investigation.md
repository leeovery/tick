---
topic: cache-schema-versioning
status: in-progress
work_type: bugfix
date: 2026-03-01
---

# Investigation: Cache Schema Versioning

## Symptoms

### Problem Description

**Expected behavior:**
CLI commands (`tick ready`, etc.) query the SQLite cache and return results normally.

**Actual behavior:**
All CLI commands that query the cache fail with: `Error: failed to query tasks: SQL logic error: no such column: t.type (1)`

### Manifestation

- SQL logic error on every cache-querying CLI command
- Deterministic failure — not intermittent
- Caused by SQLite cache schema mismatch after recent feature additions introduced new columns (e.g., `type`)
- Pre-existing `cache.db` files lack columns expected by current code

### Reproduction Steps

1. Have a project with a `.tick/cache.db` created before the recent CLI enhancements
2. Run any tick command that queries the cache (e.g., `tick ready`)
3. Observe SQL logic error about missing column

**Reproducibility:** Always (when cache predates schema changes)

### Environment

- **Affected environments:** Any local environment with a stale cache.db
- **Platform:** All (darwin, linux)
- **User conditions:** Existing tick users who upgrade to the new version

### Impact

- **Severity:** Critical — tool is completely unusable until cache is manually deleted
- **Scope:** All existing users who upgrade
- **Business impact:** Breaks the upgrade path; users must know to manually delete `.tick/cache.db`

### References

- Recent commits added new columns to the SQLite schema (e.g., `type` column)
- Cache is designed to be ephemeral — rebuilt from JSONL source of truth

---

## Analysis

### Initial Hypotheses

User hypothesis: The cache.db is ephemeral and auto-rebuilt from JSONL. The fix should introduce a hard-coded schema version number. When the DB structure changes, increment the version. On startup, compare stored version vs expected — if mismatched, delete and rebuild.

### Code Trace

{To be filled during analysis}

### Root Cause

{To be filled during analysis}

### Contributing Factors

{To be filled during analysis}

### Why It Wasn't Caught

{To be filled during analysis}

### Blast Radius

{To be filled during analysis}

---

## Fix Direction

### Proposed Approach

{To be filled after analysis}

### Alternatives Considered

{To be filled after analysis}

### Testing Recommendations

{To be filled after analysis}

### Risk Assessment

{To be filled after analysis}

---

## Notes

{To be filled during investigation}
