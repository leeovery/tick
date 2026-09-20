# Consolidation Tasks: Free Text Round Trip (Phase 6)

## Task 1: The Inventory's Declared Command Must Match The Arguments It Runs
placement: phase 6
severity: behaviour

**Problem**: `TestConformanceInventoryCoversEveryCommand` proves every key of `commandFlags` is claimed by exactly one of the must-parse, prose and out-of-scope sets, and it builds the must-parse set from `conformanceDoc.Command` alone (`internal/cli/conformance_test.go:1486-1499`, `:1526-1533`). `Command` is free text written beside the `Setup` args, and nothing checks the two against each other (`TestConformanceInventoryWellFormed`, `:762-822`, checks duplicate names and the `Setup`/`NotADocument` exclusivity only). An entry whose `Command` reads `ready` while its `Setup` runs `list --ready`, or a copy-pasted entry keeping `Command: "list"` on a `blocked` setup, makes the guard green for a command no entry ever runs. The gap is silent for the object-shaped commands: `jsonShapeProblem` (`:673-684`) only distinguishes list-shaped from object-shaped documents, so swapping `show` for `stats` or `create` for `update` in a `Command` field raises nothing. Nobody notices until a malformed document ships from the uncovered command with the whole conformance suite passing.

**Solution**: Extend `inventoryProblems` (`internal/cli/conformance_test.go:763`) with a check that each entry's `Setup` args begin with `strings.Fields(entry.Command)`, reporting the entry name and both spellings when they disagree; entries carrying `NotADocument` have no `Setup` and are skipped. Give it a negative case in `TestConformanceInventoryWellFormed` alongside the existing three, in their fixture style. The current inventory already satisfies the rule, so no entry changes.

**Outcome**: The coverage guard can no longer certify a command the suite never exercises, because an entry's declared command and the command it runs are held together by a test.

**Do**: (line numbers are the current tree; the findings' references sit 11 lines higher, predating the comment-correction commit)
- Extend the `inventoryProblems` closure in `TestConformanceInventoryWellFormed` (`internal/cli/conformance_test.go:752-771`) so an entry carrying a `Setup` and no `NotADocument` reaches a new check rather than the loop's `continue`: call `entry.Setup(t)` for its args, take `strings.Fields(entry.Command)`, and append a problem when the args are shorter than those fields or do not begin with them. The message names the entry, the declared command and the args the setup returned. An entry carrying both a setup and a reason, or neither, keeps reporting its single existing problem and skips the check.
- Update the fixture in "it rejects a duplicate document name" (`:779-789`) so both entries' `Setup` returns args beginning with their `Command` — `[]string{"list"}` rather than `nil` — leaving the subtest asserting one problem, the duplicate name.
- Add two negative subtests alongside the existing three, in their fixture style: an entry declaring `ready` whose setup runs `list --ready`, and an entry declaring `dep tree` whose setup returns `[]string{"dep"}`. Each expects exactly one problem, and the mismatch message carries the declared command and the args.
- Add a positive subtest that an entry whose setup returns the command's words followed by positionals and flags — `show tick-a00001 --field notes` — yields no problem, pinning the rule as a prefix rather than an equality.
- Leave `conformanceDocs` (`:199-528`) unedited: "it accepts the inventory" now runs all 45 setups and must stay green, the multi-word `dep tree`, `note add` and `note remove` entries included.

**Acceptance Criteria**:
- [ ] `inventoryProblems` reports one problem for each entry whose setup args do not begin with `strings.Fields(entry.Command)`, naming the entry, the declared command and the args
- [ ] An entry whose args are shorter than its declared command's words is reported, not a panic on the slice
- [ ] Args beginning with the command's words and continuing with flags or positionals raise nothing
- [ ] An entry carrying `NotADocument` and no setup is not checked, and the both/neither negative subtests each still report exactly one problem
- [ ] "it accepts the inventory" passes against `conformanceDocs` with no inventory entry edited, and `go test ./internal/cli` is green

**Tests**:
- `"it accepts the inventory"` (existing, now holding every entry's declared command against the args its setup returns)
- `"it rejects an entry whose setup runs a different command"`
- `"it rejects an entry whose setup args are shorter than its declared command"`
- `"it accepts an entry whose setup args carry flags after the command"`

## Task 2: The Prose Exemption Is Declared Per Format
placement: phase 6
severity: behaviour

**Problem**: `conformanceProseCommands` (`internal/cli/conformance_test.go:1478-1480`) declares `dep add`, `dep remove`, `remove`, `init` and `rebuild` to emit a confirmation rather than a document. That holds for toon and pretty, where `baseFormatter` returns plain text. It is false under `--json`, where all five already return an object and predate this work: `FormatDepChange` (`internal/cli/json_formatter.go:193`) returns `{"action","task_id","blocked_by"}`, `FormatMessage` (`:265`) returns `{"message"}` for `init` and `rebuild`, and `FormatRemoval` (`:283`) returns `{"removed":[…],"deps_updated":[…]}`. `TestJSONOutputConformance` drives the same single inventory (`:1566-1570`), so it never runs those five under `--json`, and the coverage guard — whose whole job is to catch exactly this gap — reports the inventory complete because the prose set claims them. `jsonConformanceListKeys` (`:639-642`) also omits `removed` and `deps_updated`, the removal document's two always-array keys. An agent running `tick remove <id> --json` or `tick dep add … --json` gets a stream nothing end-to-end asserts is one parseable JSON value, and nothing asserts those two keys are arrays rather than `null`. Existing coverage is formatter-level only (`json_formatter_test.go:451`, `:531`), so a handler printing anything alongside the object — the shape §4.2 records the dep-tree handler as having had — ships with the suite green.

**Solution**: Make the exemption per-format rather than global: an entry declares which drivers feed it, so the five stay out of the toon driver and each gains a must-parse entry the JSON driver runs, while `conformanceCoverageProblems` keeps counting each command exactly once. Add `removed` and `deps_updated` to `jsonConformanceListKeys`. Correct the comment at `internal/cli/conformance_test.go:1478-1479` in the same edit — the commands are prose in toon and pretty, not in JSON. The direction is settled against the specification as corrected this pass: §3.2's corrigendum of 2026-09-20 scopes the prose exemption to toon and pretty and states that under JSON these five are documents §11's must-parse coverage reaches.

**Outcome**: Every command the tool registers has its JSON document parsed end to end, and the coverage guard's claim of completeness is true in both machine formats rather than one.

**Do**: (line numbers are the current tree; the findings' references sit 11 lines higher, predating the comment-correction commit)
- Add a per-driver exemption to `conformanceDoc` (`internal/cli/conformance_test.go:23-33`): a `ToonNotADocument` field beside `NotADocument`, carrying the reason the toon driver skips an entry the JSON driver runs. Add one helper stating what a named driver skips — `NotADocument` for every driver, `ToonNotADocument` for the toon driver alone — and give `driveConformanceEntry`/`driveConformanceInventory` (`:601-618`) the driver name, so `TestToonOutputConformance` (`:620`) drives toon and `TestJSONOutputConformance` (`:1555`) drives json off the one inventory.
- Add five entries to `conformanceDocs`, each carrying a `ToonNotADocument` reason and a `Setup` whose args begin with its `Command` (task 1's rule): `dep add` over `conformanceUnconnectedTasks()`; `dep remove` over `conformanceBlockedPair()` releasing `tick-eee555` from `tick-ddd444`; `remove tick-bbb222 --force` over `conformanceListTasks`, whose `deps_updated` comes back empty; `init` in a bare `t.TempDir()` carrying no `.tick`; `rebuild` over `conformanceListTasks`.
- Add `"removed"` and `"deps_updated"` to `jsonConformanceListKeys` (`:626-631`).
- Make the coverage guard per driver: derive `conformanceMustParseCommands` (`:1475-1488`) from the entries a named driver runs, keep the five in a prose set only the toon driver declares, and call `conformanceCoverageProblems` (`:1490-1513`) once per driver in `TestConformanceInventoryCoversEveryCommand` (`:1515`) — under toon the prose set claims the five, under json the must-parse set does, each registered command claimed exactly once in both runs. Correct the comment above `conformanceProseCommands` (`:1467-1468`) in the same edit so it no longer reads format-independent; its wording is yours.
- Update "it drives both formats from one inventory" (`:1656-1671`), which asserts the two drivers run the identical entry list: it now asserts the toon driver skips the five and the JSON driver runs them.

**Acceptance Criteria**:
- [ ] `TestJSONOutputConformance` runs `dep add`, `dep remove`, `remove`, `init` and `rebuild` end to end, and each command's stdout decodes as exactly one JSON value that `jsonShapeProblem` accepts as an object
- [ ] The removal entry's document decodes with `removed` a one-element list and `deps_updated` an empty list, and `jsonInvariantProblems` reports a problem when either key carries null
- [ ] `TestToonOutputConformance` skips the five with their stated reason, and every entry it ran before still runs
- [ ] `conformanceCoverageProblems` reports nothing under either driver, with every key of `commandFlags` claimed exactly once in each run
- [ ] `go test ./internal/cli` is green and `go vet ./...` is clean

**Tests**:
- `"dep add joining two unconnected tasks"`, `"dep remove releasing a blocked task"`, `"remove on a task nothing depends on"`, `"init in an uninitialised directory"`, `"rebuild on a populated project"` — the five entry names, run by the JSON driver and skipped by the toon driver
- `"it rejects a null list value"` (existing, now covering `removed` and `deps_updated`)
- `"it covers every registered command under toon"`
- `"it covers every registered command under json"`
- `"it skips a toon-exempt entry in the toon driver and runs it under json"`
