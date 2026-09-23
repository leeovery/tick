TASK: Keep Import Order For In-Process Migration Batches (same-second-tasks-sort-by-id-1-5, tick-a21794)

ACCEPTANCE CRITERIA:
- A beads source of five issues carrying no `created_at` and one priority, imported by a single `tick migrate` run whose imported tasks record one creation second: the stored tasks carry sequences in import order, and `tick list` returns them in import order — a result ascending task-ID order could not produce (§1.3, §8.1)
- A beads import into a project that already holds tasks: each imported task carries a sequence above the project's highest, increasing in import order (§2.2)
- Issues of one priority whose `created_at` values fall in different seconds and run against import order: `tick list` returns them by those creation times, not by import order (§1.3, §4.2)
- Issues of one priority whose `created_at` values share one second but carry fractions that run against import order: the stored creation times are whole-second and equal, and `tick list` returns the tasks in import order (§4.2)

STATUS: complete

SPEC CONTEXT: §1.3 guarantees that an import with no source creation times reads back in import order, while one carrying real historical times is ordered by those times to whole-second resolution. §2.2: a new task takes the next number above the highest sequence in the task set as read inside Store.Mutate's exclusive lock. §4.2: `created` outranks `seq`; source fractions are flattened on write, so imported tasks sharing a second fall to the sequence (import order). §7.1: timestamp handling is unchanged. §8.1/§8 fixture constraints: the migration framework is the in-process-batch fixture; IDs are generated so the assertion must be one ascending-ID order could not satisfy, and the one-second tie is a condition the test sets up, not a result it asserts.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/migrate/store_creator.go:82 — `Seq: task.NextSeq(tasks)` inside the per-task `c.store.Mutate` closure (store_creator.go:36), so each imported task is numbered from the task set read under the exclusive lock
  - internal/task/task.go:274-280 — `NextSeq`, the same rule `tick create` uses (one above the highest carried sequence, 1 when none)
  - internal/migrate/store_creator.go:60-63 — provider-supplied `mt.Created` kept as-is; only a zero value is stamped with `time.Now().UTC().Truncate(time.Second)` (unchanged)
  - Fraction flattening is the existing write path: internal/task/task.go:99 (`FormatTimestamp` in MarshalJSON) and internal/storage/cache.go:205 (`FormatTimestamp(t.Created)` into the cache), so both tasks.jsonl and the cache `created` column are whole-second
  - Sort consumed: internal/cli/list.go:313 and :315 end on `t.created ASC, t.seq ASC, t.id ASC`
  - Beads parsing unchanged: internal/migrate/beads/beads.go:119 (`time.Parse(time.RFC3339, …)`, which accepts a fraction)
- Notes: Matches the task's Do section exactly — one-line assignment, no timestamp handling changed. Store.Mutate also backfills any unnumbered task the mutation returns (internal/storage/store.go:192, built under task 1-6), which would give imports the same numbers; the plan's task 1-6 row explicitly keeps the per-site `NextSeq` calls, so the explicit assignment is intended, not dead code.

TESTS:
- Status: Adequate
- Coverage:
  - AC1: internal/cli/migrate_seq_test.go:52-77 — five beads issues with empty `created_at`, priority 2; each attempt is discarded unless all stored tasks share one creation second and the import-order IDs are not ascending (the tie is set up as a condition, per §8); then asserts raw on-disk seqs [1 2 3 4 5] and `tick list` equal to import order
  - AC2: internal/cli/migrate_seq_test.go:79-94 — existing records at seq 4 and 9, three-issue import, asserts raw seqs [4 9 10 11 12]
  - AC3: internal/cli/migrate_seq_test.go:96-110 — created_at seconds 04..00 against import order; `tick list` returns reverse import order (by created)
  - AC4: internal/cli/migrate_seq_test.go:112-141 — fractions .9/.7/.5/.3/.1 within one second against import order; asserts every stored `Created` equals the whole second (readPersistedTasks parses through time.Parse(TimestampFormat), which would read back a stored fraction, so this is a real check) and `tick list` in import order, retrying until IDs are not ascending
  - Creator unit level: internal/migrate/store_creator_test.go:262-282 (store seqs 3, 7, 5 → new task 8, so "highest" is distinguished from "last" or "count") and :284-295 (empty store → 1). The mockStore does not backfill, so these fail if store_creator.go:82 is removed — the CLI tests alone would not, because Mutate's backfill assigns the same numbers.
- Notes: The AC2 CLI fixture puts the highest sequence on the last record, so on its own it cannot tell "highest" from "last + 1"; the store_creator unit test at store_creator_test.go:262-282 covers that distinction, so the criterion is covered as a whole. No over-testing: each subtest maps to one criterion and the two unit subtests are short.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, `t.Run` "it …" subtests, `t.Helper()` on helpers, toon decoding via listedIDs)
- SOLID principles: Good — numbering delegated to the shared `task.NextSeq` rule rather than reimplemented
- Complexity: Low
- Modern idioms: Yes (`for range maxAttempts`, `slices.IsSorted`, `slices.Reverse`, `slices.Equal`)
- Readability: Good; test comments at migrate_seq_test.go:12-13, :31, :53-54 and :113 are accurate against the code
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
