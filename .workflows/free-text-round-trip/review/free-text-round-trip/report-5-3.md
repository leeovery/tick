TASK: free-text-round-trip-5-3 (tick-7962a1) — Note Add Stops Inspecting Flags After The Task ID

ACCEPTANCE CRITERIA:
- `tick note add <id> "- read the header"` exits zero and stores the note text `- read the header`
- Any other dash-leading text after the ID is accepted, including text beginning with two dashes
- A flag-shaped first argument is still inspected: `tick note add --bogus <id> text` exits non-zero with the unknown-flag error
- `tick note add <id> text --json` prints a JSON document, unchanged from today
- Note text that spells a global flag exactly still needs the marker: `tick note add <id> -- --json` stores `--json`
- `tick note remove <id> --bogus 1` still exits non-zero with the unknown-flag error
- Every other command's validation is unchanged — `flagScanLimit` has exactly one entry
- `tick note add` with no arguments still reports `task ID is required. Usage: tick note add <task_id> <text>`
- `tick note add <id>` with no text still reports `note text is required and cannot be empty`
- Multi-word text still joins with single spaces
- `commandFlags["note add"]` stays empty and `note`'s help entry is unchanged, so `TestCommandFlagsMatchHelp` passes untouched
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §10.2, second half of the fix — flag inspection stops after the task ID on `note add`, because the command registers no flags at all, so nothing dash-leading after the ID could be a flag the check would have caught. §10.2 is explicit that what stops is the check, not flag handling: global flags are consumed by `parseArgs` wherever they appear before the marker, so `--json` after the note text still resolves, and note text that spells a global flag exactly still needs `--`. §10.1 frames this as the one hole in the §2.2 round-trip guarantee. §10.2 also rules `create` out of this half of the fix (its title shares the argument list with real flags), which the single-entry map respects.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/flags.go:98-103 — `flagScanLimit` declared beside `commandFlags`, one entry `"note add": 1`, documented as the number of leading arguments inspected for a command whose remaining arguments are free text by definition.
  - internal/cli/flags.go:132-137 — `ValidateFlags` computes `scanEnd := len(args)`, narrows it via `min(limit, scanEnd)` when the command is in the map, and bounds the existing loop on `scanEnd`. A command absent from the map keeps `scanEnd == len(args)`, i.e. the prior behaviour byte for byte (diff in commit 11a5ee7e changes only the loop bound).
  - internal/cli/flags.go:80 — `"note add": {}` is unchanged, so the registry and the help entry stay in sync.
  - internal/cli/app.go:116-117 — `qualifyCommand` strips the `add` sub-subcommand before validation, so the args `ValidateFlags` sees for `"note add"` begin at the task ID; a limit of 1 therefore inspects exactly the ID slot. Verified against `qualifyCommand` at app.go:401-420.
- Notes:
  - Criterion-by-criterion, by reading: dash-leading and two-dash text after the ID reaches index 1+, past `scanEnd`, so it is never inspected. A flag in the first position is still at index 0 and still rejected — which is also why the two `note add`/`note remove` rows in internal/cli/unknown_flag_test.go:106-107 keep passing unmodified (both put `--unknown` in the first position after the sub-subcommand).
  - `--json` after the note text is stripped by `parseArgs` (app.go:352-392) before `ValidateFlags` or the handler sees the arguments, so JSON output is unaffected by the limit; text spelling a global flag exactly is consumed the same way and needs the marker, matching §10.2.
  - Missing-ID, empty-text and multi-word-join behaviour live in `RunNoteAdd` (internal/cli/note.go:39-56) and are untouched by this change.
  - The limit is keyed on the fully-qualified `"note add"`, which is what `qualifyCommand` produces; no other call site consults it (`grep` finds `flagScanLimit` only at flags.go:98, 101, 128, 133 plus the test).

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/note_test.go:735 `TestNoteAddFreeText` — six subtests through `App.Run`: dash-leading text stored verbatim (757), two-dash text stored verbatim (769), `--bogus` before the ID still rejected with the exact stderr string (781), `--json` after the text still yields a parseable JSON document (794), `-- --json` stores `--json` (807), `note remove <id> --bogus 1` still rejected with the exact stderr string (819). Each asserts the persisted note text read back off `tasks.jsonl`, not just the exit code.
  - internal/cli/flag_validation_test.go:385 `TestFlagScanLimit` — unit subtests over `ValidateFlags`: the limit applies past the ID on `note add`, the first argument is still inspected, and every other key in `commandFlags` still rejects `--bogus` in the second position, with `len(flagScanLimit) != 1` failing fast.
  - Pre-existing coverage carries the unchanged criteria: note_test.go:62 (multi-word join), :90 (missing ID), :102 (empty text), and flag_validation_test.go:318 `TestCommandFlagsMatchHelp` (registry/help sync, reads `info.Flags` structurally so the new `--` guidance in the note help text cannot be mistaken for a flag).
  - The tests pin the limit's value, not merely its presence: raising it to 2 breaks the dash-leading subtest, removing it breaks nothing else but is caught by the same subtest; moving the single entry to another command fails both the `note add` subtest and the len check.
- Notes: The unit subtest "it still inspects the first note add argument" overlaps the integration subtest "it still rejects a flag before the task id", but at a different layer (`ValidateFlags` directly vs. `App.Run` end to end, where only the latter pins the stderr text). Not redundant enough to matter, and both were prescribed by the task.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` "it does X" subtest naming, `t.Helper()` on the local `addedNoteText` helper, `t.TempDir()`-backed project setup via `setupTickProjectWithTasks`, tests driven through `App.Run` with `IsTTY: true` per the existing `runNote` helper.
- SOLID principles: Good — the rule is data beside the data it qualifies (`flagScanLimit` next to `commandFlags`), not a `note add` branch buried inside `ValidateFlags`; adding another free-text command is a map entry.
- Complexity: Low — three added lines of control flow, one extra loop bound.
- Modern idioms: Yes — builtin `min` (go 1.26 in go.mod), comma-ok map lookup. The `for i := 0; i < scanEnd; i++` form is retained correctly: the body advances `i` to skip a flag's value, so the `range`-over-int rewrite does not apply.
- Readability: Good — `scanEnd`/`capped` read plainly; the doc comment on `ValidateFlags` (flags.go:128) gained one accurate line about the limit.
- Issues: None. Comments in the changed code hold against it and carry no process artifacts.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the toolchain; reading shows no formatting or vet hazard in the change, but the suite, vet and gofmt were not executed here.
