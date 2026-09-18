# Review Tracking: Free Text Round Trip - Integrity

## Findings

### 1. Pretty's restructure deletes the cascade tree from `create` and `update`

**Severity**: Important
**Plan Reference**: Phase 4, task `free-text-round-trip-4-5` (Pretty renders the filtered document)
**Category**: Task Self-Containment — the task rewrites a function whose current contents it does not fully describe
**Move**: settled
**Change Type**: update-task

**Problem**:
A terminal user running `tick create --parent <done task>` or `tick update <id> --parent <other>` sees the task record followed by the box-drawing cascade tree that tells them which other tasks moved. Phase 2 task `free-text-round-trip-2-4` put that tree inside `PrettyFormatter.FormatTaskDetail`, appended after the detail body as `"\n"` plus each `detail.Changes.Blocks` entry. This task then rewrites the same function from scratch and describes its whole contents as two things: a group of header lines and a list of blocks. The cascade tail is in neither list. An implementer working from the Do steps produces a function that drops it, and the tree vanishes from `create` and `update` — the one piece of pretty output §4.1 names explicitly as unchanged by this work.

The task's own criterion ("byte-identical to before this task") and the golden test named in its Tests list would catch the loss, but only after the implementer has written code that contradicts the instruction they were given. At that point they have to decide where the tail goes and with what separator, and the answer is not in this task: `"\n"` rather than the `"\n\n"` this task joins groups with, because two `Fprintln` calls in today's handlers produced a single newline between the record and the block. That decision sits in `free-text-round-trip-2-4` alone. The sibling task that does the same job in toon — `free-text-round-trip-4-3` — names the `changed` section explicitly and states why it never appears in a filtered document; the pretty task is the one that leaves it out.

**Proposal**:
Name the cascade tail in Do step 1, with the separator and the reason it sits outside the group join, and pin it in one acceptance criterion and one edge case. Everything stated is already fixed by the plan: `free-text-round-trip-2-4` fixes the append and its `"\n"` separator and proves the byte arithmetic; `free-text-round-trip-4-1` fixes that `--field` is `show`'s flag alone, so a filtered render never carries `Changes`; and `free-text-round-trip-4-3` is the shape this brings the pretty task into line with. No new decision is taken — the tail keeps exactly the form Phase 2 left it in.

**Current**:
```
**Do**:
1. `internal/cli/pretty_formatter.go` — restructure `FormatTaskDetail` to collect a `[]string` of header lines (`ID`, `Title`, `Status`, `Priority`, `Type`, `Tags`, `Parent`, `Created`, `Updated`, `Closed` in today's order and with today's padding) and a `[]string` of blocks (`Blocked by`, `Children`, `Refs`, `Notes`, `Description` in today's order), then join the header group with `"\n"`, and join that group and each non-empty block with `"\n\n"`.
...

**Acceptance Criteria**:
- [ ] Unfiltered pretty output for `show`, `create`, `update`, `note add` and `note remove` is byte-identical to before this task, including blank-line spacing and column alignment
...

**Edge Cases**:
- Unfiltered pretty stays byte-identical: today's output is one header group followed by blocks separated by `"\n\n"`, so the group-join restructure must reproduce it exactly, including the absence of a trailing newline
...
```

**Proposed Text**:
```
**Do**:
1. `internal/cli/pretty_formatter.go` — restructure `FormatTaskDetail` to collect a `[]string` of header lines (`ID`, `Title`, `Status`, `Priority`, `Type`, `Tags`, `Parent`, `Created`, `Updated`, `Closed` in today's order and with today's padding) and a `[]string` of blocks (`Blocked by`, `Children`, `Refs`, `Notes`, `Description` in today's order), then join the header group with `"\n"`, and join that group and each non-empty block with `"\n\n"`. The cascade tail Phase 2 task `free-text-round-trip-2-4` appended to this function stays exactly as that task left it: when `detail.Changes != nil && len(detail.Changes.Blocks) > 0`, `"\n"` followed by each block rendered through `f.FormatCascadeTransition(block)` joined by `"\n"`, appended to the joined groups rather than added to either list. It is not a group and must not take the `"\n\n"` join — that would put a blank line into `create` and `update` output that is not there today.
...

**Acceptance Criteria**:
- [ ] Unfiltered pretty output for `show`, `create`, `update`, `note add` and `note remove` is byte-identical to before this task, including blank-line spacing and column alignment
- [ ] `create` and `update` still print their cascade tree, appended after the joined groups with the single `"\n"` separator Phase 2 gave it
...

**Edge Cases**:
- Unfiltered pretty stays byte-identical: today's output is one header group followed by blocks separated by `"\n\n"`, so the group-join restructure must reproduce it exactly, including the absence of a trailing newline
- The cascade tail is neither a header line nor a block: it is appended after the joined groups with a single `"\n"`, and a filtered render never carries one, because `--field` is `show`'s flag alone and `show` builds its detail with `Changes` nil
...
```

**Resolution**: Fixed
**Notes**: Applied verbatim to `free-text-round-trip-4-5` (tick-86b409) — cascade tail named in Do step 1 with its separator, plus one acceptance criterion and one edge case, in both the task detail file and the tick store.

---

### 2. Phase 5's acceptance promises no behaviour change, and the plan knows of one

**Severity**: Minor
**Plan Reference**: Phase 5 (Dash-leading free text and the whitespace invariant), phase acceptance criteria
**Category**: Acceptance Criteria Quality — a phase-close criterion the plan's own tasks contradict
**Move**: settled
**Change Type**: update-phase

**Problem**:
Phase 5's acceptance is the checklist someone ticks to call the phase done, and one of its lines reads "Every invocation that works today produces identical output and exit status". Task `free-text-round-trip-5-1` records that this is not quite true: `tick create --description -- x` stores a two-dash description today and, once `--` becomes the end-of-flags marker, reports `--description requires a value` instead. The plan surfaced that deliberately and judged it an invocation no caller could have meant — but it lives only in a task's edge-case list. Whoever verifies the phase either ticks a box that is false, or spends the time to rediscover the exception in a task they were not reading. The exception is the honest part of the plan; the phase criterion is the part that hides it.

**Proposal**:
Carry the exception into the criterion, in the words task `free-text-round-trip-5-1` already uses for it. The plan's own record settles both that the exception exists and what it is; nothing new is decided, and the criterion becomes true as stated so a phase-close check can be taken at face value.

**Current**:
```
- [ ] Every invocation that works today produces identical output and exit status
```

**Proposed Text**:
```
- [ ] Every invocation that works today produces identical output and exit status, with one accepted exception recorded in task `free-text-round-trip-5-1`: `tick create --description -- x` now reports `--description requires a value`, because a `--` sitting where a value-taking flag's value belongs becomes the marker
```

**Resolution**: Pending
**Notes**:

---

### 3. The conformance inventory's own task miscounts what is left to cover

**Severity**: Minor
**Plan Reference**: Phase 6, task `free-text-round-trip-6-3` (Mutation and status documents complete the toon inventory), Problem statement
**Category**: Task Template Compliance — a Problem statement whose own arithmetic does not hold
**Move**: settled
**Change Type**: update-task

**Problem**:
This task's whole subject is that the inventory must be an exhaustive account of what the tool emits — its Solution closes it with a guard that fails the suite when a command is declared nowhere. Its Problem statement opens "The nine remaining commands in §3.1's must-parse table" and then lists eight: `create`, `update`, `note add`, `note remove`, `start`, `done`, `cancel`, `reopen`. A reader checking the inventory against §3.1's fourteen — three covered in task `free-text-round-trip-6-1`, three in `free-text-round-trip-6-2`, these eight here — finds the count short by one and has to work out for themselves whether a command was dropped or the number is wrong. On the one task in the plan whose purpose is completeness, a miscount is the doubt it exists to remove.

**Proposal**:
Correct the count to eight. §3.1's table holds fourteen must-parse commands; `free-text-round-trip-6-1` takes `list`, `ready` and `blocked`, `free-text-round-trip-6-2` takes `stats`, `dep tree` and `show`, and the eight named here are the remainder. The plan's own three tasks fix the arithmetic; nothing else in the task changes.

**Current**:
```
**Problem**: The nine remaining commands in §3.1's must-parse table — `create`, `update`, `note add`, `note remove`, `start`, `done`, `cancel` and `reopen` — produce the documents Phase 2 reshaped, and none is in the inventory.
```

**Proposed Text**:
```
**Problem**: The eight remaining commands in §3.1's must-parse table — `create`, `update`, `note add`, `note remove`, `start`, `done`, `cancel` and `reopen` — produce the documents Phase 2 reshaped, and none is in the inventory.
```

**Resolution**: Pending
**Notes**:

---

## Coverage Notes (no finding)

Recorded so the checks that came back clean this cycle are visible and are not re-run from scratch in a later one. Cycle 1's coverage notes still stand and are not repeated here.

- **Cycle 1's three findings are applied in full**: `free-text-round-trip-6-4` carries the `--field parent,closed` exemption with its four-format criterion and the tags-is-a-document-in-JSON reason; `free-text-round-trip-1-6` no longer asks for a `parent` in the README capture and pins the head fields; `free-text-round-trip-2-2` carries the `CLAUDE.md:49` Do step and its criterion. All three are present in both the task detail files and the tick store.
- **Traceability cycle 2's two findings are applied**: `free-text-round-trip-2-6`'s Context carries the bounded corrigendum note plus §12.2 in its Spec Reference, and `free-text-round-trip-6-3` carries the four cascading status entries with the corrected reason.
- **Code references re-verified against the tree this cycle**: `PrettyFormatter.FormatTaskDetail`'s exact label padding and its header-then-`"\n\n"`-blocks layout (`Updated` carries no trailing newline, `Closed` is prefixed with one), `PrettyFormatter.FormatCascadeTransition` returning the bare arrow line when `Cascaded` is empty and writing `"\n\nCascaded:"` otherwise, `outputMutationResult` and `outputTransitionOrCascade` in `helpers.go`, `buildCascadeResult`'s current five-parameter signature and its four call sites (`transition.go:43`, `create.go:242`, `update.go:316`, `update.go:378`), `RunTransition`'s `if len(c) > 0` guard, `create.go`'s `parentCascadeResult` and `update.go`'s `r6CascadeResult`/`r3CascadeResult` all being assigned whenever their branch fires (so Phase 2 task 2's by-value passing cannot deref nil), `ToonFormatter.FormatDepTree` and `JSONFormatter.FormatDepTree` both dispatching on `Target != nil` *before* their `Message` guard (so Phase 3 task 3's deletion of the focused early return is sufficient and task 3-2's `FormatDepTree(DepTreeResult{})` reaches the full-graph path), `formatFocusedDepTreeJSON`'s three `len(...)` guards, `jsonTaskDetail`'s always-present `tags`/`refs`/`notes` all `make`-allocated, the notes query's `ORDER BY created ASC` at `show.go:146`, `"show": {}` at `flags.go:72`, and `TestCommandFlagsMatchHelp`'s `strings.SplitSeq(fi.Name, ", ")` — which is what lets Phase 4 task 1 register both spellings behind one `flagInfo`.
- **`commandFlags` still holds exactly 21 keys after `init()`** — `init`, `create`, `update`, `list`, `show`, `start`, `done`, `cancel`, `reopen`, `dep add`, `dep remove`, `dep tree`, `note add`, `note remove`, `remove`, `stats`, `doctor`, `rebuild`, `migrate`, `ready`, `blocked` — matching the plan's 14 / 5 / 2 partition with none left over. Phase 4 adds flags to `show` rather than a key, and Phase 5's `flagScanLimit` is a separate map, so the partition survives the whole plan.
- **Section order is consistent across every task that touches it**: head, `blocked_by`, `children`, `tags`, `refs`, `notes`, `changed`, `description`. Phase 2 task 4 inserts `changed` between `notes` and `description`, Phase 4 task 3 gates the same list in the same order, and Phase 1 task 6's README target shape matches. No task contradicts another on placement.
- **The README anchor list has no first-line collisions across phases**: `tasks[3]{…}` and `tasks[2]{…}` (Phase 1), `id: tick-a1b2` (Phase 1), `changed[1]{…}` and `changed[2]{…}` (Phase 2), `dep_tree[2]{from,to}:` (Phase 3), and Phase 4's filtered document, whose first line is the notes header — distinct from the full `show` sample, which is indexed on `id: tick-a1b2`. Phase 4's two bare-value samples and Phase 5's prose are correctly left unanchored.
- **Phase 6 task 6's two greps do not collide with the count-zero headers the plan retains**: every retained `tasks[0]{…}` / `notes[0]{…}` / `changed[0]{…}` / `blocked_by[0]{…}` / `blocks[0]{…}` / `dep_tree[0]{…}` comparison named across Phases 1 to 3 is a formatter-level equality check, while the grep targets `Contains(stdout, …)` probes. The end-to-end criteria that mention those headers (`free-text-round-trip-2-5`'s `changed[0]` on a parentless `create`, for instance) are backed by decoded-value tests in the same task's Tests list, not by stdout substring matches.
- **Dependencies re-read from the tick store**: the four README tasks (`tick-477df7`, `tick-b3a485`, `tick-4de057`, `tick-d35d1b`) each carry an explicit edge to `free-text-round-trip-1-6` (`tick-a8523e`), and `free-text-round-trip-5-5` (`tick-3a59e1`) carries one to `free-text-round-trip-4-6` (`tick-377527`). No cycles. Every other cross-phase prerequisite sits earlier in creation order, which tick's natural ordering enforces. Phase 6's five later tasks all extend the inventory task 1 creates and follow it in natural order, so no edge is owed there.
- **Phase 5's argument-boundary arithmetic holds on the awkward cases**: `tick -- create "x"` (marker before the subcommand, subcommand uncounted), `tick note add <id> -- "- text"` (`handleNote` slices one leading argument, trailing count stays valid), and `tick -- note add x` (count exceeds the post-`qualifyCommand` slice, which is exactly what `splitLiteralArgs`' clamp is for). Task 1 names all three.
- **Deliberate interim states re-checked and still sound**: `free-text-round-trip-2-1` changes no output because `outputTransitionOrCascade` still branches on `len(cr.Cascaded)`; `free-text-round-trip-2-4` renders the section while handlers pass nil; `free-text-round-trip-4-1` parses a selection that changes nothing. Each names the task that closes it and carries its own unit-level tests.
- **Do-step counts**: every task holds five Do steps or fewer except `free-text-round-trip-2-2`, which has six — the sixth being the one-line `CLAUDE.md` edit the cycle 1 review added. Not flagged: it is a single-line documentation correction in the commit that falsifies the line, not a second implementation step.

---
