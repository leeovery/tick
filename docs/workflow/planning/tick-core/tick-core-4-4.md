---
id: tick-core-4-4
phase: 4
status: completed
created: 2026-01-30
---

# JSON formatter — list, show, stats output

## Goal

`--json` output for compatibility and debugging. Universal interchange — pipe to `jq`, integrate with tools. Consistent naming, correct null/empty handling.

## Implementation

- `JSONFormatter` implementing Formatter interface
- **FormatTaskList**: JSON array. Empty → `[]`, never `null`.
- **FormatTaskDetail**: Object with all fields. `blocked_by`/`children` always `[]` when empty. `parent`/`closed` omitted when null. `description` always present (empty string).
- **FormatStats**: Nested object: `total`, `by_status`, `workflow`, `by_priority` (always 5 entries).
- **FormatTransition**: `{"id", "from", "to"}`
- **FormatDepChange**: `{"action", "task_id", "blocked_by"}`
- **FormatMessage**: `{"message"}`
- All keys `snake_case`. Use `json.MarshalIndent` (2-space).

## Tests

- `"it formats list as JSON array"`
- `"it formats empty list as [] not null"`
- `"it formats show with all fields"`
- `"it omits parent/closed when null"`
- `"it includes blocked_by/children as empty arrays"`
- `"it formats description as empty string not null"`
- `"it uses snake_case for all keys"`
- `"it formats stats as structured nested object"`
- `"it includes 5 priority rows even at zero"`
- `"it formats transition/dep/message as JSON objects"`
- `"it produces valid parseable JSON"`

## Edge Cases

- Go nil slice → `null`; must initialize to empty slice for `[]`
- `parent`/`closed` omitted (not `null`) when absent
- `blocked_by`/`children` always present as arrays
- `description` always string, never null/omitted
- All keys snake_case consistently

## Acceptance Criteria

- [ ] Implements full Formatter interface
- [ ] Empty list → `[]`
- [ ] blocked_by/children always `[]` when empty
- [ ] parent/closed omitted when null
- [ ] description always present
- [ ] snake_case keys throughout
- [ ] Stats nested with 5 priority entries
- [ ] All output valid JSON
- [ ] 2-space indented

## Context

Third formatter alongside TOON and Pretty. Go nil slice gotcha requires explicit empty slice init. Keys match JSONL storage convention (snake_case).

Specification reference: `docs/workflow/specification/tick-core.md`
