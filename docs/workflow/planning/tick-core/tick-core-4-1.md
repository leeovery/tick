---
id: tick-core-4-1
phase: 4
status: pending
created: 2026-01-30
---

# Formatter abstraction & TTY-based format selection

## Goal

Define a `Formatter` interface, implement TTY detection, resolve format from flags vs auto-detection with conflict handling, and wire the chosen formatter into CLI dispatch. Foundation for tasks 4-2 through 4-4.

## Implementation

- `Formatter` interface with methods: `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage`
- `Format` enum: `FormatToon`, `FormatPretty`, `FormatJSON`
- `DetectTTY()`: `os.Stdout.Stat()` → check `ModeCharDevice`. Stat failure → default non-TTY.
- `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)`: >1 flag → error; 1 flag → that format; 0 flags + TTY → Pretty; 0 flags + no TTY → Toon
- `FormatConfig` struct: Format, Quiet, Verbose — passed to all handlers
- Stub formatter as placeholder (concrete formatters in 4-2 through 4-4)
- `--verbose` to stderr only, never contaminates stdout

## Tests

- `"it detects TTY vs non-TTY"`
- `"it defaults to Toon when non-TTY, Pretty when TTY"`
- `"it returns correct format for each flag override"`
- `"it errors when multiple format flags set"`
- `"it propagates quiet and verbose in FormatConfig"`
- `"it defaults to non-TTY on stat failure"`

## Edge Cases

- Stat failure → non-TTY default, no panic
- Conflicting flags → error before dispatch
- Verbose orthogonal to format — stderr only
- Quiet orthogonal — doesn't change format selection

## Acceptance Criteria

- [ ] Formatter interface covers all command output types
- [ ] Format enum with 3 constants
- [ ] TTY detection works correctly
- [ ] ResolveFormat handles all flag/TTY combos
- [ ] Conflicting flags → error
- [ ] FormatConfig wired into CLI dispatch
- [ ] Verbose to stderr only
- [ ] Stat failure handled gracefully

## Context

Spec: "No TTY → TOON, TTY → human-readable. --toon, --pretty, --json override." tick-core-1-5 wired flag parsing — this extracts into proper abstraction.

Specification reference: `docs/workflow/specification/tick-core.md`
