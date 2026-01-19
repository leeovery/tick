# Discussion: Archive Strategy Implementation

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick needs a strategy for handling completed tasks over time. As projects grow, the task file accumulates done tasks that slow queries and clutter output. The research phase established a baseline approach: keep done tasks in main file by default, with optional manual archiving.

Key constraints:
- Agent-first: Agents need fast, predictable queries
- JSONL as source of truth: Archive must preserve this
- SQLite cache for queries: Indexing strategy matters
- Human usability: History should be searchable when needed

### References

- [Research: Archive Strategy](../research/exploration.md) (lines 361-398)

## Questions

- [ ] Should archiving be manual-only or support automatic triggers?
      - Auto-archive on threshold (count/age)?
      - Always manual for predictability?
- [ ] How should the SQLite cache handle archived tasks?
      - Separate database per file?
      - Single database with archive flag?
      - No indexing of archive (scan on demand)?
- [ ] Is an unarchive command needed?
      - What if a task was archived by mistake?
      - What if an archived task needs to be reopened?
- [ ] What metadata should track archive operations?
      - Timestamp of archival?
      - Who/what triggered it?
- [ ] How do dependencies interact with archiving?
      - Can a task with archived dependents be archived?
      - What about active tasks blocked by archived tasks?

---

