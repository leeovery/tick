AGENT: duplication
FINDINGS:
- FINDING: Cascade output rendering pattern repeated 4 times
  SEVERITY: medium
  FILES: internal/cli/transition.go:56-63, internal/cli/create.go:293-298, internal/cli/update.go:447-453, internal/cli/update.go:457-463
  DESCRIPTION: The same conditional pattern appears 4 times across 3 files: check if cascades slice is empty, if so call FormatTransition for a simple output, otherwise call buildCascadeResult then FormatCascadeTransition. Each instance is 5-7 lines with identical structure differing only in variable names (parentID/r6ParentID/r3.parentID, result/parentResult/r6Result/r3.result, etc.).
  RECOMMENDATION: Extract a helper function like `outputTransitionOrCascade(stdout, fmtr, id, title string, result TransitionResult, cascades []CascadeChange, tasks []Task)` in transition.go or a shared helpers file. All four call sites reduce to a single function call.

- FINDING: Rule 6/7 parent validation and reopen cascade duplicated in create and update
  SEVERITY: medium
  FILES: internal/cli/create.go:228-246, internal/cli/update.go:338-357
  DESCRIPTION: Both RunCreate and RunUpdate contain nearly identical 15-18 line blocks that (1) iterate tasks to find the parent by ID, (2) call sm.ValidateAddChild to enforce Rule 7, (3) check if parent status is done, and (4) call sm.ApplyWithCascades with "reopen" to trigger Rule 6. The only differences are variable names for capturing the result (parentReopened/r6Triggered, parentResult/r6Result, etc.).
  RECOMMENDATION: Extract a function like `validateAndReopenParent(tasks []Task, parentID string, sm *StateMachine) (TransitionResult, []CascadeChange, bool, error)` that encapsulates the validate-then-reopen pattern. Both create.go and update.go call it and capture the results.

- FINDING: Rule 3 all-children-terminal logic duplicated between cascades.go and update.go
  SEVERITY: medium
  FILES: internal/task/cascades.go:113-169, internal/cli/update.go:136-201
  DESCRIPTION: cascadeUpwardCompletion (in the task package) and evaluateRule3 (in the cli package) both independently implement the same "check if all children of a parent are terminal, determine done vs cancelled" logic. Both build the same allTerminal/anyDone boolean pattern over children, determine the action string, and derive the new status. evaluateRule3 exists because reparenting needs to trigger Rule 3 on the original parent outside the normal cascade flow, but the core evaluation logic is a near-duplicate of cascadeUpwardCompletion.
  RECOMMENDATION: Extract the "are all children terminal and what status should the parent get" evaluation into a shared function in the task package (e.g., `EvaluateCompletion(tasks []Task, parentID string) (action string, shouldComplete bool)`). Both cascadeUpwardCompletion and evaluateRule3 can call it, eliminating the duplicated child-scanning logic.

SUMMARY: Three medium-severity duplication patterns found, all stemming from independent task executors implementing cascade output, parent validation+reopen, and Rule 3 evaluation logic in isolation. The cascade output helper would consolidate 4 call sites; the parent validation helper would consolidate 2; and the completion evaluation helper would unify logic split across the task and cli packages.
