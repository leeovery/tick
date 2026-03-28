TASK: RunDepTree Command Handler

ACCEPTANCE CRITERIA:
- tick dep tree with no dependencies outputs "No dependencies found."
- tick dep tree with dependencies produces non-empty output via FormatDepTree
- tick dep tree <id> with valid ID that has dependencies produces focused output
- tick dep tree <id> with valid ID that has no dependencies shows the task itself (ID + title + status) with "No dependencies."
- tick dep tree <id> with invalid/nonexistent ID returns error
- Partial ID matching works
- --quiet flag suppresses all output
- Handler uses store.ReadTasks() (read-only, shared lock)

STATUS: Complete

SPEC CONTEXT: The spec defines two modes for `tick dep tree`: full graph (all dependency chains) and focused view (single task, both directions). Edge cases include: task with no dependencies shows "No dependencies.", empty project shows "No dependencies found.", asymmetric views omit empty sections. The command is read-only with no data model changes. Dependencies only — no parent/child relationships.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/dep_tree.go:12-63
- Notes:
  - RunDepTree follows the established handler pattern: `func RunDepTree(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error`
  - Quiet mode handled at line 13-15 by early return before store access — correct pattern, avoids unnecessary I/O
  - Uses store.ReadTasks() (line 23) which acquires a shared lock — satisfies read-only requirement
  - Full graph mode delegates to BuildFullDepTree (line 36) when args is empty
  - Focused mode delegates to BuildFocusedDepTree (line 31) when args[0] is present
  - ID normalization via task.NormalizeID (line 49) and resolution via store.ResolveID (line 51) for partial ID matching
  - Edge case routing: when BuildFullDepTree returns empty Roots, the Message field ("No dependencies found.") is output via FormatMessage (line 39) — correct
  - The runFocusedDepTree helper accepts a `store interface{ ResolveID(string) (string, error) }` parameter (line 48) — uses an interface rather than concrete type, good for testability
  - Dispatch wiring confirmed in internal/cli/dep.go:31 via handleDep -> case "tree" -> RunDepTree

TESTS:
- Status: Adequate
- Coverage:
  - "it outputs no dependencies found for empty project" — verifies empty project edge case
  - "it outputs no dependencies found for project with no tasks" — verifies tasks-without-deps edge case (note: subtest name says "no tasks" but actually creates tasks with no deps — minor naming inaccuracy but semantically correct test)
  - "it outputs dep tree for project with dependencies" — verifies non-empty output, checks it's NOT the "no dependencies" message
  - "it outputs focused view for task with dependencies" — verifies focused output doesn't contain "No dependencies."
  - "it outputs no dependencies for isolated task in focused mode" — verifies task ID, title, status, and "No dependencies." message present
  - "it returns error for nonexistent task ID" — verifies non-zero exit and "not found" in stderr
  - "it resolves partial task ID" — uses bare hex "aaa111" without prefix, verifies resolution to "tick-aaa111"
  - "it suppresses output in quiet mode" — passes --quiet, verifies empty stdout
  - "it outputs valid JSON for focused no-deps case" — verifies JSON output structure with target and message fields
  - "it returns error for ambiguous partial ID" — two tasks sharing "aaa" prefix, verifies "ambiguous" error
  - "it handles focused view via full App.Run dispatch" — end-to-end integration through App.Run
  - TestDepTreeWiring (separate test function) covers: qualifyCommand for "dep tree", flag validation, help text, dispatch, and backward compatibility with dep add/remove
- Notes:
  - Tests use the runDepTree helper (line 97-108) which exercises full App.Run dispatch — integration-level tests that verify the complete chain from CLI args to output
  - Tests cover all 10 required test cases from the plan plus 2 additional useful tests (JSON validation and wiring tests)
  - No over-testing detected — each test covers a distinct acceptance criterion or edge case

CODE QUALITY:
- Project conventions: Followed — handler signature matches `Run<Command>(dir, fc, fmtr, args, stdout)` pattern exactly; error wrapping present; stdlib testing only; t.Helper() on helper; uses --pretty in tests (non-TTY buffer)
- SOLID principles: Good — single responsibility (handler delegates to graph-walking and formatting), interface segregation (runFocusedDepTree takes minimal interface), dependency inversion (formatter interface used throughout)
- Complexity: Low — RunDepTree is 20 lines, clear branching between full/focused modes, no nested conditionals
- Modern idioms: Yes — interface parameter on runFocusedDepTree, deferred Close, proper error propagation
- Readability: Good — clear function names, doc comments on all functions, logical flow is immediately apparent
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Test subtest name "it outputs no dependencies found for project with no tasks" is slightly misleading — it creates tasks that have no dependency relationships, not an empty project. Consider renaming to "it outputs no dependencies found for project with tasks but no deps" for clarity. This is cosmetic only.
