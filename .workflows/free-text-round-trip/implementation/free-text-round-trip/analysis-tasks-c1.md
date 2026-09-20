# Analysis Tasks: Free Text Round Trip (Cycle 1)

## Task 1: The Published Note Index Addresses The Note `note remove` Deletes

severity: high
sources: architecture

**Problem**: The `index` column §6.3 added exists to give a note a handle an agent can pass to `note remove`, and the two sides compute the position by different rules in different modules. `queryShowData` reads notes `ORDER BY created ASC, rowid ASC` (`internal/cli/show.go:169`); `RunNoteRemove` indexes `tasks[i].Notes`, the JSONL slice (`internal/cli/note.go:124-134`). `rowid` follows JSONL order on every rebuild, so the tiebreak makes the two agree whenever the stamps are non-decreasing — the case the equal-timestamp test covers — but `created ASC` is the dominant key, so a task whose stored notes are out of chronological order publishes an index that names a different note than the one removal deletes. Reproduced against the built binary: a task whose two JSONL notes carry descending stamps renders `1,"SECOND in file, earlier stamp"` / `2,"FIRST in file, later stamp"`, and `tick note remove <id> 1` deletes "FIRST in file" and leaves "SECOND in file" — silently, exit 0. Out-of-order stamps need nothing exotic: `tasks.jsonl` is the hand-editable source of truth, and clock skew between two machines appending to one committed file produces them. The same ordering feeds `--field notes.N` (`internal/cli/show_fields.go:282-295`) and `ValidatePositions`, so the narrowed section and the out-of-range error inherit the mismatch.

**Solution**: Order the notes query by `rowid ASC` alone in `queryShowData`, so the published index, the `notes.N` selector and `note remove` address the same position by construction rather than by the stamps happening to agree. `rowid` is the JSONL position, which is what removal counts, and the equal-timestamp case the current tiebreak was added for is subsumed. Pin it: a task whose notes carry descending stamps renders them in stored order, and removing the published index removes that row.

**Outcome**: The index an agent reads out of `tick show` is the index `tick note remove` consumes, on every task the tool can hold.

**Do**:
- In `queryShowData` (`internal/cli/show.go:168-171`), order the notes query by the JSONL position alone: `SELECT text, created FROM task_notes WHERE task_id = ? ORDER BY rowid ASC`. Nothing else in the query changes.
- Add a subtest to `TestNoteIndex` (`internal/cli/note_test.go:573`) seeding one task through `setupTickProjectWithTasks` whose two stored notes carry descending `Created` stamps — first in the slice with the later stamp — and assert the toon `notes` rows come back in stored order, `index` 1 naming the first note in the file.
- In the same fixture, run `note remove <id> 1` through `runNote` and assert with `notesTextsOf(readPersistedTasks(t, tickDir), id)` that the note published as index 1 is the one gone, exactly as `"it matches the index note remove accepts"` (`:576`) does for the ascending case.
- Assert `runShow(t, dir, id, "--toon", "--field", "notes.1")` on that same task narrows to the first stored note, so the `notes.N` selector and `ValidatePositions` address the row the index column names.
- Leave `"it keeps insertion order for notes sharing a created timestamp"` (`note_test.go:618`) standing unedited — `rowid ASC` alone subsumes the tiebreak it pins.

**Acceptance Criteria**:
- [ ] The notes query in `queryShowData` orders by `rowid ASC` and carries no `created` term.
- [ ] On a task whose stored notes carry descending stamps, `tick show <id> --toon` lists them in stored order, `index` 1 naming the first note in `tasks.jsonl`.
- [ ] On that same task, `tick note remove <id> N` removes the note published at row N, for each N the document carries.
- [ ] `--field notes.1` on that task returns the first stored note, and a position past the last note still fails as out of range.
- [ ] Notes sharing one `created` stamp still render in insertion order.
- [ ] Ordinary ascending-stamp output is unchanged: `go test ./...` green with no existing test edited.

**Tests**:
- `"it renders notes in stored order when created stamps run backwards"`
- `"it removes the note the published index names when stamps run backwards"`
- `"it narrows to the first stored note when stamps run backwards"`
- `"it keeps insertion order for notes sharing a created timestamp"` (existing, stays green unedited)

## Task 2: The Two Completed Specifications Carry §12.2's Amendments

severity: high
sources: standards

**Problem**: §12.2 decides both documents are amended rather than left to supersession — "Both documents carry a point that is plainly and load-bearingly wrong, so both are amended" — and neither has been touched. `git log` on both returns only `9d5c4aae chore(workflows): reorganize workflow files into per-unit directories`, and neither carries a Corrigenda entry. `.workflows/v1/specification/tick-core/specification.md:693` still states "Long text fields get their own unstructured sections" as numbered principle 3 and prints the indented raw description block as its worked example (`:714`); `.workflows/auto-cascade-parent-status/specification/auto-cascade-parent-status/specification.md:146` still fixes the arrow-and-`(auto)` lines as the machine-readable cascade form and still describes a `FormatTransition` method no longer on the Formatter interface. Both stay live in the knowledge base at full confidence, so the next agent designing tick output is handed the shape §6.2 and §7.2 removed, and rebuilds the defect — noticed only when that output ships malformed again, which is the history §11 opens with. This is the only §12 deliverable outstanding; §12.1's README corrections are complete and pinned by `readme_samples_test.go`.

**Solution**: Run the corrigendum route on both documents before the work unit closes, honouring the two bounds §12.2 sets. On `tick-core`: amend principle 3 and its worked description block, and stop there — the `task{…}` and `stats{…}` headers in the same passage are left to supersession. On `auto-cascade-parent-status`: replace the arrow-and-`(auto)` toon cascade form, and leave standing its requirement that unchanged terminal children be shown alongside a cascade (§7.6). Each amendment presented and confirmed, the wrong claim replaced in place, a dated corrigendum entry recording what the document used to claim and what is true instead, and the document re-indexed — the re-index being the point of the route, since a file-only edit leaves the superseded claim served as validated context. No code change, and no test. The route is a session step rather than task work — `phase-1-tasks.md:191` and `review-traceability-tracking-c2.md:21-30` put these amendments "never by this task" — and it is demonstrably live in this phase: this work unit's own specification has taken seven corrigenda during implementation.

## Task 3: The Full-Graph JSON Document Names Its Key For What It Holds

severity: medium
sources: standards, architecture

**Problem**: `tick dep tree --json` returns the widened full-graph coverage under a key that still claims rootness. `formatFullDepTreeJSON` feeds `result.fullGraphTrees()` — `Roots` concatenated with `Unrooted` — into the field keyed `roots` (`internal/cli/json_formatter.go:341`, `:388`). Verified against a built binary: a two-task cycle returns `"roots"` holding Alpha, which is blocked by Beta; a task blocked by a missing ID returns `"roots"` holding `{"id":"tick-ghost","title":"","status":""}`. A consumer reads `roots[*].task.id` as the set of tasks that block others and are not themselves blocked — the meaning the key carries and the meaning the command had before this work — and starts or reports a task `tick show` says is blocked, or an ID that names no task. The document carries no discriminator to recover the distinction from. Toon is honest under the same widening (`dep_tree[N]{from,to}` claims only that an edge exists) and pretty draws an unlabelled tree; JSON is the one format whose key asserts a property no longer true of its contents. The README now explains the quirk in prose (`README.md:321`: "which the TOON edge list and the JSON `roots` array carry as edges and nodes") — a decoding rule learned outside the output, the defect class §1 opens this work for. The `Roots`/`Unrooted` split in `DepTreeResult` (`internal/cli/format.go:217-239`) is meanwhile bought and never spent: all three formatters discard it through `fullGraphTrees()` and no consumer reads the two separately.

**Solution**: Rename the JSON key to one that claims only what the value holds — `trees` — and collapse the published pair into the single field every formatter reads, keeping the root-versus-seeded distinction local to `BuildFullDepTree`, which still needs it for the empty-answer condition. Drop the README sentence at `README.md:321` that exists to explain the mislabel, and re-pin the affected sample. `jsonConformanceListKeys` decodes the key and follows the rename. Confined to `jsonDepTreeFull` and the domain field: toon and pretty do not read the key. Settled rather than staged: this work already answered the same question once, renaming `blocked_by` to `blocker` on the dependency-change document rather than letting one key mean two things, and no specification claim names `roots` — so the correction lands in the README and the shipped output, on the precedent the corrigendum of 2026-09-20 set. The alternative the finder weighed — publishing the seeded trees under their own second key — keeps a distinction the record shows no consumer spends, and pays a second key for it.

**Outcome**: `tick dep tree --json` returns `{"mode","trees","chains","longest","blocked"}`, and every key in it is true of its contents on a cycle and on a dangling blocker.

**Do**:
- `internal/cli/format.go:213-240` — replace the published `Roots`/`Unrooted` pair with one field, `Trees []DepTreeNode`, holding every full-graph tree (those grown from roots first, then those seeded from participants no root reaches), delete `fullGraphTrees()`, and rewrite the type's doc comment for the single field.
- `internal/cli/dep_tree_graph.go:117-183` — keep `roots` and `unrooted` as locals inside `BuildFullDepTree`: the `longest` fold (`:157-160`) and the empty-answer condition (`:169-172`) still read them separately. Return `Trees: slices.Concat(roots, unrooted)`.
- Point the three read sites at the field — `toon_formatter.go:184`, `pretty_formatter.go:375`, `json_formatter.go:388` — and in `jsonDepTreeFull` (`json_formatter.go:338-345`) rename the member to `Trees` with tag `json:"trees"`, updating the `FormatDepTree` doc comment (`:373-375`) that lists the full-graph keys.
- Follow the rename through the whole measured set. `grep -rn 'Roots\|Unrooted\|fullGraphTrees' internal/cli` → the production sites above plus 23 `DepTreeResult{Roots: …}` literals (`toon_formatter_test.go` 8, `json_formatter_test.go` 7, `pretty_formatter_test.go` 8) and 29 `result.Roots` references in `dep_tree_graph_test.go`; `grep -rn '"roots"' internal/cli README.md` → `dep_tree_test.go:175` (the `jsonDepTreeRoot` helper) and `:570`, `json_formatter_test.go:1039`/`:1096`/`:1130`/`:1159`/`:1205`/`:1396` (`fullExpected`)/`:1521`, `conformance_test.go:706` (`jsonConformanceListKeys`), `README.md:321`. `"it terminates full graph with circular dependency"` (`dep_tree_graph_test.go:599-612`) asserts `Roots = 0`, which the collapse makes meaningless — re-express it as the cycle's single seeded tree in `Trees`, keeping the termination the subtest exists for.
- `README.md:321` — drop the clause naming the JSON `roots` array, keeping the sentence's statement that all three formats cover every participant. The fenced dep-tree samples are pretty and toon and do not move.

**Acceptance Criteria**:
- [ ] `tick dep tree --json` in full-graph mode returns exactly the keys `mode`, `trees`, `chains`, `longest`, `blocked`, and no shipped output carries a key named `roots`.
- [ ] On a two-task cycle, `trees` carries the cycle's participants; on a task blocked by an ID no record matches, `trees` carries that participant's tree — neither under a key claiming rootness.
- [ ] On a project where no task has dependencies, `trees` is `[]` rather than `null` and `chains`, `longest`, `blocked` read zero.
- [ ] `grep -rn 'Unrooted\|fullGraphTrees' internal/cli` returns nothing; `DepTreeResult` publishes one tree field and `BuildFullDepTree` keeps the root-versus-seeded distinction local.
- [ ] Toon and pretty `dep tree` output is byte-identical to before the change on the rooted, cycle and dangling-blocker projects.
- [ ] `jsonConformanceListKeys` carries `trees` in place of `roots`, and the JSON conformance driver decodes the full dep-tree document with no null-list problem.
- [ ] `README.md` names no JSON `roots` array, and `"it reproduces the README dep tree samples"` still passes.

**Tests**:
- `"it publishes every full-graph tree under trees"`
- `"it carries a cycle's participants under trees"`
- `"it emits trees as an empty list when no task has dependencies"`
- `"it carries a bare id for a blocker no task carries in JSON"` (existing, retargeted at `trees`)
- `"it terminates full graph with circular dependency"` (existing, re-expressed over the single field)

## Task 4: A Dependency Participant That Names No Task Says So

severity: medium
sources: standards

**Problem**: An orphaned dependency — the state `internal/doctor/orphaned_dependency.go` exists to report — now renders as a task with blank fields. `buildUnrootedTrees` seeds `DepTreeTask{ID: id}` and upgrades it from `taskIdx` only when a record exists (`internal/cli/dep_tree_graph.go:228`), so a dangling blocker keeps zero-valued Title and Status; `writeDepTreeTaskLine` then formats `%s%s  %s (%s)` over the blanks (`internal/cli/pretty_formatter.go:422`) and `toJSONDepTreeNodes` copies them through (`internal/cli/json_formatter.go:361`). Verified against a built binary: pretty draws `tick-ghost   ()` and JSON emits `{"id":"tick-ghost","title":"","status":""}`. A human reads the line as a rendering bug; an agent reads the object as a task that exists and has no title, and acts on it — `tick show tick-ghost` then answers "task not found". Before this work the ID could reach neither rendering, so both outputs are new. The toon edge list is unaffected: it carries IDs only. §8's corrigendum requires the full-graph edges of a dangling blocker to reach the output and §4.1's second corrigendum requires pretty to draw the participants no root reaches "through the existing tree rendering with no new visual form"; neither reached how a participant that is not a task is presented as a node, so the zero value stood in for a decision nobody took. §8 has since taken it: the paragraph added on 2026-09-20 fixes status `missing` with an empty title — `tick-ghost   (missing)` in pretty, `{"id":"tick-ghost","title":"","status":"missing"}` in JSON — and the corrigendum of the same date records it. The specification is amended; the code has not followed, and that is the whole of what is left here.

**Solution**: Fill the status slot with what is true instead of leaving it blank: a participant that resolves to no task renders with status `missing` and an empty title, so pretty draws `tick-ghost   (missing)` and JSON returns `{"id":"tick-ghost","title":"","status":"missing"}`. Record the shipped-output shape as a corrigendum entry against §8, since §8's corrigendum is what put this class of graph in the output. Settled rather than staged, on two grounds the record already fixes: dropping non-task participants from the node-shaped renderings — the finder's other branch — would take the blocked real task out of pretty's tree with it, since the ghost is what seeds that walk, which reverses the ad hoc pass's settled direction that the terminal stops denying dependencies that exist; and leaving the blanks makes an agent learn outside the output that an empty `status` means "no such task", which is the rule §1 exists to delete. The marker occupies the existing status slot and introduces no new visual form, so §4.1's bound holds. The four real statuses stay unambiguous — `missing` collides with none of them.

**Outcome**: An agent reading `tick dep tree` can tell a task from an ID that names none, out of the output itself: the participant `tick show` would answer "task not found" for carries status `missing` in both node-shaped renderings.

**Do**:
- `internal/cli/dep_tree_graph.go:221-239` — seed the node in `buildUnrootedTrees` as `DepTreeTask{ID: id, Status: depTreeMissingStatus}` and keep the upgrade from `taskIdx` when a record exists, so only an ID with no record keeps the marker. Declare `const depTreeMissingStatus = "missing"` in the same file, the one place the marker is spelled.
- Leave that seed as the only site able to produce a non-task node: `walkDownstream` (`:56-60`) and `walkUpstream` (`:91-95`) already skip IDs absent from `taskIdx`, so no other builder path needs touching.
- Retarget the two existing dangling-blocker assertions: `"it carries a bare id for a blocker no task carries in JSON"` (`dep_tree_test.go:351-359`) expects `assertJSONDepTreeTask(t, root, "tick-ghost1", "", "")` — take status `"missing"` and rename the subtest to say so; `"it renders a dangling blocker in the terminal"` (`:380-393`) pins `"tick-ghost1   ()\n"` — take `"tick-ghost1   (missing)\n"`, the title still empty so the spacing is unchanged.
- Add a builder-level subtest in `dep_tree_graph_test.go` over `danglingBlockerTasks(now)` (`dep_tree_test.go:224`): the seeded node carries `Status: "missing"` and an empty `Title`, and the real task beneath it keeps `Task A`/`open`.
- Add a guard that the marker is confined to participants with no record: on a two-task cycle every rendered node carries its real status, and `"it emits an edge from a blocker no task carries"` (`dep_tree_test.go:318`) stays green unedited, pinning the toon edge list as ID-only.

**Acceptance Criteria**:
- [ ] `tick dep tree --pretty` in a project whose only blocker ID matches no task draws `tick-ghost1   (missing)` with the blocked task beneath it and an unchanged summary line.
- [ ] `tick dep tree --json` returns `{"id":"tick-ghost1","title":"","status":"missing"}` for that participant, its child keeping its own title and status.
- [ ] No participant that resolves to a task record carries the marker — a cycle's members render their real statuses.
- [ ] The toon edge list is byte-identical: it carries IDs only.
- [ ] `missing` is spelled once in production code and collides with none of the four task statuses.
- [ ] `go test ./...` green, with only the two dangling-blocker assertions changed.

**Tests**:
- `"it marks a blocker no task carries as missing in JSON"` (existing subtest, retargeted)
- `"it renders a dangling blocker as missing in the terminal"` (existing subtest, retargeted)
- `"it seeds a participant with no task record with status missing"`
- `"it keeps real statuses on a cycle's participants"`
- `"it emits an edge from a blocker no task carries"` (existing, unedited — the toon edge list does not move)

## Task 5: Each Count-Zero TOON Header Derives Its Columns From The Row Struct

severity: duplication
sources: duplication

**Problem**: Five sites state a section's column list as a string literal that must equal the `toon` tags of the row struct the populated branch marshals: `tasks[0]{id,title,status,priority,type}:` (`internal/cli/toon_formatter.go:53`) against `toonTaskRow` (`:22-28`); `changed[0]{id,title,from,to,auto}:` (`:150`) against `toonChangedRow` (`:139-145`); `%s[0]{from,to}:` (`:241`) against `toonEdgeRow` (`:165-168`); `%s[0]{id,title,status}:` (`:308`) against `toonRelatedRow` (`:31-35`); `notes[0]{index,text,created}:` (`:320`) against `toonNoteRow` (`:38-42`). Add, rename or reorder a column and the populated branch picks it up from the struct tags while the count-zero branch keeps its literal, so one section declares two schemas depending on whether it has rows — `tick done` on a cascading task printing `changed[2]{…,owner}:` while `tick done` on a task that moves nothing prints `changed[0]{id,title,from,to,auto}:`. That is the branch-on-which-document-arrived defect §7.2 and §8 exist to delete, and an agent that learned the schema from an empty document mis-parses the populated one. Nothing in the suite fails: `assertCountZeroSection` (`internal/cli/toon_decode_test.go:161`) compares the empty header against its own copy of the same stale literal, `assertToonFields` (`:58`) checks only the keys it is handed, and the README samples `readme_samples_test` pins against real output are the populated ones — so the loud failure fires on the README and the struct, the developer fixes both, and the count-zero literals are the one copy nothing pushes them to touch. The literals are not gratuitous: `toon.MarshalString` of an empty typed slice emits `changed[0]:` with no columns at all, so the library cannot produce the header §8 requires. This work extended the pattern rather than closing it — `buildChangedSection` is a new copy, and `buildNotesSection`'s header gained `index` by hand-editing the literal beside the struct field.

**Solution**: Add one generic helper to `toon_formatter.go` — `emptyToonSection[T any](name string) string` — that reads `T`'s `toon` struct tags in declaration order by reflection and renders `name[0]{…}:`, and call it at each of the five sites with the same row type the populated branch marshals (`emptyToonSection[toonChangedRow]("changed")`, `emptyToonSection[toonRelatedRow](name)`, and so on). The column list then exists once, in the struct. Leave the literals standing in the test assertions and the README samples: with the production copy derived, those become the independent pin that fails loudly when a column moves — the behaviour the empty branch does not have today. Behaviour-preserving: every rendered header is byte-identical, and the existing tests are the check.

**Outcome**: A toon section declares one schema whether or not it has rows, because both branches read the same struct — a column added, renamed or reordered moves the count-zero header with the populated one, and the literals left in the tests and the README become the pin that fails when it does not.

**Do**:
- Add `emptyToonSection[T any](name string) string` to `internal/cli/toon_formatter.go`, beside `encodeToonSection` (`:334-343`): take `reflect.TypeFor[T]()`, walk its fields in declaration order, read each field's `toon` tag (the name before any comma), join the names with commas and return `fmt.Sprintf("%s[0]{%s}:", name, cols)`.
- Replace all five literals with calls passing the row type the populated branch marshals — the complete production set, `grep -n '\[0\]{' internal/cli/toon_formatter.go` → 5 matches, and `grep -rn '\[0\]{' --include='*.go'` shows no other production file holds one: `:53` → `emptyToonSection[toonTaskRow]("tasks")`, `:150` → `emptyToonSection[toonChangedRow]("changed")`, `:241` → `emptyToonSection[toonEdgeRow](name)`, `:308` → `emptyToonSection[toonRelatedRow](name)`, `:320` → `emptyToonSection[toonNoteRow]("notes")`.
- Leave every literal in the test assertions and the README samples exactly as it stands — the `assertCountZeroSection` callers, the two `tasks[0]{id,title,status,priority,type}:` comparisons (`toon_formatter_test.go:94`, `:106`) and the `dep_tree[0]{from,to}:` comparison (`:1046`). Derived production against literal test is what this task buys.
- Confirm preservation before committing: `go test ./...` green with no test file edited, then `go vet ./...`, `gofmt -w ./internal ./cmd` and `golangci-lint run ./...` clean.

**Acceptance Criteria**:
- [ ] `grep -n '\[0\]{' internal/cli/toon_formatter.go` matches only the helper's format string; the five hand-written column lists are gone.
- [ ] Each of the five call sites passes the same row type its populated branch marshals.
- [ ] Every count-zero header is byte-identical to before: `tasks[0]{id,title,status,priority,type}:`, `changed[0]{id,title,from,to,auto}:`, `dep_tree[0]{from,to}:`, `blocked_by[0]{from,to}:`, `blocks[0]{from,to}:`, `blocked_by[0]{id,title,status}:`, `children[0]{id,title,status}:`, `notes[0]{index,text,created}:`.
- [ ] No test file, no README sample and no other production file is edited.
- [ ] `go test ./...`, `go vet ./...` and `golangci-lint run ./...` all clean.

**Tests**: behaviour-preserving refactor — no test is added, renamed or weakened, and no test semantics change. The existing count-zero assertions are the check and must stay green unedited:
- `"it formats zero tasks as empty section"` (`toon_formatter_test.go:91`) and `"it formats zero tasks from nil slice as empty section"` (`:103`)
- the `assertCountZeroSection` callers at `toon_formatter_test.go:162`, `:164`, `:215`, `:216`, `:688`, `:1094`, `:1106`, `:1344`, `list_show_test.go:936`, `:1394`, `detail_changes_test.go:79`, `update_test.go:1423`, `create_test.go:1414`, `format_test.go:415`
- `"it renders the emptied full document for a result with no roots"` (`toon_formatter_test.go:1044`)

## Task 6: Corrections

severity: corrections
sources: architecture

**Problem**: `TestConformanceScopeBoundary` (`internal/cli/conformance_test.go:1310-1338`) reads four sibling test files and asserts that named function names and subtest strings still appear in their source text. It checks nothing the tool does. It pins test identifiers, the most refactor-prone text in the suite, so a rename fails the suite naming a file the maintainer did not edit; and it is satisfied by a name with no assertions behind it, so emptying one of the named test bodies loses the coverage it exists to protect while the suite stays green. It also encodes a workflow concern — an earlier phase's assertions must not be deleted by a later one — into the shipped suite, where it outlives the phases that motivated it. The guard that matters is already there: the conformance drivers decode every document in the inventory, so a deleted assertion that mattered surfaces as a document nobody checks.

**Solution**: Delete `TestConformanceScopeBoundary` and its single subtest, `internal/cli/conformance_test.go:1310-1338`. The assertions it names stay carried by their own files; nothing else in the suite reads it.

**Outcome**: No test in the suite asserts over the source text of a sibling test file; every guard that remains runs against output the tool produced.

**Do**:
- Delete `TestConformanceScopeBoundary` and its subtest `"it keeps the phase 1 and phase 3 decoded assertions"`, `internal/cli/conformance_test.go:1310-1338`.
- Drop the imports that deletion leaves unused from the file's import block (`:3-21`): `os` and `path/filepath` (used only at `:1327`) and `github.com/leeovery/tick/internal/testutil` (used only at `:1312`). `strings` stays — `:715`, `:835`, `:921`, `:938`, `:967`, `:970`, `:1751`, `:1754`.
- Touch nothing else: the four files the guard read — `toon_decode_test.go`, `toon_formatter_test.go`, `stats_test.go`, `dep_tree_test.go` — keep every assertion they carry.
- Confirm `go build ./...`, `go vet ./...` and `go test ./...` are green.

**Acceptance Criteria**:
- [ ] `grep -rn 'TestConformanceScopeBoundary' internal/cli` returns nothing.
- [ ] `internal/cli` compiles with no unused import and `go vet ./...` is clean.
- [ ] The four files the guard named are byte-unchanged.
- [ ] `go test ./...` green, with no other test deleted, renamed or weakened.

**Tests**: the deletion of a guard that tested no product behaviour — nothing is added and no test semantics change. The coverage it claimed keeps running where it lives:
- `TestToonOutputConformance` (`conformance_test.go:697`) and `TestJSONOutputConformance` (`:1725`) drive every document in the inventory and stay green
- `TestToonTaskDetailConformance` and `TestToonDepTreeFocusedConformance` (`toon_decode_test.go`), `"it decodes stats for a project with no tasks"` (`stats_test.go`) and `"it returns the emptied document when no task has dependencies"` (`dep_tree_test.go:289`) keep their assertions unedited
