# Specification: Same-Second Tasks Sort By ID

## Specification

### 1. The Ordering Contract

#### 1.1 The defect

Tasks created inside one wall-clock second come back in an order unrelated to the order they were written. There is no error and no warning — the ordering is silently wrong. A human typing `tick add` repeatedly never reaches it; anything scripted reaches it every time: an agent authoring a whole phase in one pass, the migration framework importing a project, a shell loop. Machine writers are now the normal case.

Two independent defects compound. Neither alone produces the symptom; together they mean authoring order is sometimes preserved and sometimes scrambled, with nothing to tell the two cases apart.

**The ordering information is never recorded.** Creation times are stored to whole-second granularity — the single format constant governing both storage and display names no fractional part (`sed -n 41p internal/task/task.go` → `const TimestampFormat = "2006-01-02T15:04:05Z"`), and every stamping site additionally truncates before the value is held (`rg -c 'Truncate\(time\.Second\)' internal/ --glob '!*_test.go'` → 9 sites). A batch authored inside one second records the same instant for every member. The intended order is not degraded in storage — it is absent from it.

**The sort has no deterministic final term.** Both list-family clauses end at `t.created ASC` (`rg -n 'ORDER BY' internal/cli/list.go` → `:317` ready view, `:319` neutral view). SQLite is free to emit tied rows in any order, and emits them in whatever order the chosen plan feeds the sorter. Three plans were observed across measured projects: the priority index and the status index both feed rows in rowid order, which is file order — the right answer, reached by accident; the primary-key index feeds rows in task-ID order — the reported symptom, and since task IDs are three random bytes, indistinguishable from a shuffle.

**Which plan a query gets is not predictable.** It turns on data shape and statistics, not on command name or row count: `tick ready --parent` returned file order in one 8-task project and ID order in 7-task and 18-task projects on the same schema and binary, and `ANALYZE` changed nothing. The correct mental model is not "parent-filtered reads are broken" but **tie order is undefined everywhere, and currently resolves correctly by chance in most cases**. A new index, a statistics change or a SQLite version bump can flip any of it silently.

#### 1.2 Severity

`--count` turns wrong order into wrong *selection*. `LIMIT` is appended after the tied `ORDER BY` (`internal/cli/list.go:321-324`), so it cuts the arbitrary order rather than the authored one. Measured: in a five-child phase authored step-1 … step-5, `tick ready --parent P --count 1` returned **step-4**. An implementation loop asking for the next available task is handed the wrong task outright, not a list it could re-sort.

No live exposure — no stored batch is currently being read in the wrong order, and nothing is being worked around by hand.

#### 1.3 What tick guarantees after the fix

Tasks come back in the order they were created, within a priority band — and within the `in_progress` band for `ready`, which floats above the priority terms. A batch written one after another by an agent, an importer or a shell loop reads back in the sequence it was written.

The guarantee is **total**: every query in the list family produces one defined order under every condition, with no dependence on which plan SQLite chooses. A mixed-priority batch still does not read back in write order — priority and the `ready` band are semantic ordering and continue to outrank creation order.

Creation time stops being load-bearing for ordering. The timestamp format is unchanged, and no stored timestamp changes value.

### 2. The Creation Sequence

#### 2.1 The decision

Every task carries an explicit creation sequence — a monotonic number assigned when the task is created and never renumbered afterwards. The list-family sort ends on it, and it is the only thing tick relies on for authoring order.

**Why an explicit field rather than finer timestamps or file position.** A timestamp is an *observation*, not a statement of sequence: it records when something happened and implies order only as a side effect, which is precisely the accident this bug is made of. Two alternatives were explored and reversed.

Sub-second timestamp precision was ruled out by measurement. It only ever covers tasks created by separate command invocations: 100 in-process stamps taken with tick's own ID generation in the loop produce **one** distinct millisecond; a microsecond layout gives ~40 distinct values with tie groups of 3–5, nanosecond ~37 with groups of 4. The whole `internal/migrate` surface is therefore untouched by it. Already-stored tasks gain nothing either — the first write re-emits an old whole-second value as `.000` rather than inventing precision. Its costs were real: 14 top-level tests across three packages (31 subtests), a read/write format split, a fixed-width-fraction requirement that inverts order if missed, and a new failure mode precision itself introduces — clock steps and hand-edited records produce times that *differ but are wrong*, which no tiebreak can catch because there is no tie.

Position-in-file was rejected because the ordering is invisible: nobody reading a task can see why two tasks came back in that order, no direct consumer of `tasks.jsonl` can reproduce it, and correctness would rest on an invariant of an artefact the design calls expendable — one that no test states and that any future incremental cache-update path would break silently. The sequence is that option's information promoted to a first-class, assertable field.

An explicit sequence is the only option where a clash is **impossible by construction** rather than improbable, and the only one that puts the order somewhere both a reader of `tasks.jsonl` and a test can see it.

#### 2.2 Assignment

A new task takes the next number above the highest sequence currently in the file.

Clashes cannot occur. Writes are serialised by the exclusive file lock, and `Store.Mutate` (`internal/storage/store.go:174`) reads the current task set inside that lock before calling the mutation function, so the next number is computed with no race.

**Numbers are not guaranteed unreused.** Removing the highest-numbered task frees its number for the next creation. Ordering does not need the guarantee — a reused number cannot collide, because the task that held it is gone.

#### 2.3 Backfill for records with no sequence

A record lacking a sequence is assigned one on read: walk the records in order, tracking the highest sequence seen so far, and give the next number to any record that has none.

This is a running-maximum rule, **not** line position. On a file where some records carry a sequence and some do not — the shape produced by a merge, or by a write from a binary that does not know the field — assigning line position would sort the newest record first. Running maximum gets it right and collapses backfill and new-task numbering into a single rule.

For a file with no sequences at all this yields line order, which *is* authoring order. Verified across the project's entire history: `git log -S "sort." -- internal/storage/ internal/task/ internal/cli/create.go` returns no commits — no sort call has ever existed on those paths — and the first `MarshalJSONL` (commit `4278ba09`) iterated the task slice unsorted exactly as today.

Backfill lives in `ParseJSONL` (`internal/storage/jsonl.go:89`), which is the single funnel for every read path (`rg -n 'ParseJSONL' internal/storage/store.go` → `:164` `ReadTasks`, used by `dep tree`; `:246` `Rebuild`; `:363` `readAndEnsureFresh`, used by every query and mutation). Anywhere else leaves one of those paths on a different rule.

Existing projects need no migration and no repair. The assignment becomes permanent on the next write.

#### 2.4 The sequence is not surfaced in command output

The sequence appears in `tasks.jsonl` and in the cache, and nowhere a command prints. No detail-document section, no field registry entry, no `--field` address, no README sample. It is an ordering mechanism, not information about the work; the guarantee it delivers is stated in the documentation (§6) and the one case where the sequence itself is worth inspecting is covered by the doctor check (§5.2).

### 3. Storage and Cache Representation

#### 3.1 The record

The sequence is a field on the stored task record in `tasks.jsonl`, alongside the existing fields (`sed -n '44,60p' internal/task/task.go` → the `Task` struct and its JSON tags). It round-trips through the file unchanged: written, read back, identical.

**Adding the field breaks an invariant currently asserted in the suite.** `internal/cli/migrate_test.go:702` asserts "it leaves values already in storage untouched"; every record gains the new field on the next write, so that test changes with this work. With `.tick/tasks.jsonl` git-tracked in this project, the first post-upgrade write produces a whole-file diff — accepted as a one-off.

#### 3.2 The cache

The `tasks` table gains a column for the sequence (`sed -n '17,28p' internal/storage/cache.go` → the current ten-column definition), populated by the rebuild insert (`internal/storage/cache.go:137`). The list-family sort reads it from there.

The cache column is asserted directly by test, not only through an ordering result — an ordering assertion alone cannot distinguish a working sequence from an accidentally-correct query plan (§1.1).

#### 3.3 Dependency declaration order

The `dependencies` table gains an ordinal carrying each blocker's position in the record's `blocked_by` array (`sed -n '31,35p' internal/storage/cache.go` → the current two-column definition, whose primary key is the pair), populated by the dependency insert (`internal/storage/cache.go:143`). This is what `tick show`'s blocker list orders on (§4.3).

The array order is **explicit in the record** — a JSON array is ordered — and is lost only because the cache flattens it into a junction table. Carrying it across applies the same principle driving the whole fix: the order is in the record, so the cache should carry it rather than the query inventing one. `tick dep tree` already emits edges in this order, reading the task slice directly (`internal/cli/dep_tree_graph.go:186-198` — "one edge per `BlockedBy` entry, tasks in slice order and each task's blockers in stored order").

#### 3.4 Schema version

`schemaVersion` goes from 2 to 3 (`sed -n 15p internal/storage/cache.go` → `const schemaVersion = 2`).

No migration is needed. `ensureFresh` checks the version before every query and before every mutation; a mismatch deletes the cache file, recreates it and rebuilds from `tasks.jsonl`. The two new columns arrive through that existing path.

---

## Working Notes
