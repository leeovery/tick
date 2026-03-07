---
status: complete
created: 2026-03-01
cycle: 1
phase: Gap Analysis
topic: Cache Schema Versioning
---

# Review Tracking: Cache Schema Versioning - Gap Analysis

## Findings

### 1. Ambiguous placement of version check in implementation step 2

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Implementation step 2, Files affected section

**Details**:
Step 2 says "In `OpenCache()` or early in `ensureFresh()`" — this presents two options without choosing one. The "Key design decisions" section and "Files affected" section both resolve this to `ensureFresh()` in `store.go` calling a check function defined in `cache.go`. However, the unresolved "or" in the primary implementation steps could lead an implementer to put the check inside `OpenCache()` instead, which would change the control flow (OpenCache would need to signal "version mismatch" to its caller rather than handling delete+rebuild itself, since OpenCache doesn't own the file lifecycle).

Suggest removing the "or" from step 2 and stating the placement definitively: "Early in `ensureFresh()`, query `metadata` for key `schema_version`."

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Removed OpenCache() alternative from step 2, now definitively states ensureFresh().

---

### 2. No guidance on version check interaction with brand-new (empty) cache

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Implementation steps 2-4, Key design decisions

**Details**:
When `ensureFresh()` opens a brand-new cache (first run or after deletion), `OpenCache()` creates the schema but the `schema_version` metadata row does not exist yet — it is only written during `Rebuild()` (step 4). The spec says "If missing or value differs → close DB, delete cache file, reopen, force rebuild." On a brand-new cache this triggers an unnecessary close-delete-reopen cycle before the rebuild that would have happened anyway (since the hash also won't match).

This is functionally correct but creates a minor inefficiency. More importantly, an implementer might wonder whether to optimize this case (e.g., skip the version check if the cache was just created in this call) or accept the redundant cycle. A single sentence clarifying the intended behavior would remove the ambiguity — e.g., "On a new cache the missing version triggers the same delete+rebuild path; the extra cycle is acceptable since cache creation is infrequent."

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added to key design decisions bullet list.
