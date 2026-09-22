# Review Tracking: Same-Second Tasks Sort By ID - Input Review

## Findings

### 1. The record field has no name

**Source**: Investigation → Fix Direction → Chosen Approach, resolution 1 (`priority`, `created`, `seq`, `id`) and the downgrade paragraph ("an older tick unmarshals a record carrying an unknown `seq` field without error")
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §3.1 The record, §3.2 The cache

**Problem**:
The whole argument for an explicit sequence over file position is that the order lands somewhere a person reading `tasks.jsonl` can see it. A reader needs a key name to look for, and nothing states one — so the file could come out carrying `sequence`, `creation_seq` or `createdSeq`. Whichever is written is permanent: it is baked into every record of every existing project on the first write after the upgrade, and changing it later re-runs the whole-file diff and the backfill on data that no longer matches.

**Proposal**:
Name the JSON key `seq` and the cache column likewise. The sources use `seq` consistently — in the sort terms `priority, created, seq, id` and in the downgrade verification, which describes an older binary dropping "an unknown `seq` field".

**Current**:
§3.1, first paragraph:
> The sequence is a field on the stored task record in `tasks.jsonl`, alongside the existing fields (`sed -n '44,60p' internal/task/task.go` → the `Task` struct and its JSON tags). It round-trips through the file unchanged: written, read back, identical.

§3.2, first sentence:
> The `tasks` table gains a column for the sequence (`sed -n '17,28p' internal/storage/cache.go` → the current ten-column definition), populated by the rebuild insert (`internal/storage/cache.go:137`).

**Proposed Text**:
§3.1, first paragraph:
> The sequence is the `seq` field on the stored task record in `tasks.jsonl`, alongside the existing fields (`sed -n '44,60p' internal/task/task.go` → the `Task` struct and its JSON tags). It round-trips through the file unchanged: written, read back, identical.

§3.2, first sentence:
> The `tasks` table gains a `seq` column (`sed -n '17,28p' internal/storage/cache.go` → the current ten-column definition), populated by the rebuild insert (`internal/storage/cache.go:137`).

**Resolution**: Approved
**Notes**: Applied to §3.1 and §3.2 as staged — the sources name the field, so the derivation is the record's own.

---

### 2. Numbering a write that creates more than one task

**Source**: Investigation → Fix Direction → Chosen Approach ("Why clashes cannot occur"), resolution 2 ("collapses backfill and new-task numbering into a single rule"), H4 (`migrate/store_creator.go` "appends per task" within one import), Testing Recommendations ("An in-process batch, not just separate invocations — the migration framework is the natural fixture")
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §2.2 Assignment

**Problem**:
An import creates every one of its tasks inside a single write. Read as "the highest sequence in the file", the numbering rule is evaluated once against the file as it stood before that write, so every imported task is handed the same number. The imported project then comes back in random ID order — exactly the defect being fixed, untouched for the one writer that hits it hardest — and `tick doctor` reports the entire import as one duplicate group. A user importing a project sees the fix do nothing for them.

**Proposal**:
State that numbering advances within the write: each new task takes the next number above the highest sequence in the task set being written, tasks created earlier in the same mutation included. This is the running-maximum rule the sources already adopt for backfill, which they note collapses backfill and new-task numbering into a single rule — applied to the batch case the migration framework creates.

**Current**:
> A new task takes the next number above the highest sequence currently in the file.
>
> Clashes cannot occur. Writes are serialised by the exclusive file lock, and `Store.Mutate` (`internal/storage/store.go:174`) reads the current task set inside that lock before calling the mutation function, so the next number is computed with no race.

**Proposed Text**:
> A new task takes the next number above the highest sequence in the task set being written. A write that creates several tasks at once — an import through `internal/migrate` is the ordinary case — numbers them in turn, each above the tasks created before it in the same write, so a batch is numbered in the order it was authored.
>
> Clashes cannot occur. Writes are serialised by the exclusive file lock, and `Store.Mutate` (`internal/storage/store.go:174`) reads the current task set inside that lock before calling the mutation function, so the next number is computed with no race.

**Resolution**: Declined
**Notes**: Premise measured false. The migration framework does not create its tasks in one write — `StoreTaskCreator.CreateTask` (`internal/migrate/store_creator.go:36`) opens its own `store.Mutate` per task, so every imported task is numbered against a file that already contains its predecessors. Every `Mutate` call site in the tree (`create.go:187`, `dep.go:78,156`, `remove.go:184`, `update.go:275`, `transition.go:36`, `note.go:68,124`, `store_creator.go:36`) creates at most one task. With no multi-task write path in existence, "the highest sequence currently in the file" is unambiguous, and a rule for a path that does not exist is the builder's to settle if one is ever added.

---

### 3. Nothing distinguishes "no sequence" from the first sequence

**Source**: Investigation → Fix Direction → resolution 2 ("assign the next number to any record lacking one") — the sources set no starting value and no marker for an absent sequence
**Category**: Gap/Ambiguity
**Move**: settled
**Affects**: §2.3 Backfill for records with no sequence, §2.2 Assignment

**Problem**:
Backfill has to tell a record that carries no sequence from one that carries the very first sequence ever issued. If numbering starts at zero, those two are the same value on every read: the first task a project ever created is treated as unnumbered and pushed behind tasks written after it, and on a file where later records do carry sequences it is handed a number one of them already holds. The user sees their oldest task drift to the end of the list, and `tick doctor` reports a duplicate on a file nobody merged or hand-edited.

**Proposal**:
Sequences are positive — numbering begins at 1, and an absent or zero value means the record has no sequence, with the running maximum starting at 0. The sources set no starting value, so this is a call this session makes: what leans it is that the backfill rule cannot function without the distinction, and a zero-valued sequence vanishes from the record under Go's `omitempty`, so the first task a project ever created would be re-backfilled to a different number on every read until a write froze it wrong. The alternative that also fits is a nullable field (`*int`), distinguishing absent from zero without reserving a value — rejected because it adds a nullable field to the record for no gain the reader can see, where 1-based numbering falls out of the backfill rule's own running maximum starting at 0.

**Current**:
> A record lacking a sequence is assigned one on read: walk the records in order, tracking the highest sequence seen so far, and give the next number to any record that has none.

**Proposed Text**:
§2.2, first paragraph (the value domain's home — §2.2 owns assignment, so the rule is stated there once rather than restated in §2.3):
> A new task takes the next number above the highest sequence currently in the file. Sequences are positive: numbering begins at 1, so an absent or zero value on a record means it carries no sequence.

§2.3, first paragraph:
> A record lacking a sequence is assigned one on read: walk the records in order, tracking the highest sequence seen so far — starting at 0 — and give the next number to any record that has none.

**Resolution**: Routed
**Notes**: The call is this session's, so it landed first in the investigation (Fix Direction → resolution 2), which now states the positive value domain and the `*int` alternative it was preferred over; the specification is re-aligned to it. Placement revised against the one-home rule: the value domain sits in §2.2, which owns assignment, rather than being restated in §2.3.

---

### 4. `tick show`'s children are left plan-dependent under a duplicate sequence

**Source**: Investigation → Fix Direction → resolution 1 ("Task ID becomes an absolute final sort term … Order is then total under every condition") read against resolution 5 (children sort by `created, seq` only)
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §4.3 `tick show`'s sub-lists, §5.1 Task ID is the absolute final sort term, §8.3 Duplicate sequences

**Problem**:
Two branches numbering from the same maximum merge into two children of one parent sharing a sequence. `tick list` and `tick ready` handle that deterministically, but `tick show` on the parent orders its children by created and sequence alone, so they emerge in whatever order the query plan feeds the sorter — and that can change between runs, after a `tick rebuild`, or with a SQLite version bump, silently. The user gets a stable order from the list commands and a shifting one from the command they use to read a phase's children, with nothing to tell them why.

**Proposal**:
End the children clause on the task ID, the same absolute final term the list clauses carry, so no condition leaves the sub-list order undefined. Blockers need no extra term — the dependency ordinal is unique within one task's blocker set by construction. The sources hold both sides of this and settle neither: resolution 1 promises order "total under every condition", resolution 5 gives children `created, seq` only. The fork has one live side — the other is leaving a user-facing list plan-dependent after a merge, which nothing recommends and which the duplicate-sequence resolution exists to rule out.

**Current**:
§4.3, children paragraph:
> **Children** (`internal/cli/show.go:146`) order by created, then sequence. **Priority does not participate** — today's clause has no priority term, and adding one would be a second, unrequested behaviour change. A child with better priority does not float above an earlier-created sibling.

§5.1, first sentence:
> Every clause in §4.1 ends on the task ID.

**Proposed Text**:
§4.3, children paragraph:
> **Children** (`internal/cli/show.go:146`) order by created, then sequence, then task ID — the same absolute final term the list clauses carry (§5.1), so a duplicate sequence cannot leave the list plan-dependent. **Priority does not participate** — today's clause has no priority term, and adding one would be a second, unrequested behaviour change. A child with better priority does not float above an earlier-created sibling.

§5.1, first sentence:
> Every clause in §4.1 ends on the task ID, as does `tick show`'s children clause (§4.3). Blockers order on the dependency ordinal, which is unique within a task's blocker set, so they are total already.

§8.3, new bullet after the first:
> - `tick show`'s children under a shared sequence are deterministic too — the same ID tiebreak, asserted on the parent's detail document.

**Resolution**: Routed
**Notes**: The call is this session's, so it landed first in the investigation — resolution 1 now carries the final term through to `show`'s children and states why blockers need none, resolution 5's children key becomes `created, seq, id`, the Chosen Approach restatement and the testing recommendations follow — and the specification is re-aligned to it across §4.3, §5.1 and §8.3.

---

### 5. Whether the sequence surfaces in command output is decided nowhere but the specification

**Source**: No source decides this — Investigation → Fix Direction → Chosen Approach closes with "**Open question left for specification:** whether the sequence surfaces in command output or stays an internal ordering key visible only in `tasks.jsonl`"
**Category**: Unsourced decision
**Move**: route
**Affects**: §2.4 The sequence is not surfaced in command output

**Problem**:
Whether a user can ask for the sequence — a `--field seq` address, a line in the detail document, a documented key — is a visible product call, and the investigation records it as open rather than answered. The specification answers it (nowhere a command prints), and that answer carries real consequence in both directions: hidden, a user who suspects an ordering problem has only `tasks.jsonl` and the doctor check to look at; surfaced, every detail document grows a section and the conformance inventory and README samples move with it. The record that is supposed to hold the decision does not hold it, so nothing outside this specification stands behind the call.

**Resolution**: Routed
**Notes**: Derived rather than asked — the fork whittles to one answer over the decisions the record made. The call landed in the investigation (Fix Direction → Chosen Approach), which now decides the sequence does not surface in command output and records both the derivation and the alternative that also fit. §2.4 already states exactly that, so the specification needed no change to align.

---

## Observations

- The sources name the new diagnostic `DuplicateSeqCheck`, mirroring `DuplicateIdCheck`; the specification describes it without naming it.
- Import chronology (§4.2) is preserved only to whole-second granularity — a provider's RFC3339 fraction is flattened on write (`beads.go:119` → `FormatTimestamp`), so two imported tasks inside one second tie on `created` and fall to import order.
- The sources' H6 detail — the cache's `created` column is TEXT and compares lexically, where `'Z'` sorts above `'.'` — is the mechanical reason the sub-second route needed a fixed-width fraction; §2.1 carries the requirement without the reason.
