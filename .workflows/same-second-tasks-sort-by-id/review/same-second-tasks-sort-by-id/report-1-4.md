TASK: Backfill Missing Sequences Above The File's Highest On Read (same-second-tasks-sort-by-id-1-4, tick-ecc73a)

ACCEPTANCE CRITERIA:
- A `tasks.jsonl` with no `seq` on any record, every record recording one creation second and one priority, IDs contradicting line order: `tick list` returns line order, and still does after a `tick update`, a `tick remove` and a `tick create`; from the first of those writes on, every record in the file carries a sequence and the sequences follow record order (§2.3, §8.2)
- Running `tick list` on that file leaves `tasks.jsonl` byte-identical — the assignment reaches the file only on the next write (§2.3)
- A no-`seq` file of three records: `tick create` writes the new task with sequence 4 (§2.2)
- A file whose first records carry sequences numbered above their line positions, followed by a newest record carrying none, all in one creation second and one priority: the newest record is assigned a number above every carried sequence and `tick list` returns it last (§2.3, §8.2)
- A file in the merge shape (a, b, c at 1–3; x, y none; d, e at 4, 5), one second, one priority, IDs contradicting expected order: backfill assigns x and y 6 and 7, no duplicates, and `tick list` returns a, b, c, d, e, x, y (§2.3, §8.2)
- A file stripped by a binary that did not know the field, with that binary's record appended last, one second and priority: `tick list` returns the original order with the appended record last (§7.1, §8.2)
- A no-`seq` file with blank lines between records: `tick list` returns record order, and assigned sequences follow record order rather than line numbers (§8.2)
- A file whose records carry `seq` 0 orders and backfills exactly as one whose records carry no `seq` field (§2.2)
- The same no-`seq` file read through `Store.ReadTasks`, `tick rebuild` and `tick list` yields the same sequence for every record, in the tasks returned and in the cache's `seq` column (§2.3)
- A beads import into a project holding a record with no sequence: afterwards that record's stored values are unchanged apart from the sequence it gained (§3.1, §7.2)

STATUS: complete

SPEC CONTEXT: §2.3 (as corrected by the 2026-09-22 corrigendum) requires backfill on read in `ParseJSONL`, the single funnel for `ReadTasks`, `Rebuild` and `readAndEnsureFresh`: take the highest sequence any record in the file carries (0 if none), then walk records in order giving each unnumbered record (absent or zero, §2.2) the next number above it. This is neither line-position numbering (wrong on post-merge files: the newest record would sort first) nor a running maximum (collides in the merge shape). Assignment becomes permanent on the next write, with no migration. §8.2 requires the no-seq, mixed, merge-shape, stripped and blank-line cases, with "sequence order equals record order", not "equals line number". §8's fixture constraints require one shared creation second and IDs contradicting the expected order. §3.1/§7.2 accept that every record gains a field on the next write, and that the migrate "untouched" test changes as a result.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/storage/jsonl.go:116: `ParseJSONL` calls `backfillSeqs(tasks)` after parsing every non-blank line, so blank lines never consume a number
  - internal/storage/jsonl.go:123-131: `backfillSeqs` seeds `next` from `task.NextSeq(tasks)` (the file's highest carried sequence + 1, computed over every record before the walk) and assigns `next++` to each record with `Seq == 0`, in record order. This is the file-highest rule, not a running maximum and not line position
  - internal/task/task.go:274-280: `NextSeq` is shared with creation (internal/cli/create.go:221, internal/migrate/store_creator.go:82), so backfill and creation follow one rule, as §2.3 requires. Creation runs inside `Store.Mutate` over already-backfilled tasks, so a new task numbers above the backfilled values
  - internal/storage/store.go:164 (`ReadTasks`), :248 (`Rebuild`), :365 (`readAndEnsureFresh`): all three read paths go through `ParseJSONL`, so all three follow the same rule
  - internal/task/task.go:57: `Seq int` with `json:"seq,omitempty"`. Zero is omitted on write and treated as "none" on read, so `seq: 0` and an absent field are indistinguishable after parsing
- Notes: The query path never writes `tasks.jsonl` (`Store.Query` → `readAndEnsureFresh` → `ensureFresh` only rebuilds the cache), so `tick list` leaves the file byte-identical. The first `Mutate` re-marshals every task with its backfilled `Seq`. The backfill is a pure function of the file bytes, so a cache rebuilt from those bytes and checked by hash on the next read carries the same sequences. No drift from the plan or the corrected spec.

TESTS:
- Status: Adequate
- Coverage (internal/cli/backfill_seq_test.go, `TestBackfillSequence` at :129):
  - AC1 → :130. Legacy IDs descend against line order; asserts list order after update, remove and create, plus strictly ascending, non-zero raw `seq` after each write via `assertSeqsFollowRecordOrder` (:61)
  - AC2 → :148. Byte comparison before and after `list`
  - AC3 → :167. Raw `seq` of the last record == 4
  - AC4 → :178. Seqs 5, 6, 7 then an unnumbered newest record with the lowest ID; asserts list order and cached `seq` 8. A line-position rule would give it 4 and sort it first, so the test tells the two rules apart
  - AC5 → :191. Merge shape with IDs descending through the expected order; asserts list order and all seven cached sequences (1..7, which implies no duplicates). A running maximum would give x/y 4/5 and fail both assertions
  - AC6 → :198. Stripped file plus an appended lowest-ID record; asserts list order
  - AC7 → :205. Blank lines between records; asserts cached seqs 1/2/3 and, after an update, raw seqs [1 2 3]. A line-number rule would fail
  - AC8 → :221. Zero-seq and absent-seq files side by side; identical list order and cached seqs
  - AC9 → :241. `ReadTasks` task seqs, `rebuild` cache column and `list` cache column, all against `mergeShapeSeqs`
  - AC10 → internal/cli/migrate_test.go:713. Asserts the seeded record had no `seq` before, has `seq` 1 after, and is otherwise `reflect.DeepEqual` to its pre-import form
  - Unit level: internal/storage/jsonl_test.go:761 `TestParseJSONLBackfillsSequence` covers the rule directly (no-seq, blank lines, above-highest, merge shape, zero-as-none, carried seqs kept)
- Notes: Every fixture uses one creation second (`sameSecond`), one priority, and IDs that contradict the expected order, as §8 requires. Each subtest would fail under the line-position, running-maximum or zero-as-carried alternatives. AC9 uses the merge-shape file rather than a fully unnumbered one. That is a sound choice: a fully unnumbered file numbers 1..n under every candidate rule, so it could not show a path on a different rule, while the merge shape can. The unit tests overlap the CLI tests in places (merge shape, blank lines, zero), but at different layers: the unit tests assert the parse result, the CLI tests assert the cache column and list order. This is not redundant.

CODE QUALITY:
- Project conventions: Followed (stdlib `testing`, `t.Run` subtests with "it ..." names, `t.Helper()` on helpers, `t.TempDir()` isolation via `setupTickProject`, toon output decoded via `decodeToonDoc`/`toonRows`)
- SOLID principles: Good. `backfillSeqs` has one job, and the numbering rule lives once in `task.NextSeq`
- Complexity: Low (two linear passes)
- Modern idioms: Yes (`max` builtin, `strings.SplitSeq`, `range` over int in adjacent tests)
- Readability: Good. The `backfillSeqs` comment says why the rule is file-highest rather than a running maximum, and it holds against the code
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
