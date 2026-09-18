# Phase 6: Conformance verification across the output inventory — 6 tasks

## free-text-round-trip-6-1

### Task 1: Task-list documents decode through one conformance table

**Problem**: §11's first requirement is that every structured command's output is decoded by a real TOON reader in the suite and that the test fails if it will not parse. Phases 1 to 5 each decoded the documents they changed, but that coverage is scattered across per-formatter test files and is counted in assertions rather than in documents — nothing enumerates the branches a command can take, so a branch nobody thought of is a branch nobody decodes. `tick list` is the plainest case: its empty branch returns a hand-written `tasks[0]{id,title,status,priority,type}:` literal (`internal/cli/toon_formatter.go:61-63`) that no test decodes, and `internal/cli/toon_formatter_test.go:16-56` checks the populated rows by comparing them against strings written down beside the code — the exact mechanism §11 names as the reason a malformed header survived for the tool's entire life.

**Solution**: Add one enumerated inventory of the documents the tool produces plus a driver that runs each entry through `App.Run` with `--toon` and decodes stdout with the project's TOON library, populate it with the `list`, `ready` and `blocked` documents, and convert the populated list golden strings in `toon_formatter_test.go` to decoded-value assertions.

**Outcome**: `TestToonOutputConformance` decodes every task-list document the three handlers can produce, names the document and prints its text when one fails to parse, and no populated task-list assertion compares output against a pinned string.

**Do**:
1. `internal/cli/conformance_test.go` (new file) — declare `type conformanceDoc struct { Name string; Command string; Setup func(t *testing.T) (dir string, args []string); NotADocument string }` and `var conformanceDocs = []conformanceDoc{}`. `Setup` seeds a project and returns its directory plus the arguments that follow `tick`; `NotADocument` holds the reason an entry's output is not a document, and an entry carrying one carries no `Setup`.
2. `internal/cli/conformance_test.go` — add `runTickConformance(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int)` constructing the `App` with `IsTTY: true` so the format flag the driver passes is what resolves the format, and `TestToonOutputConformance` iterating `conformanceDocs`, skipping entries with a non-empty `NotADocument`, running `tick --toon <args>`, failing on a non-zero exit, and decoding stdout through `decodeToonDoc` (added in Phase 1, `internal/cli/toon_decode_test.go`). A decode failure reports `doc.Name` and the document's text.
3. `internal/cli/conformance_test.go` — add `TestConformanceInventoryWellFormed` asserting every entry's `Name` is unique and that `NotADocument != ""` holds exactly when `Setup == nil`.
4. `internal/cli/conformance_test.go` — add the task-list entries: `list` on a project carrying several tasks including one whose `Type` is empty; `list` on an empty project; `list --status done` on a project whose tasks are all open; `ready` populated; `ready` with nothing ready; `blocked` populated; `blocked` with nothing blocked. Add one `NotADocument` entry for `list --quiet`, its reason naming bare IDs.
5. `internal/cli/toon_formatter_test.go:16-56` — rewrite the populated list subtest as a `decodeToonDoc` assertion checking the decoded `tasks` list row by row, with the empty `type` decoding as `""`. In the empty-slice and nil subtests add the decoded assertion that `tasks` is an empty list, and keep their existing `tasks[0]{id,title,status,priority,type}:` comparison as the check on the count-zero header's column schema.

**Acceptance Criteria**:
- [ ] `TestToonOutputConformance` decodes the `--toon` stdout of every non-exempt entry in `conformanceDocs` via the project's TOON library
- [ ] A decode failure reports the entry's `Name` and the undecodable document text
- [ ] The inventory carries a `list` document for a populated project, an empty project and a filter matching nothing, plus populated and empty documents for `ready` and for `blocked`
- [ ] The empty-project and filter-matches-nothing entries both decode to a document whose `tasks` key is an empty list
- [ ] A row whose `type` is empty decodes with `type` equal to `""` and every other column in its own position
- [ ] The `list --quiet` entry is present in the inventory with a stated reason and is not decoded
- [ ] `TestConformanceInventoryWellFormed` fails when two entries share a `Name` or when an entry carries both a `Setup` and a `NotADocument` reason
- [ ] No assertion on a populated task-list document compares rendered output against a pinned string
- [ ] The count-zero `tasks` header's column schema is still asserted as text, and the same subtest additionally asserts the decoded empty list
- [ ] `internal/cli/blocked_test.go` and `internal/cli/list_filter_test.go` still assert pretty's `No tasks found.` against their existing golden strings
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

**Tests**:
- `"it decodes the populated list document"` — decoded `tasks` carries one entry per seeded task with matching `id`, `title`, `status`, `priority` and `type`
- `"it decodes the empty-project list document"` — decoded `tasks` is an empty list
- `"it decodes the list document when a filter matches nothing"` — same, with tasks present but filtered out
- `"it decodes the ready document with results"` — decoded `tasks` carries the ready task
- `"it decodes the ready document with no results"` — decoded `tasks` is an empty list
- `"it decodes the blocked document with results"` — decoded `tasks` carries the blocked task
- `"it decodes the blocked document with no results"` — decoded `tasks` is an empty list
- `"it decodes an empty type as an empty string"` — the decoded row's `type` is `""` and its `priority` is the seeded number
- `"it keeps the column schema on the empty tasks header"` — `FormatTaskList(nil)` equals `tasks[0]{id,title,status,priority,type}:` and decodes to an empty list
- `"it names the failing document when a decode fails"` — a deliberately malformed string passed to the driver's decode step reports the entry name and the text
- `"it rejects a duplicate document name"` — two entries sharing a `Name` fail `TestConformanceInventoryWellFormed`
- `"it rejects an entry carrying both a setup and an exemption reason"` — the well-formedness guard fails
- `"it keeps pretty's no-tasks golden assertion"` — the existing `No tasks found.` assertions still pass unmodified

**Edge Cases**:
- `runList`, `runReady` and `runBlocked` construct the `App` with `IsTTY: true` and therefore resolve to pretty; the driver passes `--toon` explicitly rather than relying on a default
- The empty branch is the hand-written `tasks[0]{id,title,status,priority,type}:` literal, not library output, so decoding it is the only check that it parses — and decoding cannot recover its columns, since `tasks[0]:` decodes to the same empty list, which is why that one comparison stays as text
- An empty project and a filter matching nothing are two handler branches that produce the same document; both are entries, because the table counts branches
- A row whose `type` is empty is the case §12.1 records the README getting wrong — the library quotes it as `""` and the column must not shift
- `--quiet` prints bare IDs rather than a document; the entry stays in the inventory carrying its reason so the exclusion is visible rather than an omission a reader has to notice
- Pretty's `No tasks found.` branch keeps its golden-string assertions — pretty has no parser and §11 keeps its goldens
- `ready` and `blocked` are separate handlers (`internal/cli/app.go:212-237`), each building its own filter, so each needs its own entries rather than borrowing `list`'s
- The populated list goldens are converted here rather than in Task 6's sweep, because this task replaces them with the decoded assertions that take their place

**Context**:
> §11, part 1: "Every structured command's output is decoded by a real TOON reader in the suite, and the test fails if it will not parse. This alone catches the entire class of defect this work exists to fix — a section nobody can read, whatever its content. The commands are §3.1's table, and the coverage is counted in documents rather than commands: every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces, a multi-field selection and one narrowed by position included (§9.2, §9.3)."
>
> §11 on why the existing suite could not catch it: "Its assertions compare output against a string written down alongside the code, so a malformed header passed for the tool's entire life: the test compared a wrong string to the same wrong string."
>
> §11, part 3: "Rewritten assertions check decoded values, not output text. 'The notes section has two rows and the second row's text is X', not 'the output equals this blob'." And: "Pretty keeps golden-string assertions. Pretty output has no parser, so a decoded-value assertion does not exist for it; removing its golden strings would replace its only form of assertion with nothing."
>
> §3.1 lists `list`, `ready` and `blocked` as the commands producing the task-list document. §1's table records that this section is already library-written (`encodeToonSection("tasks", rows)`) and parses today — the branch that does not is the hand-written empty literal.
>
> §12.1 on the empty `type`: "it renders an empty `type` as a bare trailing comma, where the formatter quotes it… `toonTaskRow.Type` carries no `omitempty` and the library quotes an empty string."
>
> Phase 1 task free-text-round-trip-1-6 added `decodeToonDoc(t *testing.T, doc string) map[string]any` in `internal/cli/toon_decode_test.go`; `setupTickProject` and `setupTickProjectWithTasks` are in `internal/cli/create_test.go:18-46`.
>
> §11's "no byte-level pinning is kept in the machine formats" is read here as covering rendered documents. A count-zero section header is the one shape a decoded assertion cannot express — `tasks[0]{id,title,status,priority,type}:` and `tasks[0]:` both decode to an empty list — and Phases 1 to 4 already assert the other count-zero headers as text for that reason. Task 6 names the retained set explicitly rather than leaving the boundary implicit.
>
> The specification requires the coverage but does not prescribe how it is organised. One inventory with a driver is chosen over per-command tests because Task 3 turns the same inventory into a guard that a newly added command must be declared, which a scatter of independent tests cannot express.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §3.1, §12.1

## free-text-round-trip-6-2

### Task 2: Stats, dependency-tree and detail documents join the table

**Problem**: Three of the five document kinds in §3.1's must-parse table are still outside the inventory. Each reached its final shape in an earlier phase and each phase decoded what it changed — Phase 1's `TestToonTaskDetailConformance`, Phase 3's stats and dep-tree subtests — but none of those enumerates branches. `tick stats` on an empty project, the full dep tree where a cycle leaves zero roots beside non-zero counts, and the focused dep tree's four combinations of upstream and downstream edges are each a document the tool can produce that nothing decodes. §8's emptied forms are the branches most at risk: they are the ones a reader could not predict, and until Phase 3 they were not documents at all.

**Solution**: Extend `conformanceDocs` with the `stats`, `dep tree` and `show` documents, one entry per branch, so every one of them is decoded by the driver Task 1 added.

**Outcome**: `TestToonOutputConformance` decodes eleven further documents — both stats branches, the full dep tree's three branches, the focused dep tree's four, and `show`'s two — with the emptied forms of §8 among them as documents rather than exemptions.

**Do**:
1. `internal/cli/conformance_test.go` — add the `stats` entries: a project carrying tasks across several statuses and priorities, and an empty project.
2. `internal/cli/conformance_test.go` — add the full `dep tree` entries: a seeded chain of three tasks; a project whose tasks carry no dependencies; and a two-task cycle where each task blocks the other, which leaves zero roots while `BuildFullDepTree` still counts chains and blocked tasks (`internal/cli/dep_tree_graph.go:151-172`).
3. `internal/cli/conformance_test.go` — add the focused `dep tree <id>` entries for a task carrying both directions, upstream only, downstream only, and neither.
4. `internal/cli/conformance_test.go` — add the `show` entries: a task carrying type, parent, closed, tags, refs, description, notes, children and blocked_by, and a task carrying none of them. Add one `NotADocument` entry for `dep tree --quiet`, its reason naming that it prints nothing.
5. Confirm the scope boundary: `grep -n 'TestToonTaskDetailConformance' internal/cli/toon_decode_test.go` still resolves and Phase 3's stats and dep-tree decoded-value subtests in `internal/cli/toon_formatter_test.go`, `internal/cli/stats_test.go` and `internal/cli/dep_tree_test.go` are unmodified by this task.

**Acceptance Criteria**:
- [ ] `tick stats --toon` decodes on a populated project and on an empty one, the empty one carrying all-zero counts and `by_priority`'s five rows
- [ ] `tick dep tree --toon` decodes on a populated graph, on a project with no dependencies, and on a two-task cycle
- [ ] The cycle document decodes with an empty `dep_tree` list beside the counts the builder produced, not forced zeros
- [ ] `tick dep tree <id> --toon` decodes for all four combinations of upstream and downstream edges, each carrying top-level `id`, `title` and `status`
- [ ] The no-dependencies focused document decodes with both `blocked_by` and `blocks` as empty lists
- [ ] `tick show --toon` decodes for a task carrying every optional field and for one carrying none
- [ ] The `dep tree --quiet` entry is present in the inventory with a stated reason and is not decoded
- [ ] Phase 1's `TestToonTaskDetailConformance` and Phase 3's stats and dep-tree decoded-value subtests still exist and pass unmodified
- [ ] Every entry added by this task is a document in the inventory, not an exemption
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

**Tests**:
- `"it decodes the populated stats document"` — decoded document carries the seven counts and `by_priority`'s five rows
- `"it decodes the stats document for an empty project"` — every count decodes as `float64(0)` and `by_priority` still has five rows
- `"it decodes the populated full dep tree document"` — decoded `dep_tree` carries the chain's edges and the three summary counts are non-zero
- `"it decodes the emptied full dep tree document"` — decoded `dep_tree` is an empty list and the counts read zero
- `"it decodes the full dep tree document for a cycle"` — decoded `dep_tree` is empty while `blocked` and `chains` carry the builder's non-zero values
- `"it decodes the focused document with both directions"` — decoded `id`, `title`, `status` plus non-empty `blocked_by` and `blocks`
- `"it decodes the focused document with upstream only"` — `blocks` decodes to an empty list
- `"it decodes the focused document with downstream only"` — `blocked_by` decodes to an empty list
- `"it decodes the focused document with neither direction"` — both decode to empty lists and the three identity keys are present
- `"it decodes the full detail document"` — every optional key present with its stored value
- `"it decodes the bare detail document"` — `type`, `parent`, `closed`, `tags`, `refs` and `description` absent; `blocked_by`, `children` and `notes` present and empty
- `"it keeps the phase 1 and phase 3 decoded assertions"` — those tests still run and pass

**Edge Cases**:
- `stats` on an empty project is a branch of its own: every count is zero and `by_priority` still carries five rows, so the document's shape never varies with the data
- The full dep tree's cycle branch leaves zero roots while `blocked` and `chains` are non-zero — the emptied document reports what the builder found, so the entry asserts real counts rather than zeros
- The full dep tree's nothing-blocked branch and the focused no-dependencies branch are §8's two emptied forms; both are documents and therefore rows in the table, never exemptions
- The focused dep tree has four branches, not two: both directions, upstream only, downstream only, and neither
- `show`'s two detail branches are the every-optional-field task and the task carrying none; `show`'s filtered documents belong to Task 4
- Phase 1's and Phase 3's decoded-value subtests stay: the table adds enumerated parseability across branches, it does not replace per-document value assertions
- `dep tree --quiet` returns before the store is opened (`internal/cli/dep_tree.go:13-15`) and prints nothing, so it is declared in the inventory with its reason rather than decoded

**Context**:
> §11: "the coverage is counted in documents rather than commands: every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces."
>
> §8: "What replaces them is the document the non-empty branch produces, emptied: the same fields, with the summary fields of §5.2 reading zero and the edges section carrying its count-zero header. Where the caller named a task, the emptied document still identifies it exactly as the populated one does. Both empty branches take that shape — nothing blocked anywhere, and a named task with no dependencies either way — so an agent parses one document whether or not anything is blocked, and reads the counts to learn which it got."
>
> §5.2: "Which sections a document carries is unchanged by this work… `children`, `blocked_by` and `notes` are always present, carrying a count-zero header when empty, while `type`, `parent`, `closed`, `tags`, `refs` and `description` appear only when the task carries them." The two `show` entries are exactly those two shapes.
>
> §3.1 lists the commands: task detail from `show`, task list from `list`/`ready`/`blocked`, stats from `stats`, the dependency graph from `dep tree`, and status changes from `start`/`done`/`cancel`/`reopen`.
>
> `BuildFullDepTree` counts blocked tasks and chains over all participants regardless of whether any root exists (`internal/cli/dep_tree_graph.go:151-172`), and `internal/cli/dep_tree_graph_test.go:599-613` already fixes that a two-task cycle yields zero roots — which is why the cycle entry asserts the builder's counts rather than zeros.
>
> Phase 3 tasks 2 to 4 made the emptied dep-tree forms reachable through the handler; before that the nothing-blocked branch never reached a formatter at all.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §8, §3.1, §5.2

## free-text-round-trip-6-3

### Task 3: Mutation and status documents complete the toon inventory

**Problem**: The nine remaining commands in §3.1's must-parse table — `create`, `update`, `note add`, `note remove`, `start`, `done`, `cancel` and `reopen` — produce the documents Phase 2 reshaped, and none is in the inventory. `create` carries `changed[0]` with no parent and rows under a done one; `update` has four distinct branches depending on which cascade rules fire; `note add` and `note remove` carry no `changed` section at all, and that absence is part of their shape. Worse, nothing keeps the inventory complete: a command added to `commandFlags` tomorrow joins the CLI with no document declared for it and the table silently does not cover it — which is how the format broke the first time, a section added by hand with nothing that could notice.

**Solution**: Add the mutation and status entries, then close the inventory with a guard that places every key of `commandFlags` in exactly one of three declared sets — the must-parse commands the inventory covers, §3.2's prose commands, and §3.3's out-of-scope commands — so a newly added command fails the suite until it is declared.

**Outcome**: Every document `create`, `update`, `note add`, `note remove` and the four status commands can produce is decoded by the driver, and `TestConformanceInventoryCoversEveryCommand` fails when a `commandFlags` key is declared nowhere or in two places.

**Do**:
1. `internal/cli/conformance_test.go` — add the status entries: `start`, `done`, `cancel` and `reopen` each on a childless task with no cascade, plus `done` on a parent with an open child and `reopen` on a done task under a done parent, both of which cascade.
2. `internal/cli/conformance_test.go` — add the `create` entries — no `--parent`, and `--parent <done task>` — and the `note add` and `note remove` entries, each on a task already carrying a note.
3. `internal/cli/conformance_test.go` — add the four `update` entries: an edit that moves no status, a move to a done parent (Rule 6 alone), a move away from a parent whose remaining children are all terminal (Rule 3 alone), and a move where both fire and meet on a shared ancestor. Add one `NotADocument` entry for `create --quiet`, its reason naming the bare ID.
4. `internal/cli/conformance_test.go` — declare `conformanceProseCommands = []string{"dep add", "dep remove", "remove", "init", "rebuild"}` and `conformanceOutOfScopeCommands = []string{"doctor", "migrate"}`, and add `TestConformanceInventoryCoversEveryCommand` reading the live `commandFlags` map and failing when a key appears in none of the three sets, when a key appears in more than one, or when a name in either declared list is not a `commandFlags` key.
5. `internal/cli/conformance_test.go` — assert in the `note add` and `note remove` entries' own subtests that the decoded document carries no `changed` key.

**Acceptance Criteria**:
- [ ] Each of `start`, `done`, `cancel` and `reopen` has a decoded document for its no-cascade branch, and `done` and `reopen` additionally for a cascading branch
- [ ] `tick create --toon` with no parent decodes with `changed` present and empty
- [ ] `tick create --parent <done task> --toon` decodes with a row per task the reopen moved
- [ ] `tick update --toon` decodes on all four branches, and the shared-ancestor branch carries each task once
- [ ] `note add` and `note remove` documents decode and carry no `changed` key
- [ ] The `create --quiet` entry is present in the inventory with a stated reason and is not decoded
- [ ] `TestConformanceInventoryCoversEveryCommand` places every key of `commandFlags` — `ready` and `blocked` included — in exactly one of the three sets
- [ ] Adding a key to `commandFlags` without declaring it fails that test
- [ ] Declaring one command in two sets fails that test
- [ ] Naming a command in a declared list that is not a `commandFlags` key fails that test
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

**Tests**:
- `"it decodes a start document with no cascade"` — decoded `changed` has one row with `auto` false
- `"it decodes a done document with no cascade"` — same for `done`
- `"it decodes a cancel document with no cascade"` — same for `cancel`
- `"it decodes a reopen document with no cascade"` — same for `reopen`
- `"it decodes a cascading done document"` — decoded `changed` carries the requested row and the child's row
- `"it decodes a cascading reopen document"` — decoded `changed` carries the requested row and the ancestor's row
- `"it decodes a create document with no parent"` — decoded `changed` is an empty list and the key is present
- `"it decodes a create document under a done parent"` — decoded `changed` carries the reopened parent
- `"it decodes an update document with no status movement"` — decoded `changed` is an empty list
- `"it decodes an update document for rule 6 alone"` — the new parent's row is present
- `"it decodes an update document for rule 3 alone"` — the old parent's row is present
- `"it decodes an update document where both rules meet on a shared ancestor"` — that ancestor appears at most once
- `"it decodes a note add document with no changed section"` — decoded document has no `changed` key
- `"it decodes a note remove document with no changed section"` — same
- `"it covers every registered command"` — every `commandFlags` key resolves to exactly one of the three sets
- `"it fails when a command is declared nowhere"` — a key absent from all three sets fails the guard
- `"it fails when a command is declared twice"` — a name in two sets fails the guard
- `"it fails when a declared command is not registered"` — a name in a declared list with no `commandFlags` entry fails the guard

**Edge Cases**:
- The guard must place every `commandFlags` key in exactly one of must-parse / prose (`dep add`, `dep remove`, `remove`, `init`, `rebuild`) / out of scope (`doctor`, `migrate`), so a newly added command fails the suite until it is declared
- `ready` and `blocked` are registered by `init()` (`internal/cli/flags.go:93-96`), so the guard reads the map at test time rather than a hand-copied list of keys
- `create` with no parent carries `changed[0]` while `create --parent <done task>` carries rows — both are branches and both are entries
- `update` has four branches: no status movement, Rule 6 alone, Rule 3 alone, and both meeting on a shared ancestor
- `note add` and `note remove` carry no `changed` section at all, and that absence is part of the document's shape, so it is asserted rather than left unchecked
- Each of `start`, `done`, `cancel` and `reopen` gets a no-cascade document; `done` and `reopen` are the two that cascade naturally, so they carry the cascading entries
- `--quiet` on a mutating command prints the bare ID and is not a document; the entry carries its reason
- `rebuild` prints a confirmation message and belongs with the prose commands, not with the must-parse inventory

**Context**:
> §3.1's must-parse table: task detail from `show`, `create`, `update`, `note add` and `note remove`; task list from `list`, `ready` and `blocked`; stats from `stats`; the dependency graph from `dep tree`; status change from `start`, `done`, `cancel` and `reopen`.
>
> §3.2: "`dep add`, `dep remove`, `remove`, `init`, and the general-purpose messages. These are confirmations of a command the caller issued: the caller already knows what it asked for and the exit code says whether it worked."
>
> §3.3: "`doctor` and `migrate` are out of scope. Both bypass the formatter entirely and print straight to the terminal… Neither honours the format flags."
>
> §7.4: "**The section is always there.** `tick create` with no parent moves no task's status; the document still carries `changed[0]{id,title,from,to,auto}:`, exactly as an empty notes or children section carries its count-zero header (§8). A reader that must first find out whether the section exists is branching on which document arrived, which §7.2 exists to prevent."
>
> §7.3: "`show`, `note add` and `note remove` carry no `changed` section at all. Only a parent/child structural change moves another task's status (§7.5), and none of the three performs one, so there is nothing for the section to hold."
>
> §7.5 on `update`'s branches: "`tick update <id> --parent <other>` can fire two unrelated changes at once: the new parent reopens if it was finished, and the old parent may auto-complete if the moved task was the last unfinished thing under it (Rule 6 and Rule 3 together)."
>
> §11 opens with what the guard exists for: "Every other decision here is a shape. Nothing in them stops the next change adding a hand-built section and breaking the output again — which is exactly how it broke the first time."
>
> The specification names the must-parse, prose and out-of-scope sets but does not require a test that they partition the command registry. The guard is chosen because §11's stated purpose is to stop the next change re-introducing an unreadable section, and a table nobody is forced to extend stops covering the CLI the first time a command is added.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §3.1, §3.2, §3.3, §7.3, §7.4, §7.5

## free-text-round-trip-6-4

### Task 4: Field-selection documents and the two exemptions are covered

**Problem**: §11 requires both field-selection document forms — a multi-field selection and one narrowed by position — to be decoded in the suite, and requires bare-value output to be asserted as bytes. Phase 4 built those outputs and tested each behaviour where it landed, but the inventory of documents stops at full `show`. §3.1 also carves out the only two things in the whole CLI that are exempt from the must-parse rule — a bare value from `show --field`, and a selection that prints no bytes at all — and an exemption that exists only as an absence is indistinguishable from an oversight: a reader of the table cannot tell whether the bare value was considered and excluded or simply forgotten.

**Solution**: Add the three filtered documents `show` produces to the inventory so the driver decodes them, declare each exemption as an entry carrying its reason, and assert the exempt outputs as exact bytes in a test of their own.

**Outcome**: `tick show --field` documents decode through the same driver as every other document, the two §3.1 exemptions are visible in the inventory with the reason they are exempt, and the bare value is asserted byte-for-byte including its single terminating newline in every format.

**Do**:
1. `internal/cli/conformance_test.go` — add three `show` document entries: `--field description,notes` (a multi-field selection), `--field description,notes.2` (narrowed by position), and `--field notes` (a single name of a list section), each seeded on a task carrying a description and two notes.
2. `internal/cli/conformance_test.go` — add three `NotADocument` entries: `show --field description` (a bare value, §9.2), `show --field parent,closed` on a task carrying neither (zero bytes, §9.6), and `show --quiet --field title` (refused, §9.8). Each reason names why the output is not a document. The zero-byte entry names `parent` and `closed` because those two are the fields every format omits when the task does not carry them; `--field tags` on a tag-less task is zero bytes in toon and pretty but a `{"tags": []}` document in JSON, so it is a document rather than an exemption.
3. `internal/cli/conformance_test.go` — add `TestFieldSelectionExemptions` seeding one task and asserting `tick show <id> --field description` stdout equals the stored description plus exactly one `"\n"`, with stdout byte-identical under no format flag, `--toon`, `--pretty` and `--json`.
4. `internal/cli/conformance_test.go` — extend that test: `tick show <id> --field parent,closed` on a task carrying neither produces zero bytes and exit 0 with no format flag, under `--toon`, under `--pretty` and under `--json`, and `tick show <id> --quiet --field title` exits non-zero with zero bytes on stdout.
5. `internal/cli/conformance_test.go` — add decoded-value subtests for the three filtered documents: the multi-field one carries exactly `notes` and `description`; the narrowed one carries a single-row `notes` list whose `index` decodes as `2`; the list-section one carries `notes` alone.

**Acceptance Criteria**:
- [ ] `tick show <id> --field description,notes --toon` decodes and carries exactly the two selected keys
- [ ] `tick show <id> --field description,notes.2 --toon` decodes with a one-row `notes` list whose `index` is `2`, alongside the whole description
- [ ] `tick show <id> --field notes --toon` decodes and carries `notes` alone
- [ ] The bare-value output equals the stored value's bytes followed by exactly one newline
- [ ] That output is byte-identical with no format flag, with `--toon`, with `--pretty` and with `--json`
- [ ] A selection whose every name prints nothing produces zero bytes and exits zero, identically with no format flag, under `--toon`, under `--pretty` and under `--json`
- [ ] `--quiet` combined with a selection exits non-zero with zero bytes on stdout
- [ ] The three exemptions are entries in `conformanceDocs` carrying their reasons, not absences
- [ ] The driver skips them rather than attempting a decode
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

**Tests**:
- `"it decodes a multi-field selection document"` — decoded keys are exactly `notes` and `description`
- `"it decodes a position-narrowed selection document"` — decoded `notes` has one row whose `index` is `float64(2)` and `description` is present whole
- `"it decodes a single list-section selection document"` — decoded document's only key is `notes`
- `"it prints a bare value as its bytes plus one newline"` — stdout equals the stored description plus `"\n"`
- `"it prints the same bare bytes under every format flag"` — the four invocations produce identical stdout
- `"it prints zero bytes for a selection that renders nothing"` — `--field parent,closed` on a task carrying neither gives empty stdout and exit 0
- `"it prints zero bytes in every format for a selection that renders nothing"` — the four invocations all give empty stdout and exit 0
- `"it refuses quiet alongside a selection"` — exit non-zero, empty stdout
- `"it declares the bare value exempt in the inventory"` — the entry exists with a non-empty reason
- `"it declares the zero-byte selection exempt in the inventory"` — same
- `"it skips exempt entries in the toon driver"` — the driver attempts no decode for an entry carrying a reason

**Edge Cases**:
- A multi-field selection and a position-narrowed selection are documents and belong in the table; a bare value and a zero-byte selection are §3.1's two exemptions
- The bare-value assertion compares exact bytes including the single terminating newline, and holds identically under `--json`, `--pretty` and `--toon` because a bare value never reaches a formatter
- A selection whose every name prints nothing produces zero bytes — that is nothing rather than an empty document, so there is no document to decode and no `{}` to parse
- The exempt selection has to print nothing in *every* format: `parent` and `closed` are omitted when absent in toon, JSON and pretty alike, while `--field tags` on a tag-less task is zero bytes in toon and pretty and a `{"tags": []}` document in JSON, because JSON always carries a selected `tags` as `[]` (task `free-text-round-trip-4-4`)
- The exemptions are declared in the inventory with their reasons rather than being absent from it, so the table reads as a complete account of the tool's output
- A single name of a list section is a document, not a bare value, so it belongs in the table beside the multi-field form
- `--quiet` with a selection is refused, so it produces no document and no bare value either; it is declared for the same reason the other two are
- The JSON counterparts of the filtered documents belong to Task 5, which runs the same inventory under `--json`

**Context**:
> §11: "the coverage is counted in documents rather than commands: every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces, a multi-field selection and one narrowed by position included (§9.2, §9.3)." And: "Both field-selection document forms — a multi-field selection and one narrowed by position — are decoded in the suite, and bare-value output is asserted as bytes."
>
> §3.1: "Two things are exempt, both because they are not documents: a bare value from `tick show --field` (§9.2), for the reason §9.7 exempts it from the format flags, and a field selection that prints no bytes at all (§9.6), which is nothing rather than an empty document, in every format. Every document `show` produces — the full detail, and a filtered one — decodes."
>
> §9.2: "The value goes out as a line: its own bytes followed by a single newline, as the bare task ID already is." And: "A single field naming a list section returns that section, not a bare value… A list has no bare form."
>
> §9.7: "A request that returns a bare value ignores `--json`, `--pretty` and `--toon`; a request that returns a document honours them. A bare value is not a document, so there is nothing for a format flag to act on, and honouring one would re-quote the very string the flag exists to hand over unquoted."
>
> §9.6: "In the bare form that means no bytes at all: the terminating newline of §9.2 belongs to a value, so a field with no value produces an empty stream rather than a blank line… a selection whose every name prints nothing prints nothing at all and still exits successfully."
>
> §9.3: "In a multi-field selection the section renders as normal, its count following the selection while each row carries its real position via the `index` column (§6.3)." The narrowed entry asserts `notes[1]` carrying a row whose index reads 2.
>
> §9.8: "Passing `--quiet` and a field selection together is refused… an error with a non-zero exit and nothing on stdout."
>
> The specification exempts two outputs but does not say how the exemption is recorded. Declaring them as entries carrying their reason is chosen because §11's table is the artefact a later reader consults to learn what the tool produces, and an exemption held only in a specification section is invisible there.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §3.1, §9.2, §9.3, §9.6, §9.7, §9.8

## free-text-round-trip-6-5

### Task 5: JSON output is parsed and asserted by decoded value

**Problem**: §11 requires JSON output for each listed command to be parsed and asserted by decoded value, and §4.2 requires a JSON consumer to get the same structured answer as a TOON one. Today each command's JSON assertions sit beside its toon ones, so the two coverages are maintained by hand and drift: a branch added to the toon table has no JSON counterpart unless someone remembers to write one. JSON also carries invariants toon does not have and that nothing checks across the board — a list unmarshalling to `null` instead of `[]`, an `auto` rendered as a quoted string rather than a boolean, and a second document concatenated after the first, which is exactly what `create` emitted before Phase 2 folded the transition output into the record.

**Solution**: Run the same inventory under `--json` through a second driver that parses each document with `encoding/json`, asserts exactly one value per stream, and checks the invariants that hold across every document, then add decoded-value subtests where JSON's presence rules differ from toon's and cannot be copied across.

**Outcome**: Every document in the inventory is parsed as exactly one JSON value, every list key unmarshals to a non-nil slice, `auto` is a boolean and `index` a number wherever they appear, and a document covered in one format and not the other is impossible because one inventory drives both.

**Do**:
1. `internal/cli/conformance_test.go` — add `TestJSONOutputConformance` iterating `conformanceDocs`, skipping entries with a non-empty `NotADocument`, running `tick --json <args>` through `runTickConformance`, and reading stdout with a `json.Decoder`: the first `Decode` into `any` must succeed and the next must return `io.EOF`. Report the entry's `Name` and the stdout text on failure.
2. `internal/cli/conformance_test.go` — add `jsonTopLevelIsArray(command string) bool`, true for `list`, `ready` and `blocked` because `JSONFormatter.FormatTaskList` marshals an array (`internal/cli/json_formatter.go:27-40`), and assert the decoded value is a `map[string]any` for every other entry.
3. `internal/cli/conformance_test.go` — add `assertJSONInvariants(t *testing.T, name string, v any)` walking the decoded value and failing when the value under any of `changed`, `roots`, `blocked_by`, `blocks`, `tags`, `refs`, `notes`, `children` or `by_priority` is nil, when an `auto` value is not a `bool`, or when an `index` value is not a `float64`. Call it from the driver for every entry.
4. `internal/cli/conformance_test.go` — add per-document JSON subtests for the presence rules that differ from toon's: a bare task's detail carries `type`, `tags`, `refs` and `description` with empty values and carries no `parent` or `closed`; a filtered selection carries exactly the selected keys; `stats` carries `total`, `by_status`, `workflow` and `by_priority` rather than the seven flat counts toon emits.
5. Confirm the boundary: pretty appears in neither driver, and `internal/cli/json_formatter_test.go`'s existing decoded-value assertions are unmodified by this task.

**Acceptance Criteria**:
- [ ] Every non-exempt entry in `conformanceDocs` is run under `--json` and parses as exactly one JSON value, with no second value in the stream
- [ ] The decoded value is an object for every document except `list`, `ready` and `blocked`, whose documents are top-level arrays
- [ ] `changed`, `roots`, `blocked_by`, `blocks`, `tags`, `refs`, `notes`, `children` and `by_priority` unmarshal to non-nil slices wherever they appear, never `null`
- [ ] Every `auto` value unmarshals as a Go `bool` and every `index` value as a number
- [ ] A bare task's JSON detail carries `type`, `tags`, `refs` and `description` and carries no `parent` or `closed`, unlike the same task's toon document
- [ ] A filtered JSON document carries exactly the selected keys
- [ ] `stats` JSON keeps its `{total, by_status, workflow, by_priority}` shape
- [ ] Both drivers iterate the same `conformanceDocs`, so adding an entry adds it to both formats
- [ ] Pretty is in neither driver
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

**Tests**:
- `"it parses every inventory document as one JSON value"` — each entry's `--json` stdout yields one value and then EOF
- `"it fails when a document is followed by a second value"` — a stream carrying two concatenated objects fails the driver's one-value check
- `"it parses task list documents as arrays"` — `list`, `ready` and `blocked` decode to `[]any`
- `"it parses every other document as an object"` — the rest decode to `map[string]any`
- `"it rejects a null list value"` — a decoded document carrying `"changed": null` fails `assertJSONInvariants`
- `"it requires auto to be a boolean"` — a string `auto` fails the invariant check
- `"it requires index to be a number"` — a string `index` fails the invariant check
- `"it carries json's always-present detail keys"` — a bare task's JSON detail has `type`, `tags`, `refs` and `description`
- `"it omits parent and closed from a bare task's json detail"` — neither key is present
- `"it carries exactly the selected keys in a filtered json document"` — the key set equals the selection
- `"it keeps the nested stats shape in json"` — `total`, `by_status`, `workflow` and `by_priority` are present
- `"it drives both formats from one inventory"` — the set of entry names the JSON driver runs equals the set the toon driver runs

**Edge Cases**:
- One inventory drives both formats, so a document present in one table and not the other is the drift this removes
- JSON's presence rules differ from toon's — `type`, `tags`, `refs` and `description` are always carried while `parent` and `closed` are `omitempty` — so assertions cannot be copied across from the toon side
- Lists unmarshal to non-nil empty slices and never `null`, across `changed`, `roots`, `blocked_by`, `blocks`, `tags`, `refs`, `notes`, `children` and `by_priority`
- `auto` is a JSON boolean and `index` a number; a quoted `"true"` or `"1"` passes a naive key-presence check and fails here
- Each document must parse as exactly one JSON value, which is what proves no command emits two concatenated documents
- The task-list documents are top-level arrays rather than objects, because `FormatTaskList` marshals a JSON array; the driver allows that for those three commands only
- `stats` JSON keeps its own nested `{total, by_status, workflow, by_priority}` shape, which §4.2 does not move
- Pretty is in neither driver because it has no parser, and its assertions stay golden strings

**Context**:
> §11: "JSON output for each listed command is parsed and asserted by decoded value." And: "Rewritten assertions check decoded values, not output text."
>
> §4.2: "A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list, and the §8 structured empty dep-tree form in place of today's English sentence… Each note also carries its 1-based index, for the reason the toon table does (§6.3)."
>
> §7.4 on why one value per stream is the load-bearing check: "`create` and `update` today print the full task detail and then append transition lines after it… A reader handed that stream sees a task-detail document with foreign lines stuck on the end. **Where a command produces both a record and status changes, the result is one document with the changes as a section inside it.** Making each section valid is not sufficient on its own; the stream must be one document."
>
> §11 on pretty: "Pretty keeps golden-string assertions. Pretty output has no parser, so a decoded-value assertion does not exist for it."
>
> `jsonTaskDetail` (`internal/cli/json_formatter.go:58-74`) always carries `type`, `tags`, `refs` and `description` and puts `omitempty` on `parent` and `closed` — presence rules that differ from the toon document's, which omits `type`, `tags`, `refs` and `description` when empty. `toJSONRelated` documents that it "Always returns a non-nil empty slice to ensure JSON `[]` instead of `null`", which is the invariant the walker checks across every list key.
>
> The specification says each listed command's JSON is parsed and asserted by decoded value but does not fix the shape of the check. "Exactly one JSON value, then EOF" is chosen because the top-level shape is not uniform — `FormatTaskList` marshals an array while every other document marshals an object — and the property §7.4 actually requires is that the stream holds one document, not that it holds an object.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §4.2, §7.4, §3.1

## free-text-round-trip-6-6

### Task 6: No toon or JSON assertion pins a full-output string

**Problem**: §11's last requirement is that no byte-level pinning is kept in the machine formats — golden strings pin the exact output shape, and they are the mechanism that rotted into the defect this work undoes, a test comparing a wrong string to the same wrong string. Phases 1 to 5 rewrote what they each broke and Tasks 1 to 5 replaced what they superseded, but shape-pinned checks survive where the output did not change. `internal/cli/format_integration_test.go` still tells toon from pretty by matching `tasks[` (line 239) and checks the empty toon list by matching `tasks[0]` (line 554), and the empty JSON list is checked by comparing rendered text to `"[]"` (`internal/cli/json_formatter_test.go:55-60` and the JSON branch of the same empty-list table). Each is a document's shape used as a probe, and each passes as long as the shape is whatever it happens to be.

**Solution**: Convert the surviving format probes and rendered-text comparisons in toon and JSON tests to decoded checks, record the one class of text assertion that stays because decoding cannot express it, leave pretty's goldens and the non-document byte assertions alone, and state greps that re-run the check.

**Outcome**: No toon or JSON test compares a rendered document against a string literal, every pretty golden string is still in place, and two stated greps re-run the sweep.

**Do**:
1. `internal/cli/format_integration_test.go` — replace the toon list probes at lines ~239 and ~554 with `decodeToonDoc` followed by a `tasks` key check: present and carrying the seeded rows for the populated case, an empty list for the empty one.
2. `internal/cli/format_integration_test.go` (the JSON branch of the empty-list table) and `internal/cli/json_formatter_test.go:55-60` — replace the `"[]"` text comparisons with `json.Unmarshal` into `[]any` asserting a non-nil zero-length slice.
3. Sweep `internal/cli/*_test.go` for any remaining comparison of rendered toon or JSON output against a string literal and convert each to a decoded-value assertion, leaving pretty's assertions, the `--quiet` bare-ID assertions and the bare-value byte assertions as they are.
4. Keep the count-zero section header comparisons — `tasks[0]{…}`, `notes[0]{…}`, `changed[0]{…}`, `blocked_by[0]{…}`, `blocks[0]{…}`, `children[0]{…}`, `dep_tree[0]{…}` — and confirm each sits beside a decoded assertion that the same section decodes to an empty list.
5. Record the sweep's result with two greps that can be re-run: `grep -rn 'Contains(stdout, "task{\|Contains(stdout, "tasks\[\|Contains(stdout, "changed\[\|Contains(stdout, "dep_tree\[\|Contains(stdout, "summary{' internal/cli/*_test.go` returns nothing, and `grep -rn '"\[\]"' internal/cli/*_test.go` returns nothing.

**Acceptance Criteria**:
- [ ] No toon or JSON test compares a whole rendered document against a string literal
- [ ] Format detection in `format_integration_test.go` works by decoding and checking a key rather than by matching a section header
- [ ] The empty toon list is asserted by decoding to an empty `tasks` list
- [ ] The empty JSON list is asserted by unmarshalling to a non-nil zero-length slice
- [ ] The retained text assertions are exactly the count-zero section headers, each beside a decoded assertion of the same empty section
- [ ] Pretty's golden strings are all present: `No tasks found.`, `No dependencies found.`, `No dependencies.`, the single transition line and the box-drawing cascade tree
- [ ] `--quiet` bare-ID assertions and the bare-value byte assertions are unchanged
- [ ] `TestREADMEToonSamplesDecode` is unchanged, since it decodes rather than compares
- [ ] Both stated greps return nothing
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

**Tests**:
- `"it tells toon from pretty by a decoded key"` — the non-TTY and `--toon` branches decode and check a key; the pretty branch keeps its `ID:` check
- `"it decodes the empty toon list in the format table"` — the empty-list toon branch decodes to an empty `tasks` list
- `"it unmarshals the empty json list"` — the empty-list JSON branch yields a non-nil zero-length slice
- `"it keeps the count-zero header schema beside a decoded check"` — each retained header assertion has a decoded empty-list assertion in the same subtest
- `"it keeps pretty's golden strings"` — the five listed pretty assertions still match
- `"it keeps the quiet bare-id assertions"` — `create --quiet` and `list --quiet` still assert exact bytes
- `"it keeps the bare-value byte assertion"` — Task 4's exemption test still compares exact bytes
- `"it keeps the README samples decoding"` — `TestREADMEToonSamplesDecode` still resolves every anchor and decodes it

**Edge Cases**:
- Pretty's goldens stay, including `No tasks found.`, `No dependencies found.`, `No dependencies.`, the transition line and the cascade tree — pretty has no parser, and removing its goldens replaces its only form of assertion with nothing
- A substring probe used to tell one format from another is a pinned shape and becomes a decoded-key check; the pretty side of the same probe stays a substring match
- An assertion on one value inside a decoded document is not a pin, so only whole-output comparison goes
- The count-zero section headers stay as text: decoding collapses `notes[0]{index,text,created}:` and `notes[0]:` to the same empty list, so no decoded assertion can check the columns, and §8 requires the schema to be there
- `TestREADMEToonSamplesDecode` stays because it decodes the README's samples rather than comparing them to a string
- `--quiet` and bare-value byte assertions stay because that output is not a document — §3.1 exempts it and §9.2 fixes its exact bytes
- The sweep's result is checked by stated greps so it can be re-run rather than believed

**Context**:
> §11: "**No byte-level pinning is kept in the machine formats.** Golden strings pin the exact output shape, so a future change cannot reshape a section without a test noticing — but they are the mechanism that rotted into the defect this work undoes. Decoded-value assertions survive harmless reformatting while still failing when a section goes missing or a value is wrong. That trade is taken for toon and JSON."
>
> §11: "**Pretty keeps golden-string assertions.** Pretty output has no parser, so a decoded-value assertion does not exist for it; removing its golden strings would replace its only form of assertion with nothing."
>
> §11: "Rewritten assertions check decoded values, not output text. 'The notes section has two rows and the second row's text is X', not 'the output equals this blob'."
>
> §3.1 exempts the bare value and the zero-byte selection from the must-parse inventory; §9.2 fixes the bare value as "its own bytes followed by a single newline", which is a byte assertion rather than a pinned document.
>
> §8 requires the count-zero header: "the edges section carrying its count-zero header", and §7.4 requires `changed[0]{id,title,from,to,auto}:` on a `create` that moved nothing.
>
> The specification does not resolve the collision between "no byte-level pinning in the machine formats" and a count-zero header whose columns no decoder can recover. The reading taken is that the rule governs rendered documents, and that the count-zero headers stay as text assertions beside their decoded counterparts — the alternative leaves a shape the specification requires with nothing checking it at all. The retained set is listed in this task so the boundary is inspectable rather than a matter of judgement at each site.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §3.1, §7.4, §8, §9.2
