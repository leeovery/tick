# Discussion: Freshness Check & Dual Write Implementation

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick uses a dual-storage architecture:
- **JSONL** (`tasks.jsonl`) - Source of truth, committed to git
- **SQLite** (`.cache/tick.db`) - Query cache, gitignored

This design gives us git-friendly storage (JSONL) with fast querying (SQLite). But it introduces a synchronization problem: how do we keep them in sync, and what happens when they diverge?

The research phase proposed:
- Freshness checks on read (compare hash/mtime)
- Dual writes on mutations (write to both atomically)
- Full file rewrite for updates

### References

- [Research: exploration.md](../research/exploration.md) (lines 150-173)

## Questions

- [x] Hash vs mtime for freshness detection?
      - What are the trade-offs?
      - Cross-platform considerations?
- [ ] How to handle partial write failures?
      - What if JSONL write succeeds but SQLite fails?
      - What if the reverse happens?
- [ ] Is full file rewrite acceptable for updates?
      - Performance implications?
      - Alternatives?
- [ ] What triggers a cache rebuild?
      - Only freshness mismatch, or other scenarios?
- [ ] Should writes be truly atomic (both succeed or neither)?
      - Transaction semantics?

---

## Hash vs mtime for freshness detection?

### Context

On every read operation, we need to check if the SQLite cache is stale. Two approaches: compare file modification time (mtime) or compute a content hash.

### Options Considered

**Option A: mtime comparison**
- Pros: Fast (single stat call), no I/O beyond metadata
- Cons: Can miss changes (same-second edits, clock skew, git operations can preserve mtime)

**Option B: Content hash (MD5/SHA256)**
- Pros: Definitive - if content changed, hash changed
- Cons: Must read entire file to compute, O(n) on file size

**Option C: Hybrid (mtime + size, hash on mismatch)**
- Pros: Fast path for common case, definitive when needed
- Cons: More complex, still vulnerable to mtime issues

### Journey

Initial instinct was mtime for speed, but on reflection the edge cases are problematic for an agent-first tool:
- Git operations can mess with mtime
- Fast agents might edit within the same second
- Cross-platform consistency matters

The read cost argument fell apart when we realized: we're reading the file anyway on cache miss. Computing a hash on data already in memory is negligible.

### Decision

**Content hash (SHA256)**. Store hash in SQLite metadata table. On read: load JSONL into memory, compute hash, compare. If mismatch, rebuild from data already in memory (no double-read).

Trade-off accepted: Always read full file on every operation. Acceptable for expected file sizes (<1MB, <10ms).

---

## How to handle partial write failures?

### Context

A mutation (create, update, delete) needs to write to both JSONL and SQLite. What if one succeeds and the other fails?

### Options Considered

**Option A: JSONL-first, SQLite is just cache**
- Write JSONL first (with atomic temp+rename)
- Then update SQLite
- If SQLite fails: who cares, it rebuilds on next read anyway
- Pros: Simple, JSONL is already source of truth
- Cons: Next read pays rebuild cost

**Option B: True two-phase commit**
- Begin SQLite transaction
- Write temp JSONL
- Commit SQLite
- Rename JSONL
- Rollback SQLite if rename fails
- Pros: Both always in sync
- Cons: Complex, overkill?

**Option C: SQLite-first with JSONL regeneration**
- Write to SQLite first
- Generate JSONL from SQLite
- Pros: SQLite has better transaction support
- Cons: Inverts the source-of-truth model, JSONL becomes derived

### Journey

*(Discussion in progress)*

### Decision

*(Pending)*

---

