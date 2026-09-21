TASK: free-text-round-trip-2-7 (tick-30886c) — Derive Each View Of A Command's Status Movements Instead Of Storing Both

ACCEPTANCE CRITERIA:
- `StatusChanges` has exactly one field, `Blocks`, and a `Rows()` method; `grep -rn 'StatusChanges{Rows:' internal/cli` returns nothing.
- `CascadeResult` has no `Changed` field; `Changed()` derives its rows through `statusChangeSet`, and `PrimaryAuto` is the only field added.
- `buildCascadeResult` walks `cascades` once — the loop that was at `transition.go:103-120` is gone and nothing else populates a flat row set by hand.
- `CascadeResult.TaskTitle` has a production reader for the first time: the title in the toon and JSON `changed` tables comes from it.
- A zero-value `CascadeResult{}` derives an empty, non-nil row set, so toon still emits `changed[0]{id,title,from,to,auto}:` and JSON still emits `[]` rather than `null`.
- Pretty is byte-identical: its input is still the cascade-shaped value, `PrettyFormatter.FormatCascadeTransition` and `FormatTaskDetail` are untouched, and the README samples still match the built binary.
- `go build ./...`, `go vet ./...`, `go test ./...` clean and `gofmt -l internal cmd` empty.

STATUS: complete

SPEC CONTEXT: §7.2 requires one `changed` table per command, each moved task appearing at most once, reading from the status it held before the command to the status it holds after, with an `auto` column marking knock-ons; §7.4 requires the section to be present even when empty (`changed[0]{…}:`), and §4.1 requires pretty's bytes to be unchanged (the box-drawing cascade tree stays, the flat table is the agent surface only). The task is a structural repair, not a behaviour change: the two views of one command's status movements were held as two independently-populated fields, so a future producer could fill one and leave the other stale — the terminal and the machine output then disagreeing about what moved.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/format.go:144-151` — `StatusChanges` now holds only `Blocks []CascadeResult`, with `Rows()` delegating to `mergeStatusChanges(c.Blocks...)`; doc comment rewritten to name `Blocks` as stored and `Rows` as its flattening.
  - `internal/cli/format.go:199-228` — `CascadeResult` drops `Changed []StatusChange`, gains `PrimaryAuto bool` (the only field added), and `Changed()` feeds a `statusChangeSet` the primary row (`TaskID`, `TaskTitle`, `OldStatus`, `NewStatus`, `Auto: c.PrimaryAuto`) then one row per `Cascaded` entry with `Auto: true`, returning `set.rows()`.
  - `internal/cli/transition.go:64-107` — `buildCascadeResult` sets `PrimaryAuto` in the literal (`:70`, now `result.Auto` after task 2-8 rewired the source) and the hand-built flat-row loop that was at `:103-120` is gone; the only remaining pass over `cascades` that builds output is the `Cascaded` loop at `:91-104` (the `:84-89` pass is the pre-existing upward-cascade detection, not a row set).
  - `internal/cli/transition.go:152-160` — `mergeStatusChanges` ranges `b.Changed()`; its doc comment (`:149-151`) no longer describes a stored `Changed`.
  - Producers: `internal/cli/create.go:277` and `internal/cli/update.go:397` are both `changes := &StatusChanges{Blocks: blocks}`.
  - Readers moved to the methods: `internal/cli/toon_formatter.go:99` and `internal/cli/json_formatter.go:136` call `detail.Changes.Rows()`; `internal/cli/toon_formatter.go:160` and `internal/cli/json_formatter.go:318` call `result.Changed()`.
- Notes:
  - Criterion 1 holds: `grep -rn 'StatusChanges{Rows:' internal/` returns nothing, and `StatusChanges` has one field.
  - Criterion 2 holds: the struct's field set moved from {TaskID, TaskTitle, OldStatus, NewStatus, Changed, Cascaded} to {TaskID, TaskTitle, OldStatus, NewStatus, PrimaryAuto, Cascaded}; `Changed()` builds through `statusChangeSet` (`transition.go:109-147`), so the at-most-one-row-per-task and drop-a-no-op rules of §7.2 come from the same accumulator as before.
  - Criterion 4 holds: `format.go:213` (`Title: c.TaskTitle`) is the production read; it reaches the toon table via `buildChangedSection` (`toon_formatter.go:152-162`) and the JSON list via `toJSONStatusChanges` (`json_formatter.go:303-310`).
  - Criterion 5 holds by reading: on a zero value `Changed()` adds one row with `From == To`, which `rows()` (`transition.go:139-147`) drops while still returning `make([]StatusChange, 0, …)` — non-nil, so toon takes the `emptyToonSection` branch and JSON marshals `[]`.
  - Criterion 6 holds: commit `8e579a68` does not touch `internal/cli/pretty_formatter.go`, and pretty's detail path still reads the stored cascade shape (`pretty_formatter.go:231-233` ranges `detail.Changes.Blocks`).
  - Behaviour equivalence checked line by line: the derived rows are built in the same order (primary, then `Cascaded` in append order = `cascades` order) from the same values the deleted loop read, so no output moved. Deferring the derivation from inside the `Mutate` closure to format time is safe — `CascadeResult` and `CascadeEntry` hold only strings and bools, no pointer into the `tasks` slice — which is exactly what the surviving doc comment at `transition.go:149-151` asserts.
  - Divergence from the plan's wording, sound and not a loss: the plan's step 2 said `buildCascadeResult` sets `PrimaryAuto: primaryAuto` from its parameter; the parameter is gone and `:70` reads `result.Auto`. That is task 2-8's change landing on top of this one, not drift in this task — commit `8e579a68` did implement it exactly as written.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/status_change_test.go:11-28` — `statusChangeBlock` now builds the cascade-shaped value and no longer calls `set.rows()` itself; the 15 `.Changed` field reads became `Changed()` calls with assertions verbatim (`:52-56`, `:64-68`, `:75`, `:88-89`, `:166-170`, `:179-180`).
  - `internal/cli/cascade_formatter_test.go:14-65`, `:183-207`, `:269-274` — the six `CascadeResult{Changed: …}` literals became primary-plus-`Cascaded` values whose derived rows are the previous literals (auto-false row as primary, auto-true rows as cascades); assertions unchanged.
  - `internal/cli/helpers_test.go:296-301` and `internal/cli/detail_changes_test.go:50-55`, `:238-240` converted the same way; `detail_changes_test.go:46-49` keeps the `rows` fixture, still read by the JSON assertion at `:134-137`, so the conversion left no unused binding.
  - Guards that pin the criteria without modification: `format_test.go:403-418` pins the zero-value `CascadeResult{}` shapes (toon `changed[0]{id,title,from,to,auto}:` via `assertCountZeroSection`, JSON an empty `changed` list, pretty and the stub empty); `detail_changes_test.go:74-80`, `:101-120` pin the empty section on the detail document; `detail_changes_test.go:172-192` pins pretty appending each block exactly as `FormatCascadeTransition` renders it; `transition_test.go`, `create_test.go`, `update_test.go`, `toon_decode_test.go` and `readme_samples_test.go` were untouched by the commit.
- Notes:
  - The deliberate deletion of `"it appends nothing in pretty when the detail carries no blocks"` is sound: it constructed rows without blocks, a state the collapse makes unrepresentable, and pretty's empty case is still reachable structurally (`pretty_formatter.go:231-233` ranges an empty slice). No coverage of a representable state was lost.
  - Not over-tested: no test names were added, matching the plan's "the existing suite is the guard" for a pure refactor.
  - The suite would still catch a regression here — `status_change_test.go:48-91` asserts the derived rows' count, order and `auto` values straight off `buildCascadeResult`, so a wrong `PrimaryAuto` wiring or a lost cascade row fails.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` with `t.Run` subtests and "it does X" naming, value receivers on read-only methods, doc comments on every exported symbol, error wrapping untouched.
- SOLID principles: Good — the change removes a representable-illegal state by making the second view a pure function of the first, which is the task's whole point; producers can no longer desynchronise the two.
- Complexity: Low — `Changed()` is one straight-line accumulation; `buildCascadeResult` lost 20 lines and one loop.
- Modern idioms: Yes — no opportunities missed in the changed lines.
- Readability: Good — the doc comments at `format.go:142-143`, `:197-198` and `transition.go:149-151` describe the code as it now stands; none of them claims a stored `Changed` or a stored `Rows`.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go build ./...`, `go vet ./...`, `go test ./...` clean and `gofmt -l internal cmd` empty." — needs the toolchain run over the working tree; reading confirms the type/method set is internally consistent (no surviving `Changed` field read, no surviving `StatusChanges{Rows:` literal, no unused binding left by the test conversions) but cannot settle compilation, vet or formatting.
- "Pretty is byte-identical: … the README samples still match the built binary." — the code half is settled by reading (pretty untouched, its input still the cascade-shaped value); the "match the built binary" half needs `go test ./internal/cli -run TestREADMESamples` against a build.
