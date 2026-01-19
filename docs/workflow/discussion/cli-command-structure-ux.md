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

- [x] What should the default output format be for each command type?
- [x] Should aliases (`ready`, `blocked`) be true aliases or standalone commands?
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

## Q1: Default Output Format

### Options Considered

**Option A: TOON default (agent-first)**
- Always output TOON, require `--pretty` for human-readable
- Pro: Explicit agent-first philosophy
- Con: Humans see cryptic output by default

**Option B: Human-readable default**
- Pretty tables by default, `--toon` for agents
- Pro: Intuitive for humans
- Con: Agents must always remember flag

**Option C: Auto-detect (TTY vs pipe)**
- Check if stdout is a terminal
- Terminal → human-readable table
- Piped/redirected → TOON
- Pro: Best of both worlds automatically
- Con: "Magic" behavior could confuse debugging

### How TTY Detection Works

Standard Unix mechanism - program checks if stdout is connected to a terminal:

```go
import "golang.org/x/term"

if term.IsTerminal(int(os.Stdout.Fd())) {
    // Human at terminal → pretty output
} else {
    // Piped/redirected → TOON
}
```

Well-established pattern used by `ls` (colors), `git` (pager), `grep` (colors).

### Why This Works Perfectly for Agents

When an agent runs a command via Bash tool, stdout is a pipe (not TTY):

| Who | How | TTY? | Output |
|-----|-----|------|--------|
| Human in terminal | `tick ready` | Yes | Pretty table |
| Agent via Bash tool | `tick ready` | No | TOON |
| Human in script | `tick ready --pretty` | No | Pretty table |
| Anyone wanting JSON | `tick ready --json` | - | JSON |

Agents get TOON automatically without needing any flags. Simpler agent instructions.

### Decision

**Option C: Auto-detect TTY** with explicit override flags.

- No TTY (pipe/redirect) → TOON (default for agents)
- TTY (terminal) → Human-readable table
- `--toon` → Force TOON
- `--pretty` → Force human-readable
- `--json` → Force JSON

**Rationale**: Agents naturally execute via pipes, so they get TOON without remembering flags. Humans at terminals get readable output. Edge cases covered by explicit flags. This is how Unix has worked for decades - intuitive, not magic.

---

## Q2: Aliases vs Standalone Commands

### Options Considered

**Option A: Shell-level aliases**
- User sets up aliases in shell config
- Pro: Zero code in tick
- Con: Not portable, requires setup

**Option B: Subcommand aliases in tick**
- `tick ready` internally calls `tick list --ready`
- Pro: Works everywhere, single source of truth for query logic
- Con: Two ways to do the same thing

**Option C: Standalone commands**
- Separate implementation for each command
- Pro: Can optimize independently
- Con: Code duplication, divergence risk

**Option D: No aliases**
- Just use `tick list --ready`
- Pro: One way to do things
- Con: More typing for the most common operation

### Decision

**Option B: Subcommand aliases in tick.**

`tick ready` and `tick blocked` are built-in commands that internally delegate to `tick list` with the appropriate flag. No code duplication - they share the list command's query logic.

**Rationale**: `tick ready` is likely the most-used command (agents constantly checking what to work on next). It should be easy to type. But we don't want separate implementations that could diverge. Internal delegation gives us convenience without duplication.

---

