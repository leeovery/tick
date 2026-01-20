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

### Task Schema

Each task has 10 fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | Generated | Unique identifier (see ID Generation) |
| `title` | string | Yes | - | Brief description of the task |
| `status` | enum | Yes | `open` | One of: `open`, `in_progress`, `done`, `cancelled` |
| `priority` | integer | Yes | `2` | 0 (highest) to 4 (lowest) |
| `description` | string | No | `""` | Extended details, markdown supported |
| `blocked_by` | array | No | `[]` | IDs of tasks that block this one |
| `parent` | string | No | `null` | ID of parent task (organizational grouping) |
| `created` | datetime | Yes | Now | ISO 8601 timestamp |
| `updated` | datetime | Yes | Now | ISO 8601 timestamp, updated on any change |
| `closed` | datetime | No | `null` | ISO 8601 timestamp, set when status becomes `done` or `cancelled` |

#### Field Constraints

- **status**: Only valid transitions enforced by commands (e.g., can't `start` a `done` task)
- **priority**: Integer 0-4 only; invalid values rejected
- **blocked_by**: Must reference existing task IDs; self-references rejected; cycles rejected at write time
- **parent**: Must reference existing task ID; self-references rejected

#### Hierarchy Semantics

- **Parent/child is organizational** - A parent groups related tasks but does not block them
- **blocked_by controls workflow** - A task cannot be worked until its blockers are resolved
- **Cancelled tasks unblock dependents** - When a task is cancelled, tasks blocked by it become unblocked
