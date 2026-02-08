---
id: tick-core-5-1
phase: 5
status: completed
created: 2026-01-30
---

# tick stats command

## Goal

Implement `tick stats` — aggregate counts by status, priority, and workflow state (ready/blocked). Outputs via the Formatter interface in all three formats (TOON, Pretty, JSON).

## Implementation

- `StatsQuery` in the query/storage layer: query SQLite for counts grouped by status, priority, and workflow state
- Stats struct: `Total`, `ByStatus` (open/in_progress/done/cancelled), `Workflow` (ready/blocked), `ByPriority` (P0-P4 — always 5 entries)
- Ready/blocked counts reuse the ready query logic from Phase 3 (open + unblocked + no open children = ready; open + not ready = blocked)
- CLI handler calls `StatsQuery`, passes result to `Formatter.FormatStats()`
- TOON: `stats{total,open,in_progress,done,cancelled,ready,blocked}:` + `by_priority[5]{priority,count}:`
- Pretty: Three groups (Total, Status breakdown, Workflow, Priority with P0-P4 labels), right-aligned numbers
- JSON: Nested object with `total`, `by_status`, `workflow`, `by_priority`
- `--quiet`: suppress output entirely (stats has no mutation ID to return)

## Tests

- `"it counts tasks by status correctly"`
- `"it counts ready and blocked tasks correctly"`
- `"it includes all 5 priority levels even at zero"`
- `"it returns all zeros for empty project"`
- `"it formats stats in TOON format"`
- `"it formats stats in Pretty format with right-aligned numbers"`
- `"it formats stats in JSON format with nested structure"`
- `"it suppresses output with --quiet"`

## Edge Cases

- Empty project: all zeros, still shows full structure
- All tasks same status: other statuses show 0
- Priority levels with no tasks: still present as 0
- Ready/blocked counts must match ready query semantics (leaf-only, unblocked)
- `--quiet`: no output at all

## Acceptance Criteria

- [ ] StatsQuery returns correct counts by status, priority, workflow
- [ ] All 5 priority levels always present (P0-P4)
- [ ] Ready/blocked counts match Phase 3 query semantics
- [ ] Empty project returns all zeros with full structure
- [ ] TOON format matches spec example
- [ ] Pretty format matches spec example with right-aligned numbers
- [ ] JSON format nested with correct keys
- [ ] --quiet suppresses all output

## Context

Spec defines TOON stats as two sections: `stats{...}:` for totals/status/workflow counts, `by_priority[5]{priority,count}:` for priority breakdown. Pretty format uses three visual groups with right-aligned numbers and P0-P4 labels. JSON is nested object. Ready/blocked reuse Phase 3 query logic.

Specification reference: `docs/workflow/specification/tick-core.md`
