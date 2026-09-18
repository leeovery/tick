# Review Tracking: Free Text Round Trip - Integrity

## Findings

### 1. Phase 3's ordering rationale says it shares no code with Phase 1, and three of its six tasks call a function Phase 1 wrote

**Severity**: Minor
**Plan Reference**: Phase 3 (Stats and dependency-tree documents), "Why this order" — `planning.md:67`
**Category**: Phase Structure — the phase's stated independence claim is contradicted by its own tasks, in the same sentence that states the claim
**Move**: settled
**Change Type**: update-phase

**Problem**:
"Why this order" is the paragraph a reader consults before deciding whether a phase can be reordered or run in parallel with another — the same role cycle 5 established for Phase 4's version of this paragraph, and fixed there. Phase 3's says "They share no code with Phases 1 and 2, so the phase is self-contained." That is not true of Phase 1: task `free-text-round-trip-3-1`'s Do step 1, task `free-text-round-trip-3-2`'s Do step 1, and task `free-text-round-trip-3-3`'s Do step 1 each call `encodeToonFields` — the helper task `free-text-round-trip-1-1` added to `internal/cli/toon_formatter.go` in Phase 1 — to produce the named-field form each task emits. Three of the phase's six tasks call a function Phase 1 wrote; that is code the two phases share.

The same sentence goes on to say "it follows them because it reuses the named-field form Phase 1 establishes" — which is a direct description of that same reliance. So the sentence asserts "no code shared with Phase 1" and, in its next clause, names the shared code that contradicts it. The claim is true only of Phase 2: no Phase 3 task touches `StatusChange`, `CascadeResult` or anything else Phase 2 added, so nothing ties Phase 3 to Phase 2. Grouping "Phases 1 and 2" together under one "share no code" claim overstates what is true of Phase 1's half of it.

**Proposal**:
Narrow the "share no code" claim to Phase 2, where it holds, and state the Phase 1 reliance directly instead of asserting its absence one clause before describing it. The plan settles every part of it: the three tasks that call `encodeToonFields` are named in the plan's own Do steps, and the helper's origin is task `free-text-round-trip-1-1`'s own Do step 1. Nothing new is decided and no task changes — the rationale is brought level with the tasks it describes.

**Current**:
```
**Why this order**: These are the last two malformed headers and the last prose answer to a query. They share no code with Phases 1 and 2, so the phase is self-contained; it follows them because it reuses the named-field form Phase 1 establishes, and because its risk profile differs — the empty-branch fix lands in the dep-tree handler rather than a formatter, and pretty prints nothing at all on that branch unless handed its sentence explicitly.
```

**Proposed Text**:
```
**Why this order**: These are the last two malformed headers and the last prose answer to a query. They share no code with Phase 2, so nothing ties the phase to it — but three of its six tasks depend on Phase 1: `free-text-round-trip-3-1`, `free-text-round-trip-3-2` and `free-text-round-trip-3-3` each call `encodeToonFields`, the helper `free-text-round-trip-1-1` added, to produce the named-field form. It follows Phase 1 for that reason, and follows both Phases 1 and 2 because its risk profile differs from theirs — the empty-branch fix lands in the dep-tree handler rather than a formatter, and pretty prints nothing at all on that branch unless handed its sentence explicitly.
```

**Resolution**: Fixed
**Notes**: Applied verbatim to Phase 3's "Why this order" in planning.md.
**Notes**:

---

## Verification Notes (no finding)

Recorded so the checks that were run and came back clean are visible, and are not re-run from scratch next cycle.

- **The two cycle-5 fixes hold, in both places they were applied.** Phase 4's "Why this order" (`planning.md:94`) carries the corrected text verbatim, with no phase-level rationale duplicated in the phase's tick container (`tick-ec0b32` carries no description — phase containers in tick hold no rationale text, confirmed by inspecting `tick-0540b5` and `tick-ec0b32` directly). Task `free-text-round-trip-6-6` (`tick-f9b5f3`) carries the corrected acceptance criterion in both `phase-6-tasks.md` and the tick record. Task `free-text-round-trip-1-3` (`tick-a119e7`) carries the corrected Context paragraph in both `phase-1-tasks.md` and the tick record.
- **A full sweep for the same defect class** — a phase's "Why this order" asserting independence or "no shared code" from another phase — found exactly one other instance in the plan (Phase 3, the finding above); Phases 1, 2, 5 and 6 make no such claim, so there was nothing else to check them against.
- **Template compliance, dependencies/ordering, and code-reference spot checks were not re-run in full** — cycle 5's verification notes cover them exhaustively (all 38 tasks synced field-for-field against tick, natural creation-date ordering confirmed sufficient with no cycles possible, `commandFlags` partition confirmed complete) and nothing in this cycle's read-through of all six phase files or `planning.md` contradicts them. No new task content was added since cycle 5's sync check beyond the two already-verified fixes above.
- **Comments-are-not-task-content compliance re-checked**: the five "update the doc comment" Do steps across the six phase files (`free-text-round-trip-1-2`, `3-2`, `3-3`, `3-5`, plus the doc-comment note in `2-1`'s Context) each repair a comment the change would otherwise falsify; none directs new commentary referencing other tasks, phases or spec sections.
- **Scope and granularity re-checked across all 38 tasks**: Do steps run 4–6 per task (only `free-text-round-trip-2-2` at 6, an accepted outlier from cycle 1), consistent with cycle 5's count.
