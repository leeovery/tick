# Discussion: CLI Command Structure & UX

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick is a minimal, deterministic task tracker for AI coding agents. The CLI is the primary interface - agents call commands like `tick ready --json` to get structured task data. Humans may also use it, but agent consumption is the priority.

The research phase proposed a command structure, but several UX questions remain open before we can finalize the design.

### References

- [Research: exploration.md](../research/exploration.md) (lines 176-203) - Proposed CLI commands

### Proposed Commands (from research)

**Core**: `init`, `create`, `list`, `show`, `start`, `done`, `reopen`
**Aliases**: `ready` (list --ready), `blocked` (list --blocked)
**Dependencies**: `dep add`, `dep remove`
**Utilities**: `stats`, `doctor`, `archive`, `rebuild`
**Global flags**: `--json`, `--plain`, `--quiet`, `--verbose`, `--include-archived`
**Short alias**: `tk` works as alternative to `tick`

## Questions

- [ ] What should the default output format be for each command type?
      - Agent-first means TOON default, but humans need readable output too
      - How do we balance these needs?
- [ ] Should aliases (`ready`, `blocked`) be true aliases or standalone commands?
      - Aliases share code but may confuse users about what's happening
- [ ] Is `dep add/remove` the right pattern for dependency management?
      - Alternatives: `block/unblock`, `depends/undepends`, inline on create
- [ ] How should errors and feedback be communicated?
      - Exit codes, error message format, verbosity levels
- [ ] Should there be bulk operations for planning agents?
      - Creating many tasks at once, importing from other formats
- [ ] Command naming: are the verbs clear and consistent?
      - `done` vs `complete` vs `close`
      - `create` vs `add` vs `new`

---

