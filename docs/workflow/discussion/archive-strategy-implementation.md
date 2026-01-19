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

- [x] Should archiving be manual-only or support automatic triggers?
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

## Should archiving be manual-only or support automatic triggers?

### Context
Foundational decision affecting system predictability. Agents script against tick's behavior - if tasks silently move, workflows could break.

### Options Considered

**Option A: Manual-only (`tick archive`)**
- Pros: Completely predictable, user controls when history moves, no surprises for agents
- Cons: Requires user discipline, file can grow unbounded if ignored

**Option B: Auto-archive on threshold**
- Pros: Zero maintenance, file stays lean automatically
- Cons: Agents might see tasks "disappear" between runs, harder to reason about

**Option C: Manual with advisory warnings**
- Pros: Predictable like manual, but proactive guidance prevents unbounded growth
- Cons: Slightly more complexity in output handling

### Journey

Started with manual-only as the clear winner for agent-first design. The question became: how do we prevent unbounded growth without automatic intervention?

The hybrid emerged: manual archiving, but tick monitors thresholds and advises when they're exceeded. For agent output, include an `advisories` array (footer-style, at bottom of output) so agents can relay the message to users. No behavior change, just information.

**Threshold type debate:**
- Count-based: "100+ done tasks" - concrete but 100 felt low
- Size-based: "tasks.jsonl > X" - more performance-relevant
- Age-based: "done tasks older than 30 days" - rejected as too aggressive for short projects

Settled on dual thresholds (count AND size) to cover both normal and meaty-task scenarios.

**Reverse-engineering defaults** from task schema analysis:
- Minimal task: ~150 bytes
- Typical task: ~300-400 bytes
- Meaty task with description: ~500-1000 bytes
- Go parses JSONL at ~100MB/s, so 5MB still parses in <100ms

### Decision

**Manual-only archiving with advisory warnings.**

- Archiving triggered only by explicit `tick archive` command
- Tick monitors and warns when thresholds exceeded
- Warnings appear in `advisories` array (bottom of output) for agents to relay
- Dual thresholds, configurable as top-level constants:

```go
const (
    ArchiveAdvisoryDoneCount = 500
    ArchiveAdvisorySizeBytes = 2 * 1024 * 1024 // 2MB
)
```

**Rationale**: 500 done tasks is substantial history - most projects won't hit it quickly. 2MB is well before noticeable performance impact. Advisory-only preserves predictability while preventing neglect.

---

