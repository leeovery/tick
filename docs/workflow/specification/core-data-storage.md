# Specification: Core Data & Storage

**Status**: Building specification
**Type**: feature
**Last Updated**: 2026-01-20

---

## Specification

### Overview

Tick uses a dual-storage architecture:

- **JSONL** (`tasks.jsonl`) - Source of truth, committed to git
- **SQLite** (`.tick/cache.db`) - Query cache, gitignored, auto-rebuilds

This design provides git-friendly storage (JSONL diffs cleanly, one line per task) with fast querying (SQLite for complex filters and joins).

**Key principle**: SQLite is a cache, not a peer. It can always be rebuilt from JSONL. Mismatches self-heal on next read.
