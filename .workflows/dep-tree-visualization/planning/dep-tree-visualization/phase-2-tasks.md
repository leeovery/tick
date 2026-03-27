# Phase 2: Toon and JSON Formatters with Edge Cases

## dep-tree-visualization-2-1 | pending

### Task 1: ToonFormatter FormatDepTree

**Problem**: The `ToonFormatter.FormatDepTree` is a stub returning `""`. The toon format is the machine-parseable output consumed by AI agents, and it needs a flat edge-list representation of the dependency tree. Without this implementation, `tick dep tree` and `tick dep tree <id>` produce empty output when the toon format is active (the default for non-TTY contexts, which is the primary agent consumption path).

**Solution**: Implement `FormatDepTree` on `ToonFormatter` in `internal/cli/toon_formatter.go`. For full-graph mode, render a `dep_tree[N]{from,to}:` section listing every parent-to-child edge in the tree (one edge per line), plus a `summary{chains,longest,blocked}:` single-object section. For focused mode, render separate `blocked_by[N]{from,to}:` and `blocks[N]{from,to}:` sections for upstream and downstream edges respectively. Edges are extracted by walking the `DepTreeNode` trees recursively. Diamond dependencies naturally produce duplicate edges (same `to` appearing under different `from` values) since the tree already duplicates them. Asymmetric focused views omit the empty section entirely (do not render a `blocked_by[0]{from,to}:` or `blocks[0]{from,to}:` header for an empty direction).

**Outcome**: `ToonFormatter.FormatDepTree` produces correct, machine-parseable toon output for both modes. Full-graph mode outputs an edge list with summary. Focused mode outputs directional edge sections, omitting empty ones. All edge cases (diamond deps, asymmetric views, wide graphs) are handled correctly.

**Do**:
1. **`internal/cli/toon_formatter.go`** -- Define a `toonEdgeRow` struct for edge representation:
   ```go
   type toonEdgeRow struct {
       From string `toon:"from"`
       To   string `toon:"to"`
   }
   ```
2. **`internal/cli/toon_formatter.go`** -- Define a `toonDepTreeSummary` struct for the summary section:
   ```go
   type toonDepTreeSummary struct {
       Chains  int `toon:"chains"`
       Longest int `toon:"longest"`
       Blocked int `toon:"blocked"`
   }
   ```
3. **`internal/cli/toon_formatter.go`** -- Implement a helper function `collectEdges(parentID string, nodes []DepTreeNode) []toonEdgeRow` that recursively walks a `[]DepTreeNode` tree and collects edges. For each node in `nodes`, emit an edge `{From: parentID, To: node.Task.ID}`, then recursively collect edges from `node.Children` with `node.Task.ID` as the new parent. This naturally duplicates edges for diamond dependencies since the tree already has duplicated nodes.
4. **`internal/cli/toon_formatter.go`** -- Replace the stub `FormatDepTree` with the full implementation:
   - **Full graph mode** (`result.Mode == "full"`):
     a. Collect all edges by iterating `result.Roots` and for each root, calling `collectEdges(root.Task.ID, root.Children)`. The root itself is not an edge target (it is the starting point), so edges are only from root to its children and recursively deeper.
     b. Render edges using `encodeToonSection("dep_tree", edges)`. If no edges exist (should not happen since handler checks for empty roots), render `dep_tree[0]{from,to}:`.
     c. Render the summary as a single-object section using `encodeToonSingleObject("summary", toonDepTreeSummary{Chains: result.ChainCount, Longest: result.LongestChain, Blocked: result.BlockedCount})`.
     d. Join the two sections with `"\n\n"`.
   - **Focused mode** (`result.Mode == "focused"`):
     a. Collect upstream edges from `result.BlockedBy` using `collectEdges(result.Target.ID, result.BlockedBy)`. In the upstream direction, edges represent "target is blocked by X, X is blocked by Y", so from=child, to=parent. However, for consistency with the tree structure (where BlockedBy nodes are the blockers of the target), render edges as `{From: parentID, To: node.Task.ID}` where parentID starts as the target ID. This means the edge `{from: target, to: blocker}` which reads naturally as "target is blocked by blocker".
     b. Collect downstream edges from `result.Blocks` using `collectEdges(result.Target.ID, result.Blocks)`. Edges read as `{from: target, to: blocked_task}`.
     c. Build sections: only include `blocked_by[N]{from,to}:` if upstream edges are non-empty. Only include `blocks[N]{from,to}:` if downstream edges are non-empty. Use `encodeToonSection` for each.
     d. Join non-empty sections with `"\n\n"`.

5. **`internal/cli/toon_formatter_test.go`** -- Add a `TestToonFormatDepTree` test function with subtests.

**Acceptance Criteria**:
- [ ] Full graph mode outputs `dep_tree[N]{from,to}:` with one edge per line, where N is the edge count
- [ ] Full graph mode appends a `summary{chains,longest,blocked}:` single-object section separated by blank line
- [ ] Focused mode outputs `blocked_by[N]{from,to}:` for upstream edges when non-empty
- [ ] Focused mode outputs `blocks[N]{from,to}:` for downstream edges when non-empty
- [ ] Asymmetric focused view omits the empty direction section entirely (no zero-count header)
- [ ] Diamond dependencies produce duplicate edges (same `to` under different `from` values)
- [ ] Wide graphs (task blocked by many) produce one edge per line naturally
- [ ] All edges use task IDs as `from` and `to` values
- [ ] Output is valid toon format parseable by the toon-go library conventions
- [ ] All existing tests pass (`go test ./...`)

**Tests**:
- `"it renders single chain as edge list in full graph mode"` -- A blocks B: output is `dep_tree[1]{from,to}:\n  tick-aaa,tick-bbb` plus summary section
- `"it renders multi-level chain as edge list"` -- A->B->C: 2 edges `(A,B)` and `(B,C)`, summary shows chains:1, longest:2, blocked:2
- `"it renders multiple independent chains"` -- A->B and C->D: 2 edges, summary shows chains:2, longest:1, blocked:2
- `"it duplicates edges for diamond dependencies"` -- A->B, A->C, B->D, C->D: 4 edges including both `(B,D)` and `(C,D)`
- `"it renders summary as single-object section"` -- verify summary section uses `summary{chains,longest,blocked}:` format without `[1]`
- `"it renders focused view with both directions"` -- B in chain A->B->C: output has `blocked_by[1]{from,to}:` with `(B,A)` and `blocks[1]{from,to}:` with `(B,C)`
- `"it omits blocked_by section when only downstream exists"` -- root task A blocking B: output has only `blocks[1]{from,to}:`, no `blocked_by` section
- `"it omits blocks section when only upstream exists"` -- leaf task C blocked by B: output has only `blocked_by[1]{from,to}:`, no `blocks` section
- `"it renders wide graph with many edges"` -- A blocks B, C, D, E: 4 edges, one per line
- `"it duplicates edges in focused downstream for diamond"` -- focused on A where A->B, A->C, B->D, C->D: blocks section has 4 edges

**Edge Cases**:
- Diamond dependencies produce duplicate edges: since the `DepTreeNode` tree already duplicates nodes at each appearance, the edge collector naturally produces duplicate `to` entries under different `from` values. For example, A->B->D and A->C->D produces edges (A,B), (B,D), (A,C), (C,D) -- the destination D appears twice with different sources.
- Asymmetric focused view omits empty section: when `result.BlockedBy` is empty, no `blocked_by` section is rendered at all (not even a zero-count header). Similarly for `result.Blocks`. This differs from toon format's usual pattern of showing `[0]` headers for empty sections (like `blocked_by[0]{id,title,status}:` in show output), because the spec explicitly says "Only show sections that have content."
- Wide graph with many edges: a task blocked by 10 others produces 10 edge rows in the section. The toon format handles this naturally with one row per line.

**Context**:
> The specification states for toon format: "Flat edge list in standard toon format. Full graph: `dep_tree[N]{from,to}:` with one edge per line. Focused mode: separate `blocked_by[N]{from,to}:` and `blocks[N]{from,to}:` sections for upstream/downstream edges respectively. Machine-parseable for agent consumption." The asymmetric view specification says: "only show sections that have content" which means we must not render zero-count headers for empty directions. This is a departure from the show command's pattern of always showing `blocked_by[0]{id,title,status}:` -- the dep tree focused view intentionally omits empty sections.

**Spec Reference**: `.workflows/dep-tree-visualization/specification/dep-tree-visualization/specification.md` -- "Formatter Integration" (Toon format details), "Rendering" (diamond dependencies), and "Edge Cases" (asymmetric focused view).

## dep-tree-visualization-2-2 | pending

### Task 2: JSONFormatter FormatDepTree

**Problem**: The `JSONFormatter.FormatDepTree` is a stub returning `""`. The JSON format is used for debugging and programmatic consumption, and needs a structured representation of the dependency tree. Without this implementation, `tick dep tree` and `tick dep tree <id>` produce empty output when the `--json` flag is used.

**Solution**: Implement `FormatDepTree` on `JSONFormatter` in `internal/cli/json_formatter.go`. Use the nested tree structure (mirroring the `DepTreeResult` data model) rather than a flat nodes+edges representation, since the tree is already built by the graph algorithm. For full-graph mode, output a JSON object with `mode`, `roots` (nested tree), `chains`, `longest`, and `blocked`. For focused mode, output a JSON object with `mode`, `target`, `blocked_by` (upstream tree), and `blocks` (downstream tree). Empty arrays render as `[]` not `null`, following the existing JSON formatter convention. Asymmetric focused views include both keys but render the empty direction as `[]`.

**Outcome**: `JSONFormatter.FormatDepTree` produces valid, well-structured JSON output for both modes. All arrays are `[]` not `null`. Diamond dependencies appear as duplicate nodes in the nested tree. The output follows existing JSON formatter conventions (snake_case keys, 2-space indent, `json.MarshalIndent`).

**Do**:
1. **`internal/cli/json_formatter.go`** -- Define JSON-serializable types for the dep tree:
   ```go
   type jsonDepTreeTask struct {
       ID     string `json:"id"`
       Title  string `json:"title"`
       Status string `json:"status"`
   }

   type jsonDepTreeNode struct {
       Task     jsonDepTreeTask    `json:"task"`
       Children []jsonDepTreeNode  `json:"children"`
   }

   type jsonDepTreeFull struct {
       Mode     string             `json:"mode"`
       Roots    []jsonDepTreeNode  `json:"roots"`
       Chains   int                `json:"chains"`
       Longest  int                `json:"longest"`
       Blocked  int                `json:"blocked"`
   }

   type jsonDepTreeFocused struct {
       Mode      string             `json:"mode"`
       Target    jsonDepTreeTask    `json:"target"`
       BlockedBy []jsonDepTreeNode  `json:"blocked_by"`
       Blocks    []jsonDepTreeNode  `json:"blocks"`
   }
   ```
2. **`internal/cli/json_formatter.go`** -- Implement a helper function `toJSONDepTreeNodes(nodes []DepTreeNode) []jsonDepTreeNode` that recursively converts `[]DepTreeNode` to `[]jsonDepTreeNode`. For each node, convert `node.Task` to `jsonDepTreeTask` and recursively convert `node.Children`. Critically, always initialize `Children` as `make([]jsonDepTreeNode, 0)` for leaf nodes to ensure JSON serialization produces `[]` not `null`.
3. **`internal/cli/json_formatter.go`** -- Replace the stub `FormatDepTree` with the full implementation:
   - **Full graph mode** (`result.Mode == "full"`):
     a. Convert `result.Roots` to `[]jsonDepTreeNode` via `toJSONDepTreeNodes`.
     b. Build `jsonDepTreeFull{Mode: "full", Roots: roots, Chains: result.ChainCount, Longest: result.LongestChain, Blocked: result.BlockedCount}`. Ensure `Roots` is initialized as empty slice (not nil) when there are no roots.
     c. Return `marshalIndentJSON(obj)`.
   - **Focused mode** (`result.Mode == "focused"`):
     a. Convert `result.Target` to `jsonDepTreeTask`.
     b. Convert `result.BlockedBy` to `[]jsonDepTreeNode` via `toJSONDepTreeNodes`. If nil or empty, use `make([]jsonDepTreeNode, 0)` to ensure `[]` in JSON.
     c. Convert `result.Blocks` to `[]jsonDepTreeNode` via `toJSONDepTreeNodes`. Same nil-safety.
     d. Build `jsonDepTreeFocused{Mode: "focused", Target: target, BlockedBy: blockedBy, Blocks: blocks}`.
     e. Return `marshalIndentJSON(obj)`.

4. **`internal/cli/json_formatter_test.go`** -- Add a `TestJSONFormatDepTree` test function with subtests. Parse output with `json.Unmarshal` into `map[string]interface{}` to verify structure, following the existing JSON test pattern.

**Acceptance Criteria**:
- [ ] Full graph mode outputs valid JSON with `mode`, `roots`, `chains`, `longest`, `blocked` keys
- [ ] Focused mode outputs valid JSON with `mode`, `target`, `blocked_by`, `blocks` keys
- [ ] All keys use snake_case (no camelCase)
- [ ] Empty `roots` in full graph mode renders as `[]` not `null`
- [ ] Empty `blocked_by` or `blocks` in focused mode renders as `[]` not `null`
- [ ] Leaf node `children` renders as `[]` not `null`
- [ ] Diamond dependencies appear as duplicate nodes in the nested tree structure
- [ ] Output uses 2-space indentation via `marshalIndentJSON`
- [ ] All output is valid JSON parseable by `json.Unmarshal`
- [ ] All existing tests pass (`go test ./...`)

**Tests**:
- `"it renders full graph mode as structured JSON"` -- A blocks B: JSON has `mode: "full"`, `roots` array with one root node containing one child, `chains: 1`, `longest: 1`, `blocked: 1`
- `"it renders multi-level chain in full graph"` -- A->B->C: root A has child B which has child C. Verify nested structure via JSON parsing.
- `"it renders roots as [] not null when empty"` -- construct `DepTreeResult{Mode: "full", Roots: nil}`: JSON `roots` key must be `[]`
- `"it renders leaf children as [] not null"` -- A blocks B (B is leaf): B's `children` in JSON must be `[]` not `null`
- `"it duplicates diamond dependency nodes in tree"` -- A->B->D, A->C->D: root A has children B and C; B has child D, C also has child D. Both D nodes present in parsed JSON.
- `"it renders focused mode with both directions"` -- B in chain A->B->C: JSON has `mode: "focused"`, `target` with B's info, `blocked_by` array with A tree, `blocks` array with C tree
- `"it renders focused blocked_by as [] when only downstream"` -- root task A blocking B: `blocked_by` is `[]`, `blocks` has content
- `"it renders focused blocks as [] when only upstream"` -- leaf task C blocked by B: `blocks` is `[]`, `blocked_by` has content
- `"it uses snake_case for all keys"` -- parse JSON, verify no camelCase keys like `blockedBy` or `chainCount`
- `"it renders target task with id, title, status"` -- focused mode target has exactly these three fields
- `"it produces valid 2-space indented JSON"` -- output contains `\n  ` indentation patterns

**Edge Cases**:
- Diamond dependencies produce duplicate nodes in tree: since `DepTreeNode` trees already contain duplicated nodes, the JSON conversion preserves this duplication. Task D appearing under both B and C in the tree will appear as two separate node objects in the JSON output.
- Asymmetric focused view: unlike the toon format which omits empty sections, the JSON format always includes both `blocked_by` and `blocks` keys, rendering the empty direction as `[]`. This follows the existing JSON convention where arrays are always present (e.g., `blocked_by` and `children` in `FormatTaskDetail` are always `[]` not omitted). The spec says "omits empty arrays" for focused view, but this should be interpreted as "empty arrays render as `[]`" to maintain JSON structural consistency. This is noted as an ambiguity -- the spec's phrasing "Asymmetric focused view omits empty arrays" could mean either "omit the key" or "render as empty array". Given the project's strong convention of always including array keys (never omitting them), we render as `[]`.
- Empty arrays render as `[]` not `null`: the `toJSONDepTreeNodes` helper must initialize slices with `make([]T, 0)` rather than leaving them as nil, since Go's `json.Marshal` renders nil slices as `null`. This matches the pattern in `toJSONRelated` and `FormatTaskList`.

**Context**:
> The specification says for JSON format: "Structured graph -- nodes array + edges array, or nested object mirroring the tree structure. Exact shape determined during implementation." We choose the nested tree structure because: (1) the `DepTreeResult` already contains the tree, so the mapping is direct; (2) it preserves the diamond duplication naturally; (3) a flat nodes+edges format would require deduplication logic that contradicts the spec's "no deduplication" rule. The existing JSON formatter conventions are: `json.MarshalIndent` with 2-space indent, snake_case keys, `make([]T, 0)` for empty slices to produce `[]` not `null`, and `omitempty` only for truly optional scalar fields (parent, closed). The edge case note about "empty arrays render as [] not null" in the task table aligns with this existing convention.

**Spec Reference**: `.workflows/dep-tree-visualization/specification/dep-tree-visualization/specification.md` -- "Formatter Integration" (JSON format details), "Rendering" (diamond dependencies), and "Edge Cases" (asymmetric focused view).
