---
topic: tick-core
status: planning
format: local-markdown
specification: ../specification/tick-core.md
spec_commit: b74fff5d638e8cb3a13a21b7c01b83bb1821f7ce
created: 2026-01-27
updated: 2026-01-30
planning:
  phase: 1
  task: 4
---

# Plan: Tick Core

## Overview

**Goal**: Build the foundational data layer and CLI for Tick — a minimal, deterministic task tracker for AI coding agents using JSONL as git-committed source of truth and SQLite as auto-rebuilding cache.

**Done when**:
- All CLI commands implemented and working end-to-end
- JSONL / SQLite dual-storage with automatic freshness detection
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

## Phases

### Phase 1: Walking Skeleton — Init, Create, List, Show
status: approved
approved_at: 2026-01-30

**Goal**: Prove the dual-storage architecture end-to-end. A user can initialize a project, create tasks, and view them — threading through JSONL storage, SQLite cache, freshness detection, ID generation, file locking, and CLI.
**Why this order**: Foundation. Every subsequent phase builds on this data layer and command infrastructure. Must validate architecture first.

**Acceptance**:
- [ ] `tick init` creates .tick/ directory with empty tasks.jsonl
- [ ] `tick create` writes to JSONL, rebuilds SQLite, returns task details
- [ ] `tick list` shows all tasks (basic output, no filters)
- [ ] `tick show <id>` displays full task details
- [ ] Deleting cache.db and running any read rebuilds it automatically
- [ ] Concurrent access protected by file locking

#### Tasks
| ID            | Name                                  | Edge Cases                                                            | Status   |
|---------------|---------------------------------------|-----------------------------------------------------------------------|----------|
| tick-core-1-1 | Task model & ID generation            | ID collision retry, title limits, priority range, whitespace trimming | authored |
| tick-core-1-2 | JSONL storage with atomic writes      | empty file, malformed lines, optional fields omitted                  | authored |
| tick-core-1-3 | SQLite cache with freshness detection | missing cache.db, corrupted cache, hash in metadata table             | authored |
| tick-core-1-4 | Storage engine with file locking      | lock timeout, concurrent reads, stale cache during write              | authored |
| tick-core-1-5 | CLI framework & tick init             | already initialized, no parent directory                              | pending  |
| tick-core-1-6 | tick create command                   | missing title, empty title, all optional fields                       | pending  |
| tick-core-1-7 | tick list & tick show commands        | no tasks (empty list), task ID not found                              | pending  |

---

### Phase 2: Task Lifecycle — Status Transitions & Update
status: approved
approved_at: 2026-01-30

**Goal**: Track tasks through their full lifecycle. Start, complete, cancel, and reopen tasks with validated transitions. Update task fields.
**Why this order**: Tasks exist from Phase 1 but can't change state. This makes tick usable for actual work tracking.

**Acceptance**:
- [ ] `tick start` transitions open → in_progress
- [ ] `tick done` transitions open/in_progress → done (sets closed timestamp)
- [ ] `tick cancel` transitions open/in_progress → cancelled (sets closed timestamp)
- [ ] `tick reopen` transitions done/cancelled → open (clears closed timestamp)
- [ ] Invalid transitions return errors (e.g., start on a done task)
- [ ] `tick update` modifies title, description, priority
- [ ] `updated` timestamp refreshed on every mutation

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|

---

### Phase 3: Hierarchy & Dependencies
status: approved
approved_at: 2026-01-30

**Goal**: Full workflow intelligence. Parent/child grouping, dependency blocking, cycle detection, and the ready/blocked queries that agents depend on.
**Why this order**: Requires working tasks with status transitions (Phase 2). This is the core value — answering "what should I work on next?"

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

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|

---

### Phase 4: Output Formats
status: approved
approved_at: 2026-01-30

**Goal**: All commands produce correct output for both agents (TOON) and humans (aligned tables), with TTY auto-detection and format override flags.
**Why this order**: Functionality complete from Phases 1-3. Now format output correctly for all consumers. Prior phases can use minimal/basic output.

**Acceptance**:
- [ ] TTY detection selects human-readable (TTY) vs TOON (non-TTY) automatically
- [ ] TOON output matches spec format for all commands (list, show, stats)
- [ ] Human-readable output uses aligned columns, no borders/colors
- [ ] --toon, --pretty, --json flags override auto-detection
- [ ] --quiet suppresses non-essential output (create/update → ID only)
- [ ] --verbose adds debug detail
- [ ] JSON output available via --json flag
- [ ] Empty results handled correctly per format

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|

---

### Phase 5: Diagnostics & Stats
status: approved
approved_at: 2026-01-30

**Goal**: Health checking, statistics, and cache management tools for debugging and project oversight.
**Why this order**: Requires all features in place to diagnose and report on them.

**Acceptance**:
- [ ] `tick stats` shows counts by status, priority, ready/blocked
- [ ] `tick doctor` detects orphaned children, done parents with open children, non-existent references, and other integrity issues
- [ ] `tick rebuild` forces full SQLite rebuild from JSONL
- [ ] Doctor output is actionable (describes what's wrong and what to check)

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|

---

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
| 2026-01-30 | Migrated to updated plan format (Plan Index + task files) |
