---
topic: project-fundamentals
status: in-progress
date: 2026-01-21
---

# Discussion: Project Fundamentals & MVP Definition

## Context

The Tick project has 14 concluded discussions covering implementation details (ID format, schemas, file locking, CLI commands, TOON output, etc.) and 1 complete specification (core-data-storage). However, there's no foundational document that ties everything together.

The existing `exploration.md` is research - it asks questions and explores options. The discussions are implementation-focused. What's missing is the **north star**: a clear, definitive statement of what Tick is, who it's for, and what the minimum viable product looks like.

Without this foundation, implementation risks being directionless. Developers won't know which features are essential vs nice-to-have, how the pieces connect, or when v1 is "done."

### References

- [exploration.md](../research/exploration.md) - Initial research and architecture exploration
- [core-data-storage spec](../specification/core-data-storage.md) - Completed spec for data layer
- [cli-command-structure-ux](cli-command-structure-ux.md) - CLI design decisions
- [hierarchy-dependency-model](hierarchy-dependency-model.md) - Task relationships
- [toon-output-format](toon-output-format.md) - Output format for agents

## Questions

- [x] What is Tick? (Vision statement)
      - One paragraph, definitive, not exploratory
      - Who is the primary user?
      - What problem does it solve?
      - Why does it need to exist?

- [ ] What is the primary workflow?
      - End-to-end journey from `tick init` to project completion
      - How do planning agents populate tasks?
      - How do implementation agents pick up and complete work?
      - What's the human's role?

- [ ] What commands are essential for MVP (v1)?
      - Which commands from the CLI discussion are must-have?
      - Which can be deferred to future versions?
      - What's the minimum set that delivers value?

- [ ] What are the explicit non-goals?
      - What are we deliberately NOT building?
      - Consolidate scattered deferral decisions
      - Why are these out of scope?

- [ ] How does Tick integrate with claude-technical-workflows?
      - What's the adapter interface?
      - How does a planning agent output to tick?
      - How does an implementation agent query for work?
      - What metadata is needed beyond tasks?

- [ ] What are the success criteria for v1?
      - How do we know v1 is "done"?
      - What must work?
      - What quality bar must be met?

---

*Each question above gets its own section below. Check off as concluded.*

---

## What is Tick? (Vision Statement)

### Context

The exploration.md describes Tick as "a minimal, deterministic task tracker for AI coding agents" but this is buried in research context. We need a definitive statement that can serve as the project's north star.

### Options Considered

**Option A: Agent-only tool**
Tick is exclusively for AI agents. Humans never interact directly - they use tick only through agents.

**Option B: Agent-first, human-usable**
Tick is optimized for AI agents but humans can use it directly for oversight and manual intervention.

**Option C: Dual-audience tool**
Tick serves both agents and humans equally, with different output modes for each.

### Journey

Started by examining the existing alternatives and their pain points:

| Tool | Problem |
|------|---------|
| **Beads** | Too complex - daemons, hooks, 730-line uninstall, auto-commits conflicting with pipelines. User "doesn't like how it's put together." |
| **br (beads_rust)** | Promising but requires manual sync (`br sync --flush-only`, `br sync --import-only`) - easy to forget → data loss. Manual sync is a dealbreaker. |
| **Backlog.md** | No deterministic querying - agents parse markdown, can miss things. Token-expensive to analyze. Plans get large. |

The common thread: **agents need a deterministic "what should I work on next?" query, but existing tools either add complexity or require manual steps that break the workflow.**

Confirmed that **Option B (Agent-first, human-usable)** matches intent. Agents are the primary user, but humans can and will use it directly for oversight. The TTY auto-detection in CLI discussions already reflected this implicitly.

Confirmed that **minimal simplicity is a core value** - fewer features done well over more features.

### Decision

**Vision Statement:**

> **Tick** is a minimal, deterministic task tracker designed for AI coding agents.
>
> Agents need to know "what should I work on next?" without ambiguity. Existing tools either require manual sync steps (easy to forget, causing data loss), add complexity through daemons and hooks (hard to manage, harder to uninstall), or rely on markdown parsing (non-deterministic, token-expensive at scale).
>
> Tick solves this with a simple model: JSONL as the git-committed source of truth, SQLite as an auto-rebuilding local cache, and a CLI that always returns deterministic results. No sync commands. No daemons. No hooks. Just `tick ready` to get the next task.
>
> **Primary user:** AI coding agents (Claude Code and similar), with humans able to use it directly for oversight and manual intervention.

**Tagline:** "A minimal, deterministic task tracker for AI coding agents"

**Core values:**
- Minimal simplicity - fewer features done well
- Deterministic queries - same input, same output, always
- Zero sync friction - no manual sync commands ever

---

## What is the primary workflow?

### Context

Understanding the end-to-end workflow is essential for knowing which features matter. The existing discussions assume a two-agent pattern (planning + implementation) but don't explicitly document it.

### Options Considered

*(To be explored during discussion)*

### Journey

*(To be filled during discussion)*

### Decision

*(Pending)*

---

## What commands are essential for MVP (v1)?

### Context

The CLI discussion covers many commands: create, start, done, cancel, reopen, list, show, dep add/rm, ready, blocked, doctor, rebuild, migrate, init. Not all may be needed for v1.

### Options Considered

*(To be explored during discussion)*

### Journey

*(To be filled during discussion)*

### Decision

*(Pending)*

---

## What are the explicit non-goals?

### Context

Several discussions concluded with "defer" decisions (archive strategy, config file). These should be consolidated into explicit non-goals so the scope is clear.

### Options Considered

*(To be explored during discussion)*

### Journey

*(To be filled during discussion)*

### Decision

*(Pending)*

---

## How does Tick integrate with claude-technical-workflows?

### Context

Tick is meant to replace/complement existing output formats (Beads, Backlog.md, Local Markdown) in the planning phase. The integration contract needs definition.

### Options Considered

*(To be explored during discussion)*

### Journey

*(To be filled during discussion)*

### Decision

*(Pending)*

---

## What are the success criteria for v1?

### Context

Without clear success criteria, it's impossible to know when v1 is complete. This should be concrete and testable.

### Options Considered

*(To be explored during discussion)*

### Journey

*(To be filled during discussion)*

### Decision

*(Pending)*

---

## Summary

### Key Insights

*(To be filled as discussion progresses)*

### Current State

- Discussion started: 2026-01-21
- All questions pending

### Next Steps

- [ ] Work through each question with user input
- [ ] Document decisions and rationale
- [ ] Create foundation for subsequent specifications
