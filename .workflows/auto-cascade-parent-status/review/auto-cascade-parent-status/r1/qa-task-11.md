TASK: auto-cascade-parent-status-2-5 — Implement Cascades for upward completion cascade (Rule 3)

ACCEPTANCE CRITERIA:
- Upward completion cascade triggers parent done when at least one child done, cancelled when all cancelled (Rule 3)
- All existing tests pass with no regressions
- Edge cases: single child trivial case, mix of done and cancelled, parent with no children

STATUS: Complete

SPEC CONTEXT: Rule 3 states that when all children of a parent reach a terminal state (done or cancelled), the parent automatically transitions: done if at least one child is done, cancelled if all children are cancelled. Trigger: when a task reaches a terminal state, evaluate only its direct parent. If the parent transitions, evaluate the parent's parent, and so on up the chain. Only cascades to non-terminal parents.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/cascades.go:109-139 (`cascadeUpwardCompletion`)
- Location: /Users/leeovery/Code/tick/internal/task/cascades.go:173-224 (`EvaluateParentCompletion`)
- Notes: Implementation correctly follows the spec. `cascadeUpwardCompletion` evaluates the changed task's direct parent only (single level), and the queue-based processing in `ApplyWithCascades` handles recursive chaining up the ancestor chain by re-invoking `Cascades` on each cascaded parent. `EvaluateParentCompletion` is exported as a shared function, reused in `internal/cli/update.go` for reparenting re-evaluation (Rule 3 on original parent). The separation is clean and correct.

The logic in `EvaluateParentCompletion` correctly:
1. Returns false if parent not found
2. Returns false if parent is already terminal
3. Returns false if parent has no children
4. Returns false if any child is non-terminal
5. Returns "done" if any child is done, "cancel" if all cancelled

The `cascadeUpwardCompletion` wrapper correctly maps action to status and returns a single `CascadeChange` entry.

TESTS:
- Status: Adequate
- Coverage:
  - Parent done when mix of done and cancelled children (line 443-475)
  - Parent cancelled when all children cancelled (line 477-505)
  - No cascade when some children still non-terminal (line 507-522)
  - Single child done (trivial case) (line 524-544)
  - Single child cancelled (trivial case) (line 546-566)
  - Parent already terminal skipped (line 568-582)
  - No parent (no cascade) (line 795-804)
  - `EvaluateParentCompletion` standalone tests (line 851-995): all key scenarios including parent not found, parent terminal, no children, single child done/cancelled, ID normalization
  - Chained upward completion through `ApplyWithCascades` (apply_cascades_test.go:156-219)
  - Purity tests (no mutation) covered by existing cascade purity tests
- Notes: All edge cases from the plan are covered: single child trivial case (both done and cancelled variants), mix of done and cancelled, parent with no children. The `EvaluateParentCompletion` tests also cover defensive cases (parent not found, parent already terminal). Multi-level chaining is tested in `apply_cascades_test.go`. Test balance is good -- no redundant assertions.

CODE QUALITY:
- Project conventions: Followed. Stdlib testing only, t.Run subtests, no testify, error wrapping with %w where applicable.
- SOLID principles: Good. `EvaluateParentCompletion` is a pure function with single responsibility (evaluate whether completion should occur). Clean separation between evaluation (pure) and cascade change construction.
- Complexity: Low. `EvaluateParentCompletion` is a straightforward linear scan. `cascadeUpwardCompletion` is a thin wrapper.
- Modern idioms: Yes. Named return values on `EvaluateParentCompletion` for clarity.
- Readability: Good. Well-documented with clear comments. The exported function name clearly communicates intent.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
