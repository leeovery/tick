TASK: JSONFormatter FormatDepTree

ACCEPTANCE CRITERIA:
- Full graph outputs valid JSON with mode, roots, chains, longest, blocked keys
- Focused mode outputs valid JSON with mode, target, and present direction keys
- All keys use snake_case
- Empty roots renders as [] not null
- Asymmetric focused view omits the empty direction key from JSON output entirely (key absent, not [])
- Leaf node children renders as [] not null
- Diamond dependencies appear as duplicate nodes in nested tree
- Output uses 2-space indentation via marshalIndentJSON
- All output is valid JSON parseable by json.Unmarshal
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: The specification requires a FormatDepTree method on all three formatters. JSON format should produce "structured graph -- nodes array + edges array, or nested object mirroring the tree structure." The implementation chose a nested tree structure, which aligns with the spec's flexibility. Diamond dependencies should be duplicated without deduplication. Asymmetric focused views should only show sections with content. Edge cases include no-deps focused and empty full graph.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/json_formatter.go:306-408
- Notes: Clean implementation with well-separated types and methods. Key design decisions:
  - jsonDepTreeFull struct (line 320-326) covers full graph with all required keys
  - jsonDepTreeFocused struct (line 331-337) uses `omitempty` on BlockedBy and Blocks for asymmetric omission
  - toJSONDepTreeNodes (line 341-354) recursively converts domain types to JSON types; uses `make([]..., 0, len(nodes))` ensuring non-nil slices for empty arrays
  - FormatDepTree (line 360-370) dispatches based on Target presence, with message-only fallback for full graph edge case
  - formatFocusedDepTreeJSON (line 385-408) conditionally assigns directions only when non-empty, and includes Message when both directions are empty
  - All output routes through marshalIndentJSON (line 412-418) with 2-space indent

TESTS:
- Status: Adequate
- Coverage: All 11 tests specified in the plan are present, plus 1 additional test from Phase 3 (focused no-deps edge case). Tests cover:
  - Full graph with nested structure verification
  - Multi-level chain traversal (3 levels deep)
  - Empty roots as [] not null (nil input)
  - Leaf children as [] not null
  - Diamond dependency duplication (D under both B and C)
  - Focused mode with both directions present
  - Asymmetric omission of blocked_by key
  - Asymmetric omission of blocks key
  - snake_case key verification (both focused and full modes, plus camelCase absence check)
  - Target task field verification (id, title, status)
  - 2-space indentation and valid JSON check
  - Focused no-deps with target and message (added by Phase 3 task 3-1)
- Notes: Tests are well-structured. Each test parses JSON output and verifies specific structural properties. No redundancy -- each test validates a distinct acceptance criterion or edge case. Tests would fail if the feature broke (e.g., changing omitempty behavior, switching to nil slices, changing key names).

CODE QUALITY:
- Project conventions: Followed. Uses established patterns from other JSONFormatter methods: separate struct types for JSON serialization, snake_case JSON tags, marshalIndentJSON helper, non-nil slice initialization with make(). Follows the project's Formatter interface contract.
- SOLID principles: Good. Single responsibility (each struct represents one JSON shape), open/closed (new modes could be added without modifying existing methods). FormatDepTree dispatches to focused/full/message helpers cleanly.
- Complexity: Low. FormatDepTree has a simple 3-way dispatch. toJSONDepTreeNodes is a clean recursive converter. No deep nesting or complex conditionals.
- Modern idioms: Yes. Idiomatic Go with struct tags, pointer-based nil checks for optional fields, omitempty for conditional JSON presence.
- Readability: Good. All types and methods have doc comments explaining their purpose. The naming is clear and consistent with the rest of the formatter.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The marshalIndentJSON function uses `interface{}` (line 412) rather than `any` which is the Go 1.18+ alias. This is consistent with existing code in the file, so not a concern for this task, but could be modernized project-wide.
