TASK: Consolidate formatter duplication and fix Unicode arrow (tick-core-6-4)

ACCEPTANCE CRITERIA:
- FormatTransition and FormatDepChange exist in one place only
- Transition output uses Unicode right arrow matching spec line 639
- ToonFormatter, PrettyFormatter (and JsonFormatter if applicable) produce correct output
- All existing formatter tests pass

STATUS: Complete

SPEC CONTEXT: Spec line 639 shows transition output as `tick-a3f2b7: open -> in_progress` using a Unicode right arrow (U+2192). The task identified that ToonFormatter and PrettyFormatter had identical FormatTransition and FormatDepChange implementations, and that ASCII `->` was used instead of the spec-mandated Unicode arrow. dependency.go already used the correct Unicode arrow for cycle error messages.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/format.go:131-146` -- `baseFormatter` struct with shared `FormatTransition` and `FormatDepChange` methods
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:14-16` -- `ToonFormatter` embeds `baseFormatter`
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:14-16` -- `PrettyFormatter` embeds `baseFormatter`
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:11` -- `JSONFormatter` does NOT embed `baseFormatter` (correct: it has its own JSON-structured implementations at lines 110-116 and 126-132)
- Notes:
  - FormatTransition uses `\u2192` (Unicode right arrow) at format.go:137, matching spec line 639
  - FormatDepChange handles both "added" and "removed" actions at format.go:141-146, matching spec format
  - No duplicate FormatTransition or FormatDepChange methods remain on ToonFormatter or PrettyFormatter (confirmed via grep)
  - No ASCII `->` used in any formatter transition output (confirmed via grep)
  - Internal consistency achieved: both dependency.go cycle errors and formatter transition output use `\u2192`
  - JSONFormatter correctly not embedded since it renders structured JSON rather than plain text

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/base_formatter_test.go:5-96` -- Dedicated test file for baseFormatter:
    - Tests FormatTransition contains Unicode right arrow (line 6-12)
    - Table-driven test verifying spec format `id: old_status arrow new_status` with 3 transitions (lines 14-54)
    - Tests FormatDepChange for "added" action (lines 56-63)
    - Tests FormatDepChange for "removed" action (lines 65-72)
    - Tests all three formatters produce consistent transition output, verifying Toon and Pretty produce identical results (lines 75-96)
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter_test.go:312-332` -- Tests FormatTransition and FormatDepChange through ToonFormatter (verifies embedding works)
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter_test.go:333-353` -- Tests FormatTransition and FormatDepChange through PrettyFormatter (verifies embedding works)
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter_test.go:459-513` -- Tests JSONFormatter's own FormatTransition and FormatDepChange (verifies JSON structure)
  - `/Users/leeovery/Code/tick/internal/cli/format_integration_test.go:83-151` -- Integration test verifying transition output across all 3 formats end-to-end
  - `/Users/leeovery/Code/tick/internal/cli/transition_test.go:174` -- Tests RunTransition produces correct Unicode arrow output
- Notes:
  - All four test requirements from the task are covered: Unicode arrow presence, spec format matching, dep change correctness, and cross-formatter consistency
  - Tests in individual formatter test files (toon/pretty) verify that the embedded methods work correctly through the concrete types, not just through baseFormatter directly
  - The toon/pretty formatter tests for FormatTransition and FormatDepChange are technically testing the same baseFormatter code path, but this is justified since they validate the embedding actually works for each concrete type

CODE QUALITY:
- Project conventions: Followed -- Go idioms, table-driven tests, compile-time interface checks
- SOLID principles: Good -- baseFormatter follows Single Responsibility (shared text formatting), Open/Closed (JSONFormatter can override without modifying base), and Liskov Substitution (ToonFormatter and PrettyFormatter are substitutable where baseFormatter behavior is expected)
- Complexity: Low -- baseFormatter has two simple methods, embedding is straightforward
- Modern idioms: Yes -- Go struct embedding used idiomatically for shared behavior
- Readability: Good -- clear doc comments on baseFormatter, FormatTransition, and FormatDepChange; the relationship between baseFormatter and concrete types is immediately obvious
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The FormatTransition and FormatDepChange tests in toon_formatter_test.go and pretty_formatter_test.go are technically redundant with base_formatter_test.go since they exercise identical code paths via embedding. However, they serve as integration-level verification that embedding works correctly for each type, so retaining them is reasonable.
