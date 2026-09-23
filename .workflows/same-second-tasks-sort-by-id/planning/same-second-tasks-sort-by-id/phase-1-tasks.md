# Phase 1: Creation sequence and total list-family ordering — 5 tasks

## same-second-tasks-sort-by-id-1-1

### Task 1-1: Sort same-second tasks by creation sequence

**Problem**: Tasks created inside one wall-clock second come back from `tick list`, `tick ready` and `tick blocked` in an order unrelated to the order they were written, with no error and no warning. Two defects compound. Creation times are stored to whole seconds, so a batch authored inside one second records the same instant for every member and the intended order is absent from storage. Both list-family clauses end at `t.created ASC`, so SQLite emits tied rows in whatever order the chosen plan feeds the sorter: file order under the priority and status indexes (right by accident), task-ID order under the primary-key index (a shuffle, since IDs are three random bytes). Which plan a query gets turns on data shape, not command or row count. `LIMIT` then cuts that arbitrary order: in a five-child phase authored step-1 … step-5, `tick ready --parent P --count 1` returned step-4, handing an implementation loop the wrong next task outright.

**Solution**: Give the stored task record an explicit creation sequence under the key `seq`, carry it into a new `seq` column of the cache's `tasks` table at cache schema version 3, and add it as the sort term after `created` in both list-family clauses, so tasks tied on creation second come back in sequence order.

**Outcome**: For records that carry sequences, `tick list`, `tick list --parent`, `tick ready` and `tick ready --count` return same-second tasks in sequence order under every query plan. An existing v2 cache rebuilds itself at v3 with the `seq` column populated. The field round-trips through `tasks.jsonl` unchanged and appears in no command output.

**Acceptance Criteria**:
- [ ] A project holding a parent P and five open children step-1 … step-5 that record one creation second and one priority, carry sequences in authoring order, and have IDs whose ascending order contradicts authoring order: `tick list` returns step-1 … step-5 in authoring order (§1.3, §8.1)
- [ ] Same project: `tick list --parent P` returns step-1 … step-5 in authoring order (§8.1)
- [ ] Same project: `tick ready --parent P --count 1` returns step-1, not the ID-lowest child (§1.2, §8.1)
- [ ] Tasks recording one creation second and one priority, whose sequences contradict both their line order in `tasks.jsonl` and their ascending-ID order: `tick list` returns them in sequence order (§2.1, §3.2, §8.2)
- [ ] A project whose `cache.db` was built at schema version 2 — a `tasks` table with no `seq` column and `schema_version` 2 in the metadata table: the next `tick list` succeeds, and afterwards `cache.db` reports `schema_version` 3 and its `tasks` table's `seq` column holds each task's sequence, read from the table directly rather than inferred from an ordering (§3.2, §3.4, §8.2)
- [ ] A task carrying a sequence is written to `tasks.jsonl`, read back and written again: the task read back carries the same sequence and the two writes are byte-identical (§3.1, §8.2)
- [ ] No command output carries the sequence: `tick show <id>` under toon, pretty and JSON has no `seq` key or line, and `tick show <id> --field seq` fails as an unknown field name (§2.4)
- [ ] The existing priority-then-created ordering tests (`ready_test.go:306`, `blocked_test.go:212`, `list_filter_test.go:368`) pass unmodified (§8.5)

**Do**:
- `internal/task/task.go` — the `Task` struct and its JSON serialisation form carry the sequence, stored under the key `seq` on the task record alongside the existing fields (§3.1).
- `internal/storage/cache.go` — the `tasks` table in `schemaSQL` (the ten-column definition at lines 17–28) gains a `seq` column, populated by the rebuild insert (line 137). `schemaVersion` goes from 2 to 3 (line 15). No migration: `ensureFresh` in `internal/storage/store.go` already deletes, recreates and rebuilds a cache whose version mismatches, before every query and every mutation (§3.4).
- `internal/cli/list.go` — in `buildListQuery`, the neutral clause (line 319) orders by priority, created, sequence; the ready clause (line 317) orders by the `in_progress` band, priority, created, sequence. `LIMIT` stays appended after the `ORDER BY` (lines 321–324).

**Context**:
> §2.1: every task carries an explicit creation sequence, "a monotonic number assigned when the task is created and never renumbered afterwards". Sub-second timestamps and position-in-file were both rejected; the sequence is the only option "where a clash is impossible by construction rather than improbable, and the only one that puts the order somewhere both a reader of `tasks.jsonl` and a test can see it". Sequences are positive: numbering begins at 1, and an absent or zero value on a record means it carries no sequence (§2.2).
>
> §4.2: `created` outranks `seq`. Two tasks whose recorded creation dates differ are ordered by those dates; the sequence speaks only when they tie.
>
> §3.2: "The cache column is asserted directly by test, not only through an ordering result — an ordering assertion alone cannot distinguish a working sequence from an accidentally-correct query plan (§1.1)." The unfiltered listing already returns file order by accident today, which is why one scenario above has sequence order disagree with line order.
>
> §8 fixture constraints, binding every ordering fixture: IDs must contradict the authoring order; creation seconds must tie wherever the sequence is what is under test; fixture size must not be load-bearing — assertions state the required order, never the plan.
>
> §7.1: timestamp precision is out of scope. `TimestampFormat` (`internal/task/task.go:41`) is unchanged, the nine `Truncate(time.Second)` calls stay, and no stored or displayed timestamp changes value.
>
> Sequencing within the phase: this task's fixtures write sequences into `tasks.jsonl` directly. Assignment on `tick create` is Task 1-3, backfill of records without a sequence is Task 1-4, and the final task-ID sort term (§5.1) is Task 1-2. Phase 2 adds the dependency ordinal (§3.3) to the same version-3 schema.
>
> `internal/storage/cache_test.go`'s "it triggers rebuild on schema version mismatch" pins `CurrentSchemaVersion()` at 2 and moves with the bump.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §1.1, §1.2, §1.3, §2.1, §2.2, §2.4, §3.1, §3.2, §3.4, §4.1, §4.2, §7.1, §8 (fixture constraints), §8.1, §8.2, §8.5

## same-second-tasks-sort-by-id-1-2

### Task 1-2: Keep the order total across the ready band, blocked and duplicate sequences

**Problem**: Ending the sort on the sequence makes the order total only while sequences are unique, and in this project they are not guaranteed to be. `.tick/tasks.jsonl` is git-tracked and implementation runs on branches in worktrees, so two branches each numbering from the same maximum merge into duplicates. Seven children forced to one sequence were demonstrated returning plan-dependent order with no warning — the same defect, one key further down. Two list-family routes also need their own guard. `tick ready` puts the `in_progress` band above the priority terms, and a new term inserted before the band would pass every other ordering assertion. `tick blocked` reaches the sorter through the different `WHERE` shape `BlockedConditions()` builds (`internal/cli/query_helpers.go:66-80`).

**Solution**: End both list-family clauses on the task ID after the sequence, making it the absolute final sort term. Pin the ready band, the blocked view and the duplicate-sequence fallback with tie assertions of their own.

**Outcome**: Every list-family query produces one defined order under every condition. Tasks sharing a sequence and a creation second fall back to ascending task-ID order under both the filtered and unfiltered plans, the same on every run. An `in_progress` task stays at the top of `tick ready` whatever its sequence, `tick blocked` returns same-second tasks in authoring order, and every existing ordering test passes unchanged.

**Acceptance Criteria**:
- [ ] Open tasks recording one creation second and one priority, plus an `in_progress` task of the same priority authored after them in the same second, all with IDs contradicting authoring order: `tick ready` lists the `in_progress` task first, then the open tasks in authoring order (§1.3, §8.1)
- [ ] Tasks blocked by a common open blocker, recording one creation second and one priority, with IDs contradicting authoring order: `tick blocked` returns them in authoring order (§4.1, §8.1)
- [ ] Children of a parent P recording one creation second and one priority and all carrying the same non-zero sequence, with IDs whose ascending order contradicts their line order in `tasks.jsonl`: `tick list` and `tick list --parent P` both return them in ascending task-ID order, identically on repeated runs (§5.1, §8.3)
- [ ] The existing priority-then-created ordering tests (`ready_test.go:306`, `blocked_test.go:212`, `list_filter_test.go:368`) pass unmodified (§7.3, §8.5)

**Do**:
- `internal/cli/list.go` — in `buildListQuery`, both clauses end on the task ID after the sequence: the neutral clause (line 319) orders by priority, created, sequence, id; the ready clause (line 317) by the `in_progress` band, priority, created, sequence, id, with the band term still first (§4.1, §5.1).

**Context**:
> §5.1: "Every clause in §4.1 ends on the task ID … Order is then total under every condition, so the worst a duplicate sequence can do is fall back to today's ID ordering **deterministically**, rather than varying with the query plan. The current failure is that tie order changes with the plan; after this change it cannot."
>
> §4.1: `tick list`, `tick ready` and `tick blocked`, filtered and unfiltered alike, all run through `buildListQuery`. "`tick blocked` needs its own tie assertion regardless: `BlockedConditions()` builds a different `WHERE` shape … so it reaches the sorter by a different route."
>
> §8.1: "The `ready` band still wins over the sequence: an `in_progress` task authored *after* tied `open` tasks stays at the top. Without this, a final term inserted before the band term passes everything else."
>
> §7.3 / §8.5: a final sort term cannot reorder any result whose earlier keys already differ, verified empirically by adding the terms and observing zero new failures — so existing ordering tests stay green unchanged.
>
> §8 fixture constraints apply. For the duplicate case the unfiltered plan can return file order by accident, so the fixture's line order must not coincide with ascending-ID order or the unfiltered assertion proves nothing. The shared sequence must be non-zero: a zero value means "no sequence" (§2.2) and is backfilled to distinct numbers once Task 1-4 lands.
>
> A duplicate sequence is made visible by a doctor check in Phase 3 (§5.2); `tick show`'s children clause gains the same task-ID final term in Phase 2 (§4.3).

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §1.3, §4.1, §5, §5.1, §7.3, §8 (fixture constraints), §8.1, §8.3, §8.5

## same-second-tasks-sort-by-id-1-3

### Task 1-3: Assign a sequence when a task is created

**Problem**: The sort reads the sequence, but `tick create` still writes new tasks without one. A batch written one after another by an agent or a shell loop — the normal case now that machine writers dominate — therefore records one creation second and no sequence, and still reads back in plan-dependent order.

**Solution**: When `tick create` builds a new task, give it the next number above the highest sequence in the task set it read under the exclusive lock. Existing tasks are never renumbered.

**Outcome**: Every task `tick create` writes carries a positive sequence one above the highest in the file at that moment. A task's sequence never changes afterwards, through update, remove or rebuild. Tasks created one after another inside one second list in creation order.

**Acceptance Criteria**:
- [ ] In an empty project, three successive `tick create` calls write tasks carrying sequences 1, 2 and 3 in creation order — numbering begins at 1 and no created task is ever written with sequence 0 (§2.2, §8.2)
- [ ] A project whose records carry sequences 3, 7 and 5, in that line order: `tick create` writes the new task with sequence 8 — one above the highest, not above the last line's value and not the record count (§2.2)
- [ ] A task created by `tick create` keeps the same sequence in `tasks.jsonl` and in the cache's `seq` column after a `tick update` of that task, a `tick remove` of another task, further `tick create` calls, and `tick rebuild` (§2.1, §8.2)
- [ ] Tasks A, B and C created in that order carry 1, 2 and 3; after `tick remove` of C, the next `tick create` writes sequence 3, and no two tasks in the file share a sequence (§2.2, §7.2)
- [ ] Tasks created by successive `tick create` calls that record the same creation second and the same priority: `tick list` returns them in creation order, a result ascending task-ID order could not produce (§1.3, §8.1)

**Do**:
- `internal/cli/create.go` — the new task is built inside `Store.Mutate` (around line 208). `Store.Mutate` (`internal/storage/store.go:174`) reads the current task set under the exclusive file lock before calling the mutation function, so the next number is computed from that set with no race (§2.2).

**Context**:
> §2.2: "A new task takes the next number above the highest sequence currently in the file. Sequences are positive: numbering begins at 1, so an absent or zero value on a record means it carries no sequence. The highest is taken from the task set as read, not from the stored bytes: backfill (§2.3) runs first, so records that reached the file without a sequence already carry one and the next number sits above those too." Backfill is Task 1-4; §2.3 applies this same numbering to each record lacking a sequence in turn, "so backfill and creation are a single rule".
>
> §2.2 / §7.2: "Numbers are not guaranteed unreused. Removing the highest-numbered task frees its number for the next creation. Ordering does not need the guarantee — a reused number cannot collide, because the task that held it is gone." This is a dropped claim, not a defect.
>
> §8.2: the survival invariant is phrased "sequence order equals record order", not "equals line number".
>
> §8 fixture constraints apply to the ordering scenario: creation seconds must tie, and `tick create` generates random IDs, so the assertion must be one an ascending-ID result could not satisfy.
>
> Tasks created by the migration framework (`internal/migrate`) are Task 1-5.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §1.3, §2.1, §2.2, §2.3, §7.2, §8 (fixture constraints), §8.1, §8.2

## same-second-tasks-sort-by-id-1-4

### Task 1-4: Backfill missing sequences above the file's highest on read

**Problem**: Every existing project's `tasks.jsonl` predates the field, so none of its records carries a sequence and its same-second ties stay plan-dependent. Records without a sequence keep arriving afterwards too: from a merge of branches, or from a write by a binary that does not know the field and drops it. Numbering those by line position gets the post-merge shape wrong — records that already carry sequences numbered above their line positions would sort the newest, unnumbered record first. Numbering them from the highest sequence seen so far collides instead: a merge can leave unnumbered records above numbered ones, and a running maximum hands them numbers a record further down already carries — duplicates tick would manufacture itself, silent until the next write froze them into the file.

**Solution**: As records are parsed, take the highest sequence any record in the file carries (0 when none carries one), then walk the records in order and give each record that has none the next number above it — new-task numbering applied to each unnumbered record in turn, not line position and not a running maximum. Doing it in the one parse funnel puts every read path on the same rule, and the next write makes the assignment permanent.

**Outcome**: Existing projects read back in line order, which is their authoring order, with no migration and no repair. The first mutation writes the assigned sequences into the file. Post-merge and stripped files order their newest record last, a backfilled number never equals a sequence the file already carries, and a new task numbers above the backfilled values.

**Acceptance Criteria**:
- [ ] A `tasks.jsonl` with no `seq` on any record, every record recording one creation second and one priority, IDs contradicting line order: `tick list` returns line order, and still does after a `tick update`, a `tick remove` and a `tick create`; from the first of those writes on, every record in the file carries a sequence and the sequences follow record order (§2.3, §8.2)
- [ ] Running `tick list` on that file leaves `tasks.jsonl` byte-identical — the assignment reaches the file only on the next write (§2.3)
- [ ] A no-`seq` file of three records: `tick create` writes the new task with sequence 4 (§2.2)
- [ ] A file whose first records carry sequences numbered above their line positions, followed by a newest record carrying none, all in one creation second and one priority: the newest record is assigned a number above every carried sequence and `tick list` returns it last (§2.3, §8.2)
- [ ] A file in the merge shape, every record in one creation second and one priority, with IDs contradicting the expected order — in line order `a`, `b`, `c` carrying `seq` 1, 2 and 3, then `x` and `y` carrying none, then `d` and `e` carrying 4 and 5: backfill assigns `x` and `y` sequences 6 and 7, no record's sequence equals another's, and `tick list` returns `a`, `b`, `c`, `d`, `e`, `x`, `y` (§2.3, §8.2)
- [ ] A file whose records once carried sequences, rewritten by a binary that did not know the field — no `seq` on any record, line order kept, a record that binary created appended last — all in one creation second and one priority: `tick list` returns the original order with the appended record last (§7.1, §8.2)
- [ ] A no-`seq` file with blank lines between its records: `tick list` returns record order, and the assigned sequences follow record order rather than line numbers (§8.2)
- [ ] A file whose records carry `seq` 0 orders and backfills exactly as one whose records carry no `seq` field (§2.2)
- [ ] The same no-`seq` file read through `Store.ReadTasks` (the read-only path `tick dep tree` uses), through `tick rebuild` and through `tick list` yields the same sequence for every record, in the tasks returned and in the cache's `seq` column (§2.3)
- [ ] A beads import into a project holding a record with no sequence: afterwards that record's stored values are unchanged apart from the sequence it gained (§3.1, §7.2)

**Do**:
- `internal/storage/jsonl.go` — backfill lives in `ParseJSONL` (line 89), the single funnel for every read path: `ReadTasks` (`internal/storage/store.go:164`, used by `dep tree`), `Rebuild` (`:246`) and `readAndEnsureFresh` (`:363`, used by every query and mutation). Placing it anywhere else leaves one of those paths on a different rule (§2.3).
- The rule: take the highest sequence any record in the file carries, 0 when none carries one; then walk the records in order and give each record that has none — absent or zero — the next number above it. This is new-task numbering (§2.2) applied to each unnumbered record in turn, so a backfilled number can never equal a carried one (§2.2, §2.3).
- `internal/cli/migrate_test.go:702` ("it leaves values already in storage untouched") changes with this work: the seeded record gains a sequence on the import's write (§3.1).

**Context**:
> §2.3, as corrected by the specification's 2026-09-22 corrigendum: "This numbers above the file's highest sequence, **not** by line position. On a file where some records carry a sequence and some do not — the shape produced by a merge, or by a write from a binary that does not know the field — assigning line position would sort the newest record first. Numbering above the file's highest gets it right, and it is new-task numbering (§2.2) applied to each unnumbered record in turn, so backfill and creation are a single rule." It is "also **not** a running maximum — the highest seen so far — because that collides": in the merge shape (`a`, `b`, `c` at 1–3, then `x` and `y` with none, then `d` and `e` at 4 and 5) a running maximum numbers `x` and `y` 4 and 5, duplicating `d` and `e`; numbering above the file's highest gives them 6 and 7. "The cost is that an unnumbered record lying above numbered ones sorts after them where they share a creation second." On a file with no sequences, a file stripped by an older binary, and a file whose numbered records all precede its unnumbered ones, the two rules assign identically. For a file with no sequences at all the rule yields line order, which is authoring order — verified across the project's entire history: no sort call has ever existed on the storage, task or create paths, and every JSONL writer has iterated the task slice unsorted. The merge shape's end-to-end doctor assertion after the next write is Task 3-2.
>
> §2.3: "Existing projects need no migration and no repair. The assignment becomes permanent on the next write."
>
> §7.1: a binary that does not know the field drops it on write but preserves line order, and the backfill restores it on the next read. Only one binary is ever in use and nothing depends on that property, but §8.2 asserts the stripped case.
>
> §7.2 accepted risks: a `tasks.jsonl` whose lines were reordered by hand or by a git merge is backfilled in the wrong order, silently and permanently once the next write freezes it — the Phase 3 doctor check catches collisions, not reorderings. Every record gains a field on the next write; with `.tick/tasks.jsonl` git-tracked in this project, the first post-upgrade write produces a whole-file diff, accepted as a one-off (§3.1). No stored timestamp changes value (§1.3, §7.1).
>
> §8 fixture constraints: records in these fixtures record one creation second. With distinct seconds the backfill assertions "would pass over a backfill that gave every record the same number", because a recorded creation date outranks the sequence (§4.2).
>
> §8.2: the invariant is "sequence order equals record order", not "equals line number" — `ParseJSONL` skips blank lines (`internal/storage/jsonl.go:99-101`).

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §1.3, §2.2, §2.3, §3.1, §4.2, §7.1, §7.2, §8 (fixture constraints), §8.2

## same-second-tasks-sort-by-id-1-5

### Task 1-5: Keep import order for in-process migration batches

**Problem**: `tick migrate` is the in-process batch writer: one run creates every imported task back to back, so a source that carries no creation times of its own stamps the whole batch with one second. Finer timestamps would not have separated them — 100 in-process stamps taken with tick's own ID generation in the loop produced one distinct millisecond. Unless each imported task carries a sequence, such a batch reads back in plan-dependent order. Imports that do carry historical creation times must keep that chronology rather than have import order override it.

**Solution**: Give each task the migration framework creates the next sequence above the highest in the file, the same rule `tick create` follows, and leave provider-supplied creation times as they are — so recorded chronology outranks the sequence and whole-second ties fall to import order.

**Outcome**: An import whose source has no creation times lists in import order. An import with real creation times lists by those times, to whole-second resolution. Imported tasks that share a second list in import order, whatever their source fractions said.

**Acceptance Criteria**:
- [ ] A beads source of five issues carrying no `created_at` and one priority, imported by a single `tick migrate` run: the stored tasks record one identical creation second, carry sequences in import order, and `tick list` returns them in import order — a result ascending task-ID order could not produce (§1.3, §8.1)
- [ ] A beads import into a project that already holds tasks: each imported task carries a sequence above the project's highest, increasing in import order (§2.2)
- [ ] Issues of one priority whose `created_at` values fall in different seconds and run against import order: `tick list` returns them by those creation times, not by import order (§1.3, §4.2)
- [ ] Issues of one priority whose `created_at` values share one second but carry fractions that run against import order: the stored creation times are whole-second and equal, and `tick list` returns the tasks in import order (§4.2)

**Do**:
- `internal/migrate/store_creator.go` — `StoreTaskCreator.CreateTask` builds each imported task inside its own `Store.Mutate`, which reads the current task set under the exclusive lock, so the next number is computed from that set (§2.2).
- Creation times stay as the provider supplies them: the beads provider parses RFC3339 (`internal/migrate/beads/beads.go:119`), and a fraction is flattened on write as it is today. Timestamp handling does not change (§4.2, §7.1).

**Context**:
> §1.3: "A batch written one after another by an agent or a shell loop reads back in the sequence it was written, as does an import whose source carries no creation times of its own. An import that supplies real historical creation times is ordered by those times instead, to whole-second resolution: recorded chronology outranks the sequence (§4.2)."
>
> §4.2: keeping `created` above `seq` "preserves true chronology on imports, where the provider supplies real historical timestamps … and import order would otherwise override them". "The chronology carried across is whole-second: a fraction in the source is flattened on write as it is today, so imported tasks that share a second tie on `created` and fall to the sequence — within one second the import's own order decides, not the source's."
>
> §2.1: sub-second precision "only ever covers tasks created by separate command invocations … The whole `internal/migrate` surface is therefore untouched by it" — the sequence is what orders an in-process batch.
>
> §8.1: "An in-process batch, not only separate invocations — the migration framework is the natural fixture, with the explicit note that its timestamps are identical."
>
> §8 fixture constraints apply. The migration framework generates the task IDs itself, so the fixture cannot choose them; the assertion must still be one an ascending-ID result could not satisfy.
>
> §2.3: backfill is new-task numbering applied to each unnumbered record in turn, "so backfill and creation are a single rule"; `tick create`'s assignment is Task 1-3.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §1.3, §2.1, §2.2, §2.3, §4.2, §7.1, §8 (fixture constraints), §8.1
