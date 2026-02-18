---
id: tick-core-4-6
phase: 4
status: completed
created: 2026-01-30
---

# Verbose output & edge case hardening

## Goal

Implement `--verbose` / `-v` across all commands. Debug detail (cache rebuild, lock, hash comparison) to stderr only. Piped output stays clean. Quiet + verbose = silent stdout, debug stderr.

## Implementation

- `VerboseLogger` wrapping `fmt.Fprintf(os.Stderr, ...)`. No-op when verbose off.
- Instrument: freshness detection, cache rebuild, lock acquire/release, atomic write, format resolution
- All lines prefixed `verbose:` for grep-ability
- Quiet + verbose: orthogonal (different streams). Both active = silent stdout, debug stderr.
- Pipe safety: `tick list --verbose | wc -l` counts only task lines

## Tests

- `"it writes cache/lock/hash/format verbose to stderr"`
- `"it writes nothing to stderr when verbose off"`
- `"it does not write verbose to stdout"`
- `"it allows quiet + verbose simultaneously"`
- `"it works with each format flag without contamination"`
- `"it produces clean piped output with verbose enabled"`

## Edge Cases

- Verbose always stderr, never stdout
- Quiet + verbose: quiet wins stdout, verbose still writes stderr
- Verbose + format flags: valid format on stdout, debug on stderr
- Verbose off: zero additional stderr

## Acceptance Criteria

- [ ] VerboseLogger writes stderr only when Verbose true
- [ ] Key operations instrumented (cache, lock, hash, write, format)
- [ ] All lines `verbose:` prefixed
- [ ] Zero verbose on stdout
- [ ] --quiet + --verbose works correctly
- [ ] Piping captures only formatted output
- [ ] No output when verbose off

## Context

Spec: `--verbose` = "More detail for debugging." Follows stderr pattern like errors. Critical for agents — verbose must not appear in piped stdout.

Specification reference: `docs/workflow/specification/tick-core.md`
