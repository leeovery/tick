TASK: Extract Shared Box-Drawing Tree Helper From PrettyFormatter

ACCEPTANCE CRITERIA:
- A single writeTree (or similarly named) helper contains the box-drawing connector logic
- writeCascadeTree and writeDepTreeNodes delegate to the shared helper
- All existing pretty formatter tests pass unchanged
- Box-drawing output for both cascade transitions and dep tree is visually identical to before

STATUS: Complete

SPEC CONTEXT: The specification defines box-drawing characters for tree rendering in the pretty format. Two independent tree-rendering paths existed: cascade transitions (parent/child hierarchy visualization) and dependency tree visualization. Both used identical recursive control flow for connector selection, prefix computation, and child indentation, differing only in node types and line content formatting.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/pretty_formatter.go:11-42
- Notes: A generic `writeTree[T any]` function was extracted at lines 23-42. It is parameterized by:
  - `treeStyle` struct (lines 12-17) holding mid/last connector strings and childMid/childLast prefix strings
  - `renderLine` callback for node-specific content rendering
  - `getChildren` callback for child node retrieval
  - `depth` parameter for depth-aware rendering (used by dep tree title truncation)
  Both `writeCascadeTree` (line 288-297) and `writeDepTreeNodes` (line 377-386) are now thin wrappers that call `writeTree` with their respective styles and callbacks. The cascade style uses 2-char connectors without trailing space, while the dep tree style uses 4-char connectors with trailing space -- both correctly parameterized via separate `treeStyle` instances (cascadeTreeStyle at line 280, depTreeStyle at line 368). No duplicated box-drawing logic remains in the codebase.

TESTS:
- Status: Adequate
- Coverage: Existing cascade transition tests in cascade_formatter_test.go (TestPrettyFormatterCascadeTransition: 3 subtests covering flat cascades, 3-level hierarchy, and upward cascades) verify cascade tree output is unchanged. Existing dep tree tests in pretty_formatter_test.go (TestPrettyFormatDepTree: 14 subtests covering single chain, multiple roots, diamond dependencies, deep chains, focused views, title truncation, box-drawing characters, asymmetric views, and no-deps edge case) verify dep tree output is unchanged. Both test suites assert exact string equality against expected output containing specific box-drawing characters and indentation, meaning any regression in the shared helper would be caught.
- Notes: This is a pure refactoring task. The acceptance criteria explicitly state "run existing tests, verify output unchanged." No new tests are needed. The existing tests serve as comprehensive regression tests for the refactored helper.

CODE QUALITY:
- Project conventions: Followed. Uses Go generics idiomatically, follows the project's callback/functional style over interfaces (consistent with StoreOption pattern), unexported helper functions.
- SOLID principles: Good. writeTree has a single responsibility (box-drawing tree rendering). The callback parameters follow the open/closed principle -- new tree renderers can be added without modifying writeTree. Dependency inversion via callbacks rather than concrete types.
- Complexity: Low. The writeTree function is 19 lines with straightforward recursive logic. The wrappers are 8-9 lines each.
- Modern idioms: Yes. Uses Go generics (`[T any]`) for type-safe parameterization, avoiding interface{} or code generation. The io.Writer parameter on renderLine allows writing to any writer type.
- Readability: Good. The treeStyle struct has clear field names with doc comments explaining each character's purpose. The writeTree function signature is well-documented. Both wrapper functions clearly show what differs between cascade and dep tree rendering.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The treeStyle instances could potentially be declared as constants via a const block, but Go does not support const structs, so package-level vars are the correct approach.
