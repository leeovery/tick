# Plan: Dep Tree Visualization

## Phase 1: Core Graph Logic, Command Wiring, and Pretty Formatter
status: draft

**Goal**: Deliver working `tick dep tree` and `tick dep tree <id>` with Pretty output, including the graph-walking algorithm, data structures, command dispatch, flag validation, help registration, and the Pretty formatter with box-drawing tree rendering.

**Acceptance**:
- [ ] `tick dep tree` shows all dependency chains with root tasks at top level and blocked tasks indented, using box-drawing characters in Pretty format
- [ ] `tick dep tree` prints summary line: `{N} chains, longest: {M}, {B} blocked`
- [ ] `tick dep tree <id>` shows "Blocked by:" upstream tree and "Blocks:" downstream tree with full transitive depth in Pretty format
- [ ] Diamond dependencies (task reachable via multiple paths) are duplicated in the tree output without deduplication
- [ ] Each tree line shows task ID + title (truncated to fit) + status
- [ ] `dep tree` is registered in commandFlags, qualifyCommand, and help, with existing tests (flag validation, help drift) passing
- [ ] Tasks with no dependency relationships are omitted from full graph output
- [ ] All existing tests pass (`go test ./...`)

## Phase 2: Toon and JSON Formatters with Edge Cases
status: draft

**Goal**: Complete all three formatter implementations and handle every edge case defined in the specification.

**Acceptance**:
- [ ] Toon format: full graph outputs `dep_tree[N]{from,to}:` with one edge per line; focused mode outputs separate `blocked_by[N]{from,to}:` and `blocks[N]{from,to}:` sections
- [ ] JSON format: structured output with nodes and edges (or nested tree), matching the project's JSON conventions
- [ ] Task with no dependencies in focused mode shows the task itself with "No dependencies."
- [ ] Empty project (no dependencies at all) in full graph mode shows "No dependencies found."
- [ ] Asymmetric focused view: only shows sections with content (omits empty "Blocked by:" or "Blocks:" sections)
- [ ] All three formatters produce correct output for diamond dependencies, deep chains, and wide graphs
- [ ] All existing tests pass (`go test ./...`)
