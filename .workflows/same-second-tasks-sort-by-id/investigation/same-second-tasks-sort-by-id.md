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

(to be populated after findings sign-off)

---

## Notes

**User's fix lean (symptom gathering, 2026-09-21):** raise timestamp precision — store creation times at millisecond granularity or finer so the sort no longer ties. Stated as a lean, explicitly open to alternatives. To be tested against the root cause at Step 10 rather than assumed.
