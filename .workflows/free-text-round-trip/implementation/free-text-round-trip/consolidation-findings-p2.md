# Consolidation Findings: free-text-round-trip (Phase 2)

## Findings

### F1: One command's status movements are held in three hand-synchronised representations

- **Class**: duplication
- **Failure**: A user running `tick done tick-a1b2` in a terminal and an agent running the same command with `--toon`/`--json` are told different things about what moved, and neither can tell which is right. The phase left three independent statements of one fact — `CascadeResult.Cascaded` (pretty's tree), `CascadeResult.Changed` (the toon/JSON rows), and `StatusChanges.Rows` beside `StatusChanges.Blocks` — with nothing tying any of them to the others. A future command, or an edit to an existing one, populates one and not the other: it compiles, `go test ./...` stays green, and the divergence surfaces only when a human compares pretty output against an agent's transcript of the same invocation. The state is not hypothetical — `internal/cli/detail_changes_test.go:171-181` constructs `StatusChanges{Rows: …}` with no `Blocks` and asserts pretty prints nothing, pinning "the table says a task moved, the tree says nothing moved" as acceptable. It is the exact failure §7.2 exists to prevent, moved from the output format into the data that feeds it.
- **Evidence**:
  - `internal/cli/format.go:109-115` — `StatusChanges{Rows, Blocks}`, two independent fields; the doc at `:109-111` states outright that they are "both shapes the formatters need".
  - `internal/cli/format.go:161-169` — `CascadeResult` holds `Cascaded []CascadeEntry` (`:167`) and `Changed []StatusChange` (`:168`) side by side.
  - `internal/cli/transition.go:88-101` and `internal/cli/transition.go:103-120` — `buildCascadeResult` walks the same `cascades` slice twice, once to fill `Cascaded` and once to fill `Changed`. It is today the only production constructor, which is the only reason the two agree.
  - `internal/cli/create.go:271` and `internal/cli/update.go:395` — the identical line `changes := &StatusChanges{Rows: mergeStatusChanges(blocks...), Blocks: blocks}`, i.e. `Rows` is nothing but a function of `Blocks`, computed by hand at two sites.
  - `internal/cli/transition.go:165-176` — `mergeStatusChanges(blocks ...CascadeResult) []StatusChange` already *is* that derivation, sitting one call away from being the definition.
  - Consumers, each reading a different representation of the same event: `internal/cli/toon_formatter.go:100` and `:163-164` (`Rows` / `Changed`), `internal/cli/json_formatter.go:125` and `:278-279` (`Rows` / `Changed`), `internal/cli/pretty_formatter.go:183-187` and `:240-284` (`Blocks`, then `TaskID`/`OldStatus`/`NewStatus`/`Cascaded`).
  - `internal/cli/format.go:164` — `CascadeResult.TaskTitle` is written at `internal/cli/transition.go:65` and read by no production code; the title a reader sees now comes from `StatusChange.Title`. Superseded by this phase's rows, and it falls out with the rest of the collapse.
  - `internal/cli/detail_changes_test.go:54,92,125,174,249` — every test constructs `Rows` or `Blocks`, never both, which is why nothing in the suite would catch the two disagreeing.
- **Proposed shape**: Make each derived view a method, so the divergent state cannot be constructed. Two steps, either landable alone; behaviour-preserving, with pretty's bytes untouched at every point.
  1. `StatusChanges` holds `Blocks []CascadeResult` only, with `func (c StatusChanges) Rows() []StatusChange { return mergeStatusChanges(c.Blocks...) }`. `create.go:271` and `update.go:395` become `&StatusChanges{Blocks: blocks}`; `toon_formatter.go:100` and `json_formatter.go:125` call `detail.Changes.Rows()`.
  2. `CascadeResult` stores the cascade shape plus `PrimaryAuto bool` and derives the flat rows: `func (c CascadeResult) Changed() []StatusChange` builds the primary row from `TaskID`/`TaskTitle`/`OldStatus`/`NewStatus`/`PrimaryAuto` and one row per `Cascaded` entry with `Auto: true`, through the existing `statusChangeSet`. The `Changed` field and the second loop at `transition.go:103-120` go; `TaskTitle` gains its first production reader.
  - **Derive the table from the tree, not the reverse.** The two banked entries proposed deriving pretty's tree from `Changed`; that is the wrong direction. §4.1 requires pretty's bytes to be identical after this work, and the flat rows do not carry `ParentID`, so the tree cannot be rebuilt from them without re-deriving the parent links `buildCascadeResult` already computed. Storing the cascade-shaped value and flattening it leaves pretty's input byte-for-byte what it is today.
  - Consequence to accept deliberately: `detail_changes_test.go:171-181` pins a state that becomes unrepresentable and is deleted rather than rewritten; the four remaining `StatusChanges{Rows: …}` constructions become `StatusChanges{Blocks: []CascadeResult{…}}`. No production path changes what it prints.
  - Optional, and only if step 2 lands: `Formatter.FormatCascadeTransition(CascadeResult)` can take `StatusChanges` instead, making it the single entry point for both call paths and letting `outputStatusChanges` (`internal/cli/helpers.go:99-103`, now a one-line wrapper with one caller) go. This supersedes the interface-doc correction below.
- **Bank**: Confirms three entries. 2-1/reviewer ("CascadeResult will carry two parallel representations … with nothing forcing a producer to populate both") and 2-3/executor ("Pretty status output is still built from `CascadeResult.Cascaded` while toon and JSON are built from `CascadeResult.Changed`") are step 2, with the derivation direction corrected. 2-4/reviewer ("`StatusChanges` carries `Rows` and `Blocks` as independent fields where `Rows` is fully derivable from `Blocks`") is step 1, and its prediction landed exactly: Task 2-5 built both by hand at two sites.

### F2: The `auto` column is computed a second time at the call site instead of read from the transition the domain applied

- **Class**: duplication
- **Failure**: The `changed` table can contradict the task's own transition history about the same event, and an agent has no way to know which to believe. §7.2 justifies the `auto` column precisely by it *not* being new vocabulary — "Every task's transition history already records each change with an `auto` flag … stored per task in the JSONL and in the `task_transitions` table". The implementation does not read that flag. It re-states it as a literal `true`/`false` argument at each of the four `buildCascadeResult` call sites, hand-matched to whichever of `ApplyUserTransition`/`ApplySystemTransition` the caller chose a few lines above. Nothing ties the two: a new mutating command that calls `ApplySystemTransition` and copies `transition.go:40`'s `false`, or an existing site whose `Apply*` call changes, persists `auto: true` on the task and prints `auto,false` for it. It compiles, the new command's own tests assert the output they were shown, and it is caught only by someone who runs `tick show` after a status command and reads the transition history beside the `changed` table.
- **Evidence**:
  - `internal/cli/transition.go:62` — `primaryAuto bool` as the sixth parameter, and `:103-110` where it is stamped onto the primary row.
  - `internal/cli/transition.go:40` — `sm.ApplyUserTransition(...)` four lines above `buildCascadeResult(..., false)`.
  - `internal/cli/create.go:238` — `buildCascadeResult(..., true)`, its `auto`-ness decided inside `validateAndReopenParent` at `internal/cli/helpers.go:119` (`sm.ApplySystemTransition`).
  - `internal/cli/update.go:307` — same pairing through `helpers.go:119`; `internal/cli/update.go:366` — `buildCascadeResult(..., true)` for a result produced by `sm.ApplySystemTransition` at `internal/cli/update.go:148`, three call frames away.
  - `internal/task/transition.go:7-10` — `TransitionResult` carries `OldStatus`/`NewStatus` and nothing about who asked, so the fact the CLI needs has to be re-supplied from outside. The domain does hold it: `ApplyUserTransition`/`ApplySystemTransition` set exactly this flag on the `TransitionRecord` they append.
- **Proposed shape**: Carry the flag out of the domain rather than re-asserting it in the CLI. Add `Auto bool` to `task.TransitionResult`, set by `applyWithCascades` from the same value it already writes onto the primary target's `TransitionRecord`, and drop `buildCascadeResult`'s `primaryAuto` parameter in favour of `result.Auto` (or of `PrimaryAuto` fed from it, if F1 step 2 lands first). All four call sites lose their literal; the rendered `auto` and the persisted `auto` become one value by construction, which is what §7.2 claims they are. The four `buildCascadeResult` calls in tests follow mechanically. No output changes: every pairing is correct today.
- **Sequencing**: If both land, F1 first — F2's remedy feeds the field F1 step 2 introduces. Either is complete on its own.

## Comment Corrections

- `internal/cli/format.go:219` — false for two of the three implementations: toon (`toon_formatter.go:163-164`) and JSON (`json_formatter.go:278-279`) render a flat `changed` list with no primary/knock-on distinction; only pretty still renders a primary transition with cascaded children. A maintainer writing a fourth implementation from this line builds the shape §7.2 deleted. (Superseded if F1's optional signature change lands.)
  OLD: 	// FormatCascadeTransition renders a status transition with cascaded child changes.
  NEW: 	// FormatCascadeTransition renders the status changes a command made.

- `internal/cli/format.go:205` — workflow vocabulary in a comment that must outlive the process that produced it, and a provenance claim the code does not need; the three implementations are named and compile-checked at `toon_formatter.go:19`, `json_formatter.go:14` and in `NewFormatter` at `format.go:290-299`. Pre-existing, in a block this phase rewrote.
  OLD: // Concrete implementations (Toon, Pretty, JSON) are provided by tasks 4-2 through 4-4.
  NEW:

- `internal/cli/format.go:259` — same: the second sentence is workflow vocabulary plus history. The first sentence is kept. Pre-existing, in a block this phase rewrote.
  OLD: // It returns empty strings for all methods. Replaced by concrete formatters in tasks 4-2 through 4-4.
  NEW: // It returns empty strings for all methods.

- `internal/cli/helpers.go:99-100` — the second sentence states a constraint the function it sits on does not impose: `outputStatusChanges` builds nothing and never touches a tasks slice. The constraint belongs to `buildCascadeResult` (`transition.go:62`), which reads `tasks`, and a reader looking for it here finds it attached to the wrong function.
  OLD: // outputStatusChanges writes a command's status changes to stdout. Callers must build
  OLD: // the CascadeResult inside the Mutate closure where the tasks slice is still valid.
  NEW: // outputStatusChanges writes a command's status changes to stdout.

## Spec Defects

### S1: §4.3's prescription for the shared transition line was falsified by what landed

- **Claim**: §4.3, first bullet — "**The single transition line** comes from `baseFormatter.FormatTransition` (`grep -n 'func (b \*baseFormatter) FormatTransition' internal/cli/format.go` → `format.go:211`), embedded by both the toon and pretty formatters, so the two emit byte-identical text today. Restructuring the toon form requires splitting that method." §7.1 leans on the same premise: "It is a bespoke line format, shared with pretty (§4.3)".
- **Observed**: The method was removed outright, not split. `FormatTransition` is gone from `baseFormatter`, from the `Formatter` interface and from `StubFormatter` (commits `7ddbe8d4`, `be7c7371`); grepping `internal/` and `cmd/` for `FormatTransition` outside `FormatCascadeTransition` returns nothing. Pretty's single transition line is byte-identical because `PrettyFormatter.FormatCascadeTransition` already produced it on the zero-cascade branch — `internal/cli/pretty_formatter.go:245` formats `"%s: %s → %s"` and `:247-249` returns before the tree. That branch pre-dates this work, so pretty never depended on `baseFormatter.FormatTransition` for anything it could not already produce; the sharing the spec identified was redundant rather than load-bearing, and deleting it satisfied §4.1 with no split. `go vet ./...` and `go test ./...` are clean, and the pretty samples at `README.md:481-485` and `README.md:526-533` are captured from the built binary by `readme_samples_test.go`.
- **Read**: Spec stale, code right. The claim was a mechanism prescription in a section whose requirement — pretty unchanged — is met. Nothing is owed to the code. The second bullet of §4.3 (dep-tree empty messages) is untouched by this phase and still governs later work, so only the first bullet and §7.1's back-reference to it are affected.
