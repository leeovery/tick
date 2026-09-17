# Review Tracking: Free Text Round Trip - Input Review

## Findings

### 1. The narrower scope that was considered and declined is not recorded

**Source**: `discussion/free-text-round-trip.md` — "Toon Conformance Scope" → Journey and Decision (lines 137-143): the fork "whether to fix free text now and log the rest as a separate concern, or fix the format outright in this work", declined because "fixing a quarter of a broken format leaves the output exactly as unusable as before, so the free-text work would deliver nothing on its own"; the ruling "Fix the whole format in this work"; and "This brings three areas in that were previously out of scope"
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §1 Purpose

**Problem**:
The work is named for free text and delivers a rewrite of the task header, the stats summary, the dep-tree summary, tags, refs and the whole test approach. Nothing written down says a narrower version was on the table or why it was refused, so the first person who looks at the size of it and asks "can we just fix descriptions and log the rest?" has no answer in front of them and takes the obvious one. Shipped narrow, the work delivers nothing an agent can use: the document still fails a TOON reader on its first line, long before the repaired description is reached, and the agent still goes to `.tick/tasks.jsonl`. The rest of the specification records the alternative it turned down at every choice it made — the one-row table, the dash-list, `--raw`, leaving status output alone, a note edit command — and this is the one scope call with no such record, which makes it the easiest to undo by accident.

**Proposal**:
Carry the decline into §1 with the reason the source gave — a quarter of a broken format is as unusable as all of it — and name the three areas the widening brought in, so the breadth of §5, §6.1 and §11 reads as a consequence of the scope ruling rather than as unexplained extra work.

**Current**:
Each hand-rolled section exists because it wanted a shape the library does not produce directly — a singular object header, a list down the page, an unstructured text block. In every case the invented shape turned out to be invalid.

**This work delivers toon output that a standard TOON reader can read, for every command that returns data, and free text that survives an agent's read-edit-write cycle without a rule learned outside the output.**

**Proposed Text**:
Each hand-rolled section exists because it wanted a shape the library does not produce directly — a singular object header, a list down the page, an unstructured text block. In every case the invented shape turned out to be invalid.

**This work delivers toon output that a standard TOON reader can read, for every command that returns data, and free text that survives an agent's read-edit-write cycle without a rule learned outside the output.**

A narrower version was available and refused: repair free text now and log the rest as a separate concern. A quarter of a broken format is exactly as unusable as all of it — the document fails on its first line, so a reader never reaches the repaired description and the agent goes back to the data file — which means the free-text work delivers nothing on its own. Free-text encoding is therefore not the deliverable in itself; it is one of the malformed sections. Taking the whole format brings in three areas that sat outside the original framing: the single-object section headers (§5), the tags and refs lists (§6.1), and the verification that keeps the output conformant afterwards (§11).

**Resolution**: Pending
**Notes**:

---
