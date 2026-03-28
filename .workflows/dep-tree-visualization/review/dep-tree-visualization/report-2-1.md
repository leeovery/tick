TASK: ToonFormatter FormatDepTree

ACCEPTANCE CRITERIA:
- Full graph outputs dep_tree[N]{from,to}: with one edge per line where N is edge count
- Full graph appends summary{chains,longest,blocked}: single-object section
- Focused mode outputs blocked_by[N]{from,to}: for upstream edges when non-empty
- Focused mode outputs blocks[N]{from,to}: for downstream edges when non-empty
- Asymmetric focused view omits empty direction section entirely
- Diamond dependencies produce duplicate edges
- Wide graphs produce one edge per line naturally
- All edges use task IDs as from and to values
- Output is valid toon format
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: The specification defines toon format for dep tree as "Flat edge list in standard toon format. Full graph: dep_tree[N]{from,to}: with one edge per line. Focused mode: separate blocked_by[N]{from,to}: and blocks[N]{from,to}: sections for upstream/downstream edges respectively. Machine-parseable for agent consumption." Asymmetric focused views must omit empty sections entirely per "Only show sections that have content." Diamond dependencies are duplicated wherever they appear (no dedup, no back-refs).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:157-257
- Notes: Implementation is clean and complete. Two helper types (`toonEdgeRow` at line 158, `toonDepTreeSummary` at line 164) support serialization. `FormatDepTree` (line 175) dispatches to `formatFocusedDepTree` or `formatFullDepTree` based on `result.Target != nil`. Full graph mode (line 188) collects edges via `collectDownstreamEdges` and renders dep_tree + summary sections. Focused mode (line 209) collects upstream/downstream edges separately and omits empty sections. The "no dependencies" edge case (line 210-212) renders task info + message. Minor drift: plan described a `result.Mode` string discriminator, but implementation uses `result.Target != nil` — this is consistent across all three formatters and is functionally equivalent.

TESTS:
- Status: Adequate
- Coverage: All 10 tests specified in the plan are present, plus 1 bonus test for the no-deps edge case (11 total). Tests cover: single chain (line 670), multi-level chain (line 693), multiple independent chains (line 724), diamond dependencies in full graph (line 756), summary section format (line 799), focused both directions (line 829), omit blocked_by when empty (line 861), omit blocks when empty (line 880), wide graph (line 899), diamond in focused downstream (line 934), focused no-deps with task info (line 970).
- Notes: Tests verify exact header format (e.g., `dep_tree[1]{from,to}:`), exact edge values, section separation, and absence of omitted sections. Each test verifies different acceptance criteria without redundancy. The no-deps edge case test (line 970) goes beyond the plan requirements and covers the spec's "Task with no dependencies" edge case.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, "it does X" naming, toon-go library for encoding, error wrapping not needed (no errors returned). Struct types follow existing toon formatter conventions (toonXxxRow pattern).
- SOLID principles: Good. FormatDepTree is a single method on ToonFormatter implementing the Formatter interface. Helper functions (`collectDownstreamEdges`, `collectUpstreamEdges`, `buildEdgeSection`) have single responsibilities. The two edge collection functions differ in edge direction semantics — the minimal duplication is justified.
- Complexity: Low. Recursive tree walking is O(n) in nodes. No complex conditionals. Clear dispatch logic.
- Modern idioms: Yes. Uses generics via `encodeToonSection[T any]`. Clean struct embedding with `baseFormatter`.
- Readability: Good. Functions are well-named and documented with godoc comments. The distinction between downstream and upstream edge collection is clear.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The plan described a `Mode` field on `DepTreeResult` but implementation uses `Target *DepTreeTask` as the discriminator. This is consistent across all three formatters and is arguably cleaner than a string-typed discriminator, but is a minor deviation from the plan text.
