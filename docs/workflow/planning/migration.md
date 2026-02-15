---
topic: migration
status: concluded
format: local-markdown
specification: ../specification/migration.md
spec_commit: d75ee0ec089a456e34c6b10585da4cf18922a0a5
created: 2026-01-31
updated: 2026-01-31
external_dependencies:
  - topic: tick-core
    description: Migration inserts tasks into tick's data store. Cannot create tasks without the data layer, schema, and write operations.
    state: resolved
    task_id: tick-core-1-4
planning:
  phase: 3
  task: 4
---

# Plan: Migration

## Overview

**Goal**: Import task data from other tools via `tick migrate --from <provider>`, starting with beads as the first provider.

**Done when**:
- `tick migrate --from beads` imports tasks into tick's data store
- Dry-run and pending-only modes work
- Errors are handled gracefully with continue-on-error and failure reporting
- Unknown providers produce helpful error messages

**Key Decisions** (from specification):
- One-time import, not sync — run once, delete the old system
- Append-only — adds to existing tick data, never modifies or deletes
- User responsibility for duplicates from re-running
- Plugin architecture — providers are self-contained; tick core doesn't manage credentials
- Continue on error — report failures at end, no rollback

## Phases

### Phase 1: Walking Skeleton - End-to-End Beads Migration
status: approved
approved_at: 2026-01-31

**Goal**: A working `tick migrate --from beads` command that reads beads task files, normalizes them to tick's schema, inserts them via tick-core, and prints per-task output with a summary line.
**Why this order**: Must establish the end-to-end flow first — CLI entry point, provider contract, beads provider, normalization, insertion, and output. Every subsequent feature (dry-run, pending-only, error handling) builds on this working pipeline.

**Acceptance**:
- [ ] `tick migrate --from beads` reads beads task files from the filesystem and creates corresponding tasks in tick's data store
- [ ] Provider contract/interface exists that beads (and future providers) implement
- [ ] Beads provider maps all available fields to tick equivalents; missing fields use sensible defaults
- [ ] Each imported task is printed as it is processed (checkmark + title format)
- [ ] Summary line printed at end showing count of imported tasks
- [ ] Imported tasks are retrievable via `tick list` after migration completes

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| migration-1-1 | Provider Contract & Migration Types | none | authored |
| migration-1-2 | Beads Provider - Read & Map | missing .beads dir, missing issues.jsonl, empty file, malformed JSON lines, missing title, discarded fields, status mapping, priority mapping | authored |
| migration-1-3 | Migration Engine - Iterate & Insert | empty provider (zero tasks), insertion failure | authored |
| migration-1-4 | Migration Output - Per-Task & Summary | zero tasks imported, long titles | authored |
| migration-1-5 | CLI Command - tick migrate --from | missing --from flag | authored |

---

### Phase 2: Flags, Error Handling, and Edge Cases
status: approved
approved_at: 2026-01-31

**Goal**: Add --dry-run mode, --pending-only filter, continue-on-error with failure reporting, and unknown provider error handling to complete the specification.
**Why this order**: All features in this phase are variations or hardening of the working pipeline established in Phase 1. They require the end-to-end flow to exist before they can be layered on.

**Acceptance**:
- [ ] `--dry-run` previews what would be imported without writing to tick's data store
- [ ] `--pending-only` imports only non-completed tasks from the source
- [ ] When a task fails to import, migration continues processing remaining tasks and reports failures at end
- [ ] Failure detail section lists each failed task with its reason
- [ ] Summary line shows both imported and failed counts (e.g., "Done: 3 imported, 1 failed")
- [ ] Unknown provider name produces an error listing available providers

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| migration-2-1 | Engine Continue-on-Error | all tasks fail insertion, mixed validation and insertion failures | authored |
| migration-2-2 | Presenter Failure Output | failure with empty title, failure reason with special characters, zero failures (detail section omitted) | authored |
| migration-2-3 | Dry-Run Mode | dry-run with zero tasks, dry-run combined with --pending-only | authored |
| migration-2-4 | Pending-Only Filter | all tasks completed (zero remaining), no completed tasks (filter is no-op), mixed statuses | authored |
| migration-2-5 | Unknown Provider Available Listing | single provider in registry, multiple providers in registry | authored |

---

### Phase 3: Analysis (cycle 1 findings)
status: approved

**Goal**: Address findings from implementation analysis cycle 1.

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| migration-3-1 | Replace manual presenter calls in RunMigrate with Present function | — | authored |
| migration-3-2 | Surface beads provider parse/validation errors as failed results instead of silently dropping | — | authored |
| migration-3-3 | Consolidate inconsistent empty-title fallback strings | — | authored |
| migration-3-4 | Use task.Status type and constants instead of raw status strings | — | authored |

---

## Log

| Date | Change |
|------|--------|
| 2026-01-31 | Created from specification |
