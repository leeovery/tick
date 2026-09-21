TASK: free-text-round-trip-2-5 (tick-1d9e65) — Create And Update Emit One Document

ACCEPTANCE CRITERIA:
- `tick create` and `tick update` stdout decodes as exactly one TOON document and, under `--json`, parses as exactly one JSON object
- `tick create` with no `--parent` emits `changed[0]{id,title,from,to,auto}:`
- `tick create --parent <done task>` emits one row for the reopened parent and one for each ancestor the reopen travelled to, every row `auto=true`
- `tick update <id> --parent <other>` firing both Rule 6 and Rule 3 emits one table containing both blocks' rows, each task once
- A task named by both blocks that ends where it started carries no row
- Rows appear in block order — Rule 6's before Rule 3's — matching the order the two blocks print in pretty today
- `tick create --quiet` and `tick update --quiet` print only the task ID and nothing else
- Pretty output for both commands is byte-identical to before this task, including the blank line before `Cascaded:` and the order of two blocks
- `grep -n 'outputStatusChanges' internal/cli/create.go internal/cli/update.go` returns nothing
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§7.4 requires that where a command produces both a record and status changes, the result is one document with the changes as a section inside it — never a detail document with loose transition lines appended — and that the section is always present, `changed[0]{...}` when nothing moved. §7.2 requires one row per task that moved, a task appearing at most once, reading from the status it held before the command to the status it holds after, and no row at all for a task that ends where it started. §7.5 records why `create`/`update` produce cascades at all (structural change, not the edited task's own status) and that `update` can fire Rule 6 and Rule 3 at once. §7.3 keeps the full record on these two commands only. §4.1 keeps pretty byte-identical.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/create.go:172 (`var blocks []CascadeResult`), internal/cli/create.go:238 (block appended inside the `Mutate` closure), internal/cli/create.go:277-278 (`changes := &StatusChanges{Blocks: blocks}` handed to `outputMutationResult`)
  - internal/cli/update.go:271-273 (ordering comment + `blocks` slice), internal/cli/update.go:311 (Rule 6 block appended), internal/cli/update.go:368 (Rule 3 block appended), internal/cli/update.go:397-398 (`StatusChanges` built and handed to `outputMutationResult`)
  - internal/cli/format.go:142-151 (`StatusChanges` and its `Rows()` merge), internal/cli/transition.go:149-159 (`mergeStatusChanges`), internal/cli/transition.go:114-147 (`statusChangeSet`: first-seen order, first-From/last-To collapse, drop of unchanged tasks)
  - Rendering: internal/cli/toon_formatter.go:98-100 and :146-156; internal/cli/json_formatter.go:135-137; internal/cli/pretty_formatter.go:231-235
- Notes:
  - Both handlers pass a non-nil `*StatusChanges` on every path, so the `changed` section is always present for `create` and `update`; `note add`/`note remove` still pass nil (internal/cli/note.go:87, :145), matching §7.3.
  - Blocks are built inside the `Mutate` closure while the `tasks` slice is valid; only the merge and rendering happen after it returns. `Store.Mutate` calls the closure exactly once (internal/storage/store.go:187), so appending rather than assigning cannot double-count.
  - `grep -n 'outputStatusChanges' internal/cli/create.go internal/cli/update.go` returns nothing; the helper survives only for the transition commands (internal/cli/helpers.go:109, called from internal/cli/transition.go:58) and is not orphaned.
  - Drift from the plan's wording, and sound: the plan said `StatusChanges{Blocks: blocks, Rows: mergeStatusChanges(blocks...)}`; the delivered shape carries `Blocks` only and exposes `Rows()` as a method (internal/cli/format.go:149-151). Same result, no denormalised field to keep in sync — the commit for this task (1d774656) did build the `Rows` field and a later task folded it into a method. No loss.
  - `auto=true` on every row of these two commands is upheld structurally: both cascade sources go through `ApplySystemTransition` (internal/cli/helpers.go:127 via `validateAndReopenParent`, internal/cli/update.go:153 via `autoCompleteParentIfTerminal`), so `TransitionResult.Auto` is true and `buildCascadeResult` carries it into `PrimaryAuto`; cascade entries are hard-coded `Auto: true` (internal/cli/format.go:222).
  - Pretty byte-identity holds by construction: the old path was `Fprintln(detail)` then `Fprintln(cascade)`; the new path is `Fprintln(detail + "\n" + cascade)` (internal/cli/pretty_formatter.go:233, internal/cli/helpers.go:37-39), which emits the same bytes for one block and for two, and `cascadeTransition` still writes its own `"\n\nCascaded:"` (internal/cli/pretty_formatter.go:333).
  - `--fields`/`--field` are registered for `show` only (internal/cli/flags.go:71-74), so the JSON formatter's selection gate on the `changed` key cannot diverge from toon's ungated one for these two commands.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/create_test.go:1372-1503 (`TestCreateChangedSection`): whole-stdout TOON decode plus section-key order under a done parent (:1382), `changed[0]` with no parent (:1406), reopen travelling to a done grandparent as two rows `auto=true` (:1417), whole-stdout `json.Unmarshal` (:1445), `--quiet` emitting only the ID (:1476), pretty suffix unchanged (:1490).
  - internal/cli/update_test.go:1322-1453 (`TestUpdateChangedSection`): Rule 6 + Rule 3 collapsed into one table in block order (:1337), shared ancestor reopened then re-completed carrying no row (:1362), one JSON object (:1391), `changed[0]` when nothing moved (:1412), `--quiet` (:1426), pretty two-block suffix unchanged (:1439).
  - The "one document" criterion is genuinely exercised: `decodeToonDoc` runs `toon.DecodeString` over the entire stdout (internal/cli/toon_decode_test.go:17-22) and `toonSectionKeys` asserts the exact section sequence, so a loose trailing transition line would fail both.
  - The shared-ancestor fixture (update_test.go:1363-1384) is correctly constructed: moving `tick-ccc333` under `tick-bbb222` reopens `bbb222` and cascades the reopen to `ggg111`, then Rule 3 completes `aaa111` and cascades completion back up through `bbb222` and `ggg111`; only `aaa111` ends somewhere new, and the test asserts exactly one row.
  - Pre-existing pretty assertions kept rather than rewritten: create_test.go:1255-1283 ("it displays cascade output after created task detail"), create_test.go:1176-1212 (no cascade when the parent is not done), update_test.go:1033-1068 (Rule 3 pretty output).
  - Unit-level coverage of the same section sits in internal/cli/detail_changes_test.go (Task 4's surface), including pretty block-append equality against `FormatCascadeTransition` (:166-192), which pins the "blank line before `Cascaded:`" spacing that the handler-level tests only assert a suffix of.
  - `internal/cli/conformance_test.go:464-530` independently runs `create`/`update` (including the shared-ancestor case) through the document-validity harness.
- Notes: No redundancy worth flagging. The overlap between the handler-level tests and the conformance cases is complementary — conformance asserts "it decodes", these assert which rows come out and in what order. No mocking, no implementation-detail assertions; every subtest drives the real CLI through a temp `.tick` directory.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing` with `t.Run` "it does X" subtests, `t.Helper()` on helpers, temp-dir isolation, pretty flag omitted where toon is the subject and passed explicitly where the golden string is. Handler signature and `Store.Mutate` closure discipline unchanged.
- SOLID principles: Good. The handlers now collect and hand off; merging lives in `mergeStatusChanges`/`statusChangeSet` and rendering in the formatters. Replacing the `parentReopened`/`r6Triggered`/`r3CascadeResult` flag-and-pointer families with one ordered slice removes the coupling between "did it fire" and "what fired".
- Complexity: Low. Both handlers lost a branch each at the tail; the surviving code paths are linear.
- Modern idioms: Yes. Value-copy semantics of `CascadeResult` are relied on deliberately and documented at internal/cli/transition.go:149-151.
- Readability: Good. The ordering comment at internal/cli/update.go:271-272 states the one thing the slice order encodes and is true of the code beneath it.
- Comment accuracy: The comments in the changed code hold. internal/cli/format.go:107-108 ("Changes is nil when the document carries no changed section…"), internal/cli/format.go:142-143, internal/cli/transition.go:149-151 and internal/cli/update.go:271-272 all match the code; none reference task ids or phases.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — reading cannot settle a toolchain run. Requires executing those three commands from the repo root; the code reads as compiling and formatted, but nothing here measures it.
