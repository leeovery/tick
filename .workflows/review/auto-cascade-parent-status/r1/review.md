# Implementation Review: Auto-Cascade Parent Status

**Plan**: auto-cascade-parent-status
**QA Verdict**: Approve

## Summary

Excellent implementation across all 24 tasks in 4 phases. Every acceptance criterion is met with no blocking issues. The StateMachine architecture cleanly consolidates all 11 cascade/validation rules. Domain logic is pure and testable (Cascades computes without mutating, ApplyWithCascades orchestrates). CLI integration is atomic (single Mutate call), all three formatters render correctly, and analysis-cycle refactorings (helper extraction, defensive copies, Rule 9 relocation) improved the codebase without introducing regressions.

## QA Verification

### Specification Compliance

Implementation aligns with the specification across all rules:
- Rules 1-6 (transitions, upward/downward cascades, reopen behavior) — all implemented and tested
- Rules 7-9 (validation guards) — exact error messages match spec
- Rules 10-11 (cycle detection, child-blocked-by-parent) — migrated into StateMachine with full backward compatibility
- Transition history with `auto` flag — serializes correctly in JSONL, cached in `task_transitions` table
- CLI display — all three formats (Toon, Pretty, JSON) match spec examples
- One intentional deviation: Rule 9 moved from Transition to ApplyWithCascades (analysis cycle finding acps-4-2) — architecturally sound, keeps Transition as a pure single-task method

### Plan Completion
- [x] Phase 1 acceptance criteria met (StateMachine core + migration)
- [x] Phase 2 acceptance criteria met (cascade logic + transition history)
- [x] Phase 3 acceptance criteria met (CLI integration + cascade display)
- [x] Phase 4 acceptance criteria met (analysis cycle fixes)
- [x] All 24 tasks completed
- [x] No scope creep

### Code Quality

No issues found. Highlights:
- StateMachine is stateless (method grouping only) — idiomatic Go
- Cascades() is pure computation, ApplyWithCascades() orchestrates — clean separation
- Queue-based cascade processing with seen-map deduplication — correct termination guarantee
- Helper extraction (outputTransitionOrCascade, validateAndReopenParent, EvaluateParentCompletion) eliminates duplication across CLI commands
- Defensive copy via value-type CascadeResult built inside Mutate closure — elegant approach
- Old functions (task.Transition, task.ValidateDependency) preserved as thin wrappers for backward compatibility

### Test Quality

Tests adequately verify requirements. 24/24 tasks have appropriate test coverage with all plan edge cases tested. Test balance is good — no significant under-testing or over-testing detected. Minor observations:
- StateMachine tests and original wrapper tests have some overlap (acceptable — both are public APIs)
- Purity tests on Cascades() verify no mutation — good practice for a pure-computation function

### Required Changes

None.

## Recommendations

Non-blocking observations from QA verifiers:

1. **Minor naming mismatch**: Task acps-1-5 is named "Add reopen-under-cancelled-parent guard to Transition" but the guard lives in ApplyWithCascades per acps-4-2. Not a code issue — just a plan-vs-implementation naming discrepancy.

2. **`sm` prefix on helpers**: Package-level functions like `smValidateChildBlockedByParent`, `smDetectCycle`, `smDFS` use an `sm` prefix to disambiguate from old implementations. Acceptable in migration context but could be cleaned up if old functions are eventually removed.

3. **`r6`/`r3` variable naming** in update.go: Variables like `r6Triggered`, `r3Result` are compact but rely on knowing the rule numbering system. A brief comment at declaration could help newcomers.

4. **`buildCascadeResult` length**: ~70 lines handling upward detection, cascade entry building, and unchanged collection. Readable as-is but could be split if it grows further.

5. **`transitions[action]` lookup** in `cascadeDownwardTerminal` doesn't guard against unknown actions — safe because the caller only passes "done"/"cancel", but a defensive check could future-proof it.

6. **Comment typo**: `state_machine.go:70` has `/ Note:` instead of `// Note:` (minor).
