---
topic: tick-core
status: in-progress
date: 2026-01-27
format: local-markdown
specification: tick-core.md
---

# Implementation Plan: Tick Core

## Overview

**Goal**: Build the foundational data layer and CLI for Tick — a minimal, deterministic task tracker for AI coding agents using JSONL as git-committed source of truth and SQLite as auto-rebuilding cache.

**Done when**:
- All CLI commands implemented and working end-to-end
- JSONL ↔ SQLite dual-storage with automatic freshness detection
- Deterministic queries (same input = same output)
- Sub-100ms operations
- Comprehensive test coverage

**Key Decisions** (from specification):
- JSONL is source of truth; SQLite is expendable cache
- SHA256 hash-based freshness detection (always read full file, compare hash)
- Atomic file rewrite (temp + fsync + rename) for all mutations
- File locking via `gofrs/flock` (shared reads, exclusive writes)
- ID format: `tick-{6 hex chars}` from `crypto/rand`
- TTY detection for output format (TOON for agents, human-readable for terminals)
- No config file, no daemons, no hooks

## Architecture

- **Storage**: JSONL file (`.tick/tasks.jsonl`) — one JSON object per line, git-tracked
- **Cache**: SQLite (`.tick/cache.db`) — gitignored, auto-rebuilt from JSONL on hash mismatch
- **Locking**: File lock (`.tick/lock`) — exclusive for writes, shared for reads, 5s timeout
- **CLI**: Go binary with subcommands (init, create, update, start, done, cancel, reopen, list, show, ready, blocked, dep, stats, doctor, rebuild)
- **Output**: TTY-detected — TOON (pipe/redirect), human-readable (terminal), JSON (--json flag)

## Phases

Each phase is independently testable with clear acceptance criteria.
Each task is a single TDD cycle: write test → implement → commit.

---

### Phase 1: Walking Skeleton — Init, Create, List, Show

**Goal**: Prove the dual-storage architecture end-to-end. A user can initialize a project, create tasks, and view them — threading through JSONL storage, SQLite cache, freshness detection, ID generation, file locking, and CLI.

**Acceptance**:
- [ ] `tick init` creates .tick/ directory with empty tasks.jsonl
- [ ] `tick create` writes to JSONL, rebuilds SQLite, returns task details
- [ ] `tick list` shows all tasks (basic output, no filters)
- [ ] `tick show <id>` displays full task details
- [ ] Deleting cache.db and running any read rebuilds it automatically
- [ ] Concurrent access protected by file locking

**Tasks**:

1. **Task model & ID generation**

   **Problem**: Tick needs a core data structure representing tasks and a deterministic ID format. Without this, no other component can operate.

   **Solution**: Define a Go struct with all 10 task fields, field validation logic, and a `tick-{6 hex}` ID generator using `crypto/rand` with collision retry.

   **Outcome**: A validated task model that can be constructed, validated, and assigned unique IDs — fully unit-testable without any storage layer.

   **Do**:
   - Define `Task` struct with fields: `id`, `title`, `status`, `priority`, `description`, `blocked_by`, `parent`, `created`, `updated`, `closed`
   - Implement `Status` type as string enum with constants: `open`, `in_progress`, `done`, `cancelled`
   - Implement ID generation: 3 random bytes from `crypto/rand` → 6 lowercase hex chars → prefix with `tick-`
   - ID collision retry: accept a function to check existence, retry up to 5 times, error after that
   - Normalize IDs to lowercase on input (case-insensitive matching)
   - Validate title: required, non-empty, max 500 chars, no newlines, trim whitespace
   - Validate priority: integer 0-4, reject out of range
   - Validate `blocked_by`: no self-references (cycle detection is Phase 3)
   - Validate `parent`: no self-references
   - All timestamps use ISO 8601 UTC format (`YYYY-MM-DDTHH:MM:SSZ`)

   **Acceptance Criteria**:
   - [ ] Task struct has all 10 fields with correct Go types
   - [ ] ID format matches `tick-{6 hex chars}` pattern
   - [ ] IDs are generated using `crypto/rand`
   - [ ] Collision retry works up to 5 times then errors
   - [ ] Input IDs are normalized to lowercase
   - [ ] Title validation enforces non-empty, max 500 chars, no newlines, trims whitespace
   - [ ] Priority validation rejects values outside 0-4
   - [ ] Self-references in `blocked_by` and `parent` are rejected
   - [ ] Timestamps are ISO 8601 UTC

   **Tests**:
   - `"it generates IDs matching tick-{6 hex} pattern"`
   - `"it retries on collision up to 5 times"`
   - `"it errors after 5 collision retries"`
   - `"it normalizes IDs to lowercase"`
   - `"it rejects empty title"`
   - `"it rejects title exceeding 500 characters"`
   - `"it rejects title with newlines"`
   - `"it trims whitespace from title"`
   - `"it rejects priority outside 0-4"`
   - `"it rejects self-reference in blocked_by"`
   - `"it rejects self-reference in parent"`
   - `"it sets default priority to 2 when not specified"`
   - `"it sets created and updated timestamps to current UTC time"`

   **Context**:
   > Task schema has 10 fields. Optional fields (`description`, `blocked_by`, `parent`, `closed`) should use Go zero values/nil. Status enum: `open`, `in_progress`, `done`, `cancelled`. Priority is integer 0 (highest) to 4 (lowest), default 2.

---

### Phase 2: Task Lifecycle — Status Transitions & Update

**Goal**: Track tasks through their full lifecycle. Start, complete, cancel, and reopen tasks with validated transitions. Update task fields.

**Acceptance**:
- [ ] `tick start` transitions open → in_progress
- [ ] `tick done` transitions open/in_progress → done (sets closed timestamp)
- [ ] `tick cancel` transitions open/in_progress → cancelled (sets closed timestamp)
- [ ] `tick reopen` transitions done/cancelled → open (clears closed timestamp)
- [ ] Invalid transitions return errors (e.g., start on a done task)
- [ ] `tick update` modifies title, description, priority
- [ ] `updated` timestamp refreshed on every mutation

---

### Phase 3: Hierarchy & Dependencies

**Goal**: Full workflow intelligence. Parent/child grouping, dependency blocking, cycle detection, and the ready/blocked queries that agents depend on.

**Acceptance**:
- [ ] `tick create --parent <id>` sets organizational hierarchy
- [ ] `tick create --blocked-by <ids>` sets dependencies
- [ ] `tick dep add/rm` manages dependencies after creation
- [ ] Cycle detection rejects circular dependencies at write time
- [ ] Child-blocked-by-parent rejected at write time
- [ ] `tick ready` returns only tasks that are open, unblocked, and have no open children
- [ ] `tick blocked` returns open tasks that are not ready
- [ ] `tick list` supports --ready, --blocked, --status, --priority filters
- [ ] Cancelled tasks unblock their dependents

---

### Phase 4: Output Formats

**Goal**: All commands produce correct output for both agents (TOON) and humans (aligned tables), with TTY auto-detection and format override flags.

**Acceptance**:
- [ ] TTY detection selects human-readable (TTY) vs TOON (non-TTY) automatically
- [ ] TOON output matches spec format for all commands (list, show, stats)
- [ ] Human-readable output uses aligned columns, no borders/colors
- [ ] --toon, --pretty, --json flags override auto-detection
- [ ] --quiet suppresses non-essential output (create/update → ID only)
- [ ] --verbose adds debug detail
- [ ] JSON output available via --json flag
- [ ] Empty results handled correctly per format

---

### Phase 5: Diagnostics & Stats

**Goal**: Health checking, statistics, and cache management tools for debugging and project oversight.

**Acceptance**:
- [ ] `tick stats` shows counts by status, priority, ready/blocked
- [ ] `tick doctor` detects orphaned children, done parents with open children, non-existent references, and other integrity issues
- [ ] `tick rebuild` forces full SQLite rebuild from JSONL
- [ ] Doctor output is actionable (describes what's wrong and what to check)

---

## Edge Cases

| Edge Case | Solution | Phase.Task | Test |
|-----------|----------|------------|------|
| ID collision | Retry up to 5 times, then error | 1.1 | `"it retries on collision up to 5 times"` |
| Title > 500 chars | Reject with error | 1.1 | `"it rejects title exceeding 500 characters"` |
| Empty JSONL file | Return empty task list | 1.2 | `"it returns empty list for empty file"` |
| Missing cache.db | Rebuild from JSONL | 1.3 | `"it rebuilds when cache.db is missing"` |
| Corrupted cache.db | Delete and rebuild | 1.3 | `"it deletes and rebuilds corrupted cache"` |
| Lock timeout | Error after 5s | 1.4 | `"it errors after lock timeout"` |
| Already initialized | Error "already initialized" | 1.5 | `"it errors when .tick/ already exists"` |
| Task not found | Error with ID | 1.7 | `"it errors when task ID not found"` |
| Invalid status transition | Error with current/requested status | 2.x | TBD |
| Circular dependency | Reject at write time | 3.x | TBD |
| Child blocked by parent | Reject at write time | 3.x | TBD |
| Orphaned children | Doctor flags, no auto-fix | 5.x | TBD |
| Deep nesting | Leaf-only rule handles naturally | 3.x | TBD |

## Testing Strategy

**Unit**: Task model validation, ID generation, JSONL parsing, SQLite cache operations, freshness detection, cycle detection
**Integration**: Full command flows (init → create → list → show), cache rebuild after deletion, lock contention
**Manual**: TTY vs pipe output verification

## Data Models

### JSONL Format
One JSON object per line:
```jsonl
{"id":"tick-a1b2","title":"Task title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
```
Optional fields omitted when empty/null.

### SQLite Schema
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

## Internal Dependencies

- Phase 1 is standalone (walking skeleton)
- Phase 2 depends on Phase 1 (needs task model, storage, CLI framework)
- Phase 3 depends on Phase 2 (needs status transitions for ready/blocked logic)
- Phase 4 depends on Phases 1-3 (formats output for all existing commands)
- Phase 5 depends on Phases 1-3 (diagnoses and reports on all features)

## External Dependencies

None. This is the foundational data layer that other specifications depend on.

### External Libraries

| Library | Purpose |
|---------|---------|
| `github.com/gofrs/flock` | File locking for concurrent access safety |
| `github.com/mattn/go-sqlite3` | SQLite driver for Go |
| `github.com/toon-format/toon-go` | TOON encoding/decoding for agent output |

### Optional Libraries

| Library | Purpose |
|---------|---------|
| `github.com/natefinch/atomic` | Atomic file writes (alternative to hand-rolling) |

## Log

| Date | Change |
|------|--------|
| 2026-01-27 | Created from specification |
