TASK: Fix Focused No-Deps Edge Case to Route Entirely Through Formatter

ACCEPTANCE CRITERIA:
- tick dep tree <id> on a task with no dependencies produces output through the formatter in all three formats
- JSON output for this case is valid JSON containing both target task info and the "No dependencies." message
- No raw fmt.Fprintf to stdout in the no-deps focused code path
- Pretty and toon output include the task info line and the message

STATUS: Complete

SPEC CONTEXT: The specification (Edge Cases section) says: "Task with no dependencies (focused mode): Show the task itself with 'No dependencies.'" The broader requirement is that all output routes through the Formatter interface (Formatter Integration section: "every command output goes through the formatter"). The bug was that the handler was writing raw text to stdout before calling the formatter, producing mixed/invalid output in JSON and toon formats.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Handler: internal/cli/dep_tree.go:48-63 -- runFocusedDepTree always calls fmtr.FormatDepTree(result), no raw fmt.Fprintf
  - Graph builder: internal/cli/dep_tree_graph.go:248-273 -- BuildFocusedDepTree sets Target + Message="No dependencies." when both BlockedBy and Blocks are empty
  - JSON: internal/cli/json_formatter.go:360-370 -- FormatDepTree routes to formatFocusedDepTreeJSON when Target is set; lines 385-408 handle focused-with-message correctly (sets obj.Message when both directions empty)
  - Pretty: internal/cli/pretty_formatter.go:337-345 -- formatFocusedDepTree renders task info line then message when both directions empty
  - Toon: internal/cli/toon_formatter.go:209-212 -- formatFocusedDepTree renders task info + message when both directions empty
- Notes: No raw fmt.Fprintf to stdout exists in the focused code path. All three fmt.Fprintln calls in dep_tree.go route through formatter methods (FormatMessage or FormatDepTree). The fix is clean and well-structured.

TESTS:
- Status: Adequate
- Coverage:
  - JSON formatter test: json_formatter_test.go:1474-1515 -- "it renders focused no-deps as valid JSON with target and message" -- validates JSON is valid, checks mode="focused", target.id, target.title, target.status, message="No dependencies."
  - Pretty formatter test: pretty_formatter_test.go:1067-1096 -- "it renders focused no-deps with task info and message" -- checks task ID, title, status in parens, "No dependencies." text, and first line format
  - Toon formatter test: toon_formatter_test.go:970-992 -- "it renders focused no-deps with task info and message" -- checks task ID, title, status, and message text
  - End-to-end handler test (JSON): dep_tree_test.go:278-317 -- "it outputs valid JSON for focused no-deps case" -- validates JSON, checks target.id and message via App.Run with --json flag
  - End-to-end handler test (Pretty): dep_tree_test.go:193-219 -- "it outputs no dependencies for isolated task in focused mode" -- checks ID, title, status, message via App.Run with --pretty flag
- Notes: All three formatter-level tests and two handler-level integration tests cover the acceptance criteria. The tests verify behavior, not implementation details. They would fail if the feature broke. No over-testing observed.

CODE QUALITY:
- Project conventions: Followed -- uses Formatter interface pattern, handler signature conventions, error wrapping with %w, stdlib testing with t.Run subtests
- SOLID principles: Good -- single responsibility maintained (handler delegates to formatter, formatter handles rendering), formatter implementations follow interface segregation
- Complexity: Low -- the conditional logic in each formatter is straightforward (check Target != nil, then check empty BlockedBy/Blocks)
- Modern idioms: Yes -- proper use of omitempty in JSON struct tags, idiomatic Go error handling
- Readability: Good -- clear comments on each formatter method explaining the focused-no-deps case, code is self-documenting
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
