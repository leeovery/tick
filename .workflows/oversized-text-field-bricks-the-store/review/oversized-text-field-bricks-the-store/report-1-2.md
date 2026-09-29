TASK: Stores Already Over The Old Ceiling Open And Operate (oversized-text-field-bricks-the-store-1-2, tick-98d76a)

ACCEPTANCE CRITERIA:
- A fixture store whose `tasks.jsonl` holds a task line of more than 65,536 bytes among ordinary tasks: `tick list` lists every task, and `tick show <oversized-id>` shows the oversized task (§2.1, §8.1)
- Same fixture: `tick rebuild` succeeds and reports every task rebuilt (§2.1, §8.1)
- Same fixture, with no migration or repair step first: `tick update <oversized-id>` succeeds, and `tick remove <oversized-id>` succeeds and the task is gone from the next `tick list` (§2.1, §8.1)
- A task whose line is about 1 MiB: `tick show <id>` in toon, pretty and JSON output each carries the task's content in full, with no truncation, after the task has passed through the SQLite cache (§8.1)

STATUS: complete

SPEC CONTEXT: §2.1 requires that a store already holding an over-ceiling line opens once tick is upgraded, with no migration or repair step, and that the oversized task can be shown, updated and removed like any other. §8.1 asks for a fixture store over the old ceiling on which list, show, rebuild, update and remove all succeed, and for a ~1 MiB task that round-trips through the SQLite cache and renders in toon, pretty and JSON. §6.2: a description already over the 50,000-character cap is left as it is and does not block other changes, so an update that does not set the description must still pass after Phase 3.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Task deliverable (test-only commit 78019769, the only commit touching the file): internal/cli/over_ceiling_store_test.go:14-24 (fixture), :42-86 (TestOverCeilingStore), :88-146 (TestMebibyteTaskRendering)
  - Behaviour it exercises, delivered by task 1-1: internal/jsonl/lines.go:29-48 (bufio.Reader.ReadBytes, with no per-line limit); internal/storage/jsonl.go:89-110 (ParseJSONL reads through jsonl.Lines)
  - Cache round-trip: internal/cli/show.go:96-104 (queryShowData reads the task, description included, from SQLite through store.Query)
  - Rebuild count: internal/cli/rebuild.go:18-27
  - Over-cap description does not block a title update: internal/cli/update.go:194-200 (the cap check runs only when --description is set)
- Notes: The fixture writes the 100,000-byte line straight into tasks.jsonl, between two ordinary tasks, with no cache.db (setupRawProject, internal/cli/list_order_test.go:46-54). This is how a store written before the fix holds it. Each subtest starts from a fresh fixture, so update and remove each run as the first command against the over-ceiling store, with no list or rebuild beforehand. That satisfies "no migration or repair step first". The fixture's description is about 99,850 characters, over the Phase 3 cap, and the update sets only --title, so it matches the task's Context note and §6.2. No production code was changed or needed for this task, which is consistent with the plan: task 1-1 makes the line readable, and this task proves the outcome end to end.

TESTS:
- Status: Adequate
- Coverage:
  - AC1: assertListedIDs checks that all three IDs are listed in order, including the task after the long line. `show` in toon decodes the document and asserts the id and the exact full description (over_ceiling_store_test.go:43-53).
  - AC2: `rebuild` output must equal "Cache rebuilt: 3 tasks\n", then list is checked again (:55-64).
  - AC3: `update --title Renamed` runs through runToonCommand, which fails the test on a non-zero exit (toon_decode_test.go:123-125). The test then asserts the new title, that the description is preserved in full, and that the store still lists every task (:66-77). `remove --force` is followed by a list that must hold only the two ordinary IDs (:79-85).
  - AC4: a description of 1,048,579 characters (16-byte chunks plus the "END" sentinel) is compared by exact equality in all three formats. Toon is decoded (:105-113). Pretty is checked as the exact remainder after "Description:\n" (:115-127). JSON is strictly unmarshalled, and both id and description are checked (:129-145). Every read goes through queryShowData, so the SQLite round-trip is exercised by construction.
  - Every assertion would fail if a ceiling returned, if the description were truncated, or if any command failed.
- Notes: The tests are focused, with no redundant assertions. Each subtest pins one criterion. The follow-up list after update is meaningful: it confirms that the rewritten store, still over the old ceiling, reopens.

CODE QUALITY:
- Project conventions: Followed. The tests use stdlib testing and `t.Run` subtests named "it does X". Helpers call `t.Helper()`, fixtures are isolated with `t.TempDir()`, and toon assertions decode the output instead of matching strings, as CLAUDE.md requires. Pretty keeps a golden-remainder comparison.
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes (numeric literal separators, `strings.Cut`)
- Readability: Good. The comments at :14-15 and store_line_reading_test.go:16-17 hold true against the code.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
