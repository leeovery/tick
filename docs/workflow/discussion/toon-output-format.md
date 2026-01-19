# Discussion: TOON Output Format Implementation

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick needs to output task data in a format optimized for AI agent consumption. The research phase decided on TOON (Token-Oriented Object Notation) as the default output format, with JSON as a fallback. This discussion formalizes the implementation details.

**Why TOON matters**: Agents consume tick output frequently (every `tick ready` call). Token efficiency directly impacts cost and context window usage. TOON achieves 30-60% token savings over JSON while actually improving parsing accuracy (73.9% vs 69.7% in benchmarks).

**Core decision already made**: JSONL for storage, TOON for default output.

### References

- [Research: exploration.md](../research/exploration.md) (lines 473-511)
- [TOON specification](https://github.com/toon-format/toon)

## Questions

- [ ] What should the TOON output structure look like for each command type?
- [ ] How should output format selection work (flags, detection, defaults)?
- [ ] How should complex/nested data be handled in TOON?
- [ ] Should error output also use TOON format?
- [ ] What about human-readable output (--plain)?

---

*Each question above gets its own section below. Check off as concluded.*

---

## What should the TOON output structure look like for each command type?

### Context

Different commands return different data shapes. `tick list` returns arrays of tasks. `tick show` returns a single task with more detail. `tick stats` returns aggregates. Each needs a TOON representation.

### Options Considered

*To be explored in discussion*

### Journey

*Discussion will be captured here*

### Decision

*Pending*

---

## How should output format selection work?

### Context

Need to support multiple output formats: TOON (default for agents), JSON (compatibility/debugging), plain text (humans). How does the user/agent select which format?

### Options Considered

*To be explored in discussion*

### Journey

*Discussion will be captured here*

### Decision

*Pending*

---

## How should complex/nested data be handled in TOON?

### Context

TOON excels at uniform arrays (like task lists). But what about task details with nested dependencies, or hierarchical parent-child relationships? Need to decide how to represent these.

### Options Considered

*To be explored in discussion*

### Journey

*Discussion will be captured here*

### Decision

*Pending*

---

## Should error output also use TOON format?

### Context

When commands fail, what format should errors use? Should it match the requested output format, or always be plain text for readability?

### Options Considered

*To be explored in discussion*

### Journey

*Discussion will be captured here*

### Decision

*Pending*

---

## What about human-readable output (--plain)?

### Context

While agents are the primary user, humans need to read output too (debugging, oversight). What should the default human-readable format look like?

### Options Considered

*To be explored in discussion*

### Journey

*Discussion will be captured here*

### Decision

*Pending*

---

## Summary

### Key Insights

*To emerge from discussion*

### Current State

- Core decision made: TOON as default, JSON as fallback
- Implementation details need formalizing

### Next Steps

- [ ] Work through each question above
