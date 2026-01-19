# Discussion: ID Format Implementation

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick needs unique, stable task identifiers. The research phase decided on hash-based IDs with customizable prefix (`{prefix}-{hash}` format like `tick-a3f2b7`), rejecting sequential IDs (`TICK-001`) to avoid merge conflicts.

**What's settled**:
- Format: `{prefix}-{hash}`
- Prefix customizable at `tick init` (default: `tick`)
- Must be readable and referenceable in conversation

**What's open**: The implementation details of the hash portion.

### References

- [Research: exploration.md](../research/exploration.md) (lines 305-319) - Original ID format decision

## Questions

- [ ] What should be the hash length?
      - Research suggested 6-8 chars
      - Trade-off: typability vs collision risk
- [ ] Random or content-based hash?
      - Random: simple, guaranteed unique
      - Content-based: deterministic, reproducible
- [ ] How should collisions be handled?
      - Detection strategy
      - Resolution strategy
- [ ] What character set for the hash?
      - Hex (0-9, a-f) vs base36 (0-9, a-z) vs other
- [ ] Should IDs be case-sensitive?

---

## What should be the hash length?

### Context
IDs need to be short enough to type/reference in conversation but long enough to avoid collisions within a project's lifetime. Research suggested 6-8 characters.

### Options Considered

**6 characters**
- Pros: Easy to type, fits well in conversation
- Cons: ~17M combinations (hex) - collision risk grows with project size

**7 characters**
- Pros: Balance of brevity and space
- Cons: Odd number, slightly less elegant

**8 characters**
- Pros: ~4B combinations (hex), effectively collision-proof for any project
- Cons: Longer to type

### Journey

*(To be captured during discussion)*

### Decision

*(Pending)*

---

## Random or content-based hash?

### Context
The hash source affects determinism and uniqueness guarantees.

### Options Considered

**Random (UUID/nanoid slice)**
- Pros: Simple, guaranteed unique, no input dependencies
- Cons: Non-deterministic, can't recreate

**Content-based (hash of title/description)**
- Pros: Deterministic, same input = same ID
- Cons: Changing title changes ID? Or frozen at creation?

**Timestamp-based**
- Pros: Naturally ordered, unique if resolution high enough
- Cons: Reveals creation time, potential privacy concern

### Journey

*(To be captured during discussion)*

### Decision

*(Pending)*

---

## How should collisions be handled?

### Context
Even with low probability, collisions need a strategy.

### Options Considered

*(To be explored)*

### Journey

*(To be captured during discussion)*

### Decision

*(Pending)*

---

## Summary

### Key Insights
*(To emerge from discussion)*

### Current State
- Format decided: `{prefix}-{hash}`
- Implementation details: under discussion

### Next Steps
- [ ] Resolve hash length
- [ ] Decide random vs content-based
- [ ] Define collision handling
