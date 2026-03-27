# Phase 1: Core Graph Logic, Command Wiring, and Pretty Formatter

## dep-tree-visualization-1-1 | approved

### Task 1: Command Wiring and Formatter Interface

**Problem**: The `dep tree` subcommand does not exist yet. Before any graph logic or formatting can be built, the command must be wired into the CLI dispatch, flag validation, help registry, and the `Formatter` interface must be extended with a `FormatDepTree` method. Without this wiring, subsequent tasks have no entry point.

**Solution**: Register `dep tree` as a new sub-subcommand of `dep` by extending `handleDep` and `qualifyCommand`, add `"dep tree"` to `commandFlags`, add a `FormatDepTree` method to the `Formatter` interface (with stubs on all implementations and `StubFormatter`), and register updated help text for the `dep` command.

**Outcome**: `tick dep tree` and `tick dep tree <id>` are accepted by the CLI dispatcher without unknown-flag errors, `tick help dep` shows the `tree` subcommand, the `Formatter` interface includes `FormatDepTree`, all existing tests pass including the drift-detection test (`TestCommandFlagsMatchHelp`) and flag validation tests.

**Do**:
1. **`internal/cli/format.go`** — Define the `DepTreeResult` data type that `FormatDepTree` will accept. This struct should support both full-graph and focused-view modes:
   - `Mode string` — `"full"` or `"focused"`
   - For full graph: `Roots []DepTreeNode` (tree of root tasks with children representing blocked tasks), `ChainCount int`, `LongestChain int`, `BlockedCount int`
   - For focused view: `Target DepTreeTask` (the focused task), `BlockedBy []DepTreeNode` (upstream tree), `Blocks []DepTreeNode` (downstream tree)
   - `DepTreeNode` struct: `Task DepTreeTask`, `Children []DepTreeNode`
   - `DepTreeTask` struct: `ID string`, `Title string`, `Status string`
2. **`internal/cli/format.go`** — Add `FormatDepTree(result DepTreeResult) string` to the `Formatter` interface.
3. **`internal/cli/format.go`** — Add stub `FormatDepTree` on `StubFormatter` returning `""` and `baseFormatter` returning `""`.
4. **`internal/cli/pretty_formatter.go`** — Add stub `FormatDepTree` on `PrettyFormatter` returning `""` (will be implemented in Task 4).
5. **`internal/cli/toon_formatter.go`** — Add stub `FormatDepTree` on `ToonFormatter` returning `""` (Phase 2).
6. **`internal/cli/json_formatter.go`** — Add stub `FormatDepTree` on `JSONFormatter` returning `""` (Phase 2).
7. **`internal/cli/flags.go`** — Add `"dep tree": {}` to `commandFlags`.
8. **`internal/cli/app.go`** — In `qualifyCommand`, extend the `dep` case to also match `"tree"` as a sub-subcommand: `case "add", "remove", "tree":`.
9. **`internal/cli/dep.go`** — In `handleDep`, add a `case "tree":` that calls `RunDepTree(dir, fc, fmtr, rest, a.Stdout)`. Create a placeholder `RunDepTree` function in a new file `internal/cli/dep_tree.go` that returns `nil` for now (Task 3 implements it fully).
10. **`internal/cli/help.go`** — Update the `dep` command's help entry: change Usage to `"tick dep <add|remove|tree> <task-id> [<blocked-by-id>]"`, update Description to mention the `tree` subcommand, and add a line explaining `tree` shows dependency relationships.

**Acceptance Criteria**:
- [ ] `commandFlags` contains `"dep tree"` with an empty flag set
- [ ] `qualifyCommand("dep", ["tree", "tick-abc123"])` returns `("dep tree", ["tick-abc123"])`
- [ ] `Formatter` interface includes `FormatDepTree(DepTreeResult) string`
- [ ] All three formatter implementations (`PrettyFormatter`, `ToonFormatter`, `JSONFormatter`) and `StubFormatter` compile with the new method
- [ ] `tick dep tree` dispatches without "unknown command" error (placeholder returns nil)
- [ ] `tick dep tree --unknown` returns unknown flag error referencing `"dep tree"` and `tick help dep`
- [ ] `tick help dep` output mentions `tree`
- [ ] `TestCommandFlagsMatchHelp` passes (no drift between commandFlags and help)
- [ ] All existing tests pass (`go test ./...`)

**Tests**:
- `"it qualifies dep tree as a two-level command"` — `qualifyCommand("dep", []string{"tree", "tick-abc"})` returns `"dep tree"`, `["tick-abc"]`
- `"it qualifies dep tree with no args"` — `qualifyCommand("dep", []string{"tree"})` returns `"dep tree"`, `[]`
- `"it rejects unknown flag on dep tree"` — `ValidateFlags("dep tree", []string{"--unknown"}, commandFlags)` returns error containing `"dep tree"` and help reference to `"dep"`
- `"it accepts global flags on dep tree"` — `ValidateFlags("dep tree", []string{"--verbose"}, commandFlags)` returns nil (global flags passthrough)
- `"it dispatches dep tree without error"` — `App.Run([]string{"tick", "dep", "tree"})` with a setup tick project returns exit code 0
- `"it shows tree in dep help text"` — `tick help dep` output contains `"tree"`
- `"it does not break existing dep add/remove dispatch"` — existing dep add/remove commands still work

**Edge Cases**: None for this wiring task.

**Context**:
> The specification states: `tick dep tree [id]` with no command-specific flags — only global flags apply. The `tree` subcommand is a new third sub-subcommand alongside `add` and `remove` under `dep`. The `qualifyCommand` function currently only matches `"add"` and `"remove"` as known sub-subcommands for `dep` — it must be extended to also match `"tree"`.

**Spec Reference**: `.workflows/dep-tree-visualization/specification/dep-tree-visualization/specification.md` — "Command Structure" and "Formatter Integration" sections.

## dep-tree-visualization-1-2 | approved

### Task 2: Dep Tree Data Model and Graph-Walking Algorithm

**Problem**: There is no logic to walk the dependency graph and produce the tree structures needed for rendering. The `task.Task` model stores `BlockedBy` (upstream dependencies) but there is no reverse index (what a task blocks downstream) and no algorithm to walk chains transitively, identify connected components, compute longest paths, or build the `DepTreeResult` data structure defined in Task 1.

**Solution**: Implement pure functions in a new file `internal/cli/dep_tree_graph.go` that take a `[]task.Task` and produce `DepTreeResult` for both modes. Build a reverse adjacency map (`blocks` direction) from the `BlockedBy` fields. For full-graph mode, identify root tasks (tasks that block others but have empty `BlockedBy`), walk downstream recursively to build trees, count connected components, and compute the longest path. For focused mode, walk upstream (`BlockedBy`) and downstream (`blocks`) transitively from a target task. Diamond dependencies are duplicated — no visited-set deduplication.

**Outcome**: `BuildFullDepTree(tasks []task.Task) DepTreeResult` and `BuildFocusedDepTree(tasks []task.Task, targetID string) (DepTreeResult, error)` are fully tested pure functions that produce correct `DepTreeResult` values for all graph topologies including empty graphs, single chains, diamond patterns, and isolated tasks.

**Do**:
1. **`internal/cli/dep_tree_graph.go`** — Create this new file with the graph-walking functions:
   - `buildBlocksIndex(tasks []task.Task) map[string][]string` — invert `BlockedBy` into a `taskID -> []blockedTaskIDs` map (the "blocks" direction). Also build a `taskByID map[string]*task.Task` lookup.
   - `BuildFullDepTree(tasks []task.Task) DepTreeResult` — Full graph mode:
     a. Build the blocks index and task lookup.
     b. Identify all tasks that participate in any dependency relationship (have non-empty `BlockedBy` or appear in someone's `BlockedBy`). Tasks with zero dependency involvement are omitted.
     c. Identify root tasks: tasks that appear in the blocks index (they block something) AND have empty `BlockedBy` (nothing blocks them).
     d. For each root, recursively walk downstream via the blocks index to build `DepTreeNode` trees. Diamond dependencies are duplicated — no visited set.
     e. Count connected components (chains): use union-find or BFS/DFS on the undirected dependency graph among participating tasks.
     f. Compute longest path: for each root, measure the maximum depth of the downstream tree (number of edges).
     g. Count blocked tasks: tasks with at least one `BlockedBy` entry.
     h. Return `DepTreeResult{Mode: "full", Roots: roots, ChainCount: N, LongestChain: M, BlockedCount: B}`.
   - `BuildFocusedDepTree(tasks []task.Task, targetID string) (DepTreeResult, error)` — Focused view mode:
     a. Find the target task by ID. Return error if not found.
     b. Build task lookup and blocks index.
     c. Walk upstream: recursively follow `BlockedBy` from the target to build the "Blocked by" tree. Each blocker becomes a `DepTreeNode` whose children are *its* blockers, transitively.
     d. Walk downstream: recursively follow the blocks index from the target to build the "Blocks" tree. Each blocked task becomes a `DepTreeNode` whose children are what *it* blocks, transitively.
     e. If the target has no dependencies in either direction, return `DepTreeResult` with empty `BlockedBy` and `Blocks` slices (the handler will render "No dependencies.").
     f. Return `DepTreeResult{Mode: "focused", Target: targetTask, BlockedBy: upstreamNodes, Blocks: downstreamNodes}`.
   - Helper: `walkDownstream(taskID string, blocksIndex map[string][]string, taskLookup map[string]task.Task) []DepTreeNode` — recursive, no dedup.
   - Helper: `walkUpstream(taskID string, taskLookup map[string]task.Task) []DepTreeNode` — recursively follows `BlockedBy`, no dedup.

2. **`internal/cli/dep_tree_graph_test.go`** — Write tests using in-memory `[]task.Task` slices (no store, no disk). Test helper to create minimal tasks with IDs, titles, statuses, and BlockedBy fields.

**Acceptance Criteria**:
- [ ] `BuildFullDepTree` with no dependency-participating tasks returns `DepTreeResult` with empty `Roots`, `ChainCount: 0`, `LongestChain: 0`, `BlockedCount: 0`
- [ ] `BuildFullDepTree` with a linear chain A blocks B blocks C returns 1 root (A), longest chain 2, 1 chain, 2 blocked
- [ ] `BuildFullDepTree` with a diamond (A blocks B and C, both block D) returns 1 root (A), longest chain 2, 1 chain, 3 blocked; D appears twice in the tree (under B and under C)
- [ ] `BuildFullDepTree` omits tasks with no dependency relationships
- [ ] `BuildFocusedDepTree` for a mid-chain task shows both upstream and downstream trees
- [ ] `BuildFocusedDepTree` for a task with no dependencies returns empty BlockedBy and Blocks
- [ ] `BuildFocusedDepTree` for a nonexistent ID returns an error
- [ ] Diamond dependencies appear duplicated in tree output (no deduplication)
- [ ] All tests pass (`go test ./internal/cli -run TestBuildFullDepTree` and `TestBuildFocusedDepTree`)

**Tests**:
- `"it returns empty result for project with no dependencies"` — all tasks have empty `BlockedBy`, result has empty Roots, zero counts
- `"it returns empty result when task list is empty"` — nil/empty input
- `"it builds a single linear chain"` — A->B->C: 1 root (A), children [B], B's children [C], longest 2, chains 1, blocked 2
- `"it builds multiple independent chains"` — A->B and C->D: 2 roots, longest 1, chains 2, blocked 2
- `"it duplicates diamond dependency without deduplication"` — A->B, A->C, B->D, C->D: root A has children B and C; B has child D, C also has child D
- `"it omits tasks with no dependency relationships"` — task E (no BlockedBy, not in anyone's BlockedBy) does not appear in Roots
- `"it handles task blocked by multiple roots"` — A->C, B->C: 2 roots (A, B), both have child C
- `"it computes longest chain across multiple chains"` — chain of length 3 and chain of length 1: longest is 3
- `"it counts blocked tasks correctly"` — tasks with at least one BlockedBy entry
- `"it builds focused upstream tree"` — for target C in chain A->B->C: BlockedBy tree is [B [A]]
- `"it builds focused downstream tree"` — for target A in chain A->B->C: Blocks tree is [B [C]]
- `"it builds focused view for mid-chain task"` — for target B in A->B->C: BlockedBy [A], Blocks [C]
- `"it returns no dependencies for isolated task in focused mode"` — task with no BlockedBy and not blocking anything: empty BlockedBy and Blocks
- `"it returns error for nonexistent task ID in focused mode"` — error message contains the ID
- `"it duplicates diamond in focused downstream"` — focused on A where A->B, A->C, B->D, C->D: Blocks tree has B[D] and C[D]
- `"it handles focused view with only upstream dependencies"` — task at end of chain: has BlockedBy, empty Blocks
- `"it handles focused view with only downstream dependencies"` — root task: empty BlockedBy, has Blocks

**Edge Cases**:
- Diamond dependencies: task reachable via multiple paths is duplicated in the tree without deduplication, as specified.
- Task with no dependencies in focused mode: `BuildFocusedDepTree` returns `DepTreeResult` with empty `BlockedBy` and `Blocks` slices; the handler (Task 3) will check and render "No dependencies."
- Empty project with no dependencies: `BuildFullDepTree` returns a result with zero counts and empty Roots.
- Asymmetric focused view: a task may have only upstream or only downstream — the result will have one empty slice, and the formatter (Task 4) omits that section.

**Context**:
> The specification explicitly states: "Diamond dependencies (task reachable via multiple paths): Duplicate the task wherever it appears in the graph. No deduplication, no back-references, no special markers." This means the recursive walk must NOT use a visited set. The specification also states: "Tasks with zero dependencies (neither blocking nor blocked) are omitted" from the full graph view. The "Depth: Full transitive — walk the entire chain with no artificial cap" constraint means no depth limit. For the summary line: "chain = connected component of the dependency graph, longest = longest path measured in edges, blocked = tasks with at least one BlockedBy entry."

**Spec Reference**: `.workflows/dep-tree-visualization/specification/dep-tree-visualization/specification.md` — "Command Structure" (full graph and focused view definitions), "Rendering" (diamond dependencies, depth), and "Edge Cases" sections.

## dep-tree-visualization-1-3 | approved

### Task 3: RunDepTree Command Handler

**Problem**: The `RunDepTree` placeholder created in Task 1 does nothing. The command handler must read tasks from the store, determine which mode (full graph vs focused), call the graph-walking functions from Task 2, handle edge cases (no dependencies, invalid task ID), and pass the result through the formatter for output.

**Solution**: Implement `RunDepTree` in `internal/cli/dep_tree.go` following the existing handler pattern (`RunStats`, `RunShow`). It reads all tasks via `store.ReadTasks()`, determines the mode from the presence/absence of a positional ID argument, calls `BuildFullDepTree` or `BuildFocusedDepTree`, handles the "no dependencies" edge case with `FormatMessage`, and delegates to `fmtr.FormatDepTree` for rendering. This is a read-only operation — uses `ReadTasks()` with shared lock, no `Mutate`.

**Outcome**: `tick dep tree` reads all tasks, builds the full dependency graph, and outputs it via the formatter. `tick dep tree <id>` resolves the ID (partial match supported), builds the focused view, and outputs it. Edge cases (no deps, task not found, invalid ID) produce appropriate messages or errors.

**Do**:
1. **`internal/cli/dep_tree.go`** — Replace the placeholder `RunDepTree` with the full implementation:
   ```
   func RunDepTree(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error
   ```
   - If `fc.Quiet`, return nil immediately (consistent with other commands).
   - Open the store via `openStore(dir, fc)`, defer `store.Close()`.
   - Read all tasks via `store.ReadTasks()` (shared lock, read-only).
   - Determine mode:
     - If `len(args) == 0`: full graph mode.
     - If `len(args) >= 1`: focused mode. The first positional arg is the task ID. Normalize with `task.NormalizeID(args[0])`, then resolve partial ID via `store.ResolveID(args[0])`.
   - **Full graph mode**: Call `BuildFullDepTree(tasks)`. If result has empty `Roots` (no dependencies found), output `fmtr.FormatMessage("No dependencies found.")` and return. Otherwise output `fmtr.FormatDepTree(result)`.
   - **Focused mode**: Call `BuildFocusedDepTree(tasks, resolvedID)`. If error (task not found), return the error. If both `BlockedBy` and `Blocks` are empty, show the target task (ID + title + status) then output `fmtr.FormatMessage("No dependencies.")` and return. Otherwise output `fmtr.FormatDepTree(result)`.
   - Write output to `stdout` with trailing newline via `fmt.Fprintln`.

2. **`internal/cli/dep_tree_test.go`** — Integration tests using `setupTickProjectWithTasks` to create a .tick project with pre-built task data, then calling `RunDepTree` or `App.Run` and checking output. Use `--pretty` flag since test buffers are non-TTY.

**Acceptance Criteria**:
- [ ] `tick dep tree` with no dependencies outputs "No dependencies found."
- [ ] `tick dep tree` with dependencies produces non-empty output via `FormatDepTree`
- [ ] `tick dep tree <id>` with a valid task ID that has dependencies produces focused output via `FormatDepTree`
- [ ] `tick dep tree <id>` with a valid task ID that has no dependencies shows the task itself (ID + title + status) with "No dependencies."
- [ ] `tick dep tree <id>` with an invalid/nonexistent ID returns an error
- [ ] Partial ID matching works (e.g., `tick dep tree abc` resolves to `tick-abc123`)
- [ ] `--quiet` flag suppresses all output
- [ ] The handler uses `store.ReadTasks()` (read-only, shared lock) — no `Mutate`

**Tests**:
- `"it outputs no dependencies found for empty project"` — setup project with tasks but no BlockedBy, run `RunDepTree` with no args, output contains "No dependencies found."
- `"it outputs no dependencies found for project with no tasks"` — empty tasks.jsonl, output contains "No dependencies found."
- `"it outputs dep tree for project with dependencies"` — setup A blocks B, run full graph, output is non-empty and does not contain "No dependencies"
- `"it outputs focused view for task with dependencies"` — setup A blocks B blocks C, run focused on B, output is non-empty
- `"it outputs no dependencies for isolated task in focused mode"` — setup task with no deps, run focused on it, output contains task ID and title and "No dependencies."
- `"it returns error for nonexistent task ID"` — run focused on "tick-nonexist", error returned
- `"it resolves partial task ID"` — setup task tick-abc123, run focused with "abc", resolves correctly
- `"it suppresses output in quiet mode"` — `fc.Quiet = true`, output buffer is empty
- `"it returns error for ambiguous partial ID"` — setup tick-abc111 and tick-abc222, run focused with "abc", error about ambiguous match
- `"it handles focused view via full App.Run dispatch"` — `App.Run([]string{"tick", "--pretty", "dep", "tree", "<id>"})` exits 0 with correct output

**Edge Cases**:
- Invalid task ID in focused mode: `store.ResolveID` returns an error for IDs that don't match any task or match multiple tasks (ambiguous). The handler propagates this error.
- Task not found: `BuildFocusedDepTree` returns an error when the target ID is not in the task list. The handler returns this error, which `App.Run` displays via stderr.

**Context**:
> The specification states this is a "read-only operation via Store.Query()". However, examining the codebase, `Store.ReadTasks()` is the more appropriate method here since we need the full `[]task.Task` slice for in-memory graph walking (not SQL queries). `ReadTasks()` also uses a shared lock. The `Store.Query()` method provides a `*sql.DB` for SQL queries, but our graph algorithm operates on the task slice directly. Both are read-only with shared locks. The edge case "Task with no dependencies (focused mode): Show the task itself with 'No dependencies.'" and "No dependencies in project (full graph mode): 'No dependencies found.'" are handled with `FormatMessage`.

**Spec Reference**: `.workflows/dep-tree-visualization/specification/dep-tree-visualization/specification.md` — "Command Structure" (two modes), "Edge Cases" (no dependencies messages), and "Scope" (dependencies only, read-only).

## dep-tree-visualization-1-4 | approved

### Task 4: Pretty Formatter FormatDepTree

**Problem**: The `PrettyFormatter.FormatDepTree` is a stub returning `""`. The Pretty formatter must render the dependency tree with box-drawing characters, showing task ID + title (truncated to fit) + status on each line, with proper indentation for nested dependencies. It must handle both full-graph mode (root tasks with downstream trees and a summary line) and focused mode (labeled "Blocked by:" and "Blocks:" sections).

**Solution**: Implement `FormatDepTree` on `PrettyFormatter` in `internal/cli/pretty_formatter.go`. Reuse the existing box-drawing tree rendering pattern from `writeCascadeTree` (used by `FormatCascadeTransition`) but adapted for `DepTreeNode` structures. Each tree line renders as `{connector} {id}  {truncated_title} ({status})`. Full-graph mode lists each root task at the top level followed by its downstream tree, ending with a summary line. Focused mode renders with "Blocked by:" and "Blocks:" section headers, omitting empty sections.

**Outcome**: `PrettyFormatter.FormatDepTree` produces human-readable box-drawing tree output for both modes, with title truncation accounting for indentation depth, correct summary line for full graph, and labeled sections for focused view.

**Do**:
1. **`internal/cli/pretty_formatter.go`** — Replace the stub `FormatDepTree` with the full implementation:
   - **Full graph mode** (`result.Mode == "full"`):
     a. For each root in `result.Roots`, render the root task line: `{id}  {title} ({status})`.
     b. Recursively render the root's children with box-drawing indentation using `writeDepTreeNodes(b *strings.Builder, nodes []DepTreeNode, prefix string)`. Use the same `\u251c\u2500\u2500` (middle) and `\u2514\u2500\u2500` (last) connectors as `writeCascadeTree`, with `\u2502   ` (pipe + spaces) and `    ` (spaces) continuation prefixes.
     c. Each child line: `{prefix}{connector} {id}  {title} ({status})`.
     d. Separate root blocks with a blank line between them.
     e. End with a summary line: `\n{N} chains, longest: {M}, {B} blocked` using the `ChainCount`, `LongestChain`, `BlockedCount` fields from the result.
   - **Focused mode** (`result.Mode == "focused"`):
     a. Render the target task header: `{id}  {title} ({status})`.
     b. If `result.BlockedBy` is non-empty, render `\nBlocked by:` header, then the upstream tree with box-drawing indentation.
     c. If `result.Blocks` is non-empty, render `\nBlocks:` header, then the downstream tree with box-drawing indentation.
     d. Omit sections that have no content (asymmetric view).
   - **Title truncation**: Implement `truncateDepTreeTitle(title string, depth int) string` that truncates the title to fit the available width. Available width = 80 (default) minus indentation (depth * 4 chars for prefix) minus ID length (typically 11 chars for `tick-XXXXXX`) minus status display length minus formatting overhead (connectors, spaces, parens). If the remaining width is less than a reasonable minimum (e.g., 10 chars), truncate to that minimum with `...`. This ensures very deep chains still show meaningful title fragments.
   - **Helper**: `writeDepTreeNodes(b *strings.Builder, nodes []DepTreeNode, prefix string, depth int)` — recursive function similar to `writeCascadeTree` but for `DepTreeNode` structures.

2. **`internal/cli/pretty_formatter_test.go`** — Add tests for `FormatDepTree` in a new `TestPrettyFormatDepTree` test function. Construct `DepTreeResult` values directly (no store needed) and verify string output.

**Acceptance Criteria**:
- [ ] Full graph output shows root tasks at top level with box-drawing tree indentation for blocked tasks
- [ ] Full graph output ends with summary line: `{N} chains, longest: {M}, {B} blocked`
- [ ] Focused output shows "Blocked by:" header with upstream tree when task has blockers
- [ ] Focused output shows "Blocks:" header with downstream tree when task blocks others
- [ ] Asymmetric focused view omits empty sections (only "Blocked by:" or only "Blocks:")
- [ ] Each tree line shows `{id}  {title} ({status})` format
- [ ] Long titles are truncated with `...` to fit available width after indentation
- [ ] Box-drawing characters are correctly nested (pipe continuation for non-last items, spaces for last items)
- [ ] Diamond dependencies appear as duplicate entries in the tree
- [ ] All tests pass (`go test ./internal/cli -run TestPrettyFormatDepTree`)

**Tests**:
- `"it renders single linear chain in full graph mode"` — A blocks B: root A with child B, summary "1 chains, longest: 1, 1 blocked"
- `"it renders multiple roots in full graph mode"` — A blocks B, C blocks D: two root sections separated by blank line, summary "2 chains, longest: 1, 2 blocked"
- `"it renders diamond dependency duplicated in full graph mode"` — A blocks B and C, both block D: D appears under both B and C in the tree
- `"it renders deep chain with correct indentation"` — A->B->C->D: nested box-drawing with increasing indentation
- `"it renders summary line with correct counts"` — verify chain count, longest, and blocked values appear in summary
- `"it renders focused view with both sections"` — mid-chain task B: output has "Blocked by:" with A tree and "Blocks:" with C tree
- `"it renders focused view with only blocked-by section"` — leaf task C blocked by B: output has "Blocked by:" but no "Blocks:" section
- `"it renders focused view with only blocks section"` — root task A blocking B: output has "Blocks:" but no "Blocked by:" section
- `"it omits empty sections in asymmetric focused view"` — focused on root: output does not contain "Blocked by:"
- `"it truncates long titles with ellipsis"` — task with a 100-char title at depth 3: title is truncated with "..."
- `"it uses box-drawing characters for tree structure"` — output contains `\u251c\u2500\u2500` and `\u2514\u2500\u2500`
- `"it shows task ID and status in each line"` — each task line contains the ID and status in parentheses
- `"it renders target task header in focused mode"` — first line of focused output is the target task

**Edge Cases**:
- Title truncation to fit available width: at very deep nesting levels (e.g., depth 10+), the available width for the title shrinks significantly. The truncation function should ensure a minimum title display (e.g., at least 10 chars or the full title if shorter), never producing an empty title.
- Very deep chains with cumulative indentation: the prefix grows by ~4 chars per level. At depth 20, that is 80 chars of indentation alone. In practice, Tick projects will not hit this (spec notes: "Tick projects won't realistically hit problematic depths"), but the rendering should degrade gracefully by showing at minimum the task ID and status even if the title is fully truncated.
- Asymmetric focused view with only one section: if a task has blockers but blocks nothing, only the "Blocked by:" section renders. If it blocks others but has no blockers, only the "Blocks:" section renders. The implementation must not render empty section headers.

**Context**:
> The specification states: "Box-drawing characters (three-bar, angle, pipe) for the pretty format." and "Inline metadata per task: ID + title (truncated to fit terminal width) + status." The existing `writeCascadeTree` in `pretty_formatter.go` already uses this same box-drawing pattern for cascade transitions and should serve as the template for the dep tree rendering. The specification also says "Focused view section headers: a 'Blocked by:' header followed by the upstream tree, then a 'Blocks:' header followed by the downstream tree." and "Asymmetric focused view: only show sections that have content." For title truncation: "Truncate titles to fit available width after accounting for indentation + ID + status." Using 80 chars as the default width is reasonable since terminal width detection is not required by the spec and the existing `maxListTitleLen` constant (50 chars) shows a similar truncation approach already exists.

**Spec Reference**: `.workflows/dep-tree-visualization/specification/dep-tree-visualization/specification.md` — "Rendering" (box-drawing characters, inline metadata, title truncation, diamond duplication), "Formatter Integration" (Pretty format details), and "Edge Cases" (asymmetric view, very deep graphs, terminal width).
