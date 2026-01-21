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

### ID Generation

#### Format

Task IDs follow the pattern: `{prefix}-{6 hex chars}`

- **Prefix**: `tick` (hardcoded, no configuration)
- **Random part**: 6 lowercase hexadecimal characters (e.g., `a3f2b7`)
- **Example**: `tick-a3f2b7`

#### Generation

- Use `crypto/rand` from Go stdlib (3 random bytes → 6 hex characters)
- Provides 16.7 million possible IDs (2^24)
- Cryptographically secure, no seeding required

#### Collision Handling

- On collision: retry silently up to 5 times
- If still colliding after 5 retries: return error
- Collisions are practically impossible at expected scale (hundreds to low thousands of tasks)

#### Case Sensitivity

- IDs are case-insensitive for matching
- Normalize to lowercase on input (user can type `TICK-A3F2B7`, stored as `tick-a3f2b7`)
- Always output lowercase

### File Locations & Formats

#### Directory Structure

```
.tick/
├── tasks.jsonl      # Source of truth (git tracked)
├── cache.db         # SQLite cache (gitignored)
└── lock             # Lock file for concurrent access
```

#### JSONL Format

One JSON object per line, no trailing commas, no array wrapper:

```jsonl
{"id":"tick-a1b2","title":"Task title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-c3d4","title":"With all fields","status":"in_progress","priority":1,"description":"Details here","blocked_by":["tick-a1b2"],"parent":"tick-e5f6","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:00:00Z"}
{"id":"tick-g7h8","title":"Completed task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T16:00:00Z","closed":"2026-01-19T16:00:00Z"}
```

- Optional fields omitted when empty/null (not serialized as `null`)
- `updated` always present (set to `created` value initially)

#### SQLite Schema

```sql
CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  priority INTEGER NOT NULL DEFAULT 2,
  description TEXT,
  parent TEXT,
  created TEXT NOT NULL,
  updated TEXT NOT NULL,
  closed TEXT
);

CREATE TABLE dependencies (
  task_id TEXT NOT NULL,
  blocked_by TEXT NOT NULL,
  PRIMARY KEY (task_id, blocked_by)
);

CREATE TABLE metadata (
  key TEXT PRIMARY KEY,
  value TEXT
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_parent ON tasks(parent);
```

- `tasks` table mirrors JSONL fields (minus `blocked_by` which is normalized)
- `dependencies` table enables efficient joins for "ready" queries
- `metadata` table stores the JSONL content hash for freshness detection

### Synchronization

#### Freshness Detection

SHA256 hash-based comparison:

1. On every operation, read `tasks.jsonl` into memory
2. Compute SHA256 hash of file contents
3. Compare with hash stored in SQLite `metadata` table (key: `jsonl_hash`)
4. If mismatch: rebuild SQLite from JSONL data already in memory (no double-read)

**Trade-off accepted**: Always read full file on every operation. Acceptable for expected file sizes (<1MB, <10ms).

#### Cache Rebuild Triggers

Rebuild SQLite from JSONL when:

1. **Hash mismatch** - Primary case; JSONL was modified externally (e.g., git pull, manual edit)
2. **SQLite file missing** - Hash lookup fails; rebuild from scratch
3. **SQLite query errors** - Delete corrupted cache and rebuild
4. **Explicit command** - `tick rebuild` forces rebuild

The freshness check is the gatekeeper. Everything else is error recovery.

### Write Operations

#### Mutation Flow

All mutations (create, update, delete) follow this sequence:

1. Acquire exclusive lock on `.tick/lock` file
2. Read `tasks.jsonl`, compute hash, check freshness (rebuild if stale)
3. Apply mutation in memory
4. Write complete new content to temp file
5. `fsync` temp file (flush to disk)
6. `os.Rename(temp, tasks.jsonl)` - atomic on POSIX
7. Update SQLite in single transaction: apply mutation + store new hash
8. Release lock

#### Atomic Rewrite Pattern

Always use full file rewrite for all operations (no append-only mode):

- **Why**: Atomic rename guarantees complete file or no change; append can leave incomplete lines
- **How**: Write to temp file → fsync → rename
- **Performance**: ~500 tasks × ~200 bytes = 100KB; trivially fast

#### Partial Failure Handling

**JSONL-first, SQLite is expendable**:

- If JSONL write succeeds but SQLite fails: log warning, continue
- Next read will detect hash mismatch and rebuild SQLite automatically
- Trade-off accepted: SQLite failure costs one rebuild on next read

The hash update is part of the same SQLite transaction as the data update:
```sql
BEGIN;
INSERT INTO tasks (...) VALUES (...);
UPDATE metadata SET jsonl_hash = 'new_hash';
COMMIT;
```

#### File Locking

Use `github.com/gofrs/flock` for file locking:

- **Write operations**: Exclusive lock (blocks other readers and writers)
- **Lock file**: `.tick/lock` (separate from data files)
- Prevents concurrent access corruption (learned from Taskwarrior issues)

### Read Operations

#### Query Flow

All read operations (list, show, ready, etc.) follow this sequence:

1. Acquire shared lock on `.tick/lock` file (allows concurrent reads)
2. Read `tasks.jsonl`, compute hash, check freshness
3. If stale: rebuild SQLite from JSONL data already in memory
4. Query SQLite for requested data
5. Release lock

#### Shared Locking

- **Read operations**: Shared lock (allows other readers, blocks writers)
- Multiple concurrent reads are safe
- A pending write will wait for all readers to finish

## Dependencies

Prerequisites that must exist before implementation can begin:

### Required

None. This is the foundational data layer that other specifications depend on.

### External Libraries

| Library | Purpose |
|---------|---------|
| `github.com/gofrs/flock` | File locking for concurrent access safety |
| `github.com/mattn/go-sqlite3` | SQLite driver for Go |

### Notes

- All other dependencies are Go stdlib (`crypto/rand`, `os`, `encoding/json`, `crypto/sha256`)
- This specification can be implemented independently before CLI or workflow features
