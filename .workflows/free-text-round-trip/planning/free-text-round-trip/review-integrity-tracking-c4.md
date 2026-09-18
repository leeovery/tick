# Review Tracking: Free Text Round Trip - Integrity

## Findings

### 1. Deleting `FormatTransition` breaks three test sites the task does not name, one of them pretty's only golden for the arrow line

**Severity**: Important
**Plan Reference**: Phase 2, task `free-text-round-trip-2-2` (Status commands emit the changed table in toon) — tick record `tick-ed0952`, Do step 5
**Category**: Task Self-Containment — the task enumerates the assertions its own deletion breaks and the list is short by three
**Move**: settled
**Change Type**: update-task

**Problem**:
This task removes `FormatTransition` from the `Formatter` interface, from `baseFormatter`, from `StubFormatter` and from the JSON formatter, then lists the assertion sites that removal breaks. Ten call sites exist in the suite; the list reaches seven. The three it misses are `internal/cli/json_formatter_test.go:463` — the `FormatTransition` half of `"it formats transition/dep/message as JSON objects"`, whose other half the task already names at line 537; `TestOutputTransitionOrCascade`, which runs to `internal/cli/helpers_test.go:408` rather than the cited 350, and whose three further subtests (lines 355-408) all call the deleted `outputTransitionOrCascade` and one of them the deleted method; and `internal/cli/pretty_formatter_test.go:335-342`, the subtest `"it formats transition as plain text"`, which asserts pretty's arrow line `tick-a3f2b7: open → in_progress` by calling `PrettyFormatter.FormatTransition` directly.

The compiler surfaces all three, so nothing is silently missed. What is missed is the decision the third one forces. At the moment it surfaces, the implementer is holding pretty's only direct golden for the transition line, with an instruction in the same Do step reading "pretty assertions keep their golden strings unchanged" and the method that produced the string gone. Deleting the subtest is the path of least resistance and it costs pretty the one assertion that the human-facing transition line is still those exact bytes — the assertion §4.1's "what a terminal prints today is what it prints after this work" rests on, and the one §11 keeps pretty's goldens for.

**Proposal**:
Name the three sites and direct pretty's golden to move rather than go. `PrettyFormatter.FormatCascadeTransition` writes `fmt.Sprintf("%s: %s → %s", result.TaskID, result.OldStatus, result.NewStatus)` and returns at that point when `Cascaded` is empty (`internal/cli/pretty_formatter.go:234-243`), which is byte-for-byte what `FormatTransition` returned — so the subtest keeps its expected string verbatim and only the call it makes changes. The task's own Context already states this ("`PrettyFormatter.FormatCascadeTransition` already builds its head line inline (`internal/cli/pretty_formatter.go:238`) and returns exactly the old arrow line when `Cascaded` is empty"), so the fix applies the task's own reasoning to its own test list rather than adding anything. `helpers_test.go`'s cited range is corrected to the function's real extent, and `json_formatter_test.go:463` joins the sibling site at `:537` the task already carries.

**Current**:
```
**Do**:
...
5. Rewrite the assertions the removals and the new shape break: `internal/cli/base_formatter_test.go` (the `FormatTransition` subtests and `TestAllFormattersProduceConsistentTransitionOutput` at lines 129-150), `internal/cli/toon_formatter_test.go:318-325`, `internal/cli/format_test.go:207-211` and the empty-`CascadeResult` loop at `internal/cli/format_test.go:414-421`, `internal/cli/json_formatter_test.go:537`, `internal/cli/helpers_test.go:293-350`, `internal/cli/transition_test.go:389-407` and the cascade assertions beneath it, and the toon subtests in `internal/cli/cascade_formatter_test.go`. Toon assertions decode stdout with `decodeToonDoc` (`internal/cli/toon_decode_test.go`, added in Phase 1) and check row values; pretty assertions keep their golden strings unchanged.
...

**Acceptance Criteria**:
...
- [ ] Pretty output for `start`, `done`, `cancel` and `reopen` — the single line and the box-drawing tree — is byte-identical to before this task
- [ ] `grep -rn 'FormatTransition' internal/` returns nothing
...
```

**Proposed Text**:
```
**Do**:
...
5. Rewrite the assertions the removals and the new shape break: `internal/cli/base_formatter_test.go` (the `FormatTransition` subtests and `TestAllFormattersProduceConsistentTransitionOutput` at lines 129-150), `internal/cli/toon_formatter_test.go:318-325`, `internal/cli/format_test.go:207-211` and the empty-`CascadeResult` loop at `internal/cli/format_test.go:414-421`, `internal/cli/json_formatter_test.go:463` and `internal/cli/json_formatter_test.go:537`, the whole of `TestOutputTransitionOrCascade` at `internal/cli/helpers_test.go:293-408` (every one of its subtests calls the deleted helper), `internal/cli/transition_test.go:389-407` and the cascade assertions beneath it, and the toon subtests in `internal/cli/cascade_formatter_test.go`. Toon assertions decode stdout with `decodeToonDoc` (`internal/cli/toon_decode_test.go`, added in Phase 1) and check row values; pretty assertions keep their golden strings unchanged. Pretty has one site the deletion reaches: `internal/cli/pretty_formatter_test.go:335-342` calls `PrettyFormatter.FormatTransition`. Change the call to `f.FormatCascadeTransition(CascadeResult{TaskID: "tick-a3f2b7", OldStatus: "open", NewStatus: "in_progress"})` and leave the expected string `tick-a3f2b7: open → in_progress` exactly as it is — that method returns the same bytes when `Cascaded` is empty. The golden moves; it is not deleted.
...

**Acceptance Criteria**:
...
- [ ] Pretty output for `start`, `done`, `cancel` and `reopen` — the single line and the box-drawing tree — is byte-identical to before this task
- [ ] `internal/cli/pretty_formatter_test.go` still asserts the arrow line `tick-a3f2b7: open → in_progress` as a golden string, now through `FormatCascadeTransition`
- [ ] `grep -rn 'FormatTransition' internal/` returns nothing
...
```

**Resolution**: Pending
**Notes**:

---

### 2. The empty-`CascadeResult` edge case names the wrong formatter as the survivor

**Severity**: Minor
**Plan Reference**: Phase 2, task `free-text-round-trip-2-2` (Status commands emit the changed table in toon) — tick record `tick-ed0952`, Edge Cases
**Category**: Task Template Compliance — an edge case whose claim about the code is false
**Move**: settled
**Change Type**: update-task

**Problem**:
`internal/cli/format_test.go:409-422` loops `StubFormatter`, `ToonFormatter`, `PrettyFormatter` and `JSONFormatter` over an empty `CascadeResult` and asserts each returns `""`. The task's edge case tells the implementer that after this change "`StubFormatter` alone still returns `""`". Pretty also still returns `""`: `PrettyFormatter.FormatCascadeTransition` opens with `if result.TaskID == "" { return "" }` (`internal/cli/pretty_formatter.go:235-237`), and this task rewrites only the toon formatter's copy of that guard. JSON also still returns `""` at the end of this task — its early return is deleted by task `free-text-round-trip-2-3`, not here. An implementer rewriting that loop from the sentence as written drops pretty's branch along with toon's, and pretty's branch is the one of the four whose behaviour the phase promises not to change at all.

**Proposal**:
State which branches move and when. Toon's is the only expectation this task falsifies; pretty's holds permanently because its guard is untouched; JSON's holds until the next task replaces it with `{"changed": []}`. All three facts are already established elsewhere in the phase — the toon rewrite in this task's Do step 3, pretty's untouched guard in its Context, and JSON's in task `free-text-round-trip-2-3`'s Do step 3 — so the edge case only has to stop contradicting them.

**Current**:
```
**Edge Cases**:
...
- `format_test.go`'s "empty CascadeResult returns empty string" expectation no longer holds for toon — an empty result is a count-zero table, and `StubFormatter` alone still returns `""`
...
```

**Proposed Text**:
```
**Edge Cases**:
...
- `format_test.go`'s "empty CascadeResult returns empty string" expectation no longer holds for toon — an empty result is a count-zero table. The other three branches of that loop stay: `StubFormatter` and `PrettyFormatter` both still return `""`, pretty because its own `result.TaskID == ""` guard is untouched by this task, and `JSONFormatter` still returns `""` until task 3 replaces its early return with `{"changed": []}`
...
```

**Resolution**: Pending
**Notes**:

---

### 3. `--field tags.2` is specified as a section and as a bare value in the same task

**Severity**: Minor
**Plan Reference**: Phase 4, task `free-text-round-trip-4-6` (Positions narrow the list section they name) — tick record `tick-377527`
**Category**: Acceptance Criteria Quality — two criteria and two tests name an invocation the task's own rule routes elsewhere
**Move**: settled
**Change Type**: update-task

**Problem**:
The task's Do step 5 sends a lone `notes`, `tags` or `refs` name carrying one position to the bare value: `--field tags.2` prints the tag's own bytes and a newline. Two of its acceptance criteria then demand a document from that same invocation — "`--field tags.2` renders `tags[1]: <second tag>` — an inline list keeping its count and one item", and "A repeated position renders once", whose test spells the request `notes.2,notes.2` and asserts it "decodes to one row". A lone `tags.2` cannot both print a bare tag and decode as a one-element list, and `notes.2,notes.2` collapses to a single position, so on the task's own rule it is the note's text bare rather than a decodable section. Whichever way the implementer reads it, criteria in one list contradict each other and one of the two tests as written cannot pass.

Underneath the wording sits a real unanswered question: whether duplicate positions collapse before or after the count that decides bare-versus-document. Nothing in the task says.

**Proposal**:
The duplicate question is settled by the plan's own convention rather than left open. §9.1's lenience rule, quoted in task `free-text-round-trip-4-1` and already implemented there for names — "a repeated name collapses and counts once toward the one-versus-several split" — reads the same way for a repeated position: duplicates collapse first, so `notes.2,notes.2` is one position and takes the bare path, exactly as `--field title,title` takes it. The criteria and tests that mean the section form are then scoped to a multi-field selection, which is what the task's Outcome already states ("`tick show <id> --field description,notes.2` prints `notes[1]{index,text,created}:` with a single row … `--field notes.2` alone prints that note's text bare") and what its first criterion already spells out. Nothing about the behaviour changes; the criteria stop naming the invocation that produces the other one.

**Current**:
```
**Do**:
...
5. `internal/cli/show_fields.go` — extend `bareFieldValue`: a single name that is `notes`, `tags` or `refs` with exactly one selected position resolves to that item's value (a note's text, a tag, a ref); `children` and `blocked_by` never resolve bare, and a name with more than one position never resolves bare.

**Acceptance Criteria**:
...
- [ ] `--field tags.2` renders `tags[1]: <second tag>` — an inline list keeping its count and one item
...
- [ ] A repeated position renders once
...

**Tests**:
- `"it narrows the notes section to one position"` — decoded `notes` has one row
- `"it keeps the real index on a narrowed note"` — that row's `index` decodes as `2`
- `"it narrows an inline list to one item"` — decoded `tags` has one element equal to the second stored tag
...
- `"it collapses a repeated position"` — `notes.2,notes.2` decodes to one row
...
```

**Proposed Text**:
```
**Do**:
...
5. `internal/cli/show_fields.go` — extend `bareFieldValue`: a single name that is `notes`, `tags` or `refs` with exactly one selected position — counted after duplicate positions collapse, mirroring §9.1's rule that a repeated name counts once toward the one-versus-several split — resolves to that item's value (a note's text, a tag, a ref); `children` and `blocked_by` never resolve bare, and a name carrying two or more distinct positions never resolves bare.

**Acceptance Criteria**:
...
- [ ] `--field title,tags.2` renders `tags[1]: <second tag>` — an inline list keeping its count and one item
...
- [ ] `--field title,notes.2,notes.2` renders one row, and `--field notes.2,notes.2` alone collapses to one position and prints the note's text bare
...

**Tests**:
- `"it narrows the notes section to one position"` — `--field description,notes.2` decodes with `notes` carrying one row
- `"it keeps the real index on a narrowed note"` — that row's `index` decodes as `2`
- `"it narrows an inline list to one item"` — `--field title,tags.2` decodes with `tags` carrying one element equal to the second stored tag
...
- `"it collapses a repeated position"` — `--field title,notes.2,notes.2` decodes to one row, and `--field notes.2,notes.2` alone prints the note's text bare
...
```

**Resolution**: Pending
**Notes**:

---

## Coverage Notes (no finding)

Recorded so the checks that came back clean this cycle are visible and are not re-run from scratch in a later one. Cycles 1, 2 and 3's coverage notes still stand and are not repeated here.

- **All 38 task bodies re-diffed against their tick records this cycle** — every phase file section compared byte for byte against `tick show <id> --json`'s description. 38 of 38 match exactly; cycle 3's finding-1 class of divergence is gone. The `task_map` resolves all 44 records (6 phases + 38 tasks) and none is orphaned.
- **Cycle 3's three findings are applied in both records**: `free-text-round-trip-1-6` no longer tells the implementer to leave alone the samples it exists to fix; Phase 6's acceptance carries the retained count-zero header set as a named exception; `free-text-round-trip-6-1`'s `conformanceDoc.Command` field states its content and how the coverage guard consumes it. Traceability cycle 4's finding is applied too.
- **Dependency graph re-read from tick**: the same five edges, no cycles — the four README tasks to `free-text-round-trip-1-6` (`tick-a8523e`), and `free-text-round-trip-5-5` to `free-text-round-trip-4-6` (`tick-377527`). All 44 records carry priority 2, so creation order is the execution order end to end and every other cross-phase prerequisite sits earlier in it.
- **Code references verified this cycle that earlier cycles did not reach**: all ten `FormatTransition` call sites in the suite (finding 1); `TestOutputTransitionOrCascade`'s real extent, 293-408; `format_test.go`'s four-formatter empty-cascade loop at 409-422 (finding 2); `PrettyFormatter.FormatCascadeTransition`'s `TaskID == ""` guard and its `"%s: %s → %s"` head line; the three `ValidateFlags` call sites (`app.go:71`, `app.go:78`, `app.go:115`); `ValidateFlags`' indexed loop, its numeric-argument skip and its `TakesValue` value-skip; `create.go:283`, `update.go:415` and `update.go:420` with their `!fc.Quiet` guards; `outputMutationResult`'s quiet short-circuit ahead of any formatting; `Engine.Run`'s `for _, task := range tasks` shadowing loop with `Validate()` ahead of `CreateTask` and `FallbackTitle = "(untitled)"`; `task.TrimTitle`, `TrimDescription`, `TrimNoteText`, `FormatTimestamp`, `NormalizeID`, `ValidateRef` and `ValidateTag` all present; `buildTagsSection`, `buildRefsSection`, `buildStringListSection`, `buildDescriptionSection`, `buildEdgeSection`, `collectUpstreamEdges`, `collectDownstreamEdges` and `encodeToonSingleObject` all present at the cited lines; `ToonFormatter.FormatTaskDetail`'s seven-section order and its presence guards; `RunShow`'s current argument read and branch order.
- **`ValidateFlags`' third call site is cited as `app.go:113` in task `free-text-round-trip-5-1` and sits at `app.go:115`** — line 113 is the comment introducing it. Not flagged: the step names the site unambiguously as "the `qualifyCommand` site" and there is exactly one.
- **`json_formatter_test.go:55-60`, cited by task `free-text-round-trip-6-6`, covers the empty-slice half of a subtest running to line 67** — the nil-slice half at 62-66 carries the same `"[]"` comparison. Not flagged: the task's step 5 grep (`grep -rn '"\[\]"' internal/cli/*_test.go` returns nothing) is stated as the check and catches both halves of the one subtest.
- **Task `free-text-round-trip-4-2`'s "after `queryShowData` and `showDataToTaskDetail` and before the quiet branch" still requires moving a call the step does not mention moving** — `RunShow`'s quiet branch sits above `showDataToTaskDetail` today. Assessed in cycle 3 and re-assessed here as benign: `--quiet` with a selection is refused in task 1 before the store opens, so no observable behaviour depends on which side of the quiet branch the bare-value path lands.
- **CLAUDE.md re-checked against the whole plan**: the Formatter bullet is the only line the work falsifies and task `free-text-round-trip-2-2` carries its correction. The flag-validation bullet stays true when Phase 5 adds `flagScanLimit` beside `commandFlags` — `ValidateFlags` still rejects unknown flags and `commandFlags` is still the registry — which is the reading cycle 1 recorded when it added the Formatter line.
- **Phase 6's inventory partition still holds**: `commandFlags` carries exactly 21 keys after `init()`, matching the plan's 14 must-parse / 5 prose / 2 out-of-scope split with none left over. Phase 4 adds flags to `show` rather than a key and Phase 5's `flagScanLimit` is a separate map, so the partition survives the plan.
- **Task template compliance**: all 38 tasks carry Problem, Solution, Outcome, Do, Acceptance Criteria, Tests, Edge Cases, Context and Spec Reference. No task exceeds five Do steps except `free-text-round-trip-2-2` (six, the sixth the one-line CLAUDE.md correction cycle 1 added), assessed and left in cycles 2 and 3.
- **Vertical slicing and deliberate interim states re-checked**: `2-1`, `2-4` and `4-1` each ship a verifiable increment with their own unit-level tests and each names the task that closes the interim state. No horizontal slice anywhere in the plan — every task is one TDD cycle over one behaviour.

---
