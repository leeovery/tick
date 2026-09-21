TASK: free-text-round-trip-6-3 (tick-6e30b5) — Mutation And Status Documents Complete The Toon Inventory

ACCEPTANCE CRITERIA:
- Each of `start`, `done`, `cancel` and `reopen` has a decoded document for its no-cascade branch and one for its cascading branch
- `tick create --toon` with no parent decodes with `changed` present and empty
- `tick create --parent <done task> --toon` decodes with a row per task the reopen moved
- `tick update --toon` decodes on all four branches, and the shared-ancestor branch carries each task once
- `note add` and `note remove` documents decode and carry no `changed` key
- The `create --quiet` entry is present in the inventory with a stated reason and is not decoded
- `TestConformanceInventoryCoversEveryCommand` places every key of `commandFlags` — `ready` and `blocked` included — in exactly one of the three sets
- Adding a key to `commandFlags` without declaring it fails that test
- Declaring one command in two sets fails that test
- Naming a command in a declared list that is not a `commandFlags` key fails that test
- `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

STATUS: complete

SPEC CONTEXT: §3.1 names fourteen must-parse commands (show/create/update/note add/note remove; list/ready/blocked; stats; dep tree; start/done/cancel/reopen). §3.2 names five prose commands, §3.3 names `doctor` and `migrate` as out of scope. §7.3 keeps the per-command split — the four status commands return only the `changed` table, `create`/`update` carry it inside the full record, and `show`/`note add`/`note remove` carry no `changed` section at all. §7.4 requires the section to be present even when empty (`create` with no parent). §7.5 gives `update`'s two independent structural cascades (Rule 6 on the new parent, Rule 3 on the old). §11 states the guard's purpose: nothing in a set of shapes stops the next change re-introducing an unreadable section.

IMPLEMENTATION:
- Status: Implemented (test-only task; commit 81a44ff7 touches `internal/cli/conformance_test.go` alone, +477 lines). Later phases reworked the surrounding driver (per-driver prose/skip handling) without disturbing this task's substance.
- Location:
  - Status entries — `internal/cli/conformance_test.go:400-459` (four no-cascade + four cascading), fixtures `conformanceLoneTask` (`:142`) and `conformanceStatusFamily` (`:146`).
  - `create` entries — `internal/cli/conformance_test.go:464-483` (no parent, under a done parent via `conformanceDoneLineage` at `:153`, and the `--quiet` `NotADocument` entry at `:480` reading "--quiet prints the bare ID of the created task rather than a document").
  - `note add`/`note remove` entries — `internal/cli/conformance_test.go:485-499` on `conformanceNotedTask` (`:160`).
  - Four `update` entries — `internal/cli/conformance_test.go:501-533`, fixtures `conformanceMovingTask` (`:180`), `conformanceCompletingParent` (`:186`), `conformanceSharedAncestor` (`:198`).
  - Coverage guard — `conformanceProseCommands` (`:1582`), `conformanceOutOfScopeCommands` (`:1595`), `conformanceMustParseCommands` (`:1599`, derived from the inventory rather than hand-copied), `conformanceCoverageProblems` (`:1614`), `TestConformanceInventoryCoversEveryCommand` (`:1652`).
- Notes: the partition holds against the live registry. `commandFlags` (`internal/cli/flags.go:34-91`) declares 19 keys and `init()` (`internal/cli/flags.go:98-101`) adds `ready` and `blocked`, 21 in total; the toon driver's must-parse set derived from the inventory is exactly §3.1's 14 commands, prose declares 5 and out-of-scope 2. The guard reads `commandFlags` itself (`assertConformanceCoverage`, `:1639-1650`), so the `init()`-registered keys are covered without a hand-copied list, as the task's edge case required.
  Cascade expectations in the fixtures match the domain rules: `start` on a child under an open parent (Rule 2), `done`/`cancel` on a parent with an open child (Rule 4 — `done` from `open` is a legal source per `internal/task/state_machine.go:22`), `reopen` under a done parent (Rule 5), `create --parent <done>` reopening parent then grandparent (Rule 6 then Rule 5, via `validateAndReopenParent` at `internal/cli/helpers.go:118`), and `update`'s Rule 6 / Rule 3 / both-on-a-shared-ancestor split (`internal/cli/update.go:364-370`, `:397-398`). `note add`/`note remove` pass a nil change set (`internal/cli/note.go:87`, `:145`), which is what leaves the document without a `changed` key; `create` passes a non-nil empty `&StatusChanges{}` (`internal/cli/create.go:277-278`), which is what keeps the empty section present per §7.4.

TESTS:
- Status: Adequate
- Coverage: Twenty subtests, one per branch the task enumerated. `TestToonStatusCommandConformance` (`:1441-1508`) pins the four no-cascade documents (single row, `auto` false) and the four cascading ones (requested row plus the cascaded row, `auto` true). `TestToonMutationConformance` (`:1511-1578`) pins `create` with no parent (`changed` present and empty, `:1512-1517`), `create` under a done parent (both reopened ancestors), the four `update` branches, and the two note documents (row count moved by one, `changed` absent). `TestConformanceInventoryCoversEveryCommand` (`:1652-1694`) runs the live partition under both drivers and proves each of the three failure modes against a fixture registry.
- Notes: the "each task once" requirement is genuinely enforced rather than assumed — `conformanceChanges` (`:1402-1421`) keys rows by id and calls `t.Errorf` on a repeat before `assertConformanceChanges` (`:1423`) compares counts, so a duplicated ancestor row fails even though the map would collapse it. `assertToonRowsEmpty` reaches `toonRows` (`internal/cli/toon_decode_test.go:37-46`), which fatals on a missing key, so the empty-section assertions also pin presence. The `create --quiet` entry carries no `Setup`, so `driveConformanceEntry` (`:665-672`) skips it in both drivers, and `TestConformanceInventoryWellFormed` (`:826`) rejects an entry carrying neither a setup nor a reason — the exemption cannot rot into an untested blank.
  Overlap with `TestDetailCommandsCarryNoChangedSection` (`internal/cli/detail_changes_test.go:195-223`) on the note documents is deliberate — the plan required the assertion inside the inventory entries' own subtests, and the conformance pair additionally pins the notes row count — so it is not redundant coverage worth removing.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests named "it does X", `t.Helper()` on every helper, `t.TempDir()` isolation via `setupTickProjectWithTasks`.
- SOLID principles: Good — the fixtures are small named builders composed by the entries (`conformanceSharedAncestor` reuses `conformanceCompletingParent`), and the guard's pure `conformanceCoverageProblems` is separated from its assertion wrapper, which is what lets the three failure modes be tested without mutating the live registry.
- Complexity: Low.
- Modern idioms: Yes — `slices.Sorted(maps.Keys(...))` for deterministic problem ordering.
- Readability: Good — fixture comments state the rule each shape exists to fire.
- Issues: None. Comments on the helpers added here hold against the code they describe.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean" — settled only by running the four commands; reading confirms the expectations are consistent with the cascade rules and handler wiring, but not that the suite, vet, gofmt and the linter pass.
