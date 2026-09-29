TASK: Status Changes, Dep Add And Cascades On A Large Record Leave The Store Readable (oversized-text-field-bricks-the-store-1-4, tick-b4f5f5)

ACCEPTANCE CRITERIA:
- An open parent whose line sits close enough under 65,536 bytes that the start cascade's transition takes it to 65,536 or more: `tick start <child>` exits 0 and moves the parent to `in_progress`, and the next `tick list` succeeds and lists both tasks (§1.1, §8.1)
- A task whose line a new dependency takes to 65,536 bytes or more: `tick dep add` on that task exits 0 with its normal output, and the next `tick list` succeeds (§1.1, §8.1)
- A task whose line its own recorded transition takes to 65,536 bytes or more: each status change (`tick start`, `tick done`, `tick cancel`, `tick reopen`) exits 0 with its normal output, and the next `tick list` succeeds (§1.1, §8.1)
- A done parent whose line the Rule 6 reopen takes to 65,536 bytes or more: `tick create --parent <parent>` exits 0 and reopens the parent, and the next `tick list` succeeds (§1.1, §2.1)
- A task whose line a `--blocks` link takes to 65,536 bytes or more: `tick create --blocks <task>` exits 0, and the next `tick list` succeeds (§1.1, §2.1)

STATUS: complete

SPEC CONTEXT: §1.1 names status changes and `dep add` as routes that exit 0 while pushing a line over the old 65,536-byte scanner ceiling, and cascades, `--blocks` and the Rule 6 done-parent reopen as routes that grow a task the command never named. §2.1 requires every reader of tasks.jsonl to accept any line length so none of these routes can make the store unreadable. §7 says the post-write signal needs no change of its own: once the reader has no ceiling the gap between a reported success and an unreadable store disappears, and transition and blocked_by counts stay unbounded. §8.1 asks for tests that `start <child>` under a very large parent, `dep add`, and status changes on a large record leave the next `list` succeeding.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/over_ceiling_growth_test.go:56-155 (the task's only commit, b219b281, adds this file and nothing else; the file is unchanged since). The behaviour it pins comes from the ceiling-free shared reader at internal/jsonl/lines.go:29-48 (bufio.Reader.ReadBytes, no token limit), consumed by storage.ParseJSONL at internal/storage/jsonl.go:88-109.
- Notes: This is a test-only task, as §7 prescribes: no production change is needed for these routes once the reader accepts any line. Each criterion maps to a subtest:
  - AC1 start cascade: :67-82. Parent padded to 8 bytes under the ceiling (nearCeiling, :18-28), `start <child>` asserted exit 0 through runToonCommand, changed rows child (auto=false) then parent (auto=true) open->in_progress, parent line >= 65,536, `list` lists both, `show` confirms the parent is stored in_progress.
  - AC2 dep add: :84-96. Exact normal output `Dependency added: tick-aaa111 blocked by tick-bbb222\n` (matches baseFormatter.FormatDepChange at internal/cli/format.go:305-310), line >= 65,536, `list` lists both.
  - AC3 status changes: :98-123. Table over start/done/cancel/reopen with realistic seed states (in_progress for done, done with Closed set for reopen), each asserting the single changed row (id, title, from, to, auto=false), line >= 65,536, `list`, and the stored status via `show`.
  - AC4 Rule 6: :125-137. `create Child --parent <done parent>` exit 0, changed row done->open auto=true, parent line >= 65,536, `list`, stored status open.
  - AC5 --blocks: :139-154. `create Blocker --blocks <target>` exit 0, target line >= 65,536, `list`, and the target's blocked_by shows the new blocker.
  Byte arithmetic checked by reading: nearCeiling leaves the seed line at 65,528 bytes; setupNearCeilingProject (:30-37) confirms it is under the ceiling before the command, and assertLineAtLeastOldCeiling confirms it is at or over after. The smallest net growth among the routes (reopen: a transition record of about 85 bytes minus the 32-byte `closed` field) is well over the 8-byte gap, so the comment at :12-13 holds. The pre-fix bufio.Scanner could read a 65,528-byte seed line but not the grown one, so every subtest reproduces the original defect and fails if a ceiling returns.

TESTS:
- Status: Adequate
- Coverage: All five criteria are covered, each with a before/after line-length guard that proves the ceiling is crossed by the command's own write, exit-0 plus normal-output checks on the decoded document, and a following `tick list`. The unnamed-task routes (start cascade parent, Rule 6 parent, --blocks target) check the grown task, not the named one. Stored-state checks through `show` confirm the write persisted, not just the output.
- Notes: Not over-tested. The `show` status check and the changed-row check look alike but cover different things: the changed row is the command's output, and `show` is the state read back from the store. Assertions decode toon and check values, per project convention.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, t.Run subtests named "it ...", t.Helper() on every helper, t.TempDir isolation through setupTickProject, toon assertions via decodeToonDoc/toonRows/assertToonFields; reuses oldLineCeiling, storedLineLen and assertLineAtLeastOldCeiling from over_ceiling_write_test.go rather than duplicating them)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes
- Readability: Good. nearCeiling, setupNearCeilingProject, assertShownStatus and assertChangedRows are small and named for intent. The status-change table keeps the four routes uniform.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
