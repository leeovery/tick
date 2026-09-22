# Review Tracking: Same Second Tasks Sort By Id - Traceability

## Findings

### 1. Phase 1 builds the running-maximum backfill the specification rejected

**Type**: Hallucinated content
**Spec Reference**: §2.3 (as corrected by the 2026-09-22 corrigendum), §2.2, §5.2, §8.2 (merge-shape case)
**Plan Reference**: Phase 1 Goal and Acceptance (planning.md); Phase 1 task table row same-second-tasks-sort-by-id-1-4; Task same-second-tasks-sort-by-id-1-4 (tick-ecc73a) title, Problem, Solution, Outcome, Acceptance Criteria, Do, Context; Context of Tasks same-second-tasks-sort-by-id-1-3 (tick-0bb4f7) and same-second-tasks-sort-by-id-1-5 (tick-a21794)
**Move**: settled
**Change Type**: update-task

**Problem**:
Task 1-4 tells the implementer to backfill unnumbered records with a running maximum, the highest sequence seen so far. The specification rejected that rule in its 2026-09-22 corrigendum because it collides. After a merge, records with no sequence can sit above numbered ones: `a`, `b`, `c` at 1–3, then `x` and `y` with none, then `d` and `e` at 4 and 5. A running maximum gives `x` and `y` 4 and 5, duplicating `d` and `e`. Tick would create those duplicates itself, with no warning. The next write would save them to the file, and the doctor check would then send the user to fix them by hand. None of Task 1-4's criteria tell the two rules apart, since every fixture it names numbers identically under both. So Phase 1 would pass its checkpoint with the wrong rule, and only Task 3-2's end-to-end assertion in Phase 3 would catch it. The Phase 1 Goal and Acceptance state the same rejected rule. Tasks 1-3 and 1-5 quote wording that the corrected §2.3 no longer contains, and credit it to the rule the spec rejected.

**Proposal**:
Carry the corrected §2.3 rule into every place Phase 1 states it. Take the highest sequence any record in the file carries (0 when none does). Then walk the records in order and give each unnumbered record the next number above it. This is new-task numbering applied to each record in turn, so a backfilled number can never equal a carried one. Task 1-4 also gains the §8.2 merge-shape criterion, the case a running maximum gets wrong: `x` and `y` get 6 and 7, and `tick list` returns `a`, `b`, `c`, `d`, `e`, `x`, `y`. §2.3 says this outright: "an unnumbered record lying above numbered ones sorts after them where they share a creation second". With that criterion the rule is proven inside the task that builds it, not first in Phase 3. The Phase 1 Acceptance gains the same case. The stale quotes in Tasks 1-3 and 1-5 are replaced with current §2.3 wording. The specification decides all of this, so no choice remains.

**Current**:

*planning.md — Phase 1 Goal:*

**Goal**: Record an explicit creation sequence on every task (assigned on create, backfilled on read by running maximum, carried into a schema-v3 cache column) and end the list/ready/blocked sort on created, sequence, then task ID. Tied tasks then read back in authoring order, and a duplicate sequence falls back to one fixed order.

*planning.md — Phase 1 Acceptance (one bullet):*

- [ ] A file where some records carry seq and some do not (post-merge shape), all in one second: backfill assigns by running maximum and the newest record orders last

*planning.md — Phase 1 task table row:*

| same-second-tasks-sort-by-id-1-4 | Backfill missing sequences by running maximum on read | a file with no seq on any record orders by line position, including after a create, update or remove (§2.3, §8.2), a mixed seq/no-seq file orders its newest record last and not first (§2.3, §8.2), a record whose seq was stripped by a binary that does not know the field is reassigned correctly (§7.1, §8.2), blank lines are skipped so seq order matches record order rather than line number (§8.2), a new task numbers above the backfilled values (§2.2), the ReadTasks, Rebuild and readAndEnsureFresh paths all get the same rule (§2.3), the stored-record byte-immutability break is accepted (§3.1, §7.2) |

*Task 1-4 (tick-ecc73a) — title:*

Backfill Missing Sequences By Running Maximum On Read

*Task 1-4 — Problem:*

**Problem**: Every existing project's `tasks.jsonl` predates the field, so none of its records carries a sequence and its same-second ties stay plan-dependent. Records without a sequence keep arriving afterwards too: from a merge of branches, or from a write by a binary that does not know the field and drops it. Numbering those by line position gets the post-merge shape wrong — records that already carry sequences numbered above their line positions would sort the newest, unnumbered record first.

*Task 1-4 — Solution:*

**Solution**: As records are parsed, walk them in order tracking the highest sequence seen so far, starting at 0, and give the next number to each record that has none — a running maximum, not line position. Doing it in the one parse funnel puts every read path on the same rule, and the next write makes the assignment permanent.

*Task 1-4 — Outcome:*

**Outcome**: Existing projects read back in line order, which is their authoring order, with no migration and no repair. The first mutation writes the assigned sequences into the file. Post-merge and stripped files order their newest record last, and a new task numbers above the backfilled values.

*Task 1-4 — Acceptance Criteria (the criterion the new one follows):*

- [ ] A file whose first records carry sequences numbered above their line positions, followed by a newest record carrying none, all in one creation second and one priority: the newest record is assigned a number above every carried sequence and `tick list` returns it last (§2.3, §8.2)

*Task 1-4 — Do (second bullet):*

- The rule: walk the records in order, tracking the highest sequence seen so far starting at 0, and give the next number to any record that has none — absent or zero (§2.2, §2.3).

*Task 1-4 — Context (first quote):*

> §2.3: "This is a running-maximum rule, **not** line position. On a file where some records carry a sequence and some do not — the shape produced by a merge, or by a write from a binary that does not know the field — assigning line position would sort the newest record first. Running maximum gets it right and collapses backfill and new-task numbering into a single rule." For a file with no sequences at all it yields line order, which is authoring order — verified across the project's entire history: no sort call has ever existed on the storage, task or create paths, and every JSONL writer has iterated the task slice unsorted.

*Task 1-3 (tick-0bb4f7) — Context (end of the first quote):*

Backfill is Task 1-4; §2.3 notes the running-maximum rule "collapses backfill and new-task numbering into a single rule".

*Task 1-5 (tick-a21794) — Context (last line):*

> §2.3 notes the running-maximum rule "collapses backfill and new-task numbering into a single rule"; `tick create`'s assignment is Task 1-3.

**Proposed Text**:

*planning.md — Phase 1 Goal:*

**Goal**: Record an explicit creation sequence on every task (assigned on create, backfilled on read above the file's highest sequence, carried into a schema-v3 cache column) and end the list/ready/blocked sort on created, sequence, then task ID. Tied tasks then read back in authoring order, and a duplicate sequence falls back to one fixed order.

*planning.md — Phase 1 Acceptance (replaces the one bullet with two):*

- [ ] A file where some records carry seq and some do not (post-merge shape), all in one second: each record without a seq is numbered, in record order, above the highest seq the file carries, and the newest record orders last
- [ ] A file whose unnumbered records lie above numbered ones (merge shape: a, b, c at seq 1–3, then x and y with none, then d and e at 4 and 5), all in one second: backfill gives x and y seq 6 and 7, no seq equals any carried one, and listing returns a, b, c, d, e, x, y

*planning.md — Phase 1 task table row:*

| same-second-tasks-sort-by-id-1-4 | Backfill missing sequences above the file's highest on read | a file with no seq on any record orders by line position, including after a create, update or remove (§2.3, §8.2), a mixed seq/no-seq file orders its newest record last and not first (§2.3, §8.2), unnumbered records lying above numbered ones (the merge shape) are numbered above every carried sequence, never onto one (§2.3, §8.2), a record whose seq was stripped by a binary that does not know the field is reassigned correctly (§7.1, §8.2), blank lines are skipped so seq order matches record order rather than line number (§8.2), a new task numbers above the backfilled values (§2.2), the ReadTasks, Rebuild and readAndEnsureFresh paths all get the same rule (§2.3), the stored-record byte-immutability break is accepted (§3.1, §7.2) |

*Task 1-4 (tick-ecc73a) — title:*

Backfill Missing Sequences Above The File's Highest On Read

*Task 1-4 — Problem:*

**Problem**: Every existing project's `tasks.jsonl` predates the field, so none of its records carries a sequence and its same-second ties stay plan-dependent. Records without a sequence keep arriving afterwards too: from a merge of branches, or from a write by a binary that does not know the field and drops it. Numbering those by line position gets the post-merge shape wrong — records that already carry sequences numbered above their line positions would sort the newest, unnumbered record first. Numbering them from the highest sequence seen so far collides instead: a merge can leave unnumbered records above numbered ones, and a running maximum hands them numbers a record further down already carries — duplicates tick would manufacture itself, silent until the next write froze them into the file.

*Task 1-4 — Solution:*

**Solution**: As records are parsed, take the highest sequence any record in the file carries (0 when none carries one), then walk the records in order and give each record that has none the next number above it — new-task numbering applied to each unnumbered record in turn, not line position and not a running maximum. Doing it in the one parse funnel puts every read path on the same rule, and the next write makes the assignment permanent.

*Task 1-4 — Outcome:*

**Outcome**: Existing projects read back in line order, which is their authoring order, with no migration and no repair. The first mutation writes the assigned sequences into the file. Post-merge and stripped files order their newest record last, a backfilled number never equals a sequence the file already carries, and a new task numbers above the backfilled values.

*Task 1-4 — Acceptance Criteria (the existing criterion, followed by the new one):*

- [ ] A file whose first records carry sequences numbered above their line positions, followed by a newest record carrying none, all in one creation second and one priority: the newest record is assigned a number above every carried sequence and `tick list` returns it last (§2.3, §8.2)
- [ ] A file in the merge shape, every record in one creation second and one priority, with IDs contradicting the expected order — in line order `a`, `b`, `c` carrying `seq` 1, 2 and 3, then `x` and `y` carrying none, then `d` and `e` carrying 4 and 5: backfill assigns `x` and `y` sequences 6 and 7, no record's sequence equals another's, and `tick list` returns `a`, `b`, `c`, `d`, `e`, `x`, `y` (§2.3, §8.2)

*Task 1-4 — Do (second bullet):*

- The rule: take the highest sequence any record in the file carries, 0 when none carries one; then walk the records in order and give each record that has none — absent or zero — the next number above it. This is new-task numbering (§2.2) applied to each unnumbered record in turn, so a backfilled number can never equal a carried one (§2.2, §2.3).

*Task 1-4 — Context (first quote, replaced by):*

> §2.3, as corrected by the specification's 2026-09-22 corrigendum: "This numbers above the file's highest sequence, **not** by line position. On a file where some records carry a sequence and some do not — the shape produced by a merge, or by a write from a binary that does not know the field — assigning line position would sort the newest record first. Numbering above the file's highest gets it right, and it is new-task numbering (§2.2) applied to each unnumbered record in turn, so backfill and creation are a single rule." It is "also **not** a running maximum — the highest seen so far — because that collides": in the merge shape (`a`, `b`, `c` at 1–3, then `x` and `y` with none, then `d` and `e` at 4 and 5) a running maximum numbers `x` and `y` 4 and 5, duplicating `d` and `e`; numbering above the file's highest gives them 6 and 7. "The cost is that an unnumbered record lying above numbered ones sorts after them where they share a creation second." On a file with no sequences, a file stripped by an older binary, and a file whose numbered records all precede its unnumbered ones, the two rules assign identically. For a file with no sequences at all the rule yields line order, which is authoring order — verified across the project's entire history: no sort call has ever existed on the storage, task or create paths, and every JSONL writer has iterated the task slice unsorted. The merge shape's end-to-end doctor assertion after the next write is Task 3-2.

*Task 1-3 (tick-0bb4f7) — Context (end of the first quote):*

Backfill is Task 1-4; §2.3 applies this same numbering to each record lacking a sequence in turn, "so backfill and creation are a single rule".

*Task 1-5 (tick-a21794) — Context (last line):*

> §2.3: backfill is new-task numbering applied to each unnumbered record in turn, "so backfill and creation are a single rule"; `tick create`'s assignment is Task 1-3.

**Resolution**: Pending
**Notes**:

---

## Observations

- Task 3-1's Context notes that `ParentDoneWithOpenChildrenCheck`'s doc comment (`internal/doctor/parent_done_open_children.go:11-13`) calls it "the only warning-severity check", which becomes untrue once the new check lands. The spec does not decide whether that comment changes, so it stays open for the implementer.
