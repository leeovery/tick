---
id: tick-core-4-2
phase: 4
status: pending
created: 2026-01-30
---

# TOON formatter — list, show, stats output

## Goal

Concrete TOON implementation for the Formatter interface. Agent-facing format — 30-60% token savings over JSON. Schema headers, correct counts, field escaping via toon-go library.

## Implementation

- `ToonFormatter` implementing Formatter interface
- **FormatTaskList**: `tasks[N]{id,title,status,priority}:` + indented data rows. Zero → `tasks[0]{...}:`
- **FormatTaskDetail**: Multi-section. Dynamic schema (omit parent/closed when null). blocked_by/children always present (even `[0]`). Description omitted when empty, multiline as indented lines.
- **FormatStats**: stats summary + by_priority (always 5 rows, 0-4)
- **FormatTransition/FormatDepChange/FormatMessage**: plain text passthrough
- Escaping via `github.com/toon-format/toon-go`

## Tests

- `"it formats list with correct header count and schema"`
- `"it formats zero tasks as empty section"`
- `"it formats show with all sections"`
- `"it omits parent/closed from schema when null"`
- `"it renders blocked_by/children with count 0 when empty"`
- `"it omits description section when empty"`
- `"it renders multiline description as indented lines"`
- `"it escapes commas in titles"`
- `"it formats stats with all counts"`
- `"it formats by_priority with 5 rows including zeros"`
- `"it formats transition/dep as plain text"`

## Edge Cases

- Zero count: schema header present, no rows
- Dynamic schema: parent/closed omitted when null
- blocked_by/children always present; description omitted when empty
- Commas in titles delegated to toon-go escaping

## Acceptance Criteria

- [ ] Implements full Formatter interface
- [ ] List output matches spec TOON format exactly
- [ ] Show output multi-section with dynamic schema
- [ ] blocked_by/children always present, description conditional
- [ ] Stats produces summary + 5-row by_priority
- [ ] Escaping handled by toon-go
- [ ] All output matches spec examples

## Context

Spec TOON examples define exact format. toon-go library handles escaping. Non-structured outputs (transition, dep, message) are plain text regardless of format.

Specification reference: `docs/workflow/specification/tick-core.md`
