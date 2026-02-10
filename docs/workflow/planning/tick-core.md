---
topic: tick-core
status: concluded
format: local-markdown
specification: ../specification/tick-core.md
spec_commit: 4a3a40d9415de8e1bb3a1ee376efd0bef2af0dd3
created: 2026-01-27
updated: 2026-02-09
external_dependencies: []
planning:
  phase: 5
  task: 2
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
| tick-core-1-5 | CLI framework & tick init             | already initialized, no parent directory                              | authored |
| tick-core-1-6 | tick create command                   | missing title, empty title, all optional fields                       | authored |
| tick-core-1-7 | tick list & tick show commands        | no tasks (empty list), task ID not found                              | authored |

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
| ID            | Name                                      | Edge Cases                                                                                                                                                                                                                      | Status  |
|---------------|-------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| tick-core-2-1 | Status transition validation logic        | all 4 valid transitions, all invalid transitions (start on done/cancelled/in_progress, done on done/cancelled, cancel on done/cancelled, reopen on open/in_progress), closed timestamp set/cleared, updated timestamp refreshed  | authored |
| tick-core-2-2 | tick start, done, cancel, reopen commands | output format, --quiet suppresses output, task ID not found, case-insensitive ID, exit code 1 on error                                                                                                                          | authored |
| tick-core-2-3 | tick update command                       | --title/--description/--priority flags, no flags error, cannot change status/id/created/blocked_by, clear description with empty string, title validation, priority validation, updated timestamp, task not found, --quiet       | authored |

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
- [ ] `tick list --parent <id>` scopes to descendants of the specified task (recursive)
- [ ] Cancelled tasks unblock their dependents

#### Tasks
| ID            | Name                                                              | Edge Cases                                                                                                                                                                                         | Status  |
|---------------|-------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| tick-core-3-1 | Dependency validation — cycle detection & child-blocked-by-parent | direct cycle, 2-node cycle (A→B→A), 3+ node cycle with path in error, child blocked by own parent, cross-hierarchy deps allowed, parent blocked by child allowed                                   | authored |
| tick-core-3-2 | tick dep add & tick dep rm commands                               | non-existent IDs, duplicate dep, removing non-existent dep, self-reference, cycle introduced by add, child-blocked-by-parent by add, case-insensitive IDs, --quiet                                 | authored |
| tick-core-3-3 | Ready query & tick ready command                                  | open unblocked no-children (ready), open blocker (not ready), all blockers done/cancelled (ready), parent with open children (not ready), deep nesting, in_progress/done/cancelled excluded, empty  | authored |
| tick-core-3-4 | Blocked query, tick blocked & cancel-unblocks-dependents          | blocked by open/in_progress dep, parent with open children, cancel unblocks dependents, multiple dependents unblocked, empty result                                                                | authored |
| tick-core-3-5 | tick list filter flags — --ready, --blocked, --status, --priority | reuses ready/blocked queries, invalid values error, combining filters, no matches, --quiet IDs only                                                                                                | authored |
| tick-core-3-6 | Parent scoping — --parent flag with recursive descendant CTE      | non-existent parent ID, parent with no descendants (empty result), deep nesting (3+ levels), --parent combined with --status/--priority/--ready/--blocked, parent task itself excluded, --parent on tick ready vs tick blocked vs tick list | authored |

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
| ID            | Name                                                   | Edge Cases                                                                                                       | Status  |
|---------------|--------------------------------------------------------|------------------------------------------------------------------------------------------------------------------|---------|
| tick-core-4-1 | Formatter abstraction & TTY-based format selection     | TTY vs non-TTY auto-detection, flag overrides, conflicting flags, verbose propagation                             | authored |
| tick-core-4-2 | TOON formatter — list, show, stats output              | zero count empty results, multi-section show, field escaping, omitted vs empty sections, multiline description    | authored |
| tick-core-4-3 | Human-readable formatter — list, show, stats output    | column alignment, long titles, empty results, omitted sections, stats zero counts, priority labels                | authored |
| tick-core-4-4 | JSON formatter — list, show, stats output              | null vs omitted fields, empty arrays vs absent keys, snake_case keys, empty list as []                            | authored |
| tick-core-4-5 | Integrate formatters into all commands                 | create/update full task output, transition output, dep confirmation, --quiet overrides format, empty across formats | authored |
| tick-core-4-6 | Verbose output & edge case hardening                   | --verbose debug detail to stderr, verbose + quiet interaction, no verbose leakage into pipes                      | authored |

---

### Phase 5: Stats & Cache Management
status: approved
approved_at: 2026-01-30

**Goal**: Statistics and cache management tools for project oversight and debugging. (Doctor/validation deferred to separate `doctor-validation` specification.)
**Why this order**: Requires all features in place to report on them.

**Acceptance**:
- [ ] `tick stats` shows counts by status, priority, ready/blocked
- [ ] `tick rebuild` forces full SQLite rebuild from JSONL

#### Tasks
| ID            | Name                | Edge Cases                                                                                        | Status  |
|---------------|---------------------|---------------------------------------------------------------------------------------------------|---------|
| tick-core-5-1 | tick stats command   | zero counts, all statuses present, priority breakdown P0-P4, ready/blocked counts, empty project  | authored |
| tick-core-5-2 | tick rebuild command | missing cache.db, valid cache overwritten, concurrent access during rebuild, confirmation output   | authored |

---

### Phase 6: Analysis Fixes — Validation, Deduplication & Compliance
status: pending

**Goal**: Address gaps found during implementation analysis: missing dependency validation on write paths, duplicated logic, spec compliance issues, and missing integration test coverage.
**Why this order**: All core functionality is implemented. These tasks fix correctness gaps, reduce duplication, and align with the specification.

**Acceptance**:
- [ ] All write paths that modify blocked_by arrays enforce cycle detection and child-blocked-by-parent validation
- [ ] Rebuild logic flows through Store abstraction
- [ ] Cache freshness/recovery has a single code path
- [ ] Formatter duplication consolidated, Unicode arrow matches spec
- [ ] Shared helpers extracted for --blocks application and ID parsing
- [ ] End-to-end workflow integration test passes
- [ ] Child-blocked-by-parent error message matches spec exactly

#### Tasks
| ID            | Name                                                                     | Status  |
|---------------|--------------------------------------------------------------------------|---------|
| tick-core-6-1 | Add dependency validation to create and update --blocked-by/--blocks      | pending |
| tick-core-6-2 | Move rebuild logic behind Store abstraction                               | pending |
| tick-core-6-3 | Consolidate cache freshness/recovery logic                                | pending |
| tick-core-6-4 | Consolidate formatter duplication and fix Unicode arrow                   | pending |
| tick-core-6-5 | Extract shared helpers for --blocks application and ID parsing            | pending |
| tick-core-6-6 | Add end-to-end workflow integration test                                  | pending |
| tick-core-6-7 | Add explanatory second line to child-blocked-by-parent error              | pending |

---

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
| 2026-01-30 | Phase 5: removed doctor (separate spec), renamed to Stats & Cache Management |
| 2026-01-30 | Plan review complete, status concluded |
| 2026-02-10 | Phase 6 added: 7 analysis tasks from cycle 1 implementation analysis |
