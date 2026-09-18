# Review Tracking: Free Text Round Trip - Integrity

## Findings

### 1. Phase 4's ordering rationale says it is independent of Phase 2, and two of its tasks are not

**Severity**: Minor
**Plan Reference**: Phase 4 (Field selection on `show`), "Why this order" — `planning.md:94`
**Category**: Phase Structure — the phase's stated ordering guarantee is contradicted by its own tasks
**Move**: settled
**Change Type**: update-phase

**Problem**:
"Why this order" is the plan's account of what may be reordered and what may not, and it is the paragraph someone reads before scheduling a phase early or running two in parallel. Phase 4's says it "depends on nothing from Phases 2 and 3". Two of its tasks do.

Task `free-text-round-trip-4-3` gates the toon detail document's pieces field by field, and the list it gates includes the `changed` section — a section that exists only because Phase 2 task `free-text-round-trip-2-4` added it. Task `free-text-round-trip-4-5` restructures `PrettyFormatter.FormatTaskDetail` into header lines and blocks, and its first Do step is an instruction about Phase 2's work: "The cascade tail Phase 2 task `free-text-round-trip-2-4` appended to this function stays exactly as that task left it… It is not a group and must not take the `"\n\n"` join — that would put a blank line into `create` and `update` output that is not there today." That step exists because cycle 2 found the restructure silently dropping Phase 2's cascade tail from `create` and `update`.

So the sentence's reason is true — `show` carries no `changed` section, and nothing Phase 4 projects comes from Phase 2 — while its conclusion is not. Taken at face value it licenses running Phase 4 before Phase 2, which hands an implementer two instructions naming code that does not exist yet, and moves the restructure Phase 2's pretty step has to land into.

**Proposal**:
Keep the true half of the sentence and replace the false conclusion with what the plan's own tasks say. The plan settles it: the dependency is named in `free-text-round-trip-4-3`'s Do step 2 and `free-text-round-trip-4-5`'s Do step 1, and cycle 2's fix is the reason the second one is worded as it is. Nothing new is decided and no task changes — the rationale is brought level with the tasks it describes, so the ordering guarantee can be taken at face value.

**Current**:
```
**Why this order**: It is the case that started the work, and it is a projection of the document Phase 1 defines — the names it accepts are the names that document uses, and the bare form exists only because Phase 1 removed the wrapper. It depends on nothing from Phases 2 and 3, since `show` carries no `changed` section, so it follows once every document it could project is conformant.
```

**Proposed Text**:
```
**Why this order**: It is the case that started the work, and it is a projection of the document Phase 1 defines — the names it accepts are the names that document uses, and the bare form exists only because Phase 1 removed the wrapper. Nothing it projects comes from Phases 2 and 3, since `show` carries no `changed` section — but it follows Phase 2 rather than preceding it, because two of its formatter tasks build on what Phase 2 left: task `free-text-round-trip-4-3` gates the `changed` section task `free-text-round-trip-2-4` added to the toon detail document, and task `free-text-round-trip-4-5` restructures the pretty detail around the cascade tail that same task appended. It lands once every document it could project is conformant.
```

**Resolution**: Pending
**Notes**:

---

### 2. Task 6-6 closes on "no string-literal comparisons" while the same checklist keeps seven

**Severity**: Minor
**Plan Reference**: Phase 6, task `free-text-round-trip-6-6` (No toon or JSON assertion pins a full-output string) — tick record `tick-f9b5f3`, first acceptance criterion
**Category**: Acceptance Criteria Quality — two criteria in one checklist cannot both be ticked
**Move**: settled
**Change Type**: update-task

**Problem**:
This task's acceptance list is the last checklist in the plan — the one someone ticks to call the whole work done. Its first line reads "No toon or JSON test compares a whole rendered document against a string literal". Its fifth line reads "The retained text assertions are exactly the count-zero section headers, each beside a decoded assertion of the same empty section", and Do step 4 spells the retention out: keep `tasks[0]{…}`, `notes[0]{…}`, `changed[0]{…}`, `blocked_by[0]{…}`, `blocks[0]{…}`, `children[0]{…}` and `dep_tree[0]{…}`.

`FormatTaskList(nil)` renders exactly `tasks[0]{id,title,status,priority,type}:` and nothing else, so comparing it against that literal is a whole rendered document compared against a string literal. The two criteria are checked against the same code and cannot both be true. Whoever verifies the phase either ticks a box that is false, or deletes assertions Do step 4 and criterion five require — and the retained set is the considered part: this task's own Context works through why a decoder cannot recover a count-zero header's columns and §8 requires those columns to be there.

The same defect one level up was found in cycle 3 and fixed: Phase 6's acceptance criterion now carries the retained set explicitly. The task-level criterion the phase criterion points at was left as it was.

**Proposal**:
Carry the exception into the criterion, as the phase acceptance already does, and name the concrete collision so the reader sees what the clause is for. The plan settles every part of it: Do step 4 fixes the retained set, criterion five states the rule it obeys, and the phase-level criterion fixed in cycle 3 supplies the form. Nothing new is decided, and the criterion becomes true as stated.

**Current**:
```
- [ ] No toon or JSON test compares a whole rendered document against a string literal
```

**Proposed Text**:
```
- [ ] No toon or JSON test compares a whole rendered document against a string literal, apart from the count-zero section headers retained below — `FormatTaskList(nil)` compared against `tasks[0]{id,title,status,priority,type}:` is one of them
```

**Resolution**: Pending
**Notes**:

---

## Verification Notes (no finding)

Recorded so the checks that were run and came back clean are visible, and are not re-run from scratch next cycle.

- **Planning file and tick store are in sync.** All 38 task records under plan `tick-1fe74e` were extracted and compared field for field against the six `phase-N-tasks.md` files: zero mismatches. Cycle 3's divergence (a stray Do step on `tick-a8523e`) has not recurred, and cycle 4's and cycle 5-traceability's edits landed in both records.
- **Template compliance is complete.** Every one of the 38 tasks carries Problem, Solution, Outcome, Do, Acceptance Criteria, Tests, Edge Cases, Context and Spec Reference. Do steps run 4–6 (only `free-text-round-trip-2-2` at 6, from cycle 1's `CLAUDE.md` addition); acceptance criteria 7–15; named tests 4–20. No task states a criterion or test that asks for reasoning to be recorded in source. The four "update the doc comment" Do steps (`1-2`, `3-2`, `3-3`, `3-5`) each repair a comment the change would otherwise falsify, which is maintenance rather than directed commentary.
- **Code references were spot-checked against the live tree and hold.** `toon_formatter_test.go`'s six `task{…}` header assertions are at lines 88, 143, 369, 440, 470 and 602 exactly as `free-text-round-trip-1-1` states; `format_integration_test.go`'s four `task{` probes at 28, 304, 620 and 663 as `1-5` states; the ten `FormatTransition` call sites in the suite all fall inside the ranges `2-2` Do step 5 names; `formatFocusedDepTreeJSON` carries exactly the three `len(...)` guards `3-5` deletes; `TestCommandFlagsMatchHelp` does split `flagInfo.Name` on `", "` and match both long forms, which is what `4-1`'s single `--field, --fields` help entry relies on; `ValidateFlags` does reject a bare `--` today, which is what `5-1`'s "no existing invocation regresses" rests on; pretty's header-line and block order matches `4-5`'s mapping line for line. Residual drift is one or two lines at three sites (`buildRelatedSection` 296→298, `update.go` 377→378, `app.go` 113→115) and is not worth an edit.
- **Every phase acceptance criterion has an owning task**, and every task's work is claimed by a phase criterion. No orphan criteria, no unclaimed tasks.
- **Dependencies and ordering.** No explicit dependencies exist anywhere in the plan and every task sits at priority 2, so execution is the natural creation-date order the tick format's `reading.md` defines. Intra-phase order matches the order each phase's tasks must run in; the cross-phase dependencies (Phase 4 on Phase 1's document and Phase 2's formatter changes, Phase 5's fixture on Phase 4's `--field notes.1`, Phase 6 on all five) all run forward, so natural order already produces the correct sequence and no explicit edges are needed. No cycles are possible in an empty graph.
- **The conformance inventory partitions the live registry.** `commandFlags` carries 21 keys after `init()`; Phase 6's three declared sets account for 14 must-parse, 5 prose and 2 out-of-scope with none doubled and none missing. Phase 4 adds no command key and Phase 5's `--` is registered nowhere, so neither disturbs the guard.
- **JSON invariants Phase 6 task 6-5 asserts are already true of the code they will run against**: every list field in `jsonTaskDetail`, `toJSONRelated`, `toJSONDepTreeNodes` and `FormatStats`' `by_priority` is allocated with `make`, so none can unmarshal to `null`.
- **Cross-format asymmetries the plan knowingly carries were re-checked and are each stated where they land**: `--field tags` on a tag-less task (zero bytes in toon and pretty, `{"tags": []}` in JSON) is named in `free-text-round-trip-6-4`, which picks `parent,closed` for the zero-byte inventory entry because of it; pretty's `Type:     -` against toon's omission is named in `4-5`; filtered JSON's sorted key order against unfiltered struct order is named in `4-4` as the plan's own call on an open point.
