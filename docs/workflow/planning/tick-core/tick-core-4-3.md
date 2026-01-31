---
id: tick-core-4-3
phase: 4
status: pending
created: 2026-01-30
---

# Human-readable formatter — list, show, stats output

## Goal

Concrete Pretty implementation of Formatter interface. Terminal-facing — aligned columns, no borders/colors/icons. Matches spec examples exactly.

## Implementation

- `PrettyFormatter` implementing Formatter interface
- **FormatTaskList**: Column-aligned table with header. Dynamic widths. Empty → `No tasks found.` (no headers).
- **FormatTaskDetail**: Key-value with aligned labels. Sections: base fields, Blocked by (indented), Children (indented), Description (indented block). Omit empty sections.
- **FormatStats**: Three groups — total, status breakdown, workflow counts, priority with labels P0-P4. Right-align numbers. All rows present even at zero.
- **FormatTransition/FormatDepChange/FormatMessage**: plain text passthrough
- Long titles truncated with `...` in list; full in show.

## Tests

- `"it formats list with aligned columns"`
- `"it aligns with variable-width data"`
- `"it shows 'No tasks found.' for empty list"`
- `"it formats show with all sections"`
- `"it omits empty sections in show"`
- `"it formats stats with all groups, right-aligned"`
- `"it shows zero counts in stats"`
- `"it renders P0-P4 priority labels"`
- `"it truncates long titles in list"`
- `"it does not truncate in show"`

## Edge Cases

- Variable-width: `in_progress` (11) vs `open` (4) — dynamic column width
- Long titles: truncate + `...` in list only
- Empty results: `No tasks found.`, no headers
- Show omits empty sections entirely
- Stats: all rows present including zeros

## Acceptance Criteria

- [ ] Implements full Formatter interface
- [ ] List matches spec format — aligned columns with header
- [ ] Empty list → `No tasks found.`
- [ ] Show matches spec — aligned labels, omitted empty sections
- [ ] Stats three groups with right-aligned numbers
- [ ] Priority P0-P4 labels always present
- [ ] Long titles truncated in list
- [ ] All output matches spec examples

## Context

Spec: "Minimalist and clean. Human output secondary to agent output — no TUI libraries, no interactivity." Aligned columns, no borders/colors.

Specification reference: `docs/workflow/specification/tick-core.md`
