# Investigation: Same Second Tasks Sort By Id

## Symptoms

### Problem Description

**Expected behavior:**
Tasks listed back in creation order come back in the order they were authored. A batch written one after another — by an agent, an importer, or a shell loop — reads back in the sequence it was written.

**Actual behavior:**
Tasks created within the same wall-clock second come back in task-ID order, which is effectively random — IDs are three random bytes. The creation-date sort ties on every row in the batch and falls through to the secondary key.

### Manifestation

- `tick ready` returned a five-task phase shuffled out of authoring order
- The plan adapter's "N of M in phase" positions were visibly wrong
- No error, no warning — the ordering is silently wrong; a batch with real sequencing between tasks would execute out of order with no signal

### Reproduction Steps

1. Create several tasks in quick succession (scripted, so they share a wall-clock second)
2. List them back — `tick list`, `tick ready`, or `tick blocked`
3. Observe the order does not match the order of creation

**Reproducibility:** Always, when the batch shares a second. A human typing `tick add` repeatedly will not hit it; anything scripted hits it every time.

### Environment

- **Affected environments:** Local — all usage
- **User conditions:** Any batch created fast enough to share a wall-clock second. The faster the writer, the more reliably authoring order is lost.

### Impact

- **Severity:** Medium — silent wrong ordering, no data loss
- **Scope:** Every consumer that treats tick as an ordered list rather than a bag of tasks: the workflow system's plan adapter and its implementation loop, `tick list` on a freshly imported project, the migration framework's output, a dependency chain authored in one pass
- **Business impact:** Trust in tick as a plan store. The observed case had independent tasks so nothing broke; a sequenced batch would run out of order undetected. Sharpened by investigation: with `--count 1` the defect does not present a list in the wrong order, it returns the **wrong task**. Severity left at Medium on the strength of no live exposure and no urgency, not because the failure mode is mild.
- **Live exposure:** None. No active work units — every plan authored into tick has been executed, so no stored batch is currently being read in the wrong order. Nothing is being worked around by hand.
- **Urgency:** None. The investigation can take the time it needs; no pressure to land a narrow fix fast.

### References

- Seed: `seeds/2026-09-21-same-second-tasks-sort-by-id.md` (inbox:bug)
- Discovery session: `.workflows/same-second-tasks-sort-by-id/discovery/sessions/session-001.md`
- Observed while dogfooding tick as the plan store for the `free-text-round-trip` workflow (Phase 10)

### Scope Note (from discovery)

Getting creation order right for new tasks going forward is confirmed sufficient. What the fix should do about tasks already stored at second granularity — existing `tasks.jsonl` files that already carry ties — was deliberately left to this investigation.

Confirmed at symptom gathering: there is no live affected data. Existing-tie handling is a question about other people's projects and about tick's own tolerance for older files, not about rescuing a plan currently on disk.

---

## Analysis

### Hypotheses

**Checkpoint depth:** straight-through

- **H1: Creation timestamps are truncated to whole-second granularity before storage, so a batch written inside one wall-clock second stores identical `created` values and the ordering information never reaches disk.** [confirmed]
  Basis: `task.TimestampFormat = "2006-01-02T15:04:05Z"` carries no fractional part, and every stamping site applies `time.Now().UTC().Truncate(time.Second)`.
  Evidence: `task.go:41` fixes the format. Truncation at nine sites: `create.go:208`, `task.go:285` (`NewTask`), `state_machine.go:49`, `note.go:71`, `note.go:135`, `update.go:318`, `dep.go:116`, `dep.go:186`, `migrate/store_creator.go:62`. The truncation is belt-and-braces — `FormatTimestamp` (`task.go:264`) would drop any fraction anyway, since Go's `Format` emits only what the layout names. Reproduced: eight tasks created in a loop all recorded `2026-09-22T09:24:49Z`.

- **H2: The list-family `ORDER BY` terminates at `t.created ASC` with no deterministic tiebreak, so tied rows emerge in whatever order the SQLite query plan produces.** [confirmed — refined]
  Original basis: `list.go:317` (ready view) and `list.go:319` (neutral view) both end on `t.created ASC`; no ID term is written anywhere.
  Evidence: the seed's "falls through to the task ID" is true only for *some* queries. Measured against a seeded project (7 tasks, one second, one parent):
  - **Unfiltered `tick list` / `tick ready`** — plan is `SCAN t USING INDEX idx_tasks_priority` + `USE TEMP B-TREE FOR LAST TERM OF ORDER BY`. Rows reach the sorter in priority-index order, whose own internal tiebreak is rowid; the sorter preserves that for equal `created`, so output is **file order — the right answer, reached by accident**.
  - **`--parent` filtered (`tick list --parent X`, `tick ready --parent X`)** — the parent filter adds `t.id IN (?,?,…)` (`list.go:302`, fed by `queryDescendantIDs` at `list.go:233`). Plan becomes `SEARCH t USING INDEX sqlite_autoindex_tasks_1 (id=?)` + `USE TEMP B-TREE FOR ORDER BY`. Rows reach the sorter already in primary-key order, so output is **exact ascending task-ID order** — the reported symptom.
  This is the mechanism behind the original report: the workflow plan adapter reads a phase's tasks, which is a parent-filtered query.
  Consequence: the unfiltered case is not *correct*, it is *unspecified*. Nothing in the code or in SQLite's contract guarantees it; a different index, a statistics change or a SQLite version bump can flip it without warning.
  **Refined after validation (2026-09-22):** the plan is not a property of the command. Independent measurements disagree at nearly identical project sizes — the validation agent's 8-task project chose `SEARCH t USING INDEX idx_tasks_status` for `tick ready --parent` and returned *file* order, while a 7-task and an 18-task project here both chose `sqlite_autoindex_tasks_1` and returned *ID* order; running `ANALYZE` changed nothing. The optimiser's choice turns on data shape and statistics, not on row count or command name. Parent-filtered queries are the ones observed reaching the primary-key plan, but no query in the family is guaranteed either way. A regression test keyed to "this command returns ID order" would be asserting a coincidence of its own fixture.

- **H3: The JSONL source of truth still holds true authoring order — records are written from the in-memory slice, which appends — so ordering is lost on read, not on write.** [confirmed]
  Evidence: `ParseJSONL` (`jsonl.go:89`) appends one task per line in file order. `create.go:253` appends the new task to the slice. No handler reorders the slice — the only other slice writes are field-level (`dep.go`, `note.go`, `helpers.go`). `Store.Mutate` (`store.go:174`) marshals the slice straight back out and rebuilds the cache from the same bytes. `Cache.Rebuild` (`cache.go:103`) does `DELETE FROM tasks` then inserts in slice order, so `rowid` restarts at 1 and tracks the JSONL line number exactly. Measured: `SELECT rowid, id FROM tasks ORDER BY rowid` returned lines 1–7 in file order.
  Consequence: authoring order is fully recoverable today, from either the file or `rowid` — no data is lost, and existing projects need no repair to be ordered correctly.
  **Verified across history after validation (2026-09-22):** the claim holds for files written by *any* past tick, not just current code. `git log -S "sort." -- internal/storage/ internal/task/ internal/cli/create.go` returns no commits — no sort call has ever existed on those paths. The first `MarshalJSONL` (`4278ba09`, "JSONL storage with atomic writes") iterated the slice unsorted exactly as today, and `create` has appended since `0d26e4b7`.

- **H4: Other fixed-order presentation paths share the defect or sort by ID deliberately — the blast radius exceeds the three list views.** [confirmed]
  Evidence:
  - `show.go:146` children — `ORDER BY id`. Explicit and unconditional: a task's sub-tasks are always ID-ordered, never authoring-ordered. Not a tie-break accident; a deliberate choice that produces the same wrong answer.
  - `show.go:137` blockers — `ORDER BY t.id`. Same.
  - `show.go:173` notes — `ORDER BY rowid ASC`, which is insertion order. Correct.
  - `dep tree` (`dep_tree.go:26`) reads via `store.ReadTasks()` — straight from JSONL, so it is in file order and **sidesteps the bug entirely**.
  - `migrate` (`migrate/store_creator.go`) appends per task and stamps `time.Now().UTC().Truncate(time.Second)` when the provider gives no creation time — so a whole import lands in one second. When the provider *does* supply a time (`beads.go:119` parses RFC3339, which may carry a fraction), `FormatTimestamp` flattens it to the second on write, so imported sub-second precision is discarded too.

- **H5: Second granularity is structural across every timestamp field, so raising precision changes the on-disk format for all of them and for files written by other versions.** [confirmed — basis corrected]
  Original basis claimed reading is strict. **That is wrong.** Go's `time.Parse` accepts a fractional second immediately after the seconds field even when the layout does not name one. Measured: `time.Parse("2006-01-02T15:04:05Z", "2026-09-21T12:00:00.123456789Z")` parses cleanly and retains the nanoseconds. Every parse site in tick uses this constant (`task.go:112,116,137`, `notes.go:41`, `transition_history.go:44`, `show.go:186,239,240,255`), so **an older tick binary reads a finer-grained file without error** — there is no forward-compatibility break on read.
  What *is* structural: one `FormatTimestamp` serves both storage and display. It is called by JSONL marshalling (`task.go:96`), by the cache writer (`cache.go:203,204,229,239`) **and** by all three formatters (`json_formatter.go:128`, `toon_formatter.go:249`, `pretty_formatter.go:170`, `show_fields.go:57`). Raising storage precision therefore also changes every user-visible timestamp unless the two are split. Test surface: 222 literal `…T..:..:..Z` occurrences across 16 test files, plus README samples at `README.md:197,198,502,503,515` which `readme_samples_test.go` compares byte-for-byte.

- **H6: A naive precision increase introduces a *new* ordering defect on mixed-precision data, because the cache sorts `created` as text and `Z` sorts after `.`.** [confirmed]
  Discovered mid-trace. The cache column is `created TEXT NOT NULL` (`cache.go:26`) and `ORDER BY t.created ASC` is a string comparison. Measured in Go: `"2026-09-21T12:00:00Z" < "2026-09-21T12:00:00.500Z"` is **false** — `'Z'` (0x5A) sorts above `'.'` (0x2E).
  Consequence: once a file holds both old whole-second records and new fractional ones, a task created at `12:00:00Z` sorts *after* one created at `12:00:00.500Z` — a fresh wrong-order case, confined to within-second comparisons (across seconds the leading digits still decide correctly). A fix that raises precision must therefore either write a fixed-width fraction on **every** record including rewrites of old ones, or stop relying on lexical comparison of the text column.

**Trace lines (in intended order):**
1. `internal/task/task.go` — `TimestampFormat`, `NewTask`, the JSON marshal/unmarshal round trip — **done**
2. Every stamping site: `create.go`, `update.go`, `note.go`, `dep.go`, `state_machine.go`, `migrate/store_creator.go` — **done**
3. `internal/cli/list.go:310-322` — the `ORDER BY`, plus an empirical `EXPLAIN QUERY PLAN` and live ordering check against a seeded project — **done**
4. `internal/storage/jsonl.go` and `cache.go` `Rebuild()` — whether authoring order survives the round trip and whether `rowid` tracks it — **done**
5. Other ordered read paths — `show.go` children/blockers/notes, dep tree, migrate output — **done**
6. Compatibility surface — strict `time.Parse`, `schemaVersion`, `internal/doctor/` checks, tests pinning the format — **done**

### Code Trace

**Entry point:** `RunList` → `buildListQuery` (`internal/cli/list.go:262`) → `Store.Query` → SQLite.

**Execution path (the reported symptom — a parent-filtered list):**
1. `internal/cli/list.go:233` — `queryDescendantIDs` walks the parent chain with a recursive CTE, returning descendant IDs in walk order
2. `internal/cli/list.go:302` — those IDs become `t.id IN (?,?,…)`
3. `internal/cli/list.go:317` or `:319` — `ORDER BY … t.created ASC`, with no further term
4. SQLite plans `SEARCH t USING INDEX sqlite_autoindex_tasks_1 (id=?)` — rows are produced in primary-key order
5. `USE TEMP B-TREE FOR ORDER BY` sorts on priority and `created`; every row ties on both, so the incoming primary-key order survives to the output
6. The caller receives tasks in ascending task-ID order — task IDs being `tick-` plus three random bytes (`task.go` `GenerateID`), this is indistinguishable from random

**Key files involved:**
- `internal/task/task.go:41` — `TimestampFormat`, the single constant governing both storage and display precision
- `internal/task/task.go:264` — `FormatTimestamp`, the only writer of that format
- `internal/cli/list.go:262-325` — `buildListQuery`, both `ORDER BY` variants and the `id IN (…)` condition
- `internal/storage/cache.go:18-28` — the `tasks` table; `created` is `TEXT`, so the sort is lexical
- `internal/storage/store.go:174` — `Mutate`, full JSONL rewrite plus full cache rebuild on every write
- `internal/cli/show.go:137,146` — children and blockers, explicitly `ORDER BY id`

### Root Cause

Two independent defects compound. Neither alone produces the reported symptom; together they mean authoring order is sometimes preserved, sometimes scrambled, with nothing to tell the two cases apart.

**1 — The ordering information is never recorded.** Tick stores creation times to whole-second granularity. `TimestampFormat` (`task.go:41`) has no fractional component, so `FormatTimestamp` cannot emit one; every stamping site additionally truncates with `Truncate(time.Second)` before the value is even held. A batch of tasks authored inside one wall-clock second therefore records the *same instant* for every member. The order the user intended is not degraded in storage — it is absent from it.

**2 — The sort has no deterministic final term, so tied rows are ordered by whatever the query plan happens to do.** `buildListQuery` ends its `ORDER BY` at `t.created ASC` (`list.go:317`, `:319`). SQLite is free to emit tied rows in any order, and in practice emits them in the order the chosen plan feeds the sorter — which differs by filter:

| Plan observed | Rows reach the sorter in | Tie order out | Seen on |
|---|---|---|---|
| `SCAN t USING INDEX idx_tasks_priority` | rowid within each priority | file order — right, by accident | every unfiltered `list` / `ready` / `blocked` measured |
| `SEARCH t USING INDEX sqlite_autoindex_tasks_1 (id=?)` | primary key | **task ID — wrong** | parent-filtered queries, most of the time |
| `SEARCH t USING INDEX idx_tasks_status (status=?)` | rowid within status | file order — right, by accident | `ready --parent` in one measured project |

**Which plan a given command gets is not predictable.** The parent filter adds `t.id IN (?,?,…)` (`list.go:302`, fed by `queryDescendantIDs` at `list.go:233`), which makes the primary-key plan attractive — and that is the shape the reporter hit, since reading a phase's tasks *is* a parent-filtered query. But independent measurement disagrees at nearly identical sizes: `tick ready --parent` returned file order in an 8-task project and ID order in 7-task and 18-task projects, on the same schema and the same binary. `ANALYZE` changed nothing. Task IDs are `tick-` plus three random bytes, so whenever the primary-key plan wins, a deliberate sequence comes back looking like noise.

The second defect is the more important finding, because it reframes the bug. **No query in the family is correct — some are merely lucky.** Nothing in tick's code, and nothing in SQLite's contract, promises any of these orders. A new index, a statistics change, a different row mix or a SQLite version bump can flip any of them silently, in either direction. The right mental model is not "parent-filtered reads are broken" but "tie order is undefined everywhere, and currently resolves correctly by chance in most cases".

**Why this happens:** tick treats creation time as both a timestamp and an ordering key, but only the timestamp role shaped the format. Second granularity is a reasonable choice for a human-readable audit field and a poor one for a sort key, and nothing in the design separates the two roles or supplies a fallback when the key ties.

### Contributing Factors

- **IDs are random.** `GenerateID` uses three bytes from `crypto/rand`, so when the tie resolves to ID order the result is not merely arbitrary — it actively shuffles, and looks like a bug in the consumer rather than in tick.
- **The bug is filter-dependent and therefore invisible to casual checking.** The obvious manual test (`tick list` after creating a batch) returns the right answer. Only the parent-filtered path — the one machines use — goes wrong.
- **Machine writers are the normal case now.** Tick is used as a plan store by an agent that authors a whole phase in one pass; the migration framework stamps an entire import with one `time.Now()` (`migrate/store_creator.go:62`). Human-speed use never reaches the tie; every scripted use does.
- **Sub-second precision is discarded even when a source has it.** `migrate/beads/beads.go:119` parses RFC3339 creation times that may carry a fraction; `FormatTimestamp` flattens them on write.
- **The documented contract stops short of the tie.** `README.md:115` promises "sorted by priority (ascending), then creation date" and says nothing about what happens when creation dates are equal — so neither the code nor the docs commit to an answer, and the workflow plan adapter's own reading instructions assume one that tick does not give.

### Why It Wasn't Caught

- **No test constructs a creation-time tie.** Every ordering test gives each task in a priority band a distinct `Created` — `ready_test.go:306` uses `now`, `now+1s`, `now+2s`; `blocked_test.go:212` and `list_filter_test.go:368` follow the same fixture shape. The tie branch has never executed under test.
- **And it would pass by coincidence if it did.** Those fixtures use hand-written IDs (`tick-hi1111`, `tick-low111`, `tick-low222`) that already sort into the expected order, so an ID-ordered tie would satisfy the assertions.
- **No test exercises ordering under a parent filter.** The two plans are never compared, so the divergence between filtered and unfiltered results is untested ground.
- **The correct-looking case masks the broken one.** Unfiltered output being right is what makes the bug survive both manual use and code review.

### Blast Radius

**Directly affected — wrong order today:**
- `tick list --parent`, `tick ready --parent`, `tick blocked --parent` — any same-second batch under a parent returns in random ID order whenever the primary-key plan wins. This is the workflow plan adapter's read path and the reported symptom.
- **`--count` turns wrong order into wrong *selection*.** `LIMIT ?` is appended after the tied `ORDER BY` (`list.go:321-324`), so it cuts the arbitrary order rather than the authored one. Measured: in a five-child phase authored step-1 … step-5, `tick ready --parent P --count 1` returned **step-4**. This is the sharpest form of the defect — the implementation loop's "take the next available task" call is handed the wrong task outright, not a list it could re-sort. The Impact section's "ordering was merely visibly wrong" understates this case.
- `tick show <id>` sub-task list (`show.go:146`, `ORDER BY id`) and blocker list (`show.go:137`, `ORDER BY t.id`) — unconditionally ID-ordered regardless of timestamps. Same wrong answer, reached by an explicit choice rather than a tie.
- Anything importing through `internal/migrate/` — the whole import shares one second, so every parent-filtered read of imported tasks is scrambled.

**Correct today, but only by accident:**
- `tick list`, `tick ready`, `tick blocked` unfiltered, and every non-parent filter combination (`--tag`, `--type`, `--status`, `--priority`) that does not introduce an `id IN (…)` condition. These depend on an unspecified SQLite behaviour and could flip at any time.
- `tick stats` counts are order-independent — unaffected either way.

**Correct by construction:**
- `tick dep tree` reads from JSONL via `store.ReadTasks()` (`dep_tree.go:26`), so it is in file order and never touches the cache sort.
- `tick show` notes (`show.go:173`, `ORDER BY rowid ASC`) — insertion order.

**Surface any fix must cross:**
- **Authoring order is not lost and needs no recovery.** JSONL holds it (append-only writes from a slice nothing reorders), and `Cache.Rebuild` deletes and re-inserts in that order, so `rowid` equals the JSONL line number after every mutation. Both the file and `rowid` are usable today as a true authoring-order key — and existing projects are already repairable without a migration.
- **Timestamp precision is one knob for storage and display.** `FormatTimestamp` is called by the JSONL marshaller (`task.go:96`), the cache writer (`cache.go:203`) **and** all three formatters (`json_formatter.go:128`, `toon_formatter.go:249`, `pretty_formatter.go:170`, `show_fields.go:57`). Raising storage precision also changes every user-visible timestamp unless the roles are split.
- **Reading is tolerant, but writing is not — the compatibility hazard is downgrade, not upgrade.** Go's `time.Parse` accepts a fractional second even when the layout omits one (measured against nanosecond input), and every parse site uses `TimestampFormat`, so an older tick binary *reads* a finer-grained file without error. But `Store.Mutate` (`store.go:174-199`) rewrites **every** record through `MarshalJSONL` on every write, and `FormatTimestamp` drops the fraction. Measured end to end: a file hand-edited to carry `…30.100Z` and `…30.500Z` was read correctly by tick 0.3.0, and one `tick create` flattened both to `…30Z` — the whole file, `created`/`updated`/`closed`/note and transition stamps alike, with the bug fully reinstated and no warning. Any released precision change means a user on an older binary silently destroys the new ordering data on their next mutation. This bears directly on the fix: it is an argument for a mechanism that does not depend on every writer being new.
- **But mixed precision breaks the lexical sort.** The cache column is `created TEXT` and the comparison is a string comparison; `'Z'` (0x5A) sorts above `'.'` (0x2E), so `"12:00:00Z" < "12:00:00.500Z"` is false. A file holding both old whole-second and new fractional records orders them wrongly within a second. Any precision change must write a fixed-width fraction on every record, or stop comparing the text column.
- **Test and doc surface:** 222 literal `…T..:..:..Z` timestamps across 16 test files (plus the format constant itself at `task.go:41`), and README samples at `README.md:197,198,502,503,515` that `readme_samples_test.go` compares byte-for-byte.
- **Cache schema:** no version bump needed for a value-format change — the SHA256 freshness hash changes with the file and forces a rebuild. A bump from `schemaVersion = 2` would only be needed if a column were added.
- **`internal/doctor/` holds no timestamp logic at all** — nothing there to update either way.

---

## Fix Direction

### Chosen Approach

**Record creation times with sub-second precision, and give the sort a deterministic final term as a floor beneath it.**

Two mechanisms, with distinct jobs:

1. **Precision (the fix).** Creation times gain a fractional part and stop being rounded to the second, so the *record itself* carries the order. Any consumer reading `tasks.jsonl` gets it, not just tick's own queries. This also makes the already-documented contract at `README.md:115` — "sorted by priority (ascending), then creation date" — true, rather than needing a new clause.
2. **Authoring position as the sort's final term (the floor).** Each list-family query ends on the task's position rather than stopping at `t.created ASC`. One extra term, closing the residue precision structurally cannot reach.

**Deciding factor:** precision puts the ordering information in the record, where nothing can lose it. That removes what this investigation itself named as the principal residual risk of the position-only route — ordering correctness living in a cache the design calls expendable. The tiebreak stays because it costs a single sort term and covers the one reachable case precision cannot: an import whose *source* recorded whole-second timestamps ties in the incoming data, and `beads.go:119` uses the provider's `created` verbatim.

**Hard constraint discovered by measurement — the read format must not widen.** Go's `time.Parse` tolerance runs one way only: a layout *without* a fractional component accepts an input that has one, but a layout *with* one **requires** it. Widening the single `TimestampFormat` constant makes every existing `tasks.jsonl` unreadable (`cannot parse "Z" as ".000"`, 41 test failures across three packages). The parse layout must stay exactly as it is; only the write layout widens.

**Measured cost, not estimated.** With the stored format widened and the *displayed* format left at whole seconds, six tests fail — `cache_test.go` round-trip, notes and transitions; `jsonl_test.go` format-spec; `task_test.go` marshal. All six assert the on-disk shape, which is precisely what changes. The entire `internal/cli` package passes: README samples, all three formatters, the conformance inventory. The earlier figure of 222 test literals was a raw grep count, not breakage.

**Still to settle at specification:**
- **Whether the displayed timestamp gains the fraction too.** Keeping display at whole seconds is what collapses the churn to six tests and leaves every README sample untouched; showing it makes the order visible to a human reading `tick show`. A product call about what a creation time should look like, not an ordering question.
- **The precision itself** — milliseconds suffice for the measured case; finer costs nothing extra.
- **How authoring position is carried** — the cache's existing implicit row number, or an explicit column (a `schemaVersion` bump from 2 to 3, absorbed by the existing delete-and-rebuild path). Fix validation established that an explicit column buys a nameable thing to assert on, not protection: both routes go stale identically under any future incremental cache-update path. The asserted invariant is the mitigation either way.

### Options Explored

**A — Precision plus the position tiebreak. CHOSEN.** As above.

**B — Precision alone. Not chosen.** It covers every reported symptom in practice: separate `tick create` invocations cannot plausibly share a millisecond. But the sort keeps no final term, so ordering is *undefined* when a tie does occur rather than merely unlikely — and the reachable case (imports carrying tied source timestamps) is one no tick-side precision fixes. The floor costs one sort term; going without it buys nothing.

**C — Position tiebreak alone. Agreed, then reversed.** Deterministic and cheapest, but it leaves the record dishonest — two tasks created in one second still claim the same instant, and the order is legible only as position in a file, not as a value. It also leaves `README.md:115` untrue, needing a new sentence rather than being satisfied, which repeats the missing-contract factor this investigation identified as contributing to the bug.

**D — An explicit sequence number stored in the JSONL record. Not presented.** Duplicates what a precise creation time states, changes the record format for every task, and obliges every import path to invent a value.

### Discussion

The direction reversed mid-exploration, and the reversal is the substance of this section.

The first recommendation was the position tiebreak alone (option C), reasoning that authoring order is already recorded in `tasks.jsonl` line order and the fix merely has to ask for it. That reasoning is sound and the empirical basis for it held up — the fix-validation agent reproduced both the symptom and the mechanism against a live project, and verified the invariant through create, remove, update and rebuild. It was agreed and sent for pressure-testing.

The reporter then reopened it, asking whether sub-second timestamps were not simply the obvious answer, and whether the objections were overcooked. Two things settled it against the earlier recommendation:

**The cost argument was wrong, and measurably so.** The claim that a format change moves 222 test literals and five README samples used the raw grep count as if every occurrence were breakage. Measured by actually making the change: with the storage and display formats separated, six tests fail and the entire command layer passes untouched. The cost that justified preferring the cheaper option was roughly two orders of magnitude smaller than stated.

**The honesty argument answers this investigation's own strongest objection to its own recommendation.** The Risk Assessment for option C named the principal residual risk as ordering correctness moving into a disposable cache, resting on an invariant no test states. Precision dissolves exactly that: the order lives in the record. It also dissolves two of the six risks the fix validation raised — the documented contract becomes true instead of needing amendment, and there is no cache-version thrash.

Three of those six risks survive the change of direction unaltered, because they are orthogonal to the mechanism: the blocker-list key, the six-command surface behind `tick show`'s sub-lists, and the within-a-priority-band scope of the guarantee. `show.go:137,146` sort by `id` **explicitly, not as a tie fall-through** — precision does not touch them, so those queries change under any option, and the same key question follows.

Two hazards are accepted rather than solved. Old and new records sort wrongly against each other within a single second, because the cache compares `created` as text and `'Z'` (0x5A) sorts above `'.'` (0x2E) — the window closes at the first write, which rewrites every record through the new format. And a user on an older binary flattens a fractional file on their next mutation (measured: one `tick create` on tick 0.3.0 turned `…30.100Z` and `…30.500Z` into `…30Z`). That reverts to today's behaviour rather than corrupting anything, and the position tiebreak keeps the ordering correct through it — which is a second reason to keep the floor.

### Testing Recommendations

**Ordering — the core, none of which exists today.** Every current ordering test gives each task in a priority band a distinct `Created` (`ready_test.go:306`, `blocked_test.go:212`, `list_filter_test.go:368`), so the tie branch has never executed.

- A same-second tie ordering correctly, with **IDs whose ascending order contradicts the authoring order** — current fixtures use hand-written IDs that already sort into the asserted order, so an ID-ordered result would pass and prove nothing.
- The same assertion from an unfiltered list **and** a `--parent`-filtered one: those take different query plans and the unfiltered case passes today by accident. Fixture size must not be load-bearing — measurements show the plan can flip on data shape alone.
- **`blocked` gets its own tie assertion.** `BlockedConditions()` builds a different WHERE shape (`query_helpers.go:66-80`) and is not covered by the list/ready cases.
- **The `ready` band interaction:** an `in_progress` task authored *after* tied `open` tasks — the band must still win over authoring position. Otherwise a final term inserted before the band term passes everything else.
- **`--count 1` under a tie returns the first-authored task**, not the ID-lowest. The sharpest symptom; needs its own assertion.

**Precision.**
- A batch created in one wall-clock second records *distinct* creation times end to end.
- **An existing whole-second file still reads.** The parse-layout constraint is the trap; a fixture file with no fractions must load without error.
- Mixed-precision records order correctly, and the first write normalises the file.
- Round-trip: a value written at the new precision reads back equal.

**The `show` sub-lists — a behaviour change reaching six commands.**
- Children and blockers follow authoring order, not ID order, and existing expectations move with them.
- The same sections render under `create`, `update` and both `note` handlers via `helpers.go:23,28` — their detail documents and the conformance inventory (`conformance_test.go`) are in scope, not just `tick show`.
- **The blocker key is pinned by a test**: a blocker authored *after* the blocked task's other blocker, so the blocker's own creation order differs from the declaration order. Whichever key is specified, this test stops it drifting.
- `--field children.N` / `blocked_by.N` positional addressing changes meaning — intended, but assert it rather than discover it.

**Invariants and documentation.**
- The position key holds after create, update, remove **and** `tick rebuild`. Phrase it as "position order equals record order", not "equals line number" — `ParseJSONL` skips blank lines (`jsonl.go:99-101`).
- Imported projects order correctly — the migration framework stamps a whole import inside one second, making it the natural end-to-end fixture.
- `README.md:115` and the `list` help text (`help.go:58`) state the sort contract; the README-sample run must cover whatever lands there.
- Existing ordering tests stay green **unchanged** — neither mechanism may disturb a result whose earlier keys already differ.

### Risk Assessment

- **Fix complexity:** Low-to-moderate. A second timestamp format constant and the write-path call sites, the nine `Truncate(time.Second)` removals, two `ORDER BY` clauses in `buildListQuery`, two sub-queries in `show.go`, and six on-disk-shape test updates.
- **Regression risk:** Low. A final sort term cannot reorder any result whose earlier keys already differ; widening the write format cannot break reads, given the parse layout is held. The one intended behaviour change is `tick show`'s children and blockers moving from ID order to authoring order.
- **Compatibility risk:** Moderate and bounded, entirely on the downgrade side. A newer tick reads any older file (parse layout unchanged). An older tick reads a newer file (Go accepts an unnamed fraction) but **flattens it on its next write**, reverting to today's behaviour — measured, not theorised. No file becomes unreadable in either direction and nothing is corrupted; the position tiebreak keeps ordering correct even through a flattening.
- **Recommended approach:** Regular release. No urgency — confirmed at symptom gathering that no active work unit holds affected data. Worth a release note about the downgrade behaviour.
- **Principal residual risk:** the mixed-precision lexical anomaly during the window between upgrade and first write. Bounded to comparisons *within one second* between an old and a new record, and closed permanently by the first mutation. The position tiebreak covers the window.

## Notes

**User's fix lean (symptom gathering, 2026-09-21):** raise timestamp precision — store creation times at millisecond granularity or finer so the sort no longer ties. Stated as a lean, explicitly open to alternatives. To be tested against the root cause at Step 10 rather than assumed.
