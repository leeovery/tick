TASK: Number Every Task A Mutation Writes, And Check Stored Sequences Raw (same-second-tasks-sort-by-id-1-6, tick-ac4d88)

ACCEPTANCE CRITERIA:
- Given three records carrying `seq` 1, 2, 3 at one creation second and one priority, a `Store.Mutate` on a real store that appends a task with no sequence (same second, same priority, an ID sorting ahead of all three) writes that record to `tasks.jsonl` carrying `seq` 4, and the cache's `seq` column holds 4 for it (§2.2, §3.2)
- Straight after that mutation, with no further write, `tick list` lists the appended task last of the four, and `tick ready --count 1` returns the first-authored task rather than the appended one (§1.2, §8.1)
- After that mutation, `tick rebuild` leaves every task's cached `seq` as the mutation cached it, and `ReadTasks` (the path `tick dep tree` reads) returns the same numbers: the cache equals a parse of the written bytes (§2.3)
- A mutation returning records numbered 1, 2, 3, then two records with no sequence, then records numbered 4 and 5 writes the two unnumbered records as 6 and 7 in record order and leaves every carried number unchanged (§2.1, §2.3)
- Every sequence assertion in `TestCreateAssignsSequence` reads the `seq` each record carries in `tasks.jsonl`, not a value `storage.ReadJSONL` backfilled, so a record written without a `seq` reads as 0 and fails it
- The rest of the suite passes unchanged, including the create and migrate `Seq: task.NextSeq(tasks)` assignments and the migrate `mockStore` subtests (§8.5)

STATUS: complete

SPEC CONTEXT: §2.2 says a new task takes the next number above the highest sequence in the task set as read, with 0 meaning "no sequence". §2.3 defines backfill: number each unnumbered record, in record order, above the highest carried sequence, so backfill and creation are one rule. §3.2 requires the cache's `seq` column to be asserted directly. §1.2 and §8.1 cover `--count 1` under a tie returning the first-authored task. §8.5 is the regression floor. The spec puts backfill in `ParseJSONL` as the funnel for every read path. This task also runs the same function on the write path, so the cache built from the mutated slice always equals a parse of the bytes written. That applies the same rule a second time and does not create a second rule, so it is a sound extension of §2.3.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/storage/store.go:192: `backfillSeqs(mutated)` runs after `fn` returns and before `MarshalJSONL` (:195), so the bytes written at :202 and `s.cache.Rebuild(mutated, newRawJSONL)` at :208 carry the same numbers.
  - internal/storage/store.go:172-174: the `Mutate` doc comment states the new contract ("Tasks fn returns without a sequence are numbered, in order, above the highest sequence any returned task carries"), and it is accurate.
  - internal/storage/jsonl.go:123-131: `backfillSeqs` is the same function `ParseJSONL` calls at :116, so read and write share one rule.
  - internal/cli/create.go:221 and internal/migrate/store_creator.go:82 still assign `Seq: task.NextSeq(tasks)`, as the task requires.
- Notes: For an appended task with `Seq` 0, `backfillSeqs` assigns `NextSeq` over the returned slice, which is the value create and migrate compute themselves. Creation and backfill therefore stay one rule. After backfill every record carries a non-zero `seq`, so `ParseJSONL`'s own backfill does nothing on the written bytes, and the cache equals a parse of them. That makes the comment at store.go:206 ("from the same bytes that were written") true again. Edge cases: a nil or empty returned slice is harmless (`NextSeq` returns 1 and the loop does nothing). Only a zero `Seq` counts as "none", which matches `ParseJSONL`.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/mutate_seq_test.go:75-139 `TestMutateNumbersUnnumberedTasks` runs four subtests against a real store through `storage.NewStore(...).Mutate`:
    - :78-87 (criterion 1): the fixture has seq 1/2/3 at `sameSecond` and priority 2, with IDs d/c/b and an appended `tick-000001` that sorts ahead of all three. The test asserts the raw `seq` values [1 2 3 4] via `rawSeqs` and the cache `seq` column via `assertCachedSeqs`.
    - :89-96 (criterion 2): `tick list` returns the appended task last, and `tick ready --count 1` returns `numberedFirstID`. No write occurs between the mutation and these reads, so they read the cache `Mutate` built.
    - :98-112 (criterion 3): the cache matches the expected numbers before and after `tick rebuild`, and `ReadTasks` returns the same numbers.
    - :114-138 (criterion 4): the merge shape [1,2,3,0,0,4,5] is written raw as [1 2 3 6 7 4 5], and the cache matches `mergeShapeSeqs`.
  - Each of these fails without the fix: `seq` is `omitempty`, so the appended record would be written with no `seq` and cached at 0. It would then list first and be returned by `ready --count 1`, and the raw decodes would show 0.
  - internal/cli/create_seq_test.go (criterion 5): `persistedSeqs` (:28-35) now decodes raw records through `rawRecords`, and `assertUniqueSeqs` (:51-60) takes the raw `storedRecord` slice and is called with `rawRecords` at :140. The first subtest reads `rawSeqs` at :76. The remaining `readPersistedTasks` uses (:17 ID-by-title, :70 titles, :154 `sameCreatedSecond`) read no sequence. `cachedSeq` at :106 reads the cache column, not the parser. Every tasks.jsonl sequence assertion now fails on a record written with seq 0.
  - internal/cli/backfill_seq_test.go:21-59: `storedRecord` and `rawRecords` factor out the raw decode, and `rawSeqs` is built on top of them. This is shared and not duplicated.
  - internal/migrate/store_creator_test.go:262-295: the `mockStore` subtests are unchanged.
- Notes: The tests are focused, with one subtest per criterion. The cache check repeated in subtest 3 before the rebuild sets up the baseline that criterion 3 compares against; it is not redundant.

CODE QUALITY:
- Project conventions: Followed. Tests use stdlib `testing`, `t.Run` subtests named "it …", `t.Helper()` on helpers, `t.TempDir` isolation through `setupTickProject`/`setupRawProject`, and TOON output decoded through `listedIDs`.
- SOLID principles: Good. `Mutate` reuses the existing `backfillSeqs` and adds no parallel rule.
- Complexity: Low. The change adds one line.
- Modern idioms: Yes (`strings.SplitSeq`, `range` over an int, `slices.Equal`).
- Readability: Good. The comments in the changed code are accurate, and none cite task IDs or spec sections.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "The rest of the suite passes unchanged, including the create and migrate `Seq: task.NextSeq(tasks)` assignments and the migrate `mockStore` subtests (§8.5)": settling this needs a full `go test ./...` run. Reading settles only that both `NextSeq` assignments and the `mockStore` subtests are unchanged, and that no `internal/storage/store_test.go` assertion compares raw bytes after a `Mutate` that appends a Seq-0 task.
