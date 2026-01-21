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

- [x] What is the primary workflow?
      - End-to-end journey from `tick init` to project completion
      - How do planning agents populate tasks?
      - How do implementation agents pick up and complete work?
      - What's the human's role?

- [x] What commands are essential for MVP (v1)?
      - Which commands from the CLI discussion are must-have?
      - Which can be deferred to future versions?
      - What's the minimum set that delivers value?

- [x] What are the explicit non-goals?
      - What are we deliberately NOT building?
      - Consolidate scattered deferral decisions
      - Why are these out of scope?

- [x] How does Tick integrate with claude-technical-workflows?
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

### Journey

Mapped out a typical workflow:

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. PROJECT SETUP                                                │
│    Human runs: tick init                                        │
│    Result: .tick/ directory created                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. PLANNING PHASE                                               │
│    Planning agent (or human) creates tasks:                     │
│    - tick create "Setup authentication" --priority 1            │
│    - tick create "Login endpoint" --blocked-by tick-a1b2        │
│    - tick create "Logout endpoint" --blocked-by tick-c3d4       │
│    Result: tasks.jsonl populated with work items                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. IMPLEMENTATION LOOP                                          │
│    Implementation agent (or human):                             │
│    a) tick ready        → "What can I work on?"                 │
│    b) tick start <id>   → "I'm working on this"                 │
│    c) [does the work]                                           │
│    d) tick done <id>    → "I finished this"                     │
│    e) Repeat until tick ready returns nothing                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. PROJECT COMPLETE                                             │
│    All tasks done. tasks.jsonl shows history.                   │
│    .tick/ can be deleted or kept for reference.                 │
└─────────────────────────────────────────────────────────────────┘
```

**Key insight: Tick is workflow-agnostic.** It's a tracker, not a workflow engine.

- Tick doesn't care *who* creates tasks (human, planning agent, implementation agent mid-work)
- Tick doesn't care *who* works tasks (human or agent)
- Tick doesn't enforce *when* things happen
- Tick just answers: "What's ready?" and tracks state changes

The workflow above is a *typical* use case, not something Tick enforces.

**Human's role:** Flexible. The tool is designed for agent-driven implementation (hands-off, semi-automated), but humans can do the work too. Primary human activities:
- Reviewing code / ensuring correctness
- Observing and intervening where necessary
- Pre-planning: ensuring plans are well-formed so agents can execute

**Mid-project planning:** Allowed. Tasks can be created at any time. This is outside Tick's concern - it just tracks whatever tasks exist.

**Success depends on good pre-planning** - if plans are well-formed with proper dependencies, the agent can work autonomously. Tick enables this but doesn't enforce it.

### Decision

**Primary workflow:** Init → Plan → Implement Loop → Complete

**Core principle:** Tick is a simple tracker. It doesn't orchestrate workflows, enforce processes, or care about who does what. It answers one question reliably: "What's ready to work on?"

---

## What commands are essential for MVP (v1)?

### Context

The CLI discussion covers many commands: create, start, done, cancel, reopen, list, show, dep add/rm, ready, blocked, doctor, rebuild, migrate, init.

### Journey

Initially attempted to split commands into "essential" vs "deferrable" categories. This was misguided - the CLI discussion already concluded with a considered command set. All commands were discussed for good reasons and are applicable to v1.

The fundamentals discussion should reference existing decisions, not re-litigate them.

### Decision

**All commands from the [cli-command-structure-ux](cli-command-structure-ux.md) discussion are in scope for v1.**

The MVP scope is defined by the concluded discussions:
- **Commands:** [cli-command-structure-ux](cli-command-structure-ux.md)
- **Data layer:** [core-data-storage spec](../specification/core-data-storage.md)
- **Output format:** [toon-output-format](toon-output-format.md)
- **Task relationships:** [hierarchy-dependency-model](hierarchy-dependency-model.md)
- **Validation:** [doctor-command-validation](doctor-command-validation.md)
- **Human output:** [tui](tui.md)

No additional features are needed. No discussed features should be cut.

---

## What are the explicit non-goals?

### Context

Several discussions concluded with "defer" decisions (archive strategy, config file). These should be consolidated into explicit non-goals so the scope is clear.

### Journey

Consolidated deferral decisions from existing discussions and confirmed additional non-goals:

**From existing discussions:**
- Archive strategy → deferred (YAGNI) per [archive-strategy-implementation](archive-strategy-implementation.md)
- Config file → deferred (YAGNI) per [config-file-design](config-file-design.md)
- Windows installation → not priority per [installation-options](installation-options.md)

**Additional non-goals confirmed:**
- Multi-agent coordination - not in scope
- Real-time sync between machines - definitely not
- GUI/web interface - no
- Plugin/extension system - no

### Decision

**Explicit non-goals for v1:**

| Non-goal | Rationale |
|----------|-----------|
| **Archive strategy** | YAGNI - single file sufficient for v1. Revisit if files get large. |
| **Config file** | YAGNI - hardcoded defaults are fine. No user customization needed yet. |
| **Windows support** | Not a priority - macOS and Linux first. |
| **Multi-agent coordination** | Out of scope - Tick is single-agent focused. Multiple agents on same project is not a use case we're solving. |
| **Real-time sync** | Definitely not - git is the sync mechanism. No live collaboration features. |
| **GUI/web interface** | No - CLI only. Agent-first means no visual UI needed. |
| **Plugin/extension system** | No - keep it simple. No hooks, no customization points. |

**Guiding principle:** When in doubt, leave it out. Minimal simplicity is a core value.

---

## How does Tick integrate with claude-technical-workflows?

### Context

Tick is meant to replace/complement existing output formats (Beads, Backlog.md, Local Markdown) in the planning phase. The integration contract needs definition.

### Journey

Asked whether Tick needs to define an integration contract or provide specific features for claude-technical-workflows integration.

### Decision

**Integration is outside Tick's scope.**

Tick provides a CLI. How other tools (claude-technical-workflows, custom scripts, etc.) use that CLI is their concern. Tick doesn't need to know about workflow systems.

This is consistent with "Tick is just a tracker" - it provides:
- `tick create` for adding tasks
- `tick dep add` for dependencies
- `tick ready` for querying what's workable
- `tick start/done` for state changes

Any planning tool can call these commands. Any implementation agent can query and update state. Tick doesn't care who calls it or why.

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
