# Phase 2: tick show sub-lists in creation and declaration order — 3 tasks

## same-second-tasks-sort-by-id-2-1

### Task 2-1: Order show's children by creation, not task ID

**Problem**: `tick show` lists a task's children in task-ID order (`internal/cli/show.go:146`, `ORDER BY id`), unconditionally. Task IDs are three random bytes, so a parent's children come back shuffled against the order they were authored. This is the same wrong answer the list-family tie produced, reached here by explicit choice rather than by accident. `--field children.N` addresses positions in that shuffled list, so `children.1` names an arbitrary child rather than the first one written. Nothing in the suite pins the sub-list order today: changing the query broke zero tests.

**Solution**: Order the children sub-list by creation date, then creation sequence, then task ID, with no priority term. The task ID is the same absolute final term the list-family clauses end on.

**Outcome**: `tick show P` lists P's children in the order they were created. Children that record the same creation second list in sequence order. Children that share both a sequence and a second fall back to ascending task-ID order, identically on every run. Priority never lifts a child above an earlier-created sibling, and `--field children.N` picks the Nth-created child.

**Acceptance Criteria**:
- [ ] A parent P whose children c1, c2, c3 record one creation second and one priority and carry sequences in that authoring order. Their ascending-ID order and their line order in `tasks.jsonl` both contradict authoring order. `tick show P` lists the children c1, c2, c3 (§4.3, §8.4, §8 fixture constraints)
- [ ] A parent whose two children record one creation second, where the later-authored child carries a higher priority (a lower number) than its earlier-authored sibling and has an ID that sorts first: `tick show` lists the earlier-authored child first (§4.3, §8.4)
- [ ] A parent with one child recorded in an earlier creation second and one in a later second, where the earlier-second child carries the higher sequence and the later-second child's ID sorts first: `tick show` lists the earlier-second child first, because creation date outranks the sequence (§4.2, §4.3)
- [ ] A parent whose children record one creation second and all carry the same non-zero sequence, with a line order in `tasks.jsonl` that differs from ascending-ID order: `tick show P` lists them in ascending task-ID order, identically on repeated runs (§5.1, §8.3)
- [ ] Same fixture as the first scenario: `tick show P --field children.1` returns a one-row children section holding c1, and `--field children.3` returns one holding c3 (§4.4, §8.4)

**Do**:
- `internal/cli/show.go`: in `queryShowData`, the children query (line 146) orders by created, then sequence, then task ID. It reads the `seq` column that Phase 1 added to the cache's `tasks` table. The clause has no priority term (§4.3, §5.1).

**Context**:
> §4.3: "**Children** … order by created, then sequence, then task ID — the same absolute final term the list clauses carry (§5.1), so a duplicate sequence cannot leave the list plan-dependent. **Priority does not participate** — today's clause has no priority term, and adding one would be a second, unrequested behaviour change. A child with better priority does not float above an earlier-created sibling."
>
> §4.2: `created` outranks `seq`. Where two recorded creation dates differ, those dates decide, and the sequence speaks only when they tie.
>
> §5.1: with the task ID as the final term, "the worst a duplicate sequence can do is fall back to today's ID ordering **deterministically**, rather than varying with the query plan". The shared sequence in the duplicate fixture must be non-zero. A zero value means "no sequence" (§2.2), and backfill gives each such record a distinct number on read (§2.3).
>
> §8 fixture constraints bind every ordering fixture here. IDs must contradict the authoring order, and creation seconds must tie wherever the sequence or the ID term is under test. Assertions state the required order, never the plan. Tied rows reach the sorter in whatever order the plan feeds them, and file order is one of the orders observed (§1.1). A fixture whose line order matched the expected order therefore could not tell a working key from an accident (§3.2). That is why the first and fourth scenarios keep line order apart from the expected order.
>
> §2.4: the sequence is an ordering mechanism and is never printed. The children rows keep their current columns.
>
> §7.3 / §4.4: moving children from ID order to creation order is the one intended behaviour change, and no existing coverage needs updating. The new assertions are the only guard against the key drifting back. `tick show`'s children list and the positional `--field children.N` form (documented at `README.md:206`) change meaning as intended.
>
> The sequence, its assignment on create and its backfill on read are Phase 1 (Tasks 1-1, 1-3, 1-4). Blocker order is Task 2-2. The detail documents that `create`, `update` and `note` print are pinned in Task 2-3.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §1.1, §2.2, §2.3, §2.4, §3.2, §4.2, §4.3, §4.4, §5.1, §7.3, §8 (fixture constraints), §8.3, §8.4

## same-second-tasks-sort-by-id-2-2

### Task 2-2: Order show's blockers by declaration order

**Problem**: `tick show` lists a task's blockers in task-ID order (`internal/cli/show.go:137`, `ORDER BY t.id`), which scrambles the order the dependencies were declared. That order is explicit in the record, because `blocked_by` is a JSON array. `tick dep tree` already emits edges in it, reading the task slice directly. The order is lost only because the cache flattens the array into the `dependencies` junction table, which has two columns keyed on the pair and keeps no position. `tick show` therefore disagrees with both the stored record and `tick dep tree`, and `--field blocked_by.N` addresses an arbitrary blocker.

**Solution**: Carry each blocker's position in the record's `blocked_by` array into the cache's `dependencies` table as an ordinal, and order the blocker sub-list by it.

**Outcome**: `tick show` lists a task's blockers in the order they were declared, whatever the blockers' IDs or creation order. This is the order of the record's `blocked_by` array and of `tick dep tree`'s edges. A cache built before the ordinal existed rebuilds with it at schema version 3, and `--field blocked_by.N` picks the Nth-declared blocker.

**Acceptance Criteria**:
- [ ] A task T whose `blocked_by` array declares its blockers in an order that contradicts both their ascending-ID order and their creation order, where the first-declared blocker was created last: `tick show T` lists the blockers in declaration order, first-declared first (§4.3, §8.4, §8 fixture constraints)
- [ ] Same fixture: T's blockers appear in the same order in `tick show T`, in T's `blocked_by` array in `tasks.jsonl`, and among the edges into T that `tick dep tree` emits (§3.3, §4.3, §8.4)
- [ ] A task whose blockers record one creation second and all carry the same non-zero sequence, declared in an order that contradicts their ascending-ID order: `tick show` lists them in declaration order, identically on repeated runs (§5.1)
- [ ] A project whose `cache.db` was built at schema version 2, with a `dependencies` table that has no ordinal. The next `tick show T` succeeds and lists T's blockers in declaration order. Afterwards `cache.db` reports `schema_version` 3, and its `dependencies` table, read directly, holds for each task an ordinal that orders the blockers as that task's `blocked_by` array does (§3.3, §3.4)
- [ ] Same fixture as the first scenario: `tick show T --field blocked_by.1` returns a one-row blocked_by section holding the first-declared blocker (§4.4, §8.4)

**Do**:
- `internal/storage/cache.go`: the `dependencies` table in `schemaSQL` (lines 31–35) gains an ordinal carrying each blocker's position in the record's `blocked_by` array. The dependency insert (line 143) populates it (§3.3).
- `schemaVersion` stays at 3. The single 2→3 bump from Task 1-1 carries both new columns. No migration is needed: `ensureFresh` already deletes, recreates and rebuilds a cache whose version mismatches, before every query and every mutation (§3.4).
- `internal/cli/show.go`: in `queryShowData`, the blockers query (line 137) orders by the dependency ordinal (§4.3). The ordinal is unique within a task's blocker set, so this order is total with no further term (§5.1).

**Context**:
> §3.3: "The array order is **explicit in the record** — a JSON array is ordered — and is lost only because the cache flattens it into a junction table. Carrying it across applies the same principle driving the whole fix: the order is in the record, so the cache should carry it rather than the query inventing one." `tick dep tree` emits "one edge per `BlockedBy` entry, tasks in slice order and each task's blockers in stored order" (`internal/cli/dep_tree_graph.go:186-198`). It reads `tasks.jsonl` through `store.ReadTasks()` (`internal/cli/dep_tree.go:26`) and never touches the cache (§7.1).
>
> §4.3: blockers order "by the dependency ordinal (§3.3) — the order the dependencies were declared, matching the `blocked_by` array in the record and the edges `tick dep tree` already emits. The alternative — the blocker task's own creation order — would put `tick show` out of step with both `dep tree` and the stored record."
>
> §8.4: "a fixture where a blocker created *later* was declared *first* must list it first, and `tick show` must agree with `tick dep tree` and with the `blocked_by` array in the record on that same fixture. This single assertion pins the key against drift." Under the §8 fixture constraints, IDs must contradict the expected order. Declaration order must also contradict blocker creation order, so a query ordering on the rejected key cannot pass.
>
> §3.4: "`ensureFresh` checks the version before every query and before every mutation; a mismatch deletes the cache file, recreates it and rebuilds from `tasks.jsonl`. The two new columns arrive through that existing path." The schema-version test in `internal/storage/cache_test.go` already moved to 3 with Task 1-1.
>
> §7.3 / §4.4: moving blockers from ID order to declaration order is an intended behaviour change, and no existing coverage needs updating. The positional `--field blocked_by.N` form (documented at `README.md:206`) changes meaning as intended.
>
> Children order is Task 2-1. The detail documents that `create`, `update` and `note` print are pinned in Task 2-3.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §3.3, §3.4, §4.3, §4.4, §5.1, §7.1, §7.3, §8 (fixture constraints), §8.4

## same-second-tasks-sort-by-id-2-3

### Task 2-3: Carry the sub-list order into mutation detail documents

**Problem**: The children and blocked_by sections render from five sites, not one. `FormatTaskDetail` is called from `tick show` (`internal/cli/show.go:84`) and from `outputMutationResult` (`internal/cli/helpers.go:30`). The latter serves `create` (`create.go:278`), `update` (`update.go:398`), `note add` (`note.go:87`) and `note remove` (`note.go:145`). A user who changes a task reads its children and blockers from the document that command prints, so that document must list them in the same order `tick show` does. No test pins the sub-list order on any of these documents today. Without assertions of their own, the mutation paths could drift back to ID order unnoticed.

**Solution**: Assert that the detail documents `create`, `update`, `note add` and `note remove` print list children in creation order and blockers in declaration order, matching `tick show`. Confirm that the conformance inventory still decodes each document.

**Outcome**: Every command that prints a task detail document lists that task's children in creation order and its blockers in declaration order, identical to `tick show` on the same task. The conformance inventory and README samples pass unchanged.

**Acceptance Criteria**:
- [ ] A parent P whose children c1, c2, c3 record one creation second and carry sequences in that authoring order, with ascending-ID order and line order in `tasks.jsonl` both contradicting it. P's `blocked_by` array declares a later-created blocker first, against ascending-ID order. `tick update P --title Renamed` prints a detail document whose children section lists c1, c2, c3 and whose blocked_by section lists the first-declared blocker first, the same order `tick show P` gives (§4.4, §8.4)
- [ ] Same fixture: `tick note add P "text"` prints a detail document whose children and blocked_by sections match `tick show P`'s order (§4.4, §8.4)
- [ ] Same fixture, with a note on P: `tick note remove P 1` prints a detail document whose children and blocked_by sections match `tick show P`'s order (§4.4, §8.4)
- [ ] Two existing tasks, the later-created of which has the higher ID: `tick create T --blocked-by <later>,<earlier>` prints a detail document whose blocked_by section lists the later-created blocker first. That order matches the new record's `blocked_by` array and `tick show T` (§4.3, §4.4, §8.4)
- [ ] The conformance inventory's entries, including those for `create`, `update`, `note add` and `note remove`, still decode under both the TOON and JSON drivers. The existing conformance and README-sample tests pass without any fixture or sample changing (§4.4, §7.3)

**Do**:
- `internal/cli/helpers.go`: `outputMutationResult` (line 17) is the route all four handlers take to `FormatTaskDetail` (line 30). The handlers call it at `create.go:278`, `update.go:398`, `note.go:87` and `note.go:145` (§4.4).
- The conformance inventory is `conformanceDocs` in `internal/cli/conformance_test.go` (§4.4).

**Context**:
> §4.4: "The two sections render from five sites, not one. … Their detail documents and the conformance inventory are in scope, not only `tick show`. (`dep add` and `dep rm` render through `FormatDepChange` and do not carry these sections.)" `dep add` and `dep rm` are out of scope.
>
> §4.4: "**Nothing in the suite pins the sub-list order today.** Changing both queries broke zero tests across `internal/cli`, conformance and README samples included. The new assertions (§8) are the only guard against the key drifting back."
>
> §8.4: "The same sections render under `create`, `update` and both `note` handlers via `outputMutationResult` — their detail documents and the conformance inventory are in scope, not only `tick show` (§4.4)."
>
> In today's code, `outputMutationResult` assembles its detail from the same `queryShowData` that `tick show` uses. The order Tasks 2-1 and 2-2 set therefore reaches these documents through it, and this task's assertions pin that it does. A newly created task has no children, so `create`'s document is pinned on blocked_by only. `create` stores the IDs given to `--blocked-by` in the order given.
>
> §8 fixture constraints apply. IDs must contradict the expected order, and children's creation seconds tie so the sequence is what orders them. Declaration order contradicts both ascending-ID order and blocker creation order (§8.4).
>
> Children order is Task 2-1. Blocker order and the dependency ordinal are Task 2-2.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §4.3, §4.4, §7.3, §8 (fixture constraints), §8.4
