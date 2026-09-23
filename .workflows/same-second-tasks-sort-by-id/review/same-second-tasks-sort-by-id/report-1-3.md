TASK: Assign A Sequence When A Task Is Created (same-second-tasks-sort-by-id-1-3, tick-0bb4f7)

ACCEPTANCE CRITERIA:
- In an empty project, three successive `tick create` calls write tasks carrying sequences 1, 2 and 3 in creation order. Numbering begins at 1 and no created task is ever written with sequence 0 (§2.2, §8.2)
- A project whose records carry sequences 3, 7 and 5, in that line order: `tick create` writes the new task with sequence 8, one above the highest, not above the last line's value and not the record count (§2.2)
- A task created by `tick create` keeps the same sequence in `tasks.jsonl` and in the cache's `seq` column after a `tick update` of that task, a `tick remove` of another task, further `tick create` calls, and `tick rebuild` (§2.1, §8.2)
- Tasks A, B and C created in that order carry 1, 2 and 3. After `tick remove` of C, the next `tick create` writes sequence 3, and no two tasks in the file share a sequence (§2.2, §7.2)
- Tasks created by successive `tick create` calls that record the same creation second and the same priority: `tick list` returns them in creation order, a result ascending task-ID order could not produce (§1.3, §8.1)

STATUS: issues_found

SPEC CONTEXT: §2.1 says every task carries a creation sequence that is assigned at creation and never renumbered. §2.2 says a new task takes one above the highest sequence in the task set read under the exclusive lock. Numbering starts at 1, a zero or absent value means no sequence, and a number freed by removing the highest task may be reused (§7.2 accepts this). §8.2 requires the sequence to survive create, update, remove and rebuild. §8 fixture constraints: the creation seconds must tie, and because IDs are random, the ordering assertion must be one that an ascending-ID result cannot satisfy.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/create.go:221 (`Seq: task.NextSeq(tasks)`, inside the `store.Mutate` closure at internal/cli/create.go:187, so `tasks` is the set read under the exclusive lock); internal/task/task.go:272-280 (`NextSeq`: one above the highest, 1 when none); internal/task/task.go:57, :77, :98, :136 (the `seq` field round-trips through JSON with `omitempty`); internal/storage/cache.go:139, :208 (the cache insert writes `t.Seq`); internal/cli/list.go:313, :315 (the sort ends on `t.created, t.seq, t.id`)
- Notes: `NextSeq` runs over the set after `ParseJSONL` has backfilled it (internal/storage/jsonl.go:116, called from internal/storage/store.go:365 via `readAndEnsureFresh` in `Mutate`), so a new task is numbered above backfilled values as well, as §2.2 requires. `Store.Mutate`'s later `backfillSeqs(mutated)` (internal/storage/store.go:192) leaves the created task alone because it already carries a non-zero sequence. The per-site `NextSeq` call was kept on purpose (plan task 1-6). `tick update` changes fields in place (internal/cli/update.go:322-362), so `Seq` survives it. There is no drift from the plan or the spec.

TESTS:
- Status: Adequate
- Coverage: internal/cli/create_seq_test.go:62-163 (`TestCreateAssignsSequence`) has one subtest per criterion. Subtest 1 reads the raw on-disk sequences, so a sequence written as 0 would show up. Subtest 2 uses the 3/7/5 fixture and expects 8, which rules out the last-line and record-count answers. Subtest 3 checks both the JSONL value and the cache `seq` column after each step. Subtest 4 covers reuse of the removed highest number and checks uniqueness. Subtest 5 retries until a five-task batch shares one second and its random IDs are not ascending, so an ascending-ID result could not pass it. internal/task/task_test.go:651-672 (`TestNextSeq`) unit-tests the rule itself (empty set, all-zero set, 3/7/5 → 8). The tests are focused and not redundant.
- Notes: Subtest 3's survival check cannot detect renumbering (see FINDINGS). Subtest 5 depends on the wall clock and retries up to 20 attempts. This is a reasonable way to satisfy the random-ID constraint in §8.

CODE QUALITY:
- Project conventions: Followed (stdlib `testing`, `t.Run` subtests with "it …" names, `t.Helper()` on helpers, `t.TempDir()` isolation)
- SOLID principles: Good. The numbering rule has one home (`task.NextSeq`), shared by create and backfill, as §2.3 asks ("backfill and creation are a single rule")
- Complexity: Low
- Modern idioms: Yes (`max` builtin, `range` over int, `strings.SplitSeq`)
- Readability: Good. Comments hold against the code
- Issues: None beyond the test finding below

BLOCKING ISSUES:
- None

FINDINGS:
- [in-scope] [contained] internal/cli/create_seq_test.go:97-99 — The survival subtest tracks the first task created. That task carries seq 1 and stays on record 1 through every step, because "other" is created after it and is the one removed at :114. Any order-preserving renumbering also assigns it 1, whether it renumbers by record position or closes gaps after a remove. So the assertion at :103-108 holds whether or not sequences are renumbered. Fix: create "other" before "kept" (swap lines 97 and 98) and set `wantSeq = 2`. After `remove` of "other", "kept" sits on record 1 with seq 2, so a positional or gap-closing renumber would write 1 and fail the check. The later creates (3, 4) and rebuild are unaffected. — FAILS: a `remove`, `update` or rebuild path that renumbered surviving tasks would pass this test even though it breaks the §2.1 "never renumbered afterwards" rule the subtest is named for. No other test catches a remove that closes gaps: `TestBackfillSequence`'s update/remove/create subtest (internal/cli/backfill_seq_test.go:130-146) only checks ascending order, and the reuse subtest (internal/cli/create_seq_test.go:123-141) removes the last task, which leaves no gap.

UNSETTLED:
- None
