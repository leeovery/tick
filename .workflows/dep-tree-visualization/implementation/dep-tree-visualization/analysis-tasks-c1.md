---
topic: dep-tree-visualization
cycle: 1
total_proposed: 3
---
# Analysis Tasks: dep-tree-visualization (Cycle 1)

## Task 1: Fix focused no-deps edge case to route entirely through formatter
status: pending
severity: high
sources: standards

**Problem**: When a focused task has no dependencies, the handler in `dep_tree.go:61-64` writes a raw `fmt.Fprintf` line to stdout before calling `fmtr.FormatMessage()`. This bypasses the formatter, producing invalid JSON output (raw text followed by a JSON object) and unstructured plain text in toon format. Additionally, the JSON formatter's `FormatDepTree` checks `result.Message != ""` before `result.Target != nil`, so even if the result were routed through the formatter, the target task info (id, title, status) would be lost from JSON output.

**Solution**: Remove the raw `fmt.Fprintf` from the handler in `dep_tree.go`. Instead, build a `DepTreeResult` with Target set and Message set to "No dependencies." (BlockedBy and Blocks empty), then pass it to `fmtr.FormatDepTree()`. In the JSON formatter's `FormatDepTree`, restructure the conditional: when `result.Target != nil`, always use the focused rendering path, embedding the message when both BlockedBy and Blocks are empty. The message-only path (`result.Message != ""` with nil Target) should only apply to full-graph mode results like "No dependencies found."

**Outcome**: The focused no-deps edge case produces valid, structured output in all three formats (pretty, toon, JSON). JSON output includes target task info alongside the message. No raw writes bypass the formatter.

**Do**:
1. In `internal/cli/dep_tree.go`, remove lines 61-64 (the `fmt.Fprintf` + `fmtr.FormatMessage` block for the no-deps case).
2. Replace with: build a `DepTreeResult{Target: &DepTreeTask{ID: task.ID, Title: task.Title, Status: string(task.Status)}, Message: "No dependencies."}` and call `fmtr.FormatDepTree(result, stdout)`.
3. In `internal/cli/json_formatter.go` `FormatDepTree`: move the `result.Target != nil` check before the `result.Message != ""` check. When Target is non-nil and both BlockedBy and Blocks are empty, include a `"message"` field in the focused JSON output alongside the target task info.
4. In `internal/cli/pretty_formatter.go` `FormatDepTree`: handle the case where Target is non-nil but BlockedBy and Blocks are both empty -- render the task info line followed by the "No dependencies." message.
5. In `internal/cli/toon_formatter.go` `FormatDepTree`: handle the same case -- render the target task info in toon format followed by the message.
6. Update or add tests for the focused no-deps case across all three formatters, verifying the output contains both the task info and the message.

**Acceptance Criteria**:
- `tick dep tree <id>` on a task with no dependencies produces output through the formatter in all three formats
- JSON output for this case is valid JSON containing both target task info and the "No dependencies." message
- No raw `fmt.Fprintf` to stdout in the no-deps focused code path
- Pretty and toon output include the task info line and the message

**Tests**:
- Test focused dep tree on a task with no dependencies using JSON formatter: verify output is valid JSON, contains task ID/title/status and message field
- Test focused dep tree on a task with no dependencies using pretty formatter: verify output contains task info and "No dependencies." text
- Test focused dep tree on a task with no dependencies using toon formatter: verify output contains task info and message in toon format

## Task 2: Add cycle guard to walkDownstream and walkUpstream
status: pending
severity: medium
sources: architecture

**Problem**: `walkDownstream` and `walkUpstream` in `dep_tree_graph.go` recurse without tracking visited nodes. While `ValidateAddDep` prevents cycles during normal use, corrupted JSONL data (manual edit, import, partial write) containing a dependency cycle would cause unbounded recursion and a stack overflow crash. The `countChains` function in the same file already correctly uses a visited set for the same graph.

**Solution**: Add a `visited map[string]bool` parameter to both `walkDownstream` and `walkUpstream`. When a node ID is already in the visited set, return nil to terminate that branch. This mirrors the existing pattern in `countChains`.

**Outcome**: Dep tree commands gracefully handle corrupted data with dependency cycles instead of crashing with a stack overflow. The tree output simply terminates at the cycle point.

**Do**:
1. In `internal/cli/dep_tree_graph.go`, add a `visited map[string]bool` parameter to the `walkDownstream` function signature.
2. At the top of `walkDownstream`, check if the current task ID is in visited. If yes, return nil. Otherwise, add it to visited and proceed.
3. Do the same for `walkUpstream`.
4. Update all call sites of `walkDownstream` and `walkUpstream` to pass `make(map[string]bool)` for the initial call (or pass through the existing set for recursive calls).
5. Add a test that constructs a Store with manually corrupted JSONL containing a dependency cycle, then runs the dep tree command and verifies it completes without panic and produces reasonable output.

**Acceptance Criteria**:
- `walkDownstream` and `walkUpstream` both accept and use a visited set
- A dependency cycle in the data does not cause a stack overflow or panic
- Normal (acyclic) dep tree output is unchanged

**Tests**:
- Test walkDownstream with a circular dependency (A blocks B, B blocks A): verify it terminates and returns a finite tree
- Test walkUpstream with a circular dependency: verify same
- Test that normal acyclic dep tree output is identical before and after the change

## Task 3: Extract shared box-drawing tree helper from PrettyFormatter
status: pending
severity: medium
sources: duplication

**Problem**: `writeCascadeTree` (pretty_formatter.go:246) and `writeDepTreeNodes` (pretty_formatter.go:328) implement the same recursive box-drawing tree pattern independently. Both iterate child nodes, compute `isLast`, select connector characters (`├──`/`└──`), write prefixed lines, compute child prefix with vertical bar or space, and recurse. The structural logic is ~18 lines each with identical control flow, differing only in node type and line content formatting.

**Solution**: Extract a generic `writeTree` helper parameterized by callbacks: one to render a node's display text and one to return its children. Both `writeCascadeTree` and `writeDepTreeNodes` become thin wrappers that pass their node-specific logic to the shared helper.

**Outcome**: The box-drawing tree rendering logic exists in one place. Future tree-rendering features (e.g., parent/child visualization) reuse the same helper without duplicating the connector logic.

**Do**:
1. In `internal/cli/pretty_formatter.go`, define a generic tree-writing helper. Use a callback approach: `writeTree(w io.Writer, nodes []T, prefix string, renderLine func(T) string, getChildren func(T) []T)` or a small interface with `Text() string` and `Children() []NodeType` methods. The callback approach avoids needing to wrap existing node types.
2. Implement the box-drawing logic once in `writeTree`: iterate nodes, compute isLast, select connector, write `prefix + connector + renderLine(node)`, compute child prefix, recurse with `getChildren(node)`.
3. Rewrite `writeCascadeTree` to call `writeTree` with a render function that formats cascade node text and a children function that returns cascade node children.
4. Rewrite `writeDepTreeNodes` to call `writeTree` with a render function that formats dep tree node text (with truncation) and a children function that returns dep tree node children.
5. Verify all existing tests pass without modification.

**Acceptance Criteria**:
- A single `writeTree` (or similarly named) helper contains the box-drawing connector logic
- `writeCascadeTree` and `writeDepTreeNodes` delegate to the shared helper
- All existing pretty formatter tests pass unchanged
- Box-drawing output for both cascade transitions and dep tree is visually identical to before

**Tests**:
- Run existing cascade transition pretty formatter tests: verify output unchanged
- Run existing dep tree pretty formatter tests: verify output unchanged
- If no direct unit tests exist for the tree rendering, verify via the integration-level command tests that produce pretty-formatted dep tree and cascade output
