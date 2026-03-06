AGENT: standards
FINDINGS:
- FINDING: Pretty format renders flat list instead of hierarchical tree
  SEVERITY: high
  FILES: internal/cli/pretty_formatter.go:191-224, internal/cli/format.go:131-154
  DESCRIPTION: The spec explicitly shows nested tree output for pretty format downward cascades, where grandchildren are indented under their parent children using box-drawing characters (e.g., tick-child2 with tick-grand1/tick-grand2 nested beneath it using "│  ├─" and "│  └─"). The implementation renders all cascaded entries as a flat list at the same indentation level. The CascadeResult/CascadeEntry types lack any parent/hierarchy information, making nested rendering impossible with the current data model. The toon format is flat by spec design, so only pretty format is affected.
  RECOMMENDATION: Add a Parent field to CascadeEntry (or restructure as a tree), then update PrettyFormatter.FormatCascadeTransition to render nested tree output matching the spec examples. The JSON format should remain flat per spec.

- FINDING: StateMachine.Transition signature diverges from spec API surface
  SEVERITY: medium
  FILES: internal/task/state_machine.go:35
  DESCRIPTION: The spec defines Transition as `func (sm *StateMachine) Transition(t *Task, action string) (TransitionResult, error)`. The implementation adds a third parameter `tasks []Task` to support Rule 9 (block reopen under cancelled parent). This changes the public API surface from what was specified. The tasks parameter is needed for Rule 9 validation, but the spec did not account for this in the Transition signature — it may have intended Rule 9 to be checked elsewhere (e.g., in ApplyWithCascades or a separate validation method). Additionally, the old Transition wrapper passes nil for tasks, meaning Rule 9 is silently skipped for callers using the backward-compatible wrapper.
  RECOMMENDATION: Consider either updating the spec to acknowledge the additional parameter, or moving Rule 9 validation into ApplyWithCascades (which already has access to the task list) to keep Transition's signature matching the spec. If the current approach is kept, the backward-compatible wrapper Transition() should be documented as not enforcing Rule 9.

- FINDING: task_transitions table missing FOREIGN KEY constraint from spec
  SEVERITY: low
  FILES: internal/storage/cache.go:54-60
  DESCRIPTION: The spec defines the task_transitions table with `FOREIGN KEY (task_id) REFERENCES tasks(id)`. The implementation omits this foreign key constraint. While SQLite does not enforce foreign keys by default (requires PRAGMA foreign_keys = ON), the spec explicitly included it, and other tables in the schema (like dependencies, task_tags, task_refs, task_notes) also lack foreign keys, so this is consistent with existing patterns.
  RECOMMENDATION: This is consistent with the project's existing cache tables (none use foreign keys). Low priority — the spec's FK was aspirational given the project pattern. No change needed unless the project decides to enable FK enforcement.

- FINDING: Unchanged entries only collected for direct children of primary task
  SEVERITY: medium
  FILES: internal/cli/transition.go:91-108
  DESCRIPTION: The buildCascadeResult function collects unchanged terminal entries only from direct children of the primary task. For deep hierarchies, terminal grandchildren that were untouched by a cascade will not appear in the "unchanged" section. The spec says "both formats show unchanged terminal children so the user can see what was not affected by the cascade." While the spec examples only show direct children as unchanged, the principle implies all unchanged terminal descendants should be visible, especially since the pretty format spec shows a nested tree where grandchildren appear.
  RECOMMENDATION: If the pretty format tree rendering is implemented (finding 1), unchanged entries should also be collected recursively for all descendants of the primary task, not just direct children.

SUMMARY: The most significant drift is the pretty format rendering cascades as a flat list instead of the hierarchical tree structure shown in the spec. The StateMachine.Transition signature gained an unspecified parameter to support Rule 9. Unchanged entry collection is limited to direct children only.
