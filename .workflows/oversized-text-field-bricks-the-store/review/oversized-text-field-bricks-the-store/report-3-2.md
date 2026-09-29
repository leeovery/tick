TASK: Description Cap On Tick Update --description (oversized-text-field-bricks-the-store-3-2, tick-a34c9c)

ACCEPTANCE CRITERIA:
- An existing task: `tick update <id> --description` with exactly 50,000 characters exits 0 and stores the full description. The same holds for exactly 50,000 multibyte characters, and for exactly 50,000 characters wrapped in leading and trailing whitespace, which are stored trimmed (§6.1, §8.5)
- A store whose cache is current: `tick update <id> --description` with 50,001 characters exits 1. The error names the description field, the 50,000-character limit and the submitted length, 50,001, and says nothing was saved. `tasks.jsonl` and `cache.db` are byte-for-byte unchanged, and the task keeps its previous description (§6.2, §6.3, §8.5)
- `tick update <id> --title "New" --description` with 50,001 characters exits 1, and neither the title nor the description changes (§6.2, §6.3)
- `tick update <id> --description` with 50,001 characters wrapped in leading and trailing whitespace is refused, and the length it reports is 50,001 (§6.1, §6.3)
- A task whose stored description is 60,000 characters, written straight into `tasks.jsonl`: `tick show <id>` returns the full description, and `tick update <id> --title "New"` exits 0 with the title changed and the description unchanged (§6.2, §8.5)
- Same task: `tick start <id>` exits 0, and `tick note add <id> "text"` exits 0. After each, the description is unchanged (§6.2)

STATUS: complete

SPEC CONTEXT: §6.1 caps the description at 50,000 Unicode characters counted after trimming, as a constant beside the title and note caps. §6.2 requires every route that sets a description (create, update, migrate) to enforce it before anything is written, so a refused update exits 1 and leaves `tasks.jsonl` and the cache untouched. The cap applies only to a description being set: a stored over-cap description reads normally and does not block a status change, a retitle or a note. §6.3 requires the refusal to carry the field, the limit, the trimmed submitted length and that nothing was saved, with wording left to the implementer (set by Task 3-1).

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/update.go:194-202 (the `--description` branch trims, then calls `task.ValidateDescriptionUpdate` and `task.ValidateDescription`); internal/task/task.go:35 (`maxDescriptionLen = 50000`), internal/task/task.go:202-210 (`ValidateDescription`, the same function create uses at internal/cli/create.go:128)
- Notes: The cap check sits in the pre-store validation block and runs before `openStore` at internal/cli/update.go:247, so a refused update never opens the store, never takes the lock, and never touches `tasks.jsonl` or `cache.db`. That meets §6.2's "before anything is written" by construction. The title is validated first (update.go:183-188), but a combined `--title "New" --description <over-cap>` still returns before any write, so both fields stay unchanged. Counting uses `utf8.RuneCountInString` on the trimmed text, so multibyte and whitespace-padded input are counted as §6.1 requires. The reported length is the trimmed rune count. The cap is applied only when `opts.description != nil`; `--title`, status transitions and `note add` never reach `ValidateDescription`, so a stored over-cap description is left as it is. The refusal wording is shared with create (same function), which gives the "same refusal" the task asks for.

TESTS:
- Status: Adequate
- Coverage: `TestUpdateDescriptionCap` (internal/cli/update_test.go:1455-1624) maps one-to-one onto the criteria:
  - AC1: a table of three accepted cases at update_test.go:1479-1514 covers exactly 50,000 ASCII, exactly 50,000 four-byte runes, and 50,000 wrapped in " \t\n" padding. Each case checks the persisted (trimmed) description and the `show` output.
  - AC2: update_test.go:1516-1545 first runs `show` so the cache is built and current, then snapshots `tasks.jsonl` and `cache.db`. It asserts exit 1, the exact stderr (field, 50000 limit, 50001, "nothing was saved"), empty stdout, both files byte-identical, and the previous description kept.
  - AC3: update_test.go:1547-1563 asserts that neither the title nor the description changes.
  - AC4: update_test.go:1565-1578 pads 50,001 characters with whitespace and asserts that the refusal reports 50001.
  - AC5: update_test.go:1582-1598 seeds 60,000 characters through `setupTickProjectWithTasks`, which writes `storage.MarshalJSONL` output straight to `tasks.jsonl` and bypasses the CLI's validation (internal/cli/create_test.go:34-46). It then checks `show` and `update --title`.
  - AC6: update_test.go:1600-1623 runs `start` and then `note add`, and asserts the description is unchanged after each.
- Notes: Every test would fail if the cap were removed (the refusals would exit 0) or if it were misapplied to stored descriptions (the over-cap retitle, start and note cases would exit 1). The exact-string stderr assertion pins all four §6.3 elements. The helper closures keep each case focused, and there are no redundant variants. Phase 1's update test that crosses the old ceiling uses 11,000 characters (internal/cli/over_ceiling_write_test.go:72), well under the cap, so it is unaffected.

CODE QUALITY:
- Project conventions: Followed. Validation happens before store access, in the same style as the neighbouring title, type, tags and refs checks. Tests use stdlib `testing`, `t.Run`, `t.Helper` and "it does X" subtest names, and toon output is checked by decoding with `decodeToonDoc`.
- SOLID principles: Good. One domain validator in `internal/task` serves both create and update.
- Complexity: Low
- Modern idioms: Yes
- Readability: Good. The doc comment on `ValidateDescription` (task.go:202-203) holds true against the code.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
