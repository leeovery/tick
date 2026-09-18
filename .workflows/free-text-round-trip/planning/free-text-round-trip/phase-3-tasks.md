# Phase 3: Stats and dependency-tree documents — 6 tasks

## free-text-round-trip-3-1

### Task 1: Stats counts become top-level named fields

**Problem**: `ToonFormatter.FormatStats` (`internal/cli/toon_formatter.go:110-136`) renders the seven counts through `encodeToonSingleObject("stats", summary)`, which marshals a struct as a one-element array and then deletes the `[1]` from the header with `strings.Replace` (`internal/cli/toon_formatter.go:369-379`). The result is `stats{total,open,in_progress,done,cancelled,ready,blocked}:` followed by an indented row — a table header with no table beneath it, which a standard TOON reader rejects on the first line of the document. `tick stats` is on the must-parse inventory and today decodes on none of its branches. It is one of the three hand-edited single-object sites §5.1 names; Phase 1 removed the first.

**Solution**: Emit the counts as top-level named fields produced by `toon.MarshalString`, reusing the `encodeToonFields` helper Phase 1 added, and leave the `by_priority` table exactly as the library already writes it.

**Outcome**: `tick stats` toon output decodes to a map carrying `total`, `open`, `in_progress`, `done`, `cancelled`, `ready` and `blocked` as numbers beside the `by_priority` list, with no `stats` key anywhere in it.

**Do**:
1. `internal/cli/toon_formatter.go` — in `FormatStats`, replace the `toonStatsSummary` value and its `encodeToonSingleObject("stats", …)` call with `encodeToonFields` over seven `toon.Field`s in the order `total`, `open`, `in_progress`, `done`, `cancelled`, `ready`, `blocked`, taking their values from the `Stats` argument.
2. `internal/cli/toon_formatter.go` — delete the `toonStatsSummary` struct.
3. `internal/cli/toon_formatter.go` — in `FormatStats`, drop empty strings from `sections` before `strings.Join(sections, "\n\n")`, as `FormatTaskDetail` does.
4. `internal/cli/toon_formatter_test.go:254-281` — rewrite `"it formats stats with all counts"` to decode the output with `decodeToonDoc` and assert each count's value and the absence of a `stats` key; leave `"it formats by_priority with 5 rows including zeros"` (line 283) asserting the same section shape it asserts today.
5. `internal/cli/stats_test.go:250-297` — rewrite `"it formats stats in TOON format"` to decode stdout and assert decoded values, and add a subtest running `tick stats` on a project seeded with no tasks.

**Acceptance Criteria**:
- [ ] `FormatStats` output decodes without error via `toon.DecodeString`
- [ ] The decoded document carries `total`, `open`, `in_progress`, `done`, `cancelled`, `ready` and `blocked` as top-level keys and carries no `stats` key
- [ ] Emitted field order is `total`, `open`, `in_progress`, `done`, `cancelled`, `ready`, `blocked`, unchanged from the old row order
- [ ] A count of zero is emitted with its name, not omitted
- [ ] `by_priority` still decodes to a five-element list of `{priority,count}` objects in priority order 0 to 4, including zero counts
- [ ] `tick stats` on a project with no tasks decodes with every count reading `0` and `by_priority` carrying its five rows
- [ ] Pretty and JSON `stats` output are byte-identical to before this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it emits stats counts as top-level named fields"` — decoded document carries all seven counts with their values and no `stats` key
- `"it emits a zero count rather than omitting it"` — a `Stats` with `InProgress: 0` decodes with `in_progress` present and equal to `float64(0)`
- `"it keeps the by_priority table unchanged"` — decoded `by_priority` has five entries, `priority` 0 to 4 in order, counts matching `Stats.ByPriority`
- `"it decodes stats for a project with no tasks"` — end-to-end `tick stats`; every count decodes as `float64(0)` and `by_priority` still has five rows
- `"it decodes counts as numbers"` — decoded `total` has Go type `float64`, not `string`
- `"it leaves pretty stats output unchanged"` — the existing golden pretty assertions still match
- `"it leaves json stats output unchanged"` — the existing nested-structure assertions still match

**Edge Cases**:
- A zero count must still be emitted — do not reach for `omitempty` struct tags, which would drop `in_progress: 0` and make the schema depend on the data
- A project with no tasks emits all-zero counts plus `by_priority`'s five rows, so the document's shape never varies
- `by_priority`'s shape and row order are untouched — it is already library-written and §1's table lists it as one of the sections that parses today
- Decoded numbers arrive as `float64` through the generic decoder, so comparisons are against `float64(n)`
- No `stats` key in the decoded document: the wrapper is removed, not renamed
- Pretty and JSON `stats` are unchanged — JSON's `{total, by_status, workflow, by_priority}` nesting is its own shape and §4.2 moves JSON with toon only where toon's answer changed

**Context**:
> §5.1: "Three places in the output describe one thing rather than a list of things: the task's own fields at the head of the task-detail document (§3.1), the counts summary in `tick stats`, and the chains/longest/blocked summary in `tick dep tree`. All three are malformed… The cause is a hand-edit. The value is marshalled as a one-element array and the `[1]` is then deleted from the header with a string replace to make it read as singular — `buildTaskSection` does it inline… and `encodeToonSingleObject` does it generically for stats and the dep-tree summary."
>
> §5.2: "**The same treatment applies to the other two single-object sites**: `tick stats`' counts and the dep-tree chains/longest/blocked summary become top-level named fields beside their tables."
>
> §5.3 rejected the one-row table alternative: "Named fields put every value beside its name, so adding or reordering a field cannot break a positional read and a value containing a comma is safe without relying on quoting discipline." §5.4 rejected a wrapping key for the same reasons it rejected one on the task document.
>
> §4.1 keeps pretty unchanged; §4.2 moves JSON with toon only for the `changed` list and the structured empty dep-tree form, neither of which is stats.
>
> Verified against `github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c`: the seven fields encoded through a flat `toon.NewObject` and joined to the `by_priority` section by a blank line produce
> ```
> total: 47
> open: 12
> in_progress: 0
> done: 28
> cancelled: 0
> ready: 8
> blocked: 4
>
> by_priority[5]{priority,count}:
>   0,0
>   1,1
>   2,2
>   3,3
>   4,4
> ```
> which decodes to one `map[string]any` carrying all seven counts as `float64` plus `by_priority` as a `[]any` of maps.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §5.1, §5.2, §5.3, §5.4, §3.1, §4.1

## free-text-round-trip-3-2

### Task 2: The full dep-tree summary becomes top-level named fields

**Problem**: `ToonFormatter.formatFullDepTree` (`internal/cli/toon_formatter.go:187-205`) closes the full-graph document with `encodeToonSingleObject("summary", summary)`, producing `summary{chains,longest,blocked}:` and an indented row. It is the third and last of §5.1's hand-edited single-object headers, and a reader that got past the edge list fails on it. `encodeToonSingleObject` itself (`internal/cli/toon_formatter.go:369-379`) is the generic header-surgery helper — once stats (Task 1) and this site stop calling it, the last code path that strips a count from a section header can go with it.

**Solution**: Emit `chains`, `longest` and `blocked` as top-level named fields produced by `toon.MarshalString` beside the `dep_tree` edge section, and delete `encodeToonSingleObject` and the `toonDepTreeSummary` struct.

**Outcome**: `tick dep tree` full-graph toon output decodes to a map carrying the `dep_tree` edge list plus `chains`, `longest` and `blocked` as numbers, with no `summary` key, and no `strings.Replace` remains anywhere in the toon formatter.

**Do**:
1. `internal/cli/toon_formatter.go` — in `formatFullDepTree`, replace the `toonDepTreeSummary` value and its `encodeToonSingleObject("summary", …)` call with `encodeToonFields` over three `toon.Field`s in the order `chains`, `longest`, `blocked`, taking their values from `result.ChainCount`, `result.LongestChain` and `result.BlockedCount`. Keep the edge section first and the named fields after it.
2. `internal/cli/toon_formatter.go` — delete the `toonDepTreeSummary` struct and the `encodeToonSingleObject` function.
3. `internal/cli/toon_formatter.go` — update the `FormatDepTree` doc comment where it names the `summary{chains,longest,blocked}:` section.
4. `internal/cli/toon_formatter_test.go:799-826` — rewrite `"it renders summary as single-object section"` to decode the output with `decodeToonDoc` and assert `chains`, `longest` and `blocked` values plus the absence of a `summary` key; rename the subtest to match what it now asserts.
5. `internal/cli/toon_formatter_test.go` — add a subtest calling `f.FormatDepTree(DepTreeResult{})` directly (no `Roots`, no `Message`) and asserting the decoded document carries an empty `dep_tree` list and all three fields reading `float64(0)`.

**Acceptance Criteria**:
- [ ] `formatFullDepTree` output decodes without error via `toon.DecodeString` for a populated graph
- [ ] The decoded document carries `chains`, `longest` and `blocked` as top-level numbers and carries no `summary` key
- [ ] The `dep_tree` edge section is unchanged — same header, same rows, same order — and stays first in the document
- [ ] A summary field whose value is zero is still emitted with its name
- [ ] A `DepTreeResult` with no roots renders `dep_tree[0]{from,to}:` plus the three named fields, and the whole document decodes
- [ ] `DepTreeResult.Summary` is still populated by the graph builder and still rendered by `PrettyFormatter.formatFullDepTree`
- [ ] `grep -n 'encodeToonSingleObject\|toonDepTreeSummary\|toonStatsSummary' internal/cli/` returns nothing
- [ ] `grep -n 'strings.Replace' internal/cli/toon_formatter.go` returns nothing
- [ ] Pretty and JSON dep-tree output are byte-identical to before this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it emits the dep tree summary as top-level named fields"` — decoded document carries `chains`, `longest`, `blocked` with their values and no `summary` key
- `"it keeps the dep_tree edge section unchanged"` — decoded `dep_tree` list matches the edges the old assertion checked, in the same order
- `"it emits zero-valued summary fields"` — a result with `ChainCount: 0` decodes with `chains` present and equal to `float64(0)`
- `"it renders the emptied full document for a result with no roots"` — `FormatDepTree(DepTreeResult{})` decodes to an empty `dep_tree` list and three zero fields
- `"it keeps the summary prose for pretty"` — `PrettyFormatter.FormatDepTree` still ends with `result.Summary`, matching its existing golden assertion
- `"it leaves json dep tree output unchanged"` — the existing `{mode, roots, chains, longest, blocked}` assertions still match

**Edge Cases**:
- Zero-valued summary fields must still be emitted, so the document's schema never depends on the data
- No `summary` key in the decoded document: the wrapper is removed, not renamed
- `DepTreeResult.Summary` carries the human sentence (`1 chain, longest: 2, 2 blocked`) and stays on the struct and in the graph builder — pretty renders it and §4.1 keeps pretty as it is
- The emptied full-graph form is not reachable through the handler until Task 4 removes the early return, so it is unit-tested on the formatter here; construct the `DepTreeResult` with `Message` left empty so the dead message guard in `FormatDepTree` does not intercept it
- Deleting `encodeToonSingleObject` is what makes "no code path strips a count from a section header" true; the `strings.Replace` grep is the check
- The named fields sit after the edge section, which is where the summary sits today; a scalar field following a section decodes correctly and the specification does not fix an order for this document

**Context**:
> §5.2: "**The same treatment applies to the other two single-object sites**: `tick stats`' counts and the dep-tree chains/longest/blocked summary become top-level named fields beside their tables."
>
> §5.1: "The cause is a hand-edit… `encodeToonSingleObject` does it generically for stats and the dep-tree summary (`grep -n 'func encodeToonSingleObject' -A 10 internal/cli/toon_formatter.go` → `toon_formatter.go:371-379`). The result is a table header with no table beneath it, a shape TOON has no equivalent for."
>
> §4.3 records that the dep-tree empty messages "are set on the result in the shared graph builder… and consumed by all three formatters, so removing them at source would strip pretty's message too." The same holds for `Summary`: it is built in `BuildFullDepTree` (`internal/cli/dep_tree_graph.go:168-172`) and consumed by pretty, so it stays.
>
> §8 fixes the emptied form this task makes renderable: "What replaces them is the document the non-empty branch produces, emptied: the same fields, with the summary fields of §5.2 reading zero and the edges section carrying its count-zero header."
>
> Verified against `github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c`:
> ```
> dep_tree[2]{from,to}:
>   tick-a1b2,tick-c3d4
>   tick-c3d4,tick-f3e4
>
> chains: 1
> longest: 2
> blocked: 2
> ```
> decodes to one map carrying `dep_tree` as a `[]any` of `{from,to}` maps plus the three counts as `float64`. The emptied form — `dep_tree[0]{from,to}:` followed by the three zero fields — decodes to an empty `[]any` and three zeros.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §5.1, §5.2, §8, §4.1, §4.3

## free-text-round-trip-3-3

### Task 3: The focused dep tree names its task on both branches

**Problem**: `ToonFormatter.formatFocusedDepTree` (`internal/cli/toon_formatter.go:207-228`) answers two different documents. When the named task has dependencies it prints only edge sections, and only the non-empty ones — nothing in the document says which task was asked about, and a reader must discover whether `blocked_by`, `blocks` or both arrived. When it has none it prints a hand-built line, `fmt.Sprintf("%s  %s (%s)\n%s", …)`, followed by the English sentence `No dependencies.` — free-form text from the formatter whose purpose is machine-readable output, on the one branch an agent could not predict. §5.2 requires the identifying line to become top-level `id`, `title` and `status` fields on both branches, and §8 requires the empty branch to be the populated document emptied.

**Solution**: Open the focused document with `id`, `title` and `status` as top-level named fields produced by the library, then always emit both `blocked_by` and `blocks` edge sections, which carry a count-zero header when the direction is empty. Delete the hand-built identity line and the early return.

**Outcome**: `tick dep tree <id>` toon output decodes on both branches to a map carrying `id`, `title`, `status`, `blocked_by` and `blocks`, and the no-dependencies branch differs from the populated one only in that both edge lists are empty.

**Do**:
1. `internal/cli/toon_formatter.go` — rewrite `formatFocusedDepTree` to build three sections in order: `encodeToonFields` over `id`, `title`, `status` taken from `result.Target`; `buildEdgeSection("blocked_by", collectUpstreamEdges(result.Target.ID, result.BlockedBy))`; `buildEdgeSection("blocks", collectDownstreamEdges(result.Target.ID, result.Blocks))`. Join with `"\n\n"`.
2. `internal/cli/toon_formatter.go` — delete the `len(result.BlockedBy) == 0 && len(result.Blocks) == 0 && result.Message != ""` early return and the two `len(...) > 0` guards around the edge sections.
3. `internal/cli/toon_formatter.go` — update the `FormatDepTree` and `formatFocusedDepTree` doc comments where they describe omitting empty directions and rendering the target info plus message.
4. `internal/cli/toon_formatter_test.go` — rewrite the focused subtests as decoded-value assertions: `"it renders focused view with both directions"` (line 829), `"it omits blocked_by section when only downstream exists"` (861), `"it omits blocks section when only upstream exists"` (880), `"it duplicates edges in focused downstream for diamond"` (934) and `"it renders focused no-deps with task info and message"` (970). The two "omits" subtests become assertions that the named direction decodes to an empty list; rename them to match.
5. `internal/cli/toon_decode_test.go` — add focused conformance subtests driven through `App.Run([]string{"tick", "--toon", "dep", "tree", id})` on a seeded project: a task with both directions, one with only upstream, one with only downstream, and one with neither.

**Acceptance Criteria**:
- [ ] `formatFocusedDepTree` output decodes without error via `toon.DecodeString` on every branch
- [ ] The decoded document carries `id`, `title` and `status` as top-level keys equal to the target's values, on both the populated and the empty branch
- [ ] `blocked_by` and `blocks` are both present in every focused document; an empty direction renders `blocked_by[0]{from,to}:` / `blocks[0]{from,to}:` and decodes to an empty list
- [ ] The edge rows for a populated direction are unchanged — same `{from,to}` pairs in the same order, diamond duplication preserved
- [ ] No hand-built identity line and no `No dependencies.` string is produced by the toon formatter
- [ ] A target whose title contains a comma decodes back as one value, and the same for an ID
- [ ] `PrettyFormatter.FormatDepTree` output for a focused result is byte-identical to before this task on both branches
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it names the target as top-level fields on the populated branch"` — decoded `id`, `title`, `status` equal the target's values alongside both edge lists
- `"it names the target as top-level fields on the no-dependencies branch"` — same three keys with both edge lists empty
- `"it carries a count-zero blocked_by when only downstream exists"` — decoded `blocked_by` is an empty list and `blocks` carries its edges
- `"it carries a count-zero blocks when only upstream exists"` — the mirror case
- `"it keeps diamond duplication in the blocks direction"` — the four edges decode in the order the old assertion checked
- `"it quotes a target title containing a comma"` — decoded `title` equals the stored title byte-for-byte
- `"it emits no prose on the no-dependencies branch"` — the output contains no `No dependencies.` substring and decodes
- `"it decodes dep tree output for a task with both directions"` — end-to-end through `App.Run` with `--toon`
- `"it decodes dep tree output for a task with no dependencies"` — end-to-end through `App.Run` with `--toon`
- `"it leaves pretty focused output unchanged"` — the existing golden first-line and message assertions still match

**Edge Cases**:
- The hand-built `id  title (status)` line and the `No dependencies.` early return both go; the branch that produced them is deleted, not repaired
- Both `blocked_by` and `blocks` are always present with count-zero headers, which changes today's omit-when-empty rule on the populated branch — the reader stops having to discover which directions arrived
- The no-dependencies branch is the populated document emptied, not a separate shape
- A comma-bearing title or ID is quoted by the library through `encodeToonFields`, never pre-quoted by the formatter
- `collectUpstreamEdges` and `collectDownstreamEdges` return nil for an empty node slice, which `buildEdgeSection` already renders as the count-zero header
- Pretty's focused view keeps its target header line, its `Blocked by:` / `Blocks:` labels, its omit-when-empty behaviour and its `No dependencies.` sentence — none of that code is touched
- JSON's focused document still carries `message` and still omits empty directions until Task 5; that is the interim state this phase closes, not a regression to assert against

**Context**:
> §5.2: "Where `tick dep tree` names a task, the line identifying that task becomes top-level `id`, `title` and `status` fields in the same form, so a focused dependency document opens exactly as a task-detail document does. It is carried on both branches: today the identity appears only when the task has no dependencies, as a free-form line (`sed -n '209,212p' internal/cli/toon_formatter.go`), and the populated branch prints edge sections with nothing naming the task at all."
>
> §8: "`tick dep tree` currently answers `No dependencies found.` when nothing in the project is blocked, and a title line plus `No dependencies.` when a named task has no dependencies either way… **Both go, in toon and JSON; pretty keeps them (§4.3).** What replaces them is the document the non-empty branch produces, emptied: the same fields, with the summary fields of §5.2 reading zero and the edges section carrying its count-zero header. Where the caller named a task, the emptied document still identifies it exactly as the populated one does. Both empty branches take that shape… so an agent parses one document whether or not anything is blocked, and reads the counts to learn which it got."
>
> §8 on why this is not an exception to the prose rule: "the exemption covers confirmations of a command the caller issued, and 'no dependencies' is the answer to a query — the answer the caller ran the command to find out."
>
> §8 on the existing pattern: "An empty task list comes back as a structured empty section, and so does an empty edge set — `buildRelatedSection` and `buildNotesSection` both emit a count-zero header rather than prose. The two dep-tree prose branches are the exception, not the pattern."
>
> §4.3: "The named-task branch already reaches the formatters (`runFocusedDepTree` calls `FormatDepTree` unconditionally) and needs no handler change." `DepTreeResult.Message` stays populated by `BuildFocusedDepTree` (`internal/cli/dep_tree_graph.go:250-257`) because `PrettyFormatter.formatFocusedDepTree` renders it.
>
> Verified against `github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c`:
> ```
> id: tick-a1b2
> title: "Task, with comma"
> status: open
>
> blocked_by[0]{from,to}:
>
> blocks[0]{from,to}:
> ```
> decodes to one map carrying `id`, `status` and `title` as strings — the comma-bearing title as a single value — plus two empty `[]any`. The populated form with one edge in each direction decodes the same way with one `{from,to}` map per list.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §5.2, §8, §4.1, §4.3, §3.1

## free-text-round-trip-3-4

### Task 4: Nothing blocked anywhere returns the emptied document

**Problem**: When no task in the project blocks another, `runFullDepTree` (`internal/cli/dep_tree.go:33-43`) returns as soon as the root set is empty, printing `No dependencies found.` through `FormatMessage`. The full-graph result never reaches a dep-tree formatter on that branch, so the guards in `ToonFormatter.FormatDepTree` and `JSONFormatter.FormatDepTree` that look like they handle it are dead code. An agent asking which chains exist gets an English sentence on the one branch it could not predict, and §8 requires the emptied document instead. Routing the branch through the formatters has a consequence for pretty: `PrettyFormatter.formatFullDepTree` (`internal/cli/pretty_formatter.go:316-319`) returns `""` for zero roots, so it would silently print a blank line where it prints a sentence today.

**Solution**: Delete the handler's early return so every full-graph result reaches `FormatDepTree`, hand pretty its sentence explicitly inside `formatFullDepTree`, and delete the now-reachable-but-wrong message guard from the toon formatter.

**Outcome**: `tick dep tree` on a project with nothing blocked emits the full-graph document with an empty `dep_tree` section and the builder's real summary counts, decodable in toon, while pretty prints the same `No dependencies found.` bytes it prints today.

**Do**:
1. `internal/cli/dep_tree.go` — in `runFullDepTree`, delete the `len(result.Roots) == 0` early return and its `fmtr.FormatMessage` call, leaving `fmt.Fprintln(stdout, fmtr.FormatDepTree(result))` as the only output path.
2. `internal/cli/pretty_formatter.go` — in `formatFullDepTree`, return `result.Message` instead of `""` when `len(result.Roots) == 0`.
3. `internal/cli/toon_formatter.go` — in `FormatDepTree`, delete the `if result.Message != "" { return result.Message }` branch.
4. `internal/cli/dep_tree_test.go:153-183` — keep the two existing pretty subtests asserting `No dependencies found.`, and add toon subtests that decode stdout from `App.Run([]string{"tick", "--toon", "dep", "tree"})` for an empty project and for a project whose tasks carry no dependencies.
5. `internal/cli/dep_tree_test.go` — add a subtest seeding a two-task dependency cycle and asserting the decoded document carries an empty `dep_tree` list alongside the counts `BuildFullDepTree` computed.

**Acceptance Criteria**:
- [ ] `tick dep tree` toon output on an empty project decodes to a document with an empty `dep_tree` list and `chains`, `longest`, `blocked` all reading `float64(0)`
- [ ] `tick dep tree` toon output on a project whose tasks carry no dependencies decodes the same way
- [ ] `tick dep tree` toon output on a project that is a single dependency cycle decodes with an empty `dep_tree` list and the counts the builder produced — `blocked` reading 2 and `chains` reading 1 for a two-task cycle, not forced zeros
- [ ] `tick dep tree --pretty` stdout is byte-identical to before this task on both empty branches: `No dependencies found.` followed by one newline
- [ ] `tick dep tree --json` output is byte-identical to before this task
- [ ] `tick dep tree --quiet` prints nothing
- [ ] `grep -n 'FormatMessage' internal/cli/dep_tree.go` returns nothing
- [ ] `ToonFormatter.FormatDepTree` carries no `result.Message` branch
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it returns the emptied document for an empty project"` — decoded toon document has an empty `dep_tree` and three zero counts
- `"it returns the emptied document when no task has dependencies"` — same, with two unconnected tasks seeded
- `"it reports real counts when a cycle leaves no roots"` — two tasks blocking each other; decoded `dep_tree` is empty while `blocked` and `chains` carry the builder's values
- `"it still prints the no-dependencies sentence in pretty"` — stdout equals `"No dependencies found.\n"` for both empty branches
- `"it prints nothing under --quiet"` — stdout is empty
- `"it still renders the populated graph"` — a seeded chain decodes with its edges and non-zero counts, unchanged by the handler change

**Edge Cases**:
- Pretty returns `""` for zero roots today, so it must be handed its sentence explicitly or the branch silently prints a blank line; the sentence is already on `DepTreeResult.Message`
- Pretty's stdout must stay byte-identical: the handler prints `FormatMessage(result.Message)` today and `FormatDepTree(result)` after, and both must yield `No dependencies found.`
- A dependency cycle leaves zero roots while `blocked` and `chains` are non-zero, so the emptied document carries the builder's real counts and nothing forces them to zero
- Both the empty project and the unconnected-tasks project take this branch — the root set is empty in each case for a different reason
- `--quiet` returns from `RunDepTree` before the store is opened, so the branch is never reached
- JSON output is unchanged by this task: `JSONFormatter.FormatDepTree`'s own message guard produces the identical `{"message": …}` document the handler produced through `FormatMessage`. Task 5 removes it, and the JSON assertions belong there

**Context**:
> §4.3: "The dep-tree empty messages are not produced by a formatter at all. They are set on the result in the shared graph builder (`grep -n 'No dependencies' internal/cli/dep_tree_graph.go` → `dep_tree_graph.go:177`, `dep_tree_graph.go:255`) and consumed by all three formatters, so removing them at source would strip pretty's message too. On the nothing-blocked branch they never reach a dep-tree formatter at all: `runFullDepTree` returns as soon as the root set is empty, printing the sentence through `FormatMessage`… The guards that look like they handle it — `json_formatter.go:366` and `toon_formatter.go:180-182` — are dead code. **The change therefore lands in the handler, not in the formatters**, and it has a consequence for pretty: `PrettyFormatter.formatFullDepTree` returns `""` for zero roots, so once that branch routes through the formatters pretty must be handed its sentence explicitly or it silently prints nothing."
>
> §8: "**Both go, in toon and JSON; pretty keeps them (§4.3).** What replaces them is the document the non-empty branch produces, emptied… Both empty branches take that shape — nothing blocked anywhere, and a named task with no dependencies either way — so an agent parses one document whether or not anything is blocked, and reads the counts to learn which it got."
>
> §3.2 draws the boundary this lands on: "a confirmation may be prose; an **answer to a query** may not. 'No dependencies' is not a confirmation — it is the answer the caller ran the command to find out."
>
> §4.1: "What a terminal prints today is what it prints after this work — … the dep-tree prose on empty branches."
>
> `BuildFullDepTree` counts blocked tasks and chains over all participants regardless of whether any root exists (`internal/cli/dep_tree_graph.go:151-166`), and `internal/cli/dep_tree_graph_test.go:599-613` already fixes that a two-task cycle yields zero roots. The emptied document therefore reports what the builder found rather than a zeroed placeholder.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §8, §4.3, §4.1, §3.2, §3.1

## free-text-round-trip-3-5

### Task 5: JSON dep-tree documents drop their message forms

**Problem**: JSON still answers dep-tree queries with prose. `JSONFormatter.FormatDepTree` (`internal/cli/json_formatter.go:360-370`) returns `{"message": "No dependencies found."}` whenever the result carries a message, and `jsonDepTreeFocused` (`internal/cli/json_formatter.go:331-337`) carries `Message string json:"message,omitempty"`, set whenever both directions are empty. The same struct puts `,omitempty` on `BlockedBy` and `Blocks`, so an empty direction vanishes from the document entirely and `null` is what a consumer gets for a direction that was never assigned. A JSON consumer therefore branches on which keys arrived and reads an English sentence on the branch it could not predict — the defect §8 removes in toon, left standing in the format §4.2 requires to move with it.

**Solution**: Delete the message guard and the `Message` field, drop `omitempty` from both direction fields, and assign both unconditionally through `toJSONDepTreeNodes`, which already returns a non-nil slice.

**Outcome**: Both JSON dep-tree documents carry the same keys on every branch — `{mode, roots, chains, longest, blocked}` for the full graph and `{mode, target, blocked_by, blocks}` for a focused one — with every list rendering as `[]` rather than `null` and no `message` key anywhere.

**Do**:
1. `internal/cli/json_formatter.go` — in `FormatDepTree`, delete the `result.Message != ""` branch and its `jsonMessage` return.
2. `internal/cli/json_formatter.go` — remove the `Message` field from `jsonDepTreeFocused` and remove `,omitempty` from its `BlockedBy` and `Blocks` tags.
3. `internal/cli/json_formatter.go` — in `formatFocusedDepTreeJSON`, assign `obj.BlockedBy = toJSONDepTreeNodes(result.BlockedBy)` and `obj.Blocks = toJSONDepTreeNodes(result.Blocks)` unconditionally, deleting the three `len(...)` guards.
4. `internal/cli/json_formatter.go` — update the `FormatDepTree` and `jsonDepTreeFocused` doc comments where they describe the message-only form and the omitted directions.
5. `internal/cli/json_formatter_test.go:1474-1513` and `internal/cli/dep_tree_test.go:317-355` — rewrite the two no-deps JSON subtests to assert `message` is absent and both directions unmarshal to non-nil empty slices; add a full-graph subtest running `App.Run` with `--json` on a project with nothing blocked.

**Acceptance Criteria**:
- [ ] `tick dep tree --json` on a project with nothing blocked emits `{"mode":"full","roots":[],"chains":0,"longest":0,"blocked":0}` and parses as one object
- [ ] `tick dep tree <id> --json` for a task with no dependencies either way emits `mode`, `target`, `blocked_by` and `blocks`, with both directions `[]`
- [ ] `roots`, `blocked_by` and `blocks` unmarshal to non-nil empty slices, never `null`, on every branch
- [ ] No dep-tree JSON document carries a `message` key
- [ ] `target` stays a nested object carrying `id`, `title` and `status`
- [ ] `mode` still reads `full` or `focused`
- [ ] `JSONFormatter.FormatMessage` still renders `{"message": …}`, and `tick init` and the cache-rebuild message still emit it
- [ ] Toon and pretty dep-tree output are unchanged by this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it emits the emptied full document instead of a message"` — `--json` on a project with nothing blocked unmarshals to an object with `roots` empty, three zero counts and no `message`
- `"it emits both directions on a focused document with no dependencies"` — `blocked_by` and `blocks` are both present and empty, and `message` is absent
- `"it emits blocked_by and blocks as empty arrays never null"` — the unmarshalled values are non-nil `[]any` of length 0
- `"it emits roots as an empty array never null"` — same for the full document
- `"it keeps target as a nested object"` — `target.id`, `target.title` and `target.status` carry the task's values
- `"it keeps mode on both documents"` — `full` and `focused` respectively
- `"it still renders a message for init"` — `FormatMessage` output unmarshals to `{"message": …}`
- `"it leaves toon dep tree output unchanged"` — the toon assertions from Tasks 2 to 4 still pass

**Edge Cases**:
- `roots`, `blocked_by` and `blocks` are `[]` and never `null` — `toJSONDepTreeNodes` already allocates with `make`, so the fix is to call it unconditionally rather than to guard on length
- `omitempty` comes off both direction fields; leaving it would keep an empty direction invisible even once it is assigned
- `message` disappears from both dep-tree branches while `FormatMessage` and `jsonMessage` stay, because §3.2's prose commands (`init`, the cache-rebuild notice) still use them
- JSON's nested `target` object is not flattened into top-level fields: §5.2's named-field form answers TOON's malformed single-object header, which JSON never had, and §4.2 moves JSON with toon in content rather than in layout
- `mode` stays on both documents — it is how a consumer tells the two shapes apart and nothing in the specification removes it
- The full-graph branch only became reachable for JSON in Task 4; before that the handler short-circuited it

**Context**:
> §4.2: "A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list, and the §8 structured empty dep-tree form in place of today's English sentence, which reaches JSON as a bare `message` object from the command handler rather than from the dep-tree formatter (§4.3)."
>
> §8: "**Both go, in toon and JSON; pretty keeps them (§4.3).** What replaces them is the document the non-empty branch produces, emptied: the same fields, with the summary fields of §5.2 reading zero and the edges section carrying its count-zero header. Where the caller named a task, the emptied document still identifies it exactly as the populated one does."
>
> §3.2 keeps the prose exemption alive for other commands: "`dep add`, `dep remove`, `remove`, `init`, and the general-purpose messages… These are confirmations of a command the caller issued." `FormatMessage` is their renderer and stays.
>
> §11: "Rewritten assertions check decoded values, not output text," and "**No byte-level pinning is kept in the machine formats.** … That trade is taken for toon and JSON" — the rewritten subtests unmarshal rather than matching strings.
>
> The JSON formatter already treats empty collections this way elsewhere: `toJSONRelated` documents that it "Always returns a non-nil empty slice to ensure JSON `[]` instead of `null`", and `toJSONDepTreeNodes` is built the same way. The defect is the `len(...)` guards and the `omitempty` tags around it, not the converter.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §4.2, §8, §3.2, §5.2, §11

## free-text-round-trip-3-6

### Task 6: README dep-tree sample matches real output

**Problem**: The README's `dep tree` section prints a TOON sample (`README.md:300-309`) ending in the malformed `summary{chains,longest,blocked}:` header and its indented row — output the tool no longer produces after Task 2, and output no TOON reader ever accepted. It is the last of §12.1's five samples this work replaces. The README is live documentation someone reads to learn the tool, so leaving it describing output the tool does not produce is shipping a defect, and the sample is currently outside the decode test Phase 1 added, so nothing would catch it rotting again.

**Solution**: Replace the sample's body with output copied from the real binary and add its first line to the anchor list of `TestREADMEToonSamplesDecode`, so the block is located and decoded by the suite.

**Outcome**: The `tick dep tree` TOON sample in the README is byte-for-byte what the tool prints, and the suite fails if it stops parsing as TOON or disappears.

**Do**:
1. Build the binary to a scratch path, initialise a throwaway `.tick` project, and seed its `tasks.jsonl` with the three tasks the sample describes — `tick-a1b2` "Setup auth" done, `tick-c3d4` "Login endpoint" open blocked by `tick-a1b2`, `tick-f3e4` "Write tests" open blocked by `tick-c3d4` — then capture `tick --toon dep tree` and `tick --pretty dep tree`.
2. `README.md:302-308` — replace the TOON cell's body with the captured toon output verbatim, keeping the `$ tick dep tree` prompt line above it. The `summary{chains,longest,blocked}:` header and its `  1,2,2` row become the three lines `chains: 1`, `longest: 2`, `blocked: 2` beneath the unchanged `dep_tree[2]{from,to}:` section.
3. `README.md:287-295` — compare the Pretty cell against the captured pretty output and leave it as it is if they match; correct it only where they differ. The cell label and the prose paragraph above the table (`README.md:281`) stay.
4. `internal/cli/readme_samples_test.go` — add `dep_tree[2]{from,to}:` to the anchor list of `TestREADMEToonSamplesDecode`.
5. Confirm the scope boundary: `grep -n '^\$ tick stats' README.md` returns nothing and no `tick stats` sample is added; the `tick show`, `tick list`, transition and cascade samples corrected in Phases 1 and 2 are untouched.

**Acceptance Criteria**:
- [ ] The TOON dep-tree sample was copied from real tool output rather than hand-written
- [ ] Its summary is three top-level `chains:` / `longest:` / `blocked:` lines, and no `summary{` header remains anywhere in the README
- [ ] The `dep_tree[2]{from,to}:` edge section and its two rows are unchanged
- [ ] `TestREADMEToonSamplesDecode` locates the block by its first line and decodes it, and fails if the block is removed or made unparseable
- [ ] The Pretty cell still shows the box-drawing tree and the `1 chain, longest: 2, 2 blocked` sentence, matching real pretty output
- [ ] The prose introducing the section still describes the two modes and the summary line
- [ ] No `tick stats` sample is added
- [ ] The samples corrected in Phases 1 and 2 are byte-identical to what those phases left
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it decodes the README dep tree sample"` — the block anchored on `dep_tree[2]{from,to}:` parses as TOON
- `"it fails when an anchored sample is missing"` — the existing anchor-miss failure still covers the new anchor
- `"it has no single-object section header in the README"` — the document contains no `summary{`
- `"it keeps the earlier anchors decoding"` — the `tasks[N]`, `id:` and `changed[N]` anchors added in Phases 1 and 2 still resolve and decode

**Edge Cases**:
- The `summary{…}:` block becomes top-level `chains` / `longest` / `blocked` lines within the same fenced block, not a separate sample
- The sample is copied from real output; seeding the throwaway project's JSONL with the README's IDs and titles is what makes the captured bytes match the graph the Pretty cell describes
- The pretty companion block and the surrounding prose stay — pretty is unchanged by this phase
- The dep-tree sample is added as an anchor to `TestREADMEToonSamplesDecode` rather than getting its own test
- `dep_tree[2]{from,to}:` is unique among the README's fenced blocks, so the anchor index cannot collide with the `tasks[N]` or `changed[N]` anchors
- The block carries a `$ tick dep tree` prompt line, which the test's collector strips before indexing by first line
- No `tick stats` sample exists in the README, so none is added — §12.1's table does not list one and this work adds no documentation it does not owe
- The focused `tick dep tree <id>` document has no README sample either; §12.1 lists only the summary header, so none is added

**Context**:
> §12.1 lists this sample in the table of README output the work replaces:
>
> | Sample | Location | Why it changes |
> |---|---|---|
> | dep-tree summary header | `README.md:307` (inside `### dep`) | single-object section becomes top-level named fields (§5.2) |
>
> §12.1: the README "is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect."
>
> §5.2 is the change the sample reflects: the dep-tree chains/longest/blocked summary becomes top-level named fields beside its table.
>
> §4.1 keeps pretty unchanged, which is why the Pretty cell beside it is verified rather than rewritten.
>
> §12.1's remaining README work is elsewhere: the `tick show` and `tick list` samples landed in Phase 1, the transition and cascade samples in Phase 2, and `--field`/`--fields` and `--` are added in Phases 4 and 5.
>
> `TestREADMEToonSamplesDecode` was added by Phase 1 task free-text-round-trip-1-6 in `internal/cli/readme_samples_test.go`: it locates `README.md` via `testutil.FindRepoRoot(t)`, collects every fenced block with an empty info string, strips a leading `$ tick …` prompt line, indexes the blocks by their first remaining line, and fails both when an anchor has no matching block and when `toon.DecodeString` rejects one.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §12.1, §5.2, §8, §4.1
