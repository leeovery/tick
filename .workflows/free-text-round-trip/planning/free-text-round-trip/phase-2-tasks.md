# Phase 2: Status changes become one `changed` table — 6 tasks

## free-text-round-trip-2-1

### Task 1: Collapse a command's status movements into one changed set

**Problem**: A command's status movements reach the formatters as a primary transition plus a list of cascade entries — `CascadeResult` (`internal/cli/format.go:139-147`) built by `buildCascadeResult` (`internal/cli/transition.go:63`). Nothing merges them. `update` produces two independent `CascadeResult`s (Rule 6 at `internal/cli/update.go:316`, Rule 3 at `internal/cli/update.go:377`), so one command can report the same task twice when the two cascades meet on a shared ancestor, and can report a task that ends the command exactly where it started. §7.2 requires the opposite: one set in which a task appears at most once, reading from the status it held before the command to the status it holds after, and a task that ends where it started carries no row at all. The structure also has no place to record whether a change was requested or was a consequence: `buildCascadeResult` marks the primary by position, and for `create`/`update` that "primary" is a parent the *system* moved (`ApplySystemTransition`), which §7.2 requires to be marked as a consequence like everything else.

**Solution**: Add a flat `StatusChange` row type carrying `auto`, an unexported accumulator that merges changes keyed by task ID, and a `Changed []StatusChange` field on `CascadeResult` populated by `buildCascadeResult`, which gains an explicit `primaryAuto bool` parameter. Add `mergeStatusChanges` for handlers that produce more than one block. This task changes no output — the rendering tasks that follow consume the new field.

**Outcome**: Every call site that produces status movements also produces a merged, deduplicated, no-op-free `[]StatusChange` whose `auto` flag is false only for a change the caller asked for, verified by unit tests, with every command's output byte-identical to before the change.

**Do**:
1. `internal/cli/format.go` — add `type StatusChange struct { ID, Title, From, To string; Auto bool }` beside `CascadeEntry`, and add `Changed []StatusChange` to `CascadeResult`. Keep `TaskID`, `TaskTitle`, `OldStatus`, `NewStatus` and `Cascaded` exactly as they are — `PrettyFormatter.FormatCascadeTransition` (`internal/cli/pretty_formatter.go:234-275`) builds its head line and its `ParentID`-linked tree from them.
2. `internal/cli/transition.go` — add an unexported `statusChangeSet` with `add(StatusChange)` and `rows() []StatusChange`. `add` keys on `task.NormalizeID(c.ID)`: an unseen ID records the row and its position in first-seen order; a seen ID keeps the stored `From`, overwrites `To`, keeps the first non-empty `Title`, and sets `Auto` to `stored.Auto && incoming.Auto`. `rows()` returns the rows in first-seen order, skipping any whose `From == To`, and returns a non-nil empty slice when nothing survives.
3. `internal/cli/transition.go` — change `buildCascadeResult` to `buildCascadeResult(id, title string, result task.TransitionResult, cascades []task.CascadeChange, tasks []task.Task, primaryAuto bool) CascadeResult`. After building `Cascaded` as it does today, feed a `statusChangeSet` with the primary (`id`, `title`, `result.OldStatus`, `result.NewStatus`, `primaryAuto`) followed by each cascade entry (`Auto: true`) in order, and assign `cr.Changed = set.rows()`.
4. `internal/cli/transition.go` — add `mergeStatusChanges(blocks ...CascadeResult) []StatusChange`: feeds every block's `Changed` rows through one `statusChangeSet` in block order and returns `rows()`.
5. Update the four call sites: `internal/cli/transition.go` in `RunTransition` passes `false` and builds the result unconditionally (drop the `if len(c) > 0` guard so `cascadeResult` is always non-nil); `internal/cli/create.go:242`, `internal/cli/update.go:316` and `internal/cli/update.go:377` pass `true`.

**Acceptance Criteria**:
- [ ] `CascadeResult.Changed` lists the requested change first, then cascaded changes in the order the state machine produced them
- [ ] The row for a task the caller named through `start`/`done`/`cancel`/`reopen` carries `Auto: false`; every other row carries `Auto: true`
- [ ] Every row of a block built with `primaryAuto: true` carries `Auto: true`
- [ ] A task appearing in more than one block collapses to one row whose `From` is its status before the command and whose `To` is its status after
- [ ] A task whose merged `From` equals its merged `To` produces no row
- [ ] `rows()` and `mergeStatusChanges()` return a non-nil empty slice, never nil, when no row survives
- [ ] `mergeStatusChanges` over a single block returns that block's `Changed` unchanged
- [ ] `RunTransition` builds a `CascadeResult` whether or not cascades fired
- [ ] Toon, pretty and JSON output for every command is byte-identical to before this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it records the requested change with auto false"` — `buildCascadeResult(..., false)` yields a first row with `Auto == false`
- `"it marks cascaded changes auto true"` — every non-first row of that block carries `Auto == true`
- `"it marks every row auto true when the primary was system-initiated"` — `buildCascadeResult(..., true)` yields no `Auto == false` row
- `"it collapses a task moved twice into one row"` — a task moved done→open by one block and open→cancelled by another yields one row reading done→cancelled
- `"it drops a task that ends where it started"` — done→open in one block and open→done in another yields no row for that task
- `"it keeps first-seen order across blocks"` — rows come back in the order their tasks were first seen, requested change first
- `"it keeps the first non-empty title when a task is seen twice"` — a later block carrying an empty title does not blank the row
- `"it returns an empty non-nil slice when nothing changed"` — `mergeStatusChanges()` with no blocks, and a block whose only change is a no-op
- `"it leaves a single block's rows unchanged"` — `mergeStatusChanges(b)` equals `b.Changed`
- `"it builds a cascade result for a transition with no cascades"` — `tick start` on a childless task yields `Changed` with exactly one row
- `"it leaves transition output unchanged"` — `tick start`/`tick done` stdout in toon and pretty matches the strings the suite asserts today

**Edge Cases**:
- A task moved twice by two cascades meeting on a shared ancestor — the merged row must read from the pre-command status, so the stored `From` is kept and only `To` is overwritten
- A shared ancestor that ends where it started — dropped, because the table lists what changed
- Every row `auto=true` when the caller asked for no status change (re-parenting, `create --parent <done task>`)
- Deterministic row order — first-seen order, requested change first; no map iteration may reach the output
- An empty set is valid and must be an empty non-nil slice, so JSON can render `[]` rather than `null` later in the phase
- Pretty's tree inputs must survive: `TaskTitle`, `OldStatus`, `NewStatus` and each `CascadeEntry.ParentID` stay on `CascadeResult` untouched
- IDs are compared through `task.NormalizeID`, as every other ID comparison in these files is

**Context**:
> §7.2: "Every task whose status moved gets a row, and an `auto` column says whether that row is the change the caller asked for":
> ```
> changed[3]{id,title,from,to,auto}:
>   tick-a1b2,Add retry to the sync worker,in_progress,done,false
>   tick-9f3c,Parse the header,open,done,true
>   tick-77ab,Validate fields,in_progress,done,true
> ```
> "A task appears at most once. One command can move the same task's status twice — the two cascades of §7.5 meeting on a shared ancestor — and the table still carries one row for it, reading from the status it held before the command to the status it holds after. A task that ends where it started carries no row: the table lists what changed."
>
> §7.2 on where the `auto` vocabulary comes from: "Every task's transition history already records each change with an `auto` flag — false when a user or agent asked for it, true when the system produced it as a consequence — stored per task in the JSONL and in the `task_transitions` table. Inventing a 'requested' column instead would be exactly the move that produced the malformed output this work removes."
>
> §7.2 on a re-parenting: "A re-parenting, where nothing the caller asked for was itself a status change, is the same shape with every row marked as a consequence."
>
> §7.5 records why `create` and `update` produce cascades at all: "`tick create --parent <done task>` reopens that parent… and that reopen travels further up if its own parent was done (Rule 6, then Rule 5)"; "`tick update <id> --parent <other>` can fire two unrelated changes at once: the new parent reopens if it was finished, and the old parent may auto-complete if the moved task was the last unfinished thing under it (Rule 6 and Rule 3 together)."
>
> The codebase already distinguishes the two flavours at the source: `ApplyUserTransition` sets `auto=false` on the primary target and `ApplySystemTransition` sets `auto=true`. `RunTransition` calls the former; `validateAndReopenParent` (`internal/cli/helpers.go:110-130`) and `autoCompleteParentIfTerminal` (`internal/cli/update.go:128`) call the latter. The new `primaryAuto` parameter mirrors that split rather than inventing a second rule.
>
> `outputTransitionOrCascade`'s doc comment warns that a `CascadeResult` must be built inside the `Mutate` closure while the `tasks` slice is valid. `StatusChange` holds only copied values, so merging across blocks after the closure returns is safe — but the blocks themselves must still be built inside it.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §7.2, §7.4, §7.5

## free-text-round-trip-2-2

### Task 2: Status commands emit the changed table in toon

**Problem**: `start`, `done`, `cancel` and `reopen` answer with an arrow diagram. With no cascade the line comes from `baseFormatter.FormatTransition` (`internal/cli/format.go:211-213`), embedded by both the toon and pretty formatters so the two emit byte-identical text; with a cascade it comes from `ToonFormatter.FormatCascadeTransition` (`internal/cli/toon_formatter.go:145-154`), the same construction with ` (auto)` appended. Neither parses as TOON, and reading either requires knowing that the ID precedes the colon, that the arrow separates old state from new, and that `(auto)` marks a knock-on. Worse, which of the two shapes arrives depends on whether a cascade fired, so a reader must branch on the document before it can read it. The shared method also cannot be repaired in place: it takes `(id, oldStatus, newStatus)` and §7.2's row needs a title, and editing it would move pretty (§4.1, §4.3).

**Solution**: Route every status command through `FormatCascadeTransition` with the always-built `CascadeResult` from Task 1, render it in toon as one `changed[N]{id,title,from,to,auto}` table produced by the library, and split the shared arrow-line method by deleting `FormatTransition` from the `Formatter` interface and its implementations — pretty's arrow line already lives inside `PrettyFormatter.FormatCascadeTransition` and stays there.

**Outcome**: `tick start`, `tick done`, `tick cancel` and `tick reopen` emit exactly one `changed` table in toon, decodable by the project's TOON library, whether or not a cascade fired; pretty's output for the same commands is byte-identical to before.

**Do**:
1. `internal/cli/helpers.go` — replace `outputTransitionOrCascade` with `outputStatusChanges(stdout io.Writer, fmtr Formatter, cr CascadeResult)` that unconditionally writes `fmtr.FormatCascadeTransition(cr)` through `fmt.Fprintln`. Update the callers at `internal/cli/transition.go` (in `RunTransition`), `internal/cli/create.go:283` and `internal/cli/update.go:415`, `internal/cli/update.go:420` to pass the `CascadeResult` by value.
2. `internal/cli/toon_formatter.go` — add `toonChangedRow` with tags `toon:"id"`, `toon:"title"`, `toon:"from"`, `toon:"to"`, `toon:"auto"` (the last a `bool`), and `buildChangedSection(changes []StatusChange) string` returning the literal `changed[0]{id,title,from,to,auto}:` when `changes` is empty and `encodeToonSection("changed", rows)` otherwise — the same empty-header pattern `buildRelatedSection` already uses (`internal/cli/toon_formatter.go:296-306`).
3. `internal/cli/toon_formatter.go` — rewrite `FormatCascadeTransition` to `return buildChangedSection(result.Changed)`, deleting the arrow-line construction and the `result.TaskID == ""` early return.
4. `internal/cli/format.go` — remove `FormatTransition` from the `Formatter` interface, from `baseFormatter` and from `StubFormatter`; `internal/cli/json_formatter.go` — remove `JSONFormatter.FormatTransition` and the `jsonTransition` struct (`internal/cli/json_formatter.go:132-146`).
5. Rewrite the assertions the removals and the new shape break: `internal/cli/base_formatter_test.go` (the `FormatTransition` subtests and `TestAllFormattersProduceConsistentTransitionOutput` at lines 129-150), `internal/cli/toon_formatter_test.go:318-325`, `internal/cli/format_test.go:207-211` and the empty-`CascadeResult` loop at `internal/cli/format_test.go:414-421`, `internal/cli/json_formatter_test.go:537`, `internal/cli/helpers_test.go:293-350`, `internal/cli/transition_test.go:389-407` and the cascade assertions beneath it, and the toon subtests in `internal/cli/cascade_formatter_test.go`. Toon assertions decode stdout with `decodeToonDoc` (`internal/cli/toon_decode_test.go`, added in Phase 1) and check row values; pretty assertions keep their golden strings unchanged.
6. `CLAUDE.md:49` — remove `FormatTransition` from the Formatter interface's method list, leaving the eight that remain and the rest of the bullet as it is.

**Acceptance Criteria**:
- [ ] `tick start` on a task with no children emits `changed[1]{id,title,from,to,auto}:` and one row — never a bare arrow line
- [ ] A cascade emits one table containing the requested change and every knock-on, with the requested row first and the count matching the row total
- [ ] `auto` renders unquoted and decodes as a Go `bool`; the requested row is `false`, every cascaded row `true`
- [ ] A title containing a comma is quoted by the library and decodes back as one value
- [ ] An empty change set renders `changed[0]{id,title,from,to,auto}:` and decodes to an empty list
- [ ] `tick start --quiet` and its siblings print nothing at all
- [ ] Pretty output for `start`, `done`, `cancel` and `reopen` — the single line and the box-drawing tree — is byte-identical to before this task
- [ ] `grep -rn 'FormatTransition' internal/` returns nothing
- [ ] `CLAUDE.md`'s Formatter bullet lists the eight remaining methods and no longer names `FormatTransition`
- [ ] Every status command's toon stdout decodes via `toon.DecodeString`
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it renders a single transition as a one-row changed table"` — `tick start` on a childless task; decoded `changed` has one row with the task's id, title, from, to and `auto == false`
- `"it renders a cascade as one table with the requested row first"` — `tick done` on a parent with an open child; decoded `changed` has two rows in that order
- `"it marks cascaded rows auto true"` — the child's row decodes with `auto == true`
- `"it quotes a title containing a comma"` — the decoded row's title equals the stored title
- `"it decodes auto as a boolean"` — the decoded value's Go type is `bool`, not `string`
- `"it renders an empty changed set as a count-zero header"` — `buildChangedSection(nil)` equals `changed[0]{id,title,from,to,auto}:` and decodes to an empty list
- `"it prints nothing under --quiet"` — stdout is empty for `tick done --quiet`
- `"it leaves pretty transition output unchanged"` — the golden single line and the golden `Cascaded:` tree still match

**Edge Cases**:
- A single transition with no cascades still renders a one-row table, never a bare line — the reader must not branch on which document arrived
- Toon and pretty stop being byte-identical, so `TestAllFormattersProduceConsistentTransitionOutput` goes rather than being weakened
- A title containing a comma, a colon or a leading dash — quoting is the library's job via `encodeToonSection`, never the formatter's
- `auto` must be a Go `bool` in `toonChangedRow`, so the encoder emits `true`/`false` unquoted rather than a quoted string
- The empty case cannot come from `encodeToonSection`: an empty slice encodes as `changed[0]:` with no field schema, so the count-zero header is a literal, exactly as `buildRelatedSection` and `buildNotesSection` write theirs
- `format_test.go`'s "empty CascadeResult returns empty string" expectation no longer holds for toon — an empty result is a count-zero table, and `StubFormatter` alone still returns `""`
- `create` and `update` keep their trailing call in this task, so their toon output is a detail document followed by a `changed` table until Task 5 folds it in; that is the interim state Phase 2 exists to close, not a regression to assert against

**Context**:
> §7.2: "Every task whose status moved gets a row, and an `auto` column says whether that row is the change the caller asked for." Two reasons carried it: "A reader must not have to branch on which document arrived before it can read either. An agent parsing status output sees one table whatever command produced it, and the count is always right. A singular block for the requested change beside a table of knock-ons would be two documents for one kind of event." And: "The table carries the title so no second lookup is needed to know what moved. The trailing `(auto)` marker of the old arrow lines disappears into the column that always meant it."
>
> §7.3: "`done`, `start`, `cancel` and `reopen` return **only** the `changed` table." The third option — leaving status output as prose — was declined: "the multi-task output carries titles the agent would otherwise have to look up, and one command's output parsing while another's does not — depending on whether a cascade fired — is exactly the branching rule this work exists to delete."
>
> §4.3: "The single transition line comes from `baseFormatter.FormatTransition`… embedded by both the toon and pretty formatters, so the two emit byte-identical text today. Restructuring the toon form requires splitting that method." §4.1: "What a terminal prints today is what it prints after this work — the single transition line, the box-drawing cascade tree."
>
> The method cannot be kept and overridden: `FormatTransition(id, oldStatus, newStatus)` carries no title, and §7.2's row requires one. `PrettyFormatter.FormatCascadeTransition` already builds its head line inline (`internal/cli/pretty_formatter.go:238`) and returns exactly the old arrow line when `Cascaded` is empty, so routing every status command through it leaves pretty's bytes untouched.
>
> Verified against `github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c`: `encodeToonSection("changed", rows)` over a struct with a `bool` field emits
> ```
> changed[2]{id,title,from,to,auto}:
>   tick-a1b2,Add retry to the sync worker,in_progress,done,false
>   tick-9f3c,"Parse, the header",open,done,true
> ```
> which decodes to `[]any` of maps with `auto` as a Go `bool`; the hand-written `changed[0]{id,title,from,to,auto}:` decodes to an empty list; and an empty slice handed to `encodeToonSection` yields `changed[0]:` with the schema lost, which is why the literal is needed.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §7.1, §7.2, §7.3, §4.1, §4.3

## free-text-round-trip-2-3

### Task 3: JSON status output becomes the changed list

**Problem**: JSON status output is two shapes for one kind of event. A cascade renders `{"transition":{"id","from","to"},"cascaded":[{"id","title","from","to"}]}` (`internal/cli/json_formatter.go:259-303`), so the requested change sits in a different object from the knock-ons, carries no title, and is distinguished by its position in the document rather than by a value a consumer can read. A consumer must branch on document shape to read either, which is the same defect the toon table just removed — and §4.2 requires JSON to move with toon so that a consumer parsing JSON gets the same structured answer as one parsing toon.

**Solution**: Replace the `transition`/`cascaded` pair with a single `changed` array of objects carrying `id`, `title`, `from`, `to` and a JSON boolean `auto`, built from the merged `CascadeResult.Changed` rows of Task 1.

**Outcome**: Every status command's JSON output is `{"changed":[…]}` with the same rows in the same order as the toon table, `changed` is `[]` and never `null`, and no `transition` or `cascaded` key exists anywhere in the codebase.

**Do**:
1. `internal/cli/json_formatter.go` — delete `jsonCascadeTransition` and `jsonCascadeResult`; replace `jsonCascadeEntry` with `jsonStatusChange` carrying `ID string \`json:"id"\``, `Title string \`json:"title"\``, `From string \`json:"from"\``, `To string \`json:"to"\`` and `Auto bool \`json:"auto"\``.
2. `internal/cli/json_formatter.go` — add `toJSONStatusChanges(changes []StatusChange) []jsonStatusChange` returning a non-nil slice (`make([]jsonStatusChange, 0, len(changes))`), mirroring `toJSONRelated` (`internal/cli/json_formatter.go:118-126`).
3. `internal/cli/json_formatter.go` — rewrite `FormatCascadeTransition` to marshal a wrapper struct with the single field `Changed []jsonStatusChange \`json:"changed"\`` populated from `toJSONStatusChanges(result.Changed)`, deleting the `result.TaskID == ""` early return so an empty set renders `{"changed": []}`.
4. `internal/cli/cascade_formatter_test.go` — rewrite the JSON subtests (lines 160-230 and the JSON branch of `TestAllFormattersCascadeEmptyArrays` at 289-300) to unmarshal into `map[string]any` and assert the `changed` array's length, each row's field values, `auto`'s Go type and the absence of `transition` and `cascaded`.
5. `internal/cli/transition_test.go` — add end-to-end subtests running the status commands with `--json` and asserting decoded values.

**Acceptance Criteria**:
- [ ] `tick start|done|cancel|reopen --json` emits a single object whose only key is `changed`
- [ ] `changed` is `[]` and never `null` when nothing moved
- [ ] Each element carries `id`, `title`, `from`, `to` and `auto`, with `auto` a JSON boolean (`true`/`false`, unquoted)
- [ ] The single-transition branch and the cascade branch produce the same shape, differing only in row count
- [ ] Row order and row content match the toon table for the same command
- [ ] `grep -rn '"transition"\|"cascaded"\|jsonCascadeResult\|jsonCascadeTransition\|jsonCascadeEntry' internal/` returns nothing
- [ ] Toon and pretty output are unchanged by this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it renders a single transition as a one-element changed list"` — `tick start --json` on a childless task; `changed` has one element with `auto == false`
- `"it renders a cascade as one changed list"` — parent plus open child; two elements, requested first, second with `auto == true`
- `"it emits changed as an empty array never null"` — an empty `CascadeResult` renders `{"changed": []}` and `json.Unmarshal` yields a non-nil `[]any`
- `"it emits auto as a JSON boolean"` — the unmarshalled value's Go type is `bool`
- `"it carries the title on every row"` — each element's `title` equals the stored task title
- `"it emits no transition or cascaded key"` — neither key is present in the unmarshalled map for either branch
- `"it matches the toon table row for row"` — the same command's `--json` and `--toon` output decode to the same ids, froms, tos and autos in the same order

**Edge Cases**:
- `changed` is `[]` and never `null` — the slice must be allocated with `make`, as `toJSONRelated` already does for `blocked_by` and `children`
- `auto` is a JSON boolean, not the string `"true"` — a `bool` struct field, not a formatted string
- The `transition` and `cascaded` keys disappear entirely rather than being kept alongside for compatibility
- Single-transition and cascade branches produce one shape, so a consumer never branches on which keys are present
- An empty `CascadeResult` no longer renders `""` — `format_test.go`'s expectation for the JSON formatter changes with the toon one

**Context**:
> §4.2: "A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list (`grep -n 'json:\"transition\"\|json:\"cascaded\"' internal/cli/json_formatter.go` → `json_formatter.go:276-277`)."
>
> §7.2 fixes the row's vocabulary: `id`, `title`, `from`, `to`, `auto`, with `auto` false only for "the change the caller asked for". The title is carried "so no second lookup is needed to know what moved".
>
> The JSON formatter already treats empty collections this way — `toJSONRelated` "Always returns a non-nil empty slice to ensure JSON `[]` instead of `null`" — so the `changed` list follows the existing convention rather than inventing one.
>
> The rows themselves are produced by Task 1 and are already merged, deduplicated and no-op-free; this task renders them and adds no merging logic of its own.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §4.2, §7.2, §7.3

## free-text-round-trip-2-4

### Task 4: The detail document carries a changed section

**Problem**: `create` and `update` print the full task detail and then append transition output after it (`internal/cli/create.go:283`, `internal/cli/update.go:415`, `internal/cli/update.go:420`). A reader handed that stream sees a task-detail document with foreign lines stuck on the end — and after Task 2 it sees a task-detail document followed by a second, complete document. In JSON it is literally two top-level documents concatenated. §7.4 requires one document with the changes as a section inside it, and requires that section to be present even when nothing moved, so a reader never has to discover whether it exists. Meanwhile `show`, `note add` and `note remove` emit the same detail document through the same helper (`outputMutationResult`, `internal/cli/helpers.go:16-30`) and must carry no such section at all (§7.3), so the detail structure has to express "no section" and "section with nothing in it" as different states.

**Solution**: Give `TaskDetail` an optional `Changes *StatusChanges` field holding both the merged rows (for toon and JSON) and the per-block cascade results (for pretty's tree), and have each formatter render it: toon as a `changed` section between `notes` and `description`, JSON as a `changed` key, pretty as the transition line or tree appended after the detail body exactly as the handler prints it today. `outputMutationResult` gains the parameter and passes it through.

**Outcome**: A `TaskDetail` carrying a non-nil `Changes` renders one document with a `changed` section — count-zero when no task moved — in all three formats; a `TaskDetail` carrying nil renders exactly what it renders today; and pretty's bytes are unchanged in both cases.

**Do**:
1. `internal/cli/format.go` — add `type StatusChanges struct { Rows []StatusChange; Blocks []CascadeResult }` and add `Changes *StatusChanges` to `TaskDetail`. A nil pointer means the document carries no `changed` section; a non-nil pointer means it always does.
2. `internal/cli/toon_formatter.go` — in `FormatTaskDetail`, when `detail.Changes != nil`, append `buildChangedSection(detail.Changes.Rows)` after the notes section and before the description section, so `description` stays last.
3. `internal/cli/json_formatter.go` — add `Changed *[]jsonStatusChange \`json:"changed,omitempty"\`` to `jsonTaskDetail` and set it to the address of `toJSONStatusChanges(detail.Changes.Rows)` when `detail.Changes != nil`, leaving it nil otherwise. It must be a pointer: `omitempty` on a plain slice drops an empty non-nil slice too, which would erase the count-zero case.
4. `internal/cli/pretty_formatter.go` — in `FormatTaskDetail`, when `detail.Changes != nil && len(detail.Changes.Blocks) > 0`, append `"\n"` followed by each block rendered through `f.FormatCascadeTransition(block)` joined by `"\n"`. Append nothing when there are no blocks.
5. `internal/cli/helpers.go` — change `outputMutationResult` to take a trailing `changes *StatusChanges` and set `detail.Changes = changes` before formatting. Pass `nil` from `internal/cli/note.go:86` and `internal/cli/note.go:142`, and `nil` from `internal/cli/create.go:277` and `internal/cli/update.go:409` for now — Task 5 supplies their real values and removes their trailing calls.

**Acceptance Criteria**:
- [ ] A `TaskDetail` with `Changes != nil` renders `changed[N]{id,title,from,to,auto}:` in toon, positioned after `notes` and before `description`
- [ ] A `TaskDetail` with `Changes` non-nil and no rows renders `changed[0]{id,title,from,to,auto}:`, which decodes to an empty list
- [ ] A `TaskDetail` with `Changes == nil` renders no `changed` key in toon or JSON and appends nothing in pretty
- [ ] JSON renders `"changed": []` for the non-nil empty case and omits the key entirely for the nil case
- [ ] The full toon document decodes via `toon.DecodeString` with the `changed` section present, and the JSON document parses as a single object
- [ ] Pretty's rendered detail plus blocks is byte-for-byte what `Fprintln(detail)` followed by `Fprintln(block)` per block produced, including blank-line spacing
- [ ] `show`, `note add` and `note remove` output carries no `changed` section in any format
- [ ] The section's position in document order is the same for every caller
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it carries a changed section when the detail carries changes"` — decoded toon document has a `changed` list with the expected rows
- `"it carries a count-zero changed section when the change set is empty"` — non-nil `Changes` with no rows decodes to an empty `changed` list
- `"it omits the changed section entirely when the detail carries no changes"` — nil `Changes` decodes to a document with no `changed` key
- `"it places the changed section after notes and before description"` — the emitted section order is asserted on the raw section split, and `description` is still last
- `"it emits changed as [] and never null in json"` — `json.Unmarshal` gives a non-nil empty `[]any`
- `"it omits changed from json when the detail carries no changes"` — the key is absent, not `null`
- `"it appends nothing in pretty when the detail carries no blocks"` — output equals the detail body exactly, with no trailing blank line
- `"it appends one block in pretty exactly as the handler printed it"` — the string equals `detail + "\n" + block`
- `"it appends two blocks in pretty in order"` — the string equals `detail + "\n" + first + "\n" + second`
- `"it carries no changed section on note add"` — end-to-end `tick --toon note add` output decodes with no `changed` key
- `"it carries no changed section on show"` — end-to-end `tick --toon show` output decodes with no `changed` key

**Edge Cases**:
- No section versus an always-present count-0 section must be distinguishable — nil pointer versus non-nil pointer to a possibly-empty `Rows`; a bare slice cannot express it and `omitempty` on a slice silently collapses the two
- Pretty must stay byte-identical including blank-line spacing: today's two `Fprintln` calls produce `detail + "\n" + block + "\n"`, which a single `Fprintln` of `detail + "\n" + block` reproduces exactly
- Pretty with a non-nil `Changes` and zero blocks appends nothing — not an empty line, not a `Cascaded:` header
- The section's position is fixed once here and is identical for `create` and `update`; `description` stays last, as Phase 1 left it
- `show` builds its detail inline (`internal/cli/show.go:52-64`) rather than through the helper, so its `Changes` is nil by construction and must be asserted rather than assumed
- The end-to-end proof that JSON stops emitting a second top-level document lands in Task 5, which removes the handlers' trailing calls; this task proves the single-document rendering at the formatter and helper level

**Context**:
> §7.4: "`create` and `update` today print the full task detail and then append transition lines after it… A reader handed that stream sees a task-detail document with foreign lines stuck on the end. **Where a command produces both a record and status changes, the result is one document with the changes as a section inside it.** Making each section valid is not sufficient on its own; the stream must be one document."
>
> §7.4 on the always-present rule: "`tick create` with no parent moves no task's status; the document still carries `changed[0]{id,title,from,to,auto}:`, exactly as an empty notes or children section carries its count-zero header (§8). A reader that must first find out whether the section exists is branching on which document arrived, which §7.2 exists to prevent."
>
> §7.3 on which commands carry it: "`show`, `note add` and `note remove` carry no `changed` section at all. Only a parent/child structural change moves another task's status (§7.5), and none of the three performs one, so there is nothing for the section to hold. The section belongs to `create` and `update`, even though all four mutating commands share one detail helper (§3.1)."
>
> §4.1 keeps pretty as it is: "the single transition line, the box-drawing cascade tree" print exactly as they do today. Moving pretty's trailing output inside `FormatTaskDetail` is a relocation, not a change — the bytes the command writes are identical, and `pretty_formatter_test.go`'s golden assertions on a `TaskDetail` with no `Changes` are untouched (§11 keeps pretty's golden strings).
>
> §5.2 fixes the rest of the document: "Which sections a document carries is unchanged by this work." The `changed` section is the one addition, and the specification does not state where in the order it sits. It is placed after `notes` and before `description` so that `description` remains the last section, which Phase 1 established and its tests assert.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §7.3, §7.4, §4.1, §4.2, §3.1

## free-text-round-trip-2-5

### Task 5: create and update emit one document

**Problem**: `create` and `update` still print their status changes after the detail document rather than inside it, and `update` prints up to two of them: the Rule 6 block when the new parent reopens and the Rule 3 block when the old parent auto-completes (`internal/cli/update.go:411-421`). Those two blocks can name the same task — the two cascades of §7.5 meeting on a shared ancestor — so the same task is reported twice with contradictory-looking rows, and an ancestor that was reopened by one block and re-completed by the other appears as two changes when nothing changed. `create` prints no status section at all when the new task has no parent, so a reader must find out whether the section exists before it can read it. Task 4 built the section; nothing yet fills it.

**Solution**: Collect each handler's cascade blocks into a `StatusChanges`, merge them into one row set with `mergeStatusChanges`, hand it to `outputMutationResult`, and delete the trailing output calls. The value is always non-nil for both commands, so the section is always present.

**Outcome**: `tick create` and `tick update` emit exactly one document in toon and JSON — one that decodes whole — carrying a `changed` section that lists every task whose status moved once each, `changed[0]` when none did; pretty prints the same bytes it prints today.

**Do**:
1. `internal/cli/create.go` — inside the `Mutate` closure, append the parent-reopen `CascadeResult` (built at `create.go:242`) to a `blocks []CascadeResult` slice instead of the `parentReopened`/`parentResult`/`parentCascadeResult` trio. After `Mutate` returns, build `changes := &StatusChanges{Blocks: blocks, Rows: mergeStatusChanges(blocks...)}` and pass it to `outputMutationResult`. Delete the trailing `outputStatusChanges` call and its `parentReopened && !fc.Quiet` guard.
2. `internal/cli/update.go` — same: append the Rule 6 block (`update.go:316`) then the Rule 3 block (`update.go:377`) to one `blocks` slice in that order, replacing `r6Triggered`/`r6ParentID`/`r6Result`/`r6CascadeResult` and `r3ParentID`/`r3Result`/`r3CascadeResult`. Build the `StatusChanges` after `Mutate` and pass it to `outputMutationResult`; delete both trailing `outputStatusChanges` calls.
3. `internal/cli/create.go`, `internal/cli/update.go` — pass a non-nil `*StatusChanges` on every path, including when no block fired, so both commands always carry the section.
4. `internal/cli/create_test.go`, `internal/cli/update_test.go` — add toon and JSON end-to-end subtests decoding the whole stdout as one document and asserting the `changed` rows; keep the existing pretty subtests (`create_test.go:1180-1210`, `create_test.go:1255-1280`, `update_test.go:1050-1065`) asserting the same strings they assert today.

**Acceptance Criteria**:
- [ ] `tick create` and `tick update` stdout decodes as exactly one TOON document and, under `--json`, parses as exactly one JSON object
- [ ] `tick create` with no `--parent` emits `changed[0]{id,title,from,to,auto}:`
- [ ] `tick create --parent <done task>` emits one row for the reopened parent and one for each ancestor the reopen travelled to, every row `auto=true`
- [ ] `tick update <id> --parent <other>` firing both Rule 6 and Rule 3 emits one table containing both blocks' rows, each task once
- [ ] A task named by both blocks that ends where it started carries no row
- [ ] Rows appear in block order — Rule 6's before Rule 3's — matching the order the two blocks print in pretty today
- [ ] `tick create --quiet` and `tick update --quiet` print only the task ID and nothing else
- [ ] Pretty output for both commands is byte-identical to before this task, including the blank line before `Cascaded:` and the order of two blocks
- [ ] `grep -n 'outputStatusChanges' internal/cli/create.go internal/cli/update.go` returns nothing
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it emits one document for create under a done parent"` — decoded toon document carries the task's own fields and a `changed` list with the parent's row
- `"it emits changed[0] for create with no parent"` — decoded `changed` is an empty list, and the key is present
- `"it cascades a reopen to a done grandparent in one table"` — parent and grandparent both appear, once each, `auto=true`
- `"it collapses update's rule 6 and rule 3 blocks into one table"` — both parents appear in one `changed` list, in block order
- `"it drops a shared ancestor that ends where it started"` — an ancestor reopened by Rule 6 and re-completed by Rule 3 has no row
- `"it prints only the task ID under --quiet"` — stdout is the ID plus a newline for both commands
- `"it emits one JSON object"` — `json.Unmarshal` of the whole stdout succeeds and yields one object carrying `changed`
- `"it leaves pretty create output unchanged"` — the golden detail-plus-cascade string still matches
- `"it leaves pretty update output unchanged with both blocks"` — the golden two-block string still matches

**Edge Cases**:
- `update`'s Rule 6 and Rule 3 blocks collapse into one table, including when they meet on a shared ancestor — the merge is `mergeStatusChanges`, not a concatenation
- A shared ancestor that ends where it started carries no row, because the table lists what changed
- `create` with no parent carries `changed[0]`, so the section is never absent from a `create` or `update` document
- Quiet mode short-circuits inside `outputMutationResult` before any formatting, so neither the detail nor the section is rendered
- Pretty must keep both blocks in today's order, with today's spacing — blocks are appended in the order they were collected, Rule 6 first
- Blocks must still be built inside the `Mutate` closure, where the `tasks` slice is valid; only the merge and the rendering happen after it returns
- The `--blocks`/`--blocked-by` paths move no task's status, so they contribute no rows and must not create a block

**Context**:
> §7.4: "**Where a command produces both a record and status changes, the result is one document with the changes as a section inside it.**" And: "`update` can carry two independent cascade blocks today because two unrelated changes can fire at once — this collapses into the single `changed` table along with everything else."
>
> §7.4 on presence: "**The section is always there.** `tick create` with no parent moves no task's status; the document still carries `changed[0]{id,title,from,to,auto}:`."
>
> §7.3 on why these two carry the full record: "`create` and `update` are edits and the caller wants the result of the edit — the new ID, the merged fields — whereas a status change is something the caller already knows it did."
>
> §7.5 on what actually moves: "`create` and `update` emit cascades not because the edited task's status moved, but because *other* tasks' statuses moved as a consequence of the parent/child structure changing… **Adding work under a finished parent.** `tick create --parent <done task>` reopens that parent — it is no longer complete — and that reopen travels further up if its own parent was done (Rule 6, then Rule 5). **Moving a task to a different parent.** `tick update <id> --parent <other>` can fire two unrelated changes at once: the new parent reopens if it was finished, and the old parent may auto-complete if the moved task was the last unfinished thing under it (Rule 6 and Rule 3 together). These tasks are elsewhere in the tree and are not in the edited task's record, which is why they cannot be folded into it."
>
> §7.2: every row of these tables is a consequence, so all of them read `auto=true` — Task 1 supplies that through `primaryAuto: true` at both call sites.
>
> §4.1 keeps pretty exactly as it is; Task 4 relocated pretty's trailing blocks into `FormatTaskDetail`, so removing the handlers' trailing calls here must leave the printed bytes identical.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §7.2, §7.3, §7.4, §7.5, §4.1

## free-text-round-trip-2-6

### Task 6: README transition and cascade samples match real output

**Problem**: The README's Transition & Cascade Output section prints output the tool no longer produces. The simple-transition sample (`README.md:471-475`) is labelled "TOON / Pretty" and shows one arrow line for both, but toon now emits a `changed` table and only pretty keeps the arrow. The JSON sample (`README.md:481-486`) shows `{"id","from","to"}`, replaced by the `changed` list. The cascade samples show arrow lines with ` (auto)` in toon (`README.md:499-505`) and a box-drawing tree in pretty (`README.md:511-518`) — and both print a `tick-f3e4: done (unchanged)` row for an unchanged terminal child, a marker the tool has never implemented. The README is live documentation someone reads to learn the tool, so leaving it describing output the tool does not produce is shipping a defect.

**Solution**: Replace all four samples with output copied from the real binary, split the "TOON / Pretty" cell into separate TOON and Pretty samples now that the two differ, drop the `(unchanged)` rows rather than reproducing them, and extend Phase 1's README decode test to cover the new toon samples.

**Outcome**: Every transition and cascade sample in the README is byte-for-byte what the tool prints, the `(unchanged)` marker appears nowhere in the document, and a test fails if either toon sample stops parsing as TOON.

**Do**:
1. Build the binary to a scratch path and, in a throwaway `.tick` project, produce the real `--toon`, `--pretty` and `--json` output for `tick start` on a childless task and for `tick done` on a parent with one open child.
2. `README.md:467-491` — replace the single "**Simple transition** (TOON / Pretty)" cell with a TOON cell carrying the real one-row `changed` table and a Pretty cell carrying the real arrow line, and replace the JSON cell's body with the real `{"changed":[…]}` document. Adjust the two cell labels so neither claims the two formats share a shape.
3. `README.md:495-522` — replace the "**TOON** (flat lines)" cell body with the real `changed[2]{id,title,from,to,auto}:` table and relabel the cell so it no longer says "flat lines"; replace the "**Pretty** (tree with box-drawing)" cell body with the real tree. Both cells lose the `tick-f3e4 … (unchanged)` row.
4. `README.md:465` — correct the sentence introducing the section where it describes the old shape.
5. `internal/cli/readme_samples_test.go` — add `changed[1]{id,title,from,to,auto}:` and `changed[2]{id,title,from,to,auto}:` to the anchor list of `TestREADMEToonSamplesDecode` (added in Phase 1 task free-text-round-trip-1-6), so each is located by first line and decoded.

**Acceptance Criteria**:
- [ ] Both toon samples were copied from real tool output and decode via `toon.DecodeString`
- [ ] The simple-transition toon sample is a one-row `changed` table carrying the task's title and `auto` reading `false`
- [ ] The cascade toon sample is one table whose count matches its rows, the requested row first with `auto` `false` and each cascaded row `true`
- [ ] The JSON sample is a single object whose only key is `changed`, with `auto` as an unquoted boolean
- [ ] The pretty samples match real pretty output, and the cascade one is still a box-drawing tree
- [ ] `grep -n '(unchanged)' README.md` returns nothing
- [ ] No sample is labelled as shared between TOON and Pretty
- [ ] `TestREADMEToonSamplesDecode` covers both new anchors and fails if either block is removed or made unparseable
- [ ] The `tick show`, `tick list` and dep-tree samples are unchanged by this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it decodes the README simple transition sample"` — the block anchored on `changed[1]{id,title,from,to,auto}:` parses as TOON
- `"it decodes the README cascade sample"` — the block anchored on `changed[2]{id,title,from,to,auto}:` parses as TOON
- `"it fails when an anchored sample is missing"` — an anchor with no matching block is a failure, not a silent pass
- `"it carries the title column in the README samples"` — the decoded rows carry a non-empty `title`
- `"it parses the README JSON transition sample"` — the ```json block for the transition unmarshals to an object carrying only `changed`
- `"it has no unchanged marker in the README"` — the document contains no `(unchanged)`

**Edge Cases**:
- The `(unchanged)` marker in both the toon and the pretty cascade samples was never implemented and goes rather than being reproduced in the new shape — §7.6 leaves unchanged terminal children out of the table
- The pretty column stays a box-drawing tree, corrected to real output rather than flattened to match the toon table
- Sample rows must carry the new `title` column, which the old arrow lines did not have
- The two toon samples must have distinct first lines (`changed[1]…` and `changed[2]…`) so the anchor index cannot collide
- The README's fenced blocks sit inside `<table>` HTML; the samples are plain fenced blocks within table cells and the decode test's block collection must still find them
- The JSON sample's fence carries the `json` info string, so it is outside `TestREADMEToonSamplesDecode`'s empty-info-string collection and needs its own assertion

**Context**:
> §12.1 lists the samples this task owns:
>
> | Sample | Location | Why it changes |
> |---|---|---|
> | arrow transition | `README.md:473-475` | replaced by the `changed` table (§7.2) |
> | JSON `{id,from,to}` transition | `README.md:481-486` | JSON moves with toon (§4.2) |
> | cascade with `(auto)` / `(unchanged)` | `README.md:501-504` | replaced by the `changed` table; the `(unchanged)` marker was never implemented (§7.6) |
>
> §12.1: the README "is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect."
>
> §7.6: "The `changed` table lists what changed, exactly as today's output does. The `auto-cascade-parent-status` specification's requirement that unchanged terminal children be shown alongside a cascade is a pre-existing unimplemented requirement in another work unit's specification; it is not reinstated and not decided here."
>
> §4.1 keeps pretty's cascade tree: "The current cascade tree nests by which task caused which, so a grandchild closing because its parent closed shows as three levels of indentation. Flattened into rows, every knock-on looks equally directly caused. The toon table drops that too, but an agent holds the parent/child links and can reconstruct the chain; a human reading a terminal cannot, which is why the tree exists." Replicating the table shape in pretty was declined explicitly.
>
> The remaining §12.1 corrections are elsewhere: the `tick show` and `tick list` samples landed in Phase 1, the dep-tree summary header lands in Phase 3, and `--field`/`--fields` and `--` are added in Phases 4 and 5.
>
> §12.2 records that the `auto-cascade-parent-status` specification is owed a correction: it fixes the arrow-and-`(auto)` lines as the machine-readable cascade form, which §7.2 replaces outright — "plainly and load-bearingly wrong", so it is amended rather than left to supersession. Two bounds come with it. The amendment does not touch that specification's requirement that unchanged terminal children be shown alongside a cascade: §7.6 records it as "a pre-existing unimplemented requirement in another work unit's specification… not reinstated and not decided here", so it stands in its own document untouched. And "corrections are made by judgement, not as a blanket rewrite" — nothing else in that document is edited. That is a completed work unit's artifact: it is corrected through the session's corrigendum route — the amendment presented and confirmed, the wrong claim replaced in place, a dated corrigendum entry recording what the document used to claim and what is true instead, and the document re-indexed, which is the point of the route since a specification's content stays live in the knowledge base at full confidence — never by this task.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §12.1, §12.2, §7.2, §7.6, §4.1, §4.2
