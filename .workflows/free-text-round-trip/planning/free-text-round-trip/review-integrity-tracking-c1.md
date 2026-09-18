# Review Tracking: Free Text Round Trip - Integrity

## Findings

### 1. The zero-byte exemption is a real JSON document, so nothing ever parses it

**Severity**: Important
**Plan Reference**: Phase 6, task `free-text-round-trip-6-4` (Field-selection documents and the two exemptions are covered)
**Category**: Acceptance Criteria Quality — a criterion that does not hold on the branch it claims to cover
**Move**: settled
**Change Type**: update-task

**Problem**:
Phase 6's inventory is meant to be the complete account of what the tool emits — every document decoded, and the two things that are not documents declared with the reason they are exempt. Task 4 picks `tick show <id> --field tags` on a tag-less task as its zero-byte exemption. That output is zero bytes in toon and in pretty, but in JSON it is `{"tags": []}`: task `free-text-round-trip-4-4` makes `tags` one of the keys JSON always carries when selected, `[]` and never null and never absent. So the inventory declares "not a document" about output that is a document in one of the three formats, the JSON driver of task 5 skips the entry on that basis, and a filtered JSON document the tool really produces is never parsed by anything. The exemption §3.1 states holds "in every format" is then proved by an example that holds in two of them.

A selection that prints nothing in all three formats exists and the plan already uses it: `--field parent,closed` on a task carrying neither. `parent` and `closed` are omitted-when-absent in toon (§5.2's presence rules), carry `omitempty` in JSON, and sit behind pretty's non-empty guards — task `free-text-round-trip-4-4` already tests exactly that invocation as its "prints nothing when no selected key survives" case.

**Proposal**:
Swap the zero-byte exemption's example from `--field tags` on a tag-less task to `--field parent,closed` on a task carrying neither, and state in the entry why the tags case is not one — so the exemption is true in every format, and the format-dependence that makes `tags` a document in JSON is recorded where a later reader meets it. The choice is determined by the plan's own content, not taken here: §3.1 requires the exemption to hold "in every format" and task `free-text-round-trip-4-4` already fixes both the JSON presence rule that rules `tags` out and the `parent,closed` invocation that satisfies it. The bare-value assertion in the same task already spells out its four-format check, so the zero-byte one is brought to the same standard.

**Current**:
```
**Do**:
...
2. `internal/cli/conformance_test.go` — add three `NotADocument` entries: `show --field description` (a bare value, §9.2), `show --field tags` on a tag-less task (zero bytes, §9.6), and `show --quiet --field title` (refused, §9.8). Each reason names why the output is not a document.
...
4. `internal/cli/conformance_test.go` — extend that test: `tick show <id> --field tags` on a tag-less task produces zero bytes and exit 0, and `tick show <id> --quiet --field title` exits non-zero with zero bytes on stdout.

**Acceptance Criteria**:
...
- [ ] A selection whose every name prints nothing produces zero bytes and exits zero
...

**Tests**:
...
- `"it prints zero bytes for a selection that renders nothing"` — empty stdout, exit 0
...

**Edge Cases**:
...
- A selection whose every name prints nothing produces zero bytes — that is nothing rather than an empty document, so there is no document to decode and no `{}` to parse
...
```

**Proposed Text**:
```
**Do**:
...
2. `internal/cli/conformance_test.go` — add three `NotADocument` entries: `show --field description` (a bare value, §9.2), `show --field parent,closed` on a task carrying neither (zero bytes, §9.6), and `show --quiet --field title` (refused, §9.8). Each reason names why the output is not a document. The zero-byte entry names `parent` and `closed` because those two are the fields every format omits when the task does not carry them; `--field tags` on a tag-less task is zero bytes in toon and pretty but a `{"tags": []}` document in JSON, so it is a document rather than an exemption.
...
4. `internal/cli/conformance_test.go` — extend that test: `tick show <id> --field parent,closed` on a task carrying neither produces zero bytes and exit 0 with no format flag, under `--toon`, under `--pretty` and under `--json`, and `tick show <id> --quiet --field title` exits non-zero with zero bytes on stdout.

**Acceptance Criteria**:
...
- [ ] A selection whose every name prints nothing produces zero bytes and exits zero, identically with no format flag, under `--toon`, under `--pretty` and under `--json`
...

**Tests**:
...
- `"it prints zero bytes for a selection that renders nothing"` — `--field parent,closed` on a task carrying neither gives empty stdout and exit 0
- `"it prints zero bytes in every format for a selection that renders nothing"` — the four invocations all give empty stdout and exit 0
...

**Edge Cases**:
...
- A selection whose every name prints nothing produces zero bytes — that is nothing rather than an empty document, so there is no document to decode and no `{}` to parse
- The exempt selection has to print nothing in *every* format: `parent` and `closed` are omitted when absent in toon, JSON and pretty alike, while `--field tags` on a tag-less task is zero bytes in toon and pretty and a `{"tags": []}` document in JSON, because JSON always carries a selected `tags` as `[]` (task `free-text-round-trip-4-4`)
...
```

**Resolution**: Fixed
**Notes**: Applied verbatim to `free-text-round-trip-6-4` (tick-ae3897) — zero-byte exemption swapped to `--field parent,closed`, four-format criterion and test added, and the tags-is-a-document-in-JSON reason recorded in Do step 2 and Edge Cases. Landed in both the task detail file and the tick store.

---

### 2. The README `tick show` sample is specified two ways in one task

**Severity**: Minor
**Plan Reference**: Phase 1, task `free-text-round-trip-1-6` (README samples match real output)
**Category**: Task Self-Containment — the task's own instruction and its verified target disagree
**Change Type**: update-task
**Move**: settled

**Problem**:
The README's `tick show` sample is supposed to be byte-for-byte what the tool prints. The task tells the implementer to capture output from a task carrying "type, parent (rendered as `parent`), tags, refs, one note and a multi-line description plus one blocker and no children", and then hands them a target shape in Context — verified against the project's toon-go version and stated to decode — that carries no `parent` line at all. Following the instruction produces a sample the target shape says is wrong; following the target shape means silently dropping a field the instruction asked for. Either way the implementer decides for themselves what the published sample shows.

**Proposal**:
Drop `parent` from the capture instruction and pin the head fields in the acceptance criterion, so the instruction and the target shape name the same document. The target shape is what settles it rather than the other way round: it is the form recorded as verified to decode, it matches the field order `buildTaskSection` actually emits (`id`, `title`, `status`, `priority`, `type`, `parent` when non-empty, `created`, `updated`, `closed` when non-nil), it matches the sample the README carries today, and adding a parent would put a task ID in the sample that appears nowhere else in the document.

**Current**:
```
**Do**:
1. Build the binary to a scratch path and, in a throwaway `.tick` project, produce real `--toon` output for a task carrying type, parent (rendered as `parent`), tags, refs, one note and a multi-line description plus one blocker and no children; copy it into `README.md:429-450` verbatim. The target shape is in Context.
...

**Acceptance Criteria**:
...
- [ ] The sample's head is top-level named fields with no `task` key and no `[1]`
...
```

**Proposed Text**:
```
**Do**:
1. Build the binary to a scratch path and, in a throwaway `.tick` project, produce real `--toon` output for a task carrying type, tags, refs, one note and a multi-line description plus one blocker and no children; copy it into `README.md:429-450` verbatim. The target shape is in Context and the captured output must match it field for field — the sample task has no parent, so the document carries no `parent` line.
...

**Acceptance Criteria**:
...
- [ ] The sample's head is the top-level named fields `id`, `title`, `status`, `priority`, `type`, `created`, `updated` — no `task` key, no `[1]` and no `parent` line, matching the target shape in Context
...
```

**Resolution**: Fixed
**Notes**: Applied verbatim to `free-text-round-trip-1-6` (tick-a8523e) — `parent` dropped from the capture instruction and the head fields pinned in the acceptance criterion, in both the task detail file and the tick store.

---

### 3. CLAUDE.md keeps naming a Formatter method the work deletes

**Severity**: Minor
**Plan Reference**: Phase 2, task `free-text-round-trip-2-2` (Status commands emit the changed table in toon)
**Category**: Scope and Granularity — a one-line consequence of the task's own deletion, unallocated
**Move**: settled
**Change Type**: add-to-task

**Problem**:
`CLAUDE.md:49` tells everyone working in this repository that `Formatter` carries "FormatTaskList, FormatTaskDetail, FormatTransition, FormatCascadeTransition, FormatDepChange, FormatDepTree, FormatStats, FormatMessage, FormatRemoval". This task deletes `FormatTransition` from the interface, from `baseFormatter`, from `StubFormatter` and from the JSON formatter. From that commit onward the file that every agent reads before touching this codebase names a method that does not exist — the same defect §12.1 refuses to ship in the README, in the document that is read more often and earlier. The task's own acceptance already greps `internal/` for the name; the file stating the interface is one directory up and goes unchecked.

**Proposal**:
Add the CLAUDE.md line to this task rather than to the phase's README task, because the claim is about the interface rather than about output samples and the edit belongs in the same commit as the deletion that falsifies it. Editing CLAUDE.md alongside a formatter change is the repository's existing convention — `e14b22fd docs: update CLAUDE.md and README.md with formatter and transition details` and `0a1d7030 docs: update CLAUDE.md and README.md for auto-cascade-parent-status feature` are both this pattern — so no new practice is introduced, and the edit is the removal of one name from one list. Nothing else in CLAUDE.md is falsified by this work: the `commandFlags` and `ValidateFlags` description stays true when Phase 5 adds `flagScanLimit` beside it, and no other bullet names a deleted symbol.

**Current**:
```
**Do**:
...
5. Rewrite the assertions the removals and the new shape break: `internal/cli/base_formatter_test.go` (the `FormatTransition` subtests and `TestAllFormattersProduceConsistentTransitionOutput` at lines 129-150), `internal/cli/toon_formatter_test.go:318-325`, `internal/cli/format_test.go:207-211` and the empty-`CascadeResult` loop at `internal/cli/format_test.go:414-421`, `internal/cli/json_formatter_test.go:537`, `internal/cli/helpers_test.go:293-350`, `internal/cli/transition_test.go:389-407` and the cascade assertions beneath it, and the toon subtests in `internal/cli/cascade_formatter_test.go`. Toon assertions decode stdout with `decodeToonDoc` (`internal/cli/toon_decode_test.go`, added in Phase 1) and check row values; pretty assertions keep their golden strings unchanged.

**Acceptance Criteria**:
...
- [ ] `grep -rn 'FormatTransition' internal/` returns nothing
...
```

**Proposed Text**:
```
**Do**:
...
5. Rewrite the assertions the removals and the new shape break: `internal/cli/base_formatter_test.go` (the `FormatTransition` subtests and `TestAllFormattersProduceConsistentTransitionOutput` at lines 129-150), `internal/cli/toon_formatter_test.go:318-325`, `internal/cli/format_test.go:207-211` and the empty-`CascadeResult` loop at `internal/cli/format_test.go:414-421`, `internal/cli/json_formatter_test.go:537`, `internal/cli/helpers_test.go:293-350`, `internal/cli/transition_test.go:389-407` and the cascade assertions beneath it, and the toon subtests in `internal/cli/cascade_formatter_test.go`. Toon assertions decode stdout with `decodeToonDoc` (`internal/cli/toon_decode_test.go`, added in Phase 1) and check row values; pretty assertions keep their golden strings unchanged.
6. `CLAUDE.md:49` — remove `FormatTransition` from the Formatter interface's method list, leaving the eight that remain and the rest of the bullet as it is.

**Acceptance Criteria**:
...
- [ ] `grep -rn 'FormatTransition' internal/` returns nothing
- [ ] `CLAUDE.md`'s Formatter bullet lists the eight remaining methods and no longer names `FormatTransition`
...
```

**Resolution**: Fixed
**Notes**: Applied verbatim to `free-text-round-trip-2-2` (tick-ed0952) — Do step 6 and the matching acceptance criterion added, in both the task detail file and the tick store.

---

## Coverage Notes (no finding)

Recorded so the checks that came back clean are visible and not re-run from scratch in a later cycle.

- **Task template compliance**: all 38 implementation tasks carry Problem, Solution, Outcome, Do, Acceptance Criteria, Tests, Edge Cases, Context and Spec Reference. No Do section exceeds five steps. No task directs a source comment, and no acceptance criterion asks for rationale to be recorded anywhere.
- **Code references spot-checked against the tree and correct**: `buildTaskSection`'s field order and the `strings.Replace` hand-edit (`toon_formatter.go:260-295`), `FormatTaskDetail`'s section order, the notes query at `show.go:146`, `"show": {}` at `flags.go:72`, `FlagDef{TakesValue: bool}`'s literal form, `note remove`'s `index %d out of range: task has %d note(s)` at `note.go:128`, `Fprintln(stdout, id)` in `outputMutationResult`, `ValidateFlags`' scan loop, `parseArgs`/`qualifyCommand`/`applyGlobalFlag`, the four `buildCascadeResult` call sites, `RunTransition`'s `!fc.Quiet` guard, `runFullDepTree`'s early return, `PrettyFormatter.FormatMessage` returning the message unchanged, `formatFullDepTree` returning `""` for zero roots, pretty's exact detail labels and padding (`Title:    `, `Status:   `, `Type:     ` with `typeOrDash`), `jsonTaskDetail`'s presence rules and its `make`-allocated `notes`, and every README line reference the plan cites (list sample 396-401, show sample 429-451, dep-tree summary 307, transition 471-486, cascade 499-518 with `(unchanged)` at 504 and 517, global flags 564-576, create 86-110, note 245-257, show 167-174).
- **Phase 6's inventory partition is satisfiable**: `commandFlags` holds exactly 21 keys after `init()` — the 14 the must-parse inventory covers, the 5 prose commands and the 2 out-of-scope ones the plan declares. No key is left over, so `TestConformanceInventoryCoversEveryCommand` can pass on the day it is written.
- **Dependencies and ordering**: no task depends on a later task. Every cross-phase prerequisite either sits earlier in creation order (which tick's natural ordering already enforces) or carries an explicit edge — the four README tasks on `free-text-round-trip-1-6` for `TestREADMEToonSamplesDecode`, and `free-text-round-trip-5-5` on `free-text-round-trip-4-6` for the `--field notes.1` read path. Uniform priority 2 changes no execution order here, since authoring order is the intended order.
- **Deliberate interim states are not horizontal slices**: `2-1` (merge logic, no output change), `2-4` (detail section rendered, handlers still pass nil) and `4-1` (flag parsed, rendering unchanged) each carry their own unit-level tests and are each one TDD cycle. Each names the task that closes the interim state. Not flagged.
- **Pretty's byte-identity claims hold arithmetically**: `2-4`'s single `Fprintln(detail + "\n" + block)` reproduces today's `Fprintln(detail)` + `Fprintln(block)` exactly, and `4-5`'s group-join restructure reproduces today's header-group-then-blocks layout including the absent trailing newline.
- **`grep -rn 'FormatTransition' internal/` is a valid acceptance check**: `FormatCascadeTransition` and `TestAllFormattersProduceConsistentTransitionOutput` do not contain the substring, so the grep cannot pass falsely.
- **Notes index truthfulness**: `task_notes` is an ordinary rowid table and the cache is rebuilt from JSONL in order, so `ORDER BY created ASC, rowid ASC` does make the emitted index the one `note remove` takes. The only gap is a stored `created` that goes backwards, which no write path produces.

---
