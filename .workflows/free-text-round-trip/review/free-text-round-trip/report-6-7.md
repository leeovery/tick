TASK: free-text-round-trip-6-7 (tick-8246c5) — The Inventory's Declared Command Must Match The Arguments It Runs

ACCEPTANCE CRITERIA:
- `inventoryProblems` reports one problem for each entry whose setup args do not begin with `strings.Fields(entry.Command)`, naming the entry, the declared command and the args
- An entry whose args are shorter than its declared command's words is reported, not a panic on the slice
- Args beginning with the command's words and continuing with flags or positionals raise nothing
- An entry carrying `NotADocument` and no setup is not checked, and the both/neither negative subtests each still report exactly one problem
- "it accepts the inventory" passes against `conformanceDocs` with no inventory entry edited, and `go test ./internal/cli` is green

STATUS: complete

SPEC CONTEXT: §11 (Conformance Verification) requires every structured command's output to be decoded by a real TOON reader, with coverage counted in documents rather than commands — every branch a listed command can take. The coverage guard `TestConformanceInventoryCoversEveryCommand` (`internal/cli/conformance_test.go:1651`) builds its must-parse set from each entry's free-text `Command` field (`conformanceMustParseCommands`, `:1599-1610`), so §11's coverage claim is only as true as the agreement between an entry's declared command and the args its `Setup` actually runs. This task closes that gap. The spec does not name the guard itself; it is a plan-level analysis task protecting §11's claim.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/conformance_test.go:826-837` — new `declaredCommandProblem(t, entry)` helper: takes `strings.Fields(entry.Command)`, returns `""` when `len(args) >= len(words) && slices.Equal(args[:len(words)], words)`, otherwise `fmt.Sprintf("entry %q declares command %q but its setup runs %v", entry.Name, entry.Command, args)`.
  - `internal/cli/conformance_test.go:849-862` — `inventoryProblems` closure: the setup/reason exclusivity branch was inverted so a well-formed entry falls through to the new check instead of `continue`-ing; `NotADocument`-only entries still `continue` at `:857`.
  - `internal/cli/conformance_test.go:874` — duplicate-name fixture setup now returns `[]string{"list"}` rather than `nil`, as the plan directed.
  - Commit `8e0c5120`, 71 insertions / 4 deletions, confined to `internal/cli/conformance_test.go`. No inventory entry was edited.
- Notes:
  - Length guard precedes the slice, so the shorter-args case returns a problem rather than panicking (`:833`).
  - The exclusivity refactor is behaviour-preserving: `(NotADocument != "") != (Setup == nil)` with the two messages moved inside reproduces the old `== ... continue` form's four cases exactly. (reason, setup) → "both" + continue; (no reason, no setup) → "neither" + continue; (reason, no setup) → falls through then skipped at `:857`; (no reason, setup) → checked.
  - `ToonNotADocument` entries (`dep add`, `dep remove`, `remove`, `init`, `rebuild`) carry a `Setup` and no `NotADocument`, so they reach the check — correct, since they are real must-parse entries under the json driver.
  - I read all 48 inventory entries (`:205-578`; 42 carry a `Setup`, 6 carry `NotADocument`, 5 carry `ToonNotADocument`). Every setup-carrying entry's args begin with its declared command's words, the multi-word `dep tree` (×6), `dep add`, `dep remove`, `note add` and `note remove` entries included. The rule holds against the current inventory with no entry edited.
  - Prefix rather than equality is the deliberate design (the plan's fourth "Do" bullet and the third acceptance criterion). It composes with the coverage guard: a hypothetical `Command: "dep"` on a `["dep","tree"]` setup would pass the prefix check but be caught by `conformanceCoverageProblems` at `:1620-1634` as a declared command no `commandFlags` key registers.
  - `entry.Setup(t)` is now called a third time per entry (toon driver, json driver, well-formedness). The setups (`setupTickProject`/`setupTickProjectWithTasks`, `internal/cli/create_test.go:19` and `:34`) only write files under `t.TempDir()` — no chdir, no env, no shared state — so the extra call is safe.

TESTS:
- Status: Adequate
- Coverage:
  - `"it rejects an entry whose setup runs a different command"` (`:906`) — `Command: "ready"` against `["list","--ready"]`, the exact silent case the task names; asserts one problem and that the message carries both the declared command and the args.
  - `"it rejects an entry whose setup args are shorter than its declared command"` (`:923`) — `Command: "dep tree"` against `["dep"]`, pinning the no-panic path.
  - `"it accepts an entry whose setup args carry flags after the command"` (`:940`) — `show tick-a00001 --field notes`, pinning the rule as a prefix rather than an equality.
  - `"it accepts the inventory"` (`:867`) now holds all 42 setup-carrying entries against their declared commands.
  - The both/neither subtests (`:885`, `:898`) still assert exactly one problem each, confirming the refactored exclusivity branch did not double-report.
- Notes: Each subtest receives its own `*testing.T` and passes it into `inventoryProblems`/`declaredCommandProblem`, so no failure is attributed to the parent. No redundancy: the three new subtests cover three distinct branches of the one predicate, and none duplicates another's assertion.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run()` "it does X" subtest names, `t.Helper()` on the new helper, `t.TempDir()` isolation.
- SOLID principles: Good — the predicate is extracted to a named helper with one job; `inventoryProblems` stays a per-entry accumulation loop.
- Complexity: Low — one length guard plus one `slices.Equal`.
- Modern idioms: Yes — `slices.Equal` and `strings.Fields` rather than a hand-rolled loop; `slices` and `strings` were already imported (`:10`, `:11`).
- Readability: Good — the helper's doc comment states the rule ("empty when the args begin with the command's words") and matches the code.
- Issues: None. The pre-existing `conformanceDoc` doc comment ("Command is the fully-qualified command name as commandFlags spells it, carrying no arguments and no flags", `:19-25`) is now enforced rather than merely asserted, and remains accurate.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./internal/cli` is green" — requires executing the package suite. Reading settles the other half of this criterion ("it accepts the inventory" passes against `conformanceDocs` with no inventory entry edited: all 42 setup-carrying entries' args begin with their declared command's words, and commit `8e0c5120` touches no inventory entry), but whether the whole `internal/cli` package compiles and passes can only be measured by a run.
