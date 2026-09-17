# Review Tracking: Free Text Round Trip - Input Review

## Findings

### 1. The no-edge-whitespace invariant is established for descriptions only, while the bar is stated over notes and titles too

**Source**: `discussion/free-text-round-trip.md` — "Round Trip Contract" → Decision, 2026-09-17 revised (lines 110-113): "The bar itself is unchanged… it holds for descriptions, and for note text and task titles it does not, because text beginning with a dash is rejected before it reaches storage"; and the Context references (line 70) naming `TrimNoteText` alongside `TrimDescription` on the write side

**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §2.2 The fidelity bar is byte-identity (with consequences in §9.2 and §11)

**Problem**:
The guarantee is that free text read out of `tick` and written back lands byte-for-byte unchanged, and the source extends that guarantee to all three free-text carriers — descriptions, note text, task titles — naming the dash rejection as the one thing standing in the way. The whole guarantee rests on one condition: nothing in storage carries leading or trailing whitespace. That condition is established for descriptions, and patched for imported descriptions and titles, and nowhere covers note text or a title typed at the command line. The consequences land on the user: an agent that fetches one note bare is told the closing newline is always the terminator and never part of the value, which is only true if note text cannot carry trailing whitespace — nothing says it cannot, so an agent editing a note either strips a byte that belonged to it or keeps one that did not. The permanent round-trip fixture is written over "a value" with the same assumption. A builder is left to decide per write path whether whitespace is normalised, which is the unpredictable exception this work exists to delete.

**Proposal**:
State the invariant over every path that stores free text, not just the description paths: note text and titles are trimmed before storing exactly as a description is. That is the same call §2.2 already makes for `tick migrate` ("trims every free-text value it imports, exactly as `create` does"), applied to the carriers the source says the bar reaches, and it is what makes the §9.2 terminator rule and the §11 fixture true whichever field was asked for. Where the trim is already in place the statement costs nothing; where it is not, it is added alongside the import trim.

**Current**:
The bar also does not hold for free of charge across all three free-text carriers. Note text and task titles are rejected before reaching storage when they begin with a dash — see §10. Reaching byte-identity for them requires that fix, which this work carries.

**Proposed Text**:
The bar also does not hold for free of charge across all three free-text carriers. Note text and task titles are rejected before reaching storage when they begin with a dash — see §10. Reaching byte-identity for them requires that fix, which this work carries.

The whitespace invariant covers all three the same way: **every path that stores free text trims edge whitespace before storing it** — note text and titles exactly as a description is, on the command line as on import. Where that trim is already in place nothing changes; where it is not, it is added, as it is for `tick migrate` above. It is what makes the bar hold for a note read out and written back, and what makes the terminating newline of §9.2 a terminator rather than part of the value, whichever field was asked for.

**Resolution**: Pending
**Notes**:

---
