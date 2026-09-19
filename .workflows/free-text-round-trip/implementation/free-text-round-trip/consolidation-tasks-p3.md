# Consolidation Tasks: Free Text Round Trip (Phase 3)

## Task 1: The Full Dep-Tree Edge List Covers Every Participant
placement: phase 3
severity: behaviour

**Problem**: The emptied full-graph dependency document can contradict itself. `BuildFullDepTree` derives the edge list by walking down from roots (`internal/cli/dep_tree_graph.go:132-149`) and derives `chains` and `blocked` from every participant (`:151-160`). When no participant is a root — every dependency edge sits inside a cycle, or every blocker is a dangling ID — the document reports `dep_tree[0]{from,to}:` beside `chains: 1` and `blocked: 2`. Before this phase that branch printed prose, so no counts surfaced and there was nothing to contradict; task 3-4's emptied document newly puts the two halves in one document that disagrees with itself, and task 3-4's own test pins that as correct (`internal/cli/dep_tree_test.go:235-250`). An agent reads "no dependency edges" beside "two blocked tasks" and cannot reconcile them: it concludes the project is unblocked and starts a task that is blocked, or reports a graph the counts say exists and the edges say does not. Both producing states are ones the project expects — `internal/doctor/dependency_cycle.go` and `internal/doctor/orphaned_dependency.go` exist for them, and `tick migrate` imports from external tools — though neither is reachable through `tick dep add`, which rejects cycles. The same split sits in `formatFullDepTreeJSON` (`internal/cli/json_formatter.go:342-350`).

**Solution**: Make the full-graph edge list cover every participant rather than only those a root reaches. After the existing walk from the roots, seed a further walk from any participant not yet emitted, so a cycle's or an orphan's edges appear and the edge list agrees with `chains` and `blocked`. The root walk is unchanged for root-reachable tasks, so diamond duplication (`README.md:281`, pinned by `internal/cli/dep_tree_graph_test.go:615`) is preserved. This changes what the command outputs on those two states, so task 3-4's cycle subtest and any JSON twin are re-pinned as part of the work. The terminal view is out of scope — §4.1 keeps it on `result.Message` for this branch.

Settled against the specification: §8 was amended this pass to fix that the edge list covers every participant. The derivation, recorded in the corrigendum, is that §8's stated purpose requires the counts to be meaningful, so zeroing them would destroy the fact that tasks are blocked, while extending the edge list preserves both halves and makes the document self-consistent — which is §1's bar, output an agent can read without a rule learned outside it.

**Outcome**: Every full-graph dependency document agrees with itself: an empty edge list means nothing is blocked, and a reader can act on the counts without cross-checking them.

**Do**:
- `internal/cli/format.go:213-228` — add `Unrooted []DepTreeNode` to `DepTreeResult` beside `Roots`, holding the trees seeded from participants no root reaches. `Roots` keeps its current meaning, so pretty (`internal/cli/pretty_formatter.go:322-338`) and `Message` need no change.
- `internal/cli/dep_tree_graph.go:117-188` — in `BuildFullDepTree`, record participants in first-seen order alongside the existing `participants` map (for each task carrying `BlockedBy`: its own ID, then each blocker ID, tasks in input order), and after the root loop collect into a set every node ID the root trees emitted, via a small recursive helper over `DepTreeNode`.
- Same function — for each ordered participant the set does not hold whose `blocks[id]` is non-empty, append to `Unrooted` a `DepTreeNode` with `Children: walkDownstream(id, blocks, taskIdx, make(map[string]bool))` and `Task: toDepTreeTask(taskIdx[id])`, or a `DepTreeTask` carrying only `ID` when the participant is a blocker ID no task record matches; add every node of each new tree to the set before the next participant. Participants that block nothing need no seed — each is reached as the `to` of one of its blockers.
- Same function — leave `Roots`, `Message`, `BlockedCount` and `countChains` exactly as they are, and compute `longest` over `Roots` and `Unrooted` together so the depth matches the edge list the document prints.
- `internal/cli/toon_formatter.go` `formatFullDepTree` and `internal/cli/json_formatter.go:342-350` `formatFullDepTreeJSON` — build the `dep_tree` edge rows, and the JSON `roots` array, from `result.Roots` followed by `result.Unrooted`. The JSON document keeps its `{mode, roots, chains, longest, blocked}` keys; no new key.

**Acceptance Criteria**:
- [ ] The `dep_tree` section carries zero rows only when `chains`, `longest` and `blocked` are all zero, and the JSON `roots` array is empty only under the same condition.
- [ ] Two tasks blocking each other emit rows `tick-aaa111,tick-bbb222` and `tick-bbb222,tick-aaa111` beside `chains: 1`, `blocked: 2` and a non-zero `longest` (2 for that fixture — the emitted chain is A → B → A, the cycle closing on the repeated leaf the existing ancestors guard produces).
- [ ] A project whose only blocker is an ID no task carries emits one row from that ID to the blocked task beside `chains: 1`, `blocked: 1`; in JSON that node carries its `id` with an empty `title` and `status`.
- [ ] Root-reachable output is unchanged: the A → B → C fixture emits the same two rows and the same counts as today, and the diamond still duplicates the shared node — `internal/cli/dep_tree_graph_test.go:615-643` stays green unedited.
- [ ] Pretty is unchanged on every input: its tree rendering still reads `result.Roots` alone, `len(result.Roots)` and `Message` keep today's values, and the zero-roots branch still prints `No dependencies found.` (`internal/cli/dep_tree_test.go:183-207` and `internal/cli/dep_tree_graph_test.go:599-613` stay green unedited).
- [ ] `BuildFullDepTree` terminates on every cycle shape and seeds each participant at most once.

**Tests**:
- `"it emits the cycle's edges beside its counts"` — re-pins `internal/cli/dep_tree_test.go:235-250`, which currently asserts an empty `dep_tree` for the two-task cycle: now the two rows above with `chains: 1`, `longest: 2`, `blocked: 2`.
- `"it emits an edge from a blocker no task carries"` — a single task blocked by a dangling ID, through `runToonCommand`.
- `"it fills the JSON roots when no participant is a root"` — the same cycle fixture through `runDepTreeJSON`: `roots` non-empty, counts unchanged.
- `"it still prints the no-dependencies sentence in pretty when every participant is blocked"` — the cycle fixture through `runDepTree`.
- `"it still renders the populated graph"` (`internal/cli/dep_tree_test.go:252-271`) and `"it preserves acyclic diamond duplication after cycle guard addition"` stay green untouched.

## Task 2: One Test Holds Every README Sample Against Real Output
placement: phase 3
severity: complexity

**Problem**: Nothing preserves the verification each README sample was authored against. `internal/cli/readme_samples_test.go:140-149` proves the anchored blocks parse as TOON; it never compares them to what the formatter emits. Each of the five samples corrected across phases 1 to 3 was hand-diffed against captured binary output once, at authoring time, and nothing re-checks it. Rename a field or shift a value — `chains` to `chain_count`, a changed count — and every sample still parses, the suite stays green, and the README goes on teaching a shape the tool no longer emits. That is noticed only when a user or agent writes a parser against the documented sample and it fails at runtime: the exact defect §12.1 exists to prevent, reintroduced in the document people read to learn the tool. Three phases have each appended an anchor plus a negative string marker (`:19-29`, `:186-190`, `:265-269`) — a pattern that grows one constant per sample and still checks only that a specific wrong shape is absent. Phase 6 keeps this file decode-only by design, so no later task closes it.

**Solution**: One fixture-driven test that seeds a project per documented sample, renders the command through the real formatter, and compares the result to the anchored README block. Every negative marker constant then falls out — a sample compared against real output cannot contain a section header the formatter no longer emits. The cost is the reason this is a boundary judgement rather than an omission: each sample needs a seeded project matching its narrative and IDs (`tick-a1b2`, `tick-c3d4`, `tick-f3e4`), and the surface spans phase 1 (`show`, `list`), phase 2 (transition, cascade) and phase 3 (dep tree), plus whatever phases 4 and 5 add. It is a pure test addition: no production code changes and no existing test semantics change.

**Outcome**: A formatter change that moves any documented shape fails the suite at the point of the change, rather than at the point a reader trusts the README.

**Do**:
- `internal/cli/readme_samples_test.go` — add a fixture table, one entry per documented sample: the fence's info string (`""` or `json`), the first line of its body once any `$ tick` prompt line is stripped, its occurrence among blocks sharing that first line, the format flag, the command args, and the tasks to seed.
- Add the lookup and comparison helper beside the existing ones: select the entry's block from the ordered `readmeFences(t)` list (the existing `readmeToonBlocks` map cannot serve — the two pretty `list` blocks share a first line, hence the occurrence field), seed with `setupTickProjectWithTasks`, run the args through `App.Run` into a `bytes.Buffer` as `runToonCommand` does, and compare `strings.TrimRight(stdout, "\n")` with the block byte-for-byte. A block the lookup cannot find is `t.Fatalf`, never a skip.
- Fill the table with the thirteen rendered blocks (`README.md` line ranges in this tree): `:289-294` and `:302-309` (`dep tree`, pretty and toon); `:397-401`, `:409-413`, `:425-427`, `:459-461` (`list`, toon and pretty, two seeds); `:431-451` (`show tick-a1b2`, toon); `:474-476`, `:484-485`, `:493-503` (`start tick-a1b2`, toon, pretty and JSON); `:518-521`, `:529-533` (`done tick-a1b2`, toon and pretty); `:545-552` (`list`, JSON). The fenced blocks that are not formatter output stay out: the `.gitignore` snippet (`:575-578`), the global-flag list (`:582-590`), the unknown-flag error (`:594-597`) and every `bash` block.
- Where captured output and the README block differ, correct the block to the captured bytes and keep the comparison exact — §12.1 owns the README's accuracy. `README.md:545-552` is known to differ: `jsonTaskListItem.Type` carries no `omitempty` (`internal/cli/json_formatter.go:17-23`), so the formatter always emits a `type` key the sample omits.
- Delete what the comparison subsumes: the five decode-only subtests (`internal/cli/readme_samples_test.go:154-184`), the two negative-marker subtests (`:186-190` and `:265-269`) and their constants `readmeObjectHeaderMarker` and `readmeUnchangedMarker`. The anchor constants stay (the table and the retained subtests use them), as do the missing-anchor guard (`:219-223`) and the three decoded-property subtests (`:192-200`, `:202-217`, `:225-237`).

**Acceptance Criteria**:
- [ ] Every fixture entry's rendered output equals its README block byte-for-byte, trailing newline aside; no covered sample is checked by decoding, substring or prefix alone.
- [ ] All thirteen blocks above are in the table, and a block whose first line or occurrence no longer matches fails the test rather than being skipped.
- [ ] Renaming a documented key or shifting a documented value in any of the three formatters fails this test — verified locally by making one such change, watching the failure, and reverting it.
- [ ] `readmeObjectHeaderMarker` and `readmeUnchangedMarker` and the subtests reading them are gone, and nothing replaces them with another one-off string marker.
- [ ] No production code changes: the diff touches `internal/cli/readme_samples_test.go` and `README.md` only.
- [ ] `go test ./...` green, `go vet ./...` clean, `gofmt -l ./internal` empty.

**Tests**:
- `"it reproduces the README show sample"` — `show tick-a1b2` in its seeded project equals `README.md:431-451`.
- `"it reproduces the README list samples"` — all four list blocks, toon and pretty, each against its own seed.
- `"it reproduces the README transition samples"` — `start tick-a1b2` in toon, pretty and JSON, and `done tick-a1b2` cascading to its child in toon and pretty.
- `"it reproduces the README dep tree samples"` — `dep tree` in toon and pretty.
- `"it fails when a documented sample has no matching README block"` — the existing missing-anchor fixture through the new lookup.

## Task 3: Corrections
placement: phase 3
severity: corrections

**Problem**: Two small residues of this phase's work. First, a superseded test guard: `internal/cli/dep_tree_test.go:273-294`, the subtest "it outputs dep tree for project with dependencies", was written to guard a handler branch task 3-4 deleted, and task 3-4 added a decoded-edge subtest covering the same branch at `:252-271`. Its only surviving assertion is that the output is not the string `No dependencies found.`, so a regression printing nothing at all passes it, and its comment at `:288-290` tells a reader `FormatDepTree` is an unimplemented stub — false since the dep-tree work completed. Second, a join drift: `internal/cli/toon_formatter.go:187` and `:203` join TOON sections with a raw `strings.Join(sections, "\n\n")`, while `FormatStats` at `:119` and `FormatTaskDetail` at `:96` use the file's `joinToonSections` helper, which drops the empty string `encodeToonFields` returns on an encoder error. A field-encoding failure in a dependency document would emit a bare blank line with nothing on one side of it — a malformed document, which is the defect class this work exists to remove. It bites for real on the next conditionally-present field added to a dependency document; the task-detail document already carries three.

**Solution**: Two edits.

- `internal/cli/dep_tree_test.go:273-294` — delete the subtest and its comment. The decoded-edge subtest above it covers the branch, and `TestRunDepTree` already pins the terminal full-graph bytes at `:183-207`. The identically-shaped focused sibling at `:296-319` is the same case and is taken with it.
- `internal/cli/toon_formatter.go:187` and `:203` — replace both `strings.Join(sections, "\n\n")` calls with `joinToonSections(sections)`. Behaviour-preserving on every input the suite exercises, since `buildEdgeSection` never returns an empty string and `encodeToonFields` only does so on an encoder error.

**Outcome**: No test in the dependency suite passes while checking nothing, and every TOON document in the formatter degrades the same way when a field cannot be encoded.

**Do**:
- `internal/cli/dep_tree_test.go` — delete the subtest `"it outputs dep tree for project with dependencies"` (`:273-294`, taking its comment at `:288-290` with it) and its sibling `"it outputs focused view for task with dependencies"` (`:296-319`). Find them by name, not by line: task 1's re-pins move these numbers. The file's `strings` import stays in use (`:71`, `:335`), so no import edit follows.
- `internal/cli/toon_formatter.go` — replace the `strings.Join(sections, "\n\n")` return in `formatFullDepTree` (currently `:186`) and in `formatFocusedDepTree` (currently `:202`) with `joinToonSections(sections)`. The set is complete as measured: `grep -rn 'strings\.Join(sections' internal/ cmd/` → 2 hits, both in this file. `strings` stays imported — `joinToonSections` (`:278-286`) joins its kept sections with it.

**Acceptance Criteria**:
- [ ] `grep -rn 'strings\.Join(sections' internal/ cmd/` returns nothing: every TOON document assembled from a `sections` slice is joined by `joinToonSections`.
- [ ] Neither deleted subtest remains, and no test in `internal/cli/dep_tree_test.go` asserts only that output differs from `No dependencies found.` or `No dependencies.`.
- [ ] Output is byte-identical on every case the suite exercises, in all three formats and both dep-tree modes — the encoder-error path is the only input whose rendering changes, and no test reaches it.
- [ ] `go test ./...` green with no test edits beyond the two deletions, `go vet ./...` clean, `gofmt -l ./internal` empty.

**Tests**:
- No new test. These stay green unedited and carry the deleted subtests' branches: `"it still renders the populated graph"` (`internal/cli/dep_tree_test.go:252-271`, the full-graph handler by decoded edges), `"it still prints the no-dependencies sentence in pretty for an empty project"` and its sibling (`:183-207`, the pretty full-graph bytes), and `"it decodes dep tree output for a task with both directions"` (`internal/cli/toon_decode_test.go:389-401`, the populated focused document end to end).
- `TestToonFormatDepTree` and `TestPrettyFormatDepTree` stay green unedited, pinning that the helper swap changes no rendered bytes.
