---
status: complete
created: 2026-02-28
cycle: 1
phase: Plan Integrity Review
topic: cli-enhancements
---

# Review Tracking: cli-enhancements - Integrity

## Findings

### 1. Note add/remove subcommands use NormalizeID instead of ResolveID

**Severity**: Critical
**Plan Reference**: Phase 4 / cli-enhancements-4-7 (tick-a4c883), cli-enhancements-4-8 (tick-7402d4)
**Category**: Task Self-Containment
**Change Type**: update-task

**Details**:
Both note subcommand tasks instruct the implementer to use `task.NormalizeID` for resolving the task ID argument. Phase 1 establishes that ALL commands accepting task IDs must resolve through `store.ResolveID()`. The specification states partial ID matching "applies everywhere an ID is accepted." Using NormalizeID would mean `tick note add a3f "text"` fails with a not-found error instead of resolving the partial ID. This contradicts the Phase 1 contract and would produce a user-facing bug.

**Current** (tick-a4c883 note add, relevant Do section):
```
  - Open store, call store.Mutate:
    - Find task by ID (use task.NormalizeID)
```

**Proposed** (tick-a4c883 note add, relevant Do section):
```
  - Resolve task ID via store.ResolveID (partial ID matching from Phase 1)
  - Open store, call store.Mutate:
    - Find task by resolved full ID
```

**Current** (tick-7402d4 note remove, relevant Do section):
```
  - Open store, call store.Mutate:
    - Find task by ID (use task.NormalizeID)
```

**Proposed** (tick-7402d4 note remove, relevant Do section):
```
  - Resolve task ID via store.ResolveID (partial ID matching from Phase 1)
  - Open store, call store.Mutate:
    - Find task by resolved full ID
```

**Resolution**: Fixed
**Notes**: Both tasks updated to use store.ResolveID. Acceptance criteria clarified to reference ResolveID error messaging.

---

### 2. Four Phase 4 tasks missing required Outcome field

**Severity**: Important
**Plan Reference**: Phase 4 / cli-enhancements-4-1 (tick-e7bb22), cli-enhancements-4-2 (tick-80ad02), cli-enhancements-4-3 (tick-6d5863), cli-enhancements-4-4 (tick-4b4e4b)
**Category**: Task Template Compliance
**Change Type**: update-task

**Details**:
The task template requires Problem, Solution, and Outcome as mandatory fields. Four Phase 4 refs tasks have Problem and Solution but no Outcome statement. The Outcome field defines what success looks like -- without it, an implementer has no verifiable end state to target. All other tasks in the plan include Outcome.

**Current** (tick-e7bb22, description after Solution):
```
  Do:
```

**Proposed** (tick-e7bb22, insert Outcome before Do):
```
  Outcome: Task struct includes Refs field that round-trips through JSON correctly, with validation functions covering all edge cases including commas, whitespace, length boundaries, and deduplication.

  Do:
```

**Current** (tick-80ad02, description after Solution):
```
  Do:
```

**Proposed** (tick-80ad02, insert Outcome before Do):
```
  Outcome: After cache rebuild, every task's refs are stored in task_refs junction table. Table cleared and repopulated on each rebuild. Schema creation idempotent.

  Do:
```

**Current** (tick-6d5863, description after Solution):
```
  Do:
```

**Proposed** (tick-6d5863, insert Outcome before Do):
```
  Outcome: Users can attach refs on create, replace on update, and clear with --clear-refs. Validation errors for invalid/empty refs. Mutual exclusivity enforced between --refs and --clear-refs.

  Do:
```

**Current** (tick-4b4e4b, description after Solution):
```
  Do:
```

**Proposed** (tick-4b4e4b, insert Outcome before Do):
```
  Outcome: tick show <id> displays refs in all three formats. No refs produces omitted section (pretty/toon) or empty array (JSON). Refs not shown in list output.

  Do:
```

**Resolution**: Fixed
**Notes**: Outcome fields added to all four tasks.

---

### 3. Phase 3 display task ordered before create/update flags task

**Severity**: Minor
**Plan Reference**: Phase 3 / cli-enhancements-3-3 (tick-d17558) and cli-enhancements-3-4 (tick-f713ec)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 3-3 (tags display in show output) is ordered before task 3-4 (create/update with --tags flags). The display task queries tags from task_tags and renders them in formatters, but the create/update task is what lets users actually assign tags through the CLI. While 3-3 is technically testable independently (tests can set up JSONL data with tags directly), the ordering is counterintuitive -- you would naturally build the ability to set tags before building the ability to display them. However, since the display task queries from SQLite (populated by Cache.Rebuild from JSONL), it only depends on the junction table (3-2) being in place, not on CLI flags. The current order works but may confuse an implementer.

This is a minor style issue. The natural order still produces correct results since both tasks depend on 3-1 (model) and 3-2 (junction table) which come first. No change needed unless the team prefers a more intuitive order.

**Current** (plan.md Phase 3 task table, rows 3-3 and 3-4):
```
| cli-enhancements-3-3 | Tags display in show output and all formatters | task with no tags, task with 10 tags | authored | tick-d17558 |
| cli-enhancements-3-4 | Create and update with --tags and --clear-tags flags | --tags and --clear-tags together, empty --tags value, --tags with duplicates, --clear-tags on task with no tags | authored | tick-f713ec |
```

**Proposed** (plan.md Phase 3 task table, swap rows 3-3 and 3-4):
```
| cli-enhancements-3-3 | Create and update with --tags and --clear-tags flags | --tags and --clear-tags together, empty --tags value, --tags with duplicates, --clear-tags on task with no tags | authored | tick-f713ec |
| cli-enhancements-3-4 | Tags display in show output and all formatters | task with no tags, task with 10 tags | authored | tick-d17558 |
```

**Resolution**: Skipped
**Notes**: Both tasks are independently testable and the current order is functional. Re-creating Tick tasks to swap creation order for a minor cosmetic concern is not worth the effort.
