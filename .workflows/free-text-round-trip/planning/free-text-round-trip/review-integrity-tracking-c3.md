# Review Tracking: Free Text Round Trip - Integrity

## Findings

### 1. The executed copy of the Phase 1 README task tells the implementer to leave alone the samples it exists to fix

**Severity**: Important
**Plan Reference**: Phase 1, task `free-text-round-trip-1-6` (README samples match real output) — tick record `tick-a8523e`
**Category**: Task Self-Containment — the copy of the task that gets executed contradicts itself
**Change Type**: remove-from-task
**Move**: settled

**Problem**:
The README is what someone reads to learn the tool, and this task is what makes its `tick show` and `tick list` samples match what the tool actually prints. The copy of the task the executor works from — the tick record — carries a fifth instruction the planning record does not: "Confirm the scope boundary: `grep -n '^\$ tick stats' README.md` returns nothing and no `tick stats` sample is added; the `tick show`, `tick list`, transition and cascade samples corrected in Phases 1 and 2 are untouched."

The `tick show` and `tick list` samples are exactly what steps 1 and 2 of this task rewrite, and Phase 2 has not happened yet. An implementer working the list top to bottom rewrites two samples and is then told to confirm those same samples were left alone. The sentence is a verbatim copy of step 5 of `free-text-round-trip-3-6`, where it is correct — there it names the dep-tree task's boundary, by which point Phases 1 and 2 have landed and the `tick stats` question is live. Landed here it either costs the implementer the time to work out which instruction to believe, or it is taken at its word and the task's actual work is skipped. Underneath the contradiction is the wider problem: the planning file and the tick record disagree about what this task says, and only one of them is the one that gets executed.

**Proposal**:
Delete the stray step from the tick record, leaving the four steps the planning record holds. The planning record settles it: `phase-1-tasks.md`'s copy of this task ends at step 4, `phase-3-tasks.md`'s copy of `free-text-round-trip-3-6` carries the same sentence as its own step 5, and this task's step 3 already states the boundary that belongs to it — the dep-tree, transition and cascade samples are left to the phases that change them. Every other task in the plan was compared field for field against its tick record this cycle and matched, so this is the one place the two records differ.

**Current** (the Do block as the tick record `tick-a8523e` holds it):
```
**Do**:
1. Build the binary to a scratch path and, in a throwaway `.tick` project, produce real `--toon` output for a task carrying type, tags, refs, one note and a multi-line description plus one blocker and no children; copy it into `README.md:429-450` verbatim. The target shape is in Context and the captured output must match it field for field — the sample task has no parent, so the document carries no `parent` line.
2. `README.md:396-401` — correct the empty-type row of the `tick list` sample to `tick-d5c6,Update docs,open,3,""`.
3. Leave the dep-tree sample (`README.md:302-307`) and the transition and cascade samples (`README.md:473-504`) alone — they are corrected by the phases that change them.
4. `internal/cli/readme_samples_test.go` (new file) — add `TestREADMEToonSamplesDecode`: locate `README.md` via `testutil.FindRepoRoot(t)`, collect every fenced block with an empty info string, strip a leading `$ tick …` prompt line from each, index the blocks by their first remaining line, then for each anchor in `{"tasks[3]{id,title,status,priority,type}:", "tasks[2]{id,title,status,priority,type}:", "id: tick-a1b2"}` fail if no block carries it and fail if `toon.DecodeString` rejects that block.
5. Confirm the scope boundary: `grep -n '^\$ tick stats' README.md` returns nothing and no `tick stats` sample is added; the `tick show`, `tick list`, transition and cascade samples corrected in Phases 1 and 2 are untouched.
```

**Proposed Text**:
```
**Do**:
1. Build the binary to a scratch path and, in a throwaway `.tick` project, produce real `--toon` output for a task carrying type, tags, refs, one note and a multi-line description plus one blocker and no children; copy it into `README.md:429-450` verbatim. The target shape is in Context and the captured output must match it field for field — the sample task has no parent, so the document carries no `parent` line.
2. `README.md:396-401` — correct the empty-type row of the `tick list` sample to `tick-d5c6,Update docs,open,3,""`.
3. Leave the dep-tree sample (`README.md:302-307`) and the transition and cascade samples (`README.md:473-504`) alone — they are corrected by the phases that change them.
4. `internal/cli/readme_samples_test.go` (new file) — add `TestREADMEToonSamplesDecode`: locate `README.md` via `testutil.FindRepoRoot(t)`, collect every fenced block with an empty info string, strip a leading `$ tick …` prompt line from each, index the blocks by their first remaining line, then for each anchor in `{"tasks[3]{id,title,status,priority,type}:", "tasks[2]{id,title,status,priority,type}:", "id: tick-a1b2"}` fail if no block carries it and fail if `toon.DecodeString` rejects that block.
```

**Resolution**: Fixed
**Notes**: Stray Do step 5 removed from tick-a8523e; the stored record now matches the planning file's four steps byte-for-byte (verified by diff).

---

### 2. Phase 6 closes on "no pinned strings" while the plan deliberately keeps seven

**Severity**: Minor
**Plan Reference**: Phase 6 (Conformance verification across the output inventory), phase acceptance criteria
**Category**: Acceptance Criteria Quality — a phase-close criterion the phase's own tasks contradict
**Change Type**: update-phase
**Move**: settled

**Problem**:
Phase 6's acceptance is the checklist someone ticks to call the work done, and one line reads "No toon or JSON test asserts against a pinned full-output string". The phase's own tasks keep seven of them. Task `free-text-round-trip-6-1` states it outright — "The count-zero `tasks` header's column schema is still asserted as text" — and `free-text-round-trip-6-6` lists the retained set and the reason: a decoder collapses `notes[0]{index,text,created}:` and `notes[0]:` to the same empty list, so no decoded assertion can check the columns a count-zero header carries, and §8 requires those columns to be there. `FormatTaskList(nil)` compared against the literal `tasks[0]{id,title,status,priority,type}:` is a whole rendered document compared to a string, which is what the criterion says will not exist.

The retention is the considered part of the plan — `free-text-round-trip-6-6`'s Context works through the collision and takes a reading. The phase criterion is the part that hides it. Whoever verifies the phase either ticks a box that is false, or spends the time rediscovering the exception in a task they were not reading.

**Proposal**:
Carry the exception into the criterion, in the words `free-text-round-trip-6-6` already uses for it. The plan's own record settles both that the exception exists and exactly which seven headers it covers; nothing new is decided, and the criterion becomes true as stated so a phase-close check can be taken at face value. The phase Goal's summary sentence stands as the phase's thrust — the criterion is the tickable check and is the line that has to be accurate.

**Current**:
```
- [ ] No toon or JSON test asserts against a pinned full-output string
```

**Proposed Text**:
```
- [ ] No toon or JSON test asserts against a pinned full-output string, with one retained set recorded in task `free-text-round-trip-6-6`: the count-zero section headers — `tasks[0]{…}`, `notes[0]{…}`, `changed[0]{…}`, `blocked_by[0]{…}`, `blocks[0]{…}`, `children[0]{…}` and `dep_tree[0]{…}` — stay as text assertions, because a decoder collapses them to the same empty list a bare `name[0]:` produces, and each sits beside a decoded assertion of the same empty section
```

**Resolution**: Fixed
**Notes**: Applied verbatim to Phase 6's acceptance list in planning.md — the criterion now names the seven retained count-zero headers in task 6-6's words.

---

### 3. The conformance inventory's `Command` field has no stated content, and the completeness guard keys on it

**Severity**: Minor
**Plan Reference**: Phase 6, task `free-text-round-trip-6-1` (Task-list documents decode through one conformance table), Do step 1
**Category**: Task Self-Containment — a field consumed by two later tasks whose content is never fixed
**Change Type**: update-task
**Move**: settled

**Problem**:
The guard that stops the next command joining the CLI with no document declared for it is the point of the inventory — task `free-text-round-trip-6-3` places every key of `commandFlags` in exactly one of must-parse, prose and out-of-scope, and fails the suite when a key lands in none. The must-parse set can only be read off `conformanceDoc.Command`. The task that declares the struct explains what `Setup` returns and what `NotADocument` holds, and says nothing about `Command`.

`commandFlags`' keys are fully qualified and argument-free — `dep tree`, `note add`, `note remove` — while the arguments the driver runs come from `Setup`. An entry written `Command: "dep tree <id>"`, or `Command: "dep"`, or carrying the whole invocation, leaves the guard unable to match, and the first thing the implementer sees is the completeness check reporting commands as undeclared when every one of them has an entry. The one field the guard reads is the one the plan does not describe.

**Proposal**:
State the field's content where the struct is declared. The plan settles the spelling twice over: `free-text-round-trip-6-3` reads the live `commandFlags` map, whose keys after `init()` are `init`, `create`, `update`, `list`, `show`, `start`, `done`, `cancel`, `reopen`, `dep add`, `dep remove`, `dep tree`, `note add`, `note remove`, `remove`, `stats`, `doctor`, `rebuild`, `migrate`, `ready` and `blocked`; and `free-text-round-trip-6-5` declares `jsonTopLevelIsArray(command string)` true for `list`, `ready` and `blocked`, which are those keys verbatim. No new decision is taken and no criterion is added — a misspelled `Command` already fails `TestConformanceInventoryCoversEveryCommand`, which is exactly the failure this sentence prevents the implementer from having to diagnose.

**Current**:
```
1. `internal/cli/conformance_test.go` (new file) — declare `type conformanceDoc struct { Name string; Command string; Setup func(t *testing.T) (dir string, args []string); NotADocument string }` and `var conformanceDocs = []conformanceDoc{}`. `Setup` seeds a project and returns its directory plus the arguments that follow `tick`; `NotADocument` holds the reason an entry's output is not a document, and an entry carrying one carries no `Setup`.
```

**Proposed Text**:
```
1. `internal/cli/conformance_test.go` (new file) — declare `type conformanceDoc struct { Name string; Command string; Setup func(t *testing.T) (dir string, args []string); NotADocument string }` and `var conformanceDocs = []conformanceDoc{}`. `Command` is the fully-qualified command name exactly as `commandFlags` spells it — `list`, `show`, `dep tree`, `note add` — carrying no arguments and no flags, because task `free-text-round-trip-6-3`'s coverage guard matches it against that map's keys; `Setup` seeds a project and returns its directory plus the arguments that follow `tick`; `NotADocument` holds the reason an entry's output is not a document, and an entry carrying one carries no `Setup`.
```

**Resolution**: Fixed
**Notes**: Applied verbatim to `free-text-round-trip-6-1` (tick-25fabf) in both the task detail file and the tick store. All 38 stored tasks were then diffed against their detail files — no mismatches remain.

---

## Coverage Notes (no finding)

Recorded so the checks that came back clean this cycle are visible and are not re-run from scratch in a later one. Cycles 1 and 2's coverage notes still stand and are not repeated here.

- **Every task was compared field for field against its tick record this cycle** — all 38 task detail bodies diffed against `tick show <id> --json`'s description. 37 matched byte for byte; `free-text-round-trip-1-6` is finding 1 and is the only divergence. The `task_map` resolves all 44 records (6 phases + 38 tasks) and no record is orphaned.
- **Cycle 2's three findings are applied in full and in both records**: `free-text-round-trip-4-5` carries the cascade-tail Do sentence, its acceptance criterion and its edge case; Phase 5's acceptance carries the `tick create --description -- x` exception; `free-text-round-trip-6-3`'s Problem reads "eight". Traceability cycle 3's finding is applied too — `free-text-round-trip-5-5` carries the tags and refs fixture constants, the five-constant Do steps, the two extra acceptance criteria and the `ValidateRef`/`ValidateTag` edge case.
- **Code references re-verified against the tree this cycle**, covering the sites cycle 2 did not: `encodeToonSingleObject`'s `strings.Replace(s, name+"[1]", name, 1)` and its two callers (`FormatStats`, `formatFullDepTree`); `toonStatsSummary` and `toonDepTreeSummary`; `buildTagsSection`/`buildRefsSection`/`buildStringListSection`/`buildDescriptionSection` all present at the lines the tasks cite; `ToonFormatter.FormatDepTree`'s `Target != nil` dispatch ahead of its `Message` guard and `formatFocusedDepTree`'s hand-built `%s  %s (%s)` line; `runFullDepTree`'s `len(result.Roots) == 0` early return at `dep_tree.go:37-40` and its single `FormatMessage` call; `PrettyFormatter.formatFullDepTree` returning `""` for zero roots and `PrettyFormatter.FormatMessage` returning the message unchanged, so Phase 3 task 4's swap is byte-identical; `JSONFormatter.FormatDepTree`'s message branch marshalling the same `jsonMessage` the handler's `FormatMessage` does, which is what makes task 3-4's "JSON unchanged" criterion true; `outputTransitionOrCascade`'s `cr == nil || len(cr.Cascaded) == 0` branch, which is why task 2-1 can build the result unconditionally without moving any output; `outputMutationResult`'s current signature; `ValidateFlags`' indexed `for i := 0; i < len(args); i++` loop, which is what `flagScanLimit` stops; `parseArgs`' single pass and `qualifyCommand`'s two-level slicing; `RunShow`'s `args[0]` read and its quiet branch sitting *above* `showDataToTaskDetail` (Phase 4 task 2's ordering requires the detail to be built first, which its Do step states); `PrettyFormatter.FormatTaskDetail`'s exact label set and order, matching task 4-5's header/block split; `task_notes` carrying a rowid and `show.go:146`'s `ORDER BY created ASC`; and `Engine.Run`'s `for _, task := range tasks` shadowing loop with `Validate()` ahead of `CreateTask`.
- **Test-site line references still resolve**: `format_integration_test.go`'s four `task{` probes (28, 304, 620, 663), its `tasks[` probes (239, 554) and its `"[]"` comparison (576), and `json_formatter_test.go`'s `"[]"` comparisons (58-59, 64-65). Phase 6 task 6-6's two stated greps match the probes that will still exist when that task runs.
- **Dependency graph re-read from tick**: five edges, no cycles — the four README tasks (`tick-477df7`, `tick-b3a485`, `tick-4de057`, `tick-d35d1b`) to `free-text-round-trip-1-6` (`tick-a8523e`), and `free-text-round-trip-5-5` (`tick-3a59e1`) to `free-text-round-trip-4-6` (`tick-377527`). Every other prerequisite sits earlier in creation order, which tick's natural ordering enforces; all 44 records carry priority 2, so creation order is the execution order end to end.
- **Cross-phase deferrals all land in a receiving phase's acceptance**: task 1-5's deferral of `create`/`update` end-to-end decode → Phase 2's "create and update emit a single decodable document"; task 1-6's transition and dep-tree sample deferrals → Phase 2's and Phase 3's README criteria; task 4-8's `--` deferral → Phase 5's "README and the command help text present `--`"; task 4-8's suite-coverage deferral → Phase 6's "Both field-selection document forms … are decoded in the suite". No deferral is held only in a task's edge cases.
- **Phase 5's argument arithmetic re-checked on the case the plan records as its one regression**: `tick create --description -- x` → `parseArgs` appends `--description` to `rest`, takes `--` as the marker, counts `x` as one literal; `splitLiteralArgs` hands `ValidateFlags` only `["--description"]`, which passes because the flag is registered as value-taking; `parseCreateArgs` then reports `--description requires a value`. The exception is real, is the one the plan records, and is now carried in Phase 5's acceptance.
- **Task template compliance**: all 38 tasks carry Problem, Solution, Outcome, Do, Acceptance Criteria, Tests, Edge Cases, Context and Spec Reference. No task exceeds five Do steps except `free-text-round-trip-2-2` (six, the sixth being the one-line `CLAUDE.md` correction cycle 1 added), which cycle 2 assessed and left.
- **No task directs a source comment beyond the allowance**: the four doc-comment steps (`encodeToonSection` in 1-2, `FormatDepTree` in 3-2 and 3-3, the JSON dep-tree comments in 3-5) each correct a comment the same commit falsifies, and `flagScanLimit`'s one-line doc in 5-3 states a non-obvious invariant. None references a task, a phase or what a test covers.
- **No task edits another work unit's artifact**: tasks 1-3 and 2-6 both name a specification owed a correction and both state explicitly that the corrigendum route carries it, "never by this task".

---
