# Tick Research Exploration

## What We're Exploring

**Tick** - A minimal, deterministic task tracker for AI coding agents.

Core thesis: Existing tools (Beads, br, Backlog.md) are either too complex or lack deterministic querying. Tick aims for zero-friction git integration with JSONL as source of truth and SQLite as auto-rebuilding cache.

## Key Architecture Decisions (Already Made)

1. **JSONL as source of truth** - Committed, human-readable, git-friendly diffs
2. **SQLite as disposable cache** - Auto-rebuilds from JSONL on staleness
3. **Dual write on mutations** - No sync commands ever
4. **No hooks, no daemons** - Just a CLI
5. **Single-agent focus** - Multi-agent is explicitly out of scope

## Open Questions to Explore

1. **ID format** - Sequential (`TICK-001`) vs hash-based (`tick-abc123`)?
2. **Subtasks** - Flat with parent reference vs nested structure?
3. **Archive strategy** - Separate file for done tasks vs keep in main?
4. **Config file** - Worth the complexity?
5. **Language choice** - Rust vs Go (or others)?

## Research Sessions

### Session 1 - 2026-01-19

**Starting context**: Comprehensive design document provided with architecture, schemas, CLI commands, and implementation plan.

---

*Research in progress...*
