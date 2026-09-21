TASK: free-text-round-trip-6-8 (tick-1f8606) — The Prose Exemption Is Declared Per Format

ACCEPTANCE CRITERIA:
- [ ] `TestJSONOutputConformance` runs `dep add`, `dep remove`, `remove`, `init` and `rebuild` end to end, and each command's stdout decodes as exactly one JSON value that `jsonShapeProblem` accepts as an object
- [ ] The removal entry's document decodes with `removed` a one-element list and `deps_updated` an empty list, and `jsonInvariantProblems` reports a problem when either key carries null
- [ ] `TestToonOutputConformance` skips the five with their stated reason, and every entry it ran before still runs
- [ ] `conformanceCoverageProblems` reports nothing under either driver, with every key of `commandFlags` claimed exactly once in each run
- [ ] `go test ./internal/cli` is green and `go vet ./...` is clean

STATUS: complete

SPEC CONTEXT:
§3.2 ("Output that stays prose", specification.md:74-80) now carries the scoping clause added by the corrigendum of 2026-09-20 (specification.md:551): the prose exemption holds for toon and pretty only; under `--json` all five commands return objects — `FormatDepChange` `{"action","task_id","blocker"}`, `FormatMessage` `{"message"}`, `FormatRemoval` `{"removed":[…],"deps_updated":[…]}` — and §11's must-parse coverage reaches them there. §11 (specification.md:480-492) requires every structured command's output to be decoded by a real reader in the suite, counted in documents rather than commands, with the coverage guard proving the inventory complete.

IMPLEMENTATION:
- Status: Implemented (with one recorded, spec-sanctioned divergence from the plan's Do list — see Notes)
- Location:
  - `internal/cli/conformance_test.go:32` — `ToonNotADocument` field added beside `NotADocument`; the struct's doc comment (`:22-26`) now distinguishes the all-driver exemption from the toon-only one.
  - `internal/cli/conformance_test.go:35-38` — `conformanceToonDriver` / `conformanceJSONDriver` constants.
  - `internal/cli/conformance_test.go:654-662` — `conformanceSkipReason(driver, entry)`: `NotADocument` for every driver, `ToonNotADocument` for toon alone.
  - `internal/cli/conformance_test.go:664-681` — `driveConformanceEntry` / `driveConformanceInventory` take the driver name; `TestToonOutputConformance` (`:694`) drives toon, `TestJSONOutputConformance` (`:1692-1696`) drives json off the one inventory.
  - `internal/cli/conformance_test.go:534-577` — the five new entries, each with a `ToonNotADocument` reason and a `Setup` whose args begin with its `Command`: `dep add` over `conformanceUnconnectedTasks()` (f22222 blocked by e11111, no cycle), `dep remove` releasing `tick-eee555` from `tick-ddd444` over `conformanceBlockedPair()`, `remove tick-bbb222 --force` over `conformanceListTasks`, `init` in a bare `t.TempDir()`, `rebuild` over `conformanceListTasks`.
  - `internal/cli/conformance_test.go:705` — `"removed", "deps_updated"` added to `jsonConformanceListKeys`.
  - `internal/cli/conformance_test.go:1579-1590` — corrected comment above `conformanceProseCommands` plus `conformanceDriverProseCommands`, which returns the prose set for toon and nil for json.
  - `internal/cli/conformance_test.go:1597-1610` — `conformanceMustParseCommands(driver)` now skips the entries the driver does not run.
  - `internal/cli/conformance_test.go:1636-1657` — `assertConformanceCoverage(t, driver)` and the two per-driver subtests.
  - `internal/cli/json_formatter.go:181-195` — `jsonDepChange`'s scalar key renamed `blocked_by` → `blocker`; `internal/cli/json_formatter_test.go:465-467` follows.
- Notes:
  - End-to-end soundness of the five under `--json`, verified by reading the handlers: each writes exactly one formatted value to stdout and nothing else — `RunDepAdd` (`internal/cli/dep.go:125`), `RunDepRemove` (`:195`), `RunRemove` (`internal/cli/remove.go:194`), `RunInit` (`internal/cli/init.go:36`), `RunRebuild` (`internal/cli/rebuild.go:26`). `remove`'s confirmation prompts all write to stderr and are bypassed by `--force` (`internal/cli/app.go:289`), so the `IsTTY: true` runner cannot introduce a second value on stdout. `jsonShapeProblem` expects an object for all five, since `jsonTopLevelIsArray` (`internal/cli/conformance_test.go:728-733`) is true only for `list`/`ready`/`blocked`.
  - Coverage arithmetic, enumerated against `commandFlags` (`internal/cli/flags.go:35-95` plus `ready` and `blocked` registered in `init()` at `:104-107`) — 21 registered commands. Under json: 19 must-parse commands from the inventory + `doctor`/`migrate` out-of-scope + empty prose set = 21, each once. Under toon: the same 19 minus the five toon-skipped = 14 must-parse + the 5 prose + 2 out-of-scope = 21, each once. Every command carrying a `NotADocument` entry (`list`, `dep tree`, `show`, `create`) also carries at least one entry that runs, so none falls out of the must-parse set.
  - Divergence from the plan's Do list, and it is sound: the commit also renamed the dependency-change JSON key `blocked_by` → `blocker`. Forced by the work — `blocked_by` is a `jsonConformanceListKeys` member (a list of nodes in the detail and dep-tree documents), so the scalar in the dep-change document would have tripped `jsonInvariantProblems` the moment the json driver ran `dep add`. This is a change to shipped output, and it is recorded: specification.md:553 carries a dated corrigendum giving the reasoning, the alternatives rejected (exempting the document, or leaving the two commands outside the driver), and the blast radius. I confirmed the corrigendum's containment claims hold: toon and pretty go through `baseFormatter.FormatDepChange`, the only test naming the old key was updated in the same commit, and README documents no JSON dep-change sample (`README.md:299-311` shows usage only; no `action`/`task_id` key appears anywhere in README).
  - The inventory well-formedness guard still holds for the new shape: an entry carrying `ToonNotADocument` but no `Setup` is caught by `(entry.NotADocument != "") != (entry.Setup == nil)` (`internal/cli/conformance_test.go:849`) as "carries neither a setup nor a reason", so the json driver cannot reach a nil `Setup`.

TESTS:
- Status: Adequate
- Coverage:
  - The five entry names the plan calls for exist and are driven by `TestJSONOutputConformance` and skipped by `TestToonOutputConformance` through the shared `conformanceSkipReason`.
  - `"it rejects a null list value"` (`internal/cli/conformance_test.go:1768-1776`) iterates `jsonConformanceListKeys`, so it now covers `removed` and `deps_updated` — the second half of AC2 — without a separate hand-written case.
  - `TestJSONRemovalConformance` (`:1843-1866`) runs the removal entry end to end and asserts `removed` is one row whose `id` is `tick-bbb222` and `deps_updated` is an empty list — the first half of AC2, and distinct from the formatter-level assertions in `json_formatter_test.go`.
  - `"it covers every registered command under toon"` / `"…under json"` (`:1653`, `:1657`) run the guard per driver; the three negative cases for `conformanceCoverageProblems` are retained.
  - `"it skips a toon-exempt entry in the toon driver and runs it under json"` (`:1793-1819`) replaces the old "drives both formats from one inventory", partitions the inventory into all-driver-exempt / toon-exempt / shared, fails fast if no entry carries a toon exemption, and asserts the toon driver runs exactly the shared set while the json driver runs the shared set plus the toon-exempt ones — which is what establishes AC3's "every entry it ran before still runs".
  - Would fail if the feature broke: dropping a `ToonNotADocument` reason makes the toon driver run a prose command and its decode fails; dropping an entry makes the coverage guard report the command declared nowhere; regressing `FormatRemoval` to `null` arrays makes `jsonInvariantProblems` fire on a real command's document rather than only a fixture.
- Notes: No redundancy found. `TestJSONRemovalConformance` overlaps `json_formatter_test.go:531` only in subject, not in level — the new test drives the handler, which is the gap the task exists to close.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests in "it does X" form, `t.Helper()` on helpers, `t.TempDir()` for isolation, table-free composition consistent with the rest of the file.
- SOLID principles: Good. The skip decision is centralised in one predicate (`conformanceSkipReason`) that both drivers and the coverage guard consume, so the driven set and the claimed set cannot drift apart.
- Complexity: Low. `conformanceSkipReason` is two branches; `conformanceMustParseCommands` gained one conjunct.
- Modern idioms: Yes — `slices.Concat`, `slices.Equal`, `slices.Sorted(maps.Keys(...))`, all consistent with existing usage and with go 1.26 in `go.mod`.
- Readability: Good. `drivenConformanceEntries` deliberately re-uses the real driver to collect names rather than re-deriving the skip rule, so the assertion tests the mechanism under test.
- Comment accuracy: checked each changed comment against the code — the struct comment (`:22-26`), the driver constants (`:34`), `conformanceSkipReason` (`:651-653`), `conformanceProseCommands` (`:1579-1581`), `conformanceDriverProseCommands` (`:1584-1585`), `conformanceMustParseCommands` (`:1597-1598`) and `assertConformanceCoverage` (`:1633-1634`) all hold. The comment the task named as false — the format-independent prose claim — is corrected.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./internal/cli` is green and `go vet ./...` is clean" — settle by running `go test ./internal/cli` and `go vet ./...`. Reading establishes that each of the five handlers emits exactly one JSON value and that the coverage sets add up to 21 registered commands claimed once per driver, but a green suite and a clean vet are only observable by executing them.
