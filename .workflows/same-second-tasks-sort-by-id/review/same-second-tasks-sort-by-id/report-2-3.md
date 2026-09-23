TASK: Carry The Sub-List Order Into Mutation Detail Documents (same-second-tasks-sort-by-id-2-3, tick-9a9967)

ACCEPTANCE CRITERIA:
- A parent P whose children c1, c2, c3 record one creation second and carry sequences in that authoring order, with ascending-ID order and line order in `tasks.jsonl` both contradicting it. P's `blocked_by` array declares a later-created blocker first, against ascending-ID order. `tick update P --title Renamed` prints a detail document whose children section lists c1, c2, c3 and whose blocked_by section lists the first-declared blocker first, the same order `tick show P` gives (§4.4, §8.4)
- Same fixture: `tick note add P "text"` prints a detail document whose children and blocked_by sections match `tick show P`'s order (§4.4, §8.4)
- Same fixture, with a note on P: `tick note remove P 1` prints a detail document whose children and blocked_by sections match `tick show P`'s order (§4.4, §8.4)
- Two existing tasks, the later-created of which has the higher ID: `tick create T --blocked-by <later>,<earlier>` prints a detail document whose blocked_by section lists the later-created blocker first. That order matches the new record's `blocked_by` array and `tick show T` (§4.3, §4.4, §8.4)
- The conformance inventory's entries, including those for `create`, `update`, `note add` and `note remove`, still decode under both the TOON and JSON drivers. The existing conformance and README-sample tests pass without any fixture or sample changing (§4.4, §7.3)

STATUS: complete

SPEC CONTEXT: §4.3 moves `tick show`'s children to created, seq, id and its blockers to the dependency ordinal (declaration order). §4.4 says the children and blocked_by sections render from five sites: `tick show` and `outputMutationResult`, which serves create, update, note add and note remove. Those detail documents and the conformance inventory are in scope. Nothing in the suite pinned the sub-list order before this work, so the new assertions are the only guard against drift. §8's fixture rules require IDs that contradict the expected order and tied creation seconds wherever the sequence decides. §8.4 also requires the declaration order to contradict both ascending-ID order and blocker creation order.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/helpers.go:17-31: `outputMutationResult` builds its detail from `queryShowData` (line 23), the same query `tick show` uses, so the order reaches all four mutation documents.
  - internal/cli/show.go:137 (`ORDER BY d.ordinal`) and internal/cli/show.go:146 (`ORDER BY created ASC, seq ASC, id ASC`) are the orderings this task pins.
  - The handlers call `outputMutationResult` at internal/cli/create.go:279, internal/cli/update.go:398, internal/cli/note.go:87 and internal/cli/note.go:145.
  - internal/storage/cache.go:213-214 writes the ordinal from the `BlockedBy` slice index. internal/cli/create.go:66 and :171-172 keep the `--blocked-by` IDs in the order given.
- Notes: This task adds tests only (commit 9f2f229b adds only internal/cli/mutation_sublist_order_test.go). The plan cites create.go:278; the call is now at create.go:279 because of earlier create.go changes in this feature. That is only line drift. Nothing is missing.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/mutation_sublist_order_test.go:18-28, fixture: the children are c1=tick-bbb222 (seq 2), c2=tick-ccc333 (seq 3) and c3=tick-aaa111 (seq 4), all created at `sameSecond` (list_order_test.go:339). Ascending-ID order would be c3, c1, c2 and line order is c2, c3, c1, so both contradict c1, c2, c3. P's `blocked_by` is [tick-000e02 (seq 6), tick-000b01 (seq 5)]. The first-declared blocker is later-created and has the higher ID, so declaration order contradicts both ascending-ID order and blocker creation order, as §8.4 requires.
  - :43-52: `assertSubListsMatchShow` checks the mutation document against the fixed expected orders and against a fresh `tick show P` on the same state. Because the expected orders are fixed, the test still fails if show and the mutation paths drift together.
  - :55-78 cover update --title, note add, and note remove after an added note, one subtest per criterion.
  - :80-96, create: the two existing blockers tie on the creation second, and the later one has the higher ID. The test asserts that the created document's blocked_by is [later, earlier], that it equals the stored record's `blocked_by` (`storedBlockedBy`), and that it equals `tick show T` (`shownBlockerIDs`). create's children are not pinned, as the task intends.
- Notes: Each subtest would fail if its path fell back to ID order, line order or blocker creation order. No subtest is redundant. There is no mocking. `sectionIDs` does not collide with any other identifier in package cli. Conformance side: internal/cli/conformance_test.go is unchanged across the feature (`git diff 937a299e^ HEAD`). README.md changes only prose at lines 115 and 396, outside every fence. The readme_samples_test.go changes add prose assertions and extract `readmeProse`; no sample group or fixture changed. The only README sample with these sections (README.md:505, :508) has one blocker and no children, so reordering cannot affect it. Conformance entries are decode-only and do not depend on order.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib `testing`, `t.Run` subtests named "it …", `t.Helper()` on helpers, `setupRawProject` for isolation, and decoded-TOON assertions (`decodeToonDoc`, `toonRows`) as CLAUDE.md prescribes.
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes
- Readability: Good. The fixture comment at lines 5-6 is accurate: the children share one second, their IDs and line order contradict the authoring order, and the later-created blocker is declared first and has the higher ID.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "The conformance inventory's entries, including those for `create`, `update`, `note add` and `note remove`, still decode under both the TOON and JSON drivers. The existing conformance and README-sample tests pass without any fixture or sample changing" — reading settles the "without any fixture or sample changing" part (conformance_test.go untouched; no README fence or sample group changed). Settling "still decode" and "pass" requires running `go test ./internal/cli -run 'TestConformance|TestREADME' -count=1` and observing it pass.
