---
id: tick-core-4-5
phase: 4
status: completed
created: 2026-01-30
---

# Integrate formatters into all commands

## Goal

Replace all hardcoded output from Phases 1-3 with the resolved Formatter. Every command produces format-aware output driven by TTY detection and flag overrides. `--quiet` overrides format entirely.

## Implementation

- Wire `FormatConfig` + `Formatter` into every handler via CLI dispatcher
- **create/update**: `FormatTaskDetail` (same as show). `--quiet`: ID only.
- **start/done/cancel/reopen**: `FormatTransition`. `--quiet`: nothing.
- **dep add/rm**: `FormatDepChange`. `--quiet`: nothing.
- **list** (with filters): `FormatTaskList`. Empty handled per format. `--quiet`: IDs only.
- **show**: `FormatTaskDetail`. `--quiet`: ID only.
- **init/rebuild**: `FormatMessage`. `--quiet`: nothing.
- Format resolved once in dispatcher, not per-command. Errors remain plain text to stderr.

## Tests

- `"it formats create/update as full task detail in each format"`
- `"it formats transitions in each format"`
- `"it formats dep confirmations in each format"`
- `"it formats list/show in each format"`
- `"it formats init/rebuild in each format"`
- `"it applies --quiet override for each command type"`
- `"it handles empty list per format"`
- `"it defaults to TOON when piped, Pretty when TTY"`
- `"it respects --toon/--pretty/--json overrides"`

## Edge Cases

- `--quiet` + `--json`: quiet wins, no JSON wrapping
- Empty list: TOON zero-count, Pretty message, JSON `[]`
- Transitions plain text in TOON/Pretty, structured in JSON
- Errors always plain text to stderr regardless of format

## Acceptance Criteria

- [ ] All commands output via Formatter
- [ ] --quiet overrides per spec (ID for mutations, nothing for transitions/deps/messages)
- [ ] Empty list correct per format
- [ ] TTY auto-detection end-to-end
- [ ] Flag overrides work for all commands
- [ ] Errors remain plain text stderr
- [ ] Format resolved once in dispatcher

## Context

Spec defines per-command output: create/update = show format, transitions = plain line, deps = confirmation line, list = table, init/rebuild = message. All TTY-aware with --quiet override.

Specification reference: `docs/workflow/specification/tick-core.md`
