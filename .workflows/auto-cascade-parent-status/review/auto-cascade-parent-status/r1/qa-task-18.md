TASK: Wire reparenting cascade logic into RunUpdate (auto-cascade-parent-status-3-5)

ACCEPTANCE CRITERIA:
- Reparenting via update triggers Rule 6 (done parent reopen) and Rule 3 (original parent re-evaluation)
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: When a child is reparented away, Rule 3 re-evaluation fires on the original parent -- if all remaining children are terminal, the parent auto-completes (done if any child done, cancelled if all cancelled). Reparenting to a done parent triggers Rule 6 (reopen to open). Clearing parent (--parent "") also counts as reparenting away.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/update.go:124-167 (evaluateRule3 helper)
  - /Users/leeovery/Code/tick/internal/cli/update.go:274-324 (Rule 6 in Mutate: validateAndReopenParent call)
  - /Users/leeovery/Code/tick/internal/cli/update.go:369-386 (Rule 3 evaluation on original parent after parent field change)
  - /Users/leeovery/Code/tick/internal/cli/update.go:418-426 (cascade output for Rule 6 and Rule 3)
  - /Users/leeovery/Code/tick/internal/cli/helpers.go:110-133 (validateAndReopenParent shared helper)
  - /Users/leeovery/Code/tick/internal/cli/helpers.go:98-108 (outputTransitionOrCascade shared helper)
- Notes: Implementation correctly captures original parent before updating (line 369), evaluates Rule 3 only when parent actually changed and original was non-empty (line 378). Rule 6 is handled via shared validateAndReopenParent helper (also used by RunCreate). Both cascades are persisted atomically within the single Mutate call. Cascade display output is built inside the Mutate closure where the tasks slice is still valid. All edge cases from plan are covered: reparent away, reparent to done, clear parent.

TESTS:
- Status: Adequate
- Coverage:
  - "it reopens done parent when reparenting to it" (Rule 6) -- verifies parent status becomes open, output contains transition
  - "it triggers Rule 3 on original parent when reparenting away" -- all remaining children done -> parent auto-done
  - "it triggers Rule 3 with cancelled result when all remaining children cancelled" -- parent auto-cancelled
  - "it triggers Rule 3 with done result when remaining children are mix of done and cancelled" -- done wins
  - "it does not trigger Rule 3 when original parent still has non-terminal children" -- negative case
  - "it handles clearing parent with Rule 3 evaluation" -- --parent "" triggers Rule 3
  - "it handles reparent to done parent plus Rule 3 on original" -- both rules fire simultaneously
  - "it blocks reparenting to cancelled parent" -- Rule 7 validation
- Notes: Good edge case coverage including the combined Rule 6 + Rule 3 scenario, clear parent, and negative case. Tests verify both persistence (task status in JSONL) and output (stdout contains transition info). Not over-tested -- each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, pointer-based optional flags, error wrapping.
- SOLID principles: Good. evaluateRule3 is extracted as a focused helper. validateAndReopenParent is shared with RunCreate (DRY). buildCascadeResult and outputTransitionOrCascade are reused across commands.
- Complexity: Acceptable. The Mutate closure is moderately complex but the logic is linear and well-commented.
- Modern idioms: Yes. Follows Go conventions throughout.
- Readability: Good. Clear variable naming (r6Triggered, r3Result, originalParent), comments explain Rule references.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The r6/r3 variable naming convention (r6Triggered, r6ParentID, r6Result, r6CascadeResult, r3Result, r3ParentID, r3CascadeResult) is compact but relies on knowing the rule numbering. Acceptable given the spec is well-documented, but a comment at the declaration site would aid newcomers.
