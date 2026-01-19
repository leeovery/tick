# Discussion: Migration Subcommand

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick needs a way to import task data from other tools. Users migrating to tick shouldn't have to manually recreate their existing tasks. The initial focus is on beads (a task format used in claude-technical-workflows), with potential to expand to other tools later.

This discussion focuses on the high-level use case and design - implementation details will be addressed in later phases.

### References

- Prior art: beads format (claude-technical-workflows)

## Questions

- [x] What's the core use case for migration?
- [x] What should the command look like?
- [x] Which source tools should we support (and in what order)?
- [x] How should we handle data that doesn't map cleanly?
- [x] Should migration be one-time or ongoing sync?
- [x] How should we handle conflicts/duplicates?
- [x] What about error handling?
- [ ] Any other considerations?

---

## What's the core use case for migration?

### Context
Understanding who needs this and why drives the design.

### Journey

Initial scenarios considered:
1. **Existing project adoption** - Project using beads wants to switch mid-stream
2. **Workflow transition** - Teams moving from beads to tick as new standard
3. **One-time bootstrap** - New tick project pulling historical context

User clarified their actual use case: They've used beads on several projects and want to migrate them to tick before deleting beads. This is the primary scenario - **active project migration** where real work has been invested and needs preserving.

### Decision

**Primary use case**: Active project migration - preserving invested work when switching tools.

---

## What should the command look like?

### Journey

Discussed explicit vs. automatic provider discovery. User wants explicit - no magic file detection needed.

Initial thought was `tick migrate from beads` but clarified that `from` should be a flag.

### Decision

```
tick migrate --from beads
```

Also supporting:
- `--dry-run` flag to preview what would be imported

---

## Which source tools should we support (and in what order)?

### Journey

User wants to start with beads but design for extensibility. Mentioned JIRA, Linear as potential future providers - really anything where data can be pulled and processed.

### Decision

**Initial**: Beads only

**Architecture**: Plugin/strategy pattern with a contract:
```
Provider → Normalize → Insert
   ↓           ↓          ↓
 (beads,    (our        (tick
  JIRA,     contract)    data)
  Linear)
```

Each provider's responsibility:
1. Connect to source system
2. Fetch data
3. Map to normalized format (adheres to contract)

The inserter is provider-agnostic - it just receives normalized data and creates tick entries.

This allows community contributions for new providers without touching core logic.

---

## How should we handle data that doesn't map cleanly?

### Journey

Considered what fields are required vs. optional. Most project management tools have more data than tick needs, so mapping should be straightforward.

### Decision

- **Required**: Title only (bare minimum)
- **Approach**: Map all available fields from source
- **Missing data**: Use sensible defaults or leave empty

---

## Should migration be one-time or ongoing sync?

### Journey

Considered whether to support coexistence/syncing between tick and other tools.

User is clear: this is "run migrate once, delete the old thing." No syncing support. If users want to run it multiple times, that's their choice, but we won't deduplicate or sync.

### Decision

**One-time append operation**. No synchronization. No deduplication.

The migration appends to existing tick data if present. Users who run it twice get duplicates - that's their responsibility, not ours.

---

## How should we handle conflicts/duplicates?

### Journey

Discussed storing original IDs for conflict detection, but decided this adds complexity for minimal value.

Considered a simple count warning: "Found 47 existing tasks. This will add X more. Continue?" but even that may be unnecessary.

### Decision

**No conflict detection**. Keep it simple. Re-imports create duplicates. User's responsibility to manage.

---

## How should we handle errors?

### Journey

Three options considered:
1. Stop at first error
2. Skip and continue, report at end
3. Rollback everything if any fail

User wants simplicity - keep going, report at the end.

### Decision

**Continue on error, report failures at end.**

Example output:
```
Successfully imported: 52 tasks
Failed: 3 tasks

Failures:
- Task "foo": Missing required field
- Task "bar": Invalid date format
...
```

---

## User choice: what to migrate?

### Journey

User suggested offering a choice on what to import.

### Decision

Prompt or flag: **All tasks** vs **Pending only**

This handles both use cases:
- Full history preservation (all tasks including done)
- Clean slate with just active work (pending only)

