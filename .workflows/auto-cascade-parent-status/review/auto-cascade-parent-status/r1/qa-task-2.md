TASK: Migrate ValidateAddDep into StateMachine

ACCEPTANCE CRITERIA:
- StateMachine.ValidateAddDep() passes all existing dependency validation tests (cycle detection, child-blocked-by-parent)
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: Rules 10 and 11 from the spec require cycle detection and child-blocked-by-parent rejection to be migrated into the StateMachine. Rule 8 (cancelled dependency guard) is also included in ValidateAddDep but is a separate task (auto-cascade-parent-status-1-4). The spec states all ID comparisons should be case-insensitive via NormalizeID.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/state_machine.go:78-195
- Notes: ValidateAddDep is implemented on StateMachine with proper case-insensitive ID handling via NormalizeID. The old dependency.go:6-9 now delegates to StateMachine as a thin wrapper (ValidateDependency calls sm.ValidateAddDep). The implementation covers self-reference check, DFS-based cycle detection with full path reporting, and child-blocked-by-parent validation. Rule 8 (cancelled blocker guard) is also wired in at line 87-94, which belongs to task auto-cascade-parent-status-1-4 but was implemented here -- acceptable since the tasks are sequential. CLI callers in create.go, update.go, dep.go, and helpers.go all use sm.ValidateAddDep directly.

TESTS:
- Status: Adequate
- Coverage: TestStateMachine_ValidateAddDep (state_machine_test.go:336-528) covers: valid dependency, self-reference, 2-node cycle, 3+ node cycle, child-blocked-by-parent, parent-blocked-by-child (allowed), mixed-case cycle detection, cancelled blocker rejection, open/in_progress/done blocker (allowed), cycle after cancelled check passes, mixed-case child-blocked-by-parent. Original TestValidateDependency tests in dependency_test.go remain and verify the wrapper still works (multi-hop cycles, sibling deps, cross-hierarchy deps, batch validation). All three edge cases from the plan (mixed-case IDs, self-reference, multi-hop cycles) are explicitly tested.
- Notes: Some test duplication between state_machine_test.go and dependency_test.go (both test cycle detection, mixed-case, child-blocked-by-parent). This is acceptable since dependency_test.go validates the wrapper contract and was pre-existing.

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run subtests, "it does X" naming, error wrapping with fmt.Errorf.
- SOLID principles: Good. StateMachine is stateless method grouping per spec. Helper functions (smValidateChildBlockedByParent, smDetectCycle, smDFS) have single responsibility. ValidateDependency wrapper preserves backward compatibility.
- Complexity: Low. DFS cycle detection is straightforward with visited map and path tracking. Linear scan for child-blocked-by-parent and cancelled check.
- Modern idioms: Yes. Range over index for slice iteration, string builder via strings.Join for path display.
- Readability: Good. Clear function names, doc comments on all exported and unexported functions, logical ordering (validate cancelled -> validate parent -> detect cycle).
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `sm` prefix on helper functions (smValidateChildBlockedByParent, smDetectCycle, smDFS) is slightly unconventional -- these are package-level functions, not methods, so the prefix is used to disambiguate from any old implementations. Acceptable given the migration context.
