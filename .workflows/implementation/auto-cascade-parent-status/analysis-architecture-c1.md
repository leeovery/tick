AGENT: architecture
FINDINGS:
- FINDING: StateMachine.Transition signature diverges from spec by requiring []Task for Rule 9
  SEVERITY: medium
  FILES: internal/task/state_machine.go:35, internal/task/transition.go:20
  DESCRIPTION: The spec defines `Transition(t *Task, action string)` but the implementation uses `Transition(t *Task, action string, tasks []Task)` because Rule 9 (block reopen under cancelled parent) needs the full task list to look up the parent. This leaks a concern into the core transition method -- Rule 9 is the only transition rule that requires external context. Every call site that does not need Rule 9 passes `nil`, and the old compatibility wrapper `task.Transition()` passes `nil` permanently, meaning any code using the wrapper silently skips Rule 9 validation. This creates a correctness trap: callers that use `task.Transition` directly (instead of going through `ApplyWithCascades`) will allow reopening under cancelled parents.
  RECOMMENDATION: Extract Rule 9 checking into `ApplyWithCascades` (which already has the task list) and keep `Transition` with its original two-parameter signature. Rule 9 is a contextual pre-condition like the cascade rules, not an intrinsic status transition rule. This removes the nil-tasks escape hatch and makes the wrapper safe.

- FINDING: evaluateRule3 in update.go duplicates cascadeUpwardCompletion detection logic
  SEVERITY: medium
  FILES: internal/cli/update.go:136-201, internal/task/cascades.go:109-169
  DESCRIPTION: `evaluateRule3()` in `update.go` re-implements the "are all children terminal? done vs cancelled?" detection that already lives in `cascadeUpwardCompletion`. The reparenting case (child moved away) is a legitimate trigger for Rule 3, but the detection should not be hand-rolled in CLI code. If Rule 3 criteria ever change (e.g., new terminal states, different precedence), two places need updating. The function also constructs its own `TransitionResult` with `OldStatus` captured before `ApplyWithCascades`, duplicating what `ApplyWithCascades` already returns.
  RECOMMENDATION: Add a method like `StateMachine.EvaluateParentCompletion(tasks []Task, parentID string)` that encapsulates the "should this parent auto-complete" check and returns the action or nil. Both `cascadeUpwardCompletion` and `evaluateRule3` should call it. This keeps Rule 3 logic in one place inside `internal/task/`.

- FINDING: Cascade output relies on allTasks slice captured inside Mutate closure
  SEVERITY: medium
  FILES: internal/cli/transition.go:46, internal/cli/create.go:270, internal/cli/update.go:434
  DESCRIPTION: The `allTasks` variable is set to the `tasks` slice from inside the `Mutate` callback, then used after `Mutate` returns to build cascade display output. The `Mutate` callback returns the same slice it received (mutated in place), so `allTasks` points to data that has been serialized to JSONL. If `Store.Mutate` ever reuses or clears that slice buffer, the display code reads stale/corrupt data. Currently safe because Mutate writes and discards, but this is a fragile coupling -- the CLI layer depends on an implementation detail of how Mutate manages its buffer lifecycle.
  RECOMMENDATION: Copy the display-relevant data (task IDs, titles, statuses for unchanged children) inside the Mutate closure into dedicated output structs, rather than holding a reference to the mutable slice. The `CascadeChange` structs already contain pointers into the slice, so at minimum document this contract or defensively copy what's needed.

- FINDING: Pretty formatter renders flat cascade list, not tree structure from spec
  SEVERITY: low
  FILES: internal/cli/pretty_formatter.go:191-224
  DESCRIPTION: The spec shows Pretty format with nested tree structure (child entries nested under their parent with indented box-drawing characters), but the implementation renders a flat list with only top-level tree characters. For example, the spec shows grandchildren indented under their parent child entry, but the implementation lists all entries at the same level. This means deep hierarchies lose their structural context in Pretty output. The Toon and JSON formats are flat by design, so this only affects Pretty.
  RECOMMENDATION: Accept as-is for now -- the flat rendering is functional and consistent across entries. Nested tree rendering would require the cascade result to carry parent-child relationships between cascade entries, which would add complexity. If pursued later, the `CascadeResult` struct would need a tree structure rather than flat slices.

SUMMARY: The main architectural concerns are (1) Rule 9 leaked into Transition's signature creating a nil-tasks escape hatch, (2) Rule 3 detection logic duplicated between domain and CLI layers, and (3) captured mutable slice references crossing the Mutate boundary for display purposes.
