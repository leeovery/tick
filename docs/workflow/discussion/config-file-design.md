# Discussion: Config File Design

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick needs a way to customize behavior per-project. The research phase proposed a simple flat key=value config file at `.tick/config`. This discussion validates that approach and works through the details.

**User preference**: Simple flat config for overriding defaults only. Minimal complexity.

**Agent-first principle**: Config should not require agent awareness - sensible defaults mean agents can ignore config entirely.

### References

- [Research: exploration.md](../research/exploration.md) (lines 401-437) - Config file proposal
- [CLI Command Structure & UX](cli-command-structure-ux.md) - Output flags, error handling decisions
- [TOON Output Format](toon-output-format.md) - Format selection decisions

### Relevant Prior Decisions

From CLI discussion:
- Output format auto-detected via TTY (agents get TOON, humans get pretty)
- Override flags: `--toon`, `--pretty`, `--json`, `--quiet`, `--verbose`
- Simple exit codes (0/1), plain text errors to stderr

## Questions

- [ ] What config options are actually needed?
- [ ] What format should the config file use?
- [ ] Where should config live and when is it created?
- [ ] How do config values interact with command-line flags?
- [ ] Should environment variables override config?
- [ ] How to handle unknown or deprecated config keys?

---

*Discussion begins below*

---

